///
/// Generated file. Do not edit.
///
// coverage:ignore-file
// ignore_for_file: type=lint, unused_import
// dart format off

part of 'strings.g.dart';

// Path: <root>
typedef TranslationsEn = Translations; // ignore: unused_element
class Translations with BaseTranslations<AppLocale, Translations> {
	/// Returns the current translations of the given [context].
	///
	/// Usage:
	/// final t = Translations.of(context);
	static Translations of(BuildContext context) => InheritedLocaleData.of<AppLocale, Translations>(context).translations;

	/// You can call this constructor and build your own translation instance of this locale.
	/// Constructing via the enum [AppLocale.build] is preferred.
	Translations({Map<String, Node>? overrides, PluralResolver? cardinalResolver, PluralResolver? ordinalResolver, TranslationMetadata<AppLocale, Translations>? meta})
		: assert(overrides == null, 'Set "translation_overrides: true" in order to enable this feature.'),
		  $meta = meta ?? TranslationMetadata(
		    locale: AppLocale.en,
		    overrides: overrides ?? {},
		    cardinalResolver: cardinalResolver,
		    ordinalResolver: ordinalResolver,
		  ) {
		$meta.setFlatMapFunction(_flatMapFunction);
	}

	/// Metadata for the translations of <en>.
	@override final TranslationMetadata<AppLocale, Translations> $meta;

	/// Access flat map
	dynamic operator[](String key) => $meta.getTranslation(key);

	late final Translations _root = this; // ignore: unused_field

	Translations $copyWith({TranslationMetadata<AppLocale, Translations>? meta}) => Translations(meta: meta ?? this.$meta);

	// Translations
	late final TranslationsCommonEn common = TranslationsCommonEn.internal(_root);
	late final TranslationsAuthEn auth = TranslationsAuthEn.internal(_root);
	late final TranslationsNavEn nav = TranslationsNavEn.internal(_root);
	late final TranslationsMoreEn more = TranslationsMoreEn.internal(_root);
	late final TranslationsSessionsEn sessions = TranslationsSessionsEn.internal(_root);
	late final TranslationsMcpEn mcp = TranslationsMcpEn.internal(_root);
	late final TranslationsProvidersEn providers = TranslationsProvidersEn.internal(_root);
	late final TranslationsIntegrationsEn integrations = TranslationsIntegrationsEn.internal(_root);
	late final TranslationsMemoryWorkersEn memoryWorkers = TranslationsMemoryWorkersEn.internal(_root);
	late final TranslationsMemoryCleanupEn memoryCleanup = TranslationsMemoryCleanupEn.internal(_root);
	late final TranslationsProjectEn project = TranslationsProjectEn.internal(_root);
	late final TranslationsBackupsEn backups = TranslationsBackupsEn.internal(_root);
	late final TranslationsBackupTargetsEn backupTargets = TranslationsBackupTargetsEn.internal(_root);
	late final TranslationsBackupSchedulesEn backupSchedules = TranslationsBackupSchedulesEn.internal(_root);
	late final TranslationsBackupTargetEditorEn backupTargetEditor = TranslationsBackupTargetEditorEn.internal(_root);
	late final TranslationsGithostsEn githosts = TranslationsGithostsEn.internal(_root);
	late final TranslationsChannelsEn channels = TranslationsChannelsEn.internal(_root);
	late final TranslationsOnboardingEn onboarding = TranslationsOnboardingEn.internal(_root);
	late final TranslationsSkillsEn skills = TranslationsSkillsEn.internal(_root);
	late final TranslationsCustomTasksEn customTasks = TranslationsCustomTasksEn.internal(_root);
	late final TranslationsNotesPageEn notesPage = TranslationsNotesPageEn.internal(_root);
	late final TranslationsMemoryEn memory = TranslationsMemoryEn.internal(_root);
	late final TranslationsAboutEn about = TranslationsAboutEn.internal(_root);
	late final TranslationsSettingsEn settings = TranslationsSettingsEn.internal(_root);
}

// Path: common
class TranslationsCommonEn {
	TranslationsCommonEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'OK'
	String get ok => 'OK';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Delete'
	String get delete => 'Delete';

	/// en: 'Edit'
	String get edit => 'Edit';

	/// en: 'Back'
	String get back => 'Back';

	/// en: 'Done'
	String get done => 'Done';

	/// en: 'Close'
	String get close => 'Close';

	/// en: 'Retry'
	String get retry => 'Retry';

	/// en: 'Copy'
	String get copy => 'Copy';

	/// en: 'Enabled'
	String get enabled => 'Enabled';

	/// en: 'Refresh'
	String get refresh => 'Refresh';
}

// Path: auth
class TranslationsAuthEn {
	TranslationsAuthEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Sign in'
	String get signInTitle => 'Sign in';

	/// en: 'Change'
	String get changeServer => 'Change';

	/// en: 'Username'
	String get username => 'Username';

	/// en: 'Password'
	String get password => 'Password';

	/// en: 'Sign in'
	String get signIn => 'Sign in';

	/// en: 'Username and password are required'
	String get errorRequired => 'Username and password are required';

	/// en: 'Login failed: {error}'
	String errorGeneric({required Object error}) => 'Login failed: ${error}';
}

// Path: nav
class TranslationsNavEn {
	TranslationsNavEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Sessions'
	String get sessions => 'Sessions';

	/// en: 'Memory'
	String get memory => 'Memory';

	/// en: 'Notes'
	String get notes => 'Notes';

	/// en: 'More'
	String get more => 'More';
}

// Path: more
class TranslationsMoreEn {
	TranslationsMoreEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'More'
	String get title => 'More';

	late final TranslationsMoreIdentityEn identity = TranslationsMoreIdentityEn.internal(_root);
	late final TranslationsMoreSectionsEn sections = TranslationsMoreSectionsEn.internal(_root);
	late final TranslationsMoreItemsEn items = TranslationsMoreItemsEn.internal(_root);

	/// en: 'Sign out'
	String get signOut => 'Sign out';
}

// Path: sessions
class TranslationsSessionsEn {
	TranslationsSessionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Sessions'
	String get title => 'Sessions';

	/// en: 'Refresh'
	String get refresh => 'Refresh';

	/// en: 'Actions'
	String get actions => 'Actions';

	/// en: 'Spawn'
	String get spawn => 'Spawn';

	late final TranslationsSessionsFiltersEn filters = TranslationsSessionsFiltersEn.internal(_root);
	late final TranslationsSessionsCardEn card = TranslationsSessionsCardEn.internal(_root);
	late final TranslationsSessionsEmptyEn empty = TranslationsSessionsEmptyEn.internal(_root);

	/// en: 'Failed to load sessions'
	String get errorTitle => 'Failed to load sessions';

	late final TranslationsSessionsRelativeEn relative = TranslationsSessionsRelativeEn.internal(_root);
	late final TranslationsSessionsDetailEn detail = TranslationsSessionsDetailEn.internal(_root);
	late final TranslationsSessionsTerminalEn terminal = TranslationsSessionsTerminalEn.internal(_root);
	late final TranslationsSessionsActionEn action = TranslationsSessionsActionEn.internal(_root);
	late final TranslationsSessionsDirPickerEn dirPicker = TranslationsSessionsDirPickerEn.internal(_root);
	late final TranslationsSessionsInspectorEn inspector = TranslationsSessionsInspectorEn.internal(_root);
	late final TranslationsSessionsSpawnSheetEn spawnSheet = TranslationsSessionsSpawnSheetEn.internal(_root);
}

// Path: mcp
class TranslationsMcpEn {
	TranslationsMcpEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'MCP'
	String get title => 'MCP';

	/// en: 'New server'
	String get newServer => 'New server';

	/// en: 'Add secret'
	String get addSecret => 'Add secret';

	/// en: 'Edit config'
	String get editConfig => 'Edit config';

	/// en: 'View raw config'
	String get viewRawConfig => 'View raw config';

	/// en: 'Copy id'
	String get copyId => 'Copy id';

	/// en: 'Copied {id}'
	String copiedSnack({required Object id}) => 'Copied ${id}';

	/// en: 'Delete MCP server?'
	String get deleteServerTitle => 'Delete MCP server?';

	/// en: 'Delete secret?'
	String get deleteSecretTitle => 'Delete secret?';

	late final TranslationsMcpErrorPrefixEn errorPrefix = TranslationsMcpErrorPrefixEn.internal(_root);

	/// en: '{prefix}: {error}'
	String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';

	late final TranslationsMcpEditorEn editor = TranslationsMcpEditorEn.internal(_root);
	late final TranslationsMcpSecretEn secret = TranslationsMcpSecretEn.internal(_root);
	late final TranslationsMcpPopupEn popup = TranslationsMcpPopupEn.internal(_root);
	late final TranslationsMcpKvEn kv = TranslationsMcpKvEn.internal(_root);

	/// en: 'Removes the vault directory for {id}. Sessions that reference this server stop being able to spawn it.'
	String deleteServerBody({required Object id}) => 'Removes the vault directory for ${id}. Sessions that reference this server stop being able to spawn it.';

	/// en: 'Deleted {id}.'
	String deleteServerSnack({required Object id}) => 'Deleted ${id}.';

	/// en: 'Servers ({count})'
	String serversCount({required Object count}) => 'Servers (${count})';

	/// en: 'Secrets ({count})'
	String secretsCount({required Object count}) => 'Secrets (${count})';

	/// en: 'No MCP servers registered. Tap "New server" to add one.'
	String get emptyServers => 'No MCP servers registered. Tap "New server" to add one.';

	/// en: 'No secrets stored. Add one to feed sensitive env / headers into MCP servers without putting them in the JSON.'
	String get emptySecrets => 'No secrets stored. Add one to feed sensitive env / headers into MCP servers without putting them in the JSON.';

	/// en: 'No vault file yet — added secrets create it.'
	String get noVaultFileYet => 'No vault file yet — added secrets create it.';

	/// en: 'Tap to replace · long-press / trash to delete'
	String get tapToReplaceHint => 'Tap to replace · long-press / trash to delete';

	/// en: 'Failed to load MCP state'
	String get failedToLoad => 'Failed to load MCP state';

	/// en: 'MCP server created.'
	String get serverCreatedSnack => 'MCP server created.';

	/// en: 'MCP server updated.'
	String get serverUpdatedSnack => 'MCP server updated.';

	/// en: 'Env'
	String get envHeading => 'Env';

	/// en: 'AES-GCM encrypted (key in OS keychain)'
	String get encryptionAes => 'AES-GCM encrypted (key in OS keychain)';

	/// en: 'PLAINTEXT — keychain unavailable'
	String get encryptionPlaintext => 'PLAINTEXT — keychain unavailable';

	/// en: '{name} enabled.'
	String toggleEnabledSnack({required Object name}) => '${name} enabled.';

	/// en: '{name} disabled.'
	String toggleDisabledSnack({required Object name}) => '${name} disabled.';
}

// Path: providers
class TranslationsProvidersEn {
	TranslationsProvidersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Providers'
	String get title => 'Providers';

	/// en: 'Provider config updated.'
	String get configSaved => 'Provider config updated.';

	/// en: 'Save failed: {error}'
	String saveFailedApi({required Object error}) => 'Save failed: ${error}';

	/// en: 'Save failed: {error}'
	String saveFailedGeneric({required Object error}) => 'Save failed: ${error}';

	/// en: 'Reload'
	String get reload => 'Reload';

	late final TranslationsProvidersErrorPrefixEn errorPrefix = TranslationsProvidersErrorPrefixEn.internal(_root);

	/// en: '{prefix}: {error}'
	String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';

	late final TranslationsProvidersAccountsEn accounts = TranslationsProvidersAccountsEn.internal(_root);
}

// Path: integrations
class TranslationsIntegrationsEn {
	TranslationsIntegrationsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Integrations'
	String get title => 'Integrations';

	/// en: 'Register'
	String get register => 'Register';

	/// en: 'Register integration'
	String get registerDialogTitle => 'Register integration';

	/// en: 'Edit'
	String get edit => 'Edit';

	/// en: 'Edit {name}'
	String editTitle({required Object name}) => 'Edit ${name}';

	/// en: 'Enabled'
	String get enabledLabel => 'Enabled';

	/// en: 'I've saved it'
	String get iSavedIt => 'I\'ve saved it';

	/// en: 'API key for {name}'
	String apiKeyForName({required Object name}) => 'API key for ${name}';

	/// en: 'Hand this to the integration so it can authenticate against /api/v1/{routePrefix}/…'
	String apiKeySubtitleRegister({required Object routePrefix}) => 'Hand this to the integration so it can authenticate against /api/v1/${routePrefix}/…';

	/// en: 'Copied request_id {id}'
	String copiedRequestId({required Object id}) => 'Copied request_id ${id}';

	/// en: 'Integration updated.'
	String get updateOk => 'Integration updated.';

	/// en: 'Register failed: {error}'
	String registerFailedApi({required Object error}) => 'Register failed: ${error}';

	/// en: 'Register failed: {error}'
	String registerFailedGeneric({required Object error}) => 'Register failed: ${error}';

	/// en: 'Update failed: {error}'
	String updateFailedApi({required Object error}) => 'Update failed: ${error}';

	/// en: 'Update failed: {error}'
	String updateFailedGeneric({required Object error}) => 'Update failed: ${error}';

	/// en: 'Delete integration?'
	String get deleteTitle => 'Delete integration?';

	/// en: 'Deleted {name}.'
	String deletedSnack({required Object name}) => 'Deleted ${name}.';

	/// en: 'Delete failed: {error}'
	String deleteFailedApi({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Delete failed: {error}'
	String deleteFailedGeneric({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Rotate key'
	String get rotateKey => 'Rotate key';

	/// en: 'Rotate API key?'
	String get rotateConfirmTitle => 'Rotate API key?';

	/// en: 'Rotate'
	String get rotate => 'Rotate';

	/// en: 'New API key for {name}'
	String newApiKeyTitle({required Object name}) => 'New API key for ${name}';

	/// en: 'Hand this to the integration. The previous key has just been invalidated.'
	String get newApiKeySubtitle => 'Hand this to the integration. The previous key has just been invalidated.';

	/// en: 'Rotate failed: {error}'
	String rotateFailedApi({required Object error}) => 'Rotate failed: ${error}';

	/// en: 'Rotate failed: {error}'
	String rotateFailedGeneric({required Object error}) => 'Rotate failed: ${error}';
}

// Path: memoryWorkers
class TranslationsMemoryWorkersEn {
	TranslationsMemoryWorkersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory workers'
	String get title => 'Memory workers';

	/// en: '{label} saved'
	String savedSnack({required Object label}) => '${label} saved';

	/// en: 'Save failed: {error}'
	String saveFailed({required Object error}) => 'Save failed: ${error}';

	/// en: 'Test call failed: {error}'
	String testFailed({required Object error}) => 'Test call failed: ${error}';

	/// en: 'Worker'
	String get workerLabel => 'Worker';

	/// en: 'Summarizer (HTTP)'
	String get summarizerHttp => 'Summarizer (HTTP)';

	/// en: 'Agent (CLI --print)'
	String get agentCliPrint => 'Agent (CLI --print)';

	/// en: 'CLI'
	String get cliLabel => 'CLI';

	/// en: 'Claude'
	String get cliClaude => 'Claude';

	/// en: 'Gemini'
	String get cliGemini => 'Gemini';

	/// en: 'Claude account'
	String get claudeAccountLabel => 'Claude account';

	/// en: 'Default'
	String get claudeAccountDefault => 'Default';

	/// en: 'Test'
	String get test => 'Test';

	/// en: 'Each memory-system LLM touchpoint can be served independently by the local summarizer endpoint (LM Studio / OpenAI-compat) or by spawning a headless Claude / Gemini agent in --print mode. High-quality narrative tasks (gitactivity, transcript) benefit from agent workers; high-frequency tasks (gatekeeper) stay on the local endpoint by design.'
	String get intro => 'Each memory-system LLM touchpoint can be served independently by the local summarizer endpoint (LM Studio / OpenAI-compat) or by spawning a headless Claude / Gemini agent in --print mode. High-quality narrative tasks (gitactivity, transcript) benefit from agent workers; high-frequency tasks (gatekeeper) stay on the local endpoint by design.';

	/// en: 'Endpoint not reachable'
	String get errorTitle => 'Endpoint not reachable';

	/// en: 'The /api/v1/memory/workers routes are new in M25 — the opendray binary may need a restart to mount them and run migration 0029.'
	String get errorDetail => 'The /api/v1/memory/workers routes are new in M25 — the opendray binary may need a restart to mount them and run migration 0029.';

	/// en: 'summarizer-only'
	String get summarizerOnlyBadge => 'summarizer-only';

	/// en: 'Uses the registry default summarizer provider. Pick a specific row on the web admin.'
	String get summarizerInfo => 'Uses the registry default summarizer provider. Pick a specific row on the web admin.';

	/// en: 'Agent mode spawns a headless CLI per call. Latency ~5-15s (vs ~1s summarizer); cost shifts from CPU to your Claude/Gemini quota.'
	String get agentWarning => 'Agent mode spawns a headless CLI per call. Latency ~5-15s (vs ~1s summarizer); cost shifts from CPU to your Claude/Gemini quota.';

	/// en: 'No calls in last 24h.'
	String get noCalls24h => 'No calls in last 24h.';

	/// en: '{label} OK — {duration}ms'
	String testOkSnack({required Object label, required Object duration}) => '${label} OK — ${duration}ms';

	/// en: '{label} failed: {error}'
	String testFailedReturnedSnack({required Object label, required Object error}) => '${label} failed: ${error}';

	/// en: 'unknown'
	String get unknownError => 'unknown';

	late final TranslationsMemoryWorkersTasksEn tasks = TranslationsMemoryWorkersTasksEn.internal(_root);
}

// Path: memoryCleanup
class TranslationsMemoryCleanupEn {
	TranslationsMemoryCleanupEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory cleanup'
	String get title => 'Memory cleanup';

	/// en: 'Approve failed: {error}'
	String approveFailed({required Object error}) => 'Approve failed: ${error}';

	/// en: 'Reject failed: {error}'
	String rejectFailed({required Object error}) => 'Reject failed: ${error}';

	/// en: 'Failed to load: {error}'
	String loadFailed({required Object error}) => 'Failed to load: ${error}';

	/// en: 'Reject'
	String get reject => 'Reject';
}

// Path: project
class TranslationsProjectEn {
	TranslationsProjectEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Project'
	String get title => 'Project';

	/// en: 'Pick a project first.'
	String get pickFirst => 'Pick a project first.';

	/// en: 'Failed to load: {error}'
	String loadFailed({required Object error}) => 'Failed to load: ${error}';

	/// en: 'Failed to load projects: {error}'
	String projectsLoadFailed({required Object error}) => 'Failed to load projects: ${error}';

	/// en: 'Project'
	String get projectLabel => 'Project';

	/// en: 'Reset project memory'
	String get resetTooltip => 'Reset project memory';

	/// en: 'Append'
	String get append => 'Append';

	/// en: 'Append journal entry'
	String get appendDialogTitle => 'Append journal entry';

	/// en: 'Title (optional)'
	String get titleFieldLabel => 'Title (optional)';

	/// en: 'Content (markdown)'
	String get contentFieldLabel => 'Content (markdown)';

	/// en: 'Failed: {error}'
	String appendFailed({required Object error}) => 'Failed: ${error}';

	/// en: 'Approve failed: {error}'
	String approveFailed({required Object error}) => 'Approve failed: ${error}';

	/// en: 'Reject failed: {error}'
	String rejectFailed({required Object error}) => 'Reject failed: ${error}';

	/// en: 'Cleanup failed: {error}'
	String cleanupFailed({required Object error}) => 'Cleanup failed: ${error}';

	/// en: 'Reset project memory?'
	String get resetConfirmTitle => 'Reset project memory?';

	/// en: 'Also delete scanner docs'
	String get alsoDeleteScanner => 'Also delete scanner docs';

	/// en: 'Also delete pgvector memories'
	String get alsoDeletePgvector => 'Also delete pgvector memories';

	/// en: 'Delete forever'
	String get deleteForever => 'Delete forever';

	/// en: 'Reset: {parts}'
	String resetDoneSnack({required Object parts}) => 'Reset: ${parts}';

	/// en: 'Reset failed: {error}'
	String resetFailed({required Object error}) => 'Reset failed: ${error}';

	/// en: '{kind} saved'
	String docSavedSnack({required Object kind}) => '${kind} saved';

	/// en: 'Save failed: {error}'
	String docSaveFailed({required Object error}) => 'Save failed: ${error}';

	/// en: 'Write the {kind} as markdown…'
	String docHintTemplate({required Object kind}) => 'Write the ${kind} as markdown…';

	/// en: 'Delete entry'
	String get deleteEntryTooltip => 'Delete entry';

	/// en: 'Agent reason'
	String get agentReason => 'Agent reason';

	/// en: 'Reject'
	String get reject => 'Reject';

	/// en: 'Approve'
	String get approve => 'Approve';

	/// en: 'Replace current {kind}?'
	String replaceConfirmTitle({required Object kind}) => 'Replace current ${kind}?';

	/// en: 'Replace {kind}'
	String replaceKind({required Object kind}) => 'Replace ${kind}';

	/// en: 'Reason'
	String get reason => 'Reason';

	/// en: 'Will merge into'
	String get willMergeInto => 'Will merge into';
}

// Path: backups
class TranslationsBackupsEn {
	TranslationsBackupsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backups'
	String get title => 'Backups';

	/// en: 'Run backup now?'
	String get runConfirmTitle => 'Run backup now?';

	/// en: 'Run'
	String get run => 'Run';

	/// en: 'Backup queued ({id}). Watching for progress…'
	String queuedSnack({required Object id}) => 'Backup queued (${id}). Watching for progress…';

	/// en: 'Run failed: {error}'
	String runFailedApi({required Object error}) => 'Run failed: ${error}';

	/// en: 'Run failed: {error}'
	String runFailedGeneric({required Object error}) => 'Run failed: ${error}';

	/// en: 'Backup detail'
	String get detailTitle => 'Backup detail';

	/// en: 'Delete backup?'
	String get deleteTitle => 'Delete backup?';

	/// en: 'Deleted {id}.'
	String deletedSnack({required Object id}) => 'Deleted ${id}.';

	/// en: 'Delete failed: {error}'
	String deleteFailedApi({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Delete failed: {error}'
	String deleteFailedGeneric({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Schedules'
	String get menuSchedules => 'Schedules';

	/// en: 'Targets'
	String get menuTargets => 'Targets';

	late final TranslationsBackupsEncryptionEn encryption = TranslationsBackupsEncryptionEn.internal(_root);
}

// Path: backupTargets
class TranslationsBackupTargetsEn {
	TranslationsBackupTargetsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backup targets'
	String get title => 'Backup targets';

	/// en: 'New target'
	String get newTarget => 'New target';

	/// en: 'Test connection'
	String get testConnection => 'Test connection';

	/// en: 'Edit config'
	String get editConfig => 'Edit config';

	/// en: 'View raw config'
	String get viewRawConfig => 'View raw config';

	/// en: '{kind} config'
	String configDialogTitle({required Object kind}) => '${kind} config';

	/// en: 'Delete target?'
	String get deleteTitle => 'Delete target?';

	/// en: '{prefix}: {error}'
	String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';
}

// Path: backupSchedules
class TranslationsBackupSchedulesEn {
	TranslationsBackupSchedulesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backup schedules'
	String get title => 'Backup schedules';

	/// en: 'Delete schedule?'
	String get deleteTitle => 'Delete schedule?';

	/// en: 'Target'
	String get targetLabel => 'Target';

	/// en: 'Interval'
	String get intervalLabel => 'Interval';

	/// en: 'Retention (keep N most recent)'
	String get retentionLabel => 'Retention (keep N most recent)';

	/// en: '{prefix}: {error}'
	String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';
}

// Path: backupTargetEditor
class TranslationsBackupTargetEditorEn {
	TranslationsBackupTargetEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Use HTTPS'
	String get useHttps => 'Use HTTPS';

	/// en: 'Path-style addressing'
	String get pathStyle => 'Path-style addressing';

	/// en: 'Legacy / MinIO'
	String get pathStyleSubtitle => 'Legacy / MinIO';
}

// Path: githosts
class TranslationsGithostsEn {
	TranslationsGithostsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Git hosts'
	String get title => 'Git hosts';

	/// en: 'Add host'
	String get addHost => 'Add host';

	/// en: 'Delete git host?'
	String get deleteTitle => 'Delete git host?';

	/// en: '{prefix}: {error}'
	String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';

	late final TranslationsGithostsErrorPrefixEn errorPrefix = TranslationsGithostsErrorPrefixEn.internal(_root);
	late final TranslationsGithostsFormEn form = TranslationsGithostsFormEn.internal(_root);
}

// Path: channels
class TranslationsChannelsEn {
	TranslationsChannelsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Channels'
	String get title => 'Channels';

	/// en: 'New'
	String get kNew => 'New';

	/// en: 'Send test message'
	String get sendTest => 'Send test message';

	/// en: 'Edit config'
	String get editConfig => 'Edit config';

	/// en: 'Edit notifications'
	String get editNotifications => 'Edit notifications';

	/// en: 'View raw config'
	String get viewRawConfig => 'View raw config';

	/// en: 'Copy channel id'
	String get copyChannelId => 'Copy channel id';

	/// en: 'Copied {id}'
	String copiedSnack({required Object id}) => 'Copied ${id}';

	/// en: 'Created {kind} channel.'
	String createdSnack({required Object kind}) => 'Created ${kind} channel.';

	/// en: 'Create failed: {error}'
	String createFailedApi({required Object error}) => 'Create failed: ${error}';

	/// en: 'Create failed: {error}'
	String createFailedGeneric({required Object error}) => 'Create failed: ${error}';

	/// en: 'Delete channel?'
	String get deleteTitle => 'Delete channel?';

	late final TranslationsChannelsConfigDialogEn configDialog = TranslationsChannelsConfigDialogEn.internal(_root);
	late final TranslationsChannelsWebhookDialogEn webhookDialog = TranslationsChannelsWebhookDialogEn.internal(_root);

	/// en: '{prefix}: {error}'
	String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';

	late final TranslationsChannelsNotificationsEn notifications = TranslationsChannelsNotificationsEn.internal(_root);
	late final TranslationsChannelsPopupEn popup = TranslationsChannelsPopupEn.internal(_root);
	late final TranslationsChannelsBadgesEn badges = TranslationsChannelsBadgesEn.internal(_root);

	/// en: '· caps: {list}'
	String capsLabel({required Object list}) => '· caps: ${list}';

	/// en: 'Bridge channels stay web-only'
	String get bridgeWebOnly => 'Bridge channels stay web-only';

	/// en: 'Add one from the web admin: Channels → New.'
	String get bridgeEmptyAdd => 'Add one from the web admin: Channels → New.';

	/// en: 'Stops the channel and removes its configuration. In-flight notifications addressed to it will be dropped silently.'
	String get deleteBody => 'Stops the channel and removes its configuration. In-flight notifications addressed to it will be dropped silently.';

	late final TranslationsChannelsSnacksEn snacks = TranslationsChannelsSnacksEn.internal(_root);
	late final TranslationsChannelsErrorPrefixEn errorPrefix = TranslationsChannelsErrorPrefixEn.internal(_root);

	/// en: 'Failed to load channels'
	String get failedToLoad => 'Failed to load channels';

	late final TranslationsChannelsKindsEn kinds = TranslationsChannelsKindsEn.internal(_root);
}

// Path: onboarding
class TranslationsOnboardingEn {
	TranslationsOnboardingEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Gateway URL'
	String get gatewayLabel => 'Gateway URL';

	/// en: 'https://opendray.example.com'
	String get gatewayHint => 'https://opendray.example.com';

	/// en: 'Continue'
	String get kContinue => 'Continue';
}

// Path: skills
class TranslationsSkillsEn {
	TranslationsSkillsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Skills'
	String get title => 'Skills';

	/// en: 'New skill'
	String get newSkill => 'New skill';

	/// en: 'Customizing built-in {id}'
	String customizingBuiltin({required Object id}) => 'Customizing built-in ${id}';

	/// en: 'Id (slug)'
	String get idLabel => 'Id (slug)';

	/// en: 'e.g. tdd-guide'
	String get idHint => 'e.g. tdd-guide';

	/// en: 'Body (markdown)'
	String get bodyLabel => 'Body (markdown)';
}

// Path: customTasks
class TranslationsCustomTasksEn {
	TranslationsCustomTasksEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Custom tasks'
	String get title => 'Custom tasks';

	/// en: 'New task'
	String get newTask => 'New task';

	/// en: 'Delete task?'
	String get deleteTitle => 'Delete task?';

	/// en: 'Deleted {name}.'
	String deletedSnack({required Object name}) => 'Deleted ${name}.';

	/// en: 'Delete failed: {error}'
	String deleteFailedApi({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Delete failed: {error}'
	String deleteFailedGeneric({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Edit'
	String get popupEdit => 'Edit';

	/// en: 'Delete'
	String get popupDelete => 'Delete';

	/// en: 'e.g. backend-tests'
	String get nameHint => 'e.g. backend-tests';

	/// en: '/run pnpm test --filter backend'
	String get commandHint => '/run pnpm test --filter backend';

	/// en: 'One-liner shown under the task name.'
	String get descriptionHint => 'One-liner shown under the task name.';

	/// en: 'Global'
	String get scopeGlobal => 'Global';

	/// en: 'Project'
	String get scopeProject => 'Project';

	/// en: '/Users/you/projects/backend'
	String get cwdHint => '/Users/you/projects/backend';
}

// Path: notesPage
class TranslationsNotesPageEn {
	TranslationsNotesPageEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Notes'
	String get title => 'Notes';

	/// en: 'New'
	String get newButton => 'New';

	/// en: 'New note'
	String get newNoteDialogTitle => 'New note';

	/// en: 'Search across the whole vault…'
	String get searchHint => 'Search across the whole vault…';

	/// en: 'Up'
	String get up => 'Up';

	/// en: 'Copy path'
	String get copyPath => 'Copy path';

	/// en: 'Open'
	String get open => 'Open';

	/// en: 'Copied {path}'
	String copiedSnack({required Object path}) => 'Copied ${path}';

	/// en: 'Delete note?'
	String get deleteTitle => 'Delete note?';

	/// en: 'Deleted {path}'
	String deletedSnack({required Object path}) => 'Deleted ${path}';

	/// en: 'Delete failed: {error}'
	String deleteFailedApi({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Delete failed: {error}'
	String deleteFailedGeneric({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Create failed: {error}'
	String createFailedApi({required Object error}) => 'Create failed: ${error}';

	/// en: 'Create failed: {error}'
	String createFailedGeneric({required Object error}) => 'Create failed: ${error}';

	/// en: 'Vault-relative path'
	String get pathLabel => 'Vault-relative path';

	/// en: 'personal/scratch.md'
	String get pathHint => 'personal/scratch.md';

	/// en: 'Create'
	String get create => 'Create';

	late final TranslationsNotesPageEditorEn editor = TranslationsNotesPageEditorEn.internal(_root);
}

// Path: memory
class TranslationsMemoryEn {
	TranslationsMemoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory'
	String get title => 'Memory';

	/// en: 'More'
	String get more => 'More';

	/// en: 'Memory workers'
	String get workers => 'Memory workers';

	/// en: 'New'
	String get kNew => 'New';

	/// en: 'Search…'
	String get searchHint => 'Search…';

	/// en: 'Project'
	String get projectLabel => 'Project';

	/// en: 'Filter by name or path…'
	String get filterHint => 'Filter by name or path…';

	/// en: 'Copied'
	String get copied => 'Copied';

	/// en: 'Copy text'
	String get copyTooltip => 'Copy text';

	late final TranslationsMemoryDeleteAllConfirmEn deleteAllConfirm = TranslationsMemoryDeleteAllConfirmEn.internal(_root);

	/// en: 'Deleted {n} memory item'
	String deletedSnackOne({required Object n}) => 'Deleted ${n} memory item';

	/// en: 'Deleted {n} memory items'
	String deletedSnackOther({required Object n}) => 'Deleted ${n} memory items';

	/// en: 'Bulk delete failed: {error}'
	String bulkDeleteFailedApi({required Object error}) => 'Bulk delete failed: ${error}';

	/// en: 'Bulk delete failed: {error}'
	String bulkDeleteFailedGeneric({required Object error}) => 'Bulk delete failed: ${error}';

	late final TranslationsMemoryDeleteOneEn deleteOne = TranslationsMemoryDeleteOneEn.internal(_root);
	late final TranslationsMemoryScopeEn scope = TranslationsMemoryScopeEn.internal(_root);
	late final TranslationsMemoryCreateEn create = TranslationsMemoryCreateEn.internal(_root);
}

// Path: about
class TranslationsAboutEn {
	TranslationsAboutEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'About'
	String get title => 'About';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	late final TranslationsAboutSectionsEn sections = TranslationsAboutSectionsEn.internal(_root);
	late final TranslationsAboutFieldsEn fields = TranslationsAboutFieldsEn.internal(_root);

	/// en: 'Copied {label}'
	String copied({required Object label}) => 'Copied ${label}';

	/// en: 'Copy'
	String get copyTooltip => 'Copy';

	late final TranslationsAboutCopyLabelsEn copyLabels = TranslationsAboutCopyLabelsEn.internal(_root);

	/// en: 'opendray mobile — multi-CLI gateway control. Source: github.com/Opendray/opendray_v2'
	String get tagline => 'opendray mobile — multi-CLI gateway control.\nSource: github.com/Opendray/opendray_v2';
}

// Path: settings
class TranslationsSettingsEn {
	TranslationsSettingsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Settings'
	String get title => 'Settings';

	late final TranslationsSettingsLanguageEn language = TranslationsSettingsLanguageEn.internal(_root);
	late final TranslationsSettingsAppearanceEn appearance = TranslationsSettingsAppearanceEn.internal(_root);
	late final TranslationsSettingsAccountEn account = TranslationsSettingsAccountEn.internal(_root);
	late final TranslationsSettingsGatewayEn gateway = TranslationsSettingsGatewayEn.internal(_root);
	late final TranslationsSettingsChangeCredentialsEn changeCredentials = TranslationsSettingsChangeCredentialsEn.internal(_root);
	late final TranslationsSettingsLogViewerEn logViewer = TranslationsSettingsLogViewerEn.internal(_root);
	late final TranslationsSettingsServerSettingsEn serverSettings = TranslationsSettingsServerSettingsEn.internal(_root);
}

// Path: more.identity
class TranslationsMoreIdentityEn {
	TranslationsMoreIdentityEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Signed in as'
	String get signedInAs => 'Signed in as';

	/// en: 'Server'
	String get server => 'Server';

	/// en: 'Token expires'
	String get tokenExpires => 'Token expires';
}

// Path: more.sections
class TranslationsMoreSectionsEn {
	TranslationsMoreSectionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Gateway'
	String get gateway => 'Gateway';

	/// en: 'Memory'
	String get memory => 'Memory';

	/// en: 'System'
	String get system => 'System';
}

// Path: more.items
class TranslationsMoreItemsEn {
	TranslationsMoreItemsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsMoreItemsIntegrationsEn integrations = TranslationsMoreItemsIntegrationsEn.internal(_root);
	late final TranslationsMoreItemsChannelsEn channels = TranslationsMoreItemsChannelsEn.internal(_root);
	late final TranslationsMoreItemsProvidersEn providers = TranslationsMoreItemsProvidersEn.internal(_root);
	late final TranslationsMoreItemsMcpEn mcp = TranslationsMoreItemsMcpEn.internal(_root);
	late final TranslationsMoreItemsSkillsEn skills = TranslationsMoreItemsSkillsEn.internal(_root);
	late final TranslationsMoreItemsGitHostsEn gitHosts = TranslationsMoreItemsGitHostsEn.internal(_root);
	late final TranslationsMoreItemsCustomTasksEn customTasks = TranslationsMoreItemsCustomTasksEn.internal(_root);
	late final TranslationsMoreItemsProjectMemoryEn projectMemory = TranslationsMoreItemsProjectMemoryEn.internal(_root);
	late final TranslationsMoreItemsCleanupInboxEn cleanupInbox = TranslationsMoreItemsCleanupInboxEn.internal(_root);
	late final TranslationsMoreItemsBackupsEn backups = TranslationsMoreItemsBackupsEn.internal(_root);
	late final TranslationsMoreItemsSettingsEn settings = TranslationsMoreItemsSettingsEn.internal(_root);
	late final TranslationsMoreItemsAboutEn about = TranslationsMoreItemsAboutEn.internal(_root);
}

// Path: sessions.filters
class TranslationsSessionsFiltersEn {
	TranslationsSessionsFiltersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'All'
	String get all => 'All';

	/// en: 'Running'
	String get running => 'Running';

	/// en: 'Idle'
	String get idle => 'Idle';

	/// en: 'Ended'
	String get ended => 'Ended';
}

// Path: sessions.card
class TranslationsSessionsCardEn {
	TranslationsSessionsCardEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '{provider} · started {when}'
	String startedRelative({required Object provider, required Object when}) => '${provider} · started ${when}';
}

// Path: sessions.empty
class TranslationsSessionsEmptyEn {
	TranslationsSessionsEmptyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No sessions yet'
	String get titleAll => 'No sessions yet';

	/// en: 'No sessions match the "{filter}" filter.'
	String titleFiltered({required Object filter}) => 'No sessions match the "${filter}" filter.';

	/// en: 'Tap the Spawn button to create one.'
	String get subtitleAll => 'Tap the Spawn button to create one.';

	/// en: 'Try a different filter or pull to refresh.'
	String get subtitleFiltered => 'Try a different filter or pull to refresh.';
}

// Path: sessions.relative
class TranslationsSessionsRelativeEn {
	TranslationsSessionsRelativeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '{n}s ago'
	String secondsAgo({required Object n}) => '${n}s ago';

	/// en: '{n}m ago'
	String minutesAgo({required Object n}) => '${n}m ago';

	/// en: '{n}h ago'
	String hoursAgo({required Object n}) => '${n}h ago';

	/// en: '{n}d ago'
	String daysAgo({required Object n}) => '${n}d ago';
}

// Path: sessions.detail
class TranslationsSessionsDetailEn {
	TranslationsSessionsDetailEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Session'
	String get fallbackTitle => 'Session';

	/// en: 'Refresh metadata'
	String get refreshMetadata => 'Refresh metadata';

	/// en: 'Inspector (Files / Git / Tasks / History / Notes)'
	String get inspector => 'Inspector (Files / Git / Tasks / History / Notes)';

	/// en: 'Project memory (goal / plan / journal / inbox)'
	String get projectMemory => 'Project memory (goal / plan / journal / inbox)';

	/// en: 'Actions'
	String get actions => 'Actions';

	/// en: 'started {when}'
	String started({required Object when}) => 'started ${when}';

	/// en: 'started {started} · ended {ended}'
	String startedEnded({required Object started, required Object ended}) => 'started ${started}  ·  ended ${ended}';

	/// en: 'id: {id}'
	String idPrefix({required Object id}) => 'id: ${id}';

	/// en: 'Failed to load session'
	String get errorTitle => 'Failed to load session';
}

// Path: sessions.terminal
class TranslationsSessionsTerminalEn {
	TranslationsSessionsTerminalEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsSessionsTerminalSnackbarEn snackbar = TranslationsSessionsTerminalSnackbarEn.internal(_root);
	late final TranslationsSessionsTerminalImageSourceEn imageSource = TranslationsSessionsTerminalImageSourceEn.internal(_root);
	late final TranslationsSessionsTerminalKeyboardEn keyboard = TranslationsSessionsTerminalKeyboardEn.internal(_root);
	late final TranslationsSessionsTerminalConnectionEn connection = TranslationsSessionsTerminalConnectionEn.internal(_root);
}

// Path: sessions.action
class TranslationsSessionsActionEn {
	TranslationsSessionsActionEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Stop'
	String get stop => 'Stop';

	/// en: 'Stopping…'
	String get stopping => 'Stopping…';

	/// en: 'Send SIGTERM, retain history'
	String get stopDescription => 'Send SIGTERM, retain history';

	/// en: 'Restart'
	String get restart => 'Restart';

	/// en: 'Restarting…'
	String get restarting => 'Restarting…';

	/// en: 'Re-spawn the CLI process'
	String get restartDescription => 'Re-spawn the CLI process';

	/// en: 'Delete'
	String get delete => 'Delete';

	/// en: 'Remove the session and its history'
	String get deleteDescription => 'Remove the session and its history';

	/// en: 'Delete this session permanently? Its ring buffer and history will be gone.'
	String get deleteConfirm => 'Delete this session permanently? Its ring buffer and history will be gone.';

	late final TranslationsSessionsActionErrorsEn errors = TranslationsSessionsActionErrorsEn.internal(_root);
}

// Path: sessions.dirPicker
class TranslationsSessionsDirPickerEn {
	TranslationsSessionsDirPickerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Parent'
	String get parent => 'Parent';

	/// en: 'New folder'
	String get newFolder => 'New folder';

	/// en: 'Use this folder'
	String get useThisFolder => 'Use this folder';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No subfolders here. Pick this folder, or create a new one.'
	String get empty => 'No subfolders here.\nPick this folder, or create a new one.';

	/// en: 'Created {path}'
	String createdSnack({required Object path}) => 'Created ${path}';

	/// en: 'mkdir failed: {error}'
	String mkdirFailedSnack({required Object error}) => 'mkdir failed: ${error}';

	late final TranslationsSessionsDirPickerDialogEn dialog = TranslationsSessionsDirPickerDialogEn.internal(_root);
}

// Path: sessions.inspector
class TranslationsSessionsInspectorEn {
	TranslationsSessionsInspectorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsSessionsInspectorShellEn shell = TranslationsSessionsInspectorShellEn.internal(_root);
	late final TranslationsSessionsInspectorSharedEn shared = TranslationsSessionsInspectorSharedEn.internal(_root);
	late final TranslationsSessionsInspectorHistoryEn history = TranslationsSessionsInspectorHistoryEn.internal(_root);
	late final TranslationsSessionsInspectorFilesEn files = TranslationsSessionsInspectorFilesEn.internal(_root);
	late final TranslationsSessionsInspectorGitEn git = TranslationsSessionsInspectorGitEn.internal(_root);
	late final TranslationsSessionsInspectorTasksEn tasks = TranslationsSessionsInspectorTasksEn.internal(_root);
	late final TranslationsSessionsInspectorNotesEn notes = TranslationsSessionsInspectorNotesEn.internal(_root);
}

// Path: sessions.spawnSheet
class TranslationsSessionsSpawnSheetEn {
	TranslationsSessionsSpawnSheetEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'New session'
	String get title => 'New session';

	/// en: 'Provider and working directory are required'
	String get errorRequired => 'Provider and working directory are required';

	/// en: 'Failed to spawn session: {error}'
	String errorGeneric({required Object error}) => 'Failed to spawn session: ${error}';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Spawn'
	String get spawn => 'Spawn';

	/// en: 'Provider'
	String get providerLabel => 'Provider';

	/// en: ' (disabled)'
	String get disabledSuffix => ' (disabled)';

	/// en: 'Working directory'
	String get cwdLabel => 'Working directory';

	/// en: '/Users/you/projects/foo'
	String get cwdHint => '/Users/you/projects/foo';

	/// en: 'Absolute path on the gateway host.'
	String get cwdHelper => 'Absolute path on the gateway host.';

	/// en: 'Browse'
	String get browse => 'Browse';

	/// en: 'Name (optional)'
	String get nameLabel => 'Name (optional)';

	/// en: 'e.g. backend-refactor'
	String get nameHint => 'e.g. backend-refactor';

	/// en: 'Extra args (optional)'
	String get argsLabel => 'Extra args (optional)';

	/// en: '--continue --verbose'
	String get argsHint => '--continue --verbose';

	/// en: 'Whitespace-separated; blank uses the provider's defaults.'
	String get argsHelper => 'Whitespace-separated; blank uses the provider\'s defaults.';

	late final TranslationsSessionsSpawnSheetBypassEn bypass = TranslationsSessionsSpawnSheetBypassEn.internal(_root);
	late final TranslationsSessionsSpawnSheetNoProvidersEn noProviders = TranslationsSessionsSpawnSheetNoProvidersEn.internal(_root);
	late final TranslationsSessionsSpawnSheetProviderLoadErrorEn providerLoadError = TranslationsSessionsSpawnSheetProviderLoadErrorEn.internal(_root);
	late final TranslationsSessionsSpawnSheetClaudeAccountEn claudeAccount = TranslationsSessionsSpawnSheetClaudeAccountEn.internal(_root);
}

// Path: mcp.errorPrefix
class TranslationsMcpErrorPrefixEn {
	TranslationsMcpErrorPrefixEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Delete failed'
	String get delete => 'Delete failed';

	/// en: 'Add failed'
	String get add => 'Add failed';

	/// en: 'Update failed'
	String get update => 'Update failed';

	/// en: 'Toggle failed'
	String get toggle => 'Toggle failed';
}

// Path: mcp.editor
class TranslationsMcpEditorEn {
	TranslationsMcpEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'my-mcp-server'
	String get nameHint => 'my-mcp-server';

	/// en: 'JSON config — name, transport: stdio, command, args…'
	String get jsonHint => 'JSON config — name, transport: stdio, command, args…';

	/// en: 'Optional one-liner'
	String get descriptionPlaceholder => 'Optional one-liner';

	/// en: 'Body must be a JSON object'
	String get validateJsonObject => 'Body must be a JSON object';

	/// en: 'Invalid JSON: {error}'
	String validateJsonInvalid({required Object error}) => 'Invalid JSON: ${error}';

	/// en: 'Edit MCP server'
	String get appBarEdit => 'Edit MCP server';

	/// en: 'New MCP server'
	String get appBarNew => 'New MCP server';

	/// en: 'Locked in edit mode — delete + recreate to change.'
	String get idLockedHint => 'Locked in edit mode — delete + recreate to change.';

	/// en: 'Server JSON'
	String get jsonLabel => 'Server JSON';

	/// en: 'Schema: transport must be stdio, http or sse. For stdio set command + args. For http/sse set url + headers. Use \$secret:KEY to reference vault secrets.'
	String get jsonSchemaHelp => 'Schema: transport must be stdio, http or sse. For stdio set command + args. For http/sse set url + headers. Use \$secret:KEY to reference vault secrets.';

	/// en: 'id (URL segment, lowercase alphanumeric / dash / underscore)'
	String get idLabel => 'id (URL segment, lowercase alphanumeric / dash / underscore)';

	/// en: 'id is required'
	String get idRequired => 'id is required';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Create'
	String get create => 'Create';
}

// Path: mcp.secret
class TranslationsMcpSecretEn {
	TranslationsMcpSecretEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Key'
	String get keyLabel => 'Key';

	/// en: 'GITHUB_TOKEN, OPENAI_KEY, …'
	String get keyHint => 'GITHUB_TOKEN, OPENAI_KEY, …';

	/// en: 'Value'
	String get valueLabel => 'Value';

	/// en: 'Key is required.'
	String get keyRequired => 'Key is required.';

	/// en: 'Key must match [A-Za-z_][A-Za-z0-9_]* — same rules as a shell env var.'
	String get keyInvalid => 'Key must match [A-Za-z_][A-Za-z0-9_]* — same rules as a shell env var.';

	/// en: 'Value is required.'
	String get valueRequired => 'Value is required.';

	/// en: 'Replace secret value'
	String get replaceTitle => 'Replace secret value';

	/// en: 'Add secret'
	String get addTitle => 'Add secret';

	/// en: 'Save'
	String get saveButton => 'Save';

	/// en: 'Add'
	String get addButton => 'Add';

	/// en: 'Shell-env-var rules: starts with a letter or _, then letters / digits / _ only.'
	String get helpRules => 'Shell-env-var rules: starts with a letter or _, then letters / digits / _ only.';

	/// en: 'Paste new value (the previous one is wiped)'
	String get replaceHint => 'Paste new value (the previous one is wiped)';

	/// en: 'Paste secret value'
	String get addHint => 'Paste secret value';

	/// en: 'Secret {key} added.'
	String addedSnack({required Object key}) => 'Secret ${key} added.';

	/// en: 'Secret {key} updated.'
	String updatedSnack({required Object key}) => 'Secret ${key} updated.';

	/// en: 'Deleted {key}.'
	String deletedSnack({required Object key}) => 'Deleted ${key}.';

	/// en: 'Removes the value from the encrypted vault. Any MCP server that references it will fail until restored.'
	String get deleteBody => 'Removes the value from the encrypted vault. Any MCP server that references it will fail until restored.';
}

// Path: mcp.popup
class TranslationsMcpPopupEn {
	TranslationsMcpPopupEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Full JSON editor — vault-backed servers only'
	String get editConfigSubtitle => 'Full JSON editor — vault-backed servers only';

	/// en: 'Read-only inspector for the server JSON'
	String get viewRawSubtitle => 'Read-only inspector for the server JSON';

	/// en: 'Delete'
	String get deleteLabel => 'Delete';
}

// Path: mcp.kv
class TranslationsMcpKvEn {
	TranslationsMcpKvEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Transport'
	String get transport => 'Transport';

	/// en: 'Description'
	String get description => 'Description';

	/// en: 'Command'
	String get command => 'Command';

	/// en: 'Args'
	String get args => 'Args';

	/// en: 'Headers'
	String get headers => 'Headers';
}

// Path: providers.errorPrefix
class TranslationsProvidersErrorPrefixEn {
	TranslationsProvidersErrorPrefixEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Toggle failed'
	String get toggle => 'Toggle failed';

	/// en: 'Rename failed'
	String get rename => 'Rename failed';

	/// en: 'Delete failed'
	String get delete => 'Delete failed';
}

// Path: providers.accounts
class TranslationsProvidersAccountsEn {
	TranslationsProvidersAccountsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Rename'
	String get rename => 'Rename';

	/// en: 'Rename {name}'
	String renameTitle({required Object name}) => 'Rename ${name}';

	/// en: 'Display name'
	String get displayNameLabel => 'Display name';

	/// en: 'Work account'
	String get displayNameHint => 'Work account';

	/// en: 'Delete account?'
	String get deleteTitle => 'Delete account?';

	/// en: 'Import failed: {error}'
	String importFailedApi({required Object error}) => 'Import failed: ${error}';

	/// en: 'Import failed: {error}'
	String importFailedGeneric({required Object error}) => 'Import failed: ${error}';
}

// Path: memoryWorkers.tasks
class TranslationsMemoryWorkersTasksEn {
	TranslationsMemoryWorkersTasksEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsMemoryWorkersTasksGatekeeperEn gatekeeper = TranslationsMemoryWorkersTasksGatekeeperEn.internal(_root);
	late final TranslationsMemoryWorkersTasksCleanerEn cleaner = TranslationsMemoryWorkersTasksCleanerEn.internal(_root);
	late final TranslationsMemoryWorkersTasksGitactivityEn gitactivity = TranslationsMemoryWorkersTasksGitactivityEn.internal(_root);
	late final TranslationsMemoryWorkersTasksTranscriptEn transcript = TranslationsMemoryWorkersTasksTranscriptEn.internal(_root);
}

// Path: backups.encryption
class TranslationsBackupsEncryptionEn {
	TranslationsBackupsEncryptionEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Check again'
	String get checkAgain => 'Check again';

	/// en: 'Generate'
	String get generate => 'Generate';

	/// en: 'Paste'
	String get paste => 'Paste';

	/// en: '256-bit random key'
	String get random256bit => '256-bit random key';

	/// en: 'Your passphrase'
	String get passphraseLabel => 'Your passphrase';

	/// en: 'At least 20 characters'
	String get passphraseHint => 'At least 20 characters';

	/// en: 'Passphrase copied to clipboard'
	String get passphraseCopied => 'Passphrase copied to clipboard';
}

// Path: githosts.errorPrefix
class TranslationsGithostsErrorPrefixEn {
	TranslationsGithostsErrorPrefixEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Toggle failed'
	String get toggle => 'Toggle failed';

	/// en: 'Delete failed'
	String get delete => 'Delete failed';
}

// Path: githosts.form
class TranslationsGithostsFormEn {
	TranslationsGithostsFormEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Kind'
	String get kindLabel => 'Kind';

	/// en: 'Host'
	String get hostLabel => 'Host';

	/// en: 'Name'
	String get nameLabel => 'Name';

	/// en: 'work-github, personal-gitlab, …'
	String get nameHint => 'work-github, personal-gitlab, …';
}

// Path: channels.configDialog
class TranslationsChannelsConfigDialogEn {
	TranslationsChannelsConfigDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '{kind} config'
	String title({required Object kind}) => '${kind} config';
}

// Path: channels.webhookDialog
class TranslationsChannelsWebhookDialogEn {
	TranslationsChannelsWebhookDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '{kind} webhook URL'
	String title({required Object kind}) => '${kind} webhook URL';

	/// en: 'Copied webhook URL.'
	String get copiedSnack => 'Copied webhook URL.';
}

// Path: channels.notifications
class TranslationsChannelsNotificationsEn {
	TranslationsChannelsNotificationsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Notification preferences'
	String get title => 'Notification preferences';

	/// en: 'Notify on'
	String get notifyOn => 'Notify on';

	/// en: 'Repeat policy'
	String get repeatPolicy => 'Repeat policy';

	/// en: 'Cooldown window'
	String get cooldownWindow => 'Cooldown window';

	/// en: 'Include terminal snippet'
	String get includeSnippet => 'Include terminal snippet';

	/// en: 'Snippet length cap'
	String get snippetLengthCap => 'Snippet length cap';

	/// en: 'All session events.'
	String get notifyOnAll => 'All session events.';

	/// en: 'No events selected — outbound notifications muted.'
	String get notifyOnEmpty => 'No events selected — outbound notifications muted.';

	/// en: 'Embeds the recent terminal tail in each notification.'
	String get snippetHelper => 'Embeds the recent terminal tail in each notification.';

	/// en: 'no cap'
	String get snippetNoCap => 'no cap';

	/// en: '{n} chars'
	String snippetChars({required Object n}) => '${n} chars';

	/// en: 'Notification preferences updated.'
	String get updatedSnack => 'Notification preferences updated.';

	late final TranslationsChannelsNotificationsModesEn modes = TranslationsChannelsNotificationsModesEn.internal(_root);
}

// Path: channels.popup
class TranslationsChannelsPopupEn {
	TranslationsChannelsPopupEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Enable'
	String get enable => 'Enable';

	/// en: 'Disable'
	String get disable => 'Disable';

	/// en: 'Mute'
	String get mute => 'Mute';

	/// en: 'Unmute'
	String get unmute => 'Unmute';

	/// en: 'Delete'
	String get deleteLabel => 'Delete';
}

// Path: channels.badges
class TranslationsChannelsBadgesEn {
	TranslationsChannelsBadgesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'running'
	String get running => 'running';

	/// en: 'starting…'
	String get starting => 'starting…';

	/// en: 'disabled'
	String get disabled => 'disabled';

	/// en: 'muted'
	String get muted => 'muted';
}

// Path: channels.snacks
class TranslationsChannelsSnacksEn {
	TranslationsChannelsSnacksEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Test message dispatched.'
	String get testDispatched => 'Test message dispatched.';

	/// en: 'Channel enabled.'
	String get channelEnabled => 'Channel enabled.';

	/// en: 'Channel disabled.'
	String get channelDisabled => 'Channel disabled.';

	/// en: 'Channel muted.'
	String get channelMuted => 'Channel muted.';

	/// en: 'Channel unmuted.'
	String get channelUnmuted => 'Channel unmuted.';

	/// en: 'Channel config updated.'
	String get configUpdated => 'Channel config updated.';

	/// en: 'Channel deleted.'
	String get channelDeleted => 'Channel deleted.';
}

// Path: channels.errorPrefix
class TranslationsChannelsErrorPrefixEn {
	TranslationsChannelsErrorPrefixEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Test failed'
	String get test => 'Test failed';

	/// en: 'Toggle failed'
	String get toggle => 'Toggle failed';

	/// en: 'Mute toggle failed'
	String get muteToggle => 'Mute toggle failed';

	/// en: 'Update failed'
	String get update => 'Update failed';

	/// en: 'Delete failed'
	String get delete => 'Delete failed';
}

// Path: channels.kinds
class TranslationsChannelsKindsEn {
	TranslationsChannelsKindsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsChannelsKindsTelegramEn telegram = TranslationsChannelsKindsTelegramEn.internal(_root);
	late final TranslationsChannelsKindsSlackEn slack = TranslationsChannelsKindsSlackEn.internal(_root);
	late final TranslationsChannelsKindsDiscordEn discord = TranslationsChannelsKindsDiscordEn.internal(_root);
	late final TranslationsChannelsKindsFeishuEn feishu = TranslationsChannelsKindsFeishuEn.internal(_root);
	late final TranslationsChannelsKindsDingtalkEn dingtalk = TranslationsChannelsKindsDingtalkEn.internal(_root);
	late final TranslationsChannelsKindsWecomEn wecom = TranslationsChannelsKindsWecomEn.internal(_root);
	late final TranslationsChannelsKindsWechatEn wechat = TranslationsChannelsKindsWechatEn.internal(_root);
}

// Path: notesPage.editor
class TranslationsNotesPageEditorEn {
	TranslationsNotesPageEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Markdown…'
	String get markdownHint => 'Markdown…';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Auto-saves as you type'
	String get autosave => 'Auto-saves as you type';
}

// Path: memory.deleteAllConfirm
class TranslationsMemoryDeleteAllConfirmEn {
	TranslationsMemoryDeleteAllConfirmEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Delete every memory in this scope?'
	String get title => 'Delete every memory in this scope?';

	/// en: 'Delete all'
	String get deleteAll => 'Delete all';
}

// Path: memory.deleteOne
class TranslationsMemoryDeleteOneEn {
	TranslationsMemoryDeleteOneEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Delete memory?'
	String get title => 'Delete memory?';

	/// en: 'This cannot be undone.'
	String get body => 'This cannot be undone.';
}

// Path: memory.scope
class TranslationsMemoryScopeEn {
	TranslationsMemoryScopeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Project'
	String get project => 'Project';

	/// en: 'Global'
	String get global => 'Global';
}

// Path: memory.create
class TranslationsMemoryCreateEn {
	TranslationsMemoryCreateEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Text'
	String get textLabel => 'Text';

	/// en: 'Scope key (project cwd)'
	String get scopeKeyLabel => 'Scope key (project cwd)';

	/// en: '/Users/you/projects/foo'
	String get scopeKeyHint => '/Users/you/projects/foo';

	/// en: 'Create'
	String get submit => 'Create';
}

// Path: about.sections
class TranslationsAboutSectionsEn {
	TranslationsAboutSectionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'App'
	String get app => 'App';

	/// en: 'Server'
	String get server => 'Server';
}

// Path: about.fields
class TranslationsAboutFieldsEn {
	TranslationsAboutFieldsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'App'
	String get app => 'App';

	/// en: 'Version'
	String get version => 'Version';

	/// en: '{version} (build {build})'
	String versionFormat({required Object version, required Object build}) => '${version} (build ${build})';

	/// en: 'Package'
	String get package => 'Package';

	/// en: 'URL'
	String get url => 'URL';

	/// en: 'Signed in as'
	String get signedInAs => 'Signed in as';

	/// en: 'Token expires'
	String get tokenExpires => 'Token expires';
}

// Path: about.copyLabels
class TranslationsAboutCopyLabelsEn {
	TranslationsAboutCopyLabelsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'version'
	String get version => 'version';

	/// en: 'server URL'
	String get serverUrl => 'server URL';
}

// Path: settings.language
class TranslationsSettingsLanguageEn {
	TranslationsSettingsLanguageEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Language'
	String get section => 'Language';

	/// en: 'System'
	String get system => 'System';

	/// en: 'Follow your phone's language setting'
	String get systemSubtitle => 'Follow your phone\'s language setting';

	/// en: 'English'
	String get english => 'English';

	/// en: '中文'
	String get chinese => '中文';
}

// Path: settings.appearance
class TranslationsSettingsAppearanceEn {
	TranslationsSettingsAppearanceEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Appearance'
	String get section => 'Appearance';

	/// en: 'System'
	String get system => 'System';

	/// en: 'Follow your phone's appearance setting'
	String get systemSubtitle => 'Follow your phone\'s appearance setting';

	/// en: 'Light'
	String get light => 'Light';

	/// en: 'Always use the light palette'
	String get lightSubtitle => 'Always use the light palette';

	/// en: 'Dark'
	String get dark => 'Dark';

	/// en: 'Always use the dark palette'
	String get darkSubtitle => 'Always use the dark palette';
}

// Path: settings.account
class TranslationsSettingsAccountEn {
	TranslationsSettingsAccountEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Account'
	String get section => 'Account';

	/// en: 'Change credentials'
	String get changeCredentials => 'Change credentials';

	/// en: 'Username and password'
	String get changeCredentialsSubtitle => 'Username and password';
}

// Path: settings.gateway
class TranslationsSettingsGatewayEn {
	TranslationsSettingsGatewayEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Gateway'
	String get section => 'Gateway';

	/// en: 'Server settings'
	String get serverSettings => 'Server settings';

	/// en: 'Listen address, logging, vault, memory, storage paths…'
	String get serverSettingsSubtitle => 'Listen address, logging, vault, memory, storage paths…';

	/// en: 'Live logs'
	String get liveLogs => 'Live logs';

	/// en: 'Tail the gateway log buffer — same source as the web admin'
	String get liveLogsSubtitle => 'Tail the gateway log buffer — same source as the web admin';
}

// Path: settings.changeCredentials
class TranslationsSettingsChangeCredentialsEn {
	TranslationsSettingsChangeCredentialsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Change credentials'
	String get title => 'Change credentials';

	/// en: 'Verify your current password, then pick new credentials. All other signed-in sessions will be revoked.'
	String get explanation => 'Verify your current password, then pick new credentials. All other signed-in sessions will be revoked.';

	/// en: 'Current password'
	String get currentPassword => 'Current password';

	/// en: 'New username'
	String get newUsername => 'New username';

	/// en: 'New password'
	String get newPassword => 'New password';

	/// en: 'Confirm new password'
	String get confirmPassword => 'Confirm new password';

	/// en: 'Required'
	String get validatorRequired => 'Required';

	/// en: 'At least 8 characters'
	String get passwordHelper => 'At least 8 characters';

	/// en: 'Must be at least 8 characters'
	String get passwordTooShort => 'Must be at least 8 characters';

	/// en: 'Doesn't match the new password'
	String get passwordMismatch => 'Doesn\'t match the new password';

	/// en: 'Credentials updated.'
	String get updatedSnack => 'Credentials updated.';

	/// en: 'Current password is wrong.'
	String get wrongCurrent => 'Current password is wrong.';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Update'
	String get update => 'Update';
}

// Path: settings.logViewer
class TranslationsSettingsLogViewerEn {
	TranslationsSettingsLogViewerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Live logs'
	String get title => 'Live logs';

	/// en: 'Reconnect'
	String get reconnect => 'Reconnect';

	/// en: 'Copy buffer'
	String get copyBuffer => 'Copy buffer';

	/// en: 'Clear local view'
	String get clearLocal => 'Clear local view';

	/// en: 'Copied buffer to clipboard'
	String get copiedSnack => 'Copied buffer to clipboard';

	/// en: 'Filter substring…'
	String get filterHint => 'Filter substring…';

	late final TranslationsSettingsLogViewerLevelsEn levels = TranslationsSettingsLogViewerLevelsEn.internal(_root);
}

// Path: settings.serverSettings
class TranslationsSettingsServerSettingsEn {
	TranslationsSettingsServerSettingsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Server settings'
	String get title => 'Server settings';

	/// en: 'Reload from server'
	String get reloadTooltip => 'Reload from server';

	/// en: 'Restart gateway'
	String get restartTooltip => 'Restart gateway';

	/// en: 'Restart opendray?'
	String get restartConfirmTitle => 'Restart opendray?';

	/// en: 'Restart'
	String get restart => 'Restart';

	/// en: 'Restart requested. Pull-to-refresh in a moment.'
	String get restartQueuedSnack => 'Restart requested. Pull-to-refresh in a moment.';

	/// en: 'Restart failed: {error}'
	String restartFailedApi({required Object error}) => 'Restart failed: ${error}';

	/// en: 'Restart failed: {error}'
	String restartFailedGeneric({required Object error}) => 'Restart failed: ${error}';

	late final TranslationsSettingsServerSettingsSectionsEn sections = TranslationsSettingsServerSettingsSectionsEn.internal(_root);
}

// Path: more.items.integrations
class TranslationsMoreItemsIntegrationsEn {
	TranslationsMoreItemsIntegrationsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Integrations'
	String get title => 'Integrations';

	/// en: 'API callers — recent activity & error rates'
	String get subtitle => 'API callers — recent activity & error rates';
}

// Path: more.items.channels
class TranslationsMoreItemsChannelsEn {
	TranslationsMoreItemsChannelsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Channels'
	String get title => 'Channels';

	/// en: 'Notification destinations'
	String get subtitle => 'Notification destinations';
}

// Path: more.items.providers
class TranslationsMoreItemsProvidersEn {
	TranslationsMoreItemsProvidersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Providers'
	String get title => 'Providers';

	/// en: 'Claude / Codex / Gemini CLI status'
	String get subtitle => 'Claude / Codex / Gemini CLI status';
}

// Path: more.items.mcp
class TranslationsMoreItemsMcpEn {
	TranslationsMoreItemsMcpEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'MCP'
	String get title => 'MCP';

	/// en: 'Model Context Protocol servers & secrets'
	String get subtitle => 'Model Context Protocol servers & secrets';
}

// Path: more.items.skills
class TranslationsMoreItemsSkillsEn {
	TranslationsMoreItemsSkillsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Skills'
	String get title => 'Skills';

	/// en: 'Agent SKILL.md library (built-in + vault)'
	String get subtitle => 'Agent SKILL.md library (built-in + vault)';
}

// Path: more.items.gitHosts
class TranslationsMoreItemsGitHostsEn {
	TranslationsMoreItemsGitHostsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Git hosts'
	String get title => 'Git hosts';

	/// en: 'PAT credentials for GitHub / GitLab / etc.'
	String get subtitle => 'PAT credentials for GitHub / GitLab / etc.';
}

// Path: more.items.customTasks
class TranslationsMoreItemsCustomTasksEn {
	TranslationsMoreItemsCustomTasksEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Custom tasks'
	String get title => 'Custom tasks';

	/// en: 'Slash commands shown in the session task picker'
	String get subtitle => 'Slash commands shown in the session task picker';
}

// Path: more.items.projectMemory
class TranslationsMoreItemsProjectMemoryEn {
	TranslationsMoreItemsProjectMemoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Project goal / plan / journal'
	String get title => 'Project goal / plan / journal';

	/// en: 'Per-cwd memory layers 2-4 + agent proposals'
	String get subtitle => 'Per-cwd memory layers 2-4 + agent proposals';
}

// Path: more.items.cleanupInbox
class TranslationsMoreItemsCleanupInboxEn {
	TranslationsMoreItemsCleanupInboxEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Cleanup inbox'
	String get title => 'Cleanup inbox';

	/// en: 'LLM-proposed deletions / merges across all projects'
	String get subtitle => 'LLM-proposed deletions / merges across all projects';
}

// Path: more.items.backups
class TranslationsMoreItemsBackupsEn {
	TranslationsMoreItemsBackupsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backups'
	String get title => 'Backups';

	/// en: 'Latest backup status & run-now'
	String get subtitle => 'Latest backup status & run-now';
}

// Path: more.items.settings
class TranslationsMoreItemsSettingsEn {
	TranslationsMoreItemsSettingsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Settings'
	String get title => 'Settings';

	/// en: 'Language, appearance, account'
	String get subtitle => 'Language, appearance, account';
}

// Path: more.items.about
class TranslationsMoreItemsAboutEn {
	TranslationsMoreItemsAboutEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'About'
	String get title => 'About';

	/// en: 'Build version & server info'
	String get subtitle => 'Build version & server info';
}

// Path: sessions.terminal.snackbar
class TranslationsSessionsTerminalSnackbarEn {
	TranslationsSessionsTerminalSnackbarEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Image picker failed: {error}'
	String imagePickerFailed({required Object error}) => 'Image picker failed: ${error}';

	/// en: 'Uploading image…'
	String get uploadingImage => 'Uploading image…';

	/// en: 'Image attached: {path}'
	String imageAttached({required Object path}) => 'Image attached: ${path}';

	/// en: 'Upload failed ({status}): {message}'
	String uploadFailed({required Object status, required Object message}) => 'Upload failed (${status}): ${message}';

	/// en: 'Upload failed: {error}'
	String uploadFailedGeneric({required Object error}) => 'Upload failed: ${error}';
}

// Path: sessions.terminal.imageSource
class TranslationsSessionsTerminalImageSourceEn {
	TranslationsSessionsTerminalImageSourceEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Photo library'
	String get photoLibrary => 'Photo library';

	/// en: 'Take a photo'
	String get takePhoto => 'Take a photo';
}

// Path: sessions.terminal.keyboard
class TranslationsSessionsTerminalKeyboardEn {
	TranslationsSessionsTerminalKeyboardEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Copy buffer'
	String get copyBuffer => 'Copy buffer';

	/// en: 'Paste'
	String get paste => 'Paste';

	/// en: 'Attach image'
	String get attachImage => 'Attach image';
}

// Path: sessions.terminal.connection
class TranslationsSessionsTerminalConnectionEn {
	TranslationsSessionsTerminalConnectionEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Connecting…'
	String get connecting => 'Connecting…';

	/// en: 'Connected'
	String get connected => 'Connected';

	/// en: 'Reconnecting…'
	String get reconnecting => 'Reconnecting…';

	/// en: 'Reconnecting ({error})…'
	String reconnectingWithError({required Object error}) => 'Reconnecting (${error})…';

	/// en: 'Disconnected'
	String get disconnected => 'Disconnected';

	/// en: 'Disconnected ({error})'
	String disconnectedWithError({required Object error}) => 'Disconnected (${error})';

	/// en: 'Session ended'
	String get ended => 'Session ended';
}

// Path: sessions.action.errors
class TranslationsSessionsActionErrorsEn {
	TranslationsSessionsActionErrorsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Stop failed: {error}'
	String stop({required Object error}) => 'Stop failed: ${error}';

	/// en: 'Restart failed: {error}'
	String start({required Object error}) => 'Restart failed: ${error}';

	/// en: 'Delete failed: {error}'
	String delete({required Object error}) => 'Delete failed: ${error}';
}

// Path: sessions.dirPicker.dialog
class TranslationsSessionsDirPickerDialogEn {
	TranslationsSessionsDirPickerDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'New folder'
	String get title => 'New folder';

	/// en: 'Folder name'
	String get hint => 'Folder name';

	/// en: 'Create'
	String get create => 'Create';
}

// Path: sessions.inspector.shell
class TranslationsSessionsInspectorShellEn {
	TranslationsSessionsInspectorShellEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Inspector'
	String get title => 'Inspector';

	/// en: 'Failed to load session: {error}'
	String loadError({required Object error}) => 'Failed to load session: ${error}';

	late final TranslationsSessionsInspectorShellTabsEn tabs = TranslationsSessionsInspectorShellTabsEn.internal(_root);
}

// Path: sessions.inspector.shared
class TranslationsSessionsInspectorSharedEn {
	TranslationsSessionsInspectorSharedEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Refresh'
	String get refresh => 'Refresh';

	/// en: 'Inserted: {text}'
	String inserted({required Object text}) => 'Inserted: ${text}';

	/// en: 'Insert failed ({status}): {message}'
	String insertFailedApi({required Object status, required Object message}) => 'Insert failed (${status}): ${message}';

	/// en: 'Insert failed: {error}'
	String insertFailedGeneric({required Object error}) => 'Insert failed: ${error}';

	/// en: 'Insert failed: {error}'
	String insertFailedShort({required Object error}) => 'Insert failed: ${error}';
}

// Path: sessions.inspector.history
class TranslationsSessionsInspectorHistoryEn {
	TranslationsSessionsInspectorHistoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Insert into terminal'
	String get insertIntoTerminal => 'Insert into terminal';

	/// en: 'Search prompts…'
	String get searchHint => 'Search prompts…';
}

// Path: sessions.inspector.files
class TranslationsSessionsInspectorFilesEn {
	TranslationsSessionsInspectorFilesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Insert as @reference'
	String get insertAtRef => 'Insert as @reference';

	/// en: 'Insert path'
	String get insertPath => 'Insert path';

	/// en: 'Pastes the absolute path verbatim'
	String get insertPathSubtitle => 'Pastes the absolute path verbatim';

	/// en: 'Read content'
	String get readContent => 'Read content';

	/// en: 'Up to 256 KiB plain text'
	String get readContentSubtitle => 'Up to 256 KiB plain text';

	/// en: 'Read failed ({status}): {message}'
	String readFailedApi({required Object status, required Object message}) => 'Read failed (${status}): ${message}';

	/// en: 'Read failed: {error}'
	String readFailedGeneric({required Object error}) => 'Read failed: ${error}';

	/// en: 'Parent'
	String get parent => 'Parent';

	/// en: 'Back to session cwd'
	String get backToCwd => 'Back to session cwd';
}

// Path: sessions.inspector.git
class TranslationsSessionsInspectorGitEn {
	TranslationsSessionsInspectorGitEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Insert as @reference'
	String get insertAtRef => 'Insert as @reference';

	/// en: 'Insert path'
	String get insertPath => 'Insert path';

	/// en: 'Show diff'
	String get showDiff => 'Show diff';

	/// en: 'Diff failed ({status}): {message}'
	String diffFailedApi({required Object status, required Object message}) => 'Diff failed (${status}): ${message}';

	/// en: 'Diff failed: {error}'
	String diffFailedGeneric({required Object error}) => 'Diff failed: ${error}';

	/// en: 'Insert hash'
	String get insertHash => 'Insert hash';

	/// en: 'Show full patch'
	String get showFullPatch => 'Show full patch';

	/// en: 'Show failed ({status}): {message}'
	String showFailedApi({required Object status, required Object message}) => 'Show failed (${status}): ${message}';

	/// en: 'Show failed: {error}'
	String showFailedGeneric({required Object error}) => 'Show failed: ${error}';

	/// en: 'Status'
	String get tabStatus => 'Status';

	/// en: 'Log'
	String get tabLog => 'Log';
}

// Path: sessions.inspector.tasks
class TranslationsSessionsInspectorTasksEn {
	TranslationsSessionsInspectorTasksEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Run command'
	String get runCommand => 'Run command';

	/// en: 'Insert command'
	String get insertCommand => 'Insert command';

	/// en: 'Pastes without return so you can edit'
	String get insertCommandSubtitle => 'Pastes without return so you can edit';
}

// Path: sessions.inspector.notes
class TranslationsSessionsInspectorNotesEn {
	TranslationsSessionsInspectorNotesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Inserted: @{path}'
	String insertedAt({required Object path}) => 'Inserted: @${path}';

	/// en: 'My notes'
	String get myNotes => 'My notes';

	/// en: 'Project docs'
	String get projectDocs => 'Project docs';

	/// en: 'Insert as @reference'
	String get insertAtRefTooltip => 'Insert as @reference';

	/// en: 'Insert @reference'
	String get insertAtRefShort => 'Insert @reference';

	/// en: '# {project} Thoughts, todos, context for the agent…'
	String draftHint({required Object project}) => '# ${project}\n\nThoughts, todos, context for the agent…';

	/// en: 'Create failed: {error}'
	String createFailed({required Object error}) => 'Create failed: ${error}';

	/// en: 'Save failed: {error}'
	String saveFailed({required Object error}) => 'Save failed: ${error}';

	/// en: 'Change project docs location'
	String get changeLocationTooltip => 'Change project docs location';

	/// en: 'filename (e.g. spec or design.md)'
	String get filenameHint => 'filename (e.g. spec or design.md)';

	/// en: 'Create'
	String get create => 'Create';

	/// en: 'Filter…'
	String get filterHint => 'Filter…';

	/// en: 'Project docs location'
	String get locationDialogTitle => 'Project docs location';
}

// Path: sessions.spawnSheet.bypass
class TranslationsSessionsSpawnSheetBypassEn {
	TranslationsSessionsSpawnSheetBypassEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Bypass permissions'
	String get labelClaude => 'Bypass permissions';

	/// en: 'Auto-approve (never ask)'
	String get labelCodex => 'Auto-approve (never ask)';

	/// en: 'YOLO mode'
	String get labelGemini => 'YOLO mode';

	/// en: 'This session will run with elevated autonomy.'
	String get subtitleOn => 'This session will run with elevated autonomy.';

	/// en: 'Off — confirmations and prompts behave normally.'
	String get subtitleOff => 'Off — confirmations and prompts behave normally.';
}

// Path: sessions.spawnSheet.noProviders
class TranslationsSessionsSpawnSheetNoProvidersEn {
	TranslationsSessionsSpawnSheetNoProvidersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No providers configured'
	String get title => 'No providers configured';

	/// en: 'The gateway has no CLI providers enabled. Configure one under Providers (web admin) or [providers] in config.toml, then tap Reload.'
	String get message => 'The gateway has no CLI providers enabled. Configure one under Providers (web admin) or [providers] in config.toml, then tap Reload.';

	/// en: 'Reload'
	String get reload => 'Reload';
}

// Path: sessions.spawnSheet.providerLoadError
class TranslationsSessionsSpawnSheetProviderLoadErrorEn {
	TranslationsSessionsSpawnSheetProviderLoadErrorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Could not load providers'
	String get title => 'Could not load providers';

	/// en: 'Network error'
	String get networkError => 'Network error';

	/// en: 'Server {code}'
	String serverPrefix({required Object code}) => 'Server ${code}';

	/// en: '{prefix}: {message}'
	String format({required Object prefix, required Object message}) => '${prefix}: ${message}';
}

// Path: sessions.spawnSheet.claudeAccount
class TranslationsSessionsSpawnSheetClaudeAccountEn {
	TranslationsSessionsSpawnSheetClaudeAccountEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Claude account'
	String get label => 'Claude account';

	/// en: 'Multiple accounts configured — pick one for this session.'
	String get helperMulti => 'Multiple accounts configured — pick one for this session.';

	/// en: 'Pick a configured account or use the default (env / system).'
	String get helperSingle => 'Pick a configured account or use the default (env / system).';

	/// en: 'Default (env / system)'
	String get kDefault => 'Default (env / system)';

	/// en: ' (disabled)'
	String get disabledSuffix => ' (disabled)';

	/// en: ' (no token)'
	String get noTokenSuffix => ' (no token)';

	/// en: 'No Claude accounts configured — the gateway will use the system ANTHROPIC_API_KEY. Add accounts under Settings → Accounts on the web admin.'
	String get noneHint => 'No Claude accounts configured — the gateway will use the system ANTHROPIC_API_KEY. Add accounts under Settings → Accounts on the web admin.';

	/// en: 'Could not load Claude accounts ({error}). The session will spawn with the gateway default.'
	String errorHint({required Object error}) => 'Could not load Claude accounts (${error}). The session will spawn with the gateway default.';
}

// Path: memoryWorkers.tasks.gatekeeper
class TranslationsMemoryWorkersTasksGatekeeperEn {
	TranslationsMemoryWorkersTasksGatekeeperEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Gatekeeper'
	String get label => 'Gatekeeper';

	/// en: 'Pre-write filter on every memory_store. High frequency (<500ms target) — summarizer-only.'
	String get description => 'Pre-write filter on every memory_store. High frequency (<500ms target) — summarizer-only.';
}

// Path: memoryWorkers.tasks.cleaner
class TranslationsMemoryWorkersTasksCleanerEn {
	TranslationsMemoryWorkersTasksCleanerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Cleaner librarian'
	String get label => 'Cleaner librarian';

	/// en: 'Periodic LLM librarian. Judges aged memories as keep / stale / duplicate.'
	String get description => 'Periodic LLM librarian. Judges aged memories as keep / stale / duplicate.';
}

// Path: memoryWorkers.tasks.gitactivity
class TranslationsMemoryWorkersTasksGitactivityEn {
	TranslationsMemoryWorkersTasksGitactivityEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Git activity summariser'
	String get label => 'Git activity summariser';

	/// en: 'git log → 2-3 paragraph narrative every 24h. Naturally fits an agent worker.'
	String get description => 'git log → 2-3 paragraph narrative every 24h. Naturally fits an agent worker.';
}

// Path: memoryWorkers.tasks.transcript
class TranslationsMemoryWorkersTasksTranscriptEn {
	TranslationsMemoryWorkersTasksTranscriptEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Session transcript summariser'
	String get label => 'Session transcript summariser';

	/// en: 'Session-end 'what did the agent do' summary. Naturally fits an agent worker.'
	String get description => 'Session-end \'what did the agent do\' summary. Naturally fits an agent worker.';
}

// Path: channels.notifications.modes
class TranslationsChannelsNotificationsModesEn {
	TranslationsChannelsNotificationsModesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Once per session'
	String get onceLabel => 'Once per session';

	/// en: 'Fire once when idle, stay silent until reply or end.'
	String get onceDescription => 'Fire once when idle, stay silent until reply or end.';

	/// en: 'Time-window cooldown'
	String get cooldownLabel => 'Time-window cooldown';

	/// en: 'Suppress repeats within the chosen window.'
	String get cooldownDescription => 'Suppress repeats within the chosen window.';

	/// en: 'Every event (noisy)'
	String get everyLabel => 'Every event (noisy)';

	/// en: 'No suppression — only for low-frequency channels.'
	String get everyDescription => 'No suppression — only for low-frequency channels.';
}

// Path: channels.kinds.telegram
class TranslationsChannelsKindsTelegramEn {
	TranslationsChannelsKindsTelegramEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Bot via @BotFather. opendray long-polls getUpdates and sends via REST. Buttons + reply_to_message work natively.'
	String get description => 'Bot via @BotFather. opendray long-polls getUpdates and sends via REST. Buttons + reply_to_message work natively.';

	/// en: 'Bot token'
	String get botTokenLabel => 'Bot token';

	/// en: 'From @BotFather. Stored in channel config; admin-only API.'
	String get botTokenHint => 'From @BotFather. Stored in channel config; admin-only API.';

	/// en: 'Default chat ID'
	String get chatIdLabel => 'Default chat ID';

	/// en: '42 (optional — used when no ReplyCtx)'
	String get chatIdPlaceholder => '42 (optional — used when no ReplyCtx)';
}

// Path: channels.kinds.slack
class TranslationsChannelsKindsSlackEn {
	TranslationsChannelsKindsSlackEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Socket Mode — no public webhook needed. Requires a bot OAuth token (xoxb-) and an app-level token (xapp-) with connections:write.'
	String get description => 'Socket Mode — no public webhook needed. Requires a bot OAuth token (xoxb-) and an app-level token (xapp-) with connections:write.';

	/// en: 'Bot token (xoxb-…)'
	String get botTokenLabel => 'Bot token (xoxb-…)';

	/// en: 'OAuth & Permissions → Bot User OAuth Token. Needs chat:write.'
	String get botTokenHint => 'OAuth & Permissions → Bot User OAuth Token. Needs chat:write.';

	/// en: 'App-level token (xapp-…)'
	String get appTokenLabel => 'App-level token (xapp-…)';

	/// en: 'Settings → Basic Information → App-Level Tokens. Scope: connections:write.'
	String get appTokenHint => 'Settings → Basic Information → App-Level Tokens. Scope: connections:write.';

	/// en: 'Default channel ID'
	String get channelIdLabel => 'Default channel ID';

	/// en: 'C0123ABC456 (optional)'
	String get channelIdPlaceholder => 'C0123ABC456 (optional)';
}

// Path: channels.kinds.discord
class TranslationsChannelsKindsDiscordEn {
	TranslationsChannelsKindsDiscordEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Bot via Discord Developer Portal with MESSAGE CONTENT INTENT enabled. Connects to Gateway WS — no public URL required.'
	String get description => 'Bot via Discord Developer Portal with MESSAGE CONTENT INTENT enabled. Connects to Gateway WS — no public URL required.';

	/// en: 'Bot token'
	String get botTokenLabel => 'Bot token';

	/// en: 'Bot token from Discord Developer Portal'
	String get botTokenPlaceholder => 'Bot token from Discord Developer Portal';

	/// en: 'Application → Bot → Reset Token. Invite bot with send_messages + embed_links.'
	String get botTokenHint => 'Application → Bot → Reset Token. Invite bot with send_messages + embed_links.';

	/// en: 'Default channel ID'
	String get channelIdLabel => 'Default channel ID';

	/// en: '123456789012345678 (optional)'
	String get channelIdPlaceholder => '123456789012345678 (optional)';
}

// Path: channels.kinds.feishu
class TranslationsChannelsKindsFeishuEn {
	TranslationsChannelsKindsFeishuEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'App-level credentials. Uses event subscription webhook for inbound. Public webhook URL is generated below — paste it into the Feishu dev console.'
	String get description => 'App-level credentials. Uses event subscription webhook for inbound. Public webhook URL is generated below — paste it into the Feishu dev console.';

	/// en: 'Open the webhook URL from the channel card and paste it into Feishu Open Platform → Event Subscriptions → Request URL.'
	String get afterCreateHint => 'Open the webhook URL from the channel card and paste it into Feishu Open Platform → Event Subscriptions → Request URL.';

	/// en: 'App ID'
	String get appIdLabel => 'App ID';

	/// en: 'App secret'
	String get appSecretLabel => 'App secret';

	/// en: 'Application credential secret'
	String get appSecretPlaceholder => 'Application credential secret';

	/// en: 'Verification token'
	String get verificationTokenLabel => 'Verification token';

	/// en: 'From Event Subscriptions → Verification Token. When set, opendray rejects webhooks with a different token.'
	String get verificationTokenHint => 'From Event Subscriptions → Verification Token. When set, opendray rejects webhooks with a different token.';

	/// en: 'Default chat ID (oc_…)'
	String get chatIdLabel => 'Default chat ID (oc_…)';

	/// en: 'oc_xxxxxxxxxx (optional)'
	String get chatIdPlaceholder => 'oc_xxxxxxxxxx (optional)';
}

// Path: channels.kinds.dingtalk
class TranslationsChannelsKindsDingtalkEn {
	TranslationsChannelsKindsDingtalkEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Custom group robot. Outbound only. Group chat → Robots → Add → Sign mode → copy webhook + secret.'
	String get description => 'Custom group robot. Outbound only. Group chat → Robots → Add → Sign mode → copy webhook + secret.';

	/// en: 'Webhook URL'
	String get webhookUrlLabel => 'Webhook URL';

	/// en: 'Sign secret'
	String get secretLabel => 'Sign secret';

	/// en: 'When the robot is set to "Sign" security mode, copy the secret here. opendray adds the timestamp + sign params automatically.'
	String get secretHint => 'When the robot is set to "Sign" security mode, copy the secret here. opendray adds the timestamp + sign params automatically.';
}

// Path: channels.kinds.wecom
class TranslationsChannelsKindsWecomEn {
	TranslationsChannelsKindsWecomEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Group robot webhook. Outbound only (text + markdown). Group settings → Group robots → Add → copy webhook URL.'
	String get description => 'Group robot webhook. Outbound only (text + markdown). Group settings → Group robots → Add → copy webhook URL.';

	/// en: 'Webhook key'
	String get webhookKeyLabel => 'Webhook key';

	/// en: 'The "key=" query value'
	String get webhookKeyPlaceholder => 'The "key=" query value';

	/// en: 'Or paste the whole webhook URL into the field below — either is enough.'
	String get webhookKeyHint => 'Or paste the whole webhook URL into the field below — either is enough.';

	/// en: 'Or full webhook URL'
	String get webhookUrlLabel => 'Or full webhook URL';
}

// Path: channels.kinds.wechat
class TranslationsChannelsKindsWechatEn {
	TranslationsChannelsKindsWechatEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Push to personal WeChat via WxPusher. Outbound-only — push services do not relay user replies. Each recipient subscribes once via QR code.'
	String get description => 'Push to personal WeChat via WxPusher. Outbound-only — push services do not relay user replies. Each recipient subscribes once via QR code.';

	/// en: 'App token (AT_…)'
	String get appTokenLabel => 'App token (AT_…)';

	/// en: 'WxPusher → 应用管理 → 创建应用 → 复制 App Token.'
	String get appTokenHint => 'WxPusher → 应用管理 → 创建应用 → 复制 App Token.';

	/// en: 'Recipient UIDs (one per line)'
	String get uidsLabel => 'Recipient UIDs (one per line)';

	/// en: 'Either UIDs or topic IDs is required.'
	String get uidsHint => 'Either UIDs or topic IDs is required.';

	/// en: 'Topic IDs (one per line)'
	String get topicIdsLabel => 'Topic IDs (one per line)';

	/// en: 'Tap-through URL'
	String get urlLabel => 'Tap-through URL';

	/// en: 'When set, tapping the WeChat notification opens this page.'
	String get urlHint => 'When set, tapping the WeChat notification opens this page.';
}

// Path: settings.logViewer.levels
class TranslationsSettingsLogViewerLevelsEn {
	TranslationsSettingsLogViewerLevelsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'All'
	String get all => 'All';

	/// en: 'Debug'
	String get debug => 'Debug';

	/// en: 'Info'
	String get info => 'Info';

	/// en: 'Warn'
	String get warn => 'Warn';

	/// en: 'Error'
	String get error => 'Error';
}

// Path: settings.serverSettings.sections
class TranslationsSettingsServerSettingsSectionsEn {
	TranslationsSettingsServerSettingsSectionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'General'
	String get general => 'General';

	/// en: 'Logging'
	String get logging => 'Logging';

	/// en: 'Sessions'
	String get sessions => 'Sessions';

	/// en: 'Vault'
	String get vault => 'Vault';

	/// en: 'MCP registry'
	String get mcpRegistry => 'MCP registry';

	/// en: 'Memory'
	String get memory => 'Memory';

	/// en: 'Backup'
	String get backup => 'Backup';

	/// en: 'Storage · Claude'
	String get storageClaude => 'Storage · Claude';

	/// en: 'Storage · Codex'
	String get storageCodex => 'Storage · Codex';

	/// en: 'Storage · Gemini'
	String get storageGemini => 'Storage · Gemini';
}

// Path: sessions.inspector.shell.tabs
class TranslationsSessionsInspectorShellTabsEn {
	TranslationsSessionsInspectorShellTabsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Files'
	String get files => 'Files';

	/// en: 'Git'
	String get git => 'Git';

	/// en: 'Tasks'
	String get tasks => 'Tasks';

	/// en: 'History'
	String get history => 'History';

	/// en: 'Notes'
	String get notes => 'Notes';
}

/// The flat map containing all translations for locale <en>.
/// Only for edge cases! For simple maps, use the map function of this library.
///
/// The Dart AOT compiler has issues with very large switch statements,
/// so the map is split into smaller functions (512 entries each).
extension on Translations {
	dynamic _flatMapFunction(String path) {
		return switch (path) {
			'common.ok' => 'OK',
			'common.cancel' => 'Cancel',
			'common.save' => 'Save',
			'common.delete' => 'Delete',
			'common.edit' => 'Edit',
			'common.back' => 'Back',
			'common.done' => 'Done',
			'common.close' => 'Close',
			'common.retry' => 'Retry',
			'common.copy' => 'Copy',
			'common.enabled' => 'Enabled',
			'common.refresh' => 'Refresh',
			'auth.signInTitle' => 'Sign in',
			'auth.changeServer' => 'Change',
			'auth.username' => 'Username',
			'auth.password' => 'Password',
			'auth.signIn' => 'Sign in',
			'auth.errorRequired' => 'Username and password are required',
			'auth.errorGeneric' => ({required Object error}) => 'Login failed: ${error}',
			'nav.sessions' => 'Sessions',
			'nav.memory' => 'Memory',
			'nav.notes' => 'Notes',
			'nav.more' => 'More',
			'more.title' => 'More',
			'more.identity.signedInAs' => 'Signed in as',
			'more.identity.server' => 'Server',
			'more.identity.tokenExpires' => 'Token expires',
			'more.sections.gateway' => 'Gateway',
			'more.sections.memory' => 'Memory',
			'more.sections.system' => 'System',
			'more.items.integrations.title' => 'Integrations',
			'more.items.integrations.subtitle' => 'API callers — recent activity & error rates',
			'more.items.channels.title' => 'Channels',
			'more.items.channels.subtitle' => 'Notification destinations',
			'more.items.providers.title' => 'Providers',
			'more.items.providers.subtitle' => 'Claude / Codex / Gemini CLI status',
			'more.items.mcp.title' => 'MCP',
			'more.items.mcp.subtitle' => 'Model Context Protocol servers & secrets',
			'more.items.skills.title' => 'Skills',
			'more.items.skills.subtitle' => 'Agent SKILL.md library (built-in + vault)',
			'more.items.gitHosts.title' => 'Git hosts',
			'more.items.gitHosts.subtitle' => 'PAT credentials for GitHub / GitLab / etc.',
			'more.items.customTasks.title' => 'Custom tasks',
			'more.items.customTasks.subtitle' => 'Slash commands shown in the session task picker',
			'more.items.projectMemory.title' => 'Project goal / plan / journal',
			'more.items.projectMemory.subtitle' => 'Per-cwd memory layers 2-4 + agent proposals',
			'more.items.cleanupInbox.title' => 'Cleanup inbox',
			'more.items.cleanupInbox.subtitle' => 'LLM-proposed deletions / merges across all projects',
			'more.items.backups.title' => 'Backups',
			'more.items.backups.subtitle' => 'Latest backup status & run-now',
			'more.items.settings.title' => 'Settings',
			'more.items.settings.subtitle' => 'Language, appearance, account',
			'more.items.about.title' => 'About',
			'more.items.about.subtitle' => 'Build version & server info',
			'more.signOut' => 'Sign out',
			'sessions.title' => 'Sessions',
			'sessions.refresh' => 'Refresh',
			'sessions.actions' => 'Actions',
			'sessions.spawn' => 'Spawn',
			'sessions.filters.all' => 'All',
			'sessions.filters.running' => 'Running',
			'sessions.filters.idle' => 'Idle',
			'sessions.filters.ended' => 'Ended',
			'sessions.card.startedRelative' => ({required Object provider, required Object when}) => '${provider} · started ${when}',
			'sessions.empty.titleAll' => 'No sessions yet',
			'sessions.empty.titleFiltered' => ({required Object filter}) => 'No sessions match the "${filter}" filter.',
			'sessions.empty.subtitleAll' => 'Tap the Spawn button to create one.',
			'sessions.empty.subtitleFiltered' => 'Try a different filter or pull to refresh.',
			'sessions.errorTitle' => 'Failed to load sessions',
			'sessions.relative.secondsAgo' => ({required Object n}) => '${n}s ago',
			'sessions.relative.minutesAgo' => ({required Object n}) => '${n}m ago',
			'sessions.relative.hoursAgo' => ({required Object n}) => '${n}h ago',
			'sessions.relative.daysAgo' => ({required Object n}) => '${n}d ago',
			'sessions.detail.fallbackTitle' => 'Session',
			'sessions.detail.refreshMetadata' => 'Refresh metadata',
			'sessions.detail.inspector' => 'Inspector (Files / Git / Tasks / History / Notes)',
			'sessions.detail.projectMemory' => 'Project memory (goal / plan / journal / inbox)',
			'sessions.detail.actions' => 'Actions',
			'sessions.detail.started' => ({required Object when}) => 'started ${when}',
			'sessions.detail.startedEnded' => ({required Object started, required Object ended}) => 'started ${started}  ·  ended ${ended}',
			'sessions.detail.idPrefix' => ({required Object id}) => 'id: ${id}',
			'sessions.detail.errorTitle' => 'Failed to load session',
			'sessions.terminal.snackbar.imagePickerFailed' => ({required Object error}) => 'Image picker failed: ${error}',
			'sessions.terminal.snackbar.uploadingImage' => 'Uploading image…',
			'sessions.terminal.snackbar.imageAttached' => ({required Object path}) => 'Image attached: ${path}',
			'sessions.terminal.snackbar.uploadFailed' => ({required Object status, required Object message}) => 'Upload failed (${status}): ${message}',
			'sessions.terminal.snackbar.uploadFailedGeneric' => ({required Object error}) => 'Upload failed: ${error}',
			'sessions.terminal.imageSource.photoLibrary' => 'Photo library',
			'sessions.terminal.imageSource.takePhoto' => 'Take a photo',
			'sessions.terminal.keyboard.copyBuffer' => 'Copy buffer',
			'sessions.terminal.keyboard.paste' => 'Paste',
			'sessions.terminal.keyboard.attachImage' => 'Attach image',
			'sessions.terminal.connection.connecting' => 'Connecting…',
			'sessions.terminal.connection.connected' => 'Connected',
			'sessions.terminal.connection.reconnecting' => 'Reconnecting…',
			'sessions.terminal.connection.reconnectingWithError' => ({required Object error}) => 'Reconnecting (${error})…',
			'sessions.terminal.connection.disconnected' => 'Disconnected',
			'sessions.terminal.connection.disconnectedWithError' => ({required Object error}) => 'Disconnected (${error})',
			'sessions.terminal.connection.ended' => 'Session ended',
			'sessions.action.stop' => 'Stop',
			'sessions.action.stopping' => 'Stopping…',
			'sessions.action.stopDescription' => 'Send SIGTERM, retain history',
			'sessions.action.restart' => 'Restart',
			'sessions.action.restarting' => 'Restarting…',
			'sessions.action.restartDescription' => 'Re-spawn the CLI process',
			'sessions.action.delete' => 'Delete',
			'sessions.action.deleteDescription' => 'Remove the session and its history',
			'sessions.action.deleteConfirm' => 'Delete this session permanently? Its ring buffer and history will be gone.',
			'sessions.action.errors.stop' => ({required Object error}) => 'Stop failed: ${error}',
			'sessions.action.errors.start' => ({required Object error}) => 'Restart failed: ${error}',
			'sessions.action.errors.delete' => ({required Object error}) => 'Delete failed: ${error}',
			'sessions.dirPicker.parent' => 'Parent',
			'sessions.dirPicker.newFolder' => 'New folder',
			'sessions.dirPicker.useThisFolder' => 'Use this folder',
			'sessions.dirPicker.loading' => 'Loading…',
			'sessions.dirPicker.empty' => 'No subfolders here.\nPick this folder, or create a new one.',
			'sessions.dirPicker.createdSnack' => ({required Object path}) => 'Created ${path}',
			'sessions.dirPicker.mkdirFailedSnack' => ({required Object error}) => 'mkdir failed: ${error}',
			'sessions.dirPicker.dialog.title' => 'New folder',
			'sessions.dirPicker.dialog.hint' => 'Folder name',
			'sessions.dirPicker.dialog.create' => 'Create',
			'sessions.inspector.shell.title' => 'Inspector',
			'sessions.inspector.shell.loadError' => ({required Object error}) => 'Failed to load session: ${error}',
			'sessions.inspector.shell.tabs.files' => 'Files',
			'sessions.inspector.shell.tabs.git' => 'Git',
			'sessions.inspector.shell.tabs.tasks' => 'Tasks',
			'sessions.inspector.shell.tabs.history' => 'History',
			'sessions.inspector.shell.tabs.notes' => 'Notes',
			'sessions.inspector.shared.refresh' => 'Refresh',
			'sessions.inspector.shared.inserted' => ({required Object text}) => 'Inserted: ${text}',
			'sessions.inspector.shared.insertFailedApi' => ({required Object status, required Object message}) => 'Insert failed (${status}): ${message}',
			'sessions.inspector.shared.insertFailedGeneric' => ({required Object error}) => 'Insert failed: ${error}',
			'sessions.inspector.shared.insertFailedShort' => ({required Object error}) => 'Insert failed: ${error}',
			'sessions.inspector.history.insertIntoTerminal' => 'Insert into terminal',
			'sessions.inspector.history.searchHint' => 'Search prompts…',
			'sessions.inspector.files.insertAtRef' => 'Insert as @reference',
			'sessions.inspector.files.insertPath' => 'Insert path',
			'sessions.inspector.files.insertPathSubtitle' => 'Pastes the absolute path verbatim',
			'sessions.inspector.files.readContent' => 'Read content',
			'sessions.inspector.files.readContentSubtitle' => 'Up to 256 KiB plain text',
			'sessions.inspector.files.readFailedApi' => ({required Object status, required Object message}) => 'Read failed (${status}): ${message}',
			'sessions.inspector.files.readFailedGeneric' => ({required Object error}) => 'Read failed: ${error}',
			'sessions.inspector.files.parent' => 'Parent',
			'sessions.inspector.files.backToCwd' => 'Back to session cwd',
			'sessions.inspector.git.insertAtRef' => 'Insert as @reference',
			'sessions.inspector.git.insertPath' => 'Insert path',
			'sessions.inspector.git.showDiff' => 'Show diff',
			'sessions.inspector.git.diffFailedApi' => ({required Object status, required Object message}) => 'Diff failed (${status}): ${message}',
			'sessions.inspector.git.diffFailedGeneric' => ({required Object error}) => 'Diff failed: ${error}',
			'sessions.inspector.git.insertHash' => 'Insert hash',
			'sessions.inspector.git.showFullPatch' => 'Show full patch',
			'sessions.inspector.git.showFailedApi' => ({required Object status, required Object message}) => 'Show failed (${status}): ${message}',
			'sessions.inspector.git.showFailedGeneric' => ({required Object error}) => 'Show failed: ${error}',
			'sessions.inspector.git.tabStatus' => 'Status',
			'sessions.inspector.git.tabLog' => 'Log',
			'sessions.inspector.tasks.runCommand' => 'Run command',
			'sessions.inspector.tasks.insertCommand' => 'Insert command',
			'sessions.inspector.tasks.insertCommandSubtitle' => 'Pastes without return so you can edit',
			'sessions.inspector.notes.insertedAt' => ({required Object path}) => 'Inserted: @${path}',
			'sessions.inspector.notes.myNotes' => 'My notes',
			'sessions.inspector.notes.projectDocs' => 'Project docs',
			'sessions.inspector.notes.insertAtRefTooltip' => 'Insert as @reference',
			'sessions.inspector.notes.insertAtRefShort' => 'Insert @reference',
			'sessions.inspector.notes.draftHint' => ({required Object project}) => '# ${project}\n\nThoughts, todos, context for the agent…',
			'sessions.inspector.notes.createFailed' => ({required Object error}) => 'Create failed: ${error}',
			'sessions.inspector.notes.saveFailed' => ({required Object error}) => 'Save failed: ${error}',
			'sessions.inspector.notes.changeLocationTooltip' => 'Change project docs location',
			'sessions.inspector.notes.filenameHint' => 'filename (e.g. spec or design.md)',
			'sessions.inspector.notes.create' => 'Create',
			'sessions.inspector.notes.filterHint' => 'Filter…',
			'sessions.inspector.notes.locationDialogTitle' => 'Project docs location',
			'sessions.spawnSheet.title' => 'New session',
			'sessions.spawnSheet.errorRequired' => 'Provider and working directory are required',
			'sessions.spawnSheet.errorGeneric' => ({required Object error}) => 'Failed to spawn session: ${error}',
			'sessions.spawnSheet.cancel' => 'Cancel',
			'sessions.spawnSheet.spawn' => 'Spawn',
			'sessions.spawnSheet.providerLabel' => 'Provider',
			'sessions.spawnSheet.disabledSuffix' => ' (disabled)',
			'sessions.spawnSheet.cwdLabel' => 'Working directory',
			'sessions.spawnSheet.cwdHint' => '/Users/you/projects/foo',
			'sessions.spawnSheet.cwdHelper' => 'Absolute path on the gateway host.',
			'sessions.spawnSheet.browse' => 'Browse',
			'sessions.spawnSheet.nameLabel' => 'Name (optional)',
			'sessions.spawnSheet.nameHint' => 'e.g. backend-refactor',
			'sessions.spawnSheet.argsLabel' => 'Extra args (optional)',
			'sessions.spawnSheet.argsHint' => '--continue --verbose',
			'sessions.spawnSheet.argsHelper' => 'Whitespace-separated; blank uses the provider\'s defaults.',
			'sessions.spawnSheet.bypass.labelClaude' => 'Bypass permissions',
			'sessions.spawnSheet.bypass.labelCodex' => 'Auto-approve (never ask)',
			'sessions.spawnSheet.bypass.labelGemini' => 'YOLO mode',
			'sessions.spawnSheet.bypass.subtitleOn' => 'This session will run with elevated autonomy.',
			'sessions.spawnSheet.bypass.subtitleOff' => 'Off — confirmations and prompts behave normally.',
			'sessions.spawnSheet.noProviders.title' => 'No providers configured',
			'sessions.spawnSheet.noProviders.message' => 'The gateway has no CLI providers enabled. Configure one under Providers (web admin) or [providers] in config.toml, then tap Reload.',
			'sessions.spawnSheet.noProviders.reload' => 'Reload',
			'sessions.spawnSheet.providerLoadError.title' => 'Could not load providers',
			'sessions.spawnSheet.providerLoadError.networkError' => 'Network error',
			'sessions.spawnSheet.providerLoadError.serverPrefix' => ({required Object code}) => 'Server ${code}',
			'sessions.spawnSheet.providerLoadError.format' => ({required Object prefix, required Object message}) => '${prefix}: ${message}',
			'sessions.spawnSheet.claudeAccount.label' => 'Claude account',
			'sessions.spawnSheet.claudeAccount.helperMulti' => 'Multiple accounts configured — pick one for this session.',
			'sessions.spawnSheet.claudeAccount.helperSingle' => 'Pick a configured account or use the default (env / system).',
			'sessions.spawnSheet.claudeAccount.kDefault' => 'Default (env / system)',
			'sessions.spawnSheet.claudeAccount.disabledSuffix' => ' (disabled)',
			'sessions.spawnSheet.claudeAccount.noTokenSuffix' => ' (no token)',
			'sessions.spawnSheet.claudeAccount.noneHint' => 'No Claude accounts configured — the gateway will use the system ANTHROPIC_API_KEY. Add accounts under Settings → Accounts on the web admin.',
			'sessions.spawnSheet.claudeAccount.errorHint' => ({required Object error}) => 'Could not load Claude accounts (${error}). The session will spawn with the gateway default.',
			'mcp.title' => 'MCP',
			'mcp.newServer' => 'New server',
			'mcp.addSecret' => 'Add secret',
			'mcp.editConfig' => 'Edit config',
			'mcp.viewRawConfig' => 'View raw config',
			'mcp.copyId' => 'Copy id',
			'mcp.copiedSnack' => ({required Object id}) => 'Copied ${id}',
			'mcp.deleteServerTitle' => 'Delete MCP server?',
			'mcp.deleteSecretTitle' => 'Delete secret?',
			'mcp.errorPrefix.delete' => 'Delete failed',
			'mcp.errorPrefix.add' => 'Add failed',
			'mcp.errorPrefix.update' => 'Update failed',
			'mcp.errorPrefix.toggle' => 'Toggle failed',
			'mcp.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'mcp.editor.nameHint' => 'my-mcp-server',
			'mcp.editor.jsonHint' => 'JSON config — name, transport: stdio, command, args…',
			'mcp.editor.descriptionPlaceholder' => 'Optional one-liner',
			'mcp.editor.validateJsonObject' => 'Body must be a JSON object',
			'mcp.editor.validateJsonInvalid' => ({required Object error}) => 'Invalid JSON: ${error}',
			'mcp.editor.appBarEdit' => 'Edit MCP server',
			'mcp.editor.appBarNew' => 'New MCP server',
			'mcp.editor.idLockedHint' => 'Locked in edit mode — delete + recreate to change.',
			'mcp.editor.jsonLabel' => 'Server JSON',
			'mcp.editor.jsonSchemaHelp' => 'Schema: transport must be stdio, http or sse. For stdio set command + args. For http/sse set url + headers. Use \$secret:KEY to reference vault secrets.',
			'mcp.editor.idLabel' => 'id (URL segment, lowercase alphanumeric / dash / underscore)',
			'mcp.editor.idRequired' => 'id is required',
			'mcp.editor.saving' => 'Saving…',
			'mcp.editor.save' => 'Save',
			'mcp.editor.create' => 'Create',
			'mcp.secret.keyLabel' => 'Key',
			'mcp.secret.keyHint' => 'GITHUB_TOKEN, OPENAI_KEY, …',
			'mcp.secret.valueLabel' => 'Value',
			'mcp.secret.keyRequired' => 'Key is required.',
			'mcp.secret.keyInvalid' => 'Key must match [A-Za-z_][A-Za-z0-9_]* — same rules as a shell env var.',
			'mcp.secret.valueRequired' => 'Value is required.',
			'mcp.secret.replaceTitle' => 'Replace secret value',
			'mcp.secret.addTitle' => 'Add secret',
			'mcp.secret.saveButton' => 'Save',
			'mcp.secret.addButton' => 'Add',
			'mcp.secret.helpRules' => 'Shell-env-var rules: starts with a letter or _, then letters / digits / _ only.',
			'mcp.secret.replaceHint' => 'Paste new value (the previous one is wiped)',
			'mcp.secret.addHint' => 'Paste secret value',
			'mcp.secret.addedSnack' => ({required Object key}) => 'Secret ${key} added.',
			'mcp.secret.updatedSnack' => ({required Object key}) => 'Secret ${key} updated.',
			'mcp.secret.deletedSnack' => ({required Object key}) => 'Deleted ${key}.',
			'mcp.secret.deleteBody' => 'Removes the value from the encrypted vault. Any MCP server that references it will fail until restored.',
			'mcp.popup.editConfigSubtitle' => 'Full JSON editor — vault-backed servers only',
			'mcp.popup.viewRawSubtitle' => 'Read-only inspector for the server JSON',
			'mcp.popup.deleteLabel' => 'Delete',
			'mcp.kv.transport' => 'Transport',
			'mcp.kv.description' => 'Description',
			'mcp.kv.command' => 'Command',
			'mcp.kv.args' => 'Args',
			'mcp.kv.headers' => 'Headers',
			'mcp.deleteServerBody' => ({required Object id}) => 'Removes the vault directory for ${id}. Sessions that reference this server stop being able to spawn it.',
			'mcp.deleteServerSnack' => ({required Object id}) => 'Deleted ${id}.',
			'mcp.serversCount' => ({required Object count}) => 'Servers (${count})',
			'mcp.secretsCount' => ({required Object count}) => 'Secrets (${count})',
			'mcp.emptyServers' => 'No MCP servers registered. Tap "New server" to add one.',
			'mcp.emptySecrets' => 'No secrets stored. Add one to feed sensitive env / headers into MCP servers without putting them in the JSON.',
			'mcp.noVaultFileYet' => 'No vault file yet — added secrets create it.',
			'mcp.tapToReplaceHint' => 'Tap to replace · long-press / trash to delete',
			'mcp.failedToLoad' => 'Failed to load MCP state',
			'mcp.serverCreatedSnack' => 'MCP server created.',
			'mcp.serverUpdatedSnack' => 'MCP server updated.',
			'mcp.envHeading' => 'Env',
			'mcp.encryptionAes' => 'AES-GCM encrypted (key in OS keychain)',
			'mcp.encryptionPlaintext' => 'PLAINTEXT — keychain unavailable',
			'mcp.toggleEnabledSnack' => ({required Object name}) => '${name} enabled.',
			'mcp.toggleDisabledSnack' => ({required Object name}) => '${name} disabled.',
			'providers.title' => 'Providers',
			'providers.configSaved' => 'Provider config updated.',
			'providers.saveFailedApi' => ({required Object error}) => 'Save failed: ${error}',
			'providers.saveFailedGeneric' => ({required Object error}) => 'Save failed: ${error}',
			'providers.reload' => 'Reload',
			'providers.errorPrefix.toggle' => 'Toggle failed',
			'providers.errorPrefix.rename' => 'Rename failed',
			'providers.errorPrefix.delete' => 'Delete failed',
			'providers.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'providers.accounts.rename' => 'Rename',
			'providers.accounts.renameTitle' => ({required Object name}) => 'Rename ${name}',
			'providers.accounts.displayNameLabel' => 'Display name',
			'providers.accounts.displayNameHint' => 'Work account',
			'providers.accounts.deleteTitle' => 'Delete account?',
			'providers.accounts.importFailedApi' => ({required Object error}) => 'Import failed: ${error}',
			'providers.accounts.importFailedGeneric' => ({required Object error}) => 'Import failed: ${error}',
			'integrations.title' => 'Integrations',
			'integrations.register' => 'Register',
			'integrations.registerDialogTitle' => 'Register integration',
			'integrations.edit' => 'Edit',
			'integrations.editTitle' => ({required Object name}) => 'Edit ${name}',
			'integrations.enabledLabel' => 'Enabled',
			'integrations.iSavedIt' => 'I\'ve saved it',
			'integrations.apiKeyForName' => ({required Object name}) => 'API key for ${name}',
			'integrations.apiKeySubtitleRegister' => ({required Object routePrefix}) => 'Hand this to the integration so it can authenticate against /api/v1/${routePrefix}/…',
			'integrations.copiedRequestId' => ({required Object id}) => 'Copied request_id ${id}',
			'integrations.updateOk' => 'Integration updated.',
			'integrations.registerFailedApi' => ({required Object error}) => 'Register failed: ${error}',
			'integrations.registerFailedGeneric' => ({required Object error}) => 'Register failed: ${error}',
			'integrations.updateFailedApi' => ({required Object error}) => 'Update failed: ${error}',
			'integrations.updateFailedGeneric' => ({required Object error}) => 'Update failed: ${error}',
			'integrations.deleteTitle' => 'Delete integration?',
			'integrations.deletedSnack' => ({required Object name}) => 'Deleted ${name}.',
			'integrations.deleteFailedApi' => ({required Object error}) => 'Delete failed: ${error}',
			'integrations.deleteFailedGeneric' => ({required Object error}) => 'Delete failed: ${error}',
			'integrations.rotateKey' => 'Rotate key',
			'integrations.rotateConfirmTitle' => 'Rotate API key?',
			'integrations.rotate' => 'Rotate',
			'integrations.newApiKeyTitle' => ({required Object name}) => 'New API key for ${name}',
			'integrations.newApiKeySubtitle' => 'Hand this to the integration. The previous key has just been invalidated.',
			'integrations.rotateFailedApi' => ({required Object error}) => 'Rotate failed: ${error}',
			'integrations.rotateFailedGeneric' => ({required Object error}) => 'Rotate failed: ${error}',
			'memoryWorkers.title' => 'Memory workers',
			'memoryWorkers.savedSnack' => ({required Object label}) => '${label} saved',
			'memoryWorkers.saveFailed' => ({required Object error}) => 'Save failed: ${error}',
			'memoryWorkers.testFailed' => ({required Object error}) => 'Test call failed: ${error}',
			'memoryWorkers.workerLabel' => 'Worker',
			'memoryWorkers.summarizerHttp' => 'Summarizer (HTTP)',
			'memoryWorkers.agentCliPrint' => 'Agent (CLI --print)',
			'memoryWorkers.cliLabel' => 'CLI',
			'memoryWorkers.cliClaude' => 'Claude',
			'memoryWorkers.cliGemini' => 'Gemini',
			'memoryWorkers.claudeAccountLabel' => 'Claude account',
			'memoryWorkers.claudeAccountDefault' => 'Default',
			'memoryWorkers.test' => 'Test',
			'memoryWorkers.intro' => 'Each memory-system LLM touchpoint can be served independently by the local summarizer endpoint (LM Studio / OpenAI-compat) or by spawning a headless Claude / Gemini agent in --print mode. High-quality narrative tasks (gitactivity, transcript) benefit from agent workers; high-frequency tasks (gatekeeper) stay on the local endpoint by design.',
			'memoryWorkers.errorTitle' => 'Endpoint not reachable',
			'memoryWorkers.errorDetail' => 'The /api/v1/memory/workers routes are new in M25 — the opendray binary may need a restart to mount them and run migration 0029.',
			'memoryWorkers.summarizerOnlyBadge' => 'summarizer-only',
			'memoryWorkers.summarizerInfo' => 'Uses the registry default summarizer provider. Pick a specific row on the web admin.',
			'memoryWorkers.agentWarning' => 'Agent mode spawns a headless CLI per call. Latency ~5-15s (vs ~1s summarizer); cost shifts from CPU to your Claude/Gemini quota.',
			'memoryWorkers.noCalls24h' => 'No calls in last 24h.',
			'memoryWorkers.testOkSnack' => ({required Object label, required Object duration}) => '${label} OK — ${duration}ms',
			'memoryWorkers.testFailedReturnedSnack' => ({required Object label, required Object error}) => '${label} failed: ${error}',
			'memoryWorkers.unknownError' => 'unknown',
			'memoryWorkers.tasks.gatekeeper.label' => 'Gatekeeper',
			'memoryWorkers.tasks.gatekeeper.description' => 'Pre-write filter on every memory_store. High frequency (<500ms target) — summarizer-only.',
			'memoryWorkers.tasks.cleaner.label' => 'Cleaner librarian',
			'memoryWorkers.tasks.cleaner.description' => 'Periodic LLM librarian. Judges aged memories as keep / stale / duplicate.',
			'memoryWorkers.tasks.gitactivity.label' => 'Git activity summariser',
			'memoryWorkers.tasks.gitactivity.description' => 'git log → 2-3 paragraph narrative every 24h. Naturally fits an agent worker.',
			'memoryWorkers.tasks.transcript.label' => 'Session transcript summariser',
			'memoryWorkers.tasks.transcript.description' => 'Session-end \'what did the agent do\' summary. Naturally fits an agent worker.',
			'memoryCleanup.title' => 'Memory cleanup',
			'memoryCleanup.approveFailed' => ({required Object error}) => 'Approve failed: ${error}',
			'memoryCleanup.rejectFailed' => ({required Object error}) => 'Reject failed: ${error}',
			'memoryCleanup.loadFailed' => ({required Object error}) => 'Failed to load: ${error}',
			'memoryCleanup.reject' => 'Reject',
			'project.title' => 'Project',
			'project.pickFirst' => 'Pick a project first.',
			'project.loadFailed' => ({required Object error}) => 'Failed to load: ${error}',
			'project.projectsLoadFailed' => ({required Object error}) => 'Failed to load projects: ${error}',
			'project.projectLabel' => 'Project',
			'project.resetTooltip' => 'Reset project memory',
			'project.append' => 'Append',
			'project.appendDialogTitle' => 'Append journal entry',
			'project.titleFieldLabel' => 'Title (optional)',
			'project.contentFieldLabel' => 'Content (markdown)',
			'project.appendFailed' => ({required Object error}) => 'Failed: ${error}',
			'project.approveFailed' => ({required Object error}) => 'Approve failed: ${error}',
			'project.rejectFailed' => ({required Object error}) => 'Reject failed: ${error}',
			'project.cleanupFailed' => ({required Object error}) => 'Cleanup failed: ${error}',
			'project.resetConfirmTitle' => 'Reset project memory?',
			'project.alsoDeleteScanner' => 'Also delete scanner docs',
			'project.alsoDeletePgvector' => 'Also delete pgvector memories',
			'project.deleteForever' => 'Delete forever',
			'project.resetDoneSnack' => ({required Object parts}) => 'Reset: ${parts}',
			'project.resetFailed' => ({required Object error}) => 'Reset failed: ${error}',
			'project.docSavedSnack' => ({required Object kind}) => '${kind} saved',
			'project.docSaveFailed' => ({required Object error}) => 'Save failed: ${error}',
			'project.docHintTemplate' => ({required Object kind}) => 'Write the ${kind} as markdown…',
			'project.deleteEntryTooltip' => 'Delete entry',
			'project.agentReason' => 'Agent reason',
			'project.reject' => 'Reject',
			'project.approve' => 'Approve',
			'project.replaceConfirmTitle' => ({required Object kind}) => 'Replace current ${kind}?',
			'project.replaceKind' => ({required Object kind}) => 'Replace ${kind}',
			'project.reason' => 'Reason',
			'project.willMergeInto' => 'Will merge into',
			'backups.title' => 'Backups',
			'backups.runConfirmTitle' => 'Run backup now?',
			'backups.run' => 'Run',
			'backups.queuedSnack' => ({required Object id}) => 'Backup queued (${id}). Watching for progress…',
			'backups.runFailedApi' => ({required Object error}) => 'Run failed: ${error}',
			'backups.runFailedGeneric' => ({required Object error}) => 'Run failed: ${error}',
			'backups.detailTitle' => 'Backup detail',
			'backups.deleteTitle' => 'Delete backup?',
			'backups.deletedSnack' => ({required Object id}) => 'Deleted ${id}.',
			'backups.deleteFailedApi' => ({required Object error}) => 'Delete failed: ${error}',
			'backups.deleteFailedGeneric' => ({required Object error}) => 'Delete failed: ${error}',
			'backups.menuSchedules' => 'Schedules',
			'backups.menuTargets' => 'Targets',
			'backups.encryption.checkAgain' => 'Check again',
			'backups.encryption.generate' => 'Generate',
			'backups.encryption.paste' => 'Paste',
			'backups.encryption.random256bit' => '256-bit random key',
			'backups.encryption.passphraseLabel' => 'Your passphrase',
			'backups.encryption.passphraseHint' => 'At least 20 characters',
			'backups.encryption.passphraseCopied' => 'Passphrase copied to clipboard',
			'backupTargets.title' => 'Backup targets',
			'backupTargets.newTarget' => 'New target',
			'backupTargets.testConnection' => 'Test connection',
			'backupTargets.editConfig' => 'Edit config',
			'backupTargets.viewRawConfig' => 'View raw config',
			'backupTargets.configDialogTitle' => ({required Object kind}) => '${kind} config',
			'backupTargets.deleteTitle' => 'Delete target?',
			'backupTargets.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'backupSchedules.title' => 'Backup schedules',
			'backupSchedules.deleteTitle' => 'Delete schedule?',
			'backupSchedules.targetLabel' => 'Target',
			'backupSchedules.intervalLabel' => 'Interval',
			'backupSchedules.retentionLabel' => 'Retention (keep N most recent)',
			'backupSchedules.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'backupTargetEditor.useHttps' => 'Use HTTPS',
			'backupTargetEditor.pathStyle' => 'Path-style addressing',
			'backupTargetEditor.pathStyleSubtitle' => 'Legacy / MinIO',
			'githosts.title' => 'Git hosts',
			'githosts.addHost' => 'Add host',
			'githosts.deleteTitle' => 'Delete git host?',
			'githosts.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'githosts.errorPrefix.toggle' => 'Toggle failed',
			'githosts.errorPrefix.delete' => 'Delete failed',
			'githosts.form.kindLabel' => 'Kind',
			'githosts.form.hostLabel' => 'Host',
			'githosts.form.nameLabel' => 'Name',
			'githosts.form.nameHint' => 'work-github, personal-gitlab, …',
			'channels.title' => 'Channels',
			'channels.kNew' => 'New',
			'channels.sendTest' => 'Send test message',
			'channels.editConfig' => 'Edit config',
			'channels.editNotifications' => 'Edit notifications',
			'channels.viewRawConfig' => 'View raw config',
			'channels.copyChannelId' => 'Copy channel id',
			'channels.copiedSnack' => ({required Object id}) => 'Copied ${id}',
			'channels.createdSnack' => ({required Object kind}) => 'Created ${kind} channel.',
			'channels.createFailedApi' => ({required Object error}) => 'Create failed: ${error}',
			'channels.createFailedGeneric' => ({required Object error}) => 'Create failed: ${error}',
			'channels.deleteTitle' => 'Delete channel?',
			'channels.configDialog.title' => ({required Object kind}) => '${kind} config',
			'channels.webhookDialog.title' => ({required Object kind}) => '${kind} webhook URL',
			'channels.webhookDialog.copiedSnack' => 'Copied webhook URL.',
			'channels.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'channels.notifications.title' => 'Notification preferences',
			'channels.notifications.notifyOn' => 'Notify on',
			'channels.notifications.repeatPolicy' => 'Repeat policy',
			'channels.notifications.cooldownWindow' => 'Cooldown window',
			'channels.notifications.includeSnippet' => 'Include terminal snippet',
			'channels.notifications.snippetLengthCap' => 'Snippet length cap',
			'channels.notifications.notifyOnAll' => 'All session events.',
			'channels.notifications.notifyOnEmpty' => 'No events selected — outbound notifications muted.',
			'channels.notifications.snippetHelper' => 'Embeds the recent terminal tail in each notification.',
			'channels.notifications.snippetNoCap' => 'no cap',
			'channels.notifications.snippetChars' => ({required Object n}) => '${n} chars',
			'channels.notifications.updatedSnack' => 'Notification preferences updated.',
			'channels.notifications.modes.onceLabel' => 'Once per session',
			'channels.notifications.modes.onceDescription' => 'Fire once when idle, stay silent until reply or end.',
			'channels.notifications.modes.cooldownLabel' => 'Time-window cooldown',
			'channels.notifications.modes.cooldownDescription' => 'Suppress repeats within the chosen window.',
			'channels.notifications.modes.everyLabel' => 'Every event (noisy)',
			'channels.notifications.modes.everyDescription' => 'No suppression — only for low-frequency channels.',
			'channels.popup.enable' => 'Enable',
			'channels.popup.disable' => 'Disable',
			'channels.popup.mute' => 'Mute',
			'channels.popup.unmute' => 'Unmute',
			'channels.popup.deleteLabel' => 'Delete',
			'channels.badges.running' => 'running',
			'channels.badges.starting' => 'starting…',
			'channels.badges.disabled' => 'disabled',
			'channels.badges.muted' => 'muted',
			'channels.capsLabel' => ({required Object list}) => '· caps: ${list}',
			'channels.bridgeWebOnly' => 'Bridge channels stay web-only',
			'channels.bridgeEmptyAdd' => 'Add one from the web admin: Channels → New.',
			'channels.deleteBody' => 'Stops the channel and removes its configuration. In-flight notifications addressed to it will be dropped silently.',
			'channels.snacks.testDispatched' => 'Test message dispatched.',
			'channels.snacks.channelEnabled' => 'Channel enabled.',
			'channels.snacks.channelDisabled' => 'Channel disabled.',
			'channels.snacks.channelMuted' => 'Channel muted.',
			'channels.snacks.channelUnmuted' => 'Channel unmuted.',
			'channels.snacks.configUpdated' => 'Channel config updated.',
			'channels.snacks.channelDeleted' => 'Channel deleted.',
			'channels.errorPrefix.test' => 'Test failed',
			'channels.errorPrefix.toggle' => 'Toggle failed',
			'channels.errorPrefix.muteToggle' => 'Mute toggle failed',
			'channels.errorPrefix.update' => 'Update failed',
			'channels.errorPrefix.delete' => 'Delete failed',
			'channels.failedToLoad' => 'Failed to load channels',
			'channels.kinds.telegram.description' => 'Bot via @BotFather. opendray long-polls getUpdates and sends via REST. Buttons + reply_to_message work natively.',
			'channels.kinds.telegram.botTokenLabel' => 'Bot token',
			'channels.kinds.telegram.botTokenHint' => 'From @BotFather. Stored in channel config; admin-only API.',
			'channels.kinds.telegram.chatIdLabel' => 'Default chat ID',
			'channels.kinds.telegram.chatIdPlaceholder' => '42 (optional — used when no ReplyCtx)',
			'channels.kinds.slack.description' => 'Socket Mode — no public webhook needed. Requires a bot OAuth token (xoxb-) and an app-level token (xapp-) with connections:write.',
			'channels.kinds.slack.botTokenLabel' => 'Bot token (xoxb-…)',
			'channels.kinds.slack.botTokenHint' => 'OAuth & Permissions → Bot User OAuth Token. Needs chat:write.',
			'channels.kinds.slack.appTokenLabel' => 'App-level token (xapp-…)',
			'channels.kinds.slack.appTokenHint' => 'Settings → Basic Information → App-Level Tokens. Scope: connections:write.',
			'channels.kinds.slack.channelIdLabel' => 'Default channel ID',
			'channels.kinds.slack.channelIdPlaceholder' => 'C0123ABC456 (optional)',
			'channels.kinds.discord.description' => 'Bot via Discord Developer Portal with MESSAGE CONTENT INTENT enabled. Connects to Gateway WS — no public URL required.',
			'channels.kinds.discord.botTokenLabel' => 'Bot token',
			'channels.kinds.discord.botTokenPlaceholder' => 'Bot token from Discord Developer Portal',
			'channels.kinds.discord.botTokenHint' => 'Application → Bot → Reset Token. Invite bot with send_messages + embed_links.',
			'channels.kinds.discord.channelIdLabel' => 'Default channel ID',
			'channels.kinds.discord.channelIdPlaceholder' => '123456789012345678 (optional)',
			'channels.kinds.feishu.description' => 'App-level credentials. Uses event subscription webhook for inbound. Public webhook URL is generated below — paste it into the Feishu dev console.',
			_ => null,
		} ?? switch (path) {
			'channels.kinds.feishu.afterCreateHint' => 'Open the webhook URL from the channel card and paste it into Feishu Open Platform → Event Subscriptions → Request URL.',
			'channels.kinds.feishu.appIdLabel' => 'App ID',
			'channels.kinds.feishu.appSecretLabel' => 'App secret',
			'channels.kinds.feishu.appSecretPlaceholder' => 'Application credential secret',
			'channels.kinds.feishu.verificationTokenLabel' => 'Verification token',
			'channels.kinds.feishu.verificationTokenHint' => 'From Event Subscriptions → Verification Token. When set, opendray rejects webhooks with a different token.',
			'channels.kinds.feishu.chatIdLabel' => 'Default chat ID (oc_…)',
			'channels.kinds.feishu.chatIdPlaceholder' => 'oc_xxxxxxxxxx (optional)',
			'channels.kinds.dingtalk.description' => 'Custom group robot. Outbound only. Group chat → Robots → Add → Sign mode → copy webhook + secret.',
			'channels.kinds.dingtalk.webhookUrlLabel' => 'Webhook URL',
			'channels.kinds.dingtalk.secretLabel' => 'Sign secret',
			'channels.kinds.dingtalk.secretHint' => 'When the robot is set to "Sign" security mode, copy the secret here. opendray adds the timestamp + sign params automatically.',
			'channels.kinds.wecom.description' => 'Group robot webhook. Outbound only (text + markdown). Group settings → Group robots → Add → copy webhook URL.',
			'channels.kinds.wecom.webhookKeyLabel' => 'Webhook key',
			'channels.kinds.wecom.webhookKeyPlaceholder' => 'The "key=" query value',
			'channels.kinds.wecom.webhookKeyHint' => 'Or paste the whole webhook URL into the field below — either is enough.',
			'channels.kinds.wecom.webhookUrlLabel' => 'Or full webhook URL',
			'channels.kinds.wechat.description' => 'Push to personal WeChat via WxPusher. Outbound-only — push services do not relay user replies. Each recipient subscribes once via QR code.',
			'channels.kinds.wechat.appTokenLabel' => 'App token (AT_…)',
			'channels.kinds.wechat.appTokenHint' => 'WxPusher → 应用管理 → 创建应用 → 复制 App Token.',
			'channels.kinds.wechat.uidsLabel' => 'Recipient UIDs (one per line)',
			'channels.kinds.wechat.uidsHint' => 'Either UIDs or topic IDs is required.',
			'channels.kinds.wechat.topicIdsLabel' => 'Topic IDs (one per line)',
			'channels.kinds.wechat.urlLabel' => 'Tap-through URL',
			'channels.kinds.wechat.urlHint' => 'When set, tapping the WeChat notification opens this page.',
			'onboarding.gatewayLabel' => 'Gateway URL',
			'onboarding.gatewayHint' => 'https://opendray.example.com',
			'onboarding.kContinue' => 'Continue',
			'skills.title' => 'Skills',
			'skills.newSkill' => 'New skill',
			'skills.customizingBuiltin' => ({required Object id}) => 'Customizing built-in ${id}',
			'skills.idLabel' => 'Id (slug)',
			'skills.idHint' => 'e.g. tdd-guide',
			'skills.bodyLabel' => 'Body (markdown)',
			'customTasks.title' => 'Custom tasks',
			'customTasks.newTask' => 'New task',
			'customTasks.deleteTitle' => 'Delete task?',
			'customTasks.deletedSnack' => ({required Object name}) => 'Deleted ${name}.',
			'customTasks.deleteFailedApi' => ({required Object error}) => 'Delete failed: ${error}',
			'customTasks.deleteFailedGeneric' => ({required Object error}) => 'Delete failed: ${error}',
			'customTasks.popupEdit' => 'Edit',
			'customTasks.popupDelete' => 'Delete',
			'customTasks.nameHint' => 'e.g. backend-tests',
			'customTasks.commandHint' => '/run pnpm test --filter backend',
			'customTasks.descriptionHint' => 'One-liner shown under the task name.',
			'customTasks.scopeGlobal' => 'Global',
			'customTasks.scopeProject' => 'Project',
			'customTasks.cwdHint' => '/Users/you/projects/backend',
			'notesPage.title' => 'Notes',
			'notesPage.newButton' => 'New',
			'notesPage.newNoteDialogTitle' => 'New note',
			'notesPage.searchHint' => 'Search across the whole vault…',
			'notesPage.up' => 'Up',
			'notesPage.copyPath' => 'Copy path',
			'notesPage.open' => 'Open',
			'notesPage.copiedSnack' => ({required Object path}) => 'Copied ${path}',
			'notesPage.deleteTitle' => 'Delete note?',
			'notesPage.deletedSnack' => ({required Object path}) => 'Deleted ${path}',
			'notesPage.deleteFailedApi' => ({required Object error}) => 'Delete failed: ${error}',
			'notesPage.deleteFailedGeneric' => ({required Object error}) => 'Delete failed: ${error}',
			'notesPage.createFailedApi' => ({required Object error}) => 'Create failed: ${error}',
			'notesPage.createFailedGeneric' => ({required Object error}) => 'Create failed: ${error}',
			'notesPage.pathLabel' => 'Vault-relative path',
			'notesPage.pathHint' => 'personal/scratch.md',
			'notesPage.create' => 'Create',
			'notesPage.editor.markdownHint' => 'Markdown…',
			'notesPage.editor.saving' => 'Saving…',
			'notesPage.editor.autosave' => 'Auto-saves as you type',
			'memory.title' => 'Memory',
			'memory.more' => 'More',
			'memory.workers' => 'Memory workers',
			'memory.kNew' => 'New',
			'memory.searchHint' => 'Search…',
			'memory.projectLabel' => 'Project',
			'memory.filterHint' => 'Filter by name or path…',
			'memory.copied' => 'Copied',
			'memory.copyTooltip' => 'Copy text',
			'memory.deleteAllConfirm.title' => 'Delete every memory in this scope?',
			'memory.deleteAllConfirm.deleteAll' => 'Delete all',
			'memory.deletedSnackOne' => ({required Object n}) => 'Deleted ${n} memory item',
			'memory.deletedSnackOther' => ({required Object n}) => 'Deleted ${n} memory items',
			'memory.bulkDeleteFailedApi' => ({required Object error}) => 'Bulk delete failed: ${error}',
			'memory.bulkDeleteFailedGeneric' => ({required Object error}) => 'Bulk delete failed: ${error}',
			'memory.deleteOne.title' => 'Delete memory?',
			'memory.deleteOne.body' => 'This cannot be undone.',
			'memory.scope.project' => 'Project',
			'memory.scope.global' => 'Global',
			'memory.create.textLabel' => 'Text',
			'memory.create.scopeKeyLabel' => 'Scope key (project cwd)',
			'memory.create.scopeKeyHint' => '/Users/you/projects/foo',
			'memory.create.submit' => 'Create',
			'about.title' => 'About',
			'about.loading' => 'Loading…',
			'about.sections.app' => 'App',
			'about.sections.server' => 'Server',
			'about.fields.app' => 'App',
			'about.fields.version' => 'Version',
			'about.fields.versionFormat' => ({required Object version, required Object build}) => '${version} (build ${build})',
			'about.fields.package' => 'Package',
			'about.fields.url' => 'URL',
			'about.fields.signedInAs' => 'Signed in as',
			'about.fields.tokenExpires' => 'Token expires',
			'about.copied' => ({required Object label}) => 'Copied ${label}',
			'about.copyTooltip' => 'Copy',
			'about.copyLabels.version' => 'version',
			'about.copyLabels.serverUrl' => 'server URL',
			'about.tagline' => 'opendray mobile — multi-CLI gateway control.\nSource: github.com/Opendray/opendray_v2',
			'settings.title' => 'Settings',
			'settings.language.section' => 'Language',
			'settings.language.system' => 'System',
			'settings.language.systemSubtitle' => 'Follow your phone\'s language setting',
			'settings.language.english' => 'English',
			'settings.language.chinese' => '中文',
			'settings.appearance.section' => 'Appearance',
			'settings.appearance.system' => 'System',
			'settings.appearance.systemSubtitle' => 'Follow your phone\'s appearance setting',
			'settings.appearance.light' => 'Light',
			'settings.appearance.lightSubtitle' => 'Always use the light palette',
			'settings.appearance.dark' => 'Dark',
			'settings.appearance.darkSubtitle' => 'Always use the dark palette',
			'settings.account.section' => 'Account',
			'settings.account.changeCredentials' => 'Change credentials',
			'settings.account.changeCredentialsSubtitle' => 'Username and password',
			'settings.gateway.section' => 'Gateway',
			'settings.gateway.serverSettings' => 'Server settings',
			'settings.gateway.serverSettingsSubtitle' => 'Listen address, logging, vault, memory, storage paths…',
			'settings.gateway.liveLogs' => 'Live logs',
			'settings.gateway.liveLogsSubtitle' => 'Tail the gateway log buffer — same source as the web admin',
			'settings.changeCredentials.title' => 'Change credentials',
			'settings.changeCredentials.explanation' => 'Verify your current password, then pick new credentials. All other signed-in sessions will be revoked.',
			'settings.changeCredentials.currentPassword' => 'Current password',
			'settings.changeCredentials.newUsername' => 'New username',
			'settings.changeCredentials.newPassword' => 'New password',
			'settings.changeCredentials.confirmPassword' => 'Confirm new password',
			'settings.changeCredentials.validatorRequired' => 'Required',
			'settings.changeCredentials.passwordHelper' => 'At least 8 characters',
			'settings.changeCredentials.passwordTooShort' => 'Must be at least 8 characters',
			'settings.changeCredentials.passwordMismatch' => 'Doesn\'t match the new password',
			'settings.changeCredentials.updatedSnack' => 'Credentials updated.',
			'settings.changeCredentials.wrongCurrent' => 'Current password is wrong.',
			'settings.changeCredentials.saving' => 'Saving…',
			'settings.changeCredentials.update' => 'Update',
			'settings.logViewer.title' => 'Live logs',
			'settings.logViewer.reconnect' => 'Reconnect',
			'settings.logViewer.copyBuffer' => 'Copy buffer',
			'settings.logViewer.clearLocal' => 'Clear local view',
			'settings.logViewer.copiedSnack' => 'Copied buffer to clipboard',
			'settings.logViewer.filterHint' => 'Filter substring…',
			'settings.logViewer.levels.all' => 'All',
			'settings.logViewer.levels.debug' => 'Debug',
			'settings.logViewer.levels.info' => 'Info',
			'settings.logViewer.levels.warn' => 'Warn',
			'settings.logViewer.levels.error' => 'Error',
			'settings.serverSettings.title' => 'Server settings',
			'settings.serverSettings.reloadTooltip' => 'Reload from server',
			'settings.serverSettings.restartTooltip' => 'Restart gateway',
			'settings.serverSettings.restartConfirmTitle' => 'Restart opendray?',
			'settings.serverSettings.restart' => 'Restart',
			'settings.serverSettings.restartQueuedSnack' => 'Restart requested. Pull-to-refresh in a moment.',
			'settings.serverSettings.restartFailedApi' => ({required Object error}) => 'Restart failed: ${error}',
			'settings.serverSettings.restartFailedGeneric' => ({required Object error}) => 'Restart failed: ${error}',
			'settings.serverSettings.sections.general' => 'General',
			'settings.serverSettings.sections.logging' => 'Logging',
			'settings.serverSettings.sections.sessions' => 'Sessions',
			'settings.serverSettings.sections.vault' => 'Vault',
			'settings.serverSettings.sections.mcpRegistry' => 'MCP registry',
			'settings.serverSettings.sections.memory' => 'Memory',
			'settings.serverSettings.sections.backup' => 'Backup',
			'settings.serverSettings.sections.storageClaude' => 'Storage · Claude',
			'settings.serverSettings.sections.storageCodex' => 'Storage · Codex',
			'settings.serverSettings.sections.storageGemini' => 'Storage · Gemini',
			_ => null,
		};
	}
}
