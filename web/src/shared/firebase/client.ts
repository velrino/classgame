import { getAnalytics, isSupported, type Analytics } from "firebase/analytics";
import { getApps, initializeApp, type FirebaseApp, type FirebaseOptions } from "firebase/app";
import { connectAuthEmulator, getAuth, type Auth } from "firebase/auth";
import { connectFirestoreEmulator, getFirestore, type Firestore } from "firebase/firestore";

const DEFAULT_FIREBASE_APP_NAME = "[DEFAULT]";
const STUDENT_FIREBASE_APP_NAME = "classgame-student";

const requiredEnv = {
  apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY,
  authDomain: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN,
  projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID,
  storageBucket: process.env.NEXT_PUBLIC_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: process.env.NEXT_PUBLIC_FIREBASE_MESSAGING_SENDER_ID,
  appId: process.env.NEXT_PUBLIC_FIREBASE_APP_ID,
} satisfies FirebaseOptions;

const optionalEnv = {
  measurementId: process.env.NEXT_PUBLIC_FIREBASE_MEASUREMENT_ID,
} satisfies Pick<FirebaseOptions, "measurementId">;

type EmulatorSettings = { auth: URL; firestore: URL };
const emulatorFlag = process.env.NEXT_PUBLIC_FIREBASE_EMULATORS_ENABLED;
if (emulatorFlag === "true" && process.env.NODE_ENV === "production")
  throw new Error("Firebase emulator mode cannot be built or run in production.");
const authEmulatorHost = process.env.NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST;
const firestoreEmulatorHost = process.env.NEXT_PUBLIC_FIRESTORE_EMULATOR_HOST;
const emulatorCache = globalThis as typeof globalThis & { __classgameEmulatorClients?: WeakSet<object> };
const connectedEmulators = emulatorCache.__classgameEmulatorClients ??= new WeakSet<object>();

function emulatorSettings(): EmulatorSettings | null {
  if (emulatorFlag !== "true") {
    if ((emulatorFlag && emulatorFlag !== "false") || authEmulatorHost || firestoreEmulatorHost)
      throw new Error("Firebase emulator hosts require explicit emulator mode.");
    return null;
  }
  if (process.env.NODE_ENV !== "development" || !/^demo-[a-z0-9][a-z0-9-]{0,50}$/.test(requiredEnv.projectId ?? ""))
    throw new Error("Firebase emulators require development and a demo- project.");
  if (typeof window === "undefined" || !["localhost", "127.0.0.1", "[::1]"].includes(window.location.hostname))
    throw new Error("Firebase emulators require a local browser origin.");
  return { auth: emulatorURL(authEmulatorHost), firestore: emulatorURL(firestoreEmulatorHost) };
}
function emulatorURL(value: string | undefined): URL {
  if (!value || !/^(127(?:\.[0-9]{1,3}){3}|\[::1\]):[0-9]+$/.test(value))
    throw new Error("Both Firebase emulators need numeric loopback host:port settings.");
  const parsed = new URL("http://" + value);
  const port = Number(parsed.port);
  if (!Number.isSafeInteger(port) || port < 1 || port > 65535 || (parsed.hostname !== "[::1]" && !parsed.hostname.startsWith("127.")))
    throw new Error("Invalid Firebase emulator endpoint.");
  return parsed;
}
function localFirestore(app: FirebaseApp): Firestore {
  const settings = emulatorSettings();
  const db = getFirestore(app);
  if (settings && !connectedEmulators.has(db)) {
    connectFirestoreEmulator(db, settings.firestore.hostname.replace(/^\[|\]$/g, ""), Number(settings.firestore.port));
    connectedEmulators.add(db);
  }
  return db;
}
function localAuth(app: FirebaseApp): Auth {
  const settings = emulatorSettings();
  const client = getAuth(app);
  if (settings && !connectedEmulators.has(client)) {
    connectAuthEmulator(client, settings.auth.origin);
    connectedEmulators.add(client);
  }
  return client;
}

let firestore: Firestore | null = null;
let studentFirestore: Firestore | null = null;
let auth: Auth | null = null;
let studentAuth: Auth | null = null;
let analyticsPromise: Promise<Analytics | null> | null = null;

export function isFirebaseConfigured() {
  return Object.values(requiredEnv).every((value) => typeof value === "string" && value.length > 0);
}

function getFirebaseConfig(): FirebaseOptions {
  if (!isFirebaseConfigured()) {
    throw new Error("Firebase is not configured. Set NEXT_PUBLIC_FIREBASE_* environment variables.");
  }

  return {
    ...requiredEnv,
    ...(optionalEnv.measurementId ? optionalEnv : {}),
  };
}

export function getFirebaseApp(): FirebaseApp {
  const existingApp = getApps().find(
    (app) => app.name === DEFAULT_FIREBASE_APP_NAME,
  );

  return existingApp ?? initializeApp(getFirebaseConfig());
}

export function getStudentFirebaseApp(): FirebaseApp {
  const existingApp = getApps().find(
    (app) => app.name === STUDENT_FIREBASE_APP_NAME,
  );

  return existingApp ?? initializeApp(getFirebaseConfig(), STUDENT_FIREBASE_APP_NAME);
}

export function getFirebaseDb(): Firestore {
  firestore ??= localFirestore(getFirebaseApp());
  return firestore;
}

export function getStudentFirebaseDb(): Firestore {
  studentFirestore ??= localFirestore(getStudentFirebaseApp());
  return studentFirestore;
}

export function getFirebaseAuth(): Auth {
  auth ??= localAuth(getFirebaseApp());
  return auth;
}

export function getStudentFirebaseAuth(): Auth {
  studentAuth ??= localAuth(getStudentFirebaseApp());
  return studentAuth;
}

export function getFirebaseAnalytics(): Promise<Analytics | null> {
  if (typeof window === "undefined" || !isFirebaseConfigured() || emulatorFlag === "true") {
    return Promise.resolve(null);
  }

  analyticsPromise ??= isSupported()
    .then((supported) => (supported ? getAnalytics(getFirebaseApp()) : null))
    .catch(() => null);

  return analyticsPromise;
}
