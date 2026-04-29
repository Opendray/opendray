import * as React from 'react'
import { cn } from '@/lib/utils'

export const Input = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, type, ...props }, ref) => {
  return (
    <input
      type={type}
      ref={ref}
      className={cn(
        'flex h-9 w-full rounded-md border border-border bg-input/40 px-3 py-1 text-[13px] text-foreground transition-colors',
        'placeholder:text-muted-foreground/70',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-foreground/20',
        'disabled:cursor-not-allowed disabled:opacity-50',
        'file:border-0 file:bg-transparent file:text-sm file:font-medium',
        className,
      )}
      {...props}
    />
  )
})
Input.displayName = 'Input'
