import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// showTerminalSelectSheet presents the terminal output as native,
// selectable text in a tall bottom sheet.
//
// The live terminal is a custom-painted xterm canvas: the OS selection
// handles don't reliably appear over it, and while a TUI has mouse
// tracking on (Claude Code / Codex / Gemini all do) a long-press is
// forwarded to the program as a mouse event instead of starting a
// selection. Rendering the buffer into a SelectableText sidesteps both
// — long-press brings up the system selection handles, so the operator
// can grab just a command or a URL and copy it, mirroring the web
// "Select & copy" dialog.
Future<void> showTerminalSelectSheet(
  BuildContext context,
  String text,
) {
  return showModalBottomSheet<void>(
    context: context,
    backgroundColor: const Color(0xFF1A1A1F),
    isScrollControlled: true,
    builder: (sheetCtx) {
      final empty = text.trim().isEmpty;
      return SafeArea(
        child: FractionallySizedBox(
          heightFactor: 0.85,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  t.sessions.terminal.selectCopy.title,
                  style: const TextStyle(
                    color: Color(0xFFE7E7EA),
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  t.sessions.terminal.selectCopy.hint,
                  style: const TextStyle(
                    color: Color(0xFF9999A0),
                    fontSize: 12,
                  ),
                ),
                const SizedBox(height: 12),
                Expanded(
                  child: Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(10),
                    decoration: BoxDecoration(
                      color: const Color(0xFF101012),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: empty
                        ? Center(
                            child: Text(
                              t.sessions.terminal.selectCopy.empty,
                              style: const TextStyle(
                                color: Color(0xFF9999A0),
                                fontSize: 12,
                              ),
                            ),
                          )
                        : SingleChildScrollView(
                            child: SelectableText(
                              text,
                              style: const TextStyle(
                                color: Color(0xFFE7E7EA),
                                fontSize: 12,
                                fontFamily: 'monospace',
                                height: 1.35,
                              ),
                            ),
                          ),
                  ),
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: FilledButton.icon(
                        icon: const Icon(Icons.copy_all, size: 16),
                        label: Text(t.sessions.terminal.selectCopy.copyAll),
                        onPressed: empty
                            ? null
                            : () async {
                                await Clipboard.setData(
                                  ClipboardData(text: text),
                                );
                                if (!sheetCtx.mounted) return;
                                ScaffoldMessenger.of(sheetCtx).showSnackBar(
                                  SnackBar(
                                    content: Text(
                                      t.sessions.terminal.selectCopy
                                          .copiedAll(count: text.length),
                                    ),
                                    behavior: SnackBarBehavior.floating,
                                    duration: const Duration(seconds: 2),
                                  ),
                                );
                                Navigator.of(sheetCtx).pop();
                              },
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: OutlinedButton(
                        onPressed: () => Navigator.of(sheetCtx).pop(),
                        child: Text(t.common.close),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      );
    },
  );
}
