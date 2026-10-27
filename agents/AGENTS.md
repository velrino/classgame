# Agent execution contract

This directory defines contribution and isolation rules. It currently contains no deployed bot, scheduler, VM service, or model runner. Mia's application workflows live in the [system architecture](../docs/architecture/system.md).

Read the [root rules](../AGENTS.md), the relevant package guide, and [workspace guidance](../docs/ai/agent-workspaces.md).

- Give each parallel task a separate Git worktree and branch, explicit owned paths, and a handoff target.
- A worktree prevents checkout collisions; it does not isolate processes, credentials, the network, or the filesystem.
- Do not alter another task's files, environment, ports, or running processes. Record processes you start; stop only those you own.
- Return actual changes, checks, limitations, commit SHA, and branch. The coordinator integrates the authorized result through the [release workflow](../docs/processes/release-workflow.md).

## If standalone automation is added

Create only pieces the implementation uses:

```text
agents/
  shared/             # Host helpers with multiple justified consumers
  <agent>/
    setup.sh          # Reviewed installation/update entry point
    host/             # Trusted orchestration and authorized external writes
    sandbox/          # Model execution and restricted workspace
```

- Agents do not import another agent's private implementation. Share code only when its meaning and owner are clear.
- Confine write credentials to the trusted host. Give a model sandbox only the read access/context it needs; never mount privileged token files into it.
- The host validates scope, patch, checks, and authorization before applying changes or writing externally. Reuse session authorization; this adds no approval requirement for already authorized work.
- Document an implemented agent under `docs/ai/` and index it. Do not describe this conditional layout as existing deployment infrastructure.
