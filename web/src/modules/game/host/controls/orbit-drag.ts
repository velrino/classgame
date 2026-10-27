const DRAG_THRESHOLD_PX = 8;
const TAP_MAX_MS = 350;
const YAW_PER_PX = 0.0065;

/**
 * One pointer on the canvas does two things: a short still press is a tap
 * (walk to what you touched), a drag swings the camera around the avatar.
 */
export function createOrbitDrag(
  canvas: HTMLCanvasElement,
  onTap: (clientX: number, clientY: number) => void,
) {
  let yawDelta = 0;
  let activePointerId: number | null = null;
  let dragging = false;
  let startX = 0;
  let startY = 0;
  let startTime = 0;
  let lastX = 0;

  function handlePointerDown(event: PointerEvent) {
    if (
      activePointerId !== null ||
      (event.pointerType === "mouse" && event.button !== 0)
    ) {
      return;
    }

    activePointerId = event.pointerId;
    dragging = false;
    startX = lastX = event.clientX;
    startY = event.clientY;
    startTime = event.timeStamp;
    canvas.setPointerCapture(event.pointerId);
  }

  function handlePointerMove(event: PointerEvent) {
    if (activePointerId !== event.pointerId) {
      return;
    }

    if (
      !dragging &&
      Math.hypot(event.clientX - startX, event.clientY - startY) >
        DRAG_THRESHOLD_PX
    ) {
      dragging = true;
    }

    if (dragging) {
      yawDelta -= (event.clientX - lastX) * YAW_PER_PX;
    }

    lastX = event.clientX;
  }

  function handlePointerUp(event: PointerEvent) {
    if (activePointerId !== event.pointerId) {
      return;
    }

    if (canvas.hasPointerCapture(event.pointerId)) {
      canvas.releasePointerCapture(event.pointerId);
    }

    if (!dragging && event.timeStamp - startTime <= TAP_MAX_MS) {
      onTap(event.clientX, event.clientY);
    }

    activePointerId = null;
    dragging = false;
  }

  canvas.addEventListener("pointerdown", handlePointerDown);
  canvas.addEventListener("pointermove", handlePointerMove);
  canvas.addEventListener("pointerup", handlePointerUp);
  canvas.addEventListener("pointercancel", handlePointerUp);

  return {
    consumeYawDelta() {
      const delta = yawDelta;
      yawDelta = 0;
      return delta;
    },
    isDragging() {
      return dragging;
    },
    dispose() {
      canvas.removeEventListener("pointerdown", handlePointerDown);
      canvas.removeEventListener("pointermove", handlePointerMove);
      canvas.removeEventListener("pointerup", handlePointerUp);
      canvas.removeEventListener("pointercancel", handlePointerUp);
    },
  };
}
