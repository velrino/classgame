# Local HTTP flow evaluation

Run the actual API against the guarded Firebase Auth/Firestore emulators and a separate local Redis. Follow [local setup](../../../docs/development/LOCAL-EMULATORS.md). GenAI and PostHog must be off; Cloud Storage is disabled. Current Firestore rules must be loaded into the emulator.

Save the local bootstrap teacher's synthetic login in an ignored file with this shape: `{"email":"browser-eval@example.test","password":"your-local-only-value"}`. Keep its permissions private and never use a real password.

```sh
python3 evals/platform/local-http/run.py --run \
  --login /absolute/path/outside/git/local-login.json
```

The suite refuses missing opt-in, a non-demo project, nonloopback endpoints, product service ports, redirects, and non-synthetic email. Before any fixture write, the API must return its validated `X-ClassGame-Emulator-Project` marker. Endpoints default to API 4005, Auth 9099, and Firestore 8787. No app credential file or cloud SDK is loaded by the runner.

The fourteen scenarios check real Firebase token verification, teacher class creation, deterministic preview SSE, stale draft conflicts, publication, anonymous join/selection, safe student class data, forged completion refusal, incorrect answers, reasoning, feedback, exact replay/conflict, accepted completion, resume/revision pins, and actual Firestore browser read/write rules.

Each run saves a new manifest and report under ignored `evals/output/`. They record source revision/dirty state, runner hash, cases, observed HTTP status/timing, and synthetic fixture IDs. No bearer token, password, or real student text is written. Synthetic data stays in the owned temporary emulator until that emulator stops; this tool never resets a shared database or deletes another fixture.

## Browser outage and delay controls

For a real browser failure/recovery check, start the guarded emulator API on port 4006 instead of 4005. Keep the web app's API URL on port 4005, then run:

```sh
python3 evals/platform/local-http/proxy.py --run
```

Open `http://127.0.0.1:4005/__local_eval` in a separate control tab. **API unavailable** returns a temporary 503 without forwarding. Other controls hold learning, mission selection, or tutor responses for five seconds. **API online** restores normal forwarding. Keep the actual game tab in the foreground while checking movement.

The proxy verifies the upstream demo-emulator marker before binding, forwards only to a fixed numeric loopback API, follows no redirects, bounds concurrent work/body size, and logs no bodies or credentials. Controls require its own origin and a local token. It is a separate evaluation process, not a production debug route. It buffers responses, so it is not a streaming-performance benchmark.

Check a student response during the outage, reload the local app, then restore the API and select **Retry**. The original pending event must survive and be acknowledged once. For a stale-response check, select a mission or start recovery with a delay enabled, leave before the response returns, then confirm the old response does not restore the previous board/session. These UI outcomes must be observed separately; HTTP passes alone do not prove them.
