import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';
import 'source_control_commit_diff_page.dart';
import 'source_control_pr_detail_page.dart';
import 'widgets/forge_selector.dart';
import 'widgets/multi_file_diff_view.dart';
import 'widgets/repo_selector.dart';

/// Unified Source Control panel — the merged successor to the separate
/// git-viewer (local repo) and git-forge (PR viewer) panels. Four tabs:
///
///   • Changes   — multi-file diff with unstaged / staged / baseline mode
///   • History   — commit log for the selected repo
///   • PRs       — pull requests for a chosen forge + remote repo
///   • Branches  — local branches for the selected repo
///
/// The panel is read-only. Writing happens through the Claude session.
class SourceControlPage extends StatefulWidget {
  /// Optional session id — enables the "changes since session start"
  /// (baseline) mode on the Changes tab.
  final String? sessionId;
  const SourceControlPage({super.key, this.sessionId});

  @override
  State<SourceControlPage> createState() => _SourceControlPageState();
}

class _SourceControlPageState extends State<SourceControlPage>
    with SingleTickerProviderStateMixin {
  // ── Plugin / config ─────────────────────────────────────────
  ProviderInfo? _plugin;
  StreamSubscription<void>? _providersSub;
  String? get _pluginName => _plugin?.provider.name;

  ApiClient get _api => context.read<ApiClient>();

  late final TabController _tabs;

  // ── Changes / History / Branches state ──────────────────────
  String _repoPath = '';
  List<String> _bookmarks = const [];
  List<String> _discoveredRepos = const [];

  Map<String, dynamic>? _status;
  List<Map<String, dynamic>> _log = const [];
  List<Map<String, dynamic>> _branches = const [];
  List<Map<String, dynamic>> _diffFiles = const [];
  String _diffMode = 'unstaged'; // 'unstaged' | 'staged' | 'baseline'
  bool _baselineActive = false;
  String _baselineHead = '';
  bool _repoLoading = false;
  bool _diffLoading = false;
  String? _repoError;

  // ── PRs state ────────────────────────────────────────────────
  List<Map<String, dynamic>> _forges = const [];
  String? _selectedForgeId;
  String _forgeRepo = '';
  final List<String> _forgeRepoHistory = <String>[];
  // Saved repos per forge id — persisted server-side via
  // /forges/{id}/saved-repos. Cached by forge id so switching
  // forges doesn't blank the picker while the reload is in flight.
  final Map<String, List<Map<String, dynamic>>> _savedReposByForge =
      <String, List<Map<String, dynamic>>>{};
  List<Map<String, dynamic>> _prs = const [];
  String _prState = 'open';
  bool _prLoading = false;
  String? _prError;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 4, vsync: this);
    _loadPlugin();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _loadPlugin());
  }

  @override
  void dispose() {
    _tabs.dispose();
    _providersSub?.cancel();
    super.dispose();
  }

  // ── Plugin boot ─────────────────────────────────────────────

  Future<void> _loadPlugin() async {
    try {
      final all = await _api.listProviders();
      final match = all
          .where((p) =>
              p.provider.type == 'panel' &&
              p.provider.name == 'source-control' &&
              p.enabled)
          .toList();
      if (!mounted) return;
      if (match.isEmpty) {
        setState(() {
          _plugin = null;
          _status = null;
          _log = const [];
          _branches = const [];
          _diffFiles = const [];
          _forges = const [];
          _prs = const [];
        });
        return;
      }
      setState(() => _plugin = match.first);
      await _loadRepos();
      await _loadForges();
    } catch (_) {
      // Surface per-call errors via their own paths; a list-providers
      // failure just leaves the setup CTA visible.
    }
  }

  // ── Repo discovery + selection ──────────────────────────────

  Future<void> _loadRepos() async {
    final name = _pluginName;
    if (name == null) return;
    try {
      final repos = await ApiClient.describeErrors(() => _api.scRepos(name));
      if (!mounted) return;
      final bms = <String>[];
      final disc = <String>[];
      for (final r in repos) {
        final path = (r['path'] as String?) ?? '';
        if (path.isEmpty) continue;
        if (r['isBookmarked'] == true) {
          bms.add(path);
        } else if (r['isGit'] == true) {
          disc.add(path);
        }
      }
      setState(() {
        _bookmarks = bms;
        _discoveredRepos = disc;
      });
      // Seed the selection on first load.
      if (_repoPath.isEmpty) {
        final seed = bms.isNotEmpty
            ? bms.first
            : (disc.isNotEmpty ? disc.first : '');
        if (seed.isNotEmpty) await _selectRepo(seed);
      }
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _repoError = e.message);
    }
  }

  Future<void> _selectRepo(String path) async {
    setState(() {
      _repoPath = path;
      _baselineActive = false;
      _diffMode = 'unstaged';
    });
    await _refreshRepo();
  }

  Future<void> _refreshRepo() async {
    final name = _pluginName;
    if (name == null || _repoPath.isEmpty) return;
    setState(() { _repoLoading = true; _repoError = null; });
    try {
      final status = await ApiClient.describeErrors(
          () => _api.scStatus(name, repo: _repoPath));
      final log = await ApiClient.describeErrors(
          () => _api.scLog(name, repo: _repoPath, limit: 50));
      final branches = await ApiClient.describeErrors(
          () => _api.scBranches(name, repo: _repoPath));
      if (!mounted) return;
      setState(() {
        _status = status;
        _log = log;
        _branches = branches;
        _repoLoading = false;
      });
      await _refreshDiff();
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() { _repoError = e.message; _repoLoading = false; });
    }
  }

  Future<void> _refreshDiff() async {
    final name = _pluginName;
    if (name == null || _repoPath.isEmpty) return;
    setState(() => _diffLoading = true);
    try {
      final result = await ApiClient.describeErrors(() => _api.scDiff(
            name,
            repo: _repoPath,
            mode: _diffMode,
            sessionId: _diffMode == 'baseline' ? (widget.sessionId ?? '') : '',
          ));
      if (!mounted) return;
      final files = ((result['files'] as List?) ?? const [])
          .cast<Map<String, dynamic>>();
      setState(() {
        _diffFiles = files;
        _diffLoading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _diffFiles = const [];
        _diffLoading = false;
        _repoError = e.message;
      });
    }
  }

  Future<void> _bookmark(String path) async {
    final name = _pluginName;
    if (name == null) return;
    try {
      await ApiClient.describeErrors(() => _api.scBookmarkAdd(name, path));
      await _loadRepos();
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  Future<void> _unbookmark(String path) async {
    final name = _pluginName;
    if (name == null) return;
    try {
      await ApiClient.describeErrors(() => _api.scBookmarkRemove(name, path));
      await _loadRepos();
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  Future<void> _takeBaseline() async {
    final name = _pluginName;
    final sid = widget.sessionId;
    if (name == null || sid == null || _repoPath.isEmpty) return;
    try {
      final b = await ApiClient.describeErrors(() =>
          _api.scBaselineSet(name, sessionId: sid, repo: _repoPath));
      if (!mounted) return;
      setState(() {
        _baselineActive = true;
        _baselineHead = (b['headSha'] as String?) ?? '';
        _diffMode = 'baseline';
      });
      await _refreshDiff();
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  // ── Forges + PRs ────────────────────────────────────────────

  Future<void> _loadForges() async {
    final name = _pluginName;
    if (name == null) return;
    try {
      final list = await ApiClient.describeErrors(() => _api.scForgesList(name));
      if (!mounted) return;
      setState(() {
        _forges = list;
        // Drop the selected id if it disappeared. Auto-select the first
        // forge when nothing is selected — common case of one-forge
        // installs.
        final ids = list.map((f) => f['id'] as String?).toSet();
        if (_selectedForgeId != null && !ids.contains(_selectedForgeId)) {
          _selectedForgeId = null;
        }
        if (_selectedForgeId == null && list.isNotEmpty) {
          _selectedForgeId = list.first['id'] as String?;
        }
      });
      if (_selectedForgeId != null) {
        await _loadRemoteRepos();
        await _loadSavedRepos();
      }
      if (_forgeRepo.isNotEmpty) await _loadPRs();
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _prError = e.message);
    }
  }

  Future<void> _loadSavedRepos() async {
    final name = _pluginName;
    final fid = _selectedForgeId;
    if (name == null || fid == null) return;
    try {
      final list = await ApiClient.describeErrors(
          () => _api.scSavedReposList(name, fid));
      if (!mounted) return;
      setState(() => _savedReposByForge[fid] = list);
    } on ApiException catch (_) {
      // Silent — an empty saved list just leaves the picker driven
      // by remote + history, which is still useful.
    }
  }

  Future<void> _toggleSavedRepo(String repo, bool currentlySaved) async {
    final name = _pluginName;
    final fid = _selectedForgeId;
    if (name == null || fid == null || repo.isEmpty) return;
    try {
      if (currentlySaved) {
        await ApiClient.describeErrors(() =>
            _api.scSavedReposRemove(name, fid, fullName: repo));
      } else {
        await ApiClient.describeErrors(() =>
            _api.scSavedReposAdd(name, fid, fullName: repo));
      }
      await _loadSavedRepos();
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  Future<void> _loadRemoteRepos() async {
    final name = _pluginName;
    final fid = _selectedForgeId;
    if (name == null || fid == null) return;
    try {
      final list =
          await ApiClient.describeErrors(() => _api.scForgeRepos(name, fid));
      if (!mounted) return;
      final remote = <String>[];
      for (final r in list) {
        final full = (r['fullName'] as String?) ??
            (r['full_name'] as String?) ??
            (r['name'] as String?) ??
            '';
        if (full.isNotEmpty) remote.add(full);
      }
      setState(() => _remoteRepoNames = remote);
    } on ApiException catch (_) {
      // Silent — not every forge type implements /repos listing
      // perfectly, and the Autocomplete still works with the history.
    }
  }

  List<String> _remoteRepoNames = const [];

  Future<void> _loadPRs() async {
    final name = _pluginName;
    final fid = _selectedForgeId;
    if (name == null || fid == null || _forgeRepo.isEmpty) {
      if (mounted) {
        setState(() {
          _prs = const [];
          _prError = null;
        });
      }
      return;
    }
    setState(() { _prLoading = true; _prError = null; });
    try {
      final prs = await ApiClient.describeErrors(() => _api.scPulls(
            name,
            fid,
            repo: _forgeRepo,
            state: _prState,
          ));
      if (!mounted) return;
      setState(() { _prs = prs; _prLoading = false; });
      if (!_forgeRepoHistory.contains(_forgeRepo)) {
        _forgeRepoHistory.insert(0, _forgeRepo);
        if (_forgeRepoHistory.length > 10) {
          _forgeRepoHistory.removeLast();
        }
      }
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() { _prLoading = false; _prError = e.message; });
    }
  }

  // ── UI ──────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    if (_plugin == null) return _setupCta();
    return Column(children: [
      TabBar(
        controller: _tabs,
        isScrollable: true,
        labelColor: AppColors.accent,
        unselectedLabelColor: AppColors.textMuted,
        indicatorColor: AppColors.accent,
        tabs: [
          Tab(text: context.tr('Changes')),
          Tab(text: context.tr('History')),
          Tab(text: context.tr('PRs')),
          Tab(text: context.tr('Branches')),
        ],
      ),
      Expanded(
        child: TabBarView(controller: _tabs, children: [
          _changesTab(),
          _historyTab(),
          _prsTab(),
          _branchesTab(),
        ]),
      ),
    ]);
  }

  Widget _setupCta() => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            const Icon(Icons.extension_off,
                size: 44, color: AppColors.textMuted),
            const SizedBox(height: 12),
            Text(context.tr('Source Control plugin not enabled'),
                style: const TextStyle(
                    fontWeight: FontWeight.w500, fontSize: 15)),
            const SizedBox(height: 8),
            Text(
              context.tr('Enable the "source-control" panel plugin in '
                  'Settings → Plugins first.'),
              textAlign: TextAlign.center,
              style: const TextStyle(
                  color: AppColors.textMuted, fontSize: 12),
            ),
          ]),
        ),
      );

  // ── Changes tab ─────────────────────────────────────────────

  Widget _changesTab() {
    return Column(children: [
      RepoSelector(
        current: _repoPath,
        bookmarks: _bookmarks,
        discovered: _discoveredRepos,
        onSelect: _selectRepo,
        onPick: () {}, // noop — RepoSelector handles its own dialog
        onBookmark: _bookmark,
        onUnbookmark: _unbookmark,
        onRefresh: _refreshRepo,
        busy: _repoLoading,
      ),
      if (_repoPath.isEmpty)
        const Expanded(child: _EmptyHint(
          icon: Icons.folder_open,
          text: 'Pick or bookmark a repository to begin.',
        ))
      else ...[
        _statusHeader(),
        _modeSwitcher(),
        if (_repoError != null) _errorBar(_repoError!),
        Expanded(
          child: _diffLoading && _diffFiles.isEmpty
              ? const Center(
                  child: CircularProgressIndicator(color: AppColors.accent))
              : MultiFileDiffView(
                  files: _diffFiles,
                  emptyMessage: context.tr('Working tree is clean.'),
                ),
        ),
      ],
    ]);
  }

  Widget _statusHeader() {
    final s = _status;
    if (s == null) return const SizedBox.shrink();
    final branch = (s['branch'] as String?) ?? '';
    final head = (s['head'] as String?) ?? '';
    final ahead = (s['ahead'] as num?)?.toInt() ?? 0;
    final behind = (s['behind'] as num?)?.toInt() ?? 0;
    final clean = s['clean'] == true;
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 6, 12, 6),
      color: AppColors.surface,
      child: Row(children: [
        _chip(Icons.alt_route, branch.isEmpty ? '—' : branch),
        if (head.isNotEmpty) ...[
          const SizedBox(width: 6),
          _chip(
              Icons.commit, head.substring(0, head.length.clamp(0, 7))),
        ],
        if (ahead > 0) ...[
          const SizedBox(width: 6),
          _chip(Icons.arrow_upward, '$ahead',
              color: AppColors.success, soft: AppColors.successSoft),
        ],
        if (behind > 0) ...[
          const SizedBox(width: 6),
          _chip(Icons.arrow_downward, '$behind',
              color: AppColors.warning, soft: AppColors.warningSoft),
        ],
        const Spacer(),
        if (clean)
          _chip(Icons.check_circle_outline, context.tr('Clean'),
              color: AppColors.success, soft: AppColors.successSoft),
      ]),
    );
  }

  Widget _modeSwitcher() {
    return Container(
      padding: const EdgeInsets.fromLTRB(8, 4, 8, 4),
      decoration: const BoxDecoration(
        border: Border(top: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        for (final m in const ['unstaged', 'staged'])
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 2),
            child: ChoiceChip(
              label: Text(context.tr(m)),
              selected: _diffMode == m,
              onSelected: (_) {
                if (_diffMode != m) {
                  setState(() { _diffMode = m; _baselineActive = false; });
                  _refreshDiff();
                }
              },
              selectedColor: AppColors.accent.withValues(alpha: 0.16),
              labelStyle: TextStyle(
                fontSize: 12,
                color: _diffMode == m ? AppColors.accent : AppColors.text,
              ),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(14),
                side: BorderSide(
                    color: _diffMode == m
                        ? AppColors.accent
                        : AppColors.border),
              ),
            ),
          ),
        if (widget.sessionId != null) ...[
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 2),
            child: ChoiceChip(
              label: Text(_baselineActive
                  ? '${context.tr('baseline')} @ '
                      '${_baselineHead.isNotEmpty ? _baselineHead.substring(0, 7) : '?'}'
                  : context.tr('baseline')),
              selected: _diffMode == 'baseline',
              onSelected: (_) {
                if (_diffMode == 'baseline') return;
                if (_baselineActive) {
                  setState(() => _diffMode = 'baseline');
                  _refreshDiff();
                } else {
                  _takeBaseline();
                }
              },
              selectedColor: AppColors.accent.withValues(alpha: 0.16),
              labelStyle: TextStyle(
                fontSize: 12,
                color: _diffMode == 'baseline'
                    ? AppColors.accent
                    : AppColors.text,
              ),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(14),
                side: BorderSide(
                    color: _diffMode == 'baseline'
                        ? AppColors.accent
                        : AppColors.border),
              ),
            ),
          ),
        ],
        const Spacer(),
        if (_diffLoading)
          const SizedBox(
            width: 14,
            height: 14,
            child: CircularProgressIndicator(
                strokeWidth: 2, color: AppColors.accent),
          ),
      ]),
    );
  }

  // ── History tab ─────────────────────────────────────────────

  Widget _historyTab() {
    if (_repoPath.isEmpty) {
      return const _EmptyHint(
          icon: Icons.history,
          text: 'Pick a repository to view its commit log.');
    }
    if (_repoLoading && _log.isEmpty) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_log.isEmpty) {
      return _EmptyHint(
          icon: Icons.history,
          text: context.tr('No commits yet.'));
    }
    return ListView.separated(
      itemCount: _log.length,
      separatorBuilder: (_, _) =>
          const Divider(height: 1, color: AppColors.border),
      itemBuilder: (_, i) {
        final c = _log[i];
        final short = (c['short'] as String?) ?? '';
        final sha = (c['sha'] as String?) ?? (c['hash'] as String?) ?? '';
        final subject = (c['subject'] as String?) ?? '';
        final author = (c['author'] as String?) ?? '';
        final date = (c['date'] as num?)?.toInt() ?? 0;
        final ref = sha.isNotEmpty ? sha : short;
        return ListTile(
          dense: true,
          onTap: ref.isEmpty ? null : () => _openCommitDiff(ref, subject),
          leading: Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(
              color: AppColors.accentSoft,
              borderRadius: BorderRadius.circular(4),
            ),
            child: Text(short,
                style: const TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 11,
                    color: AppColors.accent)),
          ),
          title: Text(subject,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 13)),
          subtitle: Text('$author · ${_relTime(date)}',
              style: const TextStyle(
                  fontSize: 11, color: AppColors.textMuted)),
          trailing: const Icon(Icons.chevron_right,
              size: 18, color: AppColors.textMuted),
        );
      },
    );
  }

  void _openCommitDiff(String sha, String subject) {
    final name = _pluginName;
    if (name == null || _repoPath.isEmpty || sha.isEmpty) return;
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => SourceControlCommitDiffPage(
        pluginName: name,
        repo: _repoPath,
        sha: sha,
        subject: subject,
      ),
    ));
  }

  // ── PRs tab ─────────────────────────────────────────────────

  Widget _prsTab() {
    final name = _pluginName;
    if (name == null) return const SizedBox.shrink();
    return Column(children: [
      ForgeSelector(
        api: _api,
        pluginName: name,
        forges: _forges,
        selectedId: _selectedForgeId,
        repo: _forgeRepo,
        repoHistory: _forgeRepoHistory,
        remoteRepos: _remoteRepoNames,
        savedRepos: _savedReposByForge[_selectedForgeId ?? ''] ?? const [],
        onSelectForge: (id) {
          setState(() {
            _selectedForgeId = id;
            _remoteRepoNames = const [];
          });
          _loadRemoteRepos();
          _loadSavedRepos();
          if (_forgeRepo.isNotEmpty) _loadPRs();
        },
        onSelectRepo: (r) {
          setState(() => _forgeRepo = r);
          _loadPRs();
        },
        onToggleSaved: _toggleSavedRepo,
        onForgesChanged: _loadForges,
        onRefresh: _loadPRs,
        busy: _prLoading,
      ),
      _prFilterBar(),
      if (_prError != null) _errorBar(_prError!),
      Expanded(child: _prList()),
    ]);
  }

  Widget _prFilterBar() {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 4, 8, 4),
      decoration: const BoxDecoration(
        border: Border(top: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        Text(context.tr('State'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
        const SizedBox(width: 8),
        for (final s in const ['open', 'closed', 'all'])
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 2),
            child: ChoiceChip(
              label: Text(context.tr(s)),
              selected: _prState == s,
              onSelected: (_) {
                if (_prState != s) {
                  setState(() => _prState = s);
                  _loadPRs();
                }
              },
              selectedColor: AppColors.accent.withValues(alpha: 0.16),
              labelStyle: TextStyle(
                fontSize: 12,
                color: _prState == s ? AppColors.accent : AppColors.text,
              ),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(14),
                side: BorderSide(
                    color:
                        _prState == s ? AppColors.accent : AppColors.border),
              ),
            ),
          ),
      ]),
    );
  }

  Widget _prList() {
    if (_selectedForgeId == null) {
      return _EmptyHint(
          icon: Icons.cloud_off,
          text: context.tr('Pick or add a forge to begin.'));
    }
    if (_forgeRepo.isEmpty) {
      return _EmptyHint(
          icon: Icons.folder_off,
          text: context.tr('Enter a repo (owner/name) to list PRs.'));
    }
    if (_prLoading && _prs.isEmpty) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_prs.isEmpty) {
      return _EmptyHint(
          icon: Icons.merge_type,
          text: context.tr('No pull requests for this filter.'));
    }
    return RefreshIndicator(
      onRefresh: _loadPRs,
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
      onTap: () => _openPrDetail(pr),
      leading: _prStateIcon(state, draft: draft),
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
                  fontWeight: FontWeight.w500, fontSize: 14)),
        ),
      ]),
      subtitle: Text(
        '$author · $head → $base'
        '${comments > 0 ? '  •  $comments ${context.tr('comments')}' : ''}',
        style: const TextStyle(color: AppColors.textMuted, fontSize: 11),
        overflow: TextOverflow.ellipsis,
      ),
    );
  }

  Widget _prStateIcon(String state, {bool draft = false}) {
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

  void _openPrDetail(Map<String, dynamic> pr) {
    final name = _pluginName;
    final fid = _selectedForgeId;
    final number = (pr['number'] as num?)?.toInt() ?? 0;
    if (name == null || fid == null || number <= 0 || _forgeRepo.isEmpty) return;
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => SourceControlPrDetailPage(
        pluginName: name,
        forgeId: fid,
        repo: _forgeRepo,
        initial: pr,
      ),
    ));
  }

  // ── Branches tab ────────────────────────────────────────────

  Widget _branchesTab() {
    if (_repoPath.isEmpty) {
      return _EmptyHint(
          icon: Icons.alt_route,
          text: context.tr('Pick a repository to view branches.'));
    }
    if (_repoLoading && _branches.isEmpty) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_branches.isEmpty) {
      return _EmptyHint(
          icon: Icons.alt_route,
          text: context.tr('No branches.'));
    }
    return ListView.separated(
      itemCount: _branches.length,
      separatorBuilder: (_, _) =>
          const Divider(height: 1, color: AppColors.border),
      itemBuilder: (_, i) {
        final b = _branches[i];
        final nameStr = (b['name'] as String?) ?? '';
        final isHead = b['isHead'] == true;
        return ListTile(
          dense: true,
          leading: Icon(
            isHead ? Icons.star : Icons.alt_route,
            color: isHead ? AppColors.accent : AppColors.textMuted,
            size: 18,
          ),
          title: Text(nameStr,
              style: TextStyle(
                  fontSize: 13,
                  fontWeight:
                      isHead ? FontWeight.w600 : FontWeight.w400)),
          subtitle: isHead
              ? Text(context.tr('HEAD'),
                  style: const TextStyle(
                      fontSize: 11, color: AppColors.accent))
              : null,
        );
      },
    );
  }

  // ── Shared UI helpers ───────────────────────────────────────

  Widget _chip(IconData icon, String text,
      {Color color = AppColors.textMuted,
      Color soft = AppColors.surfaceAlt}) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: soft,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Icon(icon, size: 12, color: color),
        const SizedBox(width: 4),
        Text(text, style: TextStyle(fontSize: 11, color: color)),
      ]),
    );
  }

  Widget _errorBar(String msg) => Container(
        width: double.infinity,
        padding:
            const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        color: AppColors.errorSoft,
        child: Row(children: [
          const Icon(Icons.error_outline,
              size: 14, color: AppColors.error),
          const SizedBox(width: 8),
          Expanded(
              child: Text(msg,
                  style: const TextStyle(
                      color: AppColors.error, fontSize: 12))),
        ]),
      );

  String _relTime(int unix) {
    if (unix == 0) return '';
    final t = DateTime.fromMillisecondsSinceEpoch(unix * 1000);
    final diff = DateTime.now().difference(t);
    if (diff.inMinutes < 1) return context.tr('just now');
    if (diff.inHours < 1) return '${diff.inMinutes}m';
    if (diff.inDays < 1) return '${diff.inHours}h';
    if (diff.inDays < 30) return '${diff.inDays}d';
    return '${t.year}-${t.month.toString().padLeft(2, '0')}-'
        '${t.day.toString().padLeft(2, '0')}';
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), duration: const Duration(seconds: 3)),
    );
  }
}

class _EmptyHint extends StatelessWidget {
  const _EmptyHint({required this.icon, required this.text});
  final IconData icon;
  final String text;
  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(icon, size: 44, color: AppColors.textMuted),
          const SizedBox(height: 12),
          Text(text,
              textAlign: TextAlign.center,
              style:
                  const TextStyle(color: AppColors.textMuted, fontSize: 12)),
        ]),
      ),
    );
  }
}
