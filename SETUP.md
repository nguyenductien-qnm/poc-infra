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
- [internal/network/network.go](file:///Users/ductiennguyen/Downloads/pulumi/internal/network/network.go)
- [internal/data/data.go](file:///Users/ductiennguyen/Downloads/pulumi/internal/data/data.go)
- [internal/platform/platform.go](file:///Users/ductiennguyen/Downloads/pulumi/internal/platform/platform.go)
- [internal/app/app.go](file:///Users/ductiennguyen/Downloads/pulumi/internal/app/app.go)
- [main.go](file:///Users/ductiennguyen/Downloads/pulumi/main.go)

### Bước 9: Kiểm tra compile & preview
```bash
go mod tidy
go build -o /dev/null .
pulumi preview
```
*Kết quả:* Compile thành công, preview tạo stack `infra-poc-dev` với 0 tài nguyên AWS.

---

### Bước 10: Viết mã nguồn triển khai thực tế cho `network` & `platform`
- [internal/network/network.go](file:///Users/ductiennguyen/Downloads/pulumi/internal/network/network.go): VPC, Internet Gateway, 2 Public Subnets, 2 Private Subnets, Route Tables & S3 Gateway Endpoint.
- [internal/platform/platform.go](file:///Users/ductiennguyen/Downloads/pulumi/internal/platform/platform.go): ECS Cluster, ECR Repo, CloudWatch Log Group, ALB HTTP, Target Group (ip type cho Fargate), Security Group.
- [main.go](file:///Users/ductiennguyen/Downloads/pulumi/main.go): Kết nối output `network` sang `platform` và export thông tin.

### Bước 11: Kiểm tra preview Phase 2
```bash
go mod tidy
go build -o /dev/null .
pulumi preview
```
*Kết quả:* Preview thành công **+21 resources** (VPC, Subnets, IGW, RouteTables, S3 Endpoint, ECS Cluster, ECR, ALB, TargetGroup, Listener, SecurityGroup, Logs).

---

## 3. Các bước tiếp theo
- **Lựa chọn 1**: Chạy `pulumi up` ngay trên stack `dev` để tạo thật 21 tài nguyên trên AWS (kiểm tra bằng `aws ec2 describe-vpcs` và ALB DNS).
- **Lựa chọn 2**: Viết tiếp code **Phase 3 (`data` + `app`)** gồm RDS PostgreSQL, EFS, ECS Fargate Service (Nginx) rồi `pulumi up` một thể.



