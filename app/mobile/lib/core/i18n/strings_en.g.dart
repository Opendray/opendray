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
