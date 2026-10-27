"use client";

import { Plus } from "lucide-react";
import { useId, useMemo, useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type {
  TeacherClassDto,
  UpsertTeacherClassRequest,
} from "@/modules/classrooms/adapters/types/contracts";
import { INPUT_LIMITS, limitInput } from "@/shared/input-limits";

const subjectSuggestions = [
  "Mathematics",
  "English Language Arts",
  "Science",
  "Social Studies",
  "Computer Science",
  "World Languages",
  "Art",
  "Music",
  "Physical Education",
].sort();

const gradeLevelSuggestions = [
  "Pre-K",
  "Kindergarten",
  "1st grade",
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
  "Higher education",
  "Adult learning",
];

type ClassFormModalProps = {
  readonly error: string | null;
  readonly open: boolean;
  readonly saving: boolean;
  readonly teacherClass: TeacherClassDto | null;
  readonly onOpenChange: (open: boolean) => void;
  readonly onSubmit: (value: UpsertTeacherClassRequest) => Promise<void>;
};

export function ClassFormModal({
  error,
  onOpenChange,
  onSubmit,
  open,
  saving,
  teacherClass,
}: ClassFormModalProps) {
  const isEdit = Boolean(teacherClass);
  const formId = useId();
  const classNameInputId = useId();
  const subjectInputId = useId();
  const gradeLevelInputId = useId();
  const [comboboxPortalContainer, setComboboxPortalContainer] =
    useState<HTMLElement | null>(null);
  const [form, setForm] = useState<UpsertTeacherClassRequest>(() => ({
    class_id: teacherClass?.class_id ?? "",
    class_name: limitInput(
      teacherClass?.class_name ?? "",
      INPUT_LIMITS.className,
    ),
    subject: limitInput(teacherClass?.subject ?? "", INPUT_LIMITS.subject),
    grade_level: limitInput(
      teacherClass?.grade_level ?? "",
      INPUT_LIMITS.gradeLevel,
    ),
  }));

  const canSubmit = form.subject.trim().length >= 2 && !saving;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!canSubmit) {
      return;
    }

    await onSubmit({
      ...(teacherClass ? { class_id: teacherClass.class_id } : {}),
      class_name: limitInput(form.class_name.trim(), INPUT_LIMITS.className),
      grade_level: limitInput(form.grade_level.trim(), INPUT_LIMITS.gradeLevel),
      subject: limitInput(form.subject.trim(), INPUT_LIMITS.subject),
    });
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        ref={setComboboxPortalContainer}
        className="dashboard-theme sm:max-w-lg"
        onInteractOutside={(event) => event.preventDefault()}
        onPointerDownOutside={(event) => event.preventDefault()}
      >
        <DialogHeader className="text-start">
          <DialogTitle>{isEdit ? "Edit class" : "Add new class"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Update the class details. The sharing code stays the same."
              : "Add the class details. We will generate a short, unique sharing code after creation."}
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[min(560px,calc(100svh-12rem))] w-[calc(100%+0.75rem)] overflow-y-auto py-1 pe-3">
          <form
            id={formId}
            className="space-y-4 px-0.5"
            onSubmit={handleSubmit}
          >
            <FieldRow label="Class name" htmlFor={classNameInputId}>
              <Input
                id={classNameInputId}
                value={form.class_name}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    class_name: limitInput(
                      event.target.value,
                      INPUT_LIMITS.className,
                    ),
                  }))
                }
                maxLength={INPUT_LIMITS.className}
                placeholder="Algebra Lab"
                autoComplete="off"
                disabled={saving}
              />
            </FieldRow>

            <FieldRow label="Subject" htmlFor={subjectInputId}>
              <CreatableFieldCombobox
                id={subjectInputId}
                required
                maxLength={INPUT_LIMITS.subject}
                value={form.subject}
                onValueChange={(subject) =>
                  setForm((current) => ({ ...current, subject }))
                }
                suggestions={subjectSuggestions}
                placeholder="Mathematics"
                portalContainer={comboboxPortalContainer}
                disabled={saving}
              />
            </FieldRow>

            <FieldRow label="Grade level" htmlFor={gradeLevelInputId}>
              <CreatableFieldCombobox
                id={gradeLevelInputId}
                maxLength={INPUT_LIMITS.gradeLevel}
                value={form.grade_level}
                onValueChange={(gradeLevel) =>
                  setForm((current) => ({
                    ...current,
                    grade_level: gradeLevel,
                  }))
                }
                suggestions={gradeLevelSuggestions}
                placeholder="7th grade"
                portalContainer={comboboxPortalContainer}
                disabled={saving}
              />
            </FieldRow>

            {error ? (
              <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm font-medium text-destructive">
                {error}
              </p>
            ) : null}
          </form>
        </div>

        <DialogFooter>
          <Button type="submit" form={formId} disabled={!canSubmit}>
            {saving ? "Saving..." : isEdit ? "Save changes" : "Create class"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type CreatableOption = {
  readonly create?: boolean;
  readonly value: string;
};

function CreatableFieldCombobox({
  disabled,
  id,
  maxLength,
  onValueChange,
  placeholder,
  portalContainer,
  required,
  suggestions,
  value,
}: {
  readonly disabled: boolean;
  readonly id: string;
  readonly maxLength: number;
  readonly onValueChange: (value: string) => void;
  readonly placeholder: string;
  readonly portalContainer: HTMLElement | null;
  readonly required?: boolean;
  readonly suggestions: readonly string[];
  readonly value: string;
}) {
  const [createdValues, setCreatedValues] = useState<string[]>(() => {
    const initialValue = value.trim();
    const isSuggested = suggestions.some(
      (suggestion) =>
        suggestion.toLocaleLowerCase() === initialValue.toLocaleLowerCase(),
    );

    return initialValue && !isSuggested ? [initialValue] : [];
  });
  const availableValues = useMemo(
    () => [...suggestions, ...createdValues],
    [createdValues, suggestions],
  );
  const items = useMemo<CreatableOption[]>(() => {
    const suggestedItems = availableValues.map((suggestion) => ({
      value: suggestion,
    }));
    const trimmedValue = value.trim();
    const hasExactMatch = availableValues.some(
      (suggestion) =>
        suggestion.toLocaleLowerCase() === trimmedValue.toLocaleLowerCase(),
    );

    if (!trimmedValue || hasExactMatch) {
      return suggestedItems;
    }

    return [...suggestedItems, { create: true, value: trimmedValue }];
  }, [availableValues, value]);
  const selectedItem =
    items.find(
      (item) =>
        !item.create &&
        item.value.toLocaleLowerCase() === value.trim().toLocaleLowerCase(),
    ) ?? null;

  return (
    <Combobox
      autoHighlight
      disabled={disabled}
      inputValue={value}
      items={items}
      value={selectedItem}
      isItemEqualToValue={(item, selected) => item.value === selected.value}
      itemToStringLabel={(item) => item.value}
      itemToStringValue={(item) => item.value}
      onInputValueChange={(nextValue) =>
        onValueChange(limitInput(nextValue, maxLength))
      }
      onValueChange={(nextItem) => {
        if (nextItem) {
          commitValue(nextItem.value);
        }
      }}
    >
      <ComboboxInput
        id={id}
        required={required}
        placeholder={placeholder}
        autoComplete="off"
        maxLength={maxLength}
      />
      <ComboboxContent container={portalContainer}>
        <ComboboxEmpty>Type a new value to use it.</ComboboxEmpty>
        <ComboboxList>
          {(item: CreatableOption) => (
            <ComboboxItem
              key={`${item.create ? "create" : "option"}:${item.value}`}
              value={item}
            >
              {item.create ? (
                <>
                  <Plus className="size-4 text-primary" aria-hidden />
                  <span>
                    Use <strong>“{item.value}”</strong>
                  </span>
                </>
              ) : (
                item.value
              )}
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );

  function commitValue(nextValue: string) {
    const trimmedValue = nextValue.trim();
    if (!trimmedValue) {
      return;
    }

    setCreatedValues((current) => {
      const alreadyAvailable = [...suggestions, ...current].some(
        (item) => item.toLocaleLowerCase() === trimmedValue.toLocaleLowerCase(),
      );

      return alreadyAvailable ? current : [...current, trimmedValue];
    });
    onValueChange(limitInput(trimmedValue, maxLength));
  }
}

function FieldRow({
  children,
  htmlFor,
  label,
}: {
  readonly children: React.ReactNode;
  readonly htmlFor: string;
  readonly label: string;
}) {
  return (
    <div className="grid grid-cols-1 items-center gap-y-1 sm:grid-cols-6 sm:gap-x-4">
      <label
        className="text-sm font-medium text-foreground sm:col-span-2 sm:text-end"
        htmlFor={htmlFor}
      >
        {label}
      </label>
      <div className="sm:col-span-4">{children}</div>
    </div>
  );
}
