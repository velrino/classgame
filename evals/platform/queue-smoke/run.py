#!/usr/bin/env python3
"""Opt-in transport smoke: isolated local Redis and synthetic Go handlers only."""
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[3]

def main():
    redis = shutil.which("redis-server")
    if not redis:
        raise SystemExit("redis-server is required for the opt-in queue smoke")
    with tempfile.TemporaryDirectory(prefix="classgame-queue-smoke-") as directory:
        with socket.socket() as reserved:
            reserved.bind(("127.0.0.1", 0))
            port = reserved.getsockname()[1]
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
                print(json.dumps(report, indent=2, sort_keys=True))
            finally:
                broker.terminate()
                try:
                    broker.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    broker.kill()
                    broker.wait(timeout=5)

if __name__ == "__main__":
    main()
