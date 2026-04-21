# OpenDray — nuclear uninstaller for Windows.
#
# Usage:
#   irm https://raw.githubusercontent.com/Opendray/opendray/main/uninstall.ps1 | iex
#
# Prefer `opendray uninstall` when the binary can still run. This
# PowerShell script is the fallback for corrupt installs, broken
# configs, or CI-driven cleanup.
#
# Environment variables (optional):
#   $env:OPENDRAY_YES          = 1   skip confirmation prompt
#   $env:OPENDRAY_DRY_RUN      = 1   print plan, remove nothing
#   $env:OPENDRAY_INSTALL_DIR        override binary dir
#                                    (default: $env:LOCALAPPDATA\Programs\OpenDray)
#
# What this script touches:
#   - %LOCALAPPDATA%\Programs\OpenDray\opendray.exe     (binary)
#   - %LOCALAPPDATA%\OpenDray\                          (config + data)
#   - %USERPROFILE%\.opendray\                          (legacy data path)
#   - HKCU\Environment PATH entry for the install dir
#   - any opendray.exe process listening on :8640 / :5433
#
# What this script does NOT touch:
#   - any external PostgreSQL databases OpenDray was pointed at.

$ErrorActionPreference = 'Stop'

function Say  { param([string]$m) Write-Host $m }
function Info { param([string]$m) Write-Host "    $m" }
function Step { param([string]$m) Write-Host "->  $m" }
function Ok   { param([string]$m) Write-Host "OK  $m" }
function Warn { param([string]$m) Write-Host "!!  $m" -ForegroundColor Yellow }
function Die  { param([string]$m) Write-Host "ERR $m" -ForegroundColor Red; exit 1 }

Say ""
Say "OpenDray uninstaller (nuclear)"
Say "---------------------------------------------------------------"
Say ""

# ── settings ──────────────────────────────────────────────────
$InstallDir = if ($env:OPENDRAY_INSTALL_DIR) {
    $env:OPENDRAY_INSTALL_DIR
} else {
    Join-Path $env:LOCALAPPDATA "Programs\OpenDray"
}
$BinPath = Join-Path $InstallDir "opendray.exe"
$DataDir = Join-Path $env:LOCALAPPDATA "OpenDray"
$LegacyDataDir = Join-Path $env:USERPROFILE ".opendray"

# ── enumerate targets ─────────────────────────────────────────
$targets = @()
if (Test-Path $BinPath)         { $targets += @{ Path = $BinPath;        Kind = "binary" } }
if (Test-Path $DataDir)         { $targets += @{ Path = $DataDir;        Kind = "data dir" } }
if (Test-Path $LegacyDataDir)   { $targets += @{ Path = $LegacyDataDir;  Kind = "legacy data dir" } }

if ($targets.Count -eq 0) {
    Say "Nothing to remove - no OpenDray files found in the default locations:"
    Info "  $BinPath"
    Info "  $DataDir"
    Info "  $LegacyDataDir"
    Say ""
    Say "If you installed to a custom location, set `$env:OPENDRAY_INSTALL_DIR."
    exit 0
}

# ── show plan ────────────────────────────────────────────────
Say "Will remove:"
Say ""
foreach ($t in $targets) {
    Info "  $($t.Path)"
    Info "    ($($t.Kind))"
}
Info "  PATH entry: $InstallDir  (in HKCU\Environment)"
Say ""
Say "Will NOT touch:"
Info "  any external PostgreSQL database you may have pointed OpenDray at."
Info "  (its tables live in your own DB server - drop them yourself if needed.)"
Say ""

# ── dry-run short-circuit ────────────────────────────────────
if ($env:OPENDRAY_DRY_RUN) {
    Warn "`$env:OPENDRAY_DRY_RUN set - nothing will be removed."
    exit 0
}

# ── confirm ──────────────────────────────────────────────────
if (-not $env:OPENDRAY_YES) {
    $reply = Read-Host "Proceed with removal? [y/N]"
    if ($reply -notmatch '^(y|yes)$') {
        Say "Aborted. Nothing changed."
        exit 0
    }
}

# ── stop running opendray processes ──────────────────────────
$procs = Get-Process -Name opendray -ErrorAction SilentlyContinue
foreach ($p in $procs) {
    Step "stopping PID $($p.Id) (opendray.exe)"
    try {
        $p.CloseMainWindow() | Out-Null
        if (-not $p.WaitForExit(2000)) { $p.Kill() }
    } catch {
        Warn "could not stop PID $($p.Id): $_"
    }
}

# ── stop whoever's listening on :8640 / :5433 ────────────────
foreach ($port in 8640, 5433) {
    try {
        $conns = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue
    } catch {
        $conns = $null
    }
    foreach ($c in $conns) {
        if ($c.OwningProcess -eq $PID) { continue }
        Step "stopping PID $($c.OwningProcess) (port $port)"
        try {
            Stop-Process -Id $c.OwningProcess -Force -ErrorAction Stop
            Ok  "stopped PID $($c.OwningProcess)"
        } catch {
            Warn "could not stop PID $($c.OwningProcess): $_"
        }
    }
}

# ── remove files ────────────────────────────────────────────
foreach ($t in $targets) {
    Step "removing $($t.Path)"
    try {
        Remove-Item -Path $t.Path -Recurse -Force -ErrorAction Stop
        Ok  "removed $($t.Path)"
    } catch {
        Warn "could not remove $($t.Path): $_"
    }
}

# ── remove install dir if empty ──────────────────────────────
if (Test-Path $InstallDir) {
    $leftover = Get-ChildItem -Path $InstallDir -ErrorAction SilentlyContinue
    if (-not $leftover) {
        try {
            Remove-Item -Path $InstallDir -Force -ErrorAction Stop
            Ok "removed empty install dir $InstallDir"
        } catch { }
    }
}

# ── PATH cleanup (HKCU only) ─────────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath) {
    $parts = $userPath -split ';' | Where-Object { $_ -ne '' }
    $normalizedTarget = $InstallDir.TrimEnd('\')
    $filtered = $parts | Where-Object {
        $_.TrimEnd('\') -ne $normalizedTarget
    }
    if ($filtered.Count -lt $parts.Count) {
        $newPath = ($filtered -join ';')
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Ok "removed $InstallDir from user PATH"
        Info "(Open shells need a restart to see the updated PATH.)"
    }
}

Say ""
Ok "OpenDray removed."
Say ""
