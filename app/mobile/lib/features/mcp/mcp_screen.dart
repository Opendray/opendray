import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/mcp_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/mcp/mcp_editor_screen.dart';

// Two-section MCP control surface for mobile.
//
// SERVERS — read-only with toggle/delete. Operators can flip a
// server on/off (which gates whether session spawns inject it via
// the provider's MCP injection point) and delete one. Create + edit
// are web-only because the per-transport config is multi-field
// (command / args / env for stdio, url / headers for sse and http)
// and pasting multi-line shell args on a phone is a tax.
//
// SECRETS — keys-only listing of the encrypted vault. Operators can
// add a new key/value pair, rotate an existing value, or delete a
// key. Values are write-only by design — the API never returns
// them, mirroring how OS keychains behave.
class McpScreen extends ConsumerStatefulWidget {
  const McpScreen({super.key});

  @override
  ConsumerState<McpScreen> createState() => _McpScreenState();
}

class _McpScreenState extends ConsumerState<McpScreen> {
  AsyncValue<_Data> _state = const AsyncValue.loading();
  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final api = ref.read(mcpApiProvider);
      final results =
          await Future.wait([api.list(), api.secretsGet()]);
      if (!mounted) return;
      final servers = results[0] as List<McpServer>;
      final secrets = results[1] as McpSecretsState;
      servers.sort((a, b) {
        if (a.enabled != b.enabled) return a.enabled ? -1 : 1;
        return a.name.toLowerCase().compareTo(b.name.toLowerCase());
      });
      setState(
        () => _state = AsyncValue.data(
          _Data(servers: servers, secrets: secrets),
        ),
      );
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _runOp({
    required String key,
    required String okMsg,
    required String failPrefix,
    required Future<void> Function() op,
  }) async {
    setState(() => _busy.add(key));
    final messenger = ScaffoldMessenger.of(context);
    try {
      await op();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(okMsg),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.mcp.errorWithMessage(prefix: failPrefix, error: e.message),
          ),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.mcp.errorWithMessage(prefix: failPrefix, error: e.toString()),
          ),
        ),
      );
    } finally {
      if (mounted) setState(() => _busy.remove(key));
    }
  }

  Future<void> _onServerTap(McpServer s) async {
    final action = await showModalBottomSheet<_ServerAction>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    s.name,
                    style: Theme.of(sheetCtx).textTheme.titleSmall,
                  ),
                  Text(
                    s.id,
                    style: Theme.of(sheetCtx)
                        .textTheme
                        .bodySmall
                        ?.copyWith(fontFamily: 'monospace'),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.edit_outlined),
              title: Text(t.mcp.editConfig),
              subtitle: Text(
                t.mcp.popup.editConfigSubtitle,
                style: const TextStyle(fontSize: 11),
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_ServerAction.edit),
            ),
            ListTile(
              leading: const Icon(Icons.code),
              title: Text(t.mcp.viewRawConfig),
              subtitle: Text(
                t.mcp.popup.viewRawSubtitle,
                style: const TextStyle(fontSize: 11),
              ),
              onTap: () =>
                  Navigator.of(sheetCtx).pop(_ServerAction.viewConfig),
            ),
            ListTile(
              leading: const Icon(Icons.copy_outlined),
              title: Text(t.mcp.copyId),
              onTap: () => Navigator.of(sheetCtx).pop(_ServerAction.copyId),
            ),
            const Divider(height: 1),
            ListTile(
              leading: Icon(
                Icons.delete_outline,
                color: Theme.of(sheetCtx).colorScheme.error,
              ),
              title: Text(
                t.mcp.popup.deleteLabel,
                style: TextStyle(color: Theme.of(sheetCtx).colorScheme.error),
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_ServerAction.delete),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _ServerAction.edit:
        await _openEditor(existing: s);
      case _ServerAction.viewConfig:
        await _showServerConfig(s);
      case _ServerAction.copyId:
        await Clipboard.setData(ClipboardData(text: s.id));
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(t.mcp.copiedSnack(id: s.id)),
            duration: const Duration(seconds: 2),
            behavior: SnackBarBehavior.floating,
          ),
        );
      case _ServerAction.delete:
        await _confirmDeleteServer(s);
    }
  }

  Future<void> _openEditor({McpServer? existing}) async {
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => McpEditorScreen(existing: existing),
      ),
    );
    if ((saved ?? false) && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(existing == null
              ? t.mcp.serverCreatedSnack
              : t.mcp.serverUpdatedSnack),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    }
  }

  Future<void> _showServerConfig(McpServer s) async {
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(s.name),
        content: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.of(ctx).size.height * 0.6,
            maxWidth: 480,
          ),
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _kv('ID', s.id, mono: true),
                _kv(t.mcp.kv.transport, s.transport),
                if ((s.description ?? '').isNotEmpty)
                  _kv(t.mcp.kv.description, s.description!),
                if ((s.command ?? '').isNotEmpty)
                  _kv(t.mcp.kv.command, s.command!, mono: true),
                if (s.args != null && s.args!.isNotEmpty)
                  _kv(t.mcp.kv.args, s.args!.join('  '), mono: true),
                if ((s.url ?? '').isNotEmpty)
                  _kv('URL', s.url!, mono: true),
                if (s.env != null && s.env!.isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text(
                    t.mcp.envHeading,
                    style: Theme.of(ctx).textTheme.bodySmall,
                  ),
                  ...s.env!.entries.map(
                    (e) => Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: Text(
                        '${e.key}=${e.value}',
                        style: const TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 11,
                        ),
                      ),
                    ),
                  ),
                ],
                if (s.headers != null && s.headers!.isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text(
                    t.mcp.kv.headers,
                    style: Theme.of(ctx).textTheme.bodySmall,
                  ),
                  ...s.headers!.entries.map(
                    (e) => Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: Text(
                        '${e.key}: ${e.value}',
                        style: const TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 11,
                        ),
                      ),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
        actions: [
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(t.common.close),
          ),
        ],
      ),
    );
  }

  Future<void> _confirmDeleteServer(McpServer s) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.mcp.deleteServerTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(s.name, style: Theme.of(ctx).textTheme.titleSmall),
            const SizedBox(height: 4),
            Text(
              s.id,
              style: Theme.of(ctx)
                  .textTheme
                  .bodySmall
                  ?.copyWith(fontFamily: 'monospace'),
            ),
            const SizedBox(height: 12),
            Text(
              t.mcp.deleteServerBody(id: s.id),
              style: Theme.of(ctx).textTheme.bodySmall,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.common.delete),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    await _runOp(
      key: 's:${s.id}',
      okMsg: t.mcp.deleteServerSnack(id: s.id),
      failPrefix: t.mcp.errorPrefix.delete,
      op: () => ref.read(mcpApiProvider).delete(s.id),
    );
  }

  Future<void> _setSecret({String? existingKey}) async {
    final res = await Navigator.of(context).push<_SecretFormResult>(
      MaterialPageRoute<_SecretFormResult>(
        builder: (_) => _SecretFormScreen(existingKey: existingKey),
        fullscreenDialog: true,
      ),
    );
    if (res == null || !mounted) return;
    await _runOp(
      key: 'k:${res.key}',
      okMsg: existingKey == null
          ? t.mcp.secret.addedSnack(key: res.key)
          : t.mcp.secret.updatedSnack(key: res.key),
      failPrefix: existingKey == null
          ? t.mcp.errorPrefix.add
          : t.mcp.errorPrefix.update,
      op: () => ref
          .read(mcpApiProvider)
          .secretsSet(res.key, res.value)
          .then((_) {}),
    );
  }

  Future<void> _confirmDeleteSecret(String key) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.mcp.deleteSecretTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              key,
              style: const TextStyle(
                fontFamily: 'monospace',
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              t.mcp.secret.deleteBody,
              style: Theme.of(ctx).textTheme.bodySmall,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.common.delete),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    await _runOp(
      key: 'k:$key',
      okMsg: t.mcp.secret.deletedSnack(key: key),
      failPrefix: t.mcp.errorPrefix.delete,
      op: () => ref.read(mcpApiProvider).secretsDelete(key),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.mcp.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
            onPressed: _state is AsyncLoading ? null : _load,
          ),
        ],
      ),
      body: _state.when(
        data: _buildBody,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
      floatingActionButton: FloatingActionButton.extended(
        heroTag: 'mcp_fab',
        onPressed: _openEditor,
        icon: const Icon(Icons.add),
        label: Text(t.mcp.newServer),
      ),
    );
  }

  Widget _buildBody(_Data data) {
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        children: [
          _SectionHeader(
            label: t.mcp.serversCount(count: data.servers.length.toString()),
          ),
          if (data.servers.isEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 16, 12),
              child: Text(
                t.mcp.emptyServers,
                style: const TextStyle(fontSize: 12),
              ),
            )
          else
            for (final s in data.servers)
              _ServerTile(
                server: s,
                busy: _busy.contains('s:${s.id}'),
                onToggle: (next) => _runOp(
                  key: 's:${s.id}',
                  okMsg: next
                      ? t.mcp.toggleEnabledSnack(name: s.name)
                      : t.mcp.toggleDisabledSnack(name: s.name),
                  failPrefix: t.mcp.errorPrefix.toggle,
                  op: () => ref
                      .read(mcpApiProvider)
                      .setEnabled(s.id, enabled: next),
                ),
                onTap: () => _onServerTap(s),
              ),
          const SizedBox(height: 8),
          _SectionHeader(
            label: t.mcp.secretsCount(
              count: data.secrets.keys.length.toString(),
            ),
          ),
          _SecretsHeader(state: data.secrets),
          if (data.secrets.keys.isEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 6, 16, 12),
              child: Text(
                t.mcp.emptySecrets,
                style: const TextStyle(fontSize: 12),
              ),
            )
          else
            for (final k in data.secrets.keys)
              _SecretTile(
                name: k,
                busy: _busy.contains('k:$k'),
                onReplace: () => _setSecret(existingKey: k),
                onDelete: () => _confirmDeleteSecret(k),
              ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
            child: OutlinedButton.icon(
              icon: const Icon(Icons.add, size: 16),
              label: Text(t.mcp.addSecret),
              // Lambda needed because _setSecret returns Future<void>
              // — VoidCallback expects () -> void.
              // ignore: unnecessary_lambdas
              onPressed: () => _setSecret(),
            ),
          ),
        ],
      ),
    );
  }

  Widget _kv(String label, String value, {bool mono = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: const TextStyle(fontSize: 11, color: Colors.grey)),
          SelectableText(
            value,
            style: TextStyle(
              fontSize: 13,
              fontFamily: mono ? 'monospace' : null,
            ),
          ),
        ],
      ),
    );
  }
}

class _Data {
  _Data({required this.servers, required this.secrets});
  final List<McpServer> servers;
  final McpSecretsState secrets;
}

enum _ServerAction { edit, viewConfig, copyId, delete }

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 16, 6),
      child: Text(
        label.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
              letterSpacing: 0.8,
              color: Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.6),
            ),
      ),
    );
  }
}

class _ServerTile extends StatelessWidget {
  const _ServerTile({
    required this.server,
    required this.busy,
    required this.onToggle,
    required this.onTap,
  });
  final McpServer server;
  final bool busy;
  final ValueChanged<bool> onToggle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return ListTile(
      onTap: busy ? null : onTap,
      leading: Container(
        width: 36,
        height: 36,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: Theme.of(context)
              .colorScheme
              .primary
              .withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          server.name.isNotEmpty
              ? server.name[0].toUpperCase()
              : server.id[0].toUpperCase(),
          style: TextStyle(
            color: Theme.of(context).colorScheme.primary,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      title: Row(
        children: [
          Flexible(
            child: Text(
              server.name,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
          ),
          const SizedBox(width: 6),
          _MiniBadge(
            label: server.transport,
            color: Theme.of(context).colorScheme.tertiary,
          ),
        ],
      ),
      subtitle: DefaultTextStyle.merge(
        style: muted ?? const TextStyle(),
        child: Wrap(
          spacing: 6,
          runSpacing: 2,
          children: [
            Text(
              server.id,
              style: const TextStyle(fontFamily: 'monospace'),
            ),
            if ((server.description ?? '').isNotEmpty)
              Text('· ${server.description}'),
          ],
        ),
      ),
      trailing: busy
          ? const SizedBox(
              width: 32,
              height: 32,
              child: Padding(
                padding: EdgeInsets.all(8),
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            )
          : Switch(
              value: server.enabled,
              onChanged: onToggle,
            ),
    );
  }
}

class _SecretsHeader extends StatelessWidget {
  const _SecretsHeader({required this.state});
  final McpSecretsState state;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
      child: Row(
        children: [
          Icon(
            state.encrypted
                ? Icons.lock_outline
                : Icons.lock_open_outlined,
            size: 14,
            color: state.encrypted
                ? Colors.greenAccent
                : Theme.of(context).colorScheme.error,
          ),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              state.encrypted
                  ? t.mcp.encryptionAes
                  : state.present
                      ? t.mcp.encryptionPlaintext
                      : t.mcp.noVaultFileYet,
              style: muted,
            ),
          ),
        ],
      ),
    );
  }
}

class _SecretTile extends StatelessWidget {
  const _SecretTile({
    required this.name,
    required this.busy,
    required this.onReplace,
    required this.onDelete,
  });

  final String name;
  final bool busy;
  final VoidCallback onReplace;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      onTap: busy ? null : onReplace,
      leading: Icon(
        Icons.vpn_key_outlined,
        color: Theme.of(context).colorScheme.primary,
      ),
      title: Text(
        name,
        style: const TextStyle(fontFamily: 'monospace'),
      ),
      subtitle: Text(
        t.mcp.tapToReplaceHint,
        style: Theme.of(context).textTheme.bodySmall,
      ),
      onLongPress: busy ? null : onDelete,
      trailing: busy
          ? const SizedBox(
              width: 32,
              height: 32,
              child: Padding(
                padding: EdgeInsets.all(8),
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            )
          : IconButton(
              icon: Icon(
                Icons.delete_outline,
                color: Theme.of(context).colorScheme.error,
              ),
              tooltip: t.common.delete,
              onPressed: onDelete,
            ),
    );
  }
}

class _MiniBadge extends StatelessWidget {
  const _MiniBadge({required this.label, required this.color});
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withValues(alpha: 0.6), width: 0.5),
      ),
      child: Text(
        label,
        style: TextStyle(color: color, fontSize: 10),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final String error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              t.mcp.failedToLoad,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}

class _SecretFormResult {
  _SecretFormResult({required this.key, required this.value});
  final String key;
  final String value;
}

// _SecretFormScreen is the Add / Replace flow. Single TextField for
// the value (multi-line so multi-line API keys / cert blobs paste
// without truncation; not obscured because the operator just pasted
// it from elsewhere and obscure + multiline is incompatible in
// Flutter anyway).
class _SecretFormScreen extends StatefulWidget {
  const _SecretFormScreen({this.existingKey});
  // When set, locks the key field — this is a "replace value" flow.
  final String? existingKey;

  @override
  State<_SecretFormScreen> createState() => _SecretFormScreenState();
}

class _SecretFormScreenState extends State<_SecretFormScreen> {
  late final TextEditingController _key;
  final _value = TextEditingController();
  String? _error;

  @override
  void initState() {
    super.initState();
    _key = TextEditingController(text: widget.existingKey ?? '');
  }

  @override
  void dispose() {
    _key.dispose();
    _value.dispose();
    super.dispose();
  }

  void _submit() {
    final key = _key.text.trim();
    final value = _value.text;
    if (key.isEmpty) {
      setState(() => _error = t.mcp.secret.keyRequired);
      return;
    }
    final keyOk = RegExp(r'^[A-Za-z_][A-Za-z0-9_]*$').hasMatch(key);
    if (!keyOk) {
      setState(() => _error = t.mcp.secret.keyInvalid);
      return;
    }
    if (value.isEmpty) {
      setState(() => _error = t.mcp.secret.valueRequired);
      return;
    }
    Navigator.of(context).pop(_SecretFormResult(key: key, value: value));
  }

  @override
  Widget build(BuildContext context) {
    final isReplace = widget.existingKey != null;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          isReplace ? t.mcp.secret.replaceTitle : t.mcp.secret.addTitle,
        ),
        actions: [
          TextButton(
            onPressed: _submit,
            child: Text(
              isReplace ? t.mcp.secret.saveButton : t.mcp.secret.addButton,
            ),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          TextField(
            controller: _key,
            autofocus: !isReplace,
            autocorrect: false,
            enabled: !isReplace,
            decoration: InputDecoration(
              labelText: t.mcp.secret.keyLabel,
              hintText: t.mcp.secret.keyHint,
              helperText: t.mcp.secret.helpRules,
              helperMaxLines: 2,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _value,
            autofocus: isReplace,
            autocorrect: false,
            maxLines: 6,
            // Visible by design — operator just pasted it; obscure +
            // multiline is forbidden by Flutter, and a single-line
            // capped field would truncate certificate blobs.
            decoration: InputDecoration(
              labelText: t.mcp.secret.valueLabel,
              hintText: isReplace
                  ? t.mcp.secret.replaceHint
                  : t.mcp.secret.addHint,
              border: const OutlineInputBorder(),
            ),
          ),
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(
              _error!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
                fontSize: 12,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
