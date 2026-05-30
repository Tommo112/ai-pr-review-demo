import { cn } from "@/lib/utils"
import { type ButtonHTMLAttributes, forwardRef } from "react"

const variants = {
  default:
    "bg-neutral-900 text-white hover:bg-neutral-800 shadow-sm",
  outline:
    "border border-neutral-200 bg-white text-neutral-700 hover:bg-neutral-50 shadow-sm",
  ghost: "text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900",
  primary:
    "bg-indigo-500 text-white hover:bg-indigo-600 shadow-sm",
} as const

const sizes = {
  sm: "h-8 px-3 text-xs rounded-md",
  md: "h-10 px-4 text-sm rounded-lg",
  lg: "h-12 px-6 text-sm rounded-lg",
} as const

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: keyof typeof variants
  size?: keyof typeof sizes
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "default", size = "md", ...props }, ref) => (
    <button
      ref={ref}
      className={cn(
        "inline-flex items-center justify-center gap-2 font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-400 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
        variants[variant],
        sizes[size],
        className,
      )}
      {...props}
    />
  ),
)
Button.displayName = "Button"

export { Button, type ButtonProps }
