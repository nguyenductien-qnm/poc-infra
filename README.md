# infra-poc: Pulumi (Go) + GitOps trên AWS

POC chứng minh cách tổ chức IaC bằng Pulumi Go: **một codebase, 3 stack** (`dev`, `staging`, `prod`), mỗi stack chỉ khác file `Pulumi.<stack>.yaml`. Deploy qua GitHub Actions với OIDC, preview bằng role read-only. Thiết kế có cổng approve (GitHub Environment *Required reviewers*) trước `up` staging/prod, nhưng **hiện chưa bật**, xem [docs/OPERATIONS.md](docs/OPERATIONS.md#approval-chưa-bật).

| Tài liệu | Dành cho | Nội dung |
| -------- | -------- | -------- |
| [docs/SETUP.md](docs/SETUP.md) | Người dựng POC | Từ account trống tới deploy đầu: bootstrap, GitHub, stack workload |
| [docs/OPERATIONS.md](docs/OPERATIONS.md) | Người vận hành | Workflow CI, role CI, bootstrap, sự cố, dọn dẹp |
| [docs/DECISIONS.md](docs/DECISIONS.md) | Reviewer, mentor | Vì sao thiết kế như vậy, khác production ở đâu |
| [docs/POC-RESULTS.md](docs/POC-RESULTS.md) | Mentor | Kịch bản chứng minh, tiêu chí đạt, số liệu |

## Kiến trúc

```
.
├── app/                    Web app Go (ghi note vào Postgres), build thành image ECR poc-app
├── infra/                  Một Go module, hai Pulumi project
│   ├── bootstrap/          Project infra-bootstrap: nền cho workload (admin setup tay một lần, sau đó qua bootstrap.yml)
│   │   ├── main.go
│   │   ├── Pulumi.poc.yaml Config 1 stack cho 1 account
│   │   └── internal/       Chỉ bootstrap import được (Go chặn lúc build)
│   │       ├── statebackend/  S3 state workload, KMS key mã hoá secret
│   │       ├── registry/      ECR poc-app
│   │       └── ciiam/         GitHub OIDC provider, role preview/deploy/bootstrap
│   ├── workload/           Project infra-poc: ứng dụng, CI deploy
│   │   ├── main.go         Đọc config stack, ghép 4 package
│   │   └── Pulumi.<stack>.yaml Config từng môi trường
│   └── internal/           Package dùng lại được
│       ├── network/        VPC, 2 public + 2 private subnet, S3 Gateway Endpoint
│       ├── platform/       ECS Cluster, ALB (HTTP), Security Group ALB/App, CloudWatch Logs
│       ├── data/           RDS PostgreSQL 16 (password trong Secrets Manager), EFS
│       ├── app/            IAM exec/task role, Task Definition, ECS Fargate Service
│       └── shared/         Hằng số, helper Tags/egress SG, kiểm tra account
├── .github/workflows/
│   ├── pr-check.yml        Workload: preview trên PR
│   ├── deploy.yml          Workload: preview rồi up (approval qua Environment, chưa bật)
│   ├── bootstrap.yml       Bootstrap: preview/up, chạy tay, cần reviewer duyệt (đã bật)
│   └── pulumi-pack.yml     Kiểm tra Pulumi skill pack (không đụng AWS)
├── .agents/skills/pulumi/  Pulumi Go skill (nguồn chuẩn); .claude/ và .codex/ là adapter
├── docs/                   SETUP, OPERATIONS, DECISIONS, POC-RESULTS
└── scripts/
    ├── setup-bootstrap-backend.sh  Tạo bucket lưu state của bootstrap (resource tạo tay duy nhất)
    └── verify-pulumi-pack.py, test_pulumi_pack.py  Kiểm tra skill pack
```

Mỗi package trong `internal/` (trừ `shared`) là một Pulumi **ComponentResource** (`infra-poc:<package>:<Tên>`), nên `pulumi preview` và console nhóm resource theo module. Lý do tách bootstrap và workload xem [DECISIONS.md](docs/DECISIONS.md#3-tách-bootstrap-khỏi-workload).

Luồng truy cập: Internet → ALB (port 80) → Fargate task (public subnet) → RDS (private subnet, chỉ nhận từ App SG).

## Config theo stack

| Config               | dev          | staging      | prod         |
| -------------------- | ------------ | ------------ | ------------ |
| `expectedAccount`    | secret       | secret       | secret       |
| `vpcCidr`            | 10.10.0.0/16 | 10.20.0.0/16 | 10.30.0.0/16 |
| `desiredCount`       | 1            | 1            | 2            |
| `dbInstanceClass`    | db.t4g.micro | db.t4g.micro | db.t4g.small |
| `protectStateful`    | false        | false        | false        |
| `deletionProtection` | false        | false        | false        |
| `ecrRepositoryName`  | poc-app      | poc-app      | poc-app      |

- `expectedAccount`: account AWS mà stack được phép chạy, lưu dạng secret (mã hoá KMS) để không lộ account ID. Credentials thuộc account khác thì preview/up dừng ngay. Set bằng `pulumi config set --secret expectedAccount <account-id>`.
- `imageTag` do CI set theo Git SHA (7 ký tự) mỗi lần deploy, không lưu trong `Pulumi.<stack>.yaml`. Thiếu hoặc bằng `latest` thì preview/up báo lỗi.
- `ecrRepositoryName`: repo ECR dùng chung, do bootstrap quản lý.
- `protectStateful`: bật thì RDS/EFS có `pulumi.Protect`, RDS giữ final snapshot khi xoá. `deletionProtection`: cờ bảo vệ xoá phía AWS của RDS. Hiện cả 3 stack đều `false`, kể cả prod; đặt `true` cho prod trước khi chạy kịch bản bảo vệ ở [docs/POC-RESULTS.md](docs/POC-RESULTS.md).

## Chạy nhanh

Local chỉ `preview` workload; mọi `up` đi qua CI. Chi tiết và các workflow xem [docs/OPERATIONS.md](docs/OPERATIONS.md).

```bash
cd infra/workload
pulumi login "s3://pulumi-state-poc-<account>?region=ap-southeast-1"
pulumi stack select dev
pulumi config set imageTag "$(pulumi stack output imageTag)"
pulumi preview
```

> `pr-check.yml` và `deploy.yml` đang tạm chỉ chạy tay (`workflow_dispatch`); cách bật lại xem [docs/OPERATIONS.md](docs/OPERATIONS.md#workflow-ci).

## Khác production thật

Fargate ở public subnet, ALB chỉ HTTP, RDS single-AZ, role deploy là Admin, chưa bật approval cho `up`, PR preview chạy code chưa review bằng role read-only. Danh sách đầy đủ kèm lý do: [docs/DECISIONS.md](docs/DECISIONS.md#9-chấp-nhận-khác-production-thật).

## Pulumi Go infrastructure skill

The standalone [Pulumi Go skill](.agents/skills/pulumi/SKILL.md) covers infrastructure composition, state, identities, workload configuration and GitOps delivery. Its complete distributable folder includes instructions, read-only reviewer contracts, scripts/tests and optional Codex/Claude adapter templates. It does not depend on this repository's infrastructure, documents or history.

See the [pack setup guide](.agents/skills/pulumi/README.md) for standalone checks, project integration, hook trust and limitations. Application implementation, build/tests and container builds remain outside the skill.
