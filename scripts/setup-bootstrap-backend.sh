#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# TAO BUCKET LUU STATE CUA PROJECT BOOTSTRAP (infra/bootstrap)
# ==============================================================================
# Day la resource DUY NHAT tao tay: bootstrap khong the luu state trong bucket ma chinh no tao.
# Moi thu con lai (bucket state workload, KMS, ECR, OIDC, role CI) do Pulumi bootstrap quan ly.
# Chay 1 lan bang credential admin. Chay lai an toan: bucket da co thi chi kiem tra cau hinh.

REGION="ap-southeast-1"
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
BUCKET="pulumi-bootstrap-poc-${ACCOUNT_ID}"

echo "Bucket: ${BUCKET} (region ${REGION})"

# 1. Tao bucket neu chua co
if aws s3api head-bucket --bucket "$BUCKET" 2>/dev/null; then
    echo "[OK] Bucket da ton tai"
else
    echo "[+] Tao bucket"
    aws s3api create-bucket \
        --bucket "$BUCKET" \
        --region "$REGION" \
        --create-bucket-configuration LocationConstraint="$REGION"
fi

# 2. Versioning: khoi phuc duoc ban state cu
aws s3api put-bucket-versioning \
    --bucket "$BUCKET" \
    --versioning-configuration Status=Enabled

# 3. Chan moi duong public
aws s3api put-public-access-block \
    --bucket "$BUCKET" \
    --public-access-block-configuration \
    BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true

echo "[OK] Xong. Buoc tiep theo: docs/BOOTSTRAP.md"
echo "  cd infra/bootstrap"
echo "  pulumi login \"s3://${BUCKET}?region=${REGION}\""
