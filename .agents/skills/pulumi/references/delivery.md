# Bootstrap, state và delivery

Đọc `infra/bootstrap/main.go`, `bootstrap/internal/{statebackend,registry,ciiam}` và workflow thật. [Current contract](current-contract.md) phân biệt PoC với production backlog.

- Bootstrap quản lý workload state S3, secrets KMS, ECR và CI OIDC/IAM; admin theo [BOOTSTRAP.md](../../../../docs/BOOTSTRAP.md). Bootstrap backend S3 do script tạo là ngoại lệ đã chốt. Một KMS key dùng chung; object SSE-S3 khác secrets encryption KMS.
- Import mode dành cho resources đang có; sau tiếp nhận review việc tắt `importExisting`. Không chạy import/up để thử pack; không xuất checkpoint, secret hoặc decrypt output vào log/artifact.
- State bucket/key/ECR protect/retain theo code. Tách project không hạn chế Admin workload CI. Trace OIDC `aud`/`sub`, repo identity và branch/environment từ config đến roles; không mở trust wildcard hoặc chuyển roles về script.
- Preview nhận OIDC cho PR và branch, có state permissions cần cho CLI và KMS decrypt. Đây vẫn là PoC risk, không suy luận fork rules là IAM enforcement. Secret schema dùng metadata/code hoặc fixture không nhạy cảm; không GetSecretValue chỉ để xem keys.
- Deploy hiện preview tag mới → environment approval (khi đã cấu hình) → build/push tag SHA ngắn → up. Chưa build image tại preview, chưa bind digest/artifact hoặc state freshness. Không báo cùng SHA là artifact đã duyệt hoặc protections đã bật.
- Actions remote pin full commit SHA, giữ major hiện có. Không đổi workflow triggers, permissions hoặc GitHub settings để thử pack.

Production delivery là task riêng: build một lần trước preview, digest/config/SHA cùng apply, deny stale artifacts/state, least privilege/boundary và protections thật. Không biến chúng thành gate bắt buộc của PoC pack.
