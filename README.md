# infra-poc: Pulumi (Go) + GitOps trên AWS

POC chứng minh cách tổ chức IaC bằng Pulumi Go: **một codebase, 3 stack** (`dev`, `staging`, `prod`), mỗi stack chỉ khác file `Pulumi.<stack>.yaml`. Kế hoạch chi tiết xem [docs/PLAN.md](docs/PLAN.md), hướng dẫn cài đặt xem [docs/SETUP.md](docs/SETUP.md), đề xuất gốc xem [docs/pulumi-proposal.md](docs/pulumi-proposal.md).

## Kiến trúc

```
.
├── app/                    Web app Go (ghi note vào Postgres), build thành image ECR poc-app
├── infra/
│   ├── main.go             Đọc config stack, ghép 4 package
│   ├── Pulumi.<stack>.yaml Config từng môi trường
│   └── internal/
│       ├── network/        VPC, 2 public + 2 private subnet, S3 Gateway Endpoint
│       ├── platform/       ECS Cluster, ALB (HTTP), Security Group ALB/App, CloudWatch Logs
│       ├── data/           RDS PostgreSQL 16 (password trong Secrets Manager), EFS
│       ├── app/            IAM exec/task role, Task Definition, ECS Fargate Service
│       └── shared/         Hằng số (port, DB name/user), helper Tags, egress SG
├── .github/workflows/      pr-check.yml (preview), deploy.yml (up)
├── docs/                   PLAN, SETUP, proposal
└── scripts/
    └── setup-ci-roles.sh   Tạo OIDC provider + 6 IAM role cho CI
```

Mỗi package trong `internal/` (trừ `shared`) là một Pulumi **ComponentResource** (`infra-poc:<package>:<Tên>`), mọi resource AWS là con của component đó, nên `pulumi preview` và console nhóm resource theo module.

Luồng truy cập: Internet → ALB (port 80) → Fargate task (public subnet) → RDS (private subnet, chỉ nhận từ App SG).

## Config theo stack

| Config               | dev          | staging      | prod         |
| -------------------- | ------------ | ------------ | ------------ |
| `vpcCidr`            | 10.10.0.0/16 | 10.20.0.0/16 | 10.30.0.0/16 |
| `desiredCount`       | 1            | 1            | 2            |
| `dbInstanceClass`    | db.t4g.micro | db.t4g.micro | db.t4g.small |
| `protectStateful`    | false        | false        | true         |
| `deletionProtection` | false        | false        | true         |
| `enableTgw`          | false        | false        | false        |

`imageTag` do CI set theo Git SHA (7 ký tự) mỗi lần deploy, không lưu trong `Pulumi.<stack>.yaml`. `enableTgw` mới là chỗ để sẵn, chưa có code dùng.

## CI/CD

- **PR** vào `dev` / `staging` / `main`: kiểm tra `go mod tidy`, build app, `go vet` + `go test` infra, rồi `pulumi preview --diff` bằng role **read-only** (giữ `imageTag` đang chạy để diff chỉ phản ánh hạ tầng). Kết quả cập nhật vào 1 comment trên PR; preview lỗi thì job fail.
- **Push** vào `dev` / `staging` / `main` (bỏ qua thay đổi chỉ ở `*.md`, `docs/`), 2 job nối tiếp:
  1. `preview`: role read-only, preview với image tag mới, diff in ra **Job Summary**.
  2. `deploy`: chạy trong GitHub Environment cùng tên (`main` → `prod`). Staging/prod có *Required reviewers*, người duyệt xem diff ở job `preview` rồi mới approve. Sau đó build & push image (chỉ tag Git SHA, không dùng `:latest`) và `pulumi up`.
- Xác thực AWS bằng **OIDC**, không lưu access key. State trên S3 `pulumi-state-poc-<account>`, secret mã hóa bằng KMS `alias/pulumi-poc-key`.

### IAM role cho CI

Trust policy dùng `StringEquals` với OIDC `sub` dạng immutable (repo tạo sau 15/07/2026):
`repo:<owner>@<owner_id>/<repo>@<repo_id>:...`

| Role                       | Tin tưởng `sub`         | Quyền                      | GitHub secret              |
| -------------------------- | ----------------------- | -------------------------- | -------------------------- |
| `poc-dev-preview-role`     | `pull_request`, ref dev | ReadOnly + state/KMS       | `AWS_ROLE_DEV_PREVIEW`     |
| `poc-dev-deploy-role`      | `environment:dev`       | Admin                      | `AWS_ROLE_DEV_DEPLOY`      |
| `poc-staging-preview-role` | `pull_request`, ref staging | ReadOnly + state/KMS   | `AWS_ROLE_STAGING_PREVIEW` |
| `poc-staging-deploy-role`  | `environment:staging`   | Admin                      | `AWS_ROLE_STAGING_DEPLOY`  |
| `poc-prod-preview-role`    | `pull_request`, ref main | ReadOnly + state/KMS      | `AWS_ROLE_PROD_PREVIEW`    |
| `poc-prod-deploy-role`     | `environment:prod`      | Admin                      | `AWS_ROLE_PROD_DEPLOY`     |

```bash
bash scripts/setup-ci-roles.sh [github-owner] [repo-name]
# Repo private: GITHUB_OWNER_ID=... GITHUB_REPO_ID=... bash scripts/setup-ci-roles.sh
```

## Chạy local

```bash
cd infra
pulumi login "s3://pulumi-state-poc-<account>?region=ap-southeast-1"
pulumi stack select dev
# Dùng image đang chạy, nếu không preview sẽ đổi image về :latest
pulumi config set imageTag "$(pulumi stack output imageTag)"
pulumi preview
go test ./...   # unit test (tính CIDR subnet)
```

Quy ước: local chỉ chạy `preview`, mọi `up` đi qua CI. Quy ước này chưa được ép bằng IAM: credential admin trên máy vẫn `up` được; muốn ép thì dev dùng role read-only (kịch bản `AccessDenied` ở docs/PLAN.md Phase 6).

## Khác biệt so với production thật

- Fargate task chạy ở **public subnet** có public IP thay vì private subnet + NAT/TGW (`enableTgw=false`).
- ALB chỉ **HTTP**, chưa có ACM/HTTPS (cần domain).
- RDS **single-AZ**, chưa có AWS Backup.
- Role deploy dùng `AdministratorAccess` cho nhanh; prod thật cần thu hẹp quyền.
- Task role có `bedrock:InvokeModel` để chứng minh IAM, app không gọi Bedrock.

## Dọn dẹp

```bash
cd infra
pulumi destroy -s staging && pulumi destroy -s dev
# prod: tắt protectStateful/deletionProtection, pulumi up, rồi mới destroy
aws elbv2 describe-load-balancers   # kiểm tra tài nguyên sót
aws rds describe-db-instances
# ECR poc-app, S3 state bucket, KMS key, IAM role CI tạo ngoài stack: xóa tay nếu không dùng nữa
```
