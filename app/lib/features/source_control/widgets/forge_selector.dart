import 'package:flutter/material.dart';

import '../../../core/api/api_client.dart';
import '../../../core/services/l10n.dart';
import '../../../shared/theme/app_theme.dart';

/// Top-of-PRs-tab control: pick a forge instance + a repo (owner/name)
/// to query for pull requests. The unified source-control backend
/// stores forges via /forges CRUD, not via plugin Configure, so this
/// widget also opens a compact "Manage forges" dialog.
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
    required this.onSelectForge,
    required this.onSelectRepo,
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
  final ValueChanged<String> onSelectForge;
  final ValueChanged<String> onSelectRepo;
  final VoidCallback onForgesChanged;
  final VoidCallback onRefresh;
  final bool busy;

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
    final seen = <String>{};
    final opts = <String>[];
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
  String _type = 'gitea';

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
    if (name.isEmpty || baseUrl.isEmpty) {
      setState(() => _error = 'name and baseUrl are required');
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
                DropdownMenuItem(value: 'gitea', child: Text('gitea')),
                DropdownMenuItem(value: 'github', child: Text('github')),
                DropdownMenuItem(value: 'gitlab', child: Text('gitlab')),
              ],
              onChanged: (v) => setState(() => _type = v ?? 'gitea'),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: TextField(
                controller: _baseUrlCtl,
                decoration: const InputDecoration(
                    isDense: true, labelText: 'Base URL'),
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
