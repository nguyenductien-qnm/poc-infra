# IaC-managed workload wiring

Inspect the target Pulumi resources and their declared consumer contracts. This reference covers configuration owned by IaC; application implementation, build/tests and container builds are outside this skill.

| Configuration | Trace in the target infrastructure |
| --- | --- |
| Ports, protocols and health-check settings | Compute/service, routing and network rules agree with the declared workload contract; do not assume a fixed port or endpoint |
| Data/service endpoints | Producer outputs and consumer inputs agree on host, port, protocol and ownership |
| Secret references | Provider-specific selectors, secret markings and consuming identity permissions match the documented schema |
| Artifact reference and platform | Supplied immutable artifact, runtime architecture and deployment configuration agree |
| Runtime permissions and operations | Identity, exposure, logging, rollout and recovery settings match the target's requirements |

Verify provider-specific fields with source/SDK documentation. Read a consumer contract when needed, without taking ownership of the application's tests or build. Use schema metadata or nonsensitive fixtures; do not fetch secret values merely to inspect keys.

Evaluate relevant network exposure, transport security, data encryption, backups, retention and deletion protection for the declared target. Identify explicit exceptions and their consequences. Source checks do not prove live connectivity, application behavior or workload health; report that evidence as missing when unavailable.
