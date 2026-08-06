import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// A colour picker, because "type a colour code" is exactly the thing a design
// system must not ask of a non-designer. Tapping a swatch opens this: a
// saturation/value field, a hue rail, and a hex box for when someone does know
// the value. Flutter ships no picker and pulling a package in for one sheet
// isn't worth the dependency, so it's a small CustomPaint.

/// Opens the picker and returns the chosen colour as `#rrggbb`, or null if the
/// operator backed out.
Future<String?> pickCanvasColor(BuildContext context, Color initial) {
  return showDialog<String>(
    context: context,
    builder: (_) => _ColorPickerDialog(initial: initial),
  );
}

/// Parses a hex colour (`#rgb` / `#rrggbb`). Anything else — oklch(), a named
/// colour — returns null: the field stays text, and the picker starts neutral.
Color? parseHexColor(String value) {
  final v = value.trim();
  if (!v.startsWith('#')) return null;
  final hex = v.substring(1);
  final full = switch (hex.length) {
    3 => hex.split('').map((c) => '$c$c').join(),
    6 => hex,
    _ => null,
  };
  if (full == null) return null;
  final n = int.tryParse(full, radix: 16);
  return n == null ? null : Color(0xFF000000 | n);
}

String hexOf(Color c) {
  final r = (c.r * 255).round();
  final g = (c.g * 255).round();
  final b = (c.b * 255).round();
  return '#${r.toRadixString(16).padLeft(2, '0')}'
      '${g.toRadixString(16).padLeft(2, '0')}'
      '${b.toRadixString(16).padLeft(2, '0')}';
}

class _ColorPickerDialog extends StatefulWidget {
  const _ColorPickerDialog({required this.initial});

  final Color initial;

  @override
  State<_ColorPickerDialog> createState() => _ColorPickerDialogState();
}

class _ColorPickerDialogState extends State<_ColorPickerDialog> {
  late HSVColor _hsv;
  late final TextEditingController _hexCtl;

  @override
  void initState() {
    super.initState();
    _hsv = HSVColor.fromColor(widget.initial);
    _hexCtl = TextEditingController(text: hexOf(widget.initial));
  }

  @override
  void dispose() {
    _hexCtl.dispose();
    super.dispose();
  }

  void _set(HSVColor v, {bool syncHex = true}) {
    setState(() => _hsv = v);
    if (syncHex) _hexCtl.text = hexOf(v.toColor());
  }

  @override
  Widget build(BuildContext context) {
    final color = _hsv.toColor();
    return AlertDialog(
      contentPadding: const EdgeInsets.fromLTRB(20, 20, 20, 8),
      content: SizedBox(
        width: 300,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Saturation (x) × value (y) for the current hue.
            SizedBox(
              height: 168,
              child: LayoutBuilder(
                builder: (context, c) => GestureDetector(
                  behavior: HitTestBehavior.opaque,
                  onPanDown: (d) => _onField(d.localPosition, c.biggest),
                  onPanUpdate: (d) => _onField(d.localPosition, c.biggest),
                  child: CustomPaint(
                    painter: _FieldPainter(_hsv),
                    size: c.biggest,
                  ),
                ),
              ),
            ),
            const SizedBox(height: 14),
            SizedBox(
              height: 26,
              child: LayoutBuilder(
                builder: (context, c) => GestureDetector(
                  behavior: HitTestBehavior.opaque,
                  onPanDown: (d) => _onHue(d.localPosition.dx, c.maxWidth),
                  onPanUpdate: (d) => _onHue(d.localPosition.dx, c.maxWidth),
                  child: CustomPaint(
                    painter: _HuePainter(_hsv.hue),
                    size: Size(c.maxWidth, 26),
                  ),
                ),
              ),
            ),
            const SizedBox(height: 14),
            Row(
              children: [
                Container(
                  width: 34,
                  height: 34,
                  decoration: BoxDecoration(
                    color: color,
                    borderRadius: BorderRadius.circular(6),
                    border: Border.all(color: Theme.of(context).dividerColor),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: TextField(
                    controller: _hexCtl,
                    style: const TextStyle(fontFamily: 'monospace'),
                    inputFormatters: [LengthLimitingTextInputFormatter(7)],
                    decoration: const InputDecoration(
                      isDense: true,
                      border: OutlineInputBorder(),
                      prefixText: '',
                    ),
                    onChanged: (v) {
                      final c = parseHexColor(v);
                      if (c != null) _set(HSVColor.fromColor(c), syncHex: false);
                    },
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(MaterialLocalizations.of(context).cancelButtonLabel),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(hexOf(color)),
          child: Text(t.sessions.inspector.canvas.designSave),
        ),
      ],
    );
  }

  void _onField(Offset p, Size size) {
    if (size.isEmpty) return;
    _set(
      _hsv.withSaturation((p.dx / size.width).clamp(0.0, 1.0)).withValue(
            (1 - p.dy / size.height).clamp(0.0, 1.0),
          ),
    );
  }

  void _onHue(double dx, double width) {
    if (width <= 0) return;
    _set(_hsv.withHue(((dx / width).clamp(0.0, 1.0)) * 360));
  }
}

class _FieldPainter extends CustomPainter {
  const _FieldPainter(this.hsv);

  final HSVColor hsv;

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final rrect = RRect.fromRectAndRadius(rect, const Radius.circular(8));
    canvas
      ..save()
      ..clipRRect(rrect)
      // White → the pure hue across x, then black over y.
      ..drawRect(
        rect,
        Paint()
          ..shader = LinearGradient(
            colors: [
              Colors.white,
              HSVColor.fromAHSV(1, hsv.hue, 1, 1).toColor(),
            ],
          ).createShader(rect),
      )
      ..drawRect(
        rect,
        Paint()
          ..shader = const LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [Colors.transparent, Colors.black],
          ).createShader(rect),
      )
      ..restore();

    final p = Offset(hsv.saturation * size.width, (1 - hsv.value) * size.height);
    canvas
      ..drawCircle(p, 9, Paint()..color = Colors.white.withValues(alpha: 0.9)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2)
      ..drawCircle(p, 11, Paint()..color = Colors.black.withValues(alpha: 0.35)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1);
  }

  @override
  bool shouldRepaint(_FieldPainter old) => old.hsv != hsv;
}

class _HuePainter extends CustomPainter {
  const _HuePainter(this.hue);

  final double hue;

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final rrect = RRect.fromRectAndRadius(rect, const Radius.circular(13));
    canvas
      ..save()
      ..clipRRect(rrect)
      ..drawRect(
        rect,
        Paint()
          ..shader = LinearGradient(
            colors: [
              for (var i = 0; i <= 6; i++)
                HSVColor.fromAHSV(1, i * 60.0 % 360, 1, 1).toColor(),
            ],
          ).createShader(rect),
      )
      ..restore();

    final x = (hue / 360) * size.width;
    canvas
      ..drawCircle(Offset(x, size.height / 2), 9,
          Paint()..color = Colors.white.withValues(alpha: 0.95)..style = PaintingStyle.stroke..strokeWidth = 2)
      ..drawCircle(Offset(x, size.height / 2), 11,
          Paint()..color = Colors.black.withValues(alpha: 0.35)..style = PaintingStyle.stroke..strokeWidth = 1);
  }

  @override
  bool shouldRepaint(_HuePainter old) => old.hue != hue;
}
