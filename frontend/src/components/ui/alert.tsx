import * as React from "react";
import { cn } from "@/lib/utils";
import { AlertCircle, CheckCircle2, Info, AlertTriangle } from "lucide-react";

export interface AlertProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: "info" | "success" | "warning" | "destructive";
  title?: string;
}

export function Alert({ className, variant = "info", title, children, ...props }: AlertProps) {
  const icons = {
    info: <Info className="h-4 w-4 text-[var(--color-status-info-text)]" />,
    success: <CheckCircle2 className="h-4 w-4 text-[var(--color-status-success-text)]" />,
    warning: <AlertTriangle className="h-4 w-4 text-[var(--color-status-warning-text)]" />,
    destructive: <AlertCircle className="h-4 w-4 text-[var(--color-status-danger-text)]" />,
  };

  const variantStyles = {
    info: "border-[var(--color-status-info-border)] bg-[var(--color-status-info-bg)] text-[var(--color-status-info-text)]",
    success: "border-[var(--color-status-success-border)] bg-[var(--color-status-success-bg)] text-[var(--color-status-success-text)]",
    warning: "border-[var(--color-status-warning-border)] bg-[var(--color-status-warning-bg)] text-[var(--color-status-warning-text)]",
    destructive: "border-[var(--color-status-danger-border)] bg-[var(--color-status-danger-bg)] text-[var(--color-status-danger-text)]",
  };

  return (
    <div
      role="alert"
      className={cn(
        "flex gap-3 rounded-[var(--radius-md)] border p-4 text-sm [&>svg]:shrink-0 [&>svg]:translate-y-0.5",
        variantStyles[variant],
        className
      )}
      {...props}
    >
      {icons[variant]}
      <div className="flex-1">
        {title && <h5 className="mb-1 font-semibold leading-none tracking-tight">{title}</h5>}
        <div className="text-sm opacity-95">{children}</div>
      </div>
    </div>
  );
}
