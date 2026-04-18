import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/services/auth_service.dart';
import '../../core/services/server_config.dart';
import '../../shared/theme/app_theme.dart';
import 'setup_api.dart';

/// First-run setup wizard.
///
/// Hits /api/setup/status on load; if needsSetup is false we bounce the
/// user to /. Otherwise we ask for a bootstrap token (loopback same-origin
/// first, URL query second, manual paste last) and then walk through
/// Welcome → (DB → Admin → Advanced) → Apply.
///
/// Two paths:
///  • Quick — embedded DB, auto JWT, admin password only.
///  • Custom — every knob exposed.
class SetupWizardPage extends StatefulWidget {
  const SetupWizardPage({super.key});
  @override
  State<SetupWizardPage> createState() => _SetupWizardPageState();
}

enum _Step { loading, needToken, welcome, db, admin, advanced, apply, done, noNeed, error }

enum _Path { quick, custom }

class _SetupWizardPageState extends State<SetupWizardPage> {
  _Step _step = _Step.loading;
  _Path _path = _Path.quick;
  SetupApi? _api;
  String _errorMsg = '';

  // Quick + Custom shared
  final _adminUserCtrl = TextEditingController(text: 'admin');
  final _adminPassCtrl = TextEditingController();
  final _adminConfirmCtrl = TextEditingController();
  bool _obscurePass = true;

  // Custom: DB choice
  String _dbMode = 'embedded'; // 'embedded' | 'external'
  final _dbHostCtrl = TextEditingController();
  final _dbPortCtrl = TextEditingController(text: '5432');
  final _dbUserCtrl = TextEditingController();
  final _dbPassCtrl = TextEditingController();
  final _dbNameCtrl = TextEditingController();
  String _dbTestResult = ''; // '' | 'ok' | 'err:<msg>'
  bool _dbTesting = false;

  // Custom: JWT
  String _jwtMode = 'auto'; // 'auto' | 'custom'
  final _jwtCustomCtrl = TextEditingController();

  // Apply step
  final List<String> _applyLog = [];
  bool _applying = false;
  String? _applyError;

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  @override
  void dispose() {
    _adminUserCtrl.dispose();
    _adminPassCtrl.dispose();
    _adminConfirmCtrl.dispose();
    _dbHostCtrl.dispose();
    _dbPortCtrl.dispose();
    _dbUserCtrl.dispose();
    _dbPassCtrl.dispose();
    _dbNameCtrl.dispose();
    _jwtCustomCtrl.dispose();
    super.dispose();
  }

  String get _baseUrl => context.read<ServerConfig>().effectiveUrl;

  Future<void> _bootstrap() async {
    final extras = context.read<ServerConfig>().cfAccessHeaders;
    try {
      final status = await SetupApi.status(_baseUrl, extraHeaders: extras);
      if (!status.needsSetup) {
        if (!mounted) return;
        setState(() => _step = _Step.noNeed);
        // After a tiny beat, navigate home — the server is already set up.
        Future.delayed(const Duration(milliseconds: 400), () {
          if (mounted) context.go('/');
        });
        return;
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _step = _Step.error;
        _errorMsg = 'Cannot reach $_baseUrl — $e';
      });
      return;
    }

    // Token discovery: query param first (user followed the stderr URL),
    // then loopback endpoint (same-origin local install), then manual.
    final uri = Uri.base;
    final qpToken = uri.queryParameters['token'];
    String? tok = (qpToken != null && qpToken.isNotEmpty) ? qpToken : null;
    tok ??= await SetupApi.loopbackToken(_baseUrl, extraHeaders: extras);

    if (!mounted) return;
    if (tok == null) {
      setState(() => _step = _Step.needToken);
      return;
    }
    setState(() {
      _api = SetupApi(baseUrl: _baseUrl, bootstrapToken: tok!, extraHeaders: extras);
      _step = _Step.welcome;
    });
  }

  void _acceptManualToken(String pasted) {
    final t = pasted.trim();
    if (t.isEmpty) return;
    final extras = context.read<ServerConfig>().cfAccessHeaders;
    setState(() {
      _api = SetupApi(baseUrl: _baseUrl, bootstrapToken: t, extraHeaders: extras);
      _step = _Step.welcome;
    });
  }

  String? _validateAdmin() {
    if (_adminPassCtrl.text.length < 8) {
      return 'Admin password must be at least 8 characters';
    }
    if (_adminPassCtrl.text != _adminConfirmCtrl.text) {
      return 'Admin passwords do not match';
    }
    return null;
  }

  Future<void> _runApply() async {
    if (_api == null) return;
    setState(() {
      _applying = true;
      _applyError = null;
      _applyLog.clear();
    });

    Future<void> step(String label, Future<void> Function() fn) async {
      setState(() => _applyLog.add('⏳ $label'));
      try {
        await fn();
        setState(() => _applyLog[_applyLog.length - 1] = '✓ $label');
      } catch (e) {
        setState(() => _applyLog[_applyLog.length - 1] = '✗ $label — $e');
        rethrow;
      }
    }

    try {
      // 1. DB commit
      if (_dbMode == 'embedded') {
        await step('Configuring embedded PostgreSQL',
            () => _api!.commitDBEmbedded());
      } else {
        await step('Saving external database settings', () async {
          await _api!.commitDBExternal(
            host: _dbHostCtrl.text.trim(),
            port: int.tryParse(_dbPortCtrl.text.trim()) ?? 5432,
            user: _dbUserCtrl.text.trim(),
            password: _dbPassCtrl.text,
            name: _dbNameCtrl.text.trim(),
          );
        });
      }

      // 2. JWT
      await step('Generating JWT secret', () async {
        final custom = _path == _Path.custom && _jwtMode == 'custom'
            ? _jwtCustomCtrl.text.trim()
            : null;
        await _api!.setJWT(customValue: custom);
      });

      // 3. Admin
      await step('Creating admin account', () async {
        await _api!.setAdmin(
          username: _adminUserCtrl.text.trim(),
          password: _adminPassCtrl.text,
        );
      });

      // 4. Finalize
      await step('Writing config & starting services', () async {
        await _api!.finalize();
      });

      // 5. Wait for the server to transition into normal mode. We poll
      // /api/auth/status — it's 503 during setup and 200 once normal.
      await step('Waiting for server to restart', () async {
        final extras = context.read<ServerConfig>().cfAccessHeaders;
        for (int i = 0; i < 60; i++) {
          try {
            final s = await SetupApi.status(_baseUrl, extraHeaders: extras);
            if (!s.needsSetup) return;
          } catch (_) {
            // Expected while the server is mid-transition
          }
          await Future.delayed(const Duration(milliseconds: 500));
        }
        throw 'server did not come back in 30s';
      });

      if (!mounted) return;
      setState(() {
        _applying = false;
        _step = _Step.done;
      });

      // Finally, clear the auth cache so the router redirect picks up the
      // new authRequired=true and the user lands on /login.
      await context.read<AuthService>().probe(
        _baseUrl,
        extraHeaders: context.read<ServerConfig>().cfAccessHeaders,
      );
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _applying = false;
        _applyError = e.toString();
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.bg,
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 560),
              child: _body(),
            ),
          ),
        ),
      ),
    );
  }

  Widget _body() {
    switch (_step) {
      case _Step.loading:
        return const _LoadingCard(msg: 'Contacting server…');
      case _Step.error:
        return _ErrorCard(msg: _errorMsg, onRetry: _bootstrap);
      case _Step.noNeed:
        return const _LoadingCard(msg: 'Already configured — redirecting…');
      case _Step.needToken:
        return _TokenPrompt(onSubmit: _acceptManualToken);
      case _Step.welcome:
        return _WelcomeCard(
          onQuick: () => setState(() { _path = _Path.quick; _step = _Step.admin; }),
          onCustom: () => setState(() { _path = _Path.custom; _step = _Step.db; }),
        );
      case _Step.db:
        return _DbCard(
          mode: _dbMode,
          onModeChange: (m) => setState(() { _dbMode = m; _dbTestResult = ''; }),
          hostCtrl: _dbHostCtrl,
          portCtrl: _dbPortCtrl,
          userCtrl: _dbUserCtrl,
          passCtrl: _dbPassCtrl,
          nameCtrl: _dbNameCtrl,
          testing: _dbTesting,
          testResult: _dbTestResult,
          onTest: _testDB,
          onBack: () => setState(() => _step = _Step.welcome),
          onNext: _dbNextEnabled() ? () => setState(() => _step = _Step.admin) : null,
        );
      case _Step.admin:
        return _AdminCard(
          userCtrl: _adminUserCtrl,
          passCtrl: _adminPassCtrl,
          confirmCtrl: _adminConfirmCtrl,
          obscure: _obscurePass,
          onToggleObscure: () => setState(() => _obscurePass = !_obscurePass),
          onBack: () => setState(() =>
              _step = _path == _Path.quick ? _Step.welcome : _Step.db),
          onNext: () {
            final err = _validateAdmin();
            if (err != null) {
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text(err), backgroundColor: AppColors.error),
              );
              return;
            }
            setState(() =>
                _step = _path == _Path.quick ? _Step.apply : _Step.advanced);
          },
        );
      case _Step.advanced:
        return _AdvancedCard(
          jwtMode: _jwtMode,
          onJwtModeChange: (m) => setState(() => _jwtMode = m),
          jwtCustomCtrl: _jwtCustomCtrl,
          onBack: () => setState(() => _step = _Step.admin),
          onNext: () => setState(() => _step = _Step.apply),
        );
      case _Step.apply:
        return _ApplyCard(
          log: _applyLog,
          applying: _applying,
          error: _applyError,
          summary: _summary(),
          onStart: _applying ? null : _runApply,
          onRetry: _applyError == null ? null : _runApply,
        );
      case _Step.done:
        return _DoneCard(
          username: _adminUserCtrl.text.trim(),
          onContinue: () => context.go('/login'),
        );
    }
  }

  bool _dbNextEnabled() {
    if (_dbMode == 'embedded') return true;
    // External — require a green test before moving on, prevents the
    // "wizard finalized with a typo in host" failure mode.
    return _dbTestResult == 'ok';
  }

  Future<void> _testDB() async {
    if (_api == null) return;
    setState(() { _dbTesting = true; _dbTestResult = ''; });
    try {
      await _api!.testDB(
        host: _dbHostCtrl.text.trim(),
        port: int.tryParse(_dbPortCtrl.text.trim()) ?? 5432,
        user: _dbUserCtrl.text.trim(),
        password: _dbPassCtrl.text,
        name: _dbNameCtrl.text.trim(),
      );
      if (!mounted) return;
      setState(() { _dbTesting = false; _dbTestResult = 'ok'; });
    } catch (e) {
      if (!mounted) return;
      setState(() { _dbTesting = false; _dbTestResult = 'err:$e'; });
    }
  }

  List<_SummaryLine> _summary() {
    final lines = <_SummaryLine>[
      _SummaryLine(
        'Database',
        _dbMode == 'embedded'
            ? 'Embedded PostgreSQL'
            : 'External @ ${_dbHostCtrl.text.trim()}:${_dbPortCtrl.text.trim()}/${_dbNameCtrl.text.trim()}',
      ),
      _SummaryLine('Admin user', _adminUserCtrl.text.trim()),
      _SummaryLine('JWT secret',
          _path == _Path.custom && _jwtMode == 'custom' ? 'Custom' : 'Auto-generated'),
    ];
    return lines;
  }
}

// ─── Individual steps ─────────────────────────────────────────────────────

class _LoadingCard extends StatelessWidget {
  final String msg;
  const _LoadingCard({required this.msg});
  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const CircularProgressIndicator(color: AppColors.accent),
          const SizedBox(height: 16),
          Text(msg, style: const TextStyle(color: AppColors.textMuted)),
        ]),
      ),
    );
  }
}

class _ErrorCard extends StatelessWidget {
  final String msg;
  final VoidCallback onRetry;
  const _ErrorCard({required this.msg, required this.onRetry});
  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Icon(Icons.cloud_off, color: AppColors.error, size: 32),
          const SizedBox(height: 12),
          const Text('Could not reach the server',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
          const SizedBox(height: 6),
          Text(msg, style: const TextStyle(fontSize: 12, color: AppColors.textMuted)),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: onRetry,
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            icon: const Icon(Icons.refresh, size: 16),
            label: const Text('Retry'),
          ),
        ]),
      ),
    );
  }
}

class _TokenPrompt extends StatefulWidget {
  final void Function(String) onSubmit;
  const _TokenPrompt({required this.onSubmit});
  @override
  State<_TokenPrompt> createState() => _TokenPromptState();
}

class _TokenPromptState extends State<_TokenPrompt> {
  final _ctrl = TextEditingController();

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Icon(Icons.key, color: AppColors.accent, size: 28),
          const SizedBox(height: 12),
          const Text('Bootstrap token required',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
          const SizedBox(height: 6),
          const Text(
            'OpenDray printed a setup URL to its stderr when it started. '
            'Either re-open that URL (it contains the token), or paste the '
            'token here.',
            style: TextStyle(fontSize: 12, color: AppColors.textMuted, height: 1.4),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _ctrl,
            autofocus: true,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
            decoration: const InputDecoration(
              labelText: 'Bootstrap token',
              hintText: 'paste here',
              prefixIcon: Icon(Icons.key_outlined, size: 18),
            ),
            onSubmitted: widget.onSubmit,
          ),
          const SizedBox(height: 12),
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton.icon(
              onPressed: () => widget.onSubmit(_ctrl.text),
              style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
              icon: const Icon(Icons.arrow_forward, size: 16),
              label: const Text('Continue'),
            ),
          ),
        ]),
      ),
    );
  }
}

class _WelcomeCard extends StatelessWidget {
  final VoidCallback onQuick;
  final VoidCallback onCustom;
  const _WelcomeCard({required this.onQuick, required this.onCustom});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Container(
              width: 48, height: 48,
              decoration: BoxDecoration(
                  color: AppColors.accent, borderRadius: BorderRadius.circular(12)),
              child: const Icon(Icons.rocket_launch, color: Colors.white, size: 28),
            ),
            const SizedBox(width: 14),
            const Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('Welcome to OpenDray',
                    style: TextStyle(fontSize: 20, fontWeight: FontWeight.w600)),
                SizedBox(height: 4),
                Text('Let\'s get you running in about a minute.',
                    style: TextStyle(fontSize: 13, color: AppColors.textMuted)),
              ]),
            ),
          ]),
          const SizedBox(height: 24),
          _PathCard(
            title: 'Quick install',
            subtitle: 'Embedded database, auto-generated secrets. Recommended.',
            icon: Icons.flash_on,
            onTap: onQuick,
            primary: true,
          ),
          const SizedBox(height: 10),
          _PathCard(
            title: 'Custom install',
            subtitle: 'Bring your own PostgreSQL, set custom JWT, tune advanced knobs.',
            icon: Icons.tune,
            onTap: onCustom,
          ),
        ]),
      ),
    );
  }
}

class _PathCard extends StatelessWidget {
  final String title;
  final String subtitle;
  final IconData icon;
  final VoidCallback onTap;
  final bool primary;
  const _PathCard({
    required this.title,
    required this.subtitle,
    required this.icon,
    required this.onTap,
    this.primary = false,
  });

  @override
  Widget build(BuildContext context) {
    final color = primary ? AppColors.accent : AppColors.border;
    return Material(
      color: primary ? AppColors.accentSoft : AppColors.surfaceAlt,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            border: Border.all(color: color),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Row(children: [
            Icon(icon, size: 22, color: primary ? AppColors.accent : AppColors.textMuted),
            const SizedBox(width: 12),
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(title,
                  style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: primary ? AppColors.accent : null)),
              const SizedBox(height: 2),
              Text(subtitle,
                  style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
            ])),
            const SizedBox(width: 8),
            Icon(Icons.chevron_right,
                size: 20, color: primary ? AppColors.accent : AppColors.textMuted),
          ]),
        ),
      ),
    );
  }
}

class _DbCard extends StatelessWidget {
  final String mode;
  final ValueChanged<String> onModeChange;
  final TextEditingController hostCtrl, portCtrl, userCtrl, passCtrl, nameCtrl;
  final bool testing;
  final String testResult;
  final VoidCallback onTest;
  final VoidCallback onBack;
  final VoidCallback? onNext;

  const _DbCard({
    required this.mode,
    required this.onModeChange,
    required this.hostCtrl,
    required this.portCtrl,
    required this.userCtrl,
    required this.passCtrl,
    required this.nameCtrl,
    required this.testing,
    required this.testResult,
    required this.onTest,
    required this.onBack,
    required this.onNext,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const _StepHeader(label: 'Database', step: 1, total: 3),
          const SizedBox(height: 16),
          _ModeRadio(
            selected: mode,
            value: 'embedded',
            label: 'Embedded PostgreSQL',
            hint: 'Runs locally, ~50 MB one-time download on first boot.',
            onTap: () => onModeChange('embedded'),
          ),
          const SizedBox(height: 8),
          _ModeRadio(
            selected: mode,
            value: 'external',
            label: 'External PostgreSQL',
            hint: 'Connect to an existing database.',
            onTap: () => onModeChange('external'),
          ),
          if (mode == 'external') ...[
            const SizedBox(height: 16),
            TextField(controller: hostCtrl, decoration: const InputDecoration(labelText: 'Host', hintText: 'localhost')),
            const SizedBox(height: 8),
            Row(children: [
              Expanded(flex: 1, child: TextField(
                controller: portCtrl,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(labelText: 'Port'),
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
              )),
              const SizedBox(width: 8),
              Expanded(flex: 2, child: TextField(
                controller: nameCtrl,
                decoration: const InputDecoration(labelText: 'Database name'),
              )),
            ]),
            const SizedBox(height: 8),
            TextField(controller: userCtrl, decoration: const InputDecoration(labelText: 'User')),
            const SizedBox(height: 8),
            TextField(
              controller: passCtrl,
              obscureText: true,
              decoration: const InputDecoration(labelText: 'Password'),
            ),
            const SizedBox(height: 12),
            Row(children: [
              OutlinedButton.icon(
                onPressed: testing ? null : onTest,
                icon: testing
                    ? const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2))
                    : const Icon(Icons.wifi_tethering, size: 16),
                label: Text(testing ? 'Testing…' : 'Test connection'),
              ),
              const SizedBox(width: 10),
              Expanded(child: _TestResultLabel(result: testResult)),
            ]),
          ],
          const SizedBox(height: 20),
          _NavRow(onBack: onBack, onNext: onNext),
        ]),
      ),
    );
  }
}

class _ModeRadio extends StatelessWidget {
  final String selected, value, label, hint;
  final VoidCallback onTap;
  const _ModeRadio({
    required this.selected,
    required this.value,
    required this.label,
    required this.hint,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final on = selected == value;
    return InkWell(
      borderRadius: BorderRadius.circular(10),
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: on ? AppColors.accentSoft : AppColors.surfaceAlt,
          border: Border.all(color: on ? AppColors.accent : AppColors.border),
          borderRadius: BorderRadius.circular(10),
        ),
        child: Row(children: [
          Icon(on ? Icons.radio_button_checked : Icons.radio_button_unchecked,
              size: 18, color: on ? AppColors.accent : AppColors.textMuted),
          const SizedBox(width: 10),
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(label,
                style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: on ? AppColors.accent : null)),
            const SizedBox(height: 2),
            Text(hint, style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
          ])),
        ]),
      ),
    );
  }
}

class _TestResultLabel extends StatelessWidget {
  final String result;
  const _TestResultLabel({required this.result});

  @override
  Widget build(BuildContext context) {
    if (result.isEmpty) return const SizedBox.shrink();
    final ok = result == 'ok';
    return Text(
      ok ? '✓ Connected' : result.replaceFirst('err:', '× '),
      style: TextStyle(
          fontSize: 11,
          color: ok ? AppColors.success : AppColors.error),
      maxLines: 2,
      overflow: TextOverflow.ellipsis,
    );
  }
}

class _AdminCard extends StatelessWidget {
  final TextEditingController userCtrl, passCtrl, confirmCtrl;
  final bool obscure;
  final VoidCallback onToggleObscure, onBack, onNext;
  const _AdminCard({
    required this.userCtrl,
    required this.passCtrl,
    required this.confirmCtrl,
    required this.obscure,
    required this.onToggleObscure,
    required this.onBack,
    required this.onNext,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const _StepHeader(label: 'Admin account', step: 2, total: 3),
          const SizedBox(height: 16),
          const Text('Create your admin login. You can change these later from Settings.',
              style: TextStyle(fontSize: 12, color: AppColors.textMuted)),
          const SizedBox(height: 16),
          TextField(
            controller: userCtrl,
            decoration: const InputDecoration(
              labelText: 'Username',
              prefixIcon: Icon(Icons.person_outline, size: 18),
            ),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: passCtrl,
            obscureText: obscure,
            decoration: InputDecoration(
              labelText: 'Password',
              prefixIcon: const Icon(Icons.lock_outline, size: 18),
              helperText: 'At least 8 characters',
              suffixIcon: IconButton(
                icon: Icon(obscure ? Icons.visibility_off : Icons.visibility, size: 18),
                onPressed: onToggleObscure,
              ),
            ),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: confirmCtrl,
            obscureText: obscure,
            decoration: const InputDecoration(
              labelText: 'Confirm password',
              prefixIcon: Icon(Icons.lock_outline, size: 18),
            ),
          ),
          const SizedBox(height: 20),
          _NavRow(onBack: onBack, onNext: onNext),
        ]),
      ),
    );
  }
}

class _AdvancedCard extends StatelessWidget {
  final String jwtMode;
  final ValueChanged<String> onJwtModeChange;
  final TextEditingController jwtCustomCtrl;
  final VoidCallback onBack, onNext;
  const _AdvancedCard({
    required this.jwtMode,
    required this.onJwtModeChange,
    required this.jwtCustomCtrl,
    required this.onBack,
    required this.onNext,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const _StepHeader(label: 'Advanced', step: 3, total: 3),
          const SizedBox(height: 16),
          const Text('JWT secret', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          _ModeRadio(
            selected: jwtMode, value: 'auto',
            label: 'Auto-generate (recommended)',
            hint: 'A fresh 48-byte secret is created for you.',
            onTap: () => onJwtModeChange('auto'),
          ),
          const SizedBox(height: 8),
          _ModeRadio(
            selected: jwtMode, value: 'custom',
            label: 'Use my own secret',
            hint: 'Paste a 32+ character value below.',
            onTap: () => onJwtModeChange('custom'),
          ),
          if (jwtMode == 'custom') ...[
            const SizedBox(height: 10),
            TextField(
              controller: jwtCustomCtrl,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              maxLines: 2,
              decoration: const InputDecoration(
                labelText: 'Custom JWT secret',
                hintText: '32+ random characters',
              ),
            ),
          ],
          const SizedBox(height: 20),
          _NavRow(onBack: onBack, onNext: onNext),
        ]),
      ),
    );
  }
}

class _SummaryLine {
  final String label, value;
  _SummaryLine(this.label, this.value);
}

class _ApplyCard extends StatelessWidget {
  final List<String> log;
  final bool applying;
  final String? error;
  final List<_SummaryLine> summary;
  final VoidCallback? onStart;
  final VoidCallback? onRetry;
  const _ApplyCard({
    required this.log,
    required this.applying,
    required this.error,
    required this.summary,
    required this.onStart,
    required this.onRetry,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('Ready to apply',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppColors.surfaceAlt,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Column(children: [
              for (final line in summary)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  child: Row(children: [
                    SizedBox(width: 100, child: Text(line.label,
                        style: const TextStyle(fontSize: 11, color: AppColors.textMuted))),
                    Expanded(child: Text(line.value,
                        style: const TextStyle(fontSize: 12, fontFamily: 'monospace'))),
                  ]),
                ),
            ]),
          ),
          const SizedBox(height: 16),
          if (log.isNotEmpty) ...[
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: const Color(0xFF0B0D11),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                for (final entry in log)
                  Text(entry,
                      style: const TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 12,
                          color: AppColors.text,
                          height: 1.6)),
              ]),
            ),
            const SizedBox(height: 12),
          ],
          if (error != null) ...[
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: AppColors.errorSoft,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(children: [
                const Icon(Icons.error_outline, size: 16, color: AppColors.error),
                const SizedBox(width: 8),
                Expanded(child: Text(error!,
                    style: const TextStyle(fontSize: 12, color: AppColors.error))),
              ]),
            ),
            const SizedBox(height: 12),
          ],
          Row(children: [
            if (error != null)
              Expanded(child: FilledButton.icon(
                onPressed: onRetry,
                style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
                icon: const Icon(Icons.refresh, size: 16),
                label: const Text('Retry'),
              ))
            else if (log.isEmpty)
              Expanded(child: FilledButton.icon(
                onPressed: onStart,
                style: FilledButton.styleFrom(
                  backgroundColor: AppColors.accent,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                ),
                icon: applying
                    ? const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                    : const Icon(Icons.check, size: 16),
                label: Text(applying ? 'Applying…' : 'Apply setup'),
              )),
          ]),
        ]),
      ),
    );
  }
}

class _DoneCard extends StatelessWidget {
  final String username;
  final VoidCallback onContinue;
  const _DoneCard({required this.username, required this.onContinue});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.check_circle, color: AppColors.success, size: 56),
          const SizedBox(height: 12),
          const Text('You\'re all set',
              style: TextStyle(fontSize: 22, fontWeight: FontWeight.w600)),
          const SizedBox(height: 6),
          Text('Sign in as "$username" to continue.',
              style: const TextStyle(fontSize: 13, color: AppColors.textMuted)),
          const SizedBox(height: 20),
          FilledButton.icon(
            onPressed: onContinue,
            style: FilledButton.styleFrom(
              backgroundColor: AppColors.accent,
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
            ),
            icon: const Icon(Icons.login, size: 16),
            label: const Text('Continue to sign in'),
          ),
        ]),
      ),
    );
  }
}

// ─── Shared bits ──────────────────────────────────────────────────────────

class _StepHeader extends StatelessWidget {
  final String label;
  final int step, total;
  const _StepHeader({required this.label, required this.step, required this.total});
  @override
  Widget build(BuildContext context) {
    return Row(children: [
      Text(label, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
      const Spacer(),
      Text('$step / $total',
          style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
    ]);
  }
}

class _NavRow extends StatelessWidget {
  final VoidCallback onBack;
  final VoidCallback? onNext;
  const _NavRow({required this.onBack, required this.onNext});

  @override
  Widget build(BuildContext context) {
    return Row(children: [
      OutlinedButton(onPressed: onBack, child: const Text('Back')),
      const Spacer(),
      FilledButton.icon(
        onPressed: onNext,
        style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
        icon: const Icon(Icons.arrow_forward, size: 16),
        label: const Text('Continue'),
      ),
    ]);
  }
}
