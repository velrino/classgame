import { HttpMethod, type HttpRequest } from "./types";

const LOCAL_API_BASE_URL = "http://localhost:4004";

export const CLASSGAME_API_BASE_URL =
  process.env.NEXT_PUBLIC_CLASSGAME_API_BASE_URL?.trim() || LOCAL_API_BASE_URL;

export const httpClient = {
  async request(parameters: HttpRequest): Promise<Response> {
    let url = `${CLASSGAME_API_BASE_URL}${parameters.path}`;
    const headers = new Headers(parameters.headers);
    const suppliedRequestId = headers.get("X-Request-ID")?.trim();
    headers.set(
      "X-Request-ID",
      suppliedRequestId && /^[A-Za-z0-9._-]{1,128}$/.test(suppliedRequestId)
        ? suppliedRequestId
        : (globalThis.crypto?.randomUUID?.() ??
            `request-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`),
    );
    // A caller may propagate its W3C traceparent; parsing and trust policy belong to the API.

    if (
      !headers.has("Content-Type") &&
      !(parameters.data instanceof FormData)
    ) {
      headers.set("Content-Type", "application/json");
    }

    if (parameters.accessToken) {
      headers.set("Authorization", `Bearer ${parameters.accessToken}`);
    }

    const options: RequestInit = {
      headers,
      keepalive: parameters.keepalive,
      method: parameters.method,
      signal: parameters.signal,
    };

    if (parameters.method === HttpMethod.GET && parameters.data) {
      const queryString = stringifyParams(
        parameters.data as Record<string, unknown>,
      );
      url = `${url}?${queryString}`;
    } else if (parameters.data) {
      options.body =
        parameters.data instanceof FormData
          ? parameters.data
          : JSON.stringify(parameters.data);
    }

    return fetch(url, options);
  },
};

const stringifyParams = (params: Record<string, unknown>) => {
  const searchParams = new URLSearchParams();

  Object.entries(params).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      value.forEach((val, index) => {
        searchParams.append(`${key}[${index}]`, String(val));
      });
    } else if (value !== undefined && value !== null) {
      searchParams.append(key, String(value));
    }
  });

  return searchParams.toString();
};
