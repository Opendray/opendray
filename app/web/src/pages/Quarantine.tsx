// QuarantinePage — /cortex/memory/quarantine: the review queue for
// third-party-captured memories (Cortex Phase 2).

import { QuarantinePanel } from '@/components/cortex/QuarantinePanel'

export function QuarantinePage() {
  return (
    <div className="mx-auto max-w-3xl">
      <QuarantinePanel />
    </div>
  )
}
