#!/usr/bin/env node
// Explicit local browser evaluation server. It serves only two production modules and the harness.
import { createServer } from "node:http";
import { readFileSync, mkdirSync, writeFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";
import { stripTypeScriptTypes } from "node:module";
import { createHash, randomBytes } from "node:crypto";
import { execFileSync } from "node:child_process";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "../../..");
const { values, positionals } = parseArgs({ options: { run: { type: "boolean", default: false }, port: { type: "string", default: "3102" } }, strict: true, allowPositionals: false });
const port = Number(values.port);
if (!values.run || positionals.length || !Number.isInteger(port) || port < 1024 || port > 65535 || [3000, 4004, 6379, 8787, 9099].includes(port)) throw new Error("Require --run and an unused local port (default 3102).");
const token = randomBytes(24).toString("hex");
const nonce = randomBytes(16).toString("base64url");
const modules = {
  "/source/outbox.js": resolve(root, "web/src/modules/learning/adapters/tutor-outbox.ts"),
  "/domain/evidence-queue": resolve(root, "web/src/modules/learning/domain/evidence-queue.ts"),
};
const sourceHash = createHash("sha256").update(Object.values(modules).map((p) => readFileSync(p)).join("\n")).digest("hex");
const index = readFileSync(resolve(here, "index.html"), "utf8").replaceAll("__NONCE__", nonce).replace("__CONFIG__", JSON.stringify({ token, sourceHash }));
const revision = execFileSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).trim();
const dirty = Boolean(execFileSync("git", ["status", "--porcelain"], { cwd: root, encoding: "utf8" }).trim());
const json = (response, status, value) => { response.writeHead(status, { "Content-Type": "application/json", "Cache-Control": "no-store" }); response.end(JSON.stringify(value)); };
const server = createServer(async (request, response) => {
  if (!["127.0.0.1:" + port, "localhost:" + port].includes(request.headers.host)) return json(response, 403, { error: "loopback host required" });
  response.setHeader("Content-Security-Policy", `default-src 'self'; script-src 'self' 'nonce-${nonce}'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'`);
  response.setHeader("X-Content-Type-Options", "nosniff");
  const path = new URL(request.url, "http://127.0.0.1:" + port).pathname;
  try {
    if (request.method === "GET" && path === "/") { response.setHeader("Content-Type", "text/html; charset=utf-8"); response.end(index); return; }
    if (request.method === "GET" && modules[path]) { response.setHeader("Content-Type", "text/javascript; charset=utf-8"); response.end(stripTypeScriptTypes(readFileSync(modules[path], "utf8"), { mode: "strip" })); return; }
    if (request.method === "GET" && path === "/scenarios.js") { response.setHeader("Content-Type", "text/javascript; charset=utf-8"); response.end(readFileSync(resolve(here, "scenarios.js"))); return; }
    if (request.method === "POST" && path === "/report") {
      if (request.headers["x-classgame-local-eval"] !== token || request.headers.origin !== "http://127.0.0.1:" + port) return json(response, 403, { error: "local evaluation token required" });
      const chunks = []; let length = 0;
      for await (const chunk of request) { length += chunk.length; if (length > 128 * 1024) return json(response, 413, { error: "report too large" }); chunks.push(chunk); }
      const report = JSON.parse(Buffer.concat(chunks).toString());
      if (!Array.isArray(report.results) || report.results.length < 10 || report.results.length > 30 || report.results.some((r) => typeof r.id !== "string" || typeof r.passed !== "boolean")) return json(response, 400, { error: "invalid report" });
      const runId = new Date().toISOString().replace(/[-:.]/g, "") + "-browser-store-" + randomBytes(4).toString("hex");
      const directory = resolve(root, "evals/output", runId); mkdirSync(directory, { recursive: true, mode: 0o700 });
      const manifest = { schema_version: 1, suite: "browser-store", mode: "real-browser-indexeddb", revision, working_tree_dirty: dirty, production_source_sha256: sourceHash, created_at: new Date().toISOString(), real_provider_calls: 0, shared_database_calls: 0, case_ids: report.results.map((r) => r.id), passed: report.results.filter((r) => r.passed).length, total: report.results.length };
      writeFileSync(resolve(directory, "manifest.json"), JSON.stringify(manifest, null, 2) + "\n", { flag: "wx" });
      writeFileSync(resolve(directory, "report.json"), JSON.stringify(report, null, 2) + "\n", { flag: "wx" });
      return json(response, 200, { run: runId, passed: manifest.passed, total: manifest.total });
    }
    json(response, 404, { error: "not found" });
  } catch (error) { json(response, 500, { error: error.message }); }
});
server.listen(port, "127.0.0.1", () => console.log(JSON.stringify({ url: "http://127.0.0.1:" + port, mode: "real-browser-indexeddb", sourceHash, real_provider_calls: 0 })));
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => server.close(() => process.exit(0)));
