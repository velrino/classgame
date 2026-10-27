export enum HttpMethod {
  GET = "GET",
  DELETE = "DELETE",
  POST = "POST",
  PUT = "PUT",
}

export interface HttpRequest {
  method: HttpMethod;
  path: string;
  data?: unknown;
  headers?: Headers;
  accessToken?: string;
  keepalive?: boolean;
  signal?: AbortSignal;
}
