import type { ReactNode } from "react";

import { TeacherAuthProvider } from "@/modules/identity";

export default function PagesLayout({
  children,
}: {
  readonly children: ReactNode;
}) {
  return <TeacherAuthProvider>{children}</TeacherAuthProvider>;
}
