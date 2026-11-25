# Compare Genkit and DeepEval

Both frameworks score the same eight saved Mia answers. The source ID check is local. The optional live check asks a Gemini judge whether the lesson supports the answer. Each judge sees the same passages, question, message, steps and follow-up. It cannot see the tutor model name or the earlier review.

[Measured results and cost](../../comparisons/2026-11-25-genkit-deepeval/README.md) · [Original questions and answers](../../comparisons/2026-11-25-mia-water-cycle/README.md)

## View the recorded runs

From the repository root, run `pnpm install` and `pnpm run eval` to start the web app, API and Genkit Developer UI. If the app and API are already running, use `pnpm run eval:ui` in a second terminal. The first launch downloads the Genkit UI assets. Viewing the saved answers requires no model calls.

Open the [side-by-side comparison](http://127.0.0.1:4000/datasets/mia-water-cycle-20261125/evaluate/mia-genkit-20261125--vs--mia-deepeval-20261125). Each column shows the same eight tutor answers with the recorded scores and judge reasons from that framework. The panel reads a local copy of the dataset. The launcher also connects a local Go evaluation runtime named `classgame-evals`. It exposes the deep equality evaluator and requires no provider credentials. The scores in the two saved runs come from the original comparison.

## Run from the repository root

Install uv and the Go toolchain specified in the API module. The first run downloads dependencies. The default command needs no cloud credentials and makes zero model calls.

```sh
uv run --python 3.12 evals/platform/framework-comparison/run.py
```

To allow 16 paid requests to Gemini 3.6 Flash on global Vertex AI, add the live flag. Set `GENAI_PROJECT_ID` and provide Application Default Credentials or the local API credential file. The runner can read the project and credential path from `api/.env`. It prints the new report directory under `evals/output/`.

```sh
uv run --python 3.12 evals/platform/framework-comparison/run.py --live
```

Genkit uses its Go deep equality evaluator and a custom evaluator action. DeepEval uses its Python exact match metric and a custom metric. Both compare sorted source IDs. One extra local control replaces the IDs with an invented ID and must fail in both frameworks. A live run alternates which framework calls the judge first. Each makes one independent request per answer, with temperature zero, minimal thinking and a 512-token output limit. Both use the same Vertex adapter so the request settings match. A failed request stops the run and preserves the partial report. [Go evaluator and judge](../../../api/cmd/evals/evalgenkit/main.go) · [Python runner and metric](run.py)

For a Go application, Genkit can keep the evaluator next to Go types and tooling. For an evaluation suite owned in Python, DeepEval fits the existing dataset and analysis workflow. This experiment compares their handling of the same rubric. It does not benchmark their built-in semantic metrics, developer UIs or performance. [Genkit evaluation](https://genkit.dev/docs/go/evaluation/) · [DeepEval custom metrics](https://deepeval.com/docs/metrics-custom)
