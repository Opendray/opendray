import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../core/api/api_client.dart';
import '../core/models/session.dart';
import 'theme/app_theme.dart';

/// Picks a running session via a bottom sheet. Returns null if the user
/// cancelled or no running sessions exist.
Future<Session?> pickSession(BuildContext context,
    {String title = 'Send to session'}) async {
  final api = context.read<ApiClient>();
  List<Session> sessions;
  try {
    sessions = (await api.listSessions()).where((s) => s.isRunning).toList();
  } catch (_) {
    if (context.mounted) _snack(context, 'Could not load sessions');
    return null;
  }
  if (sessions.isEmpty) {
    if (context.mounted) _snack(context, 'No running sessions — start one first');
    return null;
  }
  if (!context.mounted) return null;
  return showModalBottomSheet<Session>(
    context: context,
    backgroundColor: AppColors.surface,
    shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
    builder: (ctx) => SafeArea(child: Column(mainAxisSize: MainAxisSize.min, children: [
      Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
        child: Row(children: [
          const Icon(Icons.send_outlined, size: 16, color: AppColors.accent),
          const SizedBox(width: 8),
          Text(title,
              style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
        ]),
      ),
      const Divider(height: 1),
      Flexible(child: ListView.separated(
        shrinkWrap: true,
        padding: const EdgeInsets.symmetric(vertical: 4),
        itemCount: sessions.length,
        separatorBuilder: (_, _) => const Divider(height: 1, indent: 16),
        itemBuilder: (_, i) {
          final s = sessions[i];
          return ListTile(
            dense: true,
            leading: const Icon(Icons.terminal, size: 18, color: AppColors.accent),
            title: Text(s.name.isNotEmpty ? s.name : s.sessionType,
                style: const TextStyle(fontSize: 13)),
            subtitle: Text(s.sessionType,
                style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
            trailing: Container(
              width: 7, height: 7,
              decoration: const BoxDecoration(
                shape: BoxShape.circle, color: AppColors.success),
            ),
            onTap: () => Navigator.pop(ctx, s),
          );
        },
      )),
      const SizedBox(height: 8),
    ])),
  );
}

/// Signature for a function that pipes text through the *live* terminal
/// WebSocket (e.g. MobileTerminalView.sendKey). When provided, "Insert into
/// terminal" uses it instead of the HTTP /input endpoint — this guarantees
/// the text is echoed back to the on-screen xterm.js view immediately,
/// rather than waiting for the WebSocket to wake up after the iOS image
/// picker has suspended it.
typedef TerminalInserter = Future<void> Function(String text);

/// Uploads image bytes, then shows a preview sheet with explicit Insert /
/// Copy / Close actions. Nothing gets typed into the terminal automatically.
///
/// Pass [inserter] from a session-page caller so the path appears on-screen
/// without needing to navigate away and back.
Future<void> sendImageToSession({
  required BuildContext context,
  required Uint8List bytes,
  String mimeType = 'image/png',
  Session? targetSession,
  TerminalInserter? inserter,
}) async {
  if (bytes.isEmpty) { _snack(context, 'Empty image'); return; }
  final session = targetSession ?? await pickSession(context);
  if (session == null || !context.mounted) return;

  final api = context.read<ApiClient>();

  // Upload
  Map<String, dynamic> res;
  try {
    res = await api.attachImage(session.id, bytes, mimeType: mimeType);
  } catch (e) {
    if (context.mounted) _snack(context, 'Upload failed: $e');
    return;
  }
  if (!context.mounted) return;

  final path = res['path'] as String? ?? '';
  final name = res['name'] as String? ?? '';
  final size = (res['size'] as num?)?.toInt() ?? bytes.length;

  // Show the preview sheet — user picks what to do with the path.
  await _showResultSheet(
    context: context,
    session: session,
    bytes: bytes,
    path: path,
    name: name,
    size: size,
    inserter: inserter,
  );
}

/// Shows gallery/camera picker, then routes through [sendImageToSession].
Future<void> pickAndSendImage(BuildContext context,
    {Session? targetSession, TerminalInserter? inserter}) async {
  final picker = ImagePicker();
  final source = await showModalBottomSheet<ImageSource>(
    context: context,
    backgroundColor: AppColors.surface,
    shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
    builder: (ctx) => SafeArea(child: Column(mainAxisSize: MainAxisSize.min, children: [
      ListTile(
        leading: const Icon(Icons.photo_library_outlined, color: AppColors.accent),
        title: const Text('Photo Library', style: TextStyle(fontSize: 14)),
        onTap: () => Navigator.pop(ctx, ImageSource.gallery),
      ),
      ListTile(
        leading: const Icon(Icons.camera_alt_outlined, color: AppColors.accent),
        title: const Text('Take Photo', style: TextStyle(fontSize: 14)),
        onTap: () => Navigator.pop(ctx, ImageSource.camera),
      ),
      ListTile(
        leading: const Icon(Icons.close, color: AppColors.textMuted),
        title: const Text('Cancel',
            style: TextStyle(fontSize: 14, color: AppColors.textMuted)),
        onTap: () => Navigator.pop(ctx),
      ),
    ])),
  );
  if (source == null || !context.mounted) return;

  try {
    final XFile? picked = await picker.pickImage(
      source: source, maxWidth: 3000, imageQuality: 90,
    );
    if (picked == null || !context.mounted) return;
    final imgBytes = await picked.readAsBytes();
    final mime = _mimeFromPath(picked.path);
    if (!context.mounted) return;
    await sendImageToSession(
      context: context,
      bytes: imgBytes,
      mimeType: mime,
      targetSession: targetSession,
      inserter: inserter,
    );
  } catch (e) {
    if (context.mounted) _snack(context, 'Pick failed: $e');
  }
}

/// Result sheet — preview + path + action buttons. This is where the user
/// explicitly decides how to use the uploaded image.
Future<void> _showResultSheet({
  required BuildContext context,
  required Session session,
  required Uint8List bytes,
  required String path,
  required String name,
  required int size,
  TerminalInserter? inserter,
}) {
  final sizeKb = (size / 1024).toStringAsFixed(1);

  Future<void> insert({bool withNewline = false}) async {
    final suffix = withNewline ? '\n' : ' ';
    final text = path + suffix;
    try {
      if (inserter != null) {
        // Use the live WebView WebSocket — path appears on-screen immediately.
        await inserter(text);
      } else {
        // Fallback: HTTP /input (used from preview page where the target
        // session's terminal WebView isn't currently on screen).
        final api = context.read<ApiClient>();
        await api.sendInput(session.id, text);
      }
      if (context.mounted) {
        Navigator.pop(context);
        _snack(context, 'Path inserted into ${_sessionLabel(session)}');
      }
    } catch (e) {
      if (context.mounted) _snack(context, 'Insert failed: $e');
    }
  }

  Future<void> copy() async {
    await Clipboard.setData(ClipboardData(text: path));
    if (context.mounted) _snack(context, 'Path copied to clipboard');
  }

  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: AppColors.surface,
    shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
    builder: (ctx) => SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          // Header
          Row(children: [
            const Icon(Icons.check_circle_outline, size: 18, color: AppColors.success),
            const SizedBox(width: 8),
            const Text('Image uploaded',
                style: TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
            const Spacer(),
            Text('$sizeKb KB',
                style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
          ]),
          const SizedBox(height: 12),

          // Preview
          Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 200, maxWidth: double.infinity),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(8),
                child: Image.memory(bytes, fit: BoxFit.contain),
              ),
            ),
          ),
          const SizedBox(height: 12),

          // Path — monospace, tap to copy
          InkWell(
            onTap: copy,
            borderRadius: BorderRadius.circular(6),
            child: Container(
              width: double.infinity,
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: AppColors.surfaceAlt,
                borderRadius: BorderRadius.circular(6),
              ),
              child: Row(children: [
                Expanded(
                  child: Text(path,
                      style: const TextStyle(fontSize: 11, fontFamily: 'monospace'),
                      maxLines: 2, overflow: TextOverflow.ellipsis),
                ),
                const SizedBox(width: 8),
                const Icon(Icons.copy, size: 14, color: AppColors.textMuted),
              ]),
            ),
          ),
          const SizedBox(height: 12),

          // Session pill
          Row(children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
              decoration: BoxDecoration(
                color: AppColors.accentSoft,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Row(mainAxisSize: MainAxisSize.min, children: [
                const Icon(Icons.terminal, size: 12, color: AppColors.accent),
                const SizedBox(width: 4),
                Text(_sessionLabel(session),
                    style: const TextStyle(fontSize: 11, color: AppColors.accent)),
              ]),
            ),
          ]),
          const SizedBox(height: 14),

          // Actions
          Row(children: [
            Expanded(child: OutlinedButton.icon(
              onPressed: copy,
              icon: const Icon(Icons.copy, size: 14),
              label: const Text('Copy path', style: TextStyle(fontSize: 12)),
            )),
            const SizedBox(width: 8),
            Expanded(flex: 2, child: FilledButton.icon(
              onPressed: () => insert(),
              style: FilledButton.styleFrom(
                backgroundColor: AppColors.accent,
                foregroundColor: Colors.white,
              ),
              icon: const Icon(Icons.keyboard_return, size: 14),
              label: const Text('Insert into terminal',
                  style: TextStyle(fontSize: 12)),
            )),
          ]),
          const SizedBox(height: 6),
          Align(
            alignment: Alignment.centerRight,
            child: TextButton(
              onPressed: () => Navigator.pop(ctx),
              style: TextButton.styleFrom(foregroundColor: AppColors.textMuted),
              child: const Text('Close', style: TextStyle(fontSize: 11)),
            ),
          ),
        ]),
      ),
    ),
  );
}

String _sessionLabel(Session s) =>
    s.name.isNotEmpty ? s.name : s.sessionType;

String _mimeFromPath(String path) {
  final p = path.toLowerCase();
  if (p.endsWith('.jpg') || p.endsWith('.jpeg')) return 'image/jpeg';
  if (p.endsWith('.webp')) return 'image/webp';
  if (p.endsWith('.gif'))  return 'image/gif';
  if (p.endsWith('.heic')) return 'image/heic';
  return 'image/png';
}

void _snack(BuildContext context, String msg) {
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(content: Text(msg), duration: const Duration(seconds: 2)),
  );
}
