# Backup — targets

A *target* is "where the encrypted bundle ends up." opendray ships
with **6 target kinds** covering ≈99% of where users want their
backups to go. All of them encrypt sensitive fields (passwords,
secret keys, private keys) with the master backup passphrase
before persisting to `backup_targets.config`; nothing sensitive
ever appears in API responses or PG dumps.

| Kind | Covers | Quick example |
|---|---|---|
| `local` | single-machine, Docker volume, mounted external HDD | `~/.opendray/backups/` |
| `smb` | Windows shares, home NAS (Synology / QNAP / UNAS) | `//192.168.9.8/Claude_Workspace` |
| `s3` | AWS S3, Cloudflare R2, B2, MinIO, Alibaba Cloud OSS (阿里云 OSS), Tencent Cloud COS (腾讯云 COS), ... | `s3://opendray@s3.amazonaws.com` |
| `webdav` | Nextcloud, ownCloud, Synology DSM (群晖 DSM), Box, Jianguoyun (坚果云) | `https://cloud.example.com/dav/` |
| `sftp` | Hetzner Storage Box, self-hosted VPS, home Linux | `backup@vps.example.com:22` |
| `rclone` | 70+ extra (Google Drive, OneDrive, Dropbox, Baidu Pan (百度网盘), Aliyun Drive (阿里云盘), ...) | `gdrive:opendray-backups` |

Add or edit targets at **`/backups → Targets`** or **`/settings →
Backup → Where backups go`**. Both surfaces use the same
TargetEditor dialog with kind-specific fields.

---

## `local`

Writes blobs into a directory on the same host running opendray.

- Default root: `~/.opendray/backups/` (or `cfg.backup.local_dir`).
- Atomic: write to `.<id>.tar.gz.enc.part` then rename. Crash
  mid-write leaves only the temp file; next scheduler tick GCs it.
- The `local` target is auto-created on first boot.

**When to use**: dev / single-server, or as a staging directory
that an OS-level rsync / Tailscale-mounted volume copies elsewhere.

**Trade-off**: bound to one machine — if the box dies the backup
dies with it. For real disaster recovery pair `local` with at
least one off-host target.

---

## `smb` (CIFS / Windows shares)

Writes to any SMB / CIFS share via a pure-Go SMB2 client (no host
`cifs-utils` dependency, so it works inside an unprivileged LXC
or Docker container).

| Field | Example |
|---|---|
| Host | `192.168.9.8` |
| Port | `445` (default) |
| Share | `Claude_Workspace` |
| User | `opendray` |
| Password | stored AES-GCM-encrypted |
| Path prefix | `opendray/backups` |

**When to use**: home NAS appliances (Synology / QNAP / UNAS),
Windows file servers, AD-joined shares.

**Test connection** does a real write+remove probe under
`<share>/<path_prefix>/.healthcheck-<random>`.

---

## `s3` — S3-compatible

A single target kind that talks to **every S3-compatible service**.
Configure with the right `endpoint` for your provider:

| Provider | Endpoint | Region |
|---|---|---|
| AWS S3 | `s3.amazonaws.com` (or `s3.<region>.amazonaws.com`) | e.g. `us-east-1` |
| Cloudflare R2 | `<account-id>.r2.cloudflarestorage.com` | `auto` |
| Backblaze B2 | `s3.<region>.backblazeb2.com` | e.g. `us-west-001` |
| Alibaba Cloud OSS (阿里云 OSS) | `oss-<region>.aliyuncs.com` | `oss-cn-shanghai` etc. |
| Tencent Cloud COS (腾讯云 COS) | `cos.<region>.myqcloud.com` | `ap-shanghai` etc. |
| MinIO self-hosted | `minio.local:9000` | `us-east-1` (or any) |
| DigitalOcean Spaces | `<region>.digitaloceanspaces.com` | e.g. `sgp1` |
| Wasabi | `s3.<region>.wasabisys.com` | e.g. `us-east-1` |

Required fields: `endpoint`, `region`, `bucket`, `access_key`,
`secret_key`. Optional: `path_prefix`, `use_ssl` (default true),
`path_style` (legacy / MinIO).

**When to use**: cloud-first deployments, or when the operator
wants tiered durability (R2 is free up to 10 GB, B2 cheap, AWS
Glacier for archive).

**Test connection** runs `BucketExists` (HEAD /<bucket>) then a
write+remove probe — confirms credentials AND write permissions.

**Cost knobs**: enable bucket lifecycle rules at the provider for
"transition to IA / Glacier after N days" if you keep many backups.

---

## `webdav`

Spawns standard WebDAV PUT / GET / DELETE / PROPFIND requests
against any HTTP(S) server speaking class-2 WebDAV.

| Field | Example |
|---|---|
| Base URL | full URL incl. trailing slash, e.g. `https://cloud.example.com/remote.php/dav/files/me/` |
| User | usually a username; for some services this is an app password |
| Password | stored AES-GCM-encrypted |
| Path prefix | `opendray/backups` |

**Provider-specific URL shapes**:

```
Nextcloud:   https://cloud.example.com/remote.php/dav/files/<user>/
ownCloud:    https://cloud.example.com/remote.php/webdav/
Synology:    https://nas.local:5006/    (DSM Web Station + WebDAV)
Box.com:     https://dav.box.com/dav
Jianguoyun:  https://dav.jianguoyun.com/dav/   (use a "third-party app" password)
```

**When to use**: self-hosted clouds and "I have an app password
but no S3" scenarios.

**Test connection** does PROPFIND on `/` then a write+remove
probe under `<base>/<path_prefix>/.healthcheck-<random>`.

**Limitation**: gowebdav has no streaming-upload, so opendray
spools the bundle to a temp file first. Acceptable for backups
≤ 1 GiB; larger should prefer SFTP or S3.

---

## `sftp`

Writes via OpenSSH's SFTP subsystem. Works against literally any
SSH server with `internal-sftp` or `sftp-server` enabled (the
default on every Linux distro).

| Field | Example |
|---|---|
| Host | `vps.example.com` or `192.168.1.50` |
| Port | `22` (default) |
| User | `backup` |
| Password | password OR key passphrase (when paired with `private_key`) |
| Private key | full PEM contents of an OpenSSH/PEM private key — leave blank for password auth |
| Host key | `ssh-ed25519 AAAA…` from `ssh-keyscan host` (recommended for non-LAN targets) |
| Path prefix | absolute (`/var/backups/opendray`) or relative-to-home (`opendray-backups`) |

**Auth modes**:
- `password` set, `private_key` blank → password auth
- `private_key` set → public-key auth; if `password` is also
  set it's treated as the key passphrase
- both blank → rejected at create time

**Host key pinning**: leaving `host_key` blank disables pinning
(ssh-then-trust). Strongly encouraged to pin for any non-LAN
target — otherwise a MITM at L3 can capture the encrypted bundle
upload (it's still encrypted at rest by opendray's cipher, but
host-key pinning is defense-in-depth).

**When to use**: any host you can SSH into. Hetzner Storage Box
(`<user>@<user>.your-storagebox.de`), self-hosted VPS, even your
home Linux desktop with a port-forward.

**Test connection** opens SSH+SFTP, mkdir's the prefix, write+
remove probe.

---

## `rclone` (passthrough — 70+ extra backends)

For backends not natively supported above. Requires the **rclone
CLI** installed on the opendray host:

```bash
brew install rclone        # macOS
apk add rclone             # Alpine (in your LXC / Docker)
curl https://rclone.org/install.sh | sudo bash    # generic
```

Then on the operator host, configure your remote interactively:

```bash
rclone config
# > n (new remote)
# > name: gdrive
# > storage: 13 (Google Drive)
# > follow OAuth prompts
```

In opendray, add a target with kind `rclone` and `remote = "gdrive"`
(no colon). Optionally: `path_prefix`, `binary_path` (override
PATH lookup), `config_path` (override `~/.config/rclone/rclone.conf`).

**Backends rclone unlocks** (sample — full list at
[rclone.org/docs](https://rclone.org/docs/)):

```
Google Drive · OneDrive · Dropbox · iCloud-via-WebDAV
Baidu Pan (百度网盘) · Aliyun Drive (阿里云盘, via aliyundrive-fuse) · pCloud · Mega
Microsoft Graph (SharePoint) · Yandex Disk · Mail.ru Cloud
HiDrive · Internet Archive · Jottacloud · Koofr · Mailbox.org
Mega · Memory (testing) · Microsoft OneDrive · OpenStack Swift
Oracle Cloud Storage · pCloud · premiumize.me · QingStor
SeaTable · Seafile · Sharepoint · SugarSync · Tardigrade · Yandex
…and ≈40 more
```

**When to use**:
- Consumer cloud storage (Google Drive / OneDrive / Dropbox)
- Mainland China services (百度网盘 / 阿里云盘) where direct
  API access is hostile to write-heavy backup tooling
- Anything else rclone supports natively

**Trade-off**: extra dependency (the rclone binary), per-op
subprocess spawn. Throughput is fine for backup cadence (daily
runs, minutes apart) but not great for streaming-heavy workloads.
Native targets are preferred where they exist.

**Test connection** runs `rclone lsd <remote>:<path_prefix>` —
confirms auth + reachability + (optionally) creates the prefix
folder if it didn't exist.

---

## What gets persisted

`backup_targets` in PG holds:

- `id` — operator-chosen or auto-generated (`tgt_…`).
- `kind` — one of the 6 above.
- `config` — JSONB. Sensitive fields (`password`, `secret_key`,
  `private_key`) are AES-256-GCM wrapped using the master backup
  passphrase before write — leaked DB row alone doesn't reveal
  the credential.
- `enabled` — when false, target stays configured but is excluded
  from the runtime registry (cannot be picked for a backup until
  toggled on).

`GET /api/v1/backup-targets` always returns redacted config —
sensitive keys come back as `"********"` strings. The plaintext
form only ever exists in two places: in-memory inside the running
opendray process, and in the operator's secrets manager
(externally — opendray doesn't try to expose ciphertext).

## What's still missing in v1

- **Edit existing target** — TargetEditor only handles create.
  To change a target's config you delete + recreate. v1.1.
- **Free-space reporting** — UI doesn't query the target for
  capacity. For now operators monitor the volume / quota at the
  storage layer.
- **Multi-target schedules** — a schedule writes to one target.
  "Mirror to two targets" requires two schedules.
