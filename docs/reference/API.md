# API reference

The modular architecture release targets an interactive [Scalar reference](http://localhost:4004/docs) at `/docs` and a machine-readable [OpenAPI contract](http://localhost:4004/openapi.json) at `/openapi.json`. Verify these routes against the integrated API; use its base URL in other environments.

## Authentication

Protected browser requests send `Authorization: Bearer <Firebase ID token>`. Teacher providers and student providers follow configured policy. Resource ownership and enrollment are checked independently of sign-in method. Background workers use the private Redis broker; there are no public task-delivery endpoints.

## Routes

| Method and path | Purpose |
| --- | --- |
| `GET /` | Cached dependency health and uptime |
| `GET /docs`, `GET /openapi.json` | Human and machine-readable API contracts |
| `POST /users/student-session/validate` | Validate an active six-digit class code before anonymous sign-in |
| `POST /users/student-session` | Join class and establish the student session |
| `POST /users/student-mission` | Select and pin a published mission |
| `POST /users/student-presence` | Update presence |
| `GET /users/student-learning?mission_id=...` | Restore student-safe learning state |
| `POST /users/student-learning` | Save checkpoint/evidence and request finalization |
| `POST /users/student-tutor-turn` | Generate or replay a grounded tutor turn |
| `GET /dashboard/overview?class_id=...` | Authorized classroom insights |
| `GET /teacher-classes`, `POST /teacher-classes` | List or create/update teacher classes |
| `GET /teacher-classes/{class_id}` | Read an owned class |
| `GET /teacher-classes/{class_id}/missions` | List class missions |
| `POST /teacher-classes/{class_id}/missions/preview` | Generate or revise a draft |
| `POST /teacher-classes/{class_id}/missions/preview/stream` | Stream preview progress and result |
| `POST /teacher-classes/{class_id}/missions/{mission_id}/approve` | Publish the expected draft revision |
| `DELETE /teacher-classes/{class_id}/missions/{mission_id}` | Soft-delete a mission |

Requests and responses retain `snake_case` fields. Approval requires `expected_draft_revision`. Changed reviewed content advances the draft and published revision; unchanged current content reuses its publication. An identical retry of the latest accepted request replays while that publication is current. Different stale edits return `409`. Use OpenAPI for payload schemas and response details.

Class and mission list endpoints accept `limit` (default `50`, maximum `100`) and an opaque `cursor`. Responses return `next_cursor` when another page is available. Treat cursors as opaque and retain the same class/teacher scope when following a page; pagination does not change authorization.

## Preview streaming

The preview stream uses Server-Sent Events with `progress`, `done`, and `error` events. Before headers are committed, failures can use an HTTP error response. After HTTP `200`, inspect stream events: transport status alone does not establish successful generation. Cancellation must stop the request and release its reader.

Tutor turns use ordinary JSON responses. Keep student-facing teaching fields separate from teacher-only observations and evidence.

## Limits and errors

The server bounds request bodies and attachment downloads, applies authentication and operation-specific quotas, and returns structured errors. A class code is an enrollment mechanism, not teacher authorization. Health confirms dependencies, not a complete generation or learning journey.

See [configuration](CONFIGURATION.md), [data ownership](DATA-MODEL.md), and [validation](../development/VALIDATION.md).

## Teacher history pages

Use `GET /dashboard/overview/page?class_id=...&limit=25&cursor=...`. The response keeps existing overview fields and adds `page`: a class-scoped `next_cursor`, `read_at`, and explicit `metrics_scope=student_page` / `history_scope=latest_attempt_summary`. A page describes its visible students and their latest attempt. It is not a whole-class total.

Use `GET /dashboard/students/{student_id}/attempts?class_id=...&limit=5&cursor=...` for full history. Ownership is checked before reading. The default is 5 attempts, maximum 10, newest activity first with document ID as a tie break. Cursors are scoped to the route, class, and student.

The old `/dashboard/overview` route preserves full-result meaning for up to 100 student rows and 100 attempts. Larger classes get `409 pagination_required` and must use pages. There is no silent history truncation. See [query limits and rollout](../operations/QUERY-SCALE.md).
