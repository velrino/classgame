import { MissionBuilderDraft } from "@/modules/missions/adapters/types/builder";
import type {
  MissionAttachmentDto,
  TeacherClassMissionDto,
} from "@/modules/missions/adapters/types/contracts";
import { normalizeMissionPreviewQuestions } from "@/modules/missions/domain/mission-preview";
import { INPUT_LIMITS, limitInput } from "@/shared/input-limits";

export function previewMissionCanBeApproved(mission: TeacherClassMissionDto) {
  const previewQuestions = normalizeMissionPreviewQuestions(
    mission.preview_questions,
  );

  return (
    mission.title.trim().length >= 2 &&
    mission.summary.trim().length >= 10 &&
    previewQuestions.length > 0 &&
    previewQuestions.every(
      (question) =>
        question.prompt.trim().length >= 2 &&
        question.expected_answer.trim().length > 0,
    )
  );
}

export function limitMissionPreview(
  mission: TeacherClassMissionDto,
): TeacherClassMissionDto {
  return {
    ...mission,
    title: limitInput(mission.title, INPUT_LIMITS.missionTitle),
    summary: limitInput(mission.summary, INPUT_LIMITS.missionSummary),
    preview_questions: normalizeMissionPreviewQuestions(
      mission.preview_questions,
    ).map((question) => ({
      ...question,
      prompt: limitInput(question.prompt, INPUT_LIMITS.missionQuestionPrompt),
      options: question.options
        ? question.options
            .slice(0, 4)
            .map((option) =>
              limitInput(option, INPUT_LIMITS.missionQuestionOption),
            )
        : question.options,
      expected_answer: limitInput(
        question.expected_answer,
        INPUT_LIMITS.missionExpectedAnswer,
      ),
    })),
  };
}

export function buildMissionBuilderDraft(
  mission: TeacherClassMissionDto | null,
  draftPrompt?: string,
): MissionBuilderDraft {
  return {
    mission_id: mission?.id,
    expected_draft_revision: mission?.draft_revision,
    mission_name: limitInput(
      mission?.mission_name ?? "",
      INPUT_LIMITS.missionName,
    ),
    learning_goal: limitInput(
      mission?.learning_goal ?? "",
      INPUT_LIMITS.learningGoal,
    ),
    teacher_prompt: limitInput(
      mission?.teacher_prompt ?? draftPrompt ?? "",
      INPUT_LIMITS.missionTeacherPrompt,
    ),
    audience_type: "all",
    target_student_ids: [],
    attachments: mission?.attachments ?? [],
  };
}

export function limitMissionBuilderDraft(
  draft: MissionBuilderDraft,
): MissionBuilderDraft {
  return {
    ...draft,
    attachments: draft.attachments.map((attachment) => ({
      ...attachment,
      cdn_url: attachment.uploadcare_cdn_url?.trim() || attachment.cdn_url,
    })),
    mission_name: limitInput(draft.mission_name, INPUT_LIMITS.missionName),
    learning_goal: limitInput(draft.learning_goal, INPUT_LIMITS.learningGoal),
    teacher_prompt: limitInput(
      draft.teacher_prompt,
      INPUT_LIMITS.missionTeacherPrompt,
    ),
  };
}

export function mergeMissionAttachments(
  persisted: MissionAttachmentDto[],
  uploaded: MissionAttachmentDto[],
) {
  const attachments = new Map<string, MissionAttachmentDto>();
  for (const attachment of [...persisted, ...uploaded]) {
    attachments.set(attachment.uploadcare_uuid, attachment);
  }

  return [...attachments.values()];
}
