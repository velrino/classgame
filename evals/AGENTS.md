# Evaluation workspace

- Python owns dataset validation, selection, expectations, reports, and immutable run directories. Use the standard library and Python 3.9+.
- The small Node bridge calls pure TypeScript policy without browser storage. The small Go bridge under `api/cmd/evalbridge` invokes real production functions inside the existing Go module. Do not duplicate authorization, tutoring, projection, or learning rules in Python or the bridge.
- Default execution is offline, synthetic, and credential-free. This policy suite has no paid/live provider mode. A future live suite needs explicit opt-in, a reviewed budget, and its own honest coverage label.
- Validate flags, dataset fields, case kinds, selected IDs, duplicate IDs, and expectations before execution. Invalid input exits nonzero without creating a run; failed assertions also exit nonzero.
- Treat case IDs and dataset versions as historical contracts. Retire a mistaken case and append a corrected version; do not silently change expectations to preserve a green result.
- A run gets a new exclusive directory under `output/`; never overwrite it. Record revision, dirty state, dataset hash/version, selected cases, mode, and real/scripted call counts.
- Keep datasets synthetic and small. Never add credentials, real student records, private uploads, or production prompts containing personal data.
- Scripted providers measure policy/contract behavior. They do not measure LLM teaching quality, model safety, actual network timeout behavior, or deployed access controls.
- The user explicitly requested this evaluation suite. That authorization does not extend to unrelated unit/browser tests, real-provider spending, database changes, or cloud operations.
