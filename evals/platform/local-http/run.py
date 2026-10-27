#!/usr/bin/env python3
"""Opt-in real API flow against guarded local Firebase emulators only."""
import argparse
import datetime as dt
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import re
import secrets
import statistics
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[3]

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        raise ValueError("Local evaluations do not follow redirects")


def local_url(value):
    parsed = urllib.parse.urlsplit(value)
    if parsed.scheme != "http" or parsed.username or parsed.password or parsed.query or parsed.fragment or parsed.path not in {"", "/"}:
        raise ValueError("require a plain local HTTP origin")
    if not ipaddress.ip_address(parsed.hostname).is_loopback or not parsed.port or parsed.port in {3000, 4004, 6379}:
        raise ValueError("require a numeric loopback origin on a separate evaluation port")
    return value.rstrip("/")


def main():
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    parser.add_argument("--run", action="store_true")
    parser.add_argument("--api", default="http://127.0.0.1:4005")
    parser.add_argument("--auth", default="http://127.0.0.1:9099")
    parser.add_argument("--firestore", default="http://127.0.0.1:8787")
    parser.add_argument("--project", default="demo-classgame")
    parser.add_argument("--login", required=True, type=Path)
    options = parser.parse_args()
    if not options.run or not re.fullmatch(r"demo-[a-z0-9][a-z0-9-]{0,50}", options.project):
        raise ValueError("require --run and a demo- project")
    api, auth, firestore = map(local_url, (options.api, options.auth, options.firestore))
    login = json.loads(options.login.read_text())
    if not isinstance(login, dict) or not re.fullmatch(r"[a-z0-9.+_-]*eval[a-z0-9.+_-]*@example\.test", login.get("email", "")) or len(login.get("password", "")) < 6:
        raise ValueError("require a synthetic eval@example.test login")
    opener = urllib.request.build_opener(NoRedirect())
    requests, cases = [], []

    def call(base, path, method="GET", data=None, token=None, expected=200, raw=False):
        headers = {"Content-Type": "application/json"}
        if token:
            headers["Authorization"] = "Bearer " + token
        body = json.dumps(data).encode() if data is not None else None
        start = time.perf_counter()
        try:
            response = opener.open(urllib.request.Request(base + path, data=body, method=method, headers=headers), timeout=120)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            content, status = response.read(), response.status
            response_headers = dict(response.headers)
        requests.append({"method": method, "path": path.split("?")[0], "status": status, "duration_ms": round((time.perf_counter() - start) * 1000, 2)})
        if status not in ((expected,) if isinstance(expected, int) else expected):
            raise ValueError("{} {} returned {}: {}".format(method, path.split("?")[0], status, content.decode(errors="replace")[:350]))
        return (content.decode() if raw else json.loads(content) if content else {}), response_headers

    def passed(case_id):
        cases.append({"id": case_id, "passed": True})
        print("PASS", case_id, flush=True)

    _, headers = call(api, "/")
    if headers.get("X-Classgame-Emulator-Project", headers.get("X-ClassGame-Emulator-Project")) != options.project:
        raise ValueError("API does not prove its explicit demo emulator mode")
    passed("guarded-emulator-api-v1")
    teacher, _ = call(auth, "/identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=local-eval", "POST", {**login, "returnSecureToken": True})
    teacher_token = teacher["idToken"]
    student, _ = call(auth, "/identitytoolkit.googleapis.com/v1/accounts:signUp?key=local-eval", "POST", {"returnSecureToken": True})
    student_token = student["idToken"]
    passed("real-emulator-token-verification-v1")
    suffix = secrets.token_hex(4)
    classroom, _ = call(api, "/teacher-classes", "POST", {"class_name": "HTTP Eval " + suffix, "subject": "Mathematics", "grade_level": "7th grade"}, teacher_token)
    classroom = classroom["teacher_class"]
    class_id, code = classroom["class_id"], classroom["class_code"]
    prefix = "/teacher-classes/" + class_id + "/missions"
    passed("teacher-class-create-v1")
    preview = {"mission_name": "HTTP Fraction Review", "learning_goal": "Compare simple fractions", "teacher_prompt": "Practice equivalent fractions and explain the denominator.", "audience_type": "all", "attachments": []}
    stream, _ = call(api, prefix + "/preview/stream", "POST", preview, teacher_token, raw=True)
    done = [json.loads(block.split("data: ", 1)[1]) for block in stream.split("\n\n") if block.startswith("event: done\n")]
    if len(done) != 1 or "mission" not in done[0]:
        raise ValueError("preview stream did not return one done mission")
    mission = done[0]["mission"]
    if mission["orchestration"]["generated_by_ai"]:
        raise ValueError("local preview unexpectedly used AI")
    passed("deterministic-preview-sse-v1")
    mission_id, draft_revision = mission["id"], mission["draft_revision"]
    call(api, prefix + "/" + mission_id + "/approve", "POST", {"expected_draft_revision": draft_revision + 1}, teacher_token, expected=409)
    passed("stale-draft-conflict-v1")
    published, _ = call(api, prefix + "/" + mission_id + "/approve", "POST", {"expected_draft_revision": draft_revision}, teacher_token)
    published = published["mission"]
    passed("teacher-publish-v1")
    call(api, "/teacher-classes/" + class_id, token=student_token, expected=(401,403))
    call(api, "/users/student-session/validate", "POST", {"class_code": code})
    call(api, "/users/student-session", "POST", {"class_code": code, "student_name": "Synthetic HTTP Student"}, student_token)
    started, _ = call(api, "/users/student-mission", "POST", {"class_code": code, "mission_id": mission_id, "student_name": "Synthetic HTTP Student"}, student_token)
    attempt_id = started["user"]["activity"]["current_game"]["attempt_id"]
    revision = started["mission"]["published_revision"]
    scope = {"mission_id": mission_id, "published_revision": revision, "attempt_id": attempt_id}
    passed("anonymous-join-and-select-v1")
    safe_class, _ = call(api, "/users/student-class?class_code=" + code, token=student_token)
    if any(key in json.dumps(safe_class).lower() for key in ['"teacher_id"', '"teacher_email"', '"photo_url"', '"email"']):
        raise ValueError("student class response contains teacher identity")
    passed("student-class-safe-projection-v1")
    questions = started["mission"]["preview_questions"]
    stages = list(dict.fromkeys(q["stage_id"] for q in questions))
    snapshot = {"revision": 1, "progress": [{"stage_id": stage, "status": "complete", "correct": 99, "required": 1, "queue": [], "pool": []} for stage in stages], "preferences": {"explanation_style": "steps", "difficulty_bias": 0}, "notes": [], "decisions": [], "events": [], "checkpoint": {"phase": "question", "stage_id": stages[0], "question_id": questions[0]["id"]}}
    initial, _ = call(api, "/users/student-learning", "POST", {**scope, "status": "completed", "snapshot": snapshot}, student_token)
    if initial["attempt"]["status"] == "completed":
        raise ValueError("invented client counts completed a mission")
    passed("forged-completion-refused-v1")
    def turn(question, event_id, kind="answer", answer=None, feedback=""):
        return {**scope, "event_id": event_id, "type": kind, "stage_id": question["stage_id"], "question_id": question["id"], "prompt": question["prompt"], "student_response": question["expected_answer"] if answer is None else answer, "expected_answer": question["expected_answer"], "outcome": "correct" if kind == "answer" else "support", "explicit_reasoning": answer or "" if kind == "reasoning" else "", "feedback": feedback}
    first = turn(questions[0], "http-wrong-" + suffix, answer="synthetic incorrect answer")
    wrong, _ = call(api, "/users/student-tutor-turn", "POST", first, student_token)
    replay, _ = call(api, "/users/student-tutor-turn", "POST", first, student_token)
    if replay != wrong:
        raise ValueError("exact retry changed its response")
    call(api, "/users/student-tutor-turn", "POST", {**first, "student_response": "changed intent"}, student_token, expected=409)
    passed("incorrect-answer-and-strict-replay-v1")
    call(api, "/users/student-tutor-turn", "POST", turn(questions[0], "http-reasoning-" + suffix, "reasoning", "I compared the denominators."), student_token)
    call(api, "/users/student-tutor-turn", "POST", turn(questions[0], "http-feedback-" + suffix, "feedback", "helpful", "helpful"), student_token)
    passed("reasoning-and-feedback-v1")
    for index, question in enumerate(questions):
        call(api, "/users/student-tutor-turn", "POST", turn(question, "http-correct-" + suffix + "-" + str(index)), student_token)
    snapshot["revision"] = 2
    completed, _ = call(api, "/users/student-learning", "POST", {**scope, "status": "completed", "snapshot": snapshot}, student_token)
    if completed["attempt"]["status"] != "completed":
        raise ValueError("accepted answers did not confirm completion")
    passed("accepted-evidence-completion-v1")
    query = urllib.parse.urlencode(scope)
    resumed, _ = call(api, "/users/student-learning?" + query, token=student_token)
    if resumed["attempt"]["id"] != attempt_id or resumed["attempt"]["status"] != "completed":
        raise ValueError("resume changed its completed attempt")
    call(api, "/users/student-learning?" + urllib.parse.urlencode({**scope, "published_revision": revision + 1}), token=student_token, expected=409)
    passed("resume-and-revision-scope-v1")
    db = "/v1/projects/" + options.project + "/databases/(default)/documents/"
    call(firestore, db + "teacher_classes/" + class_id, token=teacher_token)
    call(firestore, db + "teacher_classes/" + class_id, token=student_token, expected=403)
    call(firestore, db + "teacher_classes/" + class_id, "PATCH", {"fields": {"class_name": {"stringValue": "forged"}}}, teacher_token, expected=403)
    projection, _ = call(firestore, db + "student_class_missions/" + class_id + "/missions/" + mission_id, token=student_token)
    if any(key in projection.get("fields", {}) for key in ["teacher", "teacher_id", "knowledge_chunks", "teacher_prompt"]):
        raise ValueError("published student projection includes private fields")
    passed("firestore-browser-read-and-write-rules-v1")
    directory = ROOT / "evals/output" / (dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%S.%fZ") + "-local-http-" + suffix)
    directory.mkdir(mode=0o700, parents=True)
    manifest = {"schema_version": 1, "suite": "local-http", "mode": "guarded-firebase-emulators", "revision": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(), "working_tree_dirty": bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).strip()), "suite_version": 1, "case_ids": [case["id"] for case in cases], "api_tree": subprocess.check_output(["git", "rev-parse", "HEAD:api"], cwd=ROOT, text=True).strip(), "runner_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), "real_provider_calls": 0, "shared_database_calls": 0, "project": options.project, "passed": len(cases), "total": len(cases)}
    report = {"results": cases, "request_metrics": requests, "median_request_ms": round(statistics.median(r["duration_ms"] for r in requests), 2), "fixture": {"class_id": class_id, "mission_id": mission_id, "attempt_id": attempt_id, "synthetic_student_uid": student["localId"]}, "cleanup": "Fixtures remain in the owned temporary emulator until its shutdown; no reset or cloud deletion is run."}
    for name, value in [("manifest.json", manifest), ("report.json", report)]:
        with (directory / name).open("x") as target:
            json.dump(value, target, indent=2); target.write("\n")
    print("PASS {}/{} local HTTP cases; real provider calls: 0".format(len(cases), len(cases)))
    print("Run:", directory)

if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, urllib.error.URLError) as error:
        raise SystemExit("local-http: " + str(error))
