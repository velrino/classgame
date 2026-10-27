import {
  normalizeStudentMemory,
  normalizeSessionCheckpoint,
  repairUnansweredCompletion,
  type StudentMemory,
  type StudentRecovery,
  type StudentSessionCheckpoint,
} from "@/modules/learning/domain/memory";
import type { GameMission } from "@/modules/learning/domain/published-mission";

const STORAGE_KEY_PREFIX = "classgame.student.v2";

export function loadStudentRecovery(
  mission: GameMission,
  recoveryID = mission.id,
): StudentRecovery | null {
  try {
    const raw = window.localStorage.getItem(storageKey(recoveryID));

    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw) as Partial<StudentMemory> & {
      checkpoint?: StudentSessionCheckpoint;
      snapshotRevision?: number;
    };

    return parsed.version === 2
      ? {
          memory: repairUnansweredCompletion(
            normalizeStudentMemory(parsed, mission), mission,
          ),
          checkpoint: normalizeSessionCheckpoint(parsed.checkpoint),
          revision: Math.max(0, parsed.snapshotRevision ?? 0),
        }
      : null;
  } catch {
    return null;
  }
}

export function saveMemory(
  memory: StudentMemory,
  recoveryID: string,
  checkpoint: StudentSessionCheckpoint | null,
  snapshotRevision: number,
) {
  try {
    // Only the recoverable game position is cached on the student's device.
    // Teacher notes, event history and tutor decisions remain server-side.
    window.localStorage.setItem(
      storageKey(recoveryID),
      JSON.stringify({
        ...memory,
        notes: [],
        decisions: [],
        events: [],
        checkpoint,
        snapshotRevision,
      }),
    );
  } catch {
    // Storage is a convenience here; the session keeps working without it.
  }
}

export function clearMemory(recoveryID: string) {
  try {
    window.localStorage.removeItem(storageKey(recoveryID));
  } catch {
    // ignore
  }
}

function storageKey(recoveryID: string) {
  return `${STORAGE_KEY_PREFIX}:${recoveryID}`;
}
