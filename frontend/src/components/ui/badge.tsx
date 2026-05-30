import { cn } from "@/lib/utils"
import { type HTMLAttributes, forwardRef } from "react"

const variants = {
  default: "bg-neutral-100 text-neutral-700",
  red: "bg-red-50 text-red-600",
  amber: "bg-amber-50 text-amber-600",
  emerald: "bg-emerald-50 text-emerald-600",
  indigo: "bg-indigo-50 text-indigo-600",
  outline: "border border-neutral-200 text-neutral-500",
} as const

type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  variant?: keyof typeof variants
}

const Badge = forwardRef<HTMLSpanElement, BadgeProps>(
  ({ className, variant = "default", ...props }, ref) => (
    <span
      ref={ref}
      className={cn(
        "inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium",
        variants[variant],
        className,
      )}
      {...props}
    />
  ),
)
Badge.displayName = "Badge"

export { Badge }
