import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:provider/provider.dart';
import 'package:xterm/xterm.dart';
import '../../core/api/api_client.dart';
import '../../core/models/session.dart';
import '../../core/services/auth_service.dart';
import '../../core/services/l10n.dart';
import '../../core/services/server_config.dart';
import '../../core/services/ws_client.dart';
import '../../shared/app_modals.dart';
import '../../shared/image_attach.dart';
import '../../shared/voice_composer.dart';
import '../../shared/theme/app_theme.dart';
import '../../shared/theme/terminal_theme.dart';
import 'widgets/quick_keys_bar.dart';
import 'widgets/web_terminal.dart';

const _icons = <String, String>{
  'claude': '\u{1F7E3}',
  'gemini': '\u{2728}',
  'codex': '\u{1F916}',
  'lmstudio': '\u{1F9E0}',
  'ollama': '\u{1F999}',
  'terminal': '\u{2B1B}',
};


class SessionPage extends StatefulWidget {
  final String sessionId;
  const SessionPage({super.key, required this.sessionId});
  @override
  State<SessionPage> createState() => _SessionPageState();
}

class _SessionPageState extends State<SessionPage> with WidgetsBindingObserver {
  late final Terminal _terminal;
  late final TerminalController _termController;
  late final ScrollController _scrollController;
  late final WsClient _ws;
  bool _showQuickKeys = true;

  Session? _session;
  bool _waitingForInput = false;
  bool _connected = false;
  bool _replaying = false;
  int _reconnectAttempt = 0;
  Timer? _resizeDebounce;

  // Claude multi-account — populated once per session load when the session
  // is a Claude agent. The chip/menu opens for hot-swap.
  List<Map<String, dynamic>> _claudeAccounts = [];
  bool _switchingAccount = false;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();

    _terminal = Terminal(maxLines: 10000);
    _termController = TerminalController();
    _scrollController = ScrollController();

    final config = context.read<ServerConfig>();
    final auth = context.read<AuthService>();
    _ws = WsClient(
      baseUrl: config.effectiveUrl,
      extraHeaders: config.cfAccessHeaders,
      tokenProvider: () => auth.token,
    );

    // Native terminal: keyboard → onOutput → WebSocket (zero bridge latency)
    if (!kIsWeb) {
      _terminal.onOutput = (data) {
        _ws.sendBinary(Uint8List.fromList(utf8.encode(data)));
      };
      _terminal.onResize = (w, h, pw, ph) {
        _scheduleResize(w, h);
      };

      _ws.onBinaryMessage = (data) {
        _terminal.write(utf8.decode(data, allowMalformed: true));
      };
      _ws.onControlMessage = (msg) {
        final type = msg['type'] as String?;
        switch (type) {
          case 'replay_start':
            if (mounted) setState(() => _replaying = true);
          case 'replay_end':
            if (mounted) setState(() => _replaying = false);
            _scrollToBottom();
          case 'waiting_for_input':
            if (mounted) setState(() => _waitingForInput = true);
          case 'process_exit':
            _terminal.write('\r\n\x1b[33m--- process exited ---\x1b[0m\r\n');
            _loadSession();
        }
      };
      _ws.onConnected = () {
        if (mounted) setState(() { _connected = true; _reconnectAttempt = 0; });
      };
      _ws.onDisconnected = () {
        if (mounted) setState(() => _connected = false);
      };
      _ws.onReconnecting = (attempt, _) {
        if (mounted) setState(() => _reconnectAttempt = attempt);
      };
    }

    WidgetsBinding.instance.addObserver(this);
    _loadSession();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed &&
        _session?.isRunning == true && !kIsWeb && !_ws.isConnected) {
      _ws.connect(widget.sessionId);
    }
  }

  void _scheduleResize(int cols, int rows) {
    _resizeDebounce?.cancel();
    _resizeDebounce = Timer(const Duration(milliseconds: 300), () {
      _ws.sendResize(cols, rows);
    });
  }

  /// Send input to terminal — native path bypasses all JS bridges.
  Future<bool> _sendToTerminal(String text) async {
    if (_session?.isRunning != true || text.isEmpty) return false;
    if (kIsWeb) {
      try { await _api.sendInput(widget.sessionId, text); return true; }
      catch (_) { return false; }
    }
    _ws.sendBinary(Uint8List.fromList(utf8.encode(text)));
    return true;
  }

  /// Opens a bottom sheet with clipboard actions for the session:
  ///   • Copy selection  — whatever the user long-pressed and highlighted
  ///   • Copy visible    — the full current viewport (useful when nothing
  ///                       is selected but the whole OAuth URL is on screen)
  ///   • Paste           — reads system clipboard, sends to terminal
  ///
  /// Driven from the clipboard toolbar icon. Web terminals have their
  /// own copy/paste from xterm.js so this is mobile-native only.
  Future<void> _showClipboardMenu() async {
    final selectionRange = _termController.selection;
    final hasSelection = selectionRange != null;

    await showAppModalBottomSheet<void>(
      context: context,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 14, 16, 6),
              child: Row(children: [
                const Icon(Icons.content_paste_outlined,
                    size: 18, color: AppColors.accent),
                const SizedBox(width: 8),
                Text(context.tr('Clipboard'),
                    style: const TextStyle(
                        fontWeight: FontWeight.w600, fontSize: 14)),
              ]),
            ),
            const Divider(height: 1),
            ListTile(
              leading: Icon(Icons.copy_outlined,
                  color: hasSelection ? AppColors.accent : AppColors.textMuted,
                  size: 20),
              title: Text(context.tr('Copy selection'),
                  style: const TextStyle(fontSize: 14)),
              subtitle: Text(
                hasSelection
                    ? context.tr('Long-pressed text → clipboard')
                    : context.tr('Long-press text in the terminal first'),
                style: const TextStyle(fontSize: 11, color: AppColors.textMuted),
              ),
              enabled: hasSelection,
              onTap: hasSelection
                  ? () {
                      Navigator.pop(ctx);
                      _copySelection();
                    }
                  : null,
            ),
            ListTile(
              leading: const Icon(Icons.select_all,
                  color: AppColors.accent, size: 20),
              title: Text(context.tr('Copy visible screen'),
                  style: const TextStyle(fontSize: 14)),
              subtitle: Text(
                context.tr('Everything currently on the terminal view'),
                style: const TextStyle(fontSize: 11, color: AppColors.textMuted),
              ),
              onTap: () {
                Navigator.pop(ctx);
                _copyVisible();
              },
            ),
            const Divider(height: 1, indent: 16),
            ListTile(
              leading: const Icon(Icons.content_paste,
                  color: AppColors.accent, size: 20),
              title: Text(context.tr('Paste'),
                  style: const TextStyle(fontSize: 14)),
              subtitle: Text(
                context.tr('Send clipboard contents to the session'),
                style: const TextStyle(fontSize: 11, color: AppColors.textMuted),
              ),
              onTap: () {
                Navigator.pop(ctx);
                _pasteFromClipboard();
              },
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }

  /// Copies the user's current terminal selection (set via long-press /
  /// drag on the xterm view) to the system clipboard.
  Future<void> _copySelection() async {
    final range = _termController.selection;
    if (range == null) return;
    final text = _terminal.buffer.getText(range);
    final emptyMsg = context.tr('Nothing selected');
    final copiedLabel = context.tr('Copied');
    if (text.isEmpty) {
      _snack(emptyMsg);
      return;
    }
    await Clipboard.setData(ClipboardData(text: text));
    _termController.clearSelection();
    _snack('$copiedLabel (${text.length} chars)');
  }

  /// Copies the entire terminal buffer (scrollback included) — useful
  /// when the user wants a multi-line OAuth URL that wraps across rows
  /// without wrestling with selection on a small touch screen.
  Future<void> _copyVisible() async {
    final rows = <String>[];
    final buf = _terminal.buffer;
    final total = buf.lines.length;
    for (int i = 0; i < total; i++) {
      rows.add(buf.lines[i].toString().trimRight());
    }
    // Trim leading blank rows so the user doesn't get N screenfuls of
    // pre-boot whitespace when the buffer isn't full yet.
    while (rows.isNotEmpty && rows.first.isEmpty) {
      rows.removeAt(0);
    }
    final text = rows.join('\n').trimRight();
    final emptyMsg = context.tr('Screen is empty');
    final copiedLabel = context.tr('Copied');
    if (text.isEmpty) {
      _snack(emptyMsg);
      return;
    }
    await Clipboard.setData(ClipboardData(text: text));
    _snack('$copiedLabel (${text.split("\n").length} lines)');
  }

  /// Writes clipboard contents into the session, as if the user had
  /// typed them. Keeps newlines intact — an OAuth `code` usually is a
  /// single line, but scripts / multiline paste works too.
  Future<void> _pasteFromClipboard() async {
    final emptyMsg = context.tr('Clipboard is empty');
    final pastedLabel = context.tr('Pasted');
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    final text = data?.text;
    if (text == null || text.isEmpty) {
      _snack(emptyMsg);
      return;
    }
    await _sendToTerminal(text);
    _snack('$pastedLabel (${text.length} chars)');
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      duration: const Duration(seconds: 2),
    ));
  }

  void _scrollToBottom() {
    if (!_scrollController.hasClients) return;
    _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _resizeDebounce?.cancel();
    _ws.dispose();
    _scrollController.dispose();
    _termController.dispose();
    super.dispose();
  }

  Future<void> _loadSession() async {
    try {
      final s = await _api.getSession(widget.sessionId);
      if (!mounted) return;
      setState(() => _session = s);
      // Auto-connect WS when session is already running (e.g. navigated from
      // Dashboard into a running session, or after app resume).
      if (!kIsWeb && s.isRunning && !_ws.isConnected) {
        _ws.connect(widget.sessionId);
      }
      if (s.sessionType == 'claude' && _claudeAccounts.isEmpty) {
        _loadClaudeAccounts();
      }
    } catch (_) {}
  }

  Future<void> _loadClaudeAccounts() async {
    try {
      final accounts = await _api.claudeAccounts();
      if (!mounted) return;
      setState(() {
        _claudeAccounts = accounts
            .where((a) => (a['enabled'] as bool? ?? true) &&
                (a['tokenFilled'] as bool? ?? false))
            .toList();
      });
    } catch (_) {
      // Older server — hide the chip silently.
    }
  }

  Future<void> _switchAccount(String? accountId) async {
    if (_switchingAccount) return;
    setState(() => _switchingAccount = true);
    try {
      _ws.close();
      await ApiClient.describeErrors(() =>
          _api.switchSessionAccount(widget.sessionId, accountId));
      await _loadSession();
      if (_session?.isRunning == true && !kIsWeb) {
        _terminal.write('\x1b[2J\x1b[H');
        _ws.connect(widget.sessionId);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text('Switch failed: $e'),
          backgroundColor: AppColors.error,
          duration: const Duration(seconds: 6),
        ));
      }
    } finally {
      if (mounted) setState(() => _switchingAccount = false);
    }
  }

  Map<String, dynamic>? get _boundAccount {
    final id = _session?.claudeAccountId;
    if (id == null || id.isEmpty) return null;
    for (final a in _claudeAccounts) {
      if (a['id'] == id) return a;
    }
    return null;
  }

  Future<void> _start() async {
    try {
      await ApiClient.describeErrors(
          () => _api.startSession(widget.sessionId));
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text('Start failed: $e'),
          backgroundColor: AppColors.error,
          duration: const Duration(seconds: 6),
        ));
      }
      await _loadSession();
      return;
    }
    await _loadSession();
    _terminal.write('\x1b[2J\x1b[H');
    _ws.connect(widget.sessionId);
  }

  Future<void> _stop() async {
    _ws.close();
    await _api.stopSession(widget.sessionId);
    await _loadSession();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            _buildToolbar(),
            Expanded(
              child: Stack(
                children: [
                  // Terminal — xterm.js iframe on web, Dart xterm on mobile
                  if (kIsWeb)
                    Container(
                      color: const Color(0xFF0B0D11),
                      child: WebTerminalView(
                        serverUrl: context.read<ServerConfig>().effectiveUrl,
                        sessionId: widget.sessionId,
                        isRunning: _session?.isRunning == true,
                        authToken: context.read<AuthService>().token,
                        onEvent: (type) {
                          switch (type) {
                            case 'idle':
                              if (mounted) setState(() => _waitingForInput = true);
                            case 'exit':
                              _loadSession();
                            case 'connected':
                              if (mounted) setState(() { _connected = true; _reconnectAttempt = 0; });
                            case 'disconnected':
                              if (mounted) setState(() => _connected = false);
                          }
                        },
                      ),
                    )
                  else
                    Container(
                      color: const Color(0xFF0B0D11),
                      child: TerminalView(
                        _terminal,
                        controller: _termController,
                        theme: ntcTerminalTheme,
                        textStyle: TerminalStyle(
                          fontSize: 13,
                          fontFamily: GoogleFonts.jetBrainsMono().fontFamily ?? 'monospace',
                        ),
                        scrollController: _scrollController,
                        autofocus: true,
                      ),
                    ),

                  // Replay loading overlay
                  if (_replaying)
                    Positioned.fill(
                      child: Container(
                        color: const Color(0xCC0B0D11),
                        child: const Center(
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              SizedBox(
                                width: 24,
                                height: 24,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                  color: AppColors.accent,
                                ),
                              ),
                              SizedBox(height: 12),
                              Text(
                                'Loading history...',
                                style: TextStyle(
                                  color: AppColors.textMuted,
                                  fontSize: 12,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),

                  // Reconnecting overlay
                  if (!_connected &&
                      _session?.isRunning == true &&
                      _reconnectAttempt > 0)
                    Positioned(
                      left: 0,
                      right: 0,
                      bottom: 0,
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 10,
                        ),
                        decoration: const BoxDecoration(
                          gradient: LinearGradient(
                            begin: Alignment.bottomCenter,
                            end: Alignment.topCenter,
                            colors: [
                              Color(0xE00B0D11),
                              Color(0x000B0D11),
                            ],
                          ),
                        ),
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            const SizedBox(
                              width: 12,
                              height: 12,
                              child: CircularProgressIndicator(
                                strokeWidth: 1.5,
                                color: AppColors.warning,
                              ),
                            ),
                            const SizedBox(width: 8),
                            Text(
                              'Reconnecting (attempt $_reconnectAttempt)...',
                              style: const TextStyle(
                                color: AppColors.warning,
                                fontSize: 11,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),

                  // Session stopped overlay
                  if (_session != null && !_session!.isRunning)
                    Container(
                      color: const Color(0xE00B0D11),
                      child: Center(
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              _icons[_session?.sessionType] ?? '?',
                              style: const TextStyle(fontSize: 40),
                            ),
                            const SizedBox(height: 12),
                            Text(
                              'Session ${_session!.status}',
                              style: const TextStyle(
                                color: AppColors.textMuted,
                                fontSize: 14,
                              ),
                            ),
                            const SizedBox(height: 16),
                            FilledButton(
                              onPressed: _start,
                              style: FilledButton.styleFrom(
                                backgroundColor: AppColors.accent,
                              ),
                              child: const Text('Start Session'),
                            ),
                          ],
                        ),
                      ),
                    ),

                  // Scroll-to-bottom reserved for future use
                ],
              ),
            ),
            // Quick keys bar — special keys (Tab, Esc, arrows, ^C, custom).
            // Regular typing is done directly in the terminal (xterm.js) —
            // tap the terminal to raise the keyboard. The hardening in
            // terminal_html.go (disabled autocapitalize/autocorrect, forced
            // re-focus on tap/visibility/focus) makes this reliable.
            if (!kIsWeb && _showQuickKeys && _session?.isRunning == true)
              QuickKeysBar(onSendKey: (data) => _sendToTerminal(data)),
          ],
        ),
      ),
    );
  }

  Future<void> _showAccountPicker() async {
    final currentId = _session?.claudeAccountId ?? '';
    final picked = await showAppModalBottomSheet<String?>(
      context: context,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
      builder: (ctx) {
        return SafeArea(
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
              child: Row(children: [
                const Icon(Icons.person, size: 16, color: AppColors.accent),
                const SizedBox(width: 8),
                Text(ctx.tr('Claude account'),
                    style: const TextStyle(
                        fontWeight: FontWeight.w600, fontSize: 14)),
                const Spacer(),
                TextButton.icon(
                  onPressed: () {
                    Navigator.pop(ctx);
                    context.push('/settings/claude-accounts');
                  },
                  icon: const Icon(Icons.settings, size: 14),
                  label: Text(ctx.tr('Manage'),
                      style: const TextStyle(fontSize: 12)),
                ),
              ]),
            ),
            const Divider(height: 1),
            ListTile(
              dense: true,
              leading: const Icon(Icons.lock_outline,
                  color: AppColors.textMuted, size: 18),
              title: Text(ctx.tr('System (keychain / env)'),
                  style: const TextStyle(fontSize: 13)),
              subtitle: Text(ctx.tr('No env override'),
                  style: const TextStyle(
                      fontSize: 10, color: AppColors.textMuted)),
              trailing: currentId.isEmpty
                  ? const Icon(Icons.check, size: 16, color: AppColors.accent)
                  : null,
              onTap: () => Navigator.pop(ctx, ''),
            ),
            const Divider(height: 1, indent: 16),
            ..._claudeAccounts.map((a) {
              final isCurrent = a['id'] == currentId;
              final display = (a['displayName'] as String?)?.isNotEmpty == true
                  ? a['displayName'] as String
                  : a['name'] as String? ?? '';
              return ListTile(
                dense: true,
                leading: const Icon(Icons.person_outline,
                    color: AppColors.accent, size: 18),
                title: Text(display,
                    style: TextStyle(
                        fontSize: 13,
                        fontWeight: isCurrent
                            ? FontWeight.w600
                            : FontWeight.normal,
                        color: isCurrent ? AppColors.accent : null)),
                subtitle: Text('claude-${a['name']}',
                    style: const TextStyle(
                        fontSize: 10,
                        fontFamily: 'monospace',
                        color: AppColors.textMuted)),
                trailing: isCurrent
                    ? const Icon(Icons.check,
                        size: 16, color: AppColors.accent)
                    : null,
                onTap: () => Navigator.pop(ctx, a['id'] as String),
              );
            }),
            const SizedBox(height: 8),
          ]),
        );
      },
    );

    if (picked == null) return;
    if (picked == currentId) return;
    await _switchAccount(picked.isEmpty ? null : picked);
  }

  Future<void> _showSessionSwitcher() async {
    List<Session> sessions = [];
    try { sessions = await _api.listSessions(); } catch (_) {}
    if (!mounted) return;
    showAppModalBottomSheet(
      context: context,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
      builder: (ctx) => Column(mainAxisSize: MainAxisSize.min, children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
          child: Row(children: [
            const Icon(Icons.layers, size: 16, color: AppColors.accent),
            const SizedBox(width: 8),
            Text('${sessions.length} sessions',
                style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
            const Spacer(),
            TextButton.icon(
              onPressed: () { Navigator.pop(ctx); context.go('/'); },
              icon: const Icon(Icons.add, size: 14),
              label: const Text('New', style: TextStyle(fontSize: 12)),
              style: TextButton.styleFrom(foregroundColor: AppColors.accent,
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4)),
            ),
          ]),
        ),
        const Divider(height: 1),
        Flexible(child: ListView.separated(
          shrinkWrap: true,
          padding: const EdgeInsets.symmetric(vertical: 4),
          itemCount: sessions.length,
          separatorBuilder: (_, _) => const Divider(height: 1, indent: 16),
          itemBuilder: (_, i) {
            final s = sessions[i];
            final isCurrent = s.id == widget.sessionId;
            return ListTile(
              dense: true,
              leading: Text(_icons[s.sessionType] ?? '?',
                  style: const TextStyle(fontSize: 18)),
              title: Text(s.name.isNotEmpty ? s.name : s.sessionType,
                  style: TextStyle(fontSize: 13,
                      color: isCurrent ? AppColors.accent : null,
                      fontWeight: isCurrent ? FontWeight.w600 : FontWeight.normal)),
              subtitle: Text(s.status,
                  style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
              trailing: Row(mainAxisSize: MainAxisSize.min, children: [
                Container(
                  width: 7, height: 7,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: s.isRunning ? AppColors.success : AppColors.textMuted,
                  ),
                ),
                if (isCurrent) ...[
                  const SizedBox(width: 6),
                  const Icon(Icons.check, size: 14, color: AppColors.accent),
                ],
              ]),
              onTap: () {
                Navigator.pop(ctx);
                if (!isCurrent) context.pushReplacement('/session/${s.id}');
              },
            );
          },
        )),
        const SizedBox(height: 8),
      ]),
    );
  }

  Widget _buildToolbar() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(
        children: [
          IconButton(
            icon: const Icon(Icons.layers_outlined, size: 20,
                color: AppColors.textMuted),
            onPressed: _showSessionSwitcher,
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
            tooltip: 'Switch session',
          ),
          Text(
            _icons[_session?.sessionType] ?? '?',
            style: const TextStyle(fontSize: 18),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  _session?.name ?? 'Session',
                  style: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
                if (_session?.model.isNotEmpty == true)
                  Text(
                    _session!.model,
                    style: const TextStyle(
                      fontSize: 10,
                      color: AppColors.textMuted,
                    ),
                  ),
              ],
            ),
          ),
          // Claude account chip — tap to hot-swap. Only rendered for Claude
          // sessions, and only when there's at least one viable account.
          if (_session?.sessionType == 'claude' && _claudeAccounts.isNotEmpty)
            _AccountChip(
              account: _boundAccount,
              switching: _switchingAccount,
              onTap: _showAccountPicker,
            ),
          // Status badges
          if (_waitingForInput)
            _StatusBadge(
              label: 'Idle',
              color: AppColors.warning,
              bgColor: AppColors.warningSoft,
            ),
          if (!_connected &&
              _session?.isRunning == true &&
              _reconnectAttempt == 0)
            const Padding(
              padding: EdgeInsets.only(left: 6),
              child: _StatusBadge(
                label: '...',
                color: AppColors.error,
                bgColor: AppColors.errorSoft,
              ),
            ),
          const SizedBox(width: 6),
          // Connection dot
          _AnimatedDot(
            color: _session?.isRunning == true
                ? (_connected ? AppColors.success : AppColors.warning)
                : AppColors.textMuted,
            animate: _session?.isRunning == true && !_connected,
          ),
          const SizedBox(width: 4),
          // Attach image — upload photo / gallery, insert path via live WS
          if (_session?.isRunning == true)
            IconButton(
              icon: const Icon(Icons.attach_file, size: 20, color: AppColors.accent),
              onPressed: () => pickAndSendImage(
                context,
                targetSession: _session,
                inserter: (text) async { await _sendToTerminal(text); },
              ),
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              tooltip: context.tr('Attach image'),
            ),
          // Voice dictation — opens a composer; the phone's IME handles STT
          if (_session?.isRunning == true)
            IconButton(
              icon: const Icon(Icons.mic_none, size: 20, color: AppColors.accent),
              onPressed: () => showVoiceComposer(
                context,
                onSend: (text) => _sendToTerminal(text),
              ),
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              tooltip: context.tr('Voice input'),
            ),
          // Clipboard — copy terminal selection / paste into the session.
          // Essential for headless-server workflows where the user needs
          // to bounce an OAuth URL out to a browser and a code back in.
          if (!kIsWeb && _session?.isRunning == true)
            IconButton(
              icon: const Icon(Icons.content_paste_outlined,
                  size: 20, color: AppColors.accent),
              onPressed: _showClipboardMenu,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              tooltip: context.tr('Clipboard'),
            ),
          // Toggle quick keys bar (mobile only)
          if (!kIsWeb && _session?.isRunning == true)
            IconButton(
              icon: Icon(
                _showQuickKeys ? Icons.keyboard_hide : Icons.keyboard,
                size: 20,
                color: _showQuickKeys ? AppColors.accent : AppColors.textMuted,
              ),
              onPressed: () => setState(() => _showQuickKeys = !_showQuickKeys),
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              tooltip: 'Toggle quick keys',
            ),
          const SizedBox(width: 4),
          if (_session?.isRunning != true)
            _SmallButton(
              label: 'Start',
              color: AppColors.success,
              onTap: _start,
            ),
          if (_session?.isRunning == true)
            _SmallButton(
              label: 'Stop',
              color: AppColors.error,
              onTap: _stop,
            ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Sub-widgets
// ---------------------------------------------------------------------------

class _StatusBadge extends StatelessWidget {
  final String label;
  final Color color;
  final Color bgColor;
  const _StatusBadge({
    required this.label,
    required this.color,
    required this.bgColor,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontSize: 10,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}

/// Animated connection dot — pulses when reconnecting.
class _AnimatedDot extends StatefulWidget {
  final Color color;
  final bool animate;
  const _AnimatedDot({required this.color, required this.animate});

  @override
  State<_AnimatedDot> createState() => _AnimatedDotState();
}

class _AnimatedDotState extends State<_AnimatedDot>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1000),
    );
    if (widget.animate) _ctrl.repeat(reverse: true);
  }

  @override
  void didUpdateWidget(_AnimatedDot old) {
    super.didUpdateWidget(old);
    if (widget.animate && !_ctrl.isAnimating) {
      _ctrl.repeat(reverse: true);
    } else if (!widget.animate && _ctrl.isAnimating) {
      _ctrl.stop();
      _ctrl.value = 1.0;
    }
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _ctrl,
      builder: (_, _) => Container(
        width: 8,
        height: 8,
        decoration: BoxDecoration(
          color: widget.color.withValues(
            alpha: widget.animate ? 0.3 + _ctrl.value * 0.7 : 1.0,
          ),
          shape: BoxShape.circle,
        ),
      ),
    );
  }
}


class _SmallButton extends StatelessWidget {
  final String label;
  final Color color;
  final VoidCallback onTap;
  const _SmallButton({
    required this.label,
    required this.color,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Material(
      color: color.withValues(alpha: 0.15),
      borderRadius: BorderRadius.circular(6),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(6),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
          child: Text(
            label,
            style: TextStyle(
              color: color,
              fontSize: 11,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
      ),
    );
  }
}

/// Small chip surfacing the bound Claude account (or "keychain" when the
/// session has no claude_account_id). Tapping opens the picker. Shows a
/// tiny spinner while a hot-swap is in flight so the user knows not to
/// spam taps.
class _AccountChip extends StatelessWidget {
  final Map<String, dynamic>? account;
  final bool switching;
  final VoidCallback onTap;
  const _AccountChip({
    required this.account,
    required this.switching,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final hasAccount = account != null;
    final label = hasAccount
        ? ((account!['displayName'] as String?)?.isNotEmpty == true
            ? account!['displayName'] as String
            : account!['name'] as String? ?? '?')
        : 'keychain';
    final color = hasAccount ? AppColors.accent : AppColors.textMuted;
    final bg = hasAccount ? AppColors.accentSoft : AppColors.surfaceAlt;

    return Padding(
      padding: const EdgeInsets.only(right: 4),
      child: Material(
        color: bg,
        borderRadius: BorderRadius.circular(6),
        child: InkWell(
          onTap: switching ? null : onTap,
          borderRadius: BorderRadius.circular(6),
          child: Padding(
            padding:
                const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            child: Row(mainAxisSize: MainAxisSize.min, children: [
              if (switching)
                SizedBox(
                  width: 10,
                  height: 10,
                  child: CircularProgressIndicator(
                    strokeWidth: 1.5,
                    color: color,
                  ),
                )
              else
                Icon(
                  hasAccount ? Icons.person : Icons.lock_outline,
                  size: 12,
                  color: color,
                ),
              const SizedBox(width: 4),
              ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 80),
                child: Text(
                  label,
                  style: TextStyle(
                    color: color,
                    fontSize: 10,
                    fontWeight: FontWeight.w500,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              const SizedBox(width: 2),
              Icon(Icons.arrow_drop_down, size: 12, color: color),
            ]),
          ),
        ),
      ),
    );
  }
}
