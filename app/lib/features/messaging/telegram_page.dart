// Telegram bridge — guided setup + status panel.
//
// Replaces the previous read-only status view with a 3-step wizard for
// non-technical users. Setup state is derived from the live status snapshot
// + recent-chats probe, so we stay in sync with the bot reconcile loop:
//
//   • Step 1 (Create your bot): visible when no token is configured / bot
//     can't reach Telegram. Inline BotFather walkthrough + deep link.
//   • Step 2 (Pick your chat): visible when bot is connected but no
//     allowedChatIds yet. "Detect" probes /api/telegram/recent-chats and
//     lets the user copy each chat ID with one tap (paste back into the
//     plugin config form).
//   • Step 3 (Test + use): visible when fully configured. Status pill,
//     send-test button, active links, command reference.
//
// Plugin config (botToken + allowedChatIds) still lives in Settings →
// Plugins → Telegram → Configure. We surface the deep link prominently
// so users don't have to hunt for it.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/api/api_client.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';
import '../plugins/plugin_configure_page.dart';

class TelegramPage extends StatefulWidget {
  const TelegramPage({super.key});
  @override
  State<TelegramPage> createState() => _TelegramPageState();
}

class _TelegramPageState extends State<TelegramPage> {
  Map<String, dynamic>? _status;
  List<Map<String, dynamic>> _links = [];
  List<Map<String, dynamic>> _recentChats = [];
  bool _loading = true;
  String? _error;
  Timer? _poll;
  StreamSubscription<void>? _providersSub;
  bool _hasPlugin = false;
  bool _detecting = false;
  bool _testing = false;
  String? _testResult;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _checkPlugin();
    _refresh();
    _poll = Timer.periodic(const Duration(seconds: 5), (_) => _refresh());
    _providersSub = ProvidersBus.instance.changes.listen((_) {
      _checkPlugin();
      _refresh();
    });
  }

  @override
  void dispose() {
    _poll?.cancel();
    _providersSub?.cancel();
    super.dispose();
  }

  Future<void> _checkPlugin() async {
    try {
      final all = await _api.listProviders();
      final tg = all.where((p) => p.provider.name == 'telegram').toList();
      if (!mounted) return;
      setState(() {
        _hasPlugin = tg.isNotEmpty && tg.first.enabled;
      });
    } catch (_) {}
  }

  Future<void> _refresh() async {
    try {
      final results = await Future.wait([
        _api.telegramStatus(),
        _api.telegramLinks(),
      ]);
      if (!mounted) return;
      setState(() {
        _status = results[0] as Map<String, dynamic>;
        _links = results[1] as List<Map<String, dynamic>>;
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

  Future<void> _detectChats() async {
    setState(() => _detecting = true);
    try {
      final res = await _api.telegramRecentChats();
      if (!mounted) return;
      final chats = (res['chats'] as List? ?? const [])
          .cast<Map<String, dynamic>>();
      setState(() {
        _recentChats = chats;
        _detecting = false;
      });
      if (chats.isEmpty) {
        _toast('No messages yet. Send your bot a message in Telegram, then try again.');
      }
    } catch (e) {
      if (!mounted) return;
      setState(() => _detecting = false);
      _toast('Detect failed: ${_friendlyError(e.toString())}', isError: true);
    }
  }

  Future<void> _sendTest() async {
    setState(() {
      _testing = true;
      _testResult = null;
    });
    try {
      final res = await _api.telegramTest();
      if (!mounted) return;
      setState(() {
        _testing = false;
        _testResult = '✓ Test message sent to chat ${res['chat']}';
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _testing = false;
        _testResult = '✗ ${_friendlyError(e.toString())}';
      });
    }
  }

  Future<void> _unlink(int chatId) async {
    try {
      await _api.telegramUnlink(chatId);
      await _refresh();
    } catch (e) {
      _toast('Unlink failed: $e', isError: true);
    }
  }

  String _friendlyError(String raw) {
    final msg = raw.toLowerCase();
    if (msg.contains('409') || msg.contains('another bot instance')) {
      return 'Another OpenDray (or bot) is using this token. Stop the other one or wait 30s.';
    }
    if (msg.contains('401') || msg.contains('unauthorized')) {
      return 'Token rejected by Telegram. Double-check the token in Settings → Plugins → Telegram.';
    }
    if (msg.contains('no notification chat') || msg.contains('errnonotifychat')) {
      return 'No notification chat configured. Pick a chat in step 2 below.';
    }
    if (msg.contains('not running') || msg.contains('errbotnotrunning')) {
      return 'Bot is not running. Make sure the plugin is enabled and a token is set.';
    }
    return raw.replaceAll('Exception: ', '').replaceAll('ApiException: ', '');
  }

  void _toast(String msg, {bool isError = false}) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: isError ? t.danger : null,
      duration: const Duration(seconds: 4),
    ));
  }

  // -- Setup-state derivation -------------------------------------------

  bool get _pluginEnabled => _hasPlugin;
  bool get _botConnected => _status?['connected'] == true;
  int get _allowedChats => (_status?['allowedChats'] as int?) ?? 0;
  String get _botUsername =>
      (_status?['username'] as String?)?.trim() ?? '';
  String get _lastError => (_status?['lastError'] as String?)?.trim() ?? '';

  /// Wizard step machine: figure out which step the user is currently
  /// blocked on. Renders the matching guidance prominently while
  /// keeping later steps visible (greyed) so users see the whole flow.
  int get _currentStep {
    if (!_pluginEnabled) return 0;
    if (!_botConnected) return 1;
    if (_allowedChats == 0) return 2;
    return 3;
  }

  // -- Build ---------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    if (_loading && _status == null) {
      return Center(child: CircularProgressIndicator(color: t.accent));
    }
    return Scrollbar(
      child: SingleChildScrollView(
        padding: EdgeInsets.symmetric(horizontal: t.sp5, vertical: t.sp4),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 900),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _Header(
                  connected: _botConnected,
                  username: _botUsername,
                  allowedChats: _allowedChats,
                  step: _currentStep),
              SizedBox(height: t.sp4),
              if (_lastError.isNotEmpty && _pluginEnabled)
                _ErrorBanner(message: _friendlyError(_lastError)),
              if (_error != null) _ErrorBanner(message: _friendlyError(_error!)),
              _Step(
                index: 1,
                title: 'Enable the plugin',
                done: _pluginEnabled,
                active: _currentStep == 0,
                child: _Step1Content(enabled: _pluginEnabled),
              ),
              SizedBox(height: t.sp3),
              _Step(
                index: 2,
                title: 'Create your bot in Telegram',
                done: _botConnected,
                active: _currentStep == 1,
                child: _Step2Content(connected: _botConnected, username: _botUsername),
              ),
              SizedBox(height: t.sp3),
              _Step(
                index: 3,
                title: 'Pick which chats to allow',
                done: _allowedChats > 0,
                active: _currentStep == 2,
                child: _Step3Content(
                  detecting: _detecting,
                  recentChats: _recentChats,
                  allowedCount: _allowedChats,
                  onDetect: _detectChats,
                ),
              ),
              SizedBox(height: t.sp3),
              _Step(
                index: 4,
                title: 'Test + use',
                done: _currentStep == 3 && _testResult?.startsWith('✓') == true,
                active: _currentStep == 3,
                child: _Step4Content(
                  ready: _currentStep == 3,
                  testing: _testing,
                  testResult: _testResult,
                  links: _links,
                  onTest: _sendTest,
                  onUnlink: _unlink,
                ),
              ),
              SizedBox(height: t.sp5),
              const _CommandsReference(),
              SizedBox(height: t.sp5),
            ],
          ),
        ),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Header — bot identity + status pill
// -----------------------------------------------------------------------------

class _Header extends StatelessWidget {
  final bool connected;
  final String username;
  final int allowedChats;
  final int step;
  const _Header({
    required this.connected,
    required this.username,
    required this.allowedChats,
    required this.step,
  });
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    final statusColor = connected ? t.success : (step == 0 ? t.textSubtle : t.warning);
    final statusLabel = connected
        ? '$username connected · $allowedChats allowed chat${allowedChats == 1 ? '' : 's'}'
        : (step == 0 ? 'Plugin not enabled' : 'Bot offline');
    return Container(
      padding: EdgeInsets.all(t.sp4),
      decoration: BoxDecoration(
        color: t.surface,
        borderRadius: BorderRadius.circular(t.rLg),
        border: Border.all(color: t.border),
      ),
      child: Row(
        children: [
          Container(
            width: 40, height: 40,
            decoration: BoxDecoration(
              color: const Color(0xFF229ED9), // Telegram blue
              borderRadius: BorderRadius.circular(20),
            ),
            alignment: Alignment.center,
            child: const Text('✈',
                style: TextStyle(color: Colors.white, fontSize: 18)),
          ),
          SizedBox(width: t.sp3),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Telegram bridge', style: theme.textTheme.titleLarge),
                SizedBox(height: 2),
                Text(
                    'Get notifications and run sessions remotely from your phone.',
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: t.textMuted, fontSize: 12)),
              ],
            ),
          ),
          Container(
            padding: EdgeInsets.symmetric(horizontal: t.sp3, vertical: 6),
            decoration: BoxDecoration(
              color: statusColor.withValues(alpha: 0.14),
              borderRadius: BorderRadius.circular(t.rXl),
              border: Border.all(color: statusColor.withValues(alpha: 0.35)),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 8, height: 8,
                  decoration:
                      BoxDecoration(color: statusColor, shape: BoxShape.circle),
                ),
                SizedBox(width: t.sp2),
                Text(statusLabel,
                    style: TextStyle(
                        fontSize: 11,
                        color: statusColor,
                        fontWeight: FontWeight.w600)),
              ],
            ),
          ),
          SizedBox(width: t.sp2),
          // Always-available reconfigure entry — non-technical users
          // shouldn't have to navigate to /plugins to clear the bot
          // token or swap to a different bot. Pushes the configure form
          // directly so the change is one click away regardless of the
          // current setup state.
          OutlinedButton.icon(
            onPressed: () => Navigator.of(context).push(MaterialPageRoute(
              builder: (_) => const PluginConfigurePage(
                pluginName: 'telegram',
                displayName: 'Telegram',
              ),
            )),
            icon: const Icon(Icons.tune, size: 14),
            label: const Text('Configure'),
            style: OutlinedButton.styleFrom(
              minimumSize: const Size(0, 32),
              padding: EdgeInsets.symmetric(horizontal: t.sp3),
            ),
          ),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Step shell — numbered cards with done / active state
// -----------------------------------------------------------------------------

class _Step extends StatelessWidget {
  final int index;
  final String title;
  final bool done;
  final bool active;
  final Widget child;
  const _Step({
    required this.index,
    required this.title,
    required this.done,
    required this.active,
    required this.child,
  });
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    final dim = !active && !done;
    final accent = done
        ? t.success
        : (active ? t.accent : t.border);
    return Opacity(
      opacity: dim ? 0.6 : 1.0,
      child: Container(
        decoration: BoxDecoration(
          color: t.surface,
          borderRadius: BorderRadius.circular(t.rLg),
          border: Border.all(
              color: active ? t.accent.withValues(alpha: 0.5) : t.border),
        ),
        padding: EdgeInsets.all(t.sp4),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  width: 22, height: 22,
                  decoration: BoxDecoration(
                    color: done ? accent : Colors.transparent,
                    shape: BoxShape.circle,
                    border: Border.all(color: accent, width: 1.5),
                  ),
                  alignment: Alignment.center,
                  child: done
                      ? const Icon(Icons.check, size: 12, color: Colors.white)
                      : Text('$index',
                          style: TextStyle(
                              fontSize: 11,
                              fontWeight: FontWeight.w700,
                              color: active ? accent : t.textSubtle)),
                ),
                SizedBox(width: t.sp3),
                Text(title,
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w600, fontSize: 14)),
              ],
            ),
            SizedBox(height: t.sp3),
            child,
          ],
        ),
      ),
    );
  }
}

class _Step1Content extends StatelessWidget {
  final bool enabled;
  const _Step1Content({required this.enabled});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    if (enabled) {
      return Text('Telegram plugin is enabled.',
          style: TextStyle(fontSize: 12, color: t.textMuted));
    }
    return Row(
      children: [
        Expanded(
          child: Text(
              'Open Plugins → flip the Telegram switch. Then come back here.',
              style: TextStyle(fontSize: 12, color: t.textMuted)),
        ),
        SizedBox(width: t.sp3),
        FilledButton.icon(
          onPressed: () => context.go('/plugins'),
          icon: const Icon(Icons.extension_outlined, size: 14),
          label: const Text('Open Plugins'),
        ),
      ],
    );
  }
}

class _Step2Content extends StatelessWidget {
  final bool connected;
  final String username;
  const _Step2Content({required this.connected, required this.username});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    if (connected) {
      return Text('Bot is online: $username',
          style: TextStyle(fontSize: 12, color: t.textMuted));
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('1. Open Telegram and message @BotFather',
            style: TextStyle(fontSize: 12, color: t.text)),
        SizedBox(height: 4),
        Text('2. Send /newbot, pick a name and a username (must end in "bot")',
            style: TextStyle(fontSize: 12, color: t.text)),
        SizedBox(height: 4),
        Text('3. Copy the token (looks like 123456789:ABCdef...)',
            style: TextStyle(fontSize: 12, color: t.text)),
        SizedBox(height: 4),
        Text('4. Paste it into Settings → Plugins → Telegram → Configure → "Bot Token"',
            style: TextStyle(fontSize: 12, color: t.text)),
        SizedBox(height: t.sp3),
        Wrap(
          spacing: t.sp2,
          runSpacing: t.sp2,
          children: [
            OutlinedButton.icon(
              onPressed: () => launchUrl(
                  Uri.parse('https://t.me/BotFather'),
                  mode: LaunchMode.externalApplication),
              icon: const Icon(Icons.send, size: 14),
              label: const Text('Open @BotFather'),
            ),
            FilledButton.icon(
              onPressed: () => Navigator.of(context).push(MaterialPageRoute(
                builder: (_) => const PluginConfigurePage(
                  pluginName: 'telegram',
                  displayName: 'Telegram',
                ),
              )),
              icon: const Icon(Icons.tune, size: 14),
              label: const Text('Paste token here'),
            ),
          ],
        ),
      ],
    );
  }
}

class _Step3Content extends StatelessWidget {
  final bool detecting;
  final List<Map<String, dynamic>> recentChats;
  final int allowedCount;
  final VoidCallback onDetect;
  const _Step3Content({
    required this.detecting,
    required this.recentChats,
    required this.allowedCount,
    required this.onDetect,
  });

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
            'Send any message to your bot in Telegram (e.g. /start), then click Detect. Copy each chat ID and paste it into "Allowed Chat IDs" in the plugin config.',
            style: TextStyle(fontSize: 12, color: t.text, height: 1.5)),
        SizedBox(height: t.sp3),
        Row(
          children: [
            FilledButton.icon(
              onPressed: detecting ? null : onDetect,
              icon: detecting
                  ? const SizedBox(
                      width: 12, height: 12,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white))
                  : const Icon(Icons.search, size: 14),
              label: Text(detecting ? 'Detecting…' : 'Detect my chats'),
            ),
            SizedBox(width: t.sp3),
            if (allowedCount > 0)
              Text('$allowedCount chat${allowedCount == 1 ? '' : 's'} allowed',
                  style: TextStyle(
                      fontSize: 11,
                      color: t.success,
                      fontWeight: FontWeight.w600)),
          ],
        ),
        if (recentChats.isNotEmpty) ...[
          SizedBox(height: t.sp3),
          Container(
            decoration: BoxDecoration(
              color: t.bgRaised,
              borderRadius: BorderRadius.circular(t.rMd),
              border: Border.all(color: t.border),
            ),
            child: Column(
              children: [
                for (final c in recentChats) _ChatRow(chat: c),
              ],
            ),
          ),
        ],
      ],
    );
  }
}

class _ChatRow extends StatelessWidget {
  final Map<String, dynamic> chat;
  const _ChatRow({required this.chat});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final id = chat['chatId'].toString();
    final type = chat['type'] as String? ?? '';
    final title = (chat['title'] as String?) ?? '';
    final username = (chat['username'] as String?) ?? '';
    final name = (chat['name'] as String?) ?? '';
    final label = title.isNotEmpty
        ? title
        : (name.isNotEmpty ? name : (username.isNotEmpty ? '@$username' : 'Chat'));
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: t.sp3, vertical: t.sp2),
      child: Row(
        children: [
          Icon(
              type == 'private'
                  ? Icons.person_outline
                  : Icons.group_outlined,
              size: 14,
              color: t.textMuted),
          SizedBox(width: t.sp2),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label,
                    style: const TextStyle(
                        fontSize: 13, fontWeight: FontWeight.w600),
                    overflow: TextOverflow.ellipsis),
                Text('$type · ID $id',
                    style: TextStyle(
                        fontSize: 10,
                        color: t.textSubtle,
                        fontFamily: 'monospace')),
              ],
            ),
          ),
          OutlinedButton.icon(
            onPressed: () {
              Clipboard.setData(ClipboardData(text: id));
              ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('Copied ID $id')));
            },
            icon: const Icon(Icons.copy, size: 12),
            label: Text('Copy ID', style: const TextStyle(fontSize: 11)),
            style: OutlinedButton.styleFrom(
              minimumSize: const Size(0, 28),
              padding: EdgeInsets.symmetric(horizontal: t.sp2),
            ),
          ),
        ],
      ),
    );
  }
}

class _Step4Content extends StatelessWidget {
  final bool ready;
  final bool testing;
  final String? testResult;
  final List<Map<String, dynamic>> links;
  final VoidCallback onTest;
  final void Function(int) onUnlink;
  const _Step4Content({
    required this.ready,
    required this.testing,
    required this.testResult,
    required this.links,
    required this.onTest,
    required this.onUnlink,
  });
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            FilledButton.icon(
              onPressed: ready && !testing ? onTest : null,
              icon: testing
                  ? const SizedBox(
                      width: 12, height: 12,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white))
                  : const Icon(Icons.send_outlined, size: 14),
              label: Text(testing ? 'Sending…' : 'Send test message'),
            ),
          ],
        ),
        if (testResult != null) ...[
          SizedBox(height: t.sp2),
          Text(testResult!,
              style: TextStyle(
                  fontSize: 12,
                  color: testResult!.startsWith('✓') ? t.success : t.danger)),
        ],
        SizedBox(height: t.sp4),
        Text('Active links',
            style: TextStyle(
                fontSize: 11,
                color: t.textSubtle,
                fontWeight: FontWeight.w700,
                letterSpacing: 0.6)),
        SizedBox(height: t.sp2),
        if (links.isEmpty)
          Text(
              'None yet. In Telegram send your bot: /link <session-id>. The chat will then be paired with that session.',
              style: TextStyle(fontSize: 12, color: t.textMuted))
        else
          Container(
            decoration: BoxDecoration(
              color: t.bgRaised,
              borderRadius: BorderRadius.circular(t.rMd),
              border: Border.all(color: t.border),
            ),
            child: Column(
              children: [
                for (final l in links)
                  Padding(
                    padding: EdgeInsets.symmetric(
                        horizontal: t.sp3, vertical: t.sp2),
                    child: Row(
                      children: [
                        Icon(Icons.link, size: 14, color: t.textMuted),
                        SizedBox(width: t.sp2),
                        Expanded(
                          child: Text(
                              'chat ${l['chatId']} → session ${(l['sessionId'] as String).substring(0, 8)}',
                              style: TextStyle(
                                  fontSize: 12, fontFamily: 'monospace')),
                        ),
                        TextButton(
                            onPressed: () => onUnlink(l['chatId'] as int),
                            child: const Text('Unlink',
                                style: TextStyle(fontSize: 11))),
                      ],
                    ),
                  ),
              ],
            ),
          ),
      ],
    );
  }
}

// -----------------------------------------------------------------------------
// Commands reference (always visible — useful before AND after setup)
// -----------------------------------------------------------------------------

class _CommandsReference extends StatelessWidget {
  const _CommandsReference();
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    const cmds = [
      ('/sessions', 'List your running sessions'),
      ('/link <id>', 'Link this chat to a session — messages forward both ways'),
      ('/unlink', 'Stop forwarding to this chat'),
      ('/screen', 'Snapshot the linked session\'s terminal'),
      ('/tail [n]', 'Show the last n lines of the linked session'),
      ('/send <text>', 'Send text to the linked session as input'),
      ('/stop', 'Stop the linked session'),
      ('/help', 'List all commands'),
    ];
    return Container(
      decoration: BoxDecoration(
        color: t.surface,
        borderRadius: BorderRadius.circular(t.rLg),
        border: Border.all(color: t.border),
      ),
      padding: EdgeInsets.all(t.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Commands you can send to the bot',
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w600, fontSize: 13)),
          SizedBox(height: t.sp3),
          for (final c in cmds)
            Padding(
              padding: EdgeInsets.only(bottom: t.sp2),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SizedBox(
                    width: 110,
                    child: Text(c.$1,
                        style: TextStyle(
                            fontSize: 11,
                            fontFamily: 'monospace',
                            color: t.accent,
                            fontWeight: FontWeight.w600)),
                  ),
                  Expanded(
                    child: Text(c.$2,
                        style: TextStyle(fontSize: 12, color: t.textMuted)),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Error banner
// -----------------------------------------------------------------------------

class _ErrorBanner extends StatelessWidget {
  final String message;
  const _ErrorBanner({required this.message});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      margin: EdgeInsets.only(bottom: t.sp3),
      padding: EdgeInsets.all(t.sp3),
      decoration: BoxDecoration(
        color: t.dangerSoft,
        borderRadius: BorderRadius.circular(t.rMd),
        border: Border.all(color: t.danger.withValues(alpha: 0.4)),
      ),
      child: Row(
        children: [
          Icon(Icons.error_outline, color: t.danger, size: 16),
          SizedBox(width: t.sp2),
          Expanded(
              child: Text(message,
                  style: TextStyle(color: t.danger, fontSize: 12))),
        ],
      ),
    );
  }
}
