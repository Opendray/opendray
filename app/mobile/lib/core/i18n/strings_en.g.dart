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
			'mcp.secret.keyLabel' => 'Key',
			'mcp.secret.keyHint' => 'GITHUB_TOKEN, OPENAI_KEY, …',
			'mcp.secret.valueLabel' => 'Value',
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
			_ => null,
		};
	}
}
