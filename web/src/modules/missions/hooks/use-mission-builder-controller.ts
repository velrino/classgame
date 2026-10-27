"use client";
import {
  MissionBuilderDraft,
  MissionBuilderProps,
} from "@/modules/missions/adapters/types/builder";
import type {
  MissionAttachmentDto,
  TeacherClassMissionDto,
} from "@/modules/missions/adapters/types/contracts";
import {
  buildMissionBuilderDraft,
  limitMissionBuilderDraft,
  mergeMissionAttachments,
  previewMissionCanBeApproved,
} from "@/modules/missions/domain/builder";
import { missionHasPublishedRevision } from "@/modules/missions/domain/mission-lifecycle";
import { useCallback, useState, type FormEvent } from "react";
export function useMissionBuilderController({
  approving,
  classId,
  draftPrompt,
  onOpenChange,
  onProcess,
  previewMission,
  processing,
  teacherClass,
}: Pick<
  MissionBuilderProps,
  | "approving"
  | "classId"
  | "draftPrompt"
  | "onOpenChange"
  | "onProcess"
  | "previewMission"
  | "processing"
  | "teacherClass"
>) {
  const [form, setForm] = useState<MissionBuilderDraft>(() =>
    buildMissionBuilderDraft(previewMission, draftPrompt),
  );
  const [persistedAttachments, setPersistedAttachments] = useState<
    MissionAttachmentDto[]
  >(() => previewMission?.attachments ?? []);
  const [removedAttachmentIds, setRemovedAttachmentIds] = useState<
    ReadonlySet<string>
  >(() => new Set());
  const [previewAttachment, setPreviewAttachment] =
    useState<MissionAttachmentDto | null>(null);
  const [editableMission, setEditableMission] =
    useState<TeacherClassMissionDto | null>(() => previewMission);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const visibleMission = previewMission;
  const previewDraft = editableMission ?? visibleMission;
  const guardrailBlocked = Boolean(
    previewDraft?.education_guardrail &&
      !previewDraft.education_guardrail.allowed &&
      previewDraft.education_guardrail.reason.trim(),
  );
  const canShowApprove = Boolean(
    previewDraft &&
      !guardrailBlocked &&
      (previewDraft.status === "preview" ||
        previewDraft.has_unpublished_changes),
  );
  const canApprove = Boolean(
    previewDraft && canShowApprove && previewMissionCanBeApproved(previewDraft),
  );
  const hasLiveRevision = Boolean(
    previewDraft && missionHasPublishedRevision(previewDraft),
  );
  const editingMission = Boolean(form.mission_id);
  const modalTitle = `${editingMission ? "Edit mission" : "Create mission"} for ${
    teacherClass?.class_name || classId
  }`;
  const classContext = [
    teacherClass?.subject || "No subject set",
    teacherClass?.grade_level,
  ]
    .filter(Boolean)
    .join(" · ");
  const modalDescription = editingMission
    ? `${classContext}. Update this mission draft. Students keep the published revision until you approve the new preview.`
    : `${classContext}. Mia will generate a student preview for you to review before publishing.`;
  const busy = processing || approving || uploading;
  const showPreview = processing || Boolean(visibleMission);
  const classDetailsReady = Boolean(
    teacherClass?.class_name.trim() && teacherClass.subject.trim(),
  );
  const remainingFileSlots = Math.max(0, 5 - persistedAttachments.length);
  const canSubmit = !busy && classDetailsReady;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setUploadError(null);

    if (!teacherClass) {
      setUploadError("Class details are still loading. Try again in a moment.");
      return;
    }

    await onProcess({
      ...limitMissionBuilderDraft(form),
      audience_type: "all",
      target_student_ids: [],
      class_id: teacherClass.class_id || classId,
      class_name: teacherClass.class_name,
      subject: teacherClass.subject,
      grade_level: teacherClass.grade_level,
    });
  }

  const handleAttachmentsChange = useCallback(
    (attachments: MissionAttachmentDto[]) => {
      setForm((current) => ({
        ...current,
        attachments: mergeMissionAttachments(
          persistedAttachments,
          attachments,
        ).filter(
          (attachment) => !removedAttachmentIds.has(attachment.uploadcare_uuid),
        ),
      }));
    },
    [persistedAttachments, removedAttachmentIds],
  );

  function removeAttachment(uploadcareUUID: string) {
    setRemovedAttachmentIds((current) => {
      const next = new Set(current);
      next.add(uploadcareUUID);
      return next;
    });
    setPersistedAttachments((current) =>
      current.filter(
        (attachment) => attachment.uploadcare_uuid !== uploadcareUUID,
      ),
    );
    setForm((current) => ({
      ...current,
      attachments: current.attachments.filter(
        (attachment) => attachment.uploadcare_uuid !== uploadcareUUID,
      ),
    }));
    setPreviewAttachment((current) =>
      current?.uploadcare_uuid === uploadcareUUID ? null : current,
    );
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) {
      setPreviewAttachment(null);
    }

    onOpenChange(nextOpen);
  }

  function updateEditableMission(
    updater: (mission: TeacherClassMissionDto) => TeacherClassMissionDto,
  ) {
    setEditableMission((current) => {
      const sourceMission = current ?? visibleMission;

      return sourceMission ? updater(sourceMission) : current;
    });
  }

  return {
    form,
    setForm,
    previewAttachment,
    setPreviewAttachment,
    uploading,
    setUploading,
    uploadError,
    setUploadError,
    visibleMission,
    previewDraft,
    canShowApprove,
    canApprove,
    hasLiveRevision,
    modalTitle,
    modalDescription,
    busy,
    showPreview,
    remainingFileSlots,
    canSubmit,
    handleSubmit,
    handleAttachmentsChange,
    removeAttachment,
    handleOpenChange,
    updateEditableMission,
  };
}
