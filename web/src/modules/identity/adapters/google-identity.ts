/**
 * Google Identity Services (GIS): the official "Sign in with Google" button.
 * The button is rendered by Google's script and hands us an ID token, which
 * `useTeacherAuth` exchanges for a Firebase session. Cancelling the account
 * chooser simply never calls back, so there is no dangling "signing in" state.
 */

/** The Firebase project's web OAuth client (public identifier, not a secret). */
export const GOOGLE_CLIENT_ID =
  "245584783597-t05j7fj2iucesiad4jpu4mlh72kgcs12.apps.googleusercontent.com";

const GIS_SCRIPT_URL = "https://accounts.google.com/gsi/client";

export type GoogleCredentialResponse = {
  readonly credential: string;
  readonly select_by?: string;
  readonly clientId?: string;
};

export type GoogleButtonOptions = {
  readonly type?: "standard" | "icon";
  readonly theme?: "outline" | "filled_blue" | "filled_black";
  readonly size?: "large" | "medium" | "small";
  readonly text?: "signin_with" | "signup_with" | "continue_with" | "signin";
  readonly shape?: "rectangular" | "pill" | "circle" | "square";
  readonly logo_alignment?: "left" | "center";
  readonly width?: number;
  readonly locale?: string;
};

export type GoogleAccountsId = {
  initialize(config: {
    client_id: string;
    callback: (response: GoogleCredentialResponse) => void;
    auto_select?: boolean;
    cancel_on_tap_outside?: boolean;
    itp_support?: boolean;
    ux_mode?: "popup" | "redirect";
  }): void;
  renderButton(parent: HTMLElement, options: GoogleButtonOptions): void;
  disableAutoSelect(): void;
};

declare global {
  interface Window {
    google?: { accounts?: { id?: GoogleAccountsId } };
  }
}

let scriptPromise: Promise<GoogleAccountsId> | null = null;

/** Load Google's script once and resolve with `google.accounts.id`. */
export function loadGoogleIdentity() {
  scriptPromise ??= new Promise<GoogleAccountsId>((resolve, reject) => {
    const existing = window.google?.accounts?.id;
    if (existing) {
      resolve(existing);
      return;
    }

    const script = document.createElement("script");
    script.src = GIS_SCRIPT_URL;
    script.async = true;
    script.defer = true;
    script.onload = () => {
      const api = window.google?.accounts?.id;
      if (api) {
        resolve(api);
      } else {
        reject(
          new Error(
            "Google Identity Services loaded without google.accounts.id",
          ),
        );
      }
    };
    script.onerror = () => {
      scriptPromise = null;
      reject(new Error("Failed to load Google Identity Services"));
    };
    document.head.appendChild(script);
  });

  return scriptPromise;
}
