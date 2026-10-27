import type { Metadata } from "next";

import { GamePage } from "@/modules/game";

export const metadata: Metadata = {
  title: "ClassGame | Student mission",
};

type HomePageProps = {
  readonly searchParams: Promise<{
    readonly classCode?: string | string[];
  }>;
};

export default async function Page({ searchParams }: HomePageProps) {
  const params = await searchParams;
  const rawClassCode = Array.isArray(params.classCode)
    ? params.classCode[0]
    : params.classCode;
  const initialClassCode = (rawClassCode ?? "").replace(/\D/g, "").slice(0, 6);

  return <GamePage initialClassCode={initialClassCode} />;
}
