#!/usr/bin/env python3
"""Opt-in transport smoke: isolated local Redis and synthetic Go handlers only."""
import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[3]
KNOWN_CASES = {"completion", "canonical-replay", "changed-payload-conflict", "scheduled-delivery", "retry", "terminal-archive", "request-trace-correlation", "archived-replay-rejected"}


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise ValueError("duplicate dataset key: " + key)
        value[key] = item
    return value


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args()  # Unknown flags fail before starting any process.
    raw = Path(__file__).with_name("cases.json").read_bytes()
    data = json.loads(raw, object_pairs_hook=unique_object)
    if set(data) != {"schema_version", "suite", "version", "synthetic", "cases"} or data["schema_version"] != 1 or data["suite"] != "queue-transport" or data["synthetic"] is not True:
        raise SystemExit("invalid synthetic queue dataset")
    cases = data["cases"]
    if not isinstance(cases, list) or any(not isinstance(case, str) for case in cases) or len(cases) != len(set(cases)) or set(cases) != KNOWN_CASES or not isinstance(data["version"], str) or not data["version"]:
        raise SystemExit("unknown/duplicate/missing queue case or version")
    redis = shutil.which("redis-server")
    if not redis or not shutil.which("go"):
        raise SystemExit("redis-server and Go are required for the opt-in queue smoke")
    revision = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    dirty = bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).strip())
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_dir = ROOT / "evals/output" / (stamp + "-queue-smoke-" + secrets.token_hex(4))
    run_dir.mkdir(parents=True, exist_ok=False)
    manifest = {"schema_version": 1, "suite": data["suite"], "dataset_version": data["version"], "dataset_sha256": hashlib.sha256(raw).hexdigest(), "bridge_sha256": hashlib.sha256((ROOT / "api/cmd/queuesmoke/main.go").read_bytes()).hexdigest(), "revision": revision, "dirty": dirty, "mode": "local-redis-synthetic", "cases": cases, "real_provider_calls": 0, "database_calls": 0}
    (run_dir / "manifest.json").write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")
    try:
        with tempfile.TemporaryDirectory(prefix="classgame-queue-smoke-") as directory:
            with socket.socket() as reserved:
                reserved.bind(("127.0.0.1", 0))
                port = reserved.getsockname()[1]
            if port == 6379:
                raise RuntimeError("refusing default Redis port")
            command = [redis, "--bind", "127.0.0.1", "--port", str(port), "--save", "", "--appendonly", "no", "--dir", directory]
            with open(Path(directory) / "redis.log", "w") as log:
                broker = subprocess.Popen(command, stdout=log, stderr=log)
                try:
                    for _ in range(100):
                        if broker.poll() is not None:
                            raise RuntimeError("temporary Redis exited before startup")
                        try:
                            with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                                break
                        except OSError:
                            time.sleep(0.05)
                    else:
                        raise RuntimeError("temporary Redis did not start")
                    environment = dict(os.environ, QUEUE_SMOKE_REDIS_URL="redis://127.0.0.1:%s/0" % port, QUEUE_SMOKE_NAME="smoke-%s" % os.getpid())
                    result = subprocess.run(["go", "run", "./cmd/queuesmoke"], cwd=ROOT / "api", env=environment, text=True, capture_output=True, timeout=90)
                    if result.returncode:
                        raise RuntimeError(result.stderr.strip() or "queue smoke failed")
                    report = json.loads(result.stdout.strip().splitlines()[-1])
                    if report.get("status") != "passed" or set(report.get("checks", [])) != KNOWN_CASES or report.get("real_provider_calls") != 0 or report.get("database_calls") != 0:
                        raise RuntimeError("unexpected queue smoke result")
                    (run_dir / "report.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
                    print(json.dumps(dict(report, run_dir=str(run_dir.relative_to(ROOT))), indent=2, sort_keys=True))
                finally:
                    broker.terminate()
                    try:
                        broker.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        broker.kill()
                        broker.wait(timeout=5)
    except Exception as error:
        (run_dir / "report.json").write_text(json.dumps({"status": "failed", "error": str(error), "real_provider_calls": 0, "database_calls": 0}, indent=2) + "\n")
        raise SystemExit("queue smoke failed; see " + str(run_dir.relative_to(ROOT) / "report.json"))


if __name__ == "__main__":
    main()
