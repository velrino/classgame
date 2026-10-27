export { useStudentClassMissions } from "./adapters/hooks/use-student-class-missions";
export {
  studentClassUnavailableEvent,
  studentSessionReadyEvent,
  useStudentPresence,
} from "./adapters/hooks/use-student-presence";
export { classroomsApi } from "./adapters/http";
export {
  rememberSelectedClass,
  resolveSelectedClassId,
  upsertTeacherClass,
} from "./adapters/selected-class";
export {
  clearStudentMissionSession,
  clearStudentSession,
  readStudentMissionSession,
  saveStudentClassSession,
  saveStudentMissionSession,
} from "./adapters/session-storage";
export type { StudentMissionSession } from "./adapters/session-storage";
export { studentMissionRevisionKey } from "./adapters/student-catalog";
export type * from "./adapters/types/contracts";
export type { StudentClassMission } from "./adapters/types/student-catalog";
export {
  studentEntryCopy,
  studentSessionErrorScope,
} from "./domain/student-entry-messages";
export { ClassesPage } from "./ui/pages/classes-page";
export { TeacherClassesPage } from "./ui/pages/teacher-classes-page";
export { StudentSessionGate } from "./ui/student-session-gate";

export { readStudentClassSession } from "./adapters/session-storage";
