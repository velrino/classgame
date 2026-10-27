"use client";
import { miaAnalytics } from "@/lib/mia-analytics";
import { useTeacherAuth } from "@/modules/identity";
import { useDashboardSnapshots } from "@/modules/insights/adapters/hooks/use-dashboard-snapshots";
import { DashboardTab } from "@/modules/insights/adapters/types/dashboard";
import {
  buildClassGlance,
  buildClassSummary,
  buildGrowthPatterns,
  buildMissionFunnel,
  buildNextMissionPrompt,
  buildStrengthPatterns,
  planSuggestions,
} from "@/modules/insights/domain/analytics";
import {
  defaultDashboardTab,
  hashByTab,
  tabByHash,
} from "@/modules/insights/domain/navigation";
import type {
  MissionPreviewProgressDto,
  TeacherClassMissionDto,
} from "@/modules/missions";
import {
  buildApproveMissionRequest,
  missionsApi,
  upsertClassMission,
  type MissionBuilderForm,
} from "@/modules/missions";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
export function useDashboardController(initialClassId?: string) {
  const router = useRouter();
  const auth = useTeacherAuth();
  const {
    overview,
    teacherClasses,
    selectedClassId,
    setSelectedClassId,
    classesLoading,
    overviewLoading,
    missions,
    setMissions,
    missionsLoading,
    error,
    setError,
    selectedStudent,
    setSelectedStudent,
    selectedClass,
    loadOverview,
  } = useDashboardSnapshots(auth, initialClassId);
  const [activeTab, setActiveTab] = useState<DashboardTab>(defaultDashboardTab);
  const [missionBuilderOpen, setMissionBuilderOpen] = useState(false);
  const [previewMission, setPreviewMission] =
    useState<TeacherClassMissionDto | null>(null);
  const [processing, setProcessing] = useState(false);
  const [missionProgress, setMissionProgress] =
    useState<MissionPreviewProgressDto | null>(null);
  const [approving, setApproving] = useState(false);
  const [deletingMissionId, setDeletingMissionId] = useState<string | null>(
    null,
  );
  const [draftPrompt, setDraftPrompt] = useState<string | null>(null);
  const [draftVersion, setDraftVersion] = useState(0);
  const [shareCodeLabel, setShareCodeLabel] = useState("Share code");

  useEffect(() => {
    function syncTabFromHash() {
      setActiveTab(tabByHash[window.location.hash] ?? defaultDashboardTab);
    }

    syncTabFromHash();
    window.addEventListener("hashchange", syncTabFromHash);

    return () => {
      window.removeEventListener("hashchange", syncTabFromHash);
    };
  }, []);

  const activeMission = overview?.active_mission ?? null;
  const suggestions = useMemo(
    () => overview?.suggestions ?? [],
    [overview?.suggestions],
  );
  const students = useMemo(
    () => overview?.students ?? [],
    [overview?.students],
  );
  const classCode =
    previewMission?.class_code ??
    activeMission?.class_code ??
    selectedClass?.class_code ??
    overview?.class_code ??
    "";
  const hasSelectedClass = Boolean(selectedClassId);
  const studentViewHref = classCode
    ? `/home?classCode=${encodeURIComponent(classCode)}`
    : "/home";
  const glance = useMemo(() => buildClassGlance(students), [students]);
  const funnel = useMemo(
    () => buildMissionFunnel(students, activeMission),
    [students, activeMission],
  );
  const growthPatterns = useMemo(
    () => buildGrowthPatterns(students),
    [students],
  );
  const strengthPatterns = useMemo(
    () => buildStrengthPatterns(students),
    [students],
  );
  const patternSource = growthPatterns.source;
  const summaryStudentCount = students.length;
  const classSummary = useMemo(
    () =>
      buildClassSummary(
        growthPatterns.data,
        strengthPatterns.data,
        summaryStudentCount,
      ),
    [growthPatterns.data, strengthPatterns.data, summaryStudentCount],
  );
  const plannedSuggestions = useMemo(
    () => planSuggestions(suggestions),
    [suggestions],
  );

  async function handleProcessMission(value: MissionBuilderForm) {
    if (!auth.user) {
      return;
    }

    const isCreatingMission = !value.mission_id?.trim();
    setProcessing(true);
    setMissionProgress({
      stage: "starting",
      message: "Starting mission preview.",
      percent: 1,
    });
    setError(null);
    try {
      const token = await auth.user.getIdToken();
      const response = await missionsApi.previewStream(
        token,
        value.class_id,
        {
          mission_id: value.mission_id,
          expected_draft_revision: value.expected_draft_revision,
          class_name: value.class_name,
          mission_name: value.mission_name,
          subject: value.subject,
          grade_level: value.grade_level,
          learning_goal: value.learning_goal,
          teacher_prompt: value.teacher_prompt,
          audience_type: value.audience_type,
          target_student_ids: value.target_student_ids,
          attachments: value.attachments,
        },
        {
          onProgress: setMissionProgress,
        },
      );
      setPreviewMission(response.mission);
      setDraftVersion((current) => current + 1);
      setMissions((current) => upsertClassMission(current, response.mission));
      setSelectedClassId(response.mission.class_id);
      setMissionBuilderOpen(true);
      setActiveDashboardTab("mission");
      if (isCreatingMission) {
        miaAnalytics.missionCreated({
          class_id: response.mission.class_id,
          mission_id: response.mission.id,
        });
      }
    } catch (caughtError) {
      if (isCreatingMission) {
        miaAnalytics.missionCreationFailed(caughtError, {
          class_id: value.class_id,
        });
      }
      setError(
        caughtError instanceof Error
          ? caughtError.message
          : "Could not save this mission.",
      );
    } finally {
      setProcessing(false);
      setMissionProgress(null);
    }
  }

  async function handleApproveMission(mission: TeacherClassMissionDto) {
    if (!auth.user) {
      return;
    }

    let missionPublished = false;
    setApproving(true);
    setError(null);
    try {
      const token = await auth.user.getIdToken();
      const response = await missionsApi.approve(
        token,
        mission.class_id,
        mission.id,
        buildApproveMissionRequest(mission),
      );
      missionPublished = true;
      miaAnalytics.missionPublished({
        class_id: response.mission.class_id,
        mission_id: response.mission.id,
      });
      setPreviewMission(response.mission);
      setMissions((current) => upsertClassMission(current, response.mission));
      await loadOverview(token, response.mission.class_id);
      setMissionBuilderOpen(false);
      setActiveDashboardTab("mission");
    } catch (caughtError) {
      if (!missionPublished) {
        miaAnalytics.missionPublishFailed(caughtError, {
          class_id: mission.class_id,
          mission_id: mission.id,
        });
      }
      setError(
        caughtError instanceof Error
          ? caughtError.message
          : "Could not share this mission.",
      );
    } finally {
      setApproving(false);
    }
  }

  async function handleSignOut() {
    await auth.signOutTeacher();
  }

  function handleOpenMissionBuilder() {
    if (!selectedClassId) {
      router.push("/dashboard/classes");
      return;
    }

    setPreviewMission(null);
    setDraftPrompt(null);
    setError(null);
    setDraftVersion((current) => current + 1);
    setActiveDashboardTab("mission");
    setMissionBuilderOpen(true);
  }

  function handleEditMission(mission: TeacherClassMissionDto) {
    setPreviewMission(mission);
    setDraftPrompt(null);
    setError(null);
    setDraftVersion((current) => current + 1);
    setMissionBuilderOpen(true);
    setActiveDashboardTab("mission");
  }

  async function handleDeleteMission(mission: TeacherClassMissionDto) {
    if (!auth.user || deletingMissionId) {
      return;
    }

    const missionLabel =
      mission.mission_name || mission.title || "this mission";
    const isLive = activeMission?.id === mission.id;
    const confirmed = window.confirm(
      isLive
        ? `Remove “${missionLabel}”? It will disappear from the student classroom.`
        : `Remove “${missionLabel}”? It will disappear from this class mission list.`,
    );
    if (!confirmed) {
      return;
    }

    setDeletingMissionId(mission.id);
    setError(null);
    try {
      const token = await auth.user.getIdToken();
      await missionsApi.remove(token, mission.class_id, mission.id);
      setMissions((current) =>
        current.filter((item) => item.id !== mission.id),
      );

      if (previewMission?.id === mission.id) {
        setPreviewMission(null);
        setMissionBuilderOpen(false);
      }

      await loadOverview(token, mission.class_id);
    } catch (caughtError) {
      setError(
        caughtError instanceof Error
          ? caughtError.message
          : "Could not remove this mission.",
      );
    } finally {
      setDeletingMissionId(null);
    }
  }

  function handleBuildNextMission() {
    if (!selectedClassId) {
      router.push("/dashboard/classes");
      return;
    }

    setPreviewMission(null);
    setError(null);
    setDraftPrompt(
      buildNextMissionPrompt(
        growthPatterns.data,
        strengthPatterns.data,
        activeMission,
      ),
    );
    setDraftVersion((current) => current + 1);
    setMissionBuilderOpen(true);
    setActiveDashboardTab("mission");
  }

  function handlePlanFollowUp() {
    setActiveDashboardTab("overview");
    window.requestAnimationFrame(() => {
      document
        .getElementById("next-lesson")
        ?.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }

  function handleOpenStudents() {
    setActiveDashboardTab("students");
  }

  async function handleShareCode() {
    if (!classCode) {
      router.push("/dashboard/classes");
      return;
    }

    try {
      await navigator.clipboard.writeText(classCode);
      setShareCodeLabel("Code copied");
    } catch {
      setShareCodeLabel(`Code: ${classCode}`);
    }
    window.setTimeout(() => setShareCodeLabel("Share code"), 2_000);
  }

  function setActiveDashboardTab(value: DashboardTab) {
    setActiveTab(value);

    const nextHash = hashByTab[value];
    if (typeof window !== "undefined" && window.location.hash !== nextHash) {
      window.history.replaceState(
        null,
        "",
        `${window.location.pathname}${window.location.search}${nextHash}`,
      );
      window.dispatchEvent(new Event("hashchange"));
    }
  }

  return {
    auth,
    activeTab,
    overview,
    teacherClasses,
    selectedClassId,
    classesLoading,
    overviewLoading,
    missionBuilderOpen,
    setMissionBuilderOpen,
    previewMission,
    missions,
    missionsLoading,
    error,
    setError,
    processing,
    missionProgress,
    approving,
    deletingMissionId,
    draftPrompt,
    draftVersion,
    shareCodeLabel,
    selectedStudent,
    setSelectedStudent,
    selectedClass,
    activeMission,
    suggestions,
    students,
    classCode,
    hasSelectedClass,
    studentViewHref,
    glance,
    funnel,
    growthPatterns,
    strengthPatterns,
    patternSource,
    summaryStudentCount,
    classSummary,
    plannedSuggestions,
    handleProcessMission,
    handleApproveMission,
    handleSignOut,
    handleOpenMissionBuilder,
    handleEditMission,
    handleDeleteMission,
    handleBuildNextMission,
    handlePlanFollowUp,
    handleOpenStudents,
    handleShareCode,
    setActiveDashboardTab,
  };
}
