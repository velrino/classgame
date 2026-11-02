---
name: classgame-review
description: Review ClassGame code, diffs or PRs for concrete correctness, security, reliability and maintenance issues when a review is requested.
---

# ClassGame Review

Review only; change files only when explicitly requested. Read the nearest package guide, relevant callers, contracts and existing safeguards before judging a change.

Use the [four review lenses](../../../docs/processes/CODE-REVIEW.md): design/readability, coupling, duplicated knowledge, and vulnerabilities. Inspect other relevant risks: incorrect business rules, failure handling, concurrency, resource costs, API/schema compatibility, tests and diagnostics. Do not force a finding into each category.

For affected ClassGame paths, verify:

- Identity checks also enforce classroom ownership or enrollment.
- Published revisions stay immutable; attempts remain pinned.
- Student projections exclude teacher-only data.
- Retryable transactions contain no external calls; repeated tasks preserve accepted outcomes.
- Scope changes cancel subscriptions/requests and dispose Three.js resources.

Trace reachable behavior; consider framework guarantees. Separate new/worsened issues from existing ones, and defects from hypotheses or optional cleanup. Recommend the smallest adequate correction. Do not invent findings, demand abstractions or expose private content.

Run relevant [existing checks](../../../docs/development/VALIDATION.md) when appropriate. Do not add tests without an explicit request.

Report briefly: scope/limits; findings ordered Critical → High → Medium → Low, each with location, evidence, trigger/impact, fix and verification; optional improvements; remaining questions and executed checks. If no material findings are supported, say so without claiming complete safety.
