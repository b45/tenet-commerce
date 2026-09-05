import * as React from "react";
import { cn } from "@/lib/utils";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: boolean;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, error, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-[42px] w-full rounded-[11px] border bg-white px-3.5 py-2 text-[14px] text-[#0B0F19] tracking-tight",
          "placeholder:text-[#8B95A5] transition-all duration-200 ease-[cubic-bezier(0.16,1,0.3,1)]",
          "border-black/[0.10] shadow-[0_1px_2px_rgba(0,0,0,0.02)]",
          "focus-visible:outline-none focus:border-[#0066CC] focus:ring-4 focus:ring-[#0066CC]/12",
          "disabled:cursor-not-allowed disabled:opacity-40 disabled:bg-[#F5F5F7]",
          error && "border-[#D92D20] focus:border-[#D92D20] focus:ring-[#D92D20]/15",
          className
        )}
        ref={ref}
        {...props}
      />
    );
  }
);
Input.displayName = "Input";
