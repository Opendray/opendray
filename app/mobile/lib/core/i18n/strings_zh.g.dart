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
