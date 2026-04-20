import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/directory_picker.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Git panel — read-only per-repo status, diff, log, and branches.
///
/// Observer role only: commits and pushes flow through the Claude
/// session, not this panel. Three states, matching the pattern used by
/// the Tasks panel:
///   1. No git-viewer plugin enabled → setup CTA.
///   2. Plugin enabled, no repo path yet → big "pick folder" CTA.
///   3. Path chosen → tabbed view: Changes / History.
class GitPage extends StatefulWidget {
  /// Optional session ID — if provided, the panel can snapshot HEAD and
  /// show only the diff since the session started.
  final String? sessionId;
  const GitPage({super.key, this.sessionId});
  @override
  State<GitPage> createState() => _GitPageState();
}

class _GitPageState extends State<GitPage> with SingleTickerProviderStateMixin {
  ProviderInfo? _plugin;
  StreamSubscription<void>? _providersSub;

  String _path = '';

  Map<String, dynamic>? _status;
  List<Map<String, dynamic>> _log = [];
  bool _loading = false;
  String? _error;

  String? _selectedFile;
  bool _selectedStaged = false;
  String _diffText = '';
  bool _diffLoading = false;

  bool _sessionBaseline = false;
  String _sessionBaselineHead = '';

  late final TabController _tabs;

  ApiClient get _api => context.read<ApiClient>();
  String? get _pluginName => _plugin?.provider.name;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 2, vsync: this);
    _loadPlugin();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _loadPlugin());
  }

  @override
  void dispose() {
    _tabs.dispose();
    _providersSub?.cancel();
    super.dispose();
  }

  // ─── Plugin / path ────────────────────────────────────────

  Future<void> _loadPlugin() async {
    try {
      final all = await _api.listProviders();
      final match = all.where((p) =>
          p.provider.type == 'panel' &&
          p.provider.name == 'git-viewer' &&
          p.enabled).toList();
      if (!mounted) return;
      if (match.isEmpty) {
        setState(() {
          _plugin = null;
          _status = null;
          _log = [];
          _error = null;
        });
        return;
      }
      final info = match.first;
      final defaultPath = (info.config['defaultPath'] as String?) ?? '';
      setState(() {
        _plugin = info;
        _path = _path.isEmpty ? defaultPath : _path;
      });
      if (_path.isNotEmpty) await _refresh();
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = '$e');
    }
  }

  Future<void> _pickPath() async {
    final picked = await pickDirectory(context, initialPath: _path);
    if (picked == null || picked.isEmpty) return;
    setState(() => _path = picked);
    await _refresh();
  }

  // ─── Data ─────────────────────────────────────────────────

  Future<void> _refresh() async {
    if (_pluginName == null || _path.isEmpty) return;
    setState(() { _loading = true; _error = null; });
    try {
      final status = await ApiClient.describeErrors(
          () => _api.gitStatus(_pluginName!, path: _path));
      final log = await ApiClient.describeErrors(
          () => _api.gitLog(_pluginName!, path: _path, limit: 30));
      if (!mounted) return;
      setState(() {
        _status = status;
        _log = log;
        _loading = false;
      });
      await _refreshDiff();
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() { _error = e.message; _loading = false; });
    } catch (e) {
      if (!mounted) return;
      setState(() { _error = '$e'; _loading = false; });
    }
  }

  Future<void> _refreshDiff() async {
    if (_pluginName == null || _path.isEmpty) {
      setState(() => _diffText = '');
      return;
    }
    setState(() => _diffLoading = true);
    try {
      final diff = _sessionBaseline && widget.sessionId != null
          ? await _api.gitSessionDiff(_pluginName!, widget.sessionId!)
          : await _api.gitDiff(
              _pluginName!,
              path: _path,
              staged: _selectedStaged,
              file: _selectedFile ?? '',
            );
      if (!mounted) return;
      setState(() { _diffText = diff; _diffLoading = false; });
    } on DioException catch (e) {
      if (!mounted) return;
      final msg = apiExceptionFrom(e).message;
      setState(() { _diffText = '# $msg'; _diffLoading = false; });
    }
  }

  Future<void> _takeSnapshot() async {
    if (widget.sessionId == null || _pluginName == null) return;
    try {
      final res = await ApiClient.describeErrors(() =>
          _api.gitSessionSnapshot(_pluginName!, widget.sessionId!, path: _path));
      if (!mounted) return;
      setState(() {
        _sessionBaseline = true;
        _sessionBaselineHead = (res['headSha'] as String?) ?? '';
      });
      await _refreshDiff();
    } on ApiException catch (e) {
      if (mounted) _toast(e.message);
    }
  }


  // ─── UI ────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    if (_plugin == null) return _setupCta();
    if (_path.isEmpty) return _pickCta();

    return Column(children: [
      _header(),
      if (_error != null) _errorBar(_error!),
      TabBar(
        controller: _tabs,
        labelColor: AppColors.accent,
        unselectedLabelColor: AppColors.textMuted,
        indicatorColor: AppColors.accent,
        tabs: [
          Tab(text: context.tr('Changes')),
          Tab(text: context.tr('History')),
        ],
      ),
      Expanded(child: TabBarView(controller: _tabs, children: [
        _changesTab(),
        _historyTab(),
      ])),
    ]);
  }

  Widget _setupCta() => _centered(
    icon: Icons.extension_off,
    title: context.tr('Git panel not enabled'),
    body: context.tr('Enable the "git" panel plugin in Settings → Plugins first.'),
  );

  Widget _pickCta() => _centered(
    icon: Icons.folder_open,
    title: context.tr('Pick a repository'),
    body: context.tr('Choose a directory containing a .git folder.'),
    action: FilledButton.icon(
      onPressed: _pickPath,
      icon: const Icon(Icons.folder_open),
      label: Text(context.tr('Pick folder')),
    ),
  );

  Widget _centered({
    required IconData icon,
    required String title,
    required String body,
    Widget? action,
  }) => Center(
    child: Padding(
      padding: const EdgeInsets.all(24),
      child: Column(mainAxisSize: MainAxisSize.min, children: [
        Icon(icon, size: 48, color: AppColors.textMuted),
        const SizedBox(height: 14),
        Text(title, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
        const SizedBox(height: 6),
        Text(body,
            textAlign: TextAlign.center,
            style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
        if (action != null) ...[const SizedBox(height: 16), action],
      ]),
    ),
  );

  Widget _header() {
    final branch = (_status?['branch'] as String?) ?? '';
    final head = (_status?['head'] as String?) ?? '';
    final ahead = (_status?['ahead'] as int?) ?? 0;
    final behind = (_status?['behind'] as int?) ?? 0;
    final clean = (_status?['clean'] as bool?) ?? true;

    return Container(
      padding: const EdgeInsets.fromLTRB(14, 10, 10, 10),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          const Icon(Icons.park_outlined, size: 16, color: AppColors.accent),
          const SizedBox(width: 6),
          Expanded(
            child: GestureDetector(
              onTap: _pickPath,
              child: Text(
                _path,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 12, color: AppColors.textMuted),
              ),
            ),
          ),
          IconButton(
            tooltip: context.tr('Refresh'),
            icon: const Icon(Icons.refresh, size: 18),
            onPressed: _loading ? null : _refresh,
          ),
        ]),
        const SizedBox(height: 4),
        Row(children: [
          _chip(Icons.alt_route, branch.isEmpty ? '—' : branch),
          if (head.isNotEmpty) ...[
            const SizedBox(width: 6),
            _chip(Icons.commit, head.substring(0, head.length.clamp(0, 7))),
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
        if (widget.sessionId != null) _sessionRow(),
      ]),
    );
  }

  Widget _sessionRow() {
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Row(children: [
        Icon(_sessionBaseline ? Icons.timer : Icons.timer_outlined,
            size: 14,
            color: _sessionBaseline ? AppColors.accent : AppColors.textMuted),
        const SizedBox(width: 6),
        Expanded(child: Text(
          _sessionBaseline
              ? context.tr('Showing changes since session start @ ') +
                  (_sessionBaselineHead.isNotEmpty
                      ? _sessionBaselineHead.substring(0, 7)
                      : '?')
              : context.tr('Snapshot HEAD to track session-only changes.'),
          style: const TextStyle(fontSize: 11, color: AppColors.textMuted),
        )),
        TextButton(
          onPressed: () {
            if (_sessionBaseline) {
              setState(() { _sessionBaseline = false; });
              _refreshDiff();
            } else {
              _takeSnapshot();
            }
          },
          child: Text(_sessionBaseline
              ? context.tr('Clear')
              : context.tr('Snapshot')),
        ),
      ]),
    );
  }

  Widget _chip(IconData icon, String text,
      {Color color = AppColors.textMuted, Color soft = AppColors.surfaceAlt}) {
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
    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
    color: AppColors.errorSoft,
    child: Row(children: [
      const Icon(Icons.error_outline, size: 14, color: AppColors.error),
      const SizedBox(width: 8),
      Expanded(child: Text(msg,
          style: const TextStyle(color: AppColors.error, fontSize: 12))),
    ]),
  );

  // ─── Changes tab ──────────────────────────────────────────

  Widget _changesTab() {
    final files = (_status?['files'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    return Column(children: [
      if (files.isEmpty && !_loading)
        Expanded(child: _centered(
          icon: Icons.done_all,
          title: context.tr('No changes'),
          body: context.tr('Working tree is clean.'),
        ))
      else
        SizedBox(
          height: 180,
          child: _loading && _status == null
              ? const Center(child: CircularProgressIndicator(color: AppColors.accent))
              : ListView.separated(
                  itemCount: files.length,
                  separatorBuilder: (_, _) =>
                      const Divider(height: 1, color: AppColors.border),
                  itemBuilder: (_, i) => _fileRow(files[i]),
                ),
        ),
      if (files.isNotEmpty) const Divider(height: 1, color: AppColors.border),
      if (files.isNotEmpty) _selectionFooter(),
      Expanded(child: _diffView()),
    ]);
  }

  Widget _fileRow(Map<String, dynamic> f) {
    final path = f['path'] as String;
    final staged = f['staged'] == true;
    final unstaged = f['unstaged'] == true;
    final untracked = f['untracked'] == true;
    final selected = _selectedFile == path;

    final code = untracked
        ? '??'
        : '${(f['index'] as String?) ?? ' '}${(f['workTree'] as String?) ?? ' '}';
    final tagColor = untracked
        ? AppColors.warning
        : staged && !unstaged
            ? AppColors.success
            : AppColors.accent;

    return InkWell(
      onTap: () {
        setState(() {
          _selectedFile = path;
          _selectedStaged = staged && !unstaged; // prefer staged when both
          if (_sessionBaseline) _sessionBaseline = false;
        });
        _refreshDiff();
      },
      child: Container(
        color: selected ? AppColors.accentSoft : null,
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        child: Row(children: [
          Container(
            width: 28,
            alignment: Alignment.center,
            padding: const EdgeInsets.symmetric(vertical: 2),
            decoration: BoxDecoration(
              color: tagColor.withValues(alpha: 0.15),
              borderRadius: BorderRadius.circular(4),
            ),
            child: Text(code.trim().isEmpty ? '·' : code,
                style: TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 11,
                    color: tagColor,
                    fontWeight: FontWeight.w600)),
          ),
          const SizedBox(width: 10),
          Expanded(child: Text(path,
              style: const TextStyle(fontSize: 13),
              overflow: TextOverflow.ellipsis)),
          if (staged)
            const Icon(Icons.check, size: 14, color: AppColors.success),
        ]),
      ),
    );
  }

  /// Thin footer that tells the user which side of the diff they are
  /// looking at (staged vs unstaged) once a file is selected. The panel
  /// is read-only, so there are no actions here — write ops live in the
  /// Claude session.
  Widget _selectionFooter() {
    if (_selectedFile == null) return const SizedBox.shrink();
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      color: AppColors.surface,
      alignment: Alignment.centerRight,
      child: Text(
        _selectedStaged
            ? context.tr('staged diff')
            : context.tr('unstaged diff'),
        style: const TextStyle(fontSize: 11, color: AppColors.textMuted),
      ),
    );
  }

  Widget _diffView() {
    if (_diffLoading) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_diffText.isEmpty) {
      return Center(child: Text(
        context.tr('Select a file to view its diff.'),
        style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
      ));
    }
    return Container(
      color: AppColors.bg,
      padding: const EdgeInsets.all(12),
      width: double.infinity,
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: SingleChildScrollView(
          child: SelectableText.rich(_highlightDiff(_diffText),
              style: const TextStyle(
                  fontFamily: 'monospace', fontSize: 11.5, height: 1.4)),
        ),
      ),
    );
  }

  TextSpan _highlightDiff(String text) {
    final lines = text.split('\n');
    return TextSpan(children: [
      for (final line in lines)
        TextSpan(
          text: '$line\n',
          style: TextStyle(color: _diffColor(line)),
        ),
    ]);
  }

  Color _diffColor(String line) {
    if (line.startsWith('+++') || line.startsWith('---')) return AppColors.textMuted;
    if (line.startsWith('@@')) return AppColors.accent;
    if (line.startsWith('+')) return AppColors.success;
    if (line.startsWith('-')) return AppColors.error;
    if (line.startsWith('diff ')) return AppColors.accent;
    return AppColors.text;
  }

  // ─── History tab ──────────────────────────────────────────

  Widget _historyTab() {
    if (_loading && _log.isEmpty) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_log.isEmpty) {
      return _centered(
        icon: Icons.history,
        title: context.tr('No commits yet'),
        body: context.tr('The log will appear here once there are commits.'),
      );
    }
    return ListView.separated(
      itemCount: _log.length,
      separatorBuilder: (_, _) => const Divider(height: 1, color: AppColors.border),
      itemBuilder: (_, i) {
        final c = _log[i];
        final short = (c['short'] as String?) ?? '';
        final subject = (c['subject'] as String?) ?? '';
        final author = (c['author'] as String?) ?? '';
        final date = (c['date'] as int?) ?? 0;
        return ListTile(
          dense: true,
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
              maxLines: 1, overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 13)),
          subtitle: Text('$author · ${_relTime(date)}',
              style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
        );
      },
    );
  }

  String _relTime(int unix) {
    if (unix == 0) return '';
    final t = DateTime.fromMillisecondsSinceEpoch(unix * 1000);
    final diff = DateTime.now().difference(t);
    if (diff.inMinutes < 1) return context.tr('just now');
    if (diff.inHours < 1) return '${diff.inMinutes}m';
    if (diff.inDays < 1) return '${diff.inHours}h';
    if (diff.inDays < 30) return '${diff.inDays}d';
    return '${t.year}-${t.month.toString().padLeft(2, '0')}-${t.day.toString().padLeft(2, '0')}';
  }

  // ─── Helpers ─────────────────────────────────────────────

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), duration: const Duration(seconds: 3)),
    );
  }

}
