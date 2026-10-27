# Agent execution contract

This directory guides coding agents. It is not a deployed bot or worker service. Mia workflows and background workers belong to the [application architecture](../docs/architecture/SYSTEM.md).

Read the [root rules](../AGENTS.md), nearest package guide, and [workspace guidance](../docs/ai/AGENT-WORKSPACES.md).

## Set up a task

- The coordinator supplies the goal, issue, starting release revision, owned paths, branch/worktree, and expected handoff.
- Give each parallel task that edits files a separate worktree and branch. Read-only audits may use an existing checkout.
- Coordinate shared-file changes before editing. Do not overwrite another task's work or commit its files.
- A worktree does not isolate processes, credentials, ports, the network, or the rest of the filesystem.

## Work and hand off

- Use tools actually available in the session. A document does not create a tool, credential, service, or permission.
- Reuse task authorization. Seek clarification only for missing context or actions outside that scope; continue useful authorized work.
- Keep runtime files private. Record the ports and processes you start, stop only those you own, and preserve the user's running app.
- Run the smallest relevant existing checks. Follow the root test rule and the evaluation guide for authorized evaluation work.
- Return the branch/SHA, changed paths, result, checks, limits, issue/PR links, and processes left running or stopped. Report observations rather than assumed success.
- The coordinator reviews and integrates through the [release workflow](../docs/processes/RELEASE-WORKFLOW.md). Keep durable guidance under `docs/`, not in temporary task assignments.

If standalone automation is later implemented, keep write credentials in its trusted host and give model execution only scoped access. Never mount privileged token files in the model workspace. The host validates scope, patches, checks, and authorization before external writes. Document the real setup and recovery under `docs/ai/`; do not add unused agent folders or describe a planned system as deployed.
