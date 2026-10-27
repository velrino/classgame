# Architecture

Current system responsibilities, module boundaries, and AI trust boundaries. [Plans](../plans/README.md) describe intended work separately.

| Guide | Purpose |
| --- | --- |
| [System](SYSTEM.md) | Learning loop, capability ownership, and invariants |
| [Modules](MODULES.md) | Dependency direction and useful package anatomy |
| [Rate limiting](RATE-LIMITING.md) | Core policy and its Redis/HTTP adapters |
| [Background jobs](TASK-PORTABILITY.md) | Queue port, delivery, and backend limits |
| [AI quality and evidence](AI-QUALITY.md) | Tutor context, retrieval limits, and private information boundaries |

The original architecture SVG illustrates the historical submitted deployment. Use the [hosting record](../operations/HOSTING-HISTORY.md) for its status.
