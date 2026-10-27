"use client";

import { useEffect, useRef, useState } from "react";

import {
  GOOGLE_CLIENT_ID,
  loadGoogleIdentity,
} from "@/modules/identity/adapters/google-identity";

type GoogleIdentityButtonProps = {
  /** Called with the Google ID token once the teacher picks an account. */
  readonly onCredential: (idToken: string) => void;
  readonly onLoadError?: (error: Error) => void;
  /** Blocks clicks (e.g. while a token is being exchanged) without re-rendering Google's button. */
  readonly disabled?: boolean;
};

/** Google's button accepts widths in this range. */
const MIN_WIDTH = 200;
const MAX_WIDTH = 400;

/**
 * The official Google button, rendered by Google Identity Services into a slot
 * we size. Google owns the look and the account chooser; we only get the token.
 */
export function GoogleIdentityButton({
  onCredential,
  onLoadError,
  disabled = false,
}: GoogleIdentityButtonProps) {
  const slotRef = useRef<HTMLDivElement>(null);
  const onCredentialRef = useRef(onCredential);
  const onLoadErrorRef = useRef(onLoadError);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    onCredentialRef.current = onCredential;
    onLoadErrorRef.current = onLoadError;
  }, [onCredential, onLoadError]);

  useEffect(() => {
    const slot = slotRef.current;
    if (!slot) {
      return;
    }

    let cancelled = false;

    loadGoogleIdentity()
      .then((google) => {
        if (cancelled) {
          return;
        }

        google.initialize({
          client_id: GOOGLE_CLIENT_ID,
          callback: (response) => onCredentialRef.current(response.credential),
          auto_select: false,
          itp_support: true,
          ux_mode: "popup",
        });

        const width = Math.round(
          Math.min(
            MAX_WIDTH,
            Math.max(MIN_WIDTH, slot.clientWidth || MAX_WIDTH),
          ),
        );
        // React never renders into the slot, so only Google's own markup is ever in here.
        slot.replaceChildren();
        google.renderButton(slot, {
          type: "standard",
          theme: "outline",
          size: "large",
          text: "continue_with",
          shape: "rectangular",
          logo_alignment: "left",
          width,
        });
        setReady(true);
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          onLoadErrorRef.current?.(
            error instanceof Error ? error : new Error(String(error)),
          );
        }
      });

    return () => {
      cancelled = true;
      slot.replaceChildren();
    };
  }, []);

  return (
    <div
      className={disabled ? "pointer-events-none opacity-60" : undefined}
      aria-busy={disabled}
    >
      {/* The skeleton is a sibling, not a child: Google owns everything inside the slot. */}
      <div className="relative flex min-h-10 w-full justify-center">
        {!ready ? (
          <div className="absolute inset-0 mx-auto h-10 w-full max-w-[400px] animate-pulse rounded border border-slate-200 bg-slate-50" />
        ) : null}
        <div ref={slotRef} className="flex w-full justify-center" />
      </div>
    </div>
  );
}
