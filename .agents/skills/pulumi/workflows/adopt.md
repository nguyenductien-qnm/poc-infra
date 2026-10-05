# Adopt

1. Inspect the target project's Pulumi configuration, Go roots, relevant packages and applicable decisions. Establish account/subscription, region, environment, backend, resource owners and desired capabilities. Mark missing inputs pending; ask only when they block dependent work.
2. Classify each resource relationship: create under this stack's ownership, consume an externally managed resource, import ownership, or refactor an already managed resource. Locate actual APIs and contracts. If support is missing, propose the smallest concrete capability instead of inventing working constructors or rebuilding externally owned infrastructure.
3. Choose direct Go composition and explicit configuration. Trace Inputs/Outputs, providers, parent relationships and resource identity. Plan migrations and expected replacements/deletions before authorized state operations.
4. Identify infrastructure delivery and drift/recovery owners. Trace the desired Git revision/configuration to the target stack, preview, authorization and apply; preserve supplied immutable artifact inputs. Address relevant production risks and record target-specific exceptions.
5. Deliver the requested proposal or implement the authorized edits, then follow [verify](verify.md). Keep pending external inputs and live checks explicit. Application builds/tests, deployment and state mutation are not implicitly authorized by adoption.

Output: concrete edit locations/composition, ownership, required inputs, migration/delivery decisions, actual changes when requested, verification and unknowns. Use verified APIs or label proposals. Select a reviewer for material ownership/identity/delivery risks; small docs can use disclosed self-review.
