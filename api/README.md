# ClassGame API

Go API for classroom missions, tutoring, and learning evidence.

- [Local setup](../docs/development/LOCAL-SETUP.md)
- [API contracts](../docs/reference/API.md) · [Scalar UI](http://localhost:4004/docs) · [OpenAPI](http://localhost:4004/openapi.json)
- [Configuration](../docs/reference/CONFIGURATION.md)
- [Module boundaries](../docs/architecture/MODULES.md)
- [Validation](../docs/development/VALIDATION.md)

## Development

After configuring `.env` through the setup guide:

```bash
go mod download
go run ./cmd/api
```

Health: [localhost:4004](http://localhost:4004/). Run commands from this directory; run Cloud Build commands from the repository root.

## Reference

<a id="runtime"></a>
<a id="users"></a>
<a id="teacher-classes-and-missions"></a>

Routes and payloads live in the [API reference](../docs/reference/API.md); storage ownership in the [data model](../docs/reference/DATA-MODEL.md).

<a id="genai"></a>
<a id="secrets"></a>
<a id="analytics-and-error-tracking"></a>

Provider setup, credentials, and telemetry are covered by [configuration](../docs/reference/CONFIGURATION.md). Keep runtime credentials outside Git.

<a id="cloud-tasks-workers"></a>
<a id="deploy-to-google-cloud-run"></a>
<a id="github-deploy-trigger"></a>

Workers, deployment, and hosted status moved to [operations](../docs/operations/CLOUD-RUN.md) and [hosting history](../docs/operations/HOSTING-HISTORY.md).
