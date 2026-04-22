import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';

import '../../../shared/theme/app_theme.dart';

/// Renders a reconstructed "new" version of a markdown file from a
/// unified diff patch. We extract the post-change content by keeping
/// every line that isn't a deletion (skip '-'), stripping the leading
/// '+' or ' ', and dropping hunk headers. The result is a best-effort
/// preview — partial patches render only the visible windows, which is
/// usually enough to eyeball documentation changes without a second
/// round-trip for the full file.
class MarkdownPreview extends StatelessWidget {
  const MarkdownPreview({super.key, required this.patch});

  final String patch;

  @override
  Widget build(BuildContext context) {
    final reconstructed = _reconstruct(patch);
    if (reconstructed.trim().isEmpty) {
      return const Padding(
        padding: EdgeInsets.all(12),
        child: Text('(nothing to preview)',
            style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
      );
    }
    return Container(
      padding: const EdgeInsets.all(12),
      color: AppColors.surface,
      width: double.infinity,
      child: MarkdownBody(
        data: reconstructed,
        selectable: true,
        styleSheet: MarkdownStyleSheet(
          p: const TextStyle(fontSize: 13, height: 1.45),
          code: const TextStyle(
              fontFamily: 'monospace', fontSize: 12, backgroundColor: Colors.transparent),
          codeblockDecoration: BoxDecoration(
            color: AppColors.bg,
            borderRadius: BorderRadius.circular(4),
          ),
        ),
      ),
    );
  }

  static String _reconstruct(String patch) {
    if (patch.isEmpty) return '';
    final buf = StringBuffer();
    var inBody = false;
    for (final raw in patch.split('\n')) {
      if (raw.startsWith('@@')) {
        inBody = true;
        continue;
      }
      // Skip every git header line until the first hunk.
      if (!inBody) continue;
      if (raw.startsWith('-')) continue;
      if (raw.startsWith('+')) {
        buf.writeln(raw.substring(1));
      } else if (raw.startsWith(' ')) {
        buf.writeln(raw.substring(1));
      } else if (raw.isEmpty) {
        buf.writeln();
      }
    }
    return buf.toString();
  }
}
