import 'package:flutter/material.dart';

import '../core/services/l10n.dart';
import 'theme/app_theme.dart';

/// Opens a bottom-sheet composer optimised for the phone's built-in voice
/// dictation. The sheet autofocuses a multi-line [TextField] so the soft
/// keyboard pops up immediately — the user taps the IME's own mic button
/// (iOS Dictation / Gboard voice typing / SwiftKey / etc.) to dictate.
///
/// We deliberately do NOT run any STT in-app: the OS-level dictation is
/// already best-in-class for the user's language, works offline on recent
/// devices, and has zero battery / binary-size cost for us.
///
/// [onSend] is called with the final text. Return `true` on successful
/// delivery so the sheet can close; return `false` to keep it open with
/// the text preserved.
Future<void> showVoiceComposer(
  BuildContext context, {
  required Future<bool> Function(String text) onSend,
  String? initialText,
}) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    backgroundColor: AppColors.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
    ),
    builder: (_) => _VoiceComposerSheet(
      onSend: onSend,
      initialText: initialText ?? '',
    ),
  );
}

class _VoiceComposerSheet extends StatefulWidget {
  final Future<bool> Function(String) onSend;
  final String initialText;
  const _VoiceComposerSheet({required this.onSend, required this.initialText});

  @override
  State<_VoiceComposerSheet> createState() => _VoiceComposerSheetState();
}

class _VoiceComposerSheetState extends State<_VoiceComposerSheet> {
  late final TextEditingController _ctrl =
      TextEditingController(text: widget.initialText);
  final FocusNode _focus = FocusNode();
  bool _appendEnter = true;
  bool _sending = false;

  @override
  void initState() {
    super.initState();
    // Nudge the keyboard up on next frame — bottom sheet focus-on-init is
    // flaky on some Android skins.
    WidgetsBinding.instance.addPostFrameCallback((_) => _focus.requestFocus());
  }

  @override
  void dispose() {
    _ctrl.dispose();
    _focus.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    final text = _ctrl.text.trim();
    if (text.isEmpty || _sending) return;
    setState(() => _sending = true);
    final payload = _appendEnter ? '$text\n' : text;
    final ok = await widget.onSend(payload);
    if (!mounted) return;
    setState(() => _sending = false);
    if (ok) Navigator.of(context).maybePop();
  }

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;
    return Padding(
      padding: EdgeInsets.only(bottom: bottomInset),
      child: SafeArea(
        top: false,
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          _header(context),
          const Divider(height: 1, color: AppColors.border),
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 12, 14, 8),
            child: _hintRow(context),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 14),
            child: TextField(
              controller: _ctrl,
              focusNode: _focus,
              minLines: 3,
              maxLines: 8,
              keyboardType: TextInputType.multiline,
              textInputAction: TextInputAction.newline,
              style: const TextStyle(fontSize: 14),
              decoration: InputDecoration(
                hintText: context.tr('Tap the mic on your keyboard and speak…'),
                hintStyle: const TextStyle(color: AppColors.textMuted, fontSize: 13),
                filled: true,
                fillColor: AppColors.surfaceAlt,
                contentPadding: const EdgeInsets.all(12),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: BorderSide.none,
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: const BorderSide(color: AppColors.accent),
                ),
              ),
              onChanged: (_) => setState(() {}),
            ),
          ),
          _controlsRow(context),
          const SizedBox(height: 8),
        ]),
      ),
    );
  }

  Widget _header(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 14, 8, 12),
      child: Row(children: [
        const Icon(Icons.mic, size: 18, color: AppColors.accent),
        const SizedBox(width: 8),
        Text(
          context.tr('Voice / Dictation'),
          style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
        ),
        const Spacer(),
        IconButton(
          icon: const Icon(Icons.close, size: 18),
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
          onPressed: () => Navigator.of(context).maybePop(),
        ),
      ]),
    );
  }

  Widget _hintRow(BuildContext context) {
    return Row(children: [
      const Icon(Icons.info_outline, size: 13, color: AppColors.textMuted),
      const SizedBox(width: 6),
      Expanded(
        child: Text(
          context.tr(
              'Dictation uses your phone\'s built-in speech recognition. Review the text before sending.'),
          style: const TextStyle(fontSize: 11, color: AppColors.textMuted, height: 1.4),
        ),
      ),
    ]);
  }

  Widget _controlsRow(BuildContext context) {
    final canSend = _ctrl.text.trim().isNotEmpty && !_sending;
    return Padding(
      padding: const EdgeInsets.fromLTRB(10, 10, 10, 4),
      child: Row(children: [
        Expanded(
          child: CheckboxListTile(
            value: _appendEnter,
            onChanged: (v) => setState(() => _appendEnter = v ?? true),
            dense: true,
            contentPadding: EdgeInsets.zero,
            controlAffinity: ListTileControlAffinity.leading,
            visualDensity: VisualDensity.compact,
            title: Text(
              context.tr('Append Enter'),
              style: const TextStyle(fontSize: 12),
            ),
            subtitle: Text(
              context.tr('Sends as a command — a newline is added after the text.'),
              style: const TextStyle(fontSize: 10, color: AppColors.textMuted),
            ),
          ),
        ),
        const SizedBox(width: 8),
        FilledButton.icon(
          onPressed: canSend ? _send : null,
          icon: _sending
              ? const SizedBox(
                  width: 14, height: 14,
                  child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                )
              : const Icon(Icons.send, size: 16),
          label: Text(context.tr('Send')),
          style: FilledButton.styleFrom(
            backgroundColor: AppColors.accent,
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          ),
        ),
      ]),
    );
  }
}
