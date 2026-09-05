"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

export interface SegmentedOption<T extends string = string> {
  value: T;
  label: string;
  badge?: string | number;
}

interface SegmentedControlProps<T extends string = string> {
  options: SegmentedOption<T>[];
  value: T;
  onChange: (value: T) => void;
  className?: string;
  size?: "sm" | "default";
}

export function SegmentedControl<T extends string = string>({
  options,
  value,
  onChange,
  className,
  size = "default",
}: SegmentedControlProps<T>) {
  return (
    <div
      role="tablist"
      className={cn(
        "inline-flex items-center p-1 rounded-[12px] bg-[#EBEBEF] border border-black/[0.04]",
        size === "sm" ? "h-8 text-xs" : "h-10 text-[13px]",
        className
      )}
    >
      {options.map((opt) => {
        const isSelected = opt.value === value;
        return (
          <button
            key={opt.value}
            role="tab"
            aria-selected={isSelected}
            type="button"
            onClick={() => onChange(opt.value)}
            className={cn(
              "relative z-10 inline-flex items-center justify-center font-medium tracking-tight rounded-[9px] px-3.5 transition-all duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] select-none",
              size === "sm" ? "h-6 text-xs" : "h-8 text-[13px]",
              isSelected
                ? "bg-white text-[#0B0F19] font-semibold shadow-[0_1px_3px_rgba(0,0,0,0.08),0_1px_1px_rgba(0,0,0,0.04)]"
                : "text-[#555D6E] hover:text-[#0B0F19]"
            )}
          >
            <span>{opt.label}</span>
            {opt.badge !== undefined && (
              <span
                className={cn(
                  "ml-1.5 px-1.5 py-0.2 rounded-full text-[10px] font-semibold",
                  isSelected ? "bg-[#F0F0F3] text-[#0B0F19]" : "bg-black/[0.06] text-[#555D6E]"
                )}
              >
                {opt.badge}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
