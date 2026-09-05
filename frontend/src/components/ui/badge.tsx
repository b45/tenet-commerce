import * as React from "react";
import { cn } from "@/lib/utils";

export interface BadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: "default" | "success" | "warning" | "danger" | "outline";
  dot?: boolean;
}

export function Badge({ className, variant = "default", dot = false, children, ...props }: BadgeProps) {
  const base =
    "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-semibold tracking-wide transition-colors border select-none";

  const variants = {
    default:
      "border-black/[0.06] bg-[#F5F5F7] text-[#555D6E]",
    success:
      "border-emerald-600/15 bg-emerald-50 text-emerald-800",
    warning:
      "border-amber-600/15 bg-amber-50 text-amber-900",
    danger:
      "border-rose-600/15 bg-rose-50 text-rose-800",
    outline:
      "border-black/[0.12] text-[#0B0F19] bg-white shadow-[0_1px_1px_rgba(0,0,0,0.02)]",
  };

  const dotColors = {
    default: "bg-slate-400",
    success: "bg-emerald-500",
    warning: "bg-amber-500",
    danger: "bg-rose-500",
    outline: "bg-[#0066CC]",
  };

  return (
    <div className={cn(base, variants[variant], className)} {...props}>
      {dot && <span className={cn("w-1.5 h-1.5 rounded-full", dotColors[variant])} />}
      {children}
    </div>
  );
}
