# Evaluations

Python runs the datasets, checks results, and saves reports. Small Go and Node bridges call the same policy functions used by the API and web app. No business rule is copied into Python.

```bash
# From the repository root; Python 3.9+ and the API Go toolchain are required.
python3 evals/run.py
python3 evals/run.py --list
python3 evals/run.py --cases=tutor-private-context-minimized-v1,auth-wrong-provider-v1
python3 evals/run.py --dataset evals/platform/offline-evidence/cases.json
python3 evals/run.py --dataset evals/platform/frontend-recovery/cases.json
```

Prepare Go dependencies normally first (`go -C api mod download`). Evaluation builds use the local module/toolchain cache with dependency downloads disabled. One bridge process executes the entire selected batch; no process is launched per case. It loads no application configuration or credentials and constructs no real provider/database client.

| Path | Responsibility |
| --- | --- |
| [run.py](run.py) | Strict preflight, selection, assertions, manifests/results, exit gate |
| [platform/ai-policy/cases.json](platform/ai-policy/cases.json) | Versioned synthetic inputs and expected policy behavior |
| [platform/offline-evidence/cases.json](platform/offline-evidence/cases.json) | Scope, queue capacity, replay, retry, and completion policies; requires Node.js 24+ |
| [platform/frontend-recovery](platform/frontend-recovery/README.md) | Production reducer retries, completion, local repair, and server resume mapping; requires Node.js 24+ |
| [api/cmd/evalbridge](../api/cmd/evalbridge/main.go) | Production Go calls with scripted provider/authentication ports |
| [api/cmd/evalfirestore](../api/cmd/evalfirestore/README.md) | Optional real transaction checks on a guarded local emulator |
| [platform/browser-store](platform/browser-store/README.md) | Real two-tab IndexedDB and reload checks on a separate local origin |
| [platform/local-http](platform/local-http/README.md) | Opt-in real API/Auth/rules flows and browser outage controls on demo emulators |
| [manual rubric](platform/ai-policy/MANUAL-RUBRIC.md) | Separate pedagogical review and held-out comparison plan |
| `output/<unique-run>/` | Ignored, newly created run artifacts |

`manifest.json` records the production revision, dirty state, dataset hash/version, mode, case IDs, and **zero real provider calls**. `results.json` records assertions and synthetic observations; raw provider prompts are replaced by their hash/length. The runner never reuses a run directory.

Exit `0` means every selected policy case passed, `1` means assertions failed, and `2` means invalid input/build/bridge/output failure. Unknown flags/IDs, duplicate IDs, malformed cases, and unsupported modes fail before case execution. `--mode=live` is deliberately unavailable; no paid run occurs.

The default, `offline-evidence`, and `frontend-recovery` suites are offline policy checks, including a scripted timeout result. They do not run IndexedDB or real browser tabs. Passing does not prove teaching quality, live recovery, or deployed access. Use the opt-in browser and local HTTP tools for their stated runtime coverage. See [evaluation scope](../docs/ai/EVALUATIONS.md) and [release safety](../docs/operations/RELEASE-SAFETY.md).

## Optional queue transport check

`python3 evals/platform/queue-smoke/run.py` starts a temporary localhost Redis on a free non-default port and runs synthetic Go handlers through the real queue adapter. It checks completion, replay/conflict, delay, retry, archive and correlation. Two real Go worker processes share a job while two producers concurrently enqueue it. Another worker is killed before its synthetic effect; a replacement must recover it after the real lease expires. Finally, an already-connected producer must fail within a budget after the temporary broker stops. The crash check usually takes 1–2 minutes. Go and `redis-server` are required. Each run has a new ignored manifest/report directory under `evals/output/`. It makes zero database or real provider calls and does not run production domain workers. It is separate from the default offline policy suite, and does not establish production persistence, failover or teaching quality.

These checks cover local transport behavior. They do not prove exactly-once effects or billing, recovery after an effect commits but its acknowledgement is lost, network partitions, Redis persistence/restore, managed failover, cluster compatibility or application recovery from a lost broker. Keep those environment-specific gates in [queue operations](../docs/operations/QUEUES.md). `database_calls=0` refers to domain databases; temporary Redis counters provide only synthetic observations.
