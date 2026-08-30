import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "group/button inline-flex shrink-0 items-center justify-center rounded-lg border border-transparent bg-clip-padding font-mono text-sm font-semibold whitespace-nowrap uppercase tracking-wide transition-all outline-none select-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 active:not-aria-[haspopup]:translate-y-px disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        /** The signature cyan->blue gradient CTA (Figma 61:110/61:60). Frozen 4px radius + glow — see theme.css `btn-gradient-primary`. */
        default: "btn-gradient-primary hover:brightness-110 rounded-tv-chip",
        /** The cyan-bordered secondary CTA (Figma 15:59 "Start Quiz"). */
        "outline-cyan":
          "border-2 border-tv-cyan text-tv-cyan bg-transparent rounded-tv-btn hover:bg-tv-cyan/10 hover:glow-text-cyan",
        outline:
          "border-tv-border bg-transparent text-tv-text-nav rounded-tv-btn hover:bg-tv-surface-2 hover:text-tv-text-hi aria-expanded:bg-tv-surface-2",
        secondary:
          "bg-tv-surface-2 text-tv-text-hi rounded-tv-btn hover:bg-tv-surface-2/70 aria-expanded:bg-tv-surface-2",
        ghost:
          "text-tv-text-nav rounded-tv-btn hover:bg-tv-surface-2 hover:text-tv-text-hi aria-expanded:bg-tv-surface-2",
        destructive:
          "bg-tv-rose/10 text-tv-rose rounded-tv-btn hover:bg-tv-rose/20 focus-visible:border-tv-rose/40 focus-visible:ring-tv-rose/20",
        link: "text-tv-cyan underline-offset-4 hover:underline normal-case tracking-normal",
      },
      size: {
        default:
          "h-8 gap-1.5 px-2.5 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        xs: "h-6 gap-1 rounded-[min(var(--radius-md),10px)] px-2 text-xs in-data-[slot=button-group]:rounded-lg has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3",
        sm: "h-7 gap-1 rounded-[min(var(--radius-md),12px)] px-2.5 text-[0.8rem] in-data-[slot=button-group]:rounded-lg has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3.5",
        lg: "h-9 gap-1.5 px-2.5 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        icon: "size-8",
        "icon-xs":
          "size-6 rounded-[min(var(--radius-md),10px)] in-data-[slot=button-group]:rounded-lg [&_svg:not([class*='size-'])]:size-3",
        "icon-sm":
          "size-7 rounded-[min(var(--radius-md),12px)] in-data-[slot=button-group]:rounded-lg",
        "icon-lg": "size-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
