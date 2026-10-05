# Go components and ownership

Locate the target Pulumi project, Go module and composition roots. Read the affected packages and consumers before editing; package and project layouts belong to the target project.

- Read and validate configuration at roots, including target account/subscription, region and environment where applicable. Pass explicit dependencies through concrete Args/Outputs; preserve Pulumi Input/Output and secret semantics instead of extracting runtime values through shell commands.
- Compose ordinary Go packages directly. Use ComponentResource for a cohesive capability with useful ownership and outputs; small helpers need no wrapper. Follow existing constructor conventions. Verify parent/provider/resource-option propagation through the actual SDK and affected children.
- Separate creating a resource, consuming one managed elsewhere, and importing ownership. External IDs/ARNs must come from an explicit contract. Importing transfers management; it is not a substitute for referencing externally managed infrastructure. Avoid two IaC owners.
- Preserve type tokens, logical names, parents and physical-name behavior during refactors. Explain aliases or other migration steps and review expected replacements/deletions, particularly stateful resources, before authorized apply.
- Split projects/stacks when ownership, lifecycle, permission or blast-radius boundaries warrant it. Do not mandate a fixed number of projects, a stack per component or an environment-specific branch inside reusable code.
- Configuration flags and outputs must have actual consumers. Add capabilities in existing packages when possible; publish a library only when reuse warrants its compatibility cost.

Use APIs verified in the target source or installed SDK. Clearly label unimplemented APIs as proposals. A successful build is evidence of local correctness, not proof of safe resource identity, cloud acceptance or live behavior.
