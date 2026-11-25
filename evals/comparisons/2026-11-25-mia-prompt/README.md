# Can a shorter prompt keep Mia closer to the lesson?

**Mia used the lesson more consistently with the compact prompt. It also used more reasoning tokens, cost more per matched set and took longer.** I kept the compact version in the evaluation tool. The production API still uses the current instruction. [Answers, scores and usage](results.json)

## What changed for the learner?

Both versions received a passage saying that evaporation changes liquid water into water vapor. A learner had reversed that direction. Both tutors corrected the mistake. Their next questions in the first repeat show the difference.

| Current follow-up | Compact follow-up |
| --- | --- |
| “Can you think of a time you saw liquid water dry up in the sun? What happened to it?” | “How does liquid water change during evaporation?” |

The current prompt asks for a personal observation. The learner can answer the compact version directly from the lesson.

| Measure | Original questions · current | Original questions · compact | New questions · current | New questions · compact |
| --- | ---: | ---: | ---: | ---: |
| Accepted model answers | 12/12 | 12/12 | 4/4 | 3/4 |
| Student received a correct answer to the question | 12/12 | 12/12 | 4/4 | 3/4 |
| Model delivered an answer supported throughout by the lesson | 4/12 | 12/12 | 2/4 | 3/4 |
| Follow-up answerable from the lesson | 0/12 | 9/12 | 0/4 | 2/4 |
| Final source IDs matched the expected set | 12/12 | 12/12 | 4/4 | 4/4 |

One compact request failed after 43.2 seconds. The API returned a fallback with the right source ID and a sentence supported by the passage, but it missed the question about infiltration speed. The table counts this as a missed model answer. The JSON also scores the fallback text the learner received. I kept this attempt and finished the two remaining calls separately. The available error does not identify the underlying cause.

## What changed in cost and speed?

The table uses the 15 pairs where both versions returned a model answer. It excludes both sides of the failed pair so missing usage cannot look like savings.

| Measure on the same 15 turns | Current | Compact | Change |
| --- | ---: | ---: | ---: |
| Input tokens | 16,798 | 14,533 | −13.5% |
| Answer tokens | 2,198 | 2,105 | −4.2% |
| Reasoning tokens reported by Vertex | 9,566 | 13,347 | +39.5% |
| Total tokens | 28,562 | 29,985 | +5.0% |
| Estimated cost (USD) | $0.05671 | $0.06884 | +21.4% |
| Median elapsed time | 5.09 s | 5.89 s | +0.79 s |

Across 32 tutor attempts and 8 judge calls, reported usage cost an estimated $0.13234 at global promotional rates of $0.75 per million input tokens and $3.75 per million output tokens including reasoning. The failed request returned no usage. Its billing is unknown. [Vertex pricing](https://cloud.google.com/model-garden/pricing)

## Did the judge improve?

I gave the Gemini judge a clearer rule for examples and follow-up questions, plus the mission title and goal. On the same eight saved answers, it agreed with the Codex reference labels on **4/8 before and 4/8 after**. It still accepted an evaporation answer that added heat and sunlight. These labels come from an AI review. A teacher could judge the teaching value of an example differently. [Earlier Genkit and DeepEval scores](../2026-11-25-genkit-deepeval/README.md) · [Revised judge instruction](judge.txt)

## Experiment

Mia's current system instruction uses 240 words. The candidate uses 131. It explicitly tells Mia to support examples and follow-up questions with the lesson and to say when a detail is missing. Both prompts received the same student question, retrieved passages and JSON schema. This experiment measures the combined effect of shorter wording and clearer instructions. [Current instruction](../../../api/internal/modules/ai/agents/tutor/prompt.go) · [Compact instruction](../../../api/cmd/evals/evalretrievalmodel/compact.txt)

The Python runner alternated which version went first. The Go command ran production retrieval and tutor generation, switching only the system instruction at the provider boundary. Vertex AI served Gemini 3.6 Flash with temperature 0.25, a 2,048-token output cap and one attempt per request. The timer covers generation and local response handling after retrieval. [Runner](run.py) · [Go adapter](../../../api/cmd/evals/evalretrievalmodel/main.go)

Codex scored the message, steps and follow-up from a shuffled view with labels and token counts hidden. The provider failure revealed its variant during recovery. Source IDs were checked by set equality. The original four questions ran three times per version. Four new questions about runoff and infiltration ran once per version. All cases are synthetic and cover one school subject. Learning gains and teacher preference need a separate study. [Scoring rules](RUBRIC.md) · [Original cases](../2026-11-25-mia-water-cycle/cases.json) · [New passages and questions](transfer-cases.json)

## Reproduce

From the apps directory, view the saved scores with zero provider calls.

```sh
jq '.summary' evals/comparisons/2026-11-25-mia-prompt/results.json
```

To repeat generation with Vertex credentials, set `GENAI_PROJECT_ID` to your authorized project. The first command makes up to 8 judge and 24 tutor calls. The second makes up to 8 tutor calls. Each writes new observations under `evals/output/`. Review new answers with the same rubric.

```sh
python3 evals/comparisons/2026-11-25-mia-prompt/run.py --live
python3 evals/comparisons/2026-11-25-mia-prompt/run.py --live --transfer
```
