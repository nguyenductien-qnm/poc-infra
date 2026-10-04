# App ↔ ECS/RDS contract

Khi sửa env, image, health hoặc data wiring, đọc `app/main.go`, Dockerfile và `infra/internal/{app,platform,data}`. [Current contract](current-contract.md) giữ ngoại lệ PoC rõ ràng.

| Giá trị | Nơi phải khớp |
| --- | --- |
| Port 80 (`shared.AppPort`), `/health` | HTTP app, ECS container port, target group/health path |
| DB host/port/name/user | `data/rds.go`, `app/service.go`, app DSN; `Address` là host, `Endpoint` có port |
| DB password | ECS `valueFrom` dùng ARN + JSON selector `:password::`; execution role đọc đúng secret |
| ECR URL/imageTag | Bootstrap repository, workload config, build/push workflow; tag bắt buộc, không `latest` |
| Logging/platform | stdout/awslogs, Docker `linux/amd64` và Fargate task |

Không fetch password để kiểm tra wiring. Tracing/build chỉ chứng minh contract cục bộ, chưa chứng minh DB connection hoặc ECS healthy. Pilot #20 cần DB connection hoặc round-trip có cleanup; `/health` 200 không đủ.

Private tasks/TLS, DB encryption/backup/Multi-AZ/protection, bỏ demo Bedrock và deployment rollback đã hoãn. Khi có task production, đề xuất config đơn giản có consumer và tác dụng thật; tránh lẫn proposal với behavior hiện tại.
