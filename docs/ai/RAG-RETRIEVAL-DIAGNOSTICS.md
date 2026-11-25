# RAG retrieval diagnostics — 2026-11-15

Related work: https://github.com/velrino/classgame/issues/125

## Problem

The mission retriever split text on whitespace and removed a small ASCII punctuation list from token edges. A curly quote remained part of a token. An em dash did not separate words. In a synthetic water-cycle lesson, `Explain condensation` found `condensation-source`, but `Explain “condensation”` and `Explain—condensation` found no chunk.

The tutor correctly used its existing no-evidence fallback. This was a retrieval failure, not an observed model hallucination. Switching models could not fix this path because the provider was not called.

## Change

Split lexical tokens on characters that are not Unicode letters, numbers or combining marks. Keep lowercasing, the minimum token byte length, score calculation, stable tie order, stage bonus and result limit. There is no embedding model, vector database or new provider dependency.

The evaluation bridge calls the same exported mission retriever used by tutor requests. It then calls production tutor generation with a disabled provider. Python owns case selection and assertions. No ranking implementation is copied into the evaluation code.

## Measured before and after

| Synthetic input | Before: selected source | After: selected source |
| --- | --- | --- |
| `Explain condensation` | condensation-source | condensation-source |
| `Explain “condensation”` | none | condensation-source |
| `Explain—condensation` | none | condensation-source |
| `Explain ‘condensation’` | none | condensation-source |
| `Explain (condensation)` | condensation-source | condensation-source |
| `Explain tectonics` | none | none |
| `Explain evaporation` | evaporation-source | evaporation-source |

The same seven v2 cases passed 4/7 before and 7/7 after. These are fixture assertions, not general retrieval accuracy, a benchmark, or a production quality percentage. Both runs made zero real and zero scripted provider calls. After recovery, the disabled-provider fallback can use the supplied condensation fact; this is not a generated model answer.

The [machine-readable observations](RAG-RETRIEVAL-DIAGNOSTICS.json) include the dataset hash and student-safe output. The baseline uses the historical revision plus instrumentation only; it is not a clean checkout measurement. Full local run manifests/results stay ignored under `evals/output/`.

A first v1 fixture was confounded by the shared word “changes” in the learning goal and both source chunks. It is retired and preserved without changing its expectations. The v2 dataset removes that confound and uses new case IDs. Neither fixture contains a real worksheet or student record.

## Reproduce

```sh
python3 evals/run.py --dataset evals/platform/retrieval-diagnostics/cases-v2.json
python3 evals/run.py
go -C api test ./...
go -C api build ./...
go -C api vet ./...
```

For a fresh before comparison, use a separate checkout with the bridge and dataset but the previous tokenization function. Do not replace the shared branch, overwrite run directories or weaken expectations.

## Review boundaries

- Design: token boundaries belong in the retriever; ranking and generation remain separate.
- Coupling: the bridge uses the students module contract; no HTTP, cloud or database setup is added.
- DRY: normal tutor requests and evaluation cases use one retrieval function.
- Security: fixtures are synthetic. No access-control change, remote call or credential handling is added. Token overlap still does not prove claim support or prompt-injection resistance.

Unicode boundaries do not add stemming, synonym search, accent folding or semantic matching. Generic mission words can still affect ranking. Live model quality, held-out questions, multilingual teaching and deployed behavior remain unverified. See [AI quality](../architecture/AI-QUALITY.md).

## Separate opt-in live sample

After explicit approval for at most six calls, one real Gemini 3.6 Flash request was made with the same synthetic curly-quote question. Before: no evidence, zero calls, existing no-evidence fallback. After: condensation-source, one call, an accepted structured answer. The returned model version was gemini-3.6-flash. Usage: 1,093 prompt, 183 candidate, 700 thought and 1,976 total tokens. Wall time for this one after turn was 7,878 ms; it is not a latency benchmark.

The answer explained the supported core phase change, but also added cooling, heat loss and everyday examples absent from the supplied sentence. Human review therefore does **not** mark it fully grounded. Passing schema validation and carrying a valid source ID did not prove every claim was supported. The [live record](RAG-RETRIEVAL-LIVE.json) preserves both outputs and this review.

The separate `api/cmd/evals/evalretrievalmodel` command requires `-run`, `-phase before|after`, `-model` and a new `-output` file. It permits at most one request per invocation and disables SDK retries. It reuses production retrieval and tutor generation with a dedicated live SDK adapter, not the deployed Redis admission path. The label does not alter retrieval; use the old tokenizer in a separate baseline build. Do not treat it as a general live-evaluation runner.

Example from `api/`, after explicit paid-call approval and with Vertex project, location and application credentials already set in the process environment:

```sh
go run ./cmd/evals/evalretrievalmodel -run -phase after -model gemini-3.6-flash -output ../evals/output/approved-live-after.json
```

The output file must not already exist. The command does not load `.env`, application configuration, databases or private classroom data. The regular Python runner remains offline and rejects live mode.

## Remote CI

PR https://github.com/velrino/classgame/pull/126 passed web validation. API functional checks ran, but its security gate failed in govulncheck: 13 called vulnerabilities were reported across existing dependencies and the Go standard library. The tokenizer patch does not change go.mod, go.sum or the Go version. This remains a release blocker; the PR is open and has not been merged or deployed.

## Security gate fix — 2026-11-15

After the failed run, the author requested a fix. Update Go 1.27.3 to 1.27.4, Echo v5.3.1 to v5.4.1 and klauspost/compress v1.17.11 to v1.20.2. CI reads the Go version from go.mod; local setup guidance now matches it.

Local Go tests, vet, build, the 66-case offline policy suite and seven retrieval cases pass. govulncheck v1.9.0 exits successfully with zero called vulnerabilities; it still reports one advisory in imported packages and three in required modules that it does not detect calls to. This does not prove that every dependency or deployed configuration is safe. Remote CI must be verified for the new commit separately. The original failed run remains historical evidence.

## Continuous regression coverage

The earlier passing remote runs checked the default 66-case dataset, not the seven-case retrieval dataset. That was a coverage gap. The API workflow now explicitly runs cases-v2.json as a separate step. Native Go table cases cover ASCII/curly quotes, em dashes, stable ties, bounded results, unrelated queries and byte-length/language limits. The byte threshold remains unchanged; pa and pá differ under this compatibility policy. Unspaced CJK text is not word-segmented. No new paid model calls are part of this change.
