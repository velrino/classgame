#!/usr/bin/env python3
"""Opt-in shared-admission evaluation on a private temporary local Redis."""
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


def stop(process):
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)


def owns_port(port, pid):
    with socket.create_connection(("127.0.0.1", port), timeout=0.5) as connection:
        connection.sendall(b"INFO server\r\n")
        response = b""
        while len(response) < 8192:
            response += connection.recv(1024)
            if b"process_id:" in response and b"\r\n" in response.split(b"process_id:", 1)[1]:
                return response.split(b"process_id:", 1)[1].split(b"\r\n", 1)[0] == str(pid).encode()
    return False


def invoke(command, environment):
    result = subprocess.run(command, env=environment, text=True, capture_output=True, timeout=30)
    if result.returncode:
        raise RuntimeError(result.stderr or "synthetic command failed")
    return json.loads(result.stdout)


def main():
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    parser.add_argument("--run", action="store_true", help="explicitly allow owned synthetic local Redis work")
    options = parser.parse_args()
    if not options.run:
        parser.error("pass --run; this check starts its own temporary local Redis")
    for binary in ["redis-server", "redis-cli", "go"]:
        if not shutil.which(binary):
            parser.error("required command: " + binary)
    nonce = secrets.token_hex(8)
    scope = "smoke-budget-" + nonce
    results = []
    processes = []
    with tempfile.TemporaryDirectory(prefix="classgame-provider-budget-") as temporary:
        directory = Path(temporary)
        binary = str(directory / "budgetsmoke")
        subprocess.run(["go", "build", "-o", binary, "./cmd/budgetsmoke"], cwd=ROOT / "api",
                       env=dict(os.environ, GOPROXY="off"), check=True, capture_output=True, timeout=180)
        with socket.socket() as available:
            available.bind(("127.0.0.1", 0))
            port = available.getsockname()[1]
        if port == 6379:
            raise RuntimeError("refuse the normal Redis port")
        environment = {"PATH": os.environ.get("PATH", os.defpath), "TZ": "UTC",
                       "BUDGET_SMOKE_REDIS_URL": "redis://127.0.0.1:" + str(port), "BUDGET_SMOKE_SCOPE": scope}
        with (directory / "redis.log").open("w") as log:
            broker = subprocess.Popen(["redis-server", "--bind", "127.0.0.1", "--port", str(port), "--save", "",
                                       "--appendonly", "no", "--maxmemory-policy", "noeviction", "--daemonize", "no", "--dir", temporary],
                                      stdin=subprocess.DEVNULL, stdout=log, stderr=log)
            environment["BUDGET_SMOKE_REDIS_PID"] = str(broker.pid)
            try:
                deadline = time.monotonic() + 10
                while True:
                    if broker.poll() is not None:
                        raise RuntimeError("owned Redis did not start")
                    try:
                        if owns_port(port, broker.pid):
                            break
                    except OSError:
                        pass
                    if time.monotonic() > deadline:
                        raise RuntimeError("cannot verify owned Redis PID")
                    time.sleep(0.05)
                basic = invoke([binary], environment)
                if basic.get("real_provider_calls") != 0 or basic.get("database_calls") != 0:
                    raise RuntimeError("synthetic report has wrong call counts")
                results.extend(basic["results"])
                for _ in range(2):
                    process = subprocess.Popen([binary, "--mode=instance"], env=environment, text=True,
                                               stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                    processes.append(process)
                    if json.loads(process.stdout.readline()).get("ready") is not True:
                        raise RuntimeError("synthetic instance not ready")
                for process in processes:
                    process.stdin.write("\n")
                    process.stdin.flush()
                observations = []
                for process in processes:
                    stdout, stderr = process.communicate(timeout=15)
                    if process.returncode:
                        raise RuntimeError(stderr)
                    observations.append(json.loads(stdout))
                peak = int(subprocess.check_output(["redis-cli", "-h", "127.0.0.1", "-p", str(port), "--raw", "GET", scope + ":peak"], text=True))
                total = sum(item["accepted"] + item["denied"] + item["failures"] for item in observations)
                results.append({"id": "two-process-provider-cap", "passed": total == 16 and peak <= 2 and peak > 0 and all(item["failures"] == 0 for item in observations),
                                "observed": {"peak": peak, "limit": 2, "processes": observations}})
                outage = subprocess.Popen([binary, "--mode=outage"], env=environment, text=True,
                                          stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                processes.append(outage)
                if json.loads(outage.stdout.readline()).get("ready") is not True:
                    raise RuntimeError("outage probe not ready")
                stop(broker)
                stdout, stderr = outage.communicate("\n", timeout=5)
                if outage.returncode:
                    raise RuntimeError(stderr)
                observed = json.loads(stdout)
                results.append({"id": "broker-outage-fails-closed", "passed": observed["unavailable"] and not observed["provider_called"] and observed["elapsed_ms"] < 2000,
                                "observed": observed})
            finally:
                for process in processes:
                    stop(process)
                stop(broker)
    passed = sum(result["passed"] for result in results)
    now = dt.datetime.now(dt.timezone.utc)
    output = ROOT / "evals/output" / (now.strftime("%Y%m%dT%H%M%S.%fZ") + "-provider-budget-" + nonce)
    output.mkdir(parents=True, mode=0o700)
    manifest = {"mode": "local-redis", "suite": "provider-budget", "revision": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
                "working_tree_dirty": bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True)),
                "real_provider_calls": 0, "database_calls": 0, "passed": passed, "total": len(results), "created_at": now.isoformat(),
                "harness_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), "owned_processes_stopped": True}
    for name, value in [("manifest.json", manifest), ("report.json", {"passed": passed, "total": len(results), "results": results})]:
        with (output / name).open("x") as target:
            json.dump(value, target, indent=2)
            target.write("\n")
    print(("PASS" if passed == len(results) else "FAIL") + " {}/{} shared provider budget cases; real provider/database calls: 0".format(passed, len(results)))
    print("Run:", output)
    if passed != len(results):
        for result in results:
            if not result["passed"]:
                print("Failed:", result)
        return 1
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print("provider-budget:", error)
        raise SystemExit(2)
