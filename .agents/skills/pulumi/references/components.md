# Components, ownership and workload wiring

- Read and validate config (account, region, environment) at composition roots; pass values through concrete Args. Keep Pulumi Input/Output and secret semantics; do not extract values through shell commands.
- Use a ComponentResource for a cohesive capability with useful outputs; plain helpers need no wrapper. Follow existing constructor conventions and verify parent/provider propagation to children.
- Creating, consuming and importing differ. External IDs come from an explicit contract; import transfers management. Never leave a resource with two IaC owners.
- In refactors keep type tokens, logical names, parents and physical names, or add aliases. List expected replacements and deletions, especially for stateful resources.
- Split projects or stacks only for real ownership, lifecycle, permission or blast-radius boundaries.
- Config flags and outputs need an actual consumer.

## Go Inputs, Outputs and components

- Pass Outputs directly to compatible Input fields. Pulumi carries dependencies, unknown values and secrets through them; add `DependsOn` only for a real dependency not expressed by inputs.
- Use `ApplyT` for value transformations, not ordinary resource creation: callbacks on unknown outputs may not run during preview, hiding resources from the plan. Keep resource registration outside callbacks with Output-valued inputs. Do not assign callback results to outer plain Go variables and consume them immediately; return an Output instead.
- `ApplyT` is valid when a transformation is needed. Preserve secret Outputs through the transformation; do not log, unwrap or declassify them to obtain strings. Return transformation errors through the callback's error result.
- A component constructor accepts caller `opts ...pulumi.ResourceOption`, passes them to the project's supported component registration API, parents children to the component and exposes the consumed outputs. Register those outputs with `ctx.RegisterResourceOutputs` before returning and propagate errors from each registration. This records component outputs and signals completion; a Go struct field alone does not do that. Missing registration does not by itself break in-program consumers of assigned Go Output fields. Base finding severity on the affected contract.
- Trace provider inheritance through the parent and provider map; do not require redundant explicit providers on every child. Use explicit provider options where the child's intended provider differs, checking the pinned SDK API.
- Distinguish a Go identifier rename from a Pulumi logical-name/type/parent change. The former alone does not change resource identity; the latter needs alias/migration analysis. Physical-name or provider changes can also cause replacement. Source review predicts risk; only an authorized preview supplies an observed plan.

## Workload wiring

Check that IaC settings match the workload's declared contract:

- Ports, protocol and health checks agree across service, load balancer and network rules.
- Producer outputs and consumer inputs agree on host, port and protocol.
- Secret references and the consuming identity's permissions match; inspect schemas, not secret values.
- Image reference and platform match what upstream supplied.

A successful build proves local correctness only, not cloud acceptance or live connectivity.
