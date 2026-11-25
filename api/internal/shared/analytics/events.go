package analytics

type Event string

const (
	EventClassCreated     Event = "mia_api_class_created"
	EventGameJoined       Event = "mia_api_game_joined"
	EventMissionCompleted Event = "mia_api_mission_completed"
	EventMissionCreated   Event = "mia_api_mission_created"
	EventMissionPublished Event = "mia_api_mission_published"
	EventMissionStarted   Event = "mia_api_mission_started"
)

type Properties map[string]any
