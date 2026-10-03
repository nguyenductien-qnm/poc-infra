## Khuyến nghị

POC giữ **đúng cấu trúc trong proposal** (một project, 4 package trong `internal/`, stack `dev/staging/prod`), nhưng thu nhỏ tài nguyên để rẻ và chạy nhanh. Mục tiêu POC là chứng minh **cách tổ chức và flow vận hành**, không phải dựng lại toàn bộ product.

## Phạm vi

| Package    | Làm thật                                                                                           | Bỏ / thu nhỏ                                                         | Lý do                                                                |
| ---------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `network`  | VPC, subnet public/private, route table, S3 Gateway Endpoint                                       | **TGW + NAT về Hub** (bật bằng config `enableTgw`, mặc định `false`) | Cần Hub thật; POC dùng public subnet cho task để khỏi tốn NAT        |
| `data`     | RDS PostgreSQL `db.t4g.micro` single-AZ (master password do RDS quản lý, lưu Secrets Manager), EFS | AWS Backup (thêm nếu còn thời gian)                                  | RDS là ví dụ tốt nhất cho `protect`, `deletionProtection`, `replace` |
| `platform` | ECS Cluster, ALB (HTTP), ECR 1 repo, log group                                                     | ACM (cần domain)                                                     | Tránh phụ thuộc domain                                               |
| `app`      | 1 ECS Fargate service chạy image nginx công khai, task role có policy `bedrock:InvokeModel`        | Gọi Bedrock thật                                                     | Chỉ cần chứng minh IAM, không cần gọi model                          |

Phần POC khác prod (public subnet thay NAT, không ACM) phải ghi rõ trong tài liệu nộp mentor.

## Chia môi trường (giữ như proposal)

Cùng một code, khác `Pulumi.<stack>.yaml`. Cả 3 stack dùng chung một account POC, nên tên resource phải có tên stack và CIDR khác nhau.

| Config                     | dev          | staging      | prod         |
| -------------------------- | ------------ | ------------ | ------------ |
| `vpcCidr`                  | 10.10.0.0/16 | 10.20.0.0/16 | 10.30.0.0/16 |
| `desiredCount`             | 1            | 1            | 2            |
| `dbInstanceClass`          | db.t4g.micro | db.t4g.micro | db.t4g.small |
| `protectStateful`          | false        | false        | true         |
| `deletionProtection` (RDS) | false        | false        | true         |
| `enableTgw`                | false        | false        | false        |

Vì tốn tiền theo giờ, gợi ý: **`up` dev và staging, chỉ `preview` prod** để demo cơ chế bảo vệ và flow approval mà không phải dựng prod thật.

## Các phase

**Phase 0: Chuẩn bị (0.5h)**

```bash
pulumi login "s3://<bucket>?region=ap-southeast-1"
mkdir infra-poc && cd infra-poc && go mod init <module-path>
pulumi new aws-go -n infra-poc -s dev \
  --secrets-provider="awskms://alias/<alias>?region=ap-southeast-1"
pulumi stack init staging --secrets-provider="awskms://alias/<alias>?region=ap-southeast-1"
pulumi stack init prod    --secrets-provider="awskms://alias/<alias>?region=ap-southeast-1"
```

Đặt budget alert trên account POC.

**Phase 1: Khung repo (0.5h)**
Tạo cây `internal/{network,data,platform,app}` với `Args`, `Outputs`, `New()` rỗng; `main.go` lắp ghép và đọc config. Chạy `pulumi preview` phải chạy được với 0 resource.

**Phase 2: `network` + `platform` (2h)**
VPC, subnet, endpoint, ECS cluster, ALB, ECR. `pulumi up` trên dev, kiểm tra bằng `aws ec2 describe-vpcs` và `pulumi stack output`.

**Phase 3: `data` + `app` (2-3h)**
RDS, EFS, Fargate service, task role. Truyền output giữa package bằng biến Go (`net.VpcID`, `db.SecretArn`). Truy cập ALB DNS trả về trang nginx.

**Phase 4: Nhân ra staging và prod (1h)**
Điền `Pulumi.staging.yaml`, `Pulumi.prod.yaml`. `up` staging, `preview` prod và đọc kỹ diff. Chứng minh **không sửa dòng code nào** giữa các env.

**Phase 5: Flow vận hành (2-3h)**

- CI: PR chạy `preview`, merge chạy `up` dev, staging/prod chờ approval; auth AWS bằng OIDC, không lưu access key.
- IAM: role deploy chỉ CI assume được, role dev read-only.
- Nếu công ty đã có CI, dùng luôn CI đó; nếu chưa, GitHub Actions là phương án nhanh nhất.

**Phase 6: Các kịch bản chứng minh (1-2h)**

| Kịch bản                                                    | Kỳ vọng chứng minh                         |
| ----------------------------------------------------------- | ------------------------------------------ |
| Đổi tên logical resource `"rds"` → `"db"` rồi `preview`     | Thấy `replace`; sửa bằng `aliases` thì hết |
| `pulumi destroy` trên prod (đã `protect`)                   | Bị chặn                                    |
| Sửa tay tag/replicas trên console, `refresh --preview-only` | Thấy drift                                 |
| Dev chạy `pulumi up` prod bằng role read-only               | `AccessDenied`                             |
| `config set --secret`, xem `Pulumi.dev.yaml`                | Secret ở dạng mã hóa KMS                   |
| Đo thời gian `preview` lúc đầy đủ resource                  | Số liệu để bàn tiêu chí tách project       |

**Phase 7: Dọn dẹp và báo cáo (1h)**

```bash
# DESTRUCTIVE: destroy dev, staging (prod nếu có up)
pulumi destroy -s staging && pulumi destroy -s dev
aws elbv2 describe-load-balancers      # kiểm tra sót
```

Bỏ `protect` trước nếu cần destroy prod. RDS đặt `skipFinalSnapshot=true` ở dev/staging để destroy gọn.

## Tài liệu nộp mentor

1. Repo với cấu trúc đúng proposal.
2. Output `preview`/`up` của dev và staging, `preview` của prod.
3. Kết quả các kịch bản ở Phase 6 (log hoặc ảnh chụp).
4. Trang tổng kết ngắn: cái gì chạy được, cái gì khác prod, thời gian `preview`, vấn đề gặp phải, đề xuất có tách project hay không.

## Tiêu chí đạt

- [ ] Một code, 3 stack, chỉ khác file config
- [ ] `up` dev và staging thành công, prod `preview` đúng
- [ ] `protect` chặn destroy, `aliases` chặn replace khi rename
- [ ] Flow PR preview → merge up → approval chạy được
- [ ] Prod không `up` được từ máy local
- [ ] Đã dọn sạch tài nguyên

## Rủi ro và cách giảm

| Rủi ro                                                        | Cách giảm                                                  |
| ------------------------------------------------------------- | ---------------------------------------------------------- |
| Tốn tiền do quên tài nguyên (ALB, RDS, Fargate tính theo giờ) | Budget alert, destroy ngay sau demo, kiểm tra bằng AWS CLI |
| Trùng tên resource giữa 3 stack trong cùng account            | Dùng auto-name của Pulumi hoặc gắn `ctx.Stack()` vào tên   |
| Thiếu quyền KMS/S3 cho role CI                                | Kiểm tra sớm bằng `pulumi stack init` trong role đó        |
| Mất thời gian ở TGW/NAT                                       | Đã loại khỏi phạm vi, để cờ `enableTgw` cho sau            |

Tổng thời gian ước tính khoảng 1-2 ngày làm việc.

## Thay đổi khi triển khai so với kế hoạch

| Kế hoạch                                     | Thực tế                                                                                                     | Lý do                                                                          |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `app` chạy image nginx công khai             | Web app Go tự viết (`app/`), đọc/ghi Postgres, có `/health`                                                 | Chứng minh được kết nối App → RDS qua Secrets Manager, không chỉ trang tĩnh    |
| ECR 1 repo trong `platform`                  | ECR dùng chung `poc-app` tạo ngoài stack (docs/SETUP.md bước 3b), image tag theo Git SHA                    | 3 stack dùng chung image; destroy stack không mất image                         |
| Mỗi package có `Args`/`Outputs` + `New()`    | `Args` + `New()` trả về **ComponentResource** (`infra-poc:<package>:<Tên>`), thêm package `shared`          | Preview/console nhóm resource theo module; gom hằng số, tag, egress dùng chung |
| CI: PR preview, merge up, approval           | Push chạy 2 job `preview` → `deploy` (GitHub Environment, Required reviewers cho staging/prod)              | Người duyệt xem diff trước khi approve                                         |
| IAM: role deploy chỉ CI assume, dev read-only | 6 role (preview read-only + deploy cho mỗi env), trust `StringEquals` với OIDC `sub` immutable (`@<id>`)   | Repo tạo sau 15/07/2026 dùng format `sub` mới; tránh wildcard bị lợi dụng      |
| Cờ `enableTgw`                               | **Đã bỏ** khỏi config và code (chưa từng có code dùng)                                                      | TGW/NAT ngoài phạm vi POC; giữ cờ không tác dụng dễ gây hiểu nhầm              |
