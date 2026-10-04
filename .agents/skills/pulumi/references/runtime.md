# App ↔ ECS/RDS contract

When changing environment variables, images, health checks or data wiring, read `app/main.go`, the Dockerfile and `infra/internal/{app,platform,data}`. The [current contract](current-contract.md) records the PoC exceptions explicitly.

| Value | Where it must match |
| --- | --- |
| Port 80 (`shared.AppPort`), `/health` | HTTP app, ECS container port, target group/health path |
| DB host/port/name/user | `data/rds.go`, `app/service.go`, app DSN; `Address` is the host, while `Endpoint` includes the port |
| DB password | ECS `valueFrom` uses the ARN + JSON selector `:password::`; the execution role reads the correct secret |
| ECR URL/imageTag | Bootstrap repository, workload config, build/push workflow; a tag is required and cannot be `latest` |
| Logging/platform | stdout/awslogs, Docker `linux/amd64` and the Fargate task |

Do not fetch passwords to check wiring. Tracing/build only verifies the local contract, not DB connectivity or ECS health. Pilot #20 requires a DB connection or a round-trip with cleanup; `/health` returning 200 is insufficient.

Private tasks/TLS, DB encryption/backup/Multi-AZ/protection, removing the Bedrock demo and deployment rollback are deferred. For production work, propose simple config with an actual consumer and effect; distinguish proposed changes from current behavior.
