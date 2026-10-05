# Pulumi POC Setup Guide

Dựng POC từ đầu trên một account AWS mới. Thứ tự: bootstrap (nền), cấu hình GitHub, stack workload, deploy đầu tiên. Chỉ ghi thứ tự và lệnh; lý do thiết kế xem [DECISIONS.md](DECISIONS.md).

## Yêu cầu

- Credential admin của account đích (chỉ dùng cho bước 1, vì CI chưa có role).
- AWS CLI, Pulumi CLI, Go (phiên bản trong [`infra/go.mod`](../infra/go.mod)).
- Repo GitHub của dự án. Lấy `owner`, `ownerId`, `repo`, `repoId` để điền vào [`Pulumi.poc.yaml`](../infra/bootstrap/Pulumi.poc.yaml) (OIDC `sub` dạng immutable).
- Region: `ap-southeast-1`. Không commit account ID dạng plaintext; mọi nơi cần account ID đều dùng `expectedAccount` (secret).

## 1. Bootstrap (admin, chạy tay một lần)

[`infra/bootstrap`](../infra/bootstrap/main.go) tạo nền cho mọi stack workload. Dùng credential admin của account đích, cần vì CI chưa có role.

| Resource                                                       | Code                                                                           |
| -------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| S3 `pulumi-state-poc-<account>` (state workload)               | [`statebackend/bucket.go`](../infra/bootstrap/internal/statebackend/bucket.go) |
| KMS key `alias/pulumi-poc-key` (mã hoá secret Pulumi)          | [`statebackend/key.go`](../infra/bootstrap/internal/statebackend/key.go)       |
| ECR `poc-app`                                                  | [`registry/module.go`](../infra/bootstrap/internal/registry/module.go)         |
| GitHub OIDC provider, 6 role CI preview/deploy, role bootstrap | [`ciiam/`](../infra/bootstrap/internal/ciiam/)                                 |

Config stack `poc` nằm ở [`Pulumi.poc.yaml`](../infra/bootstrap/Pulumi.poc.yaml): region, prefix, repo GitHub (`owner`, `ownerId`, `repo`, `repoId`), `importExisting`, `expectedAccount` mã hoá. Sửa phần GitHub cho đúng repo của bạn trước khi `up`.

**1.1. Tạo bucket lưu state của bootstrap** (resource tạo tay duy nhất, vì bootstrap không thể lưu state trong bucket do chính nó tạo; script chạy lại an toàn):

```bash
bash scripts/setup-bootstrap-backend.sh
```

**1.2. Tạo stack.** Chưa có KMS key nên dùng passphrase trước:

```bash
cd infra/bootstrap
pulumi login "s3://pulumi-bootstrap-poc-<account>?region=ap-southeast-1"
pulumi stack init poc --secrets-provider passphrase
pulumi config set --secret expectedAccount <account-id>
pulumi config set importExisting false
```

`importExisting`: đặt `true` nếu account đã có bucket state, KMS key, ECR hoặc role CI tạo tay từ trước. Lần `up` đầu import chúng thay vì tạo mới, xong đổi lại `false`. Account trống để `false`.

**1.3. Deploy lần đầu:**

```bash
pulumi preview --diff
pulumi up
```

**1.4. Chuyển secrets sang KMS** (key đã có sau `up`):

```bash
pulumi stack change-secrets-provider "awskms://alias/pulumi-poc-key?region=ap-southeast-1"
```

Commit `Pulumi.poc.yaml` (có `expectedAccount` mã hoá và `secretsprovider` KMS).

Từ đây mọi thay đổi bootstrap đi qua workflow `bootstrap.yml`, xem [OPERATIONS.md](OPERATIONS.md#bootstrap).

## 2. Cấu hình GitHub

ARN role lấy từ output của stack bootstrap. Output bị che `[secret]` nếu thiếu `--show-secrets`:

```bash
cd infra/bootstrap && pulumi stack output --show-secrets
```

**Secrets** (Settings > Secrets and variables > Actions):

| Secret                                                                      | Lấy từ output      |
| --------------------------------------------------------------------------- | ------------------ |
| `AWS_ROLE_BOOTSTRAP`                                                        | `bootstrapRoleArn` |
| `AWS_ROLE_DEV_PREVIEW`, `AWS_ROLE_STAGING_PREVIEW`, `AWS_ROLE_PROD_PREVIEW` | `previewRoleArns`  |
| `AWS_ROLE_DEV_DEPLOY`, `AWS_ROLE_STAGING_DEPLOY`, `AWS_ROLE_PROD_DEPLOY`    | `deployRoleArns`   |

**Environments** (Settings > Environments):

- `dev`, `staging`, `prod`: thiết kế bật _Required reviewers_ cho `staging` và `prod`; giới hạn deployment branch `main` cho `prod`.
- `bootstrap`: thiết kế bật _Required reviewers_, giới hạn deployment branch `main`. Nên làm, vì role bootstrap có quyền Admin và chỉ tin đúng environment này.

> Hiện POC đã bật _Required reviewers_ cho `bootstrap` nhưng **chưa bật** cho `staging`, `prod` (tạo environment thì đủ để workflow chạy). Hệ quả và cách bật xem [OPERATIONS.md](OPERATIONS.md#approval-chưa-bật).

**Nhánh:** `dev`, `staging`, `main` (map sang stack `dev`, `staging`, `prod`).

## 3. Stack workload

Ba stack đã có file config trong [`infra/workload`](../infra/workload/). Account mới thì tạo lại stack với KMS key của account đó (key mới không giải mã được `secure:` cũ):

```bash
cd infra/workload
pulumi login "s3://pulumi-state-poc-<account>?region=ap-southeast-1"

KMS_URL="awskms://alias/pulumi-poc-key?region=ap-southeast-1"
for s in dev staging prod; do
  pulumi stack init "$s" --secrets-provider="$KMS_URL"
done
```

Mỗi stack cần config (giá trị mẫu xem [README](../README.md#config-theo-stack)):

```bash
pulumi stack select dev
pulumi config set aws:region ap-southeast-1
pulumi config set vpcCidr 10.10.0.0/16
pulumi config set desiredCount 1
pulumi config set dbInstanceClass db.t4g.micro
pulumi config set protectStateful false
pulumi config set deletionProtection false
pulumi config set ecrRepositoryName poc-app
pulumi config set --secret expectedAccount <account-id>
```

Lặp lại cho `staging` (`10.20.0.0/16`) và `prod` (`10.30.0.0/16`, `desiredCount 2`, `db.t4g.small`). CIDR phải khác nhau vì 3 stack cùng một account. `imageTag` không đặt ở đây, CI set theo Git SHA.

Commit các `Pulumi.<stack>.yaml`.

## 4. Deploy đầu tiên

`pr-check.yml` và `deploy.yml` đang chỉ chạy tay (xem [README](../README.md#cicd)). Stack chưa có image trong ECR nên deploy đầu tiên đi qua CI để build và push image trước:

1. Actions > **CD Deploy Infrastructure & App** > Run workflow, chọn nhánh `dev`.
2. Job `preview`: đọc diff ở Job Summary.
3. Job `deploy` (chờ approval nếu environment đã bật reviewer, hiện `dev`, `staging`, `prod` chưa bật): build image tag Git SHA, push ECR, `pulumi up`.
4. Lặp lại với `staging`; với `main` (prod) chỉ nên chạy preview trừ khi muốn dựng prod thật (xem [POC-RESULTS.md](POC-RESULTS.md)).

Local chỉ `preview`:

```bash
cd infra/workload
pulumi stack select dev
pulumi config set imageTag "$(pulumi stack output imageTag)"   # cần stack đã deploy ít nhất một lần
pulumi preview
```

## 5. Kiểm tra

```bash
pulumi stack output serviceUrl     # ALB DNS, mở bằng trình duyệt
curl -s "http://$(pulumi stack output albDnsName)/health"
```

Trang web hiển thị trạng thái kết nối RDS và cho ghi note. Khi bật lại trigger tự động:

- Mở PR vào `dev`: job `validate-and-preview` comment diff vào PR.
- Push `dev` / `staging` / `main`: job `preview` in diff ra Job Summary, job `deploy` chờ approval (staging/prod, khi đã bật reviewer) rồi `pulumi up`.

## 6. Dọn dẹp

Xem [OPERATIONS.md](OPERATIONS.md#dọn-dẹp).
