# Code review

Use the [PR template](../../.github/PULL_REQUEST_TEMPLATE.md). Lead with the concrete problem and resulting behavior. Link the issue and state migration or rollback requirements.

Apply all four lenses to the final diff. A finding needs a file, trigger, impact, and correction. If a lens yields no finding, state the inspected scope instead of claiming complete safety.

## 1. What would you do differently?

Suggest improvements to design, naming, structure, or readability. Check capability ownership, thin routes, truthful production names, and whether a reader can understand the change without conversation history.

## 2. Coupling

Are there unnecessary dependencies between modules/classes? How to decouple? Inspect public surfaces, consumer-owned ports, provider boundaries, identity/scope lifetime, and required atomic operations. A folder move alone does not establish isolation.

## 3. DRY

Is there duplicated logic that should be extracted? Share mechanics when consumers share meaning and lifecycle. Preserve independent server validation, distinct domain models, and separate SSE handling where their guarantees differ.

## 4. Vulnerabilities

Identify security flaws: injection, data exposure, authentication, authorization, unsafe downloads, leaked credentials, dependencies, and untrusted AI input/output. Separate demonstrated flaws from deployment risks and untested hypotheses. Include preconditions and keep private content out of evidence.

## Evidence and handoff

Run relevant [existing checks and affected workflows](../development/VALIDATION.md). Report actual results and unresolved limits. Do not add tests without an explicit user request; a requested evaluation suite follows the same authorization rule.

Keep PRs focused enough to review. Separate behavior/security changes from mechanical moves where practical. The [release workflow](RELEASE-WORKFLOW.md) owns branch targets, authorship, and integration order.
