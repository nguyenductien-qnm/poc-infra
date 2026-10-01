# Proposal: Quản lý hạ tầng bằng Pulumi (Go)

## 1. Đề xuất

**Một Pulumi project, mỗi môi trường độc lập là một stack**, chia code thành các Go package theo lớp hạ tầng.

```
infra-repo/
├── go.mod
├── Pulumi.yaml
├── Pulumi.dev-<developer>.yaml
├── Pulumi.staging.yaml
├── Pulumi.prod.yaml
├── main.go
└── internal/
    ├── network/
    │   ├── network.go
    │   ├── vpc.go
    │   ├── transit.go
    │   └── endpoints.go
    ├── data/
    │   ├── data.go
    │   ├── rds.go
    │   ├── efs.go
    │   ├── backup.go
    │   └── secrets.go
    ├── platform/
    │   ├── platform.go
    │   ├── ecr.go
    │   ├── acm.go
    │   ├── alb.go
    │   ├── ecs_cluster.go
    │   └── observability.go
    └── app/
        ├── app.go
        ├── service_<name>.go
        └── iam.go
```

Quy ước:

- **State**: S3 (bật versioning). **Secrets**: mã hóa bằng KMS (`awskms://alias/<alias>`).
- Một code, nhiều stack. Khác biệt giữa env chỉ nằm ở `Pulumi.<stack>.yaml`.
- **Môi trường**: dev, staging, prod. Dev dùng stack riêng theo developer (`dev-<developer>`), mỗi stack có state và phạm vi quyền riêng; không để các stack cùng quản lý một resource.
- Mỗi package có `Args` (đầu vào) và `Outputs` (đầu ra) viết hoa; `<tên>.go` chứa `New()` và gọi các file còn lại.
- Config chỉ đọc ở `main.go`, truyền xuống qua `Args`.

## 2. Flow vận hành

```
Local   → viết code → go test → preview / up / destroy   (dev cá nhân)
PR      → CI chạy go test → review   (không AWS credentials/state/secrets)
Merge   → CI preview staging/prod → review diff → approval → up cùng SHA
Định kỳ → refresh --preview-only phát hiện drift   (staging/prod)
```

| Môi trường | Local | CI/CD |
|---|---|---|
| dev | Tự vận hành stack riêng trong phạm vi được cấp | Chỉ test, không deploy dev |
| staging / prod | Không `up`; local preview chỉ khi được cấp riêng quyền read-only, state/KMS | Chỉ CI deploy, có approval |

Staging dùng để kiểm thử tích hợp trước production. Developer mặc định xem preview staging/prod từ CI.

### 2.1. Quyền preview

**Staging/prod tách role preview và deploy.** Quyền giới hạn theo stack/key.

| Phạm vi | Preview | Deploy |
|---|---|---|
| AWS workload | Chỉ đọc metadata cần thiết | Tạo/sửa/xóa trong phạm vi quản lý |
| S3 state | Chỉ đọc, không sửa/xóa checkpoint | Đọc/ghi checkpoint, history và lock |
| KMS | Chỉ mã hóa/giải mã cần cho preview | Mã hóa/giải mã cần cho deploy |
| Secrets ứng dụng | Chỉ ARN nếu đủ, không đọc giá trị | Tương tự preview |
| Assume role | Không assume deploy role hoặc role quyền cao hơn | Chỉ target role được chỉ định |

Không quản trị KMS key; xét riêng key S3 và Pulumi. Bootstrap backend trước, không mở quyền ghi checkpoint để preview chạy được. Không công khai state/secrets/debug log hoặc dùng `--show-secrets`.

**Lý do**: preview vẫn chạy Go và có thể đọc secrets đã giải mã; read-only AWS không bảo vệ phần này.

### 2.2. Quyền CI/CD và OIDC

**4 role CI**, không tính quyền developer/khẩn cấp. Dùng chung mẫu role/workflow; drift dùng lại role preview. GitHub chỉ có deployment environment `staging`, `prod`.

| Role CI | Ngữ cảnh |
|---|---|
| `StagingPreview`, `ProdPreview` | Code đã review trên `main`; job không gắn environment |
| `StagingDeploy` | Cùng SHA đã preview; environment `staging`, có approval |
| `ProdDeploy` | Cùng SHA đã preview; environment `prod`, có approval |

- **OIDC**: một provider mỗi account nhận OIDC. Kiểm tra `aud = sts.amazonaws.com`, `sub` đúng repo/branch cho preview, đúng repo/environment cho deploy. Deploy role không trust subject preview.
- **Approval**: environment chỉ nhận `main` được bảo vệ; duyệt trước khi cấp deploy credentials. Kiểm tra gói GitHub hỗ trợ gate này.
- **Workflow/IAM**: required review qua `CODEOWNERS`; pin action SHA. Deploy không được tự sửa quyền/trust của role CI.
- **Job**: runner sạch riêng, không chạy code PR chưa tin cậy với quyền staging/prod. Trước khi chạy, kiểm tra account, role, region, backend và stack.

**Giới hạn**: job trên `main` có OIDC subject được trust có thể xin cả hai role preview. Không cách ly quyền đọc giữa từng job.

**Lý do**: giữ ranh giới đọc/ghi, chỉ cấp quyền deploy sau approval.

### 2.3. Deploy đồng thời

**Một stack, một thao tác cập nhật.**

- Mọi job ghi cùng stack dùng chung concurrency group trong `infra-repo`; `cancel-in-progress: false`. Giữ lock backend.
- Không hủy update đang chạy; run cũ đang chờ có thể bị thay thế. Stack thay đổi trong lúc chờ thì preview/approve lại.

**Lý do**: tránh deploy chồng và ngắt update giữa chừng.

### 2.4. Bảo vệ resource và xử lý lỗi

- **Delete/replace**: resource dữ liệu cần owner duyệt thao tác và phương án phục hồi.
- **Protection**: `pulumi.Protect(true)` và deletion protection AWS khi hỗ trợ; gỡ protection phải review.
- **Backup**: kiểm thử restore trước production; chốt owner, mức mất dữ liệu và thời gian phục hồi chấp nhận được.

| Tình huống | Xử lý |
|---|---|
| Update lỗi / pending operations / lock còn | Dừng deploy; xác nhận không còn update, đối chiếu resource rồi xử lý state/lock. Không retry mù |
| Khôi phục state | Sao lưu state hiện tại, đối chiếu resource; không dùng state cũ như rollback hạ tầng |
| Khôi phục dữ liệu | Backup/PITR đã kiểm thử; owner dữ liệu xác nhận |
| CI/OIDC lỗi | Role khẩn cấp riêng, approval, thời hạn và audit; đưa thay đổi hợp lệ về code qua PR |

Lưu SHA, CI run, người duyệt và thao tác khắc phục. Chỉ mở lại deploy khi state/resource khớp và health check đạt.

**Lý do**: update lỗi có thể đã thay đổi một phần hạ tầng; phục hồi state không phục hồi dữ liệu.

## 3. Vì sao chọn cách này

- Hướng dẫn chính thức của Pulumi khuyên **bắt đầu monolithic** (một project), tách khi thật sự cần.
- Một lệnh `pulumi up` dựng toàn bộ, truyền output giữa các phần bằng biến Go, không cần `StackReference`.
- Giống mô hình Terraform `environments/` + `modules/` mà team quen: `main.go` ≈ root module, `internal/` ≈ modules, `Pulumi.<stack>.yaml` ≈ tfvars.
