import 'package:flutter/material.dart';

import '../../../core/services/l10n.dart';
import '../../../shared/directory_picker.dart';
import '../../../shared/theme/app_theme.dart';

/// Top-of-page selector for the Changes / History / Branches tabs.
/// Shows the current repo path, a bookmark dropdown, "Pick folder",
/// and bookmark add/remove actions. The backend persists bookmarks
/// per plugin instance under plugin_kv; we just hand the callbacks
/// the raw path strings and let the parent wire the API calls.
class RepoSelector extends StatelessWidget {
  const RepoSelector({
    super.key,
    required this.current,
    required this.bookmarks,
    required this.discovered,
    required this.onSelect,
    required this.onPick,
    required this.onBookmark,
    required this.onUnbookmark,
    required this.onRefresh,
    this.busy = false,
  });

  final String current;
  final List<String> bookmarks;
  final List<String> discovered;
  final ValueChanged<String> onSelect;
  final VoidCallback onPick;
  final ValueChanged<String> onBookmark;
  final ValueChanged<String> onUnbookmark;
  final VoidCallback onRefresh;
  final bool busy;

  @override
  Widget build(BuildContext context) {
    final isBookmarked = current.isNotEmpty && bookmarks.contains(current);
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        const Icon(Icons.folder_outlined, size: 16, color: AppColors.accent),
        const SizedBox(width: 8),
        Expanded(child: _pathDropdown()),
        IconButton(
          tooltip: isBookmarked
              ? context.tr('Remove bookmark')
              : context.tr('Bookmark this repo'),
          icon: Icon(
              isBookmarked ? Icons.bookmark : Icons.bookmark_border,
              size: 18,
              color: isBookmarked ? AppColors.accent : AppColors.textMuted),
          onPressed: current.isEmpty
              ? null
              : () => isBookmarked
                  ? onUnbookmark(current)
                  : onBookmark(current),
        ),
        IconButton(
          tooltip: context.tr('Pick folder'),
          icon: const Icon(Icons.folder_open, size: 18),
          onPressed: () => _pick(context),
        ),
        IconButton(
          tooltip: context.tr('Refresh'),
          icon: const Icon(Icons.refresh, size: 18),
          onPressed: busy ? null : onRefresh,
        ),
      ]),
    );
  }

  Widget _pathDropdown() {
    // Combine bookmarks + newly discovered repos into one dropdown.
    // Dedup to avoid double entries when a bookmarked repo also
    // shows up from the auto-scan.
    final seen = <String>{};
    final items = <String>[];
    for (final p in bookmarks) {
      if (seen.add(p)) items.add(p);
    }
    for (final p in discovered) {
      if (seen.add(p)) items.add(p);
    }
    if (current.isNotEmpty && seen.add(current)) items.add(current);

    if (items.isEmpty) {
      return Text(current.isEmpty ? '(no repo selected)' : current,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(
              color: AppColors.textMuted, fontSize: 12));
    }
    return DropdownButtonHideUnderline(
      child: DropdownButton<String>(
        isExpanded: true,
        value: current.isEmpty ? null : current,
        hint: const Text('(no repo selected)',
            style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
        items: [
          for (final p in items)
            DropdownMenuItem(
              value: p,
              child: Text(
                p,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 12),
              ),
            ),
        ],
        onChanged: (v) {
          if (v != null && v.isNotEmpty) onSelect(v);
        },
      ),
    );
  }

  Future<void> _pick(BuildContext context) async {
    final picked = await pickDirectory(context, initialPath: current);
    if (picked == null || picked.isEmpty) return;
    onPick();
    onSelect(picked);
  }
}
