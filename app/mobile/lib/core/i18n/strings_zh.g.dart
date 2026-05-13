///
/// Generated file. Do not edit.
///
// coverage:ignore-file
// ignore_for_file: type=lint, unused_import
// dart format off

import 'package:flutter/widgets.dart';
import 'package:intl/intl.dart';
import 'package:slang/generated.dart';
import 'strings.g.dart';

// Path: <root>
class TranslationsZh extends Translations with BaseTranslations<AppLocale, Translations> {
	/// You can call this constructor and build your own translation instance of this locale.
	/// Constructing via the enum [AppLocale.build] is preferred.
	TranslationsZh({Map<String, Node>? overrides, PluralResolver? cardinalResolver, PluralResolver? ordinalResolver, TranslationMetadata<AppLocale, Translations>? meta})
		: assert(overrides == null, 'Set "translation_overrides: true" in order to enable this feature.'),
		  $meta = meta ?? TranslationMetadata(
		    locale: AppLocale.zh,
		    overrides: overrides ?? {},
		    cardinalResolver: cardinalResolver,
		    ordinalResolver: ordinalResolver,
		  ),
		  super(cardinalResolver: cardinalResolver, ordinalResolver: ordinalResolver) {
		super.$meta.setFlatMapFunction($meta.getTranslation); // copy base translations to super.$meta
		$meta.setFlatMapFunction(_flatMapFunction);
	}

	/// Metadata for the translations of <zh>.
	@override final TranslationMetadata<AppLocale, Translations> $meta;

	/// Access flat map
	@override dynamic operator[](String key) => $meta.getTranslation(key) ?? super.$meta.getTranslation(key);

	late final TranslationsZh _root = this; // ignore: unused_field

	@override 
	TranslationsZh $copyWith({TranslationMetadata<AppLocale, Translations>? meta}) => TranslationsZh(meta: meta ?? this.$meta);

	// Translations
	@override late final _TranslationsCommonZh common = _TranslationsCommonZh._(_root);
	@override late final _TranslationsAuthZh auth = _TranslationsAuthZh._(_root);
	@override late final _TranslationsNavZh nav = _TranslationsNavZh._(_root);
	@override late final _TranslationsMoreZh more = _TranslationsMoreZh._(_root);
	@override late final _TranslationsSessionsZh sessions = _TranslationsSessionsZh._(_root);
	@override late final _TranslationsAboutZh about = _TranslationsAboutZh._(_root);
	@override late final _TranslationsSettingsZh settings = _TranslationsSettingsZh._(_root);
}

// Path: common
class _TranslationsCommonZh extends TranslationsCommonEn {
	_TranslationsCommonZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get ok => '确定';
	@override String get cancel => '取消';
	@override String get save => '保存';
	@override String get delete => '删除';
	@override String get edit => '编辑';
	@override String get back => '返回';
	@override String get done => '完成';
	@override String get close => '关闭';
	@override String get retry => '重试';
}

// Path: auth
class _TranslationsAuthZh extends TranslationsAuthEn {
	_TranslationsAuthZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get signInTitle => '登录';
	@override String get changeServer => '更换';
	@override String get username => '用户名';
	@override String get password => '密码';
	@override String get signIn => '登录';
	@override String get errorRequired => '请输入用户名和密码';
	@override String errorGeneric({required Object error}) => '登录失败：${error}';
}

// Path: nav
class _TranslationsNavZh extends TranslationsNavEn {
	_TranslationsNavZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get sessions => '会话';
	@override String get memory => '记忆';
	@override String get notes => '笔记';
	@override String get more => '更多';
}

// Path: more
class _TranslationsMoreZh extends TranslationsMoreEn {
	_TranslationsMoreZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '更多';
	@override late final _TranslationsMoreIdentityZh identity = _TranslationsMoreIdentityZh._(_root);
	@override late final _TranslationsMoreSectionsZh sections = _TranslationsMoreSectionsZh._(_root);
	@override late final _TranslationsMoreItemsZh items = _TranslationsMoreItemsZh._(_root);
	@override String get signOut => '退出登录';
}

// Path: sessions
class _TranslationsSessionsZh extends TranslationsSessionsEn {
	_TranslationsSessionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '会话';
	@override String get refresh => '刷新';
	@override String get actions => '操作';
	@override String get spawn => '创建';
	@override late final _TranslationsSessionsFiltersZh filters = _TranslationsSessionsFiltersZh._(_root);
	@override late final _TranslationsSessionsCardZh card = _TranslationsSessionsCardZh._(_root);
	@override late final _TranslationsSessionsEmptyZh empty = _TranslationsSessionsEmptyZh._(_root);
	@override String get errorTitle => '加载会话失败';
	@override late final _TranslationsSessionsRelativeZh relative = _TranslationsSessionsRelativeZh._(_root);
	@override late final _TranslationsSessionsActionZh action = _TranslationsSessionsActionZh._(_root);
	@override late final _TranslationsSessionsDirPickerZh dirPicker = _TranslationsSessionsDirPickerZh._(_root);
	@override late final _TranslationsSessionsSpawnSheetZh spawnSheet = _TranslationsSessionsSpawnSheetZh._(_root);
}

// Path: about
class _TranslationsAboutZh extends TranslationsAboutEn {
	_TranslationsAboutZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '关于';
	@override String get loading => '加载中…';
	@override late final _TranslationsAboutSectionsZh sections = _TranslationsAboutSectionsZh._(_root);
	@override late final _TranslationsAboutFieldsZh fields = _TranslationsAboutFieldsZh._(_root);
	@override String copied({required Object label}) => '已复制 ${label}';
	@override String get copyTooltip => '复制';
	@override late final _TranslationsAboutCopyLabelsZh copyLabels = _TranslationsAboutCopyLabelsZh._(_root);
	@override String get tagline => 'opendray mobile — 多 CLI 网关控制。\n源码：github.com/Opendray/opendray_v2';
}

// Path: settings
class _TranslationsSettingsZh extends TranslationsSettingsEn {
	_TranslationsSettingsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '设置';
	@override late final _TranslationsSettingsLanguageZh language = _TranslationsSettingsLanguageZh._(_root);
	@override late final _TranslationsSettingsAppearanceZh appearance = _TranslationsSettingsAppearanceZh._(_root);
	@override late final _TranslationsSettingsAccountZh account = _TranslationsSettingsAccountZh._(_root);
	@override late final _TranslationsSettingsGatewayZh gateway = _TranslationsSettingsGatewayZh._(_root);
}

// Path: more.identity
class _TranslationsMoreIdentityZh extends TranslationsMoreIdentityEn {
	_TranslationsMoreIdentityZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get signedInAs => '登录账号';
	@override String get server => '服务器';
	@override String get tokenExpires => '令牌到期';
}

// Path: more.sections
class _TranslationsMoreSectionsZh extends TranslationsMoreSectionsEn {
	_TranslationsMoreSectionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get gateway => '网关';
	@override String get memory => '记忆';
	@override String get system => '系统';
}

// Path: more.items
class _TranslationsMoreItemsZh extends TranslationsMoreItemsEn {
	_TranslationsMoreItemsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsMoreItemsIntegrationsZh integrations = _TranslationsMoreItemsIntegrationsZh._(_root);
	@override late final _TranslationsMoreItemsChannelsZh channels = _TranslationsMoreItemsChannelsZh._(_root);
	@override late final _TranslationsMoreItemsProvidersZh providers = _TranslationsMoreItemsProvidersZh._(_root);
	@override late final _TranslationsMoreItemsMcpZh mcp = _TranslationsMoreItemsMcpZh._(_root);
	@override late final _TranslationsMoreItemsSkillsZh skills = _TranslationsMoreItemsSkillsZh._(_root);
	@override late final _TranslationsMoreItemsGitHostsZh gitHosts = _TranslationsMoreItemsGitHostsZh._(_root);
	@override late final _TranslationsMoreItemsCustomTasksZh customTasks = _TranslationsMoreItemsCustomTasksZh._(_root);
	@override late final _TranslationsMoreItemsProjectMemoryZh projectMemory = _TranslationsMoreItemsProjectMemoryZh._(_root);
	@override late final _TranslationsMoreItemsCleanupInboxZh cleanupInbox = _TranslationsMoreItemsCleanupInboxZh._(_root);
	@override late final _TranslationsMoreItemsBackupsZh backups = _TranslationsMoreItemsBackupsZh._(_root);
	@override late final _TranslationsMoreItemsSettingsZh settings = _TranslationsMoreItemsSettingsZh._(_root);
	@override late final _TranslationsMoreItemsAboutZh about = _TranslationsMoreItemsAboutZh._(_root);
}

// Path: sessions.filters
class _TranslationsSessionsFiltersZh extends TranslationsSessionsFiltersEn {
	_TranslationsSessionsFiltersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get all => '全部';
	@override String get running => '运行中';
	@override String get idle => '空闲';
	@override String get ended => '已结束';
}

// Path: sessions.card
class _TranslationsSessionsCardZh extends TranslationsSessionsCardEn {
	_TranslationsSessionsCardZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String startedRelative({required Object provider, required Object when}) => '${provider} · ${when}启动';
}

// Path: sessions.empty
class _TranslationsSessionsEmptyZh extends TranslationsSessionsEmptyEn {
	_TranslationsSessionsEmptyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get titleAll => '暂无会话';
	@override String titleFiltered({required Object filter}) => '没有匹配「${filter}」筛选的会话。';
	@override String get subtitleAll => '点击「创建」按钮新建一个。';
	@override String get subtitleFiltered => '试试其他筛选条件或下拉刷新。';
}

// Path: sessions.relative
class _TranslationsSessionsRelativeZh extends TranslationsSessionsRelativeEn {
	_TranslationsSessionsRelativeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String secondsAgo({required Object n}) => '${n} 秒前';
	@override String minutesAgo({required Object n}) => '${n} 分钟前';
	@override String hoursAgo({required Object n}) => '${n} 小时前';
	@override String daysAgo({required Object n}) => '${n} 天前';
}

// Path: sessions.action
class _TranslationsSessionsActionZh extends TranslationsSessionsActionEn {
	_TranslationsSessionsActionZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get stop => '停止';
	@override String get stopping => '停止中…';
	@override String get stopDescription => '发送 SIGTERM，保留历史';
	@override String get restart => '重启';
	@override String get restarting => '重启中…';
	@override String get restartDescription => '重新启动 CLI 进程';
	@override String get delete => '删除';
	@override String get deleteDescription => '移除会话及其历史';
	@override String get deleteConfirm => '确定永久删除此会话吗？其环形缓冲区和历史将全部丢失。';
	@override late final _TranslationsSessionsActionErrorsZh errors = _TranslationsSessionsActionErrorsZh._(_root);
}

// Path: sessions.dirPicker
class _TranslationsSessionsDirPickerZh extends TranslationsSessionsDirPickerEn {
	_TranslationsSessionsDirPickerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get parent => '上级';
	@override String get newFolder => '新建文件夹';
	@override String get useThisFolder => '使用此文件夹';
	@override String get loading => '加载中…';
	@override String get empty => '此处没有子文件夹。\n选择此文件夹，或新建一个。';
	@override String createdSnack({required Object path}) => '已创建 ${path}';
	@override String mkdirFailedSnack({required Object error}) => '创建文件夹失败：${error}';
	@override late final _TranslationsSessionsDirPickerDialogZh dialog = _TranslationsSessionsDirPickerDialogZh._(_root);
}

// Path: sessions.spawnSheet
class _TranslationsSessionsSpawnSheetZh extends TranslationsSessionsSpawnSheetEn {
	_TranslationsSessionsSpawnSheetZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '新建会话';
	@override String get errorRequired => '需要指定提供商和工作目录';
	@override String errorGeneric({required Object error}) => '创建会话失败：${error}';
	@override String get cancel => '取消';
	@override String get spawn => '创建';
	@override String get providerLabel => '提供商';
	@override String get disabledSuffix => '（已停用）';
	@override String get cwdLabel => '工作目录';
	@override String get cwdHint => '/Users/you/projects/foo';
	@override String get cwdHelper => '网关主机上的绝对路径。';
	@override String get browse => '浏览';
	@override String get nameLabel => '名称（可选）';
	@override String get nameHint => '例如：backend-refactor';
	@override String get argsLabel => '额外参数（可选）';
	@override String get argsHint => '--continue --verbose';
	@override String get argsHelper => '以空格分隔；留空使用提供商默认值。';
	@override late final _TranslationsSessionsSpawnSheetBypassZh bypass = _TranslationsSessionsSpawnSheetBypassZh._(_root);
	@override late final _TranslationsSessionsSpawnSheetNoProvidersZh noProviders = _TranslationsSessionsSpawnSheetNoProvidersZh._(_root);
	@override late final _TranslationsSessionsSpawnSheetProviderLoadErrorZh providerLoadError = _TranslationsSessionsSpawnSheetProviderLoadErrorZh._(_root);
	@override late final _TranslationsSessionsSpawnSheetClaudeAccountZh claudeAccount = _TranslationsSessionsSpawnSheetClaudeAccountZh._(_root);
}

// Path: about.sections
class _TranslationsAboutSectionsZh extends TranslationsAboutSectionsEn {
	_TranslationsAboutSectionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get app => '应用';
	@override String get server => '服务器';
}

// Path: about.fields
class _TranslationsAboutFieldsZh extends TranslationsAboutFieldsEn {
	_TranslationsAboutFieldsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get app => '应用';
	@override String get version => '版本';
	@override String versionFormat({required Object version, required Object build}) => '${version}（build ${build}）';
	@override String get package => '包名';
	@override String get url => 'URL';
	@override String get signedInAs => '登录账号';
	@override String get tokenExpires => '令牌到期';
}

// Path: about.copyLabels
class _TranslationsAboutCopyLabelsZh extends TranslationsAboutCopyLabelsEn {
	_TranslationsAboutCopyLabelsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get version => '版本';
	@override String get serverUrl => '服务器 URL';
}

// Path: settings.language
class _TranslationsSettingsLanguageZh extends TranslationsSettingsLanguageEn {
	_TranslationsSettingsLanguageZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get section => '语言';
	@override String get system => '跟随系统';
	@override String get systemSubtitle => '跟随手机的语言设置';
	@override String get english => 'English';
	@override String get chinese => '中文';
}

// Path: settings.appearance
class _TranslationsSettingsAppearanceZh extends TranslationsSettingsAppearanceEn {
	_TranslationsSettingsAppearanceZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get section => '外观';
	@override String get system => '跟随系统';
	@override String get systemSubtitle => '跟随手机的外观设置';
	@override String get light => '浅色';
	@override String get lightSubtitle => '始终使用浅色主题';
	@override String get dark => '深色';
	@override String get darkSubtitle => '始终使用深色主题';
}

// Path: settings.account
class _TranslationsSettingsAccountZh extends TranslationsSettingsAccountEn {
	_TranslationsSettingsAccountZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get section => '账户';
	@override String get changeCredentials => '修改凭据';
	@override String get changeCredentialsSubtitle => '用户名和密码';
}

// Path: settings.gateway
class _TranslationsSettingsGatewayZh extends TranslationsSettingsGatewayEn {
	_TranslationsSettingsGatewayZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get section => '网关';
	@override String get serverSettings => '服务器设置';
	@override String get serverSettingsSubtitle => '监听地址、日志、凭据库、内存、存储路径…';
	@override String get liveLogs => '实时日志';
	@override String get liveLogsSubtitle => '查看网关实时日志 — 与 Web 管理端同源';
}

// Path: more.items.integrations
class _TranslationsMoreItemsIntegrationsZh extends TranslationsMoreItemsIntegrationsEn {
	_TranslationsMoreItemsIntegrationsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '集成';
	@override String get subtitle => 'API 调用方 — 近期活动与错误率';
}

// Path: more.items.channels
class _TranslationsMoreItemsChannelsZh extends TranslationsMoreItemsChannelsEn {
	_TranslationsMoreItemsChannelsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '通道';
	@override String get subtitle => '通知目的地';
}

// Path: more.items.providers
class _TranslationsMoreItemsProvidersZh extends TranslationsMoreItemsProvidersEn {
	_TranslationsMoreItemsProvidersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '提供商';
	@override String get subtitle => 'Claude / Codex / Gemini CLI 状态';
}

// Path: more.items.mcp
class _TranslationsMoreItemsMcpZh extends TranslationsMoreItemsMcpEn {
	_TranslationsMoreItemsMcpZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'MCP';
	@override String get subtitle => 'Model Context Protocol 服务与密钥';
}

// Path: more.items.skills
class _TranslationsMoreItemsSkillsZh extends TranslationsMoreItemsSkillsEn {
	_TranslationsMoreItemsSkillsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '技能';
	@override String get subtitle => 'Agent SKILL.md 库（内置 + 库）';
}

// Path: more.items.gitHosts
class _TranslationsMoreItemsGitHostsZh extends TranslationsMoreItemsGitHostsEn {
	_TranslationsMoreItemsGitHostsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Git 主机';
	@override String get subtitle => 'GitHub / GitLab 等的 PAT 凭据';
}

// Path: more.items.customTasks
class _TranslationsMoreItemsCustomTasksZh extends TranslationsMoreItemsCustomTasksEn {
	_TranslationsMoreItemsCustomTasksZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '自定义任务';
	@override String get subtitle => '会话任务选择器中显示的斜杠命令';
}

// Path: more.items.projectMemory
class _TranslationsMoreItemsProjectMemoryZh extends TranslationsMoreItemsProjectMemoryEn {
	_TranslationsMoreItemsProjectMemoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '项目目标 / 计划 / 日志';
	@override String get subtitle => '按 cwd 的记忆层 2-4 + 代理提案';
}

// Path: more.items.cleanupInbox
class _TranslationsMoreItemsCleanupInboxZh extends TranslationsMoreItemsCleanupInboxEn {
	_TranslationsMoreItemsCleanupInboxZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '清理收件箱';
	@override String get subtitle => '跨项目的 LLM 提议删除 / 合并';
}

// Path: more.items.backups
class _TranslationsMoreItemsBackupsZh extends TranslationsMoreItemsBackupsEn {
	_TranslationsMoreItemsBackupsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '备份';
	@override String get subtitle => '最新备份状态与立即运行';
}

// Path: more.items.settings
class _TranslationsMoreItemsSettingsZh extends TranslationsMoreItemsSettingsEn {
	_TranslationsMoreItemsSettingsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '设置';
	@override String get subtitle => '语言、外观、账户';
}

// Path: more.items.about
class _TranslationsMoreItemsAboutZh extends TranslationsMoreItemsAboutEn {
	_TranslationsMoreItemsAboutZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '关于';
	@override String get subtitle => '构建版本与服务器信息';
}

// Path: sessions.action.errors
class _TranslationsSessionsActionErrorsZh extends TranslationsSessionsActionErrorsEn {
	_TranslationsSessionsActionErrorsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String stop({required Object error}) => '停止失败：${error}';
	@override String start({required Object error}) => '重启失败：${error}';
	@override String delete({required Object error}) => '删除失败：${error}';
}

// Path: sessions.dirPicker.dialog
class _TranslationsSessionsDirPickerDialogZh extends TranslationsSessionsDirPickerDialogEn {
	_TranslationsSessionsDirPickerDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '新建文件夹';
	@override String get hint => '文件夹名';
	@override String get create => '创建';
}

// Path: sessions.spawnSheet.bypass
class _TranslationsSessionsSpawnSheetBypassZh extends TranslationsSessionsSpawnSheetBypassEn {
	_TranslationsSessionsSpawnSheetBypassZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get labelClaude => '绕过权限';
	@override String get labelCodex => '自动批准（从不询问）';
	@override String get labelGemini => 'YOLO 模式';
	@override String get subtitleOn => '此会话将以提升的自主权运行。';
	@override String get subtitleOff => '关闭 — 确认和提示按正常方式处理。';
}

// Path: sessions.spawnSheet.noProviders
class _TranslationsSessionsSpawnSheetNoProvidersZh extends TranslationsSessionsSpawnSheetNoProvidersEn {
	_TranslationsSessionsSpawnSheetNoProvidersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '未配置任何提供商';
	@override String get message => '网关没有启用任何 CLI 提供商。在 Web 管理端的「提供商」中配置，或编辑 config.toml 的 [providers] 段，然后点击重新加载。';
	@override String get reload => '重新加载';
}

// Path: sessions.spawnSheet.providerLoadError
class _TranslationsSessionsSpawnSheetProviderLoadErrorZh extends TranslationsSessionsSpawnSheetProviderLoadErrorEn {
	_TranslationsSessionsSpawnSheetProviderLoadErrorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '无法加载提供商';
	@override String get networkError => '网络错误';
	@override String serverPrefix({required Object code}) => '服务器 ${code}';
	@override String format({required Object prefix, required Object message}) => '${prefix}：${message}';
}

// Path: sessions.spawnSheet.claudeAccount
class _TranslationsSessionsSpawnSheetClaudeAccountZh extends TranslationsSessionsSpawnSheetClaudeAccountEn {
	_TranslationsSessionsSpawnSheetClaudeAccountZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Claude 账号';
	@override String get helperMulti => '已配置多个账号 — 为此会话选择一个。';
	@override String get helperSingle => '选择已配置的账号，或使用默认（环境变量 / 系统）。';
	@override String get kDefault => '默认（环境变量 / 系统）';
	@override String get disabledSuffix => '（已停用）';
	@override String get noTokenSuffix => '（无令牌）';
	@override String get noneHint => '未配置 Claude 账号 — 网关将使用系统的 ANTHROPIC_API_KEY。在 Web 管理端的「设置 → 账号」中添加账号。';
	@override String errorHint({required Object error}) => '无法加载 Claude 账号（${error}）。会话将以网关默认配置启动。';
}

/// The flat map containing all translations for locale <zh>.
/// Only for edge cases! For simple maps, use the map function of this library.
///
/// The Dart AOT compiler has issues with very large switch statements,
/// so the map is split into smaller functions (512 entries each).
extension on TranslationsZh {
	dynamic _flatMapFunction(String path) {
		return switch (path) {
			'common.ok' => '确定',
			'common.cancel' => '取消',
			'common.save' => '保存',
			'common.delete' => '删除',
			'common.edit' => '编辑',
			'common.back' => '返回',
			'common.done' => '完成',
			'common.close' => '关闭',
			'common.retry' => '重试',
			'auth.signInTitle' => '登录',
			'auth.changeServer' => '更换',
			'auth.username' => '用户名',
			'auth.password' => '密码',
			'auth.signIn' => '登录',
			'auth.errorRequired' => '请输入用户名和密码',
			'auth.errorGeneric' => ({required Object error}) => '登录失败：${error}',
			'nav.sessions' => '会话',
			'nav.memory' => '记忆',
			'nav.notes' => '笔记',
			'nav.more' => '更多',
			'more.title' => '更多',
			'more.identity.signedInAs' => '登录账号',
			'more.identity.server' => '服务器',
			'more.identity.tokenExpires' => '令牌到期',
			'more.sections.gateway' => '网关',
			'more.sections.memory' => '记忆',
			'more.sections.system' => '系统',
			'more.items.integrations.title' => '集成',
			'more.items.integrations.subtitle' => 'API 调用方 — 近期活动与错误率',
			'more.items.channels.title' => '通道',
			'more.items.channels.subtitle' => '通知目的地',
			'more.items.providers.title' => '提供商',
			'more.items.providers.subtitle' => 'Claude / Codex / Gemini CLI 状态',
			'more.items.mcp.title' => 'MCP',
			'more.items.mcp.subtitle' => 'Model Context Protocol 服务与密钥',
			'more.items.skills.title' => '技能',
			'more.items.skills.subtitle' => 'Agent SKILL.md 库（内置 + 库）',
			'more.items.gitHosts.title' => 'Git 主机',
			'more.items.gitHosts.subtitle' => 'GitHub / GitLab 等的 PAT 凭据',
			'more.items.customTasks.title' => '自定义任务',
			'more.items.customTasks.subtitle' => '会话任务选择器中显示的斜杠命令',
			'more.items.projectMemory.title' => '项目目标 / 计划 / 日志',
			'more.items.projectMemory.subtitle' => '按 cwd 的记忆层 2-4 + 代理提案',
			'more.items.cleanupInbox.title' => '清理收件箱',
			'more.items.cleanupInbox.subtitle' => '跨项目的 LLM 提议删除 / 合并',
			'more.items.backups.title' => '备份',
			'more.items.backups.subtitle' => '最新备份状态与立即运行',
			'more.items.settings.title' => '设置',
			'more.items.settings.subtitle' => '语言、外观、账户',
			'more.items.about.title' => '关于',
			'more.items.about.subtitle' => '构建版本与服务器信息',
			'more.signOut' => '退出登录',
			'sessions.title' => '会话',
			'sessions.refresh' => '刷新',
			'sessions.actions' => '操作',
			'sessions.spawn' => '创建',
			'sessions.filters.all' => '全部',
			'sessions.filters.running' => '运行中',
			'sessions.filters.idle' => '空闲',
			'sessions.filters.ended' => '已结束',
			'sessions.card.startedRelative' => ({required Object provider, required Object when}) => '${provider} · ${when}启动',
			'sessions.empty.titleAll' => '暂无会话',
			'sessions.empty.titleFiltered' => ({required Object filter}) => '没有匹配「${filter}」筛选的会话。',
			'sessions.empty.subtitleAll' => '点击「创建」按钮新建一个。',
			'sessions.empty.subtitleFiltered' => '试试其他筛选条件或下拉刷新。',
			'sessions.errorTitle' => '加载会话失败',
			'sessions.relative.secondsAgo' => ({required Object n}) => '${n} 秒前',
			'sessions.relative.minutesAgo' => ({required Object n}) => '${n} 分钟前',
			'sessions.relative.hoursAgo' => ({required Object n}) => '${n} 小时前',
			'sessions.relative.daysAgo' => ({required Object n}) => '${n} 天前',
			'sessions.action.stop' => '停止',
			'sessions.action.stopping' => '停止中…',
			'sessions.action.stopDescription' => '发送 SIGTERM，保留历史',
			'sessions.action.restart' => '重启',
			'sessions.action.restarting' => '重启中…',
			'sessions.action.restartDescription' => '重新启动 CLI 进程',
			'sessions.action.delete' => '删除',
			'sessions.action.deleteDescription' => '移除会话及其历史',
			'sessions.action.deleteConfirm' => '确定永久删除此会话吗？其环形缓冲区和历史将全部丢失。',
			'sessions.action.errors.stop' => ({required Object error}) => '停止失败：${error}',
			'sessions.action.errors.start' => ({required Object error}) => '重启失败：${error}',
			'sessions.action.errors.delete' => ({required Object error}) => '删除失败：${error}',
			'sessions.dirPicker.parent' => '上级',
			'sessions.dirPicker.newFolder' => '新建文件夹',
			'sessions.dirPicker.useThisFolder' => '使用此文件夹',
			'sessions.dirPicker.loading' => '加载中…',
			'sessions.dirPicker.empty' => '此处没有子文件夹。\n选择此文件夹，或新建一个。',
			'sessions.dirPicker.createdSnack' => ({required Object path}) => '已创建 ${path}',
			'sessions.dirPicker.mkdirFailedSnack' => ({required Object error}) => '创建文件夹失败：${error}',
			'sessions.dirPicker.dialog.title' => '新建文件夹',
			'sessions.dirPicker.dialog.hint' => '文件夹名',
			'sessions.dirPicker.dialog.create' => '创建',
			'sessions.spawnSheet.title' => '新建会话',
			'sessions.spawnSheet.errorRequired' => '需要指定提供商和工作目录',
			'sessions.spawnSheet.errorGeneric' => ({required Object error}) => '创建会话失败：${error}',
			'sessions.spawnSheet.cancel' => '取消',
			'sessions.spawnSheet.spawn' => '创建',
			'sessions.spawnSheet.providerLabel' => '提供商',
			'sessions.spawnSheet.disabledSuffix' => '（已停用）',
			'sessions.spawnSheet.cwdLabel' => '工作目录',
			'sessions.spawnSheet.cwdHint' => '/Users/you/projects/foo',
			'sessions.spawnSheet.cwdHelper' => '网关主机上的绝对路径。',
			'sessions.spawnSheet.browse' => '浏览',
			'sessions.spawnSheet.nameLabel' => '名称（可选）',
			'sessions.spawnSheet.nameHint' => '例如：backend-refactor',
			'sessions.spawnSheet.argsLabel' => '额外参数（可选）',
			'sessions.spawnSheet.argsHint' => '--continue --verbose',
			'sessions.spawnSheet.argsHelper' => '以空格分隔；留空使用提供商默认值。',
			'sessions.spawnSheet.bypass.labelClaude' => '绕过权限',
			'sessions.spawnSheet.bypass.labelCodex' => '自动批准（从不询问）',
			'sessions.spawnSheet.bypass.labelGemini' => 'YOLO 模式',
			'sessions.spawnSheet.bypass.subtitleOn' => '此会话将以提升的自主权运行。',
			'sessions.spawnSheet.bypass.subtitleOff' => '关闭 — 确认和提示按正常方式处理。',
			'sessions.spawnSheet.noProviders.title' => '未配置任何提供商',
			'sessions.spawnSheet.noProviders.message' => '网关没有启用任何 CLI 提供商。在 Web 管理端的「提供商」中配置，或编辑 config.toml 的 [providers] 段，然后点击重新加载。',
			'sessions.spawnSheet.noProviders.reload' => '重新加载',
			'sessions.spawnSheet.providerLoadError.title' => '无法加载提供商',
			'sessions.spawnSheet.providerLoadError.networkError' => '网络错误',
			'sessions.spawnSheet.providerLoadError.serverPrefix' => ({required Object code}) => '服务器 ${code}',
			'sessions.spawnSheet.providerLoadError.format' => ({required Object prefix, required Object message}) => '${prefix}：${message}',
			'sessions.spawnSheet.claudeAccount.label' => 'Claude 账号',
			'sessions.spawnSheet.claudeAccount.helperMulti' => '已配置多个账号 — 为此会话选择一个。',
			'sessions.spawnSheet.claudeAccount.helperSingle' => '选择已配置的账号，或使用默认（环境变量 / 系统）。',
			'sessions.spawnSheet.claudeAccount.kDefault' => '默认（环境变量 / 系统）',
			'sessions.spawnSheet.claudeAccount.disabledSuffix' => '（已停用）',
			'sessions.spawnSheet.claudeAccount.noTokenSuffix' => '（无令牌）',
			'sessions.spawnSheet.claudeAccount.noneHint' => '未配置 Claude 账号 — 网关将使用系统的 ANTHROPIC_API_KEY。在 Web 管理端的「设置 → 账号」中添加账号。',
			'sessions.spawnSheet.claudeAccount.errorHint' => ({required Object error}) => '无法加载 Claude 账号（${error}）。会话将以网关默认配置启动。',
			'about.title' => '关于',
			'about.loading' => '加载中…',
			'about.sections.app' => '应用',
			'about.sections.server' => '服务器',
			'about.fields.app' => '应用',
			'about.fields.version' => '版本',
			'about.fields.versionFormat' => ({required Object version, required Object build}) => '${version}（build ${build}）',
			'about.fields.package' => '包名',
			'about.fields.url' => 'URL',
			'about.fields.signedInAs' => '登录账号',
			'about.fields.tokenExpires' => '令牌到期',
			'about.copied' => ({required Object label}) => '已复制 ${label}',
			'about.copyTooltip' => '复制',
			'about.copyLabels.version' => '版本',
			'about.copyLabels.serverUrl' => '服务器 URL',
			'about.tagline' => 'opendray mobile — 多 CLI 网关控制。\n源码：github.com/Opendray/opendray_v2',
			'settings.title' => '设置',
			'settings.language.section' => '语言',
			'settings.language.system' => '跟随系统',
			'settings.language.systemSubtitle' => '跟随手机的语言设置',
			'settings.language.english' => 'English',
			'settings.language.chinese' => '中文',
			'settings.appearance.section' => '外观',
			'settings.appearance.system' => '跟随系统',
			'settings.appearance.systemSubtitle' => '跟随手机的外观设置',
			'settings.appearance.light' => '浅色',
			'settings.appearance.lightSubtitle' => '始终使用浅色主题',
			'settings.appearance.dark' => '深色',
			'settings.appearance.darkSubtitle' => '始终使用深色主题',
			'settings.account.section' => '账户',
			'settings.account.changeCredentials' => '修改凭据',
			'settings.account.changeCredentialsSubtitle' => '用户名和密码',
			'settings.gateway.section' => '网关',
			'settings.gateway.serverSettings' => '服务器设置',
			'settings.gateway.serverSettingsSubtitle' => '监听地址、日志、凭据库、内存、存储路径…',
			'settings.gateway.liveLogs' => '实时日志',
			'settings.gateway.liveLogsSubtitle' => '查看网关实时日志 — 与 Web 管理端同源',
			_ => null,
		};
	}
}
