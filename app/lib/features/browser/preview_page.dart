import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/image_attach.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

enum _Mode { webview, simulator }

class PreviewPage extends StatefulWidget {
  /// When set, only plugins whose `category` matches are shown. Used by the
  /// launcher to split the unified preview page into two separate tiles —
  /// one for `preview` (web) and one for `simulator`.
  final String? categoryFilter;
  const PreviewPage({super.key, this.categoryFilter});
  @override
  State<PreviewPage> createState() => _PreviewPageState();
}

class _PreviewPageState extends State<PreviewPage> with WidgetsBindingObserver {
  // ── Plugins ───────────────────────────────────────────────
  List<ProviderInfo> _plugins = [];
  String? _activePlugin;
  _Mode _mode = _Mode.webview;

  // ── Browser tabs ──────────────────────────────────────────
  final List<_BrowserTab> _tabs = [];
  int _currentTab = 0;
  static const int _maxTabs = 8;
  final TextEditingController _urlController = TextEditingController();
  final FocusNode _urlFocus = FocusNode();
  StreamSubscription<void>? _providersSub;

  // ── Simulator (WebSocket stream) ────────────────────────
  Uint8List? _screenshot;
  int _simWidth = 0;
  int _simHeight = 0;
  bool _simLoading = false;
  String? _simError;
  WebSocketChannel? _simWs;
  StreamSubscription? _simSub;

  Offset? _panStart;
  Offset? _panCurrent;

  ApiClient get _api => context.read<ApiClient>();
  String get _serverHost {
    try { return Uri.parse(_api.baseUrl).host; } catch (_) { return ''; }
  }

  ProviderInfo? get _activeInfo =>
      _plugins.where((p) => p.provider.name == _activePlugin).firstOrNull;

  _BrowserTab? get _tab =>
      (_currentTab >= 0 && _currentTab < _tabs.length) ? _tabs[_currentTab] : null;

  // ── Lifecycle ─────────────────────────────────────────────

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _newTab(load: false); // start with one empty tab
    _loadPlugins();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _loadPlugins());
  }

  @override
  void dispose() {
    _disconnectSimulator();
    for (final t in _tabs) { t.loadTimer?.cancel(); }
    _urlController.dispose();
    _urlFocus.dispose();
    _providersSub?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.paused) {
      _disconnectSimulator();
    } else if (state == AppLifecycleState.resumed) {
      final pi = _activeInfo;
      if (_mode == _Mode.simulator && pi != null) _connectSimulator(pi);
    }
  }

  // ── Plugin loading ────────────────────────────────────────

  Future<void> _loadPlugins() async {
    try {
      final all = await _api.listProviders();
      final filter = widget.categoryFilter;
      final previews = all.where((p) =>
        p.provider.type == 'panel' &&
        (filter != null
            ? p.provider.category == filter
            : (p.provider.category == 'preview' || p.provider.category == 'simulator')) &&
        p.enabled).toList();
      if (!mounted) return;
      // If the previously-active plugin was just disabled, drop it so its
      // chips/buttons disappear from the UI immediately.
      final stillActive = _activePlugin != null &&
          previews.any((p) => p.provider.name == _activePlugin);
      setState(() {
        _plugins = previews;
        if (!stillActive) {
          _activePlugin = null;
          _mode = _Mode.webview;
          _disconnectSimulator();
          _screenshot = null;
        }
      });
      if (previews.isNotEmpty && _activePlugin == null) _activate(previews.first);
    } catch (_) {}
  }

  void _activate(ProviderInfo pi) {
    _disconnectSimulator();
    final isSim = pi.provider.category == 'simulator';
    setState(() {
      _activePlugin = pi.provider.name;
      _mode = isSim ? _Mode.simulator : _Mode.webview;
      _simError = null;
    });
    if (isSim) {
      _connectSimulator(pi);
      return;
    }
    // Pre-fill the URL bar from plugin config but do NOT auto-load.
    // Auto-loading the OpenDray server's own URL (e.g. port 8640) would render
    // the OpenDray admin UI inside the WebView, stacking its bottom nav behind
    // the mobile app's nav. The user presses Go to navigate explicitly.
    final url  = (pi.config['url'] as String? ?? '').trim();
    final port = pi.config['port'];
    final host = _serverHost;
    String resolved = '';
    if (url.isNotEmpty) {
      resolved = url.startsWith('http') ? url : 'http://$url';
    } else if (port != null && host.isNotEmpty) {
      resolved = 'http://$host:$port';
    }
    final tab = _tab;
    if (resolved.isNotEmpty && tab != null && !tab.hasUrl && !_urlFocus.hasFocus) {
      _urlController.text = resolved;
      setState(() {});
    }
  }

  // ── Browser tabs ──────────────────────────────────────────

  void _newTab({bool load = true}) {
    if (_tabs.length >= _maxTabs) {
      _snack('Tab limit reached ($_maxTabs). Close a tab first.');
      return;
    }
    final tab = _BrowserTab();
    tab.controller = _buildController(tab);
    setState(() {
      _tabs.add(tab);
      _currentTab = _tabs.length - 1;
      _urlController.text = '';
    });
    if (load) _urlFocus.requestFocus();
  }

  void _closeTab(int idx) {
    if (idx < 0 || idx >= _tabs.length) return;
    if (_tabs.length == 1) {
      // Reset the last tab instead of leaving zero tabs
      setState(() {
        _tabs[0].url = '';
        _tabs[0].title = 'New Tab';
        _tabs[0].hasUrl = false;
        _urlController.text = '';
      });
      _tabs[0].controller.loadRequest(Uri.parse('about:blank'));
      return;
    }
    setState(() {
      _tabs[idx].loadTimer?.cancel();
      _tabs.removeAt(idx);
      if (_currentTab >= _tabs.length) _currentTab = _tabs.length - 1;
      if (_currentTab < 0) _currentTab = 0;
      final t = _tab;
      _urlController.text = t?.url ?? '';
    });
  }

  void _switchTab(int idx) {
    if (idx < 0 || idx >= _tabs.length || idx == _currentTab) return;
    setState(() {
      _currentTab = idx;
      _urlController.text = _tabs[idx].url;
    });
  }

  WebViewController _buildController(_BrowserTab tab) {
    return WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(NavigationDelegate(
        onPageStarted: (url) {
          tab.loadTimer?.cancel();
          tab.loadTimer = Timer(const Duration(seconds: 15), () {
            if (!mounted) return;
            tab.loading = false;
            if (tab == _tab) setState(() {});
          });
          tab.loading = true;
          tab.url = url;
          if (tab == _tab) {
            setState(() {});
            // Only rewrite the URL bar if the user is not typing.
            if (!_urlFocus.hasFocus) _urlController.text = url;
          }
          _injectFontFix(tab); // catch content rendered early
        },
        onPageFinished: (url) {
          tab.loadTimer?.cancel();
          tab.loading = false;
          tab.url = url;
          if (tab == _tab) {
            setState(() {});
            if (!_urlFocus.hasFocus) _urlController.text = url;
          }
          _injectFontFix(tab); // final pass for full DOM
          _fetchTitle(tab);
        },
        onWebResourceError: (_) {
          tab.loadTimer?.cancel();
          tab.loading = false;
          if (tab == _tab) setState(() {});
        },
        onHttpError: (_) {
          tab.loadTimer?.cancel();
          tab.loading = false;
          if (tab == _tab) setState(() {});
        },
      ));
  }

  void _loadInTab(_BrowserTab tab, String rawUrl) {
    final trimmed = rawUrl.trim();
    if (trimmed.isEmpty) return;
    final full = trimmed.startsWith('http') ? trimmed : 'http://$trimmed';
    tab.url = full;
    tab.hasUrl = true;
    tab.controller.loadRequest(Uri.parse(full));
    if (tab == _tab) {
      _urlController.text = full;
      setState(() {});
    }
  }

  void _submitUrl(String raw) {
    final tab = _tab;
    if (tab == null) return;
    _loadInTab(tab, raw);
    _urlFocus.unfocus();
  }

  // Forces a CJK-capable font stack on every element via inline `style` with
  // `!important`. Inline `style !important` beats any external CSS — including
  // external !important — so this works on pages that the previous <style>-tag
  // approach failed to override. MutationObserver only processes newly added
  // nodes (cheap) rather than re-scanning the whole DOM each tick.
  Future<void> _injectFontFix(_BrowserTab tab) async {
    const js = r'''
      (function(){
        if (window.__opendrayFontInstalled) return;
        window.__opendrayFontInstalled = true;
        var FONT = "-apple-system,BlinkMacSystemFont,system-ui,'Segoe UI',Roboto,"
                 + "'PingFang SC','PingFang TC','Hiragino Sans GB','Hiragino Sans',"
                 + "'Noto Sans CJK SC','Noto Sans CJK TC','Microsoft YaHei',"
                 + "'Source Han Sans SC','Heiti SC',sans-serif";
        var ICON = /(?:^|\s|-)(fa|fas|far|fab|fal|fad|fa-|fontawesome|lucide|material-icons|ion-icon|bi-|ti-|icon-|glyphicon|mdi)/i;
        function classOf(el){
          try {
            if (typeof el.className === 'string') return el.className;
            if (el.className && el.className.baseVal) return el.className.baseVal;
          } catch(_){}
          return '';
        }
        function apply(el){
          if (!el || el.nodeType !== 1) return;
          var tag = el.tagName;
          if (tag === 'SVG' || tag === 'PATH' || tag === 'I') return;
          var cls = classOf(el);
          if (ICON.test(cls)) return;
          try { el.style.setProperty('font-family', FONT, 'important'); } catch(_){}
        }
        function scan(root){
          if (!root) return;
          apply(root);
          var list;
          try { list = root.querySelectorAll ? root.querySelectorAll('*') : []; } catch(_){ return; }
          for (var i = 0; i < list.length; i++) apply(list[i]);
        }
        scan(document.documentElement);
        try {
          var mo = new MutationObserver(function(muts){
            for (var i = 0; i < muts.length; i++) {
              var added = muts[i].addedNodes;
              for (var j = 0; j < added.length; j++) {
                if (added[j].nodeType === 1) scan(added[j]);
              }
            }
          });
          mo.observe(document.documentElement, {childList:true, subtree:true});
        } catch(_){}
      })();
    ''';
    try { await tab.controller.runJavaScript(js); } catch (_) {}
  }

  Future<void> _fetchTitle(_BrowserTab tab) async {
    try {
      final t = await tab.controller.getTitle();
      if (!mounted) return;
      if (t != null && t.isNotEmpty) {
        tab.title = t;
        if (tab == _tab) setState(() {});
      }
    } catch (_) {}
  }

  // ── Simulator WebSocket stream ──────────────────────────

  void _connectSimulator(ProviderInfo pi) {
    _disconnectSimulator();
    setState(() { _simLoading = true; _simError = null; });

    final platform = pi.config['platform'] as String? ?? 'android';
    final device   = pi.config['device']   as String? ?? '';
    final uri = _api.simulatorStreamWsUri(platform: platform, device: device);

    final ch = WebSocketChannel.connect(uri);
    _simWs = ch;
    _simSub = ch.stream.listen(
      (data) {
        if (!mounted) return;
        if (data is List<int>) {
          // Binary frame = JPEG screenshot
          setState(() {
            _screenshot = Uint8List.fromList(data);
            _simLoading = false;
          });
        } else if (data is String) {
          // Text frame = size info or error
          try {
            final msg = Map<String, dynamic>.from(
                const JsonDecoder().convert(data) as Map);
            if (msg['type'] == 'size') {
              setState(() {
                _simWidth  = (msg['width']  as num?)?.toInt() ?? 0;
                _simHeight = (msg['height'] as num?)?.toInt() ?? 0;
              });
            }
          } catch (_) {}
        }
      },
      onError: (e) {
        if (!mounted) return;
        setState(() { _simError = e.toString(); _simLoading = false; });
      },
      onDone: () {
        if (!mounted) return;
        setState(() => _simLoading = false);
      },
    );
  }

  void _disconnectSimulator() {
    _simSub?.cancel();
    _simSub = null;
    _simWs?.sink.close();
    _simWs = null;
  }

  /// Send touch/key input directly over the simulator WebSocket.
  /// Much faster than HTTP round-trip per event.
  void _simSendInput(Map<String, dynamic> event) {
    _simWs?.sink.add(const JsonEncoder().convert(event));
  }

  /// Send input over the simulator WebSocket (zero HTTP overhead).
  void _sendSimInput({
    required String action,
    int x = 0, int y = 0,
    int x2 = 0, int y2 = 0,
    int duration = 300,
    String key = '',
    String text = '',
  }) {
    _simSendInput({
      'type': action,
      'action': action,
      'x': x, 'y': y, 'x2': x2, 'y2': y2,
      'duration': duration, 'key': key, 'text': text,
    });
  }

  (int, int) _toSimCoords(Offset pos, Rect imageRect) {
    final rx = (pos.dx - imageRect.left) / imageRect.width;
    final ry = (pos.dy - imageRect.top) / imageRect.height;
    return (
      (rx.clamp(0.0, 1.0) * _simWidth).round(),
      (ry.clamp(0.0, 1.0) * _simHeight).round(),
    );
  }

  void _handlePanEnd(Rect imageRect, String platform, String device) {
    final start   = _panStart;
    final current = _panCurrent;
    _panStart = _panCurrent = null;
    if (start == null || current == null) return;
    final dx = current.dx - start.dx;
    final dy = current.dy - start.dy;
    final dist = math.sqrt(dx * dx + dy * dy);
    final (sx, sy) = _toSimCoords(start, imageRect);
    final (ex, ey) = _toSimCoords(current, imageRect);
    final ms = (dist * 1.2).round().clamp(100, 800);
    _sendSimInput(action: 'swipe', x: sx, y: sy, x2: ex, y2: ey, duration: ms);
  }

  void _showKeyboard(String platform, String device) {
    final ctrl = TextEditingController();
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(
            bottom: MediaQuery.of(ctx).viewInsets.bottom,
            left: 16, right: 16, top: 16),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Text('Type into simulator', style: TextStyle(fontWeight: FontWeight.w600)),
          const SizedBox(height: 12),
          TextField(controller: ctrl, autofocus: true,
              decoration: const InputDecoration(hintText: 'Text to send...')),
          const SizedBox(height: 12),
          Row(children: [
            Expanded(child: OutlinedButton(
              onPressed: () { Navigator.pop(ctx); },
              child: const Text('Cancel'))),
            const SizedBox(width: 12),
            Expanded(child: FilledButton(
              style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
              onPressed: () {
                final t = ctrl.text;
                Navigator.pop(ctx);
                if (t.isNotEmpty) {
                  _sendSimInput(action: 'text', text: t);
                }
              },
              child: const Text('Send'))),
          ]),
          const SizedBox(height: 8),
        ]),
      ),
    );
  }

  // ── Build ─────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    if (_plugins.isEmpty) return _buildEmpty();
    return Column(children: [
      _buildPluginChipsBar(),
      if (_mode == _Mode.webview) ...[
        _buildTabBar(),
        _buildUrlBar(),
        if (_tab?.loading == true)
          const LinearProgressIndicator(color: AppColors.accent, minHeight: 2),
      ],
      Expanded(child: _buildBody()),
    ]);
  }

  Widget _buildBody() {
    if (_mode == _Mode.simulator) return _buildSimulator();
    final t = _tab;
    if (t == null || !t.hasUrl) return _buildWebHint();
    return IndexedStack(
      index: _currentTab,
      children: _tabs.map((x) => WebViewWidget(controller: x.controller)).toList(),
    );
  }

  // ── Plugin chips ──────────────────────────────────────────

  Widget _buildPluginChipsBar() {
    final pi = _activeInfo;
    final platform = pi?.config['platform'] as String? ?? 'ios';
    final device   = pi?.config['device']   as String? ?? '';
    final isAndroid = platform == 'android';

    return Container(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      decoration: const BoxDecoration(
          border: Border(bottom: BorderSide(color: AppColors.border))),
      child: Row(children: [
        Expanded(
          child: ListView(
            scrollDirection: Axis.horizontal,
            children: _plugins.map((p) {
              final isSim = p.provider.category == 'simulator';
              return Padding(
                padding: const EdgeInsets.only(right: 6, top: 6, bottom: 6),
                child: ChoiceChip(
                  avatar: Text(isSim ? '📱' : '🌐', style: const TextStyle(fontSize: 12)),
                  label: Text(
                      context.pickL10n(p.provider.displayName, p.provider.displayNameZh),
                      style: const TextStyle(fontSize: 11)),
                  selected: _activePlugin == p.provider.name,
                  onSelected: (_) => _activate(p),
                  selectedColor: AppColors.accentSoft,
                  backgroundColor: AppColors.surfaceAlt,
                  side: BorderSide.none,
                  padding: const EdgeInsets.symmetric(horizontal: 4),
                  visualDensity: VisualDensity.compact,
                ),
              );
            }).toList(),
          ),
        ),
        if (_mode == _Mode.simulator) ...[
          if (_simLoading)
            const SizedBox(width: 32, height: 32,
                child: Padding(padding: EdgeInsets.all(8),
                    child: CircularProgressIndicator(strokeWidth: 2, color: AppColors.accent)))
          else
            _iconBtn(Icons.refresh, tooltip: 'Capture now', onPressed: () {
              final p = _activeInfo;
              if (p != null) _connectSimulator(p);
            }),
          if (_screenshot != null)
            _iconBtn(Icons.send_outlined, color: AppColors.accent,
                tooltip: 'Send screenshot to session',
                onPressed: () => sendImageToSession(
                  context: context,
                  bytes: _screenshot!,
                  mimeType: 'image/png',
                )),
          if (isAndroid)
            _iconBtn(Icons.keyboard_alt_outlined, color: AppColors.accent,
                tooltip: 'Type text', onPressed: () => _showKeyboard(platform, device)),
        ],
      ]),
    );
  }

  // ── Tab bar ───────────────────────────────────────────────

  Widget _buildTabBar() {
    return Container(
      height: 36,
      padding: const EdgeInsets.symmetric(horizontal: 4),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        Expanded(
          child: ListView.builder(
            scrollDirection: Axis.horizontal,
            itemCount: _tabs.length,
            itemBuilder: (_, i) {
              final tab = _tabs[i];
              final active = i == _currentTab;
              final label = tab.title.isNotEmpty && tab.title != 'about:blank'
                  ? tab.title
                  : (tab.url.isNotEmpty ? _shortHost(tab.url) : 'New Tab');
              return Padding(
                padding: const EdgeInsets.symmetric(horizontal: 3, vertical: 5),
                child: Material(
                  color: active ? AppColors.accentSoft : AppColors.surfaceAlt,
                  borderRadius: BorderRadius.circular(6),
                  child: InkWell(
                    borderRadius: BorderRadius.circular(6),
                    onTap: () => _switchTab(i),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                      child: Row(mainAxisSize: MainAxisSize.min, children: [
                        if (tab.loading)
                          const Padding(
                            padding: EdgeInsets.only(right: 6),
                            child: SizedBox(width: 10, height: 10,
                              child: CircularProgressIndicator(strokeWidth: 1.5, color: AppColors.accent)),
                          )
                        else
                          Padding(
                            padding: const EdgeInsets.only(right: 6),
                            child: Icon(Icons.public, size: 11,
                                color: active ? AppColors.accent : AppColors.textMuted),
                          ),
                        ConstrainedBox(
                          constraints: const BoxConstraints(maxWidth: 110),
                          child: Text(label,
                              style: TextStyle(
                                fontSize: 11,
                                color: active ? AppColors.accent : AppColors.text,
                                fontWeight: active ? FontWeight.w500 : FontWeight.normal,
                              ),
                              maxLines: 1, overflow: TextOverflow.ellipsis),
                        ),
                        const SizedBox(width: 4),
                        InkWell(
                          onTap: () => _closeTab(i),
                          borderRadius: BorderRadius.circular(3),
                          child: Padding(
                            padding: const EdgeInsets.all(2),
                            child: Icon(Icons.close, size: 12,
                                color: active ? AppColors.accent : AppColors.textMuted),
                          ),
                        ),
                      ]),
                    ),
                  ),
                ),
              );
            },
          ),
        ),
        _iconBtn(Icons.add, color: AppColors.accent,
            tooltip: 'New tab', onPressed: _newTab),
      ]),
    );
  }

  // ── URL bar ───────────────────────────────────────────────

  Widget _buildUrlBar() {
    final tab = _tab;
    return Container(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 6),
      decoration: const BoxDecoration(
          border: Border(bottom: BorderSide(color: AppColors.border))),
      child: Row(children: [
        _iconBtn(Icons.arrow_back_ios, size: 14, onPressed: () async {
          if (tab == null) return;
          if (await tab.controller.canGoBack()) tab.controller.goBack();
        }),
        _iconBtn(Icons.arrow_forward_ios, size: 14, onPressed: () async {
          if (tab == null) return;
          if (await tab.controller.canGoForward()) tab.controller.goForward();
        }),
        _iconBtn(Icons.refresh, onPressed: () {
          if (tab?.hasUrl == true) tab!.controller.reload();
        }),
        const SizedBox(width: 4),
        Expanded(
          child: TextField(
            controller: _urlController,
            focusNode: _urlFocus,
            style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
            decoration: InputDecoration(
              isDense: true,
              contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              hintText: 'Enter URL  — e.g. localhost:3000',
              hintStyle: const TextStyle(fontSize: 12, color: AppColors.textMuted),
              filled: true,
              fillColor: AppColors.surfaceAlt,
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide.none,
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: AppColors.accent, width: 1),
              ),
              suffixIcon: _urlController.text.isEmpty ? null : IconButton(
                icon: const Icon(Icons.clear, size: 14, color: AppColors.textMuted),
                onPressed: () => setState(() => _urlController.clear()),
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(minWidth: 24, minHeight: 24),
              ),
            ),
            keyboardType: TextInputType.url,
            textInputAction: TextInputAction.go,
            autocorrect: false,
            onChanged: (_) => setState(() {}), // update clear-button visibility
            onSubmitted: _submitUrl,
          ),
        ),
      ]),
    );
  }

  Widget _iconBtn(IconData icon, {
    double size = 16, Color color = AppColors.textMuted,
    VoidCallback? onPressed, String? tooltip,
  }) => IconButton(
    icon: Icon(icon, size: size, color: color),
    onPressed: onPressed,
    padding: EdgeInsets.zero,
    constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
    tooltip: tooltip,
  );

  // ── Simulator view ────────────────────────────────────────

  Widget _buildSimulator() {
    final pi = _activeInfo;
    if (pi == null) return const SizedBox.shrink();
    final platform  = pi.config['platform'] as String? ?? 'ios';
    final device    = pi.config['device']   as String? ?? '';
    final isAndroid = platform == 'android';

    if (_simError != null && _screenshot == null) {
      return _buildSimError(isAndroid: isAndroid);
    }
    if (_screenshot == null) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }

    return Column(children: [
      Expanded(
        child: LayoutBuilder(builder: (ctx, constraints) {
          final containerSize = Size(constraints.maxWidth, constraints.maxHeight);
          final imageRect = _simWidth > 0 && _simHeight > 0
              ? _fitRect(containerSize, _simWidth / _simHeight)
              : Rect.fromLTWH(0, 0, containerSize.width, containerSize.height);

          return GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTapUp: isAndroid
                ? (d) {
                    final (sx, sy) = _toSimCoords(d.localPosition, imageRect);
                    _sendSimInput(action: 'tap', x: sx, y: sy);
                  }
                : null,
            onPanStart: isAndroid
                ? (d) { _panStart = d.localPosition; _panCurrent = d.localPosition; }
                : null,
            onPanUpdate: isAndroid
                ? (d) { _panCurrent = d.localPosition; }
                : null,
            onPanEnd: isAndroid
                ? (_) => _handlePanEnd(imageRect, platform, device)
                : null,
            child: Stack(children: [
              Center(
                child: Image.memory(
                  _screenshot!,
                  width: constraints.maxWidth,
                  height: constraints.maxHeight,
                  fit: BoxFit.contain,
                  gaplessPlayback: true,
                ),
              ),
              if (_simLoading)
                const Positioned(top: 8, right: 8,
                  child: SizedBox(width: 16, height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2, color: AppColors.accent))),
              if (isAndroid && _simError == null)
                Positioned(top: 8, left: 8,
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                    decoration: BoxDecoration(
                      color: Colors.black54,
                      borderRadius: BorderRadius.circular(4)),
                    child: const Text('Touch to interact',
                        style: TextStyle(color: Colors.white, fontSize: 9)),
                  )),
            ]),
          );
        }),
      ),
      if (isAndroid) _buildAndroidButtons(platform, device),
    ]);
  }

  Widget _buildAndroidButtons(String platform, String device) {
    sendKey(String key) => _sendSimInput(action: 'keyevent', key: key);
    return Container(
      height: 48,
      decoration: const BoxDecoration(
          border: Border(top: BorderSide(color: AppColors.border))),
      child: Row(mainAxisAlignment: MainAxisAlignment.spaceEvenly, children: [
        _hwBtn(Icons.arrow_back, 'Back', () => sendKey('KEYCODE_BACK')),
        _hwBtn(Icons.circle_outlined, 'Home', () => sendKey('KEYCODE_HOME')),
        _hwBtn(Icons.apps, 'Recents', () => sendKey('KEYCODE_APP_SWITCH')),
        _hwBtn(Icons.volume_down, 'Vol-', () => sendKey('KEYCODE_VOLUME_DOWN')),
        _hwBtn(Icons.volume_up, 'Vol+', () => sendKey('KEYCODE_VOLUME_UP')),
      ]),
    );
  }

  Widget _hwBtn(IconData icon, String label, VoidCallback onPressed) {
    return InkWell(
      onTap: onPressed,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(icon, size: 20, color: AppColors.textMuted),
          const SizedBox(height: 2),
          Text(label, style: const TextStyle(fontSize: 9, color: AppColors.textMuted)),
        ]),
      ),
    );
  }

  Widget _buildSimError({required bool isAndroid}) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.error_outline, size: 40, color: AppColors.error),
          const SizedBox(height: 12),
          Text(_simError!, style: const TextStyle(color: AppColors.error, fontSize: 12),
              textAlign: TextAlign.center),
          const SizedBox(height: 16),
          Text(
            isAndroid
                ? 'Start the Android emulator:\n  flutter run -d emulator-5554'
                : 'Start the iOS simulator:\n  flutter run -d "iPhone 16"',
            style: const TextStyle(color: AppColors.textMuted, fontSize: 11),
            textAlign: TextAlign.center,
          ),
        ]),
      ),
    );
  }

  // ── Helpers ───────────────────────────────────────────────

  Rect _fitRect(Size container, double imageAspect) {
    final containerAspect = container.width / container.height;
    double w, h;
    if (imageAspect > containerAspect) {
      w = container.width;  h = w / imageAspect;
    } else {
      h = container.height; w = h * imageAspect;
    }
    return Rect.fromLTWH((container.width - w) / 2, (container.height - h) / 2, w, h);
  }

  String _shortHost(String url) {
    try {
      final u = Uri.parse(url);
      final host = u.host;
      return u.port != 0 && u.port != 80 && u.port != 443 ? '$host:${u.port}' : host;
    } catch (_) { return url; }
  }

  void _snack(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(msg), duration: const Duration(seconds: 2)));
  }

  // ── Empty states ──────────────────────────────────────────

  Widget _buildEmpty() {
    final filter = widget.categoryFilter;
    final showSim = filter == null || filter == 'simulator';
    final showWeb = filter == null || filter == 'preview';
    final icon = filter == 'preview' ? Icons.web : Icons.phone_iphone;
    final title = filter == 'preview'
        ? context.tr('No web browser plugin configured')
        : filter == 'simulator'
            ? context.tr('No simulator plugin configured')
            : context.tr('No preview plugins configured');
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(icon, size: 48, color: AppColors.textMuted),
          const SizedBox(height: 16),
          Text(title,
              textAlign: TextAlign.center,
              style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 15)),
          const SizedBox(height: 14),
          if (showSim)
            _HintCard(
              icon: Icons.phone_iphone,
              title: 'Simulator Preview (📱)',
              body: 'Register simulator-preview → platform: android → device: emulator-5554. '
                  'Touch the screenshot to interact — taps and swipes are forwarded via ADB.',
            ),
          if (showSim && showWeb) const SizedBox(height: 8),
          if (showWeb)
            _HintCard(
              icon: Icons.web,
              title: 'Web Browser (🌐)',
              body: 'Install web-browser → open multiple tabs, type any URL into the address bar.',
            ),
        ]),
      ),
    );
  }

  Widget _buildWebHint() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.public, size: 40, color: AppColors.textMuted),
          const SizedBox(height: 12),
          const Text('Type a URL above and press Go',
              style: TextStyle(color: AppColors.text, fontSize: 14, fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          const Text('Open multiple tabs with the + button.',
              style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
          const SizedBox(height: 20),
          Wrap(spacing: 8, runSpacing: 8, alignment: WrapAlignment.center, children: [
            _QuickUrl(label: 'localhost:3000', onTap: () => _submitUrl('http://localhost:3000')),
            _QuickUrl(label: 'localhost:5173', onTap: () => _submitUrl('http://localhost:5173')),
            _QuickUrl(label: 'localhost:8080', onTap: () => _submitUrl('http://localhost:8080')),
            if (_serverHost.isNotEmpty)
              _QuickUrl(label: '$_serverHost:3000', onTap: () => _submitUrl('http://$_serverHost:3000')),
          ]),
        ]),
      ),
    );
  }
}

// ── Types ─────────────────────────────────────────────────────────

class _BrowserTab {
  late WebViewController controller;
  String url = '';
  String title = 'New Tab';
  bool loading = false;
  bool hasUrl = false;
  Timer? loadTimer;
}

class _QuickUrl extends StatelessWidget {
  final String label;
  final VoidCallback onTap;
  const _QuickUrl({required this.label, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surfaceAlt,
      borderRadius: BorderRadius.circular(20),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(20),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            const Icon(Icons.link, size: 12, color: AppColors.accent),
            const SizedBox(width: 6),
            Text(label, style: const TextStyle(fontSize: 11, fontFamily: 'monospace')),
          ]),
        ),
      ),
    );
  }
}

// ── Hint card ─────────────────────────────────────────────────────

class _HintCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final String body;
  const _HintCard({required this.icon, required this.title, required this.body});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.surfaceAlt,
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Icon(icon, size: 18, color: AppColors.accent),
        const SizedBox(width: 10),
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(title, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
          const SizedBox(height: 3),
          Text(body, style: const TextStyle(fontSize: 11, color: AppColors.textMuted, height: 1.4)),
        ])),
      ]),
    );
  }
}
