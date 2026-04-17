/// Platform-conditional export for WebTerminalView.
/// Web builds get the xterm.js iframe implementation.
/// Mobile builds get a stub (never rendered due to kIsWeb gate in session_page).
export 'web_terminal_stub.dart'
    if (dart.library.js_interop) 'web_terminal_web.dart';
