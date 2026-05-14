import { useState } from 'react'
import { Check, FlaskConical, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { testServerPath, type TestPathResponse } from '@/lib/settings'
import { cn } from '@/lib/utils'

interface PathInputProps {
  value: string
  onChange: (next: string) => void
  placeholder?: string
  expectDir?: boolean
}

// PathInput is a text input + "Test" button for filesystem paths
// in the Server Settings form. Test calls /admin/settings/test-path
// and shows the resolved path + child count inline so the operator
// can verify a path before saving.
export function PathInput({
  value,
  onChange,
  placeholder,
  expectDir,
}: PathInputProps) {
  const { t } = useTranslation()
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<TestPathResponse | null>(null)

  const test = async () => {
    if (!value.trim()) return
    setBusy(true)
    try {
      const res = await testServerPath(value.trim())
      setResult(res)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-col gap-1">
      <div className="flex gap-1.5">
        <Input
          value={value}
          onChange={(e) => {
            onChange(e.target.value)
            setResult(null)
          }}
          placeholder={placeholder}
          className="h-8 text-xs font-mono flex-1"
        />
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-8 px-2 text-xs"
          disabled={busy || !value.trim()}
          onClick={test}
          title={t('web.pathInput.testTooltip')}
        >
          <FlaskConical className="size-3 mr-1" />
          {t('web.pathInput.testButton')}
        </Button>
      </div>
      {result && <PathResult res={result} expectDir={expectDir} />}
    </div>
  )
}

function PathResult({
  res,
  expectDir,
}: {
  res: TestPathResponse
  expectDir?: boolean
}) {
  const { t } = useTranslation()
  if (!res.exists) {
    return (
      <p className="text-[10px] text-destructive flex items-center gap-1 px-1">
        <X className="size-3" />
        {t('web.pathInput.notFound')} {res.path}
      </p>
    )
  }
  const wrongType = expectDir && !res.is_dir
  return (
    <p
      className={cn(
        'text-[10px] flex items-center gap-1 px-1',
        wrongType ? 'text-destructive' : 'text-muted-foreground',
      )}
    >
      {wrongType ? <X className="size-3" /> : <Check className="size-3" />}
      {res.path}
      {res.is_dir && (
        <span className="opacity-60">
          · {res.child_count ?? 0} {t('web.pathInput.childrenSuffix')}
        </span>
      )}
      {wrongType && <span className="opacity-60">{t('web.pathInput.expectedDirectory')}</span>}
    </p>
  )
}
