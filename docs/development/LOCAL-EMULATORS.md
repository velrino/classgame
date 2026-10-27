# Local Firebase and browser checks

Use this mode for synthetic teacher/student work. It never reads app credentials or connects to shared Firebase data. The normal local app on ports 3000/4004 stays separate.

## Start the local services

Use Node.js 24, the API Go toolchain, Redis, and Java 25 for the current Firestore emulator. Firebase CLI is an external development tool; it is not an app dependency.

From the repository root:

```sh
npx --yes --package firebase-tools@15.34.1 firebase emulators:start \
  --project demo-classgame --config firebase.emulators.json \
  --only auth,firestore
```

This applies [the current rules](../../firestore.rules) only inside the emulator. It does not deploy rules. Use a separate local Redis endpoint for the API, for example port 6391 with persistence off. Do not reuse a shared working broker or reset another emulator run.

Set the API environment before running `go run ./cmd/api` from `api/`:

```dotenv
HOST=127.0.0.1
PORT=4005
APP_ENV=development
APP_ROLE=api
FIREBASE_PROJECT_ID=demo-classgame
FIREBASE_EMULATORS_ENABLED=true
FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099
FIRESTORE_EMULATOR_HOST=127.0.0.1:8787
FIREBASE_STORAGE_BUCKET=demo-classgame.appspot.com
GENAI_ENABLED=false
POSTHOG_ENABLED=false
TASKS_ENABLED=false
REDIS_URL=redis://127.0.0.1:6391
AUTH_TEACHER_PROVIDERS=password
AUTH_STUDENT_PROVIDERS=anonymous
CORS_ALLOWED_ORIGINS=http://127.0.0.1:3101,http://localhost:3101
```

Leave both Firebase credential settings empty. To create a local teacher and tenant hierarchy, set `FIREBASE_BOOTSTRAP_ENABLED=true` and provide a synthetic `DEMO_TEACHER_EMAIL` ending in `@example.test`, a local-only `DEMO_TEACHER_PASSWORD`, and `DEMO_TEACHER_DISPLAY_NAME`. Bootstrap writes only to the explicit demo emulators in this mode. Do not reuse a real password.

The API refuses partial emulator settings, nonnumeric/nonloopback hosts, real project IDs, configured credentials, production mode, enabled GenAI, or enabled PostHog. Cloud Storage is disabled in this mode. Attachment access needs a separate storage check.

## Start the real web app

Use a separate checkout/build directory from the working development server. Set these public values for `web/`, plus any required nonempty Firebase identifiers from `.env.example` using synthetic values:

```dotenv
NEXT_PUBLIC_CLASSGAME_API_BASE_URL=http://127.0.0.1:4005
NEXT_PUBLIC_FIREBASE_PROJECT_ID=demo-classgame
NEXT_PUBLIC_FIREBASE_API_KEY=demo-classgame-local-only
NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=127.0.0.1
NEXT_PUBLIC_FIREBASE_STORAGE_BUCKET=demo-classgame.appspot.com
NEXT_PUBLIC_FIREBASE_MESSAGING_SENDER_ID=123456789
NEXT_PUBLIC_FIREBASE_APP_ID=1:123456789:web:demo
NEXT_PUBLIC_FIREBASE_EMULATORS_ENABLED=true
NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099
NEXT_PUBLIC_FIRESTORE_EMULATOR_HOST=127.0.0.1:8787
```

Then run `npm run dev -- --hostname 127.0.0.1 --port 3101`. The browser also rejects emulator mode outside development, outside a local origin, with a real project, or with partial endpoints. Firebase analytics is off.

Open `http://127.0.0.1:3101/login` for the local teacher. Use `http://localhost:3101/home` for a separate student origin. Teacher and student also use separate Firebase app instances. Create a class, preview and publish a mission, then join its six-digit code. Use **Go to desk** or movement controls, select **START**, and choose a mission. The 3D world pauses in background tabs; bring it to the front before checking movement.

## Evaluation tools

- [Real browser storage](../../evals/platform/browser-store/README.md): two-tab IndexedDB transactions, leases, exact ACKs, storage limits, and reload recovery.
- [Local HTTP flows](../../evals/platform/local-http/README.md): real emulator identity, publication, learning evidence, replay, completion, and read/write rules.
- [Frontend recovery policy](../../evals/platform/frontend-recovery/README.md): wrong-answer retries, safe progress repair, and server resume mapping.
- The [local proxy](../../evals/platform/local-http/README.md#browser-outage-and-delay-controls) can hold responses or return an API outage while the app keeps its real browser storage.

Each tool requires explicit opt-in and uses owned synthetic records only. No paid model call, cloud deployment, rule deployment, or shared-data reset is included. These checks do not prove production latency, browser storage eviction behavior, or external-provider teaching quality.

## Browser evidence

The local review on 2026-10-27 passed the native 14-case IndexedDB run, including the page reload. The actual app also passed teacher sign-in, class creation, preview and publication, anonymous student joining, 3D movement, and **Go to desk**. During an API outage, answer, reasoning, and feedback records survived a page reload. **Retry** acknowledged the pending records after the API returned.

That review found a wrong-answer path that marked a stage complete after an explanation. The reducer now asks the student to try again and preserves the incorrect answer as evidence. The API still requires its own accepted answers before completion. The recovery policy suite covers this fix and older local progress repair.

The review also found a resume crash when an empty server checkpoint field was `null`. One checkpoint normalizer now handles nullable fields before the board is restored. The policy suite covers both a cached checkpoint and published server question IDs.

After the fixes, the same browser resumed, retried the earlier wrong answer, and finished all three questions. The real reward appeared after confirmation. An emulator read confirmed `completed`, three distinct accepted answers, and all three stages at `1/1`. Waiting copy now makes no delivery claim; the teacher-saved message requires the server completion acknowledgement and an empty queue.
