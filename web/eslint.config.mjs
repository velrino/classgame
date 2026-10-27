import path from "node:path";
import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const moduleBoundaries = {
  rules: {
    "public-surface": {
      meta: {
        type: "problem",
        schema: [],
        messages: {
          privateImport:
            "Import {{module}} through its public index.ts surface.",
          infrastructureInDomain:
            "Domain rules must not depend on browser, React, or vendor adapters.",
        },
      },
      create(context) {
        const filename = context.filename.replaceAll("\\", "/");
        const sourceModule = filename.match(/\/src\/modules\/([^/]+)\//)?.[1];
        return {
          ImportDeclaration(node) {
            const value = node.source.value;
            if (typeof value !== "string") return;
            const resolved = value.startsWith(".")
              ? path
                  .resolve(path.dirname(filename), value)
                  .replaceAll("\\", "/")
              : value.replace("@/", "/src/");
            const target = resolved.match(
              /\/src\/modules\/([^/]+)(?:\/(.*))?$/,
            );
            if (
              target &&
              target[1] !== sourceModule &&
              target[2] &&
              target[2] !== "index"
            )
              context.report({
                node: node.source,
                messageId: "privateImport",
                data: { module: target[1] },
              });
            if (
              filename.includes("/domain/") &&
              node.importKind !== "type" &&
              !node.specifiers.every(
                (specifier) => specifier.importKind === "type",
              )
            ) {
              if (
                /^(react(?:\/|$)|firebase(?:\/|$)|three(?:\/|$))/.test(value) ||
                /\/adapters\/(?!types\/)/.test(resolved)
              )
                context.report({
                  node: node.source,
                  messageId: "infrastructureInDomain",
                });
            }
          },
        };
      },
    },
  },
};

export default defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    files: ["src/**/*.{ts,tsx}"],
    plugins: { modules: moduleBoundaries },
    rules: { "modules/public-surface": "error" },
  },
  globalIgnores([
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
    "public/vendor/**",
  ]),
]);
