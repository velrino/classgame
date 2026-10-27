import type {
  MissionPreviewProgressDto,
  TeacherClassMissionResponse,
} from "@/modules/missions/adapters/types/contracts";
import { ApiError, type ApiErrorBody } from "@/shared/http/errors";
type ParsedServerSentEvent = {
  event: string;
  data: string;
};

export async function readMissionPreviewEventStream(
  res: Response,
  onProgress?: (progress: MissionPreviewProgressDto) => void,
): Promise<TeacherClassMissionResponse> {
  const reader = res.body?.getReader();
  if (!reader) {
    throw new ApiError(
      res.status,
      "MISSION_PREVIEW_STREAM_UNAVAILABLE",
      "Mission preview stream is unavailable.",
    );
  }

  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (value) {
        buffer += decoder.decode(value, { stream: !done });
        if (buffer.length > 1024 * 1024)
          throw new ApiError(
            502,
            "MISSION_PREVIEW_EVENT_TOO_LARGE",
            "Mission preview event exceeds its size limit.",
          );
        const response = parseMissionPreviewStreamBuffer(buffer, onProgress);
        buffer = response.remaining;
        if (response.done) {
          return response.done;
        }
      }
      if (done) {
        const response = parseMissionPreviewStreamBuffer(
          `${buffer}\n\n`,
          onProgress,
        );
        if (response.done) {
          return response.done;
        }
        break;
      }
    }

    throw new ApiError(
      res.status,
      "MISSION_PREVIEW_STREAM_ENDED",
      "Mission preview stream ended before returning a mission.",
    );
  } finally {
    await reader.cancel().catch(() => undefined);
    reader.releaseLock();
  }
}

function parseMissionPreviewStreamBuffer(
  buffer: string,
  onProgress?: (progress: MissionPreviewProgressDto) => void,
): {
  remaining: string;
  done: TeacherClassMissionResponse | null;
} {
  let remaining = buffer;
  while (true) {
    const boundary = findServerSentEventBoundary(remaining);
    if (!boundary) {
      return { remaining, done: null };
    }

    const rawEvent = remaining.slice(0, boundary.index);
    remaining = remaining.slice(boundary.index + boundary.length);
    const parsed = parseServerSentEvent(rawEvent);
    if (!parsed) {
      continue;
    }

    if (parsed.event === "progress") {
      onProgress?.(JSON.parse(parsed.data) as MissionPreviewProgressDto);
      continue;
    }

    if (parsed.event === "done") {
      return {
        remaining,
        done: JSON.parse(parsed.data) as TeacherClassMissionResponse,
      };
    }

    if (parsed.event === "error") {
      throw streamApiError(JSON.parse(parsed.data) as ApiErrorBody);
    }
  }
}

function findServerSentEventBoundary(
  value: string,
): { index: number; length: number } | null {
  const lfIndex = value.indexOf("\n\n");
  const crlfIndex = value.indexOf("\r\n\r\n");
  if (lfIndex < 0 && crlfIndex < 0) {
    return null;
  }
  if (lfIndex < 0) {
    return { index: crlfIndex, length: 4 };
  }
  if (crlfIndex < 0 || lfIndex < crlfIndex) {
    return { index: lfIndex, length: 2 };
  }

  return { index: crlfIndex, length: 4 };
}

function parseServerSentEvent(rawEvent: string): ParsedServerSentEvent | null {
  const lines = rawEvent.split(/\r?\n/);
  let event = "message";
  const data: string[] = [];

  for (const line of lines) {
    if (line.startsWith("event:")) {
      event = line.slice("event:".length).trim();
      continue;
    }
    if (line.startsWith("data:")) {
      data.push(line.slice("data:".length).trimStart());
    }
  }

  if (data.length === 0) {
    return null;
  }

  return {
    event,
    data: data.join("\n"),
  };
}

function streamApiError(body: ApiErrorBody): ApiError {
  const code = body.error?.code ?? body.code ?? "MISSION_PREVIEW_STREAM_FAILED";
  const message =
    body.error?.message ?? body.message ?? "Mission preview stream failed.";

  return new ApiError(500, code, message);
}
