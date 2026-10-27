// Package apidocs serves the same-origin API contract and a pinned Scalar UI.
package apidocs

import (
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/modules/classrooms"
	healthmodule "classgame-api/internal/modules/health"
	"classgame-api/internal/modules/insights"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
)

const referenceHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>ClassGame API</title><link rel="icon" href="data:,"><style>:root{--scalar-font:system-ui,sans-serif!important;--scalar-font-code:monospace!important}</style></head>
<body><div id="classgame-api-reference"></div>
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.76.0/dist/browser/standalone.js" integrity="sha384-oH1C0RwqcwbVXkC+r8A+4qRoyEUQdtxAzOwumkxZtQBBcsy61FOoPt7B50OVJYYQ" crossorigin="anonymous"></script>
<script>Scalar.createApiReference('#classgame-api-reference', {
 sources:[{id:'classgame',url:'/openapi.json',agent:{disabled:true}}], baseServerURL:window.location.origin, servers:[{url:window.location.origin}],
 proxyUrl:'', persistAuth:false, telemetry:false, withDefaultFonts:false,
 hideClientButton:true, showDeveloperTools:'never', agent:{disabled:true},
 customFetch:(input,init)=>{const target=new URL(input instanceof Request?input.url:input,window.location.href);if(target.origin!==window.location.origin){return Promise.reject(new TypeError('API requests must remain on this origin'));}return window.fetch(input,init);},
 authentication:{createAnySecurityScheme:false}
});</script></body></html>`

func RegisterRoutes(server *echo.Echo) {
	document := OpenAPI()
	server.GET("/docs", func(ctx *echo.Context) error {
		ctx.Response().Header().Set("Content-Security-Policy", "default-src 'none'; script-src https://cdn.jsdelivr.net 'unsafe-inline'; connect-src 'self'; style-src 'unsafe-inline'; img-src 'self' data:; font-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		ctx.Response().Header().Set("Referrer-Policy", "no-referrer")
		ctx.Response().Header().Set("X-Content-Type-Options", "nosniff")
		ctx.Response().Header().Set("Cache-Control", "no-store")
		return ctx.HTML(http.StatusOK, referenceHTML)
	})
	server.GET("/openapi.json", func(ctx *echo.Context) error { return ctx.JSON(http.StatusOK, document) })
}

type object = map[string]any
type schemaBuilder struct{ definitions object }

// schema reflects only explicit JSON fields. Firestore-only metadata never
// becomes an accidental documentation contract.
func (builder *schemaBuilder) schema(typ reflect.Type) object {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == reflect.TypeOf(time.Time{}) {
		return object{"type": "string", "format": "date-time"}
	}
	switch typ.Kind() {
	case reflect.Bool:
		return object{"type": "boolean"}
	case reflect.String:
		return object{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint64:
		return object{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return object{"type": "number"}
	case reflect.Slice, reflect.Array:
		return object{"type": "array", "items": builder.schema(typ.Elem())}
	case reflect.Map:
		return object{"type": "object", "additionalProperties": builder.schema(typ.Elem())}
	case reflect.Struct:
		name := strings.ReplaceAll(strings.TrimPrefix(typ.PkgPath(), "classgame-api/internal/"), "/", "_") + "_" + typ.Name()
		ref := object{"$ref": "#/components/schemas/" + name}
		if _, exists := builder.definitions[name]; exists {
			return ref
		}
		definition := object{"type": "object", "properties": object{}}
		builder.definitions[name] = definition
		properties := definition["properties"].(object)
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			if field.PkgPath != "" {
				continue
			}
			if field.Anonymous {
				embedded := field.Type
				for embedded.Kind() == reflect.Pointer {
					embedded = embedded.Elem()
				}
				for i := 0; i < embedded.NumField(); i++ {
					f := embedded.Field(i)
					tag := strings.Split(f.Tag.Get("json"), ",")[0]
					if tag != "" && tag != "-" {
						properties[tag] = builder.schema(f.Type)
					}
				}
				continue
			}
			tag := strings.Split(field.Tag.Get("json"), ",")[0]
			if tag == "" || tag == "-" {
				continue
			}
			properties[tag] = builder.schema(field.Type)
		}
		return ref
	default:
		return object{}
	}
}
func (builder *schemaBuilder) of(value any) object { return builder.schema(reflect.TypeOf(value)) }
func jsonContent(schema object) object             { return object{"application/json": object{"schema": schema}} }
func response(description string, schema object) object {
	return object{"description": description, "content": jsonContent(schema)}
}
func query(name, description string, required bool, schema object) object {
	return object{"name": name, "in": "query", "required": required, "description": description, "schema": schema}
}
func record(properties object) object { return object{"type": "object", "properties": properties} }

// OpenAPI documents public, teacher and student HTTP routes. Runtime credentials and real classroom data are never embedded.
func OpenAPI() object {
	builder := schemaBuilder{definitions: object{}}
	errorSchema := builder.of(students.ErrorResponse{})
	paths := object{}
	add := func(path, method, id, summary, role string, request any, result object) object {
		operation := object{"operationId": id, "summary": summary, "tags": []string{role}, "responses": object{"200": response("Success", result), "400": response("Invalid JSON, parameters or request validation", errorSchema), "429": response("Rate limit exceeded; observe Retry-After", errorSchema), "500": response("Internal error", errorSchema), "503": response("Required dependency unavailable", errorSchema)}}
		if role != "public" && role != "documentation" {
			scheme := "firebaseBearer"
			operation["security"] = []any{object{scheme: []string{}}}
			operation["responses"].(object)["401"] = response("Missing, revoked, invalid or disallowed sign-in token", errorSchema)
			operation["responses"].(object)["403"] = response("Actor does not own the requested classroom or resource", errorSchema)
		}
		if strings.Contains(path, "{") {
			params := []any{}
			for _, part := range strings.Split(path, "/") {
				if strings.HasPrefix(part, "{") {
					params = append(params, object{"name": strings.Trim(part, "{}"), "in": "path", "required": true, "schema": object{"type": "string"}})
				}
			}
			operation["parameters"] = params
		}
		if request != nil {
			operation["requestBody"] = object{"required": true, "content": jsonContent(builder.of(request))}
		}
		if _, exists := paths[path]; !exists {
			paths[path] = object{}
		}
		paths[path].(object)[method] = operation
		return operation
	}
	appendParams := func(op object, params ...any) {
		existing, _ := op["parameters"].([]any)
		op["parameters"] = append(existing, params...)
	}
	integer := object{"type": "integer", "minimum": 1, "maximum": 100}
	pagination := []any{query("limit", "Optional page size (1–100). Omit pagination parameters for the existing response shape.", false, integer), query("cursor", "Opaque cursor returned by next_cursor; scoped to the authenticated actor and query.", false, object{"type": "string"})}
	health := add("/", "get", "getHealth", "Cached Firebase and Redis dependency health", "public", nil, builder.of(healthmodule.HealthCheckResponse{}))
	health["responses"].(object)["503"] = response("One or more dependencies are unavailable", builder.of(healthmodule.HealthCheckResponse{}))
	add("/docs", "get", "getApiReference", "Scalar API reference (HTML)", "documentation", nil, object{"type": "string"})["responses"] = object{"200": object{"description": "Pinned Scalar reference, same-origin requests, no auth persistence or public proxy", "content": object{"text/html": object{"schema": object{"type": "string"}}}}}
	add("/openapi.json", "get", "getOpenAPI", "OpenAPI 3.1 contract", "documentation", nil, object{"type": "object"})
	add("/users/student-session/validate", "post", "validateClassCode", "Validate an active six-digit class code before anonymous sign-in", "public", students.StudentClassCodeRequest{}, builder.of(students.StudentClassCodeResponse{}))["responses"].(object)["404"] = response("Active class code not found", errorSchema)
	add("/users/student-session", "post", "joinStudentClass", "Join a class using an anonymous Firebase session", "students", students.StudentSessionRequest{}, builder.of(students.StudentSessionResponse{}))
	add("/users/student-class", "get", "getStudentClass", "Minimal class projection for the authenticated student's joined class", "students", nil, record(object{"class": builder.of(students.StudentClassView{})}))["responses"].(object)["404"] = response("Student session or active joined class not found", errorSchema)
	add("/users/student-mission", "post", "startStudentMission", "Start or resume a published mission pinned to its immutable revision", "students", students.StudentMissionRequest{}, builder.of(students.StudentSessionResponse{}))
	add("/users/student-presence", "post", "saveStudentPresence", "Record bounded presence metadata; it never awards completion", "students", students.StudentPresenceRequest{}, builder.of(students.StudentSessionResponse{}))
	learning := add("/users/student-learning", "get", "getStudentLearning", "Restore the safe learning checkpoint for the current attempt", "students", nil, builder.of(students.StudentLearningResponse{}))
	appendParams(learning, query("mission_id", "Current joined mission ID", true, object{"type": "string"}))
	save := add("/users/student-learning", "post", "saveStudentLearning", "Save a UI checkpoint; assessment comes from verified tutor turns", "students", students.StudentLearningSnapshotRequest{}, builder.of(students.StudentLearningResponse{}))
	save["description"] = "Client progress, counts and evidence events are not authoritative. The server derives them from persisted answer/reasoning/feedback turns against the pinned question set. Completion is accepted only after the required distinct correct answers per stage; premature completed requests return learning. Await answer-turn requests before sending the final checkpoint. UI checkpoint and preferences remain recoverable presentation state."
	add("/users/student-tutor-turn", "post", "saveStudentTutorTurn", "Submit an idempotent observable tutor interaction", "students", students.StudentTutorTurnRequest{}, builder.of(students.StudentTutorTurnAPIResponse{}))["description"] = "event_id is the retry key within an attempt. The server resolves prompt, stage, expected_answer and outcome from the pinned published question. Client-supplied expected_answer/outcome cannot award credit."
	overview := add("/dashboard/overview", "get", "getDashboardOverview", "Teacher-owned classroom overview and learning evidence", "teachers", nil, builder.of(insights.OverviewResponse{}))
	appendParams(overview, query("class_id", "Optional teacher-owned class ID", false, object{"type": "string"}))
	classes := add("/teacher-classes", "get", "listTeacherClasses", "List the authenticated teacher's classes", "teachers", nil, record(object{"teacher_classes": object{"type": "array", "items": builder.of(classrooms.TeacherClassResponse{})}, "next_cursor": object{"type": "string"}}))
	appendParams(classes, pagination...)
	add("/teacher-classes", "post", "upsertTeacherClass", "Create a class or update a teacher-owned class", "teachers", classrooms.UpsertTeacherClassRequest{}, record(object{"teacher_class": builder.of(classrooms.TeacherClassResponse{})}))
	add("/teacher-classes/{class_id}", "get", "getTeacherClass", "Read one teacher-owned class", "teachers", nil, record(object{"teacher_class": builder.of(classrooms.TeacherClassResponse{})}))
	list := add("/teacher-classes/{class_id}/missions", "get", "listMissions", "List missions in a teacher-owned class", "teachers", nil, record(object{"missions": object{"type": "array", "items": builder.of(missions.MissionResponse{})}, "next_cursor": object{"type": "string"}}))
	appendParams(list, pagination...)
	add("/teacher-classes/{class_id}/missions/preview", "post", "previewMission", "Generate or revise a mission draft with optimistic revision checks", "teachers", missions.CreateMissionRequest{}, record(object{"mission": builder.of(missions.MissionResponse{})}))["responses"].(object)["409"] = response("expected_draft_revision conflicts with the current draft", errorSchema)
	stream := add("/teacher-classes/{class_id}/missions/preview/stream", "post", "streamMissionPreview", "Generate a draft with progress, done and error SSE events", "teachers", missions.CreateMissionRequest{}, object{"type": "string"})
	stream["description"] = "Authenticate and rate-limit before HTTP 200. Content-Type is text/event-stream. Event progress contains stage/message/percent; done contains mission; error contains the JSON error envelope. After stream headers are committed, failures are sent as error events. Respect cancellation and the generation deadline."
	stream["responses"].(object)["200"] = object{"description": "SSE progress followed by done or error", "content": object{"text/event-stream": object{"schema": object{"type": "string"}, "example": "event: progress\ndata: {\"stage\":\"queued\",\"message\":\"Starting mission preview.\",\"percent\":1}\n\nevent: done\ndata: {\"mission\":{\"id\":\"mission-id\"}}\n\n"}}}
	approve := add("/teacher-classes/{class_id}/missions/{mission_id}/approve", "post", "approveMission", "Publish the expected draft revision atomically", "teachers", missions.ApproveMissionRequest{}, record(object{"mission": builder.of(missions.MissionResponse{})}))
	approve["responses"].(object)["409"] = response("Stale expected draft revision", errorSchema)
	approve["responses"].(object)["422"] = response("Education guardrail prevents publication", errorSchema)
	deletion := add("/teacher-classes/{class_id}/missions/{mission_id}", "delete", "deleteMission", "Soft-delete a mission while retaining published history", "teachers", nil, object{})
	delete(deletion["responses"].(object), "200")
	deletion["responses"].(object)["204"] = object{"description": "Mission soft-deleted"}
	attachment := add("/teacher-classes/{class_id}/missions/{mission_id}/attachments/{uploadcare_uuid}", "get", "downloadMissionAttachment", "Download a private worksheet after teacher ownership verification", "teachers", nil, object{})
	appendParams(attachment, query("revision", "Optional immutable published revision; omission selects the current draft view", false, object{"type": "integer", "minimum": 1}))
	attachment["responses"].(object)["200"] = object{"description": "Bounded private bytes, no-store, attachment disposition, nosniff; never a bearer token in a URL", "content": object{"application/octet-stream": object{"schema": object{"type": "string", "format": "binary"}}}}
	attachment["responses"].(object)["404"] = response("Attachment is missing or has not reached private storage", errorSchema)
	// Explicit input requirements supplement the shared JSON DTO shape.
	for _, contract := range []struct {
		value    any
		required []string
	}{
		{students.StudentClassCodeRequest{}, []string{"class_code"}},
		{students.StudentSessionRequest{}, []string{"class_code", "student_name"}},
		{students.StudentMissionRequest{}, []string{"class_code", "mission_id", "student_name"}},
		{students.StudentPresenceRequest{}, []string{"mission_id", "status"}},
		{students.StudentLearningSnapshotRequest{}, []string{"mission_id", "status", "snapshot"}},
		{students.StudentTutorTurnRequest{}, []string{"mission_id", "event_id", "type"}},
		{classrooms.UpsertTeacherClassRequest{}, []string{"subject"}},
		{missions.CreateMissionRequest{}, []string{"learning_goal", "teacher_prompt"}},
		{missions.ApproveMissionRequest{}, []string{"expected_draft_revision"}},
	} {
		ref := builder.of(contract.value)["$ref"].(string)
		definition := builder.definitions[strings.TrimPrefix(ref, "#/components/schemas/")].(object)
		definition["required"] = contract.required
		properties := definition["properties"].(object)
		if field, ok := properties["class_code"].(object); ok {
			field["pattern"] = "^[0-9]{6}$"
		}
		if field, ok := properties["expected_draft_revision"].(object); ok {
			field["minimum"] = 1
		}
	}

	return object{"openapi": "3.1.0", "info": object{"title": "ClassGame API", "version": "1.0.0", "description": "Teacher-owned classroom workflows, anonymous student sessions, pinned missions and evidence-grounded tutoring. Bearer tokens are entered manually and kept in memory."}, "servers": []any{object{"url": "/", "description": "Same origin as this API reference"}}, "paths": paths, "components": object{"schemas": builder.definitions, "securitySchemes": object{"firebaseBearer": object{"type": "http", "scheme": "bearer", "bearerFormat": "Firebase ID token", "description": "Teacher: configured Google/password provider; student: anonymous provider. Identity and resource ownership are verified by the API."}}}}
}
