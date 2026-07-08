import { Dialog, DialogContent } from '@/components/ui/dialog'
import { DatabaseWorkbench } from './DatabaseWorkbench'
import type { TableRef } from './SchemaTree'

interface DatabaseWorkbenchDialogProps {
  cwd: string
  open: boolean
  onOpenChange: (v: boolean) => void
  connectionId?: string | null
  table?: TableRef | null
}

// DatabaseWorkbenchDialog hosts the full-size DatabaseWorkbench in a
// near-fullscreen dialog (92vw × 90vh) so the data grid and SQL console
// get real estate the narrow inspector sidebar can't offer. The dialog
// overrides the base DialogContent size/padding via className.
export function DatabaseWorkbenchDialog({
  cwd,
  open,
  onOpenChange,
  connectionId,
  table,
}: DatabaseWorkbenchDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[90vh] max-h-[90vh] w-[92vw] max-w-[92vw] flex-col overflow-hidden p-0">
        {/* key forces a fresh workbench each open so it lands on the
            connection/table the operator clicked, not a stale one. */}
        <DatabaseWorkbench
          key={`${connectionId ?? ''}:${table?.schema ?? ''}.${table?.table ?? ''}`}
          cwd={cwd}
          initialConnectionId={connectionId}
          initialTable={table}
        />
      </DialogContent>
    </Dialog>
  )
}
