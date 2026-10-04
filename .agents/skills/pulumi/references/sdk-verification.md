# Verify uncertain details

Keep the repository's chosen versions; do not upgrade major versions automatically. Look up only what the code/toolchain has not established:

1. Read `go.mod`/`go.sum`, workflow SHAs, the Dockerfile and call sites.
2. Within the module, use `go doc <import-path> <Type/Field>`, `go list -m <module>` and the matching compiler version.
3. If details remain unresolved, read official documentation matching the version: Pulumi Registry, AWS API/Service Authorization Reference, Go release notes and `action.yml` at the pinned commit. Use memory only to suggest search terms.

Examples in `infra/`: `go doc github.com/pulumi/pulumi-aws/sdk/v7/go/aws/rds InstanceArgs`; `go doc github.com/pulumi/pulumi/sdk/v3/go/pulumi Alias`.

Do not call cloud services by default to look up SDK details. Use authorized previews/read-only metadata only when the task needs them and the context permits access; do not fetch secret values/tokens, run up/import/destroy or mutate config to try an API. If credentials/tools are missing, use documentation and report unknowns. Preview does not guarantee all apply/runtime validation.

Adapters: [Codex skills](https://learn.chatgpt.com/docs/build-skills), [Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents), [Codex hooks](https://learn.chatgpt.com/docs/hooks), [Claude subagents](https://code.claude.com/docs/en/sub-agents), [Claude hooks](https://code.claude.com/docs/en/hooks).
