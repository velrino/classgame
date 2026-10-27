#!/usr/bin/env python3
"""Local-only API outage/delay control for the requested real browser evaluation."""
import argparse
import html
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import ipaddress
import json
import re
import secrets
import threading
import time
import urllib.error
import urllib.parse
import urllib.request

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        raise ValueError("Local evaluation proxy does not follow redirects")


def main():
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    parser.add_argument("--run", action="store_true")
    parser.add_argument("--upstream", default="http://127.0.0.1:4006")
    parser.add_argument("--port", type=int, default=4005)
    parser.add_argument("--project", default="demo-classgame")
    options = parser.parse_args()
    parsed = urllib.parse.urlsplit(options.upstream)
    if not options.run or not re.fullmatch(r"demo-[a-z0-9][a-z0-9-]{0,50}", options.project) or parsed.scheme != "http" or parsed.path or parsed.query or parsed.fragment or parsed.username or not ipaddress.ip_address(parsed.hostname).is_loopback or parsed.port in {None, 3000, 4004, 6379} or not 1024 <= options.port <= 65535 or options.port in {3000, 4004, 6379, parsed.port}:
        raise ValueError("Require --run, a demo project, and separate numeric loopback evaluation ports")
    opener = urllib.request.build_opener(NoRedirect())
    with opener.open(options.upstream + "/", timeout=10) as response:
        if response.headers.get("X-ClassGame-Emulator-Project") != options.project:
            raise ValueError("Upstream is not the validated demo emulator API")
    control_token = secrets.token_hex(24)
    state = {"mode": "online", "forwarded": 0, "outages": 0, "delayed": 0}
    lock, capacity = threading.Lock(), threading.BoundedSemaphore(8)
    origin = "http://127.0.0.1:" + str(options.port)
    allowed_origins = {origin, "http://127.0.0.1:3101", "http://localhost:3101"}
    delays = {"delay-learning": "/users/student-learning", "delay-selection": "/users/student-mission", "delay-tutor": "/users/student-tutor-turn"}

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def content(self, status, body, content_type="application/json", extra=None):
            self.send_response(status)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            for key, value in (extra or {}).items():
                self.send_header(key, value)
            self.end_headers()
            self.wfile.write(body)

        def control(self):
            if self.command == "POST":
                if self.headers.get("Origin") != origin:
                    return self.content(403, b'{"error":"local origin required"}')
                body = urllib.parse.parse_qs(self.rfile.read(min(int(self.headers.get("Content-Length", "0")), 4096)).decode())
                if body.get("token") != [control_token] or body.get("mode", [""])[0] not in {"online", "offline", *delays}:
                    return self.content(403, b'{"error":"invalid local control"}')
                with lock:
                    state["mode"] = body["mode"][0]
            with lock:
                current = dict(state)
            buttons = "".join('<button name="mode" value="{}">{}</button>'.format(key, label) for key, label in [("online", "API online"), ("offline", "API unavailable"), ("delay-learning", "Delay learning responses (5 s)"), ("delay-selection", "Delay mission selection (5 s)"), ("delay-tutor", "Delay tutor responses (5 s)")])
            page = '<!doctype html><html lang="en"><meta charset="utf-8"><title>ClassGame local API controls</title><style>body{font:17px system-ui;max-width:900px;margin:4rem auto;background:#101827;color:#eee}button{padding:1rem;margin:.5rem}pre{background:#172033;padding:1rem}</style><h1>ClassGame local API controls</h1><p>This proxy only reaches the validated demo emulator API. No cloud calls or request bodies are logged. Delays last five seconds. The online mode forwards normally.</p><p role="status">Mode: <strong>' + html.escape(current["mode"]) + '</strong></p><form method="post" action="/__local_eval"><input type="hidden" name="token" value="' + control_token + '">' + buttons + '</form><pre>' + html.escape(json.dumps(current, indent=2)) + '</pre></html>'
            self.content(200, page.encode(), "text/html; charset=utf-8", {"Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'"})

        def handle_request(self):
            if self.headers.get("Host") != "127.0.0.1:" + str(options.port):
                return self.content(403, b'{"error":"numeric loopback host required"}')
            if self.path == "/__local_eval":
                return self.control()
            request_origin = self.headers.get("Origin")
            if request_origin and request_origin not in allowed_origins:
                return self.content(403, b'{"error":"local evaluation origin required"}')
            if not self.path.startswith("/") or self.path.startswith("//") or not capacity.acquire(blocking=False):
                return self.content(503, b'{"error":"local evaluation capacity exceeded"}')
            try:
                with lock:
                    mode = state["mode"]
                cors = {"Access-Control-Allow-Origin": request_origin, "Access-Control-Expose-Headers": "Retry-After"} if request_origin else {}
                if mode == "offline" and self.command != "OPTIONS":
                    with lock:
                        state["outages"] += 1
                    return self.content(503, b'{"error":{"code":"local_eval_outage","message":"The local evaluation API is unavailable."}}', extra={**cors, "Retry-After": "5"})
                size = int(self.headers.get("Content-Length", "0"))
                if size < 0 or size > 2 * 1024 * 1024:
                    return self.content(413, b'{"error":"local evaluation body is too large"}')
                data = self.rfile.read(size) if size else None
                headers = {key: value for key, value in self.headers.items() if key.lower() not in {"host", "connection", "content-length", "transfer-encoding"}}
                request = urllib.request.Request(options.upstream + self.path, data=data, method=self.command, headers=headers)
                try:
                    response = opener.open(request, timeout=120)
                except urllib.error.HTTPError as error:
                    response = error
                with response:
                    body = response.read(4 * 1024 * 1024 + 1)
                    if len(body) > 4 * 1024 * 1024:
                        return self.content(502, b'{"error":"local evaluation response is too large"}')
                    response_headers = {key: value for key, value in response.headers.items() if key.lower() not in {"connection", "content-length", "transfer-encoding", "content-type"}}
                    response_status, response_type = response.status, response.headers.get("Content-Type", "application/json")
                with lock:
                    state["forwarded"] += 1
                if mode in delays and self.path.startswith(delays[mode]):
                    with lock:
                        state["delayed"] += 1
                    time.sleep(5)
                self.content(response_status, body, response_type, response_headers)
            except (OSError, ValueError) as error:
                try:
                    self.content(502, b'{"error":{"code":"local_eval_proxy_failed","message":"Local emulator connection failed."}}', extra=cors if "cors" in locals() else None)
                except OSError:
                    pass
            finally:
                capacity.release()

        do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = do_OPTIONS = handle_request

    server = ThreadingHTTPServer(("127.0.0.1", options.port), Handler)
    server.daemon_threads = True
    print(json.dumps({"control_url": origin + "/__local_eval", "mode": "demo-emulator-proxy", "project": options.project}), flush=True)
    try:
        server.serve_forever()
    finally:
        server.server_close()

if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, urllib.error.URLError) as error:
        raise SystemExit("local-proxy: " + str(error))
