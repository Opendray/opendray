import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const badgeVariants = cva(
  'inline-flex items-center gap-1 px-1.5 h-4 rounded-full border text-[10px] font-medium uppercase tracking-wide whitespace-nowrap',
  {
    variants: {
      variant: {
        default:
          'bg-card text-foreground border-border',
        accent:
          'bg-accent/15 text-accent border-accent/30',
        success:
          'bg-state-running/20 text-state-running border-state-running/30',
        warning:
          'bg-state-idle/20 text-state-idle border-state-idle/30',
        danger:
          'bg-state-failed/20 text-state-failed border-state-failed/30',
        muted:
          'bg-muted text-muted-foreground border-border',
        outline:
          'bg-transparent text-foreground border-border',
      },
    },
    defaultVariants: { variant: 'default' },
  },
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <span className={cn(badgeVariants({ variant, className }))} {...props} />
  )
}

export { badgeVariants }
