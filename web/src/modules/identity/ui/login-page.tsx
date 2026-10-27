"use client";
import { identityApi } from "@/modules/identity/adapters/http";

import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState, type FormEvent } from "react";

import { INPUT_LIMITS } from "@/shared/input-limits";

import { useTeacherAuth } from "@/modules/identity/hooks/use-teacher-auth";
import { GoogleIdentityButton } from "@/modules/identity/ui/google-identity-button";

const LOGIN_HERO_PHRASES = [
  "Set what your class should reinforce, then read Mia's notebook after students play.",
  "Turn every mission into evidence you can scan before the next lesson.",
  "Pick the skill to practice, let Mia guide the game, and review the why behind each answer.",
  "See who is ready, who needs a hint, and what the class should revisit tomorrow.",
  "Give students a playful route through practice while your dashboard keeps the signal clear.",
  "Build missions around your goals, then use Mia's notes to plan the follow-up.",
  "Watch patterns emerge from student choices, not just from final scores.",
  "Keep reinforcement focused while Mia turns each question into a small conversation.",
  "Start with a topic, launch the game, and come back to a notebook full of teachable moments.",
  "See mistakes, explanations, confidence, and next steps in one teacher view.",
  "Let students explore at game speed while you keep the learning path intentional.",
  "Use each play session to find the misconception hiding behind the answer.",
  "Create a mission in minutes and leave with sharper insight into tomorrow's class.",
  "Help students practice the why, then review Mia's notes when the room gets quiet.",
  "Track progress without losing the story behind each student's decision.",
  "Send the class into a mission and return to clear evidence for your next move.",
  "Turn practice into a feedback loop your students can play and you can trust.",
  "Meet every answer with a reason, then bring the useful patterns back to instruction.",
  "Keep the game light for students and the learning data serious for teachers.",
  "Reinforce the lesson, capture the thinking, and plan what comes next from Mia's notebook.",
] as const;

type TypewriterMode = "holding" | "deleting" | "typing";

const HERO_PHRASE_HOLD_MS = 2200;
const HERO_PHRASE_TYPE_DELAY_MS = 28;
const HERO_PHRASE_DELETE_DELAY_MS = 16;

export function LoginPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const nextPath = safeNextPath(searchParams.get("next"));
  const {
    error,
    signInWithGoogleIdToken,
    signInWithEmail,
    reportSignInError,
    status,
  } = useTeacherAuth();
  const [exchanging, setExchanging] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [apiAvailable, setApiAvailable] = useState(false);
  const [apiLoading, setApiLoading] = useState(true);
  const [apiError, setApiError] = useState<string | null>(null);
  const [heroPhraseIndex, setHeroPhraseIndex] = useState(0);
  const [heroPhraseLength, setHeroPhraseLength] = useState(
    LOGIN_HERO_PHRASES[0].length,
  );
  const [heroPhraseMode, setHeroPhraseMode] =
    useState<TypewriterMode>("holding");

  const heroPhrase = LOGIN_HERO_PHRASES[heroPhraseIndex];
  const typedHeroPhrase = heroPhrase.slice(0, heroPhraseLength);

  useEffect(() => {
    const controller = new AbortController();
    let active = true;
    const timeout = window.setTimeout(() => controller.abort(), 5000);

    void identityApi
      .get(controller.signal)
      .then(() => {
        if (!active) {
          return;
        }
        setApiAvailable(true);
        setApiError(null);
      })
      .catch(() => {
        if (!active) {
          return;
        }
        setApiAvailable(false);
        setApiError(
          "Couldn't reach the ClassGame API. Check that it is running and reload this page.",
        );
      })
      .finally(() => {
        window.clearTimeout(timeout);
        if (active) {
          setApiLoading(false);
        }
      });

    return () => {
      active = false;
      window.clearTimeout(timeout);
      controller.abort();
    };
  }, []);

  useEffect(() => {
    let timeout: number | undefined;

    if (heroPhraseMode === "holding") {
      timeout = window.setTimeout(() => {
        setHeroPhraseMode("deleting");
      }, HERO_PHRASE_HOLD_MS);
    } else if (heroPhraseMode === "deleting") {
      if (heroPhraseLength === 0) {
        timeout = window.setTimeout(() => {
          setHeroPhraseIndex(
            (index) => (index + 1) % LOGIN_HERO_PHRASES.length,
          );
          setHeroPhraseMode("typing");
        }, HERO_PHRASE_TYPE_DELAY_MS);
      } else {
        timeout = window.setTimeout(() => {
          setHeroPhraseLength((length) => Math.max(0, length - 1));
        }, HERO_PHRASE_DELETE_DELAY_MS);
      }
    } else if (heroPhraseLength >= heroPhrase.length) {
      timeout = window.setTimeout(() => {
        setHeroPhraseMode("holding");
      }, HERO_PHRASE_TYPE_DELAY_MS);
    } else {
      timeout = window.setTimeout(() => {
        setHeroPhraseLength((length) =>
          Math.min(heroPhrase.length, length + 1),
        );
      }, HERO_PHRASE_TYPE_DELAY_MS);
    }

    return () => {
      if (timeout) {
        window.clearTimeout(timeout);
      }
    };
  }, [heroPhrase.length, heroPhraseLength, heroPhraseMode]);

  useEffect(() => {
    if (status === "signed_in") {
      router.replace(nextPath);
    }
  }, [nextPath, router, status]);

  const handleCredential = useCallback(
    async (idToken: string) => {
      setExchanging(true);
      try {
        const user = await signInWithGoogleIdToken(idToken);
        if (user) {
          router.replace(nextPath);
        }
      } finally {
        setExchanging(false);
      }
    },
    [nextPath, router, signInWithGoogleIdToken],
  );

  const handleEmailSubmit = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (!apiAvailable || !email.trim() || !password) {
        return;
      }
      setExchanging(true);
      try {
        const user = await signInWithEmail(email, password);
        if (user) {
          router.replace(nextPath);
        }
      } finally {
        setExchanging(false);
      }
    },
    [apiAvailable, email, nextPath, password, router, signInWithEmail],
  );

  const handleLoadError = useCallback(() => {
    reportSignInError(
      "Couldn't load Google sign-in. Check your connection and reload the page.",
    );
  }, [reportSignInError]);

  const busy = exchanging || status === "checking";
  const apiUnavailable = apiLoading || !apiAvailable;
  const apiSignInDisabled = busy || apiUnavailable;
  const passwordSignInDisabled =
    apiSignInDisabled || !email.trim() || !password;
  let signInLabel = "Sign in";
  if (apiLoading) {
    signInLabel = "Checking API...";
  } else if (!apiAvailable) {
    signInLabel = "API unavailable";
  } else if (status === "checking") {
    signInLabel = "Checking your session...";
  } else if (exchanging) {
    signInLabel = "Signing in...";
  }

  return (
    <main className="flex min-h-svh flex-col justify-center bg-primary text-slate-950 lg:grid lg:grid-cols-2 lg:bg-white">
      {/* Brand half: solid primary, copy only. */}
      <section className="flex flex-col items-center gap-8 bg-primary px-6 pt-8 pb-6 text-center text-primary-foreground sm:px-10 lg:items-start lg:justify-between lg:gap-10 lg:rounded-r-[25%] lg:px-16 lg:py-14 lg:text-left">
        <p className="text-xs font-bold uppercase tracking-[0.2em] text-white/80">
          ClassGame
        </p>
        <div className="max-w-xl">
          <h1 className="text-3xl font-bold sm:text-4xl lg:text-6xl">
            Teacher dashboard
          </h1>
          <p
            className="mx-auto mt-4 min-h-[5.25rem] max-w-lg text-base leading-7 text-white/85 sm:min-h-16 sm:text-lg sm:leading-8 lg:mx-0"
            aria-label={heroPhrase}
          >
            <span aria-hidden="true">{typedHeroPhrase}</span>
            <span
              aria-hidden="true"
              className="ml-0.5 inline-block animate-pulse text-white"
            >
              |
            </span>
          </p>
        </div>
        <p className="hidden text-sm text-white/60 lg:block">
          Mia leads, asks why, takes notes. You read the notebook.
        </p>
      </section>

      {/* Sign-in half: a white card on the purple page on phones, the plain white half on desktop. */}
      <section className="flex justify-center px-4 pb-8 pt-2 text-center sm:px-10 lg:items-center lg:py-12 lg:text-left">
        <div className="w-full max-w-sm rounded-3xl bg-white p-6 text-slate-950 shadow-2xl shadow-black/20 sm:p-8 lg:rounded-none lg:bg-transparent lg:p-0 lg:shadow-none">
          <h2 className="text-2xl font-bold">Sign in</h2>
          <p className="mt-2 text-sm leading-6 text-slate-500">
            Use an allowed teacher account. Students enter from the game.
          </p>

          <div className="mt-6">
            <GoogleIdentityButton
              onCredential={handleCredential}
              onLoadError={handleLoadError}
              disabled={apiSignInDisabled}
            />
          </div>

          <div className="my-5 flex items-center gap-3 text-xs font-semibold uppercase tracking-wide text-slate-400">
            <span className="h-px flex-1 bg-slate-200" />
            or
            <span className="h-px flex-1 bg-slate-200" />
          </div>

          <form className="space-y-3 text-left" onSubmit={handleEmailSubmit}>
            <p className="text-xs leading-5 text-slate-500">
              Use the teacher credentials shared with you.
            </p>
            <label className="block">
              <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                Email
              </span>
              <input
                type="email"
                name="email"
                autoComplete="username"
                inputMode="email"
                spellCheck={false}
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
                maxLength={INPUT_LIMITS.email}
                className="mt-1 h-11 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </label>
            <label className="block">
              <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                Password
              </span>
              <input
                type="password"
                name="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
                maxLength={INPUT_LIMITS.password}
                className="mt-1 h-11 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </label>
            <button
              type="submit"
              disabled={passwordSignInDisabled}
              className="h-11 w-full rounded-md bg-primary text-sm font-bold text-primary-foreground shadow-sm hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {signInLabel}
            </button>
          </form>

          {apiError ? (
            <p
              role="alert"
              className="mt-4 rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm font-medium text-rose-700"
            >
              {apiError}
            </p>
          ) : null}

          {error ? (
            <p className="mt-4 rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm font-medium text-rose-700">
              {error}
            </p>
          ) : null}
        </div>
      </section>
    </main>
  );
}

function safeNextPath(value: string | null) {
  if (!value || !value.startsWith("/") || value.startsWith("//")) {
    return "/dashboard";
  }

  return value;
}
