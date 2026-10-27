"use client";

import { classroomsApi } from "@/modules/classrooms/adapters/http";
import {
  BookOpenCheck,
  ClipboardList,
  Gamepad2,
  Plus,
  Users,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { Main } from "@/components/layout/main";
import { Button } from "@/components/ui/button";
import { miaAnalytics } from "@/lib/mia-analytics";
import type {
  TeacherClassDto,
  UpsertTeacherClassRequest,
} from "@/modules/classrooms/adapters/types/contracts";

import { MetricCard } from "@/components/ui/metric-card";
import {
  rememberSelectedClass,
  sortTeacherClasses,
  upsertTeacherClass,
} from "@/modules/classrooms/adapters/selected-class";
import { ClassFormModal } from "@/modules/classrooms/ui/class-form-modal";
import { ClassesTable } from "@/modules/classrooms/ui/classes-table";
import { DashboardShell } from "@/modules/dashboard";
import { useTeacherAuth } from "@/modules/identity";

export function ClassesPage() {
  const router = useRouter();
  const auth = useTeacherAuth();
  const [teacherClasses, setTeacherClasses] = useState<TeacherClassDto[]>([]);
  const [selectedClass, setSelectedClass] = useState<TeacherClassDto | null>(
    null,
  );
  const [classFormOpen, setClassFormOpen] = useState(false);
  const [pageError, setPageError] = useState<string | null>(null);
  const [classFormError, setClassFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (auth.status !== "signed_in" || !auth.user) {
      return;
    }

    let cancelled = false;
    const user = auth.user;

    Promise.resolve()
      .then(async () => {
        setPageError(null);

        const token = await user.getIdToken();
        return classroomsApi.list(token);
      })
      .then((response) => {
        if (cancelled) {
          return;
        }

        setTeacherClasses(sortTeacherClasses(response.teacher_classes));
      })
      .catch((caughtError: unknown) => {
        if (!cancelled) {
          setPageError(
            caughtError instanceof Error
              ? caughtError.message
              : "Could not load classes.",
          );
        }
      });

    return () => {
      cancelled = true;
    };
  }, [auth.status, auth.user]);

  const activeClasses = useMemo(
    () =>
      teacherClasses.filter(
        (teacherClass) => teacherClass.status !== "archived",
      ).length,
    [teacherClasses],
  );
  const publishedMissions = useMemo(
    () =>
      teacherClasses.reduce(
        (total, teacherClass) => total + teacherClass.mission_count,
        0,
      ),
    [teacherClasses],
  );
  const playableClasses = useMemo(
    () =>
      teacherClasses.filter((teacherClass) =>
        Boolean(teacherClass.active_mission_id),
      ).length,
    [teacherClasses],
  );

  function handleCreateClass() {
    setSelectedClass(null);
    setClassFormError(null);
    setClassFormOpen(true);
  }

  function handleEditClass(teacherClass: TeacherClassDto) {
    setSelectedClass(teacherClass);
    setClassFormError(null);
    setClassFormOpen(true);
  }

  function handleOpenClass(teacherClass: TeacherClassDto) {
    rememberSelectedClass(teacherClass.class_id);
    router.push(
      `/dashboard/class/${encodeURIComponent(teacherClass.class_id)}`,
    );
  }

  async function handleSubmitClass(value: UpsertTeacherClassRequest) {
    if (!auth.user) {
      return;
    }

    const isCreatingClass = !value.class_id?.trim();
    setSaving(true);
    setClassFormError(null);
    try {
      const token = await auth.user.getIdToken();
      const response = await classroomsApi.upsert(token, value);
      setTeacherClasses((current) =>
        upsertTeacherClass(current, response.teacher_class),
      );
      setSelectedClass(response.teacher_class);
      rememberSelectedClass(response.teacher_class.class_id);
      if (isCreatingClass) {
        miaAnalytics.classCreated({
          class_id: response.teacher_class.class_id,
        });
      }
    } catch (caughtError) {
      if (isCreatingClass) {
        miaAnalytics.classCreationFailed(caughtError);
      }
      setClassFormError(
        caughtError instanceof Error
          ? caughtError.message
          : "Could not save class.",
      );
    } finally {
      setSaving(false);
    }
  }

  async function handleSignOut() {
    await auth.signOutTeacher();
  }

  return (
    <DashboardShell
      teacherClasses={teacherClasses}
      user={auth.user!}
      onSignOut={handleSignOut}
    >
      <Main id="main-content" className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold">Classes</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Create class codes, define the content teachers want to reinforce,
              and launch student game missions.
            </p>
          </div>

          <Button type="button" onClick={handleCreateClass}>
            <Plus className="size-4" aria-hidden />
            New class
          </Button>
        </div>

        {pageError ? (
          <p className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm font-semibold text-rose-700">
            {pageError}
          </p>
        ) : null}

        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            icon={BookOpenCheck}
            label="Classes"
            value={String(teacherClasses.length)}
            detail={`${activeClasses} active`}
          />
          <MetricCard
            icon={ClipboardList}
            label="Missions"
            value={String(publishedMissions)}
            detail="Published across classes"
          />
          <MetricCard
            icon={Users}
            label="Students"
            value="0"
            detail="Loaded after students play"
          />
          <MetricCard
            icon={Gamepad2}
            label="Ready to play"
            value={String(playableClasses)}
            detail="Classes with a published mission"
          />
        </section>

        <ClassesTable
          teacherClasses={teacherClasses}
          onEditClass={handleEditClass}
          onOpenClass={handleOpenClass}
        />

        <ClassFormModal
          key={selectedClass?.id ?? "new-class"}
          error={classFormError}
          open={classFormOpen}
          saving={saving}
          teacherClass={selectedClass}
          onOpenChange={setClassFormOpen}
          onSubmit={handleSubmitClass}
        />
      </Main>
    </DashboardShell>
  );
}
