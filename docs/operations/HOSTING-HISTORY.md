# Hosting history

On October 17, 2026, the ClassGame `classgame-api` and `classgame-web` Cloud Run services were deleted from project `rare-tape-466414-f4`, region `us-east1`.

ClassGame-owned Artifact Registry repository `classgame`, Cloud Build triggers `classgame-web-main` and `classgame-api-main`, Cloud Tasks queue `mission-attachment-copy`, and its Firebase credentials secret were also removed. Shared resources were preserved. This record does not establish current resource inventory or zero cloud spend.

Former URLs are historical:

- `https://classgame-web-2057331207.us-east1.run.app`
- `https://classgame-api-2057331207.us-east1.run.app`

Earlier submission documents are available in Git history. The [architecture diagram](../../assets/diagrams/architecture/architecture.png) now follows the current modular code and Redis workers. Historical screenshots and links do not establish that a deployment is currently available.

Use [local setup](../development/LOCAL-SETUP.md) to run the applications. A future deployment requires reviewing and provisioning its own [runtime requirements](CLOUD-RUN.md).
