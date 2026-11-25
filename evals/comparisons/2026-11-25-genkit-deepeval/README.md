# Genkit and DeepEval on eight saved Mia answers

On November 25, 2026, Genkit Go v1.14.0 and DeepEval 4.3.5 checked four synthetic science questions answered by two tutor models. Both frameworks passed all eight source ID checks and rejected the invented ID. Their independent calls to the same Gemini 3.6 Flash judge agreed on all eight source-support decisions. Each matched the earlier Codex review on four of eight answers.

| Check | Genkit | DeepEval | Earlier Codex review |
| --- | ---: | ---: | ---: |
| Source IDs matched | 8/8 | 8/8 | 8/8 |
| Flash answers supported by the retrieved text | 3/4 | 3/4 | 1/4 |
| Argon answers supported by the retrieved text | 3/4 | 3/4 | 3/4 |
| New judge calls | 8 | 8 | — |
| Judge input / output tokens | 2,134 / 308 | 2,134 / 373 | — |
| Estimated judge cost (USD) | $0.00276 | $0.00300 | — |

The tutor made zero new calls. The judge reported 4,949 tokens across 16 calls, including zero thinking tokens. The combined estimate is $0.00575, using $0.75 per million input tokens and $3.75 per million output tokens. These are global promotional rates on the run date. Billing can differ. The two frameworks used the same prompt. Different lengths of the judge's reasons explain the output token difference in this sample. [Vertex pricing](https://cloud.google.com/model-garden/pricing) · [Run JSON with reasons and usage](results.json)

## Where the judge and earlier review differed

| Tutor | Question | Gemini judge in both frameworks | Earlier Codex review |
| --- | --- | --- | --- |
| Flash | Misconception | Accepted the answer about evaporation. | Flagged added details about a puddle, sunlight and heat. |
| Flash | Source limit | Accepted the answer for saying the temperature was missing. | Flagged the follow-up that assumes cooling. |
| Argon | Definition | Accepted the change from vapor to droplets. | Flagged the extra detail that the vapor was in the air. |
| Argon | Source limit | Rejected the phrase about a physical process in the water cycle. | Accepted it using the broader mission context. The judge received only the retrieved passage. |

The earlier review was by Codex with model names visible. Neither review has teacher labels. With four questions, one response per tutor per question, and a judge from the same model family, agreement measures consistency on this sample. A clear example is the Flash puddle answer. Both frameworks accepted it even though the passage only says that liquid water changes into water vapor. The rubric and the allowed evidence deserve attention before using this score to choose a tutor model.

[Original cases, responses and review](../2026-11-25-mia-water-cycle/README.md) · [Run either framework comparison mode](../../platform/framework-comparison/README.md)
