# Go components and ownership

Read the composition root and a similar package before editing. `infra/go.mod` is the shared module; reusable code lives in `infra/internal`, and bootstrap-only code lives in `infra/bootstrap/internal`. Preserve this boundary when adding capabilities.

- The root reads and validates config, checks `shared.CheckAccount`, and calls `network.New → platform.New → data.New → app.New`. Resource runtime values flow through Pulumi Inputs/Outputs; do not stringify them prematurely or obtain them from shell commands.
- Keep `New(ctx, name, args, opts...)`, concrete Args/Outputs, and direct dependencies. Helpers usually do not need a ComponentResource. Pass options to the component and parent its children; verify provider/option inheritance when changing them.
- A stack creates resources it owns and consumes existing contracts for resources owned elsewhere. Import when transferring ownership through `shared.ImportOpt`/`importExisting`, without adding another management source. The workload does not yet support arbitrary existing networks, certificates or secrets.
- Preserve type tokens, logical names and parents during refactors. Identity changes require aliases or a state-move/import plan; stateful replacement/deletion in a preview requires owner review before apply.
- Follow existing naming/config conventions, and ensure flags actually affect behavior. Do not add `if stack == "prod"`, a project per component, or library publishing before there is a consumer.

**Existing API**; see `infra/workload/main.go` for the full composition:

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

Add capabilities in the relevant package and composition root. Explain account/network/backend/ownership changes in the diff; a successful build alone is insufficient.
