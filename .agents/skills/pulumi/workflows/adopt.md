# Adopt

1. Note missing inputs (account, region, stack, backend); ask only when they block the work.
2. For each resource decide: create it, consume an external one through an explicit ID/output contract, import ownership, or refactor an existing one. Use APIs that exist; label anything else as a proposal.
3. Compose directly in Go. Trace Inputs/Outputs, providers and parents; plan aliases or migrations for identity changes before any authorized apply.
4. Implement what was asked, then run [verify](verify.md).

Report: what changed and where, ownership and migration decisions, checks run, open inputs.
