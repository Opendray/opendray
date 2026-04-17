class ModelDef {
  final String id;
  final String name;
  final String? description;

  const ModelDef({required this.id, required this.name, this.description});

  factory ModelDef.fromJson(Map<String, dynamic> json) => ModelDef(
        id: json['id'] as String,
        name: json['name'] as String,
        description: json['description'] as String?,
      );
}

class ConfigField {
  final String key;
  final String label;
  final String type; // string | secret | select | number | boolean | args
  final String? description;
  final String? placeholder;
  final dynamic defaultValue;
  final List<dynamic>? options;
  final bool required;
  final String? envVar;
  final String? cliFlag;
  final bool cliValue;
  final String? group;
  final String? dependsOn;
  final String? dependsVal;

  const ConfigField({
    required this.key,
    required this.label,
    required this.type,
    this.description,
    this.placeholder,
    this.defaultValue,
    this.options,
    this.required = false,
    this.envVar,
    this.cliFlag,
    this.cliValue = false,
    this.group,
    this.dependsOn,
    this.dependsVal,
  });

  factory ConfigField.fromJson(Map<String, dynamic> json) => ConfigField(
        key: json['key'] as String,
        label: json['label'] as String,
        type: json['type'] as String,
        description: json['description'] as String?,
        placeholder: json['placeholder'] as String?,
        defaultValue: json['default'],
        options: json['options'] as List<dynamic>?,
        required: json['required'] as bool? ?? false,
        envVar: json['envVar'] as String?,
        cliFlag: json['cliFlag'] as String?,
        cliValue: json['cliValue'] as bool? ?? false,
        group: json['group'] as String?,
        dependsOn: json['dependsOn'] as String?,
        dependsVal: json['dependsVal'] as String?,
      );
}

class Capabilities {
  final List<ModelDef> models;
  final bool supportsResume;
  final bool supportsStream;
  final bool supportsImages;
  final bool supportsMcp;
  final bool dynamicModels;

  const Capabilities({
    this.models = const [],
    this.supportsResume = false,
    this.supportsStream = false,
    this.supportsImages = false,
    this.supportsMcp = false,
    this.dynamicModels = false,
  });

  factory Capabilities.fromJson(Map<String, dynamic> json) => Capabilities(
        models: (json['models'] as List<dynamic>?)
                ?.map((e) => ModelDef.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [],
        supportsResume: json['supportsResume'] as bool? ?? false,
        supportsStream: json['supportsStream'] as bool? ?? false,
        supportsImages: json['supportsImages'] as bool? ?? false,
        supportsMcp: json['supportsMcp'] as bool? ?? false,
        dynamicModels: json['dynamicModels'] as bool? ?? false,
      );
}

class Provider {
  final String name;
  final String displayName;
  final String description;
  final String icon;
  final String version;
  final String type; // cli | local | shell | panel
  final String category; // for panels: docs | files | custom
  final Capabilities capabilities;
  final List<ConfigField> configSchema;

  const Provider({
    required this.name,
    required this.displayName,
    required this.description,
    required this.icon,
    required this.version,
    required this.type,
    this.category = '',
    required this.capabilities,
    this.configSchema = const [],
  });

  factory Provider.fromJson(Map<String, dynamic> json) => Provider(
        name: json['name'] as String,
        displayName: json['displayName'] as String? ?? json['name'] as String,
        description: json['description'] as String? ?? '',
        icon: json['icon'] as String? ?? '?',
        version: json['version'] as String? ?? '0.0.0',
        type: json['type'] as String? ?? 'cli',
        category: json['category'] as String? ?? '',
        capabilities: Capabilities.fromJson(
            json['capabilities'] as Map<String, dynamic>? ?? {}),
        configSchema: (json['configSchema'] as List<dynamic>?)
                ?.map((e) => ConfigField.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [],
      );
}

class ProviderInfo {
  final Provider provider;
  final Map<String, dynamic> config;
  final bool installed;
  final bool enabled;

  const ProviderInfo({
    required this.provider,
    this.config = const {},
    this.installed = false,
    this.enabled = false,
  });

  factory ProviderInfo.fromJson(Map<String, dynamic> json) => ProviderInfo(
        provider:
            Provider.fromJson(json['provider'] as Map<String, dynamic>),
        config: (json['config'] as Map<String, dynamic>?) ?? {},
        installed: json['installed'] as bool? ?? false,
        enabled: json['enabled'] as bool? ?? false,
      );
}
