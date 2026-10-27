import posthog, { type Properties } from "posthog-js";

import { CLASSGAME_API_BASE_URL } from "@/shared/http/client";

const POSTHOG_PROJECT_TOKEN =
  "phc_rSqPFFgY7jqf7XxP7QAXXUhZ76Wt9HtJGKZx4A2EXfz4";
const POSTHOG_API_HOST = "https://us.i.posthog.com";

const MIA_EVENTS = {
  classCreated: "mia_web_class_created",
  classCreationFailed: "mia_web_class_creation_failed",
  gameAccessed: "mia_web_game_accessed",
  gameJoined: "mia_web_game_joined",
  gameJoinFailed: "mia_web_game_join_failed",
  missionCreated: "mia_web_mission_created",
  missionCreationFailed: "mia_web_mission_creation_failed",
  missionPublished: "mia_web_mission_published",
  missionPublishFailed: "mia_web_mission_publish_failed",
  missionStarted: "mia_web_mission_started",
  missionStartFailed: "mia_web_mission_start_failed",
  missionCompleted: "mia_web_mission_completed",
} as const;

type ClassCreatedProperties = {
  class_id: string;
};

type FailureProperties = {
  error_code: string;
  error_type: string;
  http_status?: number;
};

type ClassCreationFailedProperties = FailureProperties;

type GameAccessedProperties = {
  has_prefilled_class_code: boolean;
};

type GameJoinedProperties = {
  class_id: string;
};

type GameJoinFailedProperties = FailureProperties & {
  stage: "authentication" | "class_validation" | "response_validation" | "session_creation";
};

type MissionCreatedProperties = {
  class_id: string;
  mission_id: string;
};

type MissionCreationFailedProperties = FailureProperties & {
  class_id: string;
};

type MissionPublishedProperties = {
  class_id: string;
  mission_id: string;
};

type MissionPublishFailedProperties = FailureProperties & {
  class_id: string;
  mission_id: string;
};

type MissionStartedProperties = {
  attempt_id: string;
  class_id: string;
  mission_id: string;
};

type MissionStartFailedProperties = FailureProperties & {
  class_id?: string;
  mission_id: string;
};

type MissionCompletedProperties = MissionStartedProperties;

type MiaEventProperties = {
  [MIA_EVENTS.classCreated]: ClassCreatedProperties;
  [MIA_EVENTS.classCreationFailed]: ClassCreationFailedProperties;
  [MIA_EVENTS.gameAccessed]: GameAccessedProperties;
  [MIA_EVENTS.gameJoined]: GameJoinedProperties;
  [MIA_EVENTS.gameJoinFailed]: GameJoinFailedProperties;
  [MIA_EVENTS.missionCreated]: MissionCreatedProperties;
  [MIA_EVENTS.missionCreationFailed]: MissionCreationFailedProperties;
  [MIA_EVENTS.missionPublished]: MissionPublishedProperties;
  [MIA_EVENTS.missionPublishFailed]: MissionPublishFailedProperties;
  [MIA_EVENTS.missionStarted]: MissionStartedProperties;
  [MIA_EVENTS.missionStartFailed]: MissionStartFailedProperties;
  [MIA_EVENTS.missionCompleted]: MissionCompletedProperties;
};

type MiaEventName = keyof MiaEventProperties;

export function initializeMiaAnalytics() {
  if (typeof window === "undefined" || posthog.__loaded) {
    return;
  }

  posthog.init(POSTHOG_PROJECT_TOKEN, {
    api_host: POSTHOG_API_HOST,
    defaults: "2026-05-30",
    // Product analytics is intentionally explicit: only the mia_web_* events below
    // are captured, alongside PostHog's reserved $exception error events.
    autocapture: false,
    capture_pageleave: false,
    capture_pageview: false,
    capture_performance: false,
    disable_session_recording: true,
    tracing_headers: [new URL(CLASSGAME_API_BASE_URL).hostname],
    capture_exceptions: {
      capture_console_errors: true,
      capture_unhandled_errors: true,
      capture_unhandled_rejections: true,
    },
  });
}

function captureMiaEvent<EventName extends MiaEventName>(
  eventName: EventName,
  properties: MiaEventProperties[EventName],
) {
  initializeMiaAnalytics();
  posthog.capture(eventName, properties);
}

function failureProperties(error: unknown): FailureProperties {
  if (typeof error !== "object" || error === null) {
    return {
      error_code: "unknown_error",
      error_type: "UnknownError",
    };
  }

  const candidate = error as {
    readonly code?: unknown;
    readonly name?: unknown;
    readonly status?: unknown;
  };
  const properties: FailureProperties = {
    error_code:
      typeof candidate.code === "string" && candidate.code.trim()
        ? candidate.code
        : "unknown_error",
    error_type:
      typeof candidate.name === "string" && candidate.name.trim()
        ? candidate.name
        : "UnknownError",
  };

  if (
    typeof candidate.status === "number" &&
    Number.isInteger(candidate.status) &&
    candidate.status >= 100 &&
    candidate.status <= 599
  ) {
    properties.http_status = candidate.status;
  }

  return properties;
}

export const miaAnalytics = {
  classCreated(properties: ClassCreatedProperties) {
    captureMiaEvent(MIA_EVENTS.classCreated, properties);
  },

  classCreationFailed(error: unknown) {
    captureMiaEvent(MIA_EVENTS.classCreationFailed, failureProperties(error));
  },

  gameAccessed(properties: GameAccessedProperties) {
    captureMiaEvent(MIA_EVENTS.gameAccessed, properties);
  },

  gameJoined(properties: GameJoinedProperties) {
    captureMiaEvent(MIA_EVENTS.gameJoined, properties);
  },

  gameJoinFailed(
    error: unknown,
    properties: Pick<GameJoinFailedProperties, "stage">,
  ) {
    captureMiaEvent(MIA_EVENTS.gameJoinFailed, {
      ...failureProperties(error),
      ...properties,
    });
  },

  missionCreated(properties: MissionCreatedProperties) {
    captureMiaEvent(MIA_EVENTS.missionCreated, properties);
  },

  missionCreationFailed(
    error: unknown,
    properties: Pick<MissionCreationFailedProperties, "class_id">,
  ) {
    captureMiaEvent(MIA_EVENTS.missionCreationFailed, {
      ...failureProperties(error),
      ...properties,
    });
  },

  missionPublished(properties: MissionPublishedProperties) {
    captureMiaEvent(MIA_EVENTS.missionPublished, properties);
  },

  missionPublishFailed(
    error: unknown,
    properties: Pick<MissionPublishFailedProperties, "class_id" | "mission_id">,
  ) {
    captureMiaEvent(MIA_EVENTS.missionPublishFailed, {
      ...failureProperties(error),
      ...properties,
    });
  },

  missionStarted(properties: MissionStartedProperties) {
    captureMiaEvent(MIA_EVENTS.missionStarted, properties);
  },

  missionStartFailed(
    error: unknown,
    properties: Pick<MissionStartFailedProperties, "class_id" | "mission_id">,
  ) {
    captureMiaEvent(MIA_EVENTS.missionStartFailed, {
      ...failureProperties(error),
      ...properties,
    });
  },

  missionCompleted(properties: MissionCompletedProperties) {
    captureMiaEvent(MIA_EVENTS.missionCompleted, properties);
  },

  captureException(error: unknown, properties?: Properties) {
    initializeMiaAnalytics();
    posthog.captureException(error, properties);
  },
};
