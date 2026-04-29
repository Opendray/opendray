import type { LucideIcon } from 'lucide-react'

interface PlaceholderProps {
  icon: LucideIcon
  title: string
  body: string
}

export function Placeholder({ icon: Icon, title, body }: PlaceholderProps) {
  return (
    <div className="h-full flex flex-col items-center justify-center gap-3 text-center p-6">
      <Icon className="size-10 text-muted-foreground/40" strokeWidth={1.5} />
      <div className="space-y-1">
        <h2 className="text-[14px] font-semibold">{title}</h2>
        <p className="text-[12px] text-muted-foreground max-w-[320px]">
          {body}
        </p>
      </div>
    </div>
  )
}
