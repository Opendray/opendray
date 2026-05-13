import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/backups_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// BackupTargetEditorScreen — create / edit a backup destination
// from mobile. Pre-PR-51 this was web-only on the grounds that
// "S3 / SMB / SFTP each have 5+ fields and long secret pastes
// would be painful on a phone." With password managers' autofill
// + clipboard paste, the real friction is lower than that
// assumption; closing the gap unblocks the "operator runs the
// whole gateway from their phone" goal.
//
// One screen handles all six kinds (local / smb / s3 / webdav /
// sftp / rclone). The kind picker lives at the top; per-kind
// fields swap below. Submit POSTs /api/v1/backup-targets for
// create or PATCH /api/v1/backup-targets/{id} for edit.
//
// Edit mode quirk: the server redacts secrets on GET (returns
// '***' instead of plaintext for password / secret_key / etc.).
// In edit mode we leave secret fields blank — empty submission
// means "don't change" (server-side PATCH config is partial),
// so the user only has to retype a secret when they actually
// want to rotate it.

enum TargetKind { local, smb, s3, webdav, sftp, rclone }

extension TargetKindLabel on TargetKind {
  String get apiValue => switch (this) {
        TargetKind.local => 'local',
        TargetKind.smb => 'smb',
        TargetKind.s3 => 's3',
        TargetKind.webdav => 'webdav',
        TargetKind.sftp => 'sftp',
        TargetKind.rclone => 'rclone',
      };

  String get label => switch (this) {
        TargetKind.local => 'Local disk',
        TargetKind.smb => 'SMB / CIFS',
        TargetKind.s3 => 'S3-compatible',
        TargetKind.webdav => 'WebDAV',
        TargetKind.sftp => 'SFTP / SSH',
        TargetKind.rclone => 'rclone passthrough',
      };

  String get description => switch (this) {
        TargetKind.local => 'Folder on the machine running opendray',
        TargetKind.smb => 'Windows shares + most home NAS appliances',
        TargetKind.s3 => 'AWS S3 + every S3-compatible service',
        TargetKind.webdav => 'Self-hosted clouds + file-sharing services',
        TargetKind.sftp => 'Any SSH-accessible server',
        TargetKind.rclone => '70+ backends via the rclone CLI',
      };

  IconData get icon => switch (this) {
        TargetKind.local => Icons.folder_outlined,
        TargetKind.smb => Icons.dns_outlined,
        TargetKind.s3 => Icons.cloud_outlined,
        TargetKind.webdav => Icons.cloud_queue_outlined,
        TargetKind.sftp => Icons.terminal_outlined,
        TargetKind.rclone => Icons.swap_horiz_outlined,
      };

  static TargetKind fromApi(String s) => switch (s) {
        'local' => TargetKind.local,
        'smb' => TargetKind.smb,
        's3' => TargetKind.s3,
        'webdav' => TargetKind.webdav,
        'sftp' => TargetKind.sftp,
        'rclone' => TargetKind.rclone,
        _ => TargetKind.local,
      };
}

class BackupTargetEditorScreen extends ConsumerStatefulWidget {
  const BackupTargetEditorScreen({
    super.key,
    this.existing,
  });

  /// When non-null, the screen opens in edit mode pre-filled from
  /// this target. Editing the kind itself is disabled (creating a
  /// new target is the right path for that).
  final BackupTarget? existing;

  @override
  ConsumerState<BackupTargetEditorScreen> createState() =>
      _BackupTargetEditorScreenState();
}

class _BackupTargetEditorScreenState
    extends ConsumerState<BackupTargetEditorScreen> {
  late TargetKind _kind;
  late final TextEditingController _idCtrl;
  bool _enabled = true;
  bool _submitting = false;
  String? _error;

  // Per-kind text controllers. Built lazily on first use to avoid
  // 30+ pre-allocated controllers cluttering memory. Disposed in
  // dispose() for the ones we did allocate.
  final Map<String, TextEditingController> _ctrls = {};

  // Bool fields (S3 use_ssl / path_style). Defaulted on first
  // build of S3 form.
  bool? _useSsl;
  bool? _pathStyle;

  bool get _isEdit => widget.existing != null;

  @override
  void initState() {
    super.initState();
    final ex = widget.existing;
    _kind = ex != null ? TargetKindLabel.fromApi(ex.kind) : TargetKind.local;
    _idCtrl = TextEditingController(text: ex?.id ?? '');
    _enabled = ex?.enabled ?? true;
    if (ex != null) {
      _prefillFromConfig(ex.config);
    }
  }

  void _prefillFromConfig(Map<String, dynamic> cfg) {
    // Walk the entire config map and dump string values into
    // controllers keyed by field name. The per-kind form widgets
    // use the same keys (`_ctrl('host')`) so this just works.
    // Booleans get their dedicated state fields.
    cfg.forEach((k, v) {
      if (v is bool) {
        if (k == 'use_ssl') _useSsl = v;
        if (k == 'path_style') _pathStyle = v;
        return;
      }
      // Server redacts secrets to '***' on GET. Leave those blank
      // in edit mode — submitting empty means "don't change"
      // (PATCH config is partial). If the operator wants to
      // rotate, they type a new value.
      if (_isSecretField(k) && _looksRedacted(v)) {
        _ctrls[k] = TextEditingController();
        return;
      }
      _ctrls[k] = TextEditingController(text: v?.toString() ?? '');
    });
  }

  bool _isSecretField(String name) => switch (name) {
        'password' || 'secret_key' || 'private_key' => true,
        _ => false,
      };

  bool _looksRedacted(Object? v) {
    final s = v?.toString() ?? '';
    return s == '***' || s == '<redacted>' || s.isEmpty;
  }

  TextEditingController _ctrl(String name) =>
      _ctrls.putIfAbsent(name, TextEditingController.new);

  @override
  void dispose() {
    _idCtrl.dispose();
    for (final c in _ctrls.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final api = ref.read(backupsApiProvider);
      final config = _buildConfig();
      if (_isEdit) {
        await api.updateTarget(
          widget.existing!.id,
          config: config.isEmpty ? null : config,
          enabled: _enabled,
        );
      } else {
        await api.createTarget(
          id: _idCtrl.text.trim().isEmpty ? null : _idCtrl.text.trim(),
          kind: _kind.apiValue,
          config: config,
          enabled: _enabled,
        );
      }
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _error = e.message;
          _submitting = false;
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _submitting = false;
        });
      }
    }
  }

  // Walks the active controllers + bool fields and produces the
  // server-shaped config map. Empty strings are stripped so the
  // backend uses its defaults — except in edit mode where empty
  // means "leave alone" (handled by the PATCH partial semantics).
  Map<String, dynamic> _buildConfig() {
    final out = <String, dynamic>{};
    void putString(String key) {
      final v = _ctrls[key]?.text.trim() ?? '';
      if (v.isEmpty) return;
      out[key] = v;
    }

    void putInt(String key, int fallback) {
      final raw = _ctrls[key]?.text.trim() ?? '';
      if (raw.isEmpty) return;
      out[key] = int.tryParse(raw) ?? fallback;
    }

    switch (_kind) {
      case TargetKind.local:
        putString('root');
      case TargetKind.smb:
        putString('host');
        putInt('port', 445);
        putString('share');
        putString('user');
        putString('password');
        putString('path_prefix');
      case TargetKind.s3:
        putString('endpoint');
        putString('region');
        putString('bucket');
        putString('access_key');
        putString('secret_key');
        putString('path_prefix');
        if (_useSsl != null) out['use_ssl'] = _useSsl;
        if (_pathStyle != null) out['path_style'] = _pathStyle;
      case TargetKind.webdav:
        putString('base_url');
        putString('user');
        putString('password');
        putString('path_prefix');
      case TargetKind.sftp:
        putString('host');
        putInt('port', 22);
        putString('user');
        putString('password');
        putString('private_key');
        putString('host_key');
        putString('path_prefix');
      case TargetKind.rclone:
        putString('remote');
        putString('path_prefix');
        putString('binary_path');
        putString('config_path');
    }
    return out;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(_isEdit ? 'Edit target' : 'New backup target'),
      ),
      body: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
          children: [
            _kindPicker(theme),
            const SizedBox(height: 16),
            if (!_isEdit) ...[
              _label('ID (optional)'),
              const SizedBox(height: 4),
              TextField(
                controller: _idCtrl,
                decoration: _inputDeco(hint: 'Auto: ${_kind.apiValue}-1'),
                style: _monoStyle,
              ),
              const SizedBox(height: 4),
              Text(
                'Lower-case letters, digits, dashes. Defaults to the next '
                'available "{kind}-N".',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.outline),
              ),
              const SizedBox(height: 16),
            ],
            _form(theme),
            const SizedBox(height: 16),
            SwitchListTile.adaptive(
              value: _enabled,
              title: Text(t.common.enabled),
              subtitle: Text(
                _enabled
                    ? 'Scheduled and ad-hoc backups can target this.'
                    : 'Server will refuse to write backups here.',
                style: theme.textTheme.bodySmall,
              ),
              contentPadding: EdgeInsets.zero,
              onChanged: (v) => setState(() => _enabled = v),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: theme.colorScheme.error.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(6),
                  border: Border.all(
                    color: theme.colorScheme.error.withValues(alpha: 0.4),
                  ),
                ),
                child: Text(
                  _error!,
                  style:
                      TextStyle(color: theme.colorScheme.error, fontSize: 12),
                ),
              ),
            ],
            const SizedBox(height: 20),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed:
                        _submitting ? null : () => Navigator.of(context).pop(),
                    child: Text(t.common.cancel),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton.icon(
                    onPressed: _submitting ? null : _submit,
                    icon: _submitting
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.check, size: 18),
                    label: Text(_submitting
                        ? 'Saving…'
                        : _isEdit
                            ? 'Save'
                            : 'Create'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _kindPicker(ThemeData theme) {
    if (_isEdit) {
      // Editing the kind would orphan existing backups whose
      // target-id points at this row. Render it as a non-
      // interactive read-only banner instead.
      return Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color:
              theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: theme.dividerColor),
        ),
        child: Row(
          children: [
            Icon(_kind.icon, size: 22, color: theme.colorScheme.outline),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(_kind.label,
                      style: const TextStyle(fontWeight: FontWeight.w600)),
                  Text(_kind.description,
                      style: theme.textTheme.bodySmall),
                ],
              ),
            ),
          ],
        ),
      );
    }
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: TargetKind.values.map((k) {
        final active = k == _kind;
        return ChoiceChip(
          selected: active,
          avatar: Icon(k.icon, size: 16),
          label: Text(k.label),
          onSelected: (s) {
            if (!s) return;
            setState(() {
              _kind = k;
              // Clear per-kind fields so a leftover S3 secret_key
              // doesn't get submitted with a fresh SMB form.
              for (final c in _ctrls.values) {
                c.dispose();
              }
              _ctrls.clear();
              _useSsl = null;
              _pathStyle = null;
              _error = null;
            });
          },
        );
      }).toList(),
    );
  }

  // ── Per-kind field forms ─────────────────────────────────────

  Widget _form(ThemeData theme) {
    return switch (_kind) {
      TargetKind.local => _localForm(theme),
      TargetKind.smb => _smbForm(theme),
      TargetKind.s3 => _s3Form(theme),
      TargetKind.webdav => _webdavForm(theme),
      TargetKind.sftp => _sftpForm(theme),
      TargetKind.rclone => _rcloneForm(theme),
    };
  }

  Widget _localForm(ThemeData theme) {
    return _field(
      label: 'Root directory',
      hint: 'Empty = cfg.backup.local_dir (~/.opendray/backups)',
      child: TextField(
        controller: _ctrl('root'),
        decoration:
            _inputDeco(hint: '~/backups/opendray  ·  /mnt/hdd/opendray'),
        style: _monoStyle,
      ),
    );
  }

  Widget _smbForm(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              flex: 3,
              child: _field(
                label: 'Host',
                child: TextField(
                  controller: _ctrl('host'),
                  decoration: _inputDeco(hint: '192.168.9.8'),
                  style: _monoStyle,
                  keyboardType: TextInputType.url,
                  autocorrect: false,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _field(
                label: 'Port',
                child: TextField(
                  controller: _ctrl('port'),
                  decoration: _inputDeco(hint: '445'),
                  keyboardType: TextInputType.number,
                ),
              ),
            ),
          ],
        ),
        _field(
          label: 'Share',
          hint: 'Top-level share name',
          child: TextField(
            controller: _ctrl('share'),
            decoration: _inputDeco(hint: 'Claude_Workspace'),
          ),
        ),
        Row(
          children: [
            Expanded(
              child: _field(
                label: 'User',
                child: TextField(
                  controller: _ctrl('user'),
                  decoration: _inputDeco(),
                  autocorrect: false,
                  textCapitalization: TextCapitalization.none,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _field(
                label: 'Password',
                hint: _isEdit ? 'Leave blank to keep current' : null,
                child: TextField(
                  controller: _ctrl('password'),
                  decoration: _inputDeco(),
                  obscureText: true,
                  autocorrect: false,
                ),
              ),
            ),
          ],
        ),
        _field(
          label: 'Path prefix',
          hint: 'Sub-folder under the share root (optional)',
          child: TextField(
            controller: _ctrl('path_prefix'),
            decoration: _inputDeco(hint: 'opendray/backups'),
            style: _monoStyle,
          ),
        ),
      ],
    );
  }

  Widget _s3Form(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _field(
          label: 'Endpoint',
          hint:
              'AWS: s3.amazonaws.com  ·  R2: <id>.r2.cloudflarestorage.com  ·  MinIO: minio.local:9000',
          child: TextField(
            controller: _ctrl('endpoint'),
            decoration: _inputDeco(hint: 's3.amazonaws.com'),
            style: _monoStyle,
            keyboardType: TextInputType.url,
            autocorrect: false,
          ),
        ),
        Row(
          children: [
            Expanded(
              child: _field(
                label: 'Region',
                hint: 'AWS only; R2 = auto',
                child: TextField(
                  controller: _ctrl('region'),
                  decoration: _inputDeco(hint: 'us-east-1'),
                  style: _monoStyle,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _field(
                label: 'Bucket',
                child: TextField(
                  controller: _ctrl('bucket'),
                  decoration: _inputDeco(hint: 'opendray-backups'),
                  style: _monoStyle,
                ),
              ),
            ),
          ],
        ),
        _field(
          label: 'Access key',
          child: TextField(
            controller: _ctrl('access_key'),
            decoration: _inputDeco(),
            style: _monoStyle,
            autocorrect: false,
          ),
        ),
        _field(
          label: 'Secret key',
          hint: _isEdit
              ? 'Leave blank to keep current. Stored AES-256-GCM encrypted.'
              : 'Stored AES-256-GCM encrypted; never echoed back.',
          child: TextField(
            controller: _ctrl('secret_key'),
            decoration: _inputDeco(),
            obscureText: true,
            style: _monoStyle,
            autocorrect: false,
          ),
        ),
        _field(
          label: 'Path prefix',
          hint: 'Object-key prefix (optional)',
          child: TextField(
            controller: _ctrl('path_prefix'),
            decoration: _inputDeco(hint: 'opendray/backups'),
            style: _monoStyle,
          ),
        ),
        const SizedBox(height: 4),
        SwitchListTile.adaptive(
          value: _useSsl ?? true,
          onChanged: (v) => setState(() => _useSsl = v),
          title: Text(t.backupTargetEditor.useHttps),
          dense: true,
          contentPadding: EdgeInsets.zero,
        ),
        SwitchListTile.adaptive(
          value: _pathStyle ?? false,
          onChanged: (v) => setState(() => _pathStyle = v),
          title: Text(t.backupTargetEditor.pathStyle),
          subtitle: Text(t.backupTargetEditor.pathStyleSubtitle,
              style: const TextStyle(fontSize: 11)),
          dense: true,
          contentPadding: EdgeInsets.zero,
        ),
      ],
    );
  }

  Widget _webdavForm(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _field(
          label: 'Base URL',
          hint: 'Full URL including path. Nextcloud: '
              'https://cloud.example.com/remote.php/dav/files/<user>/',
          child: TextField(
            controller: _ctrl('base_url'),
            decoration:
                _inputDeco(hint: 'https://cloud.example.com/remote.php/...'),
            style: _monoStyle,
            keyboardType: TextInputType.url,
            autocorrect: false,
          ),
        ),
        Row(
          children: [
            Expanded(
              child: _field(
                label: 'User',
                child: TextField(
                  controller: _ctrl('user'),
                  decoration: _inputDeco(),
                  autocorrect: false,
                  textCapitalization: TextCapitalization.none,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _field(
                label: 'Password',
                hint: _isEdit ? 'Leave blank to keep' : null,
                child: TextField(
                  controller: _ctrl('password'),
                  decoration: _inputDeco(),
                  obscureText: true,
                  autocorrect: false,
                ),
              ),
            ),
          ],
        ),
        _field(
          label: 'Path prefix',
          hint: 'Sub-folder under the base URL (optional)',
          child: TextField(
            controller: _ctrl('path_prefix'),
            decoration: _inputDeco(hint: 'opendray/backups'),
            style: _monoStyle,
          ),
        ),
      ],
    );
  }

  Widget _sftpForm(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              flex: 3,
              child: _field(
                label: 'Host',
                child: TextField(
                  controller: _ctrl('host'),
                  decoration: _inputDeco(hint: 'vps.example.com'),
                  style: _monoStyle,
                  keyboardType: TextInputType.url,
                  autocorrect: false,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _field(
                label: 'Port',
                child: TextField(
                  controller: _ctrl('port'),
                  decoration: _inputDeco(hint: '22'),
                  keyboardType: TextInputType.number,
                ),
              ),
            ),
          ],
        ),
        _field(
          label: 'User',
          child: TextField(
            controller: _ctrl('user'),
            decoration: _inputDeco(),
            style: _monoStyle,
            autocorrect: false,
            textCapitalization: TextCapitalization.none,
          ),
        ),
        _field(
          label: 'Password',
          hint: _isEdit
              ? 'Leave blank to keep. If both password + private key are '
                  'provided, password is treated as the key passphrase.'
              : 'Either password OR private key. If both, password becomes '
                  'the key passphrase.',
          child: TextField(
            controller: _ctrl('password'),
            decoration: _inputDeco(),
            obscureText: true,
            autocorrect: false,
          ),
        ),
        _field(
          label: 'Private key (PEM)',
          hint: _isEdit
              ? 'Leave blank to keep. Paste OpenSSH/PEM contents.'
              : 'Paste the contents of an OpenSSH/PEM private key. '
                  'Multi-line input — keep the BEGIN/END markers.',
          child: TextField(
            controller: _ctrl('private_key'),
            decoration: _inputDeco(
              hint: '-----BEGIN OPENSSH PRIVATE KEY-----',
            ),
            style: _monoStyle.copyWith(fontSize: 11),
            maxLines: 4,
            minLines: 3,
            autocorrect: false,
          ),
        ),
        _field(
          label: 'Host key (pinning)',
          hint:
              'OpenSSH-style server public key. `ssh-keyscan <host>` to obtain. '
              'Blank = no pinning (NOT recommended outside LAN).',
          child: TextField(
            controller: _ctrl('host_key'),
            decoration: _inputDeco(hint: 'ssh-ed25519 AAAA...'),
            style: _monoStyle.copyWith(fontSize: 11),
            maxLines: 2,
            autocorrect: false,
          ),
        ),
        _field(
          label: 'Path prefix',
          hint: 'Absolute or relative to user home (optional)',
          child: TextField(
            controller: _ctrl('path_prefix'),
            decoration: _inputDeco(hint: '/var/backups/opendray'),
            style: _monoStyle,
          ),
        ),
      ],
    );
  }

  Widget _rcloneForm(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest
                .withValues(alpha: 0.4),
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: theme.dividerColor),
          ),
          child: Text(
            'Requires the rclone CLI on the opendray host. First run '
            '`rclone config` to set up your remote, then use its name '
            'below. opendray invokes rclone rcat / cat / lsd internally.',
            style: theme.textTheme.bodySmall,
          ),
        ),
        const SizedBox(height: 12),
        _field(
          label: 'Remote name',
          hint: 'Name from `rclone config` (no colon).',
          child: TextField(
            controller: _ctrl('remote'),
            decoration: _inputDeco(hint: 'gdrive'),
            style: _monoStyle,
            autocorrect: false,
            textCapitalization: TextCapitalization.none,
          ),
        ),
        _field(
          label: 'Path prefix',
          hint: 'Sub-folder under the remote root (optional)',
          child: TextField(
            controller: _ctrl('path_prefix'),
            decoration: _inputDeco(hint: 'opendray/backups'),
            style: _monoStyle,
          ),
        ),
        _field(
          label: 'Binary path',
          hint: 'Override `which rclone`. Empty = PATH lookup.',
          child: TextField(
            controller: _ctrl('binary_path'),
            decoration: _inputDeco(hint: '/opt/homebrew/bin/rclone'),
            style: _monoStyle,
            autocorrect: false,
          ),
        ),
        _field(
          label: 'Config path',
          hint: 'Override --config. Empty = rclone default.',
          child: TextField(
            controller: _ctrl('config_path'),
            decoration: _inputDeco(),
            style: _monoStyle,
            autocorrect: false,
          ),
        ),
      ],
    );
  }

  // ── Tiny presentation helpers ────────────────────────────────

  Widget _field({
    required String label,
    required Widget child,
    String? hint,
  }) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _label(label),
          const SizedBox(height: 4),
          child,
          if (hint != null) ...[
            const SizedBox(height: 4),
            Text(
              hint,
              style: TextStyle(
                fontSize: 11,
                color: Theme.of(context).colorScheme.outline,
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _label(String label) {
    return Text(
      label,
      style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
    );
  }

  InputDecoration _inputDeco({String? hint}) => InputDecoration(
        hintText: hint,
        isDense: true,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(6)),
      );

  static const TextStyle _monoStyle =
      TextStyle(fontFamily: 'monospace', fontSize: 13);
}
