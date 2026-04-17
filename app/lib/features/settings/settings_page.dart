import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../core/services/server_config.dart';
import '../../shared/theme/app_theme.dart';
import 'widgets/plugins_section.dart';

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key});
  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  late TextEditingController _urlController;
  late TextEditingController _cfIdController;
  late TextEditingController _cfSecretController;
  String? _testResult;
  bool _testing = false;

  @override
  void initState() {
    super.initState();
    final config = context.read<ServerConfig>();
    _urlController = TextEditingController(text: config.url);
    _cfIdController = TextEditingController(text: config.cfAccessId);
    _cfSecretController = TextEditingController(text: config.cfAccessSecret);
  }

  @override
  void dispose() {
    _urlController.dispose();
    _cfIdController.dispose();
    _cfSecretController.dispose();
    super.dispose();
  }

  Map<String, String> _pendingCfHeaders() {
    final id = _cfIdController.text.trim();
    final secret = _cfSecretController.text.trim();
    if (id.isEmpty || secret.isEmpty) return const {};
    return {'CF-Access-Client-Id': id, 'CF-Access-Client-Secret': secret};
  }

  Future<void> _testConnection() async {
    setState(() { _testing = true; _testResult = null; });
    try {
      final api = ApiClient(
        baseUrl: _urlController.text.trim(),
        extraHeaders: _pendingCfHeaders(),
      );
      final health = await api.health();
      setState(() {
        _testResult = '✅ Connected — ${health['sessions']} sessions, ${health['plugins']} plugins';
        _testing = false;
      });
    } catch (e) {
      setState(() {
        _testResult = '❌ Failed: $e';
        _testing = false;
      });
    }
  }

  Future<void> _save() async {
    final config = context.read<ServerConfig>();
    await config.setUrl(_urlController.text.trim());
    await config.setCfAccess(
        _cfIdController.text.trim(), _cfSecretController.text.trim());
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Settings saved'), backgroundColor: AppColors.success),
      );
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

          // Server URL
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      const Icon(Icons.dns, color: AppColors.accent, size: 20),
                      const SizedBox(width: 10),
                      const Text('Server Connection', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 15)),
                    ],
                  ),
                  const SizedBox(height: 14),
                  TextField(
                    controller: _urlController,
                    decoration: const InputDecoration(
                      labelText: 'Server URL',
                      hintText: 'https://ntc.example.com',
                      prefixIcon: Icon(Icons.link, size: 18),
                    ),
                    style: const TextStyle(fontSize: 14),
                  ),
                  const SizedBox(height: 14),
                  const Text('Cloudflare Access',
                      style: TextStyle(
                          fontSize: 12,
                          color: AppColors.textMuted,
                          fontWeight: FontWeight.w600)),
                  const SizedBox(height: 6),
                  TextField(
                    controller: _cfIdController,
                    decoration: const InputDecoration(
                      labelText: 'CF-Access-Client-Id',
                      hintText: 'Leave empty if not using CF Access',
                      prefixIcon: Icon(Icons.vpn_key, size: 16),
                      isDense: true,
                    ),
                    style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
                  ),
                  const SizedBox(height: 8),
                  TextField(
                    controller: _cfSecretController,
                    obscureText: true,
                    decoration: const InputDecoration(
                      labelText: 'CF-Access-Client-Secret',
                      hintText: 'Service token secret',
                      prefixIcon: Icon(Icons.lock, size: 16),
                      isDense: true,
                    ),
                    style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: _testing ? null : _testConnection,
                          icon: _testing
                              ? const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2, color: AppColors.accent))
                              : const Icon(Icons.wifi_tethering, size: 16),
                          label: const Text('Test', style: TextStyle(fontSize: 13)),
                          style: OutlinedButton.styleFrom(
                            foregroundColor: AppColors.accent,
                            side: const BorderSide(color: AppColors.border),
                            padding: const EdgeInsets.symmetric(vertical: 10),
                          ),
                        ),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: FilledButton.icon(
                          onPressed: _save,
                          icon: const Icon(Icons.save, size: 16),
                          label: const Text('Save', style: TextStyle(fontSize: 13)),
                          style: FilledButton.styleFrom(
                            backgroundColor: AppColors.accent,
                            padding: const EdgeInsets.symmetric(vertical: 10),
                          ),
                        ),
                      ),
                    ],
                  ),
                  if (_testResult != null) ...[
                    const SizedBox(height: 10),
                    Container(
                      width: double.infinity,
                      padding: const EdgeInsets.all(10),
                      decoration: BoxDecoration(
                        color: _testResult!.startsWith('✅') ? AppColors.successSoft : AppColors.errorSoft,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Text(
                        _testResult!,
                        style: TextStyle(
                          fontSize: 12,
                          color: _testResult!.startsWith('✅') ? AppColors.success : AppColors.error,
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),

          // Plugins
          const PluginsSection(),
          const SizedBox(height: 16),

          // Claude accounts entry — opens the dedicated management page.
          // Kept out of the Plugins section because accounts are a kernel-level
          // resource (they drive env injection in the hub), not a plugin.
          Card(
            child: InkWell(
              onTap: () => context.push('/settings/claude-accounts'),
              borderRadius: BorderRadius.circular(12),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(children: [
                  const Icon(Icons.people_outline,
                      color: AppColors.accent, size: 20),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(context.tr('Claude Accounts'),
                            style: const TextStyle(
                                fontWeight: FontWeight.w600, fontSize: 15)),
                        const SizedBox(height: 4),
                        Text(
                          context.tr('Manage multiple Claude subscriptions (OAuth tokens). Each session picks an account at creation time.'),
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
                        child: const Center(child: Text('N', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14))),
                      ),
                      const SizedBox(width: 10),
                      const Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('NTC', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 15)),
                          Text('Terminal-Centric Development Cockpit', style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
                        ],
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  _InfoRow(label: context.tr('Architecture'),
                      value: context.tr('Micro-kernel + Plugins')),
                  const _InfoRow(label: 'Backend', value: 'Go + chi + gorilla/websocket'),
                  const _InfoRow(label: 'Frontend', value: 'Flutter (Web + Android + iOS)'),
                  const _InfoRow(label: 'Terminal', value: 'xterm.dart + WebSocket'),
                ],
              ),
            ),
          ),
        ],
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
