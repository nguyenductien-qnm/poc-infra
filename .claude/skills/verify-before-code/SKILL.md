---
name: verify-before-code
description: Verify versions, SDK fields, valid AWS values and API/data shapes against real sources BEFORE writing code, instead of trusting memory. Use this skill whenever code you are about to write depends on a version number (Go, Go modules, Pulumi provider, GitHub Action, Docker base image, RDS/ElastiCache engine, runtime), a struct field or argument name of an SDK/provider, an allowed value or limit of a cloud service (instance class, Fargate CPU/memory, SSL policy, name length), or the shape of data returned by an API/CLI/secret/stack output - even when you feel sure, and even if the user does not ask to verify.
---

# Verify trước khi code

Trí nhớ của model về version, tên field, giá trị hợp lệ và shape dữ liệu **luôn có khả năng đã cũ hoặc sai**: SDK đổi major, action lên v6 trong khi model nhớ v4, field đổi tên giữa provider v6 và v7, AWS bỏ engine version cũ. Code sai kiểu này thường compile được hoặc chỉ lỗi lúc `pulumi up`/runtime, nên đắt nhất để phát hiện muộn.

Quy tắc gốc: **mọi chi tiết thuộc 4 nhóm dưới đây phải có nguồn kiểm chứng trong session hiện tại**, không dựa vào trí nhớ.

1. **Version**: Go toolchain, module, Pulumi CLI/provider, GitHub Action, image tag, engine DB, runtime.
2. **Field / API của SDK**: tên field trong `XxxArgs`, kiểu (`StringPtrInput` hay `StringInput`), tên output (`Endpoint` hay `Address`), tên hàm.
3. **Giá trị hợp lệ của service**: instance class, tổ hợp CPU/memory Fargate, SSL policy, IAM action, giới hạn độ dài tên, AZ.
4. **Shape dữ liệu**: JSON AWS CLI/API trả về, nội dung secret, `pulumi stack output`, payload HTTP API, claim OIDC.

Lệnh tra chi tiết cho từng nhóm: [references/commands.md](references/commands.md).

## Thứ tự nguồn tin (cao -> thấp)

1. **Repo hiện tại**: `go.mod`/`go.sum`, lock file, workflow, Dockerfile, code đang chạy. Version repo đã pin là mặc định, trừ khi task là nâng cấp.
2. **Tool cục bộ đúng version đang dùng**: `go doc` trên module trong cache, `--help`, `--generate-cli-skeleton`, compiler (`go build`, `go vet`).
3. **Hệ thống thật, chỉ đọc**: `aws ... describe-*`/`list-*`/`get-*`, `pulumi stack output`, `pulumi preview`.
4. **Tài liệu chính thức lấy ngay bây giờ**: release notes/`action.yml` trên GitHub, AWS docs, Pulumi Registry, go.dev. Dùng bản docs khớp version đang dùng.
5. **Trí nhớ**: chỉ để biết *cần tra cái gì*, không phải đáp án.

## Cách làm

1. **Trước khi viết**: liệt kê các chi tiết thuộc 4 nhóm trên mà đoạn code sắp viết phụ thuộc vào.
2. **Tra** từng mục bằng nguồn cao nhất có thể (gom nhiều lệnh vào một lần chạy cho nhanh).
3. **Viết code** dựa trên kết quả tra.
4. **Xác nhận lại bằng tool**: `go build ./... && go vet ./...` (bắt sai tên field/kiểu), `pulumi preview --diff` (bắt giá trị AWS từ chối, replace ngoài ý muốn), `docker build`, chạy thử request.
5. **Báo cáo** trong câu trả lời: mục nào đã verify bằng gì, mục nào không verify được.

```
Đã verify:
- pulumi-aws v7.48.0 (go.mod): rds.InstanceArgs có FinalSnapshotIdentifier, MultiAz (go doc)
- Postgres 16.15 có ở ap-southeast-1 (aws rds describe-db-engine-versions)
- aws-actions/configure-aws-credentials mới nhất v6.3.0, input role-to-assume/aws-region vẫn còn (action.yml@v6.3.0)
Chưa verify được:
- Tổ hợp Fargate 0.5 vCPU/3GB: không có API, chưa đọc docs. Cần kiểm tra trước khi deploy.
```

## Ràng buộc an toàn khi tra

- **Chỉ lệnh đọc**: `describe`, `list`, `get`, `help`, `preview`, `go doc`, `go list`, `curl` GET. Không `create`/`put`/`delete`/`up`/`destroy`/`config set` để "thử xem".
- **Không in giá trị secret**: secret chỉ xem tên key (`... | jq 'keys'`), không dùng `pulumi stack output --show-secrets`, không `cat` file `.env`. Account ID, ARN nội bộ cũng không chép vào code/docs public.
- Lệnh tra cần credential mà không có: không xin credential, chuyển sang nguồn 4 (docs) và ghi rõ.

## Khi không verify được

Không đoán rồi viết như thể chắc chắn. Chọn một trong các cách sau:
- Dùng version/giá trị repo đang pin (đã chạy được) thay vì "mới nhất" từ trí nhớ.
- Để placeholder rõ ràng và comment `// TODO(verify): <cần kiểm tra gì, ở đâu>`.
- Hỏi user nếu quyết định ảnh hưởng lớn (nâng major, đổi engine version prod).

## Nâng version

Bản mới nhất chưa chắc là bản nên dùng. Khi tăng **major** (action `@v4`->`@v6`, module `/v7`->`/v8`, Postgres 16->17):
- Đọc release notes/changelog các bản ở giữa, tìm "breaking", "removed", "requires".
- Kiểm tra input/field đang dùng còn tồn tại ở bản mới (`action.yml` của tag mới, `go doc` sau khi `go get`).
- Engine DB: major upgrade là thay đổi có downtime/replace, không gộp chung với thay đổi khác.
- Nâng version là một commit/PR riêng, không gộp với feature.

## Sai lầm hay gặp (đã thấy thực tế)

- Template ghi `actions/checkout@v4`, `configure-aws-credentials@v4` trong khi bản mới nhất lúc tra (2026-10) là v7 / v6.
- `rds.Instance.Endpoint` là `address:port`, còn `Address` mới là host thuần. App ghép `host=<Endpoint>` sẽ hỏng DSN.
- Dockerfile `golang:1.22` nhưng `go.mod` có `go 1.23` (hoặc dùng `go mod tidy -diff` cần Go ≥1.23): build fail.
- Secret RDS managed là JSON `{"username","password"}`, không phải chuỗi password thuần.
- Pattern routing `"POST /path"` cần Go ≥1.22, `min`/`max` builtin cần ≥1.21: kiểm tra directive `go` trong `go.mod` trước khi dùng tính năng mới của ngôn ngữ.
