import type { TeacherClassDto } from "@/modules/classrooms";
import type {
  CreateTeacherClassMissionRequest,
  MissionPreviewProgressDto,
  TeacherClassMissionDto,
} from "@/modules/missions/adapters/types/contracts";

export type MissionBuilderForm = CreateTeacherClassMissionRequest & {
  class_id: string;
};

export type MissionBuilderDraft = Omit<
  MissionBuilderForm,
  "class_id" | "class_name" | "subject" | "grade_level"
>;

export type MissionBuilderProps = {
  readonly approving: boolean;
  readonly classId: string;
  /** Pre-written teacher prompt, e.g. built from the class patterns on the dashboard. */
  readonly draftPrompt?: string;
  readonly open: boolean;
  readonly previewMission: TeacherClassMissionDto | null;
  readonly processing: boolean;
  readonly progress?: MissionPreviewProgressDto | null;
  readonly requestError?: string | null;
  readonly teacherClass: TeacherClassDto | null;
  readonly onApprove: (mission: TeacherClassMissionDto) => Promise<void>;
  readonly onOpenChange: (open: boolean) => void;
  readonly onProcess: (value: MissionBuilderForm) => Promise<void>;
};
