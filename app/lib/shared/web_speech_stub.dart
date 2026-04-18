// Stub implementation for non-web platforms. Voice dictation on mobile
// is handled by the OS keyboard's built-in mic, so these methods are
// no-ops here.

bool get webSpeechSupported => false;

class WebSpeechSession {
  WebSpeechSession({
    required void Function(String, bool) onResult,
    required void Function() onEnd,
    required void Function(String) onError,
    String lang = 'en-US',
  });

  void start() {}
  void stop() {}
  void dispose() {}
}
