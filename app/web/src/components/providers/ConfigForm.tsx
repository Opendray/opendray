import { useEffect, useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import type { ConfigField } from '@/lib/types'

interface ConfigFormProps {
  schema: ConfigField[]
  initial: Record<string, unknown>
  /** Called every time a field changes; ConfigForm is controlled. */
  onChange: (next: Record<string, unknown>) => void
}

function visible(field: ConfigField, values: Record<string, unknown>): boolean {
  if (!field.dependsOn) return true
  return values[field.dependsOn] === field.dependsVal
}

function groupOf(f: ConfigField): string {
  return f.group ?? 'general'
}

export function ConfigForm({ schema, initial, onChange }: ConfigFormProps) {
  const [values, setValues] = useState<Record<string, unknown>>(initial)

  useEffect(() => {
    setValues(initial)
  }, [initial])

  const update = (key: string, v: unknown) => {
    const next = { ...values, [key]: v }
    setValues(next)
    onChange(next)
  }

  const groups = new Map<string, ConfigField[]>()
  for (const f of schema) {
    const g = groupOf(f)
    const arr = groups.get(g) ?? []
    arr.push(f)
    groups.set(g, arr)
  }

  return (
    <div className="flex flex-col gap-6">
      {Array.from(groups.entries()).map(([g, fields]) => (
        <section key={g} className="flex flex-col gap-4">
          <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
            {g}
          </div>
          {fields.filter((f) => visible(f, values)).map((field) => (
            <FieldRow
              key={field.key}
              field={field}
              value={values[field.key]}
              onChange={(v) => update(field.key, v)}
            />
          ))}
        </section>
      ))}
    </div>
  )
}

function FieldRow({
  field,
  value,
  onChange,
}: {
  field: ConfigField
  value: unknown
  onChange: (v: unknown) => void
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-baseline justify-between">
        <Label htmlFor={field.key}>{field.label}</Label>
        {field.envVar && (
          <span className="text-[10px] text-muted-foreground/60 font-mono">
            ${field.envVar}
          </span>
        )}
      </div>
      <RenderInput field={field} value={value} onChange={onChange} />
      {field.description && (
        <p className="text-[11px] text-muted-foreground/80 leading-relaxed">
          {field.description}
        </p>
      )}
    </div>
  )
}

function RenderInput({
  field,
  value,
  onChange,
}: {
  field: ConfigField
  value: unknown
  onChange: (v: unknown) => void
}) {
  const { t } = useTranslation()
  switch (field.type) {
    case 'string':
      return (
        <Input
          id={field.key}
          value={(value as string) ?? ''}
          placeholder={field.placeholder}
          onChange={(e) => onChange(e.target.value)}
        />
      )
    case 'number':
      return (
        <Input
          id={field.key}
          type="number"
          value={(value as number | undefined)?.toString() ?? ''}
          placeholder={field.placeholder}
          onChange={(e) =>
            onChange(e.target.value === '' ? undefined : Number(e.target.value))
          }
        />
      )
    case 'boolean':
      return (
        <div className="flex items-center gap-2">
          <Switch
            id={field.key}
            checked={Boolean(value)}
            onCheckedChange={(checked) => onChange(checked)}
          />
          <span className="text-[12px] text-muted-foreground">
            {value
              ? t('web.providers.configForm.switchOn')
              : t('web.providers.configForm.switchOff')}
          </span>
        </div>
      )
    case 'select': {
      // Radix Select forbids `value=""` on Item (it reserves "" to mean
      // "cleared / show placeholder"). Some manifests (e.g. gemini's
      // `sandbox`) legitimately use the empty string as an option, so
      // map it to a sentinel inside the widget and translate back at
      // the boundary.
      const EMPTY = '__empty__'
      const cur = typeof value === 'string' ? value : ''
      return (
        <Select
          value={cur === '' ? EMPTY : cur}
          onValueChange={(v) => onChange(v === EMPTY ? '' : v)}
        >
          <SelectTrigger id={field.key}>
            <SelectValue
              placeholder={
                field.placeholder ?? t('web.providers.configForm.selectPlaceholder')
              }
            />
          </SelectTrigger>
          <SelectContent>
            {(field.options ?? []).map((opt) => (
              <SelectItem key={opt} value={opt === '' ? EMPTY : opt}>
                {opt || t('web.providers.configForm.defaultOption')}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )
    }
    case 'secret':
      return (
        <SecretInput
          id={field.key}
          value={(value as string) ?? ''}
          placeholder={field.placeholder}
          onChange={(v) => onChange(v)}
        />
      )
    case 'args':
      // textarea, one arg per line.
      return (
        <Textarea
          id={field.key}
          rows={3}
          value={Array.isArray(value) ? value.join('\n') : ''}
          placeholder={field.placeholder}
          onChange={(e) =>
            onChange(
              e.target.value
                .split('\n')
                .map((s) => s.trim())
                .filter((s) => s.length > 0),
            )
          }
          className="font-mono"
        />
      )
    default:
      return (
        <Input
          id={field.key}
          value={(value as string) ?? ''}
          onChange={(e) => onChange(e.target.value)}
        />
      )
  }
}

function SecretInput({
  id,
  value,
  placeholder,
  onChange,
}: {
  id: string
  value: string
  placeholder?: string
  onChange: (v: string) => void
}) {
  const { t } = useTranslation()
  const [reveal, setReveal] = useState(false)
  return (
    <div className="relative">
      <Input
        id={id}
        type={reveal ? 'text' : 'password'}
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        className="pr-8 font-mono"
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        onClick={() => setReveal((v) => !v)}
        className="absolute right-1 top-1 size-7"
        aria-label={
          reveal
            ? t('web.providers.configForm.hideSecret')
            : t('web.providers.configForm.showSecret')
        }
      >
        {reveal ? <EyeOff className="size-3" /> : <Eye className="size-3" />}
      </Button>
    </div>
  )
}
