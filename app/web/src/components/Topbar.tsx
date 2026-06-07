import { useNavigate } from '@tanstack/react-router'
import {
  Sun,
  Moon,
  Monitor,
  Terminal as TerminalIcon,
  LogOut,
  Search,
  Check,
  PanelLeftClose,
  PanelLeftOpen,
  Languages,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuShortcut,
} from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useTheme, type ThemeMode } from '@/stores/theme'
import { useAuth } from '@/stores/auth'
import { useLayout } from '@/stores/layout'
import { useLocale, type Locale } from '@/stores/locale'

interface TopbarProps {
  onOpenPalette?: () => void
}

const themeOptions: { mode: ThemeMode; labelKey: string; icon: typeof Sun }[] = [
  { mode: 'light', labelKey: 'web.topbar.themeLight', icon: Sun },
  { mode: 'dark', labelKey: 'web.topbar.themeDark', icon: Moon },
  { mode: 'system', labelKey: 'web.topbar.themeSystem', icon: Monitor },
]

const localeOptions: { locale: Locale; labelKey: string }[] = [
  { locale: 'en', labelKey: 'web.topbar.languageEnglish' },
  { locale: 'zh', labelKey: 'web.topbar.languageChinese' },
  { locale: 'es', labelKey: 'web.topbar.languageSpanish' },
]

export function Topbar({ onOpenPalette }: TopbarProps) {
  const { t } = useTranslation()
  const mode = useTheme((s) => s.mode)
  const setMode = useTheme((s) => s.setMode)
  const locale = useLocale((s) => s.locale)
  const setLocale = useLocale((s) => s.setLocale)
  const username = useAuth((s) => s.username)
  const expiresAt = useAuth((s) => s.expiresAt)
  const clear = useAuth((s) => s.clear)
  const navigate = useNavigate()
  const sidebarCollapsed = useLayout((s) => s.sidebarCollapsed)
  const toggleSidebar = useLayout((s) => s.toggleSidebar)

  const ThemeIcon =
    mode === 'dark' ? Moon : mode === 'light' ? Sun : Monitor

  const sidebarToggleLabel = sidebarCollapsed
    ? t('web.topbar.expandSidebar')
    : t('web.topbar.collapseSidebar')

  return (
    <div className="h-11 border-b border-border bg-background flex items-center px-3 gap-1.5 shrink-0">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleSidebar}
            aria-label={sidebarToggleLabel}
            className="size-7"
          >
            {sidebarCollapsed ? (
              <PanelLeftOpen className="size-3.5" />
            ) : (
              <PanelLeftClose className="size-3.5" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent>{sidebarToggleLabel}</TooltipContent>
      </Tooltip>
      <div className="flex items-center gap-1.5 pl-1">
        <TerminalIcon
          className="size-3.5 text-accent"
          strokeWidth={2.5}
        />
        <span className="text-[12px] font-semibold tracking-tight">
          {t('web.brand')}
        </span>
      </div>
      <div className="flex-1" />

      {onOpenPalette && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              onClick={onOpenPalette}
              className="h-7 gap-2 text-muted-foreground bg-card/50 hover:bg-card font-normal text-[12px]"
            >
              <Search className="size-3" />
              <span>{t('web.topbar.search')}</span>
              <kbd className="ml-1">⌘K</kbd>
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('web.topbar.openPalette')}</TooltipContent>
        </Tooltip>
      )}

      <DropdownMenu>
        <Tooltip>
          <TooltipTrigger asChild>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label={t('web.topbar.language')}
              >
                <Languages className="size-3.5" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>{t('web.topbar.language')}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>{t('web.topbar.language')}</DropdownMenuLabel>
          {localeOptions.map(({ locale: l, labelKey }) => (
            <DropdownMenuItem key={l} onSelect={() => setLocale(l)}>
              <span>{t(labelKey)}</span>
              {locale === l && <Check className="ml-auto size-3" />}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <DropdownMenu>
        <Tooltip>
          <TooltipTrigger asChild>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label={t('web.topbar.themeLabel', { mode })}
              >
                <ThemeIcon className="size-3.5" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>{t('web.topbar.theme')}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>{t('web.topbar.appearance')}</DropdownMenuLabel>
          {themeOptions.map(({ mode: m, labelKey, icon: Icon }) => (
            <DropdownMenuItem key={m} onSelect={() => setMode(m)}>
              <Icon />
              <span>{t(labelKey)}</span>
              {mode === m && <Check className="ml-auto size-3" />}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {username && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-[12px] font-normal text-muted-foreground gap-1.5"
            >
              <span className="size-1.5 rounded-full bg-state-running" />
              {username}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            side="bottom"
            sideOffset={6}
            // The trigger sits in the top-right corner. Opening down-and-left
            // from the right edge is always on-screen, so pin it: collision
            // avoidance was mis-flipping this menu to open rightward, pushing
            // it off the right edge of the window.
            avoidCollisions={false}
            className="min-w-[220px]"
          >
            <DropdownMenuLabel>{t('web.topbar.signedInAs')}</DropdownMenuLabel>
            <div className="px-2 pb-1.5 text-[12px]">{username}</div>
            {expiresAt && (
              <>
                <DropdownMenuLabel>{t('web.topbar.tokenExpires')}</DropdownMenuLabel>
                <div className="px-2 pb-1.5 text-[11px] text-muted-foreground font-mono">
                  {new Date(expiresAt).toLocaleString()}
                </div>
              </>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onSelect={() => {
                clear()
                navigate({ to: '/login', search: { next: undefined } })
              }}
            >
              <LogOut /> {t('web.topbar.signOut')}
              <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  )
}
