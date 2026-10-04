# Adopt

1. Read the current contract, root/config/package and latest owner decisions. Identify account/region/environment, backend/network/ECR/data owners, CI identity and required capabilities. Mark missing inputs pending and ask only about blockers; do not fill in real account IDs/ARNs.
2. New network: follow `network.New` and the existing composition; propose config, project/resource owners and the smallest diff. The operator supplies `expectedAccount` through secret config; do not commit it in plaintext. Keep the running image for infrastructure previews.
3. Existing network: report **unsupported by the current PoC** under #15; list the required VPC/subnet/security-group inputs and ownership handoff, and propose separate work if requested. Do not recreate the network or invent an ExistingVpcID constructor. Importing is not equivalent to consuming an external network.
4. Propose local checks and an authorized preview. Follow the existing bootstrap-admin and workload-CI processes; do not deploy/run a pilot automatically. For production requests, identify only relevant deferred decisions rather than adding every possible service.

Output: proposed composition/config, edit locations, ownership/prerequisites, check plan and unknowns. Go snippets must use actual APIs or be clearly marked as unimplemented proposals. Use a reviewer for ownership/identity/delivery risks; simple docs can use self-review.
