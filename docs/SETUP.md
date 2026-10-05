# Pulumi POC Setup Guide

Các bước đã thực hiện để khởi tạo hạ tầng từ đầu (AWS S3 Backend + AWS KMS Secrets Provider).

---

## 1. Cấu hình ban đầu
- **AWS Region**: `ap-southeast-1`
- **S3 Bucket Backend**: `pulumi-state-poc-730335441285`
- **KMS Alias**: `alias/pulumi-poc-key`
- **Project Name**: `infra-poc`

---

## 2. Các bước thực hiện

### Bước 1: Tạo cây thư mục `internal`
```bash
mkdir -p internal/network internal/data internal/platform internal/app
```

### Bước 2: Tạo S3 Bucket làm Backend & bật Versioning
```bash
# Tạo bucket
aws s3api create-bucket \
  --bucket pulumi-state-poc-730335441285 \
  --region ap-southeast-1 \
  --create-bucket-configuration LocationConstraint=ap-southeast-1

# Bật versioning bảo vệ state file
aws s3api put-bucket-versioning \
  --bucket pulumi-state-poc-730335441285 \
  --versioning-configuration Status=Enabled
```

### Bước 3: Tạo AWS KMS Key & Alias mã hóa Secrets
```bash
# Tạo KMS Key
KEY_ID=$(aws kms create-key \
  --description "Pulumi POC Secrets Encryption Key" \
  --region ap-southeast-1 \
  --query "KeyMetadata.KeyId" \
  --output text)

echo "KMS Key ID: $KEY_ID"

# Gắn alias
aws kms create-alias \
  --alias-name alias/pulumi-poc-key \
  --target-key-id "$KEY_ID" \
  --region ap-southeast-1
```

### Bước 3b: Tạo AWS ECR Repository dùng chung (`poc-app`)
```bash
aws ecr create-repository \
  --repository-name poc-app \
  --image-scanning-configuration scanOnPush=true \
  --region ap-southeast-1
```

### Bước 4: Đăng nhập Pulumi vào S3 Backend
```bash
pulumi login "s3://pulumi-state-poc-730335441285?region=ap-southeast-1"
```

### Bước 5: Khởi tạo Project & Stack `dev`
```bash
pulumi new aws-go -n infra-poc -s dev \
  --secrets-provider="awskms://alias/pulumi-poc-key?region=ap-southeast-1" \
  --force \
  --yes
```

### Bước 6: Khởi tạo thêm 2 Stack `staging` và `prod`
```bash
KMS_URL="awskms://alias/pulumi-poc-key?region=ap-southeast-1"

pulumi stack init staging --secrets-provider="$KMS_URL"
pulumi stack init prod    --secrets-provider="$KMS_URL"
pulumi stack select dev
```

### Bước 7: Cấu hình AWS Region cho stack
```bash
pulumi config set aws:region ap-southeast-1
```

---

### Bước 8: Tạo khung code 4 package `internal/` & liên kết `main.go`
- [infra/internal/network/](../infra/internal/network/module.go)
- [infra/internal/data/](../infra/internal/data/module.go)
- [infra/internal/platform/](../infra/internal/platform/module.go)
- [infra/internal/app/](../infra/internal/app/module.go)
- [infra/main.go](../infra/main.go)

### Bước 9: Kiểm tra compile & preview
```bash
go mod tidy
go build -o /dev/null .
pulumi preview
```
*Kết quả:* Compile thành công, preview tạo stack `infra-poc-dev` với 0 tài nguyên AWS.

---

### Bước 10: Viết mã nguồn triển khai thực tế cho `network` & `platform`
- [infra/internal/network/](../infra/internal/network/module.go): VPC, Internet Gateway, 2 Public Subnets, 2 Private Subnets, Route Tables & S3 Gateway Endpoint.
- [infra/internal/platform/](../infra/internal/platform/module.go): ECS Cluster, CloudWatch Log Group, ALB HTTP, Target Group (ip type cho Fargate), Security Group ALB/App. ECR sau đó chuyển thành repo dùng chung `poc-app` (Bước 3b), không còn tạo trong stack.
- [infra/main.go](../infra/main.go): Kết nối output `network` sang `platform` và export thông tin.

### Bước 11: Kiểm tra preview Phase 2
```bash
go mod tidy
go build -o /dev/null .
pulumi preview
```
*Kết quả:* Preview thành công **+21 resources** (VPC, Subnets, IGW, RouteTables, S3 Endpoint, ECS Cluster, ECR, ALB, TargetGroup, Listener, SecurityGroup, Logs).

---

## 3. Thiết lập CI/CD (GitHub Actions + OIDC)

### Bước 12: Tạo OIDC provider và 6 IAM role cho CI
Ban đầu tạo bằng script `scripts/setup-ci-roles.sh`. **Đã thay** bằng project Pulumi `infra/bootstrap` (package `ciiam`), các role đang có được import vào. Script cũ đổi thành `scripts/setup-bootstrap-backend.sh`, chỉ tạo bucket lưu state của bootstrap. Xem [BOOTSTRAP.md](BOOTSTRAP.md).

Trust policy dùng `StringEquals` với OIDC `sub` dạng immutable `repo:<owner>@<owner_id>/<repo>@<repo_id>:...` (repo tạo sau 15/07/2026). Role deploy tin `environment:<env>`, role preview tin `pull_request`.

### Bước 13: Cấu hình GitHub repo
- **Secrets** (Settings > Secrets and variables > Actions): `AWS_ROLE_DEV_PREVIEW`, `AWS_ROLE_DEV_DEPLOY`, `AWS_ROLE_STAGING_PREVIEW`, `AWS_ROLE_STAGING_DEPLOY`, `AWS_ROLE_PROD_PREVIEW`, `AWS_ROLE_PROD_DEPLOY`, `AWS_ROLE_BOOTSTRAP` (ARN lấy từ output `previewRoleArns` / `deployRoleArns` / `bootstrapRoleArn` của stack bootstrap).
- **Environments** (Settings > Environments): tạo `dev`, `staging`, `prod`. Bật *Required reviewers* cho `staging` và `prod`; giới hạn deployment branch `main` cho `prod`. Tạo thêm `bootstrap` (Required reviewers, deployment branch `main`) cho workflow bootstrap.

### Bước 14: Kiểm tra flow
- Mở PR vào `dev`: job `pr-check` chạy preview, comment diff vào PR.
- Merge vào `dev`: job `preview` in diff ra Job Summary, sau đó job `deploy` build image tag Git SHA và `pulumi up` stack `dev`.
- Push `staging` / `main`: job `preview` chạy trước; job `deploy` chờ approval trên environment (người duyệt xem diff ở Job Summary) rồi mới deploy.
