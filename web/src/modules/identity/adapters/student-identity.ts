import type { StudentIdentity } from "@/modules/identity/adapters/types/student-identity";
import {
  getStudentFirebaseAuth,
  isFirebaseConfigured,
} from "@/shared/firebase/client";
import { onAuthStateChanged, signInAnonymously, signOut } from "firebase/auth";

export { isFirebaseConfigured as isIdentityConfigured };
export function currentStudentIdentity(): StudentIdentity | null {
  return getStudentFirebaseAuth().currentUser;
}
export function observeStudentIdentity(
  listener: (identity: StudentIdentity | null) => void,
): () => void {
  return onAuthStateChanged(getStudentFirebaseAuth(), listener);
}
export function signOutStudentIdentity(): Promise<void> {
  return signOut(getStudentFirebaseAuth());
}
export async function ensureAnonymousStudent(): Promise<{
  isNew: boolean;
  user: StudentIdentity;
}> {
  const auth = getStudentFirebaseAuth();
  if (auth.currentUser?.isAnonymous)
    return { isNew: false, user: auth.currentUser };
  if (auth.currentUser) await signOut(auth);
  const credential = await signInAnonymously(auth);
  return { isNew: true, user: credential.user };
}
