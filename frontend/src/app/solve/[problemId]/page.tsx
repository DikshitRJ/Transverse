import type { Metadata } from "next";
import { SolveWorkspace } from "@/components/ide/solve-workspace";

export const metadata: Metadata = {
  title: "Solve — Transverse",
};

function firstParam(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

export default async function SolvePage({ params, searchParams }: PageProps<"/solve/[problemId]">) {
  const { problemId } = await params;
  const sp = await searchParams;
  const sessionId = firstParam(sp.session);

  return <SolveWorkspace problemId={problemId} sessionId={sessionId} />;
}
