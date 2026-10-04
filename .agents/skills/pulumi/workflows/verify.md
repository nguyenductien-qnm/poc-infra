# Verify

Đọc diff, current contract và implementation liên quan. Primary chạy checks; reviewer nhận scope, diff/base, file liên quan, kết quả checks và phần thiếu evidence.

## Checks theo thay đổi

- Pack/adapters/hooks: ở repo root, `python -B scripts/verify-pulumi-pack.py`, rồi `python -B -m unittest discover -s scripts -p test_pulumi_pack.py`. Không AWS, không Go build.
- Go infra: trong `infra/`, `go mod tidy -diff`, `go vet ./...`, `go test ./...`, `go build ./...`.
- App hoặc app↔infra: trong `app/`, `go mod tidy -diff`, `go vet ./...`, `go test ./...`, `go build -o <temporary-path-outside-repo> .`; kiểm tra env/port/secret/image bên task definition. Docker build khi đổi container và runtime có sẵn.
- Authorized workload preview: operator chọn backend/stack/imageTag trước; primary chạy `pulumi preview --diff` khi task cho phép. Không mặc định đọc live state, đổi config hoặc dispatch workflow để verify. Không up/destroy/import/refresh hoặc bypass gate. Thiếu live checks ghi unknown, không thay bằng mocks rồi báo pass.

## Review

Args/Outputs/identity/ownership → `pulumi-component-reviewer`; backend/IAM/workflow → `pulumi-delivery-reviewer`. Gọi bằng tên agent trong runtime và cung cấp local results; mặc định một người. Hai risk độc lập mới cần cả hai. Không có subagents thì tự review và ghi rõ; primary kiểm tra findings trước sửa.

Output: `pass | changes-requested | blocked`, scope/commit, checks thật (lệnh + kết quả), findings với file:line/scenario và unknowns. Pass chỉ cho scope đã kiểm chứng, không là production approval. Hook nhắc stale không đồng nghĩa check đã chạy. Không tạo reports/evidence lớn trong repo.
