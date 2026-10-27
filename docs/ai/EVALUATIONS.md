# Evaluation scope

Run `python3 evals/run.py` from the repository root. Python 3.9+ owns datasets, preflight, assertions, and run artifacts; the small Go bridge lives inside the existing API module so it can use production `internal` packages. Node remains responsible for frontend checks.

The [synthetic suite](../../evals/platform/ai-policy/cases.json) exercises production boundaries through scripted ports:

| Boundary | Evidence checked |
| --- | --- |
| Tutor generation | Insufficient/filtered evidence bypasses generation; disabled and scripted-timeout providers select honest fallbacks |
| Citation normalization | Invented chunk/event IDs are excluded; supplied IDs remain |
| Provider context | Private canaries and adaptation narrative are absent; approved skill IDs/explanation preference remain; caller context is preserved |
| Student projection | Teacher observations and provider metadata are excluded from student responses |
| Identity policy | Disallowed providers and disabled identities fail; a permitted student identity passes |
| Learning assessment | Client counts/completion flags do not replace persisted answers; accepted feedback survives reconstruction without crediting an answer |
| Scope and memory | Stale tutor revisions/owners and cross-class/tenant or older memory updates are rejected |
| Object paths | Valid filenames containing double dots remain usable; traversal, empty, and backslash segments fail |

These calls use the production generation service, provider policy, student response mapper, and pure learning assessment function. The bridge does not connect to Firebase, Redis, or a provider, and receives no application credentials. Python does not copy the business decisions it checks.

The runner validates flags, duplicate JSON keys/IDs, case kinds, selectors, and expectation operators before execution. The bridge validates its complete batch before any scripted call. Failed checks exit `1`; invalid execution exits `2`. Exclusive run directories contain immutable-by-runner manifests/results, revision and dirty status, dataset hash/version, selected IDs, and zero real provider calls.

This is **offline policy/contract coverage**. It does not evaluate generated teaching quality, semantic claim support, real provider cancellation/latency, concurrency saturation, Firebase token verification, browser rules, or deployed object access. Scripted timeout and canary cases cover specific boundaries; they do not prove general prompt-injection resistance or impossible leakage.

Use the separate [pedagogical rubric and held-out plan](../../evals/platform/ai-policy/MANUAL-RUBRIC.md) for model quality. There is no live/paid execution path in this runner. Future Python graders can consume the JSON artifacts without moving production policy into the harness.

For commands and output details see [evals/README.md](../../evals/README.md). Keep the broader [release gates](../operations/release-safety.md) and existing API/web checks as separate requirements.
