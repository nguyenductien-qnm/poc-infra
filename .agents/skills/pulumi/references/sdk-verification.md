# Verify chi tiết chưa rõ

Giữ version repo chọn, không tự nâng major. Chỉ tra điều chưa được code/toolchain chứng minh:

1. Đọc `go.mod`/`go.sum`, workflow SHA, Dockerfile và call site.
2. Trong module, dùng `go doc <import-path> <Type/Field>`, `go list -m <module>` và compiler đúng version.
3. Nếu còn thiếu, đọc docs chính thức khớp version: Pulumi Registry, AWS API/Service Authorization Reference, Go release notes, `action.yml` tại commit đang dùng. Trí nhớ chỉ gợi ý từ khóa.

Ví dụ trong `infra/`: `go doc github.com/pulumi/pulumi-aws/sdk/v7/go/aws/rds InstanceArgs`; `go doc github.com/pulumi/pulumi/sdk/v3/go/pulumi Alias`.

Không gọi cloud mặc định để tra SDK. Chỉ authorized preview/read-only metadata khi task cần và context có quyền; không secret value/token, up/import/destroy hoặc config mutation để thử API. Thiếu credential/tool thì dùng docs và báo unknown. Preview không đảm bảo mọi validation tại apply/runtime.

Adapters: [Codex skills](https://learn.chatgpt.com/docs/build-skills), [Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents), [Codex hooks](https://learn.chatgpt.com/docs/hooks), [Claude subagents](https://code.claude.com/docs/en/sub-agents), [Claude hooks](https://code.claude.com/docs/en/hooks).
