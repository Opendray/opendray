import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/auth_service.dart';
import '../../core/services/l10n.dart';
import '../../core/services/server_config.dart';
import '../../core/services/server_profile.dart';
import '../../shared/theme/app_theme.dart';

/// Settings surface for the multi-URL server address book. Lists the
/// saved [ServerProfile]s, lets the user switch between them, and
/// opens the Add/Edit dialog. Wired into [SettingsPage] in place of
/// the legacy single-URL card.
class ServersCard extends StatelessWidget {
  const ServersCard({super.key});

  @override
  Widget build(BuildContext context) {
    final cfg = context.watch<ServerConfig>();
    final profiles = cfg.profiles;
    final active = cfg.activeProfile;

    // Sort: active first, then most-recently-used (or alphabetical for
    // profiles that never went active). Keeps the active profile at
    // the top where the user can see status and credentials at a glance.
    final sorted = [...profiles]..sort((a, b) {
        if (a.id == active?.id) return -1;
        if (b.id == active?.id) return 1;
        final ta = a.lastUsedAt?.millisecondsSinceEpoch ?? 0;
        final tb = b.lastUsedAt?.millisecondsSinceEpoch ?? 0;
        if (ta != tb) return tb.compareTo(ta);
        return a.alias.toLowerCase().compareTo(b.alias.toLowerCase());
      });

    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 12, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.dns, color: AppColors.accent, size: 20),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    context.tr('Servers'),
                    style: const TextStyle(
                        fontWeight: FontWeight.w600, fontSize: 15),
                  ),
                ),
                IconButton(
                  tooltip: context.tr('Add server'),
                  icon: const Icon(Icons.add, size: 20,
                      color: AppColors.accent),
                  onPressed: () => _openEditor(context, null),
                ),
              ],
            ),
            const SizedBox(height: 6),
            if (sorted.isEmpty)
              _EmptyState(onAdd: () => _openEditor(context, null))
            else
              ...[
                for (var i = 0; i < sorted.length; i++) ...[
                  _ProfileTile(
                    profile: sorted[i],
                    isActive: sorted[i].id == active?.id,
                    onTap: () => _switchTo(context, sorted[i]),
                    onEdit: () => _openEditor(context, sorted[i]),
                    onDelete: () => _confirmDelete(context, sorted[i]),
                  ),
                  if (i < sorted.length - 1)
                    const Divider(height: 1, color: AppColors.border),
                ],
              ],
          ],
        ),
      ),
    );
  }

  Future<void> _openEditor(
      BuildContext context, ServerProfile? existing) async {
    await showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (_) => ProfileEditorDialog(existing: existing),
    );
  }

  Future<void> _switchTo(BuildContext context, ServerProfile p) async {
    final cfg = context.read<ServerConfig>();
    if (p.id == cfg.activeId) return;
    await cfg.setActive(p.id);
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('${context.tr('Switched to')} ${p.alias}'),
        duration: const Duration(seconds: 2),
      ),
    );
  }

  Future<void> _confirmDelete(BuildContext context, ServerProfile p) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(context.tr('Delete server?')),
        content: Text(
          '${context.tr('Remove this server, its saved credentials, and its login token from this device.')}\n\n${p.alias}\n${p.url}',
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(context.tr('Cancel'))),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              style: FilledButton.styleFrom(backgroundColor: AppColors.error),
              child: Text(context.tr('Delete'))),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;
    await context.read<ServerConfig>().deleteProfile(p.id);
  }
}

class _EmptyState extends StatelessWidget {
  final VoidCallback onAdd;
  const _EmptyState({required this.onAdd});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(4, 10, 4, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            context.tr('No servers yet. Add one to connect.'),
            style: const TextStyle(
                fontSize: 13, color: AppColors.textMuted),
          ),
          const SizedBox(height: 10),
          OutlinedButton.icon(
            onPressed: onAdd,
            icon: const Icon(Icons.add, size: 16),
            label: Text(context.tr('Add server')),
            style: OutlinedButton.styleFrom(
              foregroundColor: AppColors.accent,
              side: const BorderSide(color: AppColors.border),
              padding: const EdgeInsets.symmetric(vertical: 10),
            ),
          ),
        ],
      ),
    );
  }
}

class _ProfileTile extends StatelessWidget {
  final ServerProfile profile;
  final bool isActive;
  final VoidCallback onTap;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  const _ProfileTile({
    required this.profile,
    required this.isActive,
    required this.onTap,
    required this.onEdit,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 10),
        child: Row(
          children: [
            // Active dot — green when selected, transparent otherwise so the
            // rows don't wobble horizontally when the active profile changes.
            Container(
              width: 8,
              height: 8,
              margin: const EdgeInsets.only(right: 10),
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: isActive ? AppColors.success : Colors.transparent,
                border: isActive
                    ? null
                    : Border.all(color: AppColors.border, width: 1),
              ),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Row(children: [
                    Flexible(
                      child: Text(
                        profile.alias.isEmpty
                            ? context.tr('(unnamed)')
                            : profile.alias,
                        style: TextStyle(
                          fontSize: 14,
                          fontWeight:
                              isActive ? FontWeight.w600 : FontWeight.w500,
                          color: AppColors.text,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    if (profile.username.isNotEmpty) ...[
                      const SizedBox(width: 8),
                      Text('· ${profile.username}',
                          style: const TextStyle(
                              fontSize: 12, color: AppColors.textMuted)),
                    ],
                    if (profile.rememberPassword) ...[
                      const SizedBox(width: 6),
                      const Icon(Icons.lock_clock,
                          size: 13, color: AppColors.textMuted),
                    ],
                  ]),
                  const SizedBox(height: 2),
                  Text(
                    profile.url,
                    style: const TextStyle(
                      fontSize: 11,
                      color: AppColors.textMuted,
                      fontFamily: 'monospace',
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            PopupMenuButton<String>(
              icon: const Icon(Icons.more_vert, size: 18,
                  color: AppColors.textMuted),
              onSelected: (v) {
                switch (v) {
                  case 'edit':
                    onEdit();
                    break;
                  case 'delete':
                    onDelete();
                    break;
                }
              },
              itemBuilder: (_) => [
                PopupMenuItem(
                    value: 'edit',
                    child: Row(children: [
                      const Icon(Icons.edit, size: 16),
                      const SizedBox(width: 8),
                      Text(context.tr('Edit')),
                    ])),
                PopupMenuItem(
                    value: 'delete',
                    child: Row(children: [
                      const Icon(Icons.delete_outline,
                          size: 16, color: AppColors.error),
                      const SizedBox(width: 8),
                      Text(context.tr('Delete'),
                          style: const TextStyle(color: AppColors.error)),
                    ])),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// Add / Edit dialog for a [ServerProfile]. One form covers both flows
/// — pass [existing] for edit, null for new. On save, writes to
/// [ServerConfig]; when the edited profile is the active one, also
/// re-probes auth so the router picks up any URL change without
/// forcing a manual logout.
class ProfileEditorDialog extends StatefulWidget {
  final ServerProfile? existing;
  const ProfileEditorDialog({super.key, this.existing});

  @override
  State<ProfileEditorDialog> createState() => _ProfileEditorDialogState();
}

class _ProfileEditorDialogState extends State<ProfileEditorDialog> {
  late final TextEditingController _aliasCtrl;
  late final TextEditingController _urlCtrl;
  late final TextEditingController _userCtrl;
  late final TextEditingController _passCtrl;
  bool _remember = false;
  bool _obscurePass = true;
  bool _testing = false;
  bool _saving = false;
  String? _testResult;
  String? _error;
  /// Snapshot of whatever password we pre-loaded from the keychain at
  /// open time. Compared against the live text field at save so the
  /// "user didn't retype, leave the stored blob alone" short-circuit
  /// doesn't treat an accidental edit as a no-op.
  String _initialPassword = '';

  @override
  void initState() {
    super.initState();
    final e = widget.existing;
    _aliasCtrl = TextEditingController(text: e?.alias ?? '');
    _urlCtrl = TextEditingController(text: e?.url ?? '');
    _userCtrl = TextEditingController(text: e?.username ?? '');
    _passCtrl = TextEditingController();
    _remember = e?.rememberPassword ?? false;

    // Edit flow: pre-fill the password from the keychain if one was
    // saved, so the user sees what's on file and can tweak it rather
    // than blowing it away every time they open the dialog.
    if (e != null && _remember) {
      _loadExistingPassword(e.id);
    }
  }

  Future<void> _loadExistingPassword(String id) async {
    final cfg = context.read<ServerConfig>();
    final pwd = await cfg.credentialStore.readPassword(id);
    if (!mounted) return;
    if (pwd != null && pwd.isNotEmpty) {
      setState(() {
        _passCtrl.text = pwd;
        _initialPassword = pwd;
      });
    }
  }

  @override
  void dispose() {
    _aliasCtrl.dispose();
    _urlCtrl.dispose();
    _userCtrl.dispose();
    _passCtrl.dispose();
    super.dispose();
  }

  Future<void> _test() async {
    final url = ServerProfile.normalizeUrl(_urlCtrl.text);
    if (url.isEmpty) {
      setState(() => _testResult = '❌ ${context.tr('URL is required')}');
      return;
    }
    setState(() { _testing = true; _testResult = null; });
    try {
      final api = ApiClient(baseUrl: url);
      final health = await api.health();
      if (!mounted) return;
      setState(() {
        _testResult =
            '✅ ${context.tr('Connected')} — ${health['sessions'] ?? 0} sessions, ${health['plugins'] ?? 0} plugins';
        _testing = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() { _testResult = '❌ $e'; _testing = false; });
    }
  }

  Future<void> _save() async {
    final alias = _aliasCtrl.text.trim();
    final url = ServerProfile.normalizeUrl(_urlCtrl.text);
    if (alias.isEmpty) {
      setState(() => _error = context.tr('Alias is required'));
      return;
    }
    if (url.isEmpty) {
      setState(() => _error = context.tr('URL is required'));
      return;
    }
    setState(() { _saving = true; _error = null; });

    final cfg = context.read<ServerConfig>();
    final auth = context.read<AuthService>();
    final canStorePassword = cfg.credentialStore.isSupported;
    final remember = _remember && canStorePassword;
    final password = _passCtrl.text;
    // Edit flow: if "remember" is on and the field text equals what
    // we pre-loaded, skip rewriting the keychain. Otherwise push the
    // current text down (including empty, which is handled by
    // ServerConfig — it treats empty + remember=true as "no change").
    final passwordForSave = (widget.existing != null &&
            remember &&
            password == _initialPassword)
        ? null
        : password;

    try {
      final existing = widget.existing;
      if (existing == null) {
        await cfg.addProfile(
          alias: alias,
          url: url,
          username: _userCtrl.text,
          rememberPassword: remember,
          password: remember ? password : null,
          makeActive: true,
        );
      } else {
        final urlChanged = existing.url != url;
        await cfg.updateProfile(
          existing.id,
          alias: alias,
          url: url,
          username: _userCtrl.text,
          rememberPassword: remember,
          password: remember ? passwordForSave : null,
        );
        // If we just edited the active profile's URL, re-probe auth
        // so the /sessions page doesn't keep hitting the old server.
        if (existing.id == cfg.activeId && urlChanged) {
          unawaited(auth.probe(url));
        }
      }
      if (!mounted) return;
      Navigator.pop(context);
    } catch (e) {
      if (!mounted) return;
      setState(() { _saving = false; _error = e.toString(); });
    }
  }

  @override
  Widget build(BuildContext context) {
    final cfg = context.watch<ServerConfig>();
    final canStorePassword = cfg.credentialStore.isSupported;
    final isEdit = widget.existing != null;

    return AlertDialog(
      title: Text(isEdit ? context.tr('Edit server') : context.tr('Add server')),
      contentPadding: const EdgeInsets.fromLTRB(20, 16, 20, 8),
      content: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 420),
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _aliasCtrl,
                autofocus: !isEdit,
                enabled: !_saving,
                decoration: InputDecoration(
                  labelText: context.tr('Alias'),
                  hintText: context.tr('e.g. Home NAS, Office'),
                  prefixIcon: const Icon(Icons.label_outline, size: 18),
                ),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: _urlCtrl,
                enabled: !_saving,
                keyboardType: TextInputType.url,
                autocorrect: false,
                textCapitalization: TextCapitalization.none,
                decoration: InputDecoration(
                  labelText: context.tr('Server URL'),
                  hintText: 'https://opendray.example.com',
                  prefixIcon: const Icon(Icons.link, size: 18),
                ),
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: (_testing || _saving) ? null : _test,
                      icon: _testing
                          ? const SizedBox(
                              width: 14,
                              height: 14,
                              child: CircularProgressIndicator(
                                  strokeWidth: 2, color: AppColors.accent),
                            )
                          : const Icon(Icons.wifi_tethering, size: 16),
                      label: Text(context.tr('Test')),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: AppColors.accent,
                        side: const BorderSide(color: AppColors.border),
                        padding: const EdgeInsets.symmetric(vertical: 10),
                      ),
                    ),
                  ),
                ],
              ),
              if (_testResult != null) ...[
                const SizedBox(height: 8),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: _testResult!.startsWith('✅')
                        ? AppColors.successSoft
                        : AppColors.errorSoft,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text(
                    _testResult!,
                    style: TextStyle(
                      fontSize: 12,
                      color: _testResult!.startsWith('✅')
                          ? AppColors.success
                          : AppColors.error,
                    ),
                  ),
                ),
              ],
              const SizedBox(height: 10),
              TextField(
                controller: _userCtrl,
                enabled: !_saving,
                autocorrect: false,
                enableSuggestions: false,
                textCapitalization: TextCapitalization.none,
                decoration: InputDecoration(
                  labelText: context.tr('Username (optional)'),
                  prefixIcon: const Icon(Icons.person_outline, size: 18),
                ),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: _passCtrl,
                enabled: !_saving,
                obscureText: _obscurePass,
                autocorrect: false,
                enableSuggestions: false,
                textCapitalization: TextCapitalization.none,
                decoration: InputDecoration(
                  labelText: context.tr('Password (optional)'),
                  prefixIcon: const Icon(Icons.key_outlined, size: 18),
                  suffixIcon: IconButton(
                    icon: Icon(
                      _obscurePass
                          ? Icons.visibility_off_outlined
                          : Icons.visibility_outlined,
                      size: 18,
                    ),
                    onPressed: () =>
                        setState(() => _obscurePass = !_obscurePass),
                  ),
                ),
              ),
              const SizedBox(height: 6),
              Row(
                children: [
                  Checkbox(
                    value: canStorePassword && _remember,
                    onChanged: canStorePassword && !_saving
                        ? (v) => setState(() => _remember = v ?? false)
                        : null,
                  ),
                  Expanded(
                    child: Text(
                      canStorePassword
                          ? context.tr(
                              'Remember password on this device (encrypted in OS keychain)')
                          : context.tr(
                              'Remember password is unavailable on web — password is never saved.'),
                      style: TextStyle(
                        fontSize: 12,
                        color: canStorePassword
                            ? AppColors.textMuted
                            : AppColors.textMuted.withValues(alpha: 0.7),
                      ),
                    ),
                  ),
                ],
              ),
              if (_error != null) ...[
                const SizedBox(height: 8),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: AppColors.errorSoft,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text(
                    _error!,
                    style: const TextStyle(
                        fontSize: 12, color: AppColors.error),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: _saving ? null : () => Navigator.pop(context),
          child: Text(context.tr('Cancel')),
        ),
        FilledButton.icon(
          onPressed: _saving ? null : _save,
          style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
          icon: _saving
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Colors.white),
                )
              : const Icon(Icons.save, size: 16),
          label: Text(isEdit ? context.tr('Save') : context.tr('Add')),
        ),
      ],
    );
  }
}

