// Only the loopback development server receives the demo login values.
const { readFileSync } = require("node:fs");
const { parseEnv } = require("node:util");
const { resolve } = require("node:path");

const root = resolve(__dirname, "..");
let local = {};
try {
  local = parseEnv(readFileSync(resolve(root, "api/.env"), "utf8"));
} catch (error) {
  if (error.code !== "ENOENT") throw error;
}

process.env.CLASSGAME_LOCAL_LOGIN_EMAIL =
  process.env.DEMO_TEACHER_EMAIL || local.DEMO_TEACHER_EMAIL || "";
process.env.CLASSGAME_LOCAL_LOGIN_PASSWORD =
  process.env.DEMO_TEACHER_PASSWORD || local.DEMO_TEACHER_PASSWORD || "";

process.chdir(resolve(root, "web"));
const cli = require.resolve("next/dist/bin/next", { paths: [process.cwd()] });
process.argv = [process.execPath, cli, "dev", "--hostname", "127.0.0.1"];
require(cli);
