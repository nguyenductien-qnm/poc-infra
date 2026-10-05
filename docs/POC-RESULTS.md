# Kết quả POC

Tổng kết nộp mentor: cái gì chạy được, cái gì khác production, bằng chứng từng kịch bản. Mỗi dòng "Kết quả" chỉ điền sau khi đã chạy thật và có log hoặc ảnh; `Chưa chạy` nghĩa là chưa có bằng chứng. Thiết kế xem [DECISIONS.md](DECISIONS.md).

## Phạm vi đã dựng

| Package | Có | Bỏ / thu nhỏ |
| ------- | -- | ------------ |
| `network` | VPC, 2 public + 2 private subnet, route table, S3 Gateway Endpoint | TGW + NAT |
| `platform` | ECS Cluster, ALB HTTP, security group ALB/App, log group | ACM/HTTPS |
| `data` | RDS PostgreSQL 16 single-AZ (mật khẩu trong Secrets Manager), EFS | AWS Backup; EFS chưa mount |
| `app` | ECS Fargate service chạy web app Go, task role | Gọi Bedrock thật |
| `bootstrap` | S3 state, KMS, ECR, OIDC provider, role CI | Role tối thiểu (TODO #17) |

Môi trường: `up` dev và staging, chỉ `preview` prod (tốn tiền theo giờ).
## Kịch bản chứng minh

| # | Kịch bản | Kỳ vọng | Kết quả |
| - | -------- | ------- | ------- |
| 1 | Đổi tên logical resource `"rds"` thành `"db"` rồi `preview`; sau đó thêm `aliases` | Lần đầu thấy `replace`; có `aliases` thì hết | Chưa chạy |
| 2 | `pulumi destroy` trên prod đã `protect` (cần `protectStateful=true`) | Bị chặn | Chưa chạy |
| 3 | Sửa tay tag hoặc replicas trên console, `pulumi refresh --preview-only` | Thấy drift | Chưa chạy |
| 4 | Chạy `pulumi up` prod bằng role preview (read-only) | `AccessDenied` | Chưa chạy |
| 5 | `pulumi config set --secret`, xem `Pulumi.<stack>.yaml` | Secret ở dạng mã hoá KMS (`secure: v1:...`) | Có thể xem ngay: `expectedAccount` trong cả ba file đều là `secure:` |
| 6 | Đo thời gian `preview` khi đủ resource | Số liệu để bàn tiêu chí tách project | Chưa chạy |

Cách chạy từng kịch bản và dọn dẹp xem [OPERATIONS.md](OPERATIONS.md).

## Tiêu chí đạt

- [ ] Một code, ba stack, chỉ khác file config
- [ ] `up` dev và staging thành công, prod `preview` đúng
- [ ] `protect` chặn destroy, `aliases` chặn replace khi rename
- [ ] Flow PR preview, merge up, approval chạy được (chưa đạt: trigger tự động đang tắt, hiện chạy bằng `workflow_dispatch`; *Required reviewers* chưa bật nên chưa có cổng approve)
- [ ] Prod không `up` được từ role read-only
- [ ] Đã dọn sạch tài nguyên (`aws elbv2 describe-load-balancers`, `aws rds describe-db-instances`)

## Số liệu

| Chỉ số | dev | staging | prod |
| ------ | --- | ------- | ---- |
| Số resource | | | |
| Thời gian `preview` | | | |
| Thời gian `up` lần đầu | | | |

## Khác production thật

Xem bảng ở [DECISIONS.md](DECISIONS.md#9-chấp-nhận-khác-production-thật).

## Vấn đề gặp phải

Điền khi có (ví dụ: import resource tạo tay, KMS key mới không giải mã được config cũ, chicken-and-egg ở image đầu tiên).

## Đề xuất

Có tách project hay không, tách theo tiêu chí nào: điền sau khi có số liệu `preview`.

## Rủi ro và cách giảm

| Rủi ro | Cách giảm |
| ------ | --------- |
| Tốn tiền do quên tài nguyên (ALB, RDS, Fargate tính theo giờ) | Budget alert, destroy ngay sau demo, kiểm tra bằng AWS CLI |
| Trùng tên resource giữa ba stack trong cùng account | Tên gắn tên stack, CIDR khác nhau |
| Thiếu quyền KMS/S3 cho role CI | Kiểm tra sớm bằng `pulumi preview` trong chính role đó |
| `up` staging/prod chạy không cần người duyệt (chưa bật *Required reviewers*; `bootstrap` đã bật) | Bật trên GitHub Environment; trong lúc chưa bật chỉ cho người tin cậy quyền Write |
| Role deploy Admin, preview `kms:Decrypt` trên `*` | Ghi nhận ở DECISIONS; thu hẹp ở TODO #17 |
