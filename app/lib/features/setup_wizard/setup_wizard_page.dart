// First-run wizard. Walks a non-technical user from a fresh login to a
// working Claude session in 4 steps:
//
//   1. Detect — show what we found on the host (CLIs, existing creds)
//   2. Install Claude CLI if missing (server-side via /api/host/install-claude-cli)
//   3. Get signed in (import existing ~/.claude OR run in-app OAuth)
//   4. Done — drop them at the Hub
//
// Skippable from any step ("I'll do this later"). The wizard sets a
// shared-prefs flag on completion so it doesn't re-trigger on next
// login. App-shell redirect logic checks that flag + the
// claude_accounts list to decide whether to auto-route here.

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/api/api_client.dart';
import '../../shared/theme/app_theme.dart';

const String setupCompletedPrefKey = 'opendray.setup.completed.v1';

class SetupWizardPage extends StatefulWidget {
  const SetupWizardPage({super.key});
  @override
  State<SetupWizardPage> createState() => _SetupWizardPageState();
}

enum _Step { detect, install, signin, done }

class _SetupWizardPageState extends State<SetupWizardPage> {
  ApiClient get _api => context.read<ApiClient>();
  Map<String, dynamic>? _facts;
  String? _factsError;
  bool _installing = false;
  String? _installError;
  String? _installOutput;
  _Step _step = _Step.detect;

  @override
  void initState() {
    super.initState();
    _refreshFacts();
  }

  Future<void> _refreshFacts() async {
    try {
      final f = await _api.hostFacts();
      if (!mounted) return;
      setState(() {
        _facts = f;
        _factsError = null;
        _step = _decideStep(f);
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _factsError = e.toString());
    }
  }

  _Step _decideStep(Map<String, dynamic> facts) {
    final clis = (facts['clis'] as Map?) ?? {};
    final claude = clis['claude'] as Map?;
    final hasClaude = (claude?['found'] as bool?) ?? false;
    final creds = (facts['credentials'] as List?) ?? [];
    final hasUnimportedCreds =
        creds.any((c) => (c['valid'] as bool? ?? false) && !(c['alreadyImported'] as bool? ?? false));
    if (!hasClaude) return _Step.install;
    if (hasUnimportedCreds) return _Step.signin;
    // No CLI missing, no creds to import — user can sign in fresh OR skip.
    return _Step.signin;
  }

  Future<void> _installClaudeCLI() async {
    setState(() {
      _installing = true;
      _installError = null;
      _installOutput = null;
    });
    try {
      final res = await _api.hostInstallClaudeCLI();
      if (!mounted) return;
      final ok = res['installed'] as bool? ?? false;
      setState(() {
        _installing = false;
        _installOutput = res['output'] as String?;
        if (!ok) _installError = res['error']?.toString() ?? 'install failed';
      });
      if (ok) await _refreshFacts();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _installing = false;
        _installError = e.toString();
      });
    }
  }

  Future<void> _importCred(Map<String, dynamic> cred) async {
    try {
      await _api.hostImportClaudeCreds(path: cred['path'] as String);
      await _refreshFacts();
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Claude account imported')));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Import failed: $e')));
      }
    }
  }

  Future<void> _completeAndExit() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(setupCompletedPrefKey, true);
    if (mounted) context.go('/');
  }

  Future<void> _skip() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(setupCompletedPrefKey, true);
    if (mounted) context.go('/');
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Scaffold(
      backgroundColor: t.bg,
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 720),
            child: SingleChildScrollView(
              padding: EdgeInsets.all(t.sp6),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _Header(step: _step, onSkip: _skip),
                  SizedBox(height: t.sp6),
                  if (_factsError != null) _ErrorBanner(message: _factsError!),
                  if (_facts == null && _factsError == null)
                    Center(child: CircularProgressIndicator(color: t.accent))
                  else if (_facts != null) ...[
                    _DetectionCard(facts: _facts!),
                    SizedBox(height: t.sp4),
                    if (_step == _Step.install) _InstallCard(
                      installing: _installing,
                      output: _installOutput,
                      error: _installError,
                      onInstall: _installClaudeCLI,
                      onSkipToSignin: () => setState(() => _step = _Step.signin),
                    )
                    else if (_step == _Step.signin) _SigninCard(
                      facts: _facts!,
                      onImport: _importCred,
                      onDone: _completeAndExit,
                    )
                    else if (_step == _Step.done) _DoneCard(onContinue: _completeAndExit),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Header (steps + skip)
// -----------------------------------------------------------------------------

class _Header extends StatelessWidget {
  final _Step step;
  final VoidCallback onSkip;
  const _Header({required this.step, required this.onSkip});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Container(
              width: 40, height: 40,
              decoration: BoxDecoration(
                color: t.accent,
                borderRadius: BorderRadius.circular(t.rMd),
              ),
              alignment: Alignment.center,
              child: const Text('OD',
                  style: TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.w800,
                      fontSize: 14,
                      letterSpacing: 0.5)),
            ),
            SizedBox(width: t.sp3),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Welcome to OpenDray',
                      style: Theme.of(context).textTheme.displaySmall),
                  Text("Let's get you set up — should take 60 seconds.",
                      style: Theme.of(context)
                          .textTheme
                          .bodyLarge
                          ?.copyWith(color: t.textMuted)),
                ],
              ),
            ),
            TextButton(
                onPressed: onSkip, child: const Text("I'll do this later")),
          ],
        ),
        SizedBox(height: t.sp5),
        _StepStrip(active: step),
      ],
    );
  }
}

class _StepStrip extends StatelessWidget {
  final _Step active;
  const _StepStrip({required this.active});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final steps = const [
      ('Detect', _Step.detect),
      ('Install Claude CLI', _Step.install),
      ('Sign in', _Step.signin),
      ('Done', _Step.done),
    ];
    return Row(
      children: [
        for (final s in steps) ...[
          _StepDot(label: s.$1, state: _stateFor(s.$2)),
          if (s != steps.last)
            Expanded(
                child: Container(
                    height: 1,
                    margin: EdgeInsets.symmetric(horizontal: t.sp2),
                    color: t.border)),
        ],
      ],
    );
  }

  _StepState _stateFor(_Step s) {
    final order = [_Step.detect, _Step.install, _Step.signin, _Step.done];
    final ai = order.indexOf(active);
    final si = order.indexOf(s);
    if (si < ai) return _StepState.complete;
    if (si == ai) return _StepState.active;
    return _StepState.pending;
  }
}

enum _StepState { pending, active, complete }

class _StepDot extends StatelessWidget {
  final String label;
  final _StepState state;
  const _StepDot({required this.label, required this.state});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final isActive = state == _StepState.active;
    final isDone = state == _StepState.complete;
    final fill =
        isDone ? t.success : (isActive ? t.accent : Colors.transparent);
    final border = isDone
        ? t.success
        : (isActive ? t.accent : t.border);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 22, height: 22,
          decoration: BoxDecoration(
            color: fill,
            shape: BoxShape.circle,
            border: Border.all(color: border, width: 1.5),
          ),
          alignment: Alignment.center,
          child: isDone
              ? const Icon(Icons.check, color: Colors.white, size: 12)
              : isActive
                  ? Container(
                      width: 6,
                      height: 6,
                      decoration:
                          const BoxDecoration(color: Colors.white, shape: BoxShape.circle))
                  : null,
        ),
        SizedBox(width: t.sp2),
        Text(label,
            style: TextStyle(
                fontSize: 12,
                color: isActive
                    ? t.text
                    : (isDone ? t.success : t.textSubtle),
                fontWeight: isActive ? FontWeight.w600 : FontWeight.w500)),
      ],
    );
  }
}

// -----------------------------------------------------------------------------
// Detection card (step 1, always visible)
// -----------------------------------------------------------------------------

class _DetectionCard extends StatelessWidget {
  final Map<String, dynamic> facts;
  const _DetectionCard({required this.facts});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    final clis = (facts['clis'] as Map?) ?? {};
    final claude = clis['claude'] as Map? ?? {};
    final npm = clis['npm'] as Map? ?? {};
    final git = clis['git'] as Map? ?? {};
    final creds = (facts['credentials'] as List?) ?? const [];
    final unimported = creds.where((c) =>
        (c['valid'] as bool? ?? false) &&
        !(c['alreadyImported'] as bool? ?? false));

    return _Card(
      title: 'What we found on this host',
      subtitle:
          'OpenDray probed your host to figure out what\'s already installed.',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _DetectRow(
              label: 'Claude Code CLI',
              value: (claude['found'] as bool? ?? false)
                  ? '${claude['version'] ?? 'installed'}'
                  : 'not installed',
              ok: (claude['found'] as bool? ?? false)),
          _DetectRow(
              label: 'npm (Node.js)',
              value: (npm['found'] as bool? ?? false)
                  ? '${npm['version'] ?? 'installed'}'
                  : 'not installed',
              ok: (npm['found'] as bool? ?? false)),
          _DetectRow(
              label: 'git',
              value: (git['found'] as bool? ?? false)
                  ? '${git['version'] ?? 'installed'}'
                  : 'not installed',
              ok: (git['found'] as bool? ?? false)),
          _DetectRow(
              label: 'Default project folder',
              value:
                  '${facts['defaultProjectsDir'] ?? '~'}',
              ok: true),
          if (unimported.isNotEmpty)
            Padding(
              padding: EdgeInsets.only(top: t.sp3),
              child: Container(
                padding: EdgeInsets.all(t.sp3),
                decoration: BoxDecoration(
                  color: t.accentSoft,
                  borderRadius: BorderRadius.circular(t.rMd),
                  border: Border.all(color: t.accentBorder),
                ),
                child: Row(
                  children: [
                    Icon(Icons.info_outline, color: t.accentText, size: 16),
                    SizedBox(width: t.sp2),
                    Expanded(
                      child: Text(
                          'Found ${unimported.length} existing Claude credential${unimported.length == 1 ? '' : 's'} on disk that we can import.',
                          style: theme.textTheme.bodyMedium
                              ?.copyWith(color: t.text)),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _DetectRow extends StatelessWidget {
  final String label;
  final String value;
  final bool ok;
  const _DetectRow(
      {required this.label, required this.value, required this.ok});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Padding(
      padding: EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Icon(
            ok ? Icons.check_circle : Icons.radio_button_unchecked,
            size: 16,
            color: ok ? t.success : t.textSubtle,
          ),
          SizedBox(width: t.sp3),
          Expanded(
              child: Text(label,
                  style: Theme.of(context).textTheme.bodyMedium)),
          Text(value,
              style: TextStyle(
                  fontSize: 12,
                  color: ok ? t.text : t.textMuted,
                  fontFamily: 'monospace')),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Install card (step 2)
// -----------------------------------------------------------------------------

class _InstallCard extends StatelessWidget {
  final bool installing;
  final String? output;
  final String? error;
  final VoidCallback onInstall;
  final VoidCallback onSkipToSignin;
  const _InstallCard({
    required this.installing,
    required this.output,
    required this.error,
    required this.onInstall,
    required this.onSkipToSignin,
  });

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return _Card(
      title: 'Install Claude Code CLI',
      subtitle:
          'OpenDray needs the official Claude CLI to launch sessions. We can install it for you.',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              ElevatedButton.icon(
                onPressed: installing ? null : onInstall,
                icon: installing
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(
                            strokeWidth: 2, color: Colors.white))
                    : const Icon(Icons.download, size: 16),
                label: Text(installing ? 'Installing…' : 'Install for me'),
              ),
              SizedBox(width: t.sp3),
              TextButton(
                  onPressed: installing ? null : onSkipToSignin,
                  child: const Text('Skip — I\'ll install it myself')),
            ],
          ),
          if (output != null && output!.isNotEmpty) ...[
            SizedBox(height: t.sp3),
            Container(
              constraints: const BoxConstraints(maxHeight: 180),
              padding: EdgeInsets.all(t.sp3),
              decoration: BoxDecoration(
                color: t.bgRaised,
                borderRadius: BorderRadius.circular(t.rMd),
                border: Border.all(color: t.border),
              ),
              child: SingleChildScrollView(
                child: Text(output!,
                    style: mono(size: 11, color: t.textMuted)),
              ),
            ),
          ],
          if (error != null) ...[
            SizedBox(height: t.sp3),
            Container(
              padding: EdgeInsets.all(t.sp3),
              decoration: BoxDecoration(
                color: t.dangerSoft,
                borderRadius: BorderRadius.circular(t.rMd),
                border: Border.all(color: t.danger.withValues(alpha: 0.4)),
              ),
              child: Text(error!,
                  style: TextStyle(color: t.danger, fontSize: 13)),
            ),
          ],
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Sign-in card (step 3) — import existing OR run in-app OAuth
// -----------------------------------------------------------------------------

class _SigninCard extends StatelessWidget {
  final Map<String, dynamic> facts;
  final void Function(Map<String, dynamic> cred) onImport;
  final VoidCallback onDone;
  const _SigninCard(
      {required this.facts, required this.onImport, required this.onDone});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    final creds = (facts['credentials'] as List?) ?? const [];
    final unimported = creds
        .where((c) =>
            (c['valid'] as bool? ?? false) &&
            !(c['alreadyImported'] as bool? ?? false))
        .toList();

    return _Card(
      title: 'Sign in with Claude',
      subtitle:
          'Connect your Claude subscription so OpenDray can launch sessions on your behalf.',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (unimported.isNotEmpty) ...[
            Text('Existing credentials we can import:',
                style: theme.textTheme.bodyMedium),
            SizedBox(height: t.sp2),
            for (final cred in unimported)
              Container(
                margin: EdgeInsets.only(bottom: t.sp2),
                padding: EdgeInsets.all(t.sp3),
                decoration: BoxDecoration(
                  color: t.bgRaised,
                  borderRadius: BorderRadius.circular(t.rMd),
                  border: Border.all(color: t.border),
                ),
                child: Row(
                  children: [
                    Icon(Icons.account_circle, color: t.accent, size: 18),
                    SizedBox(width: t.sp3),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                              cred['subscription'] != null &&
                                      (cred['subscription'] as String).isNotEmpty
                                  ? 'Claude ${cred['subscription']}'
                                  : 'Claude account',
                              style: theme.textTheme.titleMedium
                                  ?.copyWith(fontWeight: FontWeight.w600)),
                          Text(cred['path'] as String? ?? '',
                              style: TextStyle(
                                  fontSize: 11,
                                  color: t.textSubtle,
                                  fontFamily: 'monospace'),
                              overflow: TextOverflow.ellipsis),
                        ],
                      ),
                    ),
                    OutlinedButton(
                        onPressed: () => onImport(
                            Map<String, dynamic>.from(cred as Map)),
                        child: const Text('Import')),
                  ],
                ),
              ),
            SizedBox(height: t.sp3),
            Text('Or sign in fresh:', style: theme.textTheme.bodyMedium),
            SizedBox(height: t.sp2),
          ],
          Row(
            children: [
              ElevatedButton.icon(
                onPressed: () => context.go('/settings/claude-accounts'),
                icon: const Icon(Icons.login, size: 16),
                label: const Text('Sign in with Claude'),
                style: ElevatedButton.styleFrom(
                  backgroundColor: const Color(0xFFCC785C),
                ),
              ),
              SizedBox(width: t.sp3),
              TextButton(
                  onPressed: onDone, child: const Text('Skip — take me to the Hub')),
            ],
          ),
          SizedBox(height: t.sp3),
          Container(
            padding: EdgeInsets.all(t.sp3),
            decoration: BoxDecoration(
              color: t.surface3,
              borderRadius: BorderRadius.circular(t.rMd),
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(Icons.lightbulb_outline,
                    size: 16, color: t.textMuted),
                SizedBox(width: t.sp2),
                Expanded(
                  child: Text(
                      "Don't have a Claude subscription? You can also bring your own API key from Settings → Connections later.",
                      style: theme.textTheme.bodySmall
                          ?.copyWith(color: t.textMuted)),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Done card
// -----------------------------------------------------------------------------

class _DoneCard extends StatelessWidget {
  final VoidCallback onContinue;
  const _DoneCard({required this.onContinue});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return _Card(
      title: "You're all set",
      subtitle: 'Take me to the Hub to start my first session.',
      child: ElevatedButton.icon(
        onPressed: onContinue,
        icon: const Icon(Icons.arrow_forward, size: 16),
        label: const Text('Continue to Hub'),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Card chrome shared by every step
// -----------------------------------------------------------------------------

class _Card extends StatelessWidget {
  final String title;
  final String? subtitle;
  final Widget child;
  const _Card({required this.title, this.subtitle, required this.child});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      padding: EdgeInsets.all(t.sp5),
      decoration: BoxDecoration(
        color: t.surface,
        borderRadius: BorderRadius.circular(t.rLg),
        border: Border.all(color: t.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title,
              style: Theme.of(context).textTheme.headlineSmall),
          if (subtitle != null) ...[
            SizedBox(height: t.sp1),
            Text(subtitle!,
                style: Theme.of(context)
                    .textTheme
                    .bodyMedium
                    ?.copyWith(color: t.textMuted)),
          ],
          SizedBox(height: t.sp4),
          child,
        ],
      ),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  final String message;
  const _ErrorBanner({required this.message});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      margin: EdgeInsets.only(bottom: t.sp4),
      padding: EdgeInsets.all(t.sp4),
      decoration: BoxDecoration(
        color: t.dangerSoft,
        borderRadius: BorderRadius.circular(t.rMd),
        border: Border.all(color: t.danger.withValues(alpha: 0.4)),
      ),
      child: Row(
        children: [
          Icon(Icons.error_outline, color: t.danger),
          SizedBox(width: t.sp2),
          Expanded(
              child: Text(message,
                  style: TextStyle(color: t.danger, fontSize: 13))),
        ],
      ),
    );
  }
}

