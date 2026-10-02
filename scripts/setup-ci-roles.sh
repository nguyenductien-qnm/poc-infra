#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# SCRIPT KHỞI TẠO 6 IAM ROLES CHO CI/CD (PHÂN TÁCH PREVIEW & DEPLOY THEO PROPOSAL)
# ==============================================================================

GITHUB_ORG_OR_USER="${1:-nguyenductien-qnm}"   # Tham so 1: Username hoac Organization GitHub
GITHUB_REPO_NAME="${2:-poc-infra}"           # Tham so 2: Ten Repository GitHub

# Repo tao sau 15/07/2026 dung OIDC sub dang immutable: repo:<owner>@<owner_id>/<repo>@<repo_id>:...
# Lay ID qua GitHub API (repo public); repo private thi truyen san GITHUB_OWNER_ID / GITHUB_REPO_ID
if [ -z "${GITHUB_OWNER_ID:-}" ] || [ -z "${GITHUB_REPO_ID:-}" ]; then
    REPO_JSON=$(curl -fsSL "https://api.github.com/repos/${GITHUB_ORG_OR_USER}/${GITHUB_REPO_NAME}")
    GITHUB_OWNER_ID=$(echo "$REPO_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["owner"]["id"])')
    GITHUB_REPO_ID=$(echo "$REPO_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
fi
SUB_PREFIX="repo:${GITHUB_ORG_OR_USER}@${GITHUB_OWNER_ID}/${GITHUB_REPO_NAME}@${GITHUB_REPO_ID}"

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION="ap-southeast-1"
S3_STATE_BUCKET="pulumi-state-poc-${ACCOUNT_ID}"
KMS_KEY_ALIAS="alias/pulumi-poc-key"
OIDC_ARN="arn:aws:iam::${ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"

echo "=================================================="
echo "AWS Account ID:     $ACCOUNT_ID"
echo "Target Repo:        $GITHUB_ORG_OR_USER/$GITHUB_REPO_NAME"
echo "OIDC sub prefix:    $SUB_PREFIX"
echo "S3 State Bucket:    $S3_STATE_BUCKET"
echo "KMS Alias:          $KMS_KEY_ALIAS"
echo "=================================================="

# ------------------------------------------------------------------------------
# BƯỚC 1: KIỂM TRA GITHUB OIDC PROVIDER
# ------------------------------------------------------------------------------
if aws iam get-open-id-connect-provider --open-id-connect-provider-arn "$OIDC_ARN" >/dev/null 2>&1; then
    echo "[OK] GitHub OIDC Provider đã tồn tại: $OIDC_ARN"
else
    echo "[+] Đang tạo mới GitHub OIDC Provider..."
    aws iam create-open-id-connect-provider \
        --url "https://token.actions.githubusercontent.com" \
        --client-id-list "sts.amazonaws.com" \
        --thumbprint-list "6938fd4d98bab03faadb97b34396831e3780aea1" \
        --thumbprint-list "1c587693c4e5d36e31532d818b6ba772d0425026"
    echo "[OK] Đã tạo thành công OIDC Provider!"
fi

# ------------------------------------------------------------------------------
# HÀM TẠO HOẶC CẬP NHẬT IAM ROLE KÈM TRUST POLICY
# ------------------------------------------------------------------------------
create_or_update_role() {
    local role_name="$1"
    local sub_conditions="$2"
    local desc="$3"

    local tmp_file
    tmp_file=$(mktemp)

    cat <<EOF > "$tmp_file"
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "${OIDC_ARN}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
          "token.actions.githubusercontent.com:sub": ${sub_conditions}
        }
      }
    }
  ]
}
EOF

    if aws iam get-role --role-name "$role_name" >/dev/null 2>&1; then
        echo "[!] Role $role_name đã tồn tại, cập nhật Trust Policy..."
        aws iam update-assume-role-policy \
            --role-name "$role_name" \
            --policy-document "file://$tmp_file"
    else
        echo "[+] Đang tạo mới IAM Role: $role_name..."
        aws iam create-role \
            --role-name "$role_name" \
            --assume-role-policy-document "file://$tmp_file" \
            --description "$desc"
    fi
    rm -f "$tmp_file"
}

# ------------------------------------------------------------------------------
# BƯỚC 2: TẠO POLICY DÀNH CHO PREVIEW (READ-ONLY)
# ------------------------------------------------------------------------------
# Preview: Chi doc metadata AWS, doc S3 state checkpoint, giai ma KMS de preview
PREVIEW_POLICY_NAME="poc-ci-preview-policy"
cat <<EOF > /tmp/preview-policy.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3ReadState",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::${S3_STATE_BUCKET}",
        "arn:aws:s3:::${S3_STATE_BUCKET}/*"
      ]
    },
    {
      "Sid": "AllowKmsDecryptForPreview",
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt",
        "kms:DescribeKey"
      ],
      "Resource": "*"
    }
  ]
}
EOF

if aws iam get-policy --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${PREVIEW_POLICY_NAME}" >/dev/null 2>&1; then
    echo "[OK] Policy $PREVIEW_POLICY_NAME đã tồn tại."
else
    echo "[+] Đang tạo Policy: $PREVIEW_POLICY_NAME..."
    aws iam create-policy \
        --policy-name "$PREVIEW_POLICY_NAME" \
        --policy-document file:///tmp/preview-policy.json
fi
rm -f /tmp/preview-policy.json

# ------------------------------------------------------------------------------
# BƯỚC 3: KHỞI TẠO 6 ROLES THEO ĐÚNG PROPOSAL
# ------------------------------------------------------------------------------
# Job deploy chạy trong GitHub Environment (dev/staging/prod) nên sub có dạng
# <prefix>:environment:<env> thay vì :ref:refs/heads/<branch>.
# Role deploy chỉ tin tưởng environment tương ứng.
# Role preview (read-only) tin tưởng PR và push vào đúng nhánh của nó (job preview trước khi deploy).

# 1. Dev Deploy Role (Chỉ tin tưởng environment dev)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-dev-deploy-role" \
    "[\"${SUB_PREFIX}:environment:dev\"]" \
    "CI Deploy role cho moi truong dev"
aws iam attach-role-policy --role-name "poc-dev-deploy-role" --policy-arn "arn:aws:iam::aws:policy/AdministratorAccess"

# 1b. Dev Preview Role (Read-only, tin tưởng PR và push vào dev)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-dev-preview-role" \
    "[\"${SUB_PREFIX}:pull_request\", \"${SUB_PREFIX}:ref:refs/heads/dev\"]" \
    "CI Preview role cho Dev (Read-Only)"
aws iam attach-role-policy --role-name "poc-dev-preview-role" --policy-arn "arn:aws:iam::aws:policy/ReadOnlyAccess"
aws iam attach-role-policy --role-name "poc-dev-preview-role" --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${PREVIEW_POLICY_NAME}"

# 2. Staging Preview Role (Read-only, tin tưởng PR và push vào staging)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-staging-preview-role" \
    "[\"${SUB_PREFIX}:pull_request\", \"${SUB_PREFIX}:ref:refs/heads/staging\"]" \
    "CI Preview role cho Staging (Read-Only)"
aws iam attach-role-policy --role-name "poc-staging-preview-role" --policy-arn "arn:aws:iam::aws:policy/ReadOnlyAccess"
aws iam attach-role-policy --role-name "poc-staging-preview-role" --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${PREVIEW_POLICY_NAME}"

# 3. Staging Deploy Role (Chỉ tin tưởng environment staging)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-staging-deploy-role" \
    "[\"${SUB_PREFIX}:environment:staging\"]" \
    "CI Deploy role cho moi truong Staging"
aws iam attach-role-policy --role-name "poc-staging-deploy-role" --policy-arn "arn:aws:iam::aws:policy/AdministratorAccess"

# 4. Prod Preview Role (Read-only, tin tưởng PR và push vào main)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-prod-preview-role" \
    "[\"${SUB_PREFIX}:pull_request\", \"${SUB_PREFIX}:ref:refs/heads/main\"]" \
    "CI Preview role cho Production (Read-Only)"
aws iam attach-role-policy --role-name "poc-prod-preview-role" --policy-arn "arn:aws:iam::aws:policy/ReadOnlyAccess"
aws iam attach-role-policy --role-name "poc-prod-preview-role" --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${PREVIEW_POLICY_NAME}"

# 5. Prod Deploy Role (Chỉ tin tưởng environment prod, có approval)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-prod-deploy-role" \
    "[\"${SUB_PREFIX}:environment:prod\"]" \
    "CI Deploy role cho moi truong Production"
aws iam attach-role-policy --role-name "poc-prod-deploy-role" --policy-arn "arn:aws:iam::aws:policy/AdministratorAccess"

echo "=================================================="
echo " HOÀN TẤT THÀNH CÔNG 6 ROLES!"
echo " Danh sách Secret cần cấu hình trên GitHub Repo:"
echo "--------------------------------------------------"
echo " AWS_ROLE_DEV_DEPLOY:      arn:aws:iam::${ACCOUNT_ID}:role/poc-dev-deploy-role"
echo " AWS_ROLE_DEV_PREVIEW:     arn:aws:iam::${ACCOUNT_ID}:role/poc-dev-preview-role"
echo " AWS_ROLE_STAGING_PREVIEW: arn:aws:iam::${ACCOUNT_ID}:role/poc-staging-preview-role"
echo " AWS_ROLE_STAGING_DEPLOY:  arn:aws:iam::${ACCOUNT_ID}:role/poc-staging-deploy-role"
echo " AWS_ROLE_PROD_PREVIEW:    arn:aws:iam::${ACCOUNT_ID}:role/poc-prod-preview-role"
echo " AWS_ROLE_PROD_DEPLOY:     arn:aws:iam::${ACCOUNT_ID}:role/poc-prod-deploy-role"
echo "=================================================="
