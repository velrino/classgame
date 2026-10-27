"use client";

import * as UC from "@uploadcare/file-uploader";
import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useSyncExternalStore,
  type DragEvent,
} from "react";

import "@uploadcare/file-uploader/web/uc-file-uploader-regular.min.css";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { MissionAttachmentDto } from "@/modules/missions/adapters/types/contracts";

UC.defineComponents(UC);

const DEFAULT_UPLOADCARE_PUBLIC_KEY = "5c2474782ec064c09ef1";
const MAX_FILES = 5;
const MAX_FILE_SIZE_BYTES = 20 * 1024 * 1024;
const ACCEPTED_FILE_TYPES =
  ".pdf,.png,.jpg,.jpeg,.docx,.txt,application/pdf,image/png,image/jpeg,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain";
const ACCEPTED_FILE_ERROR =
  "Only PDF, PNG, JPEG, DOCX, or TXT files are supported.";
const allowedFileExtensions = [
  ".pdf",
  ".png",
  ".jpg",
  ".jpeg",
  ".docx",
  ".txt",
];
const allowedMimeTypes = new Set([
  "application/pdf",
  "image/png",
  "image/jpeg",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "text/plain",
]);
const subscribeToClientEnvironment = () => () => {};

type MissionFileUploaderProps = {
  readonly disabled?: boolean;
  readonly maxFiles?: number;
  readonly onAttachmentsChange: (attachments: MissionAttachmentDto[]) => void;
  readonly onErrorChange: (error: string | null) => void;
  readonly onUploadingChange: (uploading: boolean) => void;
};

export function MissionFileUploader({
  disabled = false,
  maxFiles = MAX_FILES,
  onAttachmentsChange,
  onErrorChange,
  onUploadingChange,
}: MissionFileUploaderProps) {
  const resolvedMaxFiles = Math.max(0, Math.min(MAX_FILES, maxFiles));
  const isClient = useSyncExternalStore(
    subscribeToClientEnvironment,
    () => true,
    () => false,
  );
  const reactId = useId();
  const contextName = `mission-files-${reactId.replace(/:/g, "")}`;
  const contextProviderRef = useRef<UC.UploadCtxProvider>(null);
  const publicKey =
    process.env.NEXT_PUBLIC_UPLOADCARE_PUBLIC_KEY?.trim() ||
    DEFAULT_UPLOADCARE_PUBLIC_KEY;

  const openFilePicker = useCallback(() => {
    try {
      contextProviderRef.current?.getAPI().openSystemDialog();
    } catch {
      onErrorChange("File uploader is still loading. Try again.");
    }
  }, [onErrorChange]);

  const handleDragOver = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      if (disabled || resolvedMaxFiles === 0) {
        return;
      }

      event.preventDefault();
    },
    [disabled, resolvedMaxFiles],
  );

  const handleDrop = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      if (disabled || resolvedMaxFiles === 0) {
        return;
      }

      event.preventDefault();

      const files = Array.from(event.dataTransfer.files).slice(
        0,
        resolvedMaxFiles,
      );
      if (files.length === 0) {
        return;
      }

      const unsupportedFile = files.find((file) => !isAllowedLocalFile(file));
      if (unsupportedFile) {
        onErrorChange(`${unsupportedFile.name}: ${ACCEPTED_FILE_ERROR}`);
        return;
      }

      const oversizedFile = files.find(
        (file) => file.size > MAX_FILE_SIZE_BYTES,
      );
      if (oversizedFile) {
        onErrorChange(`${oversizedFile.name}: file is larger than 20 MB.`);
        return;
      }

      try {
        const api = contextProviderRef.current?.getAPI();
        files.forEach((file) => {
          api?.addFileFromObject(file, {
            fileName: file.name,
            source: "drop-area",
          });
        });
        api?.uploadAll();
      } catch {
        onErrorChange("File uploader is still loading. Try again.");
      }
    },
    [disabled, onErrorChange, resolvedMaxFiles],
  );

  useEffect(() => {
    if (!isClient) {
      return;
    }

    const contextProvider = contextProviderRef.current;
    if (!contextProvider) {
      return;
    }

    function handleChange(event: UC.EventMap["change"]) {
      const collection = event.detail;
      const firstFailedEntry = collection.failedEntries[0];
      const firstError = firstFailedEntry?.errors[0] ?? collection.errors[0];
      const unsupportedEntry = collection.successEntries.find(
        (entry) => !isAllowedFileEntry(entry),
      );
      const acceptedEntries =
        collection.successEntries.filter(isAllowedFileEntry);

      onUploadingChange(
        collection.uploadingEntries.length > 0 ||
          collection.idleEntries.length > 0,
      );
      onAttachmentsChange(
        acceptedEntries.map((entry) => ({
          uploadcare_uuid: entry.uuid,
          cdn_url: entry.cdnUrl,
          file_name: getFileEntryName(entry),
          mime_type: entry.mimeType || "application/octet-stream",
          size_bytes: entry.size,
        })),
      );
      onErrorChange(
        firstError
          ? `${firstFailedEntry?.name ? `${firstFailedEntry.name}: ` : ""}${firstError.message}`
          : unsupportedEntry
            ? `${getFileEntryName(unsupportedEntry)}: ${ACCEPTED_FILE_ERROR}`
            : null,
      );
    }

    contextProvider.addEventListener("change", handleChange);

    return () => {
      contextProvider.removeEventListener("change", handleChange);
      onUploadingChange(false);
    };
  }, [isClient, onAttachmentsChange, onErrorChange, onUploadingChange]);

  return (
    <div
      aria-disabled={disabled}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      className={cn(
        "flex min-h-24 flex-col items-center justify-center gap-2 rounded-md border border-dashed border-input bg-background px-4 py-5 text-center transition focus-within:border-ring focus-within:ring-2 focus-within:ring-ring/20",
        disabled && "pointer-events-none opacity-60",
      )}
    >
      {resolvedMaxFiles === 0 ? (
        <p className="py-2 text-sm text-muted-foreground">
          Remove an existing file before uploading another one.
        </p>
      ) : isClient ? (
        <>
          <uc-config
            ctx-name={contextName}
            pubkey={publicKey}
            multiple={resolvedMaxFiles > 1}
            multipleMax={resolvedMaxFiles}
            sourceList="local"
            accept={ACCEPTED_FILE_TYPES}
            maxLocalFileSizeBytes={MAX_FILE_SIZE_BYTES}
            confirmUpload={false}
            store={false}
          ></uc-config>
          <uc-file-uploader-regular
            ctx-name={contextName}
            class="uc-light"
            headless
          ></uc-file-uploader-regular>
          <Button type="button" onClick={openFilePicker}>
            Upload files
          </Button>
          <p className="text-xs font-medium text-muted-foreground">
            or Drop files here
          </p>
          <uc-upload-ctx-provider
            ref={contextProviderRef}
            ctx-name={contextName}
          ></uc-upload-ctx-provider>
        </>
      ) : (
        <div className="h-10 animate-pulse rounded-md bg-muted" aria-hidden />
      )}
    </div>
  );
}

function isAllowedFileEntry(entry: UC.OutputFileEntry<"success">) {
  const mimeType = entry.mimeType.toLocaleLowerCase();
  const fileName = getFileEntryName(entry).toLocaleLowerCase();

  return (
    allowedMimeTypes.has(mimeType) ||
    allowedFileExtensions.some((extension) => fileName.endsWith(extension))
  );
}

function isAllowedLocalFile(file: File) {
  const mimeType = file.type.toLocaleLowerCase();
  const fileName = file.name.toLocaleLowerCase();

  return (
    allowedMimeTypes.has(mimeType) ||
    allowedFileExtensions.some((extension) => fileName.endsWith(extension))
  );
}

function getFileEntryName(entry: UC.OutputFileEntry<"success">) {
  return entry.fileInfo.originalFilename || entry.name;
}
