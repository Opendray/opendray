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
	late final TranslationsWebEn web = TranslationsWebEn.internal(_root);
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
	late final TranslationsDataExportEn dataExport = TranslationsDataExportEn.internal(_root);
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

	/// en: 'Signing in…'
	String get signingIn => 'Signing in…';

	/// en: 'Use your operator credentials.'
	String get subtitle => 'Use your operator credentials.';

	/// en: 'Username and password are required'
	String get errorRequired => 'Username and password are required';

	/// en: 'Login failed: {error}'
	String errorGeneric({required Object error}) => 'Login failed: ${error}';

	/// en: 'Login failed'
	String get errorFallback => 'Login failed';
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

	/// en: 'Activity'
	String get activity => 'Activity';

	/// en: 'Providers'
	String get providers => 'Providers';

	/// en: 'Channels'
	String get channels => 'Channels';

	/// en: 'Integrations'
	String get integrations => 'Integrations';

	/// en: 'Plugins'
	String get plugins => 'Plugins';

	/// en: 'Backups'
	String get backups => 'Backups';

	/// en: 'Settings'
	String get settings => 'Settings';

	/// en: 'Tutorial'
	String get tutorial => 'Tutorial';

	/// en: 'Workspace'
	String get workspace => 'Workspace';
}

// Path: web
class TranslationsWebEn {
	TranslationsWebEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'opendray'
	String get brand => 'opendray';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	late final TranslationsWebTopbarEn topbar = TranslationsWebTopbarEn.internal(_root);
	late final TranslationsWebSessionsEn sessions = TranslationsWebSessionsEn.internal(_root);
	late final TranslationsWebMemoryEn memory = TranslationsWebMemoryEn.internal(_root);
	late final TranslationsWebJournalStaleEn journalStale = TranslationsWebJournalStaleEn.internal(_root);
	late final TranslationsWebConflictsEn conflicts = TranslationsWebConflictsEn.internal(_root);
	late final TranslationsWebMemoryHealthEn memoryHealth = TranslationsWebMemoryHealthEn.internal(_root);
	late final TranslationsWebMemoryConfigEn memoryConfig = TranslationsWebMemoryConfigEn.internal(_root);
	late final TranslationsWebMemoryWorkersEn memoryWorkers = TranslationsWebMemoryWorkersEn.internal(_root);
	late final TranslationsWebCleanupInboxEn cleanupInbox = TranslationsWebCleanupInboxEn.internal(_root);
	late final TranslationsWebProjectEn project = TranslationsWebProjectEn.internal(_root);
	late final TranslationsWebMemoryInspectorEn memoryInspector = TranslationsWebMemoryInspectorEn.internal(_root);
	late final TranslationsWebNotesEn notes = TranslationsWebNotesEn.internal(_root);
	late final TranslationsWebActivityEn activity = TranslationsWebActivityEn.internal(_root);
	late final TranslationsWebProvidersEn providers = TranslationsWebProvidersEn.internal(_root);
	late final TranslationsWebChannelsEn channels = TranslationsWebChannelsEn.internal(_root);
	late final TranslationsWebIntegrationsEn integrations = TranslationsWebIntegrationsEn.internal(_root);
	late final TranslationsWebPluginsEn plugins = TranslationsWebPluginsEn.internal(_root);
	late final TranslationsWebBackupsEn backups = TranslationsWebBackupsEn.internal(_root);
	late final TranslationsWebServerSettingsEn serverSettings = TranslationsWebServerSettingsEn.internal(_root);
	late final TranslationsWebSettingsEn settings = TranslationsWebSettingsEn.internal(_root);
	late final TranslationsWebLogViewerEn logViewer = TranslationsWebLogViewerEn.internal(_root);
	late final TranslationsWebPathInputEn pathInput = TranslationsWebPathInputEn.internal(_root);
	late final TranslationsWebMemoryAmbientEn memoryAmbient = TranslationsWebMemoryAmbientEn.internal(_root);
	late final TranslationsWebNoteEditorEn noteEditor = TranslationsWebNoteEditorEn.internal(_root);
	late final TranslationsWebExportEn export = TranslationsWebExportEn.internal(_root);
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

	/// en: 'Provider config'
	String get configFallbackTitle => 'Provider config';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Failed to load provider'
	String get configLoadFailed => 'Failed to load provider';

	/// en: 'Whitespace-separated CLI args.'
	String get argsHelper => 'Whitespace-separated CLI args.';

	/// en: 'No providers loaded.'
	String get listEmptyHeadline => 'No providers loaded.';

	/// en: 'The gateway resolves providers from its plugin directory at startup. Check the logs if you expect one.'
	String get listEmptyBody => 'The gateway resolves providers from its plugin directory at startup. Check the logs if you expect one.';

	/// en: 'Failed to load providers'
	String get listLoadFailed => 'Failed to load providers';

	/// en: 'CLI providers'
	String get cliSectionHeader => 'CLI providers';

	/// en: '{name} enabled.'
	String enabledSnack({required Object name}) => '${name} enabled.';

	/// en: '{name} disabled.'
	String disabledSnack({required Object name}) => '${name} disabled.';
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

	/// en: 'Removes the registration and revokes the API key. In-flight requests using the old key will start failing.'
	String get deleteBody => 'Removes the registration and revokes the API key. In-flight requests using the old key will start failing.';

	/// en: 'Generates a new API key for {name} and immediately invalidates the old one.'
	String rotateBody({required Object name}) => 'Generates a new API key for ${name} and immediately invalidates the old one.';

	/// en: 'Integration'
	String get appBarFallback => 'Integration';

	/// en: 'More'
	String get tooltipMore => 'More';

	/// en: 'System integration — read-only'
	String get tooltipReadOnly => 'System integration — read-only';

	/// en: 'Route prefix'
	String get kvRoutePrefix => 'Route prefix';

	/// en: 'Base URL'
	String get kvBaseUrl => 'Base URL';

	/// en: 'Scopes'
	String get kvScopes => 'Scopes';

	/// en: 'Version'
	String get kvVersion => 'Version';

	/// en: 'Last health ping'
	String get kvLastHealthPing => 'Last health ping';

	/// en: 'Created'
	String get kvCreated => 'Created';

	/// en: 'Key rotated'
	String get kvKeyRotated => 'Key rotated';

	/// en: 'Failed to load integration: {error}'
	String detailLoadFailed({required Object error}) => 'Failed to load integration: ${error}';

	/// en: 'Failed to load calls'
	String get callsLoadFailed => 'Failed to load calls';

	/// en: 'No matching calls in the log yet.'
	String get noMatchingCalls => 'No matching calls in the log yet.';

	/// en: 'All'
	String get directionAll => 'All';

	/// en: 'Inbound'
	String get directionInbound => 'Inbound';

	/// en: 'Outbound'
	String get directionOutbound => 'Outbound';

	late final TranslationsIntegrationsFormEn form = TranslationsIntegrationsFormEn.internal(_root);

	/// en: 'Register from the web admin: Integrations → New.'
	String get emptyState => 'Register from the web admin: Integrations → New.';

	/// en: 'Registered'
	String get sectionRegistered => 'Registered';

	/// en: 'System'
	String get sectionSystem => 'System';

	/// en: 'Failed to load integrations'
	String get listLoadFailed => 'Failed to load integrations';
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

	late final TranslationsProjectHealthEn health = TranslationsProjectHealthEn.internal(_root);
	late final TranslationsProjectConflictsEn conflicts = TranslationsProjectConflictsEn.internal(_root);
	late final TranslationsProjectJournalPruneEn journalPrune = TranslationsProjectJournalPruneEn.internal(_root);

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

	/// en: 'Triggers a fresh dump against the local target. The job runs server-side; this list will refresh as it progresses.'
	String get runConfirmBody => 'Triggers a fresh dump against the local target. The job runs server-side; this list will refresh as it progresses.';

	/// en: 'Run'
	String get run => 'Run';

	/// en: 'Run now'
	String get runNow => 'Run now';

	/// en: 'Queueing…'
	String get queueing => 'Queueing…';

	/// en: 'Backup queued ({id}). Watching for progress…'
	String queuedSnack({required Object id}) => 'Backup queued (${id}). Watching for progress…';

	/// en: 'Run failed: {error}'
	String runFailedApi({required Object error}) => 'Run failed: ${error}';

	/// en: 'Run failed: {error}'
	String runFailedGeneric({required Object error}) => 'Run failed: ${error}';

	/// en: 'Backup succeeded ({bytes}).'
	String rowSucceededSnack({required Object bytes}) => 'Backup succeeded (${bytes}).';

	/// en: 'Backup failed: {error}'
	String rowFailedSnack({required Object error}) => 'Backup failed: ${error}';

	/// en: 'unknown error'
	String get unknownError => 'unknown error';

	/// en: 'Backup detail'
	String get detailTitle => 'Backup detail';

	/// en: 'Delete backup?'
	String get deleteTitle => 'Delete backup?';

	/// en: 'Removes the blob from {target} and marks the row deleted in the index.'
	String deleteBody({required Object target}) => 'Removes the blob from ${target} and marks the row deleted in the index.';

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

	late final TranslationsBackupsKvEn kv = TranslationsBackupsKvEn.internal(_root);
	late final TranslationsBackupsEmptyMissingDepsEn emptyMissingDeps = TranslationsBackupsEmptyMissingDepsEn.internal(_root);
	late final TranslationsBackupsEmptyNoTargetsEn emptyNoTargets = TranslationsBackupsEmptyNoTargetsEn.internal(_root);
	late final TranslationsBackupsEmptyNoBackupsEn emptyNoBackups = TranslationsBackupsEmptyNoBackupsEn.internal(_root);

	/// en: 'Restart opendray to activate backups'
	String get restartToActivate => 'Restart opendray to activate backups';

	/// en: 'Your passphrase is saved. The gateway only loads it at startup, so changes only take effect after a restart.'
	String get passphraseSaved => 'Your passphrase is saved. The gateway only loads it at startup, so changes only take effect after a restart.';

	/// en: 'Key file'
	String get keyFileLabel => 'Key file';

	/// en: 'Configured via'
	String get configuredViaLabel => 'Configured via';

	late final TranslationsBackupsWizardEn wizard = TranslationsBackupsWizardEn.internal(_root);

	/// en: 'Backups ready'
	String get statusReady => 'Backups ready';

	/// en: 'Backups cannot run'
	String get statusCannot => 'Backups cannot run';

	/// en: 'Targets'
	String get overviewTargets => 'Targets';

	/// en: 'Schedules'
	String get overviewSchedules => 'Schedules';

	/// en: 'Backups'
	String get overviewBackups => 'Backups';

	/// en: 'Failed to load backups'
	String get failedToLoad => 'Failed to load backups';

	/// en: 'OPENDRAY_BACKUP_KEY env var'
	String get envVarConfigured => 'OPENDRAY_BACKUP_KEY env var';

	/// en: 'I have saved this passphrase to my password manager'
	String get savedConfirmCheckbox => 'I have saved this passphrase to my password manager';

	/// en: 'pg_dump is not on PATH. Install postgresql-client and restart opendray.'
	String get pgDumpMissing => 'pg_dump is not on PATH. Install postgresql-client and restart opendray.';

	late final TranslationsBackupsEncryptionEn encryption = TranslationsBackupsEncryptionEn.internal(_root);

	/// en: 'Restore from file'
	String get restoreFromFile => 'Restore from file';

	late final TranslationsBackupsRestoreEn restore = TranslationsBackupsRestoreEn.internal(_root);
	late final TranslationsBackupsInventoryEn inventory = TranslationsBackupsInventoryEn.internal(_root);
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

	/// en: 'New'
	String get newButton => 'New';

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

	/// en: 'No backup targets configured. Add one from the web admin or the Targets screen.'
	String get noTargets => 'No backup targets configured. Add one from the web admin or the Targets screen.';

	/// en: 'Schedule created.'
	String get okMsgCreate => 'Schedule created.';

	/// en: 'Schedule updated.'
	String get okMsgUpdate => 'Schedule updated.';

	/// en: 'Schedule deleted.'
	String get okMsgDelete => 'Schedule deleted.';

	/// en: 'Create failed'
	String get errorPrefixCreate => 'Create failed';

	/// en: 'Update failed'
	String get errorPrefixUpdate => 'Update failed';

	/// en: 'Delete failed'
	String get errorPrefixDelete => 'Delete failed';

	/// en: 'Removes the recurring spec for target {targetId}. Existing backup blobs are not touched.'
	String deleteBody({required Object targetId}) => 'Removes the recurring spec for target ${targetId}. Existing backup blobs are not touched.';

	/// en: 'No schedules yet. Tap "New" to create one.'
	String get emptyList => 'No schedules yet.\nTap "New" to create one.';

	/// en: 'Pick a target.'
	String get validatePickTarget => 'Pick a target.';

	/// en: 'Interval must be > 0.'
	String get validateInterval => 'Interval must be > 0.';

	/// en: 'Edit schedule'
	String get formTitleEdit => 'Edit schedule';

	/// en: 'New schedule'
	String get formTitleNew => 'New schedule';

	/// en: 'Save'
	String get saveButtonEdit => 'Save';

	/// en: 'Create'
	String get saveButtonNew => 'Create';

	/// en: 'Target is fixed once created.'
	String get targetFixedHint => 'Target is fixed once created.';

	/// en: 'Scheduler will run this on cadence.'
	String get enabledOn => 'Scheduler will run this on cadence.';

	/// en: 'Paused — no automatic runs until re-enabled.'
	String get enabledOff => 'Paused — no automatic runs until re-enabled.';

	/// en: 'Failed to load schedules'
	String get loadFailedTitle => 'Failed to load schedules';

	/// en: 'paused'
	String get pausedBadge => 'paused';

	/// en: 'every {interval}'
	String everyInterval({required Object interval}) => 'every ${interval}';

	/// en: '· keep {n}'
	String keepRetention({required Object n}) => '· keep ${n}';

	/// en: '· next {when}'
	String nextRun({required Object when}) => '· next ${when}';

	/// en: '· last {when}'
	String lastRun({required Object when}) => '· last ${when}';
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

	late final TranslationsBackupTargetEditorKindsEn kinds = TranslationsBackupTargetEditorKindsEn.internal(_root);

	/// en: 'Edit target'
	String get formTitleEdit => 'Edit target';

	/// en: 'New backup target'
	String get formTitleNew => 'New backup target';

	/// en: 'Auto: {prefix}-1'
	String idHintAuto({required Object prefix}) => 'Auto: ${prefix}-1';

	/// en: 'Lower-case letters, digits, dashes. Defaults to the next available slot.'
	String get idHelper => 'Lower-case letters, digits, dashes. Defaults to the next available slot.';

	/// en: 'Scheduled and ad-hoc backups can target this.'
	String get enabledOn => 'Scheduled and ad-hoc backups can target this.';

	/// en: 'Server will refuse to write backups here.'
	String get enabledOff => 'Server will refuse to write backups here.';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Create'
	String get create => 'Create';

	/// en: 'Root directory'
	String get rootDirLabel => 'Root directory';

	/// en: 'Empty = cfg.backup.local_dir (~/.opendray/backups)'
	String get rootDirHint => 'Empty = cfg.backup.local_dir (~/.opendray/backups)';

	/// en: 'Host'
	String get hostLabel => 'Host';

	/// en: 'Port'
	String get portLabel => 'Port';

	/// en: 'Share'
	String get shareLabel => 'Share';

	/// en: 'Top-level share name'
	String get shareHint => 'Top-level share name';

	/// en: 'Claude_Workspace'
	String get shareSampleHint => 'Claude_Workspace';

	/// en: 'User'
	String get userLabel => 'User';

	/// en: 'Password'
	String get passwordLabel => 'Password';

	/// en: 'Leave blank to keep current'
	String get passwordHintKeepCurrent => 'Leave blank to keep current';

	/// en: 'Leave blank to keep'
	String get passwordHintKeep => 'Leave blank to keep';

	/// en: 'Path prefix'
	String get pathPrefixLabel => 'Path prefix';

	/// en: 'Sub-folder under the share root (optional)'
	String get pathPrefixHintShareRoot => 'Sub-folder under the share root (optional)';

	/// en: 'Sub-folder under the base URL (optional)'
	String get pathPrefixHintBaseUrl => 'Sub-folder under the base URL (optional)';

	/// en: 'Object-key prefix (optional)'
	String get pathPrefixHintObjectKey => 'Object-key prefix (optional)';

	/// en: 'Absolute or relative to user home (optional)'
	String get pathPrefixHintSshFolder => 'Absolute or relative to user home (optional)';

	/// en: 'Sub-folder under the remote root (optional)'
	String get pathPrefixHintRemoteRoot => 'Sub-folder under the remote root (optional)';

	/// en: 'Endpoint'
	String get endpointLabel => 'Endpoint';

	/// en: 'Region'
	String get regionLabel => 'Region';

	/// en: 'Bucket'
	String get bucketLabel => 'Bucket';

	/// en: 'Access key'
	String get accessKeyLabel => 'Access key';

	/// en: 'Secret key'
	String get secretKeyLabel => 'Secret key';

	/// en: 'Leave blank to keep current. Stored AES-256-GCM encrypted.'
	String get secretKeyHintEdit => 'Leave blank to keep current. Stored AES-256-GCM encrypted.';

	/// en: 'Stored AES-256-GCM encrypted; never echoed back.'
	String get secretKeyHintNew => 'Stored AES-256-GCM encrypted; never echoed back.';

	/// en: 'Base URL'
	String get baseUrlLabel => 'Base URL';

	/// en: 'Full URL including path. Nextcloud: https://cloud.example/remote.php/dav/files/<user>'
	String get baseUrlHint => 'Full URL including path. Nextcloud: https://cloud.example/remote.php/dav/files/<user>';

	/// en: 'Leave blank to keep. If both password + private key are present, the private key wins.'
	String get sftpPasswordHintEdit => 'Leave blank to keep. If both password + private key are present, the private key wins.';

	/// en: 'Either password OR private key. If both, password becomes a fallback only.'
	String get sftpPasswordHintNew => 'Either password OR private key. If both, password becomes a fallback only.';

	/// en: 'Private key (PEM)'
	String get privateKeyLabel => 'Private key (PEM)';

	/// en: 'Leave blank to keep. Paste OpenSSH/PEM contents.'
	String get privateKeyHintEdit => 'Leave blank to keep. Paste OpenSSH/PEM contents.';

	/// en: 'Paste the contents of an OpenSSH/PEM private key. Multi-line input — keep the BEGIN/END markers.'
	String get privateKeyHintNew => 'Paste the contents of an OpenSSH/PEM private key. Multi-line input — keep the BEGIN/END markers.';

	/// en: 'Host key (pinning)'
	String get hostKeyLabel => 'Host key (pinning)';

	/// en: 'OpenSSH-style server public key. `ssh-keyscan <host>` to obtain. Blank = no pinning (NOT recommended outside LAN).'
	String get hostKeyHint => 'OpenSSH-style server public key. `ssh-keyscan <host>` to obtain. Blank = no pinning (NOT recommended outside LAN).';

	/// en: 'Requires the rclone CLI on the opendray host. First run `rclone config` once interactively to authenticate cloud accounts.'
	String get rcloneNote => 'Requires the rclone CLI on the opendray host. First run `rclone config` once interactively to authenticate cloud accounts.';

	/// en: 'Remote name'
	String get rcloneRemoteLabel => 'Remote name';

	/// en: 'Name from `rclone config` (no colon).'
	String get rcloneRemoteHint => 'Name from `rclone config` (no colon).';

	/// en: 'Binary path'
	String get rcloneBinaryLabel => 'Binary path';

	/// en: 'Override `which rclone`. Empty = PATH lookup.'
	String get rcloneBinaryHint => 'Override `which rclone`. Empty = PATH lookup.';

	/// en: 'Config path'
	String get rcloneConfigLabel => 'Config path';

	/// en: 'Override --config. Empty = rclone default.'
	String get rcloneConfigHint => 'Override --config. Empty = rclone default.';
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

	/// en: 'Removes the credential. Sessions trying to list PRs from {host} will fall back to the unauthenticated API.'
	String deleteBody({required Object host}) => 'Removes the credential. Sessions trying to list PRs from ${host} will fall back to the unauthenticated API.';

	/// en: 'Deleted {name}.'
	String deletedSnack({required Object name}) => 'Deleted ${name}.';

	/// en: '{name} enabled.'
	String enabledSnack({required Object name}) => '${name} enabled.';

	/// en: '{name} disabled.'
	String disabledSnack({required Object name}) => '${name} disabled.';

	/// en: 'No git hosts configured. Add a credential so the gateway can list pull requests across your repos.'
	String get emptyList => 'No git hosts configured.\n\nAdd a credential so the gateway can list pull requests across your repos.';

	/// en: 'Failed to load git hosts'
	String get failedToLoad => 'Failed to load git hosts';
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

	/// en: 'Load failed: {error}'
	String loadFailedApi({required Object error}) => 'Load failed: ${error}';

	/// en: 'Load failed: {error}'
	String loadFailedGeneric({required Object error}) => 'Load failed: ${error}';

	/// en: 'Id is required.'
	String get idRequired => 'Id is required.';

	/// en: 'Body cannot be empty.'
	String get bodyRequired => 'Body cannot be empty.';

	/// en: 'Skill created.'
	String get snackCreated => 'Skill created.';

	/// en: 'Saved as vault override.'
	String get snackOverride => 'Saved as vault override.';

	/// en: 'Skill updated.'
	String get snackUpdated => 'Skill updated.';

	/// en: 'Save failed: {error}'
	String saveFailedApi({required Object error}) => 'Save failed: ${error}';

	/// en: 'Save failed: {error}'
	String saveFailedGeneric({required Object error}) => 'Save failed: ${error}';

	/// en: 'Reset to built-in?'
	String get resetTitle => 'Reset to built-in?';

	/// en: 'Delete skill?'
	String get deleteTitle => 'Delete skill?';

	/// en: 'Removes the vault override for {id}. Sessions will fall back to the built-in body.'
	String resetBody({required Object id}) => 'Removes the vault override for ${id}. Sessions will fall back to the built-in body.';

	/// en: 'Reset'
	String get resetButton => 'Reset';

	/// en: 'Reset {id} to built-in.'
	String resetSnack({required Object id}) => 'Reset ${id} to built-in.';

	/// en: 'Deleted {id}.'
	String deletedSnack({required Object id}) => 'Deleted ${id}.';

	/// en: 'Delete failed: {error}'
	String deleteFailedApi({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Delete failed: {error}'
	String deleteFailedGeneric({required Object error}) => 'Delete failed: ${error}';

	/// en: 'Removes {id} from the vault. Sessions that reference it will fail until restored.'
	String deleteBody({required Object id}) => 'Removes ${id} from the vault. Sessions that reference it will fail until restored.';

	/// en: 'New skill'
	String get newSkillTitle => 'New skill';

	/// en: 'Customize {id}'
	String customizeTitle({required Object id}) => 'Customize ${id}';

	/// en: 'Edit {id}'
	String editTitle({required Object id}) => 'Edit ${id}';

	/// en: 'Reset to built-in'
	String get resetTooltip => 'Reset to built-in';

	/// en: 'Delete'
	String get deleteTooltip => 'Delete';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Save override'
	String get saveOverride => 'Save override';

	/// en: 'Saving creates a vault override with the same id. Sessions will use this body instead of the built-in until you reset.'
	String get overrideBanner => 'Saving creates a vault override with the same id. Sessions will use this body instead of the built-in until you reset.';

	/// en: 'Lowercase letters / digits / dash. Locked once created.'
	String get idHelper => 'Lowercase letters / digits / dash. Locked once created.';

	/// en: 'No skills configured. The gateway ships with built-ins (planner, code-reviewer, etc.).'
	String get emptyList => 'No skills configured. The gateway ships with built-ins (planner, code-reviewer, etc.).';

	/// en: 'Failed to load skills'
	String get failedToLoad => 'Failed to load skills';
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

	/// en: 'Task created.'
	String get snackCreated => 'Task created.';

	/// en: 'Task updated.'
	String get snackUpdated => 'Task updated.';

	/// en: 'Removes the task from the catalogue. Sessions that already inserted it stay unaffected.'
	String get deleteBody => 'Removes the task from the catalogue. Sessions that already inserted it stay unaffected.';

	/// en: 'Define your own slash commands. They appear in the session task picker alongside the built-ins.'
	String get introBanner => 'Define your own slash commands. They appear in the session task picker alongside the built-ins.';

	/// en: 'Name is required'
	String get validateNameRequired => 'Name is required';

	/// en: 'Command is required'
	String get validateCommandRequired => 'Command is required';

	/// en: 'Project-scoped tasks need an absolute cwd path'
	String get validateProjectCwd => 'Project-scoped tasks need an absolute cwd path';

	/// en: 'Edit custom task'
	String get appBarEdit => 'Edit custom task';

	/// en: 'New custom task'
	String get appBarNew => 'New custom task';

	/// en: 'Name'
	String get fieldName => 'Name';

	/// en: 'Shown in the inspector's task picker.'
	String get nameHelper => 'Shown in the inspector\'s task picker.';

	/// en: 'Command'
	String get fieldCommand => 'Command';

	/// en: 'The text inserted into the session when picked. Can be a CLI command or a Claude slash command.'
	String get commandHelper => 'The text inserted into the session when picked. Can be a CLI command or a Claude slash command.';

	/// en: 'Description (optional)'
	String get fieldDescription => 'Description (optional)';

	/// en: 'Scope'
	String get fieldScope => 'Scope';

	/// en: 'Visible from every session, regardless of cwd.'
	String get globalScopeHint => 'Visible from every session, regardless of cwd.';

	/// en: 'Visible only when a session's cwd matches the path below.'
	String get projectScopeHint => 'Visible only when a session\'s cwd matches the path below.';

	/// en: 'Project cwd'
	String get fieldProjectCwd => 'Project cwd';

	/// en: 'Absolute path. Sessions spawned with this exact cwd will see the task.'
	String get cwdHelper => 'Absolute path. Sessions spawned with this exact cwd will see the task.';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Create'
	String get create => 'Create';

	/// en: 'Failed to load custom tasks'
	String get failedToLoad => 'Failed to load custom tasks';
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

	/// en: 'Delete'
	String get popupDelete => 'Delete';

	/// en: 'This is irreversible. Vault git sync will remove the file on the gateway host too.'
	String get deleteBody => 'This is irreversible. Vault git sync will remove the file on the gateway host too.';

	/// en: 'No notes match "{query}".'
	String emptyFilterMatch({required Object query}) => 'No notes match "${query}".';

	/// en: 'Vault is empty. Tap + to create your first note.'
	String get emptyVault => 'Vault is empty. Tap + to create your first note.';

	/// en: 'Folder "{path}" is empty.'
	String emptyFolder({required Object path}) => 'Folder "${path}" is empty.';

	/// en: 'Path is required'
	String get validatePath => 'Path is required';

	/// en: 'Path cannot contain ".."'
	String get validatePathDots => 'Path cannot contain ".."';

	/// en: 'Auto-appends .md if missing.'
	String get pathHelper => 'Auto-appends .md if missing.';

	late final TranslationsNotesPageEditorEn editor = TranslationsNotesPageEditorEn.internal(_root);
}

// Path: dataExport
class TranslationsDataExportEn {
	TranslationsDataExportEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Data export & import'
	String get title => 'Data export & import';

	/// en: 'User-level bundles for migration or verification — separate from /backups (disaster recovery).'
	String get subtitle => 'User-level bundles for migration or verification — separate from /backups (disaster recovery).';

	late final TranslationsDataExportSectionsEn sections = TranslationsDataExportSectionsEn.internal(_root);
	late final TranslationsDataExportFormEn form = TranslationsDataExportFormEn.internal(_root);
	late final TranslationsDataExportHistoryEn history = TranslationsDataExportHistoryEn.internal(_root);
	late final TranslationsDataExportImportEn import = TranslationsDataExportImportEn.internal(_root);
	late final TranslationsDataExportImportsEn imports = TranslationsDataExportImportsEn.internal(_root);
	late final TranslationsDataExportRelativeEn relative = TranslationsDataExportRelativeEn.internal(_root);
	late final TranslationsDataExportStatusEn status = TranslationsDataExportStatusEn.internal(_root);
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

	late final TranslationsMemoryRankEn rank = TranslationsMemoryRankEn.internal(_root);

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

// Path: web.topbar
class TranslationsWebTopbarEn {
	TranslationsWebTopbarEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Expand sidebar'
	String get expandSidebar => 'Expand sidebar';

	/// en: 'Collapse sidebar'
	String get collapseSidebar => 'Collapse sidebar';

	/// en: 'Search'
	String get search => 'Search';

	/// en: 'Open command palette'
	String get openPalette => 'Open command palette';

	/// en: 'Theme'
	String get theme => 'Theme';

	/// en: 'Theme: {mode}'
	String themeLabel({required Object mode}) => 'Theme: ${mode}';

	/// en: 'Appearance'
	String get appearance => 'Appearance';

	/// en: 'Light'
	String get themeLight => 'Light';

	/// en: 'Dark'
	String get themeDark => 'Dark';

	/// en: 'System'
	String get themeSystem => 'System';

	/// en: 'Language'
	String get language => 'Language';

	/// en: 'English'
	String get languageEnglish => 'English';

	/// en: '中文'
	String get languageChinese => '中文';

	/// en: 'Signed in as'
	String get signedInAs => 'Signed in as';

	/// en: 'Token expires'
	String get tokenExpires => 'Token expires';

	/// en: 'Sign out'
	String get signOut => 'Sign out';
}

// Path: web.sessions
class TranslationsWebSessionsEn {
	TranslationsWebSessionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebSessionsListEn list = TranslationsWebSessionsListEn.internal(_root);
	late final TranslationsWebSessionsTabsEn tabs = TranslationsWebSessionsTabsEn.internal(_root);
	late final TranslationsWebSessionsPageEn page = TranslationsWebSessionsPageEn.internal(_root);
	late final TranslationsWebSessionsEmptyEn empty = TranslationsWebSessionsEmptyEn.internal(_root);
	late final TranslationsWebSessionsHeaderEn header = TranslationsWebSessionsHeaderEn.internal(_root);
	late final TranslationsWebSessionsTerminalEn terminal = TranslationsWebSessionsTerminalEn.internal(_root);
	late final TranslationsWebSessionsSpawnEn spawn = TranslationsWebSessionsSpawnEn.internal(_root);
	late final TranslationsWebSessionsAccountSwitcherEn accountSwitcher = TranslationsWebSessionsAccountSwitcherEn.internal(_root);
	late final TranslationsWebSessionsInspectorEn inspector = TranslationsWebSessionsInspectorEn.internal(_root);
	late final TranslationsWebSessionsEndedEn ended = TranslationsWebSessionsEndedEn.internal(_root);
	late final TranslationsWebSessionsFileBrowserEn fileBrowser = TranslationsWebSessionsFileBrowserEn.internal(_root);
}

// Path: web.memory
class TranslationsWebMemoryEn {
	TranslationsWebMemoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory'
	String get title => 'Memory';

	/// en: 'Browse, search and edit memories agents have stored via the opendray-memory MCP server.'
	String get subtitle => 'Browse, search and edit memories agents have stored via the opendray-memory MCP server.';

	/// en: 'Project'
	String get navProject => 'Project';

	/// en: 'Cleanup inbox'
	String get navCleanupInbox => 'Cleanup inbox';

	/// en: 'Workers'
	String get navWorkers => 'Workers';

	/// en: 'Configuration →'
	String get navConfiguration => 'Configuration →';
}

// Path: web.journalStale
class TranslationsWebJournalStaleEn {
	TranslationsWebJournalStaleEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Prune stale entries'
	String get title => 'Prune stale entries';

	/// en: '(older than {days} days, no pending conflicts)'
	String subtitle({required Object days}) => '(older than ${days} days, no pending conflicts)';

	/// en: 'Older than (days):'
	String get daysLabel => 'Older than (days):';

	/// en: 'Scanning…'
	String get loading => 'Scanning…';

	/// en: 'Nothing stale to prune.'
	String get empty => 'Nothing stale to prune.';

	/// en: 'Select all'
	String get selectAll => 'Select all';

	/// en: 'Deselect all'
	String get deselectAll => 'Deselect all';

	/// en: 'Delete ({count})'
	String deleteSelected({required Object count}) => 'Delete (${count})';

	/// en: '{count} entry deleted'
	String deleted_one({required Object count}) => '${count} entry deleted';

	/// en: '{count} entries deleted'
	String deleted_other({required Object count}) => '${count} entries deleted';
}

// Path: web.conflicts
class TranslationsWebConflictsEn {
	TranslationsWebConflictsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Cross-layer conflicts'
	String get title => 'Cross-layer conflicts';

	/// en: 'Contradictions the daily detector found between facts, plan, goal, and journal entries.'
	String get subtitle => 'Contradictions the daily detector found between facts, plan, goal, and journal entries.';

	/// en: 'Loading conflicts…'
	String get loading => 'Loading conflicts…';

	/// en: 'No pending conflicts. Click "Detect now" to run an on-demand sweep.'
	String get empty => 'No pending conflicts. Click "Detect now" to run an on-demand sweep.';

	/// en: 'Pick a project to see its conflicts.'
	String get pickCwd => 'Pick a project to see its conflicts.';

	/// en: 'Detect now'
	String get detectNow => 'Detect now';

	/// en: '{count} new conflict(s) found'
	String detected({required Object count}) => '${count} new conflict(s) found';

	/// en: 'Accept'
	String get accept => 'Accept';

	/// en: 'Dismiss'
	String get dismiss => 'Dismiss';

	/// en: 'Conflict accepted — remember to apply the fix'
	String get accepted => 'Conflict accepted — remember to apply the fix';

	/// en: 'Conflict dismissed'
	String get dismissed => 'Conflict dismissed';

	/// en: 'Fact deleted and conflict accepted'
	String get deletedFact => 'Fact deleted and conflict accepted';

	/// en: 'Fix:'
	String get quickActions => 'Fix:';

	/// en: 'Delete fact'
	String get deleteFact => 'Delete fact';

	/// en: 'Delete {side}: {ref}'
	String deleteFactSide({required Object side, required Object ref}) => 'Delete ${side}: ${ref}';

	late final TranslationsWebConflictsConfirmDeleteEn confirmDelete = TranslationsWebConflictsConfirmDeleteEn.internal(_root);
	late final TranslationsWebConflictsOpenLayerEn openLayer = TranslationsWebConflictsOpenLayerEn.internal(_root);
	late final TranslationsWebConflictsSeverityEn severity = TranslationsWebConflictsSeverityEn.internal(_root);
}

// Path: web.memoryHealth
class TranslationsWebMemoryHealthEn {
	TranslationsWebMemoryHealthEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory health — last {days} days'
	String title({required Object days}) => 'Memory health — last ${days} days';

	/// en: 'Aggregate signals across both memory subsystems for this project.'
	String get subtitle => 'Aggregate signals across both memory subsystems for this project.';

	/// en: 'Loading health snapshot…'
	String get loading => 'Loading health snapshot…';

	/// en: 'Failed to load health snapshot.'
	String get errorLoading => 'Failed to load health snapshot.';

	/// en: 'Pick a project to see its memory health.'
	String get pickCwd => 'Pick a project to see its memory health.';

	/// en: 'New facts'
	String get newFacts => 'New facts';

	/// en: '{total} stored in total'
	String newFactsHint({required Object total}) => '${total} stored in total';

	/// en: 'Capture fires'
	String get captureFires => 'Capture fires';

	/// en: '{stored} stored · {deduped} deduped'
	String captureFiresHint({required Object stored, required Object deduped}) => '${stored} stored · ${deduped} deduped';

	/// en: 'Journal entries'
	String get newJournal => 'Journal entries';

	/// en: '{total} in total'
	String newJournalHint({required Object total}) => '${total} in total';

	/// en: 'Plan last updated'
	String get planAge => 'Plan last updated';

	/// en: '{count} plan-drift proposal(s) pending'
	String planAgeHint({required Object count}) => '${count} plan-drift proposal(s) pending';

	/// en: 'No plan-drift proposals pending'
	String get planAgeHintNone => 'No plan-drift proposals pending';

	/// en: 'Goal last updated'
	String get goalAge => 'Goal last updated';

	/// en: 'Pending proposals'
	String get pending => 'Pending proposals';

	/// en: 'oldest {days}d old'
	String pendingHint({required Object days}) => 'oldest ${days}d old';

	/// en: 'Top hit · {hits} retrievals'
	String topHit({required Object hits}) => 'Top hit · ${hits} retrievals';

	/// en: '{count} facts older than 7d with zero retrievals — candidates for cleanup.'
	String zeroHit({required Object count}) => '${count} facts older than 7d with zero retrievals — candidates for cleanup.';

	/// en: 'never'
	String get never => 'never';

	/// en: 'today'
	String get today => 'today';

	/// en: '{count} day ago'
	String daysAgo_one({required Object count}) => '${count} day ago';

	/// en: '{count} days ago'
	String daysAgo_other({required Object count}) => '${count} days ago';
}

// Path: web.memoryConfig
class TranslationsWebMemoryConfigEn {
	TranslationsWebMemoryConfigEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory configuration'
	String get title => 'Memory configuration';

	/// en: 'All memory-related knobs in one place: HTTP providers, per-task worker routing, capture triggers, spawn-time injection, and audit costs.'
	String get subtitle => 'All memory-related knobs in one place: HTTP providers, per-task worker routing, capture triggers, spawn-time injection, and audit costs.';

	late final TranslationsWebMemoryConfigSectionsEn sections = TranslationsWebMemoryConfigSectionsEn.internal(_root);
	late final TranslationsWebMemoryConfigSectionHintsEn sectionHints = TranslationsWebMemoryConfigSectionHintsEn.internal(_root);
	late final TranslationsWebMemoryConfigMoveBannerEn moveBanner = TranslationsWebMemoryConfigMoveBannerEn.internal(_root);
}

// Path: web.memoryWorkers
class TranslationsWebMemoryWorkersEn {
	TranslationsWebMemoryWorkersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory workers'
	String get title => 'Memory workers';

	/// en: 'Loading worker config…'
	String get loading => 'Loading worker config…';

	/// en: 'Endpoint not reachable.'
	String get errorTitle => 'Endpoint not reachable.';

	/// en: 'The /api/v1/memory/workers routes are new in M25 — the opendray binary may need a restart to mount them and run migration 0029.'
	String get errorDescription => 'The /api/v1/memory/workers routes are new in M25 — the opendray binary may need a restart to mount them and run migration 0029.';

	/// en: 'Each memory-system LLM touchpoint can be served independently by the local <1>summarizer</1> endpoint (LM Studio / OpenAI-compat) or by spawning a headless <3>Claude / Gemini agent</3> in <5>--print</5> mode. High-quality narrative tasks (gitactivity, transcript) benefit from agent workers; high-frequency tasks (gatekeeper) stay on the local endpoint by design.'
	String get intro => 'Each memory-system LLM touchpoint can be served independently by the local <1>summarizer</1> endpoint (LM Studio / OpenAI-compat) or by spawning a headless <3>Claude / Gemini agent</3> in <5>--print</5> mode. High-quality narrative tasks (gitactivity, transcript) benefit from agent workers; high-frequency tasks (gatekeeper) stay on the local endpoint by design.';

	/// en: 'enabled'
	String get enabledBadge => 'enabled';

	/// en: 'disabled'
	String get disabledBadge => 'disabled';

	/// en: 'summarizer-only'
	String get summarizerOnlyBadge => 'summarizer-only';

	/// en: '{count} calls · 24h'
	String callsCount({required Object count}) => '${count} calls · 24h';

	/// en: 'avg {ms}ms'
	String avgMs({required Object ms}) => 'avg ${ms}ms';

	/// en: '{count} errors'
	String errorsCount({required Object count}) => '${count} errors';

	/// en: 'Worker'
	String get workerLabel => 'Worker';

	/// en: 'Summarizer (HTTP)'
	String get summarizerHttp => 'Summarizer (HTTP)';

	/// en: 'Agent (CLI --print)'
	String get agentCliPrint => 'Agent (CLI --print)';

	/// en: 'Summarizer provider'
	String get summarizerProviderLabel => 'Summarizer provider';

	/// en: 'Registry default'
	String get registryDefault => 'Registry default';

	/// en: 'CLI'
	String get cliLabel => 'CLI';

	/// en: 'Select'
	String get selectPlaceholder => 'Select';

	/// en: 'Claude'
	String get cliClaude => 'Claude';

	/// en: 'Gemini'
	String get cliGemini => 'Gemini';

	/// en: 'Claude account'
	String get claudeAccountLabel => 'Claude account';

	/// en: 'Default'
	String get claudeAccountDefault => 'Default';

	/// en: 'Agent mode spawns a headless CLI per call. Latency rises from <1>~1s</1> (summarizer) to <3>~5-15s</3>; cost shifts from CPU to your Claude/Gemini quota.'
	String get agentWarning => 'Agent mode spawns a headless CLI per call. Latency rises from <1>~1s</1> (summarizer) to <3>~5-15s</3>; cost shifts from CPU to your Claude/Gemini quota.';

	/// en: 'Enabled'
	String get enabledCheckbox => 'Enabled';

	/// en: 'Test'
	String get testButton => 'Test';

	/// en: 'Save'
	String get saveButton => 'Save';

	/// en: 'Recent calls ({count})'
	String recentCalls({required Object count}) => 'Recent calls (${count})';

	/// en: 'when'
	String get tableWhen => 'when';

	/// en: 'worker'
	String get tableWorker => 'worker';

	/// en: 'ms'
	String get tableMs => 'ms';

	/// en: 'ok'
	String get tableOk => 'ok';

	/// en: '{label} updated'
	String savedToast({required Object label}) => '${label} updated';

	/// en: 'Save failed'
	String get saveFailedToast => 'Save failed';

	/// en: '{label} OK — {ms}ms'
	String testOkToast({required Object label, required Object ms}) => '${label} OK — ${ms}ms';

	/// en: '{label} failed'
	String testFailedToast({required Object label}) => '${label} failed';

	/// en: 'Test call failed'
	String get testCallFailedToast => 'Test call failed';

	/// en: 'unknown error'
	String get unknownError => 'unknown error';

	late final TranslationsWebMemoryWorkersTasksEn tasks = TranslationsWebMemoryWorkersTasksEn.internal(_root);
}

// Path: web.cleanupInbox
class TranslationsWebCleanupInboxEn {
	TranslationsWebCleanupInboxEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Cleanup inbox empty'
	String get emptyTitle => 'Cleanup inbox empty';

	/// en: 'No pending cleanup decisions across any project. The LLM librarian either hasn't run yet for the eligible memories, or it found everything load-bearing.'
	String get emptyDescription => 'No pending cleanup decisions across any project. The LLM librarian either hasn\'t run yet for the eligible memories, or it found everything load-bearing.';

	/// en: 'Cleanup inbox'
	String get title => 'Cleanup inbox';

	/// en: 'Cross-project pending decisions from the LLM memory librarian. Approving stale → deletes, approving duplicate → merges, approving keep → freezes the entry from being re-judged for a while.'
	String get subtitle => 'Cross-project pending decisions from the LLM memory librarian. Approving stale → deletes, approving duplicate → merges, approving keep → freezes the entry from being re-judged for a while.';

	/// en: '(global)'
	String get globalScope => '(global)';

	/// en: 'orphan'
	String get orphanBadge => 'orphan';

	/// en: 'Truncated scope_key (old mirror import). Not a navigable project.'
	String get orphanTitle => 'Truncated scope_key (old mirror import). Not a navigable project.';

	/// en: 'Open project'
	String get openProject => 'Open project';

	/// en: '→ merge into'
	String get mergeIntoPrefix => '→ merge into';

	/// en: 'Reason:'
	String get reasonPrefix => 'Reason:';

	/// en: 'Execute'
	String get executeButton => 'Execute';

	/// en: 'Confirm keep'
	String get confirmKeepButton => 'Confirm keep';

	/// en: 'Reject'
	String get rejectButton => 'Reject';

	/// en: 'Kept'
	String get approvedKeptToast => 'Kept';

	/// en: '{verdict} executed'
	String approvedExecutedToast({required Object verdict}) => '${verdict} executed';

	/// en: 'Approve failed'
	String get approveFailedToast => 'Approve failed';

	/// en: 'Rejected — memory kept'
	String get rejectedToast => 'Rejected — memory kept';

	/// en: 'Reject failed'
	String get rejectFailedToast => 'Reject failed';
}

// Path: web.project
class TranslationsWebProjectEn {
	TranslationsWebProjectEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebProjectPickerEn picker = TranslationsWebProjectPickerEn.internal(_root);

	/// en: 'Pick a project to manage its memory.'
	String get noCwd => 'Pick a project to manage its memory.';

	late final TranslationsWebProjectHeaderEn header = TranslationsWebProjectHeaderEn.internal(_root);
	late final TranslationsWebProjectTabsEn tabs = TranslationsWebProjectTabsEn.internal(_root);
	late final TranslationsWebProjectDocLabelEn docLabel = TranslationsWebProjectDocLabelEn.internal(_root);
	late final TranslationsWebProjectVerdictLabelEn verdictLabel = TranslationsWebProjectVerdictLabelEn.internal(_root);
	late final TranslationsWebProjectEditorEn editor = TranslationsWebProjectEditorEn.internal(_root);
	late final TranslationsWebProjectReadonlyEn readonly = TranslationsWebProjectReadonlyEn.internal(_root);
	late final TranslationsWebProjectJournalEn journal = TranslationsWebProjectJournalEn.internal(_root);
	late final TranslationsWebProjectInboxEn inbox = TranslationsWebProjectInboxEn.internal(_root);
	late final TranslationsWebProjectCleanupEn cleanup = TranslationsWebProjectCleanupEn.internal(_root);
	late final TranslationsWebProjectResetEn reset = TranslationsWebProjectResetEn.internal(_root);
}

// Path: web.memoryInspector
class TranslationsWebMemoryInspectorEn {
	TranslationsWebMemoryInspectorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebMemoryInspectorStatusEn status = TranslationsWebMemoryInspectorStatusEn.internal(_root);

	/// en: 'This is the embedder the gateway is currently using for every <1>memory_search</1> / <3>memory_store</3> call. If this doesn't match the configuration above, you have unsaved changes — click Save then Restart server to apply.'
	String get statusBody => 'This is the embedder the gateway is currently using for every <1>memory_search</1> / <3>memory_store</3> call. If this doesn\'t match the configuration above, you have unsaved changes — click Save then Restart server to apply.';

	late final TranslationsWebMemoryInspectorScopeEn scope = TranslationsWebMemoryInspectorScopeEn.internal(_root);
	late final TranslationsWebMemoryInspectorSearchEn search = TranslationsWebMemoryInspectorSearchEn.internal(_root);
	late final TranslationsWebMemoryInspectorRecordsEn records = TranslationsWebMemoryInspectorRecordsEn.internal(_root);
	late final TranslationsWebMemoryInspectorRowEn row = TranslationsWebMemoryInspectorRowEn.internal(_root);
	late final TranslationsWebMemoryInspectorToastsEn toasts = TranslationsWebMemoryInspectorToastsEn.internal(_root);
	late final TranslationsWebMemoryInspectorBulkDeleteEn bulkDelete = TranslationsWebMemoryInspectorBulkDeleteEn.internal(_root);
	late final TranslationsWebMemoryInspectorAddMemEn addMem = TranslationsWebMemoryInspectorAddMemEn.internal(_root);
	late final TranslationsWebMemoryInspectorPickerEn picker = TranslationsWebMemoryInspectorPickerEn.internal(_root);
	late final TranslationsWebMemoryInspectorMigrationBannerEn migrationBanner = TranslationsWebMemoryInspectorMigrationBannerEn.internal(_root);
	late final TranslationsWebMemoryInspectorReembedEn reembed = TranslationsWebMemoryInspectorReembedEn.internal(_root);
}

// Path: web.notes
class TranslationsWebNotesEn {
	TranslationsWebNotesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Notes'
	String get title => 'Notes';

	late final TranslationsWebNotesHeaderEn header = TranslationsWebNotesHeaderEn.internal(_root);
	late final TranslationsWebNotesLeftEn left = TranslationsWebNotesLeftEn.internal(_root);
	late final TranslationsWebNotesTagsEn tags = TranslationsWebNotesTagsEn.internal(_root);
	late final TranslationsWebNotesTreeEn tree = TranslationsWebNotesTreeEn.internal(_root);
	late final TranslationsWebNotesOutlineEn outline = TranslationsWebNotesOutlineEn.internal(_root);
	late final TranslationsWebNotesNewNoteEn newNote = TranslationsWebNotesNewNoteEn.internal(_root);
	late final TranslationsWebNotesEmptyEn empty = TranslationsWebNotesEmptyEn.internal(_root);
	late final TranslationsWebNotesPickerEn picker = TranslationsWebNotesPickerEn.internal(_root);
	late final TranslationsWebNotesVaultSyncEn vaultSync = TranslationsWebNotesVaultSyncEn.internal(_root);
	late final TranslationsWebNotesSyncBadgeEn syncBadge = TranslationsWebNotesSyncBadgeEn.internal(_root);
}

// Path: web.activity
class TranslationsWebActivityEn {
	TranslationsWebActivityEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Activity'
	String get title => 'Activity';

	/// en: 'Per-call audit of API requests made by registered integrations. Includes both inbound calls (a third-party app calling opendray with its API key) and outbound proxied calls (admin → opendray proxy → integration). Calls made directly by this admin UI are not recorded.'
	String get subtitle => 'Per-call audit of API requests made by registered integrations. Includes both inbound calls (a third-party app calling opendray with its API key) and outbound proxied calls (admin → opendray proxy → integration). Calls made directly by this admin UI are not recorded.';

	/// en: 'Refresh'
	String get refresh => 'Refresh';

	/// en: 'Refresh'
	String get refreshTooltip => 'Refresh';

	late final TranslationsWebActivityFiltersEn filters = TranslationsWebActivityFiltersEn.internal(_root);

	/// en: '{count} call'
	String callsCount_one({required Object count}) => '${count} call';

	/// en: '{count} calls'
	String callsCount_other({required Object count}) => '${count} calls';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	late final TranslationsWebActivityTableEn table = TranslationsWebActivityTableEn.internal(_root);
	late final TranslationsWebActivityEmptyEn empty = TranslationsWebActivityEmptyEn.internal(_root);
	late final TranslationsWebActivityEventsEn events = TranslationsWebActivityEventsEn.internal(_root);
}

// Path: web.providers
class TranslationsWebProvidersEn {
	TranslationsWebProvidersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebProvidersListEn list = TranslationsWebProvidersListEn.internal(_root);
	late final TranslationsWebProvidersDetailEn detail = TranslationsWebProvidersDetailEn.internal(_root);
	late final TranslationsWebProvidersConfigFormEn configForm = TranslationsWebProvidersConfigFormEn.internal(_root);
	late final TranslationsWebProvidersClaudeAccountsEn claudeAccounts = TranslationsWebProvidersClaudeAccountsEn.internal(_root);
}

// Path: web.channels
class TranslationsWebChannelsEn {
	TranslationsWebChannelsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Channels'
	String get title => 'Channels';

	/// en: 'Bidirectional messaging integrations. Outbound notifications are filtered by each channel's <1>notify_on</1>.'
	String get subtitle => 'Bidirectional messaging integrations. Outbound notifications are filtered by each channel\'s <1>notify_on</1>.';

	/// en: 'New channel'
	String get newButton => 'New channel';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	late final TranslationsWebChannelsEmptyEn empty = TranslationsWebChannelsEmptyEn.internal(_root);
	late final TranslationsWebChannelsCardEn card = TranslationsWebChannelsCardEn.internal(_root);
	late final TranslationsWebChannelsToastsEn toasts = TranslationsWebChannelsToastsEn.internal(_root);
	late final TranslationsWebChannelsDialogEn dialog = TranslationsWebChannelsDialogEn.internal(_root);
	late final TranslationsWebChannelsNotificationsEn notifications = TranslationsWebChannelsNotificationsEn.internal(_root);
	late final TranslationsWebChannelsBridgeEn bridge = TranslationsWebChannelsBridgeEn.internal(_root);
	late final TranslationsWebChannelsSetupEn setup = TranslationsWebChannelsSetupEn.internal(_root);
}

// Path: web.integrations
class TranslationsWebIntegrationsEn {
	TranslationsWebIntegrationsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Integrations'
	String get title => 'Integrations';

	/// en: 'External apps that consume opendray. Reverse-proxy through <1>/api/v1/proxy/&lt;prefix&gt;/…</1> and subscribe to events via the WS endpoint.'
	String get subtitle => 'External apps that consume opendray. Reverse-proxy through <1>/api/v1/proxy/&lt;prefix&gt;/…</1> and subscribe to events via the WS endpoint.';

	/// en: 'Register'
	String get register => 'Register';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	late final TranslationsWebIntegrationsTabsEn tabs = TranslationsWebIntegrationsTabsEn.internal(_root);
	late final TranslationsWebIntegrationsEmptyEn empty = TranslationsWebIntegrationsEmptyEn.internal(_root);

	/// en: 'System (managed by opendray)'
	String get groupSystem => 'System (managed by opendray)';

	/// en: 'Operator-registered'
	String get groupOperator => 'Operator-registered';

	late final TranslationsWebIntegrationsCardEn card = TranslationsWebIntegrationsCardEn.internal(_root);
	late final TranslationsWebIntegrationsRegisterDialogEn register_dialog = TranslationsWebIntegrationsRegisterDialogEn.internal(_root);
	late final TranslationsWebIntegrationsRevealEn reveal = TranslationsWebIntegrationsRevealEn.internal(_root);
	late final TranslationsWebIntegrationsEditDialogEn edit_dialog = TranslationsWebIntegrationsEditDialogEn.internal(_root);
	late final TranslationsWebIntegrationsProxyEn proxy = TranslationsWebIntegrationsProxyEn.internal(_root);
}

// Path: web.plugins
class TranslationsWebPluginsEn {
	TranslationsWebPluginsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Inspector plugins'
	String get title => 'Inspector plugins';

	/// en: 'Configure data sources surfaced in the right-hand Inspector panel when a session is open. Each plugin is admin-only and shared across all sessions. Click a section header to collapse it.'
	String get subtitle => 'Configure data sources surfaced in the right-hand Inspector panel when a session is open. Each plugin is admin-only and shared across all sessions. Click a section header to collapse it.';

	late final TranslationsWebPluginsCommonEn common = TranslationsWebPluginsCommonEn.internal(_root);
	late final TranslationsWebPluginsMcpEn mcp = TranslationsWebPluginsMcpEn.internal(_root);
	late final TranslationsWebPluginsMcpSecretsEn mcpSecrets = TranslationsWebPluginsMcpSecretsEn.internal(_root);
	late final TranslationsWebPluginsSkillsEn skills = TranslationsWebPluginsSkillsEn.internal(_root);
	late final TranslationsWebPluginsCustomTasksEn customTasks = TranslationsWebPluginsCustomTasksEn.internal(_root);
	late final TranslationsWebPluginsGitHostsEn gitHosts = TranslationsWebPluginsGitHostsEn.internal(_root);
}

// Path: web.backups
class TranslationsWebBackupsEn {
	TranslationsWebBackupsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backups'
	String get title => 'Backups';

	/// en: 'Encrypted PostgreSQL dumps written to a pluggable target. Configure schedules + retention, or trigger one-off backups for a quick safety net.'
	String get subtitle => 'Encrypted PostgreSQL dumps written to a pluggable target. Configure schedules + retention, or trigger one-off backups for a quick safety net.';

	/// en: 'Export data'
	String get exportData => 'Export data';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Failed to load backup status'
	String get loadStatusFailedToast => 'Failed to load backup status';

	late final TranslationsWebBackupsTabsEn tabs = TranslationsWebBackupsTabsEn.internal(_root);
	late final TranslationsWebBackupsInventoryEn inventory = TranslationsWebBackupsInventoryEn.internal(_root);
	late final TranslationsWebBackupsRestartEn restart = TranslationsWebBackupsRestartEn.internal(_root);
	late final TranslationsWebBackupsSetupEn setup = TranslationsWebBackupsSetupEn.internal(_root);
	late final TranslationsWebBackupsGeneratedEn generated = TranslationsWebBackupsGeneratedEn.internal(_root);
	late final TranslationsWebBackupsStatusEn status = TranslationsWebBackupsStatusEn.internal(_root);
	late final TranslationsWebBackupsBackupsTabEn backupsTab = TranslationsWebBackupsBackupsTabEn.internal(_root);
	late final TranslationsWebBackupsRestoreEn restore = TranslationsWebBackupsRestoreEn.internal(_root);
	late final TranslationsWebBackupsSchedulesTabEn schedulesTab = TranslationsWebBackupsSchedulesTabEn.internal(_root);
	late final TranslationsWebBackupsNewScheduleEn newSchedule = TranslationsWebBackupsNewScheduleEn.internal(_root);
	late final TranslationsWebBackupsTargetsTabEn targetsTab = TranslationsWebBackupsTargetsTabEn.internal(_root);
	late final TranslationsWebBackupsTargetEditorEn targetEditor = TranslationsWebBackupsTargetEditorEn.internal(_root);
}

// Path: web.serverSettings
class TranslationsWebServerSettingsEn {
	TranslationsWebServerSettingsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebServerSettingsSectionsEn sections = TranslationsWebServerSettingsSectionsEn.internal(_root);

	/// en: 'Loading server settings…'
	String get loading => 'Loading server settings…';

	/// en: 'Failed to load: {message}'
	String loadFailed({required Object message}) => 'Failed to load: ${message}';

	/// en: 'opendray was started without a -config flag. Settings are loaded from environment variables only and cannot be edited here.'
	String get noConfigFlag => 'opendray was started without a -config flag. Settings are loaded from environment variables only and cannot be edited here.';

	/// en: 'Reset'
	String get resetButton => 'Reset';

	/// en: 'Discard unsaved changes in this section'
	String get resetButtonTitle => 'Discard unsaved changes in this section';

	/// en: 'Reset "{section}" to last-saved values?'
	String resetConfirm({required Object section}) => 'Reset "${section}" to last-saved values?';

	/// en: 'restart required'
	String get badgeRestartRequired => 'restart required';

	/// en: 'unsaved'
	String get badgeUnsaved => 'unsaved';

	/// en: 'Save changes'
	String get saveButton => 'Save changes';

	/// en: 'Settings saved'
	String get saveToastTitle => 'Settings saved';

	/// en: 'Click Restart to apply.'
	String get saveToastDesc => 'Click Restart to apply.';

	/// en: 'Save failed'
	String get saveErrorTitle => 'Save failed';

	/// en: 'You changed listen address / admin user / admin password. After restart you may need to re-authenticate or use the new address. Continue?'
	String get dangerousConfirm => 'You changed listen address / admin user / admin password. After restart you may need to re-authenticate or use the new address. Continue?';

	/// en: 'You have unsaved changes'
	String get unsavedHint => 'You have unsaved changes';

	/// en: 'All changes saved'
	String get savedHint => 'All changes saved';

	/// en: 'Filter fields…'
	String get searchPlaceholder => 'Filter fields…';

	late final TranslationsWebServerSettingsRestartEn restart = TranslationsWebServerSettingsRestartEn.internal(_root);
	late final TranslationsWebServerSettingsFormGroupsEn formGroups = TranslationsWebServerSettingsFormGroupsEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsEn fields = TranslationsWebServerSettingsFieldsEn.internal(_root);
	late final TranslationsWebServerSettingsLiveTailEn liveTail = TranslationsWebServerSettingsLiveTailEn.internal(_root);
	late final TranslationsWebServerSettingsMemoryInspectorCardEn memoryInspectorCard = TranslationsWebServerSettingsMemoryInspectorCardEn.internal(_root);

	/// en: 'Requires the binary to be compiled with <1>-tags local_onnx</1>. The standard build returns a clear stub error when this backend is selected. See <3>Memory → Local ONNX</3> tutorial for setup steps.'
	String get localOnnxBanner => 'Requires the binary to be compiled with <1>-tags local_onnx</1>. The standard build returns a clear stub error when this backend is selected. See <3>Memory → Local ONNX</3> tutorial for setup steps.';

	late final TranslationsWebServerSettingsStringListEn stringList = TranslationsWebServerSettingsStringListEn.internal(_root);
	late final TranslationsWebServerSettingsHttpHelpersEn httpHelpers = TranslationsWebServerSettingsHttpHelpersEn.internal(_root);
	late final TranslationsWebServerSettingsProbeEn probe = TranslationsWebServerSettingsProbeEn.internal(_root);
	late final TranslationsWebServerSettingsBackupEn backup = TranslationsWebServerSettingsBackupEn.internal(_root);
	late final TranslationsWebServerSettingsTargetRowEn targetRow = TranslationsWebServerSettingsTargetRowEn.internal(_root);
}

// Path: web.settings
class TranslationsWebSettingsEn {
	TranslationsWebSettingsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Settings'
	String get title => 'Settings';

	/// en: 'Workspace, account, and gateway config.'
	String get subtitle => 'Workspace, account, and gateway config.';

	late final TranslationsWebSettingsGroupsEn groups = TranslationsWebSettingsGroupsEn.internal(_root);
	late final TranslationsWebSettingsItemsEn items = TranslationsWebSettingsItemsEn.internal(_root);
	late final TranslationsWebSettingsHealthEn health = TranslationsWebSettingsHealthEn.internal(_root);
	late final TranslationsWebSettingsBreadcrumbEn breadcrumb = TranslationsWebSettingsBreadcrumbEn.internal(_root);
	late final TranslationsWebSettingsAppearanceEn appearance = TranslationsWebSettingsAppearanceEn.internal(_root);
	late final TranslationsWebSettingsFontEn font = TranslationsWebSettingsFontEn.internal(_root);
	late final TranslationsWebSettingsAccountEn account = TranslationsWebSettingsAccountEn.internal(_root);
	late final TranslationsWebSettingsChangeCredentialsEn changeCredentials = TranslationsWebSettingsChangeCredentialsEn.internal(_root);
	late final TranslationsWebSettingsSystemEn system = TranslationsWebSettingsSystemEn.internal(_root);
	late final TranslationsWebSettingsAboutEn about = TranslationsWebSettingsAboutEn.internal(_root);
}

// Path: web.logViewer
class TranslationsWebLogViewerEn {
	TranslationsWebLogViewerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Filter…'
	String get filterPlaceholder => 'Filter…';

	/// en: 'Debug count'
	String get debugTooltip => 'Debug count';

	/// en: 'Info count'
	String get infoTooltip => 'Info count';

	/// en: 'Warn count'
	String get warnTooltip => 'Warn count';

	/// en: 'Error count'
	String get errorTooltip => 'Error count';

	/// en: 'Streaming'
	String get streaming => 'Streaming';

	/// en: 'Disconnected'
	String get disconnected => 'Disconnected';

	/// en: 'live'
	String get live => 'live';

	/// en: 'offline'
	String get offline => 'offline';

	/// en: 'Pause auto-scroll'
	String get pauseTooltip => 'Pause auto-scroll';

	/// en: 'Resume auto-scroll'
	String get resumeTooltip => 'Resume auto-scroll';

	/// en: 'Clear local view (server ring untouched)'
	String get clearTooltip => 'Clear local view (server ring untouched)';

	/// en: 'Download full ring as .log file'
	String get downloadTooltip => 'Download full ring as .log file';

	/// en: 'Waiting for log records…'
	String get emptyWaiting => 'Waiting for log records…';

	/// en: 'No records match "{query}"'
	String emptyFiltered({required Object query}) => 'No records match "${query}"';
}

// Path: web.pathInput
class TranslationsWebPathInputEn {
	TranslationsWebPathInputEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Test'
	String get testButton => 'Test';

	/// en: 'Resolve and check this path'
	String get testTooltip => 'Resolve and check this path';

	/// en: 'not found ·'
	String get notFound => 'not found ·';

	/// en: 'children'
	String get childrenSuffix => 'children';

	/// en: '· expected directory'
	String get expectedDirectory => '· expected directory';
}

// Path: web.memoryAmbient
class TranslationsWebMemoryAmbientEn {
	TranslationsWebMemoryAmbientEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebMemoryAmbientHeaderEn header = TranslationsWebMemoryAmbientHeaderEn.internal(_root);

	/// en: 'Loading…'
	String get loading => 'Loading…';

	late final TranslationsWebMemoryAmbientProvidersEn providers = TranslationsWebMemoryAmbientProvidersEn.internal(_root);
	late final TranslationsWebMemoryAmbientRulesEn rules = TranslationsWebMemoryAmbientRulesEn.internal(_root);
	late final TranslationsWebMemoryAmbientProfilesEn profiles = TranslationsWebMemoryAmbientProfilesEn.internal(_root);
	late final TranslationsWebMemoryAmbientCostEn cost = TranslationsWebMemoryAmbientCostEn.internal(_root);
}

// Path: web.noteEditor
class TranslationsWebNoteEditorEn {
	TranslationsWebNoteEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Source'
	String get source => 'Source';

	/// en: 'Preview'
	String get preview => 'Preview';

	/// en: 'tag #{tag}'
	String tagTitle({required Object tag}) => 'tag #${tag}';

	/// en: 'Empty note. Switch to Source to start writing.'
	String get emptyNote => 'Empty note. Switch to Source to start writing.';

	/// en: 'Save failed'
	String get saveFailedToast => 'Save failed';

	late final TranslationsWebNoteEditorStatusEn status = TranslationsWebNoteEditorStatusEn.internal(_root);
}

// Path: web.export
class TranslationsWebExportEn {
	TranslationsWebExportEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Export data'
	String get title => 'Export data';

	/// en: 'Take a one-shot zip bundle of selected logical entities. Bundles are kept on the server for 24 hours, then automatically reaped.'
	String get subtitle => 'Take a one-shot zip bundle of selected logical entities. Bundles are kept on the server for 24 hours, then automatically reaped.';

	/// en: '← Backups'
	String get backToBackups => '← Backups';

	late final TranslationsWebExportSectionsEn sections = TranslationsWebExportSectionsEn.internal(_root);
	late final TranslationsWebExportFormEn form = TranslationsWebExportFormEn.internal(_root);
	late final TranslationsWebExportHistoryEn history = TranslationsWebExportHistoryEn.internal(_root);
	late final TranslationsWebExportImportEn import = TranslationsWebExportImportEn.internal(_root);
	late final TranslationsWebExportImportsEn imports = TranslationsWebExportImportsEn.internal(_root);
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
	late final TranslationsMoreItemsDataExportEn dataExport = TranslationsMoreItemsDataExportEn.internal(_root);
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

	/// en: 'Enable'
	String get enable => 'Enable';

	/// en: 'Disable'
	String get disable => 'Disable';

	/// en: 'Delete'
	String get deleteLabel => 'Delete';

	/// en: 'Removes the account and its stored OAuth token. Sessions already using this account stay running but reauth will fail.'
	String get deleteBody => 'Removes the account and its stored OAuth token. Sessions already using this account stay running but reauth will fail.';

	/// en: 'Deleted {name}.'
	String deletedSnack({required Object name}) => 'Deleted ${name}.';

	/// en: 'Already in sync — gateway has no new accounts.'
	String get importSyncedSnack => 'Already in sync — gateway has no new accounts.';

	/// en: 'Imported {n} account.'
	String importedSnackOne({required Object n}) => 'Imported ${n} account.';

	/// en: 'Imported {n} accounts.'
	String importedSnackOther({required Object n}) => 'Imported ${n} accounts.';

	/// en: 'Syncing…'
	String get importing => 'Syncing…';

	/// en: 'Import local'
	String get importLocal => 'Import local';

	/// en: 'Adding a new account is gateway-host only.'
	String get addHint => 'Adding a new account is gateway-host only.';

	/// en: 'The new directory shows up here automatically. See the docs for OAuth flow steps.'
	String get addBody => 'The new directory shows up here automatically. See the docs for OAuth flow steps.';

	/// en: 'Failed to load accounts: {error}'
	String loadFailed({required Object error}) => 'Failed to load accounts: ${error}';

	/// en: 'Sessions spawned with the Claude provider pick from these accounts (or fall back to env).'
	String get intro => 'Sessions spawned with the Claude provider pick from these accounts (or fall back to env).';

	/// en: '{name} enabled.'
	String enabledSnack({required Object name}) => '${name} enabled.';

	/// en: '{name} disabled.'
	String disabledSnack({required Object name}) => '${name} disabled.';

	/// en: 'Renamed to {name}.'
	String renamedSnack({required Object name}) => 'Renamed to ${name}.';
}

// Path: integrations.form
class TranslationsIntegrationsFormEn {
	TranslationsIntegrationsFormEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Name, base URL, and route prefix are required.'
	String get validateRequired => 'Name, base URL, and route prefix are required.';

	/// en: 'Name'
	String get fieldName => 'Name';

	/// en: 'My Bot'
	String get fieldNameHint => 'My Bot';

	/// en: 'Base URL'
	String get fieldBaseUrl => 'Base URL';

	/// en: 'Route prefix'
	String get fieldRoutePrefix => 'Route prefix';

	/// en: 'Reachable as /api/v1/<prefix>/...'
	String get routePrefixHelper => 'Reachable as /api/v1/<prefix>/...';

	/// en: 'Scopes (optional)'
	String get fieldScopes => 'Scopes (optional)';

	/// en: 'Comma-separated. Empty = server defaults.'
	String get scopesHelper => 'Comma-separated. Empty = server defaults.';

	/// en: 'Version (optional)'
	String get fieldVersion => 'Version (optional)';

	/// en: 'Base URL is required.'
	String get validateBaseUrl => 'Base URL is required.';

	/// en: 'Scopes'
	String get editFieldScopes => 'Scopes';

	/// en: 'Comma-separated.'
	String get editScopesHelper => 'Comma-separated.';

	/// en: 'Version'
	String get editFieldVersion => 'Version';

	/// en: 'You won't see this key again.'
	String get apiKeyWarn => 'You won\'t see this key again.';

	/// en: 'Copied'
	String get copyCopied => 'Copied';

	/// en: 'Copy'
	String get copyCopy => 'Copy';
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
	late final TranslationsMemoryWorkersTasksPlanDriftEn planDrift = TranslationsMemoryWorkersTasksPlanDriftEn.internal(_root);
	late final TranslationsMemoryWorkersTasksConflictDetectorEn conflictDetector = TranslationsMemoryWorkersTasksConflictDetectorEn.internal(_root);
	late final TranslationsMemoryWorkersTasksCaptureEn capture = TranslationsMemoryWorkersTasksCaptureEn.internal(_root);
}

// Path: project.health
class TranslationsProjectHealthEn {
	TranslationsProjectHealthEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory health — last {days} days'
	String title({required Object days}) => 'Memory health — last ${days} days';

	/// en: 'Aggregate signals across both memory subsystems for this project.'
	String get subtitle => 'Aggregate signals across both memory subsystems for this project.';

	/// en: 'New facts'
	String get newFacts => 'New facts';

	/// en: '{total} stored in total'
	String newFactsHint({required Object total}) => '${total} stored in total';

	/// en: 'Capture fires'
	String get captureFires => 'Capture fires';

	/// en: '{stored} stored · {deduped} deduped'
	String captureFiresHint({required Object stored, required Object deduped}) => '${stored} stored · ${deduped} deduped';

	/// en: 'Journal entries'
	String get newJournal => 'Journal entries';

	/// en: '{total} in total'
	String newJournalHint({required Object total}) => '${total} in total';

	/// en: 'Plan last updated'
	String get planAge => 'Plan last updated';

	/// en: '{count} plan-drift proposal(s) pending'
	String planAgeHint({required Object count}) => '${count} plan-drift proposal(s) pending';

	/// en: 'No plan-drift proposals pending'
	String get planAgeHintNone => 'No plan-drift proposals pending';

	/// en: 'Goal last updated'
	String get goalAge => 'Goal last updated';

	/// en: 'Pending proposals'
	String get pending => 'Pending proposals';

	/// en: 'oldest {days}d old'
	String pendingHint({required Object days}) => 'oldest ${days}d old';

	/// en: 'Top hit · {hits} retrievals'
	String topHit({required Object hits}) => 'Top hit · ${hits} retrievals';

	/// en: '{count} facts older than 7d with zero retrievals — candidates for cleanup.'
	String zeroHit({required Object count}) => '${count} facts older than 7d with zero retrievals — candidates for cleanup.';

	/// en: 'never'
	String get never => 'never';

	/// en: 'today'
	String get today => 'today';

	/// en: '{count}d ago'
	String daysAgo({required Object count}) => '${count}d ago';
}

// Path: project.conflicts
class TranslationsProjectConflictsEn {
	TranslationsProjectConflictsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Contradictions the daily detector found between facts, plan, goal, and journal entries.'
	String get subtitle => 'Contradictions the daily detector found between facts, plan, goal, and journal entries.';

	/// en: 'No pending conflicts. Tap Detect now for an on-demand sweep.'
	String get empty => 'No pending conflicts. Tap Detect now for an on-demand sweep.';

	/// en: 'Detect now'
	String get detectNow => 'Detect now';

	/// en: '{count} new conflict(s) found'
	String detected({required Object count}) => '${count} new conflict(s) found';

	/// en: 'Accept'
	String get accept => 'Accept';

	/// en: 'Dismiss'
	String get dismiss => 'Dismiss';

	/// en: 'Delete fact {side}'
	String deleteFact({required Object side}) => 'Delete fact ${side}';

	/// en: 'Delete fact {side}?'
	String deleteConfirmTitle({required Object side}) => 'Delete fact ${side}?';

	/// en: 'This permanently removes the fact and accepts the conflict. The other side stays as the surviving claim.'
	String get deleteConfirmBody => 'This permanently removes the fact and accepts the conflict. The other side stays as the surviving claim.';

	/// en: 'Will delete (side {side}):'
	String deleteWillDelete({required Object side}) => 'Will delete (side ${side}):';

	/// en: 'Will keep (side {side}):'
	String deleteWillKeep({required Object side}) => 'Will keep (side ${side}):';

	/// en: '({layer} entry — open the corresponding tab to inspect)'
	String deleteNonFactOther({required Object layer}) => '(${layer} entry — open the corresponding tab to inspect)';

	/// en: 'Loading fact text…'
	String get deleteLoading => 'Loading fact text…';

	/// en: 'Delete {side}'
	String deleteFactLabel({required Object side}) => 'Delete ${side}';

	/// en: 'Fact deleted and conflict accepted'
	String get deletedFact => 'Fact deleted and conflict accepted';

	/// en: 'Open plan editor'
	String get openPlanEditor => 'Open plan editor';

	/// en: 'Open goal editor'
	String get openGoalEditor => 'Open goal editor';

	late final TranslationsProjectConflictsSeverityEn severity = TranslationsProjectConflictsSeverityEn.internal(_root);
}

// Path: project.journalPrune
class TranslationsProjectJournalPruneEn {
	TranslationsProjectJournalPruneEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Prune stale journal entries'
	String get title => 'Prune stale journal entries';

	/// en: 'Older than {days} days, no pending conflicts.'
	String subtitle({required Object days}) => 'Older than ${days} days, no pending conflicts.';

	/// en: 'Older than (days):'
	String get daysLabel => 'Older than (days):';

	/// en: 'Nothing stale to prune.'
	String get empty => 'Nothing stale to prune.';

	/// en: 'Select all'
	String get selectAll => 'Select all';

	/// en: 'Deselect all'
	String get deselectAll => 'Deselect all';

	/// en: 'Delete ({count})'
	String deleteSelected({required Object count}) => 'Delete (${count})';

	/// en: '{count} entry/entries deleted'
	String deleted({required Object count}) => '${count} entry/entries deleted';
}

// Path: backups.kv
class TranslationsBackupsKvEn {
	TranslationsBackupsKvEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'Target'
	String get target => 'Target';

	/// en: 'Triggered by'
	String get triggeredBy => 'Triggered by';

	/// en: 'Started'
	String get started => 'Started';

	/// en: 'Finished'
	String get finished => 'Finished';

	/// en: 'Size'
	String get size => 'Size';

	/// en: 'Encrypted'
	String get encrypted => 'Encrypted';

	/// en: 'Target path'
	String get targetPath => 'Target path';

	/// en: 'Error'
	String get error => 'Error';

	/// en: 'yes'
	String get yes => 'yes';

	/// en: 'no'
	String get no => 'no';
}

// Path: backups.emptyMissingDeps
class TranslationsBackupsEmptyMissingDepsEn {
	TranslationsBackupsEmptyMissingDepsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backups can't run yet'
	String get headline => 'Backups can\'t run yet';

	/// en: 'Install postgresql-client and restart opendray.'
	String get body => 'Install postgresql-client and restart opendray.';
}

// Path: backups.emptyNoTargets
class TranslationsBackupsEmptyNoTargetsEn {
	TranslationsBackupsEmptyNoTargetsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No backup targets configured'
	String get headline => 'No backup targets configured';

	/// en: 'Open the More menu → Targets to add a destination (local / S3 / SMB / SFTP / WebDAV / rclone). Then come back and tap "Run now".'
	String get body => 'Open the More menu → Targets to add a destination (local / S3 / SMB / SFTP / WebDAV / rclone). Then come back and tap "Run now".';
}

// Path: backups.emptyNoBackups
class TranslationsBackupsEmptyNoBackupsEn {
	TranslationsBackupsEmptyNoBackupsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No backups yet'
	String get headline => 'No backups yet';

	/// en: 'Tap "Run now" to take a fresh snapshot, or open Schedules to set up recurring runs.'
	String get body => 'Tap "Run now" to take a fresh snapshot, or open Schedules to set up recurring runs.';
}

// Path: backups.wizard
class TranslationsBackupsWizardEn {
	TranslationsBackupsWizardEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Set up backups'
	String get title => 'Set up backups';

	/// en: 'Choose a master passphrase. opendray uses it to encrypt every backup blob with AES-256-GCM. Lose the passphrase and you lose the data — there is no recovery.'
	String get intro => 'Choose a master passphrase. opendray uses it to encrypt every backup blob with AES-256-GCM. Lose the passphrase and you lose the data — there is no recovery.';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Generate and save'
	String get generateAndSave => 'Generate and save';

	/// en: 'Save passphrase'
	String get savePassphrase => 'Save passphrase';

	/// en: 'Server generates a cryptographically random passphrase, you copy it to a password manager, then commit.'
	String get generateHint => 'Server generates a cryptographically random passphrase, you copy it to a password manager, then commit.';

	/// en: 'Recommended: 40+ chars from a password manager'
	String get helperRecommended => 'Recommended: 40+ chars from a password manager';

	/// en: 'Save this passphrase NOW'
	String get saveNowHeader => 'Save this passphrase NOW';

	/// en: 'This is shown ONCE. It will not be retrievable from opendray afterwards.'
	String get saveNowBody => 'This is shown ONCE. It will not be retrievable from opendray afterwards.';
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

// Path: backups.restore
class TranslationsBackupsRestoreEn {
	TranslationsBackupsRestoreEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Restore from bundle'
	String get title => 'Restore from bundle';

	/// en: 'Replay an encrypted .tar.gz.enc bundle into a Postgres database. The bundle is uploaded from this phone — pick a file produced by a prior backup.'
	String get subtitle => 'Replay an encrypted .tar.gz.enc bundle into a Postgres database. The bundle is uploaded from this phone — pick a file produced by a prior backup.';

	/// en: 'Bundle file (.tar.gz.enc)'
	String get bundleLabel => 'Bundle file (.tar.gz.enc)';

	/// en: 'Pick file'
	String get pickFile => 'Pick file';

	/// en: '{name} · {size}'
	String fileSelected({required Object name, required Object size}) => '${name} · ${size}';

	/// en: 'No file selected'
	String get noFile => 'No file selected';

	/// en: 'Target Postgres DSN'
	String get targetDsnLabel => 'Target Postgres DSN';

	/// en: 'Leave empty to restore into opendray's own DB.'
	String get targetDsnHint => 'Leave empty to restore into opendray\'s own DB.';

	/// en: 'postgres://user:pass@host:5432/dbname'
	String get targetDsnPlaceholder => 'postgres://user:pass@host:5432/dbname';

	/// en: 'pg_restore --clean --if-exists'
	String get cleanLabel => 'pg_restore --clean --if-exists';

	/// en: 'Drops existing objects before recreating them.'
	String get cleanHint => 'Drops existing objects before recreating them.';

	/// en: 'Audit note (optional)'
	String get auditNoteLabel => 'Audit note (optional)';

	/// en: 'e.g. recovering from #INC-481'
	String get auditNotePlaceholder => 'e.g. recovering from #INC-481';

	/// en: 'Restoring into opendray's OWN database will rewrite the rows this gateway is currently serving. Type "I understand" to confirm.'
	String get ownDbWarning => 'Restoring into opendray\'s OWN database will rewrite the rows this gateway is currently serving. Type "I understand" to confirm.';

	/// en: 'Type "I understand"'
	String get confirmPlaceholder => 'Type "I understand"';

	/// en: 'I understand'
	String get confirmSentinel => 'I understand';

	/// en: 'Restoring…'
	String get restoring => 'Restoring…';

	/// en: 'Restore'
	String get restore => 'Restore';

	/// en: 'Restore succeeded'
	String get succeededTitle => 'Restore succeeded';

	/// en: 'Replayed {bytes} from backup {id}.'
	String succeededBody({required Object bytes, required Object id}) => 'Replayed ${bytes} from backup ${id}.';

	/// en: 'Restore failed'
	String get failedTitle => 'Restore failed';

	/// en: 'Pick a bundle file first.'
	String get pickFileToast => 'Pick a bundle file first.';

	/// en: 'pg_restore output'
	String get outputTitle => 'pg_restore output';

	/// en: '(empty — restore completed silently)'
	String get noPgRestoreOutput => '(empty — restore completed silently)';

	/// en: 'Manifest'
	String get manifestTitle => 'Manifest';

	/// en: 'Backup ID'
	String get manifestBackupId => 'Backup ID';

	/// en: 'Manifest version'
	String get manifestVersion => 'Manifest version';

	/// en: 'Created'
	String get manifestCreatedAt => 'Created';

	/// en: 'pg_version'
	String get manifestPgVersion => 'pg_version';

	/// en: 'opendray version'
	String get manifestOpendrayVersion => 'opendray version';

	/// en: 'Key fingerprint'
	String get fingerprint => 'Key fingerprint';

	/// en: 'matched'
	String get fingerprintOk => 'matched';

	/// en: 'MISMATCH'
	String get fingerprintMismatch => 'MISMATCH';

	/// en: 'Encryption'
	String get encryptionAlgo => 'Encryption';

	/// en: 'Bytes read'
	String get bytesRead => 'Bytes read';

	/// en: 'Target DSN'
	String get targetDsnUsed => 'Target DSN';

	/// en: '(opendray's own DB)'
	String get targetDsnSelfLabel => '(opendray\'s own DB)';

	/// en: 'Done'
	String get done => 'Done';
}

// Path: backups.inventory
class TranslationsBackupsInventoryEn {
	TranslationsBackupsInventoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'What's in a backup'
	String get title => 'What\'s in a backup';

	/// en: '{rows} rows · {tables} tables'
	String summary({required Object rows, required Object tables}) => '${rows} rows · ${tables} tables';

	/// en: 'Live row counts from opendray's Postgres database. Backups capture every row below; binary artifacts on disk are not included.'
	String get description => 'Live row counts from opendray\'s Postgres database. Backups capture every row below; binary artifacts on disk are not included.';

	/// en: 'rows'
	String get rowsLabel => 'rows';

	/// en: 'Failed to load inventory'
	String get loadFailedToast => 'Failed to load inventory';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Tap to expand'
	String get tap => 'Tap to expand';
}

// Path: backupTargetEditor.kinds
class TranslationsBackupTargetEditorKindsEn {
	TranslationsBackupTargetEditorKindsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsBackupTargetEditorKindsLocalEn local = TranslationsBackupTargetEditorKindsLocalEn.internal(_root);
	late final TranslationsBackupTargetEditorKindsSmbEn smb = TranslationsBackupTargetEditorKindsSmbEn.internal(_root);
	late final TranslationsBackupTargetEditorKindsWebdavEn webdav = TranslationsBackupTargetEditorKindsWebdavEn.internal(_root);
	late final TranslationsBackupTargetEditorKindsSftpEn sftp = TranslationsBackupTargetEditorKindsSftpEn.internal(_root);
	late final TranslationsBackupTargetEditorKindsS3En s3 = TranslationsBackupTargetEditorKindsS3En.internal(_root);
	late final TranslationsBackupTargetEditorKindsRcloneEn rclone = TranslationsBackupTargetEditorKindsRcloneEn.internal(_root);
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

	late final TranslationsGithostsFormKindsEn kinds = TranslationsGithostsFormKindsEn.internal(_root);

	/// en: 'Host is required.'
	String get validateHost => 'Host is required.';

	/// en: 'Name is required.'
	String get validateName => 'Name is required.';

	/// en: 'Host added.'
	String get snackAdded => 'Host added.';

	/// en: 'Host updated.'
	String get snackUpdated => 'Host updated.';

	/// en: 'Save failed: {error}'
	String saveFailedApi({required Object error}) => 'Save failed: ${error}';

	/// en: 'Save failed: {error}'
	String saveFailedGeneric({required Object error}) => 'Save failed: ${error}';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Add'
	String get add => 'Add';

	/// en: 'Display name shown in PR lists.'
	String get nameHelper => 'Display name shown in PR lists.';

	/// en: 'Token (leave blank to keep existing)'
	String get tokenLabelKeep => 'Token (leave blank to keep existing)';

	/// en: 'Token'
	String get tokenLabel => 'Token';

	/// en: 'Leave blank to keep existing.'
	String get tokenHintKeep => 'Leave blank to keep existing.';

	/// en: 'Paste the personal access token.'
	String get tokenHintNew => 'Paste the personal access token.';

	/// en: 'Available to sessions for PR / remote lookups.'
	String get enabledHelper => 'Available to sessions for PR / remote lookups.';

	/// en: 'Token is required when adding a host.'
	String get validateTokenRequired => 'Token is required when adding a host.';

	/// en: 'Edit {name}'
	String appBarEdit({required Object name}) => 'Edit ${name}';

	/// en: 'Add git host'
	String get appBarNew => 'Add git host';

	/// en: 'Current preview: {preview}'
	String tokenPreviewHint({required Object preview}) => 'Current preview: ${preview}';

	/// en: '(none)'
	String get tokenPreviewNone => '(none)';

	/// en: 'Paused — sessions skip this host.'
	String get pausedSubtitle => 'Paused — sessions skip this host.';
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

	/// en: 'Load failed: {error}'
	String loadFailedApi({required Object error}) => 'Load failed: ${error}';

	/// en: 'Load failed: {error}'
	String loadFailedGeneric({required Object error}) => 'Load failed: ${error}';

	/// en: 'Save failed: {error}'
	String saveFailedApi({required Object error}) => 'Save failed: ${error}';

	/// en: 'Save failed: {error}'
	String saveFailedGeneric({required Object error}) => 'Save failed: ${error}';

	/// en: 'Saved {time}'
	String savedAt({required Object time}) => 'Saved ${time}';
}

// Path: dataExport.sections
class TranslationsDataExportSectionsEn {
	TranslationsDataExportSectionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Export'
	String get export => 'Export';

	/// en: 'Import'
	String get import => 'Import';
}

// Path: dataExport.form
class TranslationsDataExportFormEn {
	TranslationsDataExportFormEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Scope'
	String get scope => 'Scope';

	/// en: 'Memories'
	String get memories => 'Memories';

	/// en: 'All persisted memories + their embeddings.'
	String get memoriesHint => 'All persisted memories + their embeddings.';

	/// en: 'Integrations'
	String get integrations => 'Integrations';

	late final TranslationsDataExportFormIntegrationOptionsEn integrationOptions = TranslationsDataExportFormIntegrationOptionsEn.internal(_root);

	/// en: 'Plaintext key export contains decryptable secrets. Type "I understand" to confirm.'
	String get confirmWarning => 'Plaintext key export contains decryptable secrets. Type "I understand" to confirm.';

	/// en: 'Type "I understand"'
	String get confirmPlaceholder => 'Type "I understand"';

	/// en: 'I understand'
	String get confirmSentinel => 'I understand';

	/// en: 'Custom tasks'
	String get customTasks => 'Custom tasks';

	/// en: 'Per-user task definitions (cron schedules + script bodies).'
	String get customTasksHint => 'Per-user task definitions (cron schedules + script bodies).';

	/// en: 'Bundles expire 7 days after creation. Download link is single-use.'
	String get footnote => 'Bundles expire 7 days after creation. Download link is single-use.';

	/// en: 'Create bundle'
	String get create => 'Create bundle';

	/// en: 'Building…'
	String get building => 'Building…';

	/// en: 'Bundle ready'
	String get readyToast => 'Bundle ready';

	/// en: '{bytes} bytes — download from the history below.'
	String readyDescription({required Object bytes}) => '${bytes} bytes — download from the history below.';

	/// en: 'Bundle creation failed: {error}'
	String failedToast({required Object error}) => 'Bundle creation failed: ${error}';
}

// Path: dataExport.history
class TranslationsDataExportHistoryEn {
	TranslationsDataExportHistoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Export history'
	String get title => 'Export history';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No exports yet.'
	String get empty => 'No exports yet.';

	/// en: 'Failed to load exports: {error}'
	String listFailedToast({required Object error}) => 'Failed to load exports: ${error}';

	/// en: 'Failed to fetch download token: {error}'
	String downloadFailedToast({required Object error}) => 'Failed to fetch download token: ${error}';

	/// en: 'This export has no usable download token (already consumed or expired).'
	String get noTokenToast => 'This export has no usable download token (already consumed or expired).';

	/// en: 'Export deleted.'
	String get deletedToast => 'Export deleted.';

	/// en: 'Failed to delete export: {error}'
	String deleteFailedToast({required Object error}) => 'Failed to delete export: ${error}';

	/// en: 'Delete export?'
	String get deleteConfirmTitle => 'Delete export?';

	/// en: 'Removes the bundle and revokes the download token. {id}'
	String deleteConfirmBody({required Object id}) => 'Removes the bundle and revokes the download token. ${id}';

	/// en: 'Download'
	String get download => 'Download';

	/// en: 'Delete'
	String get delete => 'Delete';

	/// en: 'Download URL copied to clipboard. Paste into a browser to fetch (single-use).'
	String get downloadCopiedToast => 'Download URL copied to clipboard. Paste into a browser to fetch (single-use).';

	late final TranslationsDataExportHistoryColumnsEn columns = TranslationsDataExportHistoryColumnsEn.internal(_root);

	/// en: '(empty)'
	String get scopeEmpty => '(empty)';

	/// en: 'memories'
	String get scopeMemories => 'memories';

	/// en: 'integrations({mode})'
	String scopeIntegrations({required Object mode}) => 'integrations(${mode})';

	/// en: 'custom_tasks'
	String get scopeCustomTasks => 'custom_tasks';
}

// Path: dataExport.import
class TranslationsDataExportImportEn {
	TranslationsDataExportImportEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Replays a bundle previously produced by Export. Only the entities you tick below are imported; everything else in the bundle is ignored.'
	String get intro => 'Replays a bundle previously produced by Export. Only the entities you tick below are imported; everything else in the bundle is ignored.';

	/// en: 'Bundle file (.zip)'
	String get bundleLabel => 'Bundle file (.zip)';

	/// en: 'Pick file'
	String get pickFile => 'Pick file';

	/// en: '{name} · {size}'
	String fileSelected({required Object name, required Object size}) => '${name} · ${size}';

	/// en: 'No file selected'
	String get noFile => 'No file selected';

	/// en: 'Memories'
	String get memoriesLabel => 'Memories';

	/// en: 'Integrations'
	String get integrationsLabel => 'Integrations';

	/// en: 'Custom tasks'
	String get customTasksLabel => 'Custom tasks';

	/// en: 'Import bundle'
	String get importBundle => 'Import bundle';

	/// en: 'Importing…'
	String get importing => 'Importing…';

	/// en: 'Pick a bundle file first.'
	String get pickFileToast => 'Pick a bundle file first.';

	/// en: 'Import done'
	String get doneToast => 'Import done';

	/// en: 'Import finished with errors'
	String get finishedWithErrors => 'Import finished with errors';

	/// en: 'Import failed: {error}'
	String failedToast({required Object error}) => 'Import failed: ${error}';

	late final TranslationsDataExportImportSummaryCardEn summaryCard = TranslationsDataExportImportSummaryCardEn.internal(_root);
}

// Path: dataExport.imports
class TranslationsDataExportImportsEn {
	TranslationsDataExportImportsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Import history'
	String get title => 'Import history';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No imports yet.'
	String get empty => 'No imports yet.';

	/// en: 'Failed to load imports: {error}'
	String listFailedToast({required Object error}) => 'Failed to load imports: ${error}';

	/// en: '(no counts)'
	String get noneCounts => '(no counts)';

	/// en: '(unknown source)'
	String get sourceUnknown => '(unknown source)';

	late final TranslationsDataExportImportsColumnsEn columns = TranslationsDataExportImportsColumnsEn.internal(_root);
}

// Path: dataExport.relative
class TranslationsDataExportRelativeEn {
	TranslationsDataExportRelativeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'in {n}s'
	String inSeconds({required Object n}) => 'in ${n}s';

	/// en: 'in {n}m'
	String inMinutes({required Object n}) => 'in ${n}m';

	/// en: 'in {n}h'
	String inHours({required Object n}) => 'in ${n}h';

	/// en: 'in {n}d'
	String inDays({required Object n}) => 'in ${n}d';

	/// en: '{n}s ago'
	String secondsAgo({required Object n}) => '${n}s ago';

	/// en: '{n}m ago'
	String minutesAgo({required Object n}) => '${n}m ago';

	/// en: '{n}h ago'
	String hoursAgo({required Object n}) => '${n}h ago';
}

// Path: dataExport.status
class TranslationsDataExportStatusEn {
	TranslationsDataExportStatusEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'pending'
	String get pending => 'pending';

	/// en: 'running'
	String get running => 'running';

	/// en: 'ready'
	String get ready => 'ready';

	/// en: 'failed'
	String get failed => 'failed';

	/// en: 'expired'
	String get expired => 'expired';

	/// en: 'succeeded'
	String get succeeded => 'succeeded';
}

// Path: memory.rank
class TranslationsMemoryRankEn {
	TranslationsMemoryRankEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Rank breakdown'
	String get title => 'Rank breakdown';

	/// en: 'Effective score: {value}'
	String effective({required Object value}) => 'Effective score: ${value}';

	/// en: 'Cosine similarity'
	String get similarity => 'Cosine similarity';

	/// en: 'Age multiplier ({days}d old)'
	String ageMultiplier({required Object days}) => 'Age multiplier (${days}d old)';

	/// en: 'Hit-count multiplier ({hits} hits)'
	String hitMultiplier({required Object hits}) => 'Hit-count multiplier (${hits} hits)';

	/// en: 'Confidence multiplier'
	String get confidenceMultiplier => 'Confidence multiplier';

	/// en: 'effective = similarity × age × hits × confidence'
	String get formula => 'effective = similarity × age × hits × confidence';

	/// en: 'Close'
	String get close => 'Close';
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

	/// en: 'The gateway will exec itself. The mobile app may briefly lose connection.'
	String get restartConfirmBody => 'The gateway will exec itself. The mobile app may briefly lose connection.';

	/// en: 'Restart'
	String get restart => 'Restart';

	/// en: 'Restart requested. Pull-to-refresh in a moment.'
	String get restartQueuedSnack => 'Restart requested. Pull-to-refresh in a moment.';

	/// en: 'Restart failed: {error}'
	String restartFailedApi({required Object error}) => 'Restart failed: ${error}';

	/// en: 'Restart failed: {error}'
	String restartFailedGeneric({required Object error}) => 'Restart failed: ${error}';

	/// en: 'Loaded from: {path}'
	String loadedFrom({required Object path}) => 'Loaded from: ${path}';

	/// en: 'Most sections need a gateway restart to take effect. The restart button is in the AppBar.'
	String get restartHint => 'Most sections need a gateway restart to take effect. The restart button is in the AppBar.';

	/// en: 'Saved. Restart the gateway to apply.'
	String get savedNeedsRestart => 'Saved. Restart the gateway to apply.';

	/// en: 'Saved.'
	String get savedSimple => 'Saved.';

	/// en: 'Changes to this section need a gateway restart.'
	String get changesNeedRestart => 'Changes to this section need a gateway restart.';

	/// en: 'Failed to load server settings'
	String get loadFailed => 'Failed to load server settings';

	late final TranslationsSettingsServerSettingsSectionsEn sections = TranslationsSettingsServerSettingsSectionsEn.internal(_root);
	late final TranslationsSettingsServerSettingsSectionDescriptionsEn sectionDescriptions = TranslationsSettingsServerSettingsSectionDescriptionsEn.internal(_root);
	late final TranslationsSettingsServerSettingsFieldsEn fields = TranslationsSettingsServerSettingsFieldsEn.internal(_root);

	/// en: '"{field}" must be an integer'
	String validateInteger({required Object field}) => '"${field}" must be an integer';

	/// en: '"{field}" must be a number'
	String validateNumber({required Object field}) => '"${field}" must be a number';
}

// Path: web.sessions.list
class TranslationsWebSessionsListEn {
	TranslationsWebSessionsListEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Sessions'
	String get title => 'Sessions';

	/// en: '·'
	String get countSeparator => '·';

	/// en: 'Spawn new session'
	String get newAria => 'Spawn new session';

	/// en: 'New session'
	String get newTooltip => 'New session';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No sessions yet.'
	String get emptyTitle => 'No sessions yet.';

	/// en: 'Press {kbd} to spawn.'
	String emptyHint({required Object kbd}) => 'Press ${kbd} to spawn.';

	/// en: 'Ended ({count})'
	String endedHeader({required Object count}) => 'Ended (${count})';

	/// en: 'Clear all'
	String get clearAll => 'Clear all';

	/// en: 'Remove all {count} ended sessions?'
	String confirmClearAll({required Object count}) => 'Remove all ${count} ended sessions?';

	/// en: 'Terminate and remove {name}?'
	String confirmTerminate({required Object name}) => 'Terminate and remove ${name}?';

	/// en: ' {count} child task session will be promoted to top-level.'
	String childPromoted({required Object count}) => ' ${count} child task session will be promoted to top-level.';

	/// en: ' {count} child task sessions will be promoted to top-level.'
	String childPromotedPlural({required Object count}) => ' ${count} child task sessions will be promoted to top-level.';

	/// en: '{live} live · {ended} ended'
	String footer({required Object live, required Object ended}) => '${live} live · ${ended} ended';

	late final TranslationsWebSessionsListRowEn row = TranslationsWebSessionsListRowEn.internal(_root);

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';
}

// Path: web.sessions.tabs
class TranslationsWebSessionsTabsEn {
	TranslationsWebSessionsTabsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Close tab and remove session'
	String get closeAria => 'Close tab and remove session';

	/// en: 'Close tab and remove session'
	String get closeTitle => 'Close tab and remove session';
}

// Path: web.sessions.page
class TranslationsWebSessionsPageEn {
	TranslationsWebSessionsPageEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Session removed'
	String get removedToast => 'Session removed';

	/// en: 'Remove failed'
	String get removeFailedToast => 'Remove failed';

	/// en: 'Session stopped'
	String get stoppedToast => 'Session stopped';

	/// en: 'Stop failed'
	String get stopFailedToast => 'Stop failed';

	/// en: 'Session restarted'
	String get restartedToast => 'Session restarted';

	/// en: 'Restart failed'
	String get restartFailedToast => 'Restart failed';

	/// en: 'Stop and remove "{name}"?'
	String confirmCloseTabTitle({required Object name}) => 'Stop and remove "${name}"?';

	/// en: 'The CLI process will be terminated and the row deleted.'
	String get confirmCloseTabDescription => 'The CLI process will be terminated and the row deleted.';

	/// en: 'Stop and remove'
	String get confirmCloseTabConfirm => 'Stop and remove';

	/// en: 'Remove {name}?'
	String confirmRemoveTitle({required Object name}) => 'Remove ${name}?';

	/// en: 'Remove session?'
	String get confirmRemoveTitleFallback => 'Remove session?';

	/// en: 'This deletes the row.'
	String get confirmRemoveDescription => 'This deletes the row.';

	/// en: 'Remove'
	String get confirmRemoveConfirm => 'Remove';
}

// Path: web.sessions.empty
class TranslationsWebSessionsEmptyEn {
	TranslationsWebSessionsEmptyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No session open'
	String get title => 'No session open';

	/// en: 'Pick a session from the list, or spawn a new one. Keyboard: {kbdN} new, {kbdW} close, {kbdRange} switch.'
	String hint({required Object kbdN, required Object kbdW, required Object kbdRange}) => 'Pick a session from the list, or spawn a new one. Keyboard: ${kbdN} new, ${kbdW} close, ${kbdRange} switch.';

	/// en: 'Spawn session'
	String get spawn => 'Spawn session';
}

// Path: web.sessions.header
class TranslationsWebSessionsHeaderEn {
	TranslationsWebSessionsHeaderEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading session…'
	String get loadingSession => 'Loading session…';

	/// en: 'Show session list'
	String get showList => 'Show session list';

	/// en: 'Hide session list'
	String get hideList => 'Hide session list';

	/// en: 'Show inspector'
	String get showInspector => 'Show inspector';

	/// en: 'Hide inspector'
	String get hideInspector => 'Hide inspector';

	/// en: 'Attach image'
	String get attachImage => 'Attach image';

	/// en: 'Attach image (or paste / drop into terminal)'
	String get attachImageTooltip => 'Attach image (or paste / drop into terminal)';

	/// en: 'Restart'
	String get restart => 'Restart';

	/// en: 'Restarting…'
	String get restarting => 'Restarting…';

	/// en: 'Remove'
	String get remove => 'Remove';

	/// en: 'Removing…'
	String get removing => 'Removing…';

	/// en: 'Stop'
	String get stop => 'Stop';

	/// en: 'Stopping…'
	String get stopping => 'Stopping…';

	/// en: 'pid {pid}'
	String pid({required Object pid}) => 'pid ${pid}';
}

// Path: web.sessions.terminal
class TranslationsWebSessionsTerminalEn {
	TranslationsWebSessionsTerminalEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Uploading image…'
	String get uploadingToast => 'Uploading image…';

	/// en: 'Image attached'
	String get uploadedToast => 'Image attached';

	/// en: 'Upload failed'
	String get uploadFailedToast => 'Upload failed';

	/// en: 'Only image files can be attached'
	String get uploadInvalidTypeToast => 'Only image files can be attached';

	/// en: 'Drop image to attach'
	String get dropToAttach => 'Drop image to attach';
}

// Path: web.sessions.spawn
class TranslationsWebSessionsSpawnEn {
	TranslationsWebSessionsSpawnEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Spawn session'
	String get title => 'Spawn session';

	/// en: 'Start a CLI session under a registered provider.'
	String get description => 'Start a CLI session under a registered provider.';

	/// en: 'Provider'
	String get provider => 'Provider';

	/// en: 'Claude account'
	String get claudeAccount => 'Claude account';

	/// en: 'Loading accounts…'
	String get loadingAccounts => 'Loading accounts…';

	/// en: 'No Claude accounts configured — the gateway will use the system ANTHROPIC_API_KEY.'
	String get noAccounts => 'No Claude accounts configured — the gateway will use the system ANTHROPIC_API_KEY.';

	/// en: 'Default'
	String get kDefault => 'Default';

	/// en: 'Use system keychain / env'
	String get defaultTooltip => 'Use system keychain / env';

	/// en: '·empty'
	String get tokenEmptyBadge => '·empty';

	/// en: 'No token set — set token in Providers panel first'
	String get tokenMissingTooltip => 'No token set — set token in Providers panel first';

	/// en: 'Multiple accounts configured — pick one for this session.'
	String get multiAccountHint => 'Multiple accounts configured — pick one for this session.';

	/// en: 'Working directory'
	String get cwd => 'Working directory';

	/// en: '/Users/you/projects/foo'
	String get cwdPlaceholder => '/Users/you/projects/foo';

	/// en: 'Browse'
	String get browse => 'Browse';

	/// en: 'Name (optional)'
	String get nameLabel => 'Name (optional)';

	/// en: 'claude in pet-tracker'
	String get namePlaceholder => 'claude in pet-tracker';

	/// en: 'CLI args (one per line)'
	String get argsLabel => 'CLI args (one per line)';

	/// en: 'Bypass permission prompts'
	String get bypassClaude => 'Bypass permission prompts';

	/// en: 'Bypass approvals & sandbox (--dangerously-bypass-approvals-and-sandbox)'
	String get bypassCodex => 'Bypass approvals & sandbox (--dangerously-bypass-approvals-and-sandbox)';

	/// en: 'YOLO mode (--yolo)'
	String get bypassGemini => 'YOLO mode (--yolo)';

	/// en: 'This session will run with elevated autonomy.'
	String get bypassOnHint => 'This session will run with elevated autonomy.';

	/// en: 'Off — confirmations and prompts behave normally.'
	String get bypassOffHint => 'Off — confirmations and prompts behave normally.';

	/// en: 'Pick a provider.'
	String get errorPickProvider => 'Pick a provider.';

	/// en: 'cwd is required.'
	String get errorCwdRequired => 'cwd is required.';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Spawn'
	String get submit => 'Spawn';

	/// en: 'Spawning…'
	String get submitting => 'Spawning…';

	/// en: 'Session spawned'
	String get spawnedToast => 'Session spawned';

	/// en: '{provider} · pid {pid}'
	String spawnedDescription({required Object provider, required Object pid}) => '${provider} · pid ${pid}';

	/// en: '—'
	String get pidFallback => '—';
}

// Path: web.sessions.accountSwitcher
class TranslationsWebSessionsAccountSwitcherEn {
	TranslationsWebSessionsAccountSwitcherEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Switch Claude account (restarts the CLI process)'
	String get tooltip => 'Switch Claude account (restarts the CLI process)';

	/// en: 'default'
	String get currentDefault => 'default';

	/// en: 'Switch Claude account'
	String get menuTitle => 'Switch Claude account';

	/// en: 'Default'
	String get defaultName => 'Default';

	/// en: 'CLI's system keychain / env'
	String get defaultSubtitle => 'CLI\'s system keychain / env';

	/// en: '·empty'
	String get tokenEmpty => '·empty';

	/// en: 'Switching account will restart the Claude CLI process. In-progress conversation state inside the CLI will be lost. Continue?'
	String get confirmSwitch => 'Switching account will restart the Claude CLI process. In-progress conversation state inside the CLI will be lost. Continue?';

	/// en: 'Account switched'
	String get switchedToast => 'Account switched';

	/// en: 'Now using @{account} · pid {pid}'
	String switchedDescription({required Object account, required Object pid}) => 'Now using @${account} · pid ${pid}';

	/// en: 'default'
	String get switchedDefault => 'default';

	/// en: 'Switch failed'
	String get switchFailedToast => 'Switch failed';
}

// Path: web.sessions.inspector
class TranslationsWebSessionsInspectorEn {
	TranslationsWebSessionsInspectorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebSessionsInspectorTabsEn tabs = TranslationsWebSessionsInspectorTabsEn.internal(_root);
}

// Path: web.sessions.ended
class TranslationsWebSessionsEndedEn {
	TranslationsWebSessionsEndedEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '[buffer unavailable]'
	String get bufferUnavailable => '[buffer unavailable]';

	/// en: '[session ended — read-only buffer]'
	String get readOnlyBanner => '[session ended — read-only buffer]';
}

// Path: web.sessions.fileBrowser
class TranslationsWebSessionsFileBrowserEn {
	TranslationsWebSessionsFileBrowserEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Choose working directory'
	String get title => 'Choose working directory';

	/// en: 'Browse the gateway host's filesystem and pick a folder.'
	String get description => 'Browse the gateway host\'s filesystem and pick a folder.';

	/// en: 'Parent directory'
	String get parent => 'Parent directory';

	/// en: 'Home directory'
	String get home => 'Home directory';

	/// en: 'Refresh'
	String get refresh => 'Refresh';

	/// en: '/Users/you/projects'
	String get pathPlaceholder => '/Users/you/projects';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Empty directory.'
	String get empty => 'Empty directory.';

	/// en: 'New folder'
	String get newFolder => 'New folder';

	/// en: 'new-folder-name'
	String get newFolderPlaceholder => 'new-folder-name';

	/// en: 'Create'
	String get create => 'Create';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Use this folder'
	String get useThisFolder => 'Use this folder';

	/// en: 'Directory created'
	String get createdToast => 'Directory created';

	/// en: 'Mkdir failed'
	String get mkdirFailedToast => 'Mkdir failed';

	/// en: 'Failed to read home'
	String get homeFailedToast => 'Failed to read home';
}

// Path: web.conflicts.confirmDelete
class TranslationsWebConflictsConfirmDeleteEn {
	TranslationsWebConflictsConfirmDeleteEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Delete fact {side}?'
	String title({required Object side}) => 'Delete fact ${side}?';

	/// en: 'This permanently removes the fact and accepts the conflict. The other side stays as the surviving claim.'
	String get description => 'This permanently removes the fact and accepts the conflict. The other side stays as the surviving claim.';

	/// en: 'Will delete (side {side}):'
	String targetLabel({required Object side}) => 'Will delete (side ${side}):';

	/// en: 'Will keep (side {side}):'
	String keepLabel({required Object side}) => 'Will keep (side ${side}):';

	/// en: '({layer} entry — open the corresponding tab to inspect)'
	String nonFactOther({required Object layer}) => '(${layer} entry — open the corresponding tab to inspect)';

	/// en: 'Detector evidence:'
	String get evidenceLabel => 'Detector evidence:';

	/// en: 'Loading fact text…'
	String get loading => 'Loading fact text…';

	/// en: 'Failed to load fact text. Inspect on the Memory page.'
	String get loadError => 'Failed to load fact text. Inspect on the Memory page.';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Delete {side}'
	String confirm({required Object side}) => 'Delete ${side}';
}

// Path: web.conflicts.openLayer
class TranslationsWebConflictsOpenLayerEn {
	TranslationsWebConflictsOpenLayerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Open plan editor'
	String get plan => 'Open plan editor';

	/// en: 'Open goal editor'
	String get goal => 'Open goal editor';
}

// Path: web.conflicts.severity
class TranslationsWebConflictsSeverityEn {
	TranslationsWebConflictsSeverityEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'low'
	String get low => 'low';

	/// en: 'medium'
	String get medium => 'medium';

	/// en: 'high'
	String get high => 'high';
}

// Path: web.memoryConfig.sections
class TranslationsWebMemoryConfigSectionsEn {
	TranslationsWebMemoryConfigSectionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Providers'
	String get providers => 'Providers';

	/// en: 'Workers'
	String get workers => 'Workers';

	/// en: 'Capture rules'
	String get rules => 'Capture rules';

	/// en: 'Injection profiles'
	String get profiles => 'Injection profiles';

	/// en: 'Token cost'
	String get costs => 'Token cost';
}

// Path: web.memoryConfig.sectionHints
class TranslationsWebMemoryConfigSectionHintsEn {
	TranslationsWebMemoryConfigSectionHintsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Registered HTTP endpoints (Ollama / LM Studio / Anthropic / OpenAI / Integration) that any task can dispatch to.'
	String get providers => 'Registered HTTP endpoints (Ollama / LM Studio / Anthropic / OpenAI / Integration) that any task can dispatch to.';

	/// en: 'For each touchpoint pick HTTP provider (cheap, local) or headless Claude / Gemini Agent (higher quality, costs CLI tokens).'
	String get workers => 'For each touchpoint pick HTTP provider (cheap, local) or headless Claude / Gemini Agent (higher quality, costs CLI tokens).';

	/// en: 'When the capture engine fires per session (after N messages / on idle / K characters / manual). Rules without a pinned provider follow the Capture worker setting above.'
	String get rules => 'When the capture engine fires per session (after N messages / on idle / K characters / manual). Rules without a pinned provider follow the Capture worker setting above.';

	/// en: 'How prior memories get injected into the agent's system prompt at session spawn (recency, relevance, hybrid, or off).'
	String get profiles => 'How prior memories get injected into the agent\'s system prompt at session spawn (recency, relevance, hybrid, or off).';

	/// en: 'Aggregate spend reconstructed from memory_summarizer_calls. Local providers (Ollama, LM Studio, Integration) are free; cloud providers show real-world cost.'
	String get costs => 'Aggregate spend reconstructed from memory_summarizer_calls. Local providers (Ollama, LM Studio, Integration) are free; cloud providers show real-world cost.';
}

// Path: web.memoryConfig.moveBanner
class TranslationsWebMemoryConfigMoveBannerEn {
	TranslationsWebMemoryConfigMoveBannerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory configuration has moved'
	String get title => 'Memory configuration has moved';

	/// en: 'All memory-related settings (providers / capture rules / injection profiles / cost) now live alongside Workers in one page so related knobs sit together.'
	String get body => 'All memory-related settings (providers / capture rules / injection profiles / cost) now live alongside Workers in one page so related knobs sit together.';

	/// en: 'Open Memory configuration →'
	String get openButton => 'Open Memory configuration →';
}

// Path: web.memoryWorkers.tasks
class TranslationsWebMemoryWorkersTasksEn {
	TranslationsWebMemoryWorkersTasksEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebMemoryWorkersTasksGatekeeperEn gatekeeper = TranslationsWebMemoryWorkersTasksGatekeeperEn.internal(_root);
	late final TranslationsWebMemoryWorkersTasksCleanerEn cleaner = TranslationsWebMemoryWorkersTasksCleanerEn.internal(_root);
	late final TranslationsWebMemoryWorkersTasksGitactivityEn gitactivity = TranslationsWebMemoryWorkersTasksGitactivityEn.internal(_root);
	late final TranslationsWebMemoryWorkersTasksTranscriptEn transcript = TranslationsWebMemoryWorkersTasksTranscriptEn.internal(_root);
	late final TranslationsWebMemoryWorkersTasksPlanDriftEn plan_drift = TranslationsWebMemoryWorkersTasksPlanDriftEn.internal(_root);
	late final TranslationsWebMemoryWorkersTasksConflictDetectorEn conflict_detector = TranslationsWebMemoryWorkersTasksConflictDetectorEn.internal(_root);
	late final TranslationsWebMemoryWorkersTasksCaptureEn capture = TranslationsWebMemoryWorkersTasksCaptureEn.internal(_root);
}

// Path: web.project.picker
class TranslationsWebProjectPickerEn {
	TranslationsWebProjectPickerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Pick a project'
	String get title => 'Pick a project';

	/// en: 'Project memory is scoped by working directory. Pick one to manage its goal, plan, journal, and cleanup queue.'
	String get subtitle => 'Project memory is scoped by working directory. Pick one to manage its goal, plan, journal, and cleanup queue.';

	/// en: '/path/to/your/project'
	String get pathPlaceholder => '/path/to/your/project';

	/// en: 'Browse'
	String get browse => 'Browse';

	/// en: 'Browse the gateway host's filesystem'
	String get browseTooltip => 'Browse the gateway host\'s filesystem';

	/// en: 'Open'
	String get open => 'Open';

	/// en: 'Recent projects (from stored memory):'
	String get recentLabel => 'Recent projects (from stored memory):';

	/// en: 'Looks like a truncated scope_key (old mirror import bug). May have no project docs.'
	String get orphanTooltip => 'Looks like a truncated scope_key (old mirror import bug). May have no project docs.';

	/// en: 'orphan'
	String get orphanBadge => 'orphan';
}

// Path: web.project.header
class TranslationsWebProjectHeaderEn {
	TranslationsWebProjectHeaderEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '{count} doc'
	String docsCount_one({required Object count}) => '${count} doc';

	/// en: '{count} docs'
	String docsCount_other({required Object count}) => '${count} docs';

	/// en: '{count} journal entry'
	String journalEntries_one({required Object count}) => '${count} journal entry';

	/// en: '{count} journal entries'
	String journalEntries_other({required Object count}) => '${count} journal entries';

	/// en: '{count} pending proposal'
	String pendingProposals_one({required Object count}) => '${count} pending proposal';

	/// en: '{count} pending proposals'
	String pendingProposals_other({required Object count}) => '${count} pending proposals';

	/// en: '{count} cleanup pending'
	String cleanupPending({required Object count}) => '${count} cleanup pending';
}

// Path: web.project.tabs
class TranslationsWebProjectTabsEn {
	TranslationsWebProjectTabsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Health'
	String get health => 'Health';

	/// en: 'Goal'
	String get goal => 'Goal';

	/// en: 'Plan'
	String get plan => 'Plan';

	/// en: 'Tech'
	String get tech => 'Tech';

	/// en: 'Activity'
	String get activity => 'Activity';

	/// en: 'Journal'
	String get journal => 'Journal';

	/// en: 'Inbox'
	String get inbox => 'Inbox';

	/// en: 'Conflicts'
	String get conflicts => 'Conflicts';

	/// en: 'Cleanup'
	String get cleanup => 'Cleanup';
}

// Path: web.project.docLabel
class TranslationsWebProjectDocLabelEn {
	TranslationsWebProjectDocLabelEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Goal'
	String get goal => 'Goal';

	/// en: 'Plan'
	String get plan => 'Plan';

	/// en: 'Tech stack'
	String get tech_stack => 'Tech stack';

	/// en: 'Recent activity'
	String get recent_activity => 'Recent activity';
}

// Path: web.project.verdictLabel
class TranslationsWebProjectVerdictLabelEn {
	TranslationsWebProjectVerdictLabelEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Delete'
	String get stale => 'Delete';

	/// en: 'Merge'
	String get duplicate => 'Merge';

	/// en: 'Keep'
	String get keep => 'Keep';
}

// Path: web.project.editor
class TranslationsWebProjectEditorEn {
	TranslationsWebProjectEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Updated by'
	String get updatedBy => 'Updated by';

	/// en: 'No {label} set yet.'
	String noDocSet({required Object label}) => 'No ${label} set yet.';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Save failed'
	String get saveFailedToast => 'Save failed';

	/// en: '{label} saved'
	String savedToast({required Object label}) => '${label} saved';

	/// en: 'What are we building? One paragraph. Read by every agent on spawn.'
	String get goalPlaceholder => 'What are we building? One paragraph. Read by every agent on spawn.';

	/// en: 'Active plan — what we are doing right now and what is next. Updated as work progresses.'
	String get planPlaceholder => 'Active plan — what we are doing right now and what is next. Updated as work progresses.';
}

// Path: web.project.readonly
class TranslationsWebProjectReadonlyEn {
	TranslationsWebProjectReadonlyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebProjectReadonlyTechStackEn tech_stack = TranslationsWebProjectReadonlyTechStackEn.internal(_root);
	late final TranslationsWebProjectReadonlyRecentActivityEn recent_activity = TranslationsWebProjectReadonlyRecentActivityEn.internal(_root);

	/// en: 'No {label} captured yet.'
	String noneCaptured({required Object label}) => 'No ${label} captured yet.';

	/// en: 'Generated by'
	String get generatedBy => 'Generated by';

	/// en: 'last refresh'
	String get lastRefresh => 'last refresh';
}

// Path: web.project.journal
class TranslationsWebProjectJournalEn {
	TranslationsWebProjectJournalEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No journal entries yet. Each session-end appends one automatically.'
	String get empty => 'No journal entries yet. Each session-end appends one automatically.';
}

// Path: web.project.inbox
class TranslationsWebProjectInboxEn {
	TranslationsWebProjectInboxEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Inbox empty.'
	String get emptyTitle => 'Inbox empty.';

	/// en: 'Agents file proposals here via `project_goal_set` / `project_plan_set` MCP tools.'
	String get emptyHint => 'Agents file proposals here via `project_goal_set` / `project_plan_set` MCP tools.';

	/// en: '{label} updated'
	String approvedToast({required Object label}) => '${label} updated';

	/// en: 'Approve failed'
	String get approveFailedToast => 'Approve failed';

	/// en: 'Rejected'
	String get rejectedToast => 'Rejected';

	/// en: 'Reject failed'
	String get rejectFailedToast => 'Reject failed';

	/// en: 'ses'
	String get sessionPrefix => 'ses';

	/// en: 'Approve will REPLACE the current {label} entirely.'
	String warning({required Object label}) => 'Approve will REPLACE the current ${label} entirely.';

	/// en: 'Review the diff below; this isn't a merge.'
	String get warningSuffix => 'Review the diff below; this isn\'t a merge.';

	/// en: 'Current'
	String get current => 'Current';

	/// en: 'Proposed'
	String get proposed => 'Proposed';

	/// en: '(empty)'
	String get emptyBody => '(empty)';

	/// en: 'Approve'
	String get approve => 'Approve';

	/// en: 'Reject'
	String get reject => 'Reject';

	/// en: 'Replace {label}?'
	String confirmDialogTitle({required Object label}) => 'Replace ${label}?';

	/// en: 'The current {label} will be overwritten with the proposed content. This cannot be undone via this UI (you can manually edit it back).'
	String confirmDialogDescription({required Object label}) => 'The current ${label} will be overwritten with the proposed content. This cannot be undone via this UI (you can manually edit it back).';

	/// en: 'Cancel'
	String get confirmCancel => 'Cancel';

	/// en: 'Confirm replace'
	String get confirmReplace => 'Confirm replace';
}

// Path: web.project.cleanup
class TranslationsWebProjectCleanupEn {
	TranslationsWebProjectCleanupEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'The LLM librarian proposes keep / stale / duplicate verdicts for this project's memories. You approve before anything is deleted.'
	String get hint => 'The LLM librarian proposes keep / stale / duplicate verdicts for this project\'s memories. You approve before anything is deleted.';

	/// en: 'Run cleanup now'
	String get runNow => 'Run cleanup now';

	/// en: 'Cleanup run: {decided} decisions queued ({scanned} scanned)'
	String runSucceededToast({required Object decided, required Object scanned}) => 'Cleanup run: ${decided} decisions queued (${scanned} scanned)';

	/// en: 'Cleanup run failed'
	String get runFailedToast => 'Cleanup run failed';

	/// en: 'No pending decisions. Either nothing aged into eligibility or the last run found everything load-bearing.'
	String get empty => 'No pending decisions. Either nothing aged into eligibility or the last run found everything load-bearing.';

	/// en: '→ merge into'
	String get mergeIntoPrefix => '→ merge into';

	/// en: 'Reason:'
	String get reasonPrefix => 'Reason:';

	/// en: 'Execute'
	String get executeButton => 'Execute';

	/// en: 'Confirm keep'
	String get confirmKeepButton => 'Confirm keep';

	/// en: 'Reject'
	String get rejectButton => 'Reject';

	/// en: '{label} executed'
	String approvedExecutedToast({required Object label}) => '${label} executed';

	/// en: 'Approve failed'
	String get approveFailedToast => 'Approve failed';

	/// en: 'Rejected — memory kept'
	String get rejectedToast => 'Rejected — memory kept';

	/// en: 'Reject failed'
	String get rejectFailedToast => 'Reject failed';
}

// Path: web.project.reset
class TranslationsWebProjectResetEn {
	TranslationsWebProjectResetEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Reset'
	String get button => 'Reset';

	/// en: 'Reset project memory?'
	String get dialogTitle => 'Reset project memory?';

	/// en: 'Deletes all stored project context for this cwd. This cannot be undone.'
	String get dialogDescription => 'Deletes all stored project context for this cwd. This cannot be undone.';

	/// en: 'Always deleted: goal, plan, proposals, journal, cleanup decisions.'
	String get alwaysDeleted => 'Always deleted: goal, plan, proposals, journal, cleanup decisions.';

	/// en: 'Also delete scanner docs'
	String get alsoDeleteScannerLabel => 'Also delete scanner docs';

	/// en: '(tech_stack + recent_activity).'
	String get alsoDeleteScannerSuffix => '(tech_stack + recent_activity).';

	/// en: 'Auto-rebuild on next spawn anyway — leaving unchecked is usually fine.'
	String get alsoDeleteScannerHint => 'Auto-rebuild on next spawn anyway — leaving unchecked is usually fine.';

	/// en: 'Also delete pgvector memories'
	String get alsoDeleteMemoriesLabel => 'Also delete pgvector memories';

	/// en: 'for this scope_key.'
	String get alsoDeleteMemoriesSuffix => 'for this scope_key.';

	/// en: 'Long-term facts the agent stored (user preferences, project facts). Cannot be recovered.'
	String get alsoDeleteMemoriesHint => 'Long-term facts the agent stored (user preferences, project facts). Cannot be recovered.';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Delete forever'
	String get deleteForever => 'Delete forever';

	/// en: 'Reset: deleted {summary}'
	String successToast({required Object summary}) => 'Reset: deleted ${summary}';

	late final TranslationsWebProjectResetSummaryEn summary = TranslationsWebProjectResetSummaryEn.internal(_root);

	/// en: 'Reset failed'
	String get failedToast => 'Reset failed';
}

// Path: web.memoryInspector.status
class TranslationsWebMemoryInspectorStatusEn {
	TranslationsWebMemoryInspectorStatusEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Active embedder'
	String get label => 'Active embedder';

	/// en: 'unavailable'
	String get unavailable => 'unavailable';

	/// en: 'probing…'
	String get probing => 'probing…';

	/// en: '{dim}-dim · {state}'
	String dimensions({required Object dim, required Object state}) => '${dim}-dim · ${state}';

	/// en: 'enabled'
	String get enabled => 'enabled';

	/// en: 'disabled'
	String get disabled => 'disabled';

	/// en: 'Test embedder'
	String get testButton => 'Test embedder';
}

// Path: web.memoryInspector.scope
class TranslationsWebMemoryInspectorScopeEn {
	TranslationsWebMemoryInspectorScopeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Scope'
	String get label => 'Scope';

	/// en: 'Scope key'
	String get scopeKey => 'Scope key';

	/// en: '(ignored for global)'
	String get scopeKeyIgnored => '(ignored for global)';

	/// en: '(cwd of the project)'
	String get scopeKeyCwd => '(cwd of the project)';

	/// en: '(session id)'
	String get scopeKeySession => '(session id)';

	/// en: '/path/to/project (cwd)'
	String get placeholderProject => '/path/to/project (cwd)';

	/// en: 'session id'
	String get placeholderSession => 'session id';

	/// en: 'Sync .md'
	String get syncMd => 'Sync .md';

	/// en: 'Re-ingest Claude's <cwd>/.claude/memory/*.md files into pgvector'
	String get syncTooltip => 'Re-ingest Claude\'s <cwd>/.claude/memory/*.md files into pgvector';

	late final TranslationsWebMemoryInspectorScopeValuesEn values = TranslationsWebMemoryInspectorScopeValuesEn.internal(_root);
}

// Path: web.memoryInspector.search
class TranslationsWebMemoryInspectorSearchEn {
	TranslationsWebMemoryInspectorSearchEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Semantic search query (Enter to run; empty = browse)'
	String get placeholder => 'Semantic search query (Enter to run; empty = browse)';

	/// en: 'Search'
	String get run => 'Search';

	/// en: 'Clear'
	String get clear => 'Clear';

	/// en: 'Search failed'
	String get failedToast => 'Search failed';
}

// Path: web.memoryInspector.records
class TranslationsWebMemoryInspectorRecordsEn {
	TranslationsWebMemoryInspectorRecordsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No memories yet'
	String get noMemories => 'No memories yet';

	/// en: '{count} match'
	String matches_one({required Object count}) => '${count} match';

	/// en: '{count} matches'
	String matches_other({required Object count}) => '${count} matches';

	/// en: '{count} memory'
	String memories_one({required Object count}) => '${count} memory';

	/// en: '{count} memories'
	String memories_other({required Object count}) => '${count} memories';

	/// en: ' (global)'
	String get scopeGlobalSuffix => ' (global)';

	/// en: ' in {scope}: '
	String scopeInSuffix({required Object scope}) => ' in ${scope}: ';

	/// en: 'Add memory'
	String get addButton => 'Add memory';

	/// en: 'Manually create a memory in this scope'
	String get addTooltip => 'Manually create a memory in this scope';

	/// en: 'Delete all'
	String get deleteAll => 'Delete all';

	/// en: 'Delete every memory under this scope/scope_key'
	String get deleteAllTooltip => 'Delete every memory under this scope/scope_key';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Enter a scope key to browse memories.'
	String get enterScopeKeyHint => 'Enter a scope key to browse memories.';

	/// en: 'No matches for "{query}"'
	String noMatchesForQuery({required Object query}) => 'No matches for "${query}"';

	/// en: 'No memories in this scope yet.'
	String get noMemoriesInScope => 'No memories in this scope yet.';
}

// Path: web.memoryInspector.row
class TranslationsWebMemoryInspectorRowEn {
	TranslationsWebMemoryInspectorRowEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'sim {value}'
	String simBadge({required Object value}) => 'sim ${value}';

	/// en: 'rank {value}'
	String rankBadge({required Object value}) => 'rank ${value}';

	/// en: 'effective {effective} = sim {similarity} × age {age} ({days}d) × hits {hits} × conf {confidence}'
	String rankTooltip({required Object effective, required Object similarity, required Object age, required Object days, required Object hits, required Object confidence}) => 'effective ${effective} = sim ${similarity} × age ${age} (${days}d) × hits ${hits} × conf ${confidence}';

	/// en: '{count} hit'
	String hits_one({required Object count}) => '${count} hit';

	/// en: '{count} hits'
	String hits_other({required Object count}) => '${count} hits';

	/// en: 'Last hit {relative}'
	String lastHitTooltip({required Object relative}) => 'Last hit ${relative}';

	/// en: 'Memory text — Cmd/Ctrl+Enter to save · Esc to cancel'
	String get editPlaceholder => 'Memory text — Cmd/Ctrl+Enter to save · Esc to cancel';

	/// en: 'Save (Cmd/Ctrl+Enter)'
	String get saveTooltip => 'Save (Cmd/Ctrl+Enter)';

	/// en: 'Cancel (Esc)'
	String get cancelTooltip => 'Cancel (Esc)';

	/// en: 'Edit this memory'
	String get editTooltip => 'Edit this memory';

	/// en: 'Delete this memory'
	String get deleteTooltip => 'Delete this memory';

	/// en: 'Memory text cannot be empty'
	String get emptyError => 'Memory text cannot be empty';

	/// en: 'Delete memory {id}? This is permanent.'
	String deleteConfirm({required Object id}) => 'Delete memory ${id}? This is permanent.';
}

// Path: web.memoryInspector.toasts
class TranslationsWebMemoryInspectorToastsEn {
	TranslationsWebMemoryInspectorToastsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory deleted'
	String get deleted => 'Memory deleted';

	/// en: 'Delete failed'
	String get deleteFailed => 'Delete failed';

	/// en: 'Deleted {count} memory from this scope'
	String bulkDeleted_one({required Object count}) => 'Deleted ${count} memory from this scope';

	/// en: 'Deleted {count} memories from this scope'
	String bulkDeleted_other({required Object count}) => 'Deleted ${count} memories from this scope';

	/// en: 'Bulk delete failed'
	String get bulkDeleteFailed => 'Bulk delete failed';

	/// en: 'Memory created'
	String get created => 'Memory created';

	/// en: 'Create failed'
	String get createFailed => 'Create failed';

	/// en: 'Memory updated'
	String get updated => 'Memory updated';

	/// en: 'Update failed'
	String get updateFailed => 'Update failed';

	/// en: 'Migrated {reembed}/{examined} memories to {to}'
	String migrated({required Object reembed, required Object examined, required Object to}) => 'Migrated ${reembed}/${examined} memories to ${to}';

	/// en: 'Migration failed'
	String get migrationFailed => 'Migration failed';

	/// en: 'Ingested {count} new memory file'
	String syncIngested_one({required Object count}) => 'Ingested ${count} new memory file';

	/// en: 'Ingested {count} new memory files'
	String syncIngested_other({required Object count}) => 'Ingested ${count} new memory files';

	/// en: 'No new .md files to sync'
	String get syncEmpty => 'No new .md files to sync';

	/// en: 'Already in sync, or no Claude memory dir for this cwd.'
	String get syncEmptyDescription => 'Already in sync, or no Claude memory dir for this cwd.';

	/// en: 'Sync failed'
	String get syncFailed => 'Sync failed';

	/// en: 'Embedder OK: {embedder} · {dim} dimensions'
	String testOk({required Object embedder, required Object dim}) => 'Embedder OK: ${embedder} · ${dim} dimensions';

	/// en: 'vector_preview = [{preview}…]'
	String testOkDescription({required Object preview}) => 'vector_preview = [${preview}…]';

	/// en: 'Embedder probe failed'
	String get testFailed => 'Embedder probe failed';
}

// Path: web.memoryInspector.bulkDelete
class TranslationsWebMemoryInspectorBulkDeleteEn {
	TranslationsWebMemoryInspectorBulkDeleteEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Delete every memory in this scope?'
	String get title => 'Delete every memory in this scope?';

	/// en: 'This is a single SQL operation — all memories under the specified scope are removed atomically. Memories that were ingested via the Claude mirror reappear on the next <1>Sync .md</1> run; everything else is gone for good.'
	String get description => 'This is a single SQL operation — all memories under the specified scope are removed atomically. Memories that were ingested via the Claude mirror reappear on the next <1>Sync .md</1> run; everything else is gone for good.';

	/// en: 'Scope'
	String get scope => 'Scope';

	/// en: 'Scope key'
	String get scopeKey => 'Scope key';

	/// en: 'Currently visible'
	String get currentlyVisible => 'Currently visible';

	/// en: '{count} memory item'
	String items_one({required Object count}) => '${count} memory item';

	/// en: '{count} memory items'
	String items_other({required Object count}) => '${count} memory items';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Delete all'
	String get deleteAll => 'Delete all';
}

// Path: web.memoryInspector.addMem
class TranslationsWebMemoryInspectorAddMemEn {
	TranslationsWebMemoryInspectorAddMemEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Add memory'
	String get title => 'Add memory';

	/// en: 'Manually create a memory. Agents create these automatically via the <1>memory_store</1> MCP tool — this form is for cases where the operator wants to seed a fact without going through an agent.'
	String get description => 'Manually create a memory. Agents create these automatically via the <1>memory_store</1> MCP tool — this form is for cases where the operator wants to seed a fact without going through an agent.';

	/// en: 'Text'
	String get textLabel => 'Text';

	/// en: 'Plain prose. The embedder turns this into a vector at store time; agents will retrieve it via memory_search.'
	String get textPlaceholder => 'Plain prose. The embedder turns this into a vector at store time; agents will retrieve it via memory_search.';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Create'
	String get create => 'Create';
}

// Path: web.memoryInspector.picker
class TranslationsWebMemoryInspectorPickerEn {
	TranslationsWebMemoryInspectorPickerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Pick'
	String get button => 'Pick';

	/// en: 'Pick from saved scope keys or active sessions'
	String get buttonTooltip => 'Pick from saved scope keys or active sessions';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No saved keys or active sessions for {scope}.'
	String empty({required Object scope}) => 'No saved keys or active sessions for ${scope}.';

	/// en: 'Saved memories'
	String get savedHeader => 'Saved memories';

	/// en: 'Active sessions'
	String get activeHeader => 'Active sessions';
}

// Path: web.memoryInspector.migrationBanner
class TranslationsWebMemoryInspectorMigrationBannerEn {
	TranslationsWebMemoryInspectorMigrationBannerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '{count} memory won't appear in searches'
	String headline_one({required Object count}) => '${count} memory won\'t appear in searches';

	/// en: '{count} memories won't appear in searches'
	String headline_other({required Object count}) => '${count} memories won\'t appear in searches';

	/// en: '{summary} — current embedder is <1>{current}</1>. pgvector partitions its similarity index by embedder, so older entries are silent until reembedded.'
	String subtitle({required Object summary, required Object current}) => '${summary} — current embedder is <1>${current}</1>. pgvector partitions its similarity index by embedder, so older entries are silent until reembedded.';

	/// en: '{count} on {name}'
	String summaryItem({required Object count, required Object name}) => '${count} on ${name}';

	/// en: 'Migrate'
	String get migrateButton => 'Migrate';
}

// Path: web.memoryInspector.reembed
class TranslationsWebMemoryInspectorReembedEn {
	TranslationsWebMemoryInspectorReembedEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Reembed memories'
	String get title => 'Reembed memories';

	/// en: 'Recompute vectors for memories stored under a different embedder so they become searchable again.'
	String get description => 'Recompute vectors for memories stored under a different embedder so they become searchable again.';

	/// en: 'Target embedder'
	String get targetEmbedder => 'Target embedder';

	/// en: 'on'
	String get onName => 'on';

	/// en: 'Total to reembed'
	String get totalToReembed => 'Total to reembed';

	/// en: 'Each memory's text gets re-sent to the current embedder; the new vector replaces the old one in place. ID, scope, scope_key, metadata and timestamps are preserved. Search results take effect immediately — no restart needed.'
	String get explainer => 'Each memory\'s text gets re-sent to the current embedder; the new vector replaces the old one in place. ID, scope, scope_key, metadata and timestamps are preserved. Search results take effect immediately — no restart needed.';

	/// en: 'Examined'
	String get reportExamined => 'Examined';

	/// en: 'Reembedded'
	String get reportReembedded => 'Reembedded';

	/// en: 'Failed'
	String get reportFailed => 'Failed';

	/// en: 'From'
	String get reportFrom => 'From';

	/// en: '{count} error'
	String errors_one({required Object count}) => '${count} error';

	/// en: '{count} errors'
	String errors_other({required Object count}) => '${count} errors';

	/// en: 'Done'
	String get done => 'Done';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Reembedding…'
	String get reembedding => 'Reembedding…';

	/// en: 'Reembed {total}'
	String reembedTotal({required Object total}) => 'Reembed ${total}';
}

// Path: web.notes.header
class TranslationsWebNotesHeaderEn {
	TranslationsWebNotesHeaderEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Outline'
	String get outline => 'Outline';

	/// en: 'Show outline'
	String get showOutline => 'Show outline';

	/// en: 'Hide outline'
	String get hideOutline => 'Hide outline';

	/// en: 'Today'
	String get today => 'Today';

	/// en: 'Open or create today's daily note'
	String get todayTooltip => 'Open or create today\'s daily note';

	/// en: 'New'
	String get kNew => 'New';
}

// Path: web.notes.left
class TranslationsWebNotesLeftEn {
	TranslationsWebNotesLeftEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Tree'
	String get tree => 'Tree';

	/// en: 'Tags'
	String get tags => 'Tags';

	/// en: 'Filter notes…'
	String get filterNotes => 'Filter notes…';

	/// en: 'Filter tags…'
	String get filterTags => 'Filter tags…';

	/// en: 'filtered by'
	String get filteredBy => 'filtered by';

	/// en: 'Clear tag filter'
	String get clearTagTooltip => 'Clear tag filter';

	/// en: 'Expand all'
	String get expandAll => 'Expand all';

	/// en: 'Expand every folder'
	String get expandAllTooltip => 'Expand every folder';

	/// en: 'Collapse all'
	String get collapseAll => 'Collapse all';

	/// en: 'Collapse every folder'
	String get collapseAllTooltip => 'Collapse every folder';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: '{visible} / {total} notes'
	String footer({required Object visible, required Object total}) => '${visible} / ${total} notes';
}

// Path: web.notes.tags
class TranslationsWebNotesTagsEn {
	TranslationsWebNotesTagsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No tags in vault yet.'
	String get emptyVault => 'No tags in vault yet.';

	/// en: 'No matches for "{query}".'
	String noMatches({required Object query}) => 'No matches for "${query}".';
}

// Path: web.notes.tree
class TranslationsWebNotesTreeEn {
	TranslationsWebNotesTreeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Vault is empty.'
	String get empty => 'Vault is empty.';
}

// Path: web.notes.outline
class TranslationsWebNotesOutlineEn {
	TranslationsWebNotesOutlineEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Outline'
	String get label => 'Outline';

	/// en: 'No headings in this note. Add <1>## Title</1> lines to populate the outline.'
	String get empty => 'No headings in this note. Add <1>## Title</1> lines to populate the outline.';
}

// Path: web.notes.newNote
class TranslationsWebNotesNewNoteEn {
	TranslationsWebNotesNewNoteEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'New note path (vault-relative, must end .md)'
	String get prompt => 'New note path (vault-relative, must end .md)';

	/// en: 'library/notes-{date}.md'
	String defaultPath({required Object date}) => 'library/notes-${date}.md';

	/// en: 'Path must end in .md'
	String get errorMustEndMd => 'Path must end in .md';

	/// en: 'Note created'
	String get createdToast => 'Note created';

	/// en: 'Create failed'
	String get createFailedToast => 'Create failed';
}

// Path: web.notes.empty
class TranslationsWebNotesEmptyEn {
	TranslationsWebNotesEmptyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No note selected'
	String get title => 'No note selected';

	/// en: 'Pick a note from the tree on the left, jump straight to today's daily log, or create a fresh one. AI-written project docs live under <1>projects/</1>; your personal scratchpads under <3>personal/</3>.'
	String get hint => 'Pick a note from the tree on the left, jump straight to today\'s daily log, or create a fresh one. AI-written project docs live under <1>projects/</1>; your personal scratchpads under <3>personal/</3>.';

	/// en: 'Today's daily note'
	String get today => 'Today\'s daily note';

	/// en: 'New note'
	String get kNew => 'New note';
}

// Path: web.notes.picker
class TranslationsWebNotesPickerEn {
	TranslationsWebNotesPickerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Browse folders'
	String get browseAria => 'Browse folders';

	/// en: '{count} match'
	String matches_one({required Object count}) => '${count} match';

	/// en: '{count} matches'
	String matches_other({required Object count}) => '${count} matches';

	/// en: '{count} folders in vault'
	String foldersInVault({required Object count}) => '${count} folders in vault';

	/// en: 'No existing folder matches. Save anyway to use <1>{value}</1> (lazy-created on first write).'
	String noMatch({required Object value}) => 'No existing folder matches. Save anyway to use <1>${value}</1> (lazy-created on first write).';
}

// Path: web.notes.vaultSync
class TranslationsWebNotesVaultSyncEn {
	TranslationsWebNotesVaultSyncEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Vault sync'
	String get title => 'Vault sync';

	/// en: 'Commit, pull, and push the notes vault as a git repository. Authentication uses your gateway host's git credentials (SSH agent / credential helper).'
	String get description => 'Commit, pull, and push the notes vault as a git repository. Authentication uses your gateway host\'s git credentials (SSH agent / credential helper).';

	/// en: 'Reading vault state…'
	String get reading => 'Reading vault state…';

	late final TranslationsWebNotesVaultSyncInitEn init = TranslationsWebNotesVaultSyncInitEn.internal(_root);
	late final TranslationsWebNotesVaultSyncBranchEn branch = TranslationsWebNotesVaultSyncBranchEn.internal(_root);
	late final TranslationsWebNotesVaultSyncActionEn action = TranslationsWebNotesVaultSyncActionEn.internal(_root);
	late final TranslationsWebNotesVaultSyncCommitEn commit = TranslationsWebNotesVaultSyncCommitEn.internal(_root);
	late final TranslationsWebNotesVaultSyncFileListEn fileList = TranslationsWebNotesVaultSyncFileListEn.internal(_root);
	late final TranslationsWebNotesVaultSyncRemoteEn remote = TranslationsWebNotesVaultSyncRemoteEn.internal(_root);
	late final TranslationsWebNotesVaultSyncHistoryEn history = TranslationsWebNotesVaultSyncHistoryEn.internal(_root);
	late final TranslationsWebNotesVaultSyncConflictEn conflict = TranslationsWebNotesVaultSyncConflictEn.internal(_root);
	late final TranslationsWebNotesVaultSyncAuthEn auth = TranslationsWebNotesVaultSyncAuthEn.internal(_root);
	late final TranslationsWebNotesVaultSyncAutoSyncEn autoSync = TranslationsWebNotesVaultSyncAutoSyncEn.internal(_root);
}

// Path: web.notes.syncBadge
class TranslationsWebNotesSyncBadgeEn {
	TranslationsWebNotesSyncBadgeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Sync'
	String get syncLabel => 'Sync';

	/// en: 'Init'
	String get initLabel => 'Init';

	/// en: 'Vault is not a git repo yet'
	String get initTooltip => 'Vault is not a git repo yet';

	/// en: 'Conflict'
	String get conflictLabel => 'Conflict';

	/// en: 'Vault has unresolved conflicts — click to recover'
	String get conflictTooltip => 'Vault has unresolved conflicts — click to recover';

	/// en: 'sync'
	String get syncFallback => 'sync';

	/// en: 'branch {branch} · {files} changes · {ahead} ahead · {behind} behind'
	String tooltip({required Object branch, required Object files, required Object ahead, required Object behind}) => 'branch ${branch} · ${files} changes · ${ahead} ahead · ${behind} behind';

	/// en: ' · auto-sync on'
	String get tooltipAutoOn => ' · auto-sync on';

	/// en: ' · last error: {error}'
	String tooltipLastError({required Object error}) => ' · last error: ${error}';

	/// en: '—'
	String get branchPlaceholder => '—';
}

// Path: web.activity.filters
class TranslationsWebActivityFiltersEn {
	TranslationsWebActivityFiltersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Integration'
	String get integration => 'Integration';

	/// en: 'Direction'
	String get direction => 'Direction';

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'All integrations'
	String get allIntegrations => 'All integrations';

	/// en: 'All'
	String get all => 'All';

	/// en: 'Inbound'
	String get inbound => 'Inbound';

	/// en: 'Outbound'
	String get outbound => 'Outbound';

	/// en: 'All statuses'
	String get allStatuses => 'All statuses';

	/// en: '2xx success'
	String get status2 => '2xx success';

	/// en: '3xx redirect'
	String get status3 => '3xx redirect';

	/// en: '4xx client error'
	String get status4 => '4xx client error';

	/// en: '5xx server error'
	String get status5 => '5xx server error';
}

// Path: web.activity.table
class TranslationsWebActivityTableEn {
	TranslationsWebActivityTableEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Time'
	String get time => 'Time';

	/// en: 'Integration'
	String get integration => 'Integration';

	/// en: 'Direction'
	String get directionTitle => 'Direction';

	/// en: 'Method'
	String get method => 'Method';

	/// en: 'Path'
	String get path => 'Path';

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'Duration'
	String get duration => 'Duration';

	/// en: 'inbound'
	String get inboundAria => 'inbound';

	/// en: 'outbound'
	String get outboundAria => 'outbound';
}

// Path: web.activity.empty
class TranslationsWebActivityEmptyEn {
	TranslationsWebActivityEmptyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No calls match these filters.'
	String get filtered => 'No calls match these filters.';

	/// en: 'No API calls recorded yet'
	String get title => 'No API calls recorded yet';

	/// en: 'When a third-party app calls opendray with its integration API key, every request is logged here.'
	String get description => 'When a third-party app calls opendray with its integration API key, every request is logged here.';

	/// en: 'Use an existing integration's API key in your third-party app'
	String get stepWithIntegrations => 'Use an existing integration\'s API key in your third-party app';

	/// en: 'Register an integration in Integrations → New'
	String get stepRegister => 'Register an integration in Integrations → New';

	/// en: 'Call any endpoint, e.g. <1>POST /api/v1/sessions</1>'
	String get stepCallEndpoint => 'Call any endpoint, e.g. <1>POST /api/v1/sessions</1>';

	/// en: 'Calls appear here within seconds'
	String get stepAppears => 'Calls appear here within seconds';

	/// en: 'Calls you make from this admin UI are not logged — only integration-attributed traffic is recorded.'
	String get footnote => 'Calls you make from this admin UI are not logged — only integration-attributed traffic is recorded.';
}

// Path: web.activity.events
class TranslationsWebActivityEventsEn {
	TranslationsWebActivityEventsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading events…'
	String get loading => 'Loading events…';

	/// en: 'No events yet.'
	String get empty => 'No events yet.';

	/// en: 'No matching events.'
	String get emptyFiltered => 'No matching events.';

	/// en: 'Load older events'
	String get loadOlder => 'Load older events';

	/// en: 'Today'
	String get today => 'Today';

	/// en: 'Yesterday'
	String get yesterday => 'Yesterday';
}

// Path: web.providers.list
class TranslationsWebProvidersListEn {
	TranslationsWebProvidersListEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Providers'
	String get title => 'Providers';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'disabled'
	String get disabledBadge => 'disabled';

	/// en: 'No provider selected.'
	String get noneSelected => 'No provider selected.';
}

// Path: web.providers.detail
class TranslationsWebProvidersDetailEn {
	TranslationsWebProvidersDetailEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Enabled'
	String get enabled => 'Enabled';

	/// en: 'Disabled'
	String get disabled => 'Disabled';

	/// en: 'Toggle {name}'
	String toggleAria({required Object name}) => 'Toggle ${name}';

	/// en: 'Configuration'
	String get configuration => 'Configuration';

	/// en: 'This provider has no user-configurable fields.'
	String get noConfig => 'This provider has no user-configurable fields.';

	/// en: 'executable:'
	String get executable => 'executable:';

	/// en: 'manifest_hash:'
	String get manifestHash => 'manifest_hash:';

	/// en: 'Reset'
	String get reset => 'Reset';

	/// en: 'Save changes'
	String get save => 'Save changes';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Provider config saved'
	String get savedToast => 'Provider config saved';

	/// en: 'Save failed'
	String get saveFailedToast => 'Save failed';

	/// en: 'Toggle failed'
	String get toggleFailedToast => 'Toggle failed';

	late final TranslationsWebProvidersDetailCapsEn caps = TranslationsWebProvidersDetailCapsEn.internal(_root);
}

// Path: web.providers.configForm
class TranslationsWebProvidersConfigFormEn {
	TranslationsWebProvidersConfigFormEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Select…'
	String get selectPlaceholder => 'Select…';

	/// en: '(default)'
	String get defaultOption => '(default)';

	/// en: 'On'
	String get switchOn => 'On';

	/// en: 'Off'
	String get switchOff => 'Off';

	/// en: 'Show secret'
	String get showSecret => 'Show secret';

	/// en: 'Hide secret'
	String get hideSecret => 'Hide secret';
}

// Path: web.providers.claudeAccounts
class TranslationsWebProvidersClaudeAccountsEn {
	TranslationsWebProvidersClaudeAccountsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Claude accounts'
	String get title => 'Claude accounts';

	/// en: 'Open the multi-account tutorial section'
	String get tutorialTooltip => 'Open the multi-account tutorial section';

	/// en: 'Import local'
	String get importLocal => 'Import local';

	/// en: 'Scan ~/.claude-accounts/ on the gateway host and register any new directories. The button is gateway-host only — see the tutorial.'
	String get importLocalTooltip => 'Scan ~/.claude-accounts/ on the gateway host and register any new directories. The button is gateway-host only — see the tutorial.';

	/// en: 'Nothing to import — accounts already in sync.'
	String get importedNothingToast => 'Nothing to import — accounts already in sync.';

	/// en: 'Imported {count} account from ~/.claude-accounts'
	String importedToast_one({required Object count}) => 'Imported ${count} account from ~/.claude-accounts';

	/// en: 'Imported {count} accounts from ~/.claude-accounts'
	String importedToast_other({required Object count}) => 'Imported ${count} accounts from ~/.claude-accounts';

	/// en: 'Import failed'
	String get importFailedToast => 'Import failed';

	/// en: 'Adding a new account.'
	String get addingTitle => 'Adding a new account.';

	/// en: 'Run on the gateway host:'
	String get addingBodyPrefix => 'Run on the gateway host:';

	/// en: 'opendray's filesystem watcher will register the new directory automatically, or click <1>Import local</1> to scan immediately.'
	String get addingBodySuffix => 'opendray\'s filesystem watcher will register the new directory automatically, or click <1>Import local</1> to scan immediately.';

	/// en: 'Architecture & full guide →'
	String get architectureLink => 'Architecture & full guide →';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No Claude accounts yet. Run the shell command above on the gateway host, then click <1>Import local</1> to scan.'
	String get empty => 'No Claude accounts yet. Run the shell command above on the gateway host, then click <1>Import local</1> to scan.';

	/// en: 'no token yet'
	String get noTokenYet => 'no token yet';

	/// en: 'config_dir:'
	String get configDir => 'config_dir:';

	/// en: 'token_path:'
	String get tokenPath => 'token_path:';

	/// en: 'Toggle failed'
	String get toggleFailedToast => 'Toggle failed';

	/// en: 'Remove account "{name}"?'
	String removeConfirm({required Object name}) => 'Remove account "${name}"?';

	/// en: 'Account removed'
	String get removedToast => 'Account removed';

	/// en: 'Remove failed'
	String get removeFailedToast => 'Remove failed';

	/// en: 'Toggle {name}'
	String toggleAria({required Object name}) => 'Toggle ${name}';

	/// en: 'Remove {name}'
	String removeAria({required Object name}) => 'Remove ${name}';
}

// Path: web.channels.empty
class TranslationsWebChannelsEmptyEn {
	TranslationsWebChannelsEmptyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No channels yet'
	String get title => 'No channels yet';

	/// en: 'Bundled kinds: Telegram · Slack · Discord · Feishu · DingTalk · WeCom. Pick one and paste credentials, or use <1>bridge</1> for a custom platform via WebSocket.'
	String get description => 'Bundled kinds: Telegram · Slack · Discord · Feishu · DingTalk · WeCom. Pick one and paste credentials, or use <1>bridge</1> for a custom platform via WebSocket.';
}

// Path: web.channels.card
class TranslationsWebChannelsCardEn {
	TranslationsWebChannelsCardEn.internal(this._root);

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

	/// en: 'token:'
	String get tokenLabel => 'token:';

	/// en: 'chat_id:'
	String get chatIdLabel => 'chat_id:';

	/// en: 'channel_id:'
	String get channelIdLabel => 'channel_id:';

	/// en: 'notify_on:'
	String get notifyOnLabel => 'notify_on:';

	/// en: 'webhook:'
	String get webhookLabel => 'webhook:';

	/// en: 'Copy webhook URL'
	String get copyWebhookTooltip => 'Copy webhook URL';

	/// en: 'Webhook URL copied'
	String get webhookCopiedToast => 'Webhook URL copied';

	/// en: 'Setup'
	String get setup => 'Setup';

	/// en: 'Show adapter connection details + sample code'
	String get setupTooltip => 'Show adapter connection details + sample code';

	/// en: 'Test'
	String get test => 'Test';

	/// en: 'Channel must be running'
	String get testNotRunningTooltip => 'Channel must be running';

	/// en: 'Bridge channels cannot be tested from the admin — connect an adapter first'
	String get testBridgeTooltip => 'Bridge channels cannot be tested from the admin — connect an adapter first';

	/// en: 'Edit channel'
	String get editAria => 'Edit channel';

	/// en: 'Edit channel config'
	String get editTooltip => 'Edit channel config';

	/// en: 'Delete channel'
	String get deleteAria => 'Delete channel';

	/// en: '(bridge)'
	String get bridgeSuffix => '(bridge)';
}

// Path: web.channels.toasts
class TranslationsWebChannelsToastsEn {
	TranslationsWebChannelsToastsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Test message sent'
	String get testSent => 'Test message sent';

	/// en: 'Test failed'
	String get testFailed => 'Test failed';

	/// en: 'Delete channel {id}?'
	String deleteConfirm({required Object id}) => 'Delete channel ${id}?';

	/// en: 'Channel deleted'
	String get deleted => 'Channel deleted';

	/// en: 'Channel created'
	String get created => 'Channel created';

	/// en: 'Channel updated'
	String get updated => 'Channel updated';
}

// Path: web.channels.dialog
class TranslationsWebChannelsDialogEn {
	TranslationsWebChannelsDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Edit channel'
	String get editTitle => 'Edit channel';

	/// en: 'Register channel'
	String get createTitle => 'Register channel';

	/// en: 'External adapter (Python/Node/...) connects via WebSocket and presents this token.'
	String get descriptionBridge => 'External adapter (Python/Node/...) connects via WebSocket and presents this token.';

	/// en: 'Configure messaging integration.'
	String get descriptionDefault => 'Configure messaging integration.';

	/// en: 'Kind'
	String get kindLabel => 'Kind';

	/// en: '(immutable — delete and recreate to change kind)'
	String get kindImmutable => '(immutable — delete and recreate to change kind)';

	/// en: 'Enabled'
	String get enabledLabel => 'Enabled';

	/// en: ' (accept adapter connections immediately)'
	String get enabledBridgeHint => ' (accept adapter connections immediately)';

	/// en: ' (start receiving webhooks immediately)'
	String get enabledWebhookHint => ' (start receiving webhooks immediately)';

	/// en: ' (start immediately)'
	String get enabledDefaultHint => ' (start immediately)';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Create'
	String get create => 'Create';

	/// en: 'Creating…'
	String get creating => 'Creating…';

	/// en: 'Unknown kind: {kind}'
	String unknownKind({required Object kind}) => 'Unknown kind: ${kind}';

	/// en: 'name is required'
	String get nameRequired => 'name is required';

	/// en: 'token is required'
	String get tokenRequired => 'token is required';

	/// en: 'Topic IDs must be numeric (got {value})'
	String topicIdsNumeric({required Object value}) => 'Topic IDs must be numeric (got ${value})';

	/// en: '{label} is required'
	String fieldRequired({required Object label}) => '${label} is required';

	/// en: 'Cooldown must be a non-negative number of seconds'
	String get cooldownInvalid => 'Cooldown must be a non-negative number of seconds';

	/// en: 'Snippet cap must be a non-negative number'
	String get snippetCapInvalid => 'Snippet cap must be a non-negative number';
}

// Path: web.channels.notifications
class TranslationsWebChannelsNotificationsEn {
	TranslationsWebChannelsNotificationsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Session notifications'
	String get sectionTitle => 'Session notifications';

	/// en: 'Notify on'
	String get notifyOnLabel => 'Notify on';

	/// en: 'Receiving every session event. Click a tag to opt out.'
	String get hintAll => 'Receiving every session event. Click a tag to opt out.';

	/// en: 'No events selected — outbound notifications muted.'
	String get hintNone => 'No events selected — outbound notifications muted.';

	/// en: 'Only {selected} of {total} topics selected.'
	String hintSome({required Object selected, required Object total}) => 'Only ${selected} of ${total} topics selected.';

	/// en: 'Repeat policy'
	String get repeatPolicyLabel => 'Repeat policy';

	/// en: 'Cooldown duration'
	String get cooldownLabel => 'Cooldown duration';

	/// en: 'Replying with non-command text in this chat resets the suppression — opendray forwards your reply to the session's stdin and re-arms the notifier.'
	String get onceReplyHint => 'Replying with non-command text in this chat resets the suppression — opendray forwards your reply to the session\'s stdin and re-arms the notifier.';

	/// en: 'Terminal snippet'
	String get terminalSnippetLabel => 'Terminal snippet';

	/// en: 'Embed the recent terminal screen in idle notifications'
	String get embedSnippetLabel => 'Embed the recent terminal screen in idle notifications';

	/// en: 'When enabled, the idle card includes a code-block snippet of what the user would see in the live web terminal — Claude TUI chrome (status spinner, "bypass permissions" hint, separator lines) is filtered out automatically.'
	String get snippetExplainer => 'When enabled, the idle card includes a code-block snippet of what the user would see in the live web terminal — Claude TUI chrome (status spinner, "bypass permissions" hint, separator lines) is filtered out automatically.';

	late final TranslationsWebChannelsNotificationsModesEn modes = TranslationsWebChannelsNotificationsModesEn.internal(_root);
	late final TranslationsWebChannelsNotificationsCooldownsEn cooldowns = TranslationsWebChannelsNotificationsCooldownsEn.internal(_root);
	late final TranslationsWebChannelsNotificationsSnippetCapsEn snippetCaps = TranslationsWebChannelsNotificationsSnippetCapsEn.internal(_root);
}

// Path: web.channels.bridge
class TranslationsWebChannelsBridgeEn {
	TranslationsWebChannelsBridgeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Bridge name'
	String get nameLabel => 'Bridge name';

	/// en: 'wechat / discord-custom / whatsapp...'
	String get namePlaceholder => 'wechat / discord-custom / whatsapp...';

	/// en: 'Human label for the adapter. Shown in the channels list.'
	String get nameHint => 'Human label for the adapter. Shown in the channels list.';

	/// en: 'Adapter token'
	String get tokenLabel => 'Adapter token';

	/// en: 'Regenerate'
	String get regenerateTooltip => 'Regenerate';

	/// en: 'Copy'
	String get copyTooltip => 'Copy';

	/// en: 'Token copied'
	String get tokenCopiedToast => 'Token copied';

	/// en: 'Adapter authenticates by sending this in the WS register frame (or as <1>X-Bridge-Token</1> header).'
	String get tokenHint => 'Adapter authenticates by sending this in the WS register frame (or as <1>X-Bridge-Token</1> header).';

	/// en: 'Accept capabilities (optional whitelist)'
	String get capsLabel => 'Accept capabilities (optional whitelist)';

	/// en: 'Empty = accept whatever the adapter declares. Selected = only allow these capabilities even if the adapter offers more.'
	String get capsHint => 'Empty = accept whatever the adapter declares. Selected = only allow these capabilities even if the adapter offers more.';

	/// en: 'After <1>Create</1>, the adapter setup dialog opens automatically with the WebSocket URL and copy-pasteable Python / Node / wscat starter code.'
	String get afterCreate => 'After <1>Create</1>, the adapter setup dialog opens automatically with the WebSocket URL and copy-pasteable Python / Node / wscat starter code.';
}

// Path: web.channels.setup
class TranslationsWebChannelsSetupEn {
	TranslationsWebChannelsSetupEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Adapter setup — {name}'
	String title({required Object name}) => 'Adapter setup — ${name}';

	/// en: 'Run an adapter (any language) that connects to opendray over WebSocket using these credentials. opendray will route session notifications and slash-command actions through it.'
	String get description => 'Run an adapter (any language) that connects to opendray over WebSocket using these credentials. opendray will route session notifications and slash-command actions through it.';

	/// en: 'WebSocket URL'
	String get wsUrlLabel => 'WebSocket URL';

	/// en: 'Adapter token'
	String get tokenLabel => 'Adapter token';

	/// en: '<1>Auth:</1> send the token as <3>X-Bridge-Token</3> header, <5>?token=</5> query param, or <7>Authorization: Bearer …</7>. The first WS frame must be <9>{frame}</9>. Full spec: <11>docs/bridge-protocol.md</11> in the repo.'
	String authInfo({required Object frame}) => '<1>Auth:</1> send the token as <3>X-Bridge-Token</3> header, <5>?token=</5> query param, or <7>Authorization: Bearer …</7>. The first WS frame must be <9>${frame}</9>. Full spec: <11>docs/bridge-protocol.md</11> in the repo.';

	/// en: 'Install: <1>pip install websockets</1>. Run: <3>python adapter.py</3>.'
	String get pythonInstall => 'Install: <1>pip install websockets</1>. Run: <3>python adapter.py</3>.';

	/// en: 'Install: <1>npm i ws</1>. Run: <3>node adapter.mjs</3>.'
	String get nodeInstall => 'Install: <1>npm i ws</1>. Run: <3>node adapter.mjs</3>.';

	/// en: 'Install: <1>npm i -g wscat</1>. Once connected, paste the JSON line shown above to register, then send further frames manually.'
	String get wscatInstall => 'Install: <1>npm i -g wscat</1>. Once connected, paste the JSON line shown above to register, then send further frames manually.';

	/// en: 'Close'
	String get close => 'Close';

	/// en: 'Hide'
	String get copyHide => 'Hide';

	/// en: 'Show'
	String get copyShow => 'Show';

	/// en: '{label} copied'
	String copyLabelToast({required Object label}) => '${label} copied';

	/// en: 'Copy'
	String get copyCode => 'Copy';

	/// en: 'Copied'
	String get copied => 'Copied';

	/// en: 'Code copied'
	String get codeCopiedToast => 'Code copied';
}

// Path: web.integrations.tabs
class TranslationsWebIntegrationsTabsEn {
	TranslationsWebIntegrationsTabsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Registered'
	String get registered => 'Registered';

	/// en: 'Reverse proxy'
	String get console => 'Reverse proxy';
}

// Path: web.integrations.empty
class TranslationsWebIntegrationsEmptyEn {
	TranslationsWebIntegrationsEmptyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No integrations yet'
	String get title => 'No integrations yet';

	/// en: 'Register an external app to give it a scoped API key. Its code stays out of this repo.'
	String get description => 'Register an external app to give it a scoped API key. Its code stays out of this repo.';

	/// en: 'Register integration'
	String get register => 'Register integration';
}

// Path: web.integrations.card
class TranslationsWebIntegrationsCardEn {
	TranslationsWebIntegrationsCardEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'managed'
	String get managedBadge => 'managed';

	/// en: 'opendray manages this integration. Editing or rotating its key would orphan running sessions whose mcp.json holds the previous bearer.'
	String get managedTooltip => 'opendray manages this integration. Editing or rotating its key would orphan running sessions whose mcp.json holds the previous bearer.';

	/// en: 'consumer'
	String get consumerBadge => 'consumer';

	/// en: 'Consumer-only integration — no HTTP service to probe'
	String get consumerTooltip => 'Consumer-only integration — no HTTP service to probe';

	/// en: 'disabled'
	String get disabledBadge => 'disabled';

	/// en: 'Consumes opendray's API. No reverse proxy mounted.'
	String get consumerOnlyHint => 'Consumes opendray\'s API. No reverse proxy mounted.';

	/// en: 'last probed {relative}'
	String lastProbed({required Object relative}) => 'last probed ${relative}';

	/// en: 'rotated {relative}'
	String rotated({required Object relative}) => 'rotated ${relative}';

	/// en: 'read-only — opendray bakes its key into every spawn's mcp.json'
	String get managedReadOnly => 'read-only — opendray bakes its key into every spawn\'s mcp.json';

	/// en: 'opendray manages this row. To reset: delete ~/.opendray/memory.key and restart, or delete this row directly via SQL — it'll be re-bootstrapped at next startup.'
	String get managedReadOnlyTooltip => 'opendray manages this row. To reset: delete ~/.opendray/memory.key and restart, or delete this row directly via SQL — it\'ll be re-bootstrapped at next startup.';

	/// en: 'Edit integration'
	String get editAria => 'Edit integration';

	/// en: 'Edit scopes / base URL / version'
	String get editTooltip => 'Edit scopes / base URL / version';

	/// en: 'Rotate key'
	String get rotateKey => 'Rotate key';

	/// en: 'Delete integration'
	String get deleteAria => 'Delete integration';

	/// en: 'Rotate the API key for "{name}"? The current key will stop working immediately.'
	String rotateConfirm({required Object name}) => 'Rotate the API key for "${name}"? The current key will stop working immediately.';

	/// en: 'Delete integration {name}?'
	String deleteConfirm({required Object name}) => 'Delete integration ${name}?';

	/// en: 'Integration removed'
	String get removedToast => 'Integration removed';
}

// Path: web.integrations.register_dialog
class TranslationsWebIntegrationsRegisterDialogEn {
	TranslationsWebIntegrationsRegisterDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Register integration'
	String get title => 'Register integration';

	/// en: 'Issues a one-time API key. Copy it before closing — opendray never displays the plaintext again.'
	String get description => 'Issues a one-time API key. Copy it before closing — opendray never displays the plaintext again.';

	/// en: 'Name'
	String get nameLabel => 'Name';

	/// en: 'PetTracker'
	String get namePlaceholder => 'PetTracker';

	/// en: 'Leave the next two fields blank for a <1>consumer-only</1> integration (third-party app that calls opendray's API but doesn't expose its own service). Fill both for a <3>reverse-proxy</3> integration.'
	String get modeHint => 'Leave the next two fields blank for a <1>consumer-only</1> integration (third-party app that calls opendray\'s API but doesn\'t expose its own service). Fill both for a <3>reverse-proxy</3> integration.';

	/// en: 'Base URL'
	String get baseUrlLabel => 'Base URL';

	/// en: '(optional)'
	String get optionalSuffix => '(optional)';

	/// en: 'http://192.168.3.42:8080'
	String get baseUrlPlaceholder => 'http://192.168.3.42:8080';

	/// en: 'Route prefix'
	String get routePrefixLabel => 'Route prefix';

	/// en: 'pet-tracker'
	String get routePrefixPlaceholder => 'pet-tracker';

	/// en: 'Reachable at <1>/api/v1/proxy/{prefix}/*</1>.'
	String routePrefixHint({required Object prefix}) => 'Reachable at <1>/api/v1/proxy/${prefix}/*</1>.';

	/// en: '<prefix>'
	String get routePrefixPlaceholderToken => '<prefix>';

	/// en: 'Version (optional)'
	String get versionLabel => 'Version (optional)';

	/// en: '0.1.0'
	String get versionPlaceholder => '0.1.0';

	/// en: 'Scopes'
	String get scopesLabel => 'Scopes';

	/// en: 'Pick the API surface this integration is allowed to call. Each toggle maps to a Bearer-token claim — opendray rejects requests that touch endpoints outside the granted set.'
	String get scopesIntro => 'Pick the API surface this integration is allowed to call. Each toggle maps to a Bearer-token claim — opendray rejects requests that touch endpoints outside the granted set.';

	/// en: 'Name is required.'
	String get errorNameRequired => 'Name is required.';

	/// en: 'base_url and route_prefix go together. Set both for a reverse-proxy integration, leave both blank for a consumer-only integration.'
	String get errorBothOrNeither => 'base_url and route_prefix go together. Set both for a reverse-proxy integration, leave both blank for a consumer-only integration.';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Register'
	String get submit => 'Register';

	/// en: 'Registering…'
	String get submitting => 'Registering…';
}

// Path: web.integrations.reveal
class TranslationsWebIntegrationsRevealEn {
	TranslationsWebIntegrationsRevealEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'API key issued'
	String get titleIssued => 'API key issued';

	/// en: 'API key rotated'
	String get titleRotated => 'API key rotated';

	/// en: 'This is the only time the plaintext key will be shown. Copy it now and update every consumer app — the previous key (if any) no longer authenticates.'
	String get description => 'This is the only time the plaintext key will be shown. Copy it now and update every consumer app — the previous key (if any) no longer authenticates.';

	/// en: 'Discard new key'
	String get discardAria => 'Discard new key';

	/// en: 'Discard the new key (rotation already happened — old key is gone too)'
	String get discardTooltip => 'Discard the new key (rotation already happened — old key is gone too)';

	/// en: 'Discard the new key? Rotation has already invalidated the old key — discarding means you have NO working key for this integration until you rotate again.'
	String get discardConfirm => 'Discard the new key? Rotation has already invalidated the old key — discarding means you have NO working key for this integration until you rotate again.';

	/// en: 'Copy'
	String get copy => 'Copy';

	/// en: 'Copied'
	String get copied => 'Copied';

	/// en: '<1>Update every consumer app with this new key.</1> The previous key has been invalidated server-side and will return <3>401 unauthorized</3> on the next request.'
	String get updateHint => '<1>Update every consumer app with this new key.</1> The previous key has been invalidated server-side and will return <3>401 unauthorized</3> on the next request.';

	/// en: 'I have copied the key and will update my consumer apps. I understand opendray will not display it again.'
	String get acknowledge => 'I have copied the key and will update my consumer apps. I understand opendray will not display it again.';

	/// en: 'Discard'
	String get discard => 'Discard';

	/// en: 'Done'
	String get done => 'Done';
}

// Path: web.integrations.edit_dialog
class TranslationsWebIntegrationsEditDialogEn {
	TranslationsWebIntegrationsEditDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Edit integration · {name}'
	String title({required Object name}) => 'Edit integration · ${name}';

	/// en: 'Change scopes, version, or base URL. Name and route prefix are immutable — delete + re-register if you need to change those.'
	String get description => 'Change scopes, version, or base URL. Name and route prefix are immutable — delete + re-register if you need to change those.';

	/// en: 'Name'
	String get nameLabel => 'Name';

	/// en: 'Route prefix'
	String get routePrefixLabel => 'Route prefix';

	/// en: '(consumer-only)'
	String get consumerOnlyLabel => '(consumer-only)';

	/// en: 'Base URL'
	String get baseUrlLabel => 'Base URL';

	/// en: '(consumer-only — leave blank)'
	String get baseUrlConsumerSuffix => '(consumer-only — leave blank)';

	/// en: '(reverse-proxy target)'
	String get baseUrlProxySuffix => '(reverse-proxy target)';

	/// en: '(blank — this integration consumes opendray's API)'
	String get baseUrlConsumerPlaceholder => '(blank — this integration consumes opendray\'s API)';

	/// en: 'http://127.0.0.1:8080'
	String get baseUrlProxyPlaceholder => 'http://127.0.0.1:8080';

	/// en: 'This is a consumer-only integration. Changing base URL here would also require a route prefix; do that with delete + re-register.'
	String get consumerHint => 'This is a consumer-only integration. Changing base URL here would also require a route prefix; do that with delete + re-register.';

	/// en: 'Version'
	String get versionLabel => 'Version';

	/// en: '0.1.0'
	String get versionPlaceholder => '0.1.0';

	/// en: 'Scopes'
	String get scopesLabel => 'Scopes';

	/// en: 'Trim or widen the API surface this integration's API key authorises. Live tokens are unaffected — the new scope set takes effect on the next request.'
	String get scopesIntro => 'Trim or widen the API surface this integration\'s API key authorises. Live tokens are unaffected — the new scope set takes effect on the next request.';

	/// en: 'Switching between consumer-only and reverse-proxy mode requires deleting the integration and re-registering — name and route_prefix can't change in place.'
	String get errorModeSwitch => 'Switching between consumer-only and reverse-proxy mode requires deleting the integration and re-registering — name and route_prefix can\'t change in place.';

	/// en: 'Integration updated'
	String get updatedToast => 'Integration updated';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Save changes'
	String get save => 'Save changes';
}

// Path: web.integrations.proxy
class TranslationsWebIntegrationsProxyEn {
	TranslationsWebIntegrationsProxyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No integrations registered'
	String get emptyTitle => 'No integrations registered';

	/// en: 'Register an integration first; the console proxies through /api/v1/proxy/{prefix}/* using the admin token.'
	String emptyDescription({required Object prefix}) => 'Register an integration first; the console proxies through /api/v1/proxy/${prefix}/* using the admin token.';

	/// en: 'Target'
	String get targetLabel => 'Target';

	/// en: 'Select integration…'
	String get selectPlaceholder => 'Select integration…';

	/// en: 'base:'
	String get baseLabel => 'base:';

	/// en: 'History'
	String get history => 'History';

	/// en: 'no past requests for this integration'
	String get historyEmpty => 'no past requests for this integration';

	/// en: 'Send'
	String get send => 'Send';

	/// en: 'Sending…'
	String get sending => 'Sending…';

	/// en: 'Extra headers (one per line, Name: Value)'
	String get extraHeadersLabel => 'Extra headers (one per line, Name: Value)';

	/// en: 'Body'
	String get bodyLabel => 'Body';

	/// en: 'Headers'
	String get headers => 'Headers';

	/// en: 'Body'
	String get body => 'Body';

	/// en: '(empty)'
	String get emptyBody => '(empty)';

	/// en: 'request failed'
	String get requestFailed => 'request failed';

	/// en: 'Send a request to see the upstream response.'
	String get stubText => 'Send a request to see the upstream response.';

	/// en: 'opendray injects <1>X-Integration-ID</1> and strips your <3>Authorization</3> header.'
	String get stubInjects => 'opendray injects <1>X-Integration-ID</1> and strips your <3>Authorization</3> header.';

	/// en: '<prefix>'
	String get prefixPlaceholder => '<prefix>';
}

// Path: web.plugins.common
class TranslationsWebPluginsCommonEn {
	TranslationsWebPluginsCommonEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Edit'
	String get edit => 'Edit';

	/// en: 'Add'
	String get add => 'Add';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Create'
	String get create => 'Create';
}

// Path: web.plugins.mcp
class TranslationsWebPluginsMcpEn {
	TranslationsWebPluginsMcpEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'MCP servers'
	String get title => 'MCP servers';

	/// en: 'Model Context Protocol servers injected into every spawn (claude / codex). Vault entries live at <1>~/.opendray/vault/mcp/&lt;id&gt;/mcp.json</1>; secrets (referenced as <3>${KEY}</3> in env / headers) come from the <5>MCP secrets</5> section below.'
	String description({required Object KEY}) => 'Model Context Protocol servers injected into every spawn (claude / codex). Vault entries live at <1>~/.opendray/vault/mcp/&lt;id&gt;/mcp.json</1>; secrets (referenced as <3>\$${KEY}</3> in env / headers) come from the <5>MCP secrets</5> section below.';

	/// en: 'New server'
	String get newServer => 'New server';

	/// en: 'No MCP servers yet. Add one to expose extra tools to your agent sessions.'
	String get empty => 'No MCP servers yet. Add one to expose extra tools to your agent sessions.';

	late final TranslationsWebPluginsMcpColumnsEn columns = TranslationsWebPluginsMcpColumnsEn.internal(_root);

	/// en: 'no url'
	String get noUrl => 'no url';

	/// en: 'no command'
	String get noCommand => 'no command';

	/// en: 'Delete MCP server "{id}"?'
	String deleteConfirm({required Object id}) => 'Delete MCP server "${id}"?';

	/// en: 'MCP server removed'
	String get removedToast => 'MCP server removed';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	/// en: 'Toggle failed'
	String get toggleFailedToast => 'Toggle failed';

	late final TranslationsWebPluginsMcpEditorEn editor = TranslationsWebPluginsMcpEditorEn.internal(_root);
}

// Path: web.plugins.mcpSecrets
class TranslationsWebPluginsMcpSecretsEn {
	TranslationsWebPluginsMcpSecretsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'MCP secrets'
	String get title => 'MCP secrets';

	/// en: 'encrypted'
	String get encryptedBadge => 'encrypted';

	/// en: 'plaintext'
	String get plaintextBadge => 'plaintext';

	/// en: 'AES-GCM encrypted on disk; key stored in OS keychain'
	String get encryptedTooltip => 'AES-GCM encrypted on disk; key stored in OS keychain';

	/// en: 'OS keychain unavailable — file is plaintext on disk. Check the gateway log.'
	String get plaintextTooltip => 'OS keychain unavailable — file is plaintext on disk. Check the gateway log.';

	/// en: 'Values referenced from <1>${KEY}</1> placeholders in any <3>mcp.json</3> get substituted at spawn time. <5>Saved values are never returned over the API</5> — you can overwrite or delete them but not read them back.'
	String description({required Object KEY}) => 'Values referenced from <1>\$${KEY}</1> placeholders in any <3>mcp.json</3> get substituted at spawn time. <5>Saved values are never returned over the API</5> — you can overwrite or delete them but not read them back.';

	/// en: ' Stored at <1>{path}</1>.'
	String descriptionStored({required Object path}) => ' Stored at <1>${path}</1>.';

	/// en: 'Add secret'
	String get addSecret => 'Add secret';

	/// en: 'No secrets stored. Add one to start referencing it as <1>${KEY}</1> in your MCP server configs.'
	String empty({required Object KEY}) => 'No secrets stored. Add one to start referencing it as <1>\$${KEY}</1> in your MCP server configs.';

	late final TranslationsWebPluginsMcpSecretsColumnsEn columns = TranslationsWebPluginsMcpSecretsColumnsEn.internal(_root);

	/// en: 'Overwrite the stored value'
	String get editTooltip => 'Overwrite the stored value';

	/// en: 'Delete secret "{key}"? Any mcp.json that references ${key} will fall back to the literal placeholder until you set a new value.'
	String deleteConfirm({required Object key}) => 'Delete secret "${key}"? Any mcp.json that references \$${key} will fall back to the literal placeholder until you set a new value.';

	/// en: 'Secret removed'
	String get removedToast => 'Secret removed';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	late final TranslationsWebPluginsMcpSecretsEditorEn editor = TranslationsWebPluginsMcpSecretsEditorEn.internal(_root);
}

// Path: web.plugins.skills
class TranslationsWebPluginsSkillsEn {
	TranslationsWebPluginsSkillsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Agent skills'
	String get title => 'Agent skills';

	/// en: 'Reusable capabilities injected into Claude sessions as a Tier 1 index — the agent loads full SKILL.md on demand via <1>opendray skill describe &lt;id&gt;</1>. Built-ins ship in the binary but can be <3>customized</3> — your edits land at <5>~/.opendray/vault/skills/&lt;id&gt;/SKILL.md</5> and override the embedded version. Use Reset to revert.'
	String get description => 'Reusable capabilities injected into Claude sessions as a Tier 1 index — the agent loads full SKILL.md on demand via <1>opendray skill describe &lt;id&gt;</1>. Built-ins ship in the binary but can be <3>customized</3> — your edits land at <5>~/.opendray/vault/skills/&lt;id&gt;/SKILL.md</5> and override the embedded version. Use Reset to revert.';

	/// en: 'New skill'
	String get newSkill => 'New skill';

	/// en: 'No skills found.'
	String get empty => 'No skills found.';

	late final TranslationsWebPluginsSkillsColumnsEn columns = TranslationsWebPluginsSkillsColumnsEn.internal(_root);

	/// en: 'no description'
	String get noDescription => 'no description';

	/// en: 'builtin'
	String get builtinBadge => 'builtin';

	/// en: 'Embedded in the opendray binary — click Customize to override in your vault'
	String get builtinTooltip => 'Embedded in the opendray binary — click Customize to override in your vault';

	/// en: 'vault'
	String get vaultBadge => 'vault';

	/// en: 'overrides builtin'
	String get overridesBuiltin => 'overrides builtin';

	/// en: 'This vault skill overrides the built-in version of the same id'
	String get overridesBuiltinTooltip => 'This vault skill overrides the built-in version of the same id';

	/// en: 'Customize'
	String get customize => 'Customize';

	/// en: 'Open the SKILL.md and save changes as a vault override'
	String get customizeTooltip => 'Open the SKILL.md and save changes as a vault override';

	/// en: 'Edit this vault skill'
	String get editTooltip => 'Edit this vault skill';

	/// en: 'Delete vault override and fall back to the built-in version'
	String get resetTooltip => 'Delete vault override and fall back to the built-in version';

	/// en: 'Reset'
	String get reset => 'Reset';

	/// en: 'Reset "{id}" to the built-in version? This deletes your vault SKILL.md and falls back to the embedded copy.'
	String resetConfirm({required Object id}) => 'Reset "${id}" to the built-in version? This deletes your vault SKILL.md and falls back to the embedded copy.';

	/// en: 'Delete skill "{id}" from your vault? This removes the SKILL.md file.'
	String deleteConfirm({required Object id}) => 'Delete skill "${id}" from your vault? This removes the SKILL.md file.';

	/// en: 'Skill removed'
	String get removedToast => 'Skill removed';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	late final TranslationsWebPluginsSkillsEditorEn editor = TranslationsWebPluginsSkillsEditorEn.internal(_root);
}

// Path: web.plugins.customTasks
class TranslationsWebPluginsCustomTasksEn {
	TranslationsWebPluginsCustomTasksEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Custom tasks'
	String get title => 'Custom tasks';

	/// en: 'Click-to-run shortcuts surfaced in the Tasks tab. Leave cwd blank for global tasks visible in every session, or pin to an absolute path to scope.'
	String get description => 'Click-to-run shortcuts surfaced in the Tasks tab. Leave cwd blank for global tasks visible in every session, or pin to an absolute path to scope.';

	/// en: 'Add task'
	String get addTask => 'Add task';

	/// en: 'No custom tasks yet.'
	String get empty => 'No custom tasks yet.';

	late final TranslationsWebPluginsCustomTasksColumnsEn columns = TranslationsWebPluginsCustomTasksColumnsEn.internal(_root);

	/// en: 'global'
	String get globalScope => 'global';

	/// en: 'Delete custom task "{name}"?'
	String deleteConfirm({required Object name}) => 'Delete custom task "${name}"?';

	/// en: 'Task removed'
	String get removedToast => 'Task removed';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	late final TranslationsWebPluginsCustomTasksDialogEn dialog = TranslationsWebPluginsCustomTasksDialogEn.internal(_root);
}

// Path: web.plugins.gitHosts
class TranslationsWebPluginsGitHostsEn {
	TranslationsWebPluginsGitHostsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Git hosts'
	String get title => 'Git hosts';

	/// en: 'One token per host — used by the Git tab to fetch pull requests <1>and by the Notes vault sync</1> when its remote uses HTTPS to a private repo on the same host. GitHub.com, self-hosted GitHub Enterprise, Gitea, and GitLab are supported.'
	String get description => 'One token per host — used by the Git tab to fetch pull requests <1>and by the Notes vault sync</1> when its remote uses HTTPS to a private repo on the same host. GitHub.com, self-hosted GitHub Enterprise, Gitea, and GitLab are supported.';

	/// en: 'Add host'
	String get addHost => 'Add host';

	/// en: 'No git hosts configured. Add one to enable the PR list in the inspector's Git tab.'
	String get empty => 'No git hosts configured.\nAdd one to enable the PR list in the inspector\'s Git tab.';

	late final TranslationsWebPluginsGitHostsColumnsEn columns = TranslationsWebPluginsGitHostsColumnsEn.internal(_root);

	/// en: 'enabled'
	String get statusEnabled => 'enabled';

	/// en: 'disabled'
	String get statusDisabled => 'disabled';

	/// en: 'Remove git host {host}? PR queries against this host will stop working.'
	String deleteConfirm({required Object host}) => 'Remove git host ${host}? PR queries against this host will stop working.';

	/// en: 'Git host removed'
	String get removedToast => 'Git host removed';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	late final TranslationsWebPluginsGitHostsDialogEn dialog = TranslationsWebPluginsGitHostsDialogEn.internal(_root);
}

// Path: web.backups.tabs
class TranslationsWebBackupsTabsEn {
	TranslationsWebBackupsTabsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backups'
	String get backups => 'Backups';

	/// en: 'Schedules'
	String get schedules => 'Schedules';

	/// en: 'Targets'
	String get targets => 'Targets';
}

// Path: web.backups.inventory
class TranslationsWebBackupsInventoryEn {
	TranslationsWebBackupsInventoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'What's in a backup?'
	String get title => 'What\'s in a backup?';

	/// en: '{rows} rows across {tables} tables'
	String summary({required Object rows, required Object tables}) => '${rows} rows across ${tables} tables';

	/// en: 'Each backup is a <1>pg_dump --format=custom</1> of every table below, plus <3>manifest.json</3> and (optionally) <5>config.toml</5>. Counts are live; the bundle captures whatever's there at backup time.'
	String get description => 'Each backup is a <1>pg_dump --format=custom</1> of every table below, plus <3>manifest.json</3> and (optionally) <5>config.toml</5>. Counts are live; the bundle captures whatever\'s there at backup time.';

	/// en: 'Failed to load inventory'
	String get loadFailedToast => 'Failed to load inventory';

	/// en: 'rows'
	String get rowsLabel => 'rows';
}

// Path: web.backups.restart
class TranslationsWebBackupsRestartEn {
	TranslationsWebBackupsRestartEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Restart opendray to activate backups'
	String get title => 'Restart opendray to activate backups';

	/// en: 'Your passphrase is saved. The gateway only loads it at startup, so the feature stays off until you bounce the process.'
	String get description => 'Your passphrase is saved. The gateway only loads it at startup, so the feature stays off until you bounce the process.';

	/// en: 'Key file:'
	String get keyFile => 'Key file:';

	/// en: 'Configured via:'
	String get configuredVia => 'Configured via:';

	/// en: 'OPENDRAY_BACKUP_KEY env var'
	String get envVar => 'OPENDRAY_BACKUP_KEY env var';

	/// en: 'Check again'
	String get checkAgain => 'Check again';
}

// Path: web.backups.setup
class TranslationsWebBackupsSetupEn {
	TranslationsWebBackupsSetupEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Set up backups'
	String get title => 'Set up backups';

	/// en: 'Choose a master passphrase. opendray uses it to encrypt every backup blob. <1>Lose it and your backups become unrecoverable</1>, so save it in a password manager (Vaultwarden, 1Password, …) before continuing.'
	String get description => 'Choose a master passphrase. opendray uses it to encrypt every backup blob. <1>Lose it and your backups become unrecoverable</1>, so save it in a password manager (Vaultwarden, 1Password, …) before continuing.';

	/// en: 'Generate'
	String get generate => 'Generate';

	/// en: 'Paste my own'
	String get pasteOwn => 'Paste my own';

	/// en: '256-bit random key'
	String get generateTitle => '256-bit random key';

	/// en: 'Server generates a cryptographically random passphrase and shows it once. You must copy it before continuing — there is no recovery path.'
	String get generateHint => 'Server generates a cryptographically random passphrase and shows it once. You must copy it before continuing — there is no recovery path.';

	/// en: 'Your passphrase'
	String get pasteLabel => 'Your passphrase';

	/// en: 'At least 20 characters'
	String get pastePlaceholder => 'At least 20 characters';

	/// en: 'Recommended: 40+ characters from a password manager.'
	String get pasteHint => 'Recommended: 40+ characters from a password manager.';

	/// en: 'Saves to:'
	String get savesTo => 'Saves to:';

	/// en: 'Saving…'
	String get saving => 'Saving…';

	/// en: 'Generate and save'
	String get generateAndSave => 'Generate and save';

	/// en: 'Save'
	String get save => 'Save';
}

// Path: web.backups.generated
class TranslationsWebBackupsGeneratedEn {
	TranslationsWebBackupsGeneratedEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Save this passphrase NOW'
	String get title => 'Save this passphrase NOW';

	/// en: 'This is shown <1>once</1>. It will not be retrievable from opendray or anywhere else. Copy it into a password manager before continuing.'
	String get description => 'This is shown <1>once</1>. It will not be retrievable from opendray or anywhere else. Copy it into a password manager before continuing.';

	/// en: 'Copy'
	String get copy => 'Copy';

	/// en: 'Passphrase copied to clipboard'
	String get copiedToast => 'Passphrase copied to clipboard';

	/// en: 'Copy failed — select and copy manually'
	String get copyFailedToast => 'Copy failed — select and copy manually';

	/// en: 'Saved to:'
	String get savedTo => 'Saved to:';

	/// en: 'I have saved this passphrase to my password manager'
	String get ack => 'I have saved this passphrase to my password manager';

	/// en: 'Continue'
	String get kContinue => 'Continue';
}

// Path: web.backups.status
class TranslationsWebBackupsStatusEn {
	TranslationsWebBackupsStatusEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Key fingerprint:'
	String get keyFingerprint => 'Key fingerprint:';

	/// en: 'pg_dump:'
	String get pgDump => 'pg_dump:';

	/// en: 'unavailable'
	String get pgDumpUnavailable => 'unavailable';

	/// en: 'Backups can't run until pg_dump is on PATH (or its absolute path is set in <1>backup.pg_dump_path</1>). Install <3>postgresql-client</3> matching your server's major version and restart.'
	String get pgDumpHint => 'Backups can\'t run until pg_dump is on PATH (or its absolute path is set in <1>backup.pg_dump_path</1>). Install <3>postgresql-client</3> matching your server\'s major version and restart.';
}

// Path: web.backups.backupsTab
class TranslationsWebBackupsBackupsTabEn {
	TranslationsWebBackupsBackupsTabEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backup now'
	String get backupNow => 'Backup now';

	/// en: 'Triggering…'
	String get triggering => 'Triggering…';

	/// en: 'include config.toml'
	String get includeConfig => 'include config.toml';

	/// en: 'Restore from file'
	String get restoreFromFile => 'Restore from file';

	/// en: 'Refresh'
	String get refresh => 'Refresh';

	/// en: 'Backup queued'
	String get queuedToast => 'Backup queued';

	/// en: 'Trigger failed'
	String get triggerFailedToast => 'Trigger failed';

	/// en: 'Failed to list backups'
	String get listFailedToast => 'Failed to list backups';

	/// en: 'Delete backup {id}? The blob is removed from its target.'
	String deleteConfirm({required Object id}) => 'Delete backup ${id}? The blob is removed from its target.';

	/// en: 'Backup deleted'
	String get deletedToast => 'Backup deleted';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	/// en: 'No backups yet. Click "Backup now" above to take the first one.'
	String get empty => 'No backups yet. Click "Backup now" above to take the first one.';

	late final TranslationsWebBackupsBackupsTabColumnsEn columns = TranslationsWebBackupsBackupsTabColumnsEn.internal(_root);

	/// en: 'Download'
	String get downloadTooltip => 'Download';

	/// en: 'Delete'
	String get deleteTooltip => 'Delete';
}

// Path: web.backups.restore
class TranslationsWebBackupsRestoreEn {
	TranslationsWebBackupsRestoreEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Restore from backup bundle'
	String get title => 'Restore from backup bundle';

	/// en: 'Encrypted bundle (.tar.gz.enc)'
	String get bundleLabel => 'Encrypted bundle (.tar.gz.enc)';

	/// en: 'Target database DSN'
	String get targetDsnLabel => 'Target database DSN';

	/// en: '(blank = opendray's own DB — DANGEROUS)'
	String get targetDsnHint => '(blank = opendray\'s own DB — DANGEROUS)';

	/// en: 'postgres://user:pass@host:5432/dbname'
	String get targetDsnPlaceholder => 'postgres://user:pass@host:5432/dbname';

	/// en: '--clean --if-exists (drop existing schema first; required when restoring over a populated DB)'
	String get cleanLabel => '--clean --if-exists (drop existing schema first; required when restoring over a populated DB)';

	/// en: 'Audit note (optional)'
	String get auditNoteLabel => 'Audit note (optional)';

	/// en: 'Reason for restore — appears in slog'
	String get auditNotePlaceholder => 'Reason for restore — appears in slog';

	/// en: 'You're restoring into <1>opendray's own database</1>. With "--clean" enabled this drops every table and replays the backup verbatim — irreversible. Type <3>I understand</3> to proceed.'
	String get ownDbWarning => 'You\'re restoring into <1>opendray\'s own database</1>. With "--clean" enabled this drops every table and replays the backup verbatim — irreversible. Type <3>I understand</3> to proceed.';

	/// en: 'I understand'
	String get confirmPlaceholder => 'I understand';

	/// en: 'I understand'
	String get confirmSentinel => 'I understand';

	/// en: 'pg_restore output (last 8 KiB)'
	String get pgRestoreOutput => 'pg_restore output (last 8 KiB)';

	/// en: '(no pg_restore output)'
	String get noPgRestoreOutput => '(no pg_restore output)';

	/// en: 'Pick a bundle file first'
	String get pickFileToast => 'Pick a bundle file first';

	/// en: 'Restore succeeded'
	String get succeededToast => 'Restore succeeded';

	/// en: '{bytes} replayed from manifest {id}'
	String replayedDescription({required Object bytes, required Object id}) => '${bytes} replayed from manifest ${id}';

	/// en: 'Restore failed'
	String get failedToast => 'Restore failed';

	/// en: 'Restoring…'
	String get restoring => 'Restoring…';

	/// en: 'Restore'
	String get restore => 'Restore';
}

// Path: web.backups.schedulesTab
class TranslationsWebBackupsSchedulesTabEn {
	TranslationsWebBackupsSchedulesTabEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Recurring backups. The scheduler polls every 30s and runs the oldest due schedule.'
	String get description => 'Recurring backups. The scheduler polls every 30s and runs the oldest due schedule.';

	/// en: 'New schedule'
	String get newSchedule => 'New schedule';

	/// en: 'Failed to load schedules'
	String get loadFailedToast => 'Failed to load schedules';

	/// en: 'Delete schedule {id}?'
	String deleteConfirm({required Object id}) => 'Delete schedule ${id}?';

	/// en: 'Schedule deleted'
	String get deletedToast => 'Schedule deleted';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	/// en: 'Toggle failed'
	String get toggleFailedToast => 'Toggle failed';

	/// en: 'No schedules. Add one to take automatic recurring backups.'
	String get empty => 'No schedules. Add one to take automatic recurring backups.';

	late final TranslationsWebBackupsSchedulesTabColumnsEn columns = TranslationsWebBackupsSchedulesTabColumnsEn.internal(_root);

	/// en: '{count} backups'
	String keepCount({required Object count}) => '${count} backups';

	/// en: 'Delete'
	String get deleteTooltip => 'Delete';
}

// Path: web.backups.newSchedule
class TranslationsWebBackupsNewScheduleEn {
	TranslationsWebBackupsNewScheduleEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'New backup schedule'
	String get title => 'New backup schedule';

	/// en: 'Target'
	String get targetLabel => 'Target';

	/// en: 'Every (hours)'
	String get everyHoursLabel => 'Every (hours)';

	/// en: 'Keep last N'
	String get keepLastNLabel => 'Keep last N';

	/// en: 'Enable immediately'
	String get enableImmediately => 'Enable immediately';

	/// en: 'Schedule created'
	String get createdToast => 'Schedule created';

	/// en: 'Create failed'
	String get createFailedToast => 'Create failed';

	/// en: 'Creating…'
	String get creating => 'Creating…';

	/// en: 'Create'
	String get create => 'Create';
}

// Path: web.backups.targetsTab
class TranslationsWebBackupsTargetsTabEn {
	TranslationsWebBackupsTargetsTabEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Storage destinations. v1 supports <1>local</1> (disk on the opendray host) and <3>smb</3> (any SMB / CIFS share, e.g. UNAS or Synology).'
	String get description => 'Storage destinations. v1 supports <1>local</1> (disk on the opendray host) and <3>smb</3> (any SMB / CIFS share, e.g. UNAS or Synology).';

	/// en: 'New target'
	String get newTarget => 'New target';

	/// en: 'Failed to list targets'
	String get listFailedToast => 'Failed to list targets';

	/// en: 'Delete target {id}? Schedules referencing it will block the delete.'
	String deleteConfirm({required Object id}) => 'Delete target ${id}? Schedules referencing it will block the delete.';

	/// en: 'Target deleted'
	String get deletedToast => 'Target deleted';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	/// en: 'Connection OK'
	String get connectionOkToast => 'Connection OK';

	/// en: 'Connection failed'
	String get connectionFailedToast => 'Connection failed';

	/// en: 'Test failed'
	String get testFailedToast => 'Test failed';

	late final TranslationsWebBackupsTargetsTabColumnsEn columns = TranslationsWebBackupsTargetsTabColumnsEn.internal(_root);

	/// en: 'on'
	String get on => 'on';

	/// en: 'off'
	String get off => 'off';

	/// en: 'Test'
	String get test => 'Test';

	/// en: 'Testing…'
	String get testing => 'Testing…';

	/// en: 'Delete'
	String get deleteTooltip => 'Delete';
}

// Path: web.backups.targetEditor
class TranslationsWebBackupsTargetEditorEn {
	TranslationsWebBackupsTargetEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'New backup target'
	String get title => 'New backup target';

	/// en: 'Where do you want to back up to?'
	String get kindPicker => 'Where do you want to back up to?';

	/// en: 'ID (optional)'
	String get idLabel => 'ID (optional)';

	/// en: 'auto-generated if blank, e.g. tgt_xxx'
	String get idPlaceholder => 'auto-generated if blank, e.g. tgt_xxx';

	/// en: 'Target created'
	String get createdToast => 'Target created';

	/// en: 'Create failed'
	String get createFailedToast => 'Create failed';

	/// en: 'Creating…'
	String get creating => 'Creating…';

	/// en: 'Create target'
	String get create => 'Create target';

	/// en: 'Enable immediately (otherwise saved as disabled — useful for "configure now, turn on later")'
	String get enableImmediately => 'Enable immediately (otherwise saved as disabled — useful for "configure now, turn on later")';

	late final TranslationsWebBackupsTargetEditorLocalEn local = TranslationsWebBackupsTargetEditorLocalEn.internal(_root);
	late final TranslationsWebBackupsTargetEditorSmbEn smb = TranslationsWebBackupsTargetEditorSmbEn.internal(_root);
	late final TranslationsWebBackupsTargetEditorS3En s3 = TranslationsWebBackupsTargetEditorS3En.internal(_root);
	late final TranslationsWebBackupsTargetEditorWebdavEn webdav = TranslationsWebBackupsTargetEditorWebdavEn.internal(_root);
	late final TranslationsWebBackupsTargetEditorSftpEn sftp = TranslationsWebBackupsTargetEditorSftpEn.internal(_root);
	late final TranslationsWebBackupsTargetEditorRcloneEn rclone = TranslationsWebBackupsTargetEditorRcloneEn.internal(_root);
}

// Path: web.serverSettings.sections
class TranslationsWebServerSettingsSectionsEn {
	TranslationsWebServerSettingsSectionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebServerSettingsSectionsGeneralEn general = TranslationsWebServerSettingsSectionsGeneralEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsLoggingEn logging = TranslationsWebServerSettingsSectionsLoggingEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsSessionsEn sessions = TranslationsWebServerSettingsSectionsSessionsEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsVaultEn vault = TranslationsWebServerSettingsSectionsVaultEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsMcpEn mcp = TranslationsWebServerSettingsSectionsMcpEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsMemoryEn memory = TranslationsWebServerSettingsSectionsMemoryEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsBackupEn backup = TranslationsWebServerSettingsSectionsBackupEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsClaudeEn claude = TranslationsWebServerSettingsSectionsClaudeEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsCodexEn codex = TranslationsWebServerSettingsSectionsCodexEn.internal(_root);
	late final TranslationsWebServerSettingsSectionsGeminiEn gemini = TranslationsWebServerSettingsSectionsGeminiEn.internal(_root);
}

// Path: web.serverSettings.restart
class TranslationsWebServerSettingsRestartEn {
	TranslationsWebServerSettingsRestartEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Restart server'
	String get button => 'Restart server';

	/// en: 'Self-exec the gateway process'
	String get buttonTitle => 'Self-exec the gateway process';

	/// en: 'You have unsaved changes. Restart will use the LAST SAVED config. Continue?'
	String get dirtyConfirm => 'You have unsaved changes. Restart will use the LAST SAVED config. Continue?';

	/// en: 'Restart the opendray gateway? All open terminal sessions will reconnect automatically.'
	String get confirm => 'Restart the opendray gateway? All open terminal sessions will reconnect automatically.';

	/// en: 'Restarting server…'
	String get overlay => 'Restarting server…';

	/// en: 'Waiting for /health · {tick}s'
	String waiting({required Object tick}) => 'Waiting for /health · ${tick}s';

	/// en: 'Restart timed out'
	String get timedOutTitle => 'Restart timed out';

	/// en: 'Health endpoint never came back. Check server logs.'
	String get timedOutDesc => 'Health endpoint never came back. Check server logs.';

	/// en: 'Server restarted'
	String get successToast => 'Server restarted';
}

// Path: web.serverSettings.formGroups
class TranslationsWebServerSettingsFormGroupsEn {
	TranslationsWebServerSettingsFormGroupsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Network'
	String get network => 'Network';

	/// en: 'Operator account'
	String get operatorAccount => 'Operator account';

	/// en: 'Configuration'
	String get memoryConfiguration => 'Configuration';

	/// en: 'HTTP backend (used when backend=http)'
	String get memoryHttp => 'HTTP backend (used when backend=http)';

	/// en: 'Local ONNX (used when backend=local)'
	String get memoryLocal => 'Local ONNX (used when backend=local)';

	/// en: 'Status'
	String get backupStatus => 'Status';

	/// en: 'Where backups go'
	String get backupWhere => 'Where backups go';

	/// en: 'Schedules'
	String get backupSchedules => 'Schedules';

	/// en: 'What's in a backup?'
	String get backupWhatsInside => 'What\'s in a backup?';
}

// Path: web.serverSettings.fields
class TranslationsWebServerSettingsFieldsEn {
	TranslationsWebServerSettingsFieldsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebServerSettingsFieldsListenAddressEn listenAddress = TranslationsWebServerSettingsFieldsListenAddressEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsUsernameEn username = TranslationsWebServerSettingsFieldsUsernameEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsPasswordEn password = TranslationsWebServerSettingsFieldsPasswordEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsTokenTTLEn tokenTTL = TranslationsWebServerSettingsFieldsTokenTTLEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsLogLevelEn logLevel = TranslationsWebServerSettingsFieldsLogLevelEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsLogFormatEn logFormat = TranslationsWebServerSettingsFieldsLogFormatEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsLogFileEn logFile = TranslationsWebServerSettingsFieldsLogFileEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsIdleThresholdEn idleThreshold = TranslationsWebServerSettingsFieldsIdleThresholdEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsIdlePollIntervalEn idlePollInterval = TranslationsWebServerSettingsFieldsIdlePollIntervalEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsVaultRootEn vaultRoot = TranslationsWebServerSettingsFieldsVaultRootEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsNotesDirectoryEn notesDirectory = TranslationsWebServerSettingsFieldsNotesDirectoryEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsSkillsDirectoryEn skillsDirectory = TranslationsWebServerSettingsFieldsSkillsDirectoryEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsGitRootEn gitRoot = TranslationsWebServerSettingsFieldsGitRootEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsPersonalPrefixEn personalPrefix = TranslationsWebServerSettingsFieldsPersonalPrefixEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsProjectsPrefixEn projectsPrefix = TranslationsWebServerSettingsFieldsProjectsPrefixEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsRegistryRootEn registryRoot = TranslationsWebServerSettingsFieldsRegistryRootEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsSecretsFileEn secretsFile = TranslationsWebServerSettingsFieldsSecretsFileEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryBackendEn memoryBackend = TranslationsWebServerSettingsFieldsMemoryBackendEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryStoreEn memoryStore = TranslationsWebServerSettingsFieldsMemoryStoreEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryTopKEn memoryTopK = TranslationsWebServerSettingsFieldsMemoryTopKEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryThresholdEn memoryThreshold = TranslationsWebServerSettingsFieldsMemoryThresholdEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryScopeEn memoryScope = TranslationsWebServerSettingsFieldsMemoryScopeEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryBaseUrlEn memoryBaseUrl = TranslationsWebServerSettingsFieldsMemoryBaseUrlEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryModelEn memoryModel = TranslationsWebServerSettingsFieldsMemoryModelEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryApiKeyEn memoryApiKey = TranslationsWebServerSettingsFieldsMemoryApiKeyEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryLocalModelEn memoryLocalModel = TranslationsWebServerSettingsFieldsMemoryLocalModelEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryLibraryPathEn memoryLibraryPath = TranslationsWebServerSettingsFieldsMemoryLibraryPathEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryModelPathEn memoryModelPath = TranslationsWebServerSettingsFieldsMemoryModelPathEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryTokenizerPathEn memoryTokenizerPath = TranslationsWebServerSettingsFieldsMemoryTokenizerPathEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEn memoryMaxSeqLen = TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsClaudeHistoryRootsEn claudeHistoryRoots = TranslationsWebServerSettingsFieldsClaudeHistoryRootsEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsClaudeAccountsDirEn claudeAccountsDir = TranslationsWebServerSettingsFieldsClaudeAccountsDirEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsCodexSessionsRootEn codexSessionsRoot = TranslationsWebServerSettingsFieldsCodexSessionsRootEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsGeminiTmpRootEn geminiTmpRoot = TranslationsWebServerSettingsFieldsGeminiTmpRootEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsGeminiProjectsFileEn geminiProjectsFile = TranslationsWebServerSettingsFieldsGeminiProjectsFileEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsBackupLocalDirEn backupLocalDir = TranslationsWebServerSettingsFieldsBackupLocalDirEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsBackupExportDirEn backupExportDir = TranslationsWebServerSettingsFieldsBackupExportDirEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsBackupPgDumpPathEn backupPgDumpPath = TranslationsWebServerSettingsFieldsBackupPgDumpPathEn.internal(_root);
	late final TranslationsWebServerSettingsFieldsBackupPgRestorePathEn backupPgRestorePath = TranslationsWebServerSettingsFieldsBackupPgRestorePathEn.internal(_root);
}

// Path: web.serverSettings.liveTail
class TranslationsWebServerSettingsLiveTailEn {
	TranslationsWebServerSettingsLiveTailEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Live tail'
	String get heading => 'Live tail';

	/// en: 'In-memory ring buffer (last ~2,000 records). Resets on restart.'
	String get description => 'In-memory ring buffer (last ~2,000 records). Resets on restart.';
}

// Path: web.serverSettings.memoryInspectorCard
class TranslationsWebServerSettingsMemoryInspectorCardEn {
	TranslationsWebServerSettingsMemoryInspectorCardEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Inspector'
	String get heading => 'Inspector';

	/// en: 'Browse, search and edit stored memories on the dedicated page.'
	String get description => 'Browse, search and edit stored memories on the dedicated page.';

	/// en: 'Open Memory →'
	String get openButton => 'Open Memory →';
}

// Path: web.serverSettings.stringList
class TranslationsWebServerSettingsStringListEn {
	TranslationsWebServerSettingsStringListEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '(none — using built-in defaults)'
	String get noneDefault => '(none — using built-in defaults)';

	/// en: 'Add path'
	String get addPath => 'Add path';

	/// en: 'Remove'
	String get removeTitle => 'Remove';
}

// Path: web.serverSettings.httpHelpers
class TranslationsWebServerSettingsHttpHelpersEn {
	TranslationsWebServerSettingsHttpHelpersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Auto-detected at startup'
	String get autoDetected => 'Auto-detected at startup';

	/// en: '{count} model(s) — click to use'
	String modelCount({required Object count}) => '${count} model(s) — click to use';

	/// en: 'Presets:'
	String get presets => 'Presets:';

	/// en: 'Test connection'
	String get testConnection => 'Test connection';

	late final TranslationsWebServerSettingsHttpHelpersPresetTipEn presetTip = TranslationsWebServerSettingsHttpHelpersPresetTipEn.internal(_root);
}

// Path: web.serverSettings.probe
class TranslationsWebServerSettingsProbeEn {
	TranslationsWebServerSettingsProbeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '✗ unreachable: {error}'
	String unreachable({required Object error}) => '✗ unreachable: ${error}';

	/// en: 'connection failed'
	String get connectionFailed => 'connection failed';

	/// en: '✓ reachable {detected}· {total} model(s) total · {embedding} embedding'
	String reachable({required Object detected, required Object total, required Object embedding}) => '✓ reachable ${detected}· ${total} model(s) total · ${embedding} embedding';

	/// en: '⚠ Configured model {model} isn't in the list. Pick one of the embedding models below or fix the name.'
	String modelMissing({required Object model}) => '⚠ Configured model ${model} isn\'t in the list. Pick one of the embedding models below or fix the name.';

	/// en: 'embedding models:'
	String get embeddingModelsLabel => 'embedding models:';

	/// en: '+{count} more'
	String moreModels({required Object count}) => '+${count} more';

	/// en: '⚠ No model name contains "embed". The endpoint might not have an embedding model loaded — check your local server.'
	String get noEmbeddingFound => '⚠ No model name contains "embed". The endpoint might not have an embedding model loaded — check your local server.';

	/// en: 'Currently configured'
	String get configuredTitle => 'Currently configured';

	/// en: 'Click to apply'
	String get applyTitle => 'Click to apply';
}

// Path: web.serverSettings.backup
class TranslationsWebServerSettingsBackupEn {
	TranslationsWebServerSettingsBackupEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Feature disabled'
	String get featureDisabledTitle => 'Feature disabled';

	/// en: 'Set <1>OPENDRAY_BACKUP_ENABLED=1</1> + <3>OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;</3> in opendray's environment, then restart. The master passphrase is env-only — it never touches config.toml.'
	String get featureDisabledHint => 'Set <1>OPENDRAY_BACKUP_ENABLED=1</1> + <3>OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;</3> in opendray\'s environment, then restart. The master passphrase is env-only — it never touches config.toml.';

	/// en: 'Status'
	String get statusRowLabel => 'Status';

	/// en: 'enabled · healthy'
	String get enabledHealthy => 'enabled · healthy';

	/// en: 'enabled · degraded'
	String get enabledDegraded => 'enabled · degraded';

	/// en: 'Key fingerprint'
	String get keyFingerprintLabel => 'Key fingerprint';

	/// en: 'record in Vaultwarden — losing it locks all prior backups'
	String get keyFingerprintHint => 'record in Vaultwarden — losing it locks all prior backups';

	/// en: 'pg_dump'
	String get pgDumpLabel => 'pg_dump';

	/// en: 'unavailable'
	String get pgDumpUnavailable => 'unavailable';

	/// en: 'pg_restore'
	String get pgRestoreLabel => 'pg_restore';

	/// en: '(not resolved)'
	String get pgRestoreNotResolved => '(not resolved)';

	/// en: 'Open Backups page →'
	String get openBackups => 'Open Backups page →';

	/// en: 'Open Export / Import →'
	String get openExport => 'Open Export / Import →';

	/// en: 'Each target is one place a backup blob can be written. opendray supports <1>local disk</1>, <3>SMB/CIFS</3> (Windows / NAS), <5>S3-compatible</5> (AWS, R2, B2, MinIO, Alibaba Cloud OSS, Tencent Cloud COS, ...), <7>WebDAV</7> (Nextcloud, Synology, Jianguoyun), <9>SFTP</9>, plus an <11>rclone</11> passthrough that taps into 70+ extra backends (Google Drive, OneDrive, Dropbox, Baidu Pan, Aliyun Drive, ...).'
	String get whereDesc => 'Each target is one place a backup blob can be written. opendray supports <1>local disk</1>, <3>SMB/CIFS</3> (Windows / NAS), <5>S3-compatible</5> (AWS, R2, B2, MinIO, Alibaba Cloud OSS, Tencent Cloud COS, ...), <7>WebDAV</7> (Nextcloud, Synology, Jianguoyun), <9>SFTP</9>, plus an <11>rclone</11> passthrough that taps into 70+ extra backends (Google Drive, OneDrive, Dropbox, Baidu Pan, Aliyun Drive, ...).';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No targets yet. Add one to start backing up.'
	String get noTargets => 'No targets yet. Add one to start backing up.';

	/// en: 'Add target'
	String get addTarget => 'Add target';

	/// en: 'No recurring schedules. Add one on <1>/backups → Schedules</1> to take backups automatically.'
	String get noSchedulesHint => 'No recurring schedules. Add one on <1>/backups → Schedules</1> to take backups automatically.';

	late final TranslationsWebServerSettingsBackupScheduleHeadersEn scheduleHeaders = TranslationsWebServerSettingsBackupScheduleHeadersEn.internal(_root);

	/// en: 'every {interval}'
	String every({required Object interval}) => 'every ${interval}';

	/// en: '{count} backups'
	String backupsKeep({required Object count}) => '${count} backups';

	/// en: 'enabled'
	String get stateEnabled => 'enabled';

	/// en: 'paused'
	String get statePaused => 'paused';

	/// en: 'Manage on /backups → Schedules →'
	String get manageSchedules => 'Manage on /backups → Schedules →';

	/// en: 'Each backup is a <1>pg_dump --format=custom</1> of every opendray table (sessions, integrations, memories, audit_log, etc.) plus a <3>manifest.json</3> and (optionally) the live <5>config.toml</5>. Open the "What's in a backup?" panel on the <7>Backups page</7> to see the live inventory with row counts.'
	String get whatsInsideDesc => 'Each backup is a <1>pg_dump --format=custom</1> of every opendray table (sessions, integrations, memories, audit_log, etc.) plus a <3>manifest.json</3> and (optionally) the live <5>config.toml</5>. Open the "What\'s in a backup?" panel on the <7>Backups page</7> to see the live inventory with row counts.';

	/// en: 'Advanced (paths & client binaries) — restart required'
	String get advancedToggle => 'Advanced (paths & client binaries) — restart required';
}

// Path: web.serverSettings.targetRow
class TranslationsWebServerSettingsTargetRowEn {
	TranslationsWebServerSettingsTargetRowEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'on'
	String get on => 'on';

	/// en: 'off'
	String get off => 'off';

	/// en: 'Test'
	String get test => 'Test';

	/// en: 'Testing…'
	String get testing => 'Testing…';

	/// en: 'Delete'
	String get delete => 'Delete';

	/// en: '{id}: connection OK'
	String connectionOk({required Object id}) => '${id}: connection OK';

	/// en: 'Connection failed'
	String get connectionFailedTitle => 'Connection failed';

	/// en: 'Test failed'
	String get testFailedTitle => 'Test failed';

	/// en: 'Delete target "{id}"? Schedules referencing it will block the delete.'
	String deleteConfirm({required Object id}) => 'Delete target "${id}"? Schedules referencing it will block the delete.';

	/// en: 'Target deleted'
	String get deleteSuccess => 'Target deleted';

	/// en: 'Delete failed'
	String get deleteFailedTitle => 'Delete failed';

	/// en: 'Unknown error'
	String get unknownError => 'Unknown error';
}

// Path: web.settings.groups
class TranslationsWebSettingsGroupsEn {
	TranslationsWebSettingsGroupsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Workspace'
	String get workspace => 'Workspace';

	/// en: 'Server'
	String get server => 'Server';

	/// en: 'System'
	String get system => 'System';
}

// Path: web.settings.items
class TranslationsWebSettingsItemsEn {
	TranslationsWebSettingsItemsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Appearance'
	String get appearance => 'Appearance';

	/// en: 'Font size'
	String get font => 'Font size';

	/// en: 'Account'
	String get account => 'Account';

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'About'
	String get about => 'About';
}

// Path: web.settings.health
class TranslationsWebSettingsHealthEn {
	TranslationsWebSettingsHealthEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'connecting…'
	String get connecting => 'connecting…';

	/// en: 'db ok'
	String get dbOk => 'db ok';

	/// en: 'db down'
	String get dbDown => 'db down';
}

// Path: web.settings.breadcrumb
class TranslationsWebSettingsBreadcrumbEn {
	TranslationsWebSettingsBreadcrumbEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Server'
	String get server => 'Server';
}

// Path: web.settings.appearance
class TranslationsWebSettingsAppearanceEn {
	TranslationsWebSettingsAppearanceEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Appearance'
	String get title => 'Appearance';

	/// en: 'Choose how opendray looks.'
	String get description => 'Choose how opendray looks.';

	late final TranslationsWebSettingsAppearanceOptionsEn options = TranslationsWebSettingsAppearanceOptionsEn.internal(_root);
}

// Path: web.settings.font
class TranslationsWebSettingsFontEn {
	TranslationsWebSettingsFontEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Font size'
	String get title => 'Font size';

	/// en: 'Scales the entire interface. Persisted per browser.'
	String get description => 'Scales the entire interface. Persisted per browser.';

	late final TranslationsWebSettingsFontOptionsEn options = TranslationsWebSettingsFontOptionsEn.internal(_root);
}

// Path: web.settings.account
class TranslationsWebSettingsAccountEn {
	TranslationsWebSettingsAccountEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Account'
	String get title => 'Account';

	/// en: 'Operator and current bearer token.'
	String get description => 'Operator and current bearer token.';

	/// en: 'Username'
	String get username => 'Username';

	/// en: 'Token expires'
	String get tokenExpires => 'Token expires';

	/// en: 'Change credentials'
	String get changeCredentials => 'Change credentials';
}

// Path: web.settings.changeCredentials
class TranslationsWebSettingsChangeCredentialsEn {
	TranslationsWebSettingsChangeCredentialsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Change credentials'
	String get title => 'Change credentials';

	/// en: 'Verify your current password, then pick new credentials. All other signed-in sessions will be revoked.'
	String get description => 'Verify your current password, then pick new credentials. All other signed-in sessions will be revoked.';

	/// en: 'Current password'
	String get currentPassword => 'Current password';

	/// en: 'New username'
	String get newUsername => 'New username';

	/// en: 'New password'
	String get newPassword => 'New password';

	/// en: 'At least 8 characters.'
	String get newPasswordHint => 'At least 8 characters.';

	/// en: 'Confirm new password'
	String get confirm => 'Confirm new password';

	/// en: 'New password must be at least 8 characters.'
	String get errorTooShort => 'New password must be at least 8 characters.';

	/// en: 'New password and confirmation don't match.'
	String get errorMismatch => 'New password and confirmation don\'t match.';

	/// en: 'Current password is wrong.'
	String get errorWrongPassword => 'Current password is wrong.';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Update'
	String get update => 'Update';

	/// en: 'Saving…'
	String get saving => 'Saving…';
}

// Path: web.settings.system
class TranslationsWebSettingsSystemEn {
	TranslationsWebSettingsSystemEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'System status'
	String get title => 'System status';

	/// en: 'Live status from the gateway's /health endpoint.'
	String get description => 'Live status from the gateway\'s /health endpoint.';

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'Version'
	String get version => 'Version';

	/// en: 'Uptime'
	String get uptime => 'Uptime';

	/// en: 'Database'
	String get database => 'Database';

	/// en: 'reachable'
	String get reachable => 'reachable';

	/// en: 'unreachable'
	String get unreachable => 'unreachable';
}

// Path: web.settings.about
class TranslationsWebSettingsAboutEn {
	TranslationsWebSettingsAboutEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'About'
	String get title => 'About';

	/// en: 'opendray v2 — the multiplexer + integration gateway for AI agent CLIs. Source under Apache 2.0.'
	String get description => 'opendray v2 — the multiplexer + integration gateway for AI agent CLIs. Source under Apache 2.0.';
}

// Path: web.memoryAmbient.header
class TranslationsWebMemoryAmbientHeaderEn {
	TranslationsWebMemoryAmbientHeaderEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Ambient memory — auto-capture & inject'
	String get title => 'Ambient memory — auto-capture & inject';

	/// en: 'opendray polls every live agent session every 10 seconds, extracts durable facts via a configurable LLM, and dedups before storing them in the shared memory pool. Configure which LLM does the extraction (Provider), when extraction fires (Capture rule), and what — if anything — gets prepended to the agent's system prompt at spawn (Injection profile).'
	String get body => 'opendray polls every live agent session every 10 seconds, extracts durable facts via a configurable LLM, and dedups before storing them in the shared memory pool. Configure which LLM does the extraction (Provider), when extraction fires (Capture rule), and what — if anything — gets prepended to the agent\'s system prompt at spawn (Injection profile).';
}

// Path: web.memoryAmbient.providers
class TranslationsWebMemoryAmbientProvidersEn {
	TranslationsWebMemoryAmbientProvidersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Summarizer providers'
	String get title => 'Summarizer providers';

	/// en: 'Add provider'
	String get addButton => 'Add provider';

	/// en: 'At least one enabled provider is required for capture to actually fire. Local options (Ollama, LM Studio, Integration) keep your transcripts off external networks.'
	String get intro => 'At least one enabled provider is required for capture to actually fire. Local options (Ollama, LM Studio, Integration) keep your transcripts off external networks.';

	/// en: 'No providers configured yet.'
	String get empty => 'No providers configured yet.';

	late final TranslationsWebMemoryAmbientProvidersRowEn row = TranslationsWebMemoryAmbientProvidersRowEn.internal(_root);
	late final TranslationsWebMemoryAmbientProvidersDialogEn dialog = TranslationsWebMemoryAmbientProvidersDialogEn.internal(_root);
}

// Path: web.memoryAmbient.rules
class TranslationsWebMemoryAmbientRulesEn {
	TranslationsWebMemoryAmbientRulesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Capture rules'
	String get title => 'Capture rules';

	/// en: 'Add rule'
	String get addButton => 'Add rule';

	/// en: 'Each rule says "when this trigger fires, summarize new transcript messages and store the durable facts." Per-session rules override the global default. v1 ships 4 trigger kinds.'
	String get intro => 'Each rule says "when this trigger fires, summarize new transcript messages and store the durable facts." Per-session rules override the global default. v1 ships 4 trigger kinds.';

	/// en: 'No capture rules yet. Add one to enable auto-capture.'
	String get empty => 'No capture rules yet. Add one to enable auto-capture.';

	late final TranslationsWebMemoryAmbientRulesRowEn row = TranslationsWebMemoryAmbientRulesRowEn.internal(_root);
	late final TranslationsWebMemoryAmbientRulesDialogEn dialog = TranslationsWebMemoryAmbientRulesDialogEn.internal(_root);
}

// Path: web.memoryAmbient.profiles
class TranslationsWebMemoryAmbientProfilesEn {
	TranslationsWebMemoryAmbientProfilesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Injection profiles'
	String get title => 'Injection profiles';

	/// en: 'Add profile'
	String get addButton => 'Add profile';

	/// en: 'At spawn time opendray prepends a markdown banner of recent project memories to the agent's system prompt — IF a profile is configured. Without a profile, the model still uses memory_search on demand.'
	String get intro => 'At spawn time opendray prepends a markdown banner of recent project memories to the agent\'s system prompt — IF a profile is configured. Without a profile, the model still uses memory_search on demand.';

	/// en: 'No injection profile. Memories are not auto-injected at spawn — model still uses memory_search.'
	String get empty => 'No injection profile. Memories are not auto-injected at spawn — model still uses memory_search.';

	late final TranslationsWebMemoryAmbientProfilesRowEn row = TranslationsWebMemoryAmbientProfilesRowEn.internal(_root);
	late final TranslationsWebMemoryAmbientProfilesDialogEn dialog = TranslationsWebMemoryAmbientProfilesDialogEn.internal(_root);
}

// Path: web.memoryAmbient.cost
class TranslationsWebMemoryAmbientCostEn {
	TranslationsWebMemoryAmbientCostEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Token cost (all-time)'
	String get title => 'Token cost (all-time)';

	/// en: 'Per-provider summary aggregated from <1>memory_summarizer_calls</1>. Local providers (Ollama, LM Studio, Integration) are priced as $0 — operator owns hardware cost.'
	String get intro => 'Per-provider summary aggregated from <1>memory_summarizer_calls</1>. Local providers (Ollama, LM Studio, Integration) are priced as \$0 — operator owns hardware cost.';

	/// en: 'No enabled providers — no cost data.'
	String get empty => 'No enabled providers — no cost data.';

	late final TranslationsWebMemoryAmbientCostColumnsEn columns = TranslationsWebMemoryAmbientCostColumnsEn.internal(_root);
}

// Path: web.noteEditor.status
class TranslationsWebNoteEditorStatusEn {
	TranslationsWebNoteEditorStatusEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'save failed'
	String get saveFailed => 'save failed';

	/// en: 'saving…'
	String get saving => 'saving…';

	/// en: 'unsaved'
	String get unsaved => 'unsaved';

	/// en: 'new note'
	String get newNote => 'new note';

	/// en: 'saved'
	String get saved => 'saved';
}

// Path: web.export.sections
class TranslationsWebExportSectionsEn {
	TranslationsWebExportSectionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Export'
	String get export => 'Export';

	/// en: 'Import'
	String get import => 'Import';
}

// Path: web.export.form
class TranslationsWebExportFormEn {
	TranslationsWebExportFormEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Scope'
	String get scope => 'Scope';

	/// en: 'Memories'
	String get memories => 'Memories';

	/// en: 'Cross-CLI persistent memory rows (text + scope + metadata). Embedding vectors are omitted; importer re-embeds.'
	String get memoriesHint => 'Cross-CLI persistent memory rows (text + scope + metadata). Embedding vectors are omitted; importer re-embeds.';

	/// en: 'Integrations'
	String get integrations => 'Integrations';

	/// en: 'Custom tasks'
	String get customTasks => 'Custom tasks';

	/// en: 'Operator-defined tasks shown in the Inspector's Tasks tab.'
	String get customTasksHint => 'Operator-defined tasks shown in the Inspector\'s Tasks tab.';

	late final TranslationsWebExportFormIntegrationOptionsEn integrationOptions = TranslationsWebExportFormIntegrationOptionsEn.internal(_root);

	/// en: 'Type <1>I understand</1> to confirm. opendray currently stores only bcrypt hashes — selecting plaintext does NOT export any plaintext (the feature is reserved for a future release that keeps plaintext caches).'
	String get confirmWarning => 'Type <1>I understand</1> to confirm. opendray currently stores only bcrypt hashes — selecting plaintext does NOT export any plaintext (the feature is reserved for a future release that keeps plaintext caches).';

	/// en: 'I understand'
	String get confirmPlaceholder => 'I understand';

	/// en: 'i understand'
	String get confirmSentinel => 'i understand';

	/// en: 'Audit logs and session transcripts are out of scope — covered by /backups (operator dump) instead.'
	String get footnote => 'Audit logs and session transcripts are out of scope — covered by /backups (operator dump) instead.';

	/// en: 'Building…'
	String get building => 'Building…';

	/// en: 'Create export'
	String get create => 'Create export';

	/// en: 'Export ready'
	String get readyToast => 'Export ready';

	/// en: '{bytes} bytes'
	String readyDescription({required Object bytes}) => '${bytes} bytes';

	/// en: 'Export failed'
	String get failedToast => 'Export failed';
}

// Path: web.export.history
class TranslationsWebExportHistoryEn {
	TranslationsWebExportHistoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No exports yet. Use the form above to create one.'
	String get empty => 'No exports yet. Use the form above to create one.';

	/// en: 'History'
	String get title => 'History';

	late final TranslationsWebExportHistoryColumnsEn columns = TranslationsWebExportHistoryColumnsEn.internal(_root);

	/// en: 'Download'
	String get download => 'Download';

	/// en: 'Delete'
	String get deleteTooltip => 'Delete';

	/// en: 'Failed to list exports'
	String get listFailedToast => 'Failed to list exports';

	/// en: 'Download failed'
	String get downloadFailedToast => 'Download failed';

	/// en: 'No download token (expired?)'
	String get noTokenToast => 'No download token (expired?)';

	/// en: 'Delete export {id}?'
	String deleteConfirm({required Object id}) => 'Delete export ${id}?';

	/// en: 'Export deleted'
	String get deletedToast => 'Export deleted';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	/// en: '(empty)'
	String get scopeEmpty => '(empty)';
}

// Path: web.export.import
class TranslationsWebExportImportEn {
	TranslationsWebExportImportEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Replay an export bundle (zip) into the live database. Conflicts (matching id, or unique route_prefix for integrations) are <1>skipped</1> by default. Memories are tagged <3>embedder=imported_v1</3> and need a re-embed pass before search returns them; trigger re-embed under <5>Memory → Maintenance</5>. Integrations are imported with <7>enabled=false</7> and a non-bcrypt placeholder key — operator must rotate before use.'
	String get intro => 'Replay an export bundle (zip) into the live database. Conflicts (matching id, or unique route_prefix for integrations) are <1>skipped</1> by default. Memories are tagged <3>embedder=imported_v1</3> and need a re-embed pass before search returns them; trigger re-embed under <5>Memory → Maintenance</5>. Integrations are imported with <7>enabled=false</7> and a non-bcrypt placeholder key — operator must rotate before use.';

	/// en: 'Memory → Maintenance'
	String get memoryLink => 'Memory → Maintenance';

	/// en: 'Bundle (.zip)'
	String get bundleLabel => 'Bundle (.zip)';

	/// en: 'Memories'
	String get memoriesLabel => 'Memories';

	/// en: 'Integrations (metadata only — keys never imported)'
	String get integrationsLabel => 'Integrations (metadata only — keys never imported)';

	/// en: 'Custom tasks'
	String get customTasksLabel => 'Custom tasks';

	/// en: 'Importing…'
	String get importing => 'Importing…';

	/// en: 'Import bundle'
	String get importBundle => 'Import bundle';

	/// en: 'Pick a bundle file first'
	String get pickFileToast => 'Pick a bundle file first';

	/// en: 'Import done'
	String get doneToast => 'Import done';

	/// en: 'Import finished with errors'
	String get finishedWithErrors => 'Import finished with errors';

	/// en: 'Import failed'
	String get failedToast => 'Import failed';

	late final TranslationsWebExportImportSummaryCardEn summaryCard = TranslationsWebExportImportSummaryCardEn.internal(_root);
}

// Path: web.export.imports
class TranslationsWebExportImportsEn {
	TranslationsWebExportImportsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No imports yet.'
	String get empty => 'No imports yet.';

	/// en: 'History'
	String get title => 'History';

	late final TranslationsWebExportImportsColumnsEn columns = TranslationsWebExportImportsColumnsEn.internal(_root);

	/// en: '(none)'
	String get noneCounts => '(none)';

	/// en: 'Failed to list imports'
	String get listFailedToast => 'Failed to list imports';
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

// Path: more.items.dataExport
class TranslationsMoreItemsDataExportEn {
	TranslationsMoreItemsDataExportEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Data export & import'
	String get title => 'Data export & import';

	/// en: 'User-level data bundles (memories / integrations / custom tasks)'
	String get subtitle => 'User-level data bundles (memories / integrations / custom tasks)';
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

	/// en: 'Enter'
	String get enter => 'Enter';
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

	/// en: 'Load failed: {error}'
	String loadFailedApi({required Object error}) => 'Load failed: ${error}';

	/// en: 'Load failed: {error}'
	String loadFailedGeneric({required Object error}) => 'Load failed: ${error}';

	/// en: 'Save failed: {error}'
	String saveFailedApi({required Object error}) => 'Save failed: ${error}';

	/// en: 'Save failed: {error}'
	String saveFailedGeneric({required Object error}) => 'Save failed: ${error}';

	/// en: 'Insert failed: {error}'
	String insertFailedApi({required Object error}) => 'Insert failed: ${error}';

	/// en: 'Insert failed: {error}'
	String insertFailedGeneric({required Object error}) => 'Insert failed: ${error}';

	/// en: 'Create failed: {error}'
	String createFailedApi({required Object error}) => 'Create failed: ${error}';

	/// en: 'Create failed: {error}'
	String createFailedGeneric({required Object error}) => 'Create failed: ${error}';

	/// en: 'Personal scratchpad — auto-saves as you type. AI agents do not write here.'
	String get personalHint => 'Personal scratchpad — auto-saves as you type. AI agents do not write here.';

	/// en: 'Architecture / spec / decisions / plan / retros — typically authored or maintained by an agent.'
	String get projectDocsHint => 'Architecture / spec / decisions / plan / retros — typically authored or maintained by an agent.';

	/// en: 'Mapping cleared — using default'
	String get mappingCleared => 'Mapping cleared — using default';

	/// en: 'Mapped to {path}'
	String mappedTo({required Object path}) => 'Mapped to ${path}';

	/// en: 'Cancel'
	String get cancelTooltip => 'Cancel';

	/// en: 'New doc'
	String get newDocTooltip => 'New doc';

	/// en: 'Could not resolve a project mapping for this session. Check that the gateway has a notes vault configured and that the session cwd is set.'
	String get noProjectMapping => 'Could not resolve a project mapping for this session. Check that the gateway has a notes vault configured and that the session cwd is set.';

	/// en: 'No project docs yet. Tap + to create one, or let an AI agent generate from a prompt.'
	String get emptyProjectDocs => 'No project docs yet. Tap + to create one, or let an AI agent generate from a prompt.';

	/// en: 'No matches for "{query}".'
	String emptyFilterMatch({required Object query}) => 'No matches for "${query}".';

	/// en: 'Pin this session's cwd to a specific folder under your notes vault. Leave blank to reset.'
	String get locationDialogHelp => 'Pin this session\'s cwd to a specific folder under your notes vault. Leave blank to reset.';

	/// en: 'Session cwd'
	String get sessionCwd => 'Session cwd';

	/// en: 'Vault-relative project docs path'
	String get projectDocsPath => 'Vault-relative project docs path';

	/// en: 'Stored in <vault>/.opendray-projects.json — git-syncs with the rest of the vault.'
	String get locationStoredHint => 'Stored in <vault>/.opendray-projects.json — git-syncs with the rest of the vault.';

	/// en: 'Pinned to {path}/ (overrides {defaultPath}). AI agents author docs here too.'
	String pinnedHint({required Object path, required Object defaultPath}) => 'Pinned to ${path}/ (overrides ${defaultPath}). AI agents author docs here too.';

	/// en: '(no project mapping)'
	String get noProjectMapping2 => '(no project mapping)';

	/// en: 'Clear override'
	String get clearOverride => 'Clear override';

	/// en: 'Save'
	String get save => 'Save';
}

// Path: sessions.spawnSheet.bypass
class TranslationsSessionsSpawnSheetBypassEn {
	TranslationsSessionsSpawnSheetBypassEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Bypass permissions'
	String get labelClaude => 'Bypass permissions';

	/// en: 'Bypass approvals & sandbox'
	String get labelCodex => 'Bypass approvals & sandbox';

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

// Path: memoryWorkers.tasks.planDrift
class TranslationsMemoryWorkersTasksPlanDriftEn {
	TranslationsMemoryWorkersTasksPlanDriftEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Plan drift detector'
	String get label => 'Plan drift detector';

	/// en: 'After each session ends, checks whether the project plan needs updating and files a proposal. Fits an agent worker for richer reasoning.'
	String get description => 'After each session ends, checks whether the project plan needs updating and files a proposal. Fits an agent worker for richer reasoning.';
}

// Path: memoryWorkers.tasks.conflictDetector
class TranslationsMemoryWorkersTasksConflictDetectorEn {
	TranslationsMemoryWorkersTasksConflictDetectorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Cross-layer conflict detector'
	String get label => 'Cross-layer conflict detector';

	/// en: 'Daily scan that finds contradictions between facts / plan / goal / journal. Higher-quality model = fewer false positives.'
	String get description => 'Daily scan that finds contradictions between facts / plan / goal / journal. Higher-quality model = fewer false positives.';
}

// Path: memoryWorkers.tasks.capture
class TranslationsMemoryWorkersTasksCaptureEn {
	TranslationsMemoryWorkersTasksCaptureEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Capture engine'
	String get label => 'Capture engine';

	/// en: 'Per-trigger fact extraction from session transcripts. Agent mode gives noticeably better facts on long sessions; summarizer mode is cheap and local.'
	String get description => 'Per-trigger fact extraction from session transcripts. Agent mode gives noticeably better facts on long sessions; summarizer mode is cheap and local.';
}

// Path: project.conflicts.severity
class TranslationsProjectConflictsSeverityEn {
	TranslationsProjectConflictsSeverityEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'low'
	String get low => 'low';

	/// en: 'medium'
	String get medium => 'medium';

	/// en: 'high'
	String get high => 'high';
}

// Path: backupTargetEditor.kinds.local
class TranslationsBackupTargetEditorKindsLocalEn {
	TranslationsBackupTargetEditorKindsLocalEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Local disk'
	String get label => 'Local disk';

	/// en: 'Folder on the machine running opendray'
	String get description => 'Folder on the machine running opendray';
}

// Path: backupTargetEditor.kinds.smb
class TranslationsBackupTargetEditorKindsSmbEn {
	TranslationsBackupTargetEditorKindsSmbEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'SMB share'
	String get label => 'SMB share';

	/// en: 'Windows shares + most home NAS appliances'
	String get description => 'Windows shares + most home NAS appliances';
}

// Path: backupTargetEditor.kinds.webdav
class TranslationsBackupTargetEditorKindsWebdavEn {
	TranslationsBackupTargetEditorKindsWebdavEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'WebDAV'
	String get label => 'WebDAV';

	/// en: 'Self-hosted clouds + file-sharing services'
	String get description => 'Self-hosted clouds + file-sharing services';
}

// Path: backupTargetEditor.kinds.sftp
class TranslationsBackupTargetEditorKindsSftpEn {
	TranslationsBackupTargetEditorKindsSftpEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'SFTP'
	String get label => 'SFTP';

	/// en: 'Any SSH-accessible server'
	String get description => 'Any SSH-accessible server';
}

// Path: backupTargetEditor.kinds.s3
class TranslationsBackupTargetEditorKindsS3En {
	TranslationsBackupTargetEditorKindsS3En.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'S3 / compatible'
	String get label => 'S3 / compatible';

	/// en: 'Amazon S3 + S3-compatible buckets (MinIO, R2, B2)'
	String get description => 'Amazon S3 + S3-compatible buckets (MinIO, R2, B2)';
}

// Path: backupTargetEditor.kinds.rclone
class TranslationsBackupTargetEditorKindsRcloneEn {
	TranslationsBackupTargetEditorKindsRcloneEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'rclone (any)'
	String get label => 'rclone (any)';

	/// en: 'OneDrive, Google Drive, Dropbox via the rclone CLI'
	String get description => 'OneDrive, Google Drive, Dropbox via the rclone CLI';
}

// Path: githosts.form.kinds
class TranslationsGithostsFormKindsEn {
	TranslationsGithostsFormKindsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'GitHub'
	String get github => 'GitHub';

	/// en: 'GitLab'
	String get gitlab => 'GitLab';

	/// en: 'Bitbucket'
	String get bitbucket => 'Bitbucket';

	/// en: 'Gitea'
	String get gitea => 'Gitea';

	/// en: 'Custom'
	String get custom => 'Custom';
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

// Path: dataExport.form.integrationOptions
class TranslationsDataExportFormIntegrationOptionsEn {
	TranslationsDataExportFormIntegrationOptionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Skip'
	String get none => 'Skip';

	/// en: 'Don't include the /integrations registry.'
	String get noneHint => 'Don\'t include the /integrations registry.';

	/// en: 'Metadata only (default)'
	String get metadata => 'Metadata only (default)';

	/// en: 'Per-integration name + endpoint, no API keys.'
	String get metadataHint => 'Per-integration name + endpoint, no API keys.';

	/// en: 'Plaintext keys'
	String get plaintext => 'Plaintext keys';

	/// en: 'DANGEROUS: includes raw API tokens. v1 stores only bcrypt hashes, so this is effectively a no-op today; surface anyway.'
	String get plaintextHint => 'DANGEROUS: includes raw API tokens. v1 stores only bcrypt hashes, so this is effectively a no-op today; surface anyway.';
}

// Path: dataExport.history.columns
class TranslationsDataExportHistoryColumnsEn {
	TranslationsDataExportHistoryColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Scope'
	String get scope => 'Scope';

	/// en: 'Size'
	String get size => 'Size';

	/// en: 'Expires'
	String get expires => 'Expires';

	/// en: 'Actions'
	String get actions => 'Actions';
}

// Path: dataExport.import.summaryCard
class TranslationsDataExportImportSummaryCardEn {
	TranslationsDataExportImportSummaryCardEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memories'
	String get memories => 'Memories';

	/// en: 'Integrations'
	String get integrations => 'Integrations';

	/// en: 'Custom tasks'
	String get customTasks => 'Custom tasks';

	/// en: 'created'
	String get created => 'created';

	/// en: 'skipped'
	String get skipped => 'skipped';

	/// en: 'failed'
	String get failed => 'failed';
}

// Path: dataExport.imports.columns
class TranslationsDataExportImportsColumnsEn {
	TranslationsDataExportImportsColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'ID'
	String get id => 'ID';

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'Source'
	String get source => 'Source';

	/// en: 'Counts'
	String get counts => 'Counts';

	/// en: 'When'
	String get when => 'When';
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

// Path: settings.serverSettings.sectionDescriptions
class TranslationsSettingsServerSettingsSectionDescriptionsEn {
	TranslationsSettingsServerSettingsSectionDescriptionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Listen address, operator account, token TTL.'
	String get general => 'Listen address, operator account, token TTL.';

	/// en: 'Verbosity, format, and on-disk log path.'
	String get logging => 'Verbosity, format, and on-disk log path.';

	/// en: 'Idle detection thresholds.'
	String get sessions => 'Idle detection thresholds.';

	/// en: 'Notes, skills, and git-versioned root.'
	String get vault => 'Notes, skills, and git-versioned root.';

	/// en: 'Vault paths for MCP servers + secrets file.'
	String get mcpRegistry => 'Vault paths for MCP servers + secrets file.';

	/// en: 'Cross-CLI persistent memory subsystem.'
	String get memory => 'Cross-CLI persistent memory subsystem.';

	/// en: 'Encrypted DB backups + admin data exports. Passphrase lives in the keyfile (Settings → Backups).'
	String get backup => 'Encrypted DB backups + admin data exports. Passphrase lives in the keyfile (Settings → Backups).';

	/// en: 'Where Claude transcripts live on disk.'
	String get storageClaude => 'Where Claude transcripts live on disk.';

	/// en: 'Codex sessions root.'
	String get storageCodex => 'Codex sessions root.';

	/// en: 'Per-project tmp + projects.json paths.'
	String get storageGemini => 'Per-project tmp + projects.json paths.';
}

// Path: settings.serverSettings.fields
class TranslationsSettingsServerSettingsFieldsEn {
	TranslationsSettingsServerSettingsFieldsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Listen address'
	String get listenAddress => 'Listen address';

	/// en: 'Admin user'
	String get adminUser => 'Admin user';

	/// en: 'Effective when no keyfile or env var is set. Otherwise see Settings → Account.'
	String get adminUserHelper => 'Effective when no keyfile or env var is set. Otherwise see Settings → Account.';

	/// en: 'Admin password'
	String get adminPassword => 'Admin password';

	/// en: 'Send blank to preserve. For ongoing rotations use Settings → Account (keyfile-backed, no restart).'
	String get adminPasswordHelper => 'Send blank to preserve. For ongoing rotations use Settings → Account (keyfile-backed, no restart).';

	/// en: 'Token TTL (web)'
	String get tokenTtlWeb => 'Token TTL (web)';

	/// en: 'Go duration string, e.g. 24h, 30m.'
	String get tokenTtlHelper => 'Go duration string, e.g. 24h, 30m.';

	/// en: 'Level'
	String get level => 'Level';

	/// en: 'Format'
	String get format => 'Format';

	/// en: 'File path'
	String get filePath => 'File path';

	/// en: 'Empty = stdout only.'
	String get filePathHelper => 'Empty = stdout only.';

	/// en: 'Idle threshold'
	String get idleThreshold => 'Idle threshold';

	/// en: 'Quiet period before a session is flagged idle. Go duration.'
	String get idleThresholdHelper => 'Quiet period before a session is flagged idle. Go duration.';

	/// en: 'Idle check interval'
	String get idleCheckInterval => 'Idle check interval';

	/// en: 'How often the idle reaper runs.'
	String get idleCheckHelper => 'How often the idle reaper runs.';

	/// en: 'Root'
	String get root => 'Root';

	/// en: 'Parent of notes / skills / git_root sub-paths.'
	String get rootHelper => 'Parent of notes / skills / git_root sub-paths.';

	/// en: 'Notes path'
	String get notesPath => 'Notes path';

	/// en: 'Skills path'
	String get skillsPath => 'Skills path';

	/// en: 'Git root'
	String get gitRoot => 'Git root';

	/// en: 'Personal prefix'
	String get personalPrefix => 'Personal prefix';

	/// en: 'Projects prefix'
	String get projectsPrefix => 'Projects prefix';

	/// en: 'Registry root'
	String get registryRoot => 'Registry root';

	/// en: 'Secrets file'
	String get secretsFile => 'Secrets file';

	/// en: 'Backend'
	String get backend => 'Backend';

	/// en: 'Store'
	String get store => 'Store';

	/// en: 'Default top-k'
	String get defaultTopK => 'Default top-k';

	/// en: 'Similarity threshold'
	String get similarityThreshold => 'Similarity threshold';

	/// en: 'Default scope'
	String get defaultScope => 'Default scope';

	/// en: 'When store=chromem.'
	String get chromemHelper => 'When store=chromem.';

	/// en: 'Blank to preserve current.'
	String get preserveHelper => 'Blank to preserve current.';

	/// en: 'Local model name'
	String get localModelName => 'Local model name';

	/// en: 'Local library path'
	String get localLibraryPath => 'Local library path';

	/// en: 'Local model path'
	String get localModelPath => 'Local model path';

	/// en: 'Local tokenizer path'
	String get localTokenizerPath => 'Local tokenizer path';

	/// en: 'Local max seq len'
	String get localMaxSeqLen => 'Local max seq len';

	/// en: 'Enabled'
	String get backupEnabled => 'Enabled';

	/// en: 'Even with this on, the backup subsystem stays off until OPENDRAY_BACKUP_KEY or the keyfile is configured.'
	String get backupEnabledHelper => 'Even with this on, the backup subsystem stays off until OPENDRAY_BACKUP_KEY or the keyfile is configured.';

	/// en: 'Local dir'
	String get backupLocalDir => 'Local dir';

	/// en: 'Export dir'
	String get backupExportDir => 'Export dir';

	/// en: 'Empty = resolve from PATH at startup.'
	String get pathHelper => 'Empty = resolve from PATH at startup.';

	/// en: 'Accounts dir'
	String get accountsDir => 'Accounts dir';

	/// en: 'Parent of per-account .claude/ subdirs. Empty = ~/.claude-accounts.'
	String get accountsHelper => 'Parent of per-account .claude/ subdirs. Empty = ~/.claude-accounts.';

	/// en: 'Sessions root'
	String get sessionsRoot => 'Sessions root';

	/// en: 'Empty = ~/.codex/sessions.'
	String get sessionsRootHelper => 'Empty = ~/.codex/sessions.';

	/// en: 'host:port the gateway binds to. Restart required.'
	String get listenHelper => 'host:port the gateway binds to. Restart required.';

	/// en: 'AES-256-GCM encrypted secrets vault.'
	String get secretsHelper => 'AES-256-GCM encrypted secrets vault.';

	/// en: 'auto picks the best available; local needs ONNX.'
	String get backendHelper => 'auto picks the best available; local needs ONNX.';

	/// en: '0.0–1.0; results under this are filtered out.'
	String get similarityHelper => '0.0–1.0; results under this are filtered out.';

	/// en: 'Default: {value}'
	String defaultFallback({required Object value}) => 'Default: ${value}';

	/// en: 'chromem path'
	String get chromemPath => 'chromem path';

	/// en: 'HTTP base URL'
	String get httpBaseUrl => 'HTTP base URL';

	/// en: 'HTTP model'
	String get httpModel => 'HTTP model';

	/// en: 'HTTP api key'
	String get httpApiKey => 'HTTP api key';

	/// en: 'HTTP dimensions'
	String get httpDimensions => 'HTTP dimensions';

	/// en: 'pg_dump path'
	String get pgDumpPath => 'pg_dump path';

	/// en: 'pg_restore path'
	String get pgRestorePath => 'pg_restore path';

	/// en: 'tmp root'
	String get tmpRoot => 'tmp root';

	/// en: 'projects.json'
	String get projectsJson => 'projects.json';
}

// Path: web.sessions.list.row
class TranslationsWebSessionsListRowEn {
	TranslationsWebSessionsListRowEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Delete session'
	String get deleteAria => 'Delete session';

	/// en: 'Remove from history'
	String get titleRemoveHistory => 'Remove from history';

	/// en: 'Terminate and remove'
	String get titleTerminate => 'Terminate and remove';

	/// en: 'Remove'
	String get titleRemove => 'Remove';

	/// en: 'Claude account: {label}'
	String claudeAccountTitle({required Object label}) => 'Claude account: ${label}';
}

// Path: web.sessions.inspector.tabs
class TranslationsWebSessionsInspectorTabsEn {
	TranslationsWebSessionsInspectorTabsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Files'
	String get files => 'Files';

	/// en: 'Git'
	String get git => 'Git';

	/// en: 'Search'
	String get search => 'Search';

	/// en: 'Tasks'
	String get tasks => 'Tasks';

	/// en: 'History'
	String get history => 'History';

	/// en: 'Notes'
	String get notes => 'Notes';

	/// en: 'Memory'
	String get memory => 'Memory';
}

// Path: web.memoryWorkers.tasks.gatekeeper
class TranslationsWebMemoryWorkersTasksGatekeeperEn {
	TranslationsWebMemoryWorkersTasksGatekeeperEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Gatekeeper'
	String get label => 'Gatekeeper';

	/// en: 'Pre-write filter on every memory_store. High frequency (<500ms target) — summarizer-only.'
	String get description => 'Pre-write filter on every memory_store. High frequency (<500ms target) — summarizer-only.';
}

// Path: web.memoryWorkers.tasks.cleaner
class TranslationsWebMemoryWorkersTasksCleanerEn {
	TranslationsWebMemoryWorkersTasksCleanerEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Cleaner librarian'
	String get label => 'Cleaner librarian';

	/// en: 'Periodic LLM librarian. Judges aged memories as keep / stale / duplicate.'
	String get description => 'Periodic LLM librarian. Judges aged memories as keep / stale / duplicate.';
}

// Path: web.memoryWorkers.tasks.gitactivity
class TranslationsWebMemoryWorkersTasksGitactivityEn {
	TranslationsWebMemoryWorkersTasksGitactivityEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Git activity summariser'
	String get label => 'Git activity summariser';

	/// en: 'git log → 2-3 paragraph narrative every 24h. Naturally fits an agent worker.'
	String get description => 'git log → 2-3 paragraph narrative every 24h. Naturally fits an agent worker.';
}

// Path: web.memoryWorkers.tasks.transcript
class TranslationsWebMemoryWorkersTasksTranscriptEn {
	TranslationsWebMemoryWorkersTasksTranscriptEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Session transcript summariser'
	String get label => 'Session transcript summariser';

	/// en: 'Session-end "what did the agent do" summary. Naturally fits an agent worker.'
	String get description => 'Session-end "what did the agent do" summary. Naturally fits an agent worker.';
}

// Path: web.memoryWorkers.tasks.plan_drift
class TranslationsWebMemoryWorkersTasksPlanDriftEn {
	TranslationsWebMemoryWorkersTasksPlanDriftEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Plan drift detector'
	String get label => 'Plan drift detector';

	/// en: 'After each session ends, checks whether the project plan needs updating and files a proposal. Fits an agent worker for richer reasoning.'
	String get description => 'After each session ends, checks whether the project plan needs updating and files a proposal. Fits an agent worker for richer reasoning.';
}

// Path: web.memoryWorkers.tasks.conflict_detector
class TranslationsWebMemoryWorkersTasksConflictDetectorEn {
	TranslationsWebMemoryWorkersTasksConflictDetectorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Cross-layer conflict detector'
	String get label => 'Cross-layer conflict detector';

	/// en: 'Daily scan that finds contradictions between facts / plan / goal / journal. Higher-quality model = fewer false positives.'
	String get description => 'Daily scan that finds contradictions between facts / plan / goal / journal. Higher-quality model = fewer false positives.';
}

// Path: web.memoryWorkers.tasks.capture
class TranslationsWebMemoryWorkersTasksCaptureEn {
	TranslationsWebMemoryWorkersTasksCaptureEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Capture engine'
	String get label => 'Capture engine';

	/// en: 'Per-trigger fact extraction from session transcripts. Agent mode gives noticeably better facts on long sessions; summarizer mode is cheap and local.'
	String get description => 'Per-trigger fact extraction from session transcripts. Agent mode gives noticeably better facts on long sessions; summarizer mode is cheap and local.';
}

// Path: web.project.readonly.tech_stack
class TranslationsWebProjectReadonlyTechStackEn {
	TranslationsWebProjectReadonlyTechStackEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Tech stack & structure'
	String get label => 'Tech stack & structure';

	/// en: 'Run a Claude session in this project — scanner refreshes on every spawn.'
	String get empty => 'Run a Claude session in this project — scanner refreshes on every spawn.';
}

// Path: web.project.readonly.recent_activity
class TranslationsWebProjectReadonlyRecentActivityEn {
	TranslationsWebProjectReadonlyRecentActivityEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Recent activity (git → LLM)'
	String get label => 'Recent activity (git → LLM)';

	/// en: 'The git activity summariser runs every 24h; check back after the next scheduler tick.'
	String get empty => 'The git activity summariser runs every 24h; check back after the next scheduler tick.';
}

// Path: web.project.reset.summary
class TranslationsWebProjectResetSummaryEn {
	TranslationsWebProjectResetSummaryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '{count} doc'
	String docs_one({required Object count}) => '${count} doc';

	/// en: '{count} docs'
	String docs_other({required Object count}) => '${count} docs';

	/// en: '{count} journal'
	String journal({required Object count}) => '${count} journal';

	/// en: '{count} proposal'
	String proposals_one({required Object count}) => '${count} proposal';

	/// en: '{count} proposals'
	String proposals_other({required Object count}) => '${count} proposals';

	/// en: '{count} cleanup'
	String cleanup({required Object count}) => '${count} cleanup';

	/// en: '{count} memories'
	String memories({required Object count}) => '${count} memories';
}

// Path: web.memoryInspector.scope.values
class TranslationsWebMemoryInspectorScopeValuesEn {
	TranslationsWebMemoryInspectorScopeValuesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'project'
	String get project => 'project';

	/// en: 'session'
	String get session => 'session';

	/// en: 'global'
	String get global => 'global';
}

// Path: web.notes.vaultSync.init
class TranslationsWebNotesVaultSyncInitEn {
	TranslationsWebNotesVaultSyncInitEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Vault is not a git repo yet'
	String get title => 'Vault is not a git repo yet';

	/// en: 'Initialising will run <1>git init -b main</1> in your vault root and add a sane <3>.gitignore</3>. After that you can commit your notes and configure a remote (GitHub / Gitea / GitLab) for cross-machine sync.'
	String get body => 'Initialising will run <1>git init -b main</1> in your vault root and add a sane <3>.gitignore</3>. After that you can commit your notes and configure a remote (GitHub / Gitea / GitLab) for cross-machine sync.';

	/// en: 'Initialise vault as git repo'
	String get button => 'Initialise vault as git repo';

	/// en: 'Vault initialised as git repo'
	String get initToast => 'Vault initialised as git repo';

	/// en: 'Init failed'
	String get initFailedToast => 'Init failed';
}

// Path: web.notes.vaultSync.branch
class TranslationsWebNotesVaultSyncBranchEn {
	TranslationsWebNotesVaultSyncBranchEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'clean'
	String get clean => 'clean';

	/// en: '{count} staged'
	String staged({required Object count}) => '${count} staged';

	/// en: '{count} modified'
	String modified({required Object count}) => '${count} modified';

	/// en: '{count} untracked'
	String untracked({required Object count}) => '${count} untracked';
}

// Path: web.notes.vaultSync.action
class TranslationsWebNotesVaultSyncActionEn {
	TranslationsWebNotesVaultSyncActionEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Pull'
	String get pull => 'Pull';

	/// en: 'Push'
	String get push => 'Push';

	/// en: 'Configure a remote first'
	String get pullTitleNoRemote => 'Configure a remote first';

	/// en: 'git pull --rebase --autostash'
	String get pullTitleHasUpstream => 'git pull --rebase --autostash';

	/// en: 'Pulls origin's HEAD; sets up tracking implicitly'
	String get pullTitleNoUpstream => 'Pulls origin\'s HEAD; sets up tracking implicitly';

	/// en: 'Configure a remote first'
	String get pushTitleNoRemote => 'Configure a remote first';

	/// en: 'git push -u origin HEAD'
	String get pushTitleHasUpstream => 'git push -u origin HEAD';

	/// en: 'First push — will set upstream to origin/HEAD'
	String get pushTitleNoUpstream => 'First push — will set upstream to origin/HEAD';

	/// en: 'No remote configured — pull/push disabled'
	String get noRemote => 'No remote configured — pull/push disabled';

	/// en: 'No upstream tracking yet — first push will set it.'
	String get noUpstream => 'No upstream tracking yet — first push will set it.';

	/// en: 'Pulled'
	String get pulledToast => 'Pulled';

	/// en: 'Pull failed'
	String get pullFailedToast => 'Pull failed';

	/// en: 'Pushed'
	String get pushedToast => 'Pushed';

	/// en: 'Push failed'
	String get pushFailedToast => 'Push failed';
}

// Path: web.notes.vaultSync.commit
class TranslationsWebNotesVaultSyncCommitEn {
	TranslationsWebNotesVaultSyncCommitEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Commit'
	String get title => 'Commit';

	/// en: 'Notes: {date} (default)'
	String placeholder({required Object date}) => 'Notes: ${date} (default)';

	/// en: 'Commit all'
	String get commitAll => 'Commit all';

	/// en: 'Stages every change (<1>git add .</1>) then commits with this message. Empty message defaults to a timestamped subject.'
	String get hint => 'Stages every change (<1>git add .</1>) then commits with this message. Empty message defaults to a timestamped subject.';

	/// en: 'Committed {hash}'
	String committedToast({required Object hash}) => 'Committed ${hash}';

	/// en: 'Commit failed'
	String get commitFailedToast => 'Commit failed';
}

// Path: web.notes.vaultSync.fileList
class TranslationsWebNotesVaultSyncFileListEn {
	TranslationsWebNotesVaultSyncFileListEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Working tree · {count}'
	String title({required Object count}) => 'Working tree · ${count}';

	/// en: '+{count} more'
	String moreSuffix({required Object count}) => '+${count} more';
}

// Path: web.notes.vaultSync.remote
class TranslationsWebNotesVaultSyncRemoteEn {
	TranslationsWebNotesVaultSyncRemoteEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Remote (origin)'
	String get title => 'Remote (origin)';

	/// en: 'Cancel'
	String get cancel => 'Cancel';

	/// en: 'Change'
	String get change => 'Change';

	/// en: 'Configure'
	String get configure => 'Configure';

	/// en: 'No remote set. Add an HTTPS or SSH URL (e.g. <1>git@github.com:you/notes.git</1> or <3>https://gitea.example.com/you/notes.git</3>) to enable push / pull.'
	String get empty => 'No remote set. Add an HTTPS or SSH URL (e.g. <1>git@github.com:you/notes.git</1> or <3>https://gitea.example.com/you/notes.git</3>) to enable push / pull.';

	/// en: 'URL (HTTPS or SSH)'
	String get urlLabel => 'URL (HTTPS or SSH)';

	/// en: 'git@host:owner/notes.git'
	String get urlPlaceholder => 'git@host:owner/notes.git';

	/// en: 'Save'
	String get save => 'Save';

	/// en: 'Remote saved'
	String get savedToast => 'Remote saved';

	/// en: 'Set remote failed'
	String get saveFailedToast => 'Set remote failed';
}

// Path: web.notes.vaultSync.history
class TranslationsWebNotesVaultSyncHistoryEn {
	TranslationsWebNotesVaultSyncHistoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Recent commits'
	String get title => 'Recent commits';

	/// en: 'Loading…'
	String get loading => 'Loading…';

	/// en: 'No commits yet.'
	String get empty => 'No commits yet.';
}

// Path: web.notes.vaultSync.conflict
class TranslationsWebNotesVaultSyncConflictEn {
	TranslationsWebNotesVaultSyncConflictEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations
	late final TranslationsWebNotesVaultSyncConflictKindsEn kinds = TranslationsWebNotesVaultSyncConflictKindsEn.internal(_root);

	/// en: 'Vault has a paused {kind} with unresolved conflicts'
	String headline({required Object kind}) => 'Vault has a paused ${kind} with unresolved conflicts';

	/// en: 'Pull, push and commit are blocked until the {kind} finishes. You can either <1>abort</1> (restore the working tree to its state before the {kind} — keeps your local commits, drops the remote ones) or <3>force reset to remote</3> (discard ALL local commits + uncommitted changes; vault becomes an exact mirror of origin).'
	String explainer({required Object kind}) => 'Pull, push and commit are blocked until the ${kind} finishes. You can either <1>abort</1> (restore the working tree to its state before the ${kind} — keeps your local commits, drops the remote ones) or <3>force reset to remote</3> (discard ALL local commits + uncommitted changes; vault becomes an exact mirror of origin).';

	/// en: 'Conflicted files · {count}'
	String conflictedHeader({required Object count}) => 'Conflicted files · ${count}';

	/// en: 'Abort {kind}'
	String abort({required Object kind}) => 'Abort ${kind}';

	/// en: 'git {kind} --abort'
	String abortTitle({required Object kind}) => 'git ${kind} --abort';

	/// en: 'Force reset to remote'
	String get forceReset => 'Force reset to remote';

	/// en: 'git fetch && git reset --hard origin/<branch> && git clean -fd'
	String get forceResetTitle => 'git fetch && git reset --hard origin/<branch> && git clean -fd';

	/// en: 'DESTRUCTIVE: this will • abort the in-progress {kind} • run git fetch origin • reset --hard to origin/<branch> • clean -fd (drop untracked files) Any local commits not pushed to origin AND any uncommitted edits will be PERMANENTLY LOST. Continue?'
	String forceResetConfirm({required Object kind}) => 'DESTRUCTIVE: this will\n  • abort the in-progress ${kind}\n  • run git fetch origin\n  • reset --hard to origin/<branch>\n  • clean -fd (drop untracked files)\n\nAny local commits not pushed to origin AND any uncommitted edits will be PERMANENTLY LOST.\n\nContinue?';

	/// en: 'Aborted {kind}'
	String abortedToast({required Object kind}) => 'Aborted ${kind}';

	/// en: 'Working tree restored to pre-operation state.'
	String get abortedDescription => 'Working tree restored to pre-operation state.';

	/// en: 'Abort failed'
	String get abortFailedToast => 'Abort failed';

	/// en: 'Reset to {branch}'
	String resetToast({required Object branch}) => 'Reset to ${branch}';

	/// en: 'Local changes discarded; vault matches remote.'
	String get resetDescription => 'Local changes discarded; vault matches remote.';

	/// en: 'Reset failed'
	String get resetFailedToast => 'Reset failed';
}

// Path: web.notes.vaultSync.auth
class TranslationsWebNotesVaultSyncAuthEn {
	TranslationsWebNotesVaultSyncAuthEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Authentication'
	String get title => 'Authentication';

	/// en: 'Will use the token stored for <1>{host}</1> in Plugins → Git hosts. ✓'
	String httpsTokenOk({required Object host}) => 'Will use the token stored for <1>${host}</1> in Plugins → Git hosts. ✓';

	/// en: 'HTTPS remote on <1>{host}</1> with no opendray token configured. Push / pull will likely fail for private repos until you add one.'
	String httpsTokenMissing({required Object host}) => 'HTTPS remote on <1>${host}</1> with no opendray token configured. Push / pull will likely fail for private repos until you add one.';

	/// en: 'SSH remote on <1>{host}</1>. Auth uses the gateway host's <3>~/.ssh/</3> (ssh-agent, identity file, host config). Verify with <5>ssh -T git@{host}</5> from the host shell.'
	String ssh({required Object host}) => 'SSH remote on <1>${host}</1>. Auth uses the gateway host\'s <3>~/.ssh/</3> (ssh-agent, identity file, host config). Verify with <5>ssh -T git@${host}</5> from the host shell.';

	/// en: '→ Configure git host token'
	String get configureTokenLink => '→ Configure git host token';
}

// Path: web.notes.vaultSync.autoSync
class TranslationsWebNotesVaultSyncAutoSyncEn {
	TranslationsWebNotesVaultSyncAutoSyncEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Loading auto-sync settings…'
	String get loading => 'Loading auto-sync settings…';

	/// en: 'Auto-sync'
	String get title => 'Auto-sync';

	/// en: 'on'
	String get on => 'on';

	/// en: 'Run now'
	String get runNow => 'Run now';

	/// en: 'Wake the sync loop now (skips the wait, then runs whichever steps are due)'
	String get runNowTooltip => 'Wake the sync loop now (skips the wait, then runs whichever steps are due)';

	/// en: 'Configure'
	String get configure => 'Configure';

	/// en: 'Hide'
	String get hide => 'Hide';

	/// en: 'Enabled'
	String get enabled => 'Enabled';

	/// en: 'Configure a remote first to enable auto-sync'
	String get enabledTooltipNoRemote => 'Configure a remote first to enable auto-sync';

	/// en: 'No remote — push/pull will be skipped.'
	String get noRemoteHint => 'No remote — push/pull will be skipped.';

	/// en: 'Commit every'
	String get commitEvery => 'Commit every';

	/// en: 'Examples: <1>30s</1>, <3>10m</3>, <5>2h</5>. Min 30s.'
	String get commitEveryExamples => 'Examples: <1>30s</1>, <3>10m</3>, <5>2h</5>. Min 30s.';

	/// en: 'Pull every'
	String get pullEvery => 'Pull every';

	/// en: 'Only used when Pull is enabled.'
	String get pullEveryHint => 'Only used when Pull is enabled.';

	/// en: 'Push after commit'
	String get pushAfterCommit => 'Push after commit';

	/// en: 'Pull periodically'
	String get pullPeriodically => 'Pull periodically';

	/// en: 'Commit message template'
	String get commitTemplateLabel => 'Commit message template';

	/// en: 'Auto-sync: {date} (default if empty)'
	String commitTemplatePlaceholder({required Object date}) => 'Auto-sync: ${date}  (default if empty)';

	/// en: 'Save settings'
	String get saveSettings => 'Save settings';

	/// en: 'Discard'
	String get discard => 'Discard';

	/// en: 'last commit'
	String get lastCommit => 'last commit';

	/// en: 'last push'
	String get lastPush => 'last push';

	/// en: 'last pull'
	String get lastPull => 'last pull';

	/// en: 'never'
	String get never => 'never';

	/// en: 'Auto-sync settings saved'
	String get savedToast => 'Auto-sync settings saved';

	/// en: 'Save failed'
	String get saveFailedToast => 'Save failed';

	/// en: 'Auto-sync triggered'
	String get triggeredToast => 'Auto-sync triggered';

	/// en: 'Run failed'
	String get runFailedToast => 'Run failed';
}

// Path: web.providers.detail.caps
class TranslationsWebProvidersDetailCapsEn {
	TranslationsWebProvidersDetailCapsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'resume'
	String get resume => 'resume';

	/// en: 'stream'
	String get stream => 'stream';

	/// en: 'images'
	String get images => 'images';

	/// en: 'mcp'
	String get mcp => 'mcp';
}

// Path: web.channels.notifications.modes
class TranslationsWebChannelsNotificationsModesEn {
	TranslationsWebChannelsNotificationsModesEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Once per session (recommended)'
	String get onceLabel => 'Once per session (recommended)';

	/// en: 'Fire once when a session goes idle, then stay silent until either the session ends or you reply via this channel.'
	String get onceHint => 'Fire once when a session goes idle, then stay silent until either the session ends or you reply via this channel.';

	/// en: 'Time-window cooldown'
	String get cooldownLabel => 'Time-window cooldown';

	/// en: 'Suppress repeats for the same (session, event) within the chosen window.'
	String get cooldownHint => 'Suppress repeats for the same (session, event) within the chosen window.';

	/// en: 'Every event (noisy)'
	String get everyLabel => 'Every event (noisy)';

	/// en: 'No suppression. Use only for low-frequency channels.'
	String get everyHint => 'No suppression. Use only for low-frequency channels.';
}

// Path: web.channels.notifications.cooldowns
class TranslationsWebChannelsNotificationsCooldownsEn {
	TranslationsWebChannelsNotificationsCooldownsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '1 minute'
	String get k60 => '1 minute';

	/// en: '5 minutes'
	String get k300 => '5 minutes';

	/// en: '15 minutes'
	String get k900 => '15 minutes';

	/// en: '30 minutes'
	String get k1800 => '30 minutes';

	/// en: '1 hour'
	String get k3600 => '1 hour';
}

// Path: web.channels.notifications.snippetCaps
class TranslationsWebChannelsNotificationsSnippetCapsEn {
	TranslationsWebChannelsNotificationsSnippetCapsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'No cap — chunk into multiple messages (default)'
	String get k0 => 'No cap — chunk into multiple messages (default)';

	/// en: '1000 chars (terse)'
	String get k1000 => '1000 chars (terse)';

	/// en: '3000 chars'
	String get k3000 => '3000 chars';

	/// en: '6000 chars'
	String get k6000 => '6000 chars';

	/// en: '12000 chars'
	String get k12000 => '12000 chars';
}

// Path: web.plugins.mcp.columns
class TranslationsWebPluginsMcpColumnsEn {
	TranslationsWebPluginsMcpColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Name'
	String get name => 'Name';

	/// en: 'Transport'
	String get transport => 'Transport';

	/// en: 'Spec'
	String get spec => 'Spec';

	/// en: 'Enabled'
	String get enabled => 'Enabled';
}

// Path: web.plugins.mcp.editor
class TranslationsWebPluginsMcpEditorEn {
	TranslationsWebPluginsMcpEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'New MCP server'
	String get createTitle => 'New MCP server';

	/// en: 'Edit MCP: {id}'
	String editTitle({required Object id}) => 'Edit MCP: ${id}';

	/// en: 'JSON shape: <1>command</1>+<3>args</3>+<5>env</5> for stdio (default), or <7>transport</7> +<9> url</9>+<11>headers</11> for sse / http. Reference secrets as <13>${API_KEY}</13> — they get substituted at spawn time from the secrets file.'
	String description({required Object API_KEY}) => 'JSON shape: <1>command</1>+<3>args</3>+<5>env</5> for stdio (default), or <7>transport</7> +<9> url</9>+<11>headers</11> for sse / http. Reference secrets as <13>\$${API_KEY}</13> — they get substituted at spawn time from the secrets file.';

	/// en: 'ID'
	String get idLabel => 'ID';

	/// en: 'filesystem'
	String get idPlaceholder => 'filesystem';

	/// en: 'Lowercase / digits / dash / underscore. Becomes both the directory name and the default <1>name</1>.'
	String get idHint => 'Lowercase / digits / dash / underscore. Becomes both the directory name and the default <1>name</1>.';

	/// en: 'mcp.json'
	String get bodyLabel => 'mcp.json';

	/// en: 'Invalid JSON: {error}'
	String invalidJson({required Object error}) => 'Invalid JSON: ${error}';

	/// en: 'MCP server created'
	String get createdToast => 'MCP server created';

	/// en: 'MCP server saved'
	String get savedToast => 'MCP server saved';

	/// en: 'Create failed'
	String get createFailedToast => 'Create failed';

	/// en: 'Save failed'
	String get saveFailedToast => 'Save failed';
}

// Path: web.plugins.mcpSecrets.columns
class TranslationsWebPluginsMcpSecretsColumnsEn {
	TranslationsWebPluginsMcpSecretsColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Key'
	String get key => 'Key';

	/// en: 'Value'
	String get value => 'Value';
}

// Path: web.plugins.mcpSecrets.editor
class TranslationsWebPluginsMcpSecretsEditorEn {
	TranslationsWebPluginsMcpSecretsEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Add secret'
	String get addTitle => 'Add secret';

	/// en: 'Update {key}'
	String updateTitle({required Object key}) => 'Update ${key}';

	/// en: 'Stored encrypted on disk if the OS keychain is available. Reference it from any mcp.json env / headers / args / url with ${KEY}.'
	String addDescription({required Object KEY}) => 'Stored encrypted on disk if the OS keychain is available. Reference it from any mcp.json env / headers / args / url with \$${KEY}.';

	/// en: 'Enter the new value to overwrite. The previous value cannot be recovered.'
	String get editDescription => 'Enter the new value to overwrite. The previous value cannot be recovered.';

	/// en: 'Key'
	String get keyLabel => 'Key';

	/// en: 'BRAVE_API_KEY'
	String get keyPlaceholder => 'BRAVE_API_KEY';

	/// en: 'Must match <1>[A-Za-z_][A-Za-z0-9_]*</1>'
	String get keyPattern => 'Must match <1>[A-Za-z_][A-Za-z0-9_]*</1>';

	/// en: 'Already exists — use Edit instead, or pick a different name.'
	String get keyCollision => 'Already exists — use Edit instead, or pick a different name.';

	/// en: 'Value'
	String get valueLabel => 'Value';

	/// en: 'Hidden as you type. Saved value is never returned over the API.'
	String get valueHint => 'Hidden as you type. Saved value is never returned over the API.';

	/// en: 'Secret added'
	String get addedToast => 'Secret added';

	/// en: 'Secret updated'
	String get updatedToast => 'Secret updated';

	/// en: 'Save failed'
	String get saveFailedToast => 'Save failed';
}

// Path: web.plugins.skills.columns
class TranslationsWebPluginsSkillsColumnsEn {
	TranslationsWebPluginsSkillsColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'ID'
	String get id => 'ID';

	/// en: 'Description'
	String get description => 'Description';

	/// en: 'Source'
	String get source => 'Source';
}

// Path: web.plugins.skills.editor
class TranslationsWebPluginsSkillsEditorEn {
	TranslationsWebPluginsSkillsEditorEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'New skill'
	String get createTitle => 'New skill';

	/// en: 'Customize built-in: {id}'
	String customizeTitle({required Object id}) => 'Customize built-in: ${id}';

	/// en: 'Edit skill: {id}'
	String editTitle({required Object id}) => 'Edit skill: ${id}';

	/// en: 'You're viewing a built-in skill embedded in opendray. Saving will create a vault override at the same id — your edits live under ~/.opendray/vault/skills/<id>/SKILL.md and shadow the built-in until you Reset.'
	String get customizeDescription => 'You\'re viewing a built-in skill embedded in opendray. Saving will create a vault override at the same id — your edits live under ~/.opendray/vault/skills/<id>/SKILL.md and shadow the built-in until you Reset.';

	/// en: 'SKILL.md format — frontmatter with name + description, then markdown instructions. The description appears in the agent's Tier 1 index.'
	String get editDescription => 'SKILL.md format — frontmatter with name + description, then markdown instructions. The description appears in the agent\'s Tier 1 index.';

	/// en: 'ID'
	String get idLabel => 'ID';

	/// en: 'my-helper'
	String get idPlaceholder => 'my-helper';

	/// en: 'Lowercase / digits / dash / underscore. Becomes the directory name under <1>~/.opendray/vault/skills/&lt;id&gt;/</1>.'
	String get idHint => 'Lowercase / digits / dash / underscore. Becomes the directory name under <1>~/.opendray/vault/skills/&lt;id&gt;/</1>.';

	/// en: 'SKILL.md'
	String get bodyLabel => 'SKILL.md';

	/// en: 'Skill created'
	String get createdToast => 'Skill created';

	/// en: 'Skill saved'
	String get savedToast => 'Skill saved';

	/// en: 'Saved as vault override'
	String get savedOverrideToast => 'Saved as vault override';

	/// en: 'Create failed'
	String get createFailedToast => 'Create failed';

	/// en: 'Save failed'
	String get saveFailedToast => 'Save failed';

	/// en: 'Save as vault override'
	String get saveAsOverride => 'Save as vault override';
}

// Path: web.plugins.customTasks.columns
class TranslationsWebPluginsCustomTasksColumnsEn {
	TranslationsWebPluginsCustomTasksColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Name'
	String get name => 'Name';

	/// en: 'Command'
	String get command => 'Command';

	/// en: 'Scope'
	String get scope => 'Scope';
}

// Path: web.plugins.customTasks.dialog
class TranslationsWebPluginsCustomTasksDialogEn {
	TranslationsWebPluginsCustomTasksDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Add custom task'
	String get addTitle => 'Add custom task';

	/// en: 'Edit {name}'
	String editTitle({required Object name}) => 'Edit ${name}';

	/// en: 'The command is sent verbatim into the session's terminal. Same as typing it at the prompt and pressing Enter.'
	String get description => 'The command is sent verbatim into the session\'s terminal. Same as typing it at the prompt and pressing Enter.';

	/// en: 'Name'
	String get nameLabel => 'Name';

	/// en: 'dev'
	String get namePlaceholder => 'dev';

	/// en: 'Command'
	String get commandLabel => 'Command';

	/// en: 'docker compose up --build'
	String get commandPlaceholder => 'docker compose up --build';

	/// en: 'Description (optional)'
	String get descLabel => 'Description (optional)';

	/// en: 'Boots dev infra and tails logs'
	String get descPlaceholder => 'Boots dev infra and tails logs';

	/// en: 'Cwd scope (optional)'
	String get cwdLabel => 'Cwd scope (optional)';

	/// en: '/Users/me/projects/foo (blank = global)'
	String get cwdPlaceholder => '/Users/me/projects/foo  (blank = global)';

	/// en: 'Blank = visible in every session. Otherwise the task only shows when the session's cwd matches this absolute path.'
	String get cwdHint => 'Blank = visible in every session. Otherwise the task only shows when the session\'s cwd matches this absolute path.';

	/// en: 'Task added'
	String get addedToast => 'Task added';

	/// en: 'Task updated'
	String get updatedToast => 'Task updated';

	/// en: 'Add failed'
	String get addFailedToast => 'Add failed';

	/// en: 'Update failed'
	String get updateFailedToast => 'Update failed';
}

// Path: web.plugins.gitHosts.columns
class TranslationsWebPluginsGitHostsColumnsEn {
	TranslationsWebPluginsGitHostsColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Host'
	String get host => 'Host';

	/// en: 'Kind'
	String get kind => 'Kind';

	/// en: 'Token'
	String get token => 'Token';

	/// en: 'Enabled'
	String get enabled => 'Enabled';
}

// Path: web.plugins.gitHosts.dialog
class TranslationsWebPluginsGitHostsDialogEn {
	TranslationsWebPluginsGitHostsDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Add git host'
	String get addTitle => 'Add git host';

	/// en: 'Edit {host}'
	String editTitle({required Object host}) => 'Edit ${host}';

	/// en: 'Token is stored on the gateway. Used only for read-only API calls (list PRs, etc.).'
	String get description => 'Token is stored on the gateway. Used only for read-only API calls (list PRs, etc.).';

	/// en: 'Kind'
	String get kindLabel => 'Kind';

	/// en: 'GitHub'
	String get kindGitHub => 'GitHub';

	/// en: 'Gitea'
	String get kindGitea => 'Gitea';

	/// en: 'GitLab'
	String get kindGitLab => 'GitLab';

	/// en: 'Host'
	String get hostLabel => 'Host';

	/// en: 'github.com'
	String get hostPlaceholder => 'github.com';

	/// en: 'Display name (optional)'
	String get displayNameLabel => 'Display name (optional)';

	/// en: 'Personal'
	String get displayNamePlaceholder => 'Personal';

	/// en: 'Token'
	String get tokenLabel => 'Token';

	/// en: 'New token (leave blank to keep)'
	String get newTokenLabel => 'New token (leave blank to keep)';

	/// en: 'ghp_… / gho_… / glpat-…'
	String get tokenPlaceholder => 'ghp_… / gho_… / glpat-…';

	/// en: '…'
	String get tokenPlaceholderEdit => '…';

	/// en: 'GitHub: PAT with <1>repo</1> scope. Gitea: token with <3>read:repository</3>. GitLab: PAT with <5>read_api</5>.'
	String get tokenHint => 'GitHub: PAT with <1>repo</1> scope. Gitea: token with <3>read:repository</3>. GitLab: PAT with <5>read_api</5>.';

	/// en: 'Enabled'
	String get enabledLabel => 'Enabled';

	/// en: 'Git host added'
	String get addedToast => 'Git host added';

	/// en: 'Git host updated'
	String get updatedToast => 'Git host updated';

	/// en: 'Add failed'
	String get addFailedToast => 'Add failed';

	/// en: 'Update failed'
	String get updateFailedToast => 'Update failed';
}

// Path: web.backups.backupsTab.columns
class TranslationsWebBackupsBackupsTabColumnsEn {
	TranslationsWebBackupsBackupsTabColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'ID'
	String get id => 'ID';

	/// en: 'Target'
	String get target => 'Target';

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'Started'
	String get started => 'Started';

	/// en: 'Size'
	String get size => 'Size';

	/// en: 'Actions'
	String get actions => 'Actions';
}

// Path: web.backups.schedulesTab.columns
class TranslationsWebBackupsSchedulesTabColumnsEn {
	TranslationsWebBackupsSchedulesTabColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'ID'
	String get id => 'ID';

	/// en: 'Target'
	String get target => 'Target';

	/// en: 'Interval'
	String get interval => 'Interval';

	/// en: 'Keep'
	String get keep => 'Keep';

	/// en: 'Next run'
	String get nextRun => 'Next run';

	/// en: 'Enabled'
	String get enabled => 'Enabled';

	/// en: 'Actions'
	String get actions => 'Actions';
}

// Path: web.backups.targetsTab.columns
class TranslationsWebBackupsTargetsTabColumnsEn {
	TranslationsWebBackupsTargetsTabColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'ID'
	String get id => 'ID';

	/// en: 'Kind'
	String get kind => 'Kind';

	/// en: 'Config'
	String get config => 'Config';

	/// en: 'Enabled'
	String get enabled => 'Enabled';

	/// en: 'Actions'
	String get actions => 'Actions';
}

// Path: web.backups.targetEditor.local
class TranslationsWebBackupsTargetEditorLocalEn {
	TranslationsWebBackupsTargetEditorLocalEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Root directory'
	String get rootLabel => 'Root directory';

	/// en: 'Empty = cfg.backup.local_dir (~/.opendray/backups)'
	String get rootHint => 'Empty = cfg.backup.local_dir (~/.opendray/backups)';

	/// en: '~/backups/opendray or /mnt/external-hdd/opendray'
	String get rootPlaceholder => '~/backups/opendray  or  /mnt/external-hdd/opendray';
}

// Path: web.backups.targetEditor.smb
class TranslationsWebBackupsTargetEditorSmbEn {
	TranslationsWebBackupsTargetEditorSmbEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Host'
	String get hostLabel => 'Host';

	/// en: '192.168.9.8'
	String get hostPlaceholder => '192.168.9.8';

	/// en: 'Port'
	String get portLabel => 'Port';

	/// en: 'Share'
	String get shareLabel => 'Share';

	/// en: 'Top-level share name on the SMB server'
	String get shareHint => 'Top-level share name on the SMB server';

	/// en: 'Claude_Workspace'
	String get sharePlaceholder => 'Claude_Workspace';

	/// en: 'User'
	String get userLabel => 'User';

	/// en: 'Password'
	String get passwordLabel => 'Password';

	/// en: 'Path prefix'
	String get pathPrefixLabel => 'Path prefix';

	/// en: 'Sub-folder under the share root (optional)'
	String get pathPrefixHint => 'Sub-folder under the share root (optional)';

	/// en: 'opendray/backups'
	String get pathPrefixPlaceholder => 'opendray/backups';
}

// Path: web.backups.targetEditor.s3
class TranslationsWebBackupsTargetEditorS3En {
	TranslationsWebBackupsTargetEditorS3En.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Endpoint'
	String get endpointLabel => 'Endpoint';

	/// en: 'Host (no protocol). AWS: s3.amazonaws.com · R2: <accountid>.r2.cloudflarestorage.com · MinIO: minio.local:9000'
	String get endpointHint => 'Host (no protocol). AWS: s3.amazonaws.com · R2: <accountid>.r2.cloudflarestorage.com · MinIO: minio.local:9000';

	/// en: 's3.amazonaws.com'
	String get endpointPlaceholder => 's3.amazonaws.com';

	/// en: 'Region'
	String get regionLabel => 'Region';

	/// en: 'AWS only; R2 use 'auto''
	String get regionHint => 'AWS only; R2 use \'auto\'';

	/// en: 'us-east-1 / auto'
	String get regionPlaceholder => 'us-east-1 / auto';

	/// en: 'Bucket'
	String get bucketLabel => 'Bucket';

	/// en: 'opendray-backups'
	String get bucketPlaceholder => 'opendray-backups';

	/// en: 'Access key'
	String get accessKeyLabel => 'Access key';

	/// en: 'Secret key'
	String get secretKeyLabel => 'Secret key';

	/// en: 'Stored AES-256-GCM encrypted; never echoed back'
	String get secretKeyHint => 'Stored AES-256-GCM encrypted; never echoed back';

	/// en: 'Path prefix'
	String get pathPrefixLabel => 'Path prefix';

	/// en: 'Object-key prefix (optional)'
	String get pathPrefixHint => 'Object-key prefix (optional)';

	/// en: 'opendray/backups'
	String get pathPrefixPlaceholder => 'opendray/backups';

	/// en: 'Use HTTPS'
	String get useHttps => 'Use HTTPS';

	/// en: 'Path-style addressing (legacy / MinIO)'
	String get pathStyle => 'Path-style addressing (legacy / MinIO)';
}

// Path: web.backups.targetEditor.webdav
class TranslationsWebBackupsTargetEditorWebdavEn {
	TranslationsWebBackupsTargetEditorWebdavEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Base URL'
	String get baseUrlLabel => 'Base URL';

	/// en: 'Full URL including any path. Examples: https://cloud.example.com/remote.php/dav/files/me/ (Nextcloud), https://nas.local:5006/ (Synology), https://dav.jianguoyun.com/dav/ (Jianguoyun / 坚果云)'
	String get baseUrlHint => 'Full URL including any path. Examples: https://cloud.example.com/remote.php/dav/files/me/ (Nextcloud), https://nas.local:5006/ (Synology), https://dav.jianguoyun.com/dav/ (Jianguoyun / 坚果云)';

	/// en: 'https://cloud.example.com/remote.php/dav/files/<user>/'
	String get baseUrlPlaceholder => 'https://cloud.example.com/remote.php/dav/files/<user>/';

	/// en: 'User'
	String get userLabel => 'User';

	/// en: 'Password'
	String get passwordLabel => 'Password';

	/// en: 'Path prefix'
	String get pathPrefixLabel => 'Path prefix';

	/// en: 'Sub-folder under the base URL (optional)'
	String get pathPrefixHint => 'Sub-folder under the base URL (optional)';

	/// en: 'opendray/backups'
	String get pathPrefixPlaceholder => 'opendray/backups';
}

// Path: web.backups.targetEditor.sftp
class TranslationsWebBackupsTargetEditorSftpEn {
	TranslationsWebBackupsTargetEditorSftpEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Host'
	String get hostLabel => 'Host';

	/// en: 'vps.example.com'
	String get hostPlaceholder => 'vps.example.com';

	/// en: 'Port'
	String get portLabel => 'Port';

	/// en: 'User'
	String get userLabel => 'User';

	/// en: 'Password'
	String get passwordLabel => 'Password';

	/// en: 'Either password OR private key required. If both, password is treated as the key passphrase.'
	String get passwordHint => 'Either password OR private key required. If both, password is treated as the key passphrase.';

	/// en: 'Private key (PEM)'
	String get privateKeyLabel => 'Private key (PEM)';

	/// en: 'Paste contents of an OpenSSH/PEM private key (e.g. ~/.ssh/id_ed25519). Leave blank for password-only auth.'
	String get privateKeyHint => 'Paste contents of an OpenSSH/PEM private key (e.g. ~/.ssh/id_ed25519). Leave blank for password-only auth.';

	/// en: '-----BEGIN OPENSSH PRIVATE KEY-----...'
	String get privateKeyPlaceholder => '-----BEGIN OPENSSH PRIVATE KEY-----...';

	/// en: 'Host key (pinning)'
	String get hostKeyLabel => 'Host key (pinning)';

	/// en: 'OpenSSH-style server public key (run `ssh-keyscan host` to obtain). Leave blank to disable pinning (NOT recommended outside LAN).'
	String get hostKeyHint => 'OpenSSH-style server public key (run `ssh-keyscan host` to obtain). Leave blank to disable pinning (NOT recommended outside LAN).';

	/// en: 'ssh-ed25519 AAAA...'
	String get hostKeyPlaceholder => 'ssh-ed25519 AAAA...';

	/// en: 'Path prefix'
	String get pathPrefixLabel => 'Path prefix';

	/// en: 'Absolute or relative to user home (optional)'
	String get pathPrefixHint => 'Absolute or relative to user home (optional)';

	/// en: '/var/backups/opendray or opendray-backups'
	String get pathPrefixPlaceholder => '/var/backups/opendray  or  opendray-backups';
}

// Path: web.backups.targetEditor.rclone
class TranslationsWebBackupsTargetEditorRcloneEn {
	TranslationsWebBackupsTargetEditorRcloneEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Requires the <1>rclone</1> CLI installed on the opendray host. First configure your remote with <3>rclone config</3>, then use the remote name below. opendray invokes <5>rclone rcat / cat / lsd</5> under the hood.'
	String get rcloneHint => 'Requires the <1>rclone</1> CLI installed on the opendray host. First configure your remote with <3>rclone config</3>, then use the remote name below. opendray invokes <5>rclone rcat / cat / lsd</5> under the hood.';

	/// en: 'Remote name'
	String get remoteLabel => 'Remote name';

	/// en: 'Name from `rclone config` (no colon). Example: gdrive, onedrive, dropbox-personal, baidu-pan'
	String get remoteHint => 'Name from `rclone config` (no colon). Example: gdrive, onedrive, dropbox-personal, baidu-pan';

	/// en: 'gdrive'
	String get remotePlaceholder => 'gdrive';

	/// en: 'Path prefix'
	String get pathPrefixLabel => 'Path prefix';

	/// en: 'Sub-folder under the remote root (optional)'
	String get pathPrefixHint => 'Sub-folder under the remote root (optional)';

	/// en: 'opendray/backups'
	String get pathPrefixPlaceholder => 'opendray/backups';

	/// en: 'Binary path'
	String get binaryPathLabel => 'Binary path';

	/// en: 'Override `which rclone`. Empty uses PATH lookup.'
	String get binaryPathHint => 'Override `which rclone`. Empty uses PATH lookup.';

	/// en: '/opt/homebrew/bin/rclone'
	String get binaryPathPlaceholder => '/opt/homebrew/bin/rclone';

	/// en: 'Config path'
	String get configPathLabel => 'Config path';

	/// en: 'Override --config (default ~/.config/rclone/rclone.conf or ~/.rclone.conf)'
	String get configPathHint => 'Override --config (default ~/.config/rclone/rclone.conf or ~/.rclone.conf)';

	/// en: 'leave blank for rclone default'
	String get configPathPlaceholder => 'leave blank for rclone default';
}

// Path: web.serverSettings.sections.general
class TranslationsWebServerSettingsSectionsGeneralEn {
	TranslationsWebServerSettingsSectionsGeneralEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'General'
	String get title => 'General';

	/// en: 'Listen address, operator account, token TTL.'
	String get desc => 'Listen address, operator account, token TTL.';
}

// Path: web.serverSettings.sections.logging
class TranslationsWebServerSettingsSectionsLoggingEn {
	TranslationsWebServerSettingsSectionsLoggingEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Logging'
	String get title => 'Logging';

	/// en: 'Verbosity, format, and live tail.'
	String get desc => 'Verbosity, format, and live tail.';
}

// Path: web.serverSettings.sections.sessions
class TranslationsWebServerSettingsSectionsSessionsEn {
	TranslationsWebServerSettingsSectionsSessionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Sessions'
	String get title => 'Sessions';

	/// en: 'Idle detection thresholds.'
	String get desc => 'Idle detection thresholds.';
}

// Path: web.serverSettings.sections.vault
class TranslationsWebServerSettingsSectionsVaultEn {
	TranslationsWebServerSettingsSectionsVaultEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Vault'
	String get title => 'Vault';

	/// en: 'Notes, skills, and git-versioned root.'
	String get desc => 'Notes, skills, and git-versioned root.';
}

// Path: web.serverSettings.sections.mcp
class TranslationsWebServerSettingsSectionsMcpEn {
	TranslationsWebServerSettingsSectionsMcpEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'MCP registry'
	String get title => 'MCP registry';

	/// en: 'Server registry + secrets.'
	String get desc => 'Server registry + secrets.';
}

// Path: web.serverSettings.sections.memory
class TranslationsWebServerSettingsSectionsMemoryEn {
	TranslationsWebServerSettingsSectionsMemoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memory'
	String get title => 'Memory';

	/// en: 'Cross-CLI persistent memory subsystem.'
	String get desc => 'Cross-CLI persistent memory subsystem.';
}

// Path: web.serverSettings.sections.backup
class TranslationsWebServerSettingsSectionsBackupEn {
	TranslationsWebServerSettingsSectionsBackupEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Backup'
	String get title => 'Backup';

	/// en: 'Encrypted DB backups, restore, and admin data exports.'
	String get desc => 'Encrypted DB backups, restore, and admin data exports.';
}

// Path: web.serverSettings.sections.claude
class TranslationsWebServerSettingsSectionsClaudeEn {
	TranslationsWebServerSettingsSectionsClaudeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Storage · Claude'
	String get title => 'Storage · Claude';

	/// en: 'Where Claude transcripts live on disk.'
	String get desc => 'Where Claude transcripts live on disk.';
}

// Path: web.serverSettings.sections.codex
class TranslationsWebServerSettingsSectionsCodexEn {
	TranslationsWebServerSettingsSectionsCodexEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Storage · Codex'
	String get title => 'Storage · Codex';

	/// en: 'Codex sessions root.'
	String get desc => 'Codex sessions root.';
}

// Path: web.serverSettings.sections.gemini
class TranslationsWebServerSettingsSectionsGeminiEn {
	TranslationsWebServerSettingsSectionsGeminiEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Storage · Gemini'
	String get title => 'Storage · Gemini';

	/// en: 'Gemini per-project tmp + projects.json.'
	String get desc => 'Gemini per-project tmp + projects.json.';
}

// Path: web.serverSettings.fields.listenAddress
class TranslationsWebServerSettingsFieldsListenAddressEn {
	TranslationsWebServerSettingsFieldsListenAddressEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Listen address'
	String get label => 'Listen address';

	/// en: 'The host:port the HTTP server binds to. Example: 0.0.0.0:8770.'
	String get hint => 'The host:port the HTTP server binds to. Example: 0.0.0.0:8770.';
}

// Path: web.serverSettings.fields.username
class TranslationsWebServerSettingsFieldsUsernameEn {
	TranslationsWebServerSettingsFieldsUsernameEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Username'
	String get label => 'Username';

	/// en: 'Login name used in the sign-in form. Changing this forces a re-login on the next request.'
	String get hint => 'Login name used in the sign-in form. Changing this forces a re-login on the next request.';
}

// Path: web.serverSettings.fields.password
class TranslationsWebServerSettingsFieldsPasswordEn {
	TranslationsWebServerSettingsFieldsPasswordEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Password'
	String get label => 'Password';

	/// en: 'Leave blank to keep the current password. Sending a value overwrites it.'
	String get hint => 'Leave blank to keep the current password. Sending a value overwrites it.';

	/// en: 'Hide'
	String get hideTitle => 'Hide';

	/// en: 'Reveal'
	String get revealTitle => 'Reveal';
}

// Path: web.serverSettings.fields.tokenTTL
class TranslationsWebServerSettingsFieldsTokenTTLEn {
	TranslationsWebServerSettingsFieldsTokenTTLEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Token TTL'
	String get label => 'Token TTL';

	/// en: 'Bearer-token lifetime as a Go duration, e.g. "24h", "30m". Empty = never expire.'
	String get hint => 'Bearer-token lifetime as a Go duration, e.g. "24h", "30m". Empty = never expire.';
}

// Path: web.serverSettings.fields.logLevel
class TranslationsWebServerSettingsFieldsLogLevelEn {
	TranslationsWebServerSettingsFieldsLogLevelEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Log level'
	String get label => 'Log level';

	/// en: 'Lines below this level are dropped.'
	String get hint => 'Lines below this level are dropped.';
}

// Path: web.serverSettings.fields.logFormat
class TranslationsWebServerSettingsFieldsLogFormatEn {
	TranslationsWebServerSettingsFieldsLogFormatEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Format'
	String get label => 'Format';

	/// en: '"text" is human-readable; "json" is machine-parsable.'
	String get hint => '"text" is human-readable; "json" is machine-parsable.';
}

// Path: web.serverSettings.fields.logFile
class TranslationsWebServerSettingsFieldsLogFileEn {
	TranslationsWebServerSettingsFieldsLogFileEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Log file'
	String get label => 'Log file';

	/// en: 'Optional file path. Auto-rotates at 10 MB, keeps 5 backups. Empty = stderr only.'
	String get hint => 'Optional file path. Auto-rotates at 10 MB, keeps 5 backups. Empty = stderr only.';
}

// Path: web.serverSettings.fields.idleThreshold
class TranslationsWebServerSettingsFieldsIdleThresholdEn {
	TranslationsWebServerSettingsFieldsIdleThresholdEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Idle threshold'
	String get label => 'Idle threshold';

	/// en: 'A session is silent for this long before session.idle fires. Empty = 30s.'
	String get hint => 'A session is silent for this long before session.idle fires. Empty = 30s.';
}

// Path: web.serverSettings.fields.idlePollInterval
class TranslationsWebServerSettingsFieldsIdlePollIntervalEn {
	TranslationsWebServerSettingsFieldsIdlePollIntervalEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Idle poll interval'
	String get label => 'Idle poll interval';

	/// en: 'How often the idle detector wakes up. Lower = lower latency, more wakeups. Empty = 5s.'
	String get hint => 'How often the idle detector wakes up. Lower = lower latency, more wakeups. Empty = 5s.';
}

// Path: web.serverSettings.fields.vaultRoot
class TranslationsWebServerSettingsFieldsVaultRootEn {
	TranslationsWebServerSettingsFieldsVaultRootEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Vault root'
	String get label => 'Vault root';

	/// en: 'Top-level directory for notes, skills, and MCP registry.'
	String get hint => 'Top-level directory for notes, skills, and MCP registry.';
}

// Path: web.serverSettings.fields.notesDirectory
class TranslationsWebServerSettingsFieldsNotesDirectoryEn {
	TranslationsWebServerSettingsFieldsNotesDirectoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Notes directory'
	String get label => 'Notes directory';

	/// en: 'Override notes location. Defaults to <vault root>/notes.'
	String get hint => 'Override notes location. Defaults to <vault root>/notes.';
}

// Path: web.serverSettings.fields.skillsDirectory
class TranslationsWebServerSettingsFieldsSkillsDirectoryEn {
	TranslationsWebServerSettingsFieldsSkillsDirectoryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Skills directory'
	String get label => 'Skills directory';

	/// en: 'Override skills location. Defaults to <vault root>/skills.'
	String get hint => 'Override skills location. Defaults to <vault root>/skills.';
}

// Path: web.serverSettings.fields.gitRoot
class TranslationsWebServerSettingsFieldsGitRootEn {
	TranslationsWebServerSettingsFieldsGitRootEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Git root'
	String get label => 'Git root';

	/// en: 'Working tree the Vault Sync feature commits to.'
	String get hint => 'Working tree the Vault Sync feature commits to.';
}

// Path: web.serverSettings.fields.personalPrefix
class TranslationsWebServerSettingsFieldsPersonalPrefixEn {
	TranslationsWebServerSettingsFieldsPersonalPrefixEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Personal prefix'
	String get label => 'Personal prefix';

	/// en: 'Folder name used for personal notes when auto-deriving paths. Default "personal".'
	String get hint => 'Folder name used for personal notes when auto-deriving paths. Default "personal".';
}

// Path: web.serverSettings.fields.projectsPrefix
class TranslationsWebServerSettingsFieldsProjectsPrefixEn {
	TranslationsWebServerSettingsFieldsProjectsPrefixEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Projects prefix'
	String get label => 'Projects prefix';

	/// en: 'Folder name used for project notes. Default "projects".'
	String get hint => 'Folder name used for project notes. Default "projects".';
}

// Path: web.serverSettings.fields.registryRoot
class TranslationsWebServerSettingsFieldsRegistryRootEn {
	TranslationsWebServerSettingsFieldsRegistryRootEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Registry root'
	String get label => 'Registry root';

	/// en: 'Directory holding MCP server JSON definitions. Defaults to <vault>/mcp.'
	String get hint => 'Directory holding MCP server JSON definitions. Defaults to <vault>/mcp.';
}

// Path: web.serverSettings.fields.secretsFile
class TranslationsWebServerSettingsFieldsSecretsFileEn {
	TranslationsWebServerSettingsFieldsSecretsFileEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Secrets file'
	String get label => 'Secrets file';

	/// en: 'key=value file substituted into MCP server commands at spawn time.'
	String get hint => 'key=value file substituted into MCP server commands at spawn time.';
}

// Path: web.serverSettings.fields.memoryBackend
class TranslationsWebServerSettingsFieldsMemoryBackendEn {
	TranslationsWebServerSettingsFieldsMemoryBackendEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Embedder backend'
	String get label => 'Embedder backend';

	/// en: '"auto" / "bm25" use the cgo-free pure-Go keyword path. "http" calls any OpenAI-compatible /v1/embeddings (ollama / OpenAI / LocalAI). "local" runs an ONNX sentence-transformer in-process — requires a binary built with `-tags local_onnx`.'
	String get hint => '"auto" / "bm25" use the cgo-free pure-Go keyword path. "http" calls any OpenAI-compatible /v1/embeddings (ollama / OpenAI / LocalAI). "local" runs an ONNX sentence-transformer in-process — requires a binary built with `-tags local_onnx`.';
}

// Path: web.serverSettings.fields.memoryStore
class TranslationsWebServerSettingsFieldsMemoryStoreEn {
	TranslationsWebServerSettingsFieldsMemoryStoreEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Store'
	String get label => 'Store';

	/// en: '"pgvector" reuses opendray's existing PG with the vector extension; only option in v1.'
	String get hint => '"pgvector" reuses opendray\'s existing PG with the vector extension; only option in v1.';
}

// Path: web.serverSettings.fields.memoryTopK
class TranslationsWebServerSettingsFieldsMemoryTopKEn {
	TranslationsWebServerSettingsFieldsMemoryTopKEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Default top-K'
	String get label => 'Default top-K';

	/// en: 'How many hits memory_search returns when the agent doesn't specify. Empty = 5.'
	String get hint => 'How many hits memory_search returns when the agent doesn\'t specify. Empty = 5.';
}

// Path: web.serverSettings.fields.memoryThreshold
class TranslationsWebServerSettingsFieldsMemoryThresholdEn {
	TranslationsWebServerSettingsFieldsMemoryThresholdEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Similarity threshold'
	String get label => 'Similarity threshold';

	/// en: 'Hits below this score are dropped. Empty = 0.1 (permissive — BM25 sparse vectors rarely break 0.5).'
	String get hint => 'Hits below this score are dropped. Empty = 0.1 (permissive — BM25 sparse vectors rarely break 0.5).';
}

// Path: web.serverSettings.fields.memoryScope
class TranslationsWebServerSettingsFieldsMemoryScopeEn {
	TranslationsWebServerSettingsFieldsMemoryScopeEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Default scope'
	String get label => 'Default scope';

	/// en: 'What memory_store uses when the agent doesn't specify. "project" (recommended) groups by cwd; "session" isolates per session; "global" shares across cwds.'
	String get hint => 'What memory_store uses when the agent doesn\'t specify. "project" (recommended) groups by cwd; "session" isolates per session; "global" shares across cwds.';
}

// Path: web.serverSettings.fields.memoryBaseUrl
class TranslationsWebServerSettingsFieldsMemoryBaseUrlEn {
	TranslationsWebServerSettingsFieldsMemoryBaseUrlEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Base URL'
	String get label => 'Base URL';

	/// en: 'e.g. "http://localhost:11434/v1" for ollama, "https://api.openai.com/v1" for OpenAI.'
	String get hint => 'e.g. "http://localhost:11434/v1" for ollama, "https://api.openai.com/v1" for OpenAI.';
}

// Path: web.serverSettings.fields.memoryModel
class TranslationsWebServerSettingsFieldsMemoryModelEn {
	TranslationsWebServerSettingsFieldsMemoryModelEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Model'
	String get label => 'Model';

	/// en: 'e.g. "nomic-embed-text" for ollama, "text-embedding-3-small" for OpenAI.'
	String get hint => 'e.g. "nomic-embed-text" for ollama, "text-embedding-3-small" for OpenAI.';
}

// Path: web.serverSettings.fields.memoryApiKey
class TranslationsWebServerSettingsFieldsMemoryApiKeyEn {
	TranslationsWebServerSettingsFieldsMemoryApiKeyEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'API key'
	String get label => 'API key';

	/// en: 'Empty for ollama / local servers. Required for OpenAI / Voyage / hosted services.'
	String get hint => 'Empty for ollama / local servers. Required for OpenAI / Voyage / hosted services.';
}

// Path: web.serverSettings.fields.memoryLocalModel
class TranslationsWebServerSettingsFieldsMemoryLocalModelEn {
	TranslationsWebServerSettingsFieldsMemoryLocalModelEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Model name'
	String get label => 'Model name';

	/// en: 'Cosmetic — appears in logs / Inspector. e.g. "bge-m3", "bge-small-en-v1.5".'
	String get hint => 'Cosmetic — appears in logs / Inspector. e.g. "bge-m3", "bge-small-en-v1.5".';
}

// Path: web.serverSettings.fields.memoryLibraryPath
class TranslationsWebServerSettingsFieldsMemoryLibraryPathEn {
	TranslationsWebServerSettingsFieldsMemoryLibraryPathEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Library path'
	String get label => 'Library path';

	/// en: 'Directory holding libonnxruntime.dylib (macOS) / libonnxruntime.so (Linux). After `brew install onnxruntime`, that's /opt/homebrew/opt/onnxruntime/lib.'
	String get hint => 'Directory holding libonnxruntime.dylib (macOS) / libonnxruntime.so (Linux). After `brew install onnxruntime`, that\'s /opt/homebrew/opt/onnxruntime/lib.';
}

// Path: web.serverSettings.fields.memoryModelPath
class TranslationsWebServerSettingsFieldsMemoryModelPathEn {
	TranslationsWebServerSettingsFieldsMemoryModelPathEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Model path'
	String get label => 'Model path';

	/// en: 'Absolute path to the .onnx weights. Download from HuggingFace, e.g. Xenova/bge-m3 or Xenova/bge-small-en-v1.5.'
	String get hint => 'Absolute path to the .onnx weights. Download from HuggingFace, e.g. Xenova/bge-m3 or Xenova/bge-small-en-v1.5.';
}

// Path: web.serverSettings.fields.memoryTokenizerPath
class TranslationsWebServerSettingsFieldsMemoryTokenizerPathEn {
	TranslationsWebServerSettingsFieldsMemoryTokenizerPathEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Tokenizer path'
	String get label => 'Tokenizer path';

	/// en: 'Absolute path to tokenizer.json (HuggingFace standard format) — usually right next to the model.'
	String get hint => 'Absolute path to tokenizer.json (HuggingFace standard format) — usually right next to the model.';
}

// Path: web.serverSettings.fields.memoryMaxSeqLen
class TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEn {
	TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Max sequence length'
	String get label => 'Max sequence length';

	/// en: 'Tokens beyond this are truncated. bge-m3 default is 512. Empty = 512.'
	String get hint => 'Tokens beyond this are truncated. bge-m3 default is 512. Empty = 512.';
}

// Path: web.serverSettings.fields.claudeHistoryRoots
class TranslationsWebServerSettingsFieldsClaudeHistoryRootsEn {
	TranslationsWebServerSettingsFieldsClaudeHistoryRootsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'History roots'
	String get label => 'History roots';

	/// en: 'Directories scanned for Claude per-project JSONL transcripts. Empty = scan ~/.claude/projects + every ~/.claude-accounts/*/projects.'
	String get hint => 'Directories scanned for Claude per-project JSONL transcripts. Empty = scan ~/.claude/projects + every ~/.claude-accounts/*/projects.';
}

// Path: web.serverSettings.fields.claudeAccountsDir
class TranslationsWebServerSettingsFieldsClaudeAccountsDirEn {
	TranslationsWebServerSettingsFieldsClaudeAccountsDirEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Accounts directory'
	String get label => 'Accounts directory';

	/// en: 'Root used for opendray-managed Claude account ConfigDirs. Default ~/.claude-accounts.'
	String get hint => 'Root used for opendray-managed Claude account ConfigDirs. Default ~/.claude-accounts.';
}

// Path: web.serverSettings.fields.codexSessionsRoot
class TranslationsWebServerSettingsFieldsCodexSessionsRootEn {
	TranslationsWebServerSettingsFieldsCodexSessionsRootEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Sessions root'
	String get label => 'Sessions root';

	/// en: 'Directory walked for Codex rollout JSONL files. Default ~/.codex/sessions.'
	String get hint => 'Directory walked for Codex rollout JSONL files. Default ~/.codex/sessions.';
}

// Path: web.serverSettings.fields.geminiTmpRoot
class TranslationsWebServerSettingsFieldsGeminiTmpRootEn {
	TranslationsWebServerSettingsFieldsGeminiTmpRootEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Tmp directory'
	String get label => 'Tmp directory';

	/// en: 'Root holding Gemini per-project tmp folders. Default ~/.gemini/tmp.'
	String get hint => 'Root holding Gemini per-project tmp folders. Default ~/.gemini/tmp.';
}

// Path: web.serverSettings.fields.geminiProjectsFile
class TranslationsWebServerSettingsFieldsGeminiProjectsFileEn {
	TranslationsWebServerSettingsFieldsGeminiProjectsFileEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'projects.json'
	String get label => 'projects.json';

	/// en: 'Path to Gemini's cwd→short-name mapping file. Default ~/.gemini/projects.json.'
	String get hint => 'Path to Gemini\'s cwd→short-name mapping file. Default ~/.gemini/projects.json.';
}

// Path: web.serverSettings.fields.backupLocalDir
class TranslationsWebServerSettingsFieldsBackupLocalDirEn {
	TranslationsWebServerSettingsFieldsBackupLocalDirEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Local backup directory'
	String get label => 'Local backup directory';

	/// en: 'Default root for the auto-created `local` target. Empty = ~/.opendray/backups. Restart required.'
	String get hint => 'Default root for the auto-created `local` target. Empty = ~/.opendray/backups. Restart required.';
}

// Path: web.serverSettings.fields.backupExportDir
class TranslationsWebServerSettingsFieldsBackupExportDirEn {
	TranslationsWebServerSettingsFieldsBackupExportDirEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Export directory'
	String get label => 'Export directory';

	/// en: 'Where one-shot export zips are staged on disk. Empty = ~/.opendray/exports. Bundles auto-expire after 24h. Restart required.'
	String get hint => 'Where one-shot export zips are staged on disk. Empty = ~/.opendray/exports. Bundles auto-expire after 24h. Restart required.';
}

// Path: web.serverSettings.fields.backupPgDumpPath
class TranslationsWebServerSettingsFieldsBackupPgDumpPathEn {
	TranslationsWebServerSettingsFieldsBackupPgDumpPathEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'pg_dump path'
	String get label => 'pg_dump path';

	/// en: 'Absolute path to pg_dump. Major version must be ≥ the server's. Empty = first pg_dump on PATH.'
	String get hint => 'Absolute path to pg_dump. Major version must be ≥ the server\'s. Empty = first pg_dump on PATH.';
}

// Path: web.serverSettings.fields.backupPgRestorePath
class TranslationsWebServerSettingsFieldsBackupPgRestorePathEn {
	TranslationsWebServerSettingsFieldsBackupPgRestorePathEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'pg_restore path'
	String get label => 'pg_restore path';

	/// en: 'Absolute path to pg_restore for the /backups/restore flow. Same major-version rule.'
	String get hint => 'Absolute path to pg_restore for the /backups/restore flow. Same major-version rule.';
}

// Path: web.serverSettings.httpHelpers.presetTip
class TranslationsWebServerSettingsHttpHelpersPresetTipEn {
	TranslationsWebServerSettingsHttpHelpersPresetTipEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Local ollama daemon'
	String get ollama => 'Local ollama daemon';

	/// en: 'LM Studio local server'
	String get lmStudio => 'LM Studio local server';

	/// en: 'OpenAI cloud (needs API key)'
	String get openai => 'OpenAI cloud (needs API key)';
}

// Path: web.serverSettings.backup.scheduleHeaders
class TranslationsWebServerSettingsBackupScheduleHeadersEn {
	TranslationsWebServerSettingsBackupScheduleHeadersEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Schedule'
	String get schedule => 'Schedule';

	/// en: 'Target'
	String get target => 'Target';

	/// en: 'Cadence'
	String get cadence => 'Cadence';

	/// en: 'Keep'
	String get keep => 'Keep';

	/// en: 'State'
	String get state => 'State';
}

// Path: web.settings.appearance.options
class TranslationsWebSettingsAppearanceOptionsEn {
	TranslationsWebSettingsAppearanceOptionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Light'
	String get light => 'Light';

	/// en: 'Always light'
	String get lightDesc => 'Always light';

	/// en: 'Dark'
	String get dark => 'Dark';

	/// en: 'Always dark'
	String get darkDesc => 'Always dark';

	/// en: 'System'
	String get system => 'System';

	/// en: 'Follow the OS setting'
	String get systemDesc => 'Follow the OS setting';
}

// Path: web.settings.font.options
class TranslationsWebSettingsFontOptionsEn {
	TranslationsWebSettingsFontOptionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Compact'
	String get compact => 'Compact';

	/// en: 'Default'
	String get kDefault => 'Default';

	/// en: 'Comfy'
	String get comfy => 'Comfy';

	/// en: 'Large'
	String get large => 'Large';
}

// Path: web.memoryAmbient.providers.row
class TranslationsWebMemoryAmbientProvidersRowEn {
	TranslationsWebMemoryAmbientProvidersRowEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: '★ default'
	String get defaultBadge => '★ default';

	/// en: 'Make default'
	String get makeDefault => 'Make default';

	/// en: 'Test'
	String get test => 'Test';

	/// en: 'Testing…'
	String get testing => 'Testing…';

	/// en: 'Delete'
	String get delete => 'Delete';

	/// en: '{name}: connection OK'
	String testOk({required Object name}) => '${name}: connection OK';

	/// en: 'Test failed'
	String get testFailedToast => 'Test failed';

	/// en: 'Delete provider "{name}"?'
	String deleteConfirm({required Object name}) => 'Delete provider "${name}"?';

	/// en: 'Provider deleted'
	String get deletedToast => 'Provider deleted';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	/// en: 'Update failed'
	String get updateFailedToast => 'Update failed';

	/// en: '{name} is now the default'
	String madeDefaultToast({required Object name}) => '${name} is now the default';
}

// Path: web.memoryAmbient.providers.dialog
class TranslationsWebMemoryAmbientProvidersDialogEn {
	TranslationsWebMemoryAmbientProvidersDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Add summarizer provider'
	String get title => 'Add summarizer provider';

	/// en: 'Kind'
	String get kindLabel => 'Kind';

	/// en: 'Name'
	String get nameLabel => 'Name';

	/// en: 'e.g. lmstudio-qwen'
	String get namePlaceholder => 'e.g. lmstudio-qwen';

	/// en: 'Model'
	String get modelLabel => 'Model';

	/// en: 'Base URL'
	String get baseUrlLabel => 'Base URL';

	/// en: 'Integration providers resolve their base URL from a registered integration. Configure that under Integrations first; advanced wiring (extra_config) is DB-only in this release.'
	String get integrationNote => 'Integration providers resolve their base URL from a registered integration. Configure that under Integrations first; advanced wiring (extra_config) is DB-only in this release.';

	/// en: 'API key'
	String get apiKeyLabel => 'API key';

	/// en: 'Stored encrypted (AES-GCM with the backup master passphrase). Never echoed back; only the fingerprint is shown after save.'
	String get apiKeyHint => 'Stored encrypted (AES-GCM with the backup master passphrase). Never echoed back; only the fingerprint is shown after save.';

	/// en: 'Make this the default provider'
	String get makeDefaultLabel => 'Make this the default provider';

	/// en: 'Create'
	String get create => 'Create';

	/// en: 'Name is required'
	String get nameRequiredToast => 'Name is required';

	/// en: 'Provider {name} created'
	String createdToast({required Object name}) => 'Provider ${name} created';

	/// en: 'Create failed'
	String get createFailedToast => 'Create failed';
}

// Path: web.memoryAmbient.rules.row
class TranslationsWebMemoryAmbientRulesRowEn {
	TranslationsWebMemoryAmbientRulesRowEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'global default'
	String get globalDefault => 'global default';

	/// en: 'scope:'
	String get scopeLabel => 'scope:';

	/// en: 'dedup:'
	String get dedupLabel => 'dedup:';

	/// en: 'Run now'
	String get runNow => 'Run now';

	/// en: 'Running…'
	String get running => 'Running…';

	/// en: 'Delete'
	String get delete => 'Delete';

	/// en: 'Rule fired across {sessions} session(s)'
	String firedToast({required Object sessions}) => 'Rule fired across ${sessions} session(s)';

	/// en: 'Run-now failed'
	String get runNowFailedToast => 'Run-now failed';

	/// en: 'Delete rule "{name}"?'
	String deleteConfirm({required Object name}) => 'Delete rule "${name}"?';

	/// en: 'Rule deleted'
	String get deletedToast => 'Rule deleted';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';

	late final TranslationsWebMemoryAmbientRulesRowSummaryEn summary = TranslationsWebMemoryAmbientRulesRowSummaryEn.internal(_root);
}

// Path: web.memoryAmbient.rules.dialog
class TranslationsWebMemoryAmbientRulesDialogEn {
	TranslationsWebMemoryAmbientRulesDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Add capture rule'
	String get title => 'Add capture rule';

	/// en: 'Name'
	String get nameLabel => 'Name';

	/// en: 'Trigger'
	String get triggerLabel => 'Trigger';

	/// en: 'N (messages)'
	String get nLabel => 'N (messages)';

	/// en: 'Idle seconds'
	String get idleLabel => 'Idle seconds';

	/// en: 'K (characters)'
	String get kLabel => 'K (characters)';

	/// en: 'Target scope'
	String get scopeLabel => 'Target scope';

	/// en: 'session'
	String get scopeSession => 'session';

	/// en: 'project (recommended)'
	String get scopeProject => 'project (recommended)';

	/// en: 'global'
	String get scopeGlobal => 'global';

	/// en: 'Dedup threshold (0.0 – 1.0)'
	String get dedupLabel => 'Dedup threshold (0.0 – 1.0)';

	/// en: 'Higher = stricter de-duplication. 0.85 is the recommended sweet spot.'
	String get dedupHint => 'Higher = stricter de-duplication. 0.85 is the recommended sweet spot.';

	/// en: 'Create'
	String get create => 'Create';

	/// en: 'Name is required'
	String get nameRequiredToast => 'Name is required';

	/// en: 'Rule {name} created'
	String createdToast({required Object name}) => 'Rule ${name} created';

	/// en: 'Create failed'
	String get createFailedToast => 'Create failed';
}

// Path: web.memoryAmbient.profiles.row
class TranslationsWebMemoryAmbientProfilesRowEn {
	TranslationsWebMemoryAmbientProfilesRowEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'global default'
	String get globalDefault => 'global default';

	/// en: 'Delete'
	String get delete => 'Delete';

	/// en: 'Delete this injection profile?'
	String get deleteConfirm => 'Delete this injection profile?';

	/// en: 'Profile deleted'
	String get deletedToast => 'Profile deleted';

	/// en: 'Delete failed'
	String get deleteFailedToast => 'Delete failed';
}

// Path: web.memoryAmbient.profiles.dialog
class TranslationsWebMemoryAmbientProfilesDialogEn {
	TranslationsWebMemoryAmbientProfilesDialogEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Add injection profile'
	String get title => 'Add injection profile';

	/// en: 'Strategy'
	String get strategyLabel => 'Strategy';

	/// en: 'K (top memories to inject)'
	String get kLabel => 'K (top memories to inject)';

	/// en: 'One profile per session_id (or global default). Per-session profiles can be added later via API; UI currently only manages the global default.'
	String get hint => 'One profile per session_id (or global default). Per-session profiles can be added later via API; UI currently only manages the global default.';

	/// en: 'Create'
	String get create => 'Create';

	/// en: 'Profile created'
	String get createdToast => 'Profile created';

	/// en: 'Create failed'
	String get createFailedToast => 'Create failed';
}

// Path: web.memoryAmbient.cost.columns
class TranslationsWebMemoryAmbientCostColumnsEn {
	TranslationsWebMemoryAmbientCostColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Provider'
	String get provider => 'Provider';

	/// en: 'Calls'
	String get calls => 'Calls';

	/// en: 'In tokens'
	String get inTokens => 'In tokens';

	/// en: 'Out tokens'
	String get outTokens => 'Out tokens';

	/// en: 'USD est.'
	String get usdEst => 'USD est.';
}

// Path: web.export.form.integrationOptions
class TranslationsWebExportFormIntegrationOptionsEn {
	TranslationsWebExportFormIntegrationOptionsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'None'
	String get none => 'None';

	/// en: 'Skip the integrations table entirely.'
	String get noneHint => 'Skip the integrations table entirely.';

	/// en: 'Metadata only (recommended)'
	String get metadata => 'Metadata only (recommended)';

	/// en: 'ID, name, route prefix, scopes — no API key material.'
	String get metadataHint => 'ID, name, route prefix, scopes — no API key material.';

	/// en: 'Include plaintext API keys'
	String get plaintext => 'Include plaintext API keys';

	/// en: 'v1 bcrypt-only: no recoverable plaintext exists. Manifest documents this; nothing leaks.'
	String get plaintextHint => 'v1 bcrypt-only: no recoverable plaintext exists. Manifest documents this; nothing leaks.';
}

// Path: web.export.history.columns
class TranslationsWebExportHistoryColumnsEn {
	TranslationsWebExportHistoryColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'ID'
	String get id => 'ID';

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'Scope'
	String get scope => 'Scope';

	/// en: 'Size'
	String get size => 'Size';

	/// en: 'Expires'
	String get expires => 'Expires';

	/// en: 'Actions'
	String get actions => 'Actions';
}

// Path: web.export.import.summaryCard
class TranslationsWebExportImportSummaryCardEn {
	TranslationsWebExportImportSummaryCardEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'Memories'
	String get memories => 'Memories';

	/// en: 'Integrations'
	String get integrations => 'Integrations';

	/// en: 'Custom tasks'
	String get customTasks => 'Custom tasks';

	/// en: 'created'
	String get created => 'created';

	/// en: 'skipped'
	String get skipped => 'skipped';

	/// en: 'failed'
	String get failed => 'failed';
}

// Path: web.export.imports.columns
class TranslationsWebExportImportsColumnsEn {
	TranslationsWebExportImportsColumnsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'ID'
	String get id => 'ID';

	/// en: 'Status'
	String get status => 'Status';

	/// en: 'Source'
	String get source => 'Source';

	/// en: 'Counts'
	String get counts => 'Counts';

	/// en: 'When'
	String get when => 'When';
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

// Path: web.notes.vaultSync.conflict.kinds
class TranslationsWebNotesVaultSyncConflictKindsEn {
	TranslationsWebNotesVaultSyncConflictKindsEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'rebase'
	String get rebase => 'rebase';

	/// en: 'merge'
	String get merge => 'merge';

	/// en: 'cherry-pick'
	String get cherryPick => 'cherry-pick';

	/// en: 'operation'
	String get operation => 'operation';
}

// Path: web.memoryAmbient.rules.row.summary
class TranslationsWebMemoryAmbientRulesRowSummaryEn {
	TranslationsWebMemoryAmbientRulesRowSummaryEn.internal(this._root);

	final Translations _root; // ignore: unused_field

	// Translations

	/// en: 'every {n} messages'
	String afterMessages({required Object n}) => 'every ${n} messages';

	/// en: 'idle ≥ {seconds}s'
	String onIdle({required Object seconds}) => 'idle ≥ ${seconds}s';

	/// en: '≥ {k} chars'
	String kChars({required Object k}) => '≥ ${k} chars';

	/// en: 'manual only'
	String get manual => 'manual only';
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
			'auth.signingIn' => 'Signing in…',
			'auth.subtitle' => 'Use your operator credentials.',
			'auth.errorRequired' => 'Username and password are required',
			'auth.errorGeneric' => ({required Object error}) => 'Login failed: ${error}',
			'auth.errorFallback' => 'Login failed',
			'nav.sessions' => 'Sessions',
			'nav.memory' => 'Memory',
			'nav.notes' => 'Notes',
			'nav.more' => 'More',
			'nav.activity' => 'Activity',
			'nav.providers' => 'Providers',
			'nav.channels' => 'Channels',
			'nav.integrations' => 'Integrations',
			'nav.plugins' => 'Plugins',
			'nav.backups' => 'Backups',
			'nav.settings' => 'Settings',
			'nav.tutorial' => 'Tutorial',
			'nav.workspace' => 'Workspace',
			'web.brand' => 'opendray',
			'web.loading' => 'Loading…',
			'web.topbar.expandSidebar' => 'Expand sidebar',
			'web.topbar.collapseSidebar' => 'Collapse sidebar',
			'web.topbar.search' => 'Search',
			'web.topbar.openPalette' => 'Open command palette',
			'web.topbar.theme' => 'Theme',
			'web.topbar.themeLabel' => ({required Object mode}) => 'Theme: ${mode}',
			'web.topbar.appearance' => 'Appearance',
			'web.topbar.themeLight' => 'Light',
			'web.topbar.themeDark' => 'Dark',
			'web.topbar.themeSystem' => 'System',
			'web.topbar.language' => 'Language',
			'web.topbar.languageEnglish' => 'English',
			'web.topbar.languageChinese' => '中文',
			'web.topbar.signedInAs' => 'Signed in as',
			'web.topbar.tokenExpires' => 'Token expires',
			'web.topbar.signOut' => 'Sign out',
			'web.sessions.list.title' => 'Sessions',
			'web.sessions.list.countSeparator' => '·',
			'web.sessions.list.newAria' => 'Spawn new session',
			'web.sessions.list.newTooltip' => 'New session',
			'web.sessions.list.loading' => 'Loading…',
			'web.sessions.list.emptyTitle' => 'No sessions yet.',
			'web.sessions.list.emptyHint' => ({required Object kbd}) => 'Press ${kbd} to spawn.',
			'web.sessions.list.endedHeader' => ({required Object count}) => 'Ended (${count})',
			'web.sessions.list.clearAll' => 'Clear all',
			'web.sessions.list.confirmClearAll' => ({required Object count}) => 'Remove all ${count} ended sessions?',
			'web.sessions.list.confirmTerminate' => ({required Object name}) => 'Terminate and remove ${name}?',
			'web.sessions.list.childPromoted' => ({required Object count}) => ' ${count} child task session will be promoted to top-level.',
			'web.sessions.list.childPromotedPlural' => ({required Object count}) => ' ${count} child task sessions will be promoted to top-level.',
			'web.sessions.list.footer' => ({required Object live, required Object ended}) => '${live} live · ${ended} ended',
			'web.sessions.list.row.deleteAria' => 'Delete session',
			'web.sessions.list.row.titleRemoveHistory' => 'Remove from history',
			'web.sessions.list.row.titleTerminate' => 'Terminate and remove',
			'web.sessions.list.row.titleRemove' => 'Remove',
			'web.sessions.list.row.claudeAccountTitle' => ({required Object label}) => 'Claude account: ${label}',
			'web.sessions.list.deleteFailedToast' => 'Delete failed',
			'web.sessions.tabs.closeAria' => 'Close tab and remove session',
			'web.sessions.tabs.closeTitle' => 'Close tab and remove session',
			'web.sessions.page.removedToast' => 'Session removed',
			'web.sessions.page.removeFailedToast' => 'Remove failed',
			'web.sessions.page.stoppedToast' => 'Session stopped',
			'web.sessions.page.stopFailedToast' => 'Stop failed',
			'web.sessions.page.restartedToast' => 'Session restarted',
			'web.sessions.page.restartFailedToast' => 'Restart failed',
			'web.sessions.page.confirmCloseTabTitle' => ({required Object name}) => 'Stop and remove "${name}"?',
			'web.sessions.page.confirmCloseTabDescription' => 'The CLI process will be terminated and the row deleted.',
			'web.sessions.page.confirmCloseTabConfirm' => 'Stop and remove',
			'web.sessions.page.confirmRemoveTitle' => ({required Object name}) => 'Remove ${name}?',
			'web.sessions.page.confirmRemoveTitleFallback' => 'Remove session?',
			'web.sessions.page.confirmRemoveDescription' => 'This deletes the row.',
			'web.sessions.page.confirmRemoveConfirm' => 'Remove',
			'web.sessions.empty.title' => 'No session open',
			'web.sessions.empty.hint' => ({required Object kbdN, required Object kbdW, required Object kbdRange}) => 'Pick a session from the list, or spawn a new one. Keyboard: ${kbdN} new, ${kbdW} close, ${kbdRange} switch.',
			'web.sessions.empty.spawn' => 'Spawn session',
			'web.sessions.header.loadingSession' => 'Loading session…',
			'web.sessions.header.showList' => 'Show session list',
			'web.sessions.header.hideList' => 'Hide session list',
			'web.sessions.header.showInspector' => 'Show inspector',
			'web.sessions.header.hideInspector' => 'Hide inspector',
			'web.sessions.header.attachImage' => 'Attach image',
			'web.sessions.header.attachImageTooltip' => 'Attach image (or paste / drop into terminal)',
			'web.sessions.header.restart' => 'Restart',
			'web.sessions.header.restarting' => 'Restarting…',
			'web.sessions.header.remove' => 'Remove',
			'web.sessions.header.removing' => 'Removing…',
			'web.sessions.header.stop' => 'Stop',
			'web.sessions.header.stopping' => 'Stopping…',
			'web.sessions.header.pid' => ({required Object pid}) => 'pid ${pid}',
			'web.sessions.terminal.uploadingToast' => 'Uploading image…',
			'web.sessions.terminal.uploadedToast' => 'Image attached',
			'web.sessions.terminal.uploadFailedToast' => 'Upload failed',
			'web.sessions.terminal.uploadInvalidTypeToast' => 'Only image files can be attached',
			'web.sessions.terminal.dropToAttach' => 'Drop image to attach',
			'web.sessions.spawn.title' => 'Spawn session',
			'web.sessions.spawn.description' => 'Start a CLI session under a registered provider.',
			'web.sessions.spawn.provider' => 'Provider',
			'web.sessions.spawn.claudeAccount' => 'Claude account',
			'web.sessions.spawn.loadingAccounts' => 'Loading accounts…',
			'web.sessions.spawn.noAccounts' => 'No Claude accounts configured — the gateway will use the system ANTHROPIC_API_KEY.',
			'web.sessions.spawn.kDefault' => 'Default',
			'web.sessions.spawn.defaultTooltip' => 'Use system keychain / env',
			'web.sessions.spawn.tokenEmptyBadge' => '·empty',
			'web.sessions.spawn.tokenMissingTooltip' => 'No token set — set token in Providers panel first',
			'web.sessions.spawn.multiAccountHint' => 'Multiple accounts configured — pick one for this session.',
			'web.sessions.spawn.cwd' => 'Working directory',
			'web.sessions.spawn.cwdPlaceholder' => '/Users/you/projects/foo',
			'web.sessions.spawn.browse' => 'Browse',
			'web.sessions.spawn.nameLabel' => 'Name (optional)',
			'web.sessions.spawn.namePlaceholder' => 'claude in pet-tracker',
			'web.sessions.spawn.argsLabel' => 'CLI args (one per line)',
			'web.sessions.spawn.bypassClaude' => 'Bypass permission prompts',
			'web.sessions.spawn.bypassCodex' => 'Bypass approvals & sandbox (--dangerously-bypass-approvals-and-sandbox)',
			'web.sessions.spawn.bypassGemini' => 'YOLO mode (--yolo)',
			'web.sessions.spawn.bypassOnHint' => 'This session will run with elevated autonomy.',
			'web.sessions.spawn.bypassOffHint' => 'Off — confirmations and prompts behave normally.',
			'web.sessions.spawn.errorPickProvider' => 'Pick a provider.',
			'web.sessions.spawn.errorCwdRequired' => 'cwd is required.',
			'web.sessions.spawn.cancel' => 'Cancel',
			'web.sessions.spawn.submit' => 'Spawn',
			'web.sessions.spawn.submitting' => 'Spawning…',
			'web.sessions.spawn.spawnedToast' => 'Session spawned',
			'web.sessions.spawn.spawnedDescription' => ({required Object provider, required Object pid}) => '${provider} · pid ${pid}',
			'web.sessions.spawn.pidFallback' => '—',
			'web.sessions.accountSwitcher.tooltip' => 'Switch Claude account (restarts the CLI process)',
			'web.sessions.accountSwitcher.currentDefault' => 'default',
			'web.sessions.accountSwitcher.menuTitle' => 'Switch Claude account',
			'web.sessions.accountSwitcher.defaultName' => 'Default',
			'web.sessions.accountSwitcher.defaultSubtitle' => 'CLI\'s system keychain / env',
			'web.sessions.accountSwitcher.tokenEmpty' => '·empty',
			'web.sessions.accountSwitcher.confirmSwitch' => 'Switching account will restart the Claude CLI process. In-progress conversation state inside the CLI will be lost. Continue?',
			'web.sessions.accountSwitcher.switchedToast' => 'Account switched',
			'web.sessions.accountSwitcher.switchedDescription' => ({required Object account, required Object pid}) => 'Now using @${account} · pid ${pid}',
			'web.sessions.accountSwitcher.switchedDefault' => 'default',
			'web.sessions.accountSwitcher.switchFailedToast' => 'Switch failed',
			'web.sessions.inspector.tabs.files' => 'Files',
			'web.sessions.inspector.tabs.git' => 'Git',
			'web.sessions.inspector.tabs.search' => 'Search',
			'web.sessions.inspector.tabs.tasks' => 'Tasks',
			'web.sessions.inspector.tabs.history' => 'History',
			'web.sessions.inspector.tabs.notes' => 'Notes',
			'web.sessions.inspector.tabs.memory' => 'Memory',
			'web.sessions.ended.bufferUnavailable' => '[buffer unavailable]',
			'web.sessions.ended.readOnlyBanner' => '[session ended — read-only buffer]',
			'web.sessions.fileBrowser.title' => 'Choose working directory',
			'web.sessions.fileBrowser.description' => 'Browse the gateway host\'s filesystem and pick a folder.',
			'web.sessions.fileBrowser.parent' => 'Parent directory',
			'web.sessions.fileBrowser.home' => 'Home directory',
			'web.sessions.fileBrowser.refresh' => 'Refresh',
			'web.sessions.fileBrowser.pathPlaceholder' => '/Users/you/projects',
			'web.sessions.fileBrowser.loading' => 'Loading…',
			'web.sessions.fileBrowser.empty' => 'Empty directory.',
			'web.sessions.fileBrowser.newFolder' => 'New folder',
			'web.sessions.fileBrowser.newFolderPlaceholder' => 'new-folder-name',
			'web.sessions.fileBrowser.create' => 'Create',
			'web.sessions.fileBrowser.cancel' => 'Cancel',
			'web.sessions.fileBrowser.useThisFolder' => 'Use this folder',
			'web.sessions.fileBrowser.createdToast' => 'Directory created',
			'web.sessions.fileBrowser.mkdirFailedToast' => 'Mkdir failed',
			'web.sessions.fileBrowser.homeFailedToast' => 'Failed to read home',
			'web.memory.title' => 'Memory',
			'web.memory.subtitle' => 'Browse, search and edit memories agents have stored via the opendray-memory MCP server.',
			'web.memory.navProject' => 'Project',
			'web.memory.navCleanupInbox' => 'Cleanup inbox',
			'web.memory.navWorkers' => 'Workers',
			'web.memory.navConfiguration' => 'Configuration →',
			'web.journalStale.title' => 'Prune stale entries',
			'web.journalStale.subtitle' => ({required Object days}) => '(older than ${days} days, no pending conflicts)',
			'web.journalStale.daysLabel' => 'Older than (days):',
			'web.journalStale.loading' => 'Scanning…',
			'web.journalStale.empty' => 'Nothing stale to prune.',
			'web.journalStale.selectAll' => 'Select all',
			'web.journalStale.deselectAll' => 'Deselect all',
			'web.journalStale.deleteSelected' => ({required Object count}) => 'Delete (${count})',
			'web.journalStale.deleted_one' => ({required Object count}) => '${count} entry deleted',
			'web.journalStale.deleted_other' => ({required Object count}) => '${count} entries deleted',
			'web.conflicts.title' => 'Cross-layer conflicts',
			'web.conflicts.subtitle' => 'Contradictions the daily detector found between facts, plan, goal, and journal entries.',
			'web.conflicts.loading' => 'Loading conflicts…',
			'web.conflicts.empty' => 'No pending conflicts. Click "Detect now" to run an on-demand sweep.',
			'web.conflicts.pickCwd' => 'Pick a project to see its conflicts.',
			'web.conflicts.detectNow' => 'Detect now',
			'web.conflicts.detected' => ({required Object count}) => '${count} new conflict(s) found',
			'web.conflicts.accept' => 'Accept',
			'web.conflicts.dismiss' => 'Dismiss',
			'web.conflicts.accepted' => 'Conflict accepted — remember to apply the fix',
			'web.conflicts.dismissed' => 'Conflict dismissed',
			'web.conflicts.deletedFact' => 'Fact deleted and conflict accepted',
			'web.conflicts.quickActions' => 'Fix:',
			'web.conflicts.deleteFact' => 'Delete fact',
			'web.conflicts.deleteFactSide' => ({required Object side, required Object ref}) => 'Delete ${side}: ${ref}',
			'web.conflicts.confirmDelete.title' => ({required Object side}) => 'Delete fact ${side}?',
			'web.conflicts.confirmDelete.description' => 'This permanently removes the fact and accepts the conflict. The other side stays as the surviving claim.',
			'web.conflicts.confirmDelete.targetLabel' => ({required Object side}) => 'Will delete (side ${side}):',
			'web.conflicts.confirmDelete.keepLabel' => ({required Object side}) => 'Will keep (side ${side}):',
			'web.conflicts.confirmDelete.nonFactOther' => ({required Object layer}) => '(${layer} entry — open the corresponding tab to inspect)',
			'web.conflicts.confirmDelete.evidenceLabel' => 'Detector evidence:',
			'web.conflicts.confirmDelete.loading' => 'Loading fact text…',
			'web.conflicts.confirmDelete.loadError' => 'Failed to load fact text. Inspect on the Memory page.',
			'web.conflicts.confirmDelete.cancel' => 'Cancel',
			'web.conflicts.confirmDelete.confirm' => ({required Object side}) => 'Delete ${side}',
			'web.conflicts.openLayer.plan' => 'Open plan editor',
			'web.conflicts.openLayer.goal' => 'Open goal editor',
			'web.conflicts.severity.low' => 'low',
			'web.conflicts.severity.medium' => 'medium',
			'web.conflicts.severity.high' => 'high',
			'web.memoryHealth.title' => ({required Object days}) => 'Memory health — last ${days} days',
			'web.memoryHealth.subtitle' => 'Aggregate signals across both memory subsystems for this project.',
			'web.memoryHealth.loading' => 'Loading health snapshot…',
			'web.memoryHealth.errorLoading' => 'Failed to load health snapshot.',
			'web.memoryHealth.pickCwd' => 'Pick a project to see its memory health.',
			'web.memoryHealth.newFacts' => 'New facts',
			'web.memoryHealth.newFactsHint' => ({required Object total}) => '${total} stored in total',
			'web.memoryHealth.captureFires' => 'Capture fires',
			'web.memoryHealth.captureFiresHint' => ({required Object stored, required Object deduped}) => '${stored} stored · ${deduped} deduped',
			'web.memoryHealth.newJournal' => 'Journal entries',
			'web.memoryHealth.newJournalHint' => ({required Object total}) => '${total} in total',
			'web.memoryHealth.planAge' => 'Plan last updated',
			'web.memoryHealth.planAgeHint' => ({required Object count}) => '${count} plan-drift proposal(s) pending',
			'web.memoryHealth.planAgeHintNone' => 'No plan-drift proposals pending',
			'web.memoryHealth.goalAge' => 'Goal last updated',
			'web.memoryHealth.pending' => 'Pending proposals',
			'web.memoryHealth.pendingHint' => ({required Object days}) => 'oldest ${days}d old',
			'web.memoryHealth.topHit' => ({required Object hits}) => 'Top hit · ${hits} retrievals',
			'web.memoryHealth.zeroHit' => ({required Object count}) => '${count} facts older than 7d with zero retrievals — candidates for cleanup.',
			'web.memoryHealth.never' => 'never',
			'web.memoryHealth.today' => 'today',
			'web.memoryHealth.daysAgo_one' => ({required Object count}) => '${count} day ago',
			'web.memoryHealth.daysAgo_other' => ({required Object count}) => '${count} days ago',
			'web.memoryConfig.title' => 'Memory configuration',
			'web.memoryConfig.subtitle' => 'All memory-related knobs in one place: HTTP providers, per-task worker routing, capture triggers, spawn-time injection, and audit costs.',
			'web.memoryConfig.sections.providers' => 'Providers',
			'web.memoryConfig.sections.workers' => 'Workers',
			'web.memoryConfig.sections.rules' => 'Capture rules',
			'web.memoryConfig.sections.profiles' => 'Injection profiles',
			'web.memoryConfig.sections.costs' => 'Token cost',
			'web.memoryConfig.sectionHints.providers' => 'Registered HTTP endpoints (Ollama / LM Studio / Anthropic / OpenAI / Integration) that any task can dispatch to.',
			'web.memoryConfig.sectionHints.workers' => 'For each touchpoint pick HTTP provider (cheap, local) or headless Claude / Gemini Agent (higher quality, costs CLI tokens).',
			'web.memoryConfig.sectionHints.rules' => 'When the capture engine fires per session (after N messages / on idle / K characters / manual). Rules without a pinned provider follow the Capture worker setting above.',
			'web.memoryConfig.sectionHints.profiles' => 'How prior memories get injected into the agent\'s system prompt at session spawn (recency, relevance, hybrid, or off).',
			'web.memoryConfig.sectionHints.costs' => 'Aggregate spend reconstructed from memory_summarizer_calls. Local providers (Ollama, LM Studio, Integration) are free; cloud providers show real-world cost.',
			'web.memoryConfig.moveBanner.title' => 'Memory configuration has moved',
			'web.memoryConfig.moveBanner.body' => 'All memory-related settings (providers / capture rules / injection profiles / cost) now live alongside Workers in one page so related knobs sit together.',
			'web.memoryConfig.moveBanner.openButton' => 'Open Memory configuration →',
			'web.memoryWorkers.title' => 'Memory workers',
			'web.memoryWorkers.loading' => 'Loading worker config…',
			'web.memoryWorkers.errorTitle' => 'Endpoint not reachable.',
			'web.memoryWorkers.errorDescription' => 'The /api/v1/memory/workers routes are new in M25 — the opendray binary may need a restart to mount them and run migration 0029.',
			'web.memoryWorkers.intro' => 'Each memory-system LLM touchpoint can be served independently by the local <1>summarizer</1> endpoint (LM Studio / OpenAI-compat) or by spawning a headless <3>Claude / Gemini agent</3> in <5>--print</5> mode. High-quality narrative tasks (gitactivity, transcript) benefit from agent workers; high-frequency tasks (gatekeeper) stay on the local endpoint by design.',
			'web.memoryWorkers.enabledBadge' => 'enabled',
			'web.memoryWorkers.disabledBadge' => 'disabled',
			'web.memoryWorkers.summarizerOnlyBadge' => 'summarizer-only',
			'web.memoryWorkers.callsCount' => ({required Object count}) => '${count} calls · 24h',
			'web.memoryWorkers.avgMs' => ({required Object ms}) => 'avg ${ms}ms',
			'web.memoryWorkers.errorsCount' => ({required Object count}) => '${count} errors',
			'web.memoryWorkers.workerLabel' => 'Worker',
			'web.memoryWorkers.summarizerHttp' => 'Summarizer (HTTP)',
			'web.memoryWorkers.agentCliPrint' => 'Agent (CLI --print)',
			'web.memoryWorkers.summarizerProviderLabel' => 'Summarizer provider',
			'web.memoryWorkers.registryDefault' => 'Registry default',
			'web.memoryWorkers.cliLabel' => 'CLI',
			'web.memoryWorkers.selectPlaceholder' => 'Select',
			'web.memoryWorkers.cliClaude' => 'Claude',
			'web.memoryWorkers.cliGemini' => 'Gemini',
			'web.memoryWorkers.claudeAccountLabel' => 'Claude account',
			'web.memoryWorkers.claudeAccountDefault' => 'Default',
			'web.memoryWorkers.agentWarning' => 'Agent mode spawns a headless CLI per call. Latency rises from <1>~1s</1> (summarizer) to <3>~5-15s</3>; cost shifts from CPU to your Claude/Gemini quota.',
			'web.memoryWorkers.enabledCheckbox' => 'Enabled',
			'web.memoryWorkers.testButton' => 'Test',
			'web.memoryWorkers.saveButton' => 'Save',
			'web.memoryWorkers.recentCalls' => ({required Object count}) => 'Recent calls (${count})',
			'web.memoryWorkers.tableWhen' => 'when',
			'web.memoryWorkers.tableWorker' => 'worker',
			'web.memoryWorkers.tableMs' => 'ms',
			'web.memoryWorkers.tableOk' => 'ok',
			'web.memoryWorkers.savedToast' => ({required Object label}) => '${label} updated',
			'web.memoryWorkers.saveFailedToast' => 'Save failed',
			'web.memoryWorkers.testOkToast' => ({required Object label, required Object ms}) => '${label} OK — ${ms}ms',
			'web.memoryWorkers.testFailedToast' => ({required Object label}) => '${label} failed',
			'web.memoryWorkers.testCallFailedToast' => 'Test call failed',
			'web.memoryWorkers.unknownError' => 'unknown error',
			'web.memoryWorkers.tasks.gatekeeper.label' => 'Gatekeeper',
			'web.memoryWorkers.tasks.gatekeeper.description' => 'Pre-write filter on every memory_store. High frequency (<500ms target) — summarizer-only.',
			'web.memoryWorkers.tasks.cleaner.label' => 'Cleaner librarian',
			'web.memoryWorkers.tasks.cleaner.description' => 'Periodic LLM librarian. Judges aged memories as keep / stale / duplicate.',
			'web.memoryWorkers.tasks.gitactivity.label' => 'Git activity summariser',
			'web.memoryWorkers.tasks.gitactivity.description' => 'git log → 2-3 paragraph narrative every 24h. Naturally fits an agent worker.',
			'web.memoryWorkers.tasks.transcript.label' => 'Session transcript summariser',
			'web.memoryWorkers.tasks.transcript.description' => 'Session-end "what did the agent do" summary. Naturally fits an agent worker.',
			'web.memoryWorkers.tasks.plan_drift.label' => 'Plan drift detector',
			'web.memoryWorkers.tasks.plan_drift.description' => 'After each session ends, checks whether the project plan needs updating and files a proposal. Fits an agent worker for richer reasoning.',
			'web.memoryWorkers.tasks.conflict_detector.label' => 'Cross-layer conflict detector',
			'web.memoryWorkers.tasks.conflict_detector.description' => 'Daily scan that finds contradictions between facts / plan / goal / journal. Higher-quality model = fewer false positives.',
			'web.memoryWorkers.tasks.capture.label' => 'Capture engine',
			'web.memoryWorkers.tasks.capture.description' => 'Per-trigger fact extraction from session transcripts. Agent mode gives noticeably better facts on long sessions; summarizer mode is cheap and local.',
			'web.cleanupInbox.loading' => 'Loading…',
			'web.cleanupInbox.emptyTitle' => 'Cleanup inbox empty',
			'web.cleanupInbox.emptyDescription' => 'No pending cleanup decisions across any project. The LLM librarian either hasn\'t run yet for the eligible memories, or it found everything load-bearing.',
			'web.cleanupInbox.title' => 'Cleanup inbox',
			'web.cleanupInbox.subtitle' => 'Cross-project pending decisions from the LLM memory librarian. Approving stale → deletes, approving duplicate → merges, approving keep → freezes the entry from being re-judged for a while.',
			'web.cleanupInbox.globalScope' => '(global)',
			'web.cleanupInbox.orphanBadge' => 'orphan',
			'web.cleanupInbox.orphanTitle' => 'Truncated scope_key (old mirror import). Not a navigable project.',
			'web.cleanupInbox.openProject' => 'Open project',
			'web.cleanupInbox.mergeIntoPrefix' => '→ merge into',
			'web.cleanupInbox.reasonPrefix' => 'Reason:',
			'web.cleanupInbox.executeButton' => 'Execute',
			'web.cleanupInbox.confirmKeepButton' => 'Confirm keep',
			'web.cleanupInbox.rejectButton' => 'Reject',
			'web.cleanupInbox.approvedKeptToast' => 'Kept',
			'web.cleanupInbox.approvedExecutedToast' => ({required Object verdict}) => '${verdict} executed',
			'web.cleanupInbox.approveFailedToast' => 'Approve failed',
			'web.cleanupInbox.rejectedToast' => 'Rejected — memory kept',
			'web.cleanupInbox.rejectFailedToast' => 'Reject failed',
			'web.project.picker.title' => 'Pick a project',
			'web.project.picker.subtitle' => 'Project memory is scoped by working directory. Pick one to manage its goal, plan, journal, and cleanup queue.',
			'web.project.picker.pathPlaceholder' => '/path/to/your/project',
			'web.project.picker.browse' => 'Browse',
			'web.project.picker.browseTooltip' => 'Browse the gateway host\'s filesystem',
			'web.project.picker.open' => 'Open',
			'web.project.picker.recentLabel' => 'Recent projects (from stored memory):',
			'web.project.picker.orphanTooltip' => 'Looks like a truncated scope_key (old mirror import bug). May have no project docs.',
			'web.project.picker.orphanBadge' => 'orphan',
			'web.project.noCwd' => 'Pick a project to manage its memory.',
			'web.project.header.docsCount_one' => ({required Object count}) => '${count} doc',
			'web.project.header.docsCount_other' => ({required Object count}) => '${count} docs',
			'web.project.header.journalEntries_one' => ({required Object count}) => '${count} journal entry',
			'web.project.header.journalEntries_other' => ({required Object count}) => '${count} journal entries',
			'web.project.header.pendingProposals_one' => ({required Object count}) => '${count} pending proposal',
			'web.project.header.pendingProposals_other' => ({required Object count}) => '${count} pending proposals',
			'web.project.header.cleanupPending' => ({required Object count}) => '${count} cleanup pending',
			'web.project.tabs.health' => 'Health',
			'web.project.tabs.goal' => 'Goal',
			'web.project.tabs.plan' => 'Plan',
			'web.project.tabs.tech' => 'Tech',
			'web.project.tabs.activity' => 'Activity',
			'web.project.tabs.journal' => 'Journal',
			'web.project.tabs.inbox' => 'Inbox',
			'web.project.tabs.conflicts' => 'Conflicts',
			'web.project.tabs.cleanup' => 'Cleanup',
			'web.project.docLabel.goal' => 'Goal',
			'web.project.docLabel.plan' => 'Plan',
			'web.project.docLabel.tech_stack' => 'Tech stack',
			'web.project.docLabel.recent_activity' => 'Recent activity',
			'web.project.verdictLabel.stale' => 'Delete',
			'web.project.verdictLabel.duplicate' => 'Merge',
			'web.project.verdictLabel.keep' => 'Keep',
			'web.project.editor.updatedBy' => 'Updated by',
			'web.project.editor.noDocSet' => ({required Object label}) => 'No ${label} set yet.',
			'web.project.editor.save' => 'Save',
			'web.project.editor.saveFailedToast' => 'Save failed',
			'web.project.editor.savedToast' => ({required Object label}) => '${label} saved',
			'web.project.editor.goalPlaceholder' => 'What are we building? One paragraph. Read by every agent on spawn.',
			'web.project.editor.planPlaceholder' => 'Active plan — what we are doing right now and what is next. Updated as work progresses.',
			'web.project.readonly.tech_stack.label' => 'Tech stack & structure',
			'web.project.readonly.tech_stack.empty' => 'Run a Claude session in this project — scanner refreshes on every spawn.',
			'web.project.readonly.recent_activity.label' => 'Recent activity (git → LLM)',
			'web.project.readonly.recent_activity.empty' => 'The git activity summariser runs every 24h; check back after the next scheduler tick.',
			'web.project.readonly.noneCaptured' => ({required Object label}) => 'No ${label} captured yet.',
			'web.project.readonly.generatedBy' => 'Generated by',
			'web.project.readonly.lastRefresh' => 'last refresh',
			'web.project.journal.loading' => 'Loading…',
			'web.project.journal.empty' => 'No journal entries yet. Each session-end appends one automatically.',
			'web.project.inbox.loading' => 'Loading…',
			'web.project.inbox.emptyTitle' => 'Inbox empty.',
			'web.project.inbox.emptyHint' => 'Agents file proposals here via `project_goal_set` / `project_plan_set` MCP tools.',
			'web.project.inbox.approvedToast' => ({required Object label}) => '${label} updated',
			'web.project.inbox.approveFailedToast' => 'Approve failed',
			'web.project.inbox.rejectedToast' => 'Rejected',
			'web.project.inbox.rejectFailedToast' => 'Reject failed',
			'web.project.inbox.sessionPrefix' => 'ses',
			'web.project.inbox.warning' => ({required Object label}) => 'Approve will REPLACE the current ${label} entirely.',
			'web.project.inbox.warningSuffix' => 'Review the diff below; this isn\'t a merge.',
			'web.project.inbox.current' => 'Current',
			'web.project.inbox.proposed' => 'Proposed',
			'web.project.inbox.emptyBody' => '(empty)',
			'web.project.inbox.approve' => 'Approve',
			'web.project.inbox.reject' => 'Reject',
			'web.project.inbox.confirmDialogTitle' => ({required Object label}) => 'Replace ${label}?',
			'web.project.inbox.confirmDialogDescription' => ({required Object label}) => 'The current ${label} will be overwritten with the proposed content. This cannot be undone via this UI (you can manually edit it back).',
			'web.project.inbox.confirmCancel' => 'Cancel',
			'web.project.inbox.confirmReplace' => 'Confirm replace',
			'web.project.cleanup.hint' => 'The LLM librarian proposes keep / stale / duplicate verdicts for this project\'s memories. You approve before anything is deleted.',
			'web.project.cleanup.runNow' => 'Run cleanup now',
			'web.project.cleanup.runSucceededToast' => ({required Object decided, required Object scanned}) => 'Cleanup run: ${decided} decisions queued (${scanned} scanned)',
			'web.project.cleanup.runFailedToast' => 'Cleanup run failed',
			'web.project.cleanup.empty' => 'No pending decisions. Either nothing aged into eligibility or the last run found everything load-bearing.',
			'web.project.cleanup.mergeIntoPrefix' => '→ merge into',
			'web.project.cleanup.reasonPrefix' => 'Reason:',
			'web.project.cleanup.executeButton' => 'Execute',
			'web.project.cleanup.confirmKeepButton' => 'Confirm keep',
			'web.project.cleanup.rejectButton' => 'Reject',
			'web.project.cleanup.approvedExecutedToast' => ({required Object label}) => '${label} executed',
			'web.project.cleanup.approveFailedToast' => 'Approve failed',
			'web.project.cleanup.rejectedToast' => 'Rejected — memory kept',
			'web.project.cleanup.rejectFailedToast' => 'Reject failed',
			'web.project.reset.button' => 'Reset',
			'web.project.reset.dialogTitle' => 'Reset project memory?',
			'web.project.reset.dialogDescription' => 'Deletes all stored project context for this cwd. This cannot be undone.',
			'web.project.reset.alwaysDeleted' => 'Always deleted: goal, plan, proposals, journal, cleanup decisions.',
			'web.project.reset.alsoDeleteScannerLabel' => 'Also delete scanner docs',
			'web.project.reset.alsoDeleteScannerSuffix' => '(tech_stack + recent_activity).',
			'web.project.reset.alsoDeleteScannerHint' => 'Auto-rebuild on next spawn anyway — leaving unchecked is usually fine.',
			'web.project.reset.alsoDeleteMemoriesLabel' => 'Also delete pgvector memories',
			'web.project.reset.alsoDeleteMemoriesSuffix' => 'for this scope_key.',
			'web.project.reset.alsoDeleteMemoriesHint' => 'Long-term facts the agent stored (user preferences, project facts). Cannot be recovered.',
			'web.project.reset.cancel' => 'Cancel',
			'web.project.reset.deleteForever' => 'Delete forever',
			'web.project.reset.successToast' => ({required Object summary}) => 'Reset: deleted ${summary}',
			'web.project.reset.summary.docs_one' => ({required Object count}) => '${count} doc',
			'web.project.reset.summary.docs_other' => ({required Object count}) => '${count} docs',
			'web.project.reset.summary.journal' => ({required Object count}) => '${count} journal',
			'web.project.reset.summary.proposals_one' => ({required Object count}) => '${count} proposal',
			'web.project.reset.summary.proposals_other' => ({required Object count}) => '${count} proposals',
			'web.project.reset.summary.cleanup' => ({required Object count}) => '${count} cleanup',
			'web.project.reset.summary.memories' => ({required Object count}) => '${count} memories',
			'web.project.reset.failedToast' => 'Reset failed',
			'web.memoryInspector.status.label' => 'Active embedder',
			'web.memoryInspector.status.unavailable' => 'unavailable',
			'web.memoryInspector.status.probing' => 'probing…',
			'web.memoryInspector.status.dimensions' => ({required Object dim, required Object state}) => '${dim}-dim · ${state}',
			'web.memoryInspector.status.enabled' => 'enabled',
			'web.memoryInspector.status.disabled' => 'disabled',
			'web.memoryInspector.status.testButton' => 'Test embedder',
			'web.memoryInspector.statusBody' => 'This is the embedder the gateway is currently using for every <1>memory_search</1> / <3>memory_store</3> call. If this doesn\'t match the configuration above, you have unsaved changes — click Save then Restart server to apply.',
			'web.memoryInspector.scope.label' => 'Scope',
			'web.memoryInspector.scope.scopeKey' => 'Scope key',
			'web.memoryInspector.scope.scopeKeyIgnored' => '(ignored for global)',
			'web.memoryInspector.scope.scopeKeyCwd' => '(cwd of the project)',
			'web.memoryInspector.scope.scopeKeySession' => '(session id)',
			'web.memoryInspector.scope.placeholderProject' => '/path/to/project (cwd)',
			'web.memoryInspector.scope.placeholderSession' => 'session id',
			'web.memoryInspector.scope.syncMd' => 'Sync .md',
			'web.memoryInspector.scope.syncTooltip' => 'Re-ingest Claude\'s <cwd>/.claude/memory/*.md files into pgvector',
			'web.memoryInspector.scope.values.project' => 'project',
			'web.memoryInspector.scope.values.session' => 'session',
			'web.memoryInspector.scope.values.global' => 'global',
			'web.memoryInspector.search.placeholder' => 'Semantic search query (Enter to run; empty = browse)',
			'web.memoryInspector.search.run' => 'Search',
			'web.memoryInspector.search.clear' => 'Clear',
			'web.memoryInspector.search.failedToast' => 'Search failed',
			'web.memoryInspector.records.noMemories' => 'No memories yet',
			'web.memoryInspector.records.matches_one' => ({required Object count}) => '${count} match',
			'web.memoryInspector.records.matches_other' => ({required Object count}) => '${count} matches',
			'web.memoryInspector.records.memories_one' => ({required Object count}) => '${count} memory',
			'web.memoryInspector.records.memories_other' => ({required Object count}) => '${count} memories',
			'web.memoryInspector.records.scopeGlobalSuffix' => ' (global)',
			'web.memoryInspector.records.scopeInSuffix' => ({required Object scope}) => ' in ${scope}: ',
			'web.memoryInspector.records.addButton' => 'Add memory',
			'web.memoryInspector.records.addTooltip' => 'Manually create a memory in this scope',
			'web.memoryInspector.records.deleteAll' => 'Delete all',
			'web.memoryInspector.records.deleteAllTooltip' => 'Delete every memory under this scope/scope_key',
			'web.memoryInspector.records.loading' => 'Loading…',
			'web.memoryInspector.records.enterScopeKeyHint' => 'Enter a scope key to browse memories.',
			'web.memoryInspector.records.noMatchesForQuery' => ({required Object query}) => 'No matches for "${query}"',
			'web.memoryInspector.records.noMemoriesInScope' => 'No memories in this scope yet.',
			'web.memoryInspector.row.simBadge' => ({required Object value}) => 'sim ${value}',
			'web.memoryInspector.row.rankBadge' => ({required Object value}) => 'rank ${value}',
			'web.memoryInspector.row.rankTooltip' => ({required Object effective, required Object similarity, required Object age, required Object days, required Object hits, required Object confidence}) => 'effective ${effective} = sim ${similarity} × age ${age} (${days}d) × hits ${hits} × conf ${confidence}',
			'web.memoryInspector.row.hits_one' => ({required Object count}) => '${count} hit',
			'web.memoryInspector.row.hits_other' => ({required Object count}) => '${count} hits',
			'web.memoryInspector.row.lastHitTooltip' => ({required Object relative}) => 'Last hit ${relative}',
			'web.memoryInspector.row.editPlaceholder' => 'Memory text — Cmd/Ctrl+Enter to save · Esc to cancel',
			'web.memoryInspector.row.saveTooltip' => 'Save (Cmd/Ctrl+Enter)',
			'web.memoryInspector.row.cancelTooltip' => 'Cancel (Esc)',
			'web.memoryInspector.row.editTooltip' => 'Edit this memory',
			'web.memoryInspector.row.deleteTooltip' => 'Delete this memory',
			'web.memoryInspector.row.emptyError' => 'Memory text cannot be empty',
			'web.memoryInspector.row.deleteConfirm' => ({required Object id}) => 'Delete memory ${id}? This is permanent.',
			'web.memoryInspector.toasts.deleted' => 'Memory deleted',
			'web.memoryInspector.toasts.deleteFailed' => 'Delete failed',
			'web.memoryInspector.toasts.bulkDeleted_one' => ({required Object count}) => 'Deleted ${count} memory from this scope',
			'web.memoryInspector.toasts.bulkDeleted_other' => ({required Object count}) => 'Deleted ${count} memories from this scope',
			'web.memoryInspector.toasts.bulkDeleteFailed' => 'Bulk delete failed',
			'web.memoryInspector.toasts.created' => 'Memory created',
			'web.memoryInspector.toasts.createFailed' => 'Create failed',
			'web.memoryInspector.toasts.updated' => 'Memory updated',
			'web.memoryInspector.toasts.updateFailed' => 'Update failed',
			'web.memoryInspector.toasts.migrated' => ({required Object reembed, required Object examined, required Object to}) => 'Migrated ${reembed}/${examined} memories to ${to}',
			'web.memoryInspector.toasts.migrationFailed' => 'Migration failed',
			'web.memoryInspector.toasts.syncIngested_one' => ({required Object count}) => 'Ingested ${count} new memory file',
			'web.memoryInspector.toasts.syncIngested_other' => ({required Object count}) => 'Ingested ${count} new memory files',
			'web.memoryInspector.toasts.syncEmpty' => 'No new .md files to sync',
			'web.memoryInspector.toasts.syncEmptyDescription' => 'Already in sync, or no Claude memory dir for this cwd.',
			'web.memoryInspector.toasts.syncFailed' => 'Sync failed',
			'web.memoryInspector.toasts.testOk' => ({required Object embedder, required Object dim}) => 'Embedder OK: ${embedder} · ${dim} dimensions',
			'web.memoryInspector.toasts.testOkDescription' => ({required Object preview}) => 'vector_preview = [${preview}…]',
			'web.memoryInspector.toasts.testFailed' => 'Embedder probe failed',
			'web.memoryInspector.bulkDelete.title' => 'Delete every memory in this scope?',
			'web.memoryInspector.bulkDelete.description' => 'This is a single SQL operation — all memories under the specified scope are removed atomically. Memories that were ingested via the Claude mirror reappear on the next <1>Sync .md</1> run; everything else is gone for good.',
			'web.memoryInspector.bulkDelete.scope' => 'Scope',
			'web.memoryInspector.bulkDelete.scopeKey' => 'Scope key',
			'web.memoryInspector.bulkDelete.currentlyVisible' => 'Currently visible',
			'web.memoryInspector.bulkDelete.items_one' => ({required Object count}) => '${count} memory item',
			'web.memoryInspector.bulkDelete.items_other' => ({required Object count}) => '${count} memory items',
			'web.memoryInspector.bulkDelete.cancel' => 'Cancel',
			_ => null,
		} ?? switch (path) {
			'web.memoryInspector.bulkDelete.deleteAll' => 'Delete all',
			'web.memoryInspector.addMem.title' => 'Add memory',
			'web.memoryInspector.addMem.description' => 'Manually create a memory. Agents create these automatically via the <1>memory_store</1> MCP tool — this form is for cases where the operator wants to seed a fact without going through an agent.',
			'web.memoryInspector.addMem.textLabel' => 'Text',
			'web.memoryInspector.addMem.textPlaceholder' => 'Plain prose. The embedder turns this into a vector at store time; agents will retrieve it via memory_search.',
			'web.memoryInspector.addMem.cancel' => 'Cancel',
			'web.memoryInspector.addMem.create' => 'Create',
			'web.memoryInspector.picker.button' => 'Pick',
			'web.memoryInspector.picker.buttonTooltip' => 'Pick from saved scope keys or active sessions',
			'web.memoryInspector.picker.loading' => 'Loading…',
			'web.memoryInspector.picker.empty' => ({required Object scope}) => 'No saved keys or active sessions for ${scope}.',
			'web.memoryInspector.picker.savedHeader' => 'Saved memories',
			'web.memoryInspector.picker.activeHeader' => 'Active sessions',
			'web.memoryInspector.migrationBanner.headline_one' => ({required Object count}) => '${count} memory won\'t appear in searches',
			'web.memoryInspector.migrationBanner.headline_other' => ({required Object count}) => '${count} memories won\'t appear in searches',
			'web.memoryInspector.migrationBanner.subtitle' => ({required Object summary, required Object current}) => '${summary} — current embedder is <1>${current}</1>. pgvector partitions its similarity index by embedder, so older entries are silent until reembedded.',
			'web.memoryInspector.migrationBanner.summaryItem' => ({required Object count, required Object name}) => '${count} on ${name}',
			'web.memoryInspector.migrationBanner.migrateButton' => 'Migrate',
			'web.memoryInspector.reembed.title' => 'Reembed memories',
			'web.memoryInspector.reembed.description' => 'Recompute vectors for memories stored under a different embedder so they become searchable again.',
			'web.memoryInspector.reembed.targetEmbedder' => 'Target embedder',
			'web.memoryInspector.reembed.onName' => 'on',
			'web.memoryInspector.reembed.totalToReembed' => 'Total to reembed',
			'web.memoryInspector.reembed.explainer' => 'Each memory\'s text gets re-sent to the current embedder; the new vector replaces the old one in place. ID, scope, scope_key, metadata and timestamps are preserved. Search results take effect immediately — no restart needed.',
			'web.memoryInspector.reembed.reportExamined' => 'Examined',
			'web.memoryInspector.reembed.reportReembedded' => 'Reembedded',
			'web.memoryInspector.reembed.reportFailed' => 'Failed',
			'web.memoryInspector.reembed.reportFrom' => 'From',
			'web.memoryInspector.reembed.errors_one' => ({required Object count}) => '${count} error',
			'web.memoryInspector.reembed.errors_other' => ({required Object count}) => '${count} errors',
			'web.memoryInspector.reembed.done' => 'Done',
			'web.memoryInspector.reembed.cancel' => 'Cancel',
			'web.memoryInspector.reembed.reembedding' => 'Reembedding…',
			'web.memoryInspector.reembed.reembedTotal' => ({required Object total}) => 'Reembed ${total}',
			'web.notes.title' => 'Notes',
			'web.notes.header.outline' => 'Outline',
			'web.notes.header.showOutline' => 'Show outline',
			'web.notes.header.hideOutline' => 'Hide outline',
			'web.notes.header.today' => 'Today',
			'web.notes.header.todayTooltip' => 'Open or create today\'s daily note',
			'web.notes.header.kNew' => 'New',
			'web.notes.left.tree' => 'Tree',
			'web.notes.left.tags' => 'Tags',
			'web.notes.left.filterNotes' => 'Filter notes…',
			'web.notes.left.filterTags' => 'Filter tags…',
			'web.notes.left.filteredBy' => 'filtered by',
			'web.notes.left.clearTagTooltip' => 'Clear tag filter',
			'web.notes.left.expandAll' => 'Expand all',
			'web.notes.left.expandAllTooltip' => 'Expand every folder',
			'web.notes.left.collapseAll' => 'Collapse all',
			'web.notes.left.collapseAllTooltip' => 'Collapse every folder',
			'web.notes.left.loading' => 'Loading…',
			'web.notes.left.footer' => ({required Object visible, required Object total}) => '${visible} / ${total} notes',
			'web.notes.tags.emptyVault' => 'No tags in vault yet.',
			'web.notes.tags.noMatches' => ({required Object query}) => 'No matches for "${query}".',
			'web.notes.tree.empty' => 'Vault is empty.',
			'web.notes.outline.label' => 'Outline',
			'web.notes.outline.empty' => 'No headings in this note. Add <1>## Title</1> lines to populate the outline.',
			'web.notes.newNote.prompt' => 'New note path (vault-relative, must end .md)',
			'web.notes.newNote.defaultPath' => ({required Object date}) => 'library/notes-${date}.md',
			'web.notes.newNote.errorMustEndMd' => 'Path must end in .md',
			'web.notes.newNote.createdToast' => 'Note created',
			'web.notes.newNote.createFailedToast' => 'Create failed',
			'web.notes.empty.title' => 'No note selected',
			'web.notes.empty.hint' => 'Pick a note from the tree on the left, jump straight to today\'s daily log, or create a fresh one. AI-written project docs live under <1>projects/</1>; your personal scratchpads under <3>personal/</3>.',
			'web.notes.empty.today' => 'Today\'s daily note',
			'web.notes.empty.kNew' => 'New note',
			'web.notes.picker.browseAria' => 'Browse folders',
			'web.notes.picker.matches_one' => ({required Object count}) => '${count} match',
			'web.notes.picker.matches_other' => ({required Object count}) => '${count} matches',
			'web.notes.picker.foldersInVault' => ({required Object count}) => '${count} folders in vault',
			'web.notes.picker.noMatch' => ({required Object value}) => 'No existing folder matches. Save anyway to use <1>${value}</1> (lazy-created on first write).',
			'web.notes.vaultSync.title' => 'Vault sync',
			'web.notes.vaultSync.description' => 'Commit, pull, and push the notes vault as a git repository. Authentication uses your gateway host\'s git credentials (SSH agent / credential helper).',
			'web.notes.vaultSync.reading' => 'Reading vault state…',
			'web.notes.vaultSync.init.title' => 'Vault is not a git repo yet',
			'web.notes.vaultSync.init.body' => 'Initialising will run <1>git init -b main</1> in your vault root and add a sane <3>.gitignore</3>. After that you can commit your notes and configure a remote (GitHub / Gitea / GitLab) for cross-machine sync.',
			'web.notes.vaultSync.init.button' => 'Initialise vault as git repo',
			'web.notes.vaultSync.init.initToast' => 'Vault initialised as git repo',
			'web.notes.vaultSync.init.initFailedToast' => 'Init failed',
			'web.notes.vaultSync.branch.clean' => 'clean',
			'web.notes.vaultSync.branch.staged' => ({required Object count}) => '${count} staged',
			'web.notes.vaultSync.branch.modified' => ({required Object count}) => '${count} modified',
			'web.notes.vaultSync.branch.untracked' => ({required Object count}) => '${count} untracked',
			'web.notes.vaultSync.action.pull' => 'Pull',
			'web.notes.vaultSync.action.push' => 'Push',
			'web.notes.vaultSync.action.pullTitleNoRemote' => 'Configure a remote first',
			'web.notes.vaultSync.action.pullTitleHasUpstream' => 'git pull --rebase --autostash',
			'web.notes.vaultSync.action.pullTitleNoUpstream' => 'Pulls origin\'s HEAD; sets up tracking implicitly',
			'web.notes.vaultSync.action.pushTitleNoRemote' => 'Configure a remote first',
			'web.notes.vaultSync.action.pushTitleHasUpstream' => 'git push -u origin HEAD',
			'web.notes.vaultSync.action.pushTitleNoUpstream' => 'First push — will set upstream to origin/HEAD',
			'web.notes.vaultSync.action.noRemote' => 'No remote configured — pull/push disabled',
			'web.notes.vaultSync.action.noUpstream' => 'No upstream tracking yet — first push will set it.',
			'web.notes.vaultSync.action.pulledToast' => 'Pulled',
			'web.notes.vaultSync.action.pullFailedToast' => 'Pull failed',
			'web.notes.vaultSync.action.pushedToast' => 'Pushed',
			'web.notes.vaultSync.action.pushFailedToast' => 'Push failed',
			'web.notes.vaultSync.commit.title' => 'Commit',
			'web.notes.vaultSync.commit.placeholder' => ({required Object date}) => 'Notes: ${date} (default)',
			'web.notes.vaultSync.commit.commitAll' => 'Commit all',
			'web.notes.vaultSync.commit.hint' => 'Stages every change (<1>git add .</1>) then commits with this message. Empty message defaults to a timestamped subject.',
			'web.notes.vaultSync.commit.committedToast' => ({required Object hash}) => 'Committed ${hash}',
			'web.notes.vaultSync.commit.commitFailedToast' => 'Commit failed',
			'web.notes.vaultSync.fileList.title' => ({required Object count}) => 'Working tree · ${count}',
			'web.notes.vaultSync.fileList.moreSuffix' => ({required Object count}) => '+${count} more',
			'web.notes.vaultSync.remote.title' => 'Remote (origin)',
			'web.notes.vaultSync.remote.cancel' => 'Cancel',
			'web.notes.vaultSync.remote.change' => 'Change',
			'web.notes.vaultSync.remote.configure' => 'Configure',
			'web.notes.vaultSync.remote.empty' => 'No remote set. Add an HTTPS or SSH URL (e.g. <1>git@github.com:you/notes.git</1> or <3>https://gitea.example.com/you/notes.git</3>) to enable push / pull.',
			'web.notes.vaultSync.remote.urlLabel' => 'URL (HTTPS or SSH)',
			'web.notes.vaultSync.remote.urlPlaceholder' => 'git@host:owner/notes.git',
			'web.notes.vaultSync.remote.save' => 'Save',
			'web.notes.vaultSync.remote.savedToast' => 'Remote saved',
			'web.notes.vaultSync.remote.saveFailedToast' => 'Set remote failed',
			'web.notes.vaultSync.history.title' => 'Recent commits',
			'web.notes.vaultSync.history.loading' => 'Loading…',
			'web.notes.vaultSync.history.empty' => 'No commits yet.',
			'web.notes.vaultSync.conflict.kinds.rebase' => 'rebase',
			'web.notes.vaultSync.conflict.kinds.merge' => 'merge',
			'web.notes.vaultSync.conflict.kinds.cherryPick' => 'cherry-pick',
			'web.notes.vaultSync.conflict.kinds.operation' => 'operation',
			'web.notes.vaultSync.conflict.headline' => ({required Object kind}) => 'Vault has a paused ${kind} with unresolved conflicts',
			'web.notes.vaultSync.conflict.explainer' => ({required Object kind}) => 'Pull, push and commit are blocked until the ${kind} finishes. You can either <1>abort</1> (restore the working tree to its state before the ${kind} — keeps your local commits, drops the remote ones) or <3>force reset to remote</3> (discard ALL local commits + uncommitted changes; vault becomes an exact mirror of origin).',
			'web.notes.vaultSync.conflict.conflictedHeader' => ({required Object count}) => 'Conflicted files · ${count}',
			'web.notes.vaultSync.conflict.abort' => ({required Object kind}) => 'Abort ${kind}',
			'web.notes.vaultSync.conflict.abortTitle' => ({required Object kind}) => 'git ${kind} --abort',
			'web.notes.vaultSync.conflict.forceReset' => 'Force reset to remote',
			'web.notes.vaultSync.conflict.forceResetTitle' => 'git fetch && git reset --hard origin/<branch> && git clean -fd',
			'web.notes.vaultSync.conflict.forceResetConfirm' => ({required Object kind}) => 'DESTRUCTIVE: this will\n  • abort the in-progress ${kind}\n  • run git fetch origin\n  • reset --hard to origin/<branch>\n  • clean -fd (drop untracked files)\n\nAny local commits not pushed to origin AND any uncommitted edits will be PERMANENTLY LOST.\n\nContinue?',
			'web.notes.vaultSync.conflict.abortedToast' => ({required Object kind}) => 'Aborted ${kind}',
			'web.notes.vaultSync.conflict.abortedDescription' => 'Working tree restored to pre-operation state.',
			'web.notes.vaultSync.conflict.abortFailedToast' => 'Abort failed',
			'web.notes.vaultSync.conflict.resetToast' => ({required Object branch}) => 'Reset to ${branch}',
			'web.notes.vaultSync.conflict.resetDescription' => 'Local changes discarded; vault matches remote.',
			'web.notes.vaultSync.conflict.resetFailedToast' => 'Reset failed',
			'web.notes.vaultSync.auth.title' => 'Authentication',
			'web.notes.vaultSync.auth.httpsTokenOk' => ({required Object host}) => 'Will use the token stored for <1>${host}</1> in Plugins → Git hosts. ✓',
			'web.notes.vaultSync.auth.httpsTokenMissing' => ({required Object host}) => 'HTTPS remote on <1>${host}</1> with no opendray token configured. Push / pull will likely fail for private repos until you add one.',
			'web.notes.vaultSync.auth.ssh' => ({required Object host}) => 'SSH remote on <1>${host}</1>. Auth uses the gateway host\'s <3>~/.ssh/</3> (ssh-agent, identity file, host config). Verify with <5>ssh -T git@${host}</5> from the host shell.',
			'web.notes.vaultSync.auth.configureTokenLink' => '→ Configure git host token',
			'web.notes.vaultSync.autoSync.loading' => 'Loading auto-sync settings…',
			'web.notes.vaultSync.autoSync.title' => 'Auto-sync',
			'web.notes.vaultSync.autoSync.on' => 'on',
			'web.notes.vaultSync.autoSync.runNow' => 'Run now',
			'web.notes.vaultSync.autoSync.runNowTooltip' => 'Wake the sync loop now (skips the wait, then runs whichever steps are due)',
			'web.notes.vaultSync.autoSync.configure' => 'Configure',
			'web.notes.vaultSync.autoSync.hide' => 'Hide',
			'web.notes.vaultSync.autoSync.enabled' => 'Enabled',
			'web.notes.vaultSync.autoSync.enabledTooltipNoRemote' => 'Configure a remote first to enable auto-sync',
			'web.notes.vaultSync.autoSync.noRemoteHint' => 'No remote — push/pull will be skipped.',
			'web.notes.vaultSync.autoSync.commitEvery' => 'Commit every',
			'web.notes.vaultSync.autoSync.commitEveryExamples' => 'Examples: <1>30s</1>, <3>10m</3>, <5>2h</5>. Min 30s.',
			'web.notes.vaultSync.autoSync.pullEvery' => 'Pull every',
			'web.notes.vaultSync.autoSync.pullEveryHint' => 'Only used when Pull is enabled.',
			'web.notes.vaultSync.autoSync.pushAfterCommit' => 'Push after commit',
			'web.notes.vaultSync.autoSync.pullPeriodically' => 'Pull periodically',
			'web.notes.vaultSync.autoSync.commitTemplateLabel' => 'Commit message template',
			'web.notes.vaultSync.autoSync.commitTemplatePlaceholder' => ({required Object date}) => 'Auto-sync: ${date}  (default if empty)',
			'web.notes.vaultSync.autoSync.saveSettings' => 'Save settings',
			'web.notes.vaultSync.autoSync.discard' => 'Discard',
			'web.notes.vaultSync.autoSync.lastCommit' => 'last commit',
			'web.notes.vaultSync.autoSync.lastPush' => 'last push',
			'web.notes.vaultSync.autoSync.lastPull' => 'last pull',
			'web.notes.vaultSync.autoSync.never' => 'never',
			'web.notes.vaultSync.autoSync.savedToast' => 'Auto-sync settings saved',
			'web.notes.vaultSync.autoSync.saveFailedToast' => 'Save failed',
			'web.notes.vaultSync.autoSync.triggeredToast' => 'Auto-sync triggered',
			'web.notes.vaultSync.autoSync.runFailedToast' => 'Run failed',
			'web.notes.syncBadge.loading' => 'Loading…',
			'web.notes.syncBadge.syncLabel' => 'Sync',
			'web.notes.syncBadge.initLabel' => 'Init',
			'web.notes.syncBadge.initTooltip' => 'Vault is not a git repo yet',
			'web.notes.syncBadge.conflictLabel' => 'Conflict',
			'web.notes.syncBadge.conflictTooltip' => 'Vault has unresolved conflicts — click to recover',
			'web.notes.syncBadge.syncFallback' => 'sync',
			'web.notes.syncBadge.tooltip' => ({required Object branch, required Object files, required Object ahead, required Object behind}) => 'branch ${branch} · ${files} changes · ${ahead} ahead · ${behind} behind',
			'web.notes.syncBadge.tooltipAutoOn' => ' · auto-sync on',
			'web.notes.syncBadge.tooltipLastError' => ({required Object error}) => ' · last error: ${error}',
			'web.notes.syncBadge.branchPlaceholder' => '—',
			'web.activity.title' => 'Activity',
			'web.activity.subtitle' => 'Per-call audit of API requests made by registered integrations. Includes both inbound calls (a third-party app calling opendray with its API key) and outbound proxied calls (admin → opendray proxy → integration). Calls made directly by this admin UI are not recorded.',
			'web.activity.refresh' => 'Refresh',
			'web.activity.refreshTooltip' => 'Refresh',
			'web.activity.filters.integration' => 'Integration',
			'web.activity.filters.direction' => 'Direction',
			'web.activity.filters.status' => 'Status',
			'web.activity.filters.allIntegrations' => 'All integrations',
			'web.activity.filters.all' => 'All',
			'web.activity.filters.inbound' => 'Inbound',
			'web.activity.filters.outbound' => 'Outbound',
			'web.activity.filters.allStatuses' => 'All statuses',
			'web.activity.filters.status2' => '2xx success',
			'web.activity.filters.status3' => '3xx redirect',
			'web.activity.filters.status4' => '4xx client error',
			'web.activity.filters.status5' => '5xx server error',
			'web.activity.callsCount_one' => ({required Object count}) => '${count} call',
			'web.activity.callsCount_other' => ({required Object count}) => '${count} calls',
			'web.activity.loading' => 'Loading…',
			'web.activity.table.time' => 'Time',
			'web.activity.table.integration' => 'Integration',
			'web.activity.table.directionTitle' => 'Direction',
			'web.activity.table.method' => 'Method',
			'web.activity.table.path' => 'Path',
			'web.activity.table.status' => 'Status',
			'web.activity.table.duration' => 'Duration',
			'web.activity.table.inboundAria' => 'inbound',
			'web.activity.table.outboundAria' => 'outbound',
			'web.activity.empty.filtered' => 'No calls match these filters.',
			'web.activity.empty.title' => 'No API calls recorded yet',
			'web.activity.empty.description' => 'When a third-party app calls opendray with its integration API key, every request is logged here.',
			'web.activity.empty.stepWithIntegrations' => 'Use an existing integration\'s API key in your third-party app',
			'web.activity.empty.stepRegister' => 'Register an integration in Integrations → New',
			'web.activity.empty.stepCallEndpoint' => 'Call any endpoint, e.g. <1>POST /api/v1/sessions</1>',
			'web.activity.empty.stepAppears' => 'Calls appear here within seconds',
			'web.activity.empty.footnote' => 'Calls you make from this admin UI are not logged — only integration-attributed traffic is recorded.',
			'web.activity.events.loading' => 'Loading events…',
			'web.activity.events.empty' => 'No events yet.',
			'web.activity.events.emptyFiltered' => 'No matching events.',
			'web.activity.events.loadOlder' => 'Load older events',
			'web.activity.events.today' => 'Today',
			'web.activity.events.yesterday' => 'Yesterday',
			'web.providers.list.title' => 'Providers',
			'web.providers.list.loading' => 'Loading…',
			'web.providers.list.disabledBadge' => 'disabled',
			'web.providers.list.noneSelected' => 'No provider selected.',
			'web.providers.detail.enabled' => 'Enabled',
			'web.providers.detail.disabled' => 'Disabled',
			'web.providers.detail.toggleAria' => ({required Object name}) => 'Toggle ${name}',
			'web.providers.detail.configuration' => 'Configuration',
			'web.providers.detail.noConfig' => 'This provider has no user-configurable fields.',
			'web.providers.detail.executable' => 'executable:',
			'web.providers.detail.manifestHash' => 'manifest_hash:',
			'web.providers.detail.reset' => 'Reset',
			'web.providers.detail.save' => 'Save changes',
			'web.providers.detail.saving' => 'Saving…',
			'web.providers.detail.savedToast' => 'Provider config saved',
			'web.providers.detail.saveFailedToast' => 'Save failed',
			'web.providers.detail.toggleFailedToast' => 'Toggle failed',
			'web.providers.detail.caps.resume' => 'resume',
			'web.providers.detail.caps.stream' => 'stream',
			'web.providers.detail.caps.images' => 'images',
			'web.providers.detail.caps.mcp' => 'mcp',
			'web.providers.configForm.selectPlaceholder' => 'Select…',
			'web.providers.configForm.defaultOption' => '(default)',
			'web.providers.configForm.switchOn' => 'On',
			'web.providers.configForm.switchOff' => 'Off',
			'web.providers.configForm.showSecret' => 'Show secret',
			'web.providers.configForm.hideSecret' => 'Hide secret',
			'web.providers.claudeAccounts.title' => 'Claude accounts',
			'web.providers.claudeAccounts.tutorialTooltip' => 'Open the multi-account tutorial section',
			'web.providers.claudeAccounts.importLocal' => 'Import local',
			'web.providers.claudeAccounts.importLocalTooltip' => 'Scan ~/.claude-accounts/ on the gateway host and register any new directories. The button is gateway-host only — see the tutorial.',
			'web.providers.claudeAccounts.importedNothingToast' => 'Nothing to import — accounts already in sync.',
			'web.providers.claudeAccounts.importedToast_one' => ({required Object count}) => 'Imported ${count} account from ~/.claude-accounts',
			'web.providers.claudeAccounts.importedToast_other' => ({required Object count}) => 'Imported ${count} accounts from ~/.claude-accounts',
			'web.providers.claudeAccounts.importFailedToast' => 'Import failed',
			'web.providers.claudeAccounts.addingTitle' => 'Adding a new account.',
			'web.providers.claudeAccounts.addingBodyPrefix' => 'Run on the gateway host:',
			'web.providers.claudeAccounts.addingBodySuffix' => 'opendray\'s filesystem watcher will register the new directory automatically, or click <1>Import local</1> to scan immediately.',
			'web.providers.claudeAccounts.architectureLink' => 'Architecture & full guide →',
			'web.providers.claudeAccounts.loading' => 'Loading…',
			'web.providers.claudeAccounts.empty' => 'No Claude accounts yet. Run the shell command above on the gateway host, then click <1>Import local</1> to scan.',
			'web.providers.claudeAccounts.noTokenYet' => 'no token yet',
			'web.providers.claudeAccounts.configDir' => 'config_dir:',
			'web.providers.claudeAccounts.tokenPath' => 'token_path:',
			'web.providers.claudeAccounts.toggleFailedToast' => 'Toggle failed',
			'web.providers.claudeAccounts.removeConfirm' => ({required Object name}) => 'Remove account "${name}"?',
			'web.providers.claudeAccounts.removedToast' => 'Account removed',
			'web.providers.claudeAccounts.removeFailedToast' => 'Remove failed',
			'web.providers.claudeAccounts.toggleAria' => ({required Object name}) => 'Toggle ${name}',
			'web.providers.claudeAccounts.removeAria' => ({required Object name}) => 'Remove ${name}',
			'web.channels.title' => 'Channels',
			'web.channels.subtitle' => 'Bidirectional messaging integrations. Outbound notifications are filtered by each channel\'s <1>notify_on</1>.',
			'web.channels.newButton' => 'New channel',
			'web.channels.loading' => 'Loading…',
			'web.channels.empty.title' => 'No channels yet',
			'web.channels.empty.description' => 'Bundled kinds: Telegram · Slack · Discord · Feishu · DingTalk · WeCom. Pick one and paste credentials, or use <1>bridge</1> for a custom platform via WebSocket.',
			'web.channels.card.running' => 'running',
			'web.channels.card.starting' => 'starting…',
			'web.channels.card.disabled' => 'disabled',
			'web.channels.card.muted' => 'muted',
			'web.channels.card.tokenLabel' => 'token:',
			'web.channels.card.chatIdLabel' => 'chat_id:',
			'web.channels.card.channelIdLabel' => 'channel_id:',
			'web.channels.card.notifyOnLabel' => 'notify_on:',
			'web.channels.card.webhookLabel' => 'webhook:',
			'web.channels.card.copyWebhookTooltip' => 'Copy webhook URL',
			'web.channels.card.webhookCopiedToast' => 'Webhook URL copied',
			'web.channels.card.setup' => 'Setup',
			'web.channels.card.setupTooltip' => 'Show adapter connection details + sample code',
			'web.channels.card.test' => 'Test',
			'web.channels.card.testNotRunningTooltip' => 'Channel must be running',
			'web.channels.card.testBridgeTooltip' => 'Bridge channels cannot be tested from the admin — connect an adapter first',
			'web.channels.card.editAria' => 'Edit channel',
			'web.channels.card.editTooltip' => 'Edit channel config',
			'web.channels.card.deleteAria' => 'Delete channel',
			'web.channels.card.bridgeSuffix' => '(bridge)',
			'web.channels.toasts.testSent' => 'Test message sent',
			'web.channels.toasts.testFailed' => 'Test failed',
			'web.channels.toasts.deleteConfirm' => ({required Object id}) => 'Delete channel ${id}?',
			'web.channels.toasts.deleted' => 'Channel deleted',
			'web.channels.toasts.created' => 'Channel created',
			'web.channels.toasts.updated' => 'Channel updated',
			'web.channels.dialog.editTitle' => 'Edit channel',
			'web.channels.dialog.createTitle' => 'Register channel',
			'web.channels.dialog.descriptionBridge' => 'External adapter (Python/Node/...) connects via WebSocket and presents this token.',
			'web.channels.dialog.descriptionDefault' => 'Configure messaging integration.',
			'web.channels.dialog.kindLabel' => 'Kind',
			'web.channels.dialog.kindImmutable' => '(immutable — delete and recreate to change kind)',
			'web.channels.dialog.enabledLabel' => 'Enabled',
			'web.channels.dialog.enabledBridgeHint' => ' (accept adapter connections immediately)',
			'web.channels.dialog.enabledWebhookHint' => ' (start receiving webhooks immediately)',
			'web.channels.dialog.enabledDefaultHint' => ' (start immediately)',
			'web.channels.dialog.cancel' => 'Cancel',
			'web.channels.dialog.save' => 'Save',
			'web.channels.dialog.saving' => 'Saving…',
			'web.channels.dialog.create' => 'Create',
			'web.channels.dialog.creating' => 'Creating…',
			'web.channels.dialog.unknownKind' => ({required Object kind}) => 'Unknown kind: ${kind}',
			'web.channels.dialog.nameRequired' => 'name is required',
			'web.channels.dialog.tokenRequired' => 'token is required',
			'web.channels.dialog.topicIdsNumeric' => ({required Object value}) => 'Topic IDs must be numeric (got ${value})',
			'web.channels.dialog.fieldRequired' => ({required Object label}) => '${label} is required',
			'web.channels.dialog.cooldownInvalid' => 'Cooldown must be a non-negative number of seconds',
			'web.channels.dialog.snippetCapInvalid' => 'Snippet cap must be a non-negative number',
			'web.channels.notifications.sectionTitle' => 'Session notifications',
			'web.channels.notifications.notifyOnLabel' => 'Notify on',
			'web.channels.notifications.hintAll' => 'Receiving every session event. Click a tag to opt out.',
			'web.channels.notifications.hintNone' => 'No events selected — outbound notifications muted.',
			'web.channels.notifications.hintSome' => ({required Object selected, required Object total}) => 'Only ${selected} of ${total} topics selected.',
			'web.channels.notifications.repeatPolicyLabel' => 'Repeat policy',
			'web.channels.notifications.cooldownLabel' => 'Cooldown duration',
			'web.channels.notifications.onceReplyHint' => 'Replying with non-command text in this chat resets the suppression — opendray forwards your reply to the session\'s stdin and re-arms the notifier.',
			'web.channels.notifications.terminalSnippetLabel' => 'Terminal snippet',
			'web.channels.notifications.embedSnippetLabel' => 'Embed the recent terminal screen in idle notifications',
			'web.channels.notifications.snippetExplainer' => 'When enabled, the idle card includes a code-block snippet of what the user would see in the live web terminal — Claude TUI chrome (status spinner, "bypass permissions" hint, separator lines) is filtered out automatically.',
			'web.channels.notifications.modes.onceLabel' => 'Once per session (recommended)',
			'web.channels.notifications.modes.onceHint' => 'Fire once when a session goes idle, then stay silent until either the session ends or you reply via this channel.',
			'web.channels.notifications.modes.cooldownLabel' => 'Time-window cooldown',
			'web.channels.notifications.modes.cooldownHint' => 'Suppress repeats for the same (session, event) within the chosen window.',
			'web.channels.notifications.modes.everyLabel' => 'Every event (noisy)',
			'web.channels.notifications.modes.everyHint' => 'No suppression. Use only for low-frequency channels.',
			'web.channels.notifications.cooldowns.k60' => '1 minute',
			'web.channels.notifications.cooldowns.k300' => '5 minutes',
			'web.channels.notifications.cooldowns.k900' => '15 minutes',
			'web.channels.notifications.cooldowns.k1800' => '30 minutes',
			'web.channels.notifications.cooldowns.k3600' => '1 hour',
			'web.channels.notifications.snippetCaps.k0' => 'No cap — chunk into multiple messages (default)',
			'web.channels.notifications.snippetCaps.k1000' => '1000 chars (terse)',
			'web.channels.notifications.snippetCaps.k3000' => '3000 chars',
			'web.channels.notifications.snippetCaps.k6000' => '6000 chars',
			'web.channels.notifications.snippetCaps.k12000' => '12000 chars',
			'web.channels.bridge.nameLabel' => 'Bridge name',
			'web.channels.bridge.namePlaceholder' => 'wechat / discord-custom / whatsapp...',
			'web.channels.bridge.nameHint' => 'Human label for the adapter. Shown in the channels list.',
			'web.channels.bridge.tokenLabel' => 'Adapter token',
			'web.channels.bridge.regenerateTooltip' => 'Regenerate',
			'web.channels.bridge.copyTooltip' => 'Copy',
			'web.channels.bridge.tokenCopiedToast' => 'Token copied',
			'web.channels.bridge.tokenHint' => 'Adapter authenticates by sending this in the WS register frame (or as <1>X-Bridge-Token</1> header).',
			'web.channels.bridge.capsLabel' => 'Accept capabilities (optional whitelist)',
			'web.channels.bridge.capsHint' => 'Empty = accept whatever the adapter declares. Selected = only allow these capabilities even if the adapter offers more.',
			'web.channels.bridge.afterCreate' => 'After <1>Create</1>, the adapter setup dialog opens automatically with the WebSocket URL and copy-pasteable Python / Node / wscat starter code.',
			'web.channels.setup.title' => ({required Object name}) => 'Adapter setup — ${name}',
			'web.channels.setup.description' => 'Run an adapter (any language) that connects to opendray over WebSocket using these credentials. opendray will route session notifications and slash-command actions through it.',
			'web.channels.setup.wsUrlLabel' => 'WebSocket URL',
			'web.channels.setup.tokenLabel' => 'Adapter token',
			'web.channels.setup.authInfo' => ({required Object frame}) => '<1>Auth:</1> send the token as <3>X-Bridge-Token</3> header, <5>?token=</5> query param, or <7>Authorization: Bearer …</7>. The first WS frame must be <9>${frame}</9>. Full spec: <11>docs/bridge-protocol.md</11> in the repo.',
			'web.channels.setup.pythonInstall' => 'Install: <1>pip install websockets</1>. Run: <3>python adapter.py</3>.',
			'web.channels.setup.nodeInstall' => 'Install: <1>npm i ws</1>. Run: <3>node adapter.mjs</3>.',
			'web.channels.setup.wscatInstall' => 'Install: <1>npm i -g wscat</1>. Once connected, paste the JSON line shown above to register, then send further frames manually.',
			'web.channels.setup.close' => 'Close',
			'web.channels.setup.copyHide' => 'Hide',
			'web.channels.setup.copyShow' => 'Show',
			'web.channels.setup.copyLabelToast' => ({required Object label}) => '${label} copied',
			'web.channels.setup.copyCode' => 'Copy',
			'web.channels.setup.copied' => 'Copied',
			'web.channels.setup.codeCopiedToast' => 'Code copied',
			'web.integrations.title' => 'Integrations',
			'web.integrations.subtitle' => 'External apps that consume opendray. Reverse-proxy through <1>/api/v1/proxy/&lt;prefix&gt;/…</1> and subscribe to events via the WS endpoint.',
			'web.integrations.register' => 'Register',
			'web.integrations.loading' => 'Loading…',
			'web.integrations.tabs.registered' => 'Registered',
			'web.integrations.tabs.console' => 'Reverse proxy',
			'web.integrations.empty.title' => 'No integrations yet',
			'web.integrations.empty.description' => 'Register an external app to give it a scoped API key. Its code stays out of this repo.',
			'web.integrations.empty.register' => 'Register integration',
			'web.integrations.groupSystem' => 'System (managed by opendray)',
			'web.integrations.groupOperator' => 'Operator-registered',
			'web.integrations.card.managedBadge' => 'managed',
			'web.integrations.card.managedTooltip' => 'opendray manages this integration. Editing or rotating its key would orphan running sessions whose mcp.json holds the previous bearer.',
			'web.integrations.card.consumerBadge' => 'consumer',
			'web.integrations.card.consumerTooltip' => 'Consumer-only integration — no HTTP service to probe',
			'web.integrations.card.disabledBadge' => 'disabled',
			'web.integrations.card.consumerOnlyHint' => 'Consumes opendray\'s API. No reverse proxy mounted.',
			'web.integrations.card.lastProbed' => ({required Object relative}) => 'last probed ${relative}',
			'web.integrations.card.rotated' => ({required Object relative}) => 'rotated ${relative}',
			'web.integrations.card.managedReadOnly' => 'read-only — opendray bakes its key into every spawn\'s mcp.json',
			'web.integrations.card.managedReadOnlyTooltip' => 'opendray manages this row. To reset: delete ~/.opendray/memory.key and restart, or delete this row directly via SQL — it\'ll be re-bootstrapped at next startup.',
			'web.integrations.card.editAria' => 'Edit integration',
			'web.integrations.card.editTooltip' => 'Edit scopes / base URL / version',
			'web.integrations.card.rotateKey' => 'Rotate key',
			'web.integrations.card.deleteAria' => 'Delete integration',
			'web.integrations.card.rotateConfirm' => ({required Object name}) => 'Rotate the API key for "${name}"? The current key will stop working immediately.',
			'web.integrations.card.deleteConfirm' => ({required Object name}) => 'Delete integration ${name}?',
			'web.integrations.card.removedToast' => 'Integration removed',
			'web.integrations.register_dialog.title' => 'Register integration',
			'web.integrations.register_dialog.description' => 'Issues a one-time API key. Copy it before closing — opendray never displays the plaintext again.',
			'web.integrations.register_dialog.nameLabel' => 'Name',
			'web.integrations.register_dialog.namePlaceholder' => 'PetTracker',
			'web.integrations.register_dialog.modeHint' => 'Leave the next two fields blank for a <1>consumer-only</1> integration (third-party app that calls opendray\'s API but doesn\'t expose its own service). Fill both for a <3>reverse-proxy</3> integration.',
			'web.integrations.register_dialog.baseUrlLabel' => 'Base URL',
			'web.integrations.register_dialog.optionalSuffix' => '(optional)',
			'web.integrations.register_dialog.baseUrlPlaceholder' => 'http://192.168.3.42:8080',
			'web.integrations.register_dialog.routePrefixLabel' => 'Route prefix',
			'web.integrations.register_dialog.routePrefixPlaceholder' => 'pet-tracker',
			'web.integrations.register_dialog.routePrefixHint' => ({required Object prefix}) => 'Reachable at <1>/api/v1/proxy/${prefix}/*</1>.',
			'web.integrations.register_dialog.routePrefixPlaceholderToken' => '<prefix>',
			'web.integrations.register_dialog.versionLabel' => 'Version (optional)',
			'web.integrations.register_dialog.versionPlaceholder' => '0.1.0',
			'web.integrations.register_dialog.scopesLabel' => 'Scopes',
			'web.integrations.register_dialog.scopesIntro' => 'Pick the API surface this integration is allowed to call. Each toggle maps to a Bearer-token claim — opendray rejects requests that touch endpoints outside the granted set.',
			'web.integrations.register_dialog.errorNameRequired' => 'Name is required.',
			'web.integrations.register_dialog.errorBothOrNeither' => 'base_url and route_prefix go together. Set both for a reverse-proxy integration, leave both blank for a consumer-only integration.',
			'web.integrations.register_dialog.cancel' => 'Cancel',
			'web.integrations.register_dialog.submit' => 'Register',
			'web.integrations.register_dialog.submitting' => 'Registering…',
			'web.integrations.reveal.titleIssued' => 'API key issued',
			'web.integrations.reveal.titleRotated' => 'API key rotated',
			'web.integrations.reveal.description' => 'This is the only time the plaintext key will be shown. Copy it now and update every consumer app — the previous key (if any) no longer authenticates.',
			'web.integrations.reveal.discardAria' => 'Discard new key',
			'web.integrations.reveal.discardTooltip' => 'Discard the new key (rotation already happened — old key is gone too)',
			'web.integrations.reveal.discardConfirm' => 'Discard the new key? Rotation has already invalidated the old key — discarding means you have NO working key for this integration until you rotate again.',
			'web.integrations.reveal.copy' => 'Copy',
			'web.integrations.reveal.copied' => 'Copied',
			'web.integrations.reveal.updateHint' => '<1>Update every consumer app with this new key.</1> The previous key has been invalidated server-side and will return <3>401 unauthorized</3> on the next request.',
			'web.integrations.reveal.acknowledge' => 'I have copied the key and will update my consumer apps. I understand opendray will not display it again.',
			'web.integrations.reveal.discard' => 'Discard',
			'web.integrations.reveal.done' => 'Done',
			'web.integrations.edit_dialog.title' => ({required Object name}) => 'Edit integration · ${name}',
			'web.integrations.edit_dialog.description' => 'Change scopes, version, or base URL. Name and route prefix are immutable — delete + re-register if you need to change those.',
			'web.integrations.edit_dialog.nameLabel' => 'Name',
			'web.integrations.edit_dialog.routePrefixLabel' => 'Route prefix',
			'web.integrations.edit_dialog.consumerOnlyLabel' => '(consumer-only)',
			'web.integrations.edit_dialog.baseUrlLabel' => 'Base URL',
			'web.integrations.edit_dialog.baseUrlConsumerSuffix' => '(consumer-only — leave blank)',
			'web.integrations.edit_dialog.baseUrlProxySuffix' => '(reverse-proxy target)',
			'web.integrations.edit_dialog.baseUrlConsumerPlaceholder' => '(blank — this integration consumes opendray\'s API)',
			'web.integrations.edit_dialog.baseUrlProxyPlaceholder' => 'http://127.0.0.1:8080',
			'web.integrations.edit_dialog.consumerHint' => 'This is a consumer-only integration. Changing base URL here would also require a route prefix; do that with delete + re-register.',
			'web.integrations.edit_dialog.versionLabel' => 'Version',
			'web.integrations.edit_dialog.versionPlaceholder' => '0.1.0',
			'web.integrations.edit_dialog.scopesLabel' => 'Scopes',
			'web.integrations.edit_dialog.scopesIntro' => 'Trim or widen the API surface this integration\'s API key authorises. Live tokens are unaffected — the new scope set takes effect on the next request.',
			'web.integrations.edit_dialog.errorModeSwitch' => 'Switching between consumer-only and reverse-proxy mode requires deleting the integration and re-registering — name and route_prefix can\'t change in place.',
			'web.integrations.edit_dialog.updatedToast' => 'Integration updated',
			'web.integrations.edit_dialog.cancel' => 'Cancel',
			'web.integrations.edit_dialog.save' => 'Save changes',
			'web.integrations.proxy.emptyTitle' => 'No integrations registered',
			'web.integrations.proxy.emptyDescription' => ({required Object prefix}) => 'Register an integration first; the console proxies through /api/v1/proxy/${prefix}/* using the admin token.',
			'web.integrations.proxy.targetLabel' => 'Target',
			'web.integrations.proxy.selectPlaceholder' => 'Select integration…',
			'web.integrations.proxy.baseLabel' => 'base:',
			'web.integrations.proxy.history' => 'History',
			'web.integrations.proxy.historyEmpty' => 'no past requests for this integration',
			'web.integrations.proxy.send' => 'Send',
			'web.integrations.proxy.sending' => 'Sending…',
			'web.integrations.proxy.extraHeadersLabel' => 'Extra headers (one per line, Name: Value)',
			'web.integrations.proxy.bodyLabel' => 'Body',
			'web.integrations.proxy.headers' => 'Headers',
			'web.integrations.proxy.body' => 'Body',
			'web.integrations.proxy.emptyBody' => '(empty)',
			'web.integrations.proxy.requestFailed' => 'request failed',
			'web.integrations.proxy.stubText' => 'Send a request to see the upstream response.',
			'web.integrations.proxy.stubInjects' => 'opendray injects <1>X-Integration-ID</1> and strips your <3>Authorization</3> header.',
			'web.integrations.proxy.prefixPlaceholder' => '<prefix>',
			'web.plugins.title' => 'Inspector plugins',
			'web.plugins.subtitle' => 'Configure data sources surfaced in the right-hand Inspector panel when a session is open. Each plugin is admin-only and shared across all sessions. Click a section header to collapse it.',
			'web.plugins.common.loading' => 'Loading…',
			'web.plugins.common.cancel' => 'Cancel',
			'web.plugins.common.edit' => 'Edit',
			'web.plugins.common.add' => 'Add',
			'web.plugins.common.save' => 'Save',
			'web.plugins.common.create' => 'Create',
			'web.plugins.mcp.title' => 'MCP servers',
			'web.plugins.mcp.description' => ({required Object KEY}) => 'Model Context Protocol servers injected into every spawn (claude / codex). Vault entries live at <1>~/.opendray/vault/mcp/&lt;id&gt;/mcp.json</1>; secrets (referenced as <3>\$${KEY}</3> in env / headers) come from the <5>MCP secrets</5> section below.',
			'web.plugins.mcp.newServer' => 'New server',
			'web.plugins.mcp.empty' => 'No MCP servers yet. Add one to expose extra tools to your agent sessions.',
			'web.plugins.mcp.columns.name' => 'Name',
			'web.plugins.mcp.columns.transport' => 'Transport',
			'web.plugins.mcp.columns.spec' => 'Spec',
			'web.plugins.mcp.columns.enabled' => 'Enabled',
			'web.plugins.mcp.noUrl' => 'no url',
			'web.plugins.mcp.noCommand' => 'no command',
			'web.plugins.mcp.deleteConfirm' => ({required Object id}) => 'Delete MCP server "${id}"?',
			'web.plugins.mcp.removedToast' => 'MCP server removed',
			'web.plugins.mcp.deleteFailedToast' => 'Delete failed',
			'web.plugins.mcp.toggleFailedToast' => 'Toggle failed',
			'web.plugins.mcp.editor.createTitle' => 'New MCP server',
			'web.plugins.mcp.editor.editTitle' => ({required Object id}) => 'Edit MCP: ${id}',
			'web.plugins.mcp.editor.description' => ({required Object API_KEY}) => 'JSON shape: <1>command</1>+<3>args</3>+<5>env</5> for stdio (default), or <7>transport</7> +<9> url</9>+<11>headers</11> for sse / http. Reference secrets as <13>\$${API_KEY}</13> — they get substituted at spawn time from the secrets file.',
			'web.plugins.mcp.editor.idLabel' => 'ID',
			'web.plugins.mcp.editor.idPlaceholder' => 'filesystem',
			'web.plugins.mcp.editor.idHint' => 'Lowercase / digits / dash / underscore. Becomes both the directory name and the default <1>name</1>.',
			'web.plugins.mcp.editor.bodyLabel' => 'mcp.json',
			'web.plugins.mcp.editor.invalidJson' => ({required Object error}) => 'Invalid JSON: ${error}',
			'web.plugins.mcp.editor.createdToast' => 'MCP server created',
			'web.plugins.mcp.editor.savedToast' => 'MCP server saved',
			'web.plugins.mcp.editor.createFailedToast' => 'Create failed',
			'web.plugins.mcp.editor.saveFailedToast' => 'Save failed',
			_ => null,
		} ?? switch (path) {
			'web.plugins.mcpSecrets.title' => 'MCP secrets',
			'web.plugins.mcpSecrets.encryptedBadge' => 'encrypted',
			'web.plugins.mcpSecrets.plaintextBadge' => 'plaintext',
			'web.plugins.mcpSecrets.encryptedTooltip' => 'AES-GCM encrypted on disk; key stored in OS keychain',
			'web.plugins.mcpSecrets.plaintextTooltip' => 'OS keychain unavailable — file is plaintext on disk. Check the gateway log.',
			'web.plugins.mcpSecrets.description' => ({required Object KEY}) => 'Values referenced from <1>\$${KEY}</1> placeholders in any <3>mcp.json</3> get substituted at spawn time. <5>Saved values are never returned over the API</5> — you can overwrite or delete them but not read them back.',
			'web.plugins.mcpSecrets.descriptionStored' => ({required Object path}) => ' Stored at <1>${path}</1>.',
			'web.plugins.mcpSecrets.addSecret' => 'Add secret',
			'web.plugins.mcpSecrets.empty' => ({required Object KEY}) => 'No secrets stored. Add one to start referencing it as <1>\$${KEY}</1> in your MCP server configs.',
			'web.plugins.mcpSecrets.columns.key' => 'Key',
			'web.plugins.mcpSecrets.columns.value' => 'Value',
			'web.plugins.mcpSecrets.editTooltip' => 'Overwrite the stored value',
			'web.plugins.mcpSecrets.deleteConfirm' => ({required Object key}) => 'Delete secret "${key}"? Any mcp.json that references \$${key} will fall back to the literal placeholder until you set a new value.',
			'web.plugins.mcpSecrets.removedToast' => 'Secret removed',
			'web.plugins.mcpSecrets.deleteFailedToast' => 'Delete failed',
			'web.plugins.mcpSecrets.editor.addTitle' => 'Add secret',
			'web.plugins.mcpSecrets.editor.updateTitle' => ({required Object key}) => 'Update ${key}',
			'web.plugins.mcpSecrets.editor.addDescription' => ({required Object KEY}) => 'Stored encrypted on disk if the OS keychain is available. Reference it from any mcp.json env / headers / args / url with \$${KEY}.',
			'web.plugins.mcpSecrets.editor.editDescription' => 'Enter the new value to overwrite. The previous value cannot be recovered.',
			'web.plugins.mcpSecrets.editor.keyLabel' => 'Key',
			'web.plugins.mcpSecrets.editor.keyPlaceholder' => 'BRAVE_API_KEY',
			'web.plugins.mcpSecrets.editor.keyPattern' => 'Must match <1>[A-Za-z_][A-Za-z0-9_]*</1>',
			'web.plugins.mcpSecrets.editor.keyCollision' => 'Already exists — use Edit instead, or pick a different name.',
			'web.plugins.mcpSecrets.editor.valueLabel' => 'Value',
			'web.plugins.mcpSecrets.editor.valueHint' => 'Hidden as you type. Saved value is never returned over the API.',
			'web.plugins.mcpSecrets.editor.addedToast' => 'Secret added',
			'web.plugins.mcpSecrets.editor.updatedToast' => 'Secret updated',
			'web.plugins.mcpSecrets.editor.saveFailedToast' => 'Save failed',
			'web.plugins.skills.title' => 'Agent skills',
			'web.plugins.skills.description' => 'Reusable capabilities injected into Claude sessions as a Tier 1 index — the agent loads full SKILL.md on demand via <1>opendray skill describe &lt;id&gt;</1>. Built-ins ship in the binary but can be <3>customized</3> — your edits land at <5>~/.opendray/vault/skills/&lt;id&gt;/SKILL.md</5> and override the embedded version. Use Reset to revert.',
			'web.plugins.skills.newSkill' => 'New skill',
			'web.plugins.skills.empty' => 'No skills found.',
			'web.plugins.skills.columns.id' => 'ID',
			'web.plugins.skills.columns.description' => 'Description',
			'web.plugins.skills.columns.source' => 'Source',
			'web.plugins.skills.noDescription' => 'no description',
			'web.plugins.skills.builtinBadge' => 'builtin',
			'web.plugins.skills.builtinTooltip' => 'Embedded in the opendray binary — click Customize to override in your vault',
			'web.plugins.skills.vaultBadge' => 'vault',
			'web.plugins.skills.overridesBuiltin' => 'overrides builtin',
			'web.plugins.skills.overridesBuiltinTooltip' => 'This vault skill overrides the built-in version of the same id',
			'web.plugins.skills.customize' => 'Customize',
			'web.plugins.skills.customizeTooltip' => 'Open the SKILL.md and save changes as a vault override',
			'web.plugins.skills.editTooltip' => 'Edit this vault skill',
			'web.plugins.skills.resetTooltip' => 'Delete vault override and fall back to the built-in version',
			'web.plugins.skills.reset' => 'Reset',
			'web.plugins.skills.resetConfirm' => ({required Object id}) => 'Reset "${id}" to the built-in version? This deletes your vault SKILL.md and falls back to the embedded copy.',
			'web.plugins.skills.deleteConfirm' => ({required Object id}) => 'Delete skill "${id}" from your vault? This removes the SKILL.md file.',
			'web.plugins.skills.removedToast' => 'Skill removed',
			'web.plugins.skills.deleteFailedToast' => 'Delete failed',
			'web.plugins.skills.editor.createTitle' => 'New skill',
			'web.plugins.skills.editor.customizeTitle' => ({required Object id}) => 'Customize built-in: ${id}',
			'web.plugins.skills.editor.editTitle' => ({required Object id}) => 'Edit skill: ${id}',
			'web.plugins.skills.editor.customizeDescription' => 'You\'re viewing a built-in skill embedded in opendray. Saving will create a vault override at the same id — your edits live under ~/.opendray/vault/skills/<id>/SKILL.md and shadow the built-in until you Reset.',
			'web.plugins.skills.editor.editDescription' => 'SKILL.md format — frontmatter with name + description, then markdown instructions. The description appears in the agent\'s Tier 1 index.',
			'web.plugins.skills.editor.idLabel' => 'ID',
			'web.plugins.skills.editor.idPlaceholder' => 'my-helper',
			'web.plugins.skills.editor.idHint' => 'Lowercase / digits / dash / underscore. Becomes the directory name under <1>~/.opendray/vault/skills/&lt;id&gt;/</1>.',
			'web.plugins.skills.editor.bodyLabel' => 'SKILL.md',
			'web.plugins.skills.editor.createdToast' => 'Skill created',
			'web.plugins.skills.editor.savedToast' => 'Skill saved',
			'web.plugins.skills.editor.savedOverrideToast' => 'Saved as vault override',
			'web.plugins.skills.editor.createFailedToast' => 'Create failed',
			'web.plugins.skills.editor.saveFailedToast' => 'Save failed',
			'web.plugins.skills.editor.saveAsOverride' => 'Save as vault override',
			'web.plugins.customTasks.title' => 'Custom tasks',
			'web.plugins.customTasks.description' => 'Click-to-run shortcuts surfaced in the Tasks tab. Leave cwd blank for global tasks visible in every session, or pin to an absolute path to scope.',
			'web.plugins.customTasks.addTask' => 'Add task',
			'web.plugins.customTasks.empty' => 'No custom tasks yet.',
			'web.plugins.customTasks.columns.name' => 'Name',
			'web.plugins.customTasks.columns.command' => 'Command',
			'web.plugins.customTasks.columns.scope' => 'Scope',
			'web.plugins.customTasks.globalScope' => 'global',
			'web.plugins.customTasks.deleteConfirm' => ({required Object name}) => 'Delete custom task "${name}"?',
			'web.plugins.customTasks.removedToast' => 'Task removed',
			'web.plugins.customTasks.deleteFailedToast' => 'Delete failed',
			'web.plugins.customTasks.dialog.addTitle' => 'Add custom task',
			'web.plugins.customTasks.dialog.editTitle' => ({required Object name}) => 'Edit ${name}',
			'web.plugins.customTasks.dialog.description' => 'The command is sent verbatim into the session\'s terminal. Same as typing it at the prompt and pressing Enter.',
			'web.plugins.customTasks.dialog.nameLabel' => 'Name',
			'web.plugins.customTasks.dialog.namePlaceholder' => 'dev',
			'web.plugins.customTasks.dialog.commandLabel' => 'Command',
			'web.plugins.customTasks.dialog.commandPlaceholder' => 'docker compose up --build',
			'web.plugins.customTasks.dialog.descLabel' => 'Description (optional)',
			'web.plugins.customTasks.dialog.descPlaceholder' => 'Boots dev infra and tails logs',
			'web.plugins.customTasks.dialog.cwdLabel' => 'Cwd scope (optional)',
			'web.plugins.customTasks.dialog.cwdPlaceholder' => '/Users/me/projects/foo  (blank = global)',
			'web.plugins.customTasks.dialog.cwdHint' => 'Blank = visible in every session. Otherwise the task only shows when the session\'s cwd matches this absolute path.',
			'web.plugins.customTasks.dialog.addedToast' => 'Task added',
			'web.plugins.customTasks.dialog.updatedToast' => 'Task updated',
			'web.plugins.customTasks.dialog.addFailedToast' => 'Add failed',
			'web.plugins.customTasks.dialog.updateFailedToast' => 'Update failed',
			'web.plugins.gitHosts.title' => 'Git hosts',
			'web.plugins.gitHosts.description' => 'One token per host — used by the Git tab to fetch pull requests <1>and by the Notes vault sync</1> when its remote uses HTTPS to a private repo on the same host. GitHub.com, self-hosted GitHub Enterprise, Gitea, and GitLab are supported.',
			'web.plugins.gitHosts.addHost' => 'Add host',
			'web.plugins.gitHosts.empty' => 'No git hosts configured.\nAdd one to enable the PR list in the inspector\'s Git tab.',
			'web.plugins.gitHosts.columns.host' => 'Host',
			'web.plugins.gitHosts.columns.kind' => 'Kind',
			'web.plugins.gitHosts.columns.token' => 'Token',
			'web.plugins.gitHosts.columns.enabled' => 'Enabled',
			'web.plugins.gitHosts.statusEnabled' => 'enabled',
			'web.plugins.gitHosts.statusDisabled' => 'disabled',
			'web.plugins.gitHosts.deleteConfirm' => ({required Object host}) => 'Remove git host ${host}? PR queries against this host will stop working.',
			'web.plugins.gitHosts.removedToast' => 'Git host removed',
			'web.plugins.gitHosts.deleteFailedToast' => 'Delete failed',
			'web.plugins.gitHosts.dialog.addTitle' => 'Add git host',
			'web.plugins.gitHosts.dialog.editTitle' => ({required Object host}) => 'Edit ${host}',
			'web.plugins.gitHosts.dialog.description' => 'Token is stored on the gateway. Used only for read-only API calls (list PRs, etc.).',
			'web.plugins.gitHosts.dialog.kindLabel' => 'Kind',
			'web.plugins.gitHosts.dialog.kindGitHub' => 'GitHub',
			'web.plugins.gitHosts.dialog.kindGitea' => 'Gitea',
			'web.plugins.gitHosts.dialog.kindGitLab' => 'GitLab',
			'web.plugins.gitHosts.dialog.hostLabel' => 'Host',
			'web.plugins.gitHosts.dialog.hostPlaceholder' => 'github.com',
			'web.plugins.gitHosts.dialog.displayNameLabel' => 'Display name (optional)',
			'web.plugins.gitHosts.dialog.displayNamePlaceholder' => 'Personal',
			'web.plugins.gitHosts.dialog.tokenLabel' => 'Token',
			'web.plugins.gitHosts.dialog.newTokenLabel' => 'New token (leave blank to keep)',
			'web.plugins.gitHosts.dialog.tokenPlaceholder' => 'ghp_… / gho_… / glpat-…',
			'web.plugins.gitHosts.dialog.tokenPlaceholderEdit' => '…',
			'web.plugins.gitHosts.dialog.tokenHint' => 'GitHub: PAT with <1>repo</1> scope. Gitea: token with <3>read:repository</3>. GitLab: PAT with <5>read_api</5>.',
			'web.plugins.gitHosts.dialog.enabledLabel' => 'Enabled',
			'web.plugins.gitHosts.dialog.addedToast' => 'Git host added',
			'web.plugins.gitHosts.dialog.updatedToast' => 'Git host updated',
			'web.plugins.gitHosts.dialog.addFailedToast' => 'Add failed',
			'web.plugins.gitHosts.dialog.updateFailedToast' => 'Update failed',
			'web.backups.title' => 'Backups',
			'web.backups.subtitle' => 'Encrypted PostgreSQL dumps written to a pluggable target. Configure schedules + retention, or trigger one-off backups for a quick safety net.',
			'web.backups.exportData' => 'Export data',
			'web.backups.loading' => 'Loading…',
			'web.backups.loadStatusFailedToast' => 'Failed to load backup status',
			'web.backups.tabs.backups' => 'Backups',
			'web.backups.tabs.schedules' => 'Schedules',
			'web.backups.tabs.targets' => 'Targets',
			'web.backups.inventory.title' => 'What\'s in a backup?',
			'web.backups.inventory.summary' => ({required Object rows, required Object tables}) => '${rows} rows across ${tables} tables',
			'web.backups.inventory.description' => 'Each backup is a <1>pg_dump --format=custom</1> of every table below, plus <3>manifest.json</3> and (optionally) <5>config.toml</5>. Counts are live; the bundle captures whatever\'s there at backup time.',
			'web.backups.inventory.loadFailedToast' => 'Failed to load inventory',
			'web.backups.inventory.rowsLabel' => 'rows',
			'web.backups.restart.title' => 'Restart opendray to activate backups',
			'web.backups.restart.description' => 'Your passphrase is saved. The gateway only loads it at startup, so the feature stays off until you bounce the process.',
			'web.backups.restart.keyFile' => 'Key file:',
			'web.backups.restart.configuredVia' => 'Configured via:',
			'web.backups.restart.envVar' => 'OPENDRAY_BACKUP_KEY env var',
			'web.backups.restart.checkAgain' => 'Check again',
			'web.backups.setup.title' => 'Set up backups',
			'web.backups.setup.description' => 'Choose a master passphrase. opendray uses it to encrypt every backup blob. <1>Lose it and your backups become unrecoverable</1>, so save it in a password manager (Vaultwarden, 1Password, …) before continuing.',
			'web.backups.setup.generate' => 'Generate',
			'web.backups.setup.pasteOwn' => 'Paste my own',
			'web.backups.setup.generateTitle' => '256-bit random key',
			'web.backups.setup.generateHint' => 'Server generates a cryptographically random passphrase and shows it once. You must copy it before continuing — there is no recovery path.',
			'web.backups.setup.pasteLabel' => 'Your passphrase',
			'web.backups.setup.pastePlaceholder' => 'At least 20 characters',
			'web.backups.setup.pasteHint' => 'Recommended: 40+ characters from a password manager.',
			'web.backups.setup.savesTo' => 'Saves to:',
			'web.backups.setup.saving' => 'Saving…',
			'web.backups.setup.generateAndSave' => 'Generate and save',
			'web.backups.setup.save' => 'Save',
			'web.backups.generated.title' => 'Save this passphrase NOW',
			'web.backups.generated.description' => 'This is shown <1>once</1>. It will not be retrievable from opendray or anywhere else. Copy it into a password manager before continuing.',
			'web.backups.generated.copy' => 'Copy',
			'web.backups.generated.copiedToast' => 'Passphrase copied to clipboard',
			'web.backups.generated.copyFailedToast' => 'Copy failed — select and copy manually',
			'web.backups.generated.savedTo' => 'Saved to:',
			'web.backups.generated.ack' => 'I have saved this passphrase to my password manager',
			'web.backups.generated.kContinue' => 'Continue',
			'web.backups.status.keyFingerprint' => 'Key fingerprint:',
			'web.backups.status.pgDump' => 'pg_dump:',
			'web.backups.status.pgDumpUnavailable' => 'unavailable',
			'web.backups.status.pgDumpHint' => 'Backups can\'t run until pg_dump is on PATH (or its absolute path is set in <1>backup.pg_dump_path</1>). Install <3>postgresql-client</3> matching your server\'s major version and restart.',
			'web.backups.backupsTab.backupNow' => 'Backup now',
			'web.backups.backupsTab.triggering' => 'Triggering…',
			'web.backups.backupsTab.includeConfig' => 'include config.toml',
			'web.backups.backupsTab.restoreFromFile' => 'Restore from file',
			'web.backups.backupsTab.refresh' => 'Refresh',
			'web.backups.backupsTab.queuedToast' => 'Backup queued',
			'web.backups.backupsTab.triggerFailedToast' => 'Trigger failed',
			'web.backups.backupsTab.listFailedToast' => 'Failed to list backups',
			'web.backups.backupsTab.deleteConfirm' => ({required Object id}) => 'Delete backup ${id}? The blob is removed from its target.',
			'web.backups.backupsTab.deletedToast' => 'Backup deleted',
			'web.backups.backupsTab.deleteFailedToast' => 'Delete failed',
			'web.backups.backupsTab.empty' => 'No backups yet. Click "Backup now" above to take the first one.',
			'web.backups.backupsTab.columns.id' => 'ID',
			'web.backups.backupsTab.columns.target' => 'Target',
			'web.backups.backupsTab.columns.status' => 'Status',
			'web.backups.backupsTab.columns.started' => 'Started',
			'web.backups.backupsTab.columns.size' => 'Size',
			'web.backups.backupsTab.columns.actions' => 'Actions',
			'web.backups.backupsTab.downloadTooltip' => 'Download',
			'web.backups.backupsTab.deleteTooltip' => 'Delete',
			'web.backups.restore.title' => 'Restore from backup bundle',
			'web.backups.restore.bundleLabel' => 'Encrypted bundle (.tar.gz.enc)',
			'web.backups.restore.targetDsnLabel' => 'Target database DSN',
			'web.backups.restore.targetDsnHint' => '(blank = opendray\'s own DB — DANGEROUS)',
			'web.backups.restore.targetDsnPlaceholder' => 'postgres://user:pass@host:5432/dbname',
			'web.backups.restore.cleanLabel' => '--clean --if-exists (drop existing schema first; required when restoring over a populated DB)',
			'web.backups.restore.auditNoteLabel' => 'Audit note (optional)',
			'web.backups.restore.auditNotePlaceholder' => 'Reason for restore — appears in slog',
			'web.backups.restore.ownDbWarning' => 'You\'re restoring into <1>opendray\'s own database</1>. With "--clean" enabled this drops every table and replays the backup verbatim — irreversible. Type <3>I understand</3> to proceed.',
			'web.backups.restore.confirmPlaceholder' => 'I understand',
			'web.backups.restore.confirmSentinel' => 'I understand',
			'web.backups.restore.pgRestoreOutput' => 'pg_restore output (last 8 KiB)',
			'web.backups.restore.noPgRestoreOutput' => '(no pg_restore output)',
			'web.backups.restore.pickFileToast' => 'Pick a bundle file first',
			'web.backups.restore.succeededToast' => 'Restore succeeded',
			'web.backups.restore.replayedDescription' => ({required Object bytes, required Object id}) => '${bytes} replayed from manifest ${id}',
			'web.backups.restore.failedToast' => 'Restore failed',
			'web.backups.restore.restoring' => 'Restoring…',
			'web.backups.restore.restore' => 'Restore',
			'web.backups.schedulesTab.description' => 'Recurring backups. The scheduler polls every 30s and runs the oldest due schedule.',
			'web.backups.schedulesTab.newSchedule' => 'New schedule',
			'web.backups.schedulesTab.loadFailedToast' => 'Failed to load schedules',
			'web.backups.schedulesTab.deleteConfirm' => ({required Object id}) => 'Delete schedule ${id}?',
			'web.backups.schedulesTab.deletedToast' => 'Schedule deleted',
			'web.backups.schedulesTab.deleteFailedToast' => 'Delete failed',
			'web.backups.schedulesTab.toggleFailedToast' => 'Toggle failed',
			'web.backups.schedulesTab.empty' => 'No schedules. Add one to take automatic recurring backups.',
			'web.backups.schedulesTab.columns.id' => 'ID',
			'web.backups.schedulesTab.columns.target' => 'Target',
			'web.backups.schedulesTab.columns.interval' => 'Interval',
			'web.backups.schedulesTab.columns.keep' => 'Keep',
			'web.backups.schedulesTab.columns.nextRun' => 'Next run',
			'web.backups.schedulesTab.columns.enabled' => 'Enabled',
			'web.backups.schedulesTab.columns.actions' => 'Actions',
			'web.backups.schedulesTab.keepCount' => ({required Object count}) => '${count} backups',
			'web.backups.schedulesTab.deleteTooltip' => 'Delete',
			'web.backups.newSchedule.title' => 'New backup schedule',
			'web.backups.newSchedule.targetLabel' => 'Target',
			'web.backups.newSchedule.everyHoursLabel' => 'Every (hours)',
			'web.backups.newSchedule.keepLastNLabel' => 'Keep last N',
			'web.backups.newSchedule.enableImmediately' => 'Enable immediately',
			'web.backups.newSchedule.createdToast' => 'Schedule created',
			'web.backups.newSchedule.createFailedToast' => 'Create failed',
			'web.backups.newSchedule.creating' => 'Creating…',
			'web.backups.newSchedule.create' => 'Create',
			'web.backups.targetsTab.description' => 'Storage destinations. v1 supports <1>local</1> (disk on the opendray host) and <3>smb</3> (any SMB / CIFS share, e.g. UNAS or Synology).',
			'web.backups.targetsTab.newTarget' => 'New target',
			'web.backups.targetsTab.listFailedToast' => 'Failed to list targets',
			'web.backups.targetsTab.deleteConfirm' => ({required Object id}) => 'Delete target ${id}? Schedules referencing it will block the delete.',
			'web.backups.targetsTab.deletedToast' => 'Target deleted',
			'web.backups.targetsTab.deleteFailedToast' => 'Delete failed',
			'web.backups.targetsTab.connectionOkToast' => 'Connection OK',
			'web.backups.targetsTab.connectionFailedToast' => 'Connection failed',
			'web.backups.targetsTab.testFailedToast' => 'Test failed',
			'web.backups.targetsTab.columns.id' => 'ID',
			'web.backups.targetsTab.columns.kind' => 'Kind',
			'web.backups.targetsTab.columns.config' => 'Config',
			'web.backups.targetsTab.columns.enabled' => 'Enabled',
			'web.backups.targetsTab.columns.actions' => 'Actions',
			'web.backups.targetsTab.on' => 'on',
			'web.backups.targetsTab.off' => 'off',
			'web.backups.targetsTab.test' => 'Test',
			'web.backups.targetsTab.testing' => 'Testing…',
			'web.backups.targetsTab.deleteTooltip' => 'Delete',
			'web.backups.targetEditor.title' => 'New backup target',
			'web.backups.targetEditor.kindPicker' => 'Where do you want to back up to?',
			'web.backups.targetEditor.idLabel' => 'ID (optional)',
			'web.backups.targetEditor.idPlaceholder' => 'auto-generated if blank, e.g. tgt_xxx',
			'web.backups.targetEditor.createdToast' => 'Target created',
			'web.backups.targetEditor.createFailedToast' => 'Create failed',
			'web.backups.targetEditor.creating' => 'Creating…',
			'web.backups.targetEditor.create' => 'Create target',
			'web.backups.targetEditor.enableImmediately' => 'Enable immediately (otherwise saved as disabled — useful for "configure now, turn on later")',
			'web.backups.targetEditor.local.rootLabel' => 'Root directory',
			'web.backups.targetEditor.local.rootHint' => 'Empty = cfg.backup.local_dir (~/.opendray/backups)',
			'web.backups.targetEditor.local.rootPlaceholder' => '~/backups/opendray  or  /mnt/external-hdd/opendray',
			'web.backups.targetEditor.smb.hostLabel' => 'Host',
			'web.backups.targetEditor.smb.hostPlaceholder' => '192.168.9.8',
			'web.backups.targetEditor.smb.portLabel' => 'Port',
			'web.backups.targetEditor.smb.shareLabel' => 'Share',
			'web.backups.targetEditor.smb.shareHint' => 'Top-level share name on the SMB server',
			'web.backups.targetEditor.smb.sharePlaceholder' => 'Claude_Workspace',
			'web.backups.targetEditor.smb.userLabel' => 'User',
			'web.backups.targetEditor.smb.passwordLabel' => 'Password',
			'web.backups.targetEditor.smb.pathPrefixLabel' => 'Path prefix',
			'web.backups.targetEditor.smb.pathPrefixHint' => 'Sub-folder under the share root (optional)',
			'web.backups.targetEditor.smb.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.s3.endpointLabel' => 'Endpoint',
			'web.backups.targetEditor.s3.endpointHint' => 'Host (no protocol). AWS: s3.amazonaws.com · R2: <accountid>.r2.cloudflarestorage.com · MinIO: minio.local:9000',
			'web.backups.targetEditor.s3.endpointPlaceholder' => 's3.amazonaws.com',
			'web.backups.targetEditor.s3.regionLabel' => 'Region',
			'web.backups.targetEditor.s3.regionHint' => 'AWS only; R2 use \'auto\'',
			'web.backups.targetEditor.s3.regionPlaceholder' => 'us-east-1 / auto',
			'web.backups.targetEditor.s3.bucketLabel' => 'Bucket',
			'web.backups.targetEditor.s3.bucketPlaceholder' => 'opendray-backups',
			'web.backups.targetEditor.s3.accessKeyLabel' => 'Access key',
			'web.backups.targetEditor.s3.secretKeyLabel' => 'Secret key',
			'web.backups.targetEditor.s3.secretKeyHint' => 'Stored AES-256-GCM encrypted; never echoed back',
			'web.backups.targetEditor.s3.pathPrefixLabel' => 'Path prefix',
			'web.backups.targetEditor.s3.pathPrefixHint' => 'Object-key prefix (optional)',
			'web.backups.targetEditor.s3.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.s3.useHttps' => 'Use HTTPS',
			'web.backups.targetEditor.s3.pathStyle' => 'Path-style addressing (legacy / MinIO)',
			'web.backups.targetEditor.webdav.baseUrlLabel' => 'Base URL',
			'web.backups.targetEditor.webdav.baseUrlHint' => 'Full URL including any path. Examples: https://cloud.example.com/remote.php/dav/files/me/ (Nextcloud), https://nas.local:5006/ (Synology), https://dav.jianguoyun.com/dav/ (Jianguoyun / 坚果云)',
			'web.backups.targetEditor.webdav.baseUrlPlaceholder' => 'https://cloud.example.com/remote.php/dav/files/<user>/',
			'web.backups.targetEditor.webdav.userLabel' => 'User',
			'web.backups.targetEditor.webdav.passwordLabel' => 'Password',
			'web.backups.targetEditor.webdav.pathPrefixLabel' => 'Path prefix',
			'web.backups.targetEditor.webdav.pathPrefixHint' => 'Sub-folder under the base URL (optional)',
			'web.backups.targetEditor.webdav.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.sftp.hostLabel' => 'Host',
			'web.backups.targetEditor.sftp.hostPlaceholder' => 'vps.example.com',
			'web.backups.targetEditor.sftp.portLabel' => 'Port',
			'web.backups.targetEditor.sftp.userLabel' => 'User',
			'web.backups.targetEditor.sftp.passwordLabel' => 'Password',
			'web.backups.targetEditor.sftp.passwordHint' => 'Either password OR private key required. If both, password is treated as the key passphrase.',
			'web.backups.targetEditor.sftp.privateKeyLabel' => 'Private key (PEM)',
			'web.backups.targetEditor.sftp.privateKeyHint' => 'Paste contents of an OpenSSH/PEM private key (e.g. ~/.ssh/id_ed25519). Leave blank for password-only auth.',
			'web.backups.targetEditor.sftp.privateKeyPlaceholder' => '-----BEGIN OPENSSH PRIVATE KEY-----...',
			'web.backups.targetEditor.sftp.hostKeyLabel' => 'Host key (pinning)',
			'web.backups.targetEditor.sftp.hostKeyHint' => 'OpenSSH-style server public key (run `ssh-keyscan host` to obtain). Leave blank to disable pinning (NOT recommended outside LAN).',
			'web.backups.targetEditor.sftp.hostKeyPlaceholder' => 'ssh-ed25519 AAAA...',
			'web.backups.targetEditor.sftp.pathPrefixLabel' => 'Path prefix',
			'web.backups.targetEditor.sftp.pathPrefixHint' => 'Absolute or relative to user home (optional)',
			'web.backups.targetEditor.sftp.pathPrefixPlaceholder' => '/var/backups/opendray  or  opendray-backups',
			'web.backups.targetEditor.rclone.rcloneHint' => 'Requires the <1>rclone</1> CLI installed on the opendray host. First configure your remote with <3>rclone config</3>, then use the remote name below. opendray invokes <5>rclone rcat / cat / lsd</5> under the hood.',
			'web.backups.targetEditor.rclone.remoteLabel' => 'Remote name',
			'web.backups.targetEditor.rclone.remoteHint' => 'Name from `rclone config` (no colon). Example: gdrive, onedrive, dropbox-personal, baidu-pan',
			'web.backups.targetEditor.rclone.remotePlaceholder' => 'gdrive',
			'web.backups.targetEditor.rclone.pathPrefixLabel' => 'Path prefix',
			'web.backups.targetEditor.rclone.pathPrefixHint' => 'Sub-folder under the remote root (optional)',
			'web.backups.targetEditor.rclone.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.rclone.binaryPathLabel' => 'Binary path',
			'web.backups.targetEditor.rclone.binaryPathHint' => 'Override `which rclone`. Empty uses PATH lookup.',
			'web.backups.targetEditor.rclone.binaryPathPlaceholder' => '/opt/homebrew/bin/rclone',
			'web.backups.targetEditor.rclone.configPathLabel' => 'Config path',
			'web.backups.targetEditor.rclone.configPathHint' => 'Override --config (default ~/.config/rclone/rclone.conf or ~/.rclone.conf)',
			'web.backups.targetEditor.rclone.configPathPlaceholder' => 'leave blank for rclone default',
			'web.serverSettings.sections.general.title' => 'General',
			'web.serverSettings.sections.general.desc' => 'Listen address, operator account, token TTL.',
			'web.serverSettings.sections.logging.title' => 'Logging',
			'web.serverSettings.sections.logging.desc' => 'Verbosity, format, and live tail.',
			'web.serverSettings.sections.sessions.title' => 'Sessions',
			'web.serverSettings.sections.sessions.desc' => 'Idle detection thresholds.',
			'web.serverSettings.sections.vault.title' => 'Vault',
			'web.serverSettings.sections.vault.desc' => 'Notes, skills, and git-versioned root.',
			'web.serverSettings.sections.mcp.title' => 'MCP registry',
			'web.serverSettings.sections.mcp.desc' => 'Server registry + secrets.',
			'web.serverSettings.sections.memory.title' => 'Memory',
			'web.serverSettings.sections.memory.desc' => 'Cross-CLI persistent memory subsystem.',
			'web.serverSettings.sections.backup.title' => 'Backup',
			'web.serverSettings.sections.backup.desc' => 'Encrypted DB backups, restore, and admin data exports.',
			'web.serverSettings.sections.claude.title' => 'Storage · Claude',
			'web.serverSettings.sections.claude.desc' => 'Where Claude transcripts live on disk.',
			'web.serverSettings.sections.codex.title' => 'Storage · Codex',
			'web.serverSettings.sections.codex.desc' => 'Codex sessions root.',
			'web.serverSettings.sections.gemini.title' => 'Storage · Gemini',
			'web.serverSettings.sections.gemini.desc' => 'Gemini per-project tmp + projects.json.',
			'web.serverSettings.loading' => 'Loading server settings…',
			'web.serverSettings.loadFailed' => ({required Object message}) => 'Failed to load: ${message}',
			'web.serverSettings.noConfigFlag' => 'opendray was started without a -config flag. Settings are loaded from environment variables only and cannot be edited here.',
			'web.serverSettings.resetButton' => 'Reset',
			'web.serverSettings.resetButtonTitle' => 'Discard unsaved changes in this section',
			'web.serverSettings.resetConfirm' => ({required Object section}) => 'Reset "${section}" to last-saved values?',
			'web.serverSettings.badgeRestartRequired' => 'restart required',
			'web.serverSettings.badgeUnsaved' => 'unsaved',
			'web.serverSettings.saveButton' => 'Save changes',
			'web.serverSettings.saveToastTitle' => 'Settings saved',
			'web.serverSettings.saveToastDesc' => 'Click Restart to apply.',
			'web.serverSettings.saveErrorTitle' => 'Save failed',
			'web.serverSettings.dangerousConfirm' => 'You changed listen address / admin user / admin password. After restart you may need to re-authenticate or use the new address. Continue?',
			'web.serverSettings.unsavedHint' => 'You have unsaved changes',
			'web.serverSettings.savedHint' => 'All changes saved',
			'web.serverSettings.searchPlaceholder' => 'Filter fields…',
			'web.serverSettings.restart.button' => 'Restart server',
			'web.serverSettings.restart.buttonTitle' => 'Self-exec the gateway process',
			'web.serverSettings.restart.dirtyConfirm' => 'You have unsaved changes. Restart will use the LAST SAVED config. Continue?',
			'web.serverSettings.restart.confirm' => 'Restart the opendray gateway? All open terminal sessions will reconnect automatically.',
			'web.serverSettings.restart.overlay' => 'Restarting server…',
			'web.serverSettings.restart.waiting' => ({required Object tick}) => 'Waiting for /health · ${tick}s',
			'web.serverSettings.restart.timedOutTitle' => 'Restart timed out',
			'web.serverSettings.restart.timedOutDesc' => 'Health endpoint never came back. Check server logs.',
			'web.serverSettings.restart.successToast' => 'Server restarted',
			'web.serverSettings.formGroups.network' => 'Network',
			'web.serverSettings.formGroups.operatorAccount' => 'Operator account',
			'web.serverSettings.formGroups.memoryConfiguration' => 'Configuration',
			'web.serverSettings.formGroups.memoryHttp' => 'HTTP backend (used when backend=http)',
			'web.serverSettings.formGroups.memoryLocal' => 'Local ONNX (used when backend=local)',
			'web.serverSettings.formGroups.backupStatus' => 'Status',
			'web.serverSettings.formGroups.backupWhere' => 'Where backups go',
			'web.serverSettings.formGroups.backupSchedules' => 'Schedules',
			'web.serverSettings.formGroups.backupWhatsInside' => 'What\'s in a backup?',
			'web.serverSettings.fields.listenAddress.label' => 'Listen address',
			'web.serverSettings.fields.listenAddress.hint' => 'The host:port the HTTP server binds to. Example: 0.0.0.0:8770.',
			'web.serverSettings.fields.username.label' => 'Username',
			'web.serverSettings.fields.username.hint' => 'Login name used in the sign-in form. Changing this forces a re-login on the next request.',
			'web.serverSettings.fields.password.label' => 'Password',
			'web.serverSettings.fields.password.hint' => 'Leave blank to keep the current password. Sending a value overwrites it.',
			'web.serverSettings.fields.password.hideTitle' => 'Hide',
			'web.serverSettings.fields.password.revealTitle' => 'Reveal',
			'web.serverSettings.fields.tokenTTL.label' => 'Token TTL',
			'web.serverSettings.fields.tokenTTL.hint' => 'Bearer-token lifetime as a Go duration, e.g. "24h", "30m". Empty = never expire.',
			'web.serverSettings.fields.logLevel.label' => 'Log level',
			'web.serverSettings.fields.logLevel.hint' => 'Lines below this level are dropped.',
			'web.serverSettings.fields.logFormat.label' => 'Format',
			'web.serverSettings.fields.logFormat.hint' => '"text" is human-readable; "json" is machine-parsable.',
			'web.serverSettings.fields.logFile.label' => 'Log file',
			'web.serverSettings.fields.logFile.hint' => 'Optional file path. Auto-rotates at 10 MB, keeps 5 backups. Empty = stderr only.',
			'web.serverSettings.fields.idleThreshold.label' => 'Idle threshold',
			'web.serverSettings.fields.idleThreshold.hint' => 'A session is silent for this long before session.idle fires. Empty = 30s.',
			'web.serverSettings.fields.idlePollInterval.label' => 'Idle poll interval',
			'web.serverSettings.fields.idlePollInterval.hint' => 'How often the idle detector wakes up. Lower = lower latency, more wakeups. Empty = 5s.',
			'web.serverSettings.fields.vaultRoot.label' => 'Vault root',
			'web.serverSettings.fields.vaultRoot.hint' => 'Top-level directory for notes, skills, and MCP registry.',
			'web.serverSettings.fields.notesDirectory.label' => 'Notes directory',
			'web.serverSettings.fields.notesDirectory.hint' => 'Override notes location. Defaults to <vault root>/notes.',
			'web.serverSettings.fields.skillsDirectory.label' => 'Skills directory',
			'web.serverSettings.fields.skillsDirectory.hint' => 'Override skills location. Defaults to <vault root>/skills.',
			'web.serverSettings.fields.gitRoot.label' => 'Git root',
			'web.serverSettings.fields.gitRoot.hint' => 'Working tree the Vault Sync feature commits to.',
			'web.serverSettings.fields.personalPrefix.label' => 'Personal prefix',
			'web.serverSettings.fields.personalPrefix.hint' => 'Folder name used for personal notes when auto-deriving paths. Default "personal".',
			'web.serverSettings.fields.projectsPrefix.label' => 'Projects prefix',
			'web.serverSettings.fields.projectsPrefix.hint' => 'Folder name used for project notes. Default "projects".',
			'web.serverSettings.fields.registryRoot.label' => 'Registry root',
			'web.serverSettings.fields.registryRoot.hint' => 'Directory holding MCP server JSON definitions. Defaults to <vault>/mcp.',
			'web.serverSettings.fields.secretsFile.label' => 'Secrets file',
			'web.serverSettings.fields.secretsFile.hint' => 'key=value file substituted into MCP server commands at spawn time.',
			'web.serverSettings.fields.memoryBackend.label' => 'Embedder backend',
			'web.serverSettings.fields.memoryBackend.hint' => '"auto" / "bm25" use the cgo-free pure-Go keyword path. "http" calls any OpenAI-compatible /v1/embeddings (ollama / OpenAI / LocalAI). "local" runs an ONNX sentence-transformer in-process — requires a binary built with `-tags local_onnx`.',
			'web.serverSettings.fields.memoryStore.label' => 'Store',
			'web.serverSettings.fields.memoryStore.hint' => '"pgvector" reuses opendray\'s existing PG with the vector extension; only option in v1.',
			'web.serverSettings.fields.memoryTopK.label' => 'Default top-K',
			'web.serverSettings.fields.memoryTopK.hint' => 'How many hits memory_search returns when the agent doesn\'t specify. Empty = 5.',
			'web.serverSettings.fields.memoryThreshold.label' => 'Similarity threshold',
			'web.serverSettings.fields.memoryThreshold.hint' => 'Hits below this score are dropped. Empty = 0.1 (permissive — BM25 sparse vectors rarely break 0.5).',
			'web.serverSettings.fields.memoryScope.label' => 'Default scope',
			'web.serverSettings.fields.memoryScope.hint' => 'What memory_store uses when the agent doesn\'t specify. "project" (recommended) groups by cwd; "session" isolates per session; "global" shares across cwds.',
			'web.serverSettings.fields.memoryBaseUrl.label' => 'Base URL',
			'web.serverSettings.fields.memoryBaseUrl.hint' => 'e.g. "http://localhost:11434/v1" for ollama, "https://api.openai.com/v1" for OpenAI.',
			'web.serverSettings.fields.memoryModel.label' => 'Model',
			'web.serverSettings.fields.memoryModel.hint' => 'e.g. "nomic-embed-text" for ollama, "text-embedding-3-small" for OpenAI.',
			'web.serverSettings.fields.memoryApiKey.label' => 'API key',
			'web.serverSettings.fields.memoryApiKey.hint' => 'Empty for ollama / local servers. Required for OpenAI / Voyage / hosted services.',
			'web.serverSettings.fields.memoryLocalModel.label' => 'Model name',
			'web.serverSettings.fields.memoryLocalModel.hint' => 'Cosmetic — appears in logs / Inspector. e.g. "bge-m3", "bge-small-en-v1.5".',
			'web.serverSettings.fields.memoryLibraryPath.label' => 'Library path',
			'web.serverSettings.fields.memoryLibraryPath.hint' => 'Directory holding libonnxruntime.dylib (macOS) / libonnxruntime.so (Linux). After `brew install onnxruntime`, that\'s /opt/homebrew/opt/onnxruntime/lib.',
			'web.serverSettings.fields.memoryModelPath.label' => 'Model path',
			'web.serverSettings.fields.memoryModelPath.hint' => 'Absolute path to the .onnx weights. Download from HuggingFace, e.g. Xenova/bge-m3 or Xenova/bge-small-en-v1.5.',
			'web.serverSettings.fields.memoryTokenizerPath.label' => 'Tokenizer path',
			'web.serverSettings.fields.memoryTokenizerPath.hint' => 'Absolute path to tokenizer.json (HuggingFace standard format) — usually right next to the model.',
			'web.serverSettings.fields.memoryMaxSeqLen.label' => 'Max sequence length',
			'web.serverSettings.fields.memoryMaxSeqLen.hint' => 'Tokens beyond this are truncated. bge-m3 default is 512. Empty = 512.',
			'web.serverSettings.fields.claudeHistoryRoots.label' => 'History roots',
			'web.serverSettings.fields.claudeHistoryRoots.hint' => 'Directories scanned for Claude per-project JSONL transcripts. Empty = scan ~/.claude/projects + every ~/.claude-accounts/*/projects.',
			'web.serverSettings.fields.claudeAccountsDir.label' => 'Accounts directory',
			'web.serverSettings.fields.claudeAccountsDir.hint' => 'Root used for opendray-managed Claude account ConfigDirs. Default ~/.claude-accounts.',
			'web.serverSettings.fields.codexSessionsRoot.label' => 'Sessions root',
			'web.serverSettings.fields.codexSessionsRoot.hint' => 'Directory walked for Codex rollout JSONL files. Default ~/.codex/sessions.',
			'web.serverSettings.fields.geminiTmpRoot.label' => 'Tmp directory',
			'web.serverSettings.fields.geminiTmpRoot.hint' => 'Root holding Gemini per-project tmp folders. Default ~/.gemini/tmp.',
			'web.serverSettings.fields.geminiProjectsFile.label' => 'projects.json',
			'web.serverSettings.fields.geminiProjectsFile.hint' => 'Path to Gemini\'s cwd→short-name mapping file. Default ~/.gemini/projects.json.',
			'web.serverSettings.fields.backupLocalDir.label' => 'Local backup directory',
			'web.serverSettings.fields.backupLocalDir.hint' => 'Default root for the auto-created `local` target. Empty = ~/.opendray/backups. Restart required.',
			'web.serverSettings.fields.backupExportDir.label' => 'Export directory',
			'web.serverSettings.fields.backupExportDir.hint' => 'Where one-shot export zips are staged on disk. Empty = ~/.opendray/exports. Bundles auto-expire after 24h. Restart required.',
			'web.serverSettings.fields.backupPgDumpPath.label' => 'pg_dump path',
			'web.serverSettings.fields.backupPgDumpPath.hint' => 'Absolute path to pg_dump. Major version must be ≥ the server\'s. Empty = first pg_dump on PATH.',
			'web.serverSettings.fields.backupPgRestorePath.label' => 'pg_restore path',
			'web.serverSettings.fields.backupPgRestorePath.hint' => 'Absolute path to pg_restore for the /backups/restore flow. Same major-version rule.',
			'web.serverSettings.liveTail.heading' => 'Live tail',
			'web.serverSettings.liveTail.description' => 'In-memory ring buffer (last ~2,000 records). Resets on restart.',
			'web.serverSettings.memoryInspectorCard.heading' => 'Inspector',
			'web.serverSettings.memoryInspectorCard.description' => 'Browse, search and edit stored memories on the dedicated page.',
			'web.serverSettings.memoryInspectorCard.openButton' => 'Open Memory →',
			'web.serverSettings.localOnnxBanner' => 'Requires the binary to be compiled with <1>-tags local_onnx</1>. The standard build returns a clear stub error when this backend is selected. See <3>Memory → Local ONNX</3> tutorial for setup steps.',
			'web.serverSettings.stringList.noneDefault' => '(none — using built-in defaults)',
			'web.serverSettings.stringList.addPath' => 'Add path',
			'web.serverSettings.stringList.removeTitle' => 'Remove',
			'web.serverSettings.httpHelpers.autoDetected' => 'Auto-detected at startup',
			'web.serverSettings.httpHelpers.modelCount' => ({required Object count}) => '${count} model(s) — click to use',
			'web.serverSettings.httpHelpers.presets' => 'Presets:',
			'web.serverSettings.httpHelpers.testConnection' => 'Test connection',
			'web.serverSettings.httpHelpers.presetTip.ollama' => 'Local ollama daemon',
			'web.serverSettings.httpHelpers.presetTip.lmStudio' => 'LM Studio local server',
			'web.serverSettings.httpHelpers.presetTip.openai' => 'OpenAI cloud (needs API key)',
			'web.serverSettings.probe.unreachable' => ({required Object error}) => '✗ unreachable: ${error}',
			'web.serverSettings.probe.connectionFailed' => 'connection failed',
			'web.serverSettings.probe.reachable' => ({required Object detected, required Object total, required Object embedding}) => '✓ reachable ${detected}· ${total} model(s) total · ${embedding} embedding',
			'web.serverSettings.probe.modelMissing' => ({required Object model}) => '⚠ Configured model ${model} isn\'t in the list. Pick one of the embedding models below or fix the name.',
			'web.serverSettings.probe.embeddingModelsLabel' => 'embedding models:',
			'web.serverSettings.probe.moreModels' => ({required Object count}) => '+${count} more',
			'web.serverSettings.probe.noEmbeddingFound' => '⚠ No model name contains "embed". The endpoint might not have an embedding model loaded — check your local server.',
			'web.serverSettings.probe.configuredTitle' => 'Currently configured',
			'web.serverSettings.probe.applyTitle' => 'Click to apply',
			'web.serverSettings.backup.featureDisabledTitle' => 'Feature disabled',
			'web.serverSettings.backup.featureDisabledHint' => 'Set <1>OPENDRAY_BACKUP_ENABLED=1</1> + <3>OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;</3> in opendray\'s environment, then restart. The master passphrase is env-only — it never touches config.toml.',
			'web.serverSettings.backup.statusRowLabel' => 'Status',
			'web.serverSettings.backup.enabledHealthy' => 'enabled · healthy',
			'web.serverSettings.backup.enabledDegraded' => 'enabled · degraded',
			'web.serverSettings.backup.keyFingerprintLabel' => 'Key fingerprint',
			'web.serverSettings.backup.keyFingerprintHint' => 'record in Vaultwarden — losing it locks all prior backups',
			'web.serverSettings.backup.pgDumpLabel' => 'pg_dump',
			'web.serverSettings.backup.pgDumpUnavailable' => 'unavailable',
			'web.serverSettings.backup.pgRestoreLabel' => 'pg_restore',
			'web.serverSettings.backup.pgRestoreNotResolved' => '(not resolved)',
			'web.serverSettings.backup.openBackups' => 'Open Backups page →',
			'web.serverSettings.backup.openExport' => 'Open Export / Import →',
			'web.serverSettings.backup.whereDesc' => 'Each target is one place a backup blob can be written. opendray supports <1>local disk</1>, <3>SMB/CIFS</3> (Windows / NAS), <5>S3-compatible</5> (AWS, R2, B2, MinIO, Alibaba Cloud OSS, Tencent Cloud COS, ...), <7>WebDAV</7> (Nextcloud, Synology, Jianguoyun), <9>SFTP</9>, plus an <11>rclone</11> passthrough that taps into 70+ extra backends (Google Drive, OneDrive, Dropbox, Baidu Pan, Aliyun Drive, ...).',
			'web.serverSettings.backup.loading' => 'Loading…',
			'web.serverSettings.backup.noTargets' => 'No targets yet. Add one to start backing up.',
			'web.serverSettings.backup.addTarget' => 'Add target',
			'web.serverSettings.backup.noSchedulesHint' => 'No recurring schedules. Add one on <1>/backups → Schedules</1> to take backups automatically.',
			'web.serverSettings.backup.scheduleHeaders.schedule' => 'Schedule',
			'web.serverSettings.backup.scheduleHeaders.target' => 'Target',
			'web.serverSettings.backup.scheduleHeaders.cadence' => 'Cadence',
			'web.serverSettings.backup.scheduleHeaders.keep' => 'Keep',
			'web.serverSettings.backup.scheduleHeaders.state' => 'State',
			'web.serverSettings.backup.every' => ({required Object interval}) => 'every ${interval}',
			_ => null,
		} ?? switch (path) {
			'web.serverSettings.backup.backupsKeep' => ({required Object count}) => '${count} backups',
			'web.serverSettings.backup.stateEnabled' => 'enabled',
			'web.serverSettings.backup.statePaused' => 'paused',
			'web.serverSettings.backup.manageSchedules' => 'Manage on /backups → Schedules →',
			'web.serverSettings.backup.whatsInsideDesc' => 'Each backup is a <1>pg_dump --format=custom</1> of every opendray table (sessions, integrations, memories, audit_log, etc.) plus a <3>manifest.json</3> and (optionally) the live <5>config.toml</5>. Open the "What\'s in a backup?" panel on the <7>Backups page</7> to see the live inventory with row counts.',
			'web.serverSettings.backup.advancedToggle' => 'Advanced (paths & client binaries) — restart required',
			'web.serverSettings.targetRow.on' => 'on',
			'web.serverSettings.targetRow.off' => 'off',
			'web.serverSettings.targetRow.test' => 'Test',
			'web.serverSettings.targetRow.testing' => 'Testing…',
			'web.serverSettings.targetRow.delete' => 'Delete',
			'web.serverSettings.targetRow.connectionOk' => ({required Object id}) => '${id}: connection OK',
			'web.serverSettings.targetRow.connectionFailedTitle' => 'Connection failed',
			'web.serverSettings.targetRow.testFailedTitle' => 'Test failed',
			'web.serverSettings.targetRow.deleteConfirm' => ({required Object id}) => 'Delete target "${id}"? Schedules referencing it will block the delete.',
			'web.serverSettings.targetRow.deleteSuccess' => 'Target deleted',
			'web.serverSettings.targetRow.deleteFailedTitle' => 'Delete failed',
			'web.serverSettings.targetRow.unknownError' => 'Unknown error',
			'web.settings.title' => 'Settings',
			'web.settings.subtitle' => 'Workspace, account, and gateway config.',
			'web.settings.groups.workspace' => 'Workspace',
			'web.settings.groups.server' => 'Server',
			'web.settings.groups.system' => 'System',
			'web.settings.items.appearance' => 'Appearance',
			'web.settings.items.font' => 'Font size',
			'web.settings.items.account' => 'Account',
			'web.settings.items.status' => 'Status',
			'web.settings.items.about' => 'About',
			'web.settings.health.connecting' => 'connecting…',
			'web.settings.health.dbOk' => 'db ok',
			'web.settings.health.dbDown' => 'db down',
			'web.settings.breadcrumb.server' => 'Server',
			'web.settings.appearance.title' => 'Appearance',
			'web.settings.appearance.description' => 'Choose how opendray looks.',
			'web.settings.appearance.options.light' => 'Light',
			'web.settings.appearance.options.lightDesc' => 'Always light',
			'web.settings.appearance.options.dark' => 'Dark',
			'web.settings.appearance.options.darkDesc' => 'Always dark',
			'web.settings.appearance.options.system' => 'System',
			'web.settings.appearance.options.systemDesc' => 'Follow the OS setting',
			'web.settings.font.title' => 'Font size',
			'web.settings.font.description' => 'Scales the entire interface. Persisted per browser.',
			'web.settings.font.options.compact' => 'Compact',
			'web.settings.font.options.kDefault' => 'Default',
			'web.settings.font.options.comfy' => 'Comfy',
			'web.settings.font.options.large' => 'Large',
			'web.settings.account.title' => 'Account',
			'web.settings.account.description' => 'Operator and current bearer token.',
			'web.settings.account.username' => 'Username',
			'web.settings.account.tokenExpires' => 'Token expires',
			'web.settings.account.changeCredentials' => 'Change credentials',
			'web.settings.changeCredentials.title' => 'Change credentials',
			'web.settings.changeCredentials.description' => 'Verify your current password, then pick new credentials. All other signed-in sessions will be revoked.',
			'web.settings.changeCredentials.currentPassword' => 'Current password',
			'web.settings.changeCredentials.newUsername' => 'New username',
			'web.settings.changeCredentials.newPassword' => 'New password',
			'web.settings.changeCredentials.newPasswordHint' => 'At least 8 characters.',
			'web.settings.changeCredentials.confirm' => 'Confirm new password',
			'web.settings.changeCredentials.errorTooShort' => 'New password must be at least 8 characters.',
			'web.settings.changeCredentials.errorMismatch' => 'New password and confirmation don\'t match.',
			'web.settings.changeCredentials.errorWrongPassword' => 'Current password is wrong.',
			'web.settings.changeCredentials.cancel' => 'Cancel',
			'web.settings.changeCredentials.update' => 'Update',
			'web.settings.changeCredentials.saving' => 'Saving…',
			'web.settings.system.title' => 'System status',
			'web.settings.system.description' => 'Live status from the gateway\'s /health endpoint.',
			'web.settings.system.status' => 'Status',
			'web.settings.system.version' => 'Version',
			'web.settings.system.uptime' => 'Uptime',
			'web.settings.system.database' => 'Database',
			'web.settings.system.reachable' => 'reachable',
			'web.settings.system.unreachable' => 'unreachable',
			'web.settings.about.title' => 'About',
			'web.settings.about.description' => 'opendray v2 — the multiplexer + integration gateway for AI agent CLIs. Source under Apache 2.0.',
			'web.logViewer.filterPlaceholder' => 'Filter…',
			'web.logViewer.debugTooltip' => 'Debug count',
			'web.logViewer.infoTooltip' => 'Info count',
			'web.logViewer.warnTooltip' => 'Warn count',
			'web.logViewer.errorTooltip' => 'Error count',
			'web.logViewer.streaming' => 'Streaming',
			'web.logViewer.disconnected' => 'Disconnected',
			'web.logViewer.live' => 'live',
			'web.logViewer.offline' => 'offline',
			'web.logViewer.pauseTooltip' => 'Pause auto-scroll',
			'web.logViewer.resumeTooltip' => 'Resume auto-scroll',
			'web.logViewer.clearTooltip' => 'Clear local view (server ring untouched)',
			'web.logViewer.downloadTooltip' => 'Download full ring as .log file',
			'web.logViewer.emptyWaiting' => 'Waiting for log records…',
			'web.logViewer.emptyFiltered' => ({required Object query}) => 'No records match "${query}"',
			'web.pathInput.testButton' => 'Test',
			'web.pathInput.testTooltip' => 'Resolve and check this path',
			'web.pathInput.notFound' => 'not found ·',
			'web.pathInput.childrenSuffix' => 'children',
			'web.pathInput.expectedDirectory' => '· expected directory',
			'web.memoryAmbient.header.title' => 'Ambient memory — auto-capture & inject',
			'web.memoryAmbient.header.body' => 'opendray polls every live agent session every 10 seconds, extracts durable facts via a configurable LLM, and dedups before storing them in the shared memory pool. Configure which LLM does the extraction (Provider), when extraction fires (Capture rule), and what — if anything — gets prepended to the agent\'s system prompt at spawn (Injection profile).',
			'web.memoryAmbient.loading' => 'Loading…',
			'web.memoryAmbient.providers.title' => 'Summarizer providers',
			'web.memoryAmbient.providers.addButton' => 'Add provider',
			'web.memoryAmbient.providers.intro' => 'At least one enabled provider is required for capture to actually fire. Local options (Ollama, LM Studio, Integration) keep your transcripts off external networks.',
			'web.memoryAmbient.providers.empty' => 'No providers configured yet.',
			'web.memoryAmbient.providers.row.defaultBadge' => '★ default',
			'web.memoryAmbient.providers.row.makeDefault' => 'Make default',
			'web.memoryAmbient.providers.row.test' => 'Test',
			'web.memoryAmbient.providers.row.testing' => 'Testing…',
			'web.memoryAmbient.providers.row.delete' => 'Delete',
			'web.memoryAmbient.providers.row.testOk' => ({required Object name}) => '${name}: connection OK',
			'web.memoryAmbient.providers.row.testFailedToast' => 'Test failed',
			'web.memoryAmbient.providers.row.deleteConfirm' => ({required Object name}) => 'Delete provider "${name}"?',
			'web.memoryAmbient.providers.row.deletedToast' => 'Provider deleted',
			'web.memoryAmbient.providers.row.deleteFailedToast' => 'Delete failed',
			'web.memoryAmbient.providers.row.updateFailedToast' => 'Update failed',
			'web.memoryAmbient.providers.row.madeDefaultToast' => ({required Object name}) => '${name} is now the default',
			'web.memoryAmbient.providers.dialog.title' => 'Add summarizer provider',
			'web.memoryAmbient.providers.dialog.kindLabel' => 'Kind',
			'web.memoryAmbient.providers.dialog.nameLabel' => 'Name',
			'web.memoryAmbient.providers.dialog.namePlaceholder' => 'e.g. lmstudio-qwen',
			'web.memoryAmbient.providers.dialog.modelLabel' => 'Model',
			'web.memoryAmbient.providers.dialog.baseUrlLabel' => 'Base URL',
			'web.memoryAmbient.providers.dialog.integrationNote' => 'Integration providers resolve their base URL from a registered integration. Configure that under Integrations first; advanced wiring (extra_config) is DB-only in this release.',
			'web.memoryAmbient.providers.dialog.apiKeyLabel' => 'API key',
			'web.memoryAmbient.providers.dialog.apiKeyHint' => 'Stored encrypted (AES-GCM with the backup master passphrase). Never echoed back; only the fingerprint is shown after save.',
			'web.memoryAmbient.providers.dialog.makeDefaultLabel' => 'Make this the default provider',
			'web.memoryAmbient.providers.dialog.create' => 'Create',
			'web.memoryAmbient.providers.dialog.nameRequiredToast' => 'Name is required',
			'web.memoryAmbient.providers.dialog.createdToast' => ({required Object name}) => 'Provider ${name} created',
			'web.memoryAmbient.providers.dialog.createFailedToast' => 'Create failed',
			'web.memoryAmbient.rules.title' => 'Capture rules',
			'web.memoryAmbient.rules.addButton' => 'Add rule',
			'web.memoryAmbient.rules.intro' => 'Each rule says "when this trigger fires, summarize new transcript messages and store the durable facts." Per-session rules override the global default. v1 ships 4 trigger kinds.',
			'web.memoryAmbient.rules.empty' => 'No capture rules yet. Add one to enable auto-capture.',
			'web.memoryAmbient.rules.row.globalDefault' => 'global default',
			'web.memoryAmbient.rules.row.scopeLabel' => 'scope:',
			'web.memoryAmbient.rules.row.dedupLabel' => 'dedup:',
			'web.memoryAmbient.rules.row.runNow' => 'Run now',
			'web.memoryAmbient.rules.row.running' => 'Running…',
			'web.memoryAmbient.rules.row.delete' => 'Delete',
			'web.memoryAmbient.rules.row.firedToast' => ({required Object sessions}) => 'Rule fired across ${sessions} session(s)',
			'web.memoryAmbient.rules.row.runNowFailedToast' => 'Run-now failed',
			'web.memoryAmbient.rules.row.deleteConfirm' => ({required Object name}) => 'Delete rule "${name}"?',
			'web.memoryAmbient.rules.row.deletedToast' => 'Rule deleted',
			'web.memoryAmbient.rules.row.deleteFailedToast' => 'Delete failed',
			'web.memoryAmbient.rules.row.summary.afterMessages' => ({required Object n}) => 'every ${n} messages',
			'web.memoryAmbient.rules.row.summary.onIdle' => ({required Object seconds}) => 'idle ≥ ${seconds}s',
			'web.memoryAmbient.rules.row.summary.kChars' => ({required Object k}) => '≥ ${k} chars',
			'web.memoryAmbient.rules.row.summary.manual' => 'manual only',
			'web.memoryAmbient.rules.dialog.title' => 'Add capture rule',
			'web.memoryAmbient.rules.dialog.nameLabel' => 'Name',
			'web.memoryAmbient.rules.dialog.triggerLabel' => 'Trigger',
			'web.memoryAmbient.rules.dialog.nLabel' => 'N (messages)',
			'web.memoryAmbient.rules.dialog.idleLabel' => 'Idle seconds',
			'web.memoryAmbient.rules.dialog.kLabel' => 'K (characters)',
			'web.memoryAmbient.rules.dialog.scopeLabel' => 'Target scope',
			'web.memoryAmbient.rules.dialog.scopeSession' => 'session',
			'web.memoryAmbient.rules.dialog.scopeProject' => 'project (recommended)',
			'web.memoryAmbient.rules.dialog.scopeGlobal' => 'global',
			'web.memoryAmbient.rules.dialog.dedupLabel' => 'Dedup threshold (0.0 – 1.0)',
			'web.memoryAmbient.rules.dialog.dedupHint' => 'Higher = stricter de-duplication. 0.85 is the recommended sweet spot.',
			'web.memoryAmbient.rules.dialog.create' => 'Create',
			'web.memoryAmbient.rules.dialog.nameRequiredToast' => 'Name is required',
			'web.memoryAmbient.rules.dialog.createdToast' => ({required Object name}) => 'Rule ${name} created',
			'web.memoryAmbient.rules.dialog.createFailedToast' => 'Create failed',
			'web.memoryAmbient.profiles.title' => 'Injection profiles',
			'web.memoryAmbient.profiles.addButton' => 'Add profile',
			'web.memoryAmbient.profiles.intro' => 'At spawn time opendray prepends a markdown banner of recent project memories to the agent\'s system prompt — IF a profile is configured. Without a profile, the model still uses memory_search on demand.',
			'web.memoryAmbient.profiles.empty' => 'No injection profile. Memories are not auto-injected at spawn — model still uses memory_search.',
			'web.memoryAmbient.profiles.row.globalDefault' => 'global default',
			'web.memoryAmbient.profiles.row.delete' => 'Delete',
			'web.memoryAmbient.profiles.row.deleteConfirm' => 'Delete this injection profile?',
			'web.memoryAmbient.profiles.row.deletedToast' => 'Profile deleted',
			'web.memoryAmbient.profiles.row.deleteFailedToast' => 'Delete failed',
			'web.memoryAmbient.profiles.dialog.title' => 'Add injection profile',
			'web.memoryAmbient.profiles.dialog.strategyLabel' => 'Strategy',
			'web.memoryAmbient.profiles.dialog.kLabel' => 'K (top memories to inject)',
			'web.memoryAmbient.profiles.dialog.hint' => 'One profile per session_id (or global default). Per-session profiles can be added later via API; UI currently only manages the global default.',
			'web.memoryAmbient.profiles.dialog.create' => 'Create',
			'web.memoryAmbient.profiles.dialog.createdToast' => 'Profile created',
			'web.memoryAmbient.profiles.dialog.createFailedToast' => 'Create failed',
			'web.memoryAmbient.cost.title' => 'Token cost (all-time)',
			'web.memoryAmbient.cost.intro' => 'Per-provider summary aggregated from <1>memory_summarizer_calls</1>. Local providers (Ollama, LM Studio, Integration) are priced as \$0 — operator owns hardware cost.',
			'web.memoryAmbient.cost.empty' => 'No enabled providers — no cost data.',
			'web.memoryAmbient.cost.columns.provider' => 'Provider',
			'web.memoryAmbient.cost.columns.calls' => 'Calls',
			'web.memoryAmbient.cost.columns.inTokens' => 'In tokens',
			'web.memoryAmbient.cost.columns.outTokens' => 'Out tokens',
			'web.memoryAmbient.cost.columns.usdEst' => 'USD est.',
			'web.noteEditor.loading' => 'Loading…',
			'web.noteEditor.source' => 'Source',
			'web.noteEditor.preview' => 'Preview',
			'web.noteEditor.tagTitle' => ({required Object tag}) => 'tag #${tag}',
			'web.noteEditor.emptyNote' => 'Empty note. Switch to Source to start writing.',
			'web.noteEditor.saveFailedToast' => 'Save failed',
			'web.noteEditor.status.saveFailed' => 'save failed',
			'web.noteEditor.status.saving' => 'saving…',
			'web.noteEditor.status.unsaved' => 'unsaved',
			'web.noteEditor.status.newNote' => 'new note',
			'web.noteEditor.status.saved' => 'saved',
			'web.export.title' => 'Export data',
			'web.export.subtitle' => 'Take a one-shot zip bundle of selected logical entities. Bundles are kept on the server for 24 hours, then automatically reaped.',
			'web.export.backToBackups' => '← Backups',
			'web.export.sections.export' => 'Export',
			'web.export.sections.import' => 'Import',
			'web.export.form.scope' => 'Scope',
			'web.export.form.memories' => 'Memories',
			'web.export.form.memoriesHint' => 'Cross-CLI persistent memory rows (text + scope + metadata). Embedding vectors are omitted; importer re-embeds.',
			'web.export.form.integrations' => 'Integrations',
			'web.export.form.customTasks' => 'Custom tasks',
			'web.export.form.customTasksHint' => 'Operator-defined tasks shown in the Inspector\'s Tasks tab.',
			'web.export.form.integrationOptions.none' => 'None',
			'web.export.form.integrationOptions.noneHint' => 'Skip the integrations table entirely.',
			'web.export.form.integrationOptions.metadata' => 'Metadata only (recommended)',
			'web.export.form.integrationOptions.metadataHint' => 'ID, name, route prefix, scopes — no API key material.',
			'web.export.form.integrationOptions.plaintext' => 'Include plaintext API keys',
			'web.export.form.integrationOptions.plaintextHint' => 'v1 bcrypt-only: no recoverable plaintext exists. Manifest documents this; nothing leaks.',
			'web.export.form.confirmWarning' => 'Type <1>I understand</1> to confirm. opendray currently stores only bcrypt hashes — selecting plaintext does NOT export any plaintext (the feature is reserved for a future release that keeps plaintext caches).',
			'web.export.form.confirmPlaceholder' => 'I understand',
			'web.export.form.confirmSentinel' => 'i understand',
			'web.export.form.footnote' => 'Audit logs and session transcripts are out of scope — covered by /backups (operator dump) instead.',
			'web.export.form.building' => 'Building…',
			'web.export.form.create' => 'Create export',
			'web.export.form.readyToast' => 'Export ready',
			'web.export.form.readyDescription' => ({required Object bytes}) => '${bytes} bytes',
			'web.export.form.failedToast' => 'Export failed',
			'web.export.history.loading' => 'Loading…',
			'web.export.history.empty' => 'No exports yet. Use the form above to create one.',
			'web.export.history.title' => 'History',
			'web.export.history.columns.id' => 'ID',
			'web.export.history.columns.status' => 'Status',
			'web.export.history.columns.scope' => 'Scope',
			'web.export.history.columns.size' => 'Size',
			'web.export.history.columns.expires' => 'Expires',
			'web.export.history.columns.actions' => 'Actions',
			'web.export.history.download' => 'Download',
			'web.export.history.deleteTooltip' => 'Delete',
			'web.export.history.listFailedToast' => 'Failed to list exports',
			'web.export.history.downloadFailedToast' => 'Download failed',
			'web.export.history.noTokenToast' => 'No download token (expired?)',
			'web.export.history.deleteConfirm' => ({required Object id}) => 'Delete export ${id}?',
			'web.export.history.deletedToast' => 'Export deleted',
			'web.export.history.deleteFailedToast' => 'Delete failed',
			'web.export.history.scopeEmpty' => '(empty)',
			'web.export.import.intro' => 'Replay an export bundle (zip) into the live database. Conflicts (matching id, or unique route_prefix for integrations) are <1>skipped</1> by default. Memories are tagged <3>embedder=imported_v1</3> and need a re-embed pass before search returns them; trigger re-embed under <5>Memory → Maintenance</5>. Integrations are imported with <7>enabled=false</7> and a non-bcrypt placeholder key — operator must rotate before use.',
			'web.export.import.memoryLink' => 'Memory → Maintenance',
			'web.export.import.bundleLabel' => 'Bundle (.zip)',
			'web.export.import.memoriesLabel' => 'Memories',
			'web.export.import.integrationsLabel' => 'Integrations (metadata only — keys never imported)',
			'web.export.import.customTasksLabel' => 'Custom tasks',
			'web.export.import.importing' => 'Importing…',
			'web.export.import.importBundle' => 'Import bundle',
			'web.export.import.pickFileToast' => 'Pick a bundle file first',
			'web.export.import.doneToast' => 'Import done',
			'web.export.import.finishedWithErrors' => 'Import finished with errors',
			'web.export.import.failedToast' => 'Import failed',
			'web.export.import.summaryCard.memories' => 'Memories',
			'web.export.import.summaryCard.integrations' => 'Integrations',
			'web.export.import.summaryCard.customTasks' => 'Custom tasks',
			'web.export.import.summaryCard.created' => 'created',
			'web.export.import.summaryCard.skipped' => 'skipped',
			'web.export.import.summaryCard.failed' => 'failed',
			'web.export.imports.loading' => 'Loading…',
			'web.export.imports.empty' => 'No imports yet.',
			'web.export.imports.title' => 'History',
			'web.export.imports.columns.id' => 'ID',
			'web.export.imports.columns.status' => 'Status',
			'web.export.imports.columns.source' => 'Source',
			'web.export.imports.columns.counts' => 'Counts',
			'web.export.imports.columns.when' => 'When',
			'web.export.imports.noneCounts' => '(none)',
			'web.export.imports.listFailedToast' => 'Failed to list imports',
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
			'more.items.dataExport.title' => 'Data export & import',
			'more.items.dataExport.subtitle' => 'User-level data bundles (memories / integrations / custom tasks)',
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
			'sessions.terminal.keyboard.enter' => 'Enter',
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
			'sessions.inspector.notes.loadFailedApi' => ({required Object error}) => 'Load failed: ${error}',
			'sessions.inspector.notes.loadFailedGeneric' => ({required Object error}) => 'Load failed: ${error}',
			'sessions.inspector.notes.saveFailedApi' => ({required Object error}) => 'Save failed: ${error}',
			'sessions.inspector.notes.saveFailedGeneric' => ({required Object error}) => 'Save failed: ${error}',
			'sessions.inspector.notes.insertFailedApi' => ({required Object error}) => 'Insert failed: ${error}',
			'sessions.inspector.notes.insertFailedGeneric' => ({required Object error}) => 'Insert failed: ${error}',
			'sessions.inspector.notes.createFailedApi' => ({required Object error}) => 'Create failed: ${error}',
			'sessions.inspector.notes.createFailedGeneric' => ({required Object error}) => 'Create failed: ${error}',
			'sessions.inspector.notes.personalHint' => 'Personal scratchpad — auto-saves as you type. AI agents do not write here.',
			'sessions.inspector.notes.projectDocsHint' => 'Architecture / spec / decisions / plan / retros — typically authored or maintained by an agent.',
			'sessions.inspector.notes.mappingCleared' => 'Mapping cleared — using default',
			'sessions.inspector.notes.mappedTo' => ({required Object path}) => 'Mapped to ${path}',
			'sessions.inspector.notes.cancelTooltip' => 'Cancel',
			'sessions.inspector.notes.newDocTooltip' => 'New doc',
			'sessions.inspector.notes.noProjectMapping' => 'Could not resolve a project mapping for this session. Check that the gateway has a notes vault configured and that the session cwd is set.',
			'sessions.inspector.notes.emptyProjectDocs' => 'No project docs yet. Tap + to create one, or let an AI agent generate from a prompt.',
			'sessions.inspector.notes.emptyFilterMatch' => ({required Object query}) => 'No matches for "${query}".',
			'sessions.inspector.notes.locationDialogHelp' => 'Pin this session\'s cwd to a specific folder under your notes vault. Leave blank to reset.',
			'sessions.inspector.notes.sessionCwd' => 'Session cwd',
			'sessions.inspector.notes.projectDocsPath' => 'Vault-relative project docs path',
			'sessions.inspector.notes.locationStoredHint' => 'Stored in <vault>/.opendray-projects.json — git-syncs with the rest of the vault.',
			'sessions.inspector.notes.pinnedHint' => ({required Object path, required Object defaultPath}) => 'Pinned to ${path}/ (overrides ${defaultPath}). AI agents author docs here too.',
			'sessions.inspector.notes.noProjectMapping2' => '(no project mapping)',
			'sessions.inspector.notes.clearOverride' => 'Clear override',
			'sessions.inspector.notes.save' => 'Save',
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
			'sessions.spawnSheet.bypass.labelCodex' => 'Bypass approvals & sandbox',
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
			_ => null,
		} ?? switch (path) {
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
			'providers.accounts.enable' => 'Enable',
			'providers.accounts.disable' => 'Disable',
			'providers.accounts.deleteLabel' => 'Delete',
			'providers.accounts.deleteBody' => 'Removes the account and its stored OAuth token. Sessions already using this account stay running but reauth will fail.',
			'providers.accounts.deletedSnack' => ({required Object name}) => 'Deleted ${name}.',
			'providers.accounts.importSyncedSnack' => 'Already in sync — gateway has no new accounts.',
			'providers.accounts.importedSnackOne' => ({required Object n}) => 'Imported ${n} account.',
			'providers.accounts.importedSnackOther' => ({required Object n}) => 'Imported ${n} accounts.',
			'providers.accounts.importing' => 'Syncing…',
			'providers.accounts.importLocal' => 'Import local',
			'providers.accounts.addHint' => 'Adding a new account is gateway-host only.',
			'providers.accounts.addBody' => 'The new directory shows up here automatically. See the docs for OAuth flow steps.',
			'providers.accounts.loadFailed' => ({required Object error}) => 'Failed to load accounts: ${error}',
			'providers.accounts.intro' => 'Sessions spawned with the Claude provider pick from these accounts (or fall back to env).',
			'providers.accounts.enabledSnack' => ({required Object name}) => '${name} enabled.',
			'providers.accounts.disabledSnack' => ({required Object name}) => '${name} disabled.',
			'providers.accounts.renamedSnack' => ({required Object name}) => 'Renamed to ${name}.',
			'providers.configFallbackTitle' => 'Provider config',
			'providers.saving' => 'Saving…',
			'providers.save' => 'Save',
			'providers.configLoadFailed' => 'Failed to load provider',
			'providers.argsHelper' => 'Whitespace-separated CLI args.',
			'providers.listEmptyHeadline' => 'No providers loaded.',
			'providers.listEmptyBody' => 'The gateway resolves providers from its plugin directory at startup. Check the logs if you expect one.',
			'providers.listLoadFailed' => 'Failed to load providers',
			'providers.cliSectionHeader' => 'CLI providers',
			'providers.enabledSnack' => ({required Object name}) => '${name} enabled.',
			'providers.disabledSnack' => ({required Object name}) => '${name} disabled.',
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
			'integrations.deleteBody' => 'Removes the registration and revokes the API key. In-flight requests using the old key will start failing.',
			'integrations.rotateBody' => ({required Object name}) => 'Generates a new API key for ${name} and immediately invalidates the old one.',
			'integrations.appBarFallback' => 'Integration',
			'integrations.tooltipMore' => 'More',
			'integrations.tooltipReadOnly' => 'System integration — read-only',
			'integrations.kvRoutePrefix' => 'Route prefix',
			'integrations.kvBaseUrl' => 'Base URL',
			'integrations.kvScopes' => 'Scopes',
			'integrations.kvVersion' => 'Version',
			'integrations.kvLastHealthPing' => 'Last health ping',
			'integrations.kvCreated' => 'Created',
			'integrations.kvKeyRotated' => 'Key rotated',
			'integrations.detailLoadFailed' => ({required Object error}) => 'Failed to load integration: ${error}',
			'integrations.callsLoadFailed' => 'Failed to load calls',
			'integrations.noMatchingCalls' => 'No matching calls in the log yet.',
			'integrations.directionAll' => 'All',
			'integrations.directionInbound' => 'Inbound',
			'integrations.directionOutbound' => 'Outbound',
			'integrations.form.validateRequired' => 'Name, base URL, and route prefix are required.',
			'integrations.form.fieldName' => 'Name',
			'integrations.form.fieldNameHint' => 'My Bot',
			'integrations.form.fieldBaseUrl' => 'Base URL',
			'integrations.form.fieldRoutePrefix' => 'Route prefix',
			'integrations.form.routePrefixHelper' => 'Reachable as /api/v1/<prefix>/...',
			'integrations.form.fieldScopes' => 'Scopes (optional)',
			'integrations.form.scopesHelper' => 'Comma-separated. Empty = server defaults.',
			'integrations.form.fieldVersion' => 'Version (optional)',
			'integrations.form.validateBaseUrl' => 'Base URL is required.',
			'integrations.form.editFieldScopes' => 'Scopes',
			'integrations.form.editScopesHelper' => 'Comma-separated.',
			'integrations.form.editFieldVersion' => 'Version',
			'integrations.form.apiKeyWarn' => 'You won\'t see this key again.',
			'integrations.form.copyCopied' => 'Copied',
			'integrations.form.copyCopy' => 'Copy',
			'integrations.emptyState' => 'Register from the web admin: Integrations → New.',
			'integrations.sectionRegistered' => 'Registered',
			'integrations.sectionSystem' => 'System',
			'integrations.listLoadFailed' => 'Failed to load integrations',
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
			'memoryWorkers.tasks.planDrift.label' => 'Plan drift detector',
			'memoryWorkers.tasks.planDrift.description' => 'After each session ends, checks whether the project plan needs updating and files a proposal. Fits an agent worker for richer reasoning.',
			'memoryWorkers.tasks.conflictDetector.label' => 'Cross-layer conflict detector',
			'memoryWorkers.tasks.conflictDetector.description' => 'Daily scan that finds contradictions between facts / plan / goal / journal. Higher-quality model = fewer false positives.',
			'memoryWorkers.tasks.capture.label' => 'Capture engine',
			'memoryWorkers.tasks.capture.description' => 'Per-trigger fact extraction from session transcripts. Agent mode gives noticeably better facts on long sessions; summarizer mode is cheap and local.',
			'memoryCleanup.title' => 'Memory cleanup',
			'memoryCleanup.approveFailed' => ({required Object error}) => 'Approve failed: ${error}',
			'memoryCleanup.rejectFailed' => ({required Object error}) => 'Reject failed: ${error}',
			'memoryCleanup.loadFailed' => ({required Object error}) => 'Failed to load: ${error}',
			'memoryCleanup.reject' => 'Reject',
			'project.title' => 'Project',
			'project.pickFirst' => 'Pick a project first.',
			'project.health.title' => ({required Object days}) => 'Memory health — last ${days} days',
			'project.health.subtitle' => 'Aggregate signals across both memory subsystems for this project.',
			'project.health.newFacts' => 'New facts',
			'project.health.newFactsHint' => ({required Object total}) => '${total} stored in total',
			'project.health.captureFires' => 'Capture fires',
			'project.health.captureFiresHint' => ({required Object stored, required Object deduped}) => '${stored} stored · ${deduped} deduped',
			'project.health.newJournal' => 'Journal entries',
			'project.health.newJournalHint' => ({required Object total}) => '${total} in total',
			'project.health.planAge' => 'Plan last updated',
			'project.health.planAgeHint' => ({required Object count}) => '${count} plan-drift proposal(s) pending',
			'project.health.planAgeHintNone' => 'No plan-drift proposals pending',
			'project.health.goalAge' => 'Goal last updated',
			'project.health.pending' => 'Pending proposals',
			'project.health.pendingHint' => ({required Object days}) => 'oldest ${days}d old',
			'project.health.topHit' => ({required Object hits}) => 'Top hit · ${hits} retrievals',
			'project.health.zeroHit' => ({required Object count}) => '${count} facts older than 7d with zero retrievals — candidates for cleanup.',
			'project.health.never' => 'never',
			'project.health.today' => 'today',
			'project.health.daysAgo' => ({required Object count}) => '${count}d ago',
			'project.conflicts.subtitle' => 'Contradictions the daily detector found between facts, plan, goal, and journal entries.',
			'project.conflicts.empty' => 'No pending conflicts. Tap Detect now for an on-demand sweep.',
			'project.conflicts.detectNow' => 'Detect now',
			'project.conflicts.detected' => ({required Object count}) => '${count} new conflict(s) found',
			'project.conflicts.accept' => 'Accept',
			'project.conflicts.dismiss' => 'Dismiss',
			'project.conflicts.deleteFact' => ({required Object side}) => 'Delete fact ${side}',
			'project.conflicts.deleteConfirmTitle' => ({required Object side}) => 'Delete fact ${side}?',
			'project.conflicts.deleteConfirmBody' => 'This permanently removes the fact and accepts the conflict. The other side stays as the surviving claim.',
			'project.conflicts.deleteWillDelete' => ({required Object side}) => 'Will delete (side ${side}):',
			'project.conflicts.deleteWillKeep' => ({required Object side}) => 'Will keep (side ${side}):',
			'project.conflicts.deleteNonFactOther' => ({required Object layer}) => '(${layer} entry — open the corresponding tab to inspect)',
			'project.conflicts.deleteLoading' => 'Loading fact text…',
			'project.conflicts.deleteFactLabel' => ({required Object side}) => 'Delete ${side}',
			'project.conflicts.deletedFact' => 'Fact deleted and conflict accepted',
			'project.conflicts.openPlanEditor' => 'Open plan editor',
			'project.conflicts.openGoalEditor' => 'Open goal editor',
			'project.conflicts.severity.low' => 'low',
			'project.conflicts.severity.medium' => 'medium',
			'project.conflicts.severity.high' => 'high',
			'project.journalPrune.title' => 'Prune stale journal entries',
			'project.journalPrune.subtitle' => ({required Object days}) => 'Older than ${days} days, no pending conflicts.',
			'project.journalPrune.daysLabel' => 'Older than (days):',
			'project.journalPrune.empty' => 'Nothing stale to prune.',
			'project.journalPrune.selectAll' => 'Select all',
			'project.journalPrune.deselectAll' => 'Deselect all',
			'project.journalPrune.deleteSelected' => ({required Object count}) => 'Delete (${count})',
			'project.journalPrune.deleted' => ({required Object count}) => '${count} entry/entries deleted',
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
			'backups.runConfirmBody' => 'Triggers a fresh dump against the local target. The job runs server-side; this list will refresh as it progresses.',
			'backups.run' => 'Run',
			'backups.runNow' => 'Run now',
			'backups.queueing' => 'Queueing…',
			'backups.queuedSnack' => ({required Object id}) => 'Backup queued (${id}). Watching for progress…',
			'backups.runFailedApi' => ({required Object error}) => 'Run failed: ${error}',
			'backups.runFailedGeneric' => ({required Object error}) => 'Run failed: ${error}',
			'backups.rowSucceededSnack' => ({required Object bytes}) => 'Backup succeeded (${bytes}).',
			'backups.rowFailedSnack' => ({required Object error}) => 'Backup failed: ${error}',
			'backups.unknownError' => 'unknown error',
			'backups.detailTitle' => 'Backup detail',
			'backups.deleteTitle' => 'Delete backup?',
			'backups.deleteBody' => ({required Object target}) => 'Removes the blob from ${target} and marks the row deleted in the index.',
			'backups.deletedSnack' => ({required Object id}) => 'Deleted ${id}.',
			'backups.deleteFailedApi' => ({required Object error}) => 'Delete failed: ${error}',
			'backups.deleteFailedGeneric' => ({required Object error}) => 'Delete failed: ${error}',
			'backups.menuSchedules' => 'Schedules',
			'backups.menuTargets' => 'Targets',
			'backups.kv.status' => 'Status',
			'backups.kv.target' => 'Target',
			'backups.kv.triggeredBy' => 'Triggered by',
			'backups.kv.started' => 'Started',
			'backups.kv.finished' => 'Finished',
			'backups.kv.size' => 'Size',
			'backups.kv.encrypted' => 'Encrypted',
			'backups.kv.targetPath' => 'Target path',
			'backups.kv.error' => 'Error',
			'backups.kv.yes' => 'yes',
			'backups.kv.no' => 'no',
			'backups.emptyMissingDeps.headline' => 'Backups can\'t run yet',
			'backups.emptyMissingDeps.body' => 'Install postgresql-client and restart opendray.',
			'backups.emptyNoTargets.headline' => 'No backup targets configured',
			'backups.emptyNoTargets.body' => 'Open the More menu → Targets to add a destination (local / S3 / SMB / SFTP / WebDAV / rclone). Then come back and tap "Run now".',
			'backups.emptyNoBackups.headline' => 'No backups yet',
			'backups.emptyNoBackups.body' => 'Tap "Run now" to take a fresh snapshot, or open Schedules to set up recurring runs.',
			'backups.restartToActivate' => 'Restart opendray to activate backups',
			'backups.passphraseSaved' => 'Your passphrase is saved. The gateway only loads it at startup, so changes only take effect after a restart.',
			'backups.keyFileLabel' => 'Key file',
			'backups.configuredViaLabel' => 'Configured via',
			'backups.wizard.title' => 'Set up backups',
			'backups.wizard.intro' => 'Choose a master passphrase. opendray uses it to encrypt every backup blob with AES-256-GCM. Lose the passphrase and you lose the data — there is no recovery.',
			'backups.wizard.saving' => 'Saving…',
			'backups.wizard.generateAndSave' => 'Generate and save',
			'backups.wizard.savePassphrase' => 'Save passphrase',
			'backups.wizard.generateHint' => 'Server generates a cryptographically random passphrase, you copy it to a password manager, then commit.',
			'backups.wizard.helperRecommended' => 'Recommended: 40+ chars from a password manager',
			'backups.wizard.saveNowHeader' => 'Save this passphrase NOW',
			'backups.wizard.saveNowBody' => 'This is shown ONCE. It will not be retrievable from opendray afterwards.',
			'backups.statusReady' => 'Backups ready',
			'backups.statusCannot' => 'Backups cannot run',
			'backups.overviewTargets' => 'Targets',
			'backups.overviewSchedules' => 'Schedules',
			'backups.overviewBackups' => 'Backups',
			'backups.failedToLoad' => 'Failed to load backups',
			'backups.envVarConfigured' => 'OPENDRAY_BACKUP_KEY env var',
			'backups.savedConfirmCheckbox' => 'I have saved this passphrase to my password manager',
			'backups.pgDumpMissing' => 'pg_dump is not on PATH. Install postgresql-client and restart opendray.',
			'backups.encryption.checkAgain' => 'Check again',
			'backups.encryption.generate' => 'Generate',
			'backups.encryption.paste' => 'Paste',
			'backups.encryption.random256bit' => '256-bit random key',
			'backups.encryption.passphraseLabel' => 'Your passphrase',
			'backups.encryption.passphraseHint' => 'At least 20 characters',
			'backups.encryption.passphraseCopied' => 'Passphrase copied to clipboard',
			'backups.restoreFromFile' => 'Restore from file',
			'backups.restore.title' => 'Restore from bundle',
			'backups.restore.subtitle' => 'Replay an encrypted .tar.gz.enc bundle into a Postgres database. The bundle is uploaded from this phone — pick a file produced by a prior backup.',
			'backups.restore.bundleLabel' => 'Bundle file (.tar.gz.enc)',
			'backups.restore.pickFile' => 'Pick file',
			'backups.restore.fileSelected' => ({required Object name, required Object size}) => '${name} · ${size}',
			'backups.restore.noFile' => 'No file selected',
			'backups.restore.targetDsnLabel' => 'Target Postgres DSN',
			'backups.restore.targetDsnHint' => 'Leave empty to restore into opendray\'s own DB.',
			'backups.restore.targetDsnPlaceholder' => 'postgres://user:pass@host:5432/dbname',
			'backups.restore.cleanLabel' => 'pg_restore --clean --if-exists',
			'backups.restore.cleanHint' => 'Drops existing objects before recreating them.',
			'backups.restore.auditNoteLabel' => 'Audit note (optional)',
			'backups.restore.auditNotePlaceholder' => 'e.g. recovering from #INC-481',
			'backups.restore.ownDbWarning' => 'Restoring into opendray\'s OWN database will rewrite the rows this gateway is currently serving. Type "I understand" to confirm.',
			'backups.restore.confirmPlaceholder' => 'Type "I understand"',
			'backups.restore.confirmSentinel' => 'I understand',
			'backups.restore.restoring' => 'Restoring…',
			'backups.restore.restore' => 'Restore',
			'backups.restore.succeededTitle' => 'Restore succeeded',
			'backups.restore.succeededBody' => ({required Object bytes, required Object id}) => 'Replayed ${bytes} from backup ${id}.',
			'backups.restore.failedTitle' => 'Restore failed',
			'backups.restore.pickFileToast' => 'Pick a bundle file first.',
			'backups.restore.outputTitle' => 'pg_restore output',
			'backups.restore.noPgRestoreOutput' => '(empty — restore completed silently)',
			'backups.restore.manifestTitle' => 'Manifest',
			'backups.restore.manifestBackupId' => 'Backup ID',
			'backups.restore.manifestVersion' => 'Manifest version',
			'backups.restore.manifestCreatedAt' => 'Created',
			'backups.restore.manifestPgVersion' => 'pg_version',
			'backups.restore.manifestOpendrayVersion' => 'opendray version',
			'backups.restore.fingerprint' => 'Key fingerprint',
			'backups.restore.fingerprintOk' => 'matched',
			'backups.restore.fingerprintMismatch' => 'MISMATCH',
			'backups.restore.encryptionAlgo' => 'Encryption',
			'backups.restore.bytesRead' => 'Bytes read',
			'backups.restore.targetDsnUsed' => 'Target DSN',
			'backups.restore.targetDsnSelfLabel' => '(opendray\'s own DB)',
			'backups.restore.done' => 'Done',
			'backups.inventory.title' => 'What\'s in a backup',
			'backups.inventory.summary' => ({required Object rows, required Object tables}) => '${rows} rows · ${tables} tables',
			'backups.inventory.description' => 'Live row counts from opendray\'s Postgres database. Backups capture every row below; binary artifacts on disk are not included.',
			'backups.inventory.rowsLabel' => 'rows',
			'backups.inventory.loadFailedToast' => 'Failed to load inventory',
			'backups.inventory.loading' => 'Loading…',
			'backups.inventory.tap' => 'Tap to expand',
			'backupTargets.title' => 'Backup targets',
			'backupTargets.newTarget' => 'New target',
			'backupTargets.testConnection' => 'Test connection',
			'backupTargets.editConfig' => 'Edit config',
			'backupTargets.viewRawConfig' => 'View raw config',
			'backupTargets.configDialogTitle' => ({required Object kind}) => '${kind} config',
			'backupTargets.deleteTitle' => 'Delete target?',
			'backupTargets.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'backupSchedules.title' => 'Backup schedules',
			'backupSchedules.newButton' => 'New',
			'backupSchedules.deleteTitle' => 'Delete schedule?',
			'backupSchedules.targetLabel' => 'Target',
			'backupSchedules.intervalLabel' => 'Interval',
			'backupSchedules.retentionLabel' => 'Retention (keep N most recent)',
			'backupSchedules.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'backupSchedules.noTargets' => 'No backup targets configured. Add one from the web admin or the Targets screen.',
			'backupSchedules.okMsgCreate' => 'Schedule created.',
			'backupSchedules.okMsgUpdate' => 'Schedule updated.',
			'backupSchedules.okMsgDelete' => 'Schedule deleted.',
			'backupSchedules.errorPrefixCreate' => 'Create failed',
			'backupSchedules.errorPrefixUpdate' => 'Update failed',
			'backupSchedules.errorPrefixDelete' => 'Delete failed',
			'backupSchedules.deleteBody' => ({required Object targetId}) => 'Removes the recurring spec for target ${targetId}. Existing backup blobs are not touched.',
			'backupSchedules.emptyList' => 'No schedules yet.\nTap "New" to create one.',
			'backupSchedules.validatePickTarget' => 'Pick a target.',
			'backupSchedules.validateInterval' => 'Interval must be > 0.',
			'backupSchedules.formTitleEdit' => 'Edit schedule',
			'backupSchedules.formTitleNew' => 'New schedule',
			'backupSchedules.saveButtonEdit' => 'Save',
			'backupSchedules.saveButtonNew' => 'Create',
			'backupSchedules.targetFixedHint' => 'Target is fixed once created.',
			'backupSchedules.enabledOn' => 'Scheduler will run this on cadence.',
			'backupSchedules.enabledOff' => 'Paused — no automatic runs until re-enabled.',
			'backupSchedules.loadFailedTitle' => 'Failed to load schedules',
			'backupSchedules.pausedBadge' => 'paused',
			'backupSchedules.everyInterval' => ({required Object interval}) => 'every ${interval}',
			'backupSchedules.keepRetention' => ({required Object n}) => '· keep ${n}',
			'backupSchedules.nextRun' => ({required Object when}) => '· next ${when}',
			'backupSchedules.lastRun' => ({required Object when}) => '· last ${when}',
			'backupTargetEditor.useHttps' => 'Use HTTPS',
			'backupTargetEditor.pathStyle' => 'Path-style addressing',
			'backupTargetEditor.pathStyleSubtitle' => 'Legacy / MinIO',
			'backupTargetEditor.kinds.local.label' => 'Local disk',
			'backupTargetEditor.kinds.local.description' => 'Folder on the machine running opendray',
			'backupTargetEditor.kinds.smb.label' => 'SMB share',
			'backupTargetEditor.kinds.smb.description' => 'Windows shares + most home NAS appliances',
			'backupTargetEditor.kinds.webdav.label' => 'WebDAV',
			'backupTargetEditor.kinds.webdav.description' => 'Self-hosted clouds + file-sharing services',
			'backupTargetEditor.kinds.sftp.label' => 'SFTP',
			'backupTargetEditor.kinds.sftp.description' => 'Any SSH-accessible server',
			'backupTargetEditor.kinds.s3.label' => 'S3 / compatible',
			'backupTargetEditor.kinds.s3.description' => 'Amazon S3 + S3-compatible buckets (MinIO, R2, B2)',
			'backupTargetEditor.kinds.rclone.label' => 'rclone (any)',
			'backupTargetEditor.kinds.rclone.description' => 'OneDrive, Google Drive, Dropbox via the rclone CLI',
			'backupTargetEditor.formTitleEdit' => 'Edit target',
			'backupTargetEditor.formTitleNew' => 'New backup target',
			'backupTargetEditor.idHintAuto' => ({required Object prefix}) => 'Auto: ${prefix}-1',
			'backupTargetEditor.idHelper' => 'Lower-case letters, digits, dashes. Defaults to the next available slot.',
			'backupTargetEditor.enabledOn' => 'Scheduled and ad-hoc backups can target this.',
			'backupTargetEditor.enabledOff' => 'Server will refuse to write backups here.',
			'backupTargetEditor.saving' => 'Saving…',
			'backupTargetEditor.create' => 'Create',
			'backupTargetEditor.rootDirLabel' => 'Root directory',
			'backupTargetEditor.rootDirHint' => 'Empty = cfg.backup.local_dir (~/.opendray/backups)',
			'backupTargetEditor.hostLabel' => 'Host',
			'backupTargetEditor.portLabel' => 'Port',
			'backupTargetEditor.shareLabel' => 'Share',
			'backupTargetEditor.shareHint' => 'Top-level share name',
			'backupTargetEditor.shareSampleHint' => 'Claude_Workspace',
			'backupTargetEditor.userLabel' => 'User',
			'backupTargetEditor.passwordLabel' => 'Password',
			'backupTargetEditor.passwordHintKeepCurrent' => 'Leave blank to keep current',
			'backupTargetEditor.passwordHintKeep' => 'Leave blank to keep',
			'backupTargetEditor.pathPrefixLabel' => 'Path prefix',
			'backupTargetEditor.pathPrefixHintShareRoot' => 'Sub-folder under the share root (optional)',
			'backupTargetEditor.pathPrefixHintBaseUrl' => 'Sub-folder under the base URL (optional)',
			'backupTargetEditor.pathPrefixHintObjectKey' => 'Object-key prefix (optional)',
			'backupTargetEditor.pathPrefixHintSshFolder' => 'Absolute or relative to user home (optional)',
			'backupTargetEditor.pathPrefixHintRemoteRoot' => 'Sub-folder under the remote root (optional)',
			'backupTargetEditor.endpointLabel' => 'Endpoint',
			'backupTargetEditor.regionLabel' => 'Region',
			'backupTargetEditor.bucketLabel' => 'Bucket',
			'backupTargetEditor.accessKeyLabel' => 'Access key',
			'backupTargetEditor.secretKeyLabel' => 'Secret key',
			'backupTargetEditor.secretKeyHintEdit' => 'Leave blank to keep current. Stored AES-256-GCM encrypted.',
			'backupTargetEditor.secretKeyHintNew' => 'Stored AES-256-GCM encrypted; never echoed back.',
			'backupTargetEditor.baseUrlLabel' => 'Base URL',
			'backupTargetEditor.baseUrlHint' => 'Full URL including path. Nextcloud: https://cloud.example/remote.php/dav/files/<user>',
			'backupTargetEditor.sftpPasswordHintEdit' => 'Leave blank to keep. If both password + private key are present, the private key wins.',
			'backupTargetEditor.sftpPasswordHintNew' => 'Either password OR private key. If both, password becomes a fallback only.',
			'backupTargetEditor.privateKeyLabel' => 'Private key (PEM)',
			'backupTargetEditor.privateKeyHintEdit' => 'Leave blank to keep. Paste OpenSSH/PEM contents.',
			'backupTargetEditor.privateKeyHintNew' => 'Paste the contents of an OpenSSH/PEM private key. Multi-line input — keep the BEGIN/END markers.',
			'backupTargetEditor.hostKeyLabel' => 'Host key (pinning)',
			'backupTargetEditor.hostKeyHint' => 'OpenSSH-style server public key. `ssh-keyscan <host>` to obtain. Blank = no pinning (NOT recommended outside LAN).',
			'backupTargetEditor.rcloneNote' => 'Requires the rclone CLI on the opendray host. First run `rclone config` once interactively to authenticate cloud accounts.',
			'backupTargetEditor.rcloneRemoteLabel' => 'Remote name',
			'backupTargetEditor.rcloneRemoteHint' => 'Name from `rclone config` (no colon).',
			'backupTargetEditor.rcloneBinaryLabel' => 'Binary path',
			'backupTargetEditor.rcloneBinaryHint' => 'Override `which rclone`. Empty = PATH lookup.',
			'backupTargetEditor.rcloneConfigLabel' => 'Config path',
			'backupTargetEditor.rcloneConfigHint' => 'Override --config. Empty = rclone default.',
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
			'githosts.form.kinds.github' => 'GitHub',
			'githosts.form.kinds.gitlab' => 'GitLab',
			'githosts.form.kinds.bitbucket' => 'Bitbucket',
			'githosts.form.kinds.gitea' => 'Gitea',
			'githosts.form.kinds.custom' => 'Custom',
			'githosts.form.validateHost' => 'Host is required.',
			'githosts.form.validateName' => 'Name is required.',
			'githosts.form.snackAdded' => 'Host added.',
			'githosts.form.snackUpdated' => 'Host updated.',
			'githosts.form.saveFailedApi' => ({required Object error}) => 'Save failed: ${error}',
			'githosts.form.saveFailedGeneric' => ({required Object error}) => 'Save failed: ${error}',
			'githosts.form.saving' => 'Saving…',
			'githosts.form.save' => 'Save',
			'githosts.form.add' => 'Add',
			'githosts.form.nameHelper' => 'Display name shown in PR lists.',
			'githosts.form.tokenLabelKeep' => 'Token (leave blank to keep existing)',
			'githosts.form.tokenLabel' => 'Token',
			'githosts.form.tokenHintKeep' => 'Leave blank to keep existing.',
			'githosts.form.tokenHintNew' => 'Paste the personal access token.',
			'githosts.form.enabledHelper' => 'Available to sessions for PR / remote lookups.',
			'githosts.form.validateTokenRequired' => 'Token is required when adding a host.',
			_ => null,
		} ?? switch (path) {
			'githosts.form.appBarEdit' => ({required Object name}) => 'Edit ${name}',
			'githosts.form.appBarNew' => 'Add git host',
			'githosts.form.tokenPreviewHint' => ({required Object preview}) => 'Current preview: ${preview}',
			'githosts.form.tokenPreviewNone' => '(none)',
			'githosts.form.pausedSubtitle' => 'Paused — sessions skip this host.',
			'githosts.deleteBody' => ({required Object host}) => 'Removes the credential. Sessions trying to list PRs from ${host} will fall back to the unauthenticated API.',
			'githosts.deletedSnack' => ({required Object name}) => 'Deleted ${name}.',
			'githosts.enabledSnack' => ({required Object name}) => '${name} enabled.',
			'githosts.disabledSnack' => ({required Object name}) => '${name} disabled.',
			'githosts.emptyList' => 'No git hosts configured.\n\nAdd a credential so the gateway can list pull requests across your repos.',
			'githosts.failedToLoad' => 'Failed to load git hosts',
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
			'onboarding.gatewayLabel' => 'Gateway URL',
			'onboarding.gatewayHint' => 'https://opendray.example.com',
			'onboarding.kContinue' => 'Continue',
			'skills.title' => 'Skills',
			'skills.newSkill' => 'New skill',
			'skills.customizingBuiltin' => ({required Object id}) => 'Customizing built-in ${id}',
			'skills.idLabel' => 'Id (slug)',
			'skills.idHint' => 'e.g. tdd-guide',
			'skills.bodyLabel' => 'Body (markdown)',
			'skills.loadFailedApi' => ({required Object error}) => 'Load failed: ${error}',
			'skills.loadFailedGeneric' => ({required Object error}) => 'Load failed: ${error}',
			'skills.idRequired' => 'Id is required.',
			'skills.bodyRequired' => 'Body cannot be empty.',
			'skills.snackCreated' => 'Skill created.',
			'skills.snackOverride' => 'Saved as vault override.',
			'skills.snackUpdated' => 'Skill updated.',
			'skills.saveFailedApi' => ({required Object error}) => 'Save failed: ${error}',
			'skills.saveFailedGeneric' => ({required Object error}) => 'Save failed: ${error}',
			'skills.resetTitle' => 'Reset to built-in?',
			'skills.deleteTitle' => 'Delete skill?',
			'skills.resetBody' => ({required Object id}) => 'Removes the vault override for ${id}. Sessions will fall back to the built-in body.',
			'skills.resetButton' => 'Reset',
			'skills.resetSnack' => ({required Object id}) => 'Reset ${id} to built-in.',
			'skills.deletedSnack' => ({required Object id}) => 'Deleted ${id}.',
			'skills.deleteFailedApi' => ({required Object error}) => 'Delete failed: ${error}',
			'skills.deleteFailedGeneric' => ({required Object error}) => 'Delete failed: ${error}',
			'skills.deleteBody' => ({required Object id}) => 'Removes ${id} from the vault. Sessions that reference it will fail until restored.',
			'skills.newSkillTitle' => 'New skill',
			'skills.customizeTitle' => ({required Object id}) => 'Customize ${id}',
			'skills.editTitle' => ({required Object id}) => 'Edit ${id}',
			'skills.resetTooltip' => 'Reset to built-in',
			'skills.deleteTooltip' => 'Delete',
			'skills.saving' => 'Saving…',
			'skills.saveOverride' => 'Save override',
			'skills.overrideBanner' => 'Saving creates a vault override with the same id. Sessions will use this body instead of the built-in until you reset.',
			'skills.idHelper' => 'Lowercase letters / digits / dash. Locked once created.',
			'skills.emptyList' => 'No skills configured. The gateway ships with built-ins (planner, code-reviewer, etc.).',
			'skills.failedToLoad' => 'Failed to load skills',
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
			'customTasks.snackCreated' => 'Task created.',
			'customTasks.snackUpdated' => 'Task updated.',
			'customTasks.deleteBody' => 'Removes the task from the catalogue. Sessions that already inserted it stay unaffected.',
			'customTasks.introBanner' => 'Define your own slash commands. They appear in the session task picker alongside the built-ins.',
			'customTasks.validateNameRequired' => 'Name is required',
			'customTasks.validateCommandRequired' => 'Command is required',
			'customTasks.validateProjectCwd' => 'Project-scoped tasks need an absolute cwd path',
			'customTasks.appBarEdit' => 'Edit custom task',
			'customTasks.appBarNew' => 'New custom task',
			'customTasks.fieldName' => 'Name',
			'customTasks.nameHelper' => 'Shown in the inspector\'s task picker.',
			'customTasks.fieldCommand' => 'Command',
			'customTasks.commandHelper' => 'The text inserted into the session when picked. Can be a CLI command or a Claude slash command.',
			'customTasks.fieldDescription' => 'Description (optional)',
			'customTasks.fieldScope' => 'Scope',
			'customTasks.globalScopeHint' => 'Visible from every session, regardless of cwd.',
			'customTasks.projectScopeHint' => 'Visible only when a session\'s cwd matches the path below.',
			'customTasks.fieldProjectCwd' => 'Project cwd',
			'customTasks.cwdHelper' => 'Absolute path. Sessions spawned with this exact cwd will see the task.',
			'customTasks.saving' => 'Saving…',
			'customTasks.save' => 'Save',
			'customTasks.create' => 'Create',
			'customTasks.failedToLoad' => 'Failed to load custom tasks',
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
			'notesPage.popupDelete' => 'Delete',
			'notesPage.deleteBody' => 'This is irreversible. Vault git sync will remove the file on the gateway host too.',
			'notesPage.emptyFilterMatch' => ({required Object query}) => 'No notes match "${query}".',
			'notesPage.emptyVault' => 'Vault is empty. Tap + to create your first note.',
			'notesPage.emptyFolder' => ({required Object path}) => 'Folder "${path}" is empty.',
			'notesPage.validatePath' => 'Path is required',
			'notesPage.validatePathDots' => 'Path cannot contain ".."',
			'notesPage.pathHelper' => 'Auto-appends .md if missing.',
			'notesPage.editor.markdownHint' => 'Markdown…',
			'notesPage.editor.saving' => 'Saving…',
			'notesPage.editor.autosave' => 'Auto-saves as you type',
			'notesPage.editor.loadFailedApi' => ({required Object error}) => 'Load failed: ${error}',
			'notesPage.editor.loadFailedGeneric' => ({required Object error}) => 'Load failed: ${error}',
			'notesPage.editor.saveFailedApi' => ({required Object error}) => 'Save failed: ${error}',
			'notesPage.editor.saveFailedGeneric' => ({required Object error}) => 'Save failed: ${error}',
			'notesPage.editor.savedAt' => ({required Object time}) => 'Saved ${time}',
			'dataExport.title' => 'Data export & import',
			'dataExport.subtitle' => 'User-level bundles for migration or verification — separate from /backups (disaster recovery).',
			'dataExport.sections.export' => 'Export',
			'dataExport.sections.import' => 'Import',
			'dataExport.form.scope' => 'Scope',
			'dataExport.form.memories' => 'Memories',
			'dataExport.form.memoriesHint' => 'All persisted memories + their embeddings.',
			'dataExport.form.integrations' => 'Integrations',
			'dataExport.form.integrationOptions.none' => 'Skip',
			'dataExport.form.integrationOptions.noneHint' => 'Don\'t include the /integrations registry.',
			'dataExport.form.integrationOptions.metadata' => 'Metadata only (default)',
			'dataExport.form.integrationOptions.metadataHint' => 'Per-integration name + endpoint, no API keys.',
			'dataExport.form.integrationOptions.plaintext' => 'Plaintext keys',
			'dataExport.form.integrationOptions.plaintextHint' => 'DANGEROUS: includes raw API tokens. v1 stores only bcrypt hashes, so this is effectively a no-op today; surface anyway.',
			'dataExport.form.confirmWarning' => 'Plaintext key export contains decryptable secrets. Type "I understand" to confirm.',
			'dataExport.form.confirmPlaceholder' => 'Type "I understand"',
			'dataExport.form.confirmSentinel' => 'I understand',
			'dataExport.form.customTasks' => 'Custom tasks',
			'dataExport.form.customTasksHint' => 'Per-user task definitions (cron schedules + script bodies).',
			'dataExport.form.footnote' => 'Bundles expire 7 days after creation. Download link is single-use.',
			'dataExport.form.create' => 'Create bundle',
			'dataExport.form.building' => 'Building…',
			'dataExport.form.readyToast' => 'Bundle ready',
			'dataExport.form.readyDescription' => ({required Object bytes}) => '${bytes} bytes — download from the history below.',
			'dataExport.form.failedToast' => ({required Object error}) => 'Bundle creation failed: ${error}',
			'dataExport.history.title' => 'Export history',
			'dataExport.history.loading' => 'Loading…',
			'dataExport.history.empty' => 'No exports yet.',
			'dataExport.history.listFailedToast' => ({required Object error}) => 'Failed to load exports: ${error}',
			'dataExport.history.downloadFailedToast' => ({required Object error}) => 'Failed to fetch download token: ${error}',
			'dataExport.history.noTokenToast' => 'This export has no usable download token (already consumed or expired).',
			'dataExport.history.deletedToast' => 'Export deleted.',
			'dataExport.history.deleteFailedToast' => ({required Object error}) => 'Failed to delete export: ${error}',
			'dataExport.history.deleteConfirmTitle' => 'Delete export?',
			'dataExport.history.deleteConfirmBody' => ({required Object id}) => 'Removes the bundle and revokes the download token. ${id}',
			'dataExport.history.download' => 'Download',
			'dataExport.history.delete' => 'Delete',
			'dataExport.history.downloadCopiedToast' => 'Download URL copied to clipboard. Paste into a browser to fetch (single-use).',
			'dataExport.history.columns.scope' => 'Scope',
			'dataExport.history.columns.size' => 'Size',
			'dataExport.history.columns.expires' => 'Expires',
			'dataExport.history.columns.actions' => 'Actions',
			'dataExport.history.scopeEmpty' => '(empty)',
			'dataExport.history.scopeMemories' => 'memories',
			'dataExport.history.scopeIntegrations' => ({required Object mode}) => 'integrations(${mode})',
			'dataExport.history.scopeCustomTasks' => 'custom_tasks',
			'dataExport.import.intro' => 'Replays a bundle previously produced by Export. Only the entities you tick below are imported; everything else in the bundle is ignored.',
			'dataExport.import.bundleLabel' => 'Bundle file (.zip)',
			'dataExport.import.pickFile' => 'Pick file',
			'dataExport.import.fileSelected' => ({required Object name, required Object size}) => '${name} · ${size}',
			'dataExport.import.noFile' => 'No file selected',
			'dataExport.import.memoriesLabel' => 'Memories',
			'dataExport.import.integrationsLabel' => 'Integrations',
			'dataExport.import.customTasksLabel' => 'Custom tasks',
			'dataExport.import.importBundle' => 'Import bundle',
			'dataExport.import.importing' => 'Importing…',
			'dataExport.import.pickFileToast' => 'Pick a bundle file first.',
			'dataExport.import.doneToast' => 'Import done',
			'dataExport.import.finishedWithErrors' => 'Import finished with errors',
			'dataExport.import.failedToast' => ({required Object error}) => 'Import failed: ${error}',
			'dataExport.import.summaryCard.memories' => 'Memories',
			'dataExport.import.summaryCard.integrations' => 'Integrations',
			'dataExport.import.summaryCard.customTasks' => 'Custom tasks',
			'dataExport.import.summaryCard.created' => 'created',
			'dataExport.import.summaryCard.skipped' => 'skipped',
			'dataExport.import.summaryCard.failed' => 'failed',
			'dataExport.imports.title' => 'Import history',
			'dataExport.imports.loading' => 'Loading…',
			'dataExport.imports.empty' => 'No imports yet.',
			'dataExport.imports.listFailedToast' => ({required Object error}) => 'Failed to load imports: ${error}',
			'dataExport.imports.noneCounts' => '(no counts)',
			'dataExport.imports.sourceUnknown' => '(unknown source)',
			'dataExport.imports.columns.id' => 'ID',
			'dataExport.imports.columns.status' => 'Status',
			'dataExport.imports.columns.source' => 'Source',
			'dataExport.imports.columns.counts' => 'Counts',
			'dataExport.imports.columns.when' => 'When',
			'dataExport.relative.inSeconds' => ({required Object n}) => 'in ${n}s',
			'dataExport.relative.inMinutes' => ({required Object n}) => 'in ${n}m',
			'dataExport.relative.inHours' => ({required Object n}) => 'in ${n}h',
			'dataExport.relative.inDays' => ({required Object n}) => 'in ${n}d',
			'dataExport.relative.secondsAgo' => ({required Object n}) => '${n}s ago',
			'dataExport.relative.minutesAgo' => ({required Object n}) => '${n}m ago',
			'dataExport.relative.hoursAgo' => ({required Object n}) => '${n}h ago',
			'dataExport.status.pending' => 'pending',
			'dataExport.status.running' => 'running',
			'dataExport.status.ready' => 'ready',
			'dataExport.status.failed' => 'failed',
			'dataExport.status.expired' => 'expired',
			'dataExport.status.succeeded' => 'succeeded',
			'memory.title' => 'Memory',
			'memory.more' => 'More',
			'memory.workers' => 'Memory workers',
			'memory.rank.title' => 'Rank breakdown',
			'memory.rank.effective' => ({required Object value}) => 'Effective score: ${value}',
			'memory.rank.similarity' => 'Cosine similarity',
			'memory.rank.ageMultiplier' => ({required Object days}) => 'Age multiplier (${days}d old)',
			'memory.rank.hitMultiplier' => ({required Object hits}) => 'Hit-count multiplier (${hits} hits)',
			'memory.rank.confidenceMultiplier' => 'Confidence multiplier',
			'memory.rank.formula' => 'effective = similarity × age × hits × confidence',
			'memory.rank.close' => 'Close',
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
			'settings.serverSettings.restartConfirmBody' => 'The gateway will exec itself. The mobile app may briefly lose connection.',
			'settings.serverSettings.restart' => 'Restart',
			'settings.serverSettings.restartQueuedSnack' => 'Restart requested. Pull-to-refresh in a moment.',
			'settings.serverSettings.restartFailedApi' => ({required Object error}) => 'Restart failed: ${error}',
			'settings.serverSettings.restartFailedGeneric' => ({required Object error}) => 'Restart failed: ${error}',
			'settings.serverSettings.loadedFrom' => ({required Object path}) => 'Loaded from: ${path}',
			'settings.serverSettings.restartHint' => 'Most sections need a gateway restart to take effect. The restart button is in the AppBar.',
			'settings.serverSettings.savedNeedsRestart' => 'Saved. Restart the gateway to apply.',
			'settings.serverSettings.savedSimple' => 'Saved.',
			'settings.serverSettings.changesNeedRestart' => 'Changes to this section need a gateway restart.',
			'settings.serverSettings.loadFailed' => 'Failed to load server settings',
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
			'settings.serverSettings.sectionDescriptions.general' => 'Listen address, operator account, token TTL.',
			'settings.serverSettings.sectionDescriptions.logging' => 'Verbosity, format, and on-disk log path.',
			'settings.serverSettings.sectionDescriptions.sessions' => 'Idle detection thresholds.',
			'settings.serverSettings.sectionDescriptions.vault' => 'Notes, skills, and git-versioned root.',
			'settings.serverSettings.sectionDescriptions.mcpRegistry' => 'Vault paths for MCP servers + secrets file.',
			'settings.serverSettings.sectionDescriptions.memory' => 'Cross-CLI persistent memory subsystem.',
			'settings.serverSettings.sectionDescriptions.backup' => 'Encrypted DB backups + admin data exports. Passphrase lives in the keyfile (Settings → Backups).',
			'settings.serverSettings.sectionDescriptions.storageClaude' => 'Where Claude transcripts live on disk.',
			'settings.serverSettings.sectionDescriptions.storageCodex' => 'Codex sessions root.',
			'settings.serverSettings.sectionDescriptions.storageGemini' => 'Per-project tmp + projects.json paths.',
			'settings.serverSettings.fields.listenAddress' => 'Listen address',
			'settings.serverSettings.fields.adminUser' => 'Admin user',
			'settings.serverSettings.fields.adminUserHelper' => 'Effective when no keyfile or env var is set. Otherwise see Settings → Account.',
			'settings.serverSettings.fields.adminPassword' => 'Admin password',
			'settings.serverSettings.fields.adminPasswordHelper' => 'Send blank to preserve. For ongoing rotations use Settings → Account (keyfile-backed, no restart).',
			'settings.serverSettings.fields.tokenTtlWeb' => 'Token TTL (web)',
			'settings.serverSettings.fields.tokenTtlHelper' => 'Go duration string, e.g. 24h, 30m.',
			'settings.serverSettings.fields.level' => 'Level',
			'settings.serverSettings.fields.format' => 'Format',
			'settings.serverSettings.fields.filePath' => 'File path',
			'settings.serverSettings.fields.filePathHelper' => 'Empty = stdout only.',
			'settings.serverSettings.fields.idleThreshold' => 'Idle threshold',
			'settings.serverSettings.fields.idleThresholdHelper' => 'Quiet period before a session is flagged idle. Go duration.',
			'settings.serverSettings.fields.idleCheckInterval' => 'Idle check interval',
			'settings.serverSettings.fields.idleCheckHelper' => 'How often the idle reaper runs.',
			'settings.serverSettings.fields.root' => 'Root',
			'settings.serverSettings.fields.rootHelper' => 'Parent of notes / skills / git_root sub-paths.',
			'settings.serverSettings.fields.notesPath' => 'Notes path',
			'settings.serverSettings.fields.skillsPath' => 'Skills path',
			'settings.serverSettings.fields.gitRoot' => 'Git root',
			'settings.serverSettings.fields.personalPrefix' => 'Personal prefix',
			'settings.serverSettings.fields.projectsPrefix' => 'Projects prefix',
			'settings.serverSettings.fields.registryRoot' => 'Registry root',
			'settings.serverSettings.fields.secretsFile' => 'Secrets file',
			'settings.serverSettings.fields.backend' => 'Backend',
			'settings.serverSettings.fields.store' => 'Store',
			'settings.serverSettings.fields.defaultTopK' => 'Default top-k',
			'settings.serverSettings.fields.similarityThreshold' => 'Similarity threshold',
			'settings.serverSettings.fields.defaultScope' => 'Default scope',
			'settings.serverSettings.fields.chromemHelper' => 'When store=chromem.',
			'settings.serverSettings.fields.preserveHelper' => 'Blank to preserve current.',
			'settings.serverSettings.fields.localModelName' => 'Local model name',
			'settings.serverSettings.fields.localLibraryPath' => 'Local library path',
			'settings.serverSettings.fields.localModelPath' => 'Local model path',
			'settings.serverSettings.fields.localTokenizerPath' => 'Local tokenizer path',
			'settings.serverSettings.fields.localMaxSeqLen' => 'Local max seq len',
			'settings.serverSettings.fields.backupEnabled' => 'Enabled',
			'settings.serverSettings.fields.backupEnabledHelper' => 'Even with this on, the backup subsystem stays off until OPENDRAY_BACKUP_KEY or the keyfile is configured.',
			'settings.serverSettings.fields.backupLocalDir' => 'Local dir',
			'settings.serverSettings.fields.backupExportDir' => 'Export dir',
			'settings.serverSettings.fields.pathHelper' => 'Empty = resolve from PATH at startup.',
			'settings.serverSettings.fields.accountsDir' => 'Accounts dir',
			'settings.serverSettings.fields.accountsHelper' => 'Parent of per-account .claude/ subdirs. Empty = ~/.claude-accounts.',
			'settings.serverSettings.fields.sessionsRoot' => 'Sessions root',
			'settings.serverSettings.fields.sessionsRootHelper' => 'Empty = ~/.codex/sessions.',
			'settings.serverSettings.fields.listenHelper' => 'host:port the gateway binds to. Restart required.',
			'settings.serverSettings.fields.secretsHelper' => 'AES-256-GCM encrypted secrets vault.',
			'settings.serverSettings.fields.backendHelper' => 'auto picks the best available; local needs ONNX.',
			'settings.serverSettings.fields.similarityHelper' => '0.0–1.0; results under this are filtered out.',
			'settings.serverSettings.fields.defaultFallback' => ({required Object value}) => 'Default: ${value}',
			'settings.serverSettings.fields.chromemPath' => 'chromem path',
			'settings.serverSettings.fields.httpBaseUrl' => 'HTTP base URL',
			'settings.serverSettings.fields.httpModel' => 'HTTP model',
			'settings.serverSettings.fields.httpApiKey' => 'HTTP api key',
			'settings.serverSettings.fields.httpDimensions' => 'HTTP dimensions',
			'settings.serverSettings.fields.pgDumpPath' => 'pg_dump path',
			'settings.serverSettings.fields.pgRestorePath' => 'pg_restore path',
			'settings.serverSettings.fields.tmpRoot' => 'tmp root',
			'settings.serverSettings.fields.projectsJson' => 'projects.json',
			'settings.serverSettings.validateInteger' => ({required Object field}) => '"${field}" must be an integer',
			'settings.serverSettings.validateNumber' => ({required Object field}) => '"${field}" must be a number',
			_ => null,
		};
	}
}
