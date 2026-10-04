# Contract hiện tại

Baseline: `main` sau PR #35. Khi nhận task mới, đọc diff và comment owner mới nhất ở issue liên quan; bảng này không thay quyền quyết định của owner.

| Quyết định `nguyenductien-qnm` | Contract áp dụng | Nguồn |
| --- | --- | --- |
| Hai project chung module; existing network/ECR/cert/secret API quá mức cho PoC | `infra/workload` ghép `infra/internal/{network,platform,data,app}`; bootstrap có `bootstrap/internal` riêng | [#15](https://github.com/nguyenductien-qnm/poc-infra/issues/15#issuecomment-5970465379) |
| State workload/KMS bằng Pulumi; bootstrap state bucket bằng script; chưa tách key mỗi môi trường | Giữ `scripts/setup-bootstrap-backend.sh` làm ngoại lệ. Không chuyển OIDC/role CI về shell | [#16](https://github.com/nguyenductien-qnm/poc-infra/issues/16#issuecomment-5970478096) |
| OIDC, sáu role và preview policy bằng Pulumi | Sửa `infra/bootstrap/internal/ciiam`; admin vận hành bootstrap theo guide hiện có | [#17](https://github.com/nguyenductien-qnm/poc-infra/issues/17#issuecomment-5970483982) |
| PoC chỉ yêu cầu imageTag và ECR name từ config; runtime hardening để production | Giữ ECS public subnet/IP, ALB HTTP, DB/Bedrock/rollback theo code hiện tại | [#18](https://github.com/nguyenductien-qnm/poc-infra/issues/18#issuecomment-5970563054) |
| Giữ PR preview có read-only OIDC, Actions pin SHA ở major đang dùng | Giữ `dev → dev`, `staging → staging`, `main → prod`, image SHA 7 ký tự; digest/build-once, least privilege, pin CLI và protections là việc production riêng | [#19](https://github.com/nguyenductien-qnm/poc-infra/issues/19#issuecomment-5970634952) |
| AI tool làm trước, docs do owner làm sau | Pack #22; proposal #14 và adoption/runbook/pilot #20 còn mở. #21 đã đóng, không thêm comprehensive mocks/evidence | [#14](https://github.com/nguyenductien-qnm/poc-infra/issues/14#issuecomment-5970694983), [#20](https://github.com/nguyenductien-qnm/poc-infra/issues/20#issuecomment-5970703112), [#22](https://github.com/nguyenductien-qnm/poc-infra/issues/22#issuecomment-5970653092) |

## Giới hạn phải báo đúng

- Project separation là ranh giới code/state, **chưa là IAM boundary**: deploy role còn `AdministratorAccess`. Preview có `ReadOnlyAccess` và state/KMS permissions, nên PR code vẫn có rủi ro đọc dữ liệu. Không gọi credential-free hoặc an toàn chỉ vì tên read-only; không đọc secrets để thử.
- `infra/workload/Pulumi.prod.yaml` hiện đặt `protectStateful`/`deletionProtection` false, dù bảng README ghi true. Đánh giá theo config/code thực tế và báo discrepancy; không tự bật protection trong task pack.
- Hai workflow hạ tầng tạm tắt trigger tự động. `pr-check.yml` chỉ có dispatch nhưng dùng `github.base_ref`/payload PR, nên dispatch chưa thay thế được PR preview. Branch mapping chỉ có `dev`, `staging`, `main`; không dispatch deploy từ feature branch để thử pack.
- Network root tạo VPC/subnets mới. Existing-network composition chưa có API; nhận IDs là input đề xuất, không trình bày snippet giả như implementation đang chạy. ECR config đổi URL workload, còn deploy workflow push vào `poc-app`; đổi tên phải kiểm tra cả hai bên trong task được duyệt.
- CLI chưa pin; preview/build không chứng minh AWS chấp nhận cấu hình, IAM denies thật, DB kết nối, rollback hoặc production readiness.

Proposal production ghi riêng phần cần owner quyết định: private workload/egress, TLS, DB encryption/backup/Multi-AZ/protection, runtime IAM, state isolation và delivery cùng digest. Trình bày tradeoff, không triển khai trước.
