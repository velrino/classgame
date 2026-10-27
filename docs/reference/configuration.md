# Configuration

The runnable templates are [api/.env.example](../../api/.env.example) and [web/.env.example](../../web/.env.example). Local values belong in ignored `api/.env` and `web/.env.local`; templates do not establish valid credentials or provision services.

## API

| Group | Settings and purpose |
| --- | --- |
| Server | `HOST`, `PORT`, `APP_ENV`, `APP_ROLE=api|worker|all`; development defaults use port `4004` |
| Firebase | `FIREBASE_PROJECT_ID`, credentials file or JSON, exact `FIREBASE_STORAGE_BUCKET`, ping timeout |
| Bootstrap | `FIREBASE_BOOTSTRAP_ENABLED`, timeout, `DEMO_TEACHER_*`, and `DEFAULT_*` tenant values; disable against an already provisioned database when unnecessary |
| Identity | `AUTH_TEACHER_PROVIDERS`, `AUTH_STUDENT_PROVIDERS`; supported provider policy is enforced separately from classroom authorization |
| Redis | `REDIS_URL`, ping timeout; credentials stay server-side |
| GenAI | `GENAI_ENABLED`, backend, model, project/location or API key, credentials, request timeout, output-token budget |
| Tasks | `TASKS_ENABLED=false` by default; dedicated `TASKS_REDIS_URL`, `TASKS_QUEUE`, `TASKS_CONCURRENCY` (1–32 per worker); [broker/recovery rules](../operations/QUEUES.md) |
| Learning | `STUDENT_ABANDONMENT_SECONDS`; scheduled delivery requires a configured task worker |
| Telemetry | `POSTHOG_ENABLED`, project token, host; keep disabled for ordinary local development |

Keep secrets outside the repository. A project ID, environment value, or healthy API response does not establish its cloud IAM, billing, or deployed rule state.

### GenAI modes

- Disabled: `GENAI_ENABLED=false` uses deterministic policies and observed-evidence fallbacks.
- Gemini API: set `GENAI_BACKEND=gemini_api`, `GENAI_API_KEY`, and a supported `GENAI_MODEL`.
- Vertex AI: set `GENAI_BACKEND=vertex`, project, location, model, and valid credentials. Deployed workloads should use their service identity where possible.

Model availability and permissions depend on the configured provider. Restart the API after changing environment values; verify a real generation separately from dependency health.

## Web

`NEXT_PUBLIC_FIREBASE_*` identifies the Firebase Web App. `NEXT_PUBLIC_CLASSGAME_API_BASE_URL` points to the API; set it explicitly for each environment. `NEXT_PUBLIC_UPLOADCARE_PUBLIC_KEY` supports optional direct uploads.

Every `NEXT_PUBLIC_*` value is browser-visible. Never place service-account credentials, Redis credentials, private upload keys, or privileged tokens there. Public configuration is typically embedded during the web build.

For local commands see [setup](../development/local-setup.md); for deployed identities and workers see [operations](../operations/cloud-run.md).
