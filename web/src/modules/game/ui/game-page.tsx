"use client";
import {
  currentStudentIdentity,
  isIdentityConfigured,
  observeStudentIdentity,
  signOutStudentIdentity,
} from "@/modules/identity";

import { missionsApi } from "@/modules/missions";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { miaAnalytics } from "@/lib/mia-analytics";

import {
  StudentSessionGate,
  clearStudentMissionSession,
  clearStudentSession,
  readStudentClassSession,
  readStudentMissionSession,
  saveStudentClassSession,
  saveStudentMissionSession,
  studentClassUnavailableEvent,
  studentEntryCopy,
  studentMissionRevisionKey,
  studentSessionErrorScope,
  studentSessionReadyEvent,
  useStudentClassMissions,
  useStudentPresence,
} from "@/modules/classrooms";
import type { MovementState } from "@/modules/game/adapters/types/world";
import { useBackgroundMusic } from "@/modules/game/hooks/use-background-music";
import { useGameSession } from "@/modules/game/hooks/use-game-session";
import { Chalkboard } from "@/modules/game/ui/chalkboard";
import { GameCanvas } from "@/modules/game/ui/game-canvas";
import { GameHud } from "@/modules/game/ui/game-hud";
import { MissionReward } from "@/modules/game/ui/mission-reward";
import {
  EvidenceSyncStatus,
  sourceStageID,
  stageLabel,
} from "@/modules/learning";

export function GamePage({
  initialClassCode = "",
}: {
  readonly initialClassCode?: string;
}) {
  const movementRef = useRef<MovementState>({ forward: 0, right: 0 });
  const {
    session,
    report,
    directiveRef,
    avatarPresetId,
    handlers,
    recoveryError,
    evidenceState,
    completionAcknowledged,
    retryEvidence,
    deleteUnsentEvidence,
  } = useGameSession();
  const [startingMissionID, setStartingMissionID] = useState("");
  const [missionStartError, setMissionStartError] = useState<string | null>(
    null,
  );
  const [rewardOpen, setRewardOpen] = useState(false);
  const autoSitMissionIDRef = useRef("");
  const gameAccessTrackedRef = useRef(false);
  const missionStartGenerationRef = useRef(0);
  const missionCompletionRef = useRef<{
    attemptId: string;
    captured: boolean;
    wasComplete: boolean;
  } | null>(null);

  useEffect(() => {
    if (gameAccessTrackedRef.current) {
      return;
    }

    gameAccessTrackedRef.current = true;
    miaAnalytics.gameAccessed({
      has_prefilled_class_code: Boolean(initialClassCode),
    });
  }, [initialClassCode]);

  useEffect(() => {
    if (!isIdentityConfigured()) return;
    let actorId = currentStudentIdentity()?.uid;
    const unsubscribe = observeStudentIdentity((user) => {
      if (user?.uid === actorId) return;
      actorId = user?.uid;
      missionStartGenerationRef.current += 1;
      setStartingMissionID("");
    });
    return () => {
      missionStartGenerationRef.current += 1;
      unsubscribe();
    };
  }, []);

  const leaveMission = useCallback(() => {
    missionStartGenerationRef.current += 1;
    clearStudentMissionSession();
    handlers.leaveMission();
    setStartingMissionID("");
    setRewardOpen(false);
    autoSitMissionIDRef.current = "";
    setMissionStartError(null);
  }, [handlers]);

  const leaveClass = useCallback(() => {
    missionStartGenerationRef.current += 1;
    clearStudentSession();
    handlers.leaveMission();
    setStartingMissionID("");
    setRewardOpen(false);
    autoSitMissionIDRef.current = "";
    setMissionStartError(null);
    void signOutStudentIdentity().catch(() => undefined);
  }, [handlers]);

  const leaveUnavailableClass = useCallback(() => {
    window.dispatchEvent(new Event(studentClassUnavailableEvent));
    leaveClass();
  }, [leaveClass]);

  const {
    classSession,
    completedMissionKeys: persistedCompletedMissionKeys,
    missions,
    missionsReady,
  } = useStudentClassMissions({
    activeMissionID: session.hydrated ? session.mission.id : "",
    onClassUnavailable: leaveUnavailableClass,
    onMissionUnavailable: leaveMission,
  });
  const { musicEnabled, musicWaitingForGesture, setMusicEnabled } =
    useBackgroundMusic(classSession !== null && missionsReady);

  const seatedLesson =
    session.seatedAt && session.lesson ? session.lesson : null;
  const localMissionComplete =
    session.hydrated &&
    Object.values(session.memory.progress).every(
      (progress) => progress.status === "complete",
    );
  const completedMission =
    localMissionComplete &&
    completionAcknowledged &&
    evidenceState.pending === 0 &&
    !evidenceState.accepting &&
    !evidenceState.blocked;
  const completionPending = localMissionComplete && !completedMission;
  const syncStatus = session.hydrated ? (
    <EvidenceSyncStatus
      state={evidenceState}
      completionPending={completionPending}
      onRetry={() => void retryEvidence()}
      onDelete={() => void deleteUnsentEvidence()}
    />
  ) : null;

  useEffect(() => {
    const missionSession = readStudentMissionSession();
    if (
      !session.hydrated ||
      !classSession ||
      !missionSession ||
      missionSession.missionId !== session.mission.id
    ) {
      return;
    }

    const trackedCompletion = missionCompletionRef.current;
    if (trackedCompletion?.attemptId !== missionSession.attemptId) {
      missionCompletionRef.current = {
        attemptId: missionSession.attemptId,
        // A completed attempt restored from storage is not a new completion.
        captured: completedMission,
        wasComplete: completedMission,
      };
      return;
    }

    if (
      completedMission &&
      !trackedCompletion.wasComplete &&
      !trackedCompletion.captured
    ) {
      miaAnalytics.missionCompleted({
        attempt_id: missionSession.attemptId,
        class_id: classSession.classId,
        mission_id: missionSession.missionId,
      });
      setRewardOpen(true);
      trackedCompletion.captured = true;
    }
    trackedCompletion.wasComplete = completedMission;
  }, [classSession, completedMission, session.hydrated, session.mission.id]);

  const completedMissionKeys = useMemo(() => {
    const completed = new Set(persistedCompletedMissionKeys);
    if (completedMission) {
      completed.add(
        studentMissionRevisionKey(
          session.mission.id,
          session.mission.publishedRevision,
        ),
      );
    }
    return completed;
  }, [completedMission, persistedCompletedMissionKeys, session.mission]);
  const pendingMissions = useMemo(
    () =>
      missions.filter(
        (mission) =>
          !completedMissionKeys.has(
            studentMissionRevisionKey(mission.id, mission.publishedRevision),
          ),
      ),
    [completedMissionKeys, missions],
  );

  const presenceStatus = completedMission
    ? "completed"
    : session.lesson?.phase === "thinking"
      ? "thinking"
      : session.seatedAt
        ? session.memory.progress[session.tutorPod].status === "review"
          ? "reviewing"
          : "learning"
        : "exploring";

  useStudentPresence(
    {
      stage_id: sourceStageID(session.mission, session.tutorPod),
      stage_name: stageLabel(session.mission, session.tutorPod),
      status: presenceStatus,
    },
    {
      enabled: session.hydrated && startingMissionID === "",
      onClassUnavailable: leaveClass,
      onMissionUnavailable: leaveMission,
    },
  );

  useEffect(() => {
    const autoSitMissionID = autoSitMissionIDRef.current;
    if (
      !autoSitMissionID ||
      !session.hydrated ||
      session.mission.id !== autoSitMissionID
    ) {
      return;
    }

    autoSitMissionIDRef.current = "";
    handlers.sit();
  }, [handlers, session.hydrated, session.mission.id]);

  async function startMission(missionID: string) {
    const generation = ++missionStartGenerationRef.current;
    setStartingMissionID("");
    const selectedMission = missions.find(
      (mission) => mission.id === missionID,
    );
    const user = currentStudentIdentity();
    if (!classSession || !selectedMission || !user?.isAnonymous) {
      const errorCode = !classSession
        ? "missing_class_session"
        : !selectedMission
          ? "mission_not_found_locally"
          : "missing_anonymous_user";
      miaAnalytics.missionStartFailed(
        { code: errorCode, name: "MissionStartPreconditionError" },
        {
          ...(classSession ? { class_id: classSession.classId } : {}),
          mission_id: missionID,
        },
      );
      setMissionStartError(studentEntryCopy.errors.offline);
      return false;
    }

    const isCurrentStart = () => {
      const activeClass = readStudentClassSession();
      return (
        generation === missionStartGenerationRef.current &&
        currentStudentIdentity()?.uid === user.uid &&
        activeClass?.classId === classSession.classId &&
        activeClass.classCode === classSession.classCode
      );
    };
    if (!isCurrentStart()) return false;

    if (
      session.hydrated &&
      session.mission.id === selectedMission.id &&
      session.mission.publishedRevision === selectedMission.publishedRevision
    ) {
      handlers.sit();
      return true;
    }

    setStartingMissionID(missionID);
    setRewardOpen(false);
    setMissionStartError(null);
    try {
      const idToken = await user.getIdToken();
      if (!isCurrentStart()) return false;
      const response = await missionsApi.startStudentMission(idToken, {
        class_code: classSession.classCode,
        mission_id: missionID,
        student_name: classSession.studentName,
      });
      if (!isCurrentStart()) return false;
      if (!response.mission) {
        throw new Error("The mission payload is unavailable.");
      }

      if (response.class) {
        saveStudentClassSession(response.class, classSession.studentName);
      }
      saveStudentMissionSession({
        attemptId: response.user.activity.current_game.attempt_id,
        classCode: classSession.classCode,
        missionId: response.mission.id,
        publishedRevision: response.mission.published_revision,
        subject: response.mission.subject,
        mission: response.mission,
      });
      miaAnalytics.missionStarted({
        attempt_id: response.user.activity.current_game.attempt_id,
        class_id: classSession.classId,
        mission_id: response.mission.id,
      });
      autoSitMissionIDRef.current = response.mission.id;
      window.dispatchEvent(new Event(studentSessionReadyEvent));
      return true;
    } catch (error) {
      if (!isCurrentStart()) return false;
      miaAnalytics.missionStartFailed(error, {
        class_id: classSession.classId,
        mission_id: missionID,
      });
      const scope = studentSessionErrorScope(error);
      if (scope === "class") {
        leaveUnavailableClass();
      } else if (scope === "mission") {
        setMissionStartError(studentEntryCopy.errors.missionUnavailable);
      } else {
        setMissionStartError(studentEntryCopy.errors.offline);
      }
      return false;
    } finally {
      if (isCurrentStart()) setStartingMissionID("");
    }
  }

  return (
    <StudentSessionGate initialClassCode={initialClassCode}>
      <main className="relative isolate h-dvh overflow-hidden bg-[#05070d] text-white">
        <GameCanvas
          movementRef={movementRef}
          directiveRef={directiveRef}
          onReport={handlers.onReport}
          showSeatMarker={missionsReady && pendingMissions.length > 0}
        />

        {!seatedLesson && syncStatus ? (
          <div className="absolute left-3 right-3 top-20 z-40 sm:left-auto sm:right-5 sm:w-80">
            {syncStatus}
          </div>
        ) : null}
        {recoveryError ? (
          <p
            role="alert"
            className="absolute inset-x-3 top-20 z-30 rounded-xl border border-amber-300/30 bg-slate-950/95 px-4 py-3 text-sm text-amber-100 sm:left-5 sm:right-auto sm:max-w-lg"
          >
            {recoveryError}
          </p>
        ) : null}

        {seatedLesson ? (
          <Chalkboard
            syncStatus={syncStatus}
            completionPending={completionPending}
            assessmentDisabled={
              evidenceState.blocked || evidenceState.accepting
            }
            mission={session.mission}
            lesson={seatedLesson}
            progress={session.memory.progress}
            explanationStyle={session.memory.preferences.explanationStyle}
            onChoose={handlers.choose}
            onWriteAnswer={handlers.writeAnswer}
            onStep={handlers.step}
            onWhy={handlers.why}
            onExplainNext={handlers.explainNext}
            onFeedback={handlers.feedback}
            onContinue={handlers.continue}
            onStand={handlers.stand}
            onEndSession={leaveMission}
          />
        ) : (
          <GameHud
            key={classSession?.classId ?? "no-class"}
            mission={session.mission}
            missionActive={session.hydrated}
            className={classSession?.className ?? ""}
            missions={missions}
            completedMissionKeys={completedMissionKeys}
            pendingMissionCount={pendingMissions.length}
            missionsReady={missionsReady}
            startingMissionID={startingMissionID}
            missionStartError={missionStartError ?? recoveryError}
            movementRef={movementRef}
            progress={session.memory.progress}
            tutorPod={session.tutorPod}
            tutorLine={session.hydrated ? session.tutorLine : ""}
            nearSeat={report.nearSeat}
            walkingToDesk={report.walking}
            avatarPresetId={avatarPresetId}
            musicEnabled={musicEnabled}
            musicWaitingForGesture={musicWaitingForGesture}
            onSelectAvatar={handlers.selectAvatar}
            onToggleMusic={setMusicEnabled}
            onSelectMission={startMission}
            onWalkToDesk={handlers.walk}
            onEndSession={leaveClass}
          />
        )}

        <MissionReward
          open={rewardOpen && completedMission}
          mission={session.mission}
          onDismiss={() => setRewardOpen(false)}
        />
      </main>
    </StudentSessionGate>
  );
}
