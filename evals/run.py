#!/usr/bin/env python3
"""Offline policy evaluations using production Go and TypeScript rules."""

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import sys
import tempfile


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_DATASET = ROOT / "evals/platform/ai-policy/cases.json"
KINDS = {"tutor", "auth", "projection", "learning", "scope", "memory", "object_path", "evidence", "tutor_event", "finalization"}
OPERATORS = {"equals", "contains", "not_contains", "absent"}
MISSING = object()


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise ValueError("duplicate JSON key: " + key)
        value[key] = item
    return value


def reject_constant(value):
    raise ValueError("non-finite JSON number: " + value)


def load_dataset(path):
    with path.open("rb") as source:
        raw = source.read(2 * 1024 * 1024 + 1)
    if len(raw) > 2 * 1024 * 1024:
        raise ValueError("dataset exceeds 2 MiB")
    data = json.loads(raw, object_pairs_hook=unique_object, parse_constant=reject_constant)
    suite = data.get("suite") if isinstance(data, dict) else None
    allowed = {"schema_version", "suite", "version", "synthetic", "cases"}
    if suite == "ai-policy":
        allowed |= {"tutor_context", "mission"}
    if not isinstance(data, dict) or set(data) != allowed:
        raise ValueError("dataset requires exactly: " + ", ".join(sorted(allowed)))
    if type(data["schema_version"]) is not int or data["schema_version"] != 1 or not isinstance(suite, str) or suite not in {"ai-policy", "offline-evidence", "frontend-recovery"} or data["synthetic"] is not True:
        raise ValueError("requires schema 1, a supported suite, and synthetic=true")
    if not isinstance(data["version"], str) or not data["version"].strip():
        raise ValueError("dataset version is required")
    if suite == "ai-policy" and (not isinstance(data["tutor_context"], dict) or not isinstance(data["mission"], dict)):
        raise ValueError("tutor_context and mission must be objects")
    cases = data["cases"]
    if not isinstance(cases, list) or not 1 <= len(cases) <= 100:
        raise ValueError("dataset requires 1 to 100 cases")
    seen = set()
    for case in cases:
        if not isinstance(case, dict):
            raise ValueError("case must be an object")
        case_id, kind = case.get("id"), case.get("kind")
        if not isinstance(case_id, str) or not re.fullmatch(r"[a-z0-9][a-z0-9-]{2,79}", case_id):
            raise ValueError("case ID must be a short lowercase identifier")
        if case_id in seen:
            raise ValueError("duplicate case ID: " + case_id)
        seen.add(case_id)
        if not isinstance(kind, str) or kind not in (KINDS if suite == "ai-policy" else {"recovery"} if suite == "frontend-recovery" else {"browser"}) or set(case) != {"id", "kind", kind, "expected"}:
            raise ValueError("unknown case kind or fields: " + case_id)
        if not isinstance(case[kind], dict):
            raise ValueError("case input must be an object: " + case_id)
        if kind == "tutor" and case[kind].get("provider_mode") not in {"respond", "disabled", "timeout"}:
            raise ValueError("unknown scripted provider mode: " + case_id)
        expected = case["expected"]
        if not isinstance(expected, dict) or not expected or set(expected) - OPERATORS:
            raise ValueError("unknown or empty expectation operators: " + case_id)
        count = 0
        for operator, checks in expected.items():
            if operator == "absent":
                if not isinstance(checks, list) or not all(isinstance(item, str) and item for item in checks):
                    raise ValueError("absent requires field paths: " + case_id)
            else:
                if not isinstance(checks, dict) or not all(isinstance(item, str) and item for item in checks):
                    raise ValueError("expectations require field paths: " + case_id)
                if operator == "contains" and not all(isinstance(item, str) and item for item in checks.values()):
                    raise ValueError("contains requires nonempty strings: " + case_id)
                if operator == "not_contains" and not all(isinstance(items, list) and items and all(isinstance(item, str) and item for item in items) for items in checks.values()):
                    raise ValueError("not_contains requires nonempty string lists: " + case_id)
            count += len(checks)
        if not 1 <= count <= 100:
            raise ValueError("case requires 1 to 100 assertions: " + case_id)
    return data, hashlib.sha256(raw).hexdigest()


def select_cases(cases, selection):
    if not selection:
        return cases
    requested = selection.split(",")
    if any(not item for item in requested) or len(set(requested)) != len(requested):
        raise ValueError("selected case IDs must be nonempty and unique")
    known = {case["id"] for case in cases}
    if set(requested) - known:
        raise ValueError("unknown case IDs: " + ", ".join(sorted(set(requested) - known)))
    return [case for case in cases if case["id"] in requested]


def invoke_bridge(data, cases):
    if data["suite"] in {"offline-evidence", "frontend-recovery"}:
        payload = {"cases": [{key: value for key, value in case.items() if key != "expected"} for case in cases]}
        run = subprocess.run(["node", str(ROOT / ("evals/platform/" + data["suite"] + "/bridge.mjs"))],
                             input=json.dumps(payload), text=True, capture_output=True,
                             env={"PATH": os.environ.get("PATH", os.defpath), "TZ": "UTC"}, timeout=30)
        if run.returncode:
            raise RuntimeError("offline browser bridge rejected the batch:\n" + run.stderr)
        output = json.loads(run.stdout, object_pairs_hook=unique_object)
        results = output.get("results") if isinstance(output, dict) else None
        if not isinstance(output, dict) or output.get("mode") != "offline" or output.get("real_provider_calls") != 0 or not isinstance(results, list) or not all(isinstance(item, dict) for item in results) or [item.get("id") for item in results] != [case["id"] for case in cases]:
            raise RuntimeError("browser bridge returned an invalid result batch")
        return results
    payload = {"tutor_context": data["tutor_context"], "mission": data["mission"],
               "cases": [{key: value for key, value in case.items() if key != "expected"} for case in cases]}
    build_env = dict(os.environ, GOPROXY="off")
    with tempfile.TemporaryDirectory(prefix="classgame-evalbridge-") as temporary:
        binary = str(Path(temporary) / "evalbridge")
        build = subprocess.run(["go", "build", "-o", binary, "./cmd/evalbridge"], cwd=str(ROOT / "api"),
                               env=build_env, text=True, capture_output=True, timeout=180)
        if build.returncode:
            raise RuntimeError("Go bridge build failed (preinstall the required toolchain/dependencies):\n" + build.stderr)
        # Do not pass application credentials or provider settings to the bridge.
        run = subprocess.run([binary], input=json.dumps(payload), text=True, capture_output=True,
                             env={"PATH": os.environ.get("PATH", os.defpath), "TZ": "UTC"}, timeout=30)
    if run.returncode:
        raise RuntimeError("offline bridge rejected the batch:\n" + run.stderr)
    output = json.loads(run.stdout, object_pairs_hook=unique_object)
    if not isinstance(output, dict) or output.get("mode") != "offline" or output.get("real_provider_calls") != 0:
        raise RuntimeError("bridge did not report offline execution with zero real provider calls")
    results = output.get("results")
    if not isinstance(results, list) or not all(isinstance(item, dict) for item in results) or [item.get("id") for item in results] != [case["id"] for case in cases]:
        raise RuntimeError("bridge returned missing, duplicate, or reordered case results")
    return results


def field(value, path):
    for part in path.split("."):
        if isinstance(value, dict) and part in value:
            value = value[part]
        elif isinstance(value, list) and part.isdigit() and int(part) < len(value):
            value = value[int(part)]
        else:
            return MISSING
    return value


def rendered(value):
    return value if isinstance(value, str) else json.dumps(value, sort_keys=True, ensure_ascii=False)


def grade(case, observed):
    checks = []
    if "error" in observed:
        checks.append({"check": "production call succeeded", "passed": False, "detail": observed["error"]})
    for operator, expected in case["expected"].items():
        items = ((path, None) for path in expected) if operator == "absent" else expected.items()
        for path, wanted in items:
            actual = field(observed, path)
            if operator == "absent":
                passed = actual is MISSING
            elif actual is MISSING:
                passed = False
            elif operator == "equals":
                passed = json.dumps(actual, sort_keys=True) == json.dumps(wanted, sort_keys=True)
            elif operator == "contains":
                passed = wanted in rendered(actual)
            else:
                passed = all(item not in rendered(actual) for item in wanted)
            checks.append({"check": operator + ":" + path, "passed": passed})
    evidence = dict(observed)
    prompt = evidence.pop("provider_input", "")
    if prompt:
        evidence["provider_input_sha256"] = hashlib.sha256(prompt.encode()).hexdigest()
        evidence["provider_input_bytes"] = len(prompt.encode())
    return {"id": case["id"], "kind": case["kind"], "passed": all(check["passed"] for check in checks),
            "checks": checks, "observed": evidence}


def git_value(*arguments):
    result = subprocess.run(["git", "-C", str(ROOT), *arguments], text=True, capture_output=True, timeout=10)
    if result.returncode:
        raise RuntimeError("cannot record repository revision")
    return result.stdout.strip()


def write_run(output_root, data, digest, cases, results):
    now = dt.datetime.now(dt.timezone.utc)
    output_root.mkdir(parents=True, exist_ok=True)
    run_dir = output_root / (now.strftime("%Y%m%dT%H%M%S.%fZ") + "-" + secrets.token_hex(4))
    run_dir.mkdir(mode=0o700)  # exclusive: never reuse or overwrite a run
    passed = sum(result["passed"] for result in results)
    manifest = {"schema_version": 1, "suite": data["suite"], "dataset_version": data["version"],
                "dataset_sha256": digest, "revision": git_value("rev-parse", "HEAD"),
                "working_tree_dirty": bool(git_value("status", "--porcelain")), "mode": "offline",
                "coverage": "policy_and_contract", "llm_teaching_quality_measured": False,
                "created_at": now.isoformat(), "case_ids": [case["id"] for case in cases],
                "real_provider_calls": 0,
                "scripted_provider_calls": sum(result["observed"].get("scripted_provider_calls", 0) for result in results),
                "passed": passed, "failed": len(results) - passed}
    for name, value in [("manifest.json", manifest), ("results.json", results)]:
        with (run_dir / name).open("x", encoding="utf-8") as target:
            json.dump(value, target, ensure_ascii=False, indent=2)
            target.write("\n")
    return run_dir, passed


def main():
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    parser.add_argument("--dataset", type=Path, default=DEFAULT_DATASET)
    parser.add_argument("--cases", default="", help="comma-separated case IDs")
    parser.add_argument("--output", type=Path, default=ROOT / "evals/output")
    parser.add_argument("--mode", choices=["offline"], default="offline")
    parser.add_argument("--list", action="store_true", help="validate and list cases without running")
    options = parser.parse_args()
    try:
        data, digest = load_dataset(options.dataset)
        cases = select_cases(data["cases"], options.cases)
        if options.list:
            print("\n".join(case["id"] for case in cases))
            return 0
        observations = invoke_bridge(data, cases)
        results = [grade(case, observed) for case, observed in zip(cases, observations)]
        run_dir, passed = write_run(options.output, data, digest, cases, results)
        print("{} {}/{} offline policy cases; real provider calls: 0".format("PASS" if passed == len(cases) else "FAIL", passed, len(cases)))
        print("Run:", run_dir)
        for result in results:
            if not result["passed"]:
                print("Failed:", result["id"], ", ".join(check["check"] for check in result["checks"] if not check["passed"]))
        return 0 if passed == len(cases) else 1
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print("evals:", error, file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
