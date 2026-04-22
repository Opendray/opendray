import 'package:flutter/material.dart';

import '../../../core/api/api_client.dart';
import '../../../core/services/l10n.dart';
import '../../../shared/theme/app_theme.dart';

/// Top-of-PRs-tab control: pick a forge instance + a repo (owner/name)
/// to query for pull requests. The unified source-control backend
/// stores forges via /forges CRUD, not via plugin Configure, so this
/// widget also opens a compact "Manage forges" dialog.
///
/// Saved repos: the backend persists a curated list per forge under
/// /forges/{id}/saved-repos. The parent page loads that list and
/// passes it down via [savedRepos]; a bookmark button next to the
/// repo field toggles save state via [onToggleSaved].
class ForgeSelector extends StatelessWidget {
  const ForgeSelector({
    super.key,
    required this.api,
    required this.pluginName,
    required this.forges,
    required this.selectedId,
    required this.repo,
    required this.repoHistory,
    required this.remoteRepos,
    required this.savedRepos,
    required this.onSelectForge,
    required this.onSelectRepo,
    required this.onToggleSaved,
    required this.onForgesChanged,
    required this.onRefresh,
    this.busy = false,
  });

  final ApiClient api;
  final String pluginName;
  final List<Map<String, dynamic>> forges;
  final String? selectedId;
  final String repo;
  final List<String> repoHistory;
  final List<String> remoteRepos;
  final List<Map<String, dynamic>> savedRepos;
  final ValueChanged<String> onSelectForge;
  final ValueChanged<String> onSelectRepo;
  final void Function(String repo, bool currentlySaved) onToggleSaved;
  final VoidCallback onForgesChanged;
  final VoidCallback onRefresh;
  final bool busy;

  bool get _isCurrentRepoSaved =>
      repo.isNotEmpty &&
      savedRepos.any((r) => (r['fullName'] as String?) == repo);

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Column(children: [
        Row(children: [
          const Icon(Icons.cloud_outlined, size: 16, color: AppColors.accent),
          const SizedBox(width: 8),
          Expanded(flex: 2, child: _forgeDropdown(context)),
          const SizedBox(width: 8),
          Expanded(flex: 3, child: _repoField(context)),
          IconButton(
            tooltip: _isCurrentRepoSaved
                ? context.tr('Remove from saved')
                : context.tr('Save repo'),
            icon: Icon(
              _isCurrentRepoSaved ? Icons.bookmark : Icons.bookmark_border,
              size: 18,
              color: _isCurrentRepoSaved ? AppColors.accent : null,
            ),
            onPressed: (selectedId == null || repo.isEmpty)
                ? null
                : () => onToggleSaved(repo, _isCurrentRepoSaved),
          ),
          IconButton(
            tooltip: context.tr('Manage forges'),
            icon: const Icon(Icons.settings, size: 18),
            onPressed: () => _openManage(context),
          ),
          IconButton(
            tooltip: context.tr('Refresh'),
            icon: const Icon(Icons.refresh, size: 18),
            onPressed: busy ? null : onRefresh,
          ),
        ]),
      ]),
    );
  }

  Widget _forgeDropdown(BuildContext context) {
    if (forges.isEmpty) {
      return InkWell(
        onTap: () => _openManage(context),
        child: Text(context.tr('Add a forge…'),
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
                color: AppColors.accent, fontSize: 12)),
      );
    }
    return DropdownButtonHideUnderline(
      child: DropdownButton<String>(
        isExpanded: true,
        value: selectedId,
        hint: Text(context.tr('Pick a forge'),
            style: const TextStyle(
                color: AppColors.textMuted, fontSize: 12)),
        items: [
          for (final f in forges)
            DropdownMenuItem(
              value: (f['id'] as String?) ?? '',
              child: Text(
                '${f['name']} · ${f['type']}',
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 12),
              ),
            ),
        ],
        onChanged: (v) {
          if (v != null && v.isNotEmpty) onSelectForge(v);
        },
      ),
    );
  }

  Widget _repoField(BuildContext context) {
    // Build options in priority order: saved repos first (backend
    // sorts by lastUsedAt DESC so recents come first), then live
    // remote repos, then session-local history. Dedup by fullName.
    final seen = <String>{};
    final opts = <String>[];
    final saved = <String>{};
    for (final r in savedRepos) {
      final n = (r['fullName'] as String?) ?? '';
      if (n.isNotEmpty && seen.add(n)) {
        opts.add(n);
        saved.add(n);
      }
    }
    for (final r in remoteRepos) {
      if (seen.add(r)) opts.add(r);
    }
    for (final r in repoHistory) {
      if (seen.add(r)) opts.add(r);
    }
    if (repo.isNotEmpty && seen.add(repo)) opts.add(repo);

    return Autocomplete<String>(
      initialValue: TextEditingValue(text: repo),
      optionsBuilder: (value) {
        final q = value.text.trim();
        if (q.isEmpty) return opts;
        return opts.where((o) => o.toLowerCase().contains(q.toLowerCase()));
      },
      onSelected: onSelectRepo,
      optionsViewBuilder: (ctx, onSelected, options) {
        final list = options.toList();
        return Align(
          alignment: Alignment.topLeft,
          child: Material(
            elevation: 4,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 240, maxWidth: 360),
              child: ListView.builder(
                padding: EdgeInsets.zero,
                shrinkWrap: true,
                itemCount: list.length,
                itemBuilder: (_, i) {
                  final o = list[i];
                  final isSaved = saved.contains(o);
                  return InkWell(
                    onTap: () => onSelected(o),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 10, vertical: 6),
                      child: Row(children: [
                        Icon(
                          isSaved ? Icons.bookmark : Icons.history,
                          size: 14,
                          color: isSaved
                              ? AppColors.accent
                              : AppColors.textMuted,
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(o,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(fontSize: 12)),
                        ),
                      ]),
                    ),
                  );
                },
              ),
            ),
          ),
        );
      },
      fieldViewBuilder: (ctx, controller, focusNode, onSubmit) {
        return TextField(
          controller: controller,
          focusNode: focusNode,
          style: const TextStyle(fontSize: 12),
          decoration: InputDecoration(
            isDense: true,
            hintText: context.tr('owner/name'),
            hintStyle:
                const TextStyle(color: AppColors.textMuted, fontSize: 12),
            border: InputBorder.none,
          ),
          onSubmitted: (v) {
            onSelectRepo(v.trim());
            onSubmit();
          },
        );
      },
    );
  }

  Future<void> _openManage(BuildContext context) async {
    final changed = await showDialog<bool>(
      context: context,
      builder: (_) => _ManageForgesDialog(
        api: api,
        pluginName: pluginName,
        initial: forges,
      ),
    );
    if (changed == true) onForgesChanged();
  }
}

class _ManageForgesDialog extends StatefulWidget {
  const _ManageForgesDialog({
    required this.api,
    required this.pluginName,
    required this.initial,
  });

  final ApiClient api;
  final String pluginName;
  final List<Map<String, dynamic>> initial;

  @override
  State<_ManageForgesDialog> createState() => _ManageForgesDialogState();
}

class _ManageForgesDialogState extends State<_ManageForgesDialog> {
  late List<Map<String, dynamic>> _forges;
  bool _busy = false;
  String? _error;
  bool _dirty = false;

  final _nameCtl = TextEditingController();
  final _baseUrlCtl = TextEditingController();
  final _tokenCtl = TextEditingController();
  String _type = 'github';

  // Backend defaults (mirror gateway/sourcecontrol_forges.go). github and
  // gitlab accept an empty baseUrl and fill these in server-side; gitea
  // is self-hosted so a baseUrl is mandatory.
  static const _defaultBaseUrls = <String, String>{
    'github': 'https://api.github.com',
    'gitlab': 'https://gitlab.com',
  };

  @override
  void initState() {
    super.initState();
    _forges = List.of(widget.initial);
  }

  @override
  void dispose() {
    _nameCtl.dispose();
    _baseUrlCtl.dispose();
    _tokenCtl.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    try {
      final list = await ApiClient.describeErrors(
          () => widget.api.scForgesList(widget.pluginName));
      if (!mounted) return;
      setState(() => _forges = list);
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.message);
    }
  }

  Future<void> _add() async {
    final name = _nameCtl.text.trim();
    final baseUrl = _baseUrlCtl.text.trim();
    final token = _tokenCtl.text.trim();
    if (name.isEmpty) {
      setState(() => _error = context.tr('Name is required'));
      return;
    }
    // Gitea is self-hosted; the server can't guess the URL. github
    // and gitlab have well-known defaults the backend fills in for
    // us, so we let baseUrl through empty.
    if (baseUrl.isEmpty && _type == 'gitea') {
      setState(() => _error =
          context.tr('Base URL is required for self-hosted Gitea'));
      return;
    }
    setState(() { _busy = true; _error = null; });
    try {
      await ApiClient.describeErrors(() => widget.api.scForgeCreate(
            widget.pluginName,
            name: name,
            type: _type,
            baseUrl: baseUrl,
            token: token,
          ));
      _nameCtl.clear();
      _baseUrlCtl.clear();
      _tokenCtl.clear();
      _dirty = true;
      await _reload();
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _delete(String id) async {
    setState(() { _busy = true; _error = null; });
    try {
      await ApiClient.describeErrors(
          () => widget.api.scForgeDelete(widget.pluginName, id));
      _dirty = true;
      await _reload();
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  String? _baseUrlHelper() {
    final fallback = _defaultBaseUrls[_type];
    if (fallback == null) return null;
    return '${context.tr('Leave empty to use')} $fallback';
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(context.tr('Manage forges')),
      content: SizedBox(
        width: 440,
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          if (_forges.isEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Text(
                context.tr('No forges configured.'),
                style: const TextStyle(
                    color: AppColors.textMuted, fontSize: 12),
              ),
            )
          else
            ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 180),
              child: ListView.separated(
                shrinkWrap: true,
                itemCount: _forges.length,
                separatorBuilder: (_, _) =>
                    const Divider(height: 1, color: AppColors.border),
                itemBuilder: (_, i) {
                  final f = _forges[i];
                  return ListTile(
                    dense: true,
                    contentPadding: EdgeInsets.zero,
                    title: Text('${f['name']}',
                        style: const TextStyle(fontSize: 13)),
                    subtitle: Text(
                        '${f['type']} · ${f['baseUrl']}${f['tokenSet'] == true ? ' · 🔑' : ''}',
                        style: const TextStyle(
                            color: AppColors.textMuted, fontSize: 11)),
                    trailing: IconButton(
                      icon: const Icon(Icons.delete_outline,
                          size: 18, color: AppColors.error),
                      onPressed: _busy
                          ? null
                          : () => _delete((f['id'] as String?) ?? ''),
                    ),
                  );
                },
              ),
            ),
          const Divider(),
          Text(context.tr('Add forge'),
              style: const TextStyle(fontSize: 12, color: AppColors.textMuted)),
          const SizedBox(height: 6),
          TextField(
            controller: _nameCtl,
            decoration: InputDecoration(
                isDense: true, labelText: context.tr('Name')),
          ),
          const SizedBox(height: 6),
          Row(children: [
            DropdownButton<String>(
              value: _type,
              items: const [
                DropdownMenuItem(value: 'github', child: Text('github')),
                DropdownMenuItem(value: 'gitlab', child: Text('gitlab')),
                DropdownMenuItem(value: 'gitea', child: Text('gitea')),
              ],
              onChanged: (v) => setState(() => _type = v ?? 'github'),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: TextField(
                controller: _baseUrlCtl,
                decoration: InputDecoration(
                  isDense: true,
                  labelText: 'Base URL',
                  helperText: _baseUrlHelper(),
                  helperStyle: const TextStyle(
                      fontSize: 10, color: AppColors.textMuted),
                ),
              ),
            ),
          ]),
          const SizedBox(height: 6),
          TextField(
            controller: _tokenCtl,
            obscureText: true,
            decoration: InputDecoration(
                isDense: true,
                labelText: context.tr('Token (optional)')),
          ),
          if (_error != null) ...[
            const SizedBox(height: 6),
            Text(_error!,
                style:
                    const TextStyle(color: AppColors.error, fontSize: 11)),
          ],
        ]),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : _add,
          child: Text(context.tr('Add')),
        ),
        TextButton(
          onPressed: () => Navigator.of(context).pop(_dirty),
          child: Text(context.tr('Close')),
        ),
      ],
    );
  }
}
