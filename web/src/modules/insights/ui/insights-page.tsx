"use client";
import { Main } from "@/components/layout/main";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DashboardShell } from "@/modules/dashboard";
import { toDashboardTab } from "@/modules/insights/domain/navigation";
import { useDashboardController } from "@/modules/insights/hooks/use-dashboard-controller";
import { AnalyticsCard } from "@/modules/insights/ui/analytics/analytics-card";
import { NotebookPanel } from "@/modules/insights/ui/notebook-panel";
import {
  ActiveMissionCard,
  ClassCodeBadge,
} from "@/modules/insights/ui/overview/active-mission-card";
import { ClassGlanceTiles } from "@/modules/insights/ui/overview/class-glance-tiles";
import { ClassPatternsSummary } from "@/modules/insights/ui/overview/class-patterns-summary";
import { MissionFunnel } from "@/modules/insights/ui/overview/mission-funnel";
import { NextLessonPlan } from "@/modules/insights/ui/overview/next-lesson-plan";
import { PatternGroups } from "@/modules/insights/ui/overview/pattern-groups";
import {
  DashboardLoadingState,
  TodayGuide,
} from "@/modules/insights/ui/overview/today-guide";
import { RecentPlayersTable } from "@/modules/insights/ui/recent-players-table";
import { SectionHeading } from "@/modules/insights/ui/section-heading";
import { StudentLearningDetail } from "@/modules/insights/ui/student-learning-detail";
import { StudentsTableSection } from "@/modules/insights/ui/student-panel";
import { ClassMissionsTable, MissionBuilder } from "@/modules/missions";
import { Gamepad2 } from "lucide-react";
import Link from "next/link";

export function DashboardPage({
  initialClassId,
}: { readonly initialClassId?: string } = {}) {
  const {
    auth,
    canGoBack, nextStudentPage, previousStudentPage,
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
  } = useDashboardController(initialClassId);

  return (
    <DashboardShell
      teacherClasses={teacherClasses}
      user={auth.user!}
      onSignOut={handleSignOut}
    >
      <Main id="main-content" className="space-y-5">
        <div
          id="overview"
          className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between"
        >
          <div>
            <p className="text-xs font-semibold text-[#9164f9]">CLASS HOME</p>
            <h1 className="mt-1 text-2xl font-bold">
              {selectedClass?.class_name || "Your class"}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {selectedClass
                ? [selectedClass.subject, selectedClass.grade_level]
                    .filter(Boolean)
                    .join(" · ")
                : "Create a class to get started."}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {classCode ? <ClassCodeBadge value={classCode} compact /> : null}
            <Button variant="outline" asChild>
              <Link
                href={studentViewHref}
                target="_blank"
                rel="noopener noreferrer"
              >
                <Gamepad2 className="size-4" aria-hidden />
                Student view
              </Link>
            </Button>
          </div>
        </div>

        {error ? (
          <p className="rounded-xl border border-[#9164f9]/35 bg-[#d2c2e9]/20 px-4 py-3 text-sm font-semibold text-foreground">
            {error}
          </p>
        ) : null}

        {overview?.page ? (
          <section aria-label="Student page" className="rounded-xl border bg-muted/20 p-4 text-sm">
            <p className="font-medium">Showing {overview.page.returned} students. Insights below cover this page and each student&apos;s latest attempt.</p>
            <p className="mt-1 text-muted-foreground">Open a student to browse their full mission history. Visible pages refresh every 15 seconds.</p>
            <div className="mt-3 flex flex-wrap items-center gap-3">
              <Button variant="outline" disabled={!canGoBack || overviewLoading} onClick={previousStudentPage}>Previous students</Button>
              <Button variant="outline" disabled={!overview.page.next_cursor || overviewLoading} onClick={nextStudentPage}>Next students</Button>
              <time className="text-xs text-muted-foreground" dateTime={overview.page.read_at}>Read at {new Date(overview.page.read_at).toLocaleTimeString()}</time>
            </div>
          </section>
        ) : null}

        <Tabs
          value={activeTab}
          onValueChange={(value) =>
            setActiveDashboardTab(toDashboardTab(value))
          }
        >
          <div className="overflow-x-auto pb-1">
            <TabsList className="min-w-max bg-[#d2c2e9]/25">
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="mission">Missions</TabsTrigger>
              <TabsTrigger value="students">Students</TabsTrigger>
              <TabsTrigger value="notebook">Teacher notes</TabsTrigger>
            </TabsList>
          </div>

          <TabsContent id="mission-builder" value="mission" className="mt-2">
            <ClassMissionsTable
              activeMissionId={activeMission?.id}
              deletingMissionId={deletingMissionId}
              missions={missions}
              loading={missionsLoading}
              onCreateMission={handleOpenMissionBuilder}
              onDeleteMission={handleDeleteMission}
              onEditMission={handleEditMission}
            />
          </TabsContent>

          <TabsContent id="students" value="students" className="mt-2">
            <StudentsTableSection
              classLearning={overview?.class_learning}
              students={students}
              onStudentClick={setSelectedStudent}
            />
          </TabsContent>

          <TabsContent id="notebook" value="notebook" className="mt-2">
            <NotebookPanel
              activeMission={activeMission}
              students={students}
              suggestions={suggestions}
            />
          </TabsContent>

          <TabsContent value="overview" className="mt-3 space-y-5">
            {classesLoading || overviewLoading ? (
              <DashboardLoadingState />
            ) : (
              <>
                <TodayGuide
                  activeMission={activeMission}
                  classCode={classCode}
                  glance={glance}
                  hasSelectedClass={hasSelectedClass}
                  teacherName={
                    overview?.teacher.display_name ??
                    auth.user?.displayName ??
                    ""
                  }
                  onCreateMission={handleOpenMissionBuilder}
                  onOpenStudents={handleOpenStudents}
                  onPlanFollowUp={handlePlanFollowUp}
                  onShareCode={handleShareCode}
                />

                <div className="grid min-w-0 gap-5 xl:grid-cols-2">
                  <ClassGlanceTiles
                    glance={glance}
                    shareCodeLabel={shareCodeLabel}
                    onPlanFollowUp={handlePlanFollowUp}
                    onShareCode={handleShareCode}
                  />

                  <AnalyticsCard
                    title="Where students are in today's mission"
                    description={
                      activeMission
                        ? "The longer the bar, the more students are at that step. Names below help you know who to check in with."
                        : "Publish a mission to see each student's current step here."
                    }
                    source={activeMission ? funnel.source : undefined}
                  >
                    <MissionFunnel
                      steps={funnel.data}
                      totalStudents={summaryStudentCount}
                    />
                  </AnalyticsCard>
                </div>

                <SectionHeading
                  eyebrow="WHAT MIA NOTICED"
                  title="What students found easy — and difficult"
                  detail="These notes come from students' answers and explanations, so you can plan without reading every response."
                  source={students.length > 0 ? patternSource : undefined}
                />
                <ClassPatternsSummary
                  growth={growthPatterns.data}
                  strengths={strengthPatterns.data}
                  summary={classSummary}
                  totalStudents={summaryStudentCount}
                />

                <div className="grid gap-x-5 gap-y-2 xl:grid-cols-2">
                  <section className="min-w-0">
                    <SectionHeading
                      eyebrow="REVIEW"
                      title="Students who may need help"
                      detail="Start with “check in soon”, then use the names to make a small group."
                    />
                    <div className="mt-3">
                      <PatternGroups
                        variant="growth"
                        groups={growthPatterns.data}
                        limit={4}
                        totalStudents={summaryStudentCount}
                        emptyMessage="Nothing to review yet. This section fills in after students explain their answers."
                      />
                    </div>
                  </section>

                  <section className="min-w-0">
                    <SectionHeading
                      eyebrow="BUILD ON"
                      title="Students showing confidence"
                      detail="Use these names for peer explanations, pairs, or an extra challenge."
                    />
                    <div className="mt-3">
                      <PatternGroups
                        variant="strength"
                        groups={strengthPatterns.data}
                        limit={4}
                        totalStudents={summaryStudentCount}
                        emptyMessage="Strengths will appear as students explain answers and finish steps."
                      />
                    </div>
                  </section>
                </div>

                <div id="next-lesson" className="scroll-mt-24">
                  <SectionHeading
                    eyebrow="NEXT LESSON"
                    title="Ready-to-use ideas"
                    detail="Choose a suggestion as-is, adapt it, or let Mia turn the class results into the next mission."
                    source={
                      plannedSuggestions.length > 0
                        ? overview?.generated_by_ai
                          ? "live"
                          : "estimated"
                        : undefined
                    }
                  />
                </div>
                <NextLessonPlan
                  canBuild={hasSelectedClass}
                  suggestions={plannedSuggestions}
                  onBuildNextMission={handleBuildNextMission}
                />

                <ActiveMissionCard
                  activeMission={activeMission}
                  classCode={classCode}
                  generatedByAi={overview?.generated_by_ai ?? false}
                  teacherClass={selectedClass}
                />

                <Card className="gap-0 border-[#dad2e6] py-0">
                  <CardHeader className="border-b border-[#dad2e6] bg-[#d2c2e9]/10 px-4 py-4 sm:px-6">
                    <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
                      <div>
                        <CardTitle>Latest student activity</CardTitle>
                        <CardDescription className="mt-2">
                          The 10 most recently active students on this page.
                        </CardDescription>
                      </div>
                      <span className="text-xs font-medium text-muted-foreground">
                        Newest first
                      </span>
                    </div>
                  </CardHeader>
                  <CardContent className="p-0">
                    <RecentPlayersTable
                      fallbackSubject={
                        activeMission?.subject ?? selectedClass?.subject
                      }
                      students={students}
                    />
                  </CardContent>
                </Card>
              </>
            )}
          </TabsContent>
        </Tabs>

        {selectedClassId ? (
          <MissionBuilder
            key={`${selectedClassId}:${draftVersion}`}
            approving={approving}
            classId={selectedClassId}
            draftPrompt={draftPrompt ?? undefined}
            open={missionBuilderOpen}
            previewMission={previewMission}
            processing={processing}
            progress={missionProgress}
            requestError={error}
            teacherClass={selectedClass}
            onApprove={handleApproveMission}
            onOpenChange={(open) => {
              setMissionBuilderOpen(open);
              if (!open) {
                setError(null);
              }
            }}
            onProcess={handleProcessMission}
          />
        ) : null}

        <StudentLearningDetail
          key={selectedStudent ? `${selectedStudent.class_id}:${selectedStudent.id}` : "no-student"}
          open={selectedStudent !== null}
          student={selectedStudent}
          onOpenChange={(open) => {
            if (!open) {
              setSelectedStudent(null);
            }
          }}
        />
      </Main>
    </DashboardShell>
  );
}
