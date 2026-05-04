// V1 login page — two-panel split design from the prototype.
//
// Wide viewport (>= 960 px) renders a 50/50 split:
//   left  — accent-tinted brand panel with OD mark, wordmark, tagline.
//   right — sign-in form (server URL + username + password + button).
// Narrow viewport collapses to form-only fullscreen with a small brand
// strip on top so phone users still see who they're signing in to.
//
// Behaviour preserved from the previous LoginPage:
//   • Username pre-filled from the active server profile.
//   • Server URL shown with Change action (clears stored URL → router
//     bounces to /connect).
//   • Submit hits AuthService.login; on success the router redirect
//     lands the user at /.
//
// Visual: OpendrayTokens for colors / spacing / radii. Inter typography
// inherits from the global theme.

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
  final _userCtrl = TextEditingController();
  final _passCtrl = TextEditingController();
  final _passFocus = FocusNode();
  bool _submitting = false;
  String? _error;
  bool _obscure = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final p = context.read<ServerConfig>().activeProfile;
      final defaultUser = (p?.username ?? '').isNotEmpty ? p!.username : 'admin';
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

  Future<void> _changeServer() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(context.tr('Change server?')),
        content: Text(context.tr(
            "You'll be asked for a new server URL. Your saved token for this server stays on device — switching back restores it.")),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(context.tr('Cancel'))),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: Text(context.tr('Change'))),
        ],
      ),
    );
    if (ok != true || !mounted) return;
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
    setState(() {
      _submitting = true;
      _error = null;
    });
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
    if (err == null) {
      final active = cfg.activeProfile;
      if (active != null && active.username != user) {
        unawaited(cfg.updateProfile(active.id, username: user));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Scaffold(
      backgroundColor: t.bg,
      body: LayoutBuilder(builder: (ctx, c) {
        final wide = c.maxWidth >= 960;
        if (wide) {
          return Row(
            children: [
              Expanded(child: _BrandPanel()),
              Container(width: 1, color: t.border),
              Expanded(child: _FormPanel(state: this)),
            ],
          );
        }
        // Narrow — single column. Small brand strip above the form.
        return SafeArea(
          child: SingleChildScrollView(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 480),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _BrandStripCompact(),
                  _FormPanel(state: this),
                ],
              ),
            ),
          ),
        );
      }),
    );
  }
}

// -----------------------------------------------------------------------------
// Left panel — brand
// -----------------------------------------------------------------------------

class _BrandPanel extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    return Container(
      color: t.surface,
      padding: EdgeInsets.all(t.sp10),
      child: Stack(
        children: [
          // Subtle radial accent in the background to give the panel
          // a sense of depth without adding an asset dependency.
          Positioned.fill(
            child: DecoratedBox(
              decoration: BoxDecoration(
                gradient: RadialGradient(
                  center: const Alignment(-0.4, -0.6),
                  radius: 1.2,
                  colors: [
                    t.accent.withValues(alpha: 0.18),
                    Colors.transparent,
                  ],
                ),
              ),
            ),
          ),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    width: 36, height: 36,
                    decoration: BoxDecoration(
                      color: t.accent,
                      borderRadius: BorderRadius.circular(t.rSm),
                    ),
                    alignment: Alignment.center,
                    child: const Text('OD',
                        style: TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w800,
                            fontSize: 13,
                            letterSpacing: 0.5)),
                  ),
                  SizedBox(width: t.sp3),
                  Text('Opendray',
                      style: theme.textTheme.titleLarge
                          ?.copyWith(fontWeight: FontWeight.w700, fontSize: 18)),
                ],
              ),
              const Spacer(),
              Text(
                  'Run AI coding agents on your hardware.\nReach them from any device.',
                  style: theme.textTheme.displaySmall
                      ?.copyWith(fontSize: 26, height: 1.3)),
              SizedBox(height: t.sp4),
              Text(
                  'Self-hosted cockpit for Claude Code, Codex, Gemini and OpenCode. Pair sessions with Telegram for control on the go. Plugin platform extends every workspace.',
                  style: theme.textTheme.bodyLarge?.copyWith(
                      color: t.textMuted, height: 1.55, fontSize: 14)),
              SizedBox(height: t.sp6),
              Container(
                padding: EdgeInsets.all(t.sp4),
                decoration: BoxDecoration(
                  color: t.surface2,
                  borderRadius: BorderRadius.circular(t.rLg),
                  border: Border.all(color: t.border),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                        '"Our staff engineers don\'t sit at their desks anymore — they ship from their phones. Opendray became the way we run Claude Code."',
                        style: theme.textTheme.bodyMedium?.copyWith(
                            color: t.text,
                            fontStyle: FontStyle.italic,
                            height: 1.5)),
                    SizedBox(height: t.sp3),
                    Text('— Platform team, early access',
                        style: theme.textTheme.bodySmall
                            ?.copyWith(color: t.textSubtle, fontSize: 11)),
                  ],
                ),
              ),
              const Spacer(),
              Row(
                children: [
                  Container(
                    width: 6, height: 6,
                    decoration: BoxDecoration(
                        color: t.success, shape: BoxShape.circle),
                  ),
                  SizedBox(width: t.sp2),
                  Text('All systems operational',
                      style: theme.textTheme.bodySmall
                          ?.copyWith(color: t.textMuted, fontSize: 11)),
                  const Spacer(),
                  Text('opendray.com',
                      style: theme.textTheme.bodySmall?.copyWith(
                          color: t.textSubtle, fontSize: 11)),
                ],
              ),
            ],
          ),
        ],
      ),
    );
  }
}

// Compact brand strip for narrow viewports — header above the form.
class _BrandStripCompact extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Padding(
      padding: EdgeInsets.fromLTRB(t.sp5, t.sp6, t.sp5, t.sp4),
      child: Row(
        children: [
          Container(
            width: 32, height: 32,
            decoration: BoxDecoration(
              color: t.accent,
              borderRadius: BorderRadius.circular(t.rSm),
            ),
            alignment: Alignment.center,
            child: const Text('OD',
                style: TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w800,
                    fontSize: 12)),
          ),
          SizedBox(width: t.sp3),
          Text('Opendray',
              style: Theme.of(context).textTheme.titleLarge
                  ?.copyWith(fontWeight: FontWeight.w700, fontSize: 16)),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Right panel — sign-in form
// -----------------------------------------------------------------------------

class _FormPanel extends StatelessWidget {
  final _LoginPageState state;
  const _FormPanel({required this.state});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    final cfg = context.watch<ServerConfig>();
    return Container(
      color: t.bg,
      padding: EdgeInsets.all(t.sp8),
      alignment: Alignment.center,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 380),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('Sign in to Opendray',
                style: theme.textTheme.displaySmall?.copyWith(fontSize: 24)),
            SizedBox(height: t.sp2),
            Text(
                'Use the admin account that was set up when this server was first installed.',
                style: theme.textTheme.bodyMedium?.copyWith(color: t.textMuted)),
            SizedBox(height: t.sp5),
            _ServerUrlRow(
              url: cfg.effectiveUrl,
              onChange: state._submitting ? null : state._changeServer,
            ),
            SizedBox(height: t.sp4),
            _Label(text: 'Username'),
            SizedBox(height: t.sp1),
            TextField(
              controller: state._userCtrl,
              enabled: !state._submitting,
              textInputAction: TextInputAction.next,
              autofocus: false,
              onSubmitted: (_) => state._passFocus.requestFocus(),
              decoration: const InputDecoration(
                hintText: 'admin',
                prefixIcon: Icon(Icons.person_outline, size: 16),
              ),
            ),
            SizedBox(height: t.sp3),
            _Label(text: 'Password'),
            SizedBox(height: t.sp1),
            TextField(
              controller: state._passCtrl,
              focusNode: state._passFocus,
              enabled: !state._submitting,
              obscureText: state._obscure,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => state._submit(),
              decoration: InputDecoration(
                hintText: '••••••••',
                prefixIcon: const Icon(Icons.lock_outline, size: 16),
                suffixIcon: IconButton(
                  icon: Icon(
                      state._obscure
                          ? Icons.visibility_off_outlined
                          : Icons.visibility_outlined,
                      size: 16),
                  onPressed: () => state.setState(() => state._obscure = !state._obscure),
                ),
              ),
            ),
            if (state._error != null) ...[
              SizedBox(height: t.sp3),
              Container(
                padding: EdgeInsets.all(t.sp3),
                decoration: BoxDecoration(
                  color: t.dangerSoft,
                  borderRadius: BorderRadius.circular(t.rMd),
                  border: Border.all(color: t.danger.withValues(alpha: 0.4)),
                ),
                child: Row(children: [
                  Icon(Icons.error_outline, size: 14, color: t.danger),
                  SizedBox(width: t.sp2),
                  Expanded(
                      child: Text(state._error!,
                          style: TextStyle(color: t.danger, fontSize: 12))),
                ]),
              ),
            ],
            SizedBox(height: t.sp4),
            SizedBox(
              height: 44,
              child: FilledButton(
                onPressed: state._submitting ? null : state._submit,
                style: FilledButton.styleFrom(
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(t.rMd)),
                ),
                child: state._submitting
                    ? const SizedBox(
                        width: 16, height: 16,
                        child: CircularProgressIndicator(
                            strokeWidth: 2, color: Colors.white))
                    : const Text('Sign in',
                        style: TextStyle(
                            fontSize: 14, fontWeight: FontWeight.w600)),
              ),
            ),
            SizedBox(height: t.sp4),
            Center(
              child: Text(
                'Don\'t have a server yet? Run install.sh on your VPS or LXC.',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: t.textSubtle, fontSize: 11),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _Label extends StatelessWidget {
  final String text;
  const _Label({required this.text});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Text(text,
        style: TextStyle(
            fontSize: 11,
            color: t.textMuted,
            fontWeight: FontWeight.w600,
            letterSpacing: 0.4));
  }
}

class _ServerUrlRow extends StatelessWidget {
  final String url;
  final VoidCallback? onChange;
  const _ServerUrlRow({required this.url, required this.onChange});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      padding: EdgeInsets.symmetric(horizontal: t.sp3, vertical: t.sp2),
      decoration: BoxDecoration(
        color: t.surface,
        borderRadius: BorderRadius.circular(t.rMd),
        border: Border.all(color: t.border),
      ),
      child: Row(
        children: [
          Icon(Icons.dns_outlined, size: 14, color: t.textMuted),
          SizedBox(width: t.sp2),
          Expanded(
            child: Text(
              url.isEmpty ? context.tr('No server configured') : url,
              style: TextStyle(
                  fontSize: 11, fontFamily: 'monospace', color: t.text),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          TextButton.icon(
            onPressed: onChange,
            icon: const Icon(Icons.swap_horiz, size: 14),
            label: Text(context.tr('Change'),
                style: const TextStyle(fontSize: 11)),
            style: TextButton.styleFrom(
              padding: EdgeInsets.symmetric(horizontal: t.sp2),
              minimumSize: const Size(0, 28),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
          ),
        ],
      ),
    );
  }
}
