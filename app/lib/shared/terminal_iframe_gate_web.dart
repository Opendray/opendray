import 'package:web/web.dart' as web;

/// Reference-counted toggle that disables pointer events on every iframe
/// in the document while at least one modal is open. Re-enables them once
/// the last modal dismisses.
///
/// Why: on Flutter Web, iframes rendered via HtmlElementView sit in their
/// own DOM layer and absorb pointer events before they reach Flutter's
/// canvas. That makes barrier-tap-to-dismiss and even close-button clicks
/// unreliable. Turning off `pointer-events` on the iframe while a modal is
/// open gives Flutter clean input for the duration of the modal.

int _count = 0;

void _setIframesPointerEvents(String value) {
  final iframes = web.document.querySelectorAll('iframe');
  for (var i = 0; i < iframes.length; i++) {
    final el = iframes.item(i);
    if (el == null) continue;
    // Every querySelectorAll('iframe') match is an HTMLIFrameElement which
    // IS-A HTMLElement — it has a `.style` property at runtime. The Dart
    // type system can't see that through the generic Node? return, hence
    // the unchecked cast.
    (el as web.HTMLElement).style.pointerEvents = value;
  }
}

void pushModalIframeMute() {
  _count++;
  if (_count == 1) _setIframesPointerEvents('none');
}

void popModalIframeMute() {
  if (_count == 0) return;
  _count--;
  if (_count == 0) _setIframesPointerEvents('');
}
