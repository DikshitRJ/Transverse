"use client";

import Link from "next/link";
import { useState } from "react";
import { ExternalLinkIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ScrollArea } from "@/components/ui/scroll-area";
import { formatTopicLabel } from "@/components/charts/chart-theme";
import { DIFFICULTY_BADGE_VARIANT } from "../_lib/difficulty";
import { SanitizedHtml } from "./sanitized-html";
import type { ProblemPayload, ScrapedProblem, TestCase } from "@/lib/api/types";

export type ProblemPreviewSource =
  | { kind: "problem"; problem: ProblemPayload }
  | { kind: "scraped"; scraped: ScrapedProblem; matchedId?: string };

function TestCaseList({ testCases }: { testCases: TestCase[] }) {
  if (testCases.length === 0) {
    return <p className="font-body text-sm text-tv-text-body">No test cases available for this problem.</p>;
  }
  return (
    <div className="flex flex-col gap-3">
      {testCases.map((tc, i) => (
        <div key={i} className="rounded-tv-btn border border-tv-border bg-tv-surface-deep p-3">
          <div className="mb-1 flex items-center gap-2">
            <span className="font-mono text-[11px] text-tv-text-body uppercase">Case {i + 1}</span>
            {tc.is_hidden && (
              <Badge variant="locked" className="text-[10px]">
                Hidden
              </Badge>
            )}
          </div>
          <pre className="overflow-x-auto font-mono text-xs text-tv-text-hi">
            <span className="text-tv-text-body">Input: </span>
            {tc.input}
          </pre>
          <pre className="overflow-x-auto font-mono text-xs text-tv-text-hi">
            <span className="text-tv-text-body">Output: </span>
            {tc.output}
          </pre>
          {tc.explanation && <p className="mt-1 font-body text-xs text-tv-text-body">{tc.explanation}</p>}
        </div>
      ))}
    </div>
  );
}

function TemplatesTabs({ templates }: { templates: Record<string, string> | undefined }) {
  const entries = Object.entries(templates ?? {});
  const [active, setActive] = useState(entries[0]?.[0]);
  if (entries.length === 0) return null;

  return (
    <Tabs value={active} onValueChange={setActive}>
      <TabsList className="flex-wrap">
        {entries.map(([lang]) => (
          <TabsTrigger key={lang} value={lang} className="font-mono">
            {lang}
          </TabsTrigger>
        ))}
      </TabsList>
      {entries.map(([lang, code]) => (
        <TabsContent key={lang} value={lang}>
          <pre className="max-h-64 overflow-auto rounded-tv-btn border border-tv-border bg-tv-surface-deep p-3 font-mono text-xs text-tv-text-hi">
            {code}
          </pre>
        </TabsContent>
      ))}
    </Tabs>
  );
}

export function ProblemPreviewDialog({
  source,
  onOpenChange,
}: {
  source: ProblemPreviewSource | null;
  onOpenChange: (open: boolean) => void;
}) {
  const open = source !== null;

  const title = source?.kind === "problem" ? source.problem.name : (source?.scraped.title ?? "");
  const sourceLabel = source?.kind === "problem" ? source.problem.source : source?.scraped.source;
  const difficulty = source?.kind === "problem" ? source.problem.difficulty_label : source?.scraped.difficulty;
  const tags = source?.kind === "problem" ? source.problem.tags : (source?.scraped.tags ?? []);
  const statement = source?.kind === "problem" ? source.problem.statement : source?.scraped.statement;
  const testCases = source?.kind === "problem" ? (source.problem.test_cases ?? []) : source?.scraped.test_cases;
  const templates = source?.kind === "problem" ? source.problem.templates : source?.scraped.templates;
  const solvableId = source?.kind === "problem" ? source.problem.id : source?.matchedId;
  const externalUrl = source?.kind === "problem" ? source.problem.url : source?.scraped.url;
  const topic = source?.kind === "problem" ? source.problem.topic : undefined;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-2xl overflow-hidden sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="font-heading text-h4 text-tv-text-hi">{title}</DialogTitle>
          <DialogDescription render={<div />} className="flex flex-wrap items-center gap-2">
            {sourceLabel && (
              <Badge variant="outline" className="capitalize">
                {sourceLabel}
              </Badge>
            )}
            {difficulty && (
              <Badge variant={DIFFICULTY_BADGE_VARIANT[difficulty] ?? "outline"} className="capitalize">
                {difficulty}
              </Badge>
            )}
            {topic && <Badge variant="secondary">{formatTopicLabel(topic)}</Badge>}
            {tags?.slice(0, 4).map((tag) => (
              <Badge key={tag} variant="ghost" className="capitalize">
                {tag}
              </Badge>
            ))}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="h-[55vh] pr-3">
          <div className="flex flex-col gap-5">
            {source?.kind === "scraped" && !solvableId && (
              <div className="rounded-tv-btn border border-tv-warning/30 bg-tv-warning/10 px-3 py-2 font-body text-xs text-tv-warning">
                This was scraped for preview only — it isn&apos;t saved to the problem bank yet, so solving it
                in-app isn&apos;t available. Open it on the source site instead, or search the bank for it later.
              </div>
            )}
            <SanitizedHtml html={statement} />
            {testCases && testCases.length > 0 && (
              <div>
                <p className="mb-2 font-mono text-xs tracking-wide text-tv-text-body uppercase">Test cases</p>
                <TestCaseList testCases={testCases} />
              </div>
            )}
            {templates && Object.keys(templates).length > 0 && (
              <div>
                <p className="mb-2 font-mono text-xs tracking-wide text-tv-text-body uppercase">Starter code</p>
                <TemplatesTabs templates={templates} />
              </div>
            )}
          </div>
        </ScrollArea>

        <div className="flex flex-wrap justify-end gap-2 border-t border-tv-border pt-3">
          {externalUrl && (
            <Button render={<a href={externalUrl} target="_blank" rel="noopener noreferrer" />} variant="ghost" size="sm">
              <ExternalLinkIcon />
              Open source
            </Button>
          )}
          {solvableId && (
            <Button render={<Link href={`/solve/${solvableId}`} />} size="sm">
              Solve this problem
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
