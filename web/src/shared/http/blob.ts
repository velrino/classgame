/** Enforce the actual byte limit even when Content-Length is missing or incorrect. */
export async function readBoundedBlob(
  response: Response,
  maximumBytes: number,
): Promise<Blob> {
  if (Number(response.headers.get("Content-Length")) > maximumBytes)
    throw new Error("Attachment exceeds the download limit.");
  const reader = response.body?.getReader();
  if (!reader) throw new Error("Attachment response is empty.");
  const chunks: Uint8Array<ArrayBuffer>[] = [];
  let bytes = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      bytes += value.byteLength;
      if (bytes > maximumBytes)
        throw new Error("Attachment exceeds the download limit.");
      chunks.push(Uint8Array.from(value));
    }
  } catch (error) {
    await reader.cancel().catch(() => undefined);
    throw error;
  } finally {
    reader.releaseLock();
  }
  return new Blob(chunks, {
    type: response.headers.get("Content-Type") ?? "application/octet-stream",
  });
}
