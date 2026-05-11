import { Toaster as Sonner } from 'sonner'
import { useTheme } from '@/stores/theme'

export function Toaster() {
  const theme = useTheme((s) => s.applied())
  return (
    <Sonner
      theme={theme}
      position="bottom-right"
      richColors={false}
      offset={16}
      toastOptions={{
        classNames: {
          toast:
            'group toast bg-popover text-popover-foreground border border-border shadow-md text-[12px] rounded-md',
          description: 'text-muted-foreground text-[11px]',
          actionButton:
            'bg-primary text-primary-foreground rounded-md px-2 py-1 text-[11px]',
          cancelButton:
            'bg-muted text-muted-foreground rounded-md px-2 py-1 text-[11px]',
        },
      }}
    />
  )
}
