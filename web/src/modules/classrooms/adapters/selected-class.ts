import type { TeacherClassDto } from "@/modules/classrooms/adapters/types/contracts";

export const selectedClassStorageKey = "classgame:selected_class_id";

export function resolveSelectedClassId(
  current: string | null,
  teacherClasses: TeacherClassDto[],
) {
  const availableClassIds = new Set(
    teacherClasses.map((teacherClass) => teacherClass.class_id),
  );
  if (current && availableClassIds.has(current)) {
    return current;
  }

  const storedClassId =
    typeof window === "undefined"
      ? null
      : window.localStorage.getItem(selectedClassStorageKey)?.trim() || null;
  if (storedClassId && availableClassIds.has(storedClassId)) {
    return storedClassId;
  }

  return teacherClasses[0]?.class_id ?? null;
}

export function rememberSelectedClass(classId: string) {
  if (typeof window === "undefined" || !classId) {
    return;
  }

  window.localStorage.setItem(selectedClassStorageKey, classId);
}

export function upsertTeacherClass(
  current: TeacherClassDto[],
  teacherClass: TeacherClassDto,
) {
  const exists = current.some(
    (item) => item.class_id === teacherClass.class_id,
  );
  const nextItems = exists
    ? current.map((item) =>
        item.class_id === teacherClass.class_id ? teacherClass : item,
      )
    : [teacherClass, ...current];

  return sortTeacherClasses(nextItems);
}

export function sortTeacherClasses(teacherClasses: TeacherClassDto[]) {
  return [...teacherClasses].sort((left, right) => {
    return (
      new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime()
    );
  });
}
