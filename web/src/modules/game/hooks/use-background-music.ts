"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";

export const BACKGROUND_MUSIC = {
  src: "/assets/audio/lockers-left-open.mp3",
  title: "Lockers Left Open",
  genre: "Lo-Fi Hip Hop",
  credit: "Generated via Google Lyria AI (Google DeepMind)",
};

const STORAGE_KEY = "classgame.background-music.v1";
const CHANGE_EVENT = "classgame-background-music-change";
const DEFAULT_ENABLED = true;
const VOLUME = 0.32;

let fallbackEnabled = DEFAULT_ENABLED;

export function readBackgroundMusicEnabled() {
  if (typeof window === "undefined") {
    return DEFAULT_ENABLED;
  }

  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored === null) {
      return fallbackEnabled;
    }

    fallbackEnabled = stored !== "off";
    return fallbackEnabled;
  } catch {
    return fallbackEnabled;
  }
}

export function storeBackgroundMusicEnabled(enabled: boolean) {
  fallbackEnabled = enabled;

  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(STORAGE_KEY, enabled ? "on" : "off");
  } catch {
    // Storage is optional; the in-memory preference still updates this tab.
  }

  window.dispatchEvent(new Event(CHANGE_EVENT));
}

export function subscribeBackgroundMusicEnabled(onStoreChange: () => void) {
  if (typeof window === "undefined") {
    return () => {};
  }

  const handleStorage = (event: StorageEvent) => {
    if (event.key === STORAGE_KEY) {
      onStoreChange();
    }
  };

  window.addEventListener("storage", handleStorage);
  window.addEventListener(CHANGE_EVENT, onStoreChange);

  return () => {
    window.removeEventListener("storage", handleStorage);
    window.removeEventListener(CHANGE_EVENT, onStoreChange);
  };
}

export function useBackgroundMusic(active: boolean) {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const [waitingForGesture, setWaitingForGesture] = useState(false);
  const enabled = useSyncExternalStore(
    subscribeBackgroundMusicEnabled,
    readBackgroundMusicEnabled,
    () => DEFAULT_ENABLED,
  );

  const ensureAudio = useCallback(() => {
    if (typeof Audio === "undefined") {
      return null;
    }

    if (audioRef.current) {
      return audioRef.current;
    }

    const audio = new Audio(BACKGROUND_MUSIC.src);
    audio.loop = true;
    audio.preload = "auto";
    audio.volume = VOLUME;
    audioRef.current = audio;

    return audio;
  }, []);

  useEffect(() => {
    return () => {
      audioRef.current?.pause();
      audioRef.current = null;
    };
  }, []);

  const playAudio = useCallback(async () => {
    if (!active || document.visibilityState === "hidden") {
      return false;
    }

    const audio = ensureAudio();
    if (!audio) {
      return false;
    }

    try {
      audio.volume = VOLUME;
      await audio.play();
      return true;
    } catch {
      return false;
    }
  }, [active, ensureAudio]);

  const setEnabled = useCallback(
    (nextEnabled: boolean) => {
      storeBackgroundMusicEnabled(nextEnabled);

      const audio = audioRef.current;
      if (!nextEnabled) {
        audio?.pause();
        setWaitingForGesture(false);
        return;
      }

      void playAudio().then((started) => setWaitingForGesture(!started));
    },
    [playAudio],
  );

  useEffect(() => {
    const audio = audioRef.current;

    if (!active || !enabled) {
      audio?.pause();
      return;
    }

    void playAudio();
  }, [active, enabled, playAudio]);

  useEffect(() => {
    if (!active || !enabled) {
      return;
    }

    const unlockAudio = () => {
      void playAudio().then((started) => {
        if (started) {
          setWaitingForGesture(false);
        }
      });
    };

    window.addEventListener("pointerdown", unlockAudio, true);
    window.addEventListener("keydown", unlockAudio, true);

    return () => {
      window.removeEventListener("pointerdown", unlockAudio, true);
      window.removeEventListener("keydown", unlockAudio, true);
    };
  }, [active, enabled, playAudio]);

  useEffect(() => {
    if (!active || !enabled) {
      return;
    }

    const syncVisibility = () => {
      if (document.visibilityState === "hidden") {
        audioRef.current?.pause();
      } else {
        void playAudio().then((started) => {
          if (started) {
            setWaitingForGesture(false);
          }
        });
      }
    };

    document.addEventListener("visibilitychange", syncVisibility);
    return () =>
      document.removeEventListener("visibilitychange", syncVisibility);
  }, [active, enabled, playAudio]);

  return {
    musicEnabled: enabled,
    musicWaitingForGesture: waitingForGesture,
    setMusicEnabled: setEnabled,
  };
}
