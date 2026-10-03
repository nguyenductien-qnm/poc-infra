# Production hardening

Những chỗ POC tham khảo làm tắt để rẻ/nhanh. Hệ thống thật làm theo cột "Production". Khi một mục chưa làm được, ghi rõ trong README mục "Khác biệt so với production".

| Hạng mục | POC làm tắt | Production |
| --- | --- | --- |
| Vị trí workload | Fargate ở public subnet, public IP | Private subnet, ra Internet qua NAT Gateway (1/AZ ở prod) hoặc TGW về hub; `AssignPublicIp: false` |
| Truy cập AWS API từ VPC | Qua Internet | VPC endpoint: Gateway (S3, DynamoDB), Interface (ECR api/dkr, Secrets Manager, CloudWatch Logs, STS) |
| ALB | HTTP :80 | Listener HTTPS :443 với ACM cert, `SslPolicy` hiện đại; :80 chỉ redirect 301 sang 443; cân nhắc WAF |
| RDS | Single-AZ, không backup | `MultiAz`, `BackupRetentionPeriod` ≥ 7, `StorageEncrypted` + KMS, Performance Insights, parameter group riêng (`rds.force_ssl=1`), AWS Backup |
| Bảo vệ dữ liệu | `protect` tắt ở mọi env | `protectStateful` + `deletionProtection` bật ở prod (và staging nếu có dữ liệu thật), `FinalSnapshotIdentifier` |
| IAM role deploy CI | `AdministratorAccess` | Policy thu hẹp theo service thực dùng, permission boundary; role preview read-only |
| Task role | `"*"` resource cho demo | ARN cụ thể |
| ECS service | Không autoscaling, không circuit breaker | `DeploymentCircuitBreaker{Enable, Rollback}`, Application Auto Scaling theo CPU/request, `HealthCheckGracePeriodSeconds`, ≥2 task trải đa AZ |
| Container | Chạy root | User non-root, `ReadonlyRootFilesystem` khi được, image scan on push |
| Log | Log group mặc định | `RetentionInDays` đặt rõ, KMS khi log nhạy cảm |
| Monitoring | Không | CloudWatch alarm (5xx ALB, CPU/memory ECS, RDS CPU/storage/connections) gửi SNS |
| Chi phí | Budget alert thủ công | Tag `CostCenter`/`Owner` bắt buộc, AWS Budgets theo tag |
| State | S3 + KMS | Như POC, thêm bucket policy chặn xoá, block public access, replication nếu cần DR |

## Gợi ý triển khai bằng config

Biến các khác biệt thành config key thay vì if/else theo stack:

```yaml
# Pulumi.prod.yaml
config:
  <project>:multiAz: "true"
  <project>:backupRetentionDays: "14"
  <project>:natGatewayPerAz: "true"
  <project>:protectStateful: "true"
  <project>:deletionProtection: "true"
  <project>:certificateArn: arn:aws:acm:...
```

Dev có thể tắt bớt (`natGatewayPerAz: "false"` dùng 1 NAT) để tiết kiệm, nhưng code đường đi giống prod.
