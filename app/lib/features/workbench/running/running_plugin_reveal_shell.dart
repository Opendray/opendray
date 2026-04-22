import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'running_plugins_models.dart';
import 'running_plugins_service.dart';

/// Route-level shim that ensures a plugin is in the running set and
/// marked active whenever its GoRoute is visited. Renders a
/// transparent placeholder; the actual plugin widget lives in
/// [RunningPluginsHost] underneath.
class RunningPluginRevealShell extends StatefulWidget {
  final RunningPluginEntry Function() seed;
  const RunningPluginRevealShell({required this.seed, super.key});

  @override
  State<RunningPluginRevealShell> createState() =>
      _RunningPluginRevealShellState();
}

class _RunningPluginRevealShellState extends State<RunningPluginRevealShell> {
  late final RunningPluginEntry _seed;

  @override
  void initState() {
    super.initState();
    // Resolve the seed exactly once — if the user taps back into this
    // plugin, the host will find the existing entry by id and keep its
    // pre-existing widget instance. The seed here is only used to
    // create the entry on first visit.
    _seed = widget.seed();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final service = context.read<RunningPluginsService>();
      service.ensureOpened(_seed);
      service.setActive(_seed.id);
    });
  }

  @override
  Widget build(BuildContext context) {
    // Transparent hole — the host paints the mounted plugin widget
    // below this layer. Returning SizedBox.expand keeps GoRouter's
    // page at the full shell body size so hit-tests land on the
    // plugin widget that's peeking through.
    return const SizedBox.expand();
  }
}
