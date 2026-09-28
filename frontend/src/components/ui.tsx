import { forwardRef } from "react";
import { cn } from "@/lib/format";

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger" | "accent";
  size?: "sm" | "md" | "lg" | "icon";
};

const buttonVariants = {
  primary: "bg-primary text-primary-fg hover:brightness-105 shadow-[0_8px_30px_-12px_var(--primary)]",
  accent: "bg-accent text-[#2a0a0e] hover:brightness-105 shadow-[0_8px_30px_-12px_var(--accent)]",
  secondary: "bg-surface-2 text-fg border border-border hover:bg-primary-soft",
  ghost: "text-muted hover:text-fg hover:bg-surface-2",
  danger: "bg-danger/15 text-danger hover:bg-danger/25",
};

const buttonSizes = {
  sm: "h-8 px-3 text-sm gap-1.5 rounded-lg",
  md: "h-10 px-4 text-sm gap-2 rounded-xl",
  lg: "h-12 px-6 text-base gap-2 rounded-2xl",
  icon: "h-10 w-10 rounded-xl",
};

export function buttonClass(variant: ButtonProps["variant"] = "primary", size: ButtonProps["size"] = "md", className?: string) {
  return cn(
    "inline-flex items-center justify-center font-medium transition-all outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 disabled:pointer-events-none active:scale-[0.98] cursor-pointer select-none",
    buttonVariants[variant],
    buttonSizes[size],
    className,
  );
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant, size, className, type = "button", ...props },
  ref,
) {
  return <button ref={ref} type={type} className={buttonClass(variant, size, className)} {...props} />;
});

export function Card({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("rounded-3xl border border-border bg-surface", className)} {...props} />;
}

export function Badge({
  className,
  tone = "default",
  ...props
}: React.HTMLAttributes<HTMLSpanElement> & { tone?: "default" | "primary" | "accent" | "lime" }) {
  const tones = {
    default: "bg-surface-2 text-muted border-border",
    primary: "bg-primary-soft text-primary border-primary/20",
    accent: "bg-accent-soft text-accent border-accent/20",
    lime: "bg-lime/15 text-lime border-lime/20",
  };
  return (
    <span
      className={cn("inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium", tones[tone], className)}
      {...props}
    />
  );
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn("animate-pulse rounded-xl bg-surface-2", className)} />;
}

export const Input = forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(function Input(
  { className, ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      className={cn(
        "h-11 w-full rounded-xl border border-border bg-surface-2 px-4 text-sm text-fg placeholder:text-muted outline-none transition focus:border-primary/50 focus:ring-2 focus:ring-ring",
        className,
      )}
      {...props}
    />
  );
});

export function Equalizer({ className, paused }: { className?: string; paused?: boolean }) {
  return (
    <span className={cn("inline-flex h-4 items-end gap-[3px]", className)} aria-hidden>
      {[0, 0.2, 0.4, 0.1].map((delay, i) => (
        <span
          key={i}
          className={cn("eq-bar w-[3px] rounded-full bg-primary", paused && "[animation-play-state:paused]")}
          style={{ height: "100%", animationDelay: `${delay}s` }}
        />
      ))}
    </span>
  );
}

export function EmptyState({
  icon,
  title,
  children,
  className,
}: {
  icon?: React.ReactNode;
  title: string;
  children?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col items-center justify-center gap-3 px-6 py-14 text-center", className)}>
      {icon && <div className="grid h-12 w-12 place-items-center rounded-2xl bg-primary-soft text-primary">{icon}</div>}
      <h3 className="font-display text-lg font-semibold">{title}</h3>
      {children && <div className="max-w-sm text-sm text-muted">{children}</div>}
    </div>
  );
}
