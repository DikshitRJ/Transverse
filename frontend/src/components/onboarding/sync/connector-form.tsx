"use client";

import { useId, useState, type FormEvent } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import type { ConnectorKind } from "@/components/onboarding/sync/types";

const CONNECTOR_COPY: Record<ConnectorKind, { label: string; placeholder: string; fieldLabel: string }> = {
  github: { label: "GitHub", placeholder: "octocat", fieldLabel: "GitHub username" },
  leetcode: { label: "LeetCode", placeholder: "leetcode_user", fieldLabel: "LeetCode username" },
  codeforces: { label: "Codeforces", placeholder: "tourist", fieldLabel: "Codeforces handle" },
};

export function ConnectorForm({
  kind,
  onSubmit,
  disabled,
}: {
  kind: ConnectorKind;
  onSubmit: (value: string) => void;
  disabled?: boolean;
}) {
  const [value, setValue] = useState("");
  const id = useId();
  const copy = CONNECTOR_COPY[kind];

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed) return;
    onSubmit(trimmed);
    setValue("");
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-1.5">
      <Label htmlFor={id} className="font-mono text-xs text-tv-text-body">
        {copy.fieldLabel}
      </Label>
      <div className="flex gap-2">
        <Input
          id={id}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={copy.placeholder}
          disabled={disabled}
          className="border-tv-border bg-tv-surface text-tv-text-hi"
        />
        <Button type="submit" variant="outline-cyan" size="default" disabled={disabled || !value.trim()} className="normal-case">
          Connect
        </Button>
      </div>
    </form>
  );
}
