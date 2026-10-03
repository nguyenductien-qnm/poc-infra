# Bootstrap

Project `infra/bootstrap` quản lý phần nền mà mọi stack workload dựa vào. Admin chạy tay, CI không chạy project này, nên pipeline workload không tự sửa được quyền của chính nó.

| Resource | Package | Bảo vệ |
| --- | --- | --- |
| S3 `pulumi-state-poc-<account>` (state workload) + versioning, mã hoá, chặn public, tắt ACL | `bootstrap/internal/statebackend` | bucket: `protect` + giữ lại khi destroy |
| KMS key + alias `alias/pulumi-poc-key` (mã hoá secret Pulumi) | `bootstrap/internal/statebackend` | key: `protect` + giữ lại khi destroy |
| ECR `poc-app` | `bootstrap/internal/registry` | `protect` + giữ lại khi destroy |
| GitHub OIDC provider, 6 role CI, policy `poc-ci-preview-policy` | `bootstrap/internal/ciiam` | provider, role: `protect` |

## State của bootstrap lưu ở đâu

Ở bucket riêng `pulumi-bootstrap-poc-<account>`, tạo bằng `scripts/setup-bootstrap-backend.sh`. Đây là resource **duy nhất tạo tay**: bootstrap không thể lưu state trong chính bucket nó quản lý. Workload vẫn lưu state ở `pulumi-state-poc-<account>`.

## Lần đầu: tiếp nhận resource đang có

Các resource trên đã tạo bằng tay/script từ trước. Bootstrap **import** chúng, không tạo mới. Code khai báo đúng cấu hình hiện tại (kể cả quyền Admin của role deploy) để import không thay đổi gì.

Chạy bằng credential admin:

```bash
# 1. Tạo bucket state cho bootstrap
bash scripts/setup-bootstrap-backend.sh

# 2. Login vào bucket bootstrap, tạo stack poc (1 stack cho 1 account)
cd infra/bootstrap
pulumi login "s3://pulumi-bootstrap-poc-<account>?region=ap-southeast-1"
pulumi stack init poc --secrets-provider "awskms://alias/pulumi-poc-key?region=ap-southeast-1"
pulumi config set --secret expectedAccount <account-id>

# 3. Preview: chỉ được có import
pulumi preview --diff
```

Kết quả preview đúng: `25 to import`, không có `update`, `replace`, `delete`. Có thì **dừng lại**: cấu hình thật đã lệch khỏi code, sửa code cho khớp rồi preview lại, không `up`.

```bash
# 4. Import vào state
pulumi up

# 5. Tắt chế độ import, preview phải không còn thay đổi
pulumi config set importExisting false
pulumi preview --diff
```

Commit `Pulumi.poc.yaml` (có `expectedAccount` đã mã hoá, `importExisting: "false"`).

Kiểm tra cục bộ khi viết code này (state tạm trên máy, chỉ đọc AWS): 25 import, 0 update/replace/delete; sai account bị chặn; `importExisting=false` ra 29 create.

## Account mới

Đặt `importExisting: "false"`. Lúc này chưa có KMS key nên tạo stack bằng `--secrets-provider passphrase`. Sau `up` (key đã có), chuyển sang KMS:

```bash
pulumi stack change-secrets-provider "awskms://alias/pulumi-poc-key?region=ap-southeast-1"
```

## Chạy xen kẽ bootstrap và workload

Hai project ở hai bucket khác nhau. `pulumi login` đổi backend cho cả máy, nên khi quay lại workload phải login lại bucket workload. Hoặc set backend theo từng lệnh:

```bash
PULUMI_BACKEND_URL="s3://pulumi-bootstrap-poc-<account>?region=ap-southeast-1" pulumi preview
```

## Thay đổi sau này

Mọi thay đổi bootstrap đi qua PR rồi admin preview/up. Không sửa role, OIDC, bucket, key bằng console hay AWS CLI nữa.

## Giới hạn hiện tại

- Quyền CI giữ nguyên như script cũ: role deploy có `AdministratorAccess`, role preview có `ReadOnlyAccess` và `kms:Decrypt` trên `*`. Siết quyền là bước sau (issue #17).
- `ReadOnlyAccess` cho role preview đọc được mọi bucket, kể cả bucket bootstrap. State bootstrap không chứa secret dạng rõ, nhưng nên chặn bằng bucket policy ở bước siết quyền.
- Bucket, key chưa có tag; key chưa bật rotation. Giữ nguyên để import khớp, thêm ở thay đổi riêng.
- Chưa chạy `up` thật: import mới được kiểm chứng bằng preview.
