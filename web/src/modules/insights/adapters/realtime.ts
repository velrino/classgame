import { getFirebaseDb } from "@/shared/firebase/client";
import { collection, doc, onSnapshot, query, where } from "firebase/firestore";
export function observeDashboardChanges(
  classId: string,
  onChange: () => void,
): () => void {
  const database = getFirebaseDb();
  let timer: ReturnType<typeof setTimeout> | undefined;
  const invalidation = () => {
    clearTimeout(timer);
    timer = setTimeout(onChange, 250);
  };
  let initialAttempts = true;
  let initialSummary = true;
  const attempts = onSnapshot(
    query(
      collection(database, "student_mission_attempts"),
      where("class_id", "==", classId),
    ),
    () => {
      if (!initialAttempts) invalidation();
      initialAttempts = false;
    },
    () => undefined,
  );
  const summary = onSnapshot(
    doc(database, "class_learning_summaries", classId),
    () => {
      if (!initialSummary) invalidation();
      initialSummary = false;
    },
    () => undefined,
  );
  return () => {
    clearTimeout(timer);
    attempts();
    summary();
  };
}
