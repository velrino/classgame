# Evaluate Firestore transactions locally

This optional command calls the real student repository with synthetic fixtures. It does not start the API, load its environment file, authenticate a real student, or call a model provider.

Start a Firestore emulator separately. From `api/`, run:

```sh
FIRESTORE_EMULATOR_HOST=127.0.0.1:8787 go run ./cmd/evalfirestore -run -project demo-classgame
```

The command requires `-run`, a numeric loopback address, and a `demo-` project ID. It uses `option.WithoutAuthentication` and connects only to that emulator endpoint. There is no hosted mode and no emulator reset. Fixtures use a new random prefix on each run; cleanup deletes only documents that this run created. The prefix is printed before execution so an interrupted run can be inspected safely. A cleanup failure fails the run.

The JSON report goes to stdout and case progress goes to stderr. Exit status is nonzero for a failed case, failed cleanup, invalid input, or unavailable emulator. The whole scenario run has a three-minute deadline; cleanup has a separate thirty-second deadline. Save reports outside the source tree, or in the ignored evaluation output folder.

## Coverage

The ten scenarios cover:

- Six simultaneous claims for one tutor event: one durable admission and five pending results.
- Three accepted answers followed by ninety feedback turns: recent context stays bounded, durable facts and paid-admission count remain, and the snapshot can still complete. Exact replay of the evicted answer adds no evidence or admission; changed intent conflicts.
- Six different events at the attempt budget boundary: one admission fills the last slot.
- Expired tutor ownership: old success and failure cannot overwrite a successor.
- Five simultaneous finalization claims: one owner.
- Stale finalizer success and failure after a new owner, new feedback evidence, or an actual snapshot resume.
- Late failure and duplicate success after a finalized result: the ready result stays unchanged.
- Bounded recovery pages, stable cursor advancement, and exclusion of active leases and closed legacy completions.

These are local repository transaction checks. They do not measure paid provider execution, browser IndexedDB, authentication middleware, Firestore security rules, hosted networking, production contention, or deployment safety. The production policy suite remains a separate offline evaluation.
