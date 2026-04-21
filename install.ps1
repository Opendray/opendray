# OpenDray — installer for Windows.
#
# Usage:
#   irm https://raw.githubusercontent.com/Opendray/opendray/main/install.ps1 | iex
#
# Environment variables (optional):
#   $env:OPENDRAY_VERSION       pin a specific tag (e.g. v0.4.0). Default: latest.
#   $env:OPENDRAY_INSTALL_DIR   binary destination. Default:
#                               $env:LOCALAPPDATA\Programs\OpenDray
#   $env:OPENDRAY_NO_SETUP      any value → skip auto-launching `opendray setup`.
#   $env:OPENDRAY_REPO          override "Opendray/opendray" (fork / mirror testing).
#
# The script:
#   1. detects architecture, refuses unsupported combos early
#   2. resolves the release tag (latest by default) via the GitHub API
#   3. downloads the raw binary + SHA256SUMS, verifies the checksum
#   4. installs to $OPENDRAY_INSTALL_DIR\opendray.exe
#   5. removes the zone-identifier ADS so SmartScreen doesn't block first run
#   6. adds the install dir to the current-user PATH (persisted via the
#      registry + broadcast so already-open shells see it too)
#   7. launches `opendray setup` in the same session.

$ErrorActionPreference = 'Stop'

# ── ASCII-only output. Windows Terminal handles ANSI fine but older
#    conhost + non-UTF-8 code pages butcher box-drawing characters.
function Say  { param([string]$m) Write-Host $m }
function Info { param([string]$m) Write-Host "    $m" }
function Step { param([string]$m) Write-Host "->  $m" }
function Ok   { param([string]$m) Write-Host "OK  $m" }
function Die  { param([string]$m) Write-Host "ERR $m" -ForegroundColor Red; exit 1 }

Say ""
Say "OpenDray installer"
Say "---------------------------------------------------------------"
Say ""

# ── settings ──────────────────────────────────────────────────
$Repo = if ($env:OPENDRAY_REPO) { $env:OPENDRAY_REPO } else { "Opendray/opendray" }
$InstallDir = if ($env:OPENDRAY_INSTALL_DIR) {
    $env:OPENDRAY_INSTALL_DIR
} else {
    Join-Path $env:LOCALAPPDATA "Programs\OpenDray"
}

# ── detect arch ───────────────────────────────────────────────
$archRaw = $env:PROCESSOR_ARCHITECTURE
# On 32-bit PowerShell running on a 64-bit OS, PROCESSOR_ARCHITECTURE is
# x86 even though the system is amd64. PROCESSOR_ARCHITEW6432 exposes
# the real architecture in that case.
if ($env:PROCESSOR_ARCHITEW6432) { $archRaw = $env:PROCESSOR_ARCHITEW6432 }
switch -regex ($archRaw) {
    'AMD64|x86_64'      { $Arch = 'amd64' }
    'ARM64|aarch64'     { $Arch = 'arm64' }
    default             { Die "Unsupported architecture: $archRaw. OpenDray supports amd64 and arm64." }
}

$Asset = "opendray-windows-$Arch.exe"
Step "Platform: windows/$Arch"

# ── resolve release tag ───────────────────────────────────────
if ([string]::IsNullOrEmpty($env:OPENDRAY_VERSION) -or $env:OPENDRAY_VERSION -eq 'latest') {
    Step "Resolving latest release"
    $api = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $api -UseBasicParsing -Headers @{ 'User-Agent' = 'opendray-install' }
    } catch {
        Die "GitHub API unreachable ($api).
    Retry later, or pin a version: `$env:OPENDRAY_VERSION='v0.4.0'; irm ... | iex"
    }
    $Tag = $release.tag_name
    if (-not $Tag) { Die "Could not parse release tag from GitHub API response." }
} else {
    $Tag = $env:OPENDRAY_VERSION
}
Info "Version: $Tag"

$BaseUrl  = "https://github.com/$Repo/releases/download/$Tag"
$AssetUrl = "$BaseUrl/$Asset"
$SumsUrl  = "$BaseUrl/SHA256SUMS"

# ── install dir ───────────────────────────────────────────────
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
$Target = Join-Path $InstallDir "opendray.exe"

# ── download binary ───────────────────────────────────────────
$Tmp = [System.IO.Path]::GetTempFileName()
$TmpSums = "$Tmp.sums"
try {
    Step "Downloading $Asset"
    try {
        Invoke-WebRequest -Uri $AssetUrl -OutFile $Tmp -UseBasicParsing
    } catch {
        Die "Download failed: $AssetUrl
    The release might not yet have a windows/$Arch build."
    }

    # ── verify SHA256 ─────────────────────────────────────────
    Step "Verifying checksum"
    try {
        Invoke-WebRequest -Uri $SumsUrl -OutFile $TmpSums -UseBasicParsing
    } catch {
        Die "Could not download SHA256SUMS from $SumsUrl
    Release is missing its checksum file. Refusing to install unverified binary."
    }

    $expected = $null
    foreach ($line in Get-Content $TmpSums) {
        # Format: "<sha>  <filename>"  (two spaces between on goreleaser output)
        if ($line -match "^([0-9a-fA-F]{64})\s+$([regex]::Escape($Asset))\s*$") {
            $expected = $Matches[1].ToLower()
            break
        }
    }
    if (-not $expected) {
        Die "$Asset is not listed in SHA256SUMS.
    Either the release is incomplete or the asset name changed.
    Refusing to install unverified binary."
    }

    $actual = (Get-FileHash -Path $Tmp -Algorithm SHA256).Hash.ToLower()
    if ($expected -ne $actual) {
        Die "SHA256 mismatch for $Asset
    expected: $expected
    got:      $actual
    The downloaded binary is either corrupt or tampered with.
    Aborting install."
    }
    Ok "SHA256 verified"

    # ── move into place ──────────────────────────────────────
    Step "Installing to $Target"
    Move-Item -Path $Tmp -Destination $Target -Force

    # Remove the Zone.Identifier alternate-data-stream so SmartScreen
    # doesn't block the first run with a "unknown publisher" modal.
    Unblock-File -Path $Target -ErrorAction SilentlyContinue

} finally {
    if (Test-Path $Tmp)     { Remove-Item $Tmp     -Force -ErrorAction SilentlyContinue }
    if (Test-Path $TmpSums) { Remove-Item $TmpSums -Force -ErrorAction SilentlyContinue }
}

Ok "Installed opendray $Tag"

# ── PATH update (HKCU) ────────────────────────────────────────
# Update the current user's persistent PATH, then broadcast the change
# so open shells + Explorer refresh without a logoff. We only touch
# HKCU — never machine-wide — to avoid needing admin rights.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ([string]::IsNullOrEmpty($userPath)) { $userPath = '' }

$pathParts = $userPath -split ';' | Where-Object { $_ -ne '' }
$already = $pathParts | Where-Object {
    $_.TrimEnd('\') -eq $InstallDir.TrimEnd('\')
} | Select-Object -First 1

if (-not $already) {
    $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Ok "Added $InstallDir to user PATH"
    Info "(New shells will see it automatically. Restart existing ones.)"
}

# Also update the CURRENT process's PATH so the immediate setup launch
# resolves `opendray.exe`. Without this, `& $Target setup` still works
# (we use the absolute path) but a subsequent manual `opendray` call
# in this same shell would fail.
$env:Path = "$env:Path;$InstallDir"

# ── hand off to setup wizard ──────────────────────────────────
if ($env:OPENDRAY_NO_SETUP) {
    Say ""
    Say "Skipping setup (OPENDRAY_NO_SETUP is set)."
    Say "Run when ready:"
    Say ""
    Say "    opendray setup"
    Say ""
    exit 0
}

Say ""
Say "Starting setup wizard..."
Say ""

# PowerShell passes stdin/out through directly, so the interactive
# wizard's prompts work as expected.
& $Target setup
exit $LASTEXITCODE
