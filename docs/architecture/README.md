# Architecture

Current system responsibilities, module boundaries, and AI trust boundaries. [Plans](../plans/README.md) describe intended work separately.

| Guide | Purpose |
| --- | --- |
| [System](system.md) | Learning loop, capability ownership, and invariants |
| [Modules](modules.md) | Dependency direction and useful package anatomy |
| [Rate limiting](RATE-LIMITING.md) | Core policy and its Redis/HTTP adapters |
| [AI quality and evidence](ai-quality.md) | Tutor context, retrieval limits, and private information boundaries |

The original architecture SVG illustrates the historical submitted deployment. Use the [hosting record](../operations/hosting-history.md) for its status.
