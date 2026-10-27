export class ApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly requestId?: string;
  readonly traceparent?: string;
  readonly retryAfterSeconds?: number;

  constructor(
    status: number,
    code: string,
    message: string,
    correlation?: {
      requestId?: string;
      traceparent?: string;
      retryAfterSeconds?: number;
    },
  ) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.requestId = correlation?.requestId;
    this.traceparent = correlation?.traceparent;
    this.retryAfterSeconds = correlation?.retryAfterSeconds;
  }
}

export type ApiErrorBody = {
  code?: string;
  message?: string;
  error?: { code?: string; message?: string };
};

/** Builds an ApiError from a failed response; `fallbackCode` is used when the body has no code. */
export async function apiError(
  res: Response,
  fallbackCode: string,
): Promise<ApiError> {
  let body: ApiErrorBody = {};
  try {
    body = (await res.json()) as ApiErrorBody;
  } catch {
    // Non-JSON error body (proxy, network). Fall back to the status text.
  }

  const code = body.error?.code ?? body.code ?? fallbackCode;
  const message =
    body.error?.message || body.message || res.statusText || fallbackCode;
  return new ApiError(res.status, code, message, {
    retryAfterSeconds: /^\d+$/.test(res.headers.get("Retry-After") ?? "")
      ? Math.min(86400, Number(res.headers.get("Retry-After")))
      : undefined,
    requestId: res.headers.get("X-Request-ID") || undefined,
    traceparent: res.headers.get("traceparent") || undefined,
  });
}
