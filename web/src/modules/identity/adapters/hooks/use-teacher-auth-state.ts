"use client";
import {
  getFirebaseAuth,
  isFirebaseConfigured,
} from "@/shared/firebase/client";
import { FirebaseError } from "firebase/app";
import {
  GoogleAuthProvider,
  onAuthStateChanged,
  signInWithCredential,
  signInWithEmailAndPassword,
  signOut,
  type AuthCredential,
  type User,
} from "firebase/auth";
import { useCallback, useEffect, useState } from "react";
type TeacherAuthStatus = "checking" | "signed_out" | "signed_in";

type TeacherAuthState = {
  readonly status: TeacherAuthStatus;
  readonly user: User | null;
  readonly error: string | null;
};

const ERROR_MESSAGES: Record<string, string> = {
  "auth/network-request-failed":
    "Couldn't reach the sign-in service. Check your connection and try again.",
  "auth/unauthorized-domain": "This domain isn't authorised for sign-in.",
  // With email-enumeration protection on, Firebase reports wrong email *or* password as one code.
  "auth/invalid-credential":
    "Email or password didn't match. Please try again.",
  "auth/invalid-email": "That doesn't look like an email address.",
  "auth/missing-password": "Enter the password.",
  "auth/wrong-password": "Email or password didn't match. Please try again.",
  "auth/user-not-found": "Email or password didn't match. Please try again.",
  "auth/too-many-requests": "Too many attempts. Wait a moment and try again.",
  "auth/operation-not-allowed":
    "This sign-in method isn't enabled for the project.",
  "auth/account-exists-with-different-credential":
    "This email is already linked to another sign-in method.",
  "auth/user-disabled": "This account has been disabled.",
};

function describeSignInError(error: unknown) {
  if (error instanceof FirebaseError) {
    return ERROR_MESSAGES[error.code] ?? "Sign-in failed. Please try again.";
  }

  return "Sign-in failed. Please try again.";
}

export function useTeacherAuthState() {
  const firebaseConfigured = isFirebaseConfigured();
  const [state, setState] = useState<TeacherAuthState>(() => ({
    status: firebaseConfigured ? "checking" : "signed_out",
    user: null,
    error: firebaseConfigured ? null : "Firebase is not configured.",
  }));

  useEffect(() => {
    if (!firebaseConfigured) {
      return;
    }

    const auth = getFirebaseAuth();
    return onAuthStateChanged(auth, (user) => {
      setState({
        status: user && !user.isAnonymous ? "signed_in" : "signed_out",
        user: user && !user.isAnonymous ? user : null,
        error: null,
      });
    });
  }, [firebaseConfigured]);

  /** Run one Firebase sign-in, dropping any anonymous student session first. Returns null (with `error` set) on failure. */
  const runSignIn = useCallback(
    async (
      signIn: (
        auth: ReturnType<typeof getFirebaseAuth>,
      ) => Promise<{ user: User }>,
    ) => {
      setState((current) => ({ ...current, error: null }));

      try {
        const auth = getFirebaseAuth();
        if (auth.currentUser?.isAnonymous) {
          await signOut(auth);
        }

        const credential = await signIn(auth);
        setState({
          status: "signed_in",
          user: credential.user,
          error: null,
        });

        return credential.user;
      } catch (error) {
        setState({
          status: "signed_out",
          user: null,
          error: describeSignInError(error),
        });
        return null;
      }
    },
    [],
  );

  /** Exchange the ID token from Google's button for a Firebase session. */
  const signInWithGoogleIdToken = useCallback(
    (idToken: string) => {
      const credential: AuthCredential = GoogleAuthProvider.credential(idToken);
      return runSignIn((auth) => signInWithCredential(auth, credential));
    },
    [runSignIn],
  );

  /** Sign in with teacher credentials entered on the login page. */
  const signInWithEmail = useCallback(
    (email: string, password: string) =>
      runSignIn((auth) =>
        signInWithEmailAndPassword(auth, email.trim(), password),
      ),
    [runSignIn],
  );

  /** Surface a problem that happened before Firebase was involved (Google's script failed to load). */
  const reportSignInError = useCallback((message: string) => {
    setState((current) => ({ ...current, error: message }));
  }, []);

  const signOutTeacher = useCallback(async () => {
    await signOut(getFirebaseAuth());
    setState({
      status: "signed_out",
      user: null,
      error: null,
    });
  }, []);

  return {
    ...state,
    firebaseConfigured,
    signInWithGoogleIdToken,
    signInWithEmail,
    reportSignInError,
    signOutTeacher,
  };
}
