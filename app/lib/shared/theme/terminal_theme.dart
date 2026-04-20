import 'package:flutter/widgets.dart';
import 'package:xterm/xterm.dart';

/// OpenDray custom terminal theme — dark, high-contrast, inspired by iTerm2/Warp.
/// Colors tuned for readability on the app's #0B0D11 background.
const opendrayTerminalTheme = TerminalTheme(
  cursor: Color(0xFFA0A4B8),
  selection: Color(0x406366F1), // accent with transparency
  foreground: Color(0xFFD4D7E0),
  background: Color(0xFF0B0D11),
  black: Color(0xFF1C1F26),
  red: Color(0xFFF87171),
  green: Color(0xFF4ADE80),
  yellow: Color(0xFFFBBF24),
  blue: Color(0xFF60A5FA),
  magenta: Color(0xFFC084FC),
  cyan: Color(0xFF22D3EE),
  white: Color(0xFFE1E4ED),
  brightBlack: Color(0xFF6B7280),
  brightRed: Color(0xFFFCA5A5),
  brightGreen: Color(0xFF86EFAC),
  brightYellow: Color(0xFFFDE68A),
  brightBlue: Color(0xFF93C5FD),
  brightMagenta: Color(0xFFD8B4FE),
  brightCyan: Color(0xFF67E8F9),
  brightWhite: Color(0xFFF9FAFB),
  searchHitBackground: Color(0xFFFBBF24),
  searchHitBackgroundCurrent: Color(0xFF4ADE80),
  searchHitForeground: Color(0xFF0B0D11),
);
