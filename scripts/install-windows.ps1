# scripts/install-windows.ps1
# Windows entry point for the opendray installer.
#
# opendray does NOT run as a native Windows process — the session
# subsystem spawns AI CLIs through Unix PTYs, which the Windows kernel
# doesn't expose to opendray. So the Windows install is: set up WSL2 +
# Ubuntu, then run the Linux installer inside it.
#
# Unlike a manual guide, this script does the whole thing for you:
#   1) Ensures the WSL2 feature is present (enables it if missing —
#      that one path needs admin + a reboot, then re-run).
#   2) Ensures a usable Ubuntu/Debian distro exists (installs
#      Ubuntu-24.04 if there isn't one — a Docker/Podman WSL distro
#      does NOT count).
#   3) Ensures systemd is enabled inside the distro (opendray runs as a
#      systemd service).
#   4) Runs the opendray Linux installer inside WSL.
#   5) Registers a logon task so the distro (and the gateway) comes back
#      up after idle/reboot — WSL stops idle distros otherwise.
#
# Usage (from PowerShell; elevate if WSL isn't installed yet):
#   irm https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/install-windows.ps1 | iex

$ErrorActionPreference = "Stop"

$Distro    = "Ubuntu-24.04"
$InstallSh = "https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/install.sh"
$TaskName  = "opendray-wsl-keepalive"

function Write-Banner { Write-Host ""; Write-Host "+-- opendray installer - Windows entry --+" -ForegroundColor Cyan; Write-Host "" }
function Write-Info { param([string]$m) Write-Host "[*] $m" -ForegroundColor Blue }
function Write-Ok   { param([string]$m) Write-Host "[OK] $m" -ForegroundColor Green }
function Write-Warn { param([string]$m) Write-Host "[!] $m" -ForegroundColor Yellow }
function Write-Err  { param([string]$m) Write-Host "[X] $m" -ForegroundColor Red }

function Test-IsAdmin {
    $id = [System.Security.Principal.WindowsIdentity]::GetCurrent()
    return (New-Object System.Security.Principal.WindowsPrincipal($id)).IsInRole(
        [System.Security.Principal.WindowsBuiltInRole]::Administrator)
}

# WSL emits UTF-16; WSL_UTF8 makes its output parseable.
$env:WSL_UTF8 = "1"

Write-Banner

# -- Step 1: ensure the WSL2 feature ---------------------------------
$wslReady = $false
try { $null = & wsl.exe --status 2>&1; if ($LASTEXITCODE -eq 0) { $wslReady = $true } } catch { $wslReady = $false }

if (-not $wslReady) {
    Write-Warn "WSL2 is not enabled on this machine."
    if (-not (Test-IsAdmin)) {
        Write-Err "Enabling WSL2 needs an elevated PowerShell."
        Write-Host "  Right-click PowerShell -> Run as administrator, then re-run this command."
        exit 1
    }
    Write-Info "Enabling WSL2 (this is a one-time step)..."
    & wsl.exe --install --no-distribution
    Write-Ok "WSL2 enabled. REBOOT, then re-run this same command to finish."
    exit 0
}
Write-Ok "WSL2 is available."

# -- Step 2: ensure a usable Ubuntu/Debian distro --------------------
# Bug this replaces: the old script treated ANY WSL distro (e.g. a
# Docker/Podman helper) as 'ready' and pointed users at an Ubuntu that
# wasn't installed. Match only real Debian-family distros.
$installed = @()
try { $installed = (& wsl.exe --list --quiet 2>$null) | ForEach-Object { ($_ -replace '\0','').Trim() } | Where-Object { $_ } } catch {}
$usable = $installed | Where-Object { $_ -match '^(Ubuntu|Debian)' }

if (-not $usable) {
    Write-Warn "No Ubuntu/Debian WSL distro found (installed: $([string]::Join(', ', $installed)))."
    Write-Info "Installing $Distro ..."
    & wsl.exe --install -d $Distro --no-launch
    if ($LASTEXITCODE -ne 0) { Write-Err "Failed to install $Distro."; exit 1 }
    Write-Ok "$Distro installed."
} else {
    $Distro = ($usable | Select-Object -First 1)
    Write-Ok "Using existing distro: $Distro"
}

# -- Step 3: ensure systemd is enabled inside the distro -------------
# opendray installs a systemd service; default WSL has systemd off.
$wantConf = "[boot]`nsystemd=true`n"
$b64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($wantConf))
$needsRestart = $false
$current = (& wsl.exe -d $Distro -u root -- sh -c "cat /etc/wsl.conf 2>/dev/null") -join "`n"
if ($current -notmatch 'systemd\s*=\s*true') {
    Write-Info "Enabling systemd in $Distro ..."
    & wsl.exe -d $Distro -u root -- sh -c "echo $b64 | base64 -d > /etc/wsl.conf"
    $needsRestart = $true
}
if ($needsRestart) {
    & wsl.exe --terminate $Distro 2>&1 | Out-Null
    Start-Sleep -Seconds 3
}
# wait for systemd to come up
$ok = $false
foreach ($i in 1..20) {
    $st = (& wsl.exe -d $Distro -u root -- sh -c "systemctl is-system-running 2>/dev/null") -join ""
    if ($st -match 'running|degraded|starting') { $ok = $true; break }
    Start-Sleep -Seconds 2
}
if ($ok) { Write-Ok "systemd is active in $Distro." } else { Write-Warn "systemd may not be fully up; continuing." }

# -- Step 4: run the opendray Linux installer inside WSL -------------
Write-Host ""
Write-Host "+-- Running the opendray installer inside $Distro --+" -ForegroundColor Cyan
Write-Host "You'll answer a few prompts (Postgres, admin password, listen address)." -ForegroundColor Yellow
Write-Host "Tip: choose 0.0.0.0:8770 for the listen address to reach it from other LAN devices." -ForegroundColor Yellow
Write-Host ""
& wsl.exe -d $Distro -u root -- bash -c "curl -fsSL $InstallSh | bash"
$installRc = $LASTEXITCODE
if ($installRc -ne 0) { Write-Err "The Linux installer exited with code $installRc. See output above."; exit $installRc }

# -- Step 5: persistence - keep the distro (and gateway) up ----------
# WSL stops idle distros, which would stop the systemd service. A hidden
# logon task boots the distro and holds it open with `sleep infinity`,
# so the gateway stays reachable at http://localhost:8770/.
Write-Info "Registering logon task '$TaskName' to keep the gateway running..."
try {
    $action  = New-ScheduledTaskAction -Execute "wsl.exe" `
        -Argument "-d $Distro -u root --exec /bin/sh -c `"exec sleep infinity`""
    $trigger = New-ScheduledTaskTrigger -AtLogOn
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries `
        -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit ([TimeSpan]::Zero)
    $principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Limited
    Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger `
        -Settings $settings -Principal $principal -Force | Out-Null
    Start-ScheduledTask -TaskName $TaskName
    Write-Ok "Keep-alive task registered and started."
} catch {
    Write-Warn "Could not register the keep-alive task: $($_.Exception.Message)"
    Write-Warn "The gateway will still run while a WSL session is open; to make it persistent,"
    Write-Warn "create a logon task that runs: wsl -d $Distro -u root --exec /bin/sh -c 'exec sleep infinity'"
}

Write-Host ""
Write-Ok "Done. Open the admin UI from Windows:  http://localhost:8770/admin/"
Write-Host "  (WSL forwards loopback to the Windows host. For LAN access from other"
Write-Host "   devices, they reach the Windows machine's IP - and you must have picked"
Write-Host "   0.0.0.0:8770 as the listen address during install.)"
Write-Host ""
