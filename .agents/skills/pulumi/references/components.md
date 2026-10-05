# Components, ownership and workload wiring

- Read and validate config (account, region, environment) at composition roots; pass values through concrete Args. Keep Pulumi Input/Output and secret semantics; do not extract values through shell commands.
- Use a ComponentResource for a cohesive capability with useful outputs; plain helpers need no wrapper. Follow existing constructor conventions and verify parent/provider propagation to children.
- Creating, consuming and importing differ. External IDs come from an explicit contract; import transfers management. Never leave a resource with two IaC owners.
- In refactors keep type tokens, logical names, parents and physical names, or add aliases. List expected replacements and deletions, especially for stateful resources.
- Split projects or stacks only for real ownership, lifecycle, permission or blast-radius boundaries.
- Config flags and outputs need an actual consumer.

## Workload wiring

Check that IaC settings match the workload's declared contract:

- Ports, protocol and health checks agree across service, load balancer and network rules.
- Producer outputs and consumer inputs agree on host, port and protocol.
- Secret references and the consuming identity's permissions match; inspect schemas, not secret values.
- Image reference and platform match what upstream supplied.

A successful build proves local correctness only, not cloud acceptance or live connectivity.
