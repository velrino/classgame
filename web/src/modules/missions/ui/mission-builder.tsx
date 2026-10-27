"use client";
import { Button } from "@/components/ui/button";
import { Modal } from "@/components/ui/modal";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { MissionBuilderProps } from "@/modules/missions/adapters/types/builder";
import { limitMissionPreview } from "@/modules/missions/domain/builder";
import { useMissionBuilderController } from "@/modules/missions/hooks/use-mission-builder-controller";
import {
  AttachmentPreviewModal,
  UploadedMissionFilesGrid,
} from "@/modules/missions/ui/attachment-preview";
import {
  CharacterCounter,
  EditableMissionPreview,
  EditablePreviewField,
} from "@/modules/missions/ui/editable-mission-preview";
import { MissionFileUploader } from "@/modules/missions/ui/mission-file-uploader";
import { MissionFlowPreview } from "@/modules/missions/ui/mission-flow-preview";
import {
  PreviewLoadingOverlay,
  PreviewLoadingState,
} from "@/modules/missions/ui/mission-generation-progress";
import { INPUT_LIMITS, limitInput } from "@/shared/input-limits";

export function MissionBuilder({
  approving,
  classId,
  draftPrompt,
  onApprove,
  onOpenChange,
  onProcess,
  open,
  previewMission,
  processing,
  progress,
  requestError,
  teacherClass,
}: MissionBuilderProps) {
  const {
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
  } = useMissionBuilderController({
    approving,
    classId,
    draftPrompt,
    onOpenChange,
    onProcess,
    previewMission,
    processing,
    teacherClass,
  });

  return (
    <Modal
      open={open}
      onOpenChange={handleOpenChange}
      title={modalTitle}
      description={modalDescription}
      size="xl"
      layout="fill"
    >
      <section className="flex min-h-0 flex-1 flex-col overflow-y-auto p-4 sm:p-6 lg:flex-row lg:overflow-hidden">
        <form
          className="min-w-0 flex-1 rounded-lg border border-border lg:min-h-0 lg:overflow-y-auto"
          onSubmit={handleSubmit}
        >
          <div className="px-4 py-5 sm:px-6">
            {hasLiveRevision ? (
              <p className="mb-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm font-semibold text-amber-800">
                The current published revision stays live until you approve
                these draft changes.
              </p>
            ) : null}

            <label className="grid gap-2 text-sm font-medium text-foreground">
              Mission name
              <input
                required
                minLength={2}
                value={form.mission_name}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    mission_name: limitInput(
                      event.target.value,
                      INPUT_LIMITS.missionName,
                    ),
                  }))
                }
                maxLength={INPUT_LIMITS.missionName}
                placeholder="Fractions in equations review"
                className="h-10 rounded-md border border-input bg-background px-3 text-sm outline-none transition focus:border-ring focus:ring-2 focus:ring-ring/20"
              />
              <CharacterCounter
                value={form.mission_name}
                maxLength={INPUT_LIMITS.missionName}
              />
            </label>

            <label className="mt-4 grid gap-2 text-sm font-medium text-foreground">
              Learning goal
              <input
                required
                minLength={6}
                value={form.learning_goal}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    learning_goal: limitInput(
                      event.target.value,
                      INPUT_LIMITS.learningGoal,
                    ),
                  }))
                }
                maxLength={INPUT_LIMITS.learningGoal}
                placeholder="Help students review fractions inside two-step equations."
                className="h-10 rounded-md border border-input bg-background px-3 text-sm outline-none transition focus:border-ring focus:ring-2 focus:ring-ring/20"
              />
              <CharacterCounter
                value={form.learning_goal}
                maxLength={INPUT_LIMITS.learningGoal}
              />
            </label>

            <label className="mt-4 grid gap-2 text-sm font-medium text-foreground">
              Teacher prompt
              <textarea
                required
                minLength={10}
                value={form.teacher_prompt}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    teacher_prompt: limitInput(
                      event.target.value,
                      INPUT_LIMITS.missionTeacherPrompt,
                    ),
                  }))
                }
                maxLength={INPUT_LIMITS.missionTeacherPrompt}
                placeholder="What did students study today? What should Mia watch for? What should count as success?"
                className="min-h-40 resize-none rounded-md border border-input bg-background px-3 py-3 text-sm leading-6 outline-none transition focus:border-ring focus:ring-2 focus:ring-ring/20"
              />
              <CharacterCounter
                value={form.teacher_prompt}
                maxLength={INPUT_LIMITS.missionTeacherPrompt}
              />
            </label>

            <div className="mt-4 grid gap-2">
              <div>
                <p className="text-sm font-medium text-foreground">
                  Optional files
                </p>
                <p className="mt-1 text-xs leading-5 text-muted-foreground">
                  PDF, PNG, JPEG, DOCX, or TXT only. Up to 5 files, 20 MB each.
                </p>
              </div>
              <MissionFileUploader
                disabled={processing || approving}
                maxFiles={remainingFileSlots}
                onAttachmentsChange={handleAttachmentsChange}
                onErrorChange={setUploadError}
                onUploadingChange={setUploading}
              />
            </div>

            {form.attachments.length > 0 ? (
              <UploadedMissionFilesGrid
                attachments={form.attachments}
                disabled={busy}
                onPreview={setPreviewAttachment}
                onRemove={removeAttachment}
              />
            ) : null}

            {uploadError ? (
              <p className="mt-3 rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm font-semibold text-rose-700">
                {uploadError}
              </p>
            ) : null}

            {requestError ? (
              <p
                role="alert"
                className="mt-3 rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm font-semibold text-rose-700"
              >
                {requestError}
              </p>
            ) : null}

            <div className="mt-5 flex justify-end">
              <Button type="submit" disabled={!canSubmit}>
                {uploading
                  ? "Uploading files..."
                  : processing
                    ? "Generating preview..."
                    : visibleMission
                      ? "Update preview"
                      : "Generate preview"}
              </Button>
            </div>
          </div>
        </form>

        <div
          aria-hidden={!showPreview}
          inert={showPreview ? undefined : true}
          className={cn(
            "grid min-w-0 overflow-hidden transition-[grid-template-rows,margin,width,opacity,transform] duration-500 ease-out motion-reduce:transition-none lg:shrink-0",
            showPreview
              ? "mt-4 grid-rows-[1fr] opacity-100 lg:mt-0 lg:ml-4 lg:w-[520px] lg:translate-x-0"
              : "pointer-events-none mt-0 grid-rows-[0fr] translate-y-2 opacity-0 lg:ml-0 lg:w-0 lg:translate-x-4 lg:translate-y-0",
          )}
        >
          <div className="min-h-0 overflow-hidden lg:h-full lg:w-[520px]">
            <aside className="relative flex max-h-[calc(100svh-11rem)] flex-col overflow-hidden rounded-lg border border-border lg:h-full lg:max-h-none">
              {processing && previewDraft ? (
                <PreviewLoadingOverlay progress={progress} />
              ) : null}
              <header className="shrink-0 border-b border-border px-4 py-4 sm:px-6">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-bold uppercase text-cyan-700">
                      Preview
                    </p>
                    {previewDraft ? (
                      <EditablePreviewField
                        ariaLabel="Mission title"
                        disabled={busy}
                        maxLength={INPUT_LIMITS.missionTitle}
                        value={previewDraft.title}
                        onChange={(title) =>
                          updateEditableMission((mission) => ({
                            ...mission,
                            title,
                          }))
                        }
                        className="mt-2 text-lg font-semibold leading-7 text-foreground"
                      />
                    ) : (
                      <h3 className="mt-2 text-lg font-semibold">
                        Waiting for Mia
                      </h3>
                    )}
                  </div>
                  {previewDraft ? (
                    <span className="rounded-md bg-muted px-2 py-1 text-xs font-bold uppercase text-muted-foreground">
                      {previewDraft.status}
                    </span>
                  ) : null}
                </div>
              </header>

              <div className="grid min-h-0 flex-1 gap-4 overflow-y-auto px-4 py-5 sm:px-6">
                {previewDraft ? (
                  <>
                    {/* {previewDraft.education_guardrail &&
                    !previewDraft.education_guardrail.allowed ? (
                      <p className="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm font-semibold text-rose-700">
                        {previewDraft.education_guardrail.reason}
                      </p>
                    ) : previewDraft.education_guardrail?.attachment_notes ? (
                      <p className="rounded-md border border-cyan-100 bg-cyan-50 px-3 py-2 text-xs font-semibold leading-5 text-cyan-800">
                        {previewDraft.education_guardrail.attachment_notes}
                      </p>
                    ) : null} */}

                    <Tabs defaultValue="preview">
                      <TabsList className="grid w-full grid-cols-2">
                        <TabsTrigger value="preview">Preview</TabsTrigger>
                        <TabsTrigger value="flow">Flow</TabsTrigger>
                      </TabsList>

                      <TabsContent value="preview" className="mt-3 grid gap-4">
                        <EditableMissionPreview
                          key={`${previewDraft.id}:${previewDraft.draft_revision}`}
                          disabled={busy}
                          mission={previewDraft}
                          onMissionChange={updateEditableMission}
                        />
                      </TabsContent>

                      <TabsContent value="flow" className="mt-3">
                        <MissionFlowPreview flow={previewDraft.mission_flow} />
                      </TabsContent>
                    </Tabs>

                    {previewDraft.orchestration.generated_by_mock ? (
                      <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm font-semibold text-amber-700">
                        Local fallback generated this preview.
                      </p>
                    ) : null}

                    {canShowApprove ? (
                      <div className="grid gap-2">
                        {!canApprove ? (
                          <p className="text-xs font-medium text-muted-foreground">
                            Title, summary, prompt, and expected answer are
                            required before approval.
                          </p>
                        ) : null}
                        <Button
                          type="button"
                          disabled={!canApprove || approving || processing}
                          onClick={() => {
                            if (!previewDraft) {
                              return;
                            }

                            void onApprove(limitMissionPreview(previewDraft));
                          }}
                          className="bg-emerald-600 text-white hover:bg-emerald-500"
                        >
                          {approving
                            ? "Publishing..."
                            : hasLiveRevision
                              ? "Approve and publish changes"
                              : "Approve and assign to class"}
                        </Button>
                      </div>
                    ) : null}
                  </>
                ) : processing ? (
                  <PreviewLoadingState progress={progress} />
                ) : (
                  <p className="text-sm leading-6 text-muted-foreground">
                    The preview will appear here after Mia processes the
                    mission.
                  </p>
                )}
              </div>
            </aside>
          </div>
        </div>
      </section>
      <AttachmentPreviewModal
        access={
          previewMission ? { classId, missionId: previewMission.id } : undefined
        }
        attachment={previewAttachment}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            setPreviewAttachment(null);
          }
        }}
      />
    </Modal>
  );
}
