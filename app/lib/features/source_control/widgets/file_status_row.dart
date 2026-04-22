import 'package:flutter/material.dart';

import '../../../shared/theme/app_theme.dart';

/// One-line row describing a changed file. Shared between the Changes
/// tab's status list and the diff list's per-file header. Status codes
/// come from the backend's FileDiff.status: modified / added / deleted
/// / renamed / untracked.
class FileStatusRow extends StatelessWidget {
  const FileStatusRow({
    super.key,
    required this.status,
    required this.path,
    this.oldPath,
    this.add = 0,
    this.del = 0,
    this.selected = false,
    this.onTap,
    this.trailing,
  });

  final String status;
  final String path;
  final String? oldPath;
  final int add;
  final int del;
  final bool selected;
  final VoidCallback? onTap;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final display = (oldPath != null && oldPath!.isNotEmpty)
        ? '$oldPath → $path'
        : path;
    final row = Row(children: [
      _statusChip(status),
      const SizedBox(width: 8),
      Expanded(
        child: Text(
          display,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
        ),
      ),
      if (add > 0)
        Padding(
          padding: const EdgeInsets.only(left: 6),
          child: Text('+$add',
              style: const TextStyle(color: AppColors.success, fontSize: 11)),
        ),
      if (del > 0)
        Padding(
          padding: const EdgeInsets.only(left: 4),
          child: Text('-$del',
              style: const TextStyle(color: AppColors.error, fontSize: 11)),
        ),
      if (trailing != null) ...[const SizedBox(width: 6), trailing!],
    ]);
    return InkWell(
      onTap: onTap,
      child: Container(
        color: selected ? AppColors.accentSoft : null,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: row,
      ),
    );
  }

  Widget _statusChip(String s) {
    final (label, color) = switch (s) {
      'added'     => ('A', AppColors.success),
      'deleted'   => ('D', AppColors.error),
      'renamed'   => ('R', AppColors.accent),
      'untracked' => ('?', AppColors.warning),
      _           => ('M', AppColors.textMuted),
    };
    return Container(
      width: 20,
      alignment: Alignment.center,
      padding: const EdgeInsets.symmetric(vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(label,
          style: TextStyle(
              color: color,
              fontSize: 11,
              fontWeight: FontWeight.w600,
              fontFamily: 'monospace')),
    );
  }
}
