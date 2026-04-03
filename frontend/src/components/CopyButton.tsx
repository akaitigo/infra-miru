"use client";

import { useCallback, useState } from "react";
import { copyToClipboard } from "@/lib/clipboard";

interface CopyButtonProps {
  text: string;
}

export default function CopyButton({ text }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(text);
    setCopied(true);
    const timer = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timer);
  }, [text]);

  return (
    <button
      type="button"
      onClick={() => void handleCopy()}
      className="px-3 py-1 text-sm rounded bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 transition-colors"
      aria-label="Copy to clipboard"
    >
      {copied ? "Copied!" : "Copy"}
    </button>
  );
}
