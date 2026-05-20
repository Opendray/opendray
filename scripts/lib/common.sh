#!/usr/bin/env bash
# scripts/lib/common.sh
# Shared shell helpers for the opendray installer wizards.
# Sourced by install-linux.sh and install-macos.sh.
# Compatible with bash 3.2+ (macOS ships 3.2 as /bin/bash). `printf -v`
# (bash 3.1+) and `read -ra` (bash 3.x) are fine; the only 4.0-only
# construct was ${var,,} case-conversion, replaced with tr.

# Detect colour-capable terminal once.
# Use $'…' (ANSI-C quoting) so the vars hold *real* ESC bytes — that
# lets `cat <<EOF` heredocs interpolate colours correctly, not just
# printf. The old '\033[…]m' form only worked when fed through printf
# (printf processes \033); heredocs printed the literal 4 characters.
if [ -t 1 ] && command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
    C_RED=$'\033[0;31m'
    C_GRN=$'\033[0;32m'
    C_YEL=$'\033[1;33m'
    C_BLU=$'\033[0;34m'
    C_CYA=$'\033[0;36m'
    C_DIM=$'\033[2m'
    C_NC=$'\033[0m'
else
    C_RED='' C_GRN='' C_YEL='' C_BLU='' C_CYA='' C_DIM='' C_NC=''
fi

# ── Logging ──────────────────────────────────────────────────────────

log_info()  { printf "${C_BLU}[*]${C_NC} %s\n" "$*"; }
log_ok()    { printf "${C_GRN}[✓]${C_NC} %s\n" "$*"; }
log_warn()  { printf "${C_YEL}[!]${C_NC} %s\n" "$*"; }
log_err()   { printf "${C_RED}[✗]${C_NC} %s\n" "$*" >&2; }
log_die()   { log_err "$*"; exit 1; }
log_dim()   { printf "${C_DIM}%s${C_NC}\n" "$*"; }

log_section() {
    printf "\n${C_CYA}━━━ %s ━━━${C_NC}\n\n" "$*"
}

log_step() {
    local n="$1"; shift
    printf "\n${C_BLU}┌── Step %s ── %s${C_NC}\n" "$n" "$*"
}

# ── Prompts ──────────────────────────────────────────────────────────

# ask_with_default <prompt> <default-or-empty> <out-var-name>
ask_with_default() {
    local prompt="$1"
    local default="$2"
    local var_name="$3"
    local response=""
    if [ -n "$default" ]; then
        printf "${C_BLU}?${C_NC} %s ${C_DIM}[%s]${C_NC}: " "$prompt" "$default"
    else
        printf "${C_BLU}?${C_NC} %s: " "$prompt"
    fi
    # `|| response=""` — never trip `set -e` if stdin EOFs (curl|bash with
    # no /dev/tty fallback, CI containers without a controlling terminal).
    read -r response || response=""
    printf -v "$var_name" '%s' "${response:-$default}"
}

# ask_password <prompt> <out-var-name>
ask_password() {
    local prompt="$1"
    local var_name="$2"
    local response=""
    printf "${C_BLU}?${C_NC} %s: " "$prompt"
    read -rs response || response=""
    echo
    printf -v "$var_name" '%s' "$response"
}

# ask_yes_no <prompt> <default y|n> <out-var-name>
ask_yes_no() {
    local prompt="$1"
    local default="${2:-n}"
    local var_name="$3"
    local hint response
    if [ "$default" = "y" ]; then hint="[Y/n]"; else hint="[y/N]"; fi
    while true; do
        printf "${C_BLU}?${C_NC} %s %s: " "$prompt" "$hint"
        response=""
        read -r response || response=""
        response="${response:-$default}"
        # tr instead of ${response,,} — the latter is bash 4.0+ and macOS
        # ships bash 3.2 as /bin/bash.
        case "$(printf '%s' "$response" | tr '[:upper:]' '[:lower:]')" in
            y|yes) printf -v "$var_name" '%s' 'y'; return 0 ;;
            n|no)  printf -v "$var_name" '%s' 'n'; return 0 ;;
            *) log_warn "Please answer y or n" ;;
        esac
    done
}

# ask_menu <prompt> "Option A|Option B|..." <out-var-name>
ask_menu() {
    local prompt="$1"
    local options_pipe="$2"
    local var_name="$3"
    local IFS='|'
    read -ra _menu_opts <<< "$options_pipe"
    local choice
    printf "${C_BLU}?${C_NC} %s\n" "$prompt"
    local i=1
    for opt in "${_menu_opts[@]}"; do
        printf "  ${C_BLU}%d)${C_NC} %s\n" "$i" "$opt"
        i=$((i + 1))
    done
    while true; do
        printf "  Enter 1-%d: " "${#_menu_opts[@]}"
        choice=""
        read -r choice || choice=""
        if [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le "${#_menu_opts[@]}" ]; then
            printf -v "$var_name" '%s' "${_menu_opts[$((choice - 1))]}"
            return 0
        fi
        log_warn "Please pick a number between 1 and ${#_menu_opts[@]}"
    done
}

# ── Random password generator ────────────────────────────────────────
gen_password() {
    local length="${1:-24}"
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c "$length"
    else
        # Fallback when openssl is missing — /dev/urandom + base64 from coreutils.
        LC_ALL=C tr -dc 'a-zA-Z0-9' </dev/urandom | head -c "$length"
    fi
}

# ── Tool detection ───────────────────────────────────────────────────

have_cmd() { command -v "$1" >/dev/null 2>&1; }

require_cmds() {
    local missing=()
    for c in "$@"; do
        have_cmd "$c" || missing+=("$c")
    done
    if [ "${#missing[@]}" -gt 0 ]; then
        log_die "Missing required tools: ${missing[*]}"
    fi
}

# Run a command with root privileges.
#   - If we're already root: exec directly.
#   - Otherwise, prefix with sudo.
# Don't pass sudo-only flags (-E, -u, etc.) here — those have to go through
# run_priv_env / run_priv_as so we don't leak them to root's direct exec.
run_priv() {
    if [ "$EUID" -eq 0 ]; then
        "$@"
    elif have_cmd sudo; then
        sudo "$@"
    else
        log_die "This step needs root; please install sudo or rerun as root."
    fi
}

# Run a command with root privileges, preserving the caller's environment.
#   - Root: env is already ours, just exec.
#   - Non-root: `sudo -E …` (preserve env). `-E` is a sudo flag; can't
#     pass it through run_priv when we're already root, because root
#     would try to exec `-E` as a command (that was the v2.0.0 bug).
run_priv_env() {
    if [ "$EUID" -eq 0 ]; then
        "$@"
    elif have_cmd sudo; then
        sudo -E "$@"
    else
        log_die "This step needs root; please install sudo or rerun as root."
    fi
}

# Run a command AS a specific (non-self) user.
#   - sudo available (root or not): `sudo -u <user> …` (works for both).
#   - root, no sudo: fall back to `runuser -u <user> -- …`.
#   - non-root, no sudo: dead end (can't switch user without privilege).
# Same reasoning as run_priv_env — `-u <user>` is a sudo flag that
# root's direct exec can't consume.
run_priv_as() {
    local target_user="$1"; shift
    if have_cmd sudo; then
        sudo -u "$target_user" "$@"
    elif [ "$EUID" -eq 0 ] && have_cmd runuser; then
        runuser -u "$target_user" -- "$@"
    else
        log_die "Need sudo or runuser to run as $target_user (running as $(id -un))."
    fi
}

# Same as run_priv_as, but preserves the caller's environment across
# the privilege transition. Used for `opendray migrate` in the wizard:
# the service isn't running yet (so systemd's EnvironmentFile injection
# hasn't happened), but we still want migrate to read
# OPENDRAY_DATABASE_URL / OPENDRAY_ADMIN_PASSWORD from the wizard's
# shell environment.
run_priv_as_env() {
    local target_user="$1"; shift
    if have_cmd sudo; then
        sudo -E -u "$target_user" "$@"
    elif [ "$EUID" -eq 0 ] && have_cmd runuser; then
        runuser --preserve-environment -u "$target_user" -- "$@"
    else
        log_die "Need sudo or runuser to run as $target_user (running as $(id -un))."
    fi
}

# Ensure either root or sudo is reachable for later privileged steps.
require_root_or_sudo() {
    if [ "$EUID" -eq 0 ]; then return 0; fi
    if have_cmd sudo; then
        log_info "Some steps need root; sudo will prompt when used."
        return 0
    fi
    log_die "Wizard needs root or sudo — install sudo or rerun as root."
}

# ── DSN helpers ──────────────────────────────────────────────────────

# Build a Postgres DSN. URL-encodes the password so special chars survive.
build_dsn() {
    local user="$1" pw="$2" host="$3" port="$4" db="$5"
    local pw_enc
    pw_enc="$(printf '%s' "$pw" | python3 -c 'import sys,urllib.parse;print(urllib.parse.quote(sys.stdin.read(),safe=""))' 2>/dev/null || true)"
    if [ -z "$pw_enc" ]; then
        # python3 unavailable — fall back to perl, then to raw (warn).
        pw_enc="$(printf '%s' "$pw" | perl -MURI::Escape -ne 'print uri_escape($_)' 2>/dev/null || true)"
    fi
    if [ -z "$pw_enc" ]; then
        log_warn "Could not URL-encode the password (no python3/perl); the DSN may break on special chars."
        pw_enc="$pw"
    fi
    printf 'postgres://%s:%s@%s:%s/%s?sslmode=disable' "$user" "$pw_enc" "$host" "$port" "$db"
}

# Test a Postgres DSN with a one-shot query. Returns 0 on success.
test_pg_dsn() {
    local user="$1" pw="$2" host="$3" port="$4" db="$5"
    PGPASSWORD="$pw" psql -h "$host" -p "$port" -U "$user" -d "$db" \
        -tAc "SELECT 1" >/dev/null 2>&1
}

# Check pgvector extension is installed in a given DB.
test_pg_has_vector() {
    local user="$1" pw="$2" host="$3" port="$4" db="$5"
    local result
    result="$(PGPASSWORD="$pw" psql -h "$host" -p "$port" -U "$user" -d "$db" \
        -tAc "SELECT 1 FROM pg_extension WHERE extname='vector'" 2>/dev/null || true)"
    [ "$result" = "1" ]
}

# ── Network helpers ──────────────────────────────────────────────────

# Probe whether a TCP port is bound on localhost. Returns 0 if free.
port_is_free() {
    local port="$1"
    if have_cmd ss; then
        ! ss -ltn "sport = :$port" | grep -q ":$port "
    elif have_cmd lsof; then
        ! lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
    else
        # No tool — assume free and let the gateway fail loudly on bind.
        return 0
    fi
}

# ── Cleanup trap registration ────────────────────────────────────────

_cleanup_files=()
register_cleanup_file() { _cleanup_files+=("$1"); }
_run_cleanup() {
    local f
    # Guard the expansion: under `set -u`, bash 3.2 (macOS /bin/bash)
    # errors on "${arr[@]}" when the array is empty — which it usually
    # is here, since most runs register no cleanup files. ${#arr[@]} is
    # safe to read when empty; only iterate when there's something.
    [ "${#_cleanup_files[@]}" -gt 0 ] || return 0
    for f in "${_cleanup_files[@]}"; do
        [ -f "$f" ] && rm -f -- "$f"
    done
}
trap _run_cleanup EXIT
