"use client";
import { useTeacherAuth } from "@/modules/identity";
import { missionsApi } from "@/modules/missions/adapters/http";
import type { AttachmentAccess } from "@/modules/missions/adapters/types/attachment-access";
import type { MissionAttachmentDto } from "@/modules/missions/adapters/types/contracts";
import { attachmentOriginalURL } from "@/modules/missions/domain/attachments";
import { readBoundedBlob } from "@/shared/http/blob";
import { useEffect, useState } from "react";

/** Bearer tokens stay in fetch headers; preview URLs exist only while the view is mounted. */
export function useAttachmentResource(
  attachment: MissionAttachmentDto,
  access?: AttachmentAccess,
) {
  const { user } = useTeacherAuth();
  const [state, setState] = useState<{
    key: string;
    url: string;
    loading: boolean;
    error: string | null;
  }>({ key: "", url: "", loading: true, error: null });
  const { classId = "", missionId = "", revision } = access ?? {};
  const uuid = attachment.uploadcare_uuid;
  const stored = attachment.storage?.status === "stored";
  const source = attachmentOriginalURL(attachment);
  const version =
    attachment.storage?.updated_at ?? attachment.storage?.stored_at ?? "";
  const key = `${user?.uid ?? "signed-out"}:${stored}:${classId}:${missionId}:${revision ?? "current"}:${uuid}:${version}:${source}`;
  useEffect(() => {
    const controller = new AbortController();
    let objectUrl = "";
    const load = async () => {
      try {
        let blob: Blob;
        if (stored) {
          if (!user || !classId || !missionId)
            throw new Error("Sign in to download this mission attachment.");
          blob = await missionsApi.downloadAttachment(
            await user.getIdToken(),
            { classId, missionId, revision },
            uuid,
            controller.signal,
          );
        } else {
          if (!source) throw new Error("This attachment is not available yet.");
          const response = await fetch(source, { signal: controller.signal });
          if (!response.ok) throw new Error("Could not load this attachment.");
          blob = await readBoundedBlob(response, 20 * 1024 * 1024);
        }
        if (controller.signal.aborted) return;
        objectUrl = URL.createObjectURL(blob);
        setState({ key, url: objectUrl, loading: false, error: null });
      } catch (error) {
        if (!controller.signal.aborted)
          setState({
            key,
            url: "",
            loading: false,
            error:
              error instanceof Error
                ? error.message
                : "Could not load this attachment.",
          });
      }
    };
    void load();
    return () => {
      controller.abort();
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [classId, key, missionId, revision, source, stored, user, uuid]);
  return state.key === key ? state : { url: "", loading: true, error: null };
}
