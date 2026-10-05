# Kết quả POC: thời gian chạy

Đo ngày 05/10/2026, region `ap-southeast-1`, commit `cc6d42a`. Mỗi stack có 39 resource do Pulumi quản lý (40 tính cả stack).

## Thời gian

| Thao tác | dev | staging | prod |
| -------- | --- | ------- | ---- |
| `pulumi preview` local (admin, gồm compile) | 9,1 giây | n/a | 8,1 giây |
| Bước preview trong CI (login, quyết định tag, preview) | 26 giây (lần 1), 31 giây (lần 2) | 21 giây | n/a |
| `deploy.yml` lần đầu: build image, preview, `up` | 9 phút 9 giây | 9 phút 5 giây | n/a |
| `deploy.yml` lần hai, `app/` không đổi (không build) | 2 phút 30 giây | n/a | n/a |
| `pulumi destroy` (39 resource) | 10 phút 24 giây | 10 phút 25 giây | n/a |

- Prod chỉ `preview` (39 resource sẽ tạo), không `up`.
- `up` lần đầu chiếm gần hết thời gian (tạo RDS và ECS service, khoảng 7 phút). `preview` dưới 10 giây ở 39 resource.
- Bỏ qua build khi `app/` không đổi rút workflow từ 9 phút xuống 2 phút 30 giây, và không restart task.
