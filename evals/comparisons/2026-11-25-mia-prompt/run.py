"""Replay the eight calibration answers and run the paired synthetic tutor experiment."""
import argparse
import hashlib
import json
import os
import subprocess
import tempfile
from datetime import datetime, timezone
from pathlib import Path

HERE = Path(__file__).resolve().parent
APPS = HERE.parents[2]
FIXTURE = HERE.parent / "2026-11-25-mia-water-cycle"
MODEL = "gemini-3.6-flash"


def load_frozen(path, digest):
    raw = path.read_bytes()
    if hashlib.sha256(raw).hexdigest() != digest:
        raise ValueError(f"Unexpected fixture version: {path.name}")
    return json.loads(raw)


def paired_runs(tutor, dataset, cases, repeats, output, env):
    for rep in range(repeats):
        for index, case_id in enumerate(cases):
            order = ["current", "compact"] if (rep + index) % 2 == 0 else ["compact", "current"]
            for variant in order:
                target = output / f"{rep + 1}-{case_id}-{variant}.json"
                result = subprocess.run([tutor, "-run", "-phase", "after", "-prompt", variant, "-model", MODEL,
                                "-dataset", dataset, "-case", case_id, "-output", target],
                               cwd=APPS / "api", env=env)
                if result.returncode:
                    # Keep a recorded fallback in the denominator and finish the paired cases.
                    if not target.is_file() or not target.stat().st_size:
                        result.check_returncode()
                    observed = json.loads(target.read_text())
                    if observed["real_provider_calls"] != 1 or observed["tutor"]["generated_by_ai"]:
                        result.check_returncode()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--live", action="store_true", required=True,
                        help="authorize up to 8 judge and 24 tutor calls")
    parser.add_argument("--transfer", action="store_true",
                        help="run only the new passages (8 tutor calls, no judge calls)")
    args = parser.parse_args()
    if not os.environ.get("GENAI_PROJECT_ID"):
        parser.error("Set GENAI_PROJECT_ID and configure Vertex application credentials.")
    env = dict(os.environ, GENAI_LOCATION="global")
    cases_path = FIXTURE / "cases.json"
    data = load_frozen(cases_path, "143520fc713b502cfc70a6c876a5e3019584d55df385781c654e66fdbcdc5889")
    saved = load_frozen(FIXTURE / "results.json", "73f32bedb445168431db16f7669f0af695c8d7b90389c79b9b716604bc690c60")
    cases = {c["id"]: c for c in data["cases"]}
    sources = {c["id"]: c["content"] for c in data["mission"]["knowledge_chunks"]}
    if len(cases) != 4 or len(saved["runs"]) != 8 or not data["synthetic"]:
        raise ValueError("Expected the four synthetic water-cycle cases and eight saved answers")
    output = APPS / "evals/output" / datetime.now(timezone.utc).strftime("mia-prompt-%Y%m%dT%H%M%S%fZ")
    output.mkdir(parents=True, exist_ok=False)
    print(output, flush=True)
    with tempfile.TemporaryDirectory() as tmp:
        tutor = Path(tmp) / "evalretrievalmodel"
        judge = Path(tmp) / "evalgenkit"
        for binary in (tutor, judge):
            subprocess.run(["go", "build", "-o", binary, "./cmd/evals/" + binary.name],
                           cwd=APPS / "api", check=True)
        if args.transfer:
            dataset = HERE / "transfer-cases.json"
            transfer = load_frozen(dataset, "df3419b81f0cb6a737c5e9389ce7323dae8164866335c2700ad023a946bf029c")
            paired_runs(tutor, dataset, [c["id"] for c in transfer["cases"]], 1, output, env)
            return
        for idx, run in enumerate(saved["runs"]):
            case = cases[run["case_id"]]
            evidence = {"mission": data["mission"]["title"], "goal": data["mission"]["learning_goal"],
                        "sources": [sources[i] for i in run["retrieved_chunk_ids"]],
                        "question": case["retrieval"]["turn"]["Prompt"],
                        "response": {k: run["tutor"][k] for k in ("message", "explanation_steps", "why_prompt")}}
            prompt = (HERE / "judge.txt").read_text() + json.dumps(evidence, ensure_ascii=False)
            out = subprocess.run([judge, "genkit-live"], input=json.dumps({"input": prompt, "testCaseId": case["id"]}),
                                 env=env, text=True, check=True, stdout=subprocess.PIPE)
            score = json.loads(out.stdout)
            record = {"case": case["id"], "tutor_model": run["model_requested"],
                      "reference": run["review"]["source_only"], "judge": score}
            (output / f"judge-{idx}.json").write_text(json.dumps(record, indent=2) + "\n")
        paired_runs(tutor, cases_path, cases, 3, output, env)


if __name__ == "__main__":
    main()
