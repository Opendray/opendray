import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Claude Multi-Account management page.
///
/// Lists claude_accounts rows. Each maps to an on-disk `~/.claude-accounts/<name>/`
/// sandbox managed by the `claude-acc` host tool. Tokens never flow through
/// Flutter — they're written server-side with chmod 600 and read only at
/// session-spawn time.
///
/// Renders in two modes:
///   - default (standalone page) — wraps the content in a Scaffold + FAB so
///     the /settings/claude-accounts deep-link route still works for users
///     arriving from older bookmarks.
///   - inline=true (M5 A3.2) — returns just the intro/list/add-button
///     column so the Settings → Plugins → Claude card can embed account
///     management directly. No Scaffold, no FAB; the "Add" action lives in
///     a normal button at the top of the column.
class ClaudeAccountsPage extends StatefulWidget {
  const ClaudeAccountsPage({super.key, this.inline = false});
  final bool inline;
  @override
  State<ClaudeAccountsPage> createState() => _ClaudeAccountsPageState();
}

class _ClaudeAccountsPageState extends State<ClaudeAccountsPage> {
  List<Map<String, dynamic>> _accounts = [];
  bool _loading = true;
  String? _error;
  bool _importing = false;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    try {
      final accounts = await _api.claudeAccounts();
      if (!mounted) return;
      setState(() {
        _accounts = accounts;
        _error = null;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _importLocal() async {
    setState(() => _importing = true);
    try {
      final summary = await ApiClient.describeErrors(_api.claudeAccountImportLocal);
      final imported = (summary['imported'] as List?)?.length ?? 0;
      final skipped = (summary['skipped'] as List?)?.length ?? 0;
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(imported == 0 && skipped == 0
            ? context.trOnce('No ~/.claude-accounts/tokens — run `claude-acc init` on the host first')
            : '${context.trOnce('Imported')} $imported · ${context.trOnce('Skipped')} $skipped'),
        backgroundColor: imported > 0 ? AppColors.success : AppColors.surface,
      ));
      _refresh();
      ProvidersBus.instance.notify();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    } finally {
      if (mounted) setState(() => _importing = false);
    }
  }

  Future<void> _openEditor([Map<String, dynamic>? account]) async {
    final changed = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      builder: (_) => _AccountEditorDialog(api: _api, initial: account),
    );
    if (changed == true) {
      _refresh();
      ProvidersBus.instance.notify();
    }
  }

  /// Opens the in-app OAuth modal that wraps `claude auth login --claudeai`.
  /// On success, the new account row appears in the list automatically.
  /// Falls back gracefully when the host doesn't have the Claude CLI
  /// installed (modal shows install instructions instead of stalling).
  Future<void> _signInWithClaude() async {
    final added = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      barrierDismissible: false, // user must click Cancel — auto-cleanup runs there
      builder: (_) => _OAuthDialog(api: _api),
    );
    if (added == true) {
      _refresh();
      ProvidersBus.instance.notify();
    }
  }

  Future<void> _setToken(Map<String, dynamic> a) async {
    final token = await showDialog<String>(
      context: context,
      useRootNavigator: true,
      builder: (_) => _TokenDialog(
        accountName: a['name'] as String? ?? '',
        tokenFilled: a['tokenFilled'] as bool? ?? false,
      ),
    );
    if (token == null || token.isEmpty) return;
    try {
      await ApiClient.describeErrors(
          () => _api.claudeAccountSetToken(a['id'] as String, token));
      _refresh();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  Future<void> _toggle(Map<String, dynamic> a, bool enabled) async {
    try {
      await _api.claudeAccountToggle(a['id'] as String, enabled);
      setState(() => a['enabled'] = enabled);
      ProvidersBus.instance.notify();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
      }
    }
  }

  Future<void> _delete(Map<String, dynamic> a) async {
    final ok = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      builder: (dialogCtx) => AlertDialog(
        title: Text(dialogCtx.tr('Delete Claude account?')),
        content: Text(
          dialogCtx.tr('This removes "@name" from OpenDray. The on-disk token file and config directory are left intact.')
              .replaceAll('@name', a['name'] as String? ?? ''),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx, rootNavigator: true).pop(false),
            child: Text(dialogCtx.tr('Cancel')),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.of(dialogCtx, rootNavigator: true).pop(true),
            child: Text(dialogCtx.tr('Delete')),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.claudeAccountDelete(a['id'] as String);
      _refresh();
      ProvidersBus.instance.notify();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_error != null) {
      return _emptyState(
        icon: Icons.error_outline,
        title: context.tr('Failed to load Claude accounts'),
        body: _error!,
      );
    }
    if (widget.inline) {
      return _buildInlineBody();
    }
    return Scaffold(
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _openEditor(),
        backgroundColor: AppColors.accent,
        icon: const Icon(Icons.add),
        label: Text(context.tr('Add Claude account')),
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 96),
          children: [
            _intro(),
            const SizedBox(height: 12),
            _signInButton(),
            const SizedBox(height: 8),
            _importButton(),
            const SizedBox(height: 12),
            if (_accounts.isEmpty)
              _emptyState(
                icon: Icons.person_outline,
                title: context.tr('No Claude accounts yet'),
                body: context.tr('Click "Sign in with Claude" above to add an account using your Claude subscription — no terminal, no file paths. Or use the "+" button below for manual setup.'),
              )
            else
              ..._accounts.map((a) => Padding(
                    padding: const EdgeInsets.only(bottom: 10),
                    child: _accountCard(a),
                  )),
          ],
        ),
      ),
    );
  }

  /// Inline rendering for the Settings → Plugins → Claude card (A3.2).
  ///
  /// Drops the Scaffold + FAB that the standalone route uses and replaces
  /// them with a normal "Add account" button at the top of the column.
  /// Intro + import + list are identical to the standalone render so the
  /// two views stay behaviourally in sync.
  Widget _buildInlineBody() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _intro(),
        const SizedBox(height: 8),
        _signInButton(),
        const SizedBox(height: 8),
        Row(
          children: [
            Expanded(child: _importButton()),
            const SizedBox(width: 8),
            OutlinedButton.icon(
              onPressed: () => _openEditor(),
              style: OutlinedButton.styleFrom(
                foregroundColor: AppColors.textMuted,
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              ),
              icon: const Icon(Icons.tune, size: 16),
              label: Text(context.tr('Manual setup'),
                  style: const TextStyle(fontSize: 12)),
            ),
          ],
        ),
        const SizedBox(height: 12),
        if (_accounts.isEmpty)
          _emptyState(
            icon: Icons.person_outline,
            title: context.tr('No Claude accounts yet'),
            body: context.tr('Register a Claude subscription token so this server can launch sessions as that account. Tokens stay chmod 600 on the host — this UI only tracks metadata.'),
          )
        else
          ..._accounts.map((a) => Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: _accountCard(a),
              )),
      ],
    );
  }

  Widget _intro() {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.accentSoft,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.accent.withValues(alpha: 0.35)),
      ),
      child: Row(children: [
        const Icon(Icons.info_outline, size: 18, color: AppColors.accent),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            context.tr('Sign in with your Claude subscription to add an account — no terminal, no file paths. Tokens stay on the host (chmod 600) and never enter Postgres or this UI.'),
            style: const TextStyle(fontSize: 11, color: AppColors.text),
          ),
        ),
      ]),
    );
  }

  /// Primary CTA — opens the in-app OAuth modal. The widget below the
  /// label uses the official Anthropic-orange accent so it stands out
  /// from the secondary "Import" / "Manual setup" buttons.
  Widget _signInButton() {
    return FilledButton.icon(
      onPressed: () => _signInWithClaude(),
      style: FilledButton.styleFrom(
        backgroundColor: const Color(0xFFCC785C), // Anthropic accent
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
      icon: const Icon(Icons.login, size: 18),
      label: Text(
        context.tr('Sign in with Claude'),
        style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
      ),
    );
  }

  Widget _importButton() {
    return OutlinedButton.icon(
      onPressed: _importing ? null : _importLocal,
      icon: _importing
          ? const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2, color: AppColors.accent))
          : const Icon(Icons.download, size: 16),
      label: Text(context.tr('Import from ~/.claude-accounts')),
      style: OutlinedButton.styleFrom(
        foregroundColor: AppColors.accent,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      ),
    );
  }

  Widget _emptyState({required IconData icon, required String title, required String body}) {
    return Padding(
      padding: const EdgeInsets.all(28),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: 48, color: AppColors.textMuted),
          const SizedBox(height: 16),
          Text(title, style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 15)),
          const SizedBox(height: 8),
          Text(body, style: const TextStyle(color: AppColors.textMuted, fontSize: 12), textAlign: TextAlign.center),
        ],
      ),
    );
  }

  Widget _accountCard(Map<String, dynamic> a) {
    final enabled = a['enabled'] as bool? ?? true;
    final tokenFilled = a['tokenFilled'] as bool? ?? false;
    final display = (a['displayName'] as String?)?.isNotEmpty == true
        ? a['displayName'] as String
        : a['name'] as String? ?? '';

    return Card(
      color: AppColors.surface,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(
          color: enabled ? AppColors.accent.withValues(alpha: 0.25) : AppColors.border,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 12, 8, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      Flexible(
                        child: Text(
                          display,
                          style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 8),
                      _statusChip(tokenFilled),
                    ]),
                    const SizedBox(height: 2),
                    Text('claude-${a['name']}',
                        style: const TextStyle(
                            fontFamily: 'monospace', fontSize: 11, color: AppColors.textMuted)),
                    if ((a['description'] as String?)?.isNotEmpty == true) ...[
                      const SizedBox(height: 4),
                      Text(a['description'] as String,
                          style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
                    ],
                  ],
                ),
              ),
              Switch(
                value: enabled,
                onChanged: (v) => _toggle(a, v),
                activeThumbColor: AppColors.accent,
              ),
            ]),
            const SizedBox(height: 8),
            _kv(context.tr('Config dir'), a['configDir'] as String? ?? ''),
            _kv(context.tr('Token path'), a['tokenPath'] as String? ?? ''),
            const SizedBox(height: 4),
            Row(mainAxisAlignment: MainAxisAlignment.end, children: [
              TextButton.icon(
                onPressed: () => _setToken(a),
                icon: const Icon(Icons.vpn_key, size: 16),
                label: Text(tokenFilled
                    ? context.tr('Rotate token')
                    : context.tr('Set token')),
              ),
              TextButton.icon(
                onPressed: () => _openEditor(a),
                icon: const Icon(Icons.edit, size: 16),
                label: Text(context.tr('Edit')),
              ),
              TextButton.icon(
                onPressed: () => _delete(a),
                icon: const Icon(Icons.delete_outline, size: 16, color: Colors.red),
                label: Text(context.tr('Delete'), style: const TextStyle(color: Colors.red)),
              ),
            ]),
          ],
        ),
      ),
    );
  }

  Widget _statusChip(bool filled) {
    final color = filled ? AppColors.success : AppColors.warning;
    final bg = filled ? AppColors.successSoft : AppColors.warningSoft;
    final label = filled
        ? context.tr('token set')
        : context.tr('no token');
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(color: bg, borderRadius: BorderRadius.circular(4)),
      child: Text(label,
          style: TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: color)),
    );
  }

  Widget _kv(String k, String v) {
    return Padding(
      padding: const EdgeInsets.only(top: 2),
      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        SizedBox(
          width: 92,
          child: Text(k,
              style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
        ),
        Expanded(
          child: Text(v,
              style: const TextStyle(
                  fontSize: 11, fontFamily: 'monospace', color: AppColors.text),
              overflow: TextOverflow.ellipsis),
        ),
      ]),
    );
  }
}

// ═════════════════════════════════════════════════════════════════
// Editor dialog
// ═════════════════════════════════════════════════════════════════

class _AccountEditorDialog extends StatefulWidget {
  final ApiClient api;
  final Map<String, dynamic>? initial;
  const _AccountEditorDialog({required this.api, required this.initial});

  @override
  State<_AccountEditorDialog> createState() => _AccountEditorDialogState();
}

class _AccountEditorDialogState extends State<_AccountEditorDialog> {
  late final TextEditingController _name;
  late final TextEditingController _displayName;
  late final TextEditingController _configDir;
  late final TextEditingController _tokenPath;
  late final TextEditingController _description;
  late final TextEditingController _token;
  late bool _enabled;

  @override
  void initState() {
    super.initState();
    final s = widget.initial ?? const <String, dynamic>{};
    _name        = TextEditingController(text: s['name']        as String? ?? '');
    _displayName = TextEditingController(text: s['displayName'] as String? ?? '');
    _configDir   = TextEditingController(text: s['configDir']   as String? ?? '');
    _tokenPath   = TextEditingController(text: s['tokenPath']   as String? ?? '');
    _description = TextEditingController(text: s['description'] as String? ?? '');
    _token       = TextEditingController();
    _enabled     = s['enabled'] as bool? ?? true;
  }

  @override
  void dispose() {
    _name.dispose();
    _displayName.dispose();
    _configDir.dispose();
    _tokenPath.dispose();
    _description.dispose();
    _token.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (widget.initial == null) {
      final name = _name.text.trim();
      if (!RegExp(r'^[a-z0-9_-]{1,32}$').hasMatch(name)) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text(context.tr('Name must be 1-32 chars of [a-z0-9_-]'))));
        return;
      }
    }
    try {
      if (widget.initial == null) {
        await ApiClient.describeErrors(() => widget.api.claudeAccountCreate(
              name: _name.text.trim(),
              displayName: _displayName.text.trim(),
              configDir: _configDir.text.trim(),
              tokenPath: _tokenPath.text.trim(),
              token: _token.text.trim(),
              description: _description.text.trim(),
              enabled: _enabled,
            ));
      } else {
        await ApiClient.describeErrors(() => widget.api.claudeAccountUpdate(
              widget.initial!['id'] as String,
              {
                'displayName': _displayName.text.trim(),
                'configDir': _configDir.text.trim(),
                'tokenPath': _tokenPath.text.trim(),
                'description': _description.text.trim(),
                'enabled': _enabled,
              },
            ));
        final newToken = _token.text.trim();
        if (newToken.isNotEmpty) {
          await ApiClient.describeErrors(() =>
              widget.api.claudeAccountSetToken(widget.initial!['id'] as String, newToken));
        }
      }
      if (!mounted) return;
      Navigator.of(context, rootNavigator: true).pop(true);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final isNew = widget.initial == null;
    return AlertDialog(
      title: Text(isNew
          ? context.tr('New Claude account')
          : context.tr('Edit Claude account')),
      contentPadding: const EdgeInsets.fromLTRB(20, 12, 20, 0),
      content: SizedBox(
        width: 520,
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              if (isNew) _field(context.tr('Name'), _name, hint: 'kev',
                  help: context.tr('Lowercase alphanumeric, dash or underscore. Matches claude-<name> shortcut.')),
              _field(context.tr('Display Name'), _displayName,
                  hint: 'Kev (main)'),
              _field(context.tr('Config dir'), _configDir,
                  hint: '~/.claude-accounts/<name>',
                  help: context.tr('Leave empty to use the claude-acc default.')),
              _field(context.tr('Token path'), _tokenPath,
                  hint: '~/.claude-accounts/tokens/<name>.token',
                  help: context.tr('Leave empty to use the claude-acc default.')),
              _field(context.tr('Description'), _description),
              _field(
                isNew ? context.tr('OAuth token (optional)')
                      : context.tr('OAuth token (leave empty to keep)'),
                _token,
                obscure: true,
                hint: 'sk-ant-oat01-...',
                help: context.tr('Generated by `claude setup-token`. Written chmod 600 at the token path.'),
              ),
              const SizedBox(height: 8),
              SwitchListTile(
                value: _enabled,
                onChanged: (v) => setState(() => _enabled = v),
                title: Text(context.tr('Enabled')),
                contentPadding: EdgeInsets.zero,
                activeThumbColor: AppColors.accent,
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context, rootNavigator: true).pop(false),
          child: Text(context.tr('Cancel')),
        ),
        FilledButton(
          onPressed: _save,
          style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
          child: Text(context.tr('Save')),
        ),
      ],
    );
  }

  Widget _field(String label, TextEditingController c,
      {String? hint, String? help, bool obscure = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextField(
            controller: c,
            obscureText: obscure,
            decoration: InputDecoration(labelText: label, hintText: hint),
          ),
          if (help != null) ...[
            const SizedBox(height: 4),
            Text(help,
                style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
          ],
        ],
      ),
    );
  }
}

// ═════════════════════════════════════════════════════════════════
// Token-only dialog (quick rotate from a card action)
// ═════════════════════════════════════════════════════════════════

class _TokenDialog extends StatefulWidget {
  final String accountName;
  final bool tokenFilled;
  const _TokenDialog({required this.accountName, required this.tokenFilled});
  @override
  State<_TokenDialog> createState() => _TokenDialogState();
}

class _TokenDialogState extends State<_TokenDialog> {
  final _c = TextEditingController();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(widget.tokenFilled
          ? context.tr('Rotate OAuth token')
          : context.tr('Set OAuth token')),
      content: Column(mainAxisSize: MainAxisSize.min, children: [
        Text(
          context.tr('Paste the token for @name. Generated by `claude setup-token` on the host.')
              .replaceAll('@name', widget.accountName),
          style: const TextStyle(fontSize: 12, color: AppColors.textMuted),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _c,
          obscureText: true,
          decoration: const InputDecoration(hintText: 'sk-ant-oat01-...'),
        ),
      ]),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context, rootNavigator: true).pop(),
          child: Text(context.tr('Cancel')),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context, rootNavigator: true)
              .pop(_c.text.trim()),
          style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
          child: Text(context.tr('Save')),
        ),
      ],
    );
  }
}

// ── Sign in with Claude — in-app OAuth modal ─────────────────────────
//
// State machine for this dialog:
//
//   _phase = preflighting → calls /oauth/preflight
//             ├─ CLI present → start
//             └─ CLI missing → showInstallHint  (terminal state)
//   _phase = starting     → calls /oauth/start
//             ├─ ok → showCode (with authURL + code input)
//             └─ err → showError  (terminal state)
//   _phase = showCode     → user copies URL, opens browser, returns
//             with auth code, types it in
//             └─ user submits → calls /oauth/complete
//   _phase = completing   → backend exchanges code for tokens
//             ├─ ok  → showSuccess (terminal — caller refreshes list)
//             └─ err → back to showCode with error banner
//
// Cancellation rule: any close path (Cancel button, system back, error
// terminal "Close") MUST call /oauth/cancel for the active flowId, so
// the server kills the PTY child + cleans the temp dir. We track this
// via _flowId — non-empty means a server-side flow is active.

enum _OAuthPhase {
  preflighting,
  showInstallHint,
  starting,
  showCode,
  completing,
  showSuccess,
  showError,
}

class _OAuthDialog extends StatefulWidget {
  const _OAuthDialog({required this.api});
  final ApiClient api;

  @override
  State<_OAuthDialog> createState() => _OAuthDialogState();
}

class _OAuthDialogState extends State<_OAuthDialog> {
  _OAuthPhase _phase = _OAuthPhase.preflighting;
  String _authURL = '';
  String _flowId = '';
  String _installHint = '';
  String _errorMessage = '';
  String _successName = '';
  String _successEmail = '';
  final _codeController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _runPreflight();
  }

  @override
  void dispose() {
    _codeController.dispose();
    super.dispose();
  }

  Future<void> _runPreflight() async {
    try {
      final res = await widget.api.claudeOAuthPreflight();
      if (!mounted) return;
      if (res['available'] == true) {
        await _runStart();
      } else {
        setState(() {
          _phase = _OAuthPhase.showInstallHint;
          _installHint =
              (res['installHint'] as String?) ?? 'Claude CLI not found on host.';
        });
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _phase = _OAuthPhase.showError;
        _errorMessage = 'Preflight failed: $e';
      });
    }
  }

  Future<void> _runStart() async {
    setState(() => _phase = _OAuthPhase.starting);
    try {
      final res = await widget.api.claudeOAuthStart();
      if (!mounted) return;
      setState(() {
        _phase = _OAuthPhase.showCode;
        _authURL = (res['authorizationUrl'] as String?) ?? '';
        _flowId = (res['flowId'] as String?) ?? '';
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _phase = _OAuthPhase.showError;
        _errorMessage = 'Could not start sign-in: $e';
      });
    }
  }

  Future<void> _runComplete() async {
    final code = _codeController.text.trim();
    if (code.isEmpty || _flowId.isEmpty) return;
    setState(() => _phase = _OAuthPhase.completing);
    try {
      final res = await widget.api.claudeOAuthComplete(_flowId, code);
      if (!mounted) return;
      _flowId = ''; // server-side flow is now closed (success path)
      setState(() {
        _phase = _OAuthPhase.showSuccess;
        _successName = (res['name'] as String?) ?? '';
        final profile = res['profile'] as Map?;
        _successEmail = (profile?['email'] as String?) ?? '';
      });
    } catch (e) {
      if (!mounted) return;
      // Stay on the code screen so the user can retry / paste again.
      setState(() {
        _phase = _OAuthPhase.showCode;
        _errorMessage = _firstLine('$e');
      });
    }
  }

  Future<void> _cancelFlowIfActive() async {
    if (_flowId.isEmpty) return;
    final id = _flowId;
    _flowId = '';
    try {
      await widget.api.claudeOAuthCancel(id);
    } catch (_) {
      // Best-effort: server cleans up via timeout sweeper if cancel fails.
    }
  }

  Future<void> _close({required bool added}) async {
    await _cancelFlowIfActive();
    if (!mounted) return;
    Navigator.of(context, rootNavigator: true).pop(added);
  }

  Future<void> _openInBrowser() async {
    if (_authURL.isEmpty) return;
    try {
      await launchUrl(Uri.parse(_authURL), mode: LaunchMode.externalApplication);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not open browser: $e')),
      );
    }
  }

  Future<void> _copyToClipboard(String value, String snackMessage) async {
    await Clipboard.setData(ClipboardData(text: value));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(snackMessage), duration: const Duration(seconds: 1)),
    );
  }

  @override
  Widget build(BuildContext context) {
    return WillPopScope(
      onWillPop: () async {
        await _cancelFlowIfActive();
        return true;
      },
      child: AlertDialog(
        title: Row(
          children: [
            const Icon(Icons.login, size: 18, color: Color(0xFFCC785C)),
            const SizedBox(width: 8),
            Text(context.tr('Sign in with Claude')),
          ],
        ),
        content: SizedBox(
          width: 480,
          child: _buildContent(),
        ),
        actions: _buildActions(),
      ),
    );
  }

  Widget _buildContent() {
    switch (_phase) {
      case _OAuthPhase.preflighting:
        return _busy(context.tr('Checking host setup…'));
      case _OAuthPhase.starting:
        return _busy(context.tr('Starting sign-in flow…'));
      case _OAuthPhase.completing:
        return _busy(context.tr('Verifying with Claude…'));
      case _OAuthPhase.showInstallHint:
        return _installHintView();
      case _OAuthPhase.showCode:
        return _codeView();
      case _OAuthPhase.showSuccess:
        return _successView();
      case _OAuthPhase.showError:
        return _errorView();
    }
  }

  Widget _busy(String label) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 24),
      child: Column(mainAxisSize: MainAxisSize.min, children: [
        const CircularProgressIndicator(color: Color(0xFFCC785C)),
        const SizedBox(height: 16),
        Text(label, style: const TextStyle(color: AppColors.textMuted)),
      ]),
    );
  }

  Widget _installHintView() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          context.tr('OpenDray needs the official Claude Code CLI on the server before it can sign you in.'),
          style: const TextStyle(fontSize: 13),
        ),
        const SizedBox(height: 12),
        Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: AppColors.surfaceAlt,
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: AppColors.border),
          ),
          child: SelectableText(
            _installHint,
            style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 11.5,
                color: AppColors.text),
          ),
        ),
        const SizedBox(height: 12),
        Text(
          context.tr('After installing, click "Try again" or use Manual setup as a fallback.'),
          style: const TextStyle(fontSize: 11.5, color: AppColors.textMuted),
        ),
      ],
    );
  }

  Widget _codeView() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          context.tr('1. Open the sign-in page on any device:'),
          style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 6),
        Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: AppColors.surfaceAlt,
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: AppColors.border),
          ),
          child: Row(
            children: [
              Expanded(
                child: SelectableText(
                  _truncateUrl(_authURL),
                  style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 10.5,
                      color: AppColors.text),
                  maxLines: 2,
                ),
              ),
              IconButton(
                icon: const Icon(Icons.copy, size: 16),
                tooltip: context.tr('Copy URL'),
                onPressed: () => _copyToClipboard(
                    _authURL, context.trOnce('URL copied to clipboard')),
              ),
              IconButton(
                icon: const Icon(Icons.open_in_new, size: 16),
                tooltip: context.tr('Open in browser'),
                onPressed: _openInBrowser,
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        Text(
          context.tr('2. After signing in, paste the code Anthropic shows you:'),
          style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 6),
        TextField(
          controller: _codeController,
          autofocus: true,
          decoration: InputDecoration(
            hintText: context.tr('Paste auth code here'),
            border: const OutlineInputBorder(),
            isDense: true,
          ),
          style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
          onSubmitted: (_) => _runComplete(),
        ),
        if (_errorMessage.isNotEmpty) ...[
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: AppColors.errorSoft,
              borderRadius: BorderRadius.circular(6),
              border: Border.all(color: AppColors.error.withValues(alpha: 0.4)),
            ),
            child: Row(children: [
              const Icon(Icons.error_outline, size: 16, color: AppColors.error),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  _errorMessage,
                  style: const TextStyle(fontSize: 11.5, color: AppColors.text),
                ),
              ),
            ]),
          ),
        ],
      ],
    );
  }

  Widget _successView() {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        const Icon(Icons.check_circle, size: 48, color: AppColors.success),
        const SizedBox(height: 12),
        Text(
          _successEmail.isNotEmpty
              ? context.trOnce('Signed in as') + ' $_successEmail'
              : context.tr('Signed in successfully'),
          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
        ),
        if (_successName.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(
            context.trOnce('Account name') + ': $_successName',
            style: const TextStyle(fontSize: 11.5, color: AppColors.textMuted),
          ),
        ],
      ],
    );
  }

  Widget _errorView() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        const Icon(Icons.error_outline, size: 36, color: AppColors.error),
        const SizedBox(height: 12),
        Text(
          _errorMessage,
          style: const TextStyle(fontSize: 13, color: AppColors.text),
        ),
      ],
    );
  }

  List<Widget> _buildActions() {
    switch (_phase) {
      case _OAuthPhase.preflighting:
      case _OAuthPhase.starting:
      case _OAuthPhase.completing:
        return [
          TextButton(
            onPressed: () => _close(added: false),
            child: Text(context.tr('Cancel')),
          ),
        ];
      case _OAuthPhase.showInstallHint:
        return [
          TextButton(
            onPressed: () => _close(added: false),
            child: Text(context.tr('Close')),
          ),
          FilledButton(
            onPressed: _runPreflight,
            style: FilledButton.styleFrom(
                backgroundColor: const Color(0xFFCC785C)),
            child: Text(context.tr('Try again')),
          ),
        ];
      case _OAuthPhase.showCode:
        return [
          TextButton(
            onPressed: () => _close(added: false),
            child: Text(context.tr('Cancel')),
          ),
          FilledButton(
            onPressed: _codeController.text.trim().isEmpty ? null : _runComplete,
            style: FilledButton.styleFrom(
                backgroundColor: const Color(0xFFCC785C)),
            child: Text(context.tr('Submit code')),
          ),
        ];
      case _OAuthPhase.showSuccess:
        return [
          FilledButton(
            onPressed: () => _close(added: true),
            style: FilledButton.styleFrom(
                backgroundColor: AppColors.success),
            child: Text(context.tr('Done')),
          ),
        ];
      case _OAuthPhase.showError:
        return [
          TextButton(
            onPressed: () => _close(added: false),
            child: Text(context.tr('Close')),
          ),
        ];
    }
  }
}

// _truncateUrl shortens a URL for display while keeping the start +
// query-recognisable middle. Only used in the OAuth modal where
// authorization URLs are ~500 chars and would push the dialog wide.
String _truncateUrl(String url, {int max = 110}) {
  if (url.length <= max) return url;
  final head = max ~/ 2 - 2;
  final tail = max - head - 1;
  return '${url.substring(0, head)}…${url.substring(url.length - tail)}';
}

String _firstLine(String s) {
  final i = s.indexOf('\n');
  if (i < 0) return s;
  return s.substring(0, i);
}
