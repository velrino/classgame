import type { StudentClassDto } from "@/modules/classrooms/adapters/types/contracts";
import type { StudentMissionDto } from "@/modules/missions";

export const studentClassSessionStorageKey = "classgame:student-class-session";
const studentMissionSessionKey = "classgame:student-mission-session";
export const studentClassSessionChangedEvent =
  "classgame:student-class-session-changed";

export type StudentClassSession = {
  readonly classId: string;
  readonly classCode: string;
  readonly className: string;
  readonly subject: string;
  readonly gradeLevel: string;
  readonly studentName: string;
};

export type StudentMissionSession = {
  readonly attemptId: string;
  readonly classCode: string;
  readonly missionId: string;
  readonly publishedRevision: number;
  readonly subject: string;
  readonly mission: StudentMissionDto;
};

export function saveStudentClassSession(
  studentClass: StudentClassDto,
  studentName: string,
) {
  const session: StudentClassSession = {
    classId: studentClass.id,
    classCode: studentClass.class_code,
    className: studentClass.class_name,
    subject: studentClass.subject,
    gradeLevel: studentClass.grade_level,
    studentName,
  };
  window.localStorage.setItem(
    studentClassSessionStorageKey,
    JSON.stringify(session),
  );
  window.sessionStorage.removeItem(studentClassSessionStorageKey);
  window.dispatchEvent(new Event(studentClassSessionChangedEvent));
}

export function readStudentClassSession(): StudentClassSession | null {
  try {
    const localValue = window.localStorage.getItem(
      studentClassSessionStorageKey,
    );
    if (localValue) {
      return JSON.parse(localValue) as StudentClassSession;
    }

    // Preserve a session created before class sessions were shared across tabs.
    const value = window.sessionStorage.getItem(studentClassSessionStorageKey);
    if (value) {
      window.localStorage.setItem(studentClassSessionStorageKey, value);
      window.sessionStorage.removeItem(studentClassSessionStorageKey);
    }
    return value ? (JSON.parse(value) as StudentClassSession) : null;
  } catch {
    return null;
  }
}

export function saveStudentMissionSession(session: StudentMissionSession) {
  window.sessionStorage.setItem(
    studentMissionSessionKey,
    JSON.stringify(session),
  );
}

export function readStudentMissionSession(): StudentMissionSession | null {
  try {
    const value = window.sessionStorage.getItem(studentMissionSessionKey);
    return value ? (JSON.parse(value) as StudentMissionSession) : null;
  } catch {
    return null;
  }
}

export function clearStudentMissionSession() {
  window.sessionStorage.removeItem(studentMissionSessionKey);
}

export function clearStudentSession() {
  window.localStorage.removeItem(studentClassSessionStorageKey);
  window.sessionStorage.removeItem(studentClassSessionStorageKey);
  clearStudentMissionSession();
  window.dispatchEvent(new Event(studentClassSessionChangedEvent));
}
