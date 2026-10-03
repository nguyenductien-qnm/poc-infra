# infra-poc: Pulumi (Go) + GitOps trên AWS

POC chứng minh cách tổ chức IaC bằng Pulumi Go: **một codebase, 3 stack** (`dev`, `staging`, `prod`), mỗi stack chỉ khác file `Pulumi.<stack>.yaml`. Kế hoạch chi tiết xem [docs/PLAN.md](docs/PLAN.md), hướng dẫn cài đặt xem [docs/SETUP.md](docs/SETUP.md), đề xuất gốc xem [docs/pulumi-proposal.md](docs/pulumi-proposal.md).

## Kiến trúc

```
.
├── app/                    Web app Go (ghi note vào Postgres), build thành image ECR poc-app
├── infra/                  Một Go module, hai Pulumi project
│   ├── bootstrap/          Project infra-bootstrap: nền cho workload, admin chạy tay (docs/BOOTSTRAP.md)
│   │   ├── main.go
│   │   ├── Pulumi.poc.yaml Config 1 stack cho 1 account
│   │   └── internal/       Chỉ bootstrap import được (Go chặn lúc build)
│   │       ├── statebackend/  S3 state workload, KMS key mã hoá secret
│   │       ├── registry/      ECR poc-app
│   │       └── ciiam/         GitHub OIDC provider, role preview/deploy
│   ├── workload/           Project infra-poc: ứng dụng, CI deploy
│   │   ├── main.go         Đọc config stack, ghép 4 package
│   │   └── Pulumi.<stack>.yaml Config từng môi trường
│   └── internal/           Package dùng lại được
│       ├── network/        VPC, 2 public + 2 private subnet, S3 Gateway Endpoint
│       ├── platform/       ECS Cluster, ALB (HTTP), Security Group ALB/App, CloudWatch Logs
│       ├── data/           RDS PostgreSQL 16 (password trong Secrets Manager), EFS
│       ├── app/            IAM exec/task role, Task Definition, ECS Fargate Service
│       └── shared/         Hằng số, helper Tags/egress SG, kiểm tra account
├── .github/workflows/      pr-check.yml (preview), deploy.yml (up) cho workload
├── docs/                   PLAN, SETUP, BOOTSTRAP, proposal
└── scripts/
    └── setup-bootstrap-backend.sh  Tạo bucket lưu state của bootstrap (resource tạo tay duy nhất)
```

Mỗi package trong `internal/` (trừ `shared`) là một Pulumi **ComponentResource** (`infra-poc:<package>:<Tên>`), mọi resource AWS là con của component đó, nên `pulumi preview` và console nhóm resource theo module.

Hai project tách vì khác vòng đời và quyền: workload đổi thường xuyên, deploy qua CI; bootstrap hiếm đổi, chỉ admin chạy, nên pipeline workload không sửa được role CI hay xoá state/ECR. State workload ở bucket `pulumi-state-poc-<account>`, state bootstrap ở bucket riêng `pulumi-bootstrap-poc-<account>`.

Luồng truy cập: Internet → ALB (port 80) → Fargate task (public subnet) → RDS (private subnet, chỉ nhận từ App SG).

## Config theo stack

| Config               | dev          | staging      | prod         |
| -------------------- | ------------ | ------------ | ------------ |
| `expectedAccount`    | secret       | secret       | secret       |
| `vpcCidr`            | 10.10.0.0/16 | 10.20.0.0/16 | 10.30.0.0/16 |
| `desiredCount`       | 1            | 1            | 2            |
| `dbInstanceClass`    | db.t4g.micro | db.t4g.micro | db.t4g.small |
| `protectStateful`    | false        | false        | true         |
| `deletionProtection` | false        | false        | true         |
| `ecrRepositoryName`  | poc-app      | poc-app      | poc-app      |

- `expectedAccount`: account AWS mà stack được phép chạy, lưu dạng secret (mã hoá KMS) để không lộ account ID. Credentials thuộc account khác thì preview/up dừng ngay, chưa tạo gì. Set bằng `pulumi config set --secret expectedAccount <account-id>`.
- `imageTag` do CI set theo Git SHA (7 ký tự) mỗi lần deploy, không lưu trong `Pulumi.<stack>.yaml`. Thiếu hoặc bằng `latest` thì preview/up báo lỗi, không tự dùng `:latest`.
- `ecrRepositoryName`: repo ECR dùng chung, do bootstrap quản lý.

## CI/CD

> **Đang tạm tắt trigger tự động** (chỉ chạy tay bằng `workflow_dispatch`) trong lúc chuyển sang bootstrap/workload. Bật lại sau khi `up` bootstrap theo [docs/BOOTSTRAP.md](docs/BOOTSTRAP.md): bỏ comment khối `pull_request` / `push` trong `.github/workflows/*.yml`. Mô tả dưới đây là hành vi khi bật.

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

OIDC provider, các role và policy preview do project bootstrap quản lý (package `ciiam`), thay cho script `setup-ci-roles.sh` cũ. Đổi quyền CI: sửa code, mở PR, admin preview/up theo [docs/BOOTSTRAP.md](docs/BOOTSTRAP.md). ARN role cho GitHub secret lấy từ output `previewRoleArns` / `deployRoleArns` của stack bootstrap.

## Chạy local

```bash
cd infra/workload
pulumi login "s3://pulumi-state-poc-<account>?region=ap-southeast-1"
pulumi stack select dev
# Bắt buộc: dùng image đang chạy (thiếu imageTag thì preview báo lỗi)
pulumi config set imageTag "$(pulumi stack output imageTag)"
pulumi preview
```

Quy ước: local chỉ chạy `preview` workload, mọi `up` workload đi qua CI. Bootstrap là ngoại lệ: admin chạy tay theo [docs/BOOTSTRAP.md](docs/BOOTSTRAP.md). Quy ước này chưa được ép bằng IAM: credential admin trên máy vẫn `up` được; muốn ép thì dev dùng role read-only (kịch bản `AccessDenied` ở docs/PLAN.md Phase 6).

## Khác biệt so với production thật

- Fargate task chạy ở **public subnet** có public IP thay vì private subnet + NAT/TGW.
- ALB chỉ **HTTP**, chưa có ACM/HTTPS (cần domain).
- RDS **single-AZ**, chưa có AWS Backup.
- Role deploy dùng `AdministratorAccess` cho nhanh; prod thật cần thu hẹp quyền.
- Task role có `bedrock:InvokeModel` để chứng minh IAM, app không gọi Bedrock.
- PR preview chạy code của PR (chưa review) bằng role preview có `ReadOnlyAccess` + `kms:Decrypt`. Chấp nhận cho POC để có diff ngay ở PR: chỉ người trong team mở PR, PR từ fork không được GitHub cấp OIDC token. Production nên chặn bằng approval trước khi job preview nhận credentials.
- Action trong workflow pin theo commit SHA nhưng vẫn ở major cũ (`checkout@v4`, `setup-go@v5`...). Nâng major làm riêng.

## Dọn dẹp

```bash
cd infra/workload
pulumi destroy -s staging && pulumi destroy -s dev
# prod: tắt protectStateful/deletionProtection, pulumi up, rồi mới destroy
aws elbv2 describe-load-balancers   # kiểm tra tài nguyên sót
aws rds describe-db-instances
```

Destroy workload không đụng tới bootstrap. Bucket state, KMS key, ECR được đánh dấu giữ lại khi destroy bootstrap; muốn xoá hẳn phải gỡ `protect` và xoá tay có chủ đích.
