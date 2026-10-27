"use client";

import { classroomsApi } from "@/modules/classrooms/adapters/http";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { Main } from "@/components/layout/main";
import { miaAnalytics } from "@/lib/mia-analytics";
import type {
  TeacherClassDto,
  UpsertTeacherClassRequest,
} from "@/modules/classrooms/adapters/types/contracts";

import {
  rememberSelectedClass,
  sortTeacherClasses,
  upsertTeacherClass,
} from "@/modules/classrooms/adapters/selected-class";
import { ClassFormModal } from "@/modules/classrooms/ui/class-form-modal";
import { ClassroomCardGrid } from "@/modules/classrooms/ui/classroom-card-grid";
import { DashboardShell } from "@/modules/dashboard";
import { useTeacherAuth } from "@/modules/identity";

export function TeacherClassesPage() {
  const router = useRouter();
  const auth = useTeacherAuth();
  const [teacherClasses, setTeacherClasses] = useState<TeacherClassDto[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedClass, setSelectedClass] = useState<TeacherClassDto | null>(
    null,
  );
  const [classFormOpen, setClassFormOpen] = useState(false);
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
        setLoading(true);
        setError(null);

        const token = await user.getIdToken();
        return classroomsApi.list(token);
      })
      .then((response) => {
        if (!cancelled) {
          setTeacherClasses(sortTeacherClasses(response.teacher_classes));
        }
      })
      .catch((caughtError: unknown) => {
        if (!cancelled) {
          setError(
            caughtError instanceof Error
              ? caughtError.message
              : "Could not load classes.",
          );
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [auth.status, auth.user]);

  async function handleSignOut() {
    await auth.signOutTeacher();
  }

  function handleSelectClass(teacherClass: TeacherClassDto) {
    rememberSelectedClass(teacherClass.class_id);
    router.push(
      `/dashboard/class/${encodeURIComponent(teacherClass.class_id)}`,
    );
  }

  function handleEditClass(teacherClass: TeacherClassDto) {
    setSelectedClass(teacherClass);
    setClassFormError(null);
    setClassFormOpen(true);
  }

  function handleCreateClass() {
    setSelectedClass(null);
    setClassFormError(null);
    setClassFormOpen(true);
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
      setClassFormOpen(false);

      if (isCreatingClass) {
        miaAnalytics.classCreated({
          class_id: response.teacher_class.class_id,
        });
        handleSelectClass(response.teacher_class);
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

  return (
    <DashboardShell
      teacherClasses={teacherClasses}
      user={auth.user!}
      onSignOut={handleSignOut}
    >
      <Main id="main-content" className="space-y-4 py-4 sm:py-6">
        {error ? (
          <p className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm font-semibold text-rose-700">
            {error}
          </p>
        ) : null}
        <ClassroomCardGrid
          loading={loading}
          teacherClasses={teacherClasses}
          onCreateClass={handleCreateClass}
          onEditClass={handleEditClass}
          onOpenClass={handleSelectClass}
        />

        <ClassFormModal
          key={selectedClass?.id ?? "dashboard-class"}
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
