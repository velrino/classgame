# Retrieval diagnostics

Run the current synthetic scenario from the repository root:

```sh
python3 evals/run.py --dataset evals/platform/retrieval-diagnostics/cases-v2.json
```

The existing Python runner calls the Go bridge. The bridge calls the production mission retriever, then the production tutor with a disabled provider. It records selected chunk IDs and the fallback. No ranking rule is copied into the harness. No real provider calls, network, database or student data are used.

`cases-v2.json` compares plain text, curly double and single quotes, an em dash, parentheses, another lesson topic and an unrelated question. Each case has the same published synthetic lesson and a retrieval limit of one. Expected IDs are reference labels for these cases, not a general quality score.

`cases.json` (v1.0.0) is retired. Its generic learning-goal word “changes” also occurred in the source chunks and masked the punctuation failure. The unrelated-question expectation failed. Do not count this run as the baseline or change its historical expectations. Version 1.1.0 uses v2 case IDs and removes that fixture confound.

The current scenario checks evidence selection and deterministic fallback only. It does not measure a live model answer, groundedness, teaching quality, latency or a production incident. See the [measured record](../../../docs/ai/RAG-RETRIEVAL-DIAGNOSTICS.md).

A separately authorized [live sample](../../../docs/ai/RAG-RETRIEVAL-DIAGNOSTICS.md#separate-opt-in-live-sample) used one real request. It does not change the offline command or its zero-provider-call coverage.
