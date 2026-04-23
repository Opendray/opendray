import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/services/auth_service.dart';
import '../../core/services/l10n.dart';
import '../../core/services/server_config.dart';
import '../../shared/theme/app_theme.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});
  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  // Seeded empty — we populate from the active profile in initState so
  // users who save a username per server never retype it. Falls back to
  // 'admin' when the active profile has no saved username (fresh
  // install flow that goes /connect → /login without editing the
  // profile first).
  final _userCtrl = TextEditingController();
  final _passCtrl = TextEditingController();
  final _passFocus = FocusNode();
  bool _submitting = false;
  String? _error;
  bool _obscure = true;

  @override
  void initState() {
    super.initState();
    // Defer one frame so context.read is legal and the ServerConfig
    // value listen-less read is safe.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final p = context.read<ServerConfig>().activeProfile;
      final defaultUser = (p?.username ?? '').isNotEmpty
          ? p!.username
          : 'admin';
      _userCtrl.text = defaultUser;
    });
  }

  @override
  void dispose() {
    _userCtrl.dispose();
    _passCtrl.dispose();
    _passFocus.dispose();
    super.dispose();
  }

  /// Clears the saved server URL so the router redirect sends us to
  /// /connect. Used for switching between multiple OpenDray instances —
  /// previously the URL was locked in read-only display and the only
  /// out was uninstalling the app.
  Future<void> _changeServer() async {
    // Confirm before discarding — users who tapped the label by mistake
    // shouldn't lose their config just for that.
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(context.tr('Change server?')),
        content: Text(context.tr(
            'You\'ll be asked for a new server URL. Your saved token for this server stays on device — switching back restores it.')),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(context.tr('Cancel'))),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
              child: Text(context.tr('Change'))),
        ],
      ),
    );
    if (ok != true || !mounted) return;

    // Drop the URL — ServerConfig.setUrl('') flips isConfigured to false
    // and fires notifyListeners. The router's refreshListenable re-runs
    // redirect, sees !isConfigured, sends us to /connect.
    await context.read<ServerConfig>().setUrl('');
  }

  Future<void> _submit() async {
    if (_submitting) return;
    final user = _userCtrl.text.trim();
    final pass = _passCtrl.text;
    if (user.isEmpty || pass.isEmpty) {
      setState(() => _error = context.tr('Enter username and password'));
      return;
    }
    setState(() { _submitting = true; _error = null; });
    final cfg = context.read<ServerConfig>();
    final auth = context.read<AuthService>();
    final err = await auth.login(
      serverUrl: cfg.effectiveUrl,
      username: user,
      password: pass,
    );
    if (!mounted) return;
    setState(() {
      _submitting = false;
      _error = err;
    });
    // On success, persist the username into the active profile so the
    // next login prompt is one field lighter. Password is NOT saved
    // here — "Remember password" is an explicit opt-in from the
    // profile editor, not a side-effect of a successful login.
    if (err == null) {
      final active = cfg.activeProfile;
      if (active != null && active.username != user) {
        unawaited(cfg.updateProfile(active.id, username: user));
      }
    }
    // On success the router's redirect picks up the AuthService change and
    // moves us to '/'. No manual navigation needed.
  }

  @override
  Widget build(BuildContext context) {
    final cfg = context.watch<ServerConfig>();
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 380),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Logo
                  Center(
                    child: Container(
                      width: 64,
                      height: 64,
                      decoration: BoxDecoration(
                        color: AppColors.accent,
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: const Icon(Icons.lock_outline,
                          color: Colors.white, size: 34),
                    ),
                  ),
                  const SizedBox(height: 20),
                  Text(context.tr('Sign in to OpenDray'),
                      textAlign: TextAlign.center,
                      style: const TextStyle(
                          fontSize: 20, fontWeight: FontWeight.w600)),
                  const SizedBox(height: 10),
                  // Server URL row — tapping "Change" clears the stored URL
                  // so the router kicks us back to /connect. This is the
                  // only way to switch between multiple OpenDray instances
                  // once a URL has been saved; see also the router redirect
                  // which otherwise force-bounces /connect → / when a URL
                  // is set.
                  _ServerUrlRow(
                    url: cfg.effectiveUrl,
                    onChange: _submitting ? null : _changeServer,
                  ),
                  const SizedBox(height: 20),

                  TextField(
                    controller: _userCtrl,
                    enabled: !_submitting,
                    textInputAction: TextInputAction.next,
                    onSubmitted: (_) => _passFocus.requestFocus(),
                    decoration: InputDecoration(
                      labelText: context.tr('Username'),
                      prefixIcon: const Icon(Icons.person_outline, size: 20),
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _passCtrl,
                    focusNode: _passFocus,
                    enabled: !_submitting,
                    obscureText: _obscure,
                    textInputAction: TextInputAction.done,
                    onSubmitted: (_) => _submit(),
                    decoration: InputDecoration(
                      labelText: context.tr('Password'),
                      prefixIcon: const Icon(Icons.key_outlined, size: 20),
                      suffixIcon: IconButton(
                        icon: Icon(
                          _obscure
                              ? Icons.visibility_off_outlined
                              : Icons.visibility_outlined,
                          size: 18,
                        ),
                        onPressed: () => setState(() => _obscure = !_obscure),
                      ),
                    ),
                  ),

                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Container(
                      padding: const EdgeInsets.all(10),
                      decoration: BoxDecoration(
                        color: AppColors.errorSoft,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Row(children: [
                        const Icon(Icons.error_outline,
                            size: 16, color: AppColors.error),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(_error!,
                              style: const TextStyle(
                                  color: AppColors.error, fontSize: 12)),
                        ),
                      ]),
                    ),
                  ],

                  const SizedBox(height: 18),
                  FilledButton(
                    onPressed: _submitting ? null : _submit,
                    style: FilledButton.styleFrom(
                      backgroundColor: AppColors.accent,
                      padding: const EdgeInsets.symmetric(vertical: 14),
                    ),
                    child: _submitting
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : Text(context.tr('Sign in')),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Compact display of the current server URL with a "Change" action.
/// Shown on the login page so users can switch between multiple
/// OpenDray deployments without reinstalling the app.
class _ServerUrlRow extends StatelessWidget {
  final String url;
  final VoidCallback? onChange;
  const _ServerUrlRow({required this.url, required this.onChange});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: AppColors.surfaceAlt,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Row(
        children: [
          const Icon(Icons.dns_outlined, size: 16, color: AppColors.textMuted),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              url.isEmpty ? context.tr('No server configured') : url,
              style: const TextStyle(
                fontSize: 12,
                fontFamily: 'monospace',
                color: AppColors.text,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          TextButton.icon(
            onPressed: onChange,
            icon: const Icon(Icons.swap_horiz, size: 16),
            label: Text(context.tr('Change')),
            style: TextButton.styleFrom(
              foregroundColor: AppColors.accent,
              padding: const EdgeInsets.symmetric(horizontal: 8),
              minimumSize: const Size(0, 32),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
          ),
        ],
      ),
    );
  }
}
