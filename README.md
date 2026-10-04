# infra-poc: Pulumi (Go) + GitOps trên AWS

POC chứng minh cách tổ chức IaC bằng Pulumi Go: **một codebase, 3 stack** (`dev`, `staging`, `prod`), mỗi stack chỉ khác file `Pulumi.<stack>.yaml`. Kế hoạch chi tiết xem [docs/PLAN.md](docs/PLAN.md), hướng dẫn cài đặt xem [docs/SETUP.md](docs/SETUP.md), đề xuất gốc xem [docs/pulumi-proposal.md](docs/pulumi-proposal.md).

## Kiến trúc

```
.
├── app/                    Web app Go (ghi note vào Postgres), build thành image ECR poc-app
├── infra/                  Một Go module, hai Pulumi project
│   ├── bootstrap/          Project infra-bootstrap: nền cho workload, admin chạy tay (docs/BOOTSTRAP.md)
│   │   ├── main.go
│   │   ├── Pulumi.poc.yaml Config 1 stack cho 1 account
│   │   └── internal/       Chỉ bootstrap import được (Go chặn lúc build)
│   │       ├── statebackend/  S3 state workload, KMS key mã hoá secret
│   │       ├── registry/      ECR poc-app
│   │       └── ciiam/         GitHub OIDC provider, role preview/deploy
│   ├── workload/           Project infra-poc: ứng dụng, CI deploy
│   │   ├── main.go         Đọc config stack, ghép 4 package
│   │   └── Pulumi.<stack>.yaml Config từng môi trường
│   └── internal/           Package dùng lại được
│       ├── network/        VPC, 2 public + 2 private subnet, S3 Gateway Endpoint
│       ├── platform/       ECS Cluster, ALB (HTTP), Security Group ALB/App, CloudWatch Logs
│       ├── data/           RDS PostgreSQL 16 (password trong Secrets Manager), EFS
│       ├── app/            IAM exec/task role, Task Definition, ECS Fargate Service
│       └── shared/         Hằng số, helper Tags/egress SG, kiểm tra account
├── .github/workflows/      pr-check.yml (preview), deploy.yml (up) cho workload
├── docs/                   PLAN, SETUP, BOOTSTRAP, proposal
└── scripts/
    └── setup-bootstrap-backend.sh  Tạo bucket lưu state của bootstrap (resource tạo tay duy nhất)
```

Mỗi package trong `internal/` (trừ `shared`) là một Pulumi **ComponentResource** (`infra-poc:<package>:<Tên>`), mọi resource AWS là con của component đó, nên `pulumi preview` và console nhóm resource theo module.

Hai project tách vì khác vòng đời và quyền: workload đổi thường xuyên, deploy qua CI; bootstrap hiếm đổi, chỉ admin chạy, nên pipeline workload không sửa được role CI hay xoá state/ECR. State workload ở bucket `pulumi-state-poc-<account>`, state bootstrap ở bucket riêng `pulumi-bootstrap-poc-<account>`.

Luồng truy cập: Internet → ALB (port 80) → Fargate task (public subnet) → RDS (private subnet, chỉ nhận từ App SG).

## Config theo stack

| Config               | dev          | staging      | prod         |
| -------------------- | ------------ | ------------ | ------------ |
| `expectedAccount`    | secret       | secret       | secret       |
| `vpcCidr`            | 10.10.0.0/16 | 10.20.0.0/16 | 10.30.0.0/16 |
| `desiredCount`       | 1            | 1            | 2            |
| `dbInstanceClass`    | db.t4g.micro | db.t4g.micro | db.t4g.small |
| `protectStateful`    | false        | false        | true         |
| `deletionProtection` | false        | false        | true         |
| `ecrRepositoryName`  | poc-app      | poc-app      | poc-app      |

- `expectedAccount`: account AWS mà stack được phép chạy, lưu dạng secret (mã hoá KMS) để không lộ account ID. Credentials thuộc account khác thì preview/up dừng ngay, chưa tạo gì. Set bằng `pulumi config set --secret expectedAccount <account-id>`.
- `imageTag` do CI set theo Git SHA (7 ký tự) mỗi lần deploy, không lưu trong `Pulumi.<stack>.yaml`. Thiếu hoặc bằng `latest` thì preview/up báo lỗi, không tự dùng `:latest`.
- `ecrRepositoryName`: repo ECR dùng chung, do bootstrap quản lý.

## CI/CD

> **Đang tạm tắt trigger tự động** (chỉ chạy tay bằng `workflow_dispatch`) trong lúc chuyển sang bootstrap/workload. Bật lại sau khi `up` bootstrap theo [docs/BOOTSTRAP.md](docs/BOOTSTRAP.md): bỏ comment khối `pull_request` / `push` trong `.github/workflows/*.yml`. Mô tả dưới đây là hành vi khi bật.

- **PR** vào `dev` / `staging` / `main`: kiểm tra `go mod tidy`, build app, `go vet` + `go test` infra, rồi `pulumi preview --diff` bằng role **read-only** (giữ `imageTag` đang chạy để diff chỉ phản ánh hạ tầng). Kết quả cập nhật vào 1 comment trên PR; preview lỗi thì job fail.
- **Push** vào `dev` / `staging` / `main` (bỏ qua thay đổi chỉ ở `*.md`, `docs/`), 2 job nối tiếp:
  1. `preview`: role read-only, preview với image tag mới, diff in ra **Job Summary**.
  2. `deploy`: chạy trong GitHub Environment cùng tên (`main` → `prod`). Staging/prod có *Required reviewers*, người duyệt xem diff ở job `preview` rồi mới approve. Sau đó build & push image (chỉ tag Git SHA, không dùng `:latest`) và `pulumi up`.
- Xác thực AWS bằng **OIDC**, không lưu access key. State trên S3 `pulumi-state-poc-<account>`, secret mã hóa bằng KMS `alias/pulumi-poc-key`.

### IAM role cho CI

Trust policy dùng `StringEquals` với OIDC `sub` dạng immutable (repo tạo sau 15/07/2026):
`repo:<owner>@<owner_id>/<repo>@<repo_id>:...`

| Role                       | Tin tưởng `sub`         | Quyền                      | GitHub secret              |
| -------------------------- | ----------------------- | -------------------------- | -------------------------- |
| `poc-dev-preview-role`     | `pull_request`, ref dev | ReadOnly + state/KMS       | `AWS_ROLE_DEV_PREVIEW`     |
| `poc-dev-deploy-role`      | `environment:dev`       | Admin                      | `AWS_ROLE_DEV_DEPLOY`      |
| `poc-staging-preview-role` | `pull_request`, ref staging | ReadOnly + state/KMS   | `AWS_ROLE_STAGING_PREVIEW` |
| `poc-staging-deploy-role`  | `environment:staging`   | Admin                      | `AWS_ROLE_STAGING_DEPLOY`  |
| `poc-prod-preview-role`    | `pull_request`, ref main | ReadOnly + state/KMS      | `AWS_ROLE_PROD_PREVIEW`    |
| `poc-prod-deploy-role`     | `environment:prod`      | Admin                      | `AWS_ROLE_PROD_DEPLOY`     |

OIDC provider, các role và policy preview do project bootstrap quản lý (package `ciiam`), thay cho script `setup-ci-roles.sh` cũ. Đổi quyền CI: sửa code, mở PR, admin preview/up theo [docs/BOOTSTRAP.md](docs/BOOTSTRAP.md). ARN role cho GitHub secret lấy từ output `previewRoleArns` / `deployRoleArns` của stack bootstrap.

## Chạy local

```bash
cd infra/workload
pulumi login "s3://pulumi-state-poc-<account>?region=ap-southeast-1"
pulumi stack select dev
# Bắt buộc: dùng image đang chạy (thiếu imageTag thì preview báo lỗi)
pulumi config set imageTag "$(pulumi stack output imageTag)"
pulumi preview
```

Quy ước: local chỉ chạy `preview` workload, mọi `up` workload đi qua CI. Bootstrap là ngoại lệ: admin chạy tay theo [docs/BOOTSTRAP.md](docs/BOOTSTRAP.md). Quy ước này chưa được ép bằng IAM: credential admin trên máy vẫn `up` được; muốn ép thì dev dùng role read-only (kịch bản `AccessDenied` ở docs/PLAN.md Phase 6).

## Khác biệt so với production thật

- Fargate task chạy ở **public subnet** có public IP thay vì private subnet + NAT/TGW.
- ALB chỉ **HTTP**, chưa có ACM/HTTPS (cần domain).
- RDS **single-AZ**, chưa có AWS Backup.
- Role deploy dùng `AdministratorAccess` cho nhanh; prod thật cần thu hẹp quyền.
- Task role có `bedrock:InvokeModel` để chứng minh IAM, app không gọi Bedrock.
- PR preview chạy code của PR (chưa review) bằng role preview có `ReadOnlyAccess` + `kms:Decrypt`. Chấp nhận cho POC để có diff ngay ở PR: chỉ người trong team mở PR, PR từ fork không được GitHub cấp OIDC token. Production nên chặn bằng approval trước khi job preview nhận credentials.
- Action trong workflow pin theo commit SHA nhưng vẫn ở major cũ (`checkout@v4`, `setup-go@v5`...). Nâng major làm riêng.

## Pulumi AI pack (#22)

Pack lấy cách tách router/references/workflows/agents/hooks từ [Taurus](https://github.com/KKloudTarus/taurus-skill), giới hạn ở repo này. Nội dung chung ở [`.agents/skills/pulumi`](.agents/skills/pulumi/SKILL.md); `.claude/skills/pulumi` là adapter. [Contract hiện tại](.agents/skills/pulumi/references/current-contract.md) ghi các quyết định PoC của owner ở #15–#19; production hardening là proposal riêng. Docs #14/#20 vẫn do owner hoàn thiện sau pack, pilot chưa được chạy bởi PR này.

- **Adopt:** Codex `$pulumi adopt <workload/env>`; Claude `/pulumi adopt <workload/env>`. Trả config/composition, ownership và input còn thiếu. Existing network chưa có API trong PoC: pack báo giới hạn và đề xuất task riêng.
- **Verify:** `$pulumi verify <diff/scope>` hoặc `/pulumi verify <diff/scope>`. Primary chạy local checks phù hợp, gọi `pulumi-component-reviewer` cho identity/composition hoặc `pulumi-delivery-reviewer` cho IAM/state/CI. Mặc định một reviewer; nếu không có/không được phép dùng subagents thì ghi self-review rõ ràng.
- **Reviewers:** Codex dùng `.codex/agents/*.toml` với sandbox read-only; Claude dùng `.claude/agents/*.md` chỉ có Read/Grep/Glob. Cả hai đọc contract chung; không tự chạy checks/cloud/apply. Codex sandbox hạn chế ghi filesystem, không tự biến MCP của session thành read-only; prompt reviewer cấm dùng cloud/MCP và primary phải kiểm tra tool surface trong runtime đang dùng.

### Enable và fallback

Yêu cầu Git và Python **3.11+**, không thêm pip dependency. Trên Windows cần `python.exe` thật trên PATH (không dùng WindowsApps `python3` shim); Codex `commandWindows` chạy qua cmd.exe. Trên Linux/macOS Codex dùng `python3`; Claude exec-form gọi `python`. Cả hai tìm repo root bằng Git: `${CLAUDE_PROJECT_DIR}` có thể là thư mục con nơi mở Claude. Không cần symlink, global installer, sửa HOME, git hooks hoặc tự bật workflow AWS.

Start phiên mới từ checkout. Codex cần trust project `.codex` rồi mở `/hooks`, đọc hai command và trust đúng definition; hook thay đổi cần review lại. Claude đọc project `.claude/settings.json` sau project trust; kiểm tra `/hooks` và bảo đảm hooks không bị disabled trong settings của bạn. Nếu mở Claude từ thư mục con, truyền `--settings "<repo-root>/.claude/settings.json"`: bản CLI đã thử không tự load hook ở root trong trường hợp này. Không tự persist trust hoặc dùng sandbox/approval bypass để nghiệm thu. Gọi reviewer bằng tên đầy đủ trong yêu cầu; adapter không tự đổi model/effort người dùng đã chọn.

SessionStart chạy verifier tĩnh; PostToolUse chỉ nhận Edit/Write (Codex thêm apply_patch), lọc file thuộc pack/infra/app/workflows rồi check nhẹ và nhắc checks/review cũ có thể stale. Header apply_patch nhận cả LF và CRLF, kể cả file scope liệt kê riêng như README và scripts verifier. Không build/test toàn repo mỗi edit, không format hoặc cloud writes. Shell, MCP, editor ngoài runtime và tool payload chưa hỗ trợ không được cover; chạy verifier thủ công trước bàn giao:

```bash
python -B scripts/verify-pulumi-pack.py
python -B -m unittest discover -s scripts -p test_pulumi_pack.py
```

Hai lệnh trên cũng chạy trong PowerShell ở repo root. Verifier kiểm tra links/frontmatter, agent/adapters, hook wiring và remote action SHA; không chứng minh runtime behavior/IAM/live state. Hook có timeout 5 giây và check budget 2 giây, input tối đa 64 KiB; diagnostic không echo payload. Hook lỗi/malformed/timeout/thiếu Python, disabled hoặc chưa trust **không là pass**. Manual verifier trả nonzero khi lỗi và hoạt động không cần hooks/subagents; hook chỉ advisory, không phải approval/security boundary.

### Mức đã kiểm chứng

- Windows: Python 3.14.7, Codex CLI 0.156.1, Claude Code 2.1.284. Local verifier/tests đã chạy, gồm command từ config, path dấu cách, thư mục con, payload/drift/link lỗi, missing interpreter và timeout budget. CI `Pulumi Pack Checks` kiểm tra cơ chế AWS-free trên Ubuntu/Windows; không gọi inference hoặc thay thế runtime smoke.
- Native Windows: cả hai CLI đã đọc router và chạy bốn case — adopt network mới với input pending; existing VPC báo API chưa hỗ trợ; rename stateful thiếu alias báo nguy cơ identity/physical-name replacement; OIDC wildcard bị yêu cầu sửa. Cả hai đã gọi thành công đúng `pulumi-component-reviewer` và `pulumi-delivery-reviewer`; không dùng generic agent thay thế. Codex dùng model hợp lệ trong `codex debug models`, phiên `exec` thông thường: `--ephemeral` ở bản đã thử làm spawn reviewer lỗi `no thread with id`. Claude dùng user + project settings để giữ cấu hình provider/login; chỉ load project settings từng gây `Not logged in`.
- Claude SessionStart và PostToolUse đã chạy thật trong bản sao Git tạm có path dấu cách, từ `infra/workload` với `--settings` trỏ root: file `app/` nhận nhắc stale, file ngoài scope im lặng. Smoke này bắt được lỗi dùng `${CLAUDE_PROJECT_DIR}`; launcher Git-root và regression test đã sửa lỗi.
- Codex native hooks **chưa nghiệm thu**: SessionStart chưa phát context trong phép thử project layer tạm; thử PostToolUse bị sandbox read-only chặn cả hai apply_patch trước khi ghi. One-off hook trust chỉ áp dụng cho definition đã review trong bản sao tạm, không persist trust hoặc bypass sandbox/approval. Chạy command trực tiếp/JSON-TOML checks không thay cho native pass; #22 còn mở đến khi maintainer xác minh hook delivery trong runtime phù hợp.
- CI `Pulumi Pack Checks` đang chờ maintainer approve workflow từ fork, chưa có hosted pass. Chưa thử native Linux/macOS runtime; local/CI cơ chế không đủ để công bố support native các OS này. Các findings ở smoke là source/proposal review, chưa chạy Go hoặc cloud/preview/pilot.

Mapping: #14 quyết định kiến trúc; #15–#19 implementation và owner exceptions; #22 pack; #20 docs/pilot/sync sau đó. Pack không đóng thay các acceptance live/pilot, không mở lại #21 hoặc biến production backlog thành yêu cầu triển khai PoC.

## Dọn dẹp

```bash
cd infra/workload
pulumi destroy -s staging && pulumi destroy -s dev
# prod: tắt protectStateful/deletionProtection, pulumi up, rồi mới destroy
aws elbv2 describe-load-balancers   # kiểm tra tài nguyên sót
aws rds describe-db-instances
```

Destroy workload không đụng tới bootstrap. Bucket state, KMS key, ECR được đánh dấu giữ lại khi destroy bootstrap; muốn xoá hẳn phải gỡ `protect` và xoá tay có chủ đích.
