# Vận hành

Runbook cho người vận hành POC. Dựng từ đầu xem [SETUP.md](SETUP.md); lý do thiết kế xem [DECISIONS.md](DECISIONS.md).

## Quy ước chung

- Local chỉ chạy `preview` workload. Mọi `up` workload đi qua CI (`deploy.yml`).
- Bootstrap chỉ đổi qua workflow `bootstrap.yml`. Không sửa tay trên console hay AWS CLI.
- Không dùng `--show-secrets` ở nơi log công khai.

## Workflow CI

> `pr-check.yml` và `deploy.yml` đang chỉ chạy tay (`workflow_dispatch`). Bật lại tự động: bỏ comment khối `pull_request` / `push` trong hai file. `bootstrap.yml` luôn chỉ chạy tay; `pulumi-pack.yml` chạy tự động và không đụng AWS.

Nhánh map sang stack: `dev` → `dev`, `staging` → `staging`, `main` → `prod`.

| Workflow          | Khi nào                                                      | Làm gì                                                                                                                                                                                                                                               |
| ----------------- | ------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pr-check.yml`    | PR vào `dev` / `staging` / `main`                            | `go mod tidy -diff`, build app, `go vet` + `go test` + `go build` infra, rồi `pulumi preview --diff` bằng role read-only. Giữ `imageTag` đang chạy để diff chỉ phản ánh hạ tầng. Kết quả cập nhật vào một comment trên PR; preview lỗi thì job fail. |
| `deploy.yml`      | Push vào `dev` / `staging` / `main` (bỏ qua `*.md`, `docs/`) | Job `preview` (read-only, quyết định image tag, diff ra Job Summary), rồi job `deploy` trong GitHub Environment cùng tên: build và push image tag Git SHA 7 ký tự (chỉ khi `app/` đổi), `pulumi up`.                                                  |
| `bootstrap.yml`   | Chạy tay, nhánh `main`                                       | `preview` hoặc `up` cho project bootstrap, qua environment `bootstrap`.                                                                                                                                                                              |
| `pulumi-pack.yml` | PR / push `main` đụng skill pack                             | Kiểm tra tĩnh skill pack trên Ubuntu và Windows.                                                                                                                                                                                                     |

### Deploy workload

1. Mở PR, đọc comment preview.
2. Merge. Job `preview` chạy, đọc diff ở Job Summary.
3. Staging/prod: thiết kế là người duyệt (_Required reviewers_) xem diff của job `preview` rồi mới approve job `deploy`. **Hiện chưa bật**, job `deploy` chạy ngay sau `preview` (xem mục dưới).
4. Sau deploy, `pulumi stack output` in ra `serviceUrl`, `albDnsName`, `imageTag`.

**Khi nào build image.** Job `preview` so `app/` (gồm Dockerfile, `go.mod`) giữa commit này và commit của `imageTag` đang chạy trên stack:

| Tình huống | Image tag | Build | Task Definition |
| ---------- | --------- | ----- | --------------- |
| `app/` đổi, hoặc stack chưa có `imageTag` | SHA commit này | Có | Đổi, ECS rolling-update |
| `app/` không đổi (chỉ sửa `infra/`, workflow...) | Giữ tag đang chạy | Không | Không đổi, task không restart |
| Tag đang chạy không còn trong lịch sử git hoặc không còn trong ECR | SHA commit này | Có | Đổi |

Job Summary của `preview` ghi rõ tag nào và có build hay không. Job `deploy` dùng đúng tag đó (`needs.preview.outputs.image_tag`), không tự tính lại. So sánh theo nội dung `app/`, không theo quan hệ tổ tiên, nên chạy đúng khi mỗi stack deploy từ nhánh khác nhau.

Muốn ép build lại dù `app/` không đổi (ví dụ đổi base image phía ngoài): sửa một dòng trong `app/` hoặc xoá tag trong ECR.

### Approval chưa bật

Cổng approve nằm ở cấu hình GitHub (Settings > Environments), không nằm trong code. Workflow đã khai báo `environment:` cho job `deploy` và `bootstrap`, nên chỉ cần bật *Required reviewers* là có cổng, không phải sửa workflow.

Hiện trạng (kiểm tra bằng `gh api repos/<owner>/<repo>/environments`): `bootstrap` đã bật, `dev`, `staging`, `prod` chưa bật reviewer, `prod` cũng chưa giới hạn deployment branch. Hệ quả:
- `staging`, `prod`: `pulumi up` chạy ngay khi `preview` xong, không ai bắt buộc phải đọc diff.
- `bootstrap`: **đã bật** *Required reviewers* và giới hạn deployment branch. Đây là environment quan trọng nhất vì role bootstrap là Admin và chỉ environment này bảo vệ nó.
- `dev`: không cần reviewer, đúng thiết kế.

Bật khi sẵn sàng (làm tay trên GitHub): `staging`, `prod` tick *Required reviewers*; `prod` giới hạn *Deployment branches* là `main`. Lưu ý: với repo private, *Required reviewers* cần gói GitHub Pro, Team hoặc Enterprise.

### Image đến ECS thế nào

`imageTag` là cầu nối duy nhất giữa build app và hạ tầng. Pulumi không build image, chỉ nhận tag làm input.

1. Job `deploy` build `./app`, push lên ECR `poc-app:<sha7>` (bỏ qua nếu `app/` không đổi).
2. `pulumi up` với `imageTag` mới. `infra/internal/app/service.go` ghép `<ecr-url>:<imageTag>` vào `containerDefinitions` của Task Definition.
3. Tag đổi thì Task Definition thành revision mới, ECS Service trỏ sang revision đó.
4. ECS rolling deployment (AWS làm, không phải Pulumi): start task mới, kéo image từ ECR, ALB gọi `/health` phải trả `200` thì task mới nhận traffic, rồi tắt task cũ.

Thứ tự bắt buộc: push image trước, `pulumi up` sau; ngược lại ECS kéo image chưa tồn tại và task không start. Pull code về không tự cập nhật ECS: chỉ `pulumi up` trong CI mới áp dụng tag.

**Rollback app:** đặt `imageTag` về SHA cũ (image còn trong ECR) rồi chạy `up`; không cần build lại.

**Giới hạn đã biết:** ECS Service chưa bật deployment circuit breaker. App mới lỗi health check thì ECS không chuyển traffic (task cũ vẫn phục vụ) nhưng `pulumi up` chờ rồi báo lỗi, stack ở trạng thái update dở, không tự rollback. Xử lý ở mục "Sự cố".

Chỉ một deploy chạy trên mỗi nhánh tại một thời điểm (`concurrency: deploy-<ref>`, `cancel-in-progress: false`). PR mới đẩy lên thì preview cũ bị huỷ.

### Preview local

```bash
cd infra/workload
pulumi login "s3://pulumi-state-poc-<account>?region=ap-southeast-1"
pulumi stack select dev
pulumi config set imageTag "$(pulumi stack output imageTag)"   # thiếu hoặc `latest` thì preview báo lỗi
pulumi preview
```

Lệnh `config set` lấy tag **đang chạy** trên stack (output `imageTag`) và ghi vào `Pulumi.<stack>.yaml` ở máy bạn. `main.go` bắt buộc có `imageTag`, mà tag do CI set nên file config không có sẵn. Dùng đúng tag đang chạy thì `preview` thấy image không đổi, diff chỉ phản ánh thay đổi hạ tầng bạn sửa. Đừng commit dòng `imageTag` này. Stack chưa deploy lần nào thì đặt tay một tag bất kỳ khác `latest` (preview không cần image tồn tại).

`pulumi login` đổi backend cho cả máy. Quay lại bootstrap phải login lại, hoặc đặt backend theo lệnh:

```bash
PULUMI_BACKEND_URL="s3://pulumi-bootstrap-poc-<account>?region=ap-southeast-1" pulumi preview
```

## Role CI

Trust policy dùng `StringEquals` với `aud = sts.amazonaws.com` và OIDC `sub` immutable `repo:<owner>@<owner_id>/<repo>@<repo_id>:...` (repo tạo sau 15/07/2026). Role do project bootstrap quản lý (package `ciiam`) và có `protect`.

| Role                       | Tin `sub`                                | Quyền                                         | GitHub secret              |
| -------------------------- | ---------------------------------------- | --------------------------------------------- | -------------------------- |
| `poc-dev-preview-role`     | `pull_request`, `ref:refs/heads/dev`     | ReadOnlyAccess + đọc state S3 + `kms:Decrypt` | `AWS_ROLE_DEV_PREVIEW`     |
| `poc-dev-deploy-role`      | `environment:dev`                        | AdministratorAccess                           | `AWS_ROLE_DEV_DEPLOY`      |
| `poc-staging-preview-role` | `pull_request`, `ref:refs/heads/staging` | như trên                                      | `AWS_ROLE_STAGING_PREVIEW` |
| `poc-staging-deploy-role`  | `environment:staging`                    | AdministratorAccess                           | `AWS_ROLE_STAGING_DEPLOY`  |
| `poc-prod-preview-role`    | `pull_request`, `ref:refs/heads/main`    | như trên                                      | `AWS_ROLE_PROD_PREVIEW`    |
| `poc-prod-deploy-role`     | `environment:prod`                       | AdministratorAccess                           | `AWS_ROLE_PROD_DEPLOY`     |
| `poc-bootstrap-role`       | `environment:bootstrap`                  | AdministratorAccess                           | `AWS_ROLE_BOOTSTRAP`       |

Đổi quyền CI: sửa code `ciiam`, mở PR, merge vào `main`, rồi chạy bootstrap.

## Bootstrap

Actions > **Bootstrap Infrastructure** > Run workflow (nhánh `main`):

1. Chọn `preview`, duyệt, đọc diff ở Job Summary.
2. Chạy lại với `up`, duyệt lần nữa.

(Bước "duyệt" do *Required reviewers* của environment `bootstrap`, đã bật.)

Đổi bootstrap: sửa code, mở PR, merge vào `main`, rồi chạy hai bước trên.

### Bị khoá thì làm gì

Role bootstrap do chính bootstrap quản lý. PR sửa sai trust policy của nó thì workflow không assume được nữa. Admin sửa code rồi `pulumi up` từ máy:

```bash
cd infra/bootstrap
pulumi login "s3://pulumi-bootstrap-poc-<account>?region=ap-southeast-1"
pulumi stack select poc
pulumi up
```

Ghi lại lý do và đưa thay đổi hợp lệ về code qua PR.

## Sự cố

| Tình huống                                                       | Xử lý                                                                                                                                               |
| ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pulumi up` lỗi giữa chừng, còn pending operations hoặc lock     | Dừng deploy. Xác nhận không còn update nào đang chạy, đối chiếu resource thật với state, rồi mới `pulumi refresh` / `pulumi cancel`. Không retry mù |
| Sai account (`aws credentials belong to a different account...`) | Credential không khớp `expectedAccount` của stack. Dừng trước khi tạo gì. Đổi credential, không đổi config để lách                                  |
| App mới lỗi (task không qua health check `/health`)              | Task cũ vẫn phục vụ. Xem log ở CloudWatch Logs của service. Sửa code rồi push lại, hoặc đặt `imageTag` về SHA cũ và chạy `up`. Không tự rollback (chưa có circuit breaker) |
| Thiếu `imageTag`                                                 | Chạy local thì set theo lệnh ở mục preview local; CI tự set                                                                                         |
| Drift (ai đó sửa tay trên console)                               | `pulumi refresh --preview-only` để xem, sửa code hoặc `up` để kéo về đúng                                                                           |
| Khôi phục state                                                  | S3 có versioning. Sao lưu state hiện tại trước, rồi đối chiếu resource. State cũ không phải rollback hạ tầng                                        |
| CI/OIDC lỗi                                                      | Kiểm tra `sub`, environment, secret ARN. Với bootstrap xem "Bị khoá thì làm gì"                                                                     |

## Dọn dẹp

```bash
cd infra/workload
pulumi destroy -s staging && pulumi destroy -s dev
# prod: nếu đã bật protectStateful/deletionProtection thì tắt, pulumi up, rồi mới destroy
aws elbv2 describe-load-balancers   # kiểm tra tài nguyên sót
aws rds describe-db-instances
```

- RDS có `skipFinalSnapshot = !protectStateful`: `protectStateful=false` thì destroy không giữ snapshot.
- Destroy workload không đụng bootstrap. Bucket state, KMS key, ECR có `protect` và giữ lại khi destroy bootstrap; muốn xoá hẳn phải gỡ `protect` và xoá tay có chủ đích.
