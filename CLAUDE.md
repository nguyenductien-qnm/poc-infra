# infra-poc

POC Pulumi (Go) + GitOps trên AWS `ap-southeast-1`: một codebase, 3 stack `dev`/`staging`/`prod`. Web app Go chạy ECS Fargate sau ALB, dữ liệu RDS Postgres.

POC có vài chỗ làm tắt (public subnet, HTTP, Admin role CI...), liệt kê ở README mục "Khác biệt so với production thật". Đó không phải chuẩn để chép khi viết code mới: theo skill.

## Repo và skill

| Thư mục | Nội dung | Skill |
| --- | --- | --- |
| `infra/` | Pulumi Go: `main.go` ghép `internal/{network,platform,data,app,shared}` | `pulumi-go-aws` |
| `app/` | Go service, Dockerfile, image ECR `poc-app` | `go-ecs-service` |
| `.github/workflows/`, `scripts/` | CI/CD, IAM role CI (OIDC) | `gitops-github-actions` |
| mọi chỗ | version, field SDK, giá trị AWS, shape dữ liệu | `verify-before-code` |

## Lệnh

```bash
# Kiểm tra (giống CI)
cd infra && go mod tidy -diff && go vet ./... && go test ./... && go build -o /dev/null .
cd app && go mod tidy -diff && go vet ./... && go build -o /dev/null .

# Preview local (cần credential AWS)
cd infra
pulumi login "s3://pulumi-state-poc-<account>?region=ap-southeast-1"
pulumi stack select dev
pulumi config set imageTag "$(pulumi stack output imageTag)"   # giữ image đang chạy
pulumi preview --diff
```

Không chạy ở local: `pulumi up`, `pulumi destroy`, `pulumi stack output --show-secrets`, mọi lệnh AWS ghi/xoá. `up` chỉ chạy qua CI.

## Quy trình khi sửa code

1. Đọc code có sẵn cùng loại trước (repo có `.codegraph/`: dùng codegraph). Code đang chạy là chuẩn về style.
2. Verify version, field, giá trị AWS, shape dữ liệu theo `verify-before-code`. Không viết theo trí nhớ.
3. Chạy lệnh kiểm tra ở trên. Báo kết quả thật, lệnh nào không chạy được thì nói rõ.
4. Đổi hành vi thì cập nhật README (bảng config, kiến trúc).

## Luật cứng

- Stack chỉ khác nhau ở `infra/Pulumi.<stack>.yaml`. Không `if stack == "prod"`.
- Không secret plaintext trong code/yaml/log. Không access key AWS trong CI.
- Image chỉ tag Git SHA 7 ký tự, không `:latest`. `imageTag` không commit vào yaml.
- Đổi tên logical resource phải có `pulumi.Aliases`. Preview có `replace`/`delete` resource có state thì dừng lại hỏi.
- Không commit account ID, ARN nội bộ vào docs/code mới.

## Môi trường

| Nhánh | Stack | Deploy |
| --- | --- | --- |
| `dev` | `dev` | tự động sau merge |
| `staging` | `staging` | cần approve (GitHub Environment) |
| `main` | `prod` | cần approve |

State: S3 `pulumi-state-poc-<account>`, secret mã hoá KMS `alias/pulumi-poc-key`. ECR `poc-app` dùng chung, tạo ngoài stack.

## Quy ước

- Commit: Conventional Commits tiếng Anh, chữ thường, có scope: `feat(infra)`, `fix(app)`, `fix(ci)`, `refactor(infra)`, `docs`.
- Comment trong code: tiếng Việt không dấu. Docs (`README.md`, `docs/`): tiếng Việt có dấu.

## Bẫy đã gặp

Gặp lỗi mới đáng nhớ thì thêm một dòng.

- `rds.Instance.Endpoint` là `host:port`; host thuần là `Address`.
- Secret RDS managed là JSON `{"username","password"}`; ECS lấy key bằng `valueFrom: <arn>:password::`.
- Đổi `Description` của Security Group gây replace SG.
- Preview mà không set `imageTag` sẽ diff đổi image về `:latest`.
