# Go components và ownership

Đọc root và package cùng loại trước khi sửa. `infra/go.mod` là module chung; reusable code ở `infra/internal`, code chỉ bootstrap ở `infra/bootstrap/internal`. Giữ ranh giới này khi thêm capability.

- Root đọc/validate config, kiểm tra `shared.CheckAccount`, gọi `network.New → platform.New → data.New → app.New`. Runtime values từ resource đi qua Pulumi Input/Output, không stringify sớm hoặc lấy từ shell.
- Giữ `New(ctx, name, args, opts...)`, Args/Outputs cụ thể và dependencies trực tiếp. Helpers thường không cần ComponentResource. Truyền options vào component, parent vào children; kiểm tra provider/options kế thừa khi thay đổi chúng.
- Stack tạo resource nó sở hữu; resource owner khác consume contract đã có. Import khi bàn giao ownership theo `shared.ImportOpt`/`importExisting`, không thêm nguồn quản lý. Workload chưa hỗ trợ existing network, certificate hoặc secret tùy ý.
- Giữ type token, logical name và parent khi refactor. Thay identity cần aliases hoặc kế hoạch state move/import; preview replace/delete stateful cần owner review trước apply.
- Naming/config theo conventions hiện có, flags phải có tác dụng. Không thêm `if stack == "prod"`, project cho mỗi component hoặc library publishing trước khi có consumer.

API **đang có**, xem `infra/workload/main.go` cho composition đầy đủ:

```go
netOut, err := network.New(ctx, "network", &network.Args{
    VpcCidr: vpcCidr,
    Region: region.Name,
})
if err != nil { return err }
platOut, err := platform.New(ctx, "platform", &platform.Args{
    VpcID: netOut.VpcID,
    PublicSubnetIDs: netOut.PublicSubnetIDs,
})
if err != nil { return err }
```

Thêm capability ở package tương ứng và root. Account/network/backend/ownership changes phải được giải thích trong diff, không chỉ để build pass.
