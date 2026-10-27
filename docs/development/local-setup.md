# Local setup

Run the web application and API locally against your configured Firebase project. Use development data and keep service-account files outside Git.

## Requirements

- Go 1.27.3, Node.js 24 or a compatible newer release, npm, and Git. Keep the toolchain aligned with [go.mod](../../api/go.mod).
- Firebase Authentication, Firestore, and Storage. Enable Anonymous and Email/Password authentication; configure Google sign-in if needed.
- A reachable Redis instance. Firebase CLI is needed when managing the project's rules.

```bash
git clone https://github.com/velrino/classgame.git
cd classgame
```

The following copy commands are for a fresh setup. Keep existing environment files when resuming development.

## API

```bash
cd api
cp .env.example .env
```

Set these values in `.env`, using your Firebase project and an existing local Redis instance:

```dotenv
HOST=127.0.0.1
PORT=4004
APP_ENV=development
FIREBASE_PROJECT_ID=your-development-project
FIREBASE_CREDENTIALS_FILE=/absolute/path/outside/git/service-account.json
FIREBASE_STORAGE_BUCKET=your-development-bucket
FIREBASE_BOOTSTRAP_ENABLED=false
AUTH_TEACHER_PROVIDERS=google,password
AUTH_STUDENT_PROVIDERS=anonymous
REDIS_URL=redis://127.0.0.1:6379
CLOUD_TASKS_ENABLED=false
POSTHOG_ENABLED=false
GENAI_ENABLED=false
```

For a new database, enable bootstrap and set `DEMO_TEACHER_EMAIL`, `DEMO_TEACHER_PASSWORD`, and `DEMO_TEACHER_DISPLAY_NAME` to create the minimum tenant hierarchy and teacher. Disable bootstrap when it is no longer needed. The password has no source-code default.

If Redis is not already running, one option is:

```bash
docker run --name classgame-redis -p 127.0.0.1:6379:6379 -d redis:7-alpine
```

Then start the API:

```bash
go mod download
go run ./cmd/api
```

Check [health](http://localhost:4004/): Firebase and Redis should report up. Open the [Scalar API reference](http://localhost:4004/docs) or [OpenAPI contract](http://localhost:4004/openapi.json).

## Web

In a second terminal, from the repository root:

```bash
cd web
cp .env.example .env.local
```

Fill the `NEXT_PUBLIC_FIREBASE_*` values from the same Firebase Web App and set:

```dotenv
NEXT_PUBLIC_CLASSGAME_API_BASE_URL=http://localhost:4004
```

Then:

```bash
npm ci
npm run dev
```

Open [teacher login](http://localhost:3000/login) or the [student game](http://localhost:3000/home). Create a class and publish a mission as a teacher, then join its six-digit code as a student. Asset decoder synchronization runs during dependency installation.

## Optional services

`GENAI_ENABLED=false` selects deterministic fallbacks. For live generation, configure a supported backend and model using the [configuration reference](../reference/configuration.md); restart the API after changing its environment.

With Cloud Tasks disabled, completion finalization can fall back to synchronous work. Background copies and scheduled abandonment processing need their configured worker infrastructure; disabling the queue does not reproduce scheduled delivery.

For a new Firebase project, deploy the reviewed [Firestore rules](../../firestore.rules) explicitly to that project. This changes remote access rules and is separate from starting local processes.

Next: [validation](validation.md), [API contracts](../reference/api.md), and [configuration](../reference/configuration.md).
