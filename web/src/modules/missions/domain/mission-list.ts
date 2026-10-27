import type {
  ApproveTeacherClassMissionRequest,
  TeacherClassMissionDto,
} from "@/modules/missions/adapters/types/contracts";
import { missionIsStandaloneDraft } from "@/modules/missions/domain/mission-lifecycle";
import { normalizeMissionPreviewQuestions } from "@/modules/missions/domain/mission-preview";

export function upsertClassMission(
  missions: TeacherClassMissionDto[],
  mission: TeacherClassMissionDto,
) {
  return [
    mission,
    ...missions.filter(
      (current) =>
        current.id !== mission.id &&
        !shouldDropStandaloneDraftAfterUpsert(current, mission),
    ),
  ].sort(
    (left, right) =>
      toTimestamp(right.updated_at) - toTimestamp(left.updated_at),
  );
}

export function shouldDropStandaloneDraftAfterUpsert(
  current: TeacherClassMissionDto,
  incoming: TeacherClassMissionDto,
) {
  return (
    current.class_id === incoming.class_id &&
    missionIsStandaloneDraft(incoming) &&
    missionIsStandaloneDraft(current)
  );
}

export function buildApproveMissionRequest(
  mission: TeacherClassMissionDto,
): ApproveTeacherClassMissionRequest {
  return {
    expected_draft_revision: mission.draft_revision,
    draft: {
      title: mission.title,
      summary: mission.summary,
      preview_questions: normalizeMissionPreviewQuestions(
        mission.preview_questions,
      ),
    },
  };
}

export function toTimestamp(value: string) {
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}
