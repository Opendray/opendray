import 'package:flutter/material.dart';

import '../../../shared/theme/app_theme.dart';
import 'file_status_row.dart';
import 'markdown_preview.dart';

/// Stacked list of per-file diff cards. Each card expands to show the
/// unified patch, with an optional markdown-preview toggle when the
/// file looks like a .md. The backend already ships a `previewHtml`
/// field for .md files, but rendering arbitrary HTML in Flutter needs
/// an extra dependency — we re-render from the patch text via
/// flutter_markdown instead, which is already in pubspec.
class MultiFileDiffView extends StatelessWidget {
  const MultiFileDiffView({
    super.key,
    required this.files,
    this.emptyMessage,
  });

  final List<Map<String, dynamic>> files;
  final String? emptyMessage;

  @override
  Widget build(BuildContext context) {
    if (files.isEmpty) {
      return Center(
        child: Text(emptyMessage ?? 'No changes',
            style:
                const TextStyle(color: AppColors.textMuted, fontSize: 12)),
      );
    }
    return ListView.separated(
      itemCount: files.length,
      separatorBuilder: (_, _) => const SizedBox(height: 6),
      itemBuilder: (_, i) => _FileDiffCard(file: files[i]),
    );
  }
}

class _FileDiffCard extends StatefulWidget {
  const _FileDiffCard({required this.file});
  final Map<String, dynamic> file;
  @override
  State<_FileDiffCard> createState() => _FileDiffCardState();
}

class _FileDiffCardState extends State<_FileDiffCard> {
  bool _preview = false;

  @override
  Widget build(BuildContext context) {
    final f = widget.file;
    final path = (f['path'] as String?) ?? '';
    final oldPath = ((f['oldPath'] as String?) ?? '').trim();
    final status = (f['status'] as String?) ?? 'modified';
    // Tolerate both {add,del} (source-control multidiff) and
    // {additions,deletions} (forge.Diff) — the UI doesn't care which
    // backend rendered this card.
    final add = (f['add'] as num?)?.toInt()
        ?? (f['additions'] as num?)?.toInt()
        ?? 0;
    final del = (f['del'] as num?)?.toInt()
        ?? (f['deletions'] as num?)?.toInt()
        ?? 0;
    final patch = (f['patch'] as String?) ?? '';
    final isBinary = f['isBinary'] == true;
    final supportsMd = path.toLowerCase().endsWith('.md') ||
        path.toLowerCase().endsWith('.markdown');

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      child: ExpansionTile(
        tilePadding: EdgeInsets.zero,
        childrenPadding: EdgeInsets.zero,
        title: FileStatusRow(
          status: status,
          path: path,
          oldPath: oldPath,
          add: add,
          del: del,
          trailing: supportsMd
              ? IconButton(
                  iconSize: 16,
                  padding: EdgeInsets.zero,
                  constraints: const BoxConstraints(minWidth: 24, minHeight: 24),
                  tooltip: _preview ? 'Show diff' : 'Preview markdown',
                  icon: Icon(_preview ? Icons.code : Icons.article_outlined),
                  onPressed: () => setState(() => _preview = !_preview),
                )
              : null,
        ),
        children: [
          if (isBinary)
            const Padding(
              padding: EdgeInsets.all(12),
              child: Text('(binary file)',
                  style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
            )
          else if (_preview && supportsMd)
            MarkdownPreview(patch: patch)
          else
            _DiffBody(patch: patch),
        ],
      ),
    );
  }
}

class _DiffBody extends StatelessWidget {
  const _DiffBody({required this.patch});
  final String patch;

  @override
  Widget build(BuildContext context) {
    if (patch.isEmpty) {
      return const Padding(
        padding: EdgeInsets.all(12),
        child: Text('(no patch body)',
            style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
      );
    }
    return Container(
      width: double.infinity,
      color: AppColors.bg,
      padding: const EdgeInsets.all(10),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: SelectableText.rich(
          _highlight(patch),
          style: const TextStyle(
              fontFamily: 'monospace', fontSize: 11.5, height: 1.4),
        ),
      ),
    );
  }

  TextSpan _highlight(String text) {
    final out = <TextSpan>[];
    for (final line in text.split('\n')) {
      out.add(TextSpan(
        text: '$line\n',
        style: TextStyle(color: _colorFor(line)),
      ));
    }
    return TextSpan(children: out);
  }

  Color _colorFor(String line) {
    if (line.startsWith('+++') || line.startsWith('---')) {
      return AppColors.textMuted;
    }
    if (line.startsWith('@@')) return AppColors.accent;
    if (line.startsWith('+')) return AppColors.success;
    if (line.startsWith('-')) return AppColors.error;
    if (line.startsWith('diff ')) return AppColors.accent;
    return AppColors.text;
  }
}
