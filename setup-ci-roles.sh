#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# SCRIPT KHỞI TẠO 5 IAM ROLES CHO CI/CD (PHÂN TÁCH PREVIEW & DEPLOY THEO PROPOSAL)
# ==============================================================================

GITHUB_ORG_OR_USER="${1:-nguyenductien-qnm}"   # Tham so 1: Username hoac Organization GitHub
GITHUB_REPO_NAME="${2:-poc-infra}"           # Tham so 2: Ten Repository GitHub

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION="ap-southeast-1"
S3_STATE_BUCKET="pulumi-state-poc-${ACCOUNT_ID}"
KMS_KEY_ALIAS="alias/pulumi-poc-key"
OIDC_ARN="arn:aws:iam::${ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"

echo "=================================================="
echo "AWS Account ID:     $ACCOUNT_ID"
echo "Target Repo:        $GITHUB_ORG_OR_USER/$GITHUB_REPO_NAME"
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
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
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
# BƯỚC 3: KHỞI TẠO 5 ROLES THEO ĐÚNG PROPOSAL
# ------------------------------------------------------------------------------

# 1. Dev Deploy Role (Tin tưởng branch dev và PR)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-dev-deploy-role" \
    "[\"repo:${GITHUB_ORG_OR_USER}*/${GITHUB_REPO_NAME}*:ref:refs/heads/dev\", \"repo:${GITHUB_ORG_OR_USER}*/${GITHUB_REPO_NAME}*:pull_request*\"]" \
    "CI Deploy role cho moi truong dev"
aws iam attach-role-policy --role-name "poc-dev-deploy-role" --policy-arn "arn:aws:iam::aws:policy/AdministratorAccess"

# 2. Staging Preview Role (Read-only, tin tưởng PR vào staging)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-staging-preview-role" \
    "[\"repo:${GITHUB_ORG_OR_USER}*/${GITHUB_REPO_NAME}*:pull_request*\", \"repo:${GITHUB_ORG_OR_USER}*/${GITHUB_REPO_NAME}*:ref:refs/heads/dev\"]" \
    "CI Preview role cho Staging (Read-Only)"
aws iam attach-role-policy --role-name "poc-staging-preview-role" --policy-arn "arn:aws:iam::aws:policy/ReadOnlyAccess"
aws iam attach-role-policy --role-name "poc-staging-preview-role" --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${PREVIEW_POLICY_NAME}"

# 3. Staging Deploy Role (Chỉ tin tưởng branch staging khi đã merge)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-staging-deploy-role" \
    "[\"repo:${GITHUB_ORG_OR_USER}*/${GITHUB_REPO_NAME}*:ref:refs/heads/staging\"]" \
    "CI Deploy role cho moi truong Staging"
aws iam attach-role-policy --role-name "poc-staging-deploy-role" --policy-arn "arn:aws:iam::aws:policy/AdministratorAccess"

# 4. Prod Preview Role (Read-only, tin tưởng PR vào main)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-prod-preview-role" \
    "[\"repo:${GITHUB_ORG_OR_USER}*/${GITHUB_REPO_NAME}*:pull_request*\", \"repo:${GITHUB_ORG_OR_USER}*/${GITHUB_REPO_NAME}*:ref:refs/heads/staging\"]" \
    "CI Preview role cho Production (Read-Only)"
aws iam attach-role-policy --role-name "poc-prod-preview-role" --policy-arn "arn:aws:iam::aws:policy/ReadOnlyAccess"
aws iam attach-role-policy --role-name "poc-prod-preview-role" --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${PREVIEW_POLICY_NAME}"

# 5. Prod Deploy Role (Chỉ tin tưởng branch main)
echo "--------------------------------------------------"
create_or_update_role \
    "poc-prod-deploy-role" \
    "[\"repo:${GITHUB_ORG_OR_USER}*/${GITHUB_REPO_NAME}*:ref:refs/heads/main\"]" \
    "CI Deploy role cho moi truong Production"
aws iam attach-role-policy --role-name "poc-prod-deploy-role" --policy-arn "arn:aws:iam::aws:policy/AdministratorAccess"

echo "=================================================="
echo " HOÀN TẤT THÀNH CÔNG 5 ROLES!"
echo " Danh sách Secret cần cấu hình trên GitHub Repo:"
echo "--------------------------------------------------"
echo " AWS_ROLE_DEV_DEPLOY:      arn:aws:iam::${ACCOUNT_ID}:role/poc-dev-deploy-role"
echo " AWS_ROLE_STAGING_PREVIEW: arn:aws:iam::${ACCOUNT_ID}:role/poc-staging-preview-role"
echo " AWS_ROLE_STAGING_DEPLOY:  arn:aws:iam::${ACCOUNT_ID}:role/poc-staging-deploy-role"
echo " AWS_ROLE_PROD_PREVIEW:    arn:aws:iam::${ACCOUNT_ID}:role/poc-prod-preview-role"
echo " AWS_ROLE_PROD_DEPLOY:     arn:aws:iam::${ACCOUNT_ID}:role/poc-prod-deploy-role"
echo "=================================================="
