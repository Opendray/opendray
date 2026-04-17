import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Telegram Bridge panel (M1).
///
/// Read-only status view: bot connection, last poll, sent/received counters,
/// allowed chat count, and a "Send test message" button. The bot itself
/// runs in the Go process — this page just observes and pokes it.
class TelegramPage extends StatefulWidget {
  const TelegramPage({super.key});
  @override
  State<TelegramPage> createState() => _TelegramPageState();
}

class _TelegramPageState extends State<TelegramPage> {
  Map<String, dynamic>? _status;
  List<Map<String, dynamic>> _links = [];
  String? _error;
  bool _loading = true;
  bool _testing = false;
  String? _testResult;
  Timer? _poll;
  StreamSubscription<void>? _providersSub;
  bool _hasPlugin = false;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _checkPlugin();
    _refresh();
    // Poll every 5 s so the connection dot stays live.
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

  Future<void> _unlink(int chatId) async {
    try {
      await _api.telegramUnlink(chatId);
      await _refresh();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text('Unlink failed: $e'),
        backgroundColor: AppColors.error,
      ));
    }
  }

  Future<void> _sendTest() async {
    setState(() {
      _testing = true;
      _testResult = null;
    });
    // read (not watch) — we're in an async callback, not build().
    final l10n = context.read<L10n>();
    try {
      final res = await _api.telegramTest();
      if (!mounted) return;
      setState(() {
        _testing = false;
        _testResult = '✓ ${l10n.t('Test message sent to chat')} ${res['chat']}';
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _testing = false;
        _testResult = '✗ $e';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!_hasPlugin) return _buildNoPlugin();
    if (_loading && _status == null) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    return RefreshIndicator(
      onRefresh: () async {
        await _checkPlugin();
        await _refresh();
      },
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildStatusCard(),
          const SizedBox(height: 12),
          _buildLinksCard(),
          const SizedBox(height: 12),
          _buildTestCard(),
          const SizedBox(height: 12),
          _buildHelpCard(),
        ],
      ),
    );
  }

  Widget _buildLinksCard() {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            const Icon(Icons.link, size: 16, color: AppColors.accent),
            const SizedBox(width: 8),
            Text(context.tr('Active Links'),
                style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
            const Spacer(),
            Text('${_links.length}',
                style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
          ]),
          const SizedBox(height: 4),
          Text(
            context.tr(
                'Each linked Telegram chat sends plain messages to its bound session, and receives the session\'s output (coalesced into 2-second chunks).'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 11),
          ),
          if (_links.isEmpty) ...[
            const SizedBox(height: 12),
            Text(
              context.tr(
                  'No active links yet. In Telegram, send the bot:  /link <session_id>'),
              style: const TextStyle(
                  fontStyle: FontStyle.italic, fontSize: 11, color: AppColors.textMuted),
            ),
          ] else
            ..._links.map(_buildLinkRow),
        ]),
      ),
    );
  }

  Widget _buildLinkRow(Map<String, dynamic> link) {
    final chatId = (link['chatId'] as num?)?.toInt() ?? 0;
    final sessionId = (link['sessionId'] as String?) ?? '';
    final linkedAt = (link['linkedAt'] as num?)?.toInt() ?? 0;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(children: [
        Expanded(
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              const Icon(Icons.chat_outlined, size: 12, color: AppColors.accent),
              const SizedBox(width: 4),
              Text('$chatId',
                  style: const TextStyle(fontFamily: 'monospace', fontSize: 12)),
              const SizedBox(width: 6),
              const Icon(Icons.arrow_forward, size: 12, color: AppColors.textMuted),
              const SizedBox(width: 6),
              const Icon(Icons.terminal, size: 12, color: AppColors.warning),
              const SizedBox(width: 4),
              Flexible(
                child: Text(sessionId,
                    style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
                    maxLines: 1, overflow: TextOverflow.ellipsis),
              ),
            ]),
            if (linkedAt > 0)
              Text(_ago(linkedAt),
                  style: const TextStyle(color: AppColors.textMuted, fontSize: 10)),
          ]),
        ),
        IconButton(
          icon: const Icon(Icons.link_off, size: 16, color: AppColors.error),
          tooltip: context.tr('Unlink'),
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
          onPressed: () => _unlink(chatId),
        ),
      ]),
    );
  }

  Widget _buildNoPlugin() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Text('✈️', style: TextStyle(fontSize: 48)),
          const SizedBox(height: 12),
          Text(context.tr('Telegram Bridge not enabled'),
              style: const TextStyle(fontWeight: FontWeight.w500)),
          const SizedBox(height: 8),
          Text(
            context.tr(
                'Enable the Telegram plugin in Settings → Plugins, then add a Bot Token (from @BotFather) and Allowed Chat IDs.'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
            textAlign: TextAlign.center,
          ),
        ]),
      ),
    );
  }

  Widget _buildStatusCard() {
    final st = _status ?? const {};
    final connected = st['connected'] == true;
    final username = (st['username'] as String?) ?? '';
    final lastPoll = (st['lastPollAt'] as num?)?.toInt() ?? 0;
    final lastErr = (st['lastError'] as String?) ?? '';
    final allowed = (st['allowedChats'] as num?)?.toInt() ?? 0;
    final sent = (st['sent'] as num?)?.toInt() ?? 0;
    final received = (st['received'] as num?)?.toInt() ?? 0;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(connected ? Icons.circle : Icons.circle_outlined,
                size: 12,
                color: connected ? AppColors.success : AppColors.warning),
            const SizedBox(width: 8),
            Text(
              connected
                  ? '${context.tr('Connected as')} $username'
                  : context.tr('Bot offline'),
              style: const TextStyle(
                  fontWeight: FontWeight.w600, fontSize: 14),
            ),
          ]),
          const SizedBox(height: 14),
          _row(context.tr('Allowed chats'), '$allowed'),
          _row(context.tr('Messages sent'), '$sent'),
          _row(context.tr('Updates received'), '$received'),
          if (lastPoll > 0)
            _row(context.tr('Last poll'), _ago(lastPoll)),
          if (lastErr.isNotEmpty) ...[
            const SizedBox(height: 8),
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                  color: AppColors.errorSoft,
                  borderRadius: BorderRadius.circular(6)),
              child: Text(lastErr,
                  style: const TextStyle(
                      color: AppColors.error, fontSize: 11, fontFamily: 'monospace')),
            ),
          ],
          if (_error != null) ...[
            const SizedBox(height: 8),
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                  color: AppColors.errorSoft,
                  borderRadius: BorderRadius.circular(6)),
              child: Text(_error!,
                  style: const TextStyle(color: AppColors.error, fontSize: 11)),
            ),
          ],
        ]),
      ),
    );
  }

  Widget _buildTestCard() {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(context.tr('Send Test'),
              style:
                  const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
          const SizedBox(height: 4),
          Text(
            context.tr(
                'Send a test message to the configured notifications chat — verifies the bot can reach Telegram and that your chat ID is correct.'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 11),
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: _testing ? null : _sendTest,
              style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
              icon: _testing
                  ? const SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white))
                  : const Icon(Icons.send_outlined, size: 16),
              label: Text(context.tr('Send test message'),
                  style: const TextStyle(fontSize: 13)),
            ),
          ),
          if (_testResult != null) ...[
            const SizedBox(height: 8),
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                  color: _testResult!.startsWith('✓')
                      ? AppColors.successSoft
                      : AppColors.errorSoft,
                  borderRadius: BorderRadius.circular(6)),
              child: Text(_testResult!,
                  style: TextStyle(
                      color: _testResult!.startsWith('✓')
                          ? AppColors.success
                          : AppColors.error,
                      fontSize: 12)),
            ),
          ],
        ]),
      ),
    );
  }

  Widget _buildHelpCard() {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(context.tr('Available Commands'),
              style:
                  const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
          const SizedBox(height: 8),
          _cmd('/help', context.tr('Show command list')),
          _cmd('/status', context.tr('List running sessions')),
          _cmd('/tail <id> [n]', context.tr('Last N lines of a session')),
          _cmd('/stop <id>', context.tr('Stop a running session')),
          _cmd('/whoami', context.tr('Show your chat id')),
          const SizedBox(height: 6),
          Text(context.tr('Linked-chat commands'),
              style: const TextStyle(
                  fontWeight: FontWeight.w600, fontSize: 12)),
          const SizedBox(height: 4),
          _cmd('/link <id>', context.tr('Bind chat → session (two-way)')),
          _cmd('/unlink', context.tr('Remove this chat\'s binding')),
          _cmd('/links', context.tr('List all active links')),
          _cmd('/send <id> ...', context.tr('One-shot send without /link')),
          const SizedBox(height: 6),
          Text(context.tr('Quick keys'),
              style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 12)),
          const SizedBox(height: 4),
          _cmd('/cc', 'Ctrl+C'),
          _cmd('/cd', 'Ctrl+D'),
          _cmd('/tab', 'Tab'),
          _cmd('/enter', 'Enter'),
          _cmd('/yes /no', context.tr('Answer yes/no + Enter')),
          const SizedBox(height: 6),
          Text(
              context.tr(
                  'Plain text in a linked chat → session input. Output is polled every 5 s (only sent when content changes). Reply directly to any idle notification → routed automatically.'),
              style: const TextStyle(color: AppColors.textMuted, fontSize: 11)),
        ]),
      ),
    );
  }

  Widget _row(String k, String v) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 3),
        child: Row(children: [
          SizedBox(
              width: 130,
              child: Text(k,
                  style: const TextStyle(
                      color: AppColors.textMuted, fontSize: 12))),
          Expanded(child: Text(v, style: const TextStyle(fontSize: 12))),
        ]),
      );

  Widget _cmd(String cmd, String desc) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 2),
        child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
          SizedBox(
              width: 120,
              child: Text(cmd,
                  style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 11,
                      color: AppColors.accent))),
          Expanded(
              child: Text(desc,
                  style:
                      const TextStyle(fontSize: 11, color: AppColors.text))),
        ]),
      );

  String _ago(int ms) {
    final d = DateTime.fromMillisecondsSinceEpoch(ms);
    final diff = DateTime.now().difference(d);
    if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    return '${diff.inHours}h ago';
  }
}
