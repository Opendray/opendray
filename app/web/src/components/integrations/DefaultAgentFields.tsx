import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ChevronDown } from 'lucide-react'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { listProviders } from '@/lib/catalog'
import { listClaudeAccounts } from '@/lib/claudeAccounts'

// NONE is the sentinel for "no default" — Radix Select can't hold an
// empty-string value, so the caller maps it to '' on change.
const NONE = '__none__'

export interface DefaultAgentValue {
  providerId: string
  model: string
  claudeAccountId: string
}

interface DefaultAgentFieldsProps {
  value: DefaultAgentValue
  onChange: (next: DefaultAgentValue) => void
}

// DefaultAgentFields renders the provider / model / claude-account
// selectors for an integration's spawn defaults. Used by both the
// register and edit dialogs. Fully controlled and immutable: every
// change emits a fresh value object, never mutating the prop.
//
// The model field is a free-text input with an explicit dropdown of the
// selected provider's known models (chevron button) — picking one fills
// the field, while typing a custom value still wins (knownModels is only
// a suggestion source, defaults shouldn't be locked to a curated list).
// The claude-account selector applies only when the default provider is
// "claude"; it stays visible (as an advisory) but the hint makes the
// scope clear.
export function DefaultAgentFields({ value, onChange }: DefaultAgentFieldsProps) {
  const { t } = useTranslation()
  const providers = useQuery({ queryKey: ['providers'], queryFn: listProviders })
  const accounts = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
  })

  const selectedProvider = providers.data?.find(
    (p) => p.manifest.id === value.providerId,
  )
  const knownModels = selectedProvider?.manifest.knownModels ?? []
  const isClaude = value.providerId === 'claude'

  return (
    <div className="space-y-4 rounded-md border border-border/60 bg-muted/10 p-3">
      <div className="space-y-1">
        <p className="text-[12px] font-medium">
          {t('web.integrations.defaultAgent.title')}
        </p>
        <p className="text-[10.5px] text-muted-foreground/70 leading-snug">
          {t('web.integrations.defaultAgent.description')}
        </p>
      </div>

      <div className="space-y-1.5">
        <Label className="text-[11px] text-muted-foreground/80">
          {t('web.integrations.defaultAgent.providerLabel')}
        </Label>
        <Select
          value={value.providerId || NONE}
          onValueChange={(v) =>
            onChange({
              ...value,
              providerId: v === NONE ? '' : v,
              // Dropping the provider clears the claude-account default,
              // which is only meaningful for the claude provider.
              claudeAccountId:
                v === 'claude' ? value.claudeAccountId : '',
            })
          }
        >
          <SelectTrigger className="h-9 text-[12px]">
            <SelectValue
              placeholder={t('web.integrations.defaultAgent.providerNone')}
            />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NONE}>
              {t('web.integrations.defaultAgent.providerNone')}
            </SelectItem>
            {(providers.data ?? []).map((p) => (
              <SelectItem key={p.manifest.id} value={p.manifest.id}>
                {p.manifest.displayName}{' '}
                <span className="text-muted-foreground/60 font-mono">
                  {p.manifest.id}
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-1.5">
        <Label
          htmlFor="default_model"
          className="text-[11px] text-muted-foreground/80"
        >
          {t('web.integrations.defaultAgent.modelLabel')}
        </Label>
        <div className="relative">
          <Input
            id="default_model"
            value={value.model}
            onChange={(e) => onChange({ ...value, model: e.target.value })}
            placeholder={t('web.integrations.defaultAgent.modelPlaceholder')}
            className={cn('font-mono', knownModels.length > 0 && 'pr-9')}
          />
          {knownModels.length > 0 && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="absolute right-0 top-0 h-9 w-9 text-muted-foreground hover:text-foreground"
                  aria-label={t('web.integrations.defaultAgent.modelPickAria')}
                >
                  <ChevronDown className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="end"
                className="max-h-64 overflow-y-auto"
              >
                {knownModels.map((m) => (
                  <DropdownMenuItem
                    key={m}
                    className="font-mono text-xs"
                    onSelect={() => onChange({ ...value, model: m })}
                  >
                    {m}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>

      <div className="space-y-1.5">
        <Label className="text-[11px] text-muted-foreground/80">
          {t('web.integrations.defaultAgent.accountLabel')}
        </Label>
        <Select
          value={value.claudeAccountId || NONE}
          onValueChange={(v) =>
            onChange({ ...value, claudeAccountId: v === NONE ? '' : v })
          }
          disabled={!isClaude}
        >
          <SelectTrigger className="h-9 text-[12px]">
            <SelectValue
              placeholder={t('web.integrations.defaultAgent.accountNone')}
            />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NONE}>
              {t('web.integrations.defaultAgent.accountNone')}
            </SelectItem>
            {(accounts.data ?? []).map((a) => (
              <SelectItem key={a.id} value={a.id}>
                {a.display_name || a.name}
                {!a.token_filled && (
                  <span className="text-muted-foreground/60">
                    {' '}
                    {t('web.integrations.defaultAgent.accountTokenMissing')}
                  </span>
                )}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-[10.5px] text-muted-foreground/60">
          {t('web.integrations.defaultAgent.accountHint')}
        </p>
      </div>
    </div>
  )
}
