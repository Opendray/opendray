# scripts/install-windows.ps1
# Windows entry point for the opendray installer.
#
# opendray does NOT support native Windows. The session subsystem
# spawns AI CLIs through PTYs (creack/pty), and PTYs only exist on
# Unix kernels. ConPTY exists on Windows but is not wired into
# opendray and there are no plans to support it in V1.
#
# Recommended path: install WSL2 + Ubuntu, then run install-linux.sh
# inside the WSL distribution. This script:
#   1) Detects whether WSL is already installed.
#   2) Offers to install WSL2 + Ubuntu (one-time, needs admin + reboot).
#   3) Once WSL is up, prints the exact commands to clone the repo
#      and run the Linux installer inside Ubuntu.
#
# Usage (from an elevated PowerShell):
#   pwsh -ExecutionPolicy Bypass -File scripts/install-windows.ps1

$ErrorActionPreference = "Stop"

function Write-Banner {
    Write-Host ""
    Write-Host "━━━ opendray installer — Windows entry ━━━" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Info  { param([string]$m) Write-Host "[*] $m" -ForegroundColor Blue }
function Write-Ok    { param([string]$m) Write-Host "[✓] $m" -ForegroundColor Green }
function Write-Warn  { param([string]$m) Write-Host "[!] $m" -ForegroundColor Yellow }
function Write-Err   { param([string]$m) Write-Host "[✗] $m" -ForegroundColor Red }

function Test-IsAdmin {
    $id  = [System.Security.Principal.WindowsIdentity]::GetCurrent()
    $win = New-Object System.Security.Principal.WindowsPrincipal($id)
    return $win.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)
}

Write-Banner

Write-Host @"
opendray cannot run as a native Windows process. The AI-CLI session
subsystem uses Unix PTYs, which Windows does not expose to opendray.
This wizard helps you set up WSL2 + Ubuntu so you can run the Linux
installer inside it.

"@

# ── Step 1: detect existing WSL ─────────────────────────────────────
$wslAvailable = $false
try {
    $null = & wsl.exe --status 2>&1
    if ($LASTEXITCODE -eq 0) { $wslAvailable = $true }
} catch {
    $wslAvailable = $false
}

if ($wslAvailable) {
    Write-Ok "WSL is already enabled."
    Write-Host ""
    Write-Host "Available distributions:"
    & wsl.exe --list --verbose 2>&1 | ForEach-Object { "    $_" }
    Write-Host ""
} else {
    Write-Warn "WSL is not enabled on this machine."
    Write-Host ""
    Write-Host "To install WSL2 + Ubuntu (Windows 10 2004+ / Windows 11):"
    Write-Host ""
    Write-Host "  1) Open PowerShell as Administrator." -ForegroundColor Yellow
    Write-Host "  2) Run:" -ForegroundColor Yellow
    Write-Host "       wsl --install -d Ubuntu" -ForegroundColor Green
    Write-Host "  3) Reboot when prompted." -ForegroundColor Yellow
    Write-Host "  4) On first launch, Ubuntu asks for a Linux username/password." -ForegroundColor Yellow
    Write-Host "  5) After Ubuntu's prompt, rerun this script in normal PowerShell" -ForegroundColor Yellow
    Write-Host "     to get the inside-WSL command to install opendray." -ForegroundColor Yellow
    Write-Host ""

    if (-not (Test-IsAdmin)) {
        Write-Warn "This PowerShell is NOT elevated. 'wsl --install' needs admin."
        Write-Host "Right-click PowerShell → Run as administrator, then rerun."
        exit 1
    }

    $answer = Read-Host "Run 'wsl --install -d Ubuntu' now? [y/N]"
    if ($answer -match '^[yY]') {
        Write-Info "Running 'wsl --install -d Ubuntu' ..."
        & wsl.exe --install -d Ubuntu
        Write-Ok "Install initiated. Reboot when prompted, then rerun this script."
    }
    exit 0
}

# ── Step 2: WSL is up — print the cross-over command ────────────────
Write-Host ""
Write-Host "━━━ Next step: inside Ubuntu ━━━" -ForegroundColor Cyan
Write-Host ""
Write-Host @"
Open your Ubuntu (or other WSL distribution) shell — either via
Windows Terminal, the 'Ubuntu' Start menu entry, or:

    wsl -d Ubuntu

Then run:

"@

Write-Host "    sudo apt update && sudo apt install -y git" -ForegroundColor Green
Write-Host "    git clone https://github.com/Opendray/opendray_v2.git" -ForegroundColor Green
Write-Host "    cd opendray_v2" -ForegroundColor Green
Write-Host "    bash scripts/install-linux.sh" -ForegroundColor Green
Write-Host ""

Write-Host @"
The Linux installer will walk you through Postgres, AI CLI install,
opendray credentials, and systemd registration — inside the WSL2
Ubuntu environment. Once it finishes, the admin UI is reachable from
the Windows host at http://localhost:8770/admin/ (WSL2 forwards
loopback ports to the host automatically).

If the gateway needs to be reachable from other LAN devices, pick
0.0.0.0:8770 when the wizard asks for the listen address — and
remember WSL2 NAT means LAN hosts hit the Windows machine's IP, not
the WSL distribution's IP.

"@

Write-Host "Heads up:" -ForegroundColor Yellow
Write-Host "  - WSL2 Ubuntu does not auto-start systemd in default installs"
Write-Host "    older than Win11 22H2. If 'systemctl' refuses to run, add"
Write-Host "    'systemd=true' under [boot] in /etc/wsl.conf and 'wsl --shutdown'"
Write-Host "    from PowerShell once before retrying."
Write-Host ""
