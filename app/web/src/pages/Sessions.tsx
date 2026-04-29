import { Layers } from 'lucide-react'

export function SessionsPage() {
  return (
    <div className="h-full flex flex-col items-center justify-center gap-3 text-center p-6">
      <Layers className="size-10 text-muted-foreground/40" strokeWidth={1.5} />
      <div className="space-y-1">
        <h2 className="text-[14px] font-semibold">No sessions yet</h2>
        <p className="text-[12px] text-muted-foreground max-w-[320px]">
          Spawn a Claude / Codex / Gemini / shell session from the catalog.
          Sessions list and the workbench arrive in W2.
        </p>
      </div>
    </div>
  )
}
