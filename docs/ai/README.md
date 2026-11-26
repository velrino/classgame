# AI

Coding-agent work and application AI quality. Repository assistance and Mia's runtime workflows have separate responsibilities.

| Guide | Purpose |
| --- | --- |
| [Agent workspaces](AGENT-WORKSPACES.md) | Current execution model, ownership, and secure handoff |
| [Agent contract](../../agents/AGENTS.md) | Scoped worktrees and conditions for future standalone automation |
| [Tutor evidence and private context](../architecture/AI-QUALITY.md) | Runtime retrieval, context boundaries, and known limits |
| [Evaluations](EVALUATIONS.md) | Python runner, production Go rules, and offline scope |

The [root router](../../AGENTS.md) and package guides select context. This index does not register agents, configure tools, deploy workers, or grant external write permissions.

## Runtime workflows

| Entry point | LangChainGo workflow | Result |
| --- | --- | --- |
| Mission preview | Mission, plus material summaries when needed | Draft for teacher review |
| Student turn | Tutor, after evidence retrieval | Student-safe reply and a separate teacher observation |
| Attempt finalization | Learning trace | Private evidence and bounded student memory |
| Class refresh | Class synthesis | Summary from finalized class evidence |

The [shared Go chain](../../api/internal/modules/ai/agents/structured/chain.go) binds values to fixed prompt templates. Task agents own their instructions, JSON schemas and output limits. The provider owns model selection, file bytes, budget checks and usage reporting. Each run has a fresh chain and receives the caller's cancellation context. The orchestrator applies workflow-specific validation and fallback rules.

Publication and file transfer use application code. The model has no tool for publishing a mission or writing teacher records. See the [AI system diagram](../../assets/diagrams/architecture/ai-system.png) and [LangChainGo source](https://github.com/tmc/langchaingo).
