# Verify uncertain details

Keep the repository's chosen versions; do not upgrade major versions automatically. Look up only what the code/toolchain has not established:

1. Read infrastructure `go.mod`/`go.sum`, workflow SHAs and Pulumi call sites.
2. Within the module, use `go doc <import-path> <Type/Field>`, `go list -m <module>` and the matching compiler version.
3. If details remain unresolved, read official documentation matching the version: Pulumi Registry, the selected cloud/provider's API and authorization references, Go release notes and workflow dependency definitions at their pinned revisions. Use memory only to suggest search terms.

For example, in the relevant Go module: `go doc github.com/pulumi/pulumi/sdk/v3/go/pulumi Alias`. Resolve provider import paths and versions from that project's module rather than assuming a cloud or SDK major version.

Do not call cloud services by default to look up SDK details. Use authorized previews/read-only metadata only when the task needs them and the context permits access; do not fetch secret values/tokens, run up/import/destroy or mutate config to try an API. If credentials/tools are missing, use documentation and report unknowns. Preview does not guarantee all apply/runtime validation.

Adapters: [Codex skills](https://learn.chatgpt.com/docs/build-skills), [Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents), [Codex hooks](https://learn.chatgpt.com/docs/hooks), [Claude subagents](https://code.claude.com/docs/en/sub-agents), [Claude hooks](https://code.claude.com/docs/en/hooks).
