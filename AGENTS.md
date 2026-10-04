# infra-poc

Pulumi Go PoC: `infra/bootstrap` và `infra/workload` chung module; Go service ở `app/`. Bootstrap admin chạy tay, workload deploy qua CI. Quyết định owner ở issues mới hơn checklist Epic; pack phân biệt PoC và production deferred.

## Hướng dẫn theo công việc

Dùng [Pulumi pack](.agents/skills/pulumi/SKILL.md) khi adopt/review infra, bootstrap IAM/state, CI hoặc app↔infra wiring. Đọc references theo scope. App-only không cần reviewer infra. Codex gọi `$pulumi adopt …` / `$pulumi verify …`; Claude gọi `/pulumi adopt …` / `/pulumi verify …`.

Code đang có là chuẩn về style. Config tại root, concrete Args/Outputs, composition trực tiếp, một owner mỗi resource; không framework/registry. Giữ resource identity hoặc có migration/alias được review. Không secret/token/account/ARN nội bộ plaintext trong code/docs/log mới; không fetch secret value để kiểm tra schema.

Checks theo [verify workflow](.agents/skills/pulumi/workflows/verify.md); báo kết quả thật và checks thiếu. Preview trong authorized context. Không tự up/destroy/import, đổi GitHub settings hoặc bật workflow. Bootstrap operator có quyền riêng theo `docs/BOOTSTRAP.md`, không chạy thay admin khi task chưa cho phép.

Commit Conventional Commits tiếng Anh, chữ thường, theo scope; comments code tiếng Việt không dấu, docs tiếng Việt có dấu. Behavior changes cập nhật hướng dẫn liên quan; tránh thêm docs/report khi đã có nơi thích hợp.
