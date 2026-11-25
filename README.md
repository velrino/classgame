# ClassGame

**The optional homework that takes notes for the teacher.**

A teacher describes today's lesson. Mia creates a short review mission; students play in a 3D classroom, receive adaptive help, and leave observable learning evidence for the teacher.

[Watch the demo](https://www.youtube.com/watch?v=mvsbSKlGfyE) · [Run locally](docs/development/LOCAL-SETUP.md) · [Documentation](docs/README.md) · [API reference](docs/reference/API.md)

The previous hosted deployment is offline. See [hosting history](docs/operations/HOSTING-HISTORY.md) for its dated shutdown record.

## Architecture

One Next.js application, one Go API, and Firebase-backed learning state form a modular monolith. Optional Go workers use a dedicated Redis/Asynq broker; interactive tutoring stays synchronous.

### System design

The system design is documented in the guides below. They describe the implemented architecture; the diagram does not represent a currently hosted deployment.

| Guide | Covers |
| --- | --- |
| [System architecture](docs/architecture/SYSTEM.md) | Components, capability ownership, learning loop, and system invariants |
| [Module boundaries](docs/architecture/MODULES.md) | Module structure, dependency direction, and public contracts |
| [Data model](docs/reference/DATA-MODEL.md) | Storage ownership, immutable revisions, and student projections |
| [API contracts](docs/reference/API.md) | Authentication, routes, pagination, streaming, and errors |
| [Background jobs](docs/architecture/TASK-PORTABILITY.md) | Queue boundaries, delivery behavior, and recovery constraints |
| [AI quality and evidence](docs/architecture/AI-QUALITY.md) | Tutor context, retrieval, fallbacks, and private information boundaries |

The diagram below shows the components and their connections. Continue with [architecture decisions and trade-offs](#architecture-decisions-and-trade-offs) for the choices behind the design, and [mission flows](#mission-flows) for six sequence diagrams.

<p align="center">
  <img src="assets/diagrams/architecture/architecture.png" alt="ClassGame current modular architecture" width="100%" />
</p>

The Excalidraw version below shows the same architecture as editable boxes and connections. Download the [editable source](assets/diagrams/architecture/excalidraw/architecture.excalidraw) and open it in [Excalidraw](https://excalidraw.com/).

<p align="center">
  <a href="assets/diagrams/architecture/excalidraw/architecture.png"><img src="assets/diagrams/architecture/excalidraw/architecture.png" alt="ClassGame architecture drawn in Excalidraw" width="100%" /></a>
</p>

## Architecture decisions and trade-offs

These choices fit the current code. The alternatives below are comparison options, not a record of benchmark wins or a plan to rewrite the stack. Each row compares at most three options, including the current choice.

| Decision and current choice | Relevant alternatives | Why it fits ClassGame | Trade-off / when to reconsider |
| --- | --- | --- | --- |
| **API: Go** | Node.js/TypeScript; Python | One typed module shares domain policies between HTTP handlers and workers. Request cancellation and bounded model calls are explicit. The [API container](api/infra/cloud-run/Dockerfile) runs one static binary. | Go and frontend TypeScript contracts still need to stay aligned. Node.js would share more language/tooling with the web; Python would fit Python-native ML libraries. No comparative latency or memory benchmark is claimed; provider response time still needs measurement. |
| **Evals: Python with small Go/Node bridges** | TypeScript runner; Go runner | Python's standard library handles JSON cases, selection, assertions and reports. Bridges call actual production Go/TypeScript policies instead of rewriting business rules in Python. See [evaluation scope](docs/ai/EVALUATIONS.md). | This adds a third language/toolchain. A Go or TypeScript runner could reduce one part of the setup, but still needs access to the other language's policies. Default synthetic checks verify contracts and fallbacks, not live teaching quality; local integration and human/model assessment are separate. |
| **Web: Next.js + React** | [React + Vite](https://vite.dev/guide/) | One app provides teacher routes, shared layouts and React UI around the browser-based Three.js classroom. The [standalone build](web/next.config.ts) can run in a Node container rather than requiring one hosting provider. | The current deployment includes a Node server and needs clear server/client boundaries; Three.js still runs in the browser. A Vite SPA could simplify static hosting, with a routing/layout setup and migration cost. Next.js is useful structure here, not a requirement for 3D or proof of better game performance. |
| **Durable data: Firestore, alongside Firebase Auth** | PostgreSQL; MongoDB | Document state and transactions fit mission revisions and learning attempts. Selected projections use realtime browser reads; Firebase Auth handles identity separately. Browser Firestore writes are denied; mutations use the API. See [data model](docs/reference/DATA-MODEL.md) and [access rules](firestore.rules). | Firestore needs planned indexes and denormalized projections, and keeps a Google-specific data/access model. PostgreSQL fits relational constraints and join-heavy reporting; MongoDB keeps a document model with different query/hosting choices. Either migration must replace the current realtime/read-rule integration and preserve transactions and access checks; it is not just a new connection URL. |
| **Background delivery: Redis + Asynq** | Google Cloud Tasks | A dedicated Redis broker and Go workers run locally or on another cloud without Cloud Tasks delivery endpoints or Google OIDC setup. Module-owned messages keep job meaning outside the transport. See [task portability](docs/architecture/TASK-PORTABILITY.md). | Cloud Tasks manages the queue service; Redis moves broker persistence, capacity, backups and worker operation to us. Delivery can repeat: stable IDs and domain checks remain necessary. The current adapter needs standalone-compatible Redis, not an assumed Redis Cluster deployment. Prefer managed delivery if lower operating effort matters more than queue portability. |
| **Application shape: modular monolith with API/worker roles** | Separate microservices | One Go module reuses the same policies and processors, with narrow module contracts. API and worker processes can have separate roles without splitting business logic into network services. See [system boundaries](docs/architecture/SYSTEM.md). | Releases must keep producers, workers and stored contracts compatible. Separate services become useful when independent ownership, isolation or scaling needs justify extra network calls, deployments and recovery work. A small project does not need that cost by default. |

Two boundaries matter when discussing portability:

- **Redis is not the durable learning database.** `TASKS_REDIS_URL` is a dedicated job broker; `REDIS_URL` handles quotas and shared AI admission. Firestore keeps accepted learning state and recovery markers. Broker loss still needs recovery; retries do not guarantee exactly-once external effects. [Queue operations](docs/operations/QUEUES.md).
- **Moving compute is different from replacing services.** API/web containers and a compatible Redis broker can move to another host, but Firebase Auth, Firestore, object storage and Google AI remain dependencies. Mia's interactive tutor calls stay synchronous; file copies and selected learning jobs run in the background. [Architecture](docs/architecture/SYSTEM.md).

Provider references: [Go concurrency](https://go.dev/doc/faq#goroutines), [Node.js I/O and CPU work](https://nodejs.org/en/learn/asynchronous-work/dont-block-the-event-loop), [Next.js static-export limits](https://nextjs.org/docs/app/guides/static-exports#unsupported-features), [Firestore transactions](https://firebase.google.com/docs/firestore/manage-data/transactions), [PostgreSQL constraints](https://www.postgresql.org/docs/current/ddl-constraints.html), [MongoDB data modeling](https://www.mongodb.com/docs/manual/data-modeling/), and [Cloud Tasks delivery](https://docs.cloud.google.com/tasks/docs/dual-overview).

## Mission flows

Read the steps in order. **Starts with** shows who begins each flow. Select a diagram to open the full PNG. See the [step index](assets/diagrams/steps/README.md) for all diagrams and the style guide.

| Step | Starts with | Flow | Diagram |
|:---:|---|---|:---:|
| 01 | **Teacher** | **Upload material and prepare a mission draft**<br>Upload optional files, generate questions, and save a draft for teacher review.<br>[Guide](assets/diagrams/steps/01-mission-draft/README.md) · [Mermaid source](assets/diagrams/steps/01-mission-draft/SEQUENCE.txt) | <a href="assets/diagrams/steps/01-mission-draft/diagram.png"><img src="assets/diagrams/steps/01-mission-draft/diagram.png" alt="Sequence diagram for uploading material and preparing a mission draft" width="480" /></a> |
| 02 | **Teacher** | **Review and publish the mission**<br>Check the reviewed revision, prepare file-copy jobs, and publish through a DB transaction.<br>[Guide](assets/diagrams/steps/02-mission-publication/README.md) · [Mermaid source](assets/diagrams/steps/02-mission-publication/SEQUENCE.txt) | <a href="assets/diagrams/steps/02-mission-publication/diagram.png"><img src="assets/diagrams/steps/02-mission-publication/diagram.png" alt="Sequence diagram for reviewing and publishing a mission" width="480" /></a> |
| 03 | **Redis** | **Copy mission files to private storage**<br>Workers load the published revision, copy files with retry checks, and save each result.<br>[Guide](assets/diagrams/steps/03-attachment-transfer/README.md) · [Mermaid source](assets/diagrams/steps/03-attachment-transfer/SEQUENCE.txt) | <a href="assets/diagrams/steps/03-attachment-transfer/diagram.png"><img src="assets/diagrams/steps/03-attachment-transfer/diagram.png" alt="Sequence diagram for copying mission attachments to private storage" width="480" /></a> |
| 04 | **Student** | **Answer a quiz question with Mia**<br>Save the answer locally, get grounded feedback with bounded context, and save progress.<br>[Guide](assets/diagrams/steps/04-quiz-play/README.md) · [Mermaid source](assets/diagrams/steps/04-quiz-play/SEQUENCE.txt) | <a href="assets/diagrams/steps/04-quiz-play/diagram.png"><img src="assets/diagrams/steps/04-quiz-play/diagram.png" alt="Sequence diagram for a quiz turn, Mia feedback, and checkpoint saving" width="480" /></a> |
| 05 | **Student** | **Finish the quiz and refresh student memory**<br>Confirm completion, finalize private evidence, and replace the compact student memory.<br>[Guide](assets/diagrams/steps/05-learning-memory/README.md) · [Mermaid source](assets/diagrams/steps/05-learning-memory/SEQUENCE.txt) | <a href="assets/diagrams/steps/05-learning-memory/diagram.png"><img src="assets/diagrams/steps/05-learning-memory/diagram.png" alt="Sequence diagram for quiz completion, finalization, and compact student memory" width="480" /></a> |
| 06 | **Redis** | **Refresh class memory and teacher insights**<br>Build a bounded class summary, reject stale work, and refresh the teacher's visible insights.<br>[Guide](assets/diagrams/steps/06-class-insights/README.md) · [Mermaid source](assets/diagrams/steps/06-class-insights/SEQUENCE.txt) | <a href="assets/diagrams/steps/06-class-insights/diagram.png"><img src="assets/diagrams/steps/06-class-insights/diagram.png" alt="Sequence diagram for class synthesis, versioned persistence, and teacher insight refresh" width="480" /></a> |

## Product tour

Every screen below is from the working product. Select any image to open it at full resolution.

| Student experience | Teacher experience |
|:---:|:---:|
| **Join a class in seconds**<br><a href="assets/pitch/01-0.png"><img src="assets/pitch/01-0.png" alt="Student joining a ClassGame classroom" width="100%" /></a> | **Student and teacher experiences, side by side**<br><a href="assets/pitch/01-1.png"><img src="assets/pitch/01-1.png" alt="ClassGame student entry and teacher class dashboard" width="100%" /></a> |
| **Explore the 3D classroom**<br><a href="assets/pitch/01-2.png"><img src="assets/pitch/01-2.png" alt="Student exploring the ClassGame 3D classroom" width="100%" /></a> | **Create a mission with Mia**<br><a href="assets/pitch/02-1.png"><img src="assets/pitch/02-1.png" alt="Teacher creating a new mission with Mia" width="100%" /></a> |
| **Ground missions in real worksheets**<br><a href="assets/pitch/02-2.png"><img src="assets/pitch/02-2.png" alt="Teacher attaching worksheets to a ClassGame mission" width="100%" /></a> | **Review and refine AI-generated questions**<br><a href="assets/pitch/02-3.png"><img src="assets/pitch/02-3.png" alt="Teacher reviewing questions generated by Mia" width="100%" /></a> |
| **Inspect the mission flow**<br><a href="assets/pitch/02-4.png"><img src="assets/pitch/02-4.png" alt="Teacher inspecting the generated mission flow" width="100%" /></a> | **Publish the mission to the class**<br><a href="assets/pitch/03-1.png"><img src="assets/pitch/03-1.png" alt="Published mission in the teacher dashboard" width="100%" /></a> |
| **Choose a mission from the classroom**<br><a href="assets/pitch/03-2.png"><img src="assets/pitch/03-2.png" alt="Student choosing a published mission" width="100%" /></a> | **Follow live, grounded tutor turns**<br><a href="assets/pitch/03-3.png"><img src="assets/pitch/03-3.png" alt="Teacher following a live student and Mia tutoring conversation" width="100%" /></a> |
| **See the evidence-backed learning flow**<br><a href="assets/pitch/03-4.png"><img src="assets/pitch/03-4.png" alt="Evidence-backed student learning flow" width="100%" /></a> | **Turn the completed mission into insight**<br><a href="assets/pitch/03-5.png"><img src="assets/pitch/03-5.png" alt="Completed student mission and teacher learning report" width="100%" /></a> |
| **Understand the whole class at a glance**<br><a href="assets/pitch/04-1.png"><img src="assets/pitch/04-1.png" alt="Teacher overview of live class progress and learning needs" width="100%" /></a> | **Inspect the production architecture in-app**<br><a href="assets/pitch/05-1.png"><img src="assets/pitch/05-1.png" alt="ClassGame production architecture inside the teacher dashboard" width="100%" /></a> |

---

## Reproducible setup and testing

Use the canonical [local setup](docs/development/LOCAL-SETUP.md), [configuration](docs/reference/CONFIGURATION.md), and [validation](docs/development/VALIDATION.md) guides.

Clone [velrino/classgame](https://github.com/velrino/classgame), configure Firebase and Redis, then start `api/` and `web/` as described in [local setup](docs/development/LOCAL-SETUP.md). Open [teacher login](http://localhost:3000/login) or the [student game](http://localhost:3000/home).

See [evaluations](evals/README.md) for synthetic policy checks and optional local integration suites. The default Python runner sends JSON through standard input to a Go bridge built from `api/cmd/evalbridge`. The bridge calls the API's production functions and returns JSON through standard output for Python to check and report. Frontend policy suites use Node bridges. Default checks run offline without a running API server or paid AI calls.

## Documentation

Start with [docs/README.md](docs/README.md): architecture, development, operations, API/configuration reference, and implementation plans each have a canonical home.

Coding assistance starts at the [root router](AGENTS.md) and scoped package guides. Use the [AI index](docs/ai/README.md) for workspace/evaluation guidance and [processes](docs/processes/README.md) for review and release integration.

Repository skills live in [.agents/skills](.agents/skills); contribution and workspace contracts live in [agents/AGENTS.md](agents/AGENTS.md).

## Deployment

See [Cloud Run operations](docs/operations/CLOUD-RUN.md). Local execution does not recreate deleted hosted resources.

## Monthly costs

Each column estimates one month of ClassGame usage in USD for the stated number of active students and teachers combined. Both totals include the Go API, live tutoring and Firebase. The frontend host is the only change. These are planning estimates for text-only missions. The previous deployment is offline.

| Estimated monthly cost (USD) | 1 user | 100 users | 1,000 users | 10,000 users | 100,000 users |
| --- | ---: | ---: | ---: | ---: | ---: |
| Vertex AI · Gemini 3.7 Flash | $0.05 | $4.50 | $45.00 | $450.00 | $4,500.00 |
| Cloud Run · Go API compute | $0.09 | $2.34 | $16.85 | $67.39 | $134.78 |
| Go API · outbound traffic | $0.00 | $0.01 | $0.12 | $1.17 | $11.72 |
| Firestore · reads and writes | $0.00 | $0.00 | $0.00 | $0.81 | $17.01 |
| Firestore · storage and outbound traffic | $0.00 | $0.00 | $0.00 | $0.00 | $3.04 |
| Redis · API quotas and shared AI limits | $0.00 | $0.08 | $0.80 | $8.00 | $80.00 |
| Firebase Auth with Identity Platform | $0.00 | $0.00 | $0.00 | $0.00 | $275.00 |
| Uploadcare and Cloud Storage · no attachments | $0.00 | $0.00 | $0.00 | $0.00 | $0.00 |
| Asynq worker and dedicated Redis broker · disabled | $0.00 | $0.00 | $0.00 | $0.00 | $0.00 |
| Frontend option 1 · Cloud Run compute, requests and traffic | $0.00 | $0.10 | $1.01 | $10.14 | $101.35 |
| Frontend option 2 · Netlify Free or Pro | $0.00 | $0.00 | $0.00 | $20.00 | $120.00 |
| **Total · Cloud Run frontend** | **$0.14** | **$7.03** | **$63.78** | **$537.51** | **$5,122.90** |
| **Total · Netlify frontend** | **$0.14** | **$6.93** | **$62.77** | **$547.37** | **$5,141.55** |

**What drives the estimate**

- **AI usage.** Allow 10 paid calls per active user across tutoring and mission generation. Budget 1,500 uncached input tokens and 900 output tokens per call, including reasoning. The global Gemini 3.7 Flash rates through December 31, 2026 are $0.75 and $3.75 per million tokens after promotional credits. At the published January 2027 rates, the AI row doubles. [Vertex AI pricing](https://cloud.google.com/model-garden/pricing)
- **Cloud Run and traffic.** Both services use 1 vCPU and 4 GiB in South Carolina. For the instance-billed API, assume 1, 25, 180, 720 and 1,440 total instance-hours across the five columns, including idle time. That costs $0.0936 per instance-hour. The request-billed frontend uses two active instance-seconds and 20 requests per user. Budget 8 MiB per user for the web app and classroom assets, plus 1 MiB for API responses, at $0.12/GiB to North America. These rows assume the billing account has used its Cloud Run and network free allowances. [API configuration](api/infra/cloud-run/production/cloudbuild.yaml) · [Web configuration](web/infra/cloud-run/production/cloudbuild.yaml) · [Cloud Run pricing](https://cloud.google.com/run/pricing) · [Network pricing](https://cloud.google.com/vpc/network-pricing)
- **Database and sign-in.** Per user, budget 300 Firestore reads including listeners, 100 writes, 0.1 MiB stored and 0.25 MiB sent to browsers. Assume even traffic over 30 days and the default database's free quota. Budget overages at $0.03 per 100,000 reads, $0.09 per 100,000 writes, $0.15/GiB-month stored and $0.12/GiB sent. Sign-in assumes one Tier 1 identity per user, anonymous cleanup off and the first 50,000 monthly users free. [Firestore pricing](https://cloud.google.com/firestore/pricing) · [Identity Platform pricing](https://cloud.google.com/identity-platform/pricing)
- **Redis and optional jobs.** Budget 400 Redis commands per user at $0.20 per 100,000 commands on Upstash Pay as You Go, with storage and transfer within its included amounts. The API runs with jobs disabled. Enabling attachment copying adds a continuous worker, a separate Redis broker, file storage and transfer. [Redis pricing](https://upstash.com/pricing/redis) · [ClassGame queue requirements](docs/operations/QUEUES.md)
- **Netlify frontend.** The alternative hosts the Next.js app from `web/` and calls the same Go API. Budget one production deploy per month, 20 web requests, 8 MiB delivered and 0.25 GB-seconds of function compute per user. At 100,000 users this uses about 17,262 credits. The estimate uses Free through 1,000 users, then the $20 Pro plan with 3,000 credits and $10 per additional pack of 1,500. [Next.js on Netlify](https://docs.netlify.com/build/frameworks/framework-setup-guides/nextjs/overview/) · [Netlify pricing](https://www.netlify.com/pricing/)

Totals sum the displayed resource costs to the cent. Build and image storage, secrets, logging and optional analytics are assumed to stay within their included allowances. Taxes, backups and paid support are outside the estimate.

## Repository layout

```text
.agents/skills/          Repository skills for reviews, evals, and background jobs
.github/workflows/      CI checks for the API, web app, and policy evaluations
agents/                 AI contribution and workspace contracts
api/                    Go API, worker roles, provider integrations, and eval bridges
web/                    Next.js teacher dashboard and student 3D game
evals/                  Python runners, synthetic datasets, and local evaluation tools
docs/                   Canonical project documentation
assets/                 Source media, credits, diagrams, and generation notes
scripts/                Operational and asset commands
AGENTS.md               Contributor router and repository rules
firebase*.json          Firebase configuration and local emulator settings
firestore.rules         Browser access rules
firestore.indexes.json  Firestore query indexes
```

## Credits

[Classroom model](https://sketchfab.com/3d-models/low-poly-classroom-8b1bf103c90349d981c9c2ae1c0596e5) by Mumladze28; [avatar credits and generation notes](assets/3d/avatars/chibi-band/README.md); [worksheet sources](assets/grade/README.md). Asset licenses stay beside their files. The dashboard uses shadcn/ui and a shell adapted from shadcn-admin.
