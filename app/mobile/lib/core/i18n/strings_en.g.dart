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
	late final TranslationsSessionsActionEn action = TranslationsSessionsActionEn.internal(_root);
	late final TranslationsSessionsDirPickerEn dirPicker = TranslationsSessionsDirPickerEn.internal(_root);
	late final TranslationsSessionsSpawnSheetEn spawnSheet = TranslationsSessionsSpawnSheetEn.internal(_root);
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
