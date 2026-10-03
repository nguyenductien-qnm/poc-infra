# Lệnh verify

Tất cả là lệnh đọc. Thay `<...>`. Không có `gh` thì dùng `curl https://api.github.com/...` (giới hạn 60 req/giờ khi không có token).

## Mục lục
1. Go toolchain và module
2. Field / API của Go SDK (Pulumi, AWS SDK, thư viện)
3. Pulumi CLI, stack, preview
4. GitHub Actions
5. Docker image
6. Giá trị hợp lệ AWS
7. Shape dữ liệu AWS
8. HTTP API bên thứ ba
9. Tài liệu (khi không có tool)

---

## 1. Go toolchain và module

```bash
go version                                    # toolchain local
grep -E '^(go|toolchain) ' go.mod              # version ngôn ngữ repo yêu cầu
go list -m all | grep <module>                 # version đang dùng thật (sau MVS)
go list -m -versions <module>                  # mọi version có
go list -m -u <module>                         # đang dùng [bản mới hơn]
go list -m -json <module>@latest | jq -r .Version
```

Module major mới đổi import path (`.../sdk/v7` -> `/v8`): `go list -m -versions <module>/v8`.

Tính năng ngôn ngữ/stdlib theo version: https://go.dev/doc/go1.N (release notes), `go doc <pkg>.<Symbol>` cho thấy symbol có tồn tại với toolchain hiện tại không.

## 2. Field / API của Go SDK

`go doc` đọc đúng version trong `go.mod` (chạy trong thư mục module; cần `go mod download` nếu chưa có cache):

```bash
# Args của resource Pulumi AWS
go doc github.com/pulumi/pulumi-aws/sdk/v7/go/aws/<service> <Resource>Args
# Một field cụ thể (kèm doc comment)
go doc github.com/pulumi/pulumi-aws/sdk/v7/go/aws/<service> <Resource>Args.<Field>
# Output của resource (thuộc tính đọc được sau khi tạo)
go doc github.com/pulumi/pulumi-aws/sdk/v7/go/aws/<service> <Resource>
# Kiểu lồng nhau
go doc github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs ServiceDeploymentCircuitBreakerArgs
# Tìm nhanh field theo từ khoá
go doc github.com/pulumi/pulumi-aws/sdk/v7/go/aws/rds InstanceArgs | grep -n -i snapshot
# Option của Pulumi core
go doc github.com/pulumi/pulumi/sdk/v3/go/pulumi Alias
```

Không chắc package nào chứa resource: `ls $(go env GOMODCACHE)/github.com/pulumi/pulumi-aws/sdk/v7@<ver>/go/aws/`.

Cuối cùng compiler là trọng tài: `go build ./... && go vet ./...`.

Thư viện khác (AWS SDK v2, pgx...): cùng cách, `go doc <import path> <Type>`.

## 3. Pulumi CLI, stack, preview

```bash
pulumi version
pulumi stack ls
pulumi config -s <stack>                       # key hiện có (secret hiện [secret])
pulumi stack output -s <stack> --json          # KHÔNG thêm --show-secrets
pulumi preview -s <stack> --diff               # AWS validate giá trị, lộ replace
```

Field nào gây replace: chạy `pulumi preview --diff` sau khi sửa, tìm `+-`/`replace`. Docs Terraform AWS provider ghi "Forces new resource" cho từng argument (Pulumi AWS bridge từ provider này).

## 4. GitHub Actions

```bash
# Bản mới nhất
curl -fsS https://api.github.com/repos/<owner>/<action>/releases/latest | jq -r .tag_name
# Input/output thật của một tag
curl -fsSL https://raw.githubusercontent.com/<owner>/<action>/<tag>/action.yml
# Breaking change trong release notes
curl -fsS https://api.github.com/repos/<owner>/<action>/releases/tags/<tag> | jq -r .body | grep -iE 'breaking|removed|require'
# Commit SHA của tag (khi pin theo SHA)
curl -fsS https://api.github.com/repos/<owner>/<action>/git/ref/tags/<tag> | jq -r .object.sha
```

Có `gh`: `gh api repos/<owner>/<action>/releases/latest --jq .tag_name`.

Context/claim: payload event và claim OIDC (`sub`, `environment`, `ref`) theo docs GitHub "OpenID Connect" và "Webhook events and payloads". Trong workflow có thể in debug: `echo '${{ toJSON(github.event) }}' | jq 'keys'` (không in secret).

## 5. Docker image

```bash
docker manifest inspect <image>:<tag> >/dev/null && echo exists
docker manifest inspect <image>:<tag> | jq -r '.manifests[].platform | "\(.os)/\(.architecture)"'   # platform hỗ trợ
# Docker Hub: liệt kê tag
curl -fsS "https://hub.docker.com/v2/repositories/library/<image>/tags?name=<prefix>&page_size=20" | jq -r '.results[].name'
# ECR của mình
aws ecr describe-images --repository-name <repo> --image-ids imageTag=<tag> --query 'imageDetails[0].imagePushedAt'
```

Base image Go phải ≥ directive `go` trong `go.mod`.

## 6. Giá trị hợp lệ AWS

```bash
# Region/AZ
aws ec2 describe-availability-zones --region <r> --query 'AvailabilityZones[].ZoneName' --output text
# RDS engine version
aws rds describe-db-engine-versions --engine postgres --region <r> \
  --query "DBEngineVersions[?starts_with(EngineVersion, '16')].EngineVersion" --output text
# Instance class có cho engine/version đó ở region đó không
aws rds describe-orderable-db-instance-options --engine postgres --engine-version <v> \
  --db-instance-class <class> --region <r> --query 'OrderableDBInstanceOptions[0].AvailabilityZones[].Name'
# ElastiCache
aws elasticache describe-cache-engine-versions --engine redis --query 'CacheEngineVersions[].EngineVersion'
# EC2 instance type có ở AZ nào
aws ec2 describe-instance-type-offerings --location-type availability-zone \
  --filters Name=instance-type,Values=<type> --region <r> --query 'InstanceTypeOfferings[].Location'
# ALB SSL policy
aws elbv2 describe-ssl-policies --query 'SslPolicies[].Name' --output text
# Managed policy tồn tại
aws iam get-policy --policy-arn arn:aws:iam::aws:policy/<path>/<name> --query 'Policy.Arn'
# Validate IAM policy (action sai, ARN sai format, quyền quá rộng)
aws accessanalyzer validate-policy --policy-type IDENTITY_POLICY --policy-document file://policy.json \
  --query 'findings[].[findingType,issueCode,findingDetails]'
# Quota
aws service-quotas list-service-quotas --service-code <ecs|rds|vpc> --query 'Quotas[].[QuotaName,Value]' --output table
```

Không có API, phải đọc docs: tổ hợp CPU/memory Fargate, giới hạn độ dài tên (ALB/TG 32 ký tự), action IAM của từng service (Service Authorization Reference).

## 7. Shape dữ liệu AWS

```bash
# Shape output mà không cần gọi thật
aws <service> <operation> --generate-cli-skeleton output
# Gọi thật, chỉ xem cấu trúc
aws rds describe-db-instances --query 'DBInstances[0]' | jq 'keys'
aws rds describe-db-instances --query 'DBInstances[0].Endpoint'
# Secret: CHỈ xem tên key, không in giá trị
aws secretsmanager get-secret-value --secret-id <arn> --query SecretString --output text | jq 'keys'
aws secretsmanager describe-secret --secret-id <arn>          # metadata, không có giá trị
# Output Pulumi
pulumi stack output --json | jq 'keys'
```

## 8. HTTP API bên thứ ba

- Có OpenAPI/JSON schema chính thức: đọc schema đúng version API đang gọi.
- Có endpoint an toàn (GET, sandbox): `curl -fsS <url> | jq 'keys'` hoặc `jq '.[0]'`, chỉ endpoint không đổi dữ liệu, không gửi credential thật trong lệnh in ra màn hình.
- Struct Go viết theo sample thật; field có thể vắng mặt dùng pointer hoặc `omitempty`; không dựa vào thứ tự key.

## 9. Tài liệu (khi không có tool)

Lấy bằng WebFetch/WebSearch ngay trong session, ghi lại URL đã dùng:

| Cần | Nguồn |
| --- | --- |
| Field/arg Pulumi | `https://www.pulumi.com/registry/packages/aws/api-docs/<service>/<resource>/` |
| Force-new, ràng buộc arg | Terraform AWS provider docs của resource tương ứng |
| Giá trị/limit AWS | AWS docs của service (User Guide, API Reference) |
| IAM action/ARN | AWS "Service Authorization Reference" |
| Fargate CPU/memory | AWS ECS docs "Task size" |
| Go release | `https://go.dev/doc/devel/release` |
| Action input | `action.yml` tại tag trên GitHub |
