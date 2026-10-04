# Adopt

1. Đọc current contract, root/config/package và quyết định owner mới nhất. Xác định account/region/env, owner backend/network/ECR/data, CI identity và capability cần. Input thiếu ghi pending, chỉ hỏi phần chặn công việc; không điền account/ARN thật.
2. Network mới: bám `network.New` và composition hiện có; đề xuất config, project/resource owner và diff nhỏ nhất. `expectedAccount` qua secret config của operator, không commit plaintext; giữ image đang chạy cho preview hạ tầng.
3. Network có sẵn: báo **chưa hỗ trợ trong PoC hiện tại** theo #15; liệt kê VPC/subnets/SG và ownership cần bàn giao, đề xuất task riêng nếu user muốn. Không tạo lại network hoặc bịa constructor ExistingVpcID. Import không đồng nghĩa consume external network.
4. Đề xuất local checks và authorized preview. Bootstrap admin và workload CI theo quy trình hiện có; không tự deploy/pilot. Production request chỉ nêu quyết định deferred liên quan, không thêm mọi dịch vụ dự phòng.

Output: composition/config đề xuất, nơi sửa, ownership/prerequisites, check plan và unknowns. Go snippet dùng API thực tế hoặc ghi rõ proposal chưa implement. Reviewer khi ownership/identity/delivery có rủi ro; docs đơn giản tự review.
