# Bootstrap

[`infra/bootstrap`](../infra/bootstrap/main.go) tạo nền cho mọi stack workload. Chạy qua workflow [`bootstrap.yml`](../.github/workflows/bootstrap.yml) (chỉ chạy tay).

| Resource                                                     | Code                                                                           |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| S3 `pulumi-state-poc-<account>` (state workload)             | [`statebackend/bucket.go`](../infra/bootstrap/internal/statebackend/bucket.go) |
| KMS key `alias/pulumi-poc-key` (mã hoá secret Pulumi)        | [`statebackend/key.go`](../infra/bootstrap/internal/statebackend/key.go)       |
| ECR `poc-app`                                                | [`registry/module.go`](../infra/bootstrap/internal/registry/module.go)         |
| GitHub OIDC provider, role CI preview/deploy, role bootstrap | [`ciiam/`](../infra/bootstrap/internal/ciiam/)                                 |

Config: [`Pulumi.yaml`](../infra/bootstrap/Pulumi.yaml).

## Setup một lần (admin, chạy tay)

Dùng credential admin của account đích. Cần vì CI chưa có role để chạy.

**1. Tạo bucket lưu state của bootstrap** ([`setup-bootstrap-backend.sh`](../scripts/setup-bootstrap-backend.sh)):

```bash
bash scripts/setup-bootstrap-backend.sh
```

**2. Tạo stack.** Chưa có KMS key nên dùng passphrase trước:

```bash
cd infra/bootstrap
pulumi login "s3://pulumi-bootstrap-poc-<account>?region=ap-southeast-1"
pulumi stack init poc --secrets-provider passphrase
pulumi config set --secret expectedAccount <account-id>
pulumi config set importExisting false
```

**3. Deploy lần đầu:**

```bash
pulumi preview --diff
pulumi up
```

**4. Chuyển secrets sang KMS** (key đã có sau `up`):

```bash
pulumi stack change-secrets-provider "awskms://alias/pulumi-poc-key?region=ap-southeast-1"
```

**5. Cấu hình GitHub** cho workflow bootstrap:

```bash
pulumi stack output bootstrapRoleArn
```

- Secret `AWS_ROLE_BOOTSTRAP` = ARN trên.
- Environment `bootstrap`: bật _Required reviewers_, giới hạn deployment branch `main`.

Ngoài ra lấy ARN cho các secret của workload: `pulumi stack output previewRoleArns` / `deployRoleArns`.

Commit `Pulumi.poc.yaml`.

## Vận hành (CI)

Actions > **Bootstrap Infrastructure** > Run workflow (nhánh `main`):

1. Chọn `preview`, duyệt, đọc diff ở Job Summary.
2. Chạy lại với `up`, duyệt lần nữa.

Đổi bootstrap: sửa code, mở PR, merge vào `main`, rồi chạy 2 bước trên. Không sửa tay trên console hay AWS CLI.

## Bị khoá thì làm gì

Role bootstrap do chính bootstrap quản lý. PR sửa sai trust policy của nó thì workflow không assume được nữa. Khi đó admin sửa code rồi `pulumi up` từ máy:

```bash
cd infra/bootstrap
pulumi login "s3://pulumi-bootstrap-poc-<account>?region=ap-southeast-1"
pulumi stack select poc
pulumi up
```

## Chạy tay xen kẽ với workload

`pulumi login` đổi backend cho cả máy. Quay lại workload phải login lại, hoặc set backend theo lệnh:

```bash
PULUMI_BACKEND_URL="s3://pulumi-bootstrap-poc-<account>?region=ap-southeast-1" pulumi preview
```
