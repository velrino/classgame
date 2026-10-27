import type { MissionAttachmentDto } from "@/modules/missions/adapters/types/contracts";

export type AttachmentKind = "docx" | "file" | "image" | "pdf" | "txt";

export const attachmentKindLabel: Record<AttachmentKind, string> = {
  docx: "DOCX",
  file: "FILE",
  image: "IMAGE",
  pdf: "PDF",
  txt: "TXT",
};

export const textPreviewLimit = 200_000;

/** Temporary upload previews use only the allowlisted CDN; stored objects require API authorization. */
export function attachmentOriginalURL(attachment: MissionAttachmentDto) {
  const value =
    attachment.uploadcare_cdn_url?.trim() || attachment.cdn_url.trim();
  try {
    const url = new URL(value);
    if (
      url.protocol !== "https:" ||
      url.username ||
      url.password ||
      (url.port && url.port !== "443")
    )
      return "";
    const host = url.hostname.toLowerCase();
    return ["ucarecdn.com", "ucarecd.net"].some(
      (suffix) => host === suffix || host.endsWith("." + suffix),
    )
      ? url.href
      : "";
  } catch {
    return "";
  }
}

export function getAttachmentKind(
  attachment: MissionAttachmentDto,
): AttachmentKind {
  const mimeType = attachment.mime_type.toLocaleLowerCase();
  const fileName = attachment.file_name.toLocaleLowerCase();

  if (mimeType === "application/pdf" || fileName.endsWith(".pdf")) {
    return "pdf";
  }

  if (
    mimeType === "image/png" ||
    mimeType === "image/jpeg" ||
    fileName.endsWith(".png") ||
    fileName.endsWith(".jpg") ||
    fileName.endsWith(".jpeg")
  ) {
    return "image";
  }

  if (
    mimeType ===
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
    fileName.endsWith(".docx")
  ) {
    return "docx";
  }

  if (mimeType === "text/plain" || fileName.endsWith(".txt")) {
    return "txt";
  }

  return "file";
}

export function formatFileSize(sizeBytes: number) {
  if (!Number.isFinite(sizeBytes) || sizeBytes <= 0) {
    return "Unknown size";
  }

  const units = ["B", "KB", "MB"];
  let value = sizeBytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const digits = value >= 10 || unitIndex === 0 ? 0 : 1;
  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}
