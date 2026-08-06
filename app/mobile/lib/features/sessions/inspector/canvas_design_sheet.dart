import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/canvas_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/sessions/inspector/canvas_color_picker.dart';

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

String _paletteLabel(String id) => switch (id) {
      'indigo' => t.sessions.inspector.canvas.paletteIndigo,
      'sky' => t.sessions.inspector.canvas.paletteSky,
      'emerald' => t.sessions.inspector.canvas.paletteEmerald,
      'amber' => t.sessions.inspector.canvas.paletteAmber,
      'rose' => t.sessions.inspector.canvas.paletteRose,
      'violet' => t.sessions.inspector.canvas.paletteViolet,
      'graphite' => t.sessions.inspector.canvas.paletteGraphite,
      _ => id,
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

class CanvasDesignSheet extends ConsumerStatefulWidget {
  const CanvasDesignSheet({
    required this.sessionId,
    required this.cwd,
    super.key,
  });

  final String sessionId;
  final String cwd;

  @override
  ConsumerState<CanvasDesignSheet> createState() => _CanvasDesignSheetState();
}

class _CanvasDesignSheetState extends ConsumerState<CanvasDesignSheet> {
  final Map<String, TextEditingController> _ctls = {};
  final Map<String, TextEditingController> _darkCtls = {};
  final _notesCtl = TextEditingController();
  bool _dark = false;
  bool _loading = true;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    for (final k in canvasDesignTokens) {
      _ctls[k] = TextEditingController();
    }
    for (final k in canvasThemedTokens) {
      _darkCtls[k] = TextEditingController();
    }
    unawaited(_load());
  }

  @override
  void dispose() {
    for (final c in _ctls.values) {
      c.dispose();
    }
    for (final c in _darkCtls.values) {
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
        d.tokensDark.forEach((k, v) => _darkCtls[k]?.text = v);
        _notesCtl.text = d.notes;
        _loading = false;
      });
    } on Object catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  /// Hand a one-click job to the agent: read the project's real theme and
  /// record it, or draw the system as a canvas.
  Future<void> _task(String kind) async {
    try {
      await ref.read(canvasApiProvider).runDesignTask(
            sessionId: widget.sessionId,
            cwd: widget.cwd,
            task: kind,
          );
      if (!mounted) return;
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.canvas.designTaskSent)),
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
    }
  }

  /// One tap fills BOTH themes with a checked pair — the answer to "I don't
  /// know what colours to pick".
  void _applyPalette(CanvasPalette p) {
    setState(() {
      p.tokens.forEach((k, v) => _ctls[k]?.text = v);
      for (final k in canvasThemedTokens) {
        _darkCtls[k]?.text = p.tokensDark[k] ?? '';
      }
    });
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final tokens = <String, String>{};
      _ctls.forEach((k, c) {
        if (c.text.trim().isNotEmpty) tokens[k] = c.text.trim();
      });
      final dark = <String, String>{};
      _darkCtls.forEach((k, c) {
        if (c.text.trim().isNotEmpty) dark[k] = c.text.trim();
      });
      await ref.read(canvasApiProvider).setDesign(
            cwd: widget.cwd,
            tokens: tokens,
            tokensDark: dark,
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
                        Wrap(
                          spacing: 8,
                          runSpacing: 8,
                          children: [
                            FilledButton.tonalIcon(
                              onPressed: () => unawaited(_task('extract')),
                              icon: const Icon(Icons.travel_explore, size: 16),
                              label:
                                  Text(t.sessions.inspector.canvas.extractBtn),
                            ),
                            OutlinedButton.icon(
                              onPressed: () => unawaited(_task('showcase')),
                              icon: const Icon(Icons.dashboard_outlined, size: 16),
                              label:
                                  Text(t.sessions.inspector.canvas.showcaseBtn),
                            ),
                          ],
                        ),
                        const SizedBox(height: 16),
                        Text(
                          t.sessions.inspector.canvas.paletteLabel,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                        const SizedBox(height: 8),
                        Wrap(
                          spacing: 6,
                          runSpacing: 6,
                          children: [
                            for (final p in canvasPalettes)
                              ActionChip(
                                avatar: CircleAvatar(
                                  radius: 7,
                                  backgroundColor: Color(p.accent),
                                ),
                                label: Text(_paletteLabel(p.id)),
                                onPressed: () => _applyPalette(p),
                              ),
                          ],
                        ),
                        const SizedBox(height: 14),
                        SegmentedButton<bool>(
                          showSelectedIcon: false,
                          style: const ButtonStyle(
                            visualDensity: VisualDensity.compact,
                          ),
                          segments: [
                            ButtonSegment(
                              value: false,
                              icon: const Icon(Icons.light_mode_outlined, size: 15),
                              label: Text(t.sessions.inspector.canvas.themeLight),
                            ),
                            ButtonSegment(
                              value: true,
                              icon: const Icon(Icons.dark_mode_outlined, size: 15),
                              label: Text(t.sessions.inspector.canvas.themeDark),
                            ),
                          ],
                          selected: {_dark},
                          onSelectionChanged: (s) => setState(() => _dark = s.first),
                        ),
                        if (_dark)
                          Padding(
                            padding: const EdgeInsets.only(top: 6),
                            child: Text(
                              t.sessions.inspector.canvas.themeDarkHint,
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ),
                        const SizedBox(height: 12),
                        for (final key in (_dark
                            ? canvasThemedTokens
                            : canvasDesignTokens))
                          _TokenField(
                            label: _tokenLabel(key),
                            hint: _dark
                                ? (_ctls[key]?.text.isNotEmpty ?? false
                                    ? _ctls[key]!.text
                                    : cssVarForToken(key))
                                : cssVarForToken(key),
                            controller: (_dark ? _darkCtls : _ctls)[key]!,
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
                final c = parseHexColor(value.text);
                return Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: InkWell(
                    borderRadius: BorderRadius.circular(4),
                    // Tap the swatch to choose a colour — nobody should have to
                    // know a colour code to set a design system.
                    onTap: () async {
                      final picked = await pickCanvasColor(
                        context,
                        c ?? const Color(0xFF888888),
                      );
                      if (picked != null) controller.text = picked;
                    },
                    child: Container(
                      width: 22,
                      height: 22,
                      decoration: BoxDecoration(
                        color: c ??
                            Theme.of(context).colorScheme.surfaceContainerHighest,
                        borderRadius: BorderRadius.circular(4),
                        border: Border.all(color: Theme.of(context).dividerColor),
                      ),
                      child: c == null
                          ? Icon(Icons.colorize,
                              size: 13,
                              color: Theme.of(context).colorScheme.onSurfaceVariant)
                          : null,
                    ),
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
