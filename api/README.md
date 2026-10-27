# ClassGame API

Go API for classroom missions, tutoring, and learning evidence.

- [Local setup](../docs/development/local-setup.md)
- [API contracts](../docs/reference/api.md) · [Scalar UI](http://localhost:4004/docs) · [OpenAPI](http://localhost:4004/openapi.json)
- [Configuration](../docs/reference/configuration.md)
- [Module boundaries](../docs/architecture/modules.md)
- [Validation](../docs/development/validation.md)

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

Routes and payloads live in the [API reference](../docs/reference/api.md); storage ownership in the [data model](../docs/reference/data-model.md).

<a id="genai"></a>
<a id="secrets"></a>
<a id="analytics-and-error-tracking"></a>

Provider setup, credentials, and telemetry are covered by [configuration](../docs/reference/configuration.md). Keep runtime credentials outside Git.

<a id="cloud-tasks-workers"></a>
<a id="deploy-to-google-cloud-run"></a>
<a id="github-deploy-trigger"></a>

Workers, deployment, and hosted status moved to [operations](../docs/operations/cloud-run.md) and [hosting history](../docs/operations/hosting-history.md).
