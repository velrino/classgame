"use client";

import { useCallback, useEffect, useRef, type PointerEvent } from "react";

import type {
  MovementRef,
  MovementState,
} from "@/modules/game/adapters/types/world";

const joystickHideDelay = 1000;
const joystickKnobMaxDistance = 46;
const joystickKnobResistanceStart = 20;
const joystickFullMovementDistance = 68;

type Origin = { readonly clientX: number; readonly clientY: number };

function setMovement(movementRef: MovementRef, movement: MovementState) {
  movementRef.current = movement;
}

function applyDeadzone(value: number) {
  return Math.abs(value) < 0.16 ? 0 : value;
}

function getElasticKnobDistance(distance: number) {
  if (distance <= joystickKnobResistanceStart) {
    return distance;
  }

  return Math.min(
    joystickKnobMaxDistance,
    joystickKnobResistanceStart +
      Math.log1p(distance - joystickKnobResistanceStart) * 7,
  );
}

function getPadVector(event: PointerEvent<HTMLDivElement>, origin: Origin) {
  const rawX = event.clientX - origin.clientX;
  const rawY = event.clientY - origin.clientY;
  const distance = Math.hypot(rawX, rawY);

  if (distance === 0) {
    return { knobX: 0, knobY: 0, movementX: 0, movementY: 0 };
  }

  const strength = Math.min(distance / joystickFullMovementDistance, 1);
  const knobDistance = getElasticKnobDistance(distance);

  return {
    knobX: (rawX / distance) * knobDistance,
    knobY: (rawY / distance) * knobDistance,
    movementX: (rawX / distance) * strength,
    movementY: (rawY / distance) * strength,
  };
}

/**
 * Floating touch joystick. Writes straight into a ref so dragging never
 * re-renders React; the 3D loop reads the ref each frame.
 */
export function MovementPad({
  movementRef,
}: {
  readonly movementRef: MovementRef;
}) {
  const padRef = useRef<HTMLDivElement | null>(null);
  const activePointerIdRef = useRef<number | null>(null);
  const hideTimerRef = useRef<number | null>(null);
  const originRef = useRef<Origin | null>(null);

  const setStyle = useCallback((name: string, value: string) => {
    padRef.current?.style.setProperty(name, value);
  }, []);

  const clearHideTimer = useCallback(() => {
    if (hideTimerRef.current !== null) {
      window.clearTimeout(hideTimerRef.current);
      hideTimerRef.current = null;
    }
  }, []);

  const release = useCallback(
    (pointerId?: number) => {
      if (
        pointerId !== undefined &&
        activePointerIdRef.current !== null &&
        activePointerIdRef.current !== pointerId
      ) {
        return;
      }

      activePointerIdRef.current = null;
      originRef.current = null;
      setStyle("--pad-x", "0px");
      setStyle("--pad-y", "0px");
      setMovement(movementRef, { forward: 0, right: 0 });
      clearHideTimer();
      hideTimerRef.current = window.setTimeout(() => {
        hideTimerRef.current = null;
        setStyle("--pad-opacity", "0");
      }, joystickHideDelay);
    },
    [clearHideTimer, movementRef, setStyle],
  );

  useEffect(() => {
    const onWindowEnd = (event: globalThis.PointerEvent) =>
      release(event.pointerId);
    const onBlur = () => release();

    window.addEventListener("blur", onBlur);
    window.addEventListener("pointercancel", onWindowEnd);
    window.addEventListener("pointerup", onWindowEnd);

    return () => {
      window.removeEventListener("blur", onBlur);
      window.removeEventListener("pointercancel", onWindowEnd);
      window.removeEventListener("pointerup", onWindowEnd);
      clearHideTimer();
      setMovement(movementRef, { forward: 0, right: 0 });
    };
  }, [clearHideTimer, movementRef, release]);

  function update(event: PointerEvent<HTMLDivElement>) {
    event.preventDefault();
    const origin = originRef.current;

    if (!origin) {
      return;
    }

    const vector = getPadVector(event, origin);
    clearHideTimer();
    setStyle("--pad-opacity", "0.55");
    setStyle("--pad-x", `${vector.knobX}px`);
    setStyle("--pad-y", `${vector.knobY}px`);
    setMovement(movementRef, {
      forward: applyDeadzone(-vector.movementY),
      right: applyDeadzone(vector.movementX),
    });
  }

  function end(event?: PointerEvent<HTMLDivElement>) {
    if (event && event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }

    release(event?.pointerId);
  }

  return (
    <div
      ref={padRef}
      aria-label="Movement"
      className="relative h-full w-full touch-none select-none [-webkit-tap-highlight-color:transparent]"
      role="application"
      onLostPointerCapture={end}
      onPointerCancel={end}
      onPointerDown={(event) => {
        event.preventDefault();

        if (activePointerIdRef.current !== null) {
          return;
        }

        const rect = event.currentTarget.getBoundingClientRect();
        activePointerIdRef.current = event.pointerId;
        originRef.current = { clientX: event.clientX, clientY: event.clientY };
        setStyle("--pad-origin-x", `${event.clientX - rect.left}px`);
        setStyle("--pad-origin-y", `${event.clientY - rect.top}px`);
        event.currentTarget.setPointerCapture(event.pointerId);
        update(event);
      }}
      onPointerMove={(event) => {
        if (activePointerIdRef.current === event.pointerId) {
          update(event);
        }
      }}
      onPointerUp={end}
    >
      <div
        className="absolute size-[60px] rounded-full border border-white/15 bg-slate-950 shadow-xl shadow-black/50 transition-[opacity,transform] duration-100"
        style={{
          left: "var(--pad-origin-x, 50%)",
          opacity: "var(--pad-opacity, 0)",
          top: "var(--pad-origin-y, 50%)",
          transform:
            "translate(calc(-50% + var(--pad-x, 0px)), calc(-50% + var(--pad-y, 0px)))",
        }}
      />
    </div>
  );
}
