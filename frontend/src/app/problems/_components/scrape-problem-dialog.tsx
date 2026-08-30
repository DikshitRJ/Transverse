"use client";

import { useState, type ReactElement } from "react";
import { useMutation } from "@tanstack/react-query";
import { LinkIcon, LoaderCircleIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { scrapeProblem } from "@/lib/api";
import { describeError } from "@/components/dashboard/async-panels";
import type { ScrapedProblem } from "@/lib/api/types";

export function ScrapeProblemDialog({
  trigger,
  onScraped,
}: {
  trigger: ReactElement;
  onScraped: (scraped: ScrapedProblem) => void;
}) {
  const [open, setOpen] = useState(false);
  const [url, setUrl] = useState("");

  const mutation = useMutation({
    mutationFn: (u: string) => scrapeProblem(u),
    onSuccess: (scraped) => {
      setOpen(false);
      setUrl("");
      onScraped(scraped);
    },
  });

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) mutation.reset();
      }}
    >
      <DialogTrigger render={trigger} />
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Scrape a problem by URL</DialogTitle>
          <DialogDescription>
            Paste a LeetCode or Codeforces problem link — we&apos;ll pull the statement, test cases, and starter
            templates for a quick preview.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (url.trim()) mutation.mutate(url.trim());
          }}
          className="flex flex-col gap-3"
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="scrape-url">Problem URL</Label>
            <Input
              id="scrape-url"
              type="url"
              placeholder="https://leetcode.com/problems/two-sum/"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              required
              disabled={mutation.isPending}
            />
          </div>

          {mutation.isError && (
            <p role="alert" className="font-body text-xs text-tv-rose">
              {describeError(mutation.error)}
            </p>
          )}

          <DialogFooter>
            <Button type="submit" disabled={mutation.isPending || !url.trim()}>
              {mutation.isPending ? (
                <>
                  <LoaderCircleIcon className="animate-spin" />
                  Scraping…
                </>
              ) : (
                <>
                  <LinkIcon />
                  Scrape & preview
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
