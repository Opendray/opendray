import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../core/api/api_client.dart';
import '../../core/services/auth_service.dart';
import '../../core/services/l10n.dart';
import '../../core/services/server_config.dart';
import '../../shared/app_modals.dart';
import '../../shared/theme/app_theme.dart';
import 'servers_card.dart';

const String _kBuildDate =
    String.fromEnvironment('BUILD_DATE', defaultValue: '');
// UTC ISO8601-basic timestamp stamped by app/build_release.sh. Sits
// alongside BUILD_DATE so ops can tell two APKs/IPAs built on the same
// calendar day apart — the APK you just pushed to the phone vs the one
// that was there before. Empty when dev-run (`flutter run`) or when
// the build didn't set the --dart-define.
const String _kBuildTimestamp =
    String.fromEnvironment('BUILD_TIMESTAMP', defaultValue: '');
const String _kRepoUrl = 'https://github.com/Opendray/opendray';

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key});
  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  /// Backend identity pulled from /api/health — version + short git SHA.
  /// Both start null while the request is in flight, then settle to
  /// either a real value or "—" when the call fails or the server is
  /// too old to report them.
  String? _backendVersion;
  String? _backendSha;
  String? _backendBuildTime;

  @override
  void initState() {
    super.initState();
    _loadBackendInfo();
  }

  Future<void> _loadBackendInfo() async {
    try {
      final health = await context.read<ApiClient>().health();
      if (!mounted) return;
      setState(() {
        _backendVersion = (health['version'] as String?) ?? '—';
        final sha = (health['buildSha'] as String?) ?? '';
        // Short-SHA for display — full SHA is rarely useful in a UI.
        _backendSha = sha.isEmpty ? '—' : sha.substring(0, sha.length.clamp(0, 7));
        final bt = (health['buildTime'] as String?) ?? '';
        _backendBuildTime = bt.isEmpty ? '—' : bt;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _backendVersion = '—';
        _backendSha = '—';
        _backendBuildTime = '—';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    context.watch<ServerConfig>();

    return Scaffold(
      appBar: AppBar(title: Text(context.tr('Settings'))),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Language
          const _LanguageCard(),
          const SizedBox(height: 16),

          // Servers (multi-URL address book). Replaces the old single
          // "Server URL" card — users now manage the full list of
          // deployments plus per-entry saved credentials in one place.
          const ServersCard(),
          const SizedBox(height: 16),

          // Plugins management lives in the dedicated bottom-nav tabs:
          //   • /plugins — installed CRUD
          //   • /hub     — marketplace
          // Settings no longer carries a link card — redundant surface.

          // Claude Accounts lives on the Plugin tab now: Plugins → Claude
          // card's "Accounts" popup menu entry (or tap-to-open) routes to
          // /settings/claude-accounts, the same page this card used to
          // link to. The deep-link URL is preserved.

          // LLM Endpoints — the address book of OpenAI-compatible model
          // endpoints (Ollama / LM Studio / Groq / Gemini / custom). It
          // used to be a `llm-providers` panel plugin, but the data and
          // spawn-time injection are kernel-level — shared by any agent,
          // not owned by a single plugin — so it lives here in Settings
          // alongside Claude Accounts.
          Card(
            child: InkWell(
              onTap: () => context.push('/settings/llm-endpoints'),
              borderRadius: BorderRadius.circular(12),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(children: [
                  const Icon(Icons.satellite_alt_outlined,
                      color: AppColors.accent, size: 20),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(context.tr('LLM Endpoints'),
                            style: const TextStyle(
                                fontSize: 14, fontWeight: FontWeight.w600)),
                        const SizedBox(height: 2),
                        Text(
                          context.tr(
                              'Address book of OpenAI-compatible model endpoints (Ollama, LM Studio, Groq, Gemini, custom). Shared by every agent.'),
                          style: const TextStyle(
                              fontSize: 12, color: AppColors.textMuted),
                        ),
                      ],
                    ),
                  ),
                  const Icon(Icons.chevron_right,
                      color: AppColors.textMuted, size: 20),
                ]),
              ),
            ),
          ),
          const SizedBox(height: 12),

          // Built-in plugins — the "I uninstalled something bundled
          // with OpenDray and now I can't get it back" escape hatch.
          // The Plugins page lets users Uninstall any non-required
          // built-in, which tombstones the row so LoadAll stops
          // re-seeding. Hub is hidden during v1 (and won't ever list
          // built-ins anyway), so without this entry there's no
          // recovery path — hence a dedicated page under Settings.
          Card(
            child: InkWell(
              onTap: () => context.push('/settings/builtin-plugins'),
              borderRadius: BorderRadius.circular(12),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(children: [
                  const Icon(Icons.restore,
                      color: AppColors.accent, size: 20),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(context.tr('Built-in plugins'),
                            style: const TextStyle(
                                fontSize: 14, fontWeight: FontWeight.w600)),
                        const SizedBox(height: 2),
                        Text(
                          context.tr(
                              'Browse the plugins bundled with OpenDray and restore anything you previously uninstalled.'),
                          style: const TextStyle(
                              fontSize: 12, color: AppColors.textMuted),
                        ),
                      ],
                    ),
                  ),
                  const Icon(Icons.chevron_right,
                      color: AppColors.textMuted, size: 20),
                ]),
              ),
            ),
          ),
          const SizedBox(height: 16),

          // Account — only shown when the server has auth enabled and we
          // actually hold a token; otherwise there's nothing to sign out of.
          _buildAccountCard(context),

          // About
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Container(
                        width: 28, height: 28,
                        decoration: BoxDecoration(color: AppColors.accent, borderRadius: BorderRadius.circular(7)),
                        child: const Icon(Icons.terminal_rounded, color: Colors.white, size: 18),
                      ),
                      const SizedBox(width: 10),
                      const Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('OpenDray', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 15)),
                          Text('Terminal-Centric Development Cockpit', style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
                        ],
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  FutureBuilder<PackageInfo>(
                    future: PackageInfo.fromPlatform(),
                    builder: (context, snap) {
                      final info = snap.data;
                      // Compose the app version line as
                      // "1.0.0+30 · 20260420T193412Z" so two APKs from
                      // the same pubspec version still render distinct
                      // labels — ops uses this to spot a stale install.
                      final base = info == null
                          ? '—'
                          : '${info.version} (${info.buildNumber})';
                      final version = _kBuildTimestamp.isEmpty
                          ? base
                          : '$base · $_kBuildTimestamp';
                      final buildDate =
                          _kBuildDate.isEmpty ? '—' : _kBuildDate;
                      return Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          _InfoRow(
                              label: context.tr('App version'), value: version),
                          _InfoRow(
                              label: context.tr('Build date'),
                              value: buildDate),
                          _InfoRow(
                              label: context.tr('Backend version'),
                              value: _backendVersion ?? '…'),
                          _InfoRow(
                              label: context.tr('Backend build'),
                              value: _backendSha ?? '…'),
                          _InfoRow(
                              label: context.tr('Backend built at'),
                              value: _backendBuildTime ?? '…'),
                        ],
                      );
                    },
                  ),
                  const SizedBox(height: 4),
                  InkWell(
                    borderRadius: BorderRadius.circular(6),
                    onTap: () => launchUrl(Uri.parse(_kRepoUrl),
                        mode: LaunchMode.externalApplication),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(vertical: 6),
                      child: Row(
                        children: [
                          const Icon(Icons.code, size: 16, color: AppColors.accent),
                          const SizedBox(width: 8),
                          Text(context.tr('GitHub'),
                              style: const TextStyle(
                                  color: AppColors.accent,
                                  fontSize: 13,
                                  fontWeight: FontWeight.w500)),
                          const SizedBox(width: 6),
                          const Icon(Icons.open_in_new,
                              size: 13, color: AppColors.accent),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// Account card — Change credentials + Sign out. Shown whenever there's
  /// a stored token so the user always has an escape hatch, even if the
  /// server is currently unreachable or the auth state is unknown.
  /// "Change credentials" still requires a live server (it hits an API),
  /// so we disable it when we're not confirmed-authed.
  Widget _buildAccountCard(BuildContext context) {
    final auth = context.watch<AuthService>();
    if (!auth.hasStoredToken && auth.state != AuthState.authed) {
      return const SizedBox.shrink();
    }
    final canChange = auth.state == AuthState.authed;

    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Card(
        child: Column(children: [
          ListTile(
            enabled: canChange,
            leading: Icon(Icons.lock_reset,
                color: canChange ? AppColors.accent : AppColors.textMuted),
            title: Text(context.tr('Change credentials'),
                style: const TextStyle(
                    fontSize: 14, fontWeight: FontWeight.w500)),
            subtitle: Text(
              canChange
                  ? context.tr('Update the admin username and password')
                  : context.tr('Reconnect to the server to change credentials'),
              style: const TextStyle(
                  fontSize: 11, color: AppColors.textMuted),
            ),
            trailing: const Icon(Icons.chevron_right,
                size: 20, color: AppColors.textMuted),
            onTap: canChange ? () => _showChangeCredentialsSheet(context) : null,
          ),
          const Divider(height: 1, indent: 16),
          ListTile(
            leading: const Icon(Icons.logout, color: AppColors.error),
            title: Text(context.tr('Sign out'),
                style: const TextStyle(
                    fontSize: 14, fontWeight: FontWeight.w500)),
            subtitle: Text(
              context.tr('Clear the saved token on this device'),
              style: const TextStyle(fontSize: 11, color: AppColors.textMuted),
            ),
            trailing: const Icon(Icons.chevron_right,
                size: 20, color: AppColors.textMuted),
            onTap: () async {
              await context.read<AuthService>().logout();
              // Router's redirect handles the actual navigation to /login.
            },
          ),
        ]),
      ),
    );
  }

  Future<void> _showChangeCredentialsSheet(BuildContext context) async {
    await showAppModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
      builder: (_) => const _ChangeCredentialsSheet(),
    );
  }
}

/// Bottom-sheet form for rotating the admin credentials. Keyboard-aware,
/// keeps the submit button above the IME, mirrors the NewFolder sheet's
/// layout conventions so the UX feels consistent.
class _ChangeCredentialsSheet extends StatefulWidget {
  const _ChangeCredentialsSheet();

  @override
  State<_ChangeCredentialsSheet> createState() =>
      _ChangeCredentialsSheetState();
}

class _ChangeCredentialsSheetState extends State<_ChangeCredentialsSheet> {
  final _currentCtrl = TextEditingController();
  final _newUserCtrl = TextEditingController();
  final _newPassCtrl = TextEditingController();
  final _confirmCtrl = TextEditingController();
  bool _obscureCurrent = true;
  bool _obscureNew = true;
  bool _submitting = false;
  String? _error;

  @override
  void dispose() {
    _currentCtrl.dispose();
    _newUserCtrl.dispose();
    _newPassCtrl.dispose();
    _confirmCtrl.dispose();
    super.dispose();
  }

  String? _validate() {
    if (_currentCtrl.text.isEmpty) return 'Enter your current password';
    if (_newPassCtrl.text.length < 8) {
      return 'New password must be at least 8 characters';
    }
    if (_newPassCtrl.text != _confirmCtrl.text) {
      return 'New passwords do not match';
    }
    return null;
  }

  Future<void> _submit() async {
    final v = _validate();
    if (v != null) { setState(() => _error = v); return; }
    setState(() { _submitting = true; _error = null; });
    // Pull Providers BEFORE the first await so we don't touch `context`
    // after an async gap (lint + real hazard if widget unmounts mid-request).
    final api = context.read<ApiClient>();
    final auth = context.read<AuthService>();
    try {
      final res = await ApiClient.describeErrors(() => api.changeCredentials(
            currentPassword: _currentCtrl.text,
            newUsername: _newUserCtrl.text.trim(),
            newPassword: _newPassCtrl.text,
          ));
      final newToken = res['token'] as String? ?? '';
      if (newToken.isNotEmpty) {
        await auth.acceptNewToken(newToken);
      }
      if (!mounted) return;
      Navigator.pop(context);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Credentials updated'),
          duration: Duration(seconds: 2),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      setState(() { _submitting = false; _error = e.toString(); });
    }
  }

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;
    return Padding(
      padding: EdgeInsets.only(bottom: bottomInset),
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 14),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Center(
                child: Container(
                  width: 36, height: 4,
                  margin: const EdgeInsets.only(bottom: 10),
                  decoration: BoxDecoration(
                    color: AppColors.border,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              Row(children: [
                const Icon(Icons.lock_reset, size: 18, color: AppColors.accent),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(context.tr('Change credentials'),
                      style: const TextStyle(
                          fontSize: 15, fontWeight: FontWeight.w600)),
                ),
                IconButton(
                  icon: const Icon(Icons.close, size: 18),
                  onPressed: () => Navigator.pop(context),
                  padding: EdgeInsets.zero,
                  constraints:
                      const BoxConstraints(minWidth: 32, minHeight: 32),
                ),
              ]),
              const SizedBox(height: 14),
              _field(
                controller: _currentCtrl,
                label: context.tr('Current password'),
                obscure: _obscureCurrent,
                onToggle: () =>
                    setState(() => _obscureCurrent = !_obscureCurrent),
                hint: '••••••••',
              ),
              const SizedBox(height: 10),
              TextField(
                controller: _newUserCtrl,
                autocorrect: false,
                enableSuggestions: false,
                textCapitalization: TextCapitalization.none,
                decoration: InputDecoration(
                  labelText: context.tr('New username (leave blank to keep)'),
                  isDense: true,
                  filled: true,
                  fillColor: AppColors.surfaceAlt,
                  border: const OutlineInputBorder(
                    borderRadius: BorderRadius.all(Radius.circular(10)),
                    borderSide: BorderSide.none,
                  ),
                ),
              ),
              const SizedBox(height: 10),
              _field(
                controller: _newPassCtrl,
                label: context.tr('New password'),
                obscure: _obscureNew,
                onToggle: () => setState(() => _obscureNew = !_obscureNew),
                hint: context.tr('at least 8 characters'),
              ),
              const SizedBox(height: 10),
              _field(
                controller: _confirmCtrl,
                label: context.tr('Confirm new password'),
                obscure: _obscureNew,
                onToggle: () => setState(() => _obscureNew = !_obscureNew),
                hint: '',
                onSubmitted: (_) => _submit(),
              ),
              if (_error != null) ...[
                const SizedBox(height: 10),
                Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: AppColors.errorSoft,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Row(children: [
                    const Icon(Icons.error_outline,
                        size: 16, color: AppColors.error),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(_error!,
                          style: const TextStyle(
                              color: AppColors.error, fontSize: 12)),
                    ),
                  ]),
                ),
              ],
              const SizedBox(height: 14),
              Row(children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: _submitting
                        ? null
                        : () => Navigator.pop(context),
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 14),
                    ),
                    child: Text(context.tr('Cancel')),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  flex: 2,
                  child: FilledButton.icon(
                    onPressed: _submitting ? null : _submit,
                    style: FilledButton.styleFrom(
                      backgroundColor: AppColors.accent,
                      padding: const EdgeInsets.symmetric(vertical: 14),
                    ),
                    icon: _submitting
                        ? const SizedBox(
                            width: 14,
                            height: 14,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : const Icon(Icons.check, size: 16),
                    label: Text(context.tr('Update')),
                  ),
                ),
              ]),
            ],
          ),
        ),
      ),
    );
  }

  Widget _field({
    required TextEditingController controller,
    required String label,
    required bool obscure,
    required VoidCallback onToggle,
    required String hint,
    void Function(String)? onSubmitted,
  }) {
    return TextField(
      controller: controller,
      obscureText: obscure,
      autocorrect: false,
      enableSuggestions: false,
      textCapitalization: TextCapitalization.none,
      onSubmitted: onSubmitted,
      decoration: InputDecoration(
        labelText: label,
        hintText: hint,
        isDense: true,
        filled: true,
        fillColor: AppColors.surfaceAlt,
        border: const OutlineInputBorder(
          borderRadius: BorderRadius.all(Radius.circular(10)),
          borderSide: BorderSide.none,
        ),
        suffixIcon: IconButton(
          icon: Icon(
            obscure ? Icons.visibility_off_outlined : Icons.visibility_outlined,
            size: 18,
          ),
          onPressed: onToggle,
        ),
      ),
    );
  }
}

/// Language picker — switches the app-wide translation catalog via the
/// L10n provider. Takes effect immediately, persisted to SharedPreferences.
class _LanguageCard extends StatelessWidget {
  const _LanguageCard();

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<L10n>();
    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 16, 10),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            const Icon(Icons.language, color: AppColors.accent, size: 20),
            const SizedBox(width: 10),
            Text(l10n.t('Language'),
                style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15)),
          ]),
          const SizedBox(height: 10),
          Wrap(spacing: 8, runSpacing: 8,
            children: L10n.languages.map((lang) {
              final selected = lang.code == l10n.code;
              return GestureDetector(
                onTap: () => l10n.setLanguage(lang.code),
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                  decoration: BoxDecoration(
                    color: selected ? AppColors.accentSoft : AppColors.surfaceAlt,
                    border: Border.all(
                        color: selected ? AppColors.accent : AppColors.border),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Row(mainAxisSize: MainAxisSize.min, children: [
                    Container(
                      width: 22, height: 22,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: selected ? AppColors.accent : AppColors.surface,
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(lang.flag,
                          style: TextStyle(
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              color: selected ? Colors.white : AppColors.textMuted)),
                    ),
                    const SizedBox(width: 8),
                    Text(lang.name,
                        style: TextStyle(
                          fontSize: 13,
                          color: selected ? AppColors.accent : AppColors.text,
                          fontWeight: selected ? FontWeight.w600 : FontWeight.normal,
                        )),
                    if (selected) ...[
                      const SizedBox(width: 6),
                      const Icon(Icons.check, size: 14, color: AppColors.accent),
                    ],
                  ]),
                ),
              );
            }).toList(),
          ),
        ]),
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  final String label;
  final String value;
  const _InfoRow({required this.label, required this.value});
  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: [
          SizedBox(width: 100, child: Text(label, style: const TextStyle(color: AppColors.textMuted, fontSize: 12))),
          Expanded(child: Text(value, style: const TextStyle(fontSize: 12))),
        ],
      ),
    );
  }
}
