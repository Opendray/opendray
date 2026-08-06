import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/canvas_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// CanvasDesignSheet — the project's canvas design contract.
//
// It exists because an agent asked to "design a login page" re-invents colours,
// type and spacing every time, so successive canvases drift apart. Whatever is
// set here rides along with every canvas request AND is injected into each
// rendered canvas as CSS variables, which is why the agent is told to write
// var(--od-primary) rather than a hex value: change a token later and every
// canvas that used the variable restyles itself.
//
// Hand-editing works, but the accurate route is to ask the agent to read the
// project's real theme and write it through the canvas_design MCP tool.

/// Tokens that get a colour swatch; the rest are plain text.
const _colorTokens = {
  'primary',
  'secondary',
  'background',
  'surface',
  'text',
  'muted',
  'border',
};

String _tokenLabel(String key) => switch (key) {
      'primary' => t.sessions.inspector.canvas.tokenPrimary,
      'secondary' => t.sessions.inspector.canvas.tokenSecondary,
      'background' => t.sessions.inspector.canvas.tokenBackground,
      'surface' => t.sessions.inspector.canvas.tokenSurface,
      'text' => t.sessions.inspector.canvas.tokenText,
      'muted' => t.sessions.inspector.canvas.tokenMuted,
      'border' => t.sessions.inspector.canvas.tokenBorder,
      'font' => t.sessions.inspector.canvas.tokenFont,
      'headingFont' => t.sessions.inspector.canvas.tokenHeadingFont,
      'baseSize' => t.sessions.inspector.canvas.tokenBaseSize,
      'radius' => t.sessions.inspector.canvas.tokenRadius,
      'spacing' => t.sessions.inspector.canvas.tokenSpacing,
      'shadow' => t.sessions.inspector.canvas.tokenShadow,
      _ => key,
    };

/// Mirrors the gateway's naming so the sheet can show the variable the agent
/// is told to use: headingFont → --od-heading-font.
String cssVarForToken(String key) =>
    '--od-${key.replaceAllMapped(RegExp('[A-Z]'), (m) => '-${m[0]!.toLowerCase()}')}';

/// Parses a CSS colour well enough to preview a swatch (#rgb / #rrggbb).
Color? _swatch(String value) {
  final v = value.trim();
  if (!v.startsWith('#')) return null;
  final hex = v.substring(1);
  final full = hex.length == 3
      ? hex.split('').map((c) => '$c$c').join()
      : hex.length == 6
          ? hex
          : null;
  if (full == null) return null;
  final n = int.tryParse(full, radix: 16);
  return n == null ? null : Color(0xFF000000 | n);
}

class CanvasDesignSheet extends ConsumerStatefulWidget {
  const CanvasDesignSheet({required this.cwd, super.key});

  final String cwd;

  @override
  ConsumerState<CanvasDesignSheet> createState() => _CanvasDesignSheetState();
}

class _CanvasDesignSheetState extends ConsumerState<CanvasDesignSheet> {
  final Map<String, TextEditingController> _ctls = {};
  final _notesCtl = TextEditingController();
  bool _loading = true;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    for (final k in canvasDesignTokens) {
      _ctls[k] = TextEditingController();
    }
    unawaited(_load());
  }

  @override
  void dispose() {
    for (final c in _ctls.values) {
      c.dispose();
    }
    _notesCtl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final d = await ref.read(canvasApiProvider).getDesign(widget.cwd);
      if (!mounted) return;
      setState(() {
        d.tokens.forEach((k, v) => _ctls[k]?.text = v);
        _notesCtl.text = d.notes;
        _loading = false;
      });
    } on Object catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final tokens = <String, String>{};
      _ctls.forEach((k, c) {
        if (c.text.trim().isNotEmpty) tokens[k] = c.text.trim();
      });
      await ref.read(canvasApiProvider).setDesign(
            cwd: widget.cwd,
            tokens: tokens,
            notes: _notesCtl.text.trim(),
          );
      if (!mounted) return;
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.canvas.designSaved)),
      );
    } on Object catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.shared.insertFailedGeneric(error: '$e'),
          ),
        ),
      );
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: FractionallySizedBox(
        heightFactor: 0.92,
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 14, 8, 6),
              child: Row(
                children: [
                  Icon(Icons.palette_outlined,
                      size: 20, color: theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          t.sessions.inspector.canvas.designTitle,
                          style: theme.textTheme.titleMedium,
                        ),
                        Text(
                          t.sessions.inspector.canvas.designBlurb,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: () => Navigator.of(context).pop(),
                  ),
                ],
              ),
            ),
            Divider(height: 1, color: theme.dividerColor),
            Expanded(
              child: _loading
                  ? const Center(child: CircularProgressIndicator())
                  : ListView(
                      padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
                      children: [
                        for (final key in canvasDesignTokens)
                          _TokenField(
                            label: _tokenLabel(key),
                            hint: cssVarForToken(key),
                            controller: _ctls[key]!,
                            showSwatch: _colorTokens.contains(key),
                          ),
                        const SizedBox(height: 12),
                        Text(
                          t.sessions.inspector.canvas.designNotesLabel,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                        const SizedBox(height: 6),
                        TextField(
                          controller: _notesCtl,
                          minLines: 3,
                          maxLines: 6,
                          decoration: InputDecoration(
                            isDense: true,
                            border: const OutlineInputBorder(),
                            hintText: t
                                .sessions.inspector.canvas.designNotesPlaceholder,
                          ),
                        ),
                        const SizedBox(height: 14),
                        Text(
                          t.sessions.inspector.canvas.designAgentHint,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
            ),
            SafeArea(
              top: false,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 10),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    TextButton(
                      onPressed: () => Navigator.of(context).pop(),
                      child: Text(
                        MaterialLocalizations.of(context).cancelButtonLabel,
                      ),
                    ),
                    const SizedBox(width: 8),
                    FilledButton(
                      onPressed: _saving || _loading ? null : () => unawaited(_save()),
                      child: Text(t.sessions.inspector.canvas.designSave),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _TokenField extends StatelessWidget {
  const _TokenField({
    required this.label,
    required this.hint,
    required this.controller,
    required this.showSwatch,
  });

  final String label;
  final String hint;
  final TextEditingController controller;
  final bool showSwatch;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          SizedBox(
            width: 96,
            child: Text(label, style: Theme.of(context).textTheme.bodySmall),
          ),
          if (showSwatch)
            ValueListenableBuilder<TextEditingValue>(
              valueListenable: controller,
              builder: (context, value, _) {
                final c = _swatch(value.text);
                return Container(
                  width: 18,
                  height: 18,
                  margin: const EdgeInsets.only(right: 8),
                  decoration: BoxDecoration(
                    color: c ?? Theme.of(context).colorScheme.surfaceContainerHighest,
                    borderRadius: BorderRadius.circular(4),
                    border: Border.all(color: Theme.of(context).dividerColor),
                  ),
                );
              },
            ),
          Expanded(
            child: TextField(
              controller: controller,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
              decoration: InputDecoration(
                isDense: true,
                border: const OutlineInputBorder(),
                hintText: hint,
                hintStyle: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
