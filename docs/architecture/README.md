# Architecture

Current system responsibilities, module boundaries, and AI trust boundaries. [Plans](../plans/README.md) describe intended work separately.

| Guide | Purpose |
| --- | --- |
| [System](SYSTEM.md) | Learning loop, capability ownership, and invariants |
| [Modules](MODULES.md) | Dependency direction and useful package anatomy |
| [Rate limiting](RATE-LIMITING.md) | Core policy and its Redis/HTTP adapters |
| [Background jobs](TASK-PORTABILITY.md) | Queue port, delivery, and backend limits |
| [AI quality and evidence](AI-QUALITY.md) | Tutor context, retrieval limits, and private information boundaries |

The [current diagram](../../assets/diagrams/architecture/architecture.png) follows the code and optional services. Its [Graphviz source](../../assets/diagrams/architecture/architecture_graphviz.py) also writes the dashboard copy. It does not claim a live deployment; see the [hosting record](../operations/HOSTING-HISTORY.md).
