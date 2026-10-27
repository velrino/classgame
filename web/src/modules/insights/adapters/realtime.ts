import { getFirebaseDb } from "@/shared/firebase/client";
import { doc, onSnapshot } from "firebase/firestore";

// One small document invalidates the visible page. Never subscribe to all attempts.
export function observeDashboardChanges(classId: string, onChange: () => void): () => void {
  let initial = true;
  return onSnapshot(doc(getFirebaseDb(), "class_learning_summaries", classId), () => {
    if (!initial) onChange();
    initial = false;
  }, () => undefined);
}
