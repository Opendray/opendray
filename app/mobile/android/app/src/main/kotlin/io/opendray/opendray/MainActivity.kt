package io.opendray.opendray

import io.flutter.embedding.android.FlutterFragmentActivity

// FlutterFragmentActivity rather than FlutterActivity: local_auth
// shows androidx.biometric's BiometricPrompt, which is a Fragment and
// needs a FragmentActivity host. With a plain FlutterActivity the
// app-lock prompt throws at runtime instead of appearing.
class MainActivity : FlutterFragmentActivity()
