# Quyết định thiết kế

Lý do đằng sau cấu trúc POC. Mỗi mục: chọn gì, vì sao, đánh đổi. Lệnh và thứ tự dựng xem [SETUP.md](SETUP.md), vận hành xem [OPERATIONS.md](OPERATIONS.md).

POC giữ đúng hướng thiết kế mục tiêu (một codebase, nhiều stack, state S3, secret KMS, deploy qua CI có approval) nhưng thu nhỏ tài nguyên cho rẻ và nhanh. Mục tiêu là chứng minh **cách tổ chức và flow vận hành**, không dựng lại toàn bộ product.

## 1. Một codebase, ba stack

**Chọn:** `infra/workload` là một Pulumi project, stack `dev`, `staging`, `prod`. Khác biệt giữa môi trường chỉ nằm ở `Pulumi.<stack>.yaml`.

**Vì sao:**

- Một lệnh `pulumi up` dựng toàn bộ; output truyền giữa package bằng biến Go, không cần `StackReference`.
- Pulumi khuyên bắt đầu monolithic, chỉ tách khi cần.
- Giống mô hình Terraform quen thuộc: `main.go` ≈ root module, `internal/` ≈ modules, `Pulumi.<stack>.yaml` ≈ tfvars.

**Đánh đổi:** cả ba stack chung một account POC, nên CIDR phải khác nhau (`10.10/16`, `10.20/16`, `10.30/16`) và tên resource phải gắn tên stack. `preview` chậm dần khi số resource tăng; đo ở [POC-RESULTS.md](POC-RESULTS.md) để làm căn cứ khi nào nên tách project.

## 2. Package theo lớp, mỗi package là ComponentResource

**Chọn:** `internal/{network,platform,data,app}` cộng `shared`. Mỗi package nhận `Args`, trả về ComponentResource `infra-poc:<package>:<Tên>`. Config chỉ đọc ở `main.go`, truyền xuống qua `Args`.

**Vì sao:** `pulumi preview` và console nhóm resource theo module. `shared` gom hằng số, tag, egress SG và kiểm tra account để không lặp.

## 3. Tách bootstrap khỏi workload

**Chọn:** `infra/bootstrap` (stack `poc`) là project riêng quản lý S3 state workload, KMS key, ECR và toàn bộ danh tính CI (OIDC provider, role preview/deploy, role bootstrap). Package của nó nằm trong `infra/bootstrap/internal`, Go chặn workload import. State bootstrap ở bucket riêng `pulumi-bootstrap-poc-<account>`.

**Vì sao:**

- Khác vòng đời: workload đổi thường xuyên, bootstrap hiếm đổi.
- Khác quyền: pipeline workload không sửa được role CI của chính nó, không xoá được state hay ECR.
- Backend, KMS, ECR, role CI cũng là hạ tầng; để trong code thì có review và lịch sử thay vì script tạo tay.

**Đánh đổi:**

- Vẫn còn đúng một resource tạo tay: bucket state của bootstrap (bootstrap không thể lưu state trong bucket do chính nó tạo).
- Role bootstrap do chính bootstrap quản lý, nên PR sửa sai trust policy có thể tự khoá mình. Xử lý ở [OPERATIONS.md](OPERATIONS.md#bị-khoá-thì-làm-gì).
- Bucket state, KMS key, ECR có `protect` và `retainOnDelete`.

## 4. Preview và deploy là hai role riêng

**Chọn:** mỗi môi trường có `preview` (ReadOnlyAccess + đọc state S3 + `kms:Decrypt`) và `deploy` (AdministratorAccess). Role bootstrap riêng, chỉ tin `environment:bootstrap`.

**Vì sao:** preview vẫn chạy code Go và đọc được secret đã giải mã, nên không cho quyền ghi state hay quyền hạ tầng. Thiết kế: chỉ cấp quyền deploy sau khi người duyệt approve trong GitHub Environment.

**Trạng thái:** cổng approve chưa bật trên GitHub Environment. Quyền vẫn tách (role preview không `up` được, role deploy chỉ assume được từ đúng environment), nhưng không có người duyệt giữa `preview` và `up`. Xem [OPERATIONS.md](OPERATIONS.md#approval-chưa-bật).

## 5. OIDC thay access key, `sub` immutable

**Chọn:** CI xác thực AWS bằng OIDC, không lưu access key. Trust policy dùng `StringEquals` (không wildcard) với `sub` dạng `repo:<owner>@<owner_id>/<repo>@<repo_id>:...`.

**Vì sao:** `sub` theo tên có thể bị chiếm lại khi đổi tên hoặc xoá repo; ID thì không. Repo tạo sau 15/07/2026 dùng format này. Role deploy chỉ tin `environment:<env>`, không tin `sub` của preview.

## 6. Image tag là Git SHA, không nhận `latest`

**Chọn:** CI set `imageTag` = 7 ký tự đầu của commit SHA của lần build gần nhất. Thiếu hoặc `latest` thì preview/up dừng với lỗi. `imageTag` không lưu trong `Pulumi.<stack>.yaml`.

**Chỉ build khi `app/` đổi.** Job `preview` so `app/` giữa commit hiện tại và commit của tag đang chạy; không đổi (và image còn trong ECR) thì giữ tag cũ, không build. Nếu luôn tag theo commit hiện tại, sửa mỗi một security group cũng đổi chuỗi image trong Task Definition và restart toàn bộ task. Tag vì vậy chỉ truy ngược về commit **đã build ra image**, không nhất thiết là commit vừa deploy. Quyết định build nằm ở job `preview` và truyền sang `deploy`, nên người duyệt thấy đúng thứ sẽ được deploy.

**Vì sao:** stack không tự chạy một image khác ý muốn; diff giữa hai lần deploy truy được về commit. ECR dùng chung `poc-app` cho cả ba stack, destroy stack không mất image.

**Đánh đổi:** phát hiện lệch về phía an toàn (thiếu điều kiện nào thì build). Thay đổi ngoài `app/` nhưng ảnh hưởng image (ví dụ base image đổi phía registry với tag cố định `golang:1.22-alpine`) không được phát hiện; muốn build lại thì sửa `app/`.

## 7. `expectedAccount` dạng secret

**Chọn:** mỗi stack (workload và bootstrap) có config `expectedAccount` mã hoá bằng KMS. `main.go` so với `GetCallerIdentity` ngay đầu, lệch thì dừng trước khi tạo resource nào. Thông báo lỗi không in account ID vì log CI của repo là public.

**Vì sao:** ba stack chung một account và nhiều người có nhiều profile AWS; chạy nhầm account là rủi ro thực.

## 8. Cờ bảo vệ resource có dữ liệu

**Chọn:** `protectStateful` bật `pulumi.Protect` cho RDS/EFS và quyết định `skipFinalSnapshot`; `deletionProtection` bật cờ xoá của AWS cho RDS. Hai cờ độc lập, để chứng minh hai tầng bảo vệ (Pulumi và AWS).

**Đánh đổi:** hiện cả ba stack để `false`, kể cả prod. Cần `true` cho prod trước khi chạy kịch bản `protect` ([POC-RESULTS.md](POC-RESULTS.md)).

## 9. Chấp nhận khác production thật

| POC                                                              | Production thật                                | Lý do                                                                        |
| ---------------------------------------------------------------- | ---------------------------------------------- | ---------------------------------------------------------------------------- |
| Fargate task ở public subnet, có public IP                       | Private subnet + NAT/TGW về Hub                | Không cần Hub, không tốn NAT                                                 |
| ALB chỉ HTTP                                                     | HTTPS với ACM                                  | Cần domain                                                                   |
| RDS single-AZ, chưa có AWS Backup                                | Multi-AZ + backup, đã thử restore              | Chi phí; RDS vẫn đủ để chứng minh `protect`, `deletionProtection`, `replace` |
| EFS tạo nhưng app chưa mount                                     | Mount theo nhu cầu app                         | Chỉ chứng minh package `data`                                                |
| Role deploy dùng `AdministratorAccess`                           | Policy tối thiểu + permissions boundary        | Nhanh; việc thu hẹp theo dõi ở TODO #17 trong `ciiam`                        |
| Preview `kms:Decrypt` trên `*`                                   | Giới hạn theo ARN key và môi trường            | Như trên (#17)                                                               |
| Chưa bật _Required reviewers_ cho `staging`, `prod` (`bootstrap` đã bật); `prod` chưa giới hạn deployment branch | Bắt buộc bật cho cả hai | Chưa cấu hình trên GitHub; bật không cần sửa workflow                        |
| PR preview chạy code PR chưa review bằng role read-only          | Approval trước khi job preview nhận credential | Chỉ người trong team mở PR; PR từ fork không được GitHub cấp OIDC token      |
| Action workflow pin theo SHA nhưng ở major cũ                    | Nâng major                                     | Nâng major làm riêng                                                         |
| Không có TGW/NAT                                                 | Có                                             | Tiết kiệm chi phí POC                                                        |

## 10. App là web app Go thật, không phải nginx

**Chọn:** `app/` ghi và đọc note trong Postgres, có `/health`. Mật khẩu DB do RDS tự quản lý trong Secrets Manager (`manageMasterUserPassword`).

**Vì sao:** chứng minh được chuỗi App → RDS qua Secrets Manager và IAM task role, không chỉ một trang tĩnh.

## 11. Đã cân nhắc, chưa làm

| Hướng                                      | Mô tả                                                                                                                                                                                 | Vì sao chưa                                                                                                                                                                                                                                                      |
| ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Write-back `imageTag` vào Git              | Sau khi `up` thành công, bot (`github-actions[bot]`) commit tag mới vào `Pulumi.<stack>.yaml` để Git là nguồn sự thật về bản đang chạy; local preview không cần `config set` thủ công | Bot phải push được vào nhánh có branch protection (hoặc mở PR rồi người merge), thêm commit nhiễu và race với dev đang push. Pull code về cũng không tự cập nhật ECS (không có controller như Argo/Flux), vẫn cần CI `pulumi up`. POC giữ tag trong state Pulumi |
| Deployment circuit breaker cho ECS Service | `DeploymentCircuitBreaker` với `rollback: true` để ECS tự rollback khi task mới không khỏe                                                                                            | Chưa cần cho POC; ghi nhận ở [OPERATIONS.md](OPERATIONS.md#image-đến-ecs-thế-nào)                                                                                                                                                                                |
| Tag image theo hash nội dung `app/`        | Nội dung không đổi thì tag không đổi                                                                                                                                                  | Hiện tại đã chọn cách bỏ qua build khi `app/` không đổi (mục 6)                                                                                                                                                                                                  |
