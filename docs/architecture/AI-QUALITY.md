# Tutor evidence and private context

The tutor uses the student's pinned published mission. The server resolves the question and expected answer before retrieval; model output does not determine authorization or authoritative progress.

## Retrieval and insufficient evidence

Lexical retrieval splits words at Unicode punctuation, excludes chunks with zero relevance score and retains the existing bounded ranking. Positive overlap is a selection rule, not a confidence estimate or proof that a claim is supported.

If normalization leaves no usable chunks, tutor generation returns the deterministic policy without calling the provider. Its explanation uses the server-grounded published question/expected answer where available; otherwise it offers a bounded next step. Teacher metadata records the evidence gap and no source IDs are invented.

Do not add a vector database to compensate for missing source quality or an undefined acceptance criterion. Source location, original-excerpt links, and claim-support validation remain separate improvements.

## Provider context

Before a learner-facing provider call, prepare a narrow copy of adaptation context:

- Keep only the allowlisted explanation preference (`steps` or `visual`).
- Preserve skill signals only when their normalized identifier matches a stage in the published mission.
- Exclude free-text student-memory strengths/needs, next-adaptation narrative, and class-pattern prose.
- Retain the current student's bounded observable history, the published mission, and selected chunks as untrusted task data.

The original server context remains available to teacher-only learning analysis and the existing output leak filter. Do not persist this sanitized copy over durable learning memory. Published stage IDs are identifiers, not evidence of skill mastery.

This reduces unnecessary private narrative disclosure. It does not prove semantic leakage or prompt injection impossible. Retrieved material, student input, and generated claims still need independent access, grounding, and output controls.

## Synthetic review matrix

These are proposed manual acceptance cases, not measured results or a new executable suite. Use synthetic records and keep each case isolated.

| Case | Expected behavior |
| --- | --- |
| Relevant mission chunk | Bounded retrieval; generated source IDs belong to supplied chunks |
| Only zero-score chunks | No provider call; question-grounded deterministic response and evidence-gap metadata |
| Empty/filtered chunks | Same bounded fallback; no invented citation |
| Private memory/class prose | Raw narrative absent from provider payload; durable teacher context remains intact |
| Published stage identifier | Normalized stage ID retained; unknown narrative or arbitrary labels omitted |
| Visual explanation preference | Preference preserved; unsupported preference values fall back to steps |
| Hostile worksheet/student text | Treated as untrusted evidence; cannot authorize actions or change access policy |
| Provider timeout | Existing fallback selected; timeout is not reported as a successful live generation |

For a later evaluation workflow, version the prompt, model, document revision, retrieval configuration, and rubric. Review teaching quality and factual support separately from deterministic access/source-ID checks. Keep held-out cases and report privacy failures independently of overall quality scores.

Run existing [Go checks and affected workflow checks](../development/VALIDATION.md) for implementation changes. Repository instructions require an explicit request before adding tests.

The [retrieval diagnostic case](../ai/RAG-RETRIEVAL-DIAGNOSTICS.md) records a synthetic punctuation failure and its before/after evidence. It does not claim live model quality.

## Token boundaries and language limits

The tokenizer splits on punctuation and whitespace using Unicode letter, number and combining-mark classes. It is not a full Unicode word segmenter, does not segment unspaced CJK text, and does not normalize composed/decomposed accents. The existing minimum is three UTF-8 bytes, not three letters: pa is excluded while pá is retained. This compatibility policy is pinned by native Go cases. A rune-count threshold would change the accepted vocabulary and needs a separate language-policy decision; rune count is not grapheme count.

CI runs both the default 66-case policy dataset and the seven retrieval-diagnostic v2 cases. Native Go cases cover punctuation, token boundaries, byte-length compatibility and the retrieval result bound. The real Gemini sample stays opt-in and never runs in CI.
