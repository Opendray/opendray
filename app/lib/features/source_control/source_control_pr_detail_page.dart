import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/theme/app_theme.dart';
import 'widgets/multi_file_diff_view.dart';

/// PR detail + diff + comments for the unified Source Control plugin.
/// Mirrors the legacy git-forge detail page but calls the new
/// /forges/{id}/pulls/{number}/* routes, which take ?repo=owner/name
/// so one forge instance can answer across many repos.
class SourceControlPrDetailPage extends StatefulWidget {
  const SourceControlPrDetailPage({
    super.key,
    required this.pluginName,
    required this.forgeId,
    required this.repo,
    required this.initial,
  });

  final String pluginName;
  final String forgeId;
  final String repo;
  final Map<String, dynamic> initial;

  @override
  State<SourceControlPrDetailPage> createState() =>
      _SourceControlPrDetailPageState();
}

class _SourceControlPrDetailPageState extends State<SourceControlPrDetailPage>
    with SingleTickerProviderStateMixin {
  late Map<String, dynamic> _pr;
  late final TabController _tabs;

  List<Map<String, dynamic>>? _diff;
  bool _diffLoading = false;
  String? _diffError;

  List<Map<String, dynamic>>? _comments;
  bool _commentsLoading = false;
  String? _commentsError;

  List<Map<String, dynamic>> _reviews = const [];
  List<Map<String, dynamic>> _checks = const [];
  List<Map<String, dynamic>> _reviewComments = const [];

  ApiClient get _api => context.read<ApiClient>();
  int get _number => (_pr['number'] as num?)?.toInt() ?? 0;

  @override
  void initState() {
    super.initState();
    _pr = Map<String, dynamic>.from(widget.initial);
    _tabs = TabController(length: 2, vsync: this)..addListener(_onTabChanged);
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
      final pr = await ApiClient.describeErrors(() => _api.scPullDetail(
          widget.pluginName, widget.forgeId, _number,
          repo: widget.repo));
      if (!mounted) return;
      setState(() => _pr = pr);
    } on ApiException catch (_) {
      // Keep the list-row seed when detail fails; the tabs below
      // continue loading independently.
    }
    if (!mounted) return;
    unawaited(_loadReviews());
    unawaited(_loadChecks());
    unawaited(_loadReviewComments());
    _loadDiff();
  }

  Future<void> _loadReviews() async {
    try {
      final r = await ApiClient.describeErrors(() => _api.scPullReviews(
          widget.pluginName, widget.forgeId, _number,
          repo: widget.repo));
      if (mounted) setState(() => _reviews = r);
    } on ApiException catch (_) {/* silent */}
  }

  Future<void> _loadChecks() async {
    try {
      final c = await ApiClient.describeErrors(() => _api.scPullChecks(
          widget.pluginName, widget.forgeId, _number,
          repo: widget.repo));
      if (mounted) setState(() => _checks = c);
    } on ApiException catch (_) {/* silent */}
  }

  Future<void> _loadReviewComments() async {
    try {
      final rc =
          await ApiClient.describeErrors(() => _api.scPullReviewComments(
              widget.pluginName, widget.forgeId, _number,
              repo: widget.repo));
      if (mounted) setState(() => _reviewComments = rc);
    } on ApiException catch (_) {/* silent */}
  }

  Future<void> _loadDiff() async {
    setState(() { _diffLoading = true; _diffError = null; });
    try {
      final files = await ApiClient.describeErrors(() => _api.scPullDiff(
          widget.pluginName, widget.forgeId, _number,
          repo: widget.repo));
      if (!mounted) return;
      setState(() { _diff = files; _diffLoading = false; });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() { _diffLoading = false; _diffError = e.message; });
    }
  }

  Future<void> _loadComments() async {
    setState(() { _commentsLoading = true; _commentsError = null; });
    try {
      final cs = await ApiClient.describeErrors(() => _api.scPullComments(
          widget.pluginName, widget.forgeId, _number,
          repo: widget.repo));
      if (!mounted) return;
      setState(() { _comments = cs; _commentsLoading = false; });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() { _commentsLoading = false; _commentsError = e.message; });
    }
  }

  // ── UI ──────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final title = (_pr['title'] as String?) ?? '';
    final url = (_pr['url'] as String?) ?? '';
    return Scaffold(
      appBar: AppBar(
        title: Text('#$_number · ${title.isEmpty ? "—" : title}',
            overflow: TextOverflow.ellipsis),
        actions: [
          IconButton(
            icon: const Icon(Icons.auto_awesome, size: 20),
            tooltip: context.tr('Explain this PR'),
            onPressed: () => _aiHandoff(reviewMode: false),
          ),
          IconButton(
            icon: const Icon(Icons.rate_review_outlined, size: 20),
            tooltip: context.tr('Review this diff'),
            onPressed: () => _aiHandoff(reviewMode: true),
          ),
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
            children: [_diffTab(), _commentsTab()],
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
          if (_reviews.isNotEmpty) ...[
            const SizedBox(height: 8),
            _reviewsStrip(),
          ],
          if (_checks.isNotEmpty) ...[
            const SizedBox(height: 8),
            _checksStrip(),
          ],
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

  /// Aggregates per-reviewer verdicts into approved / changes-requested /
  /// commented counters so the reader can tell at a glance whether a
  /// PR is ready-to-merge. We keep the latest state per reviewer since
  /// someone who commented then approved should count as approved once.
  Widget _reviewsStrip() {
    final latest = <String, String>{};
    for (final r in _reviews) {
      final author = (r['author'] as String?) ?? '';
      final state = (r['state'] as String?) ?? 'commented';
      if (author.isEmpty) continue;
      latest[author] = state;
    }
    var approved = 0, changes = 0, commented = 0;
    for (final s in latest.values) {
      switch (s) {
        case 'approved':
          approved++;
          break;
        case 'changes_requested':
          changes++;
          break;
        case 'commented':
          commented++;
          break;
      }
    }
    return Wrap(spacing: 6, runSpacing: 4, children: [
      if (approved > 0)
        _badge('✓ $approved ${context.tr('approved')}', AppColors.success),
      if (changes > 0)
        _badge('✗ $changes ${context.tr('changes')}', AppColors.error),
      if (commented > 0)
        _badge('💬 $commented ${context.tr('commented')}',
            AppColors.textMuted),
    ]);
  }

  Widget _checksStrip() {
    return Wrap(spacing: 6, runSpacing: 4, children: [
      for (final c in _checks) _checkBadge(c),
    ]);
  }

  Widget _checkBadge(Map<String, dynamic> c) {
    final status = (c['status'] as String?) ?? 'pending';
    final name = (c['name'] as String?) ?? '';
    final url = (c['targetUrl'] as String?) ?? '';
    final (icon, color) = switch (status) {
      'success' => (Icons.check_circle, AppColors.success),
      'failure' => (Icons.cancel, AppColors.error),
      'skipped' => (Icons.remove_circle_outline, AppColors.textMuted),
      _         => (Icons.access_time, AppColors.warning),
    };
    final badge = Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Icon(icon, size: 12, color: color),
        const SizedBox(width: 4),
        Text(name.isEmpty ? status : name,
            style: TextStyle(color: color, fontSize: 10)),
      ]),
    );
    if (url.isEmpty) return badge;
    return InkWell(
      borderRadius: BorderRadius.circular(10),
      onTap: () =>
          launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication),
      child: badge,
    );
  }

  Widget _badge(String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text(text,
          style: TextStyle(
              color: color, fontSize: 10, fontWeight: FontWeight.w600)),
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

  // ── Diff tab ────────────────────────────────────────────────

  Widget _diffTab() {
    if (_diffLoading && _diff == null) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_diffError != null && _diff == null) {
      return _inlineError(_diffError!, _loadDiff);
    }
    final files = _diff ?? const [];
    if (_reviewComments.isEmpty) {
      return MultiFileDiffView(files: files);
    }
    // When there are inline review comments, show them under each
    // file. We can't feed that into the shared MultiFileDiffView
    // without expanding its contract; drop to a bespoke list here
    // that reuses the shared card via a trailing block.
    return ListView.separated(
      itemCount: files.length,
      separatorBuilder: (_, _) => const SizedBox(height: 6),
      itemBuilder: (_, i) {
        final f = files[i];
        final path = (f['path'] as String?) ?? '';
        final inline = _reviewComments
            .where((c) => (c['path'] as String?) == path)
            .toList();
        return Column(children: [
          // Reuse a single-file view by wrapping a one-item list.
          MultiFileDiffView(files: [f]),
          if (inline.isNotEmpty) _inlineCommentsBlock(inline),
        ]);
      },
    );
  }

  Widget _inlineCommentsBlock(List<Map<String, dynamic>> cs) {
    final byLine = <int, List<Map<String, dynamic>>>{};
    for (final c in cs) {
      final line = (c['line'] as num?)?.toInt() ?? 0;
      byLine.putIfAbsent(line, () => []).add(c);
    }
    final sortedLines = byLine.keys.toList()..sort();
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 0, 12, 8),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.06),
        borderRadius: BorderRadius.circular(6),
        border:
            Border.all(color: AppColors.accent.withValues(alpha: 0.2)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final line in sortedLines) ...[
            Padding(
              padding: const EdgeInsets.only(top: 6, bottom: 2),
              child: Text(
                line > 0 ? 'Line $line' : context.tr('Context comment'),
                style: const TextStyle(
                    color: AppColors.accent,
                    fontSize: 10,
                    fontWeight: FontWeight.w600),
              ),
            ),
            for (final c in byLine[line]!) _inlineCommentRow(c),
          ],
        ],
      ),
    );
  }

  Widget _inlineCommentRow(Map<String, dynamic> c) {
    return Padding(
      padding: const EdgeInsets.only(left: 8, top: 2, bottom: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text((c['author'] as String?) ?? '',
              style: const TextStyle(
                  color: AppColors.accent,
                  fontWeight: FontWeight.w600,
                  fontSize: 11)),
          const SizedBox(height: 2),
          SelectableText((c['body'] as String?) ?? '',
              style: const TextStyle(fontSize: 12, height: 1.35)),
        ],
      ),
    );
  }

  // ── Comments tab ────────────────────────────────────────────

  Widget _commentsTab() {
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
              style:
                  const TextStyle(color: AppColors.error, fontSize: 12)),
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

  /// Copies the PR's diff plus a preset Claude prompt to the clipboard
  /// and navigates to the dashboard so the user can paste into a fresh
  /// Claude session. Deliberately dumb: no bridge RPC, no auto-open —
  /// the clipboard is the universal hand-off.
  Future<void> _aiHandoff({required bool reviewMode}) async {
    final title = (_pr['title'] as String?) ?? '';
    final number = _number;
    final body = ((_pr['body'] as String?) ?? '').trim();
    final files = _diff ?? const [];
    final buf = StringBuffer();
    final instruction = reviewMode
        ? 'Code review this diff. Flag anything risky, confusing, or '
            'missing. Be specific — cite file paths and line numbers. '
            'Note security / performance / correctness concerns. If '
            'the change looks fine, say so briefly.'
        : 'Summarise this pull request. Lead with what it changes '
            'and why. Call out the parts that most need reviewer '
            'attention. Stay concise — two short paragraphs.';
    buf.writeln(instruction);
    buf.writeln();
    buf.writeln('--- PR #$number: $title ---');
    if (body.isNotEmpty) {
      buf.writeln();
      buf.writeln('Description:');
      buf.writeln(body);
    }
    buf.writeln();
    buf.writeln('Diff:');
    for (final f in files) {
      final patch = (f['patch'] as String?) ?? '';
      if (patch.isEmpty) continue;
      buf.writeln(patch);
    }
    await Clipboard.setData(ClipboardData(text: buf.toString()));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(context.tr(
          'Diff copied — start a Claude session on the dashboard and paste.')),
      duration: const Duration(seconds: 4),
    ));
    GoRouter.of(context).go('/');
  }
}
