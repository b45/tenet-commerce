import * as React from "react";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "outline" | "ghost" | "destructive";
  size?: "default" | "sm" | "lg" | "icon";
  isLoading?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "primary", size = "default", isLoading = false, children, disabled, ...props }, ref) => {
    const baseStyles =
      "inline-flex items-center justify-center font-medium tracking-tight select-none " +
      "transition-all duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] " +
      "focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-[#0066CC]/25 " +
      "disabled:pointer-events-none disabled:opacity-40 active:scale-[0.985]";

    const variantStyles = {
      primary:
        "bg-[#0066CC] text-white hover:bg-[#0055B3] active:bg-[#004C9E] " +
        "shadow-[0_1px_2px_rgba(0,0,0,0.08),inset_0_1px_0_rgba(255,255,255,0.22)] " +
        "border border-[#0055B3]/50",
      secondary:
        "bg-[#F5F5F7] text-[#0B0F19] hover:bg-[#EBEBF0] active:bg-[#E1E1E6] " +
        "border border-black/[0.06] shadow-[0_1px_1px_rgba(0,0,0,0.02)]",
      outline:
        "border border-black/[0.10] bg-white text-[#0B0F19] hover:bg-[#F5F5F7] " +
        "shadow-[0_1px_2px_rgba(0,0,0,0.02)]",
      ghost:
        "text-[#555D6E] hover:text-[#0B0F19] hover:bg-black/[0.04] active:bg-black/[0.07]",
      destructive:
        "bg-[#D92D20] text-white hover:bg-[#C02619] active:bg-[#A81F13] " +
        "shadow-[0_1px_2px_rgba(0,0,0,0.08),inset_0_1px_0_rgba(255,255,255,0.2)] " +
        "border border-[#C02619]/60",
    };

    const sizeStyles = {
      default: "h-10 px-4 text-[13px] rounded-[11px] gap-2 font-semibold",
      sm: "h-8 px-3 text-[12px] rounded-[9px] gap-1.5 font-medium",
      lg: "h-11 px-5 text-[14px] rounded-[13px] gap-2.5 font-semibold",
      icon: "h-10 w-10 p-0 rounded-[11px] justify-center",
    };

    return (
      <button
        ref={ref}
        disabled={disabled || isLoading}
        className={cn(baseStyles, variantStyles[variant], sizeStyles[size], className)}
        {...props}
      >
        {isLoading && <Loader2 className="h-4 w-4 animate-spin text-current" />}
        {children}
      </button>
    );
  }
);
Button.displayName = "Button";
