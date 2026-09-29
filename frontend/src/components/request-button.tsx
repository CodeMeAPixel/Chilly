"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Check, HandHeart, Loader2 } from "lucide-react";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { useMe } from "@/hooks/use-me";
import { api, ApiError, json, loginUrl } from "@/lib/api";
import { cn } from "@/lib/format";
import type { SongRequest } from "@/lib/types";
import { Button } from "./ui";

export function RequestButton({
  songKey,
  title,
  compact,
  className,
}: {
  songKey: string;
  title: string;
  compact?: boolean;
  className?: string;
}) {
  const { data: me } = useMe();
  const pathname = usePathname();
  const queryClient = useQueryClient();
  const [state, setState] = useState<"idle" | "busy" | "done">("idle");

  const request = async () => {
    if (!me) {
      window.location.href = loginUrl(pathname);
      return;
    }
    setState("busy");
    try {
      const res = await api<{ request: SongRequest }>("/requests", { method: "POST", body: json({ key: songKey }) });
      toast.success(`Requested ${title}`, { description: "It'll play on the station soon. Follow it under My requests." });
      setState("done");
      queryClient.invalidateQueries({ queryKey: ["my-requests"] });
      return res;
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Couldn't send that request");
      setState("idle");
    }
  };

  const label = state === "done" ? "Requested" : "Request";
  const icon =
    state === "busy" ? (
      <Loader2 className="h-4 w-4 animate-spin" />
    ) : state === "done" ? (
      <Check className="h-4 w-4" />
    ) : (
      <HandHeart className="h-4 w-4" />
    );

  if (compact) {
    return (
      <Button
        variant="ghost"
        size="icon"
        className={cn("h-8 w-8", state === "done" && "text-lime", className)}
        onClick={request}
        disabled={state !== "idle"}
        aria-label={`Request ${title} on the station`}
        title="Request on the station"
      >
        {icon}
      </Button>
    );
  }
  return (
    <Button variant={state === "done" ? "secondary" : "ghost"} size="sm" className={className} onClick={request} disabled={state !== "idle"}>
      {icon} {label}
    </Button>
  );
}
