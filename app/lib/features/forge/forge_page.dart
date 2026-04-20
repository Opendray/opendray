import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Forge panel — read-only Pull Request viewer for the git-forge
/// plugin. Same three-state shell as the git-viewer panel:
///
///   1. No git-forge plugin enabled → setup CTA pointing at /plugins.
///   2. Plugin enabled but missing repo / baseUrl → "go configure" CTA.
///   3. Configured → list with state filter; tap to open detail +
///      diff + comments.
///
/// The panel is observer-only. Creating PRs, merging, approving, and
/// commenting happen through the Claude session workflow, not here.
class ForgePage extends StatefulWidget {
  const ForgePage({super.key});
  @override
  State<ForgePage> createState() => _ForgePageState();
}

class _ForgePageState extends State<ForgePage> {
  ProviderInfo? _plugin;
  StreamSubscription<void>? _providersSub;

  String _stateFilter = 'open';
  List<Map<String, dynamic>> _prs = [];
  bool _loading = false;
  String? _error;

  ApiClient get _api => context.read<ApiClient>();
  String? get _pluginName => _plugin?.provider.name;

  @override
  void initState() {
    super.initState();
    _loadPlugin();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _loadPlugin());
  }

  @override
  void dispose() {
    _providersSub?.cancel();
    super.dispose();
  }

  // ─── Plugin / state boot ─────────────────────────────────

  Future<void> _loadPlugin() async {
    try {
      final all = await _api.listProviders();
      final match = all.where((p) =>
          p.provider.type == 'panel' &&
          p.provider.name == 'git-forge' &&
          p.enabled).toList();
      if (!mounted) return;
      if (match.isEmpty) {
        setState(() {
          _plugin = null;
          _prs = [];
        });
        return;
      }
      final p = match.first;
      setState(() {
        _plugin = p;
        // Seed the filter from the plugin's configured default (only
        // if the user hasn't already picked one this session).
        final defaultState =
            (p.config['defaultState'] as String?)?.trim() ?? '';
        if (defaultState.isNotEmpty) {
          _stateFilter = defaultState;
        }
      });
      await _loadPulls();
    } catch (_) {
      // _loadPulls handles error surfacing; a list-providers failure
      // just leaves the setup CTA visible.
    }
  }

  Future<void> _loadPulls() async {
    final name = _pluginName;
    if (name == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final prs = await ApiClient.describeErrors(
          () => _api.forgePulls(name, state: _stateFilter));
      if (!mounted) return;
      setState(() {
        _prs = prs;
        _loading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = e.message;
      });
    }
  }

  // ─── UI ──────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    if (_plugin == null) return _setupCta();
    return Column(children: [
      _filterBar(),
      if (_error != null) _errorBar(_error!),
      Expanded(child: _listView()),
    ]);
  }

  Widget _setupCta() {
    return _centered(
      icon: Icons.extension_off,
      title: context.tr('Git Forge plugin not enabled'),
      subtitle: context.tr(
          'Install git-forge from the Hub and configure forgeType + baseUrl + repo in Plugins → Configure.'),
    );
  }

  Widget _filterBar() {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        Text(context.tr('State'),
            style: const TextStyle(
                color: AppColors.textMuted, fontSize: 12)),
        const SizedBox(width: 8),
        for (final s in const ['open', 'closed', 'all']) _stateChip(s),
        const Spacer(),
        IconButton(
          icon: const Icon(Icons.refresh, size: 18),
          tooltip: context.tr('Refresh'),
          onPressed: _loading ? null : _loadPulls,
        ),
      ]),
    );
  }

  Widget _stateChip(String value) {
    final selected = _stateFilter == value;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 2),
      child: ChoiceChip(
        label: Text(context.tr(value)),
        selected: selected,
        onSelected: (_) {
          if (!selected) {
            setState(() => _stateFilter = value);
            _loadPulls();
          }
        },
        selectedColor: AppColors.accent.withValues(alpha: 0.16),
        labelStyle: TextStyle(
          fontSize: 12,
          color: selected ? AppColors.accent : AppColors.text,
          fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(14),
          side: BorderSide(
              color: selected ? AppColors.accent : AppColors.border),
        ),
      ),
    );
  }

  Widget _errorBar(String msg) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      color: AppColors.errorSoft,
      child: Row(children: [
        const Icon(Icons.error_outline, color: AppColors.error, size: 16),
        const SizedBox(width: 8),
        Expanded(
            child: Text(msg,
                style:
                    const TextStyle(color: AppColors.error, fontSize: 12))),
      ]),
    );
  }

  Widget _listView() {
    if (_loading && _prs.isEmpty) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_prs.isEmpty) {
      return _centered(
        icon: Icons.merge_type,
        title: context.tr('No pull requests'),
        subtitle: context.tr(
            'Change the state filter above or confirm the repo has PRs.'),
      );
    }
    return RefreshIndicator(
      onRefresh: _loadPulls,
      child: ListView.separated(
        itemCount: _prs.length,
        separatorBuilder: (_, _) =>
            const Divider(height: 1, color: AppColors.border),
        itemBuilder: (_, i) => _prRow(_prs[i]),
      ),
    );
  }

  Widget _prRow(Map<String, dynamic> pr) {
    final number = (pr['number'] as num?)?.toInt() ?? 0;
    final title = (pr['title'] as String?) ?? '';
    final state = (pr['state'] as String?) ?? 'open';
    final author = (pr['author'] as String?) ?? '';
    final comments = (pr['commentCount'] as num?)?.toInt() ?? 0;
    final draft = pr['draft'] == true;
    final head = (pr['headRef'] as String?) ?? '';
    final base = (pr['baseRef'] as String?) ?? '';
    return ListTile(
      dense: true,
      onTap: () => _openDetail(pr),
      leading: _stateIcon(state, draft: draft),
      title: Row(children: [
        Text('#$number',
            style: const TextStyle(
                color: AppColors.textMuted,
                fontSize: 12,
                fontFeatures: [FontFeature.tabularFigures()])),
        const SizedBox(width: 8),
        Expanded(
            child: Text(title,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                    fontWeight: FontWeight.w500, fontSize: 14))),
      ]),
      subtitle: Text(
        '$author · $head → $base${comments > 0 ? '  •  $comments ${context.tr('comments')}' : ''}',
        style: const TextStyle(color: AppColors.textMuted, fontSize: 11),
        overflow: TextOverflow.ellipsis,
      ),
    );
  }

  Widget _stateIcon(String state, {bool draft = false}) {
    if (draft) {
      return const Icon(Icons.edit_note,
          color: AppColors.textMuted, size: 20);
    }
    final (icon, color) = switch (state) {
      'open'   => (Icons.merge_type, AppColors.success),
      'merged' => (Icons.call_merge, AppColors.accent),
      'closed' => (Icons.cancel_outlined, AppColors.error),
      _        => (Icons.circle_outlined, AppColors.textMuted),
    };
    return Icon(icon, color: color, size: 20);
  }

  Widget _centered({
    required IconData icon,
    required String title,
    String? subtitle,
  }) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(icon, size: 44, color: AppColors.textMuted),
          const SizedBox(height: 12),
          Text(title,
              textAlign: TextAlign.center,
              style:
                  const TextStyle(fontWeight: FontWeight.w500, fontSize: 15)),
          if (subtitle != null) ...[
            const SizedBox(height: 8),
            Text(subtitle,
                textAlign: TextAlign.center,
                style:
                    const TextStyle(color: AppColors.textMuted, fontSize: 12)),
          ],
        ]),
      ),
    );
  }

  // ─── Detail page dispatch ────────────────────────────────

  void _openDetail(Map<String, dynamic> pr) {
    final number = (pr['number'] as num?)?.toInt() ?? 0;
    if (number <= 0 || _pluginName == null) return;
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => _ForgeDetailPage(
        pluginName: _pluginName!,
        initial: pr,
      ),
    ));
  }
}

/// Detail + diff + comments for one PR. Fetched on first mount; data
/// below the summary loads lazily as the user switches tabs.
class _ForgeDetailPage extends StatefulWidget {
  const _ForgeDetailPage({
    required this.pluginName,
    required this.initial,
  });

  final String pluginName;
  final Map<String, dynamic> initial;

  @override
  State<_ForgeDetailPage> createState() => _ForgeDetailPageState();
}

class _ForgeDetailPageState extends State<_ForgeDetailPage>
    with SingleTickerProviderStateMixin {
  late Map<String, dynamic> _pr;
  late final TabController _tabs;

  List<Map<String, dynamic>>? _diff;
  bool _diffLoading = false;
  String? _diffError;

  List<Map<String, dynamic>>? _comments;
  bool _commentsLoading = false;
  String? _commentsError;

  ApiClient get _api => context.read<ApiClient>();

  int get _number => (_pr['number'] as num?)?.toInt() ?? 0;

  @override
  void initState() {
    super.initState();
    _pr = Map<String, dynamic>.from(widget.initial);
    _tabs = TabController(length: 2, vsync: this)
      ..addListener(_onTabChanged);
    _loadDetail();
  }

  @override
  void dispose() {
    _tabs
      ..removeListener(_onTabChanged)
      ..dispose();
    super.dispose();
  }

  void _onTabChanged() {
    if (_tabs.indexIsChanging) return;
    if (_tabs.index == 0 && _diff == null) _loadDiff();
    if (_tabs.index == 1 && _comments == null) _loadComments();
  }

  Future<void> _loadDetail() async {
    try {
      final pr = await ApiClient.describeErrors(
          () => _api.forgePullDetail(widget.pluginName, _number));
      if (!mounted) return;
      setState(() => _pr = pr);
    } on ApiException catch (_) {
      // Keep the list-view seed row rather than clobber with an
      // empty shell; the tabs below still load independently.
    }
    if (mounted) _loadDiff();
  }

  Future<void> _loadDiff() async {
    setState(() {
      _diffLoading = true;
      _diffError = null;
    });
    try {
      final files = await ApiClient.describeErrors(
          () => _api.forgePullDiff(widget.pluginName, _number));
      if (!mounted) return;
      setState(() {
        _diff = files;
        _diffLoading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _diffLoading = false;
        _diffError = e.message;
      });
    }
  }

  Future<void> _loadComments() async {
    setState(() {
      _commentsLoading = true;
      _commentsError = null;
    });
    try {
      final cs = await ApiClient.describeErrors(
          () => _api.forgePullComments(widget.pluginName, _number));
      if (!mounted) return;
      setState(() {
        _comments = cs;
        _commentsLoading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _commentsLoading = false;
        _commentsError = e.message;
      });
    }
  }

  // ─── UI ──────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final title = (_pr['title'] as String?) ?? '';
    final url = (_pr['url'] as String?) ?? '';
    return Scaffold(
      appBar: AppBar(
        title: Text('#$_number · ${title.isEmpty ? "—" : title}',
            overflow: TextOverflow.ellipsis),
        actions: [
          if (url.isNotEmpty)
            IconButton(
              icon: const Icon(Icons.open_in_browser, size: 20),
              tooltip: context.tr('Open in browser'),
              onPressed: () => launchUrl(Uri.parse(url),
                  mode: LaunchMode.externalApplication),
            ),
        ],
      ),
      body: Column(children: [
        _summary(),
        TabBar(
          controller: _tabs,
          labelColor: AppColors.accent,
          unselectedLabelColor: AppColors.textMuted,
          indicatorColor: AppColors.accent,
          tabs: [
            Tab(text: context.tr('Diff')),
            Tab(text: context.tr('Comments')),
          ],
        ),
        Expanded(
          child: TabBarView(
            controller: _tabs,
            children: [_diffView(), _commentsView()],
          ),
        ),
      ]),
    );
  }

  Widget _summary() {
    final state = (_pr['state'] as String?) ?? 'open';
    final author = (_pr['author'] as String?) ?? '';
    final head = (_pr['headRef'] as String?) ?? '';
    final base = (_pr['baseRef'] as String?) ?? '';
    final body = ((_pr['body'] as String?) ?? '').trim();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(14, 10, 14, 10),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            _stateBadge(state, draft: _pr['draft'] == true),
            const SizedBox(width: 8),
            Expanded(
                child: Text('$author · $head → $base',
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                        color: AppColors.textMuted, fontSize: 12))),
          ]),
          if (body.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(body,
                maxLines: 4,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 12)),
          ],
        ],
      ),
    );
  }

  Widget _stateBadge(String state, {bool draft = false}) {
    final (label, color) = draft
        ? ('draft', AppColors.textMuted)
        : switch (state) {
            'open'   => ('open',   AppColors.success),
            'merged' => ('merged', AppColors.accent),
            'closed' => ('closed', AppColors.error),
            _        => (state,    AppColors.textMuted),
          };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text(label,
          style: TextStyle(
              color: color, fontSize: 11, fontWeight: FontWeight.w600)),
    );
  }

  // ── Diff tab ─────────────────────────────────────────────

  Widget _diffView() {
    if (_diffLoading && _diff == null) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_diffError != null && _diff == null) {
      return _inlineError(_diffError!, _loadDiff);
    }
    final files = _diff ?? const [];
    if (files.isEmpty) {
      return Center(
        child: Text(context.tr('No diff'),
            style:
                const TextStyle(color: AppColors.textMuted, fontSize: 12)),
      );
    }
    return ListView.separated(
      itemCount: files.length,
      separatorBuilder: (_, _) => const SizedBox(height: 6),
      itemBuilder: (_, i) => _diffFileCard(files[i]),
    );
  }

  Widget _diffFileCard(Map<String, dynamic> f) {
    final path = (f['path'] as String?) ?? '';
    final oldPath = ((f['oldPath'] as String?) ?? '').trim();
    final status = (f['status'] as String?) ?? 'modified';
    final adds = (f['additions'] as num?)?.toInt() ?? 0;
    final dels = (f['deletions'] as num?)?.toInt() ?? 0;
    final patch = (f['patch'] as String?) ?? '';
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      child: ExpansionTile(
        tilePadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 0),
        childrenPadding: const EdgeInsets.symmetric(horizontal: 10),
        title: Row(children: [
          _diffStatusChip(status),
          const SizedBox(width: 8),
          Expanded(
              child: Text(
                  oldPath.isNotEmpty ? '$oldPath → $path' : path,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                      fontFamily: 'monospace', fontSize: 12))),
          if (adds > 0)
            Padding(
              padding: const EdgeInsets.only(left: 6),
              child: Text('+$adds',
                  style: const TextStyle(
                      color: AppColors.success, fontSize: 11)),
            ),
          if (dels > 0)
            Padding(
              padding: const EdgeInsets.only(left: 4),
              child: Text('-$dels',
                  style: const TextStyle(
                      color: AppColors.error, fontSize: 11)),
            ),
        ]),
        children: [
          Container(
            alignment: Alignment.topLeft,
            padding: const EdgeInsets.all(8),
            child: SelectableText(
              patch.isEmpty ? '(no patch body)' : patch,
              style: const TextStyle(
                  fontFamily: 'monospace', fontSize: 11, height: 1.35),
            ),
          ),
        ],
      ),
    );
  }

  Widget _diffStatusChip(String status) {
    final color = switch (status) {
      'added'    => AppColors.success,
      'deleted'  => AppColors.error,
      'renamed'  => AppColors.accent,
      _          => AppColors.textMuted,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(status,
          style: TextStyle(
              color: color, fontSize: 10, fontWeight: FontWeight.w600)),
    );
  }

  // ── Comments tab ─────────────────────────────────────────

  Widget _commentsView() {
    if (_commentsLoading && _comments == null) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_commentsError != null && _comments == null) {
      return _inlineError(_commentsError!, _loadComments);
    }
    final cs = _comments ?? const [];
    if (cs.isEmpty) {
      return Center(
        child: Text(context.tr('No comments'),
            style:
                const TextStyle(color: AppColors.textMuted, fontSize: 12)),
      );
    }
    return ListView.separated(
      itemCount: cs.length,
      separatorBuilder: (_, _) =>
          const Divider(height: 1, color: AppColors.border),
      itemBuilder: (_, i) => _commentRow(cs[i]),
    );
  }

  Widget _commentRow(Map<String, dynamic> c) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text((c['author'] as String?) ?? '',
              style: const TextStyle(
                  color: AppColors.accent,
                  fontWeight: FontWeight.w600,
                  fontSize: 12)),
          const SizedBox(height: 4),
          SelectableText((c['body'] as String?) ?? '',
              style: const TextStyle(fontSize: 13, height: 1.4)),
        ],
      ),
    );
  }

  Widget _inlineError(String msg, Future<void> Function() retry) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.error_outline,
              color: AppColors.error, size: 32),
          const SizedBox(height: 10),
          Text(msg,
              textAlign: TextAlign.center,
              style: const TextStyle(color: AppColors.error, fontSize: 12)),
          const SizedBox(height: 10),
          OutlinedButton.icon(
            onPressed: retry,
            icon: const Icon(Icons.refresh, size: 16),
            label: Text(context.tr('Retry')),
          ),
        ]),
      ),
    );
  }
}
