---
name: go-ecs-service
description: Best practices for writing Go HTTP services that run as containers on AWS ECS Fargate behind an ALB, with Postgres (RDS) and secrets from Secrets Manager - env-based config, DB connection with retry, health checks, graceful shutdown, structured logging, multi-stage Dockerfile, image tagging. Use this skill whenever writing or changing a Go web service/API, its main.go, handlers, DB access, Dockerfile, or the env vars/secrets passed from the ECS task definition - even if the user only says "thêm API", "viết service", "sửa Dockerfile".
---

# Go service trên ECS Fargate

Service phải: khởi động được khi DB chưa sẵn sàng, cho ALB biết khi nào khoẻ, tắt êm khi ECS gửi SIGTERM, và không bao giờ cầm secret ngoài env do ECS inject.

> **Trước khi viết code:** version, tên field SDK, giá trị hợp lệ của AWS và shape dữ liệu phải verify theo skill `verify-before-code`, không lấy từ trí nhớ. Repo đã có code cùng loại thì đọc code đó làm chuẩn; mẫu trong skill chỉ là khung cho pattern dễ viết lệch.

## Cấu trúc

Service nhỏ: một package `main` chia file theo trách nhiệm (`main.go`, `config.go`, `db.go`, `handlers.go`). Lớn hơn:

```
app/
  cmd/<service>/main.go     Wiring: load config, mở DB, đăng ký route, chạy server
  internal/config/          Đọc + validate env
  internal/store/           Truy cập DB (SQL, migration)
  internal/httpapi/         Handler, middleware
  Dockerfile
```

Ưu tiên stdlib (`net/http` với pattern routing Go 1.22+, `log/slog`, `database/sql`). Thêm thư viện khi có lý do rõ (driver DB: `pgx` hoặc `lib/pq`).

## Quy tắc

### 1. Config qua env (12-factor)
- Mọi config đọc từ env vào một struct `Config`, validate một lần lúc start, thiếu biến bắt buộc thì exit với lỗi rõ ràng.
- Biến tuỳ chọn có default hợp lý và an toàn (vd `DB_SSLMODE` mặc định `require`, `PORT` mặc định trùng port container trong task def).
- Tên env khớp chính xác với task definition bên infra. Thêm env mới = sửa cả 2 phía trong cùng PR.
- Không đọc file config theo môi trường, không có `if env == "prod"` trong code: khác biệt đi qua env.

### 2. Secret
- Secret do ECS inject từ Secrets Manager qua `secrets[].valueFrom` (lấy đúng key JSON bằng cú pháp `<arn>:<key>::`), app chỉ đọc env.
- Phòng trường hợp nhận cả JSON (`{"username","password"}`): parse an toàn, fallback giá trị thô.
- Không log secret, DSN, header `Authorization`. Lỗi kết nối log host/db, không log password.

### 3. Database
- Mở pool một lần, cấu hình `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime` (nhỏ hơn timeout của RDS proxy/NLB nếu có).
- Kết nối có retry + backoff ở background, **không chặn server start**: ALB health check vẫn trả lời trong lúc DB khởi động.
- Mọi query dùng `ctx` của request (`QueryContext`, `ExecContext`) và tham số `$1`, không ghép chuỗi SQL.
- `rows.Close()` qua `defer`, kiểm tra `rows.Err()`.
- Migration: tool riêng (golang-migrate, goose) chạy trước deploy hoặc lúc start có lock; `CREATE TABLE IF NOT EXISTS` chỉ chấp nhận cho POC.
- TLS bắt buộc tới RDS (`sslmode=require` hoặc `verify-full` với CA bundle).
- Biến toàn cục dùng chung giữa goroutine phải có mutex hoặc tránh hẳn bằng dependency injection.

### 4. Health check
- `/health` (liveness): trả 200 nhanh, **không** phụ thuộc DB, để ALB không giết task chỉ vì DB chậm.
- `/ready` (tuỳ chọn): kiểm tra DB ping với timeout ngắn, dùng cho dashboard/monitor, không gắn vào target group trừ khi muốn rút task khỏi LB khi mất DB.
- Path health khớp `HealthCheck.Path` của target group.

### 5. HTTP server
- Dùng `http.Server` với `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`; không dùng `http.ListenAndServe` trần.
- Giới hạn body (`http.MaxBytesReader`), kiểm tra method (pattern `"POST /notes"` của Go 1.22+).
- Middleware: recover panic, request log (method, path, status, duration), request ID (`X-Request-Id` hoặc header ALB `X-Amzn-Trace-Id`).
- Template HTML dùng `html/template` (tự escape), parse một lần lúc start.

### 6. Graceful shutdown
- Bắt `SIGTERM`/`SIGINT` bằng `signal.NotifyContext`, gọi `srv.Shutdown(ctx)` với timeout nhỏ hơn `stopTimeout` của ECS (mặc định 30s) và deregistration delay của target group.
- Đóng DB sau khi server shutdown xong.

### 7. Logging
- `log/slog` JSON handler ra stdout (CloudWatch awslogs đọc stdout). Có field cố định `service`, `version` (Git SHA).
- Level qua env `LOG_LEVEL`. Lỗi log kèm `err`, không `log.Fatal` ngoài `main`.

### 8. Container image
- Multi-stage: build bằng `golang:<version>` (≥ directive `go` trong `go.mod`, verify tag tồn tại), `CGO_ENABLED=0`, `-trimpath -ldflags="-s -w -X main.version=<sha>"`; runtime `distroless/static` hoặc `alpine` + `ca-certificates` + `tzdata`.
- Chạy **non-root** (`USER nonroot` / `USER 65532`), port > 1024 (vd 8080) để không cần quyền root.
- Build `--platform linux/amd64` (hoặc arm64 nếu task def dùng Graviton, phải khớp `runtimePlatform`).
- Tag image bằng Git SHA, không deploy `:latest`. ECR bật scan on push và tag immutability.
- `.dockerignore` loại `.git`, binary, file local.

### 9. Error và style
- Bọc lỗi `fmt.Errorf("doing x: %w", err)`, trả lỗi lên, chỉ log ở tầng biên (handler/main).
- Handler trả status đúng nghĩa (400 input sai, 404, 500 lỗi nội bộ không lộ chi tiết).
- `go vet`, `go test ./...`, `go mod tidy -diff` sạch trước khi commit; handler quan trọng có test bằng `httptest`.

## Phía infra phải khớp

Lỗi hay gặp nhất là app và task definition lệch nhau. Sửa một bên thì soát bên kia:

| App | Infra |
| --- | --- |
| `PORT` | `portMappings.containerPort`, target group `Port` |
| `GET /health` | target group `HealthCheck.Path` |
| `DB_HOST` | `rds.Instance.Address` (host thuần). `Endpoint` là `address:port`, đừng dùng làm host |
| `DB_PASSWORD` | `secrets[].valueFrom = <secretArn>:password::`; execution role có `secretsmanager:GetSecretValue` trên ARN đó |
| Log stdout | `logConfiguration.logDriver = awslogs` |
| Shutdown timeout | nhỏ hơn `stopTimeout` ECS và deregistration delay target group |
| Platform image | `runtimePlatform.cpuArchitecture` |

## Mẫu

**Đọc secret có thể là JSON hoặc chuỗi thô:**

```go
// secretValue tra ve key trong JSON secret neu raw la JSON, nguoc lai tra raw
func secretValue(raw, key string) string {
	var m map[string]string
	if json.Unmarshal([]byte(raw), &m) == nil && m[key] != "" {
		return m[key]
	}
	return raw
}
```

## Kiểm tra

```bash
cd app && go mod tidy -diff && go vet ./... && go test ./... && go build -o /dev/null ./...
docker build --platform linux/amd64 -t <service>:local .
```

## Checklist review

- [ ] Env mới khớp task definition; secret qua `valueFrom`, không log
- [ ] Server có timeout, graceful shutdown, recover middleware
- [ ] `/health` không phụ thuộc DB; DB retry không chặn start
- [ ] Query dùng ctx + tham số; pool được cấu hình; TLS tới DB
- [ ] Image non-root, multi-stage, tag Git SHA, platform khớp task def
