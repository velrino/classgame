"use client";

import { Button } from "@/components/ui/button";
import { Modal } from "@/components/ui/modal";
import { useAttachmentResource } from "@/modules/missions/adapters/hooks/use-attachment-resource";
import type { AttachmentAccess } from "@/modules/missions/adapters/types/attachment-access";
import type { MissionAttachmentDto } from "@/modules/missions/adapters/types/contracts";
import {
  attachmentKindLabel,
  formatFileSize,
  getAttachmentKind,
  textPreviewLimit,
} from "@/modules/missions/domain/attachments";
import { Download, File, FileText, FileType2, X } from "lucide-react";
import { useEffect, useState } from "react";

type AttachmentProps = {
  readonly attachment: MissionAttachmentDto;
  readonly access?: AttachmentAccess;
};

export function UploadedMissionFilesGrid({
  attachments,
  disabled,
  onPreview,
  onRemove,
}: {
  readonly attachments: MissionAttachmentDto[];
  readonly disabled: boolean;
  readonly onPreview: (attachment: MissionAttachmentDto) => void;
  readonly onRemove: (uuid: string) => void;
}) {
  return (
    <ul className="mt-3 grid gap-2 sm:grid-cols-2">
      {attachments.map((attachment) => {
        const kind = getAttachmentKind(attachment);
        const Icon =
          kind === "pdf" ? FileType2 : kind === "txt" ? FileText : File;
        return (
          <li
            key={attachment.uploadcare_uuid}
            className="flex min-w-0 items-center gap-2 rounded-lg border border-border bg-muted/20 p-2"
          >
            <button
              type="button"
              className="flex min-w-0 flex-1 items-center gap-3 text-left"
              onClick={() => onPreview(attachment)}
            >
              <span className="grid size-10 shrink-0 place-items-center rounded-md bg-primary/10 text-primary">
                <Icon className="size-5" aria-hidden />
              </span>
              <span className="min-w-0">
                <span className="block truncate text-sm font-medium">
                  {attachment.file_name}
                </span>
                <span className="text-xs text-muted-foreground">
                  {attachmentKindLabel[kind]} ·{" "}
                  {formatFileSize(attachment.size_bytes)}
                </span>
              </span>
            </button>
            <button
              type="button"
              disabled={disabled}
              onClick={() => onRemove(attachment.uploadcare_uuid)}
              aria-label={`Remove ${attachment.file_name}`}
              className="rounded p-1 text-muted-foreground hover:text-foreground disabled:opacity-50"
            >
              <X className="size-4" aria-hidden />
            </button>
          </li>
        );
      })}
    </ul>
  );
}

export function AttachmentPreviewModal({
  attachment,
  access,
  onOpenChange,
}: {
  readonly attachment: MissionAttachmentDto | null;
  readonly access?: AttachmentAccess;
  readonly onOpenChange: (open: boolean) => void;
}) {
  if (!attachment) return null;
  const kind = getAttachmentKind(attachment);
  return (
    <Modal
      open
      onOpenChange={onOpenChange}
      title={attachment.file_name}
      description={`${attachmentKindLabel[kind]} · ${formatFileSize(attachment.size_bytes)}`}
      size="xl"
    >
      <AttachmentPreview
        key={attachment.uploadcare_uuid}
        attachment={attachment}
        access={access}
      />
    </Modal>
  );
}

function AttachmentPreview({ attachment, access }: AttachmentProps) {
  const resource = useAttachmentResource(attachment, access);
  const kind = getAttachmentKind(attachment);
  if (resource.loading)
    return (
      <p
        role="status"
        className="grid min-h-52 place-items-center text-sm text-muted-foreground"
      >
        Loading preview...
      </p>
    );
  if (resource.error)
    return (
      <p
        role="alert"
        className="rounded-md border border-border p-6 text-sm text-muted-foreground"
      >
        {resource.error}
      </p>
    );
  return (
    <div className="grid gap-3">
      {kind === "image" ? (
        <div className="grid max-h-[70svh] place-items-center overflow-auto rounded-md border border-border bg-muted/30 p-3">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={resource.url}
            alt={attachment.file_name}
            className="max-h-[calc(70svh-1.5rem)] max-w-full rounded object-contain"
          />
        </div>
      ) : kind === "pdf" ? (
        <iframe
          title={attachment.file_name}
          src={resource.url}
          sandbox="allow-same-origin"
          className="h-[70svh] w-full rounded-md border border-border bg-white"
        />
      ) : kind === "txt" ? (
        <TextPreview url={resource.url} />
      ) : (
        <p className="rounded-md border border-border bg-muted/30 p-6 text-sm text-muted-foreground">
          Download this file to open it in a compatible application.
        </p>
      )}
      <div className="flex justify-end">
        <Button variant="outline" asChild>
          <a href={resource.url} download={attachment.file_name}>
            <Download className="size-4" aria-hidden />
            Download original
          </a>
        </Button>
      </div>
    </div>
  );
}

function TextPreview({ url }: { readonly url: string }) {
  const [text, setText] = useState<string | null>(null);
  const [error, setError] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    fetch(url, { signal: controller.signal })
      .then((response) => response.text())
      .then((value) => {
        if (!controller.signal.aborted) setText(value);
      })
      .catch(() => {
        if (!controller.signal.aborted) setError(true);
      });
    return () => controller.abort();
  }, [url]);
  if (error) return <p role="alert">Could not load text preview.</p>;
  if (text === null) return <p role="status">Loading preview...</p>;
  return (
    <>
      <pre className="max-h-[70svh] overflow-auto rounded-md border border-border bg-muted/30 p-4 text-xs leading-5">
        {text.slice(0, textPreviewLimit)}
      </pre>
      {text.length > textPreviewLimit ? (
        <p className="text-xs text-muted-foreground">
          Preview truncated. Download the original file to read everything.
        </p>
      ) : null}
    </>
  );
}
