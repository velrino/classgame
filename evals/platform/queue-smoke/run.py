#!/usr/bin/env python3
"""Opt-in transport smoke: isolated local Redis and synthetic Go handlers only."""
import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import secrets
import select
import shutil
import socket
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[3]
BASIC_CASES = {"completion", "canonical-replay", "changed-payload-conflict", "scheduled-delivery", "retry", "terminal-archive", "request-trace-correlation", "archived-replay-rejected"}
KNOWN_CASES = BASIC_CASES | {"two-process-duplicate-enqueue", "killed-worker-lease-recovery", "broker-down-enqueue-fails"}


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise ValueError("duplicate dataset key: " + key)
        value[key] = item
    return value


def own_broker(port, expected_pid):
    # Read-only identity check closes the free-port race before any enqueue.
    with socket.create_connection(("127.0.0.1", port), timeout=0.5) as connection:
        connection.sendall(b"INFO server\r\n")
        response = b""
        while len(response) < 8192:
            chunk = connection.recv(1024)
            if not chunk:
                return False
            response += chunk
            if b"process_id:" in response and b"\r\n" in response.split(b"process_id:", 1)[1]:
                actual = response.split(b"process_id:", 1)[1].split(b"\r\n", 1)[0]
                return actual == str(expected_pid).encode("ascii")
        return False


def invoke(executable, environment, *arguments, timeout=30):
    result = subprocess.run([executable] + list(arguments), env=environment, text=True, capture_output=True, timeout=timeout)
    if result.returncode:
        raise RuntimeError(result.stderr.strip() or "synthetic queue process failed")
    return json.loads(result.stdout.strip().splitlines()[-1])


def stop(process):
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)


def ready(process):
    readable, _, _ = select.select([process.stdout], [], [], 10)
    if not readable or json.loads(process.stdout.readline()).get("ready") is not True:
        raise RuntimeError("synthetic process did not become ready")


def process_checks(executable, environment, directory, broker):
    workers = []
    producers = []
    logs = []
    outage = None

    def worker():
        log = open(Path(directory) / ("worker-%s.log" % len(logs)), "w")
        logs.append(log)
        process = subprocess.Popen([executable, "--mode=worker"], env=environment, text=True, stdout=subprocess.PIPE, stderr=log)
        workers.append(process)
        ready(process)
        return process

    try:
        first, second = worker(), worker()
        producers = [subprocess.Popen([executable, "--mode=enqueue", "--id=distributed-once", "--count=16"], env=environment, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE) for _ in range(2)]
        accepted = 0
        for producer in producers:
            stdout, stderr = producer.communicate(timeout=20)
            if producer.returncode:
                raise RuntimeError(stderr.strip() or "concurrent producer failed")
            accepted += json.loads(stdout)["accepted"]
        result = invoke(executable, environment, "--mode=wait", "--id=distributed-once")
        if accepted != 32 or result["starts"] != 1 or result["effects"] != 1:
            raise RuntimeError("two workers did not converge on one healthy-run effect")
        duplicate = dict(result, producer_processes=2, worker_processes=2, accepted_enqueue_calls=accepted)
        stop(first)
        stop(second)

        victim = worker()
        invoke(executable, environment, "--mode=enqueue", "--id=crash-recover")
        started = invoke(executable, environment, "--mode=wait", "--id=crash-recover", "--wait-for=started")
        if started["starts"] != 1 or started["effects"] != 0:
            raise RuntimeError("crash barrier did not stop before the synthetic effect")
        victim.kill()  # SIGKILL on the real worker process, not graceful shutdown.
        victim.wait(timeout=5)
        if victim.returncode != -9:
            raise RuntimeError("worker was not terminated by SIGKILL")
        recovery_started = time.monotonic()
        replacement = worker()
        recovered = invoke(executable, environment, "--mode=wait", "--id=crash-recover", timeout=170)
        elapsed = round(time.monotonic() - recovery_started, 3)
        if recovered["starts"] != 2 or recovered["effects"] != 1:
            raise RuntimeError("killed worker task did not recover once before the effect")
        recovered["recovery_seconds"] = elapsed
        recovered["termination"] = "SIGKILL"
        stop(replacement)

        outage = subprocess.Popen([executable, "--mode=outage"], env=environment, text=True, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        ready(outage)  # Adapter is already connected before Redis is stopped.
        stop(broker)
        stdout, stderr = outage.communicate("broker-down\n", timeout=15)
        if outage.returncode:
            raise RuntimeError(stderr.strip() or "broker-down enqueue did not fail within its budget")
        failed_enqueue = json.loads(stdout.strip().splitlines()[-1])
        if failed_enqueue.get("failed") is not True or failed_enqueue["elapsed_ms"] > failed_enqueue["limit_ms"]:
            raise RuntimeError("unexpected broker outage result")
        return {"two_process_duplicate": duplicate, "killed_worker_recovery": recovered, "broker_down_enqueue": failed_enqueue}
    finally:
        for producer in producers:
            stop(producer)
        for process in workers:
            stop(process)
        if outage is not None:
            stop(outage)
        for log in logs:
            log.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args()  # Unknown flags fail before starting any process.
    raw = Path(__file__).with_name("cases.json").read_bytes()
    data = json.loads(raw, object_pairs_hook=unique_object)
    if set(data) != {"schema_version", "suite", "version", "synthetic", "cases"} or type(data["schema_version"]) is not int or data["schema_version"] != 1 or data["suite"] != "queue-transport" or data["synthetic"] is not True:
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
                    if not own_broker(port, broker.pid):
                        raise RuntimeError("temporary broker identity does not match; no jobs sent")
                    environment = dict(os.environ, QUEUE_SMOKE_REDIS_URL="redis://127.0.0.1:%s/0" % port, QUEUE_SMOKE_NAME="smoke-%s" % os.getpid())
                    executable = str(Path(directory) / "queuesmoke")
                    subprocess.run(["go", "build", "-o", executable, "./cmd/queuesmoke"], cwd=ROOT / "api", env=environment, text=True, capture_output=True, check=True, timeout=90)
                    report = invoke(executable, environment)
                    if report.get("status") != "passed" or set(report.get("checks", [])) != BASIC_CASES or report.get("real_provider_calls") != 0 or report.get("database_calls") != 0:
                        raise RuntimeError("unexpected basic queue smoke result")
                    report["process_checks"] = process_checks(executable, environment, directory, broker)
                    report["checks"] += ["two-process-duplicate-enqueue", "killed-worker-lease-recovery", "broker-down-enqueue-fails"]
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
