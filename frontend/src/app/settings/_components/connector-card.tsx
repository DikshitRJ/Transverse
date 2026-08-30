"use client";

import { useState, type ReactNode } from "react";
import { useMutation } from "@tanstack/react-query";
import { CheckCircle2Icon, LoaderCircleIcon, XIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { describeError } from "@/components/dashboard/async-panels";
import type { EvidenceConnectorResponse } from "@/lib/api/types";
import { clearConnectorRecord, saveConnectorRecord, type ConnectorKind, type ConnectorRecord } from "../_lib/connector-store";

export function ConnectorCard({
  kind,
  label,
  description,
  fieldLabel,
  placeholder,
  icon,
  initialRecord,
  connect,
}: {
  kind: ConnectorKind;
  label: string;
  description: string;
  fieldLabel: string;
  placeholder: string;
  icon: ReactNode;
  initialRecord: ConnectorRecord | undefined;
  connect: (value: string) => Promise<EvidenceConnectorResponse>;
}) {
  const [value, setValue] = useState("");
  const [record, setRecord] = useState(initialRecord);

  const mutation = useMutation({
    mutationFn: connect,
    onSuccess: (res, submittedValue) => {
      const next: ConnectorRecord = {
        identifier: submittedValue,
        evidenceId: res.evidence_id,
        queuedAt: new Date().toISOString(),
      };
      saveConnectorRecord(kind, next);
      setRecord(next);
      setValue("");
    },
  });

  function handleDisconnect() {
    clearConnectorRecord(kind);
    setRecord(undefined);
  }

  return (
    <div className="glass-panel flex flex-col gap-3 rounded-tv-card border border-tv-border p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-tv-btn bg-tv-surface-2 text-tv-cyan">
            {icon}
          </div>
          <div>
            <h3 className="font-heading text-sm font-medium text-tv-text-hi">{label}</h3>
            <p className="font-body text-xs text-tv-text-body">{description}</p>
          </div>
        </div>
        {record && (
          <Badge variant="success" className="shrink-0">
            <CheckCircle2Icon className="size-3" aria-hidden="true" />
            Sync queued
          </Badge>
        )}
      </div>

      {record ? (
        <div className="flex items-center justify-between rounded-tv-btn border border-tv-border bg-tv-surface-deep px-3 py-2">
          <div>
            <p className="font-mono text-sm text-tv-text-hi">{record.identifier}</p>
            <p className="font-body text-xs text-tv-text-body">
              Requested {new Date(record.queuedAt).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" })}
            </p>
          </div>
          <Button variant="ghost" size="icon-sm" aria-label={`Disconnect ${label}`} onClick={handleDisconnect}>
            <XIcon />
          </Button>
        </div>
      ) : (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (value.trim()) mutation.mutate(value.trim());
          }}
          className="flex items-end gap-2"
        >
          <div className="flex flex-1 flex-col gap-1">
            <Label htmlFor={`connector-${kind}`} className="text-xs">
              {fieldLabel}
            </Label>
            <Input
              id={`connector-${kind}`}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={placeholder}
              disabled={mutation.isPending}
            />
          </div>
          <Button type="submit" variant="outline-cyan" size="sm" disabled={mutation.isPending || !value.trim()}>
            {mutation.isPending ? <LoaderCircleIcon className="animate-spin" /> : "Connect"}
          </Button>
        </form>
      )}

      {mutation.isError && (
        <p role="alert" className="font-body text-xs text-tv-rose">
          {describeError(mutation.error)}
        </p>
      )}
    </div>
  );
}
