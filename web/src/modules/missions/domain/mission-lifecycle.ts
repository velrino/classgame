import type { TeacherClassMissionDto } from "@/modules/missions/adapters/types/contracts";

const zeroTimestampPrefix = "0001-";

export function missionHasPublishedRevision(mission: TeacherClassMissionDto) {
  return (
    (mission.published_revision ?? 0) > 0 ||
    Boolean(
      mission.published_at &&
        !mission.published_at.startsWith(zeroTimestampPrefix),
    )
  );
}

export function missionHasUnpublishedChanges(mission: TeacherClassMissionDto) {
  if (mission.has_unpublished_changes !== undefined) {
    return mission.has_unpublished_changes;
  }

  return (
    (mission.published_revision ?? 0) > 0 &&
    (mission.draft_revision ?? 0) > (mission.published_revision ?? 0)
  );
}

export function missionIsStandaloneDraft(mission: TeacherClassMissionDto) {
  return (
    (mission.status === "draft" || mission.status === "preview") &&
    !missionHasPublishedRevision(mission)
  );
}
