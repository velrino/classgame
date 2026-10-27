import type { StudentClassMission } from "@/modules/classrooms/adapters/types/student-catalog";
import { getStudentFirebaseDb } from "@/shared/firebase/client";
import { collection, onSnapshot, query, where } from "firebase/firestore";

export function studentMissionRevisionKey(
  missionID: string,
  publishedRevision: number,
) {
  return `${missionID}:${publishedRevision}`;
}

function studentClassMissionFromData(
  documentID: string,
  data: Record<string, unknown>,
): StudentClassMission | null {
  const id = text(data.id) || documentID;
  const publishedRevision = number(data.published_revision);
  const title = text(data.title) || text(data.mission_name);
  if (!id || publishedRevision <= 0 || !title || data.status !== "published") {
    return null;
  }

  return {
    id,
    publishedRevision,
    missionName: text(data.mission_name) || title,
    title,
    summary: text(data.summary),
    subject: text(data.subject),
    gradeLevel: text(data.grade_level),
    learningGoal: text(data.learning_goal),
    suggestedDurationMinutes: number(data.suggested_duration_minutes),
    publishedAt: timestampMilliseconds(data.published_at),
  };
}

function text(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}

function number(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function timestampMilliseconds(value: unknown) {
  if (
    typeof value === "object" &&
    value !== null &&
    "toMillis" in value &&
    typeof value.toMillis === "function"
  ) {
    return value.toMillis();
  }

  return 0;
}

export function observeStudentMissions(
  classId: string,
  onMissions: (missions: StudentClassMission[]) => void,
  onError: (error: Error) => void,
) {
  return onSnapshot(
    collection(
      getStudentFirebaseDb(),
      "student_class_missions",
      classId,
      "missions",
    ),
    (snapshot) =>
      onMissions(
        snapshot.docs
          .map((item) => studentClassMissionFromData(item.id, item.data()))
          .filter((item): item is StudentClassMission => item !== null)
          .sort((a, b) => b.publishedAt - a.publishedAt),
      ),
    onError,
  );
}
export function observeStudentCompletions(
  classId: string,
  studentId: string,
  onCompletions: (keys: Set<string>) => void,
) {
  return onSnapshot(
    query(
      collection(
        getStudentFirebaseDb(),
        "student_mission_completions",
        studentId,
        "missions",
      ),
      where("student_id", "==", studentId),
    ),
    (snapshot) => {
      const keys = new Set<string>();
      for (const item of snapshot.docs) {
        const data = item.data();
        if (
          data.class_id === classId &&
          data.status === "completed" &&
          typeof data.mission_id === "string" &&
          typeof data.published_revision === "number"
        )
          keys.add(
            studentMissionRevisionKey(data.mission_id, data.published_revision),
          );
      }
      onCompletions(keys);
    },
    () => undefined,
  );
}
