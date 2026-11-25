# Mia tutor comparison · Gemini 3.6 Flash and Gemini 4 Argon

**Gemini 4 Argon stayed closer to the lesson. Gemini 3.6 Flash usually answered sooner.** Both models answered all four grade-five science questions correctly. Argon kept every factual claim within the supplied lesson on three questions. Flash did so on one. This was a small live comparison on November 25, 2026 UTC. [Cases and scoring rules](cases.json) · [All eight responses, scores and token counts](results.json)

## Results

| Measure across the same four questions | 3.6 Flash | 4 Argon | What changed |
| --- | ---: | ---: | --- |
| Accepted tutor responses | 4/4 | 4/4 | Both worked with Mia's JSON schema |
| Correct answer to the learner's question | 4/4 | 4/4 | Tie |
| Every factual claim supported by the lesson | 1/4 | 3/4 | Argon passed two more cases |
| Follow-up answerable using the lesson alone | 0/4 | 4/4 | Argon asked learners to explain the material back |
| Expected source IDs in the final tutor response | 4/4 | 4/4 | Tie after the app filtered the IDs |
| Median elapsed time per turn | 5.15 s | 7.20 s | Argon took 2.06 s longer |
| Input tokens | 4,473 | 5,028 | Argon used 555 more |
| Response tokens | 614 | 732 | Argon used 118 more |
| Thinking tokens reported by Vertex | 2,790 | 0 | Provider-reported count |
| Total tokens reported | 7,877 | 5,760 | Argon used 26.9% fewer |
| Estimated cost for all four turns (USD) | $0.01612 | $0.01738 | Argon cost about 7.8% more at the rates below |

## What the learner saw

| Question | 3.6 Flash | 4 Argon | Elapsed time, Flash / Argon |
| --- | --- | --- | ---: |
| Explain condensation | Got the direction right and added cooling, warm vapor and a cold-glass example. | Asked the learner to restate the change. It still added one detail absent from the passage: the vapor was “in the air”. | 5.609 / 5.604 s |
| “I think evaporation turns vapor into liquid. Is my direction right?” | Corrected the direction, then introduced sunlight and a puddle. | Recognized that the learner had identified both states. It corrected their order and asked the learner to explain the change. | 6.316 / 7.507 s |
| At exactly what temperature does condensation start? | Correctly said the lesson gave no temperature. Its follow-up assumed cooling. | Said the temperature was missing and asked about the change the passage did describe. | 4.416 / 7.249 s |
| Compare the directions of evaporation and condensation | Explained both directions correctly and invited a personal example. | Explained both directions correctly and asked which state each process starts with. | 4.687 / 7.157 s |

Flash's everyday examples offer a way to connect the lesson to a learner's experience. Mia's current instructions require factual claims to stay within the published material. Under that rule, a plausible example can still lose a grounding point. The follow-up score asks a narrower question: could a learner answer using only the supplied lesson? Learner engagement and learning gains were not measured.

## Method and limits

- Four synthetic prompts shared the same two sentences about water changing state. The cases covered a definition, a reversed explanation, a missing fact and a comparison. Production retrieval returned the expected passages for both models in every pair.
- The existing live runner called the production tutor function with a dedicated Vertex SDK adapter. Each model received the same case, prompt builder, JSON schema and 2,048-token output cap. Eight requests ran in sequence with one attempt each. The first model alternated across cases. Gemini 4 Argon returned four accepted responses, so the 3.7 fallback was unnecessary. [Live runner](../../../api/cmd/evalretrievalmodel/main.go)
- The adapter requests temperature 0.25 and leaves thinking settings at each model's default. Argon ignores custom temperature. Its zero reported thinking tokens describe the usage record. The experiment had no independent measure of internal reasoning. [Argon model guide](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/guides/gemini-4-argon)
- Codex reviewed the student-visible message, steps and follow-up against the saved rubric, with model names visible. Each question ran once per model. The timer includes generation, application checks and a small amount of Git reporting. It starts after retrieval. A 5 ms difference on the definition is too small to interpret.
- Cost uses the published global promotional rates applicable on the run date. Per million tokens, Flash cost $0.75 for input and $3.75 for output including thinking. Argon cost $2 and $10 respectively. Estimated combined cost was $0.03350, assuming the promotional credits apply. Actual billing can differ. [Google Cloud model pricing](https://cloud.google.com/model-garden/pricing)

## Reproduce one observation

From `api/`, with the Vertex project, global location and application credentials configured in the environment, run:

```sh
go run ./cmd/evalretrievalmodel -run -phase after -model gemini-4-argon \
  -dataset ../evals/comparisons/2026-11-25-mia-water-cycle/cases.json \
  -case misconception -output /tmp/classgame-argon-misconception.json
```

Each invocation can make one paid request and requires a new output path. The other model ID is `gemini-3.6-flash`. All eight original invocations exited with code 0. The JSON includes their execution order, model versions, fixture hash, Git revision, selected sources and individual timings.
