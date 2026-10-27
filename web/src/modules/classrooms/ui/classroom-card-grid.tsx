"use client";

import type { LucideIcon } from "lucide-react";
import {
  ArrowUpDown,
  ArrowUpRight,
  Atom,
  BookOpenText,
  Calculator,
  Check,
  ChevronDown,
  Dumbbell,
  Globe2,
  Landmark,
  Laptop2,
  LoaderCircle,
  Music2,
  Palette,
  Pencil,
  Plus,
  Search,
  Settings,
  Sparkles,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { formatDate } from "@/lib/date";
import type { TeacherClassDto } from "@/modules/classrooms/adapters/types/contracts";
import { INPUT_LIMITS, limitInput } from "@/shared/input-limits";

type ClassroomCardItem = {
  readonly activeMissionId: string;
  readonly missionCount: number;
  readonly classCode: string;
  readonly classId: string;
  readonly createdAt: string;
  readonly gradeLevel: string;
  readonly isPreview: boolean;
  readonly name: string;
  readonly subject: string;
  readonly updatedAt: string;
};

type ClassroomSort = "created" | "name" | "updated";

type ClassroomCardGridProps = {
  readonly loading: boolean;
  readonly teacherClasses: TeacherClassDto[];
  readonly onCreateClass: () => void;
  readonly onEditClass: (teacherClass: TeacherClassDto) => void;
  readonly onOpenClass: (teacherClass: TeacherClassDto) => void;
};

type SubjectTheme = {
  readonly icon: LucideIcon;
  readonly keywords: readonly string[];
};

type CardColor = {
  readonly background: string;
  readonly foreground: string;
  readonly label: string;
};

const cardColorStorageKey = "classgame:classroom-card-colors";

// Set this to a number such as 99 to fill the dashboard with preview classes.
const simulatedClassroomTotal: number | null = null;

const cardColors: readonly CardColor[] = [
  {
    background: "#9164f9",
    foreground: "#ffffff",
    label: "Violet",
  },
  {
    background: "#7e4bff",
    foreground: "#ffffff",
    label: "Purple",
  },
  {
    background: "#166a87",
    foreground: "#ffffff",
    label: "Ocean",
  },
  {
    background: "#0065d1",
    foreground: "#ffffff",
    label: "Blue",
  },
  {
    background: "#007ea8",
    foreground: "#ffffff",
    label: "Sky",
  },
  {
    background: "#007a74",
    foreground: "#ffffff",
    label: "Teal",
  },
  {
    background: "#24851a",
    foreground: "#ffffff",
    label: "Green",
  },
  {
    background: "#687600",
    foreground: "#ffffff",
    label: "Lime",
  },
  {
    background: "#a55f00",
    foreground: "#ffffff",
    label: "Amber",
  },
  {
    background: "#bd236f",
    foreground: "#ffffff",
    label: "Pink",
  },
];

const previewClassNames = [
  "Curious Thinkers",
  "Discovery Lab",
  "Bright Minds",
  "Problem Solvers",
  "Future Builders",
  "Creative Crew",
  "Young Explorers",
  "Trailblazers",
  "Idea Studio",
  "Learning League",
  "Wonder Workshop",
];

const previewSubjects = [
  "Mathematics",
  "English Language Arts",
  "Science",
  "Social Studies",
  "Computer Science",
  "World Languages",
  "Art",
  "Music",
  "Physical Education",
];

const previewGradeLevels = [
  "2nd grade",
  "3rd grade",
  "4th grade",
  "5th grade",
  "6th grade",
  "7th grade",
  "8th grade",
  "9th grade",
  "10th grade",
  "11th grade",
  "12th grade",
];

const subjectThemes: readonly SubjectTheme[] = [
  {
    icon: Calculator,
    keywords: ["math", "algebra", "geometry"],
  },
  {
    icon: BookOpenText,
    keywords: ["english", "language arts", "literature", "reading"],
  },
  {
    icon: Atom,
    keywords: ["science", "biology", "chemistry", "physics"],
  },
  {
    icon: Landmark,
    keywords: ["social", "history", "geography"],
  },
  {
    icon: Laptop2,
    keywords: ["computer", "technology", "coding"],
  },
  {
    icon: Globe2,
    keywords: ["world language", "spanish", "french", "german"],
  },
  {
    icon: Palette,
    keywords: ["art", "design"],
  },
  {
    icon: Music2,
    keywords: ["music"],
  },
  {
    icon: Dumbbell,
    keywords: ["physical", "health", "sports"],
  },
];

const fallbackTheme: SubjectTheme = {
  icon: Sparkles,
  keywords: [],
};

export function ClassroomCardGrid({
  loading,
  onCreateClass,
  onEditClass,
  onOpenClass,
  teacherClasses,
}: ClassroomCardGridProps) {
  const [query, setQuery] = useState("");
  const [sortBy, setSortBy] = useState<ClassroomSort>("updated");
  const [colorOverrides, setColorOverrides] = useState<Record<string, number>>(
    {},
  );
  const classrooms = useMemo(
    () =>
      buildClassroomCards(
        teacherClasses,
        simulatedClassroomTotal ?? teacherClasses.length,
      ),
    [teacherClasses],
  );
  const filteredClassrooms = useMemo(() => {
    const normalizedQuery = query.trim().toLocaleLowerCase();
    const matchingClassrooms = normalizedQuery
      ? classrooms.filter((classroom) =>
          [
            classroom.name,
            classroom.classCode,
            classroom.subject,
            classroom.gradeLevel,
          ]
            .join(" ")
            .toLocaleLowerCase()
            .includes(normalizedQuery),
        )
      : classrooms;

    return [...matchingClassrooms].sort((left, right) => {
      if (sortBy === "name") {
        return left.name.localeCompare(right.name, undefined, {
          sensitivity: "base",
        });
      }

      const dateKey = sortBy === "created" ? "createdAt" : "updatedAt";
      return toTimestamp(right[dateKey]) - toTimestamp(left[dateKey]);
    });
  }, [classrooms, query, sortBy]);
  const realClassesById = useMemo(
    () =>
      new Map(
        teacherClasses.map((teacherClass) => [
          teacherClass.class_id,
          teacherClass,
        ]),
      ),
    [teacherClasses],
  );

  useEffect(() => {
    const restoreColors = window.setTimeout(() => {
      try {
        const storedColors = window.localStorage.getItem(cardColorStorageKey);
        if (storedColors) {
          setColorOverrides(JSON.parse(storedColors) as Record<string, number>);
        }
      } catch {
        // Keep the deterministic subject colors when storage is unavailable or malformed.
      }
    }, 0);

    return () => window.clearTimeout(restoreColors);
  }, []);

  function handleColorChange(classId: string, colorIndex: number) {
    setColorOverrides((current) => {
      const next = { ...current, [classId]: colorIndex };
      try {
        window.localStorage.setItem(cardColorStorageKey, JSON.stringify(next));
      } catch {
        // The in-memory selection still works when browser storage is unavailable.
      }
      return next;
    });
  }

  return (
    <div className="space-y-5 sm:space-y-6">
      {classrooms.length > 0 ? (
        <div className="flex items-center justify-between gap-3">
          <h1 className="text-2xl font-bold tracking-tight text-primary sm:text-3xl">
            Your classes
          </h1>
          <Button onClick={onCreateClass}>
            <Plus className="size-4" aria-hidden />
            New class
          </Button>
        </div>
      ) : null}

      {classrooms.length > 0 ? (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex w-full min-w-0 gap-2 sm:max-w-xl">
            <div className="relative min-w-0 flex-1">
              <Search
                className="pointer-events-none absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                aria-hidden
              />
              <Input
                value={query}
                onChange={(event) =>
                  setQuery(limitInput(event.target.value, INPUT_LIMITS.search))
                }
                maxLength={INPUT_LIMITS.search}
                className="h-10 bg-background ps-9"
                placeholder="Find a classroom..."
                aria-label="Find a classroom"
              />
            </div>

            <div className="relative shrink-0">
              <ArrowUpDown
                className="pointer-events-none absolute start-3 top-1/2 z-10 size-3.5 -translate-y-1/2 text-muted-foreground"
                aria-hidden
              />
              <select
                value={sortBy}
                onChange={(event) =>
                  setSortBy(event.target.value as ClassroomSort)
                }
                className="h-10 w-[8.75rem] cursor-pointer appearance-none rounded-md border border-input bg-background ps-8 pe-8 text-xs font-medium outline-none transition focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 sm:w-40 sm:text-sm"
                aria-label="Sort classrooms"
              >
                <option value="name">Name</option>
                <option value="updated">Last updated</option>
                <option value="created">Last created</option>
              </select>
              <ChevronDown
                className="pointer-events-none absolute end-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
                aria-hidden
              />
            </div>
          </div>
          <p className="text-sm text-muted-foreground" aria-live="polite">
            {loading
              ? "Syncing live classes..."
              : `Showing ${filteredClassrooms.length} of ${classrooms.length}`}
          </p>
        </div>
      ) : null}

      {loading && classrooms.length === 0 ? (
        <section
          className="grid min-h-[calc(100svh-6rem)] place-items-center px-6 text-center sm:min-h-[calc(100svh-7rem)] md:min-h-[calc(100svh-8rem)]"
          aria-busy="true"
          aria-live="polite"
        >
          <div>
            <LoaderCircle
              className="mx-auto size-6 animate-spin text-muted-foreground"
              aria-hidden
            />
            <p className="mt-3 text-sm font-medium text-muted-foreground">
              Loading your classes...
            </p>
          </div>
        </section>
      ) : classrooms.length === 0 ? (
        <section className="grid min-h-[calc(100svh-6rem)] place-items-center rounded-2xl border border-dashed bg-muted/20 px-6 py-12 text-center sm:min-h-[calc(100svh-7rem)] md:min-h-[calc(100svh-8rem)]">
          <div className="max-w-sm">
            <span className="mx-auto grid size-12 place-items-center rounded-2xl bg-primary text-primary-foreground shadow-sm">
              <BookOpenText className="size-5" aria-hidden />
            </span>
            <h2 className="mt-5 text-xl font-bold tracking-tight text-foreground">
              Create your first class
            </h2>
            <p className="mt-2 text-sm leading-6 text-muted-foreground">
              Add a subject and grade level. We will create a short code your
              students can easily enter.
            </p>
            <Button className="mt-6" onClick={onCreateClass}>
              <Plus className="size-4" aria-hidden />
              Add new class
            </Button>
          </div>
        </section>
      ) : filteredClassrooms.length > 0 ? (
        <section
          className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5"
          aria-label="Teacher classrooms"
        >
          {filteredClassrooms.map((classroom) => {
            const teacherClass = realClassesById.get(classroom.classId);
            return (
              <ClassroomCard
                key={classroom.classId}
                classroom={classroom}
                colorIndex={
                  colorOverrides[classroom.classId] ??
                  getDefaultColorIndex(classroom.subject)
                }
                onColorChange={(colorIndex) =>
                  handleColorChange(classroom.classId, colorIndex)
                }
                onEdit={
                  teacherClass ? () => onEditClass(teacherClass) : undefined
                }
                onOpen={
                  teacherClass ? () => onOpenClass(teacherClass) : undefined
                }
              />
            );
          })}
        </section>
      ) : (
        <div className="rounded-2xl border border-dashed bg-muted/20 px-6 py-16 text-center">
          <p className="font-semibold text-foreground">
            No classrooms match “{query}”.
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Try a subject, grade, name, or class code.
          </p>
        </div>
      )}
    </div>
  );
}

function ClassroomCard({
  classroom,
  colorIndex,
  onColorChange,
  onEdit,
  onOpen,
}: {
  readonly classroom: ClassroomCardItem;
  readonly colorIndex: number;
  readonly onColorChange: (colorIndex: number) => void;
  readonly onEdit?: () => void;
  readonly onOpen?: () => void;
}) {
  const theme = resolveSubjectTheme(classroom.subject);
  const SubjectIcon = theme.icon;
  const cardColor = cardColors[colorIndex] ?? cardColors[0];
  const missionStatus = getMissionStatus(classroom);

  return (
    <Card
      className="group relative min-h-52 gap-0 overflow-visible border-white/10 py-0 text-start shadow-sm transition-[border-color,box-shadow] duration-200 ease-out before:pointer-events-none before:absolute before:inset-0 before:rounded-lg before:bg-white/10 before:opacity-0 before:transition-opacity before:duration-200 before:ease-out hover:border-white/20 hover:shadow-lg hover:before:opacity-100"
      style={{
        backgroundColor: cardColor.background,
        color: cardColor.foreground,
      }}
    >
      <CardHeader className="gap-3 px-4 pb-2 pt-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2">
            <span className="grid size-9 place-items-center border-white/10">
              <SubjectIcon className="size-4.5" aria-hidden />
            </span>
            <span className="rounded-full border-white/10 px-0 py-1 text-[10px] font-bold">
              {missionStatus.label}
            </span>
          </div>

          <ClassroomSettings
            colorIndex={colorIndex}
            isPreview={classroom.isPreview}
            name={classroom.name}
            onColorChange={onColorChange}
            onEdit={onEdit}
          />
        </div>

        <div>
          <CardTitle className="line-clamp-1 text-lg leading-5 text-current">
            {classroom.name}
          </CardTitle>
          <CardDescription className="mt-1 line-clamp-1 text-current opacity-75">
            {classroom.subject} · {classroom.gradeLevel}
          </CardDescription>
        </div>
      </CardHeader>

      <CardContent className="mt-auto px-4 pb-3">
        <div className="border-white/10 py-2.5">
          <div className="flex items-end justify-between gap-3">
            <div>
              <p className="text-[9px] font-bold opacity-65">Class code</p>
              <p className="mt-0.5 font-mono text-base font-bold tracking-[0.12em]">
                {classroom.classCode}
              </p>
            </div>
            <div className="text-end">
              <p className="text-[10px] font-bold  opacity-65">
                {classroom.missionCount === 1 ? "mission" : "missions"}
              </p>
              <p className="text-base font-bold">{classroom.missionCount}</p>
            </div>
          </div>
        </div>
        <div className="text-[11px]">
          <span className="opacity-70">
            {classroom.isPreview
              ? "Layout preview"
              : `Updated ${formatDate(classroom.updatedAt, {
                  fallback: "recently",
                  preset: "day-month",
                  relative: true,
                })}`}
          </span>
        </div>
      </CardContent>

      <div className="flex-col items-stretch gap-2 rounded-b-lg border-t border-white/10 px-4 py-2.5">
        {onOpen ? (
          <button
            type="button"
            className="flex h-10 w-full cursor-pointer items-center justify-center gap-1 rounded-md px-1.5 font-bold outline-none transition hover:bg-white/10 focus-visible:ring-2 focus-visible:ring-white/70 pt-0"
            onClick={onOpen}
            aria-label={`Open ${classroom.name}`}
          >
            Open <ArrowUpRight className="size-3.5" aria-hidden />
          </button>
        ) : (
          <span className="self-start rounded-full border border-white/10 px-2 py-0.5 font-bold">
            Demo
          </span>
        )}
      </div>
    </Card>
  );
}

function ClassroomSettings({
  colorIndex,
  isPreview,
  name,
  onColorChange,
  onEdit,
}: {
  readonly colorIndex: number;
  readonly isPreview: boolean;
  readonly name: string;
  readonly onColorChange: (colorIndex: number) => void;
  readonly onEdit?: () => void;
}) {
  return (
    <details
      className="relative z-30"
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) {
          event.currentTarget.removeAttribute("open");
        }
      }}
    >
      <summary
        className="grid size-8 cursor-pointer list-none place-items-center rounded-lg  outline-none transition hover:brightness-105 focus-visible:ring-2 focus-visible:ring-current [&::-webkit-details-marker]:hidden"
        aria-label={`Settings for ${name}`}
      >
        <Settings className="size-4" aria-hidden />
      </summary>

      <div className="absolute end-0 top-10 z-50 w-56 rounded-xl border border-border bg-popover p-2 text-popover-foreground shadow-xl">
        <button
          type="button"
          className="flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-sm font-medium hover:bg-accent disabled:cursor-not-allowed disabled:opacity-45"
          disabled={!onEdit}
          onClick={(event) => {
            event.currentTarget.closest("details")?.removeAttribute("open");
            onEdit?.();
          }}
        >
          <Pencil className="size-4" aria-hidden />
          {isPreview ? "Demo class" : "Edit class"}
        </button>

        <div className="my-2 h-px bg-border" />
        <p className="px-2.5 pb-2 text-xs font-semibold text-muted-foreground">
          Card color
        </p>
        <div className="grid grid-cols-5 gap-2 px-2 pb-2">
          {cardColors.map((color, index) => (
            <button
              key={color.background}
              type="button"
              className="grid size-7 place-items-center rounded-full border border-black/10 outline-none ring-offset-2 hover:scale-110 focus-visible:ring-2 focus-visible:ring-ring"
              style={{
                backgroundColor: color.background,
                color: color.foreground,
              }}
              onClick={() => onColorChange(index)}
              aria-label={`Use ${color.label}`}
              aria-pressed={colorIndex === index}
            >
              {colorIndex === index ? (
                <Check className="size-3.5" strokeWidth={3} aria-hidden />
              ) : null}
            </button>
          ))}
        </div>
      </div>
    </details>
  );
}

function buildClassroomCards(
  teacherClasses: TeacherClassDto[],
  total: number,
): ClassroomCardItem[] {
  const realClassrooms = teacherClasses.slice(0, total).map((teacherClass) => ({
    activeMissionId: teacherClass.active_mission_id,
    missionCount: teacherClass.mission_count,
    classCode: teacherClass.class_code,
    classId: teacherClass.class_id,
    createdAt: teacherClass.created_at,
    gradeLevel: teacherClass.grade_level || "Grade not set",
    isPreview: false,
    name: teacherClass.class_name || teacherClass.class_code,
    subject: teacherClass.subject || "General learning",
    updatedAt: teacherClass.updated_at,
  }));
  const previewCount = Math.max(0, total - realClassrooms.length);
  const previewClassrooms = Array.from({ length: previewCount }, (_, index) => {
    const subject = previewSubjects[index % previewSubjects.length];
    const missionCount = (index * 3) % 8;

    return {
      activeMissionId: index % 3 === 0 ? `preview-mission-${index + 1}` : "",
      missionCount: index % 3 === 0 ? Math.max(missionCount, 1) : missionCount,
      classCode: createPreviewCode(index),
      classId: `preview-class-${index + 1}`,
      createdAt: createPreviewDate(index, 1),
      gradeLevel: previewGradeLevels[(index * 2) % previewGradeLevels.length],
      isPreview: true,
      name: `${previewClassNames[index % previewClassNames.length]} ${Math.floor(index / previewClassNames.length) + 1}`,
      subject,
      updatedAt: createPreviewDate(index, 7),
    } satisfies ClassroomCardItem;
  });

  return [...realClassrooms, ...previewClassrooms];
}

function createPreviewCode(index: number) {
  const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ";
  let value = index + 4_096;
  let code = "";

  while (code.length < 6) {
    code = `${alphabet[value % alphabet.length]}${code}`;
    value = Math.floor(value / alphabet.length);
  }

  return code;
}

function createPreviewDate(index: number, interval: number) {
  const previewBaseDate = Date.UTC(2026, 7, 28, 12);
  const dayInMilliseconds = 86_400_000;
  const daysAgo = (index * interval) % 90;
  return new Date(previewBaseDate - daysAgo * dayInMilliseconds).toISOString();
}

function toTimestamp(value: string) {
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

function resolveSubjectTheme(subject: string) {
  const normalizedSubject = subject.toLocaleLowerCase();
  return (
    subjectThemes.find((theme) =>
      theme.keywords.some((keyword) => normalizedSubject.includes(keyword)),
    ) ?? fallbackTheme
  );
}

function getDefaultColorIndex(subject: string) {
  const normalizedSubject = subject.toLocaleLowerCase();
  const themeIndex = subjectThemes.findIndex((theme) =>
    theme.keywords.some((keyword) => normalizedSubject.includes(keyword)),
  );

  return themeIndex >= 0 ? themeIndex : cardColors.length - 1;
}

function getMissionStatus(classroom: ClassroomCardItem) {
  if (classroom.activeMissionId) {
    return {
      className: "border-emerald-200 bg-emerald-50 text-emerald-700",
      label: "Ready to play",
    };
  }

  if (classroom.missionCount > 0) {
    return {
      className: "border-amber-200 bg-amber-50 text-amber-700",
      label: "Needs publishing",
    };
  }

  return {
    className: "border-slate-200 bg-slate-50 text-slate-600",
    label: "Needs mission",
  };
}
