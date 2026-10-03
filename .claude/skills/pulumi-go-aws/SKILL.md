---
name: pulumi-go-aws
description: Best practices for writing AWS infrastructure as code with Pulumi in Go - multi-stack layout (dev/staging/prod from one codebase), ComponentResource packages, naming and tagging, replace-safety, stateful resource protection, IAM least privilege, Security Groups, secrets, config per stack, testing pure logic. Use this skill whenever writing, refactoring or reviewing Pulumi Go code, adding any AWS resource (VPC, subnet, ALB, ECS, RDS, EFS, IAM, S3, KMS...), adding a stack config key, or designing an infra/ folder - even if the user just says "thêm resource", "tạo module network", "viết IaC" without naming Pulumi.
---

# Pulumi Go trên AWS

Ba mục tiêu mọi quy tắc dưới đây phục vụ:
1. **Một codebase, N stack.** Môi trường chỉ khác nhau ở file config, không ở code.
2. **Preview đọc được.** Diff gom theo module, người duyệt hiểu ngay thay đổi gì.
3. **Không phá dữ liệu ngoài ý muốn.** Không `replace`/`delete` bất ngờ, resource có state được bảo vệ.

> **Trước khi viết code:** version, tên field SDK, giá trị hợp lệ của AWS và shape dữ liệu phải verify theo skill `verify-before-code`, không lấy từ trí nhớ. Repo đã có code cùng loại thì đọc code đó làm chuẩn; mẫu trong skill chỉ là khung cho pattern dễ viết lệch.

Danh sách hardening cho production: [references/production.md](references/production.md). Đọc khi thiết kế network, ALB, DB, IAM cho môi trường thật.

## Cấu trúc thư mục

```
infra/
  main.go                 Đọc config stack, gọi các package theo thứ tự phụ thuộc, export output
  Pulumi.yaml             Project
  Pulumi.<stack>.yaml     Config từng môi trường (chỗ DUY NHẤT stack khác nhau)
  internal/
    network/              VPC, subnet, routing, endpoint
    platform/             Cluster, ALB, log group, SG dùng chung
    data/                 RDS, EFS, cache... (resource có state)
    app/                  IAM role của service, task definition, service
    shared/               Hằng số dùng chung, Tags(), helper SG/policy
```

Chia package theo **domain và vòng đời**: thứ đổi thường xuyên (app) tách khỏi thứ hiếm đổi và nguy hiểm (data, network). Package không import lẫn nhau (trừ `shared`); dữ liệu đi qua `main.go`.

State backend: S3 (bật versioning) + secrets provider KMS (`awskms://alias/...`), hoặc Pulumi Cloud. Không dùng backend local cho môi trường dùng chung.

## Quy tắc

### 1. Config theo stack
- Không có `if ctx.Stack() == "prod"`. Cần khác giữa env thì thêm config key.
- Key mới thêm vào **mọi** `Pulumi.<stack>.yaml`, đọc **chỉ trong `main.go`** bằng `cfg.Require*` (bắt buộc, fail sớm nếu thiếu) hoặc `cfg.Get*` + default rõ ràng, rồi truyền qua `Args`. Package trong `internal/` không đọc config.
- Giá trị CI set lúc deploy (image tag theo Git SHA...) không commit vào yaml.
- Secret: `pulumi config set --secret` (mã hoá bằng secrets provider), đọc bằng `cfg.RequireSecret`. Tốt hơn nữa: để AWS tự sinh và quản lý (vd RDS `ManageMasterUserPassword`), app đọc từ Secrets Manager. Không plaintext trong code/yaml/output.
- Có bảng config theo stack trong README, cập nhật khi thêm key.

### 2. Mỗi package là một ComponentResource
- `module.go`: `Args`, struct component (nhúng `pulumi.ResourceState` + field output), `New(ctx, name, args, opts...)`.
- Type token `<project>:<package>:<Component>`.
- Mọi resource con dùng `opt := pulumi.Parent(comp)`; cuối `New` gọi `ctx.RegisterResourceOutputs`.
- `New` chỉ điều phối, đánh số bước kèm tên file. Resource thật nằm ở file theo nhóm (`vpc.go`, `alb.go`, `rds.go`), hàm private `newXxx(ctx, name, stack string, ..., opt pulumi.ResourceOption)` trả struct private `xxxResult`. File giữ dưới ~150 dòng.
- Field `Args`: `pulumi.XxxInput` cho giá trị có thể là Output; kiểu Go thuần cho giá trị cần lúc compile (CIDR để tính toán, cờ `Protect`).
- Output giữa package trong cùng project: biến Go qua `Args`. Giữa project khác nhau: `StackReference`.

### 3. Tên và tag
- Tên logical: `fmt.Sprintf("%s-<suffix>", name)`. Không nhét stack vào tên logical (state đã tách theo stack).
- Tên vật lý (`Name`, `Identifier`, `Family`...): `<logical>-<stack>`, vì nhiều stack có thể chung account. Để Pulumi auto-name khi AWS không yêu cầu tên cố định. Chú ý giới hạn độ dài (ALB/TG 32 ký tự).
- Mọi resource hỗ trợ tag phải có tag qua **một helper** `shared.Tags(name, stack, extra...)` (tối thiểu `Name`, `Stack`; hệ thống thật thêm `Project`, `Owner`, `CostCenter`). Hoặc dùng provider `DefaultTags` cho tag chung, helper cho `Name`.

### 4. An toàn trước replace
- **Đổi tên logical = xoá + tạo mới.** Bắt buộc đổi thì thêm `pulumi.Aliases(...)` với tên cũ, preview phải sạch `replace`.
- Chuyển resource sang parent khác (vd đưa vào component) cũng đổi URN, cần alias `ParentURN`/`Parent`.
- Field force-new giữ nguyên trừ khi chủ đích: SG `Description`, RDS `Identifier`, subnet/VPC `CidrBlock`, `Name` của nhiều resource. Comment giải thích khi giữ giá trị cũ.
- Resource cần thay thế không downtime: `pulumi.DeleteBeforeReplace(false)` (mặc định) + tên vật lý auto-name; tên cố định thì phải chấp nhận delete-before-replace.
- Đọc kỹ preview: `~` update ok, `+-`/`-` trên resource có state = dừng lại hỏi.

### 5. Resource có state (RDS, EFS, S3, DynamoDB, KMS)
- `pulumi.Protect(args.ProtectStateful)`; cờ bật ở prod.
- Bật deletion protection của chính AWS (`DeletionProtection`), RDS `SkipFinalSnapshot: !protect` + `FinalSnapshotIdentifier` khi protect.
- Mã hoá at-rest (`StorageEncrypted`, KMS), backup/retention bật ở prod.
- Đặt ở private subnet, không public.

### 6. Network và Security Group
- SG ingress tham chiếu **SG nguồn** (`SecurityGroups: {sourceSgID}`), không mở theo CIDR rộng. Chỉ ALB internet-facing nhận `0.0.0.0/0` trên 443 (và 80 để redirect).
- Egress dùng helper chung; prod cân nhắc thu hẹp egress.
- Mỗi rule có `Description`.
- Tính CIDR bằng hàm thuần có test (xem mục 9), không hardcode từng subnet; CIDR VPC lấy từ config, không trùng giữa stack/account cần peering.
- Dùng VPC endpoint (Gateway cho S3/DynamoDB, Interface cho ECR/Secrets Manager/Logs) để giảm phụ thuộc NAT.

### 7. IAM
- Một role cho một mục đích: ECS **execution role** (kéo image, ghi log, đọc secret) tách **task role** (quyền của code trong container).
- Least privilege: action cụ thể, `Resource` là ARN cụ thể (truyền từ Output qua `ApplyT`), tránh `"*"`. Managed policy AWS chỉ dùng cho service-role chuẩn (vd `AmazonECSTaskExecutionRolePolicy`).
- Policy JSON dựng bằng struct + `json.Marshal` qua helper (`allowPolicy`, `assumeRolePolicy`), không ghép chuỗi, không heredoc trong Go.
- Trust policy chỉ đúng principal cần (service hoặc OIDC với `StringEquals`).

### 8. Output và giá trị bất đồng bộ
- Một Output: `x.ApplyT(func(v string) string {...}).(pulumi.StringOutput)`. Nhiều Output: `pulumi.All(a, b).ApplyT(func(v []any) ...)`. Chuỗi đơn giản: `pulumi.Sprintf`.
- Không tạo resource bên trong `ApplyT` (preview không thấy được).
- JSON phức tạp (container definitions...) dựng bằng struct trong `ApplyT`.
- Export ở `main.go`, tên camelCase; export đủ để CI/script khác dùng (URL, ARN, image tag hiện tại).

### 9. Logic thuần tách riêng và có test
- Tính toán không cần AWS (chia CIDR, validate config, dựng tên) viết thành hàm thuần ở file riêng, test table-driven cạnh nó, có cả case lỗi. Không cần Pulumi mock cho phần này.
- Logic phụ thuộc resource (vd SG không mở 0.0.0.0/0 ở port DB) có thể test bằng `pulumi.WithMocks` khi đáng giá.

### 10. Error, comment, style
- Bọc lỗi `fmt.Errorf("creating <thứ>: %w", err)`, chữ thường, động từ -ing (`creating`, `attaching`, `registering`). Lỗi helper con đã bọc thì trả thẳng.
- Hằng số dùng ≥2 package (port, DB name/user) đặt ở `shared`.
- Comment giải thích *vì sao*, không lặp lại code. Theo ngôn ngữ team (repo hiện tại: tiếng Việt không dấu trong code, có dấu trong docs).
- Import 3 nhóm: stdlib, thư viện ngoài, package nội bộ.

## Mẫu

Chỉ những pattern hay bị viết lệch. Phần còn lại viết theo quy tắc ở trên.

**Component (`module.go`):**

```go
type <Name> struct {
	pulumi.ResourceState
	SomeArn pulumi.StringOutput
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*<Name>, error) {
	stack := ctx.Stack()
	comp := &<Name>{}
	if err := ctx.RegisterComponentResource("<project>:<pkg>:<Name>", name, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering <pkg> component: %w", err)
	}
	opt := pulumi.Parent(comp)

	// 1. <Buoc> (<file>.go)
	res, err := newXxx(ctx, name, stack, args, opt)
	if err != nil {
		return nil, err
	}

	comp.SomeArn = res.arn
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"someArn": comp.SomeArn}); err != nil {
		return nil, fmt.Errorf("registering <pkg> outputs: %w", err)
	}
	return comp, nil
}
```

**Resource trong file con (tên logical, tên vật lý, tag, parent):**

```go
func newXxx(ctx *pulumi.Context, name, stack string, args *Args, opt pulumi.ResourceOption) (*xxxResult, error) {
	resName := fmt.Sprintf("%s-<suffix>", name)
	r, err := <svc>.New<Resource>(ctx, resName, &<svc>.<Resource>Args{
		Name: pulumi.String(resName + "-" + stack), // chi khi can ten co dinh
		Tags: shared.Tags(resName, stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating <resource>: %w", err)
	}
	return &xxxResult{arn: r.Arn}, nil
}
```

**Policy theo ARN là Output (struct, không ghép chuỗi):**

```go
Policy: args.SecretArn.ToStringOutput().ApplyT(func(arn string) string {
	return allowPolicy(arn, "secretsmanager:GetSecretValue") // allowPolicy = struct + json.Marshal
}).(pulumi.StringOutput),
```

**Resource có state và đổi tên an toàn:**

```go
}, opt, pulumi.Protect(args.ProtectStateful))

}, opt, pulumi.Aliases([]pulumi.Alias{{Name: pulumi.String(fmt.Sprintf("%s-<ten-cu>", name))}}))
```

## Quy trình

1. Đọc code liên quan trước (dùng codegraph nếu repo có `.codegraph/`).
2. Viết theo quy tắc + mẫu.
3. Kiểm tra:
   ```bash
   cd infra && go mod tidy -diff && go vet ./... && go test ./... && go build -o /dev/null .
   ```
4. `pulumi preview --diff` trên stack dev nếu có credential. Giữ nguyên các giá trị CI set (vd image tag hiện tại lấy từ `pulumi stack output`) để diff chỉ phản ánh thay đổi hạ tầng. **Không `pulumi up`/`destroy` từ máy local** trên stack dùng chung: `up` đi qua CI.
5. Cập nhật README (bảng config, kiến trúc) khi đổi hành vi.

## Checklist review

- [ ] Không logic theo tên stack; key config mới có ở mọi stack + README
- [ ] Resource mới: parent `opt`, tag qua helper, tên vật lý có stack
- [ ] Không đổi tên logical / field force-new (hoặc có alias + lý do)
- [ ] Resource có state: `Protect`, deletion protection, encryption, private subnet
- [ ] SG theo SG nguồn, có description; IAM action + ARN cụ thể
- [ ] JSON dựng bằng struct; không tạo resource trong `ApplyT`
- [ ] Không secret plaintext; logic thuần có test; vet/test/tidy sạch
