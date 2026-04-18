import 'dart:js_interop';
import 'dart:js_interop_unsafe';
import 'package:web/web.dart' as web;

/// True when the current browser exposes SpeechRecognition (Chromium/Safari).
/// Firefox still doesn't ship it (2026-04), so we hide the button there.
bool get webSpeechSupported {
  final w = web.window as JSObject;
  return w.hasProperty('SpeechRecognition'.toJS).toDart ||
      w.hasProperty('webkitSpeechRecognition'.toJS).toDart;
}

/// Thin wrapper over Web Speech API's SpeechRecognition. Calls [onResult]
/// with (transcript, isFinal) for every recognition event. [onEnd] fires
/// when the browser stops recognition (user stopped, timeout, or error).
class WebSpeechSession {
  final void Function(String text, bool isFinal) onResult;
  final void Function() onEnd;
  final void Function(String error) onError;
  final String lang;

  JSObject? _rec;
  bool _started = false;

  WebSpeechSession({
    required this.onResult,
    required this.onEnd,
    required this.onError,
    this.lang = 'en-US',
  });

  void start() {
    if (_started) return;
    try {
      final w = web.window as JSObject;
      final ctor = w.hasProperty('SpeechRecognition'.toJS).toDart
          ? w.getProperty('SpeechRecognition'.toJS)
          : w.getProperty('webkitSpeechRecognition'.toJS);
      if (ctor == null) {
        onError('SpeechRecognition not available');
        return;
      }
      final rec = (ctor as JSFunction).callAsConstructor() as JSObject;
      rec.setProperty('continuous'.toJS, true.toJS);
      rec.setProperty('interimResults'.toJS, true.toJS);
      rec.setProperty('lang'.toJS, lang.toJS);

      rec.setProperty(
        'onresult'.toJS,
        ((JSAny evt) {
          try {
            final e = evt as JSObject;
            final results = e.getProperty('results'.toJS) as JSObject;
            final resIdx = (e.getProperty('resultIndex'.toJS) as JSNumber)
                .toDartInt;
            final length =
                (results.getProperty('length'.toJS) as JSNumber).toDartInt;
            for (var i = resIdx; i < length; i++) {
              final result =
                  results.getProperty(i.toString().toJS) as JSObject;
              final alt = result.getProperty('0'.toJS) as JSObject;
              final transcript =
                  (alt.getProperty('transcript'.toJS) as JSString).toDart;
              final isFinal =
                  (result.getProperty('isFinal'.toJS) as JSBoolean).toDart;
              onResult(transcript, isFinal);
            }
          } catch (_) {}
        }).toJS,
      );

      rec.setProperty(
        'onerror'.toJS,
        ((JSAny evt) {
          try {
            final e = evt as JSObject;
            final err = (e.getProperty('error'.toJS) as JSString?)?.toDart ??
                'unknown';
            onError(err);
          } catch (_) {
            onError('unknown');
          }
        }).toJS,
      );

      rec.setProperty(
        'onend'.toJS,
        (() {
          _started = false;
          onEnd();
        }).toJS,
      );

      (rec.getProperty('start'.toJS) as JSFunction).callMethod('call'.toJS, rec);
      _started = true;
      _rec = rec;
    } catch (e) {
      onError(e.toString());
    }
  }

  void stop() {
    final rec = _rec;
    if (rec == null || !_started) return;
    try {
      (rec.getProperty('stop'.toJS) as JSFunction).callMethod('call'.toJS, rec);
    } catch (_) {}
  }

  void dispose() {
    stop();
    _rec = null;
  }
}
