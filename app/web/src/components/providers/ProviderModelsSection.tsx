import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, X, Sparkles } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import type { ProviderManifest } from '@/lib/types'
import { cn } from '@/lib/utils'

// ProviderModelsSection edits the operator-managed model list + default
// for a provider. The list lives in provider config `models` (string[])
// and the default in `model` (string); the backend passes the default
// to every spawn via the manifest's modelFlag. "Suggested" pulls the
// manifest's knownModels (the CLIs expose no live model list).
export function ProviderModelsSection({
  manifest,
  value,
  onChange,
}: {
  manifest: ProviderManifest
  value: Record<string, unknown>
  onChange: (v: Record<string, unknown>) => void
}) {
  const { t } = useTranslation()
  const [draftModel, setDraftModel] = useState('')

  const models = Array.isArray(value.models) ? (value.models as string[]) : []
  const defaultModel = typeof value.model === 'string' ? value.model : ''
  const known = manifest.knownModels ?? []
  const suggestable = known.filter((k) => !models.includes(k))

  function commit(next: { models?: string[]; model?: string }) {
    onChange({
      ...value,
      models: next.models ?? models,
      model: next.model ?? defaultModel,
    })
  }

  function add(id: string) {
    const m = id.trim()
    if (!m || models.includes(m)) return
    commit({ models: [...models, m] })
    setDraftModel('')
  }

  function remove(id: string) {
    commit({
      models: models.filter((x) => x !== id),
      // Clear the default if we just removed it.
      model: defaultModel === id ? '' : defaultModel,
    })
  }

  return (
    <div>
      <h2 className="text-[12px] font-semibold uppercase tracking-wider text-muted-foreground/80 mb-1">
        {t('web.providers.models.title')}
      </h2>
      <p className="text-[12px] text-muted-foreground mb-3 max-w-[60ch]">
        {t('web.providers.models.help')}
      </p>

      {models.length === 0 ? (
        <p className="text-[12px] text-muted-foreground italic mb-3">
          {t('web.providers.models.empty')}
        </p>
      ) : (
        <ul className="space-y-1.5 mb-3">
          {models.map((id) => {
            const isDefault = id === defaultModel
            return (
              <li key={id} className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => commit({ model: isDefault ? '' : id })}
                  className={cn(
                    'text-[11px] px-2 py-0.5 rounded border font-mono',
                    isDefault
                      ? 'border-primary text-primary'
                      : 'border-border text-muted-foreground hover:text-foreground',
                  )}
                  title={t('web.providers.models.setDefault')}
                >
                  {isDefault
                    ? t('web.providers.models.default')
                    : t('web.providers.models.makeDefault')}
                </button>
                <span className="font-mono text-[13px] flex-1">{id}</span>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-6 w-6 p-0"
                  onClick={() => remove(id)}
                  aria-label={t('web.providers.models.remove', { model: id })}
                >
                  <X className="h-3.5 w-3.5" />
                </Button>
              </li>
            )
          })}
        </ul>
      )}

      <div className="flex items-center gap-2">
        <Input
          value={draftModel}
          onChange={(e) => setDraftModel(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              add(draftModel)
            }
          }}
          placeholder={t('web.providers.models.addPlaceholder')}
          className="h-8 text-[13px] font-mono"
        />
        <Button
          size="sm"
          variant="outline"
          className="h-8 shrink-0"
          onClick={() => add(draftModel)}
          disabled={!draftModel.trim()}
        >
          <Plus className="h-3.5 w-3.5" />
          {t('web.providers.models.add')}
        </Button>
        {suggestable.length > 0 ? (
          <Button
            size="sm"
            variant="ghost"
            className="h-8 shrink-0"
            onClick={() => commit({ models: [...models, ...suggestable] })}
            title={suggestable.join(', ')}
          >
            <Sparkles className="h-3.5 w-3.5" />
            {t('web.providers.models.suggested', { count: suggestable.length })}
          </Button>
        ) : null}
      </div>
    </div>
  )
}
