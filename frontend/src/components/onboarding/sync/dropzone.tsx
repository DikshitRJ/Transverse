"use client";

import { useId, useRef, useState } from "react";
import Image from "next/image";
import { cn } from "@/lib/utils";
import type { UploadKind } from "@/components/onboarding/sync/types";

interface DropzoneProps {
  kind: UploadKind;
  title: string;
  hint: string;
  accept: string;
  onFile: (file: File) => void;
  disabled?: boolean;
}

/**
 * Drag-and-drop resume/codebase upload target. Net-new UI (no Figma source
 * for `/onboarding/sync`) built from the frozen token set, reusing the
 * `icon-upload-cloud.svg` decoration already exported for the chooser's
 * "Sync Past Experiences" card so the visual language stays consistent.
 */
export function Dropzone({ kind, title, hint, accept, onFile, disabled }: DropzoneProps) {
  const [isDragOver, setIsDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const inputId = useId();

  function handleFiles(files: FileList | null) {
    const file = files?.[0];
    if (file) onFile(file);
  }

  return (
    <div
      onDragOver={(e) => {
        e.preventDefault();
        if (!disabled) setIsDragOver(true);
      }}
      onDragLeave={() => setIsDragOver(false)}
      onDrop={(e) => {
        e.preventDefault();
        setIsDragOver(false);
        if (!disabled) handleFiles(e.dataTransfer.files);
      }}
      className={cn(
        "relative flex flex-col items-center gap-2 rounded-tv-card border border-dashed p-6 text-center transition-colors",
        isDragOver ? "border-tv-cyan bg-tv-cyan/5" : "border-tv-border bg-tv-surface",
        disabled && "pointer-events-none opacity-50",
      )}
      data-kind={kind}
    >
      <Image src="/figma/icon-upload-cloud.svg" alt="" width={48} height={40} className="opacity-80" />
      <p className="font-display text-sm font-bold text-tv-text-hi uppercase">{title}</p>
      <p className="font-body text-xs text-tv-text-body">{hint}</p>
      <label
        htmlFor={inputId}
        className="mt-2 cursor-pointer rounded-tv-btn border-2 border-tv-cyan px-4 py-2 font-mono text-xs font-semibold text-tv-cyan transition-colors hover:bg-tv-cyan/10"
      >
        Browse file
      </label>
      <input
        ref={inputRef}
        id={inputId}
        type="file"
        accept={accept}
        disabled={disabled}
        className="sr-only"
        onChange={(e) => {
          handleFiles(e.target.files);
          e.target.value = "";
        }}
      />
    </div>
  );
}
