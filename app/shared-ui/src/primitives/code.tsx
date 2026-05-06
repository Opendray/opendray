import { cn } from '@/lib/utils'

interface CodeProps extends React.HTMLAttributes<HTMLPreElement> {
  language?: string
}

export function Code({ className, children, ...props }: CodeProps) {
  return (
    <pre
      className={cn(
        'rounded-md border border-border bg-input/30 p-3 text-[11px] font-mono leading-relaxed overflow-auto whitespace-pre-wrap break-all',
        className,
      )}
      {...props}
    >
      {children}
    </pre>
  )
}
