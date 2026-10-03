---
name: gitops-github-actions
description: Best practices for GitOps CI/CD with GitHub Actions deploying Pulumi stacks and container images to AWS - branch-to-stack mapping, PR preview with sticky comment, preview-then-approve deploy via GitHub Environments, AWS OIDC roles (no access keys) with immutable sub claims, read-only preview vs deploy roles, concurrency, image tagging by Git SHA. Use this skill whenever writing or changing .github/workflows files, CI IAM roles/trust policies, deploy pipelines, or approval flow - even if the user only says "sửa CI", "thêm bước deploy", "pipeline lỗi", "thêm môi trường".
---

# GitOps với GitHub Actions + Pulumi + AWS

Nguyên tắc: **Git là nguồn sự thật, máy local chỉ preview.** Mọi thay đổi lên môi trường dùng chung đi qua PR -> preview -> merge -> (approve) -> deploy. CI không giữ credential dài hạn.

> **Trước khi viết code:** version, tên field SDK, giá trị hợp lệ của AWS và shape dữ liệu phải verify theo skill `verify-before-code`, không lấy từ trí nhớ. Repo đã có code cùng loại thì đọc code đó làm chuẩn; mẫu trong skill chỉ là khung cho pattern dễ viết lệch.

## Mô hình

| Sự kiện | Job | Role AWS | Kết quả |
| --- | --- | --- | --- |
| PR vào nhánh env | validate (tidy/vet/test/build) + `pulumi preview --diff` | preview (read-only) | Sticky comment trên PR, fail nếu preview lỗi |
| Push vào nhánh env | `preview` | preview (read-only) | Diff vào Job Summary |
| -> sau `preview` | `deploy` trong GitHub Environment | deploy của env đó | Build/push image tag SHA, `pulumi up` |

Nhánh -> stack: một bảng mapping duy nhất (vd `dev`->`dev`, `staging`->`staging`, `main`->`prod`), lặp lại giống hệt ở mọi workflow. Thêm môi trường = thêm nhánh, stack, Environment, 2 role, 2 secret.

## Quy tắc

### 1. Xác thực AWS
- Chỉ OIDC (`aws-actions/configure-aws-credentials` + `permissions: id-token: write`). Không `AWS_ACCESS_KEY_ID` trong secret.
- Mỗi env 2 role: **preview** (ReadOnlyAccess + đọc state S3 + `kms:Decrypt` key secrets) và **deploy** (quyền ghi, prod thu hẹp thay vì Admin).
- Trust policy dùng `StringEquals` (không `StringLike` với `*`) trên `aud` và `sub`:
  - preview: `repo:<owner>/<repo>:pull_request` (và ref nhánh nếu job preview chạy trên push).
  - deploy: `repo:<owner>/<repo>:environment:<env>`, nên chỉ job chạy trong Environment đó assume được.
  - Repo dùng format `sub` immutable (`repo:<owner>@<owner_id>/<repo>@<repo_id>:...`) thì trust theo format đó: chống trường hợp repo bị xoá rồi tạo lại cùng tên.
- ARN role lưu ở secret `AWS_ROLE_<ENV>_<PREVIEW|DEPLOY>` (hoặc variables, ARN không bí mật). Script tạo role commit trong repo (`scripts/`), idempotent (create-or-update).

### 2. Approval
- Job `deploy` khai báo `environment: <env>`. Staging/prod bật *Required reviewers*, prod giới hạn deployment branch.
- `deploy` `needs: preview`: người duyệt đọc diff ở Job Summary của `preview` rồi mới approve. Diff phải tạo bằng **cùng input** với deploy (cùng image tag mới).
- Không `pulumi up` khi preview lỗi.

### 3. Preview trên PR
- Giữ image tag đang chạy (`pulumi stack output imageTag`) khi preview PR để diff chỉ phản ánh hạ tầng.
- Ghi output ra file, giữ exit code (`set +e` ... `set -e`), comment trước rồi mới fail ở bước cuối, để PR luôn có comment kể cả khi lỗi.
- Sticky comment: tìm comment có marker HTML (`<!-- pulumi-preview -->`) và update thay vì tạo mới. Cắt output ~60k ký tự.
- `permissions` tối thiểu: `id-token: write`, `contents: read`, `pull-requests: write` chỉ ở workflow cần comment.

### 4. Concurrency
- Deploy: `concurrency: deploy-${{ github.ref }}`, `cancel-in-progress: false` (không huỷ `up` đang chạy, tránh lock/hỏng state).
- PR: `concurrency: pr-check-${{ github.event.pull_request.number }}`, `cancel-in-progress: true`.

### 5. Image
- Build trong job deploy (sau approve) hoặc job build riêng trước preview, tag **Git SHA 7 ký tự**, không push/deploy `:latest`.
- Cùng một SHA dùng cho cả preview và deploy (`env: IMAGE_TAG: ${{ github.sha }}`, dùng `${IMAGE_TAG::7}`).
- Repo ECR dùng chung giữa env, tạo ngoài stack (destroy stack không mất image), bật tag immutability + scan.
- `pulumi config set imageTag` trong CI, không commit tag vào yaml.

### 6. An toàn workflow
- Pin action theo major tối thiểu (`@vN`, N tra bản mới nhất, không lấy từ trí nhớ); repo nhạy cảm pin theo commit SHA. Lên major thì đọc release notes tìm breaking change.
- Secret/expression đưa vào `env:` rồi dùng `"$VAR"` trong script, không nội suy `${{ }}` trực tiếp vào lệnh shell (chống injection từ tên nhánh/title PR).
- `paths-ignore` cho `**.md`, `docs/**` để đổi tài liệu không deploy.
- Job validate chạy giống hệt lệnh local: `go mod tidy -diff`, `go vet ./...`, `go test ./...`, build.
- Setup Go bằng `go-version-file` + `cache-dependency-path`, không hardcode version.

### 7. Quan sát
- Job Summary: stack, image tag, diff. Bước cuối `pulumi stack output` để có URL/ARN.
- Bước rollout ECS chờ ổn định (`aws ecs wait services-stable`) nếu `up` không tự chờ.

## Mẫu

Version action (`@vN`) cố ý không ghi: tra bản hiện tại trước khi viết.

**Chọn role/stack theo nhánh (giữ giống nhau ở mọi workflow):**

```yaml
env:
  STACK: ${{ github.ref_name == 'main' && 'prod' || github.ref_name }}   # PR: dùng github.base_ref
# ...
role-to-assume: ${{ github.ref_name == 'main' && secrets.AWS_ROLE_PROD_DEPLOY || github.ref_name == 'staging' && secrets.AWS_ROLE_STAGING_DEPLOY || secrets.AWS_ROLE_DEV_DEPLOY }}
```

**Preview giữ exit code, ghi Job Summary, fail sau cùng:**

```bash
set +e
pulumi preview --diff > preview_output.txt 2>&1
EXIT_CODE=$?
set -e
cat preview_output.txt
{
  echo "### Pulumi Preview (\`$STACK\`, image \`${IMAGE_TAG::7}\`)"
  echo '```'; head -c 60000 preview_output.txt; echo '```'
} >> "$GITHUB_STEP_SUMMARY"
exit $EXIT_CODE
```

Trên PR: ghi `exit_code` vào `$GITHUB_OUTPUT`, bước comment chạy `if: always()`, bước cuối mới `exit 1`.

**Sticky comment (github-script):**

```js
const marker = '<!-- pulumi-preview -->';
const comments = await github.paginate(github.rest.issues.listComments, { owner, repo, issue_number });
const existing = comments.find(c => c.body && c.body.includes(marker));
if (existing) await github.rest.issues.updateComment({ owner, repo, comment_id: existing.id, body });
else await github.rest.issues.createComment({ owner, repo, issue_number, body });
```

**Trust policy role CI:**

```json
"Condition": {
  "StringEquals": {
    "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
    "token.actions.githubusercontent.com:sub": "repo:<owner>/<repo>:environment:<env>"
  }
}
```

Role preview: `sub` là `repo:<owner>/<repo>:pull_request` (thêm `:ref:refs/heads/<branch>` nếu preview chạy khi push). Policy preview ngoài `ReadOnlyAccess`: `s3:GetObject`/`s3:ListBucket` trên bucket state, `kms:Decrypt`/`kms:DescribeKey` trên key secrets.

## Checklist review workflow

- [ ] Không access key; `permissions` tối thiểu; OIDC `StringEquals`
- [ ] Mapping nhánh -> stack giống nhau ở mọi workflow
- [ ] Preview read-only; deploy trong Environment, `needs: preview`
- [ ] Concurrency deploy không cancel; PR cancel
- [ ] Image tag SHA, cùng tag ở preview và deploy, không `:latest`
- [ ] `${{ }}` không nội suy thẳng vào shell; output preview giữ exit code
