---
name: classgame-evals
description: Select, run, interpret or maintain ClassGame evaluations when checking AI policies, browser recovery contracts or local runtime behavior.
---

# ClassGame Evals

Read [evaluation commands](../../../evals/README.md) and [coverage limits](../../../docs/ai/EVALUATIONS.md). Choose the existing suite that matches the changed boundary.

Routine checks use `python3 evals/run.py` from the repository root. Python 3.9+ handles datasets, assertions and reports. Go and Node bridges call real production policies; never copy those policies into Python or bridge code. Prepare required toolchain/dependency caches as documented.

Keep the default run synthetic, offline and credential-free. There is no live/paid provider mode. Local browser, HTTP, Redis and Firestore checks are separate opt-in tools: use them only within the task's authorization and preserve unrelated services. Emulator checks require numeric loopback hosts, `demo-` projects and owned fixtures.

When editing requested evaluation cases, preserve historical IDs and dataset versions; retire mistakes and append corrections. Validate flags, selectors, duplicate IDs and expectations before execution. Do not add new tests without an explicit request.

Each run needs a fresh exclusive output directory, revision/dirty state, dataset hash/version, selected cases, mode and real/scripted call counts. Never overwrite old runs or change expectations just to pass.

Report the exact command, selected scope, exit status and artifact path. Distinguish assertion failures from invalid execution. Scripted checks establish policy/contracts, not teaching quality, real provider timeouts, browser storage behavior or deployed security; use the separate tools/rubric for their stated coverage.
