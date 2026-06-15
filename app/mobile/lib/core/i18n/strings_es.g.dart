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
class TranslationsEs extends Translations with BaseTranslations<AppLocale, Translations> {
	/// You can call this constructor and build your own translation instance of this locale.
	/// Constructing via the enum [AppLocale.build] is preferred.
	TranslationsEs({Map<String, Node>? overrides, PluralResolver? cardinalResolver, PluralResolver? ordinalResolver, TranslationMetadata<AppLocale, Translations>? meta})
		: assert(overrides == null, 'Set "translation_overrides: true" in order to enable this feature.'),
		  $meta = meta ?? TranslationMetadata(
		    locale: AppLocale.es,
		    overrides: overrides ?? {},
		    cardinalResolver: cardinalResolver,
		    ordinalResolver: ordinalResolver,
		  ),
		  super(cardinalResolver: cardinalResolver, ordinalResolver: ordinalResolver) {
		super.$meta.setFlatMapFunction($meta.getTranslation); // copy base translations to super.$meta
		$meta.setFlatMapFunction(_flatMapFunction);
	}

	/// Metadata for the translations of <es>.
	@override final TranslationMetadata<AppLocale, Translations> $meta;

	/// Access flat map
	@override dynamic operator[](String key) => $meta.getTranslation(key) ?? super.$meta.getTranslation(key);

	late final TranslationsEs _root = this; // ignore: unused_field

	@override 
	TranslationsEs $copyWith({TranslationMetadata<AppLocale, Translations>? meta}) => TranslationsEs(meta: meta ?? this.$meta);

	// Translations
	@override late final _TranslationsCommonEs common = _TranslationsCommonEs._(_root);
	@override late final _TranslationsAuthEs auth = _TranslationsAuthEs._(_root);
	@override late final _TranslationsNavEs nav = _TranslationsNavEs._(_root);
	@override late final _TranslationsWebEs web = _TranslationsWebEs._(_root);
	@override late final _TranslationsMoreEs more = _TranslationsMoreEs._(_root);
	@override late final _TranslationsActivityEs activity = _TranslationsActivityEs._(_root);
	@override late final _TranslationsMemoryAmbientEs memoryAmbient = _TranslationsMemoryAmbientEs._(_root);
	@override late final _TranslationsSessionsEs sessions = _TranslationsSessionsEs._(_root);
	@override late final _TranslationsMcpEs mcp = _TranslationsMcpEs._(_root);
	@override late final _TranslationsProvidersEs providers = _TranslationsProvidersEs._(_root);
	@override late final _TranslationsIntegrationsEs integrations = _TranslationsIntegrationsEs._(_root);
	@override late final _TranslationsMemoryWorkersEs memoryWorkers = _TranslationsMemoryWorkersEs._(_root);
	@override late final _TranslationsMemoryArchivedEs memoryArchived = _TranslationsMemoryArchivedEs._(_root);
	@override late final _TranslationsProjectEs project = _TranslationsProjectEs._(_root);
	@override late final _TranslationsBackupsEs backups = _TranslationsBackupsEs._(_root);
	@override late final _TranslationsBackupTargetsEs backupTargets = _TranslationsBackupTargetsEs._(_root);
	@override late final _TranslationsBackupSchedulesEs backupSchedules = _TranslationsBackupSchedulesEs._(_root);
	@override late final _TranslationsBackupTargetEditorEs backupTargetEditor = _TranslationsBackupTargetEditorEs._(_root);
	@override late final _TranslationsGithostsEs githosts = _TranslationsGithostsEs._(_root);
	@override late final _TranslationsChannelsEs channels = _TranslationsChannelsEs._(_root);
	@override late final _TranslationsOnboardingEs onboarding = _TranslationsOnboardingEs._(_root);
	@override late final _TranslationsSkillsEs skills = _TranslationsSkillsEs._(_root);
	@override late final _TranslationsCustomTasksEs customTasks = _TranslationsCustomTasksEs._(_root);
	@override late final _TranslationsNotesPageEs notesPage = _TranslationsNotesPageEs._(_root);
	@override late final _TranslationsDataExportEs dataExport = _TranslationsDataExportEs._(_root);
	@override late final _TranslationsMemoryEs memory = _TranslationsMemoryEs._(_root);
	@override late final _TranslationsAboutEs about = _TranslationsAboutEs._(_root);
	@override late final _TranslationsSettingsEs settings = _TranslationsSettingsEs._(_root);
	@override late final _TranslationsMemoryQuarantineEs memoryQuarantine = _TranslationsMemoryQuarantineEs._(_root);
	@override late final _TranslationsCortexHubEs cortexHub = _TranslationsCortexHubEs._(_root);
	@override late final _TranslationsCortexSettingsEs cortexSettings = _TranslationsCortexSettingsEs._(_root);
}

// Path: common
class _TranslationsCommonEs extends TranslationsCommonEn {
	_TranslationsCommonEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get ok => 'OK';
	@override String get cancel => 'Cancelar';
	@override String get save => 'Guardar';
	@override String get delete => 'Eliminar';
	@override String get edit => 'Editar';
	@override String get back => 'Atrás';
	@override String get done => 'Hecho';
	@override String get close => 'Cerrar';
	@override String get retry => 'Reintentar';
	@override String get copy => 'Copiar';
	@override String get enabled => 'Activado';
	@override String get refresh => 'Actualizar';
	@override String get clear => 'Limpiar';
}

// Path: auth
class _TranslationsAuthEs extends TranslationsAuthEn {
	_TranslationsAuthEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get signInTitle => 'Iniciar sesión';
	@override String get changeServer => 'Cambiar';
	@override String get username => 'Usuario';
	@override String get password => 'Contraseña';
	@override String get signIn => 'Iniciar sesión';
	@override String get signingIn => 'Iniciando sesión…';
	@override String get subtitle => 'Usa tus credenciales de operador.';
	@override String get errorRequired => 'El usuario y la contraseña son obligatorios';
	@override String errorGeneric({required Object error}) => 'Error al iniciar sesión: ${error}';
	@override String get errorFallback => 'Error al iniciar sesión';
}

// Path: nav
class _TranslationsNavEs extends TranslationsNavEn {
	_TranslationsNavEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get sessions => 'Sessions';
	@override String get memory => 'Memoria';
	@override String get notes => 'Notas';
	@override String get more => 'Más';
	@override String get activity => 'Actividad';
	@override String get providers => 'Proveedores';
	@override String get channels => 'Canales';
	@override String get integrations => 'Integraciones';
	@override String get plugins => 'Plugins';
	@override String get backups => 'Copias de seguridad';
	@override String get settings => 'Ajustes';
	@override String get workspace => 'Espacio de trabajo';
	@override String get knowledge => 'Conocimiento';
	@override String get vault => 'Bóveda';
	@override String get cortex => 'Cortex';
	@override String get updateAvailable => 'Actualización disponible';
}

// Path: web
class _TranslationsWebEs extends TranslationsWebEn {
	_TranslationsWebEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get brand => 'opendray';
	@override String get loading => 'Cargando…';
	@override late final _TranslationsWebTopbarEs topbar = _TranslationsWebTopbarEs._(_root);
	@override late final _TranslationsWebSessionsEs sessions = _TranslationsWebSessionsEs._(_root);
	@override late final _TranslationsWebMemoryEs memory = _TranslationsWebMemoryEs._(_root);
	@override late final _TranslationsWebJournalStaleEs journalStale = _TranslationsWebJournalStaleEs._(_root);
	@override late final _TranslationsWebConflictsEs conflicts = _TranslationsWebConflictsEs._(_root);
	@override late final _TranslationsWebMemoryHealthEs memoryHealth = _TranslationsWebMemoryHealthEs._(_root);
	@override late final _TranslationsWebMemoryConfigEs memoryConfig = _TranslationsWebMemoryConfigEs._(_root);
	@override late final _TranslationsWebMemoryWorkersEs memoryWorkers = _TranslationsWebMemoryWorkersEs._(_root);
	@override late final _TranslationsWebArchivedEs archived = _TranslationsWebArchivedEs._(_root);
	@override late final _TranslationsWebProjectEs project = _TranslationsWebProjectEs._(_root);
	@override late final _TranslationsWebMemoryInspectorEs memoryInspector = _TranslationsWebMemoryInspectorEs._(_root);
	@override late final _TranslationsWebNotesEs notes = _TranslationsWebNotesEs._(_root);
	@override late final _TranslationsWebActivityEs activity = _TranslationsWebActivityEs._(_root);
	@override late final _TranslationsWebProvidersEs providers = _TranslationsWebProvidersEs._(_root);
	@override late final _TranslationsWebChannelsEs channels = _TranslationsWebChannelsEs._(_root);
	@override late final _TranslationsWebIntegrationsEs integrations = _TranslationsWebIntegrationsEs._(_root);
	@override late final _TranslationsWebPluginsEs plugins = _TranslationsWebPluginsEs._(_root);
	@override late final _TranslationsWebBackupsEs backups = _TranslationsWebBackupsEs._(_root);
	@override late final _TranslationsWebServerSettingsEs serverSettings = _TranslationsWebServerSettingsEs._(_root);
	@override late final _TranslationsWebSettingsEs settings = _TranslationsWebSettingsEs._(_root);
	@override late final _TranslationsWebLogViewerEs logViewer = _TranslationsWebLogViewerEs._(_root);
	@override late final _TranslationsWebPathInputEs pathInput = _TranslationsWebPathInputEs._(_root);
	@override late final _TranslationsWebMemoryAmbientEs memoryAmbient = _TranslationsWebMemoryAmbientEs._(_root);
	@override late final _TranslationsWebNoteEditorEs noteEditor = _TranslationsWebNoteEditorEs._(_root);
	@override late final _TranslationsWebExportEs export = _TranslationsWebExportEs._(_root);
	@override late final _TranslationsWebKnowledgeEs knowledge = _TranslationsWebKnowledgeEs._(_root);
	@override late final _TranslationsWebCortexEs cortex = _TranslationsWebCortexEs._(_root);
}

// Path: more
class _TranslationsMoreEs extends TranslationsMoreEn {
	_TranslationsMoreEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Más';
	@override late final _TranslationsMoreIdentityEs identity = _TranslationsMoreIdentityEs._(_root);
	@override late final _TranslationsMoreSectionsEs sections = _TranslationsMoreSectionsEs._(_root);
	@override late final _TranslationsMoreItemsEs items = _TranslationsMoreItemsEs._(_root);
	@override String get signOut => 'Cerrar sesión';
}

// Path: activity
class _TranslationsActivityEs extends TranslationsActivityEn {
	_TranslationsActivityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Actividad';
	@override String get empty => 'Aún no hay llamadas de integración registradas.';
	@override String get loadFailed => 'Error al cargar la actividad';
	@override String callsCount({required Object count}) => '${count} llamadas';
	@override String get directionInbound => 'entrante';
	@override String get directionOutbound => 'saliente';
	@override late final _TranslationsActivityFilterEs filter = _TranslationsActivityFilterEs._(_root);
	@override late final _TranslationsActivityDetailEs detail = _TranslationsActivityDetailEs._(_root);
}

// Path: memoryAmbient
class _TranslationsMemoryAmbientEs extends TranslationsMemoryAmbientEn {
	_TranslationsMemoryAmbientEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Captura e inyección';
	@override String get intro => 'Cómo se resumen las sesiones en memoria y qué contexto se precarga. La creación de reglas y la edición detallada están en el panel web.';
	@override String get captureSection => 'Reglas de captura';
	@override String get injectionSection => 'Perfiles de inyección';
	@override String get empty => 'Nada configurado aún.';
	@override String get loadFailed => 'Error al cargar';
	@override String get runNow => 'Ejecutar ahora';
	@override String ranSnack({required Object count}) => 'Ejecutado en ${count} sesión(es)';
	@override String actionFailed({required Object error}) => 'Acción fallida: ${error}';
	@override String get strategyLabel => 'Estrategia';
	@override String get scopeProject => 'proyecto';
	@override String get scopeGlobal => 'global';
	@override String get triggerAfterMessages => 'Tras N mensajes';
	@override String get triggerOnIdle => 'En inactividad';
	@override String get triggerKChars => 'Tras K caracteres';
	@override String get triggerManual => 'Manual';
	@override String get triggerUnknown => 'Desconocido';
	@override String get strategyNone => 'Ninguna (búsqueda bajo demanda)';
	@override String get strategyTopKRecent => 'Top-K recientes';
	@override String get strategyTopKRelevant => 'Top-K relevantes';
	@override String get strategyOnKeyword => 'Por palabra clave';
	@override String get strategyManualOnly => 'Solo manual';
	@override String get strategyHybrid => 'Resumen híbrido';
	@override String get strategyUnknown => 'Desconocido';
}

// Path: sessions
class _TranslationsSessionsEs extends TranslationsSessionsEn {
	_TranslationsSessionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Sesiones';
	@override String get refresh => 'Actualizar';
	@override String get actions => 'Acciones';
	@override String get spawn => 'Crear';
	@override late final _TranslationsSessionsFiltersEs filters = _TranslationsSessionsFiltersEs._(_root);
	@override late final _TranslationsSessionsCardEs card = _TranslationsSessionsCardEs._(_root);
	@override late final _TranslationsSessionsEmptyEs empty = _TranslationsSessionsEmptyEs._(_root);
	@override String get errorTitle => 'No se pudieron cargar las sesiones';
	@override late final _TranslationsSessionsRelativeEs relative = _TranslationsSessionsRelativeEs._(_root);
	@override late final _TranslationsSessionsDetailEs detail = _TranslationsSessionsDetailEs._(_root);
	@override late final _TranslationsSessionsTerminalEs terminal = _TranslationsSessionsTerminalEs._(_root);
	@override late final _TranslationsSessionsActionEs action = _TranslationsSessionsActionEs._(_root);
	@override late final _TranslationsSessionsDirPickerEs dirPicker = _TranslationsSessionsDirPickerEs._(_root);
	@override late final _TranslationsSessionsInspectorEs inspector = _TranslationsSessionsInspectorEs._(_root);
	@override late final _TranslationsSessionsSpawnSheetEs spawnSheet = _TranslationsSessionsSpawnSheetEs._(_root);
}

// Path: mcp
class _TranslationsMcpEs extends TranslationsMcpEn {
	_TranslationsMcpEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'MCP';
	@override String get newServer => 'Nuevo servidor';
	@override String get addSecret => 'Añadir secreto';
	@override String get editConfig => 'Editar configuración';
	@override String get viewRawConfig => 'Ver configuración en bruto';
	@override String get copyId => 'Copiar id';
	@override String copiedSnack({required Object id}) => 'Copiado ${id}';
	@override String get deleteServerTitle => '¿Eliminar servidor MCP?';
	@override String get deleteSecretTitle => '¿Eliminar secreto?';
	@override late final _TranslationsMcpErrorPrefixEs errorPrefix = _TranslationsMcpErrorPrefixEs._(_root);
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';
	@override late final _TranslationsMcpEditorEs editor = _TranslationsMcpEditorEs._(_root);
	@override late final _TranslationsMcpSecretEs secret = _TranslationsMcpSecretEs._(_root);
	@override late final _TranslationsMcpPopupEs popup = _TranslationsMcpPopupEs._(_root);
	@override late final _TranslationsMcpKvEs kv = _TranslationsMcpKvEs._(_root);
	@override String deleteServerBody({required Object id}) => 'Elimina el directorio del vault para ${id}. Las sesiones que referencian este servidor dejan de poder iniciarlo.';
	@override String deleteServerSnack({required Object id}) => 'Eliminado ${id}.';
	@override String serversCount({required Object count}) => 'Servidores (${count})';
	@override String secretsCount({required Object count}) => 'Secretos (${count})';
	@override String get emptyServers => 'No hay servidores MCP registrados. Toca "Nuevo servidor" para añadir uno.';
	@override String get emptySecrets => 'No hay secretos almacenados. Añade uno para pasar variables de entorno / headers sensibles a los servidores MCP sin ponerlos en el JSON.';
	@override String get noVaultFileYet => 'Aún no hay archivo de vault, los secretos añadidos lo crean.';
	@override String get tapToReplaceHint => 'Toca para reemplazar · mantén pulsado / papelera para eliminar';
	@override String get failedToLoad => 'No se pudo cargar el estado de MCP';
	@override String get serverCreatedSnack => 'Servidor MCP creado.';
	@override String get serverUpdatedSnack => 'Servidor MCP actualizado.';
	@override String get envHeading => 'Env';
	@override String get encryptionAes => 'Cifrado AES-GCM (clave en el keychain del SO)';
	@override String get encryptionPlaintext => 'PLAINTEXT, keychain no disponible';
	@override String toggleEnabledSnack({required Object name}) => '${name} activado.';
	@override String toggleDisabledSnack({required Object name}) => '${name} desactivado.';
	@override String get builtinBadge => 'integrado';
	@override String get builtinAlwaysOn => 'siempre activo';
	@override String get builtinHint => 'Provisto por opendray — se adjunta a cada session. No se puede editar ni eliminar.';
}

// Path: providers
class _TranslationsProvidersEs extends TranslationsProvidersEn {
	_TranslationsProvidersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Proveedores';
	@override String get configSaved => 'Configuración del proveedor actualizada.';
	@override String saveFailedApi({required Object error}) => 'Error al guardar: ${error}';
	@override String saveFailedGeneric({required Object error}) => 'Error al guardar: ${error}';
	@override String get reload => 'Recargar';
	@override late final _TranslationsProvidersErrorPrefixEs errorPrefix = _TranslationsProvidersErrorPrefixEs._(_root);
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';
	@override late final _TranslationsProvidersUpdateCheckEs updateCheck = _TranslationsProvidersUpdateCheckEs._(_root);
	@override late final _TranslationsProvidersAccountsEs accounts = _TranslationsProvidersAccountsEs._(_root);
	@override String get configFallbackTitle => 'Configuración del proveedor';
	@override String get saving => 'Guardando…';
	@override String get save => 'Guardar';
	@override String get configLoadFailed => 'Error al cargar el proveedor';
	@override String get argsHelper => 'Argumentos de CLI separados por espacios.';
	@override String get listEmptyHeadline => 'No hay proveedores cargados.';
	@override String get listEmptyBody => 'El gateway resuelve los proveedores desde su directorio de plugins al arrancar. Revisa los logs si esperabas alguno.';
	@override String get listLoadFailed => 'Error al cargar los proveedores';
	@override String get cliSectionHeader => 'Proveedores de CLI';
	@override String enabledSnack({required Object name}) => '${name} activada.';
	@override String disabledSnack({required Object name}) => '${name} desactivada.';
}

// Path: integrations
class _TranslationsIntegrationsEs extends TranslationsIntegrationsEn {
	_TranslationsIntegrationsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Integraciones';
	@override String get register => 'Registrar';
	@override String get registerDialogTitle => 'Registrar integración';
	@override String get edit => 'Editar';
	@override String editTitle({required Object name}) => 'Editar ${name}';
	@override String get enabledLabel => 'Habilitada';
	@override String get iSavedIt => 'Ya la he guardado';
	@override String apiKeyForName({required Object name}) => 'API key de ${name}';
	@override String apiKeySubtitleRegister({required Object routePrefix}) => 'Entrégasela a la integración para que pueda autenticarse contra /api/v1/${routePrefix}/…';
	@override String copiedRequestId({required Object id}) => 'request_id ${id} copiado';
	@override String get updateOk => 'Integración actualizada.';
	@override String registerFailedApi({required Object error}) => 'Error al registrar: ${error}';
	@override String registerFailedGeneric({required Object error}) => 'Error al registrar: ${error}';
	@override String updateFailedApi({required Object error}) => 'Error al actualizar: ${error}';
	@override String updateFailedGeneric({required Object error}) => 'Error al actualizar: ${error}';
	@override String get deleteTitle => '¿Eliminar integración?';
	@override String deletedSnack({required Object name}) => '${name} eliminada.';
	@override String deleteFailedApi({required Object error}) => 'Error al eliminar: ${error}';
	@override String deleteFailedGeneric({required Object error}) => 'Error al eliminar: ${error}';
	@override String get rotateKey => 'Rotar key';
	@override String get rotateConfirmTitle => '¿Rotar la API key?';
	@override String get rotate => 'Rotar';
	@override String newApiKeyTitle({required Object name}) => 'Nueva API key de ${name}';
	@override String get newApiKeySubtitle => 'Entrégasela a la integración. La key anterior acaba de quedar invalidada.';
	@override String rotateFailedApi({required Object error}) => 'Error al rotar: ${error}';
	@override String rotateFailedGeneric({required Object error}) => 'Error al rotar: ${error}';
	@override String get deleteBody => 'Elimina el registro y revoca la API key. Las solicitudes en curso que usen la key antigua empezarán a fallar.';
	@override String rotateBody({required Object name}) => 'Genera una nueva API key para ${name} e invalida la antigua de inmediato.';
	@override String get appBarFallback => 'Integración';
	@override String get tooltipMore => 'Más';
	@override String get tooltipReadOnly => 'Integración del sistema (solo lectura)';
	@override String get kvRoutePrefix => 'Prefijo de ruta';
	@override String get kvBaseUrl => 'URL base';
	@override String get kvScopes => 'Ámbitos';
	@override String get kvVersion => 'Versión';
	@override String get kvLastHealthPing => 'Último ping de estado';
	@override String get kvCreated => 'Creada';
	@override String get kvKeyRotated => 'Key rotada';
	@override String detailLoadFailed({required Object error}) => 'Error al cargar la integración: ${error}';
	@override String get callsLoadFailed => 'Error al cargar las llamadas';
	@override String get noMatchingCalls => 'Aún no hay llamadas coincidentes en el registro.';
	@override String get directionAll => 'Todas';
	@override String get directionInbound => 'Entrantes';
	@override String get directionOutbound => 'Salientes';
	@override late final _TranslationsIntegrationsFormEs form = _TranslationsIntegrationsFormEs._(_root);
	@override String get emptyState => 'Regístrala desde el admin web: Integraciones → Nueva.';
	@override String get sectionRegistered => 'Registradas';
	@override String get sectionSystem => 'Sistema';
	@override String get listLoadFailed => 'Error al cargar las integraciones';
}

// Path: memoryWorkers
class _TranslationsMemoryWorkersEs extends TranslationsMemoryWorkersEn {
	_TranslationsMemoryWorkersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Workers de memoria';
	@override String savedSnack({required Object label}) => '${label} guardado';
	@override String saveFailed({required Object error}) => 'Error al guardar: ${error}';
	@override String testFailed({required Object error}) => 'La llamada de prueba falló: ${error}';
	@override String get workerLabel => 'Worker';
	@override String get summarizerHttp => 'Resumidor (HTTP)';
	@override String get agentCliPrint => 'Agente (CLI --print)';
	@override String get cliLabel => 'CLI';
	@override String get cliClaude => 'Claude';
	@override String get cliGemini => 'Gemini';
	@override String get cliCodex => 'Codex (codex exec)';
	@override String get cliAntigravity => 'Antigravity (agy --print)';
	@override String get modelLabel => 'Modelo';
	@override String get modelCliDefault => 'Predeterminado del CLI (último)';
	@override String get modelCustom => 'Personalizado…';
	@override String get modelCustomPlaceholder => 'id de modelo exacto';
	@override String get modelBackToList => 'Lista';
	@override String get claudeAccountLabel => 'Cuenta de Claude';
	@override String get claudeAccountDefault => 'Predeterminada';
	@override String get test => 'Probar';
	@override String get intro => 'Cada punto de contacto con el LLM del sistema de memoria puede atenderse de forma independiente mediante el endpoint del resumidor local (LM Studio / compatible con OpenAI) o lanzando un agente headless de Claude / Gemini en modo --print. Las tareas narrativas de alta calidad (gitactivity, transcript) se benefician de los workers de agente; las tareas de alta frecuencia (gatekeeper) permanecen en el endpoint local por diseño.';
	@override String get errorTitle => 'Endpoint no accesible';
	@override String get errorDetail => 'Las rutas /api/v1/memory/workers son nuevas en M25. Puede que el binario de opendray necesite un reinicio para montarlas y ejecutar la migración 0029.';
	@override String get summarizerOnlyBadge => 'solo resumidor';
	@override String get summarizerProviderLabel => 'Proveedor de resumidor';
	@override String get registryDefault => 'Predeterminado del registro';
	@override String get agentWarning => 'El modo agente lanza un CLI headless por cada llamada. Latencia ~5-15s (frente a ~1s del resumidor); el coste pasa de la CPU a tu quota de Claude/Gemini.';
	@override String get noCalls24h => 'Sin llamadas en las últimas 24h.';
	@override String testOkSnack({required Object label, required Object duration}) => '${label} OK, ${duration}ms';
	@override String testFailedReturnedSnack({required Object label, required Object error}) => '${label} falló: ${error}';
	@override String get unknownError => 'desconocido';
	@override late final _TranslationsMemoryWorkersTasksEs tasks = _TranslationsMemoryWorkersTasksEs._(_root);
}

// Path: memoryArchived
class _TranslationsMemoryArchivedEs extends TranslationsMemoryArchivedEn {
	_TranslationsMemoryArchivedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Memorias archivadas';
	@override String loadFailed({required Object error}) => 'Error al cargar: ${error}';
	@override String restoreFailed({required Object error}) => 'Error al restaurar: ${error}';
	@override String get emptyTitle => 'Nada archivado';
	@override String get emptyBody => 'No hay memorias archivadas en ningún proyecto. El limpiador automático archiva aquí los hechos obsoletos y duplicados (restaurables durante 30 días); todavía no se ha eliminado nada.';
	@override String get globalScope => '(global)';
	@override String countBadge({required Object count}) => '${count} archivadas';
	@override String get restore => 'Restaurar';
	@override String get restoreAll => 'Restaurar todo';
	@override String get deleteAll => 'Eliminar todo';
	@override String restoreAllConfirm({required Object count, required Object project}) => '¿Restaurar las ${count} memorias archivadas de ${project}?';
	@override String deleteAllConfirm({required Object count, required Object project}) => '¿Eliminar permanentemente las ${count} memorias archivadas de ${project}? Omite la ventana de gracia de 30 días y no se puede deshacer.';
	@override String get deletePermanently => 'Eliminar';
	@override String get deleteConfirm => '¿Eliminar permanentemente esta memoria ahora? Omite la ventana de gracia de 30 días y no se puede deshacer.';
	@override String get restoredToast => 'Restaurada';
	@override String restoredAllToast({required Object count}) => '${count} memorias restauradas';
	@override String get deletedToast => 'Eliminada permanentemente';
	@override String deletedAllToast({required Object count}) => '${count} memorias eliminadas';
	@override String deleteFailed({required Object error}) => 'Error al eliminar: ${error}';
	@override String summary({required Object projects, required Object memories}) => '${projects} proyectos · ${memories} archivadas';
}

// Path: project
class _TranslationsProjectEs extends TranslationsProjectEn {
	_TranslationsProjectEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Proyecto';
	@override String get pickFirst => 'Elige primero un proyecto.';
	@override late final _TranslationsProjectHealthEs health = _TranslationsProjectHealthEs._(_root);
	@override late final _TranslationsProjectConflictsEs conflicts = _TranslationsProjectConflictsEs._(_root);
	@override late final _TranslationsProjectJournalPruneEs journalPrune = _TranslationsProjectJournalPruneEs._(_root);
	@override String loadFailed({required Object error}) => 'Error al cargar: ${error}';
	@override String projectsLoadFailed({required Object error}) => 'Error al cargar los proyectos: ${error}';
	@override String get projectLabel => 'Proyecto';
	@override String get browseFolder => 'Explorar carpeta…';
	@override String get resetTooltip => 'Restablecer la memoria del proyecto';
	@override String get append => 'Añadir';
	@override String get appendDialogTitle => 'Añadir entrada de diario';
	@override String get titleFieldLabel => 'Título (opcional)';
	@override String get contentFieldLabel => 'Contenido (markdown)';
	@override String appendFailed({required Object error}) => 'Error: ${error}';
	@override String approveFailed({required Object error}) => 'Error al aprobar: ${error}';
	@override String rejectFailed({required Object error}) => 'Error al rechazar: ${error}';
	@override String get resetConfirmTitle => '¿Restablecer la memoria del proyecto?';
	@override String get alsoDeleteScanner => 'Eliminar también los documentos del scanner';
	@override String get alsoDeletePgvector => 'Eliminar también las memorias de pgvector';
	@override String get deleteForever => 'Eliminar para siempre';
	@override String resetDoneSnack({required Object parts}) => 'Restablecido: ${parts}';
	@override String resetFailed({required Object error}) => 'Error al restablecer: ${error}';
	@override String docSavedSnack({required Object kind}) => '${kind} guardado';
	@override String docSaveFailed({required Object error}) => 'Error al guardar: ${error}';
	@override String docHintTemplate({required Object kind}) => 'Escribe el ${kind} como markdown…';
	@override String get deleteEntryTooltip => 'Eliminar entrada';
	@override String get agentReason => 'Motivo del agente';
	@override String get reject => 'Rechazar';
	@override String get approve => 'Aprobar';
	@override String replaceConfirmTitle({required Object kind}) => '¿Reemplazar el ${kind} actual?';
	@override String replaceKind({required Object kind}) => 'Reemplazar ${kind}';
	@override late final _TranslationsProjectArchivedEs archived = _TranslationsProjectArchivedEs._(_root);
}

// Path: backups
class _TranslationsBackupsEs extends TranslationsBackupsEn {
	_TranslationsBackupsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Copias de seguridad';
	@override String get runConfirmTitle => '¿Ejecutar copia de seguridad ahora?';
	@override String get runConfirmBody => 'Lanza un nuevo volcado contra el destino local. El trabajo se ejecuta en el servidor; esta lista se actualizará a medida que avance.';
	@override String get runFullInstance => 'Instancia completa';
	@override String get runFullInstanceHint => 'Incluye también el vault, secrets.env y config.toml, no solo la base de datos.';
	@override String get kindDbOnly => 'Solo BD';
	@override String get kindFullInstance => 'Instancia completa';
	@override String get dedupValue => 'reutilizó el blob existente (contenido idéntico)';
	@override String get verifyOk => 'verificada';
	@override String get verifyFailed => 'sin verificar (falló la comprobación)';
	@override String get verifyPending => 'sin verificar';
	@override String get run => 'Ejecutar';
	@override String get runNow => 'Ejecutar ahora';
	@override String get queueing => 'Encolando…';
	@override String queuedSnack({required Object id}) => 'Copia de seguridad encolada (${id}). Esperando el progreso…';
	@override String runFailedApi({required Object error}) => 'Error al ejecutar: ${error}';
	@override String runFailedGeneric({required Object error}) => 'Error al ejecutar: ${error}';
	@override String rowSucceededSnack({required Object bytes}) => 'Copia de seguridad completada (${bytes}).';
	@override String rowFailedSnack({required Object error}) => 'Error en la copia de seguridad: ${error}';
	@override String get unknownError => 'error desconocido';
	@override String get detailTitle => 'Detalle de la copia de seguridad';
	@override String get deleteTitle => '¿Eliminar copia de seguridad?';
	@override String deleteBody({required Object target}) => 'Elimina el blob de ${target} y marca la fila como eliminada en el índice.';
	@override String deletedSnack({required Object id}) => 'Eliminado ${id}.';
	@override String deleteFailedApi({required Object error}) => 'Error al eliminar: ${error}';
	@override String deleteFailedGeneric({required Object error}) => 'Error al eliminar: ${error}';
	@override String get menuSchedules => 'Programaciones';
	@override String get menuTargets => 'Destinos';
	@override late final _TranslationsBackupsKvEs kv = _TranslationsBackupsKvEs._(_root);
	@override late final _TranslationsBackupsRecoveryKitEs recoveryKit = _TranslationsBackupsRecoveryKitEs._(_root);
	@override late final _TranslationsBackupsEmptyMissingDepsEs emptyMissingDeps = _TranslationsBackupsEmptyMissingDepsEs._(_root);
	@override late final _TranslationsBackupsEmptyNoTargetsEs emptyNoTargets = _TranslationsBackupsEmptyNoTargetsEs._(_root);
	@override late final _TranslationsBackupsEmptyNoBackupsEs emptyNoBackups = _TranslationsBackupsEmptyNoBackupsEs._(_root);
	@override String get restartToActivate => 'Reinicia opendray para activar las copias de seguridad';
	@override String get passphraseSaved => 'Tu passphrase está guardada. El gateway solo la carga al iniciarse, así que los cambios solo surten efecto tras un reinicio.';
	@override String get keyFileLabel => 'Archivo de clave';
	@override String get configuredViaLabel => 'Configurado mediante';
	@override late final _TranslationsBackupsWizardEs wizard = _TranslationsBackupsWizardEs._(_root);
	@override String get overviewTargets => 'Destinos';
	@override String get overviewSchedules => 'Programaciones';
	@override String get overviewBackups => 'Copias de seguridad';
	@override late final _TranslationsBackupsHealthEs health = _TranslationsBackupsHealthEs._(_root);
	@override String get failedToLoad => 'Error al cargar las copias de seguridad';
	@override String get envVarConfigured => 'variable de entorno OPENDRAY_BACKUP_KEY';
	@override String get savedConfirmCheckbox => 'He guardado esta passphrase en mi gestor de contraseñas';
	@override String get pgDumpMissing => 'pg_dump no está en el PATH. Instala postgresql-client y reinicia opendray.';
	@override late final _TranslationsBackupsEncryptionEs encryption = _TranslationsBackupsEncryptionEs._(_root);
	@override String get restoreFromFile => 'Restaurar desde archivo';
	@override late final _TranslationsBackupsRestoreEs restore = _TranslationsBackupsRestoreEs._(_root);
	@override late final _TranslationsBackupsInventoryEs inventory = _TranslationsBackupsInventoryEs._(_root);
}

// Path: backupTargets
class _TranslationsBackupTargetsEs extends TranslationsBackupTargetsEn {
	_TranslationsBackupTargetsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Destinos de copia de seguridad';
	@override String get newTarget => 'Nuevo destino';
	@override String get testConnection => 'Probar conexión';
	@override String get editConfig => 'Editar configuración';
	@override String get viewRawConfig => 'Ver configuración sin procesar';
	@override String configDialogTitle({required Object kind}) => 'Configuración de ${kind}';
	@override String get deleteTitle => '¿Eliminar destino?';
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';
}

// Path: backupSchedules
class _TranslationsBackupSchedulesEs extends TranslationsBackupSchedulesEn {
	_TranslationsBackupSchedulesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Programaciones de copia de seguridad';
	@override String get newButton => 'Nueva';
	@override String get deleteTitle => '¿Eliminar programación?';
	@override String get targetLabel => 'Destinos';
	@override String get targetsHint => 'Elige uno o más: la misma copia se escribe en cada destino (3-2-1).';
	@override String get intervalLabel => 'Intervalo';
	@override String get retentionLabel => 'Retención (conservar las N más recientes)';
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';
	@override String get noTargets => 'No hay destinos de copia de seguridad configurados. Añade uno desde el panel de administración web o la pantalla de Destinos.';
	@override String get okMsgCreate => 'Programación creada.';
	@override String get okMsgUpdate => 'Programación actualizada.';
	@override String get okMsgDelete => 'Programación eliminada.';
	@override String get errorPrefixCreate => 'Error al crear';
	@override String get errorPrefixUpdate => 'Error al actualizar';
	@override String get errorPrefixDelete => 'Error al eliminar';
	@override String deleteBody({required Object targetId}) => 'Elimina la especificación recurrente para el destino ${targetId}. Los blobs de copia de seguridad existentes no se modifican.';
	@override String get emptyList => 'Aún no hay programaciones.\nToca "Nueva" para crear una.';
	@override String get validatePickTarget => 'Elige un destino.';
	@override String get validateInterval => 'El intervalo debe ser > 0.';
	@override String get formTitleEdit => 'Editar programación';
	@override String get formTitleNew => 'Nueva programación';
	@override String get saveButtonEdit => 'Guardar';
	@override String get saveButtonNew => 'Crear';
	@override String get targetFixedHint => 'El destino queda fijado una vez creado.';
	@override String get enabledOn => 'El programador la ejecutará según la cadencia.';
	@override String get enabledOff => 'En pausa. No habrá ejecuciones automáticas hasta volver a activarla.';
	@override String get loadFailedTitle => 'Error al cargar las programaciones';
	@override String get pausedBadge => 'en pausa';
	@override String everyInterval({required Object interval}) => 'cada ${interval}';
	@override String keepRetention({required Object n}) => '· conservar ${n}';
	@override String nextRun({required Object when}) => '· siguiente ${when}';
	@override String lastRun({required Object when}) => '· última ${when}';
}

// Path: backupTargetEditor
class _TranslationsBackupTargetEditorEs extends TranslationsBackupTargetEditorEn {
	_TranslationsBackupTargetEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get useHttps => 'Usar HTTPS';
	@override String get pathStyle => 'Direccionamiento por ruta (path-style)';
	@override String get pathStyleSubtitle => 'Heredado / MinIO';
	@override late final _TranslationsBackupTargetEditorKindsEs kinds = _TranslationsBackupTargetEditorKindsEs._(_root);
	@override String get formTitleEdit => 'Editar destino';
	@override String get formTitleNew => 'Nuevo destino de backup';
	@override String idHintAuto({required Object prefix}) => 'Automático: ${prefix}-1';
	@override String get idHelper => 'Letras minúsculas, dígitos, guiones. Por defecto, la siguiente ranura disponible.';
	@override String get enabledOn => 'Los backups programados y puntuales pueden usar este destino.';
	@override String get enabledOff => 'El servidor se negará a escribir backups aquí.';
	@override String get saving => 'Guardando…';
	@override String get create => 'Crear';
	@override String get rootDirLabel => 'Directorio raíz';
	@override String get rootDirHint => 'Vacío = cfg.backup.local_dir (~/.opendray/backups)';
	@override String get hostLabel => 'Host';
	@override String get portLabel => 'Puerto';
	@override String get shareLabel => 'Recurso compartido';
	@override String get shareHint => 'Nombre del recurso compartido de nivel superior';
	@override String get shareSampleHint => 'Claude_Workspace';
	@override String get userLabel => 'Usuario';
	@override String get passwordLabel => 'Contraseña';
	@override String get passwordHintKeepCurrent => 'Déjalo en blanco para conservar la actual';
	@override String get passwordHintKeep => 'Déjalo en blanco para conservarla';
	@override String get pathPrefixLabel => 'Prefijo de ruta';
	@override String get pathPrefixHintShareRoot => 'Subcarpeta bajo la raíz del recurso compartido (opcional)';
	@override String get pathPrefixHintBaseUrl => 'Subcarpeta bajo la URL base (opcional)';
	@override String get pathPrefixHintObjectKey => 'Prefijo de clave de objeto (opcional)';
	@override String get pathPrefixHintSshFolder => 'Absoluta o relativa al home del usuario (opcional)';
	@override String get pathPrefixHintRemoteRoot => 'Subcarpeta bajo la raíz remota (opcional)';
	@override String get endpointLabel => 'Endpoint';
	@override String get regionLabel => 'Región';
	@override String get bucketLabel => 'Bucket';
	@override String get accessKeyLabel => 'Clave de acceso';
	@override String get secretKeyLabel => 'Clave secreta';
	@override String get secretKeyHintEdit => 'Déjalo en blanco para conservar la actual. Se almacena cifrada con AES-256-GCM.';
	@override String get secretKeyHintNew => 'Se almacena cifrada con AES-256-GCM; nunca se devuelve.';
	@override String get baseUrlLabel => 'URL base';
	@override String get baseUrlHint => 'URL completa incluyendo la ruta. Nextcloud: https://cloud.example/remote.php/dav/files/<user>';
	@override String get sftpPasswordHintEdit => 'Déjalo en blanco para conservarla. Si están presentes tanto la contraseña como la clave privada, prevalece la clave privada.';
	@override String get sftpPasswordHintNew => 'Contraseña O clave privada. Si están ambas, la contraseña pasa a ser solo un respaldo.';
	@override String get privateKeyLabel => 'Clave privada (PEM)';
	@override String get privateKeyHintEdit => 'Déjalo en blanco para conservarla. Pega el contenido OpenSSH/PEM.';
	@override String get privateKeyHintNew => 'Pega el contenido de una clave privada OpenSSH/PEM. Entrada de varias líneas: conserva los marcadores BEGIN/END.';
	@override String get hostKeyLabel => 'Clave de host (fijación)';
	@override String get hostKeyHint => 'Clave pública del servidor en formato OpenSSH. Usa `ssh-keyscan <host>` para obtenerla. En blanco = sin fijación (NO recomendado fuera de la LAN).';
	@override String get rcloneNote => 'Requiere la CLI de rclone en el host de opendray. Ejecuta primero `rclone config` una vez de forma interactiva para autenticar las cuentas en la nube.';
	@override String get rcloneRemoteLabel => 'Nombre del remoto';
	@override String get rcloneRemoteHint => 'Nombre de `rclone config` (sin los dos puntos).';
	@override String get rcloneBinaryLabel => 'Ruta del binario';
	@override String get rcloneBinaryHint => 'Anula `which rclone`. Vacío = búsqueda en PATH.';
	@override String get rcloneConfigLabel => 'Ruta de configuración';
	@override String get rcloneConfigHint => 'Anula --config. Vacío = valor por defecto de rclone.';
}

// Path: githosts
class _TranslationsGithostsEs extends TranslationsGithostsEn {
	_TranslationsGithostsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Hosts de Git';
	@override String get addHost => 'Añadir host';
	@override String get deleteTitle => '¿Eliminar host de Git?';
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';
	@override late final _TranslationsGithostsErrorPrefixEs errorPrefix = _TranslationsGithostsErrorPrefixEs._(_root);
	@override late final _TranslationsGithostsFormEs form = _TranslationsGithostsFormEs._(_root);
	@override String deleteBody({required Object host}) => 'Elimina la credencial. Las sessions que intenten listar PRs de ${host} recurrirán a la API sin autenticar.';
	@override String deletedSnack({required Object name}) => '${name} eliminado.';
	@override String enabledSnack({required Object name}) => '${name} habilitado.';
	@override String disabledSnack({required Object name}) => '${name} deshabilitado.';
	@override String get emptyList => 'No hay hosts de Git configurados.\n\nAñade una credencial para que el gateway pueda listar pull requests en todos tus repos.';
	@override String get failedToLoad => 'Error al cargar los hosts de Git';
}

// Path: channels
class _TranslationsChannelsEs extends TranslationsChannelsEn {
	_TranslationsChannelsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Canales';
	@override String get kNew => 'Nuevo';
	@override String get sendTest => 'Enviar mensaje de prueba';
	@override String get editConfig => 'Editar configuración';
	@override String get editNotifications => 'Editar notificaciones';
	@override String get viewRawConfig => 'Ver configuración sin procesar';
	@override String get copyChannelId => 'Copiar id del canal';
	@override String copiedSnack({required Object id}) => 'Copiado ${id}';
	@override String createdSnack({required Object kind}) => 'Canal ${kind} creado.';
	@override String createFailedApi({required Object error}) => 'Error al crear: ${error}';
	@override String createFailedGeneric({required Object error}) => 'Error al crear: ${error}';
	@override String get deleteTitle => '¿Eliminar canal?';
	@override late final _TranslationsChannelsConfigDialogEs configDialog = _TranslationsChannelsConfigDialogEs._(_root);
	@override late final _TranslationsChannelsWebhookDialogEs webhookDialog = _TranslationsChannelsWebhookDialogEs._(_root);
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}: ${error}';
	@override late final _TranslationsChannelsNotificationsEs notifications = _TranslationsChannelsNotificationsEs._(_root);
	@override late final _TranslationsChannelsPopupEs popup = _TranslationsChannelsPopupEs._(_root);
	@override late final _TranslationsChannelsBadgesEs badges = _TranslationsChannelsBadgesEs._(_root);
	@override String capsLabel({required Object list}) => '· caps: ${list}';
	@override String get bridgeWebOnly => 'Los canales bridge solo están disponibles en la web';
	@override String get bridgeEmptyAdd => 'Añade uno desde el admin web: Canales → Nuevo.';
	@override String get deleteBody => 'Detiene el canal y elimina su configuración. Las notificaciones en curso dirigidas a él se descartarán de forma silenciosa.';
	@override late final _TranslationsChannelsSnacksEs snacks = _TranslationsChannelsSnacksEs._(_root);
	@override late final _TranslationsChannelsErrorPrefixEs errorPrefix = _TranslationsChannelsErrorPrefixEs._(_root);
	@override String get failedToLoad => 'Error al cargar los canales';
	@override late final _TranslationsChannelsKindsEs kinds = _TranslationsChannelsKindsEs._(_root);
}

// Path: onboarding
class _TranslationsOnboardingEs extends TranslationsOnboardingEn {
	_TranslationsOnboardingEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get gatewayLabel => 'URL del gateway';
	@override String get gatewayHint => 'https://opendray.example.com';
	@override String get kContinue => 'Continuar';
}

// Path: skills
class _TranslationsSkillsEs extends TranslationsSkillsEn {
	_TranslationsSkillsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Skills';
	@override String get newSkill => 'Nuevo skill';
	@override String get install => 'Instalar SKILL.md';
	@override String installedSnack({required Object name}) => 'Instalado ${name}';
	@override String installFailed({required Object error}) => 'Instalación fallida: ${error}';
	@override String customizingBuiltin({required Object id}) => 'Personalizando ${id} integrado';
	@override String get idLabel => 'Id (slug)';
	@override String get idHint => 'p. ej. tdd-guide';
	@override String get bodyLabel => 'Cuerpo (markdown)';
	@override String loadFailedApi({required Object error}) => 'Error al cargar: ${error}';
	@override String loadFailedGeneric({required Object error}) => 'Error al cargar: ${error}';
	@override String get idRequired => 'El Id es obligatorio.';
	@override String get bodyRequired => 'El cuerpo no puede estar vacío.';
	@override String get snackCreated => 'Skill creado.';
	@override String get snackOverride => 'Guardado como override del vault.';
	@override String get snackUpdated => 'Skill actualizado.';
	@override String saveFailedApi({required Object error}) => 'Error al guardar: ${error}';
	@override String saveFailedGeneric({required Object error}) => 'Error al guardar: ${error}';
	@override String get resetTitle => '¿Restablecer al integrado?';
	@override String get deleteTitle => '¿Eliminar skill?';
	@override String resetBody({required Object id}) => 'Elimina el override del vault para ${id}. Las sesiones recurrirán al cuerpo integrado.';
	@override String get resetButton => 'Restablecer';
	@override String resetSnack({required Object id}) => '${id} restablecido al integrado.';
	@override String deletedSnack({required Object id}) => '${id} eliminado.';
	@override String deleteFailedApi({required Object error}) => 'Error al eliminar: ${error}';
	@override String deleteFailedGeneric({required Object error}) => 'Error al eliminar: ${error}';
	@override String deleteBody({required Object id}) => 'Elimina ${id} del vault. Las sesiones que lo referencian fallarán hasta que se restaure.';
	@override String get newSkillTitle => 'Nuevo skill';
	@override String customizeTitle({required Object id}) => 'Personalizar ${id}';
	@override String editTitle({required Object id}) => 'Editar ${id}';
	@override String get resetTooltip => 'Restablecer al integrado';
	@override String get deleteTooltip => 'Eliminar';
	@override String get saving => 'Guardando…';
	@override String get saveOverride => 'Guardar override';
	@override String get overrideBanner => 'Al guardar se crea un override del vault con el mismo id. Las sesiones usarán este cuerpo en lugar del integrado hasta que lo restablezcas.';
	@override String get idHelper => 'Letras minúsculas / dígitos / guion. Se bloquea una vez creado.';
	@override String get emptyList => 'No hay skills configurados. El gateway incluye integrados (planner, code-reviewer, etc.).';
	@override String get failedToLoad => 'Error al cargar los skills';
}

// Path: customTasks
class _TranslationsCustomTasksEs extends TranslationsCustomTasksEn {
	_TranslationsCustomTasksEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Tareas personalizadas';
	@override String get newTask => 'Nueva tarea';
	@override String get deleteTitle => '¿Eliminar tarea?';
	@override String deletedSnack({required Object name}) => '${name} eliminada.';
	@override String deleteFailedApi({required Object error}) => 'Error al eliminar: ${error}';
	@override String deleteFailedGeneric({required Object error}) => 'Error al eliminar: ${error}';
	@override String get popupEdit => 'Editar';
	@override String get popupDelete => 'Eliminar';
	@override String get nameHint => 'p. ej. backend-tests';
	@override String get commandHint => '/run pnpm test --filter backend';
	@override String get descriptionHint => 'Frase breve que aparece bajo el nombre de la tarea.';
	@override String get scopeGlobal => 'Global';
	@override String get scopeProject => 'Proyecto';
	@override String get cwdHint => '/Users/you/projects/backend';
	@override String get snackCreated => 'Tarea creada.';
	@override String get snackUpdated => 'Tarea actualizada.';
	@override String get deleteBody => 'Quita la tarea del catálogo. Las sessions que ya la insertaron no se ven afectadas.';
	@override String get introBanner => 'Define tus propios slash commands. Aparecen en el selector de tareas de la session junto a los integrados.';
	@override String get validateNameRequired => 'El nombre es obligatorio';
	@override String get validateCommandRequired => 'El comando es obligatorio';
	@override String get validateProjectCwd => 'Las tareas con ámbito de proyecto necesitan una ruta cwd absoluta';
	@override String get appBarEdit => 'Editar tarea personalizada';
	@override String get appBarNew => 'Nueva tarea personalizada';
	@override String get fieldName => 'Nombre';
	@override String get nameHelper => 'Aparece en el selector de tareas del inspector.';
	@override String get fieldCommand => 'Comando';
	@override String get commandHelper => 'El texto que se inserta en la session al elegirlo. Puede ser un comando de CLI o un slash command de Claude.';
	@override String get fieldDescription => 'Descripción (opcional)';
	@override String get fieldScope => 'Ámbito';
	@override String get globalScopeHint => 'Visible desde cualquier session, sin importar el cwd.';
	@override String get projectScopeHint => 'Visible solo cuando el cwd de una session coincide con la ruta de abajo.';
	@override String get fieldProjectCwd => 'cwd del proyecto';
	@override String get cwdHelper => 'Ruta absoluta. Las sessions creadas con este cwd exacto verán la tarea.';
	@override String get saving => 'Guardando…';
	@override String get save => 'Guardar';
	@override String get create => 'Crear';
	@override String get failedToLoad => 'Error al cargar las tareas personalizadas';
}

// Path: notesPage
class _TranslationsNotesPageEs extends TranslationsNotesPageEn {
	_TranslationsNotesPageEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Notas';
	@override String get newButton => 'Nueva';
	@override String get newNoteDialogTitle => 'Nueva nota';
	@override String get searchHint => 'Busca en todo el vault…';
	@override String get up => 'Arriba';
	@override String get copyPath => 'Copiar ruta';
	@override String get open => 'Abrir';
	@override String copiedSnack({required Object path}) => 'Copiado ${path}';
	@override String get deleteTitle => '¿Eliminar nota?';
	@override String deletedSnack({required Object path}) => 'Eliminado ${path}';
	@override String deleteFailedApi({required Object error}) => 'Error al eliminar: ${error}';
	@override String deleteFailedGeneric({required Object error}) => 'Error al eliminar: ${error}';
	@override String createFailedApi({required Object error}) => 'Error al crear: ${error}';
	@override String createFailedGeneric({required Object error}) => 'Error al crear: ${error}';
	@override String get pathLabel => 'Ruta relativa al vault';
	@override String get pathHint => 'personal/scratch.md';
	@override String get create => 'Crear';
	@override String get popupDelete => 'Eliminar';
	@override String get deleteBody => 'Esto es irreversible. La sincronización git del vault también eliminará el archivo en el host del gateway.';
	@override String emptyFilterMatch({required Object query}) => 'Ninguna nota coincide con "${query}".';
	@override String get emptyVault => 'El vault está vacío. Toca + para crear tu primera nota.';
	@override String emptyFolder({required Object path}) => 'La carpeta "${path}" está vacía.';
	@override String get validatePath => 'La ruta es obligatoria';
	@override String get validatePathDots => 'La ruta no puede contener ".."';
	@override String get pathHelper => 'Añade .md automáticamente si falta.';
	@override late final _TranslationsNotesPageEditorEs editor = _TranslationsNotesPageEditorEs._(_root);
}

// Path: dataExport
class _TranslationsDataExportEs extends TranslationsDataExportEn {
	_TranslationsDataExportEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Exportación e importación de datos';
	@override String get subtitle => 'Paquetes a nivel de usuario para migración o verificación, independientes de /backups (recuperación ante desastres).';
	@override late final _TranslationsDataExportSectionsEs sections = _TranslationsDataExportSectionsEs._(_root);
	@override late final _TranslationsDataExportFormEs form = _TranslationsDataExportFormEs._(_root);
	@override late final _TranslationsDataExportHistoryEs history = _TranslationsDataExportHistoryEs._(_root);
	@override late final _TranslationsDataExportImportEs import = _TranslationsDataExportImportEs._(_root);
	@override late final _TranslationsDataExportImportsEs imports = _TranslationsDataExportImportsEs._(_root);
	@override late final _TranslationsDataExportRelativeEs relative = _TranslationsDataExportRelativeEs._(_root);
	@override late final _TranslationsDataExportStatusEs status = _TranslationsDataExportStatusEs._(_root);
}

// Path: memory
class _TranslationsMemoryEs extends TranslationsMemoryEn {
	_TranslationsMemoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsMemoryStatusEs status = _TranslationsMemoryStatusEs._(_root);
	@override String get title => 'Memoria';
	@override String get more => 'Más';
	@override String get workers => 'Workers de memoria';
	@override late final _TranslationsMemoryRankEs rank = _TranslationsMemoryRankEs._(_root);
	@override String get kNew => 'Nuevo';
	@override String get searchHint => 'Buscar…';
	@override String get projectLabel => 'Proyecto';
	@override String get filterHint => 'Filtrar por nombre o ruta…';
	@override String get copied => 'Copiado';
	@override String get copyTooltip => 'Copiar texto';
	@override late final _TranslationsMemoryDeleteAllConfirmEs deleteAllConfirm = _TranslationsMemoryDeleteAllConfirmEs._(_root);
	@override String deletedSnackOne({required Object n}) => 'Se eliminó ${n} elemento de memoria';
	@override String deletedSnackOther({required Object n}) => 'Se eliminaron ${n} elementos de memoria';
	@override String bulkDeleteFailedApi({required Object error}) => 'Error al eliminar en bloque: ${error}';
	@override String bulkDeleteFailedGeneric({required Object error}) => 'Error al eliminar en bloque: ${error}';
	@override late final _TranslationsMemoryDeleteOneEs deleteOne = _TranslationsMemoryDeleteOneEs._(_root);
	@override late final _TranslationsMemoryScopeEs scope = _TranslationsMemoryScopeEs._(_root);
	@override late final _TranslationsMemoryCreateEs create = _TranslationsMemoryCreateEs._(_root);
	@override String get archive => 'Archivar';
	@override String get quarantine => 'Cuarentena';
	@override String get archivedToast => 'Memoria archivada — restaurable desde Archivado';
	@override String get quarantinedToast => 'Memoria en cuarentena — revísala en Cortex → Cuarentena';
	@override String archiveFailed({required Object error}) => 'Error al archivar: ${error}';
	@override String quarantineFailed({required Object error}) => 'Error al poner en cuarentena: ${error}';
	@override late final _TranslationsMemoryReembedEs reembed = _TranslationsMemoryReembedEs._(_root);
}

// Path: about
class _TranslationsAboutEs extends TranslationsAboutEn {
	_TranslationsAboutEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Acerca de';
	@override String get loading => 'Cargando…';
	@override late final _TranslationsAboutSectionsEs sections = _TranslationsAboutSectionsEs._(_root);
	@override late final _TranslationsAboutFieldsEs fields = _TranslationsAboutFieldsEs._(_root);
	@override String copied({required Object label}) => '${label} copiado';
	@override String get copyTooltip => 'Copiar';
	@override late final _TranslationsAboutCopyLabelsEs copyLabels = _TranslationsAboutCopyLabelsEs._(_root);
	@override String get tagline => 'opendray móvil, control del gateway multi-CLI.\nFuente: github.com/Opendray/opendray';
	@override late final _TranslationsAboutGatewayEs gateway = _TranslationsAboutGatewayEs._(_root);
}

// Path: settings
class _TranslationsSettingsEs extends TranslationsSettingsEn {
	_TranslationsSettingsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Ajustes';
	@override late final _TranslationsSettingsLanguageEs language = _TranslationsSettingsLanguageEs._(_root);
	@override late final _TranslationsSettingsAppearanceEs appearance = _TranslationsSettingsAppearanceEs._(_root);
	@override late final _TranslationsSettingsAccountEs account = _TranslationsSettingsAccountEs._(_root);
	@override late final _TranslationsSettingsGatewayEs gateway = _TranslationsSettingsGatewayEs._(_root);
	@override late final _TranslationsSettingsChangeCredentialsEs changeCredentials = _TranslationsSettingsChangeCredentialsEs._(_root);
	@override late final _TranslationsSettingsLogViewerEs logViewer = _TranslationsSettingsLogViewerEs._(_root);
	@override late final _TranslationsSettingsServerSettingsEs serverSettings = _TranslationsSettingsServerSettingsEs._(_root);
}

// Path: memoryQuarantine
class _TranslationsMemoryQuarantineEs extends TranslationsMemoryQuarantineEn {
	_TranslationsMemoryQuarantineEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cuarentena';
	@override String get subtitle => 'Hechos que necesitan revisión antes de contar como memoria durable: las capturas de integraciones llegan aquí por política, y puedes poner cualquier memoria en cuarentena a mano. Promueve lo verdadero; descarta el resto — las filas sin revisar expiran solas.';
	@override String get empty => 'Nada en cuarentena.';
	@override String loadFailed({required Object error}) => 'Error al cargar: ${error}';
	@override String get promote => 'Promover';
	@override String get discard => 'Descartar';
	@override String get promotedToast => 'Promovida a memoria durable';
	@override String get discardedToast => 'Descartada';
	@override String actionFailed({required Object error}) => 'La acción falló: ${error}';
	@override String expires({required Object date}) => 'expira ${date}';
	@override String countBadge({required Object count}) => '${count} pendientes';
}

// Path: cortexHub
class _TranslationsCortexHubEs extends TranslationsCortexHubEn {
	_TranslationsCortexHubEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cortex';
	@override String get subtitle => 'El volante de experiencia: Memoria → Notas → Conocimiento, realimentado en cada session.';
	@override String idleBadge({required Object days}) => 'inactivo ${days}d';
	@override String activeProjectsBadge({required Object count}) => '${count} activos';
	@override String get activeProjectsTitle => 'Proyectos activos';
	@override String get loopHint => 'Las sesiones alimentan la Memoria → la Memoria se destila en Notas → las Notas se compilan en Conocimiento → el Conocimiento guía cada nueva sesión.';
	@override String get settings => 'Ajustes';
	@override String get memory => 'Memoria';
	@override String get memoryDesc => 'Hechos crudos entre sessions que los agentes guardan y recuerdan.';
	@override String get notes => 'Notas';
	@override String get notesDesc => 'El objetivo / plan / diario oficial de cada proyecto.';
	@override String get knowledge => 'Conocimiento';
	@override String get knowledgeDesc => 'Experiencia destilada entre proyectos.';
	@override String quarantineBadge({required Object count}) => '${count} por revisar';
	@override String pendingBadge({required Object count}) => '${count} pendientes';
	@override String get disabled => 'desactivado';
	@override String inboxTitle({required Object count}) => 'Propuestas pendientes (${count})';
	@override String get inboxHint => 'Actualizaciones propuestas por la IA para notas y páginas KB. Aprueba para publicar, rechaza para descartar.';
	@override String get kbLabel => 'Base de conocimiento';
	@override String get preview => 'Vista previa';
	@override String get hide => 'Ocultar';
	@override String get approve => 'Aprobar';
	@override String get reject => 'Rechazar';
	@override String get approvedToast => 'Propuesta aprobada';
	@override String get rejectedToast => 'Propuesta rechazada';
	@override String actionFailed({required Object error}) => 'La acción falló: ${error}';
	@override String loadFailed({required Object error}) => 'Error al cargar: ${error}';
}

// Path: cortexSettings
class _TranslationsCortexSettingsEs extends TranslationsCortexSettingsEn {
	_TranslationsCortexSettingsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Ajustes de Cortex';
	@override String get tabWorkers => 'Workers';
	@override String get tabCapture => 'Captura e inyección';
	@override String get tabProviders => 'Proveedores';
	@override String get providersHint => 'Endpoints LLM a los que enrutan los workers de resumen/agente.';
	@override String get providersEmpty => 'Sin proveedores configurados.';
	@override String get providersManageOnWeb => 'Añade o edita proveedores en el panel web.';
	@override String get providersLoadFailed => 'Error al cargar proveedores';
	@override String get defaultBadge => 'predeterminado';
}

// Path: web.topbar
class _TranslationsWebTopbarEs extends TranslationsWebTopbarEn {
	_TranslationsWebTopbarEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get expandSidebar => 'Expandir barra lateral';
	@override String get collapseSidebar => 'Contraer barra lateral';
	@override String get search => 'Buscar';
	@override String get openPalette => 'Abrir paleta de comandos';
	@override String get theme => 'Tema';
	@override String themeLabel({required Object mode}) => 'Tema: ${mode}';
	@override String get appearance => 'Apariencia';
	@override String get themeLight => 'Claro';
	@override String get themeDark => 'Oscuro';
	@override String get themeSystem => 'Sistema';
	@override String get language => 'Idioma';
	@override String get languageEnglish => 'English';
	@override String get languageChinese => '中文';
	@override String get languageSpanish => 'Español';
	@override String get signedInAs => 'Sesión iniciada como';
	@override String get tokenExpires => 'El token caduca';
	@override String get signOut => 'Cerrar sesión';
}

// Path: web.sessions
class _TranslationsWebSessionsEs extends TranslationsWebSessionsEn {
	_TranslationsWebSessionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebSessionsListEs list = _TranslationsWebSessionsListEs._(_root);
	@override late final _TranslationsWebSessionsTabsEs tabs = _TranslationsWebSessionsTabsEs._(_root);
	@override late final _TranslationsWebSessionsPageEs page = _TranslationsWebSessionsPageEs._(_root);
	@override late final _TranslationsWebSessionsEmptyEs empty = _TranslationsWebSessionsEmptyEs._(_root);
	@override late final _TranslationsWebSessionsHeaderEs header = _TranslationsWebSessionsHeaderEs._(_root);
	@override late final _TranslationsWebSessionsTerminalEs terminal = _TranslationsWebSessionsTerminalEs._(_root);
	@override late final _TranslationsWebSessionsSpawnEs spawn = _TranslationsWebSessionsSpawnEs._(_root);
	@override late final _TranslationsWebSessionsAccountSwitcherEs accountSwitcher = _TranslationsWebSessionsAccountSwitcherEs._(_root);
	@override late final _TranslationsWebSessionsInspectorEs inspector = _TranslationsWebSessionsInspectorEs._(_root);
	@override late final _TranslationsWebSessionsEndedEs ended = _TranslationsWebSessionsEndedEs._(_root);
	@override late final _TranslationsWebSessionsFileBrowserEs fileBrowser = _TranslationsWebSessionsFileBrowserEs._(_root);
}

// Path: web.memory
class _TranslationsWebMemoryEs extends TranslationsWebMemoryEn {
	_TranslationsWebMemoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Memoria';
	@override String get subtitle => 'Explora, busca y edita las memorias que los agentes han almacenado a través del servidor MCP opendray-memory.';
	@override String get navProject => 'Proyecto';
	@override String get navArchived => 'Archivadas';
	@override String get navWorkers => 'Ajustes de Cortex';
	@override String get navConfiguration => 'Almacenamiento y embedder →';
	@override String get navQuarantine => 'Cuarentena';
}

// Path: web.journalStale
class _TranslationsWebJournalStaleEs extends TranslationsWebJournalStaleEn {
	_TranslationsWebJournalStaleEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Purgar entradas obsoletas';
	@override String subtitle({required Object days}) => '(con más de ${days} días, sin conflictos pendientes)';
	@override String get daysLabel => 'Con más de (días):';
	@override String get loading => 'Escaneando…';
	@override String get empty => 'No hay nada obsoleto que purgar.';
	@override String get selectAll => 'Seleccionar todo';
	@override String get deselectAll => 'Deseleccionar todo';
	@override String deleteSelected({required Object count}) => 'Eliminar (${count})';
	@override String deleted_one({required Object count}) => '${count} entrada eliminada';
	@override String deleted_other({required Object count}) => '${count} entradas eliminadas';
}

// Path: web.conflicts
class _TranslationsWebConflictsEs extends TranslationsWebConflictsEn {
	_TranslationsWebConflictsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Conflictos entre capas';
	@override String get subtitle => 'Contradicciones que el detector diario encontró entre hechos, plan, objetivo y entradas del diario.';
	@override String get loading => 'Cargando conflictos…';
	@override String get empty => 'No hay conflictos pendientes. Haz clic en "Detectar ahora" para ejecutar un análisis bajo demanda.';
	@override String get pickCwd => 'Elige un proyecto para ver sus conflictos.';
	@override String get detectNow => 'Detectar ahora';
	@override String detected({required Object count}) => 'Se encontraron ${count} conflicto(s) nuevo(s)';
	@override String get accept => 'Aceptar';
	@override String get dismiss => 'Descartar';
	@override String get accepted => 'Conflicto aceptado. Recuerda aplicar la corrección';
	@override String get dismissed => 'Conflicto descartado';
	@override String get deletedFact => 'Hecho eliminado y conflicto aceptado';
	@override String get quickActions => 'Corrección:';
	@override String get deleteFact => 'Eliminar hecho';
	@override String deleteFactSide({required Object side, required Object ref}) => 'Eliminar ${side}: ${ref}';
	@override late final _TranslationsWebConflictsConfirmDeleteEs confirmDelete = _TranslationsWebConflictsConfirmDeleteEs._(_root);
	@override late final _TranslationsWebConflictsOpenLayerEs openLayer = _TranslationsWebConflictsOpenLayerEs._(_root);
	@override late final _TranslationsWebConflictsSeverityEs severity = _TranslationsWebConflictsSeverityEs._(_root);
}

// Path: web.memoryHealth
class _TranslationsWebMemoryHealthEs extends TranslationsWebMemoryHealthEn {
	_TranslationsWebMemoryHealthEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object days}) => 'Estado de la memoria, últimos ${days} días';
	@override String get subtitle => 'Señales agregadas de ambos subsistemas de memoria de este proyecto.';
	@override String get loading => 'Cargando instantánea de estado…';
	@override String get errorLoading => 'No se pudo cargar la instantánea de estado.';
	@override String get pickCwd => 'Elige un proyecto para ver el estado de su memoria.';
	@override String get newFacts => 'Datos nuevos';
	@override String newFactsHint({required Object total}) => '${total} almacenados en total';
	@override String get captureFires => 'Capturas activadas';
	@override String captureFiresHint({required Object stored, required Object deduped}) => '${stored} almacenadas · ${deduped} deduplicadas';
	@override String get newJournal => 'Entradas de diario';
	@override String newJournalHint({required Object total}) => '${total} en total';
	@override String get planAge => 'Plan actualizado por última vez';
	@override String planAgeHint({required Object count}) => '${count} propuesta(s) de desvío de plan pendiente(s)';
	@override String get planAgeHintNone => 'No hay propuestas de desvío de plan pendientes';
	@override String get goalAge => 'Objetivo actualizado por última vez';
	@override String get pending => 'Propuestas pendientes';
	@override String pendingHint({required Object days}) => 'la más antigua, ${days}d de antigüedad';
	@override String topHit({required Object hits}) => 'Más consultado · ${hits} recuperaciones';
	@override String zeroHit({required Object count}) => '${count} datos con más de 7d sin ninguna recuperación, candidatos para limpieza.';
	@override String get never => 'nunca';
	@override String get today => 'hoy';
	@override String daysAgo_one({required Object count}) => 'hace ${count} día';
	@override String daysAgo_other({required Object count}) => 'hace ${count} días';
}

// Path: web.memoryConfig
class _TranslationsWebMemoryConfigEs extends TranslationsWebMemoryConfigEn {
	_TranslationsWebMemoryConfigEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Ajustes de Cortex';
	@override String get subtitle => 'Todos los mandos de runtime del ciclo de IA en un solo lugar — inyección de spawn, providers LLM, workers por tarea, triggers de captura, perfiles de inyección, costes de tokens. Los cambios aplican al instante; sin reinicio.';
	@override late final _TranslationsWebMemoryConfigSectionsEs sections = _TranslationsWebMemoryConfigSectionsEs._(_root);
	@override late final _TranslationsWebMemoryConfigSectionHintsEs sectionHints = _TranslationsWebMemoryConfigSectionHintsEs._(_root);
	@override late final _TranslationsWebMemoryConfigMoveBannerEs moveBanner = _TranslationsWebMemoryConfigMoveBannerEs._(_root);
	@override late final _TranslationsWebMemoryConfigInfraEs infra = _TranslationsWebMemoryConfigInfraEs._(_root);
}

// Path: web.memoryWorkers
class _TranslationsWebMemoryWorkersEs extends TranslationsWebMemoryWorkersEn {
	_TranslationsWebMemoryWorkersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Workers de memoria';
	@override String get loading => 'Cargando configuración de workers…';
	@override String get errorTitle => 'No se puede acceder al endpoint.';
	@override String get errorDescription => 'Las rutas /api/v1/memory/workers son nuevas en M25. Puede que el binario de opendray necesite un reinicio para montarlas y ejecutar la migración 0029.';
	@override String get intro => 'Cada punto de contacto del LLM con el sistema de memoria puede atenderse de forma independiente mediante el endpoint local <1>summarizer</1> (LM Studio / compatible con OpenAI) o lanzando un <3>agente Claude / Gemini</3> sin interfaz en modo <5>--print</5>. Las tareas narrativas de alta calidad (gitactivity, transcript) se benefician de los workers de agente; las tareas de alta frecuencia (gatekeeper) permanecen en el endpoint local por diseño.';
	@override String get enabledBadge => 'habilitado';
	@override String get disabledBadge => 'deshabilitado';
	@override String get summarizerOnlyBadge => 'solo-summarizer';
	@override String callsCount({required Object count}) => '${count} llamadas · 24h';
	@override String avgMs({required Object ms}) => 'media ${ms}ms';
	@override String errorsCount({required Object count}) => '${count} errores';
	@override String get workerLabel => 'Worker';
	@override String get summarizerHttp => 'Summarizer (HTTP)';
	@override String get agentCliPrint => 'Agente (CLI --print)';
	@override String get summarizerProviderLabel => 'Proveedor del summarizer';
	@override String get registryDefault => 'Predeterminado del registro';
	@override String get cliLabel => 'CLI';
	@override String get selectPlaceholder => 'Seleccionar';
	@override String get cliClaude => 'Claude';
	@override String get cliGemini => 'Gemini';
	@override String get claudeAccountLabel => 'Cuenta de Claude';
	@override String get claudeAccountDefault => 'Predeterminada';
	@override String get agentWarning => 'El modo agente lanza un CLI sin interfaz por cada llamada. La latencia sube de <1>~1s</1> (summarizer) a <3>~5-15s</3>; el coste pasa de la CPU a tu quota de Claude/Gemini.';
	@override String get enabledCheckbox => 'Habilitado';
	@override String get testButton => 'Probar';
	@override String get saveButton => 'Guardar';
	@override String recentCalls({required Object count}) => 'Llamadas recientes (${count})';
	@override String get tableWhen => 'cuándo';
	@override String get tableWorker => 'worker';
	@override String get tableMs => 'ms';
	@override String get tableOk => 'ok';
	@override String savedToast({required Object label}) => '${label} actualizado';
	@override String get saveFailedToast => 'Error al guardar';
	@override String testOkToast({required Object label, required Object ms}) => '${label} OK. ${ms}ms';
	@override String testFailedToast({required Object label}) => '${label} falló';
	@override String get testCallFailedToast => 'La llamada de prueba falló';
	@override String get unknownError => 'error desconocido';
	@override late final _TranslationsWebMemoryWorkersTasksEs tasks = _TranslationsWebMemoryWorkersTasksEs._(_root);
	@override String get modelLabel => 'Modelo';
	@override String get modelHint => 'Fija el modelo del CLI para esta tarea (p. ej. haiku para tareas básicas). Vacío = predeterminado del CLI.';
	@override String get modelCliDefault => 'Predeterminado del CLI (último)';
	@override String get modelCustom => 'Personalizado…';
	@override String get modelCustomPlaceholder => 'id exacto del modelo';
	@override String get modelBackToList => 'Lista';
	@override String get cliCodex => 'Codex (codex exec)';
	@override String get cliAntigravity => 'Antigravity (agy --print)';
	@override String infraGateOff({required Object label}) => 'El enrutado de ${label} está guardado, pero su puerta de función está APAGADA en Server Settings — no se ejecutará nada hasta que la actives allí.';
	@override String get infraGateOpen => 'Activarla';
	@override String get providerModel => 'modelo:';
}

// Path: web.archived
class _TranslationsWebArchivedEs extends TranslationsWebArchivedEn {
	_TranslationsWebArchivedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando…';
	@override String get emptyTitle => 'Nada archivado';
	@override String get emptyDescription => 'No hay memorias archivadas en ningún proyecto. El limpiador automático archiva aquí los hechos obsoletos y duplicados (restaurables durante 30 días); todavía no se ha eliminado nada.';
	@override String get title => 'Memorias archivadas';
	@override String get subtitle => 'Memorias archivadas (reversibles) hasta que la ventana de gracia de 30 días las purgue. Fuentes: los veredictos del auto-cleaner, el archivado manual por memoria, y los proyectos que archivas — las memorias de un proyecto llegan juntas y vuelven juntas al desarchivarlo (los archivados de proyecto están exentos de la purga).';
	@override String get globalScope => '(global)';
	@override String summary({required Object projects, required Object memories}) => '${projects} proyectos · ${memories} memorias archivadas';
	@override String memCount({required Object count}) => '${count} memorias';
	@override String get restoreAll => 'Restaurar todo';
	@override String get restoreAllTooltip => 'Restaurar todas las memorias archivadas de este proyecto';
	@override String restoreAllConfirm({required Object count, required Object project}) => '¿Restaurar las ${count} memorias archivadas de ${project}?';
	@override String restoredAllToast({required Object count}) => '${count} memorias restauradas';
	@override String get deleteButton => 'Eliminar';
	@override String get deleteTooltip => 'Eliminar permanentemente ahora — omite la ventana de gracia de 30 días, no se puede deshacer';
	@override String get deleteConfirm => '¿Eliminar permanentemente esta memoria ahora? Omite la ventana de gracia de 30 días y no se puede deshacer.';
	@override String get deletedToast => 'Eliminada permanentemente';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override String get deleteAll => 'Eliminar todo';
	@override String get deleteAllTooltip => 'Eliminar permanentemente ahora todas las memorias archivadas de este proyecto';
	@override String deleteAllConfirm({required Object count, required Object project}) => '¿Eliminar permanentemente ahora las ${count} memorias archivadas de ${project}? Omite la ventana de gracia de 30 días y no se puede deshacer.';
	@override String deletedAllToast({required Object count}) => '${count} memorias eliminadas';
	@override String get openProject => 'Abrir proyecto';
	@override String get archivedAtPrefix => 'Archivado';
	@override String get restoreButton => 'Restaurar';
	@override String get restoredToast => 'Restaurado';
	@override String get restoreFailedToast => 'Error al restaurar';
}

// Path: web.project
class _TranslationsWebProjectEs extends TranslationsWebProjectEn {
	_TranslationsWebProjectEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebProjectPickerEs picker = _TranslationsWebProjectPickerEs._(_root);
	@override String get noCwd => 'Elige un proyecto para gestionar su memoria.';
	@override late final _TranslationsWebProjectHeaderEs header = _TranslationsWebProjectHeaderEs._(_root);
	@override late final _TranslationsWebProjectTabsEs tabs = _TranslationsWebProjectTabsEs._(_root);
	@override late final _TranslationsWebProjectDocLabelEs docLabel = _TranslationsWebProjectDocLabelEs._(_root);
	@override late final _TranslationsWebProjectEditorEs editor = _TranslationsWebProjectEditorEs._(_root);
	@override late final _TranslationsWebProjectReadonlyEs readonly = _TranslationsWebProjectReadonlyEs._(_root);
	@override late final _TranslationsWebProjectJournalEs journal = _TranslationsWebProjectJournalEs._(_root);
	@override late final _TranslationsWebProjectInboxEs inbox = _TranslationsWebProjectInboxEs._(_root);
	@override late final _TranslationsWebProjectArchivedEs archived = _TranslationsWebProjectArchivedEs._(_root);
	@override late final _TranslationsWebProjectResetEs reset = _TranslationsWebProjectResetEs._(_root);
	@override late final _TranslationsWebProjectLifecycleEs lifecycle = _TranslationsWebProjectLifecycleEs._(_root);
	@override late final _TranslationsWebProjectDocMetaEs docMeta = _TranslationsWebProjectDocMetaEs._(_root);
	@override late final _TranslationsWebProjectProposalBannerEs proposalBanner = _TranslationsWebProjectProposalBannerEs._(_root);
	@override late final _TranslationsWebProjectOverviewEs overview = _TranslationsWebProjectOverviewEs._(_root);
}

// Path: web.memoryInspector
class _TranslationsWebMemoryInspectorEs extends TranslationsWebMemoryInspectorEn {
	_TranslationsWebMemoryInspectorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebMemoryInspectorStatusEs status = _TranslationsWebMemoryInspectorStatusEs._(_root);
	@override late final _TranslationsWebMemoryInspectorScopeEs scope = _TranslationsWebMemoryInspectorScopeEs._(_root);
	@override late final _TranslationsWebMemoryInspectorSearchEs search = _TranslationsWebMemoryInspectorSearchEs._(_root);
	@override late final _TranslationsWebMemoryInspectorRecordsEs records = _TranslationsWebMemoryInspectorRecordsEs._(_root);
	@override late final _TranslationsWebMemoryInspectorRowEs row = _TranslationsWebMemoryInspectorRowEs._(_root);
	@override late final _TranslationsWebMemoryInspectorToastsEs toasts = _TranslationsWebMemoryInspectorToastsEs._(_root);
	@override late final _TranslationsWebMemoryInspectorBulkDeleteEs bulkDelete = _TranslationsWebMemoryInspectorBulkDeleteEs._(_root);
	@override late final _TranslationsWebMemoryInspectorAddMemEs addMem = _TranslationsWebMemoryInspectorAddMemEs._(_root);
	@override late final _TranslationsWebMemoryInspectorPickerEs picker = _TranslationsWebMemoryInspectorPickerEs._(_root);
	@override late final _TranslationsWebMemoryInspectorMigrationBannerEs migrationBanner = _TranslationsWebMemoryInspectorMigrationBannerEs._(_root);
	@override late final _TranslationsWebMemoryInspectorReembedEs reembed = _TranslationsWebMemoryInspectorReembedEs._(_root);
}

// Path: web.notes
class _TranslationsWebNotesEs extends TranslationsWebNotesEn {
	_TranslationsWebNotesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Notas';
	@override late final _TranslationsWebNotesHeaderEs header = _TranslationsWebNotesHeaderEs._(_root);
	@override late final _TranslationsWebNotesLeftEs left = _TranslationsWebNotesLeftEs._(_root);
	@override late final _TranslationsWebNotesTagsEs tags = _TranslationsWebNotesTagsEs._(_root);
	@override late final _TranslationsWebNotesTreeEs tree = _TranslationsWebNotesTreeEs._(_root);
	@override late final _TranslationsWebNotesOutlineEs outline = _TranslationsWebNotesOutlineEs._(_root);
	@override late final _TranslationsWebNotesNewNoteEs newNote = _TranslationsWebNotesNewNoteEs._(_root);
	@override late final _TranslationsWebNotesEmptyEs empty = _TranslationsWebNotesEmptyEs._(_root);
	@override late final _TranslationsWebNotesPickerEs picker = _TranslationsWebNotesPickerEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncEs vaultSync = _TranslationsWebNotesVaultSyncEs._(_root);
	@override late final _TranslationsWebNotesSyncBadgeEs syncBadge = _TranslationsWebNotesSyncBadgeEs._(_root);
}

// Path: web.activity
class _TranslationsWebActivityEs extends TranslationsWebActivityEn {
	_TranslationsWebActivityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Actividad';
	@override String get subtitle => 'Auditoría por llamada de las solicitudes API realizadas por las integraciones registradas. Incluye tanto las llamadas entrantes (una app de terceros que llama a opendray con su clave de API) como las llamadas salientes a través del proxy (admin → proxy de opendray → integración). Las llamadas hechas directamente por esta UI de administración no se registran.';
	@override String get refresh => 'Actualizar';
	@override String get refreshTooltip => 'Actualizar';
	@override late final _TranslationsWebActivityFiltersEs filters = _TranslationsWebActivityFiltersEs._(_root);
	@override String callsCount_one({required Object count}) => '${count} llamada';
	@override String callsCount_other({required Object count}) => '${count} llamadas';
	@override String get loading => 'Cargando…';
	@override late final _TranslationsWebActivityTableEs table = _TranslationsWebActivityTableEs._(_root);
	@override late final _TranslationsWebActivityEmptyEs empty = _TranslationsWebActivityEmptyEs._(_root);
	@override late final _TranslationsWebActivityEventsEs events = _TranslationsWebActivityEventsEs._(_root);
}

// Path: web.providers
class _TranslationsWebProvidersEs extends TranslationsWebProvidersEn {
	_TranslationsWebProvidersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebProvidersListEs list = _TranslationsWebProvidersListEs._(_root);
	@override late final _TranslationsWebProvidersDetailEs detail = _TranslationsWebProvidersDetailEs._(_root);
	@override late final _TranslationsWebProvidersConfigFormEs configForm = _TranslationsWebProvidersConfigFormEs._(_root);
	@override late final _TranslationsWebProvidersClaudeAccountsEs claudeAccounts = _TranslationsWebProvidersClaudeAccountsEs._(_root);
	@override late final _TranslationsWebProvidersModelsEs models = _TranslationsWebProvidersModelsEs._(_root);
}

// Path: web.channels
class _TranslationsWebChannelsEs extends TranslationsWebChannelsEn {
	_TranslationsWebChannelsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Canales';
	@override String get subtitle => 'Integraciones de mensajería bidireccional. Cada canal habilitado y no silenciado recibe notificaciones de sesión.';
	@override String get newButton => 'Nuevo canal';
	@override String get loading => 'Cargando…';
	@override late final _TranslationsWebChannelsEmptyEs empty = _TranslationsWebChannelsEmptyEs._(_root);
	@override late final _TranslationsWebChannelsCardEs card = _TranslationsWebChannelsCardEs._(_root);
	@override late final _TranslationsWebChannelsToastsEs toasts = _TranslationsWebChannelsToastsEs._(_root);
	@override late final _TranslationsWebChannelsDialogEs dialog = _TranslationsWebChannelsDialogEs._(_root);
	@override late final _TranslationsWebChannelsNotificationsEs notifications = _TranslationsWebChannelsNotificationsEs._(_root);
	@override late final _TranslationsWebChannelsBridgeEs bridge = _TranslationsWebChannelsBridgeEs._(_root);
	@override late final _TranslationsWebChannelsSetupEs setup = _TranslationsWebChannelsSetupEs._(_root);
}

// Path: web.integrations
class _TranslationsWebIntegrationsEs extends TranslationsWebIntegrationsEn {
	_TranslationsWebIntegrationsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Integraciones';
	@override String get subtitle => 'Aplicaciones externas que consumen opendray. Reenvían mediante reverse-proxy a través de <1>/api/v1/proxy/&lt;prefix&gt;/…</1> y se suscriben a eventos a través del endpoint WS.';
	@override String get register => 'Registrar';
	@override String get loading => 'Cargando…';
	@override late final _TranslationsWebIntegrationsTabsEs tabs = _TranslationsWebIntegrationsTabsEs._(_root);
	@override late final _TranslationsWebIntegrationsEmptyEs empty = _TranslationsWebIntegrationsEmptyEs._(_root);
	@override String get groupSystem => 'Sistema (gestionado por opendray)';
	@override String get groupOperator => 'Registradas por el operador';
	@override late final _TranslationsWebIntegrationsCardEs card = _TranslationsWebIntegrationsCardEs._(_root);
	@override late final _TranslationsWebIntegrationsRegisterDialogEs register_dialog = _TranslationsWebIntegrationsRegisterDialogEs._(_root);
	@override late final _TranslationsWebIntegrationsRevealEs reveal = _TranslationsWebIntegrationsRevealEs._(_root);
	@override late final _TranslationsWebIntegrationsEditDialogEs edit_dialog = _TranslationsWebIntegrationsEditDialogEs._(_root);
	@override late final _TranslationsWebIntegrationsProxyEs proxy = _TranslationsWebIntegrationsProxyEs._(_root);
}

// Path: web.plugins
class _TranslationsWebPluginsEs extends TranslationsWebPluginsEn {
	_TranslationsWebPluginsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Plugins del Inspector';
	@override String get subtitle => 'Configura las fuentes de datos que se muestran en el panel Inspector de la derecha cuando hay una session abierta. Cada plugin es solo para administradores y se comparte entre todas las sessions. Haz clic en el encabezado de una sección para contraerla.';
	@override late final _TranslationsWebPluginsCommonEs common = _TranslationsWebPluginsCommonEs._(_root);
	@override late final _TranslationsWebPluginsMcpEs mcp = _TranslationsWebPluginsMcpEs._(_root);
	@override late final _TranslationsWebPluginsMcpSecretsEs mcpSecrets = _TranslationsWebPluginsMcpSecretsEs._(_root);
	@override late final _TranslationsWebPluginsSkillsEs skills = _TranslationsWebPluginsSkillsEs._(_root);
	@override late final _TranslationsWebPluginsCustomTasksEs customTasks = _TranslationsWebPluginsCustomTasksEs._(_root);
	@override late final _TranslationsWebPluginsGitHostsEs gitHosts = _TranslationsWebPluginsGitHostsEs._(_root);
}

// Path: web.backups
class _TranslationsWebBackupsEs extends TranslationsWebBackupsEn {
	_TranslationsWebBackupsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Copias de seguridad';
	@override String get subtitle => 'Volcados cifrados de PostgreSQL escritos en un destino conectable. Configura programaciones y retención, o lanza copias puntuales para tener una red de seguridad rápida.';
	@override String get exportData => 'Exportar datos';
	@override String get loading => 'Cargando…';
	@override String get loadStatusFailedToast => 'No se pudo cargar el estado de la copia de seguridad';
	@override late final _TranslationsWebBackupsTabsEs tabs = _TranslationsWebBackupsTabsEs._(_root);
	@override late final _TranslationsWebBackupsInventoryEs inventory = _TranslationsWebBackupsInventoryEs._(_root);
	@override late final _TranslationsWebBackupsRestartEs restart = _TranslationsWebBackupsRestartEs._(_root);
	@override late final _TranslationsWebBackupsSetupEs setup = _TranslationsWebBackupsSetupEs._(_root);
	@override late final _TranslationsWebBackupsGeneratedEs generated = _TranslationsWebBackupsGeneratedEs._(_root);
	@override late final _TranslationsWebBackupsStatusEs status = _TranslationsWebBackupsStatusEs._(_root);
	@override late final _TranslationsWebBackupsBackupsTabEs backupsTab = _TranslationsWebBackupsBackupsTabEs._(_root);
	@override late final _TranslationsWebBackupsRestoreEs restore = _TranslationsWebBackupsRestoreEs._(_root);
	@override late final _TranslationsWebBackupsKindEs kind = _TranslationsWebBackupsKindEs._(_root);
	@override late final _TranslationsWebBackupsVerifyEs verify = _TranslationsWebBackupsVerifyEs._(_root);
	@override late final _TranslationsWebBackupsHealthEs health = _TranslationsWebBackupsHealthEs._(_root);
	@override late final _TranslationsWebBackupsTriggerEs trigger = _TranslationsWebBackupsTriggerEs._(_root);
	@override late final _TranslationsWebBackupsRecoveryKitEs recoveryKit = _TranslationsWebBackupsRecoveryKitEs._(_root);
	@override late final _TranslationsWebBackupsSchedulesTabEs schedulesTab = _TranslationsWebBackupsSchedulesTabEs._(_root);
	@override late final _TranslationsWebBackupsNewScheduleEs newSchedule = _TranslationsWebBackupsNewScheduleEs._(_root);
	@override late final _TranslationsWebBackupsFanoutEs fanout = _TranslationsWebBackupsFanoutEs._(_root);
	@override late final _TranslationsWebBackupsDedupEs dedup = _TranslationsWebBackupsDedupEs._(_root);
	@override late final _TranslationsWebBackupsTargetsTabEs targetsTab = _TranslationsWebBackupsTargetsTabEs._(_root);
	@override late final _TranslationsWebBackupsTargetEditorEs targetEditor = _TranslationsWebBackupsTargetEditorEs._(_root);
}

// Path: web.serverSettings
class _TranslationsWebServerSettingsEs extends TranslationsWebServerSettingsEn {
	_TranslationsWebServerSettingsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebServerSettingsSectionsEs sections = _TranslationsWebServerSettingsSectionsEs._(_root);
	@override String get loading => 'Cargando ajustes del servidor…';
	@override String loadFailed({required Object message}) => 'Error al cargar: ${message}';
	@override String get noConfigFlag => 'opendray se inició sin la opción -config. Los ajustes se cargan únicamente desde variables de entorno y no pueden editarse aquí.';
	@override String get resetButton => 'Restablecer';
	@override String get resetButtonTitle => 'Descartar los cambios sin guardar de esta sección';
	@override String resetConfirm({required Object section}) => '¿Restablecer "${section}" a los últimos valores guardados?';
	@override String get badgeRestartRequired => 'requiere reinicio';
	@override String get badgeUnsaved => 'sin guardar';
	@override String get saveButton => 'Guardar cambios';
	@override String get saveToastTitle => 'Ajustes guardados';
	@override String get saveToastDesc => 'Haz clic en Reiniciar para aplicarlos.';
	@override String get saveErrorTitle => 'Error al guardar';
	@override String get dangerousConfirm => 'Has cambiado la dirección de escucha, el usuario de administración o la contraseña de administración. Tras reiniciar puede que necesites volver a autenticarte o usar la nueva dirección. ¿Continuar?';
	@override String get unsavedHint => 'Tienes cambios sin guardar';
	@override String get savedHint => 'Todos los cambios guardados';
	@override String get searchPlaceholder => 'Filtrar campos…';
	@override late final _TranslationsWebServerSettingsRestartEs restart = _TranslationsWebServerSettingsRestartEs._(_root);
	@override late final _TranslationsWebServerSettingsFormGroupsEs formGroups = _TranslationsWebServerSettingsFormGroupsEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsEs fields = _TranslationsWebServerSettingsFieldsEs._(_root);
	@override late final _TranslationsWebServerSettingsLiveTailEs liveTail = _TranslationsWebServerSettingsLiveTailEs._(_root);
	@override late final _TranslationsWebServerSettingsMemoryInspectorCardEs memoryInspectorCard = _TranslationsWebServerSettingsMemoryInspectorCardEs._(_root);
	@override String get localOnnxBanner => 'Requiere que el binario se compile con <1>-tags local_onnx</1>. La compilación estándar devuelve un error de stub claro cuando se selecciona este backend. Consulta el tutorial <3>Memory → ONNX local</3> para los pasos de configuración.';
	@override late final _TranslationsWebServerSettingsStringListEs stringList = _TranslationsWebServerSettingsStringListEs._(_root);
	@override late final _TranslationsWebServerSettingsHttpHelpersEs httpHelpers = _TranslationsWebServerSettingsHttpHelpersEs._(_root);
	@override late final _TranslationsWebServerSettingsProbeEs probe = _TranslationsWebServerSettingsProbeEs._(_root);
	@override late final _TranslationsWebServerSettingsBackupEs backup = _TranslationsWebServerSettingsBackupEs._(_root);
	@override late final _TranslationsWebServerSettingsTargetRowEs targetRow = _TranslationsWebServerSettingsTargetRowEs._(_root);
	@override late final _TranslationsWebServerSettingsToggleEs toggle = _TranslationsWebServerSettingsToggleEs._(_root);
	@override String get memoryRuntimeBanner => 'El comportamiento de IA en runtime — workers, reglas de captura, perfiles de inyección y modo de spawn — vive en los ajustes de Cortex y se aplica al instante. Esta sección es la mitad de infraestructura: embedder, almacenamiento y gobernanza de fondo (requiere reinicio).';
	@override String get memoryRuntimeBannerButton => 'Abrir ajustes de Cortex';
}

// Path: web.settings
class _TranslationsWebSettingsEs extends TranslationsWebSettingsEn {
	_TranslationsWebSettingsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Ajustes';
	@override String get subtitle => 'Configuración del espacio de trabajo, la cuenta y el gateway.';
	@override late final _TranslationsWebSettingsGroupsEs groups = _TranslationsWebSettingsGroupsEs._(_root);
	@override late final _TranslationsWebSettingsItemsEs items = _TranslationsWebSettingsItemsEs._(_root);
	@override late final _TranslationsWebSettingsHealthEs health = _TranslationsWebSettingsHealthEs._(_root);
	@override late final _TranslationsWebSettingsBreadcrumbEs breadcrumb = _TranslationsWebSettingsBreadcrumbEs._(_root);
	@override late final _TranslationsWebSettingsAppearanceEs appearance = _TranslationsWebSettingsAppearanceEs._(_root);
	@override late final _TranslationsWebSettingsFontEs font = _TranslationsWebSettingsFontEs._(_root);
	@override late final _TranslationsWebSettingsAccountEs account = _TranslationsWebSettingsAccountEs._(_root);
	@override late final _TranslationsWebSettingsChangeCredentialsEs changeCredentials = _TranslationsWebSettingsChangeCredentialsEs._(_root);
	@override late final _TranslationsWebSettingsSystemEs system = _TranslationsWebSettingsSystemEs._(_root);
	@override late final _TranslationsWebSettingsAboutEs about = _TranslationsWebSettingsAboutEs._(_root);
}

// Path: web.logViewer
class _TranslationsWebLogViewerEs extends TranslationsWebLogViewerEn {
	_TranslationsWebLogViewerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get filterPlaceholder => 'Filtrar…';
	@override String get debugTooltip => 'Recuento de debug';
	@override String get infoTooltip => 'Recuento de info';
	@override String get warnTooltip => 'Recuento de advertencias';
	@override String get errorTooltip => 'Recuento de errores';
	@override String get streaming => 'Transmitiendo';
	@override String get disconnected => 'Desconectado';
	@override String get live => 'en directo';
	@override String get offline => 'sin conexión';
	@override String get pauseTooltip => 'Pausar el desplazamiento automático';
	@override String get resumeTooltip => 'Reanudar el desplazamiento automático';
	@override String get clearTooltip => 'Limpiar la vista local (el ring del servidor no se toca)';
	@override String get downloadTooltip => 'Descargar el ring completo como archivo .log';
	@override String get emptyWaiting => 'Esperando registros de log…';
	@override String emptyFiltered({required Object query}) => 'Ningún registro coincide con "${query}"';
}

// Path: web.pathInput
class _TranslationsWebPathInputEs extends TranslationsWebPathInputEn {
	_TranslationsWebPathInputEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get testButton => 'Probar';
	@override String get testTooltip => 'Resolver y comprobar esta ruta';
	@override String get notFound => 'no encontrado ·';
	@override String get childrenSuffix => 'elementos';
	@override String get expectedDirectory => '· se esperaba un directorio';
}

// Path: web.memoryAmbient
class _TranslationsWebMemoryAmbientEs extends TranslationsWebMemoryAmbientEn {
	_TranslationsWebMemoryAmbientEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebMemoryAmbientHeaderEs header = _TranslationsWebMemoryAmbientHeaderEs._(_root);
	@override String get loading => 'Cargando…';
	@override late final _TranslationsWebMemoryAmbientProvidersEs providers = _TranslationsWebMemoryAmbientProvidersEs._(_root);
	@override late final _TranslationsWebMemoryAmbientRulesEs rules = _TranslationsWebMemoryAmbientRulesEs._(_root);
	@override late final _TranslationsWebMemoryAmbientProfilesEs profiles = _TranslationsWebMemoryAmbientProfilesEs._(_root);
	@override late final _TranslationsWebMemoryAmbientCostEs cost = _TranslationsWebMemoryAmbientCostEs._(_root);
}

// Path: web.noteEditor
class _TranslationsWebNoteEditorEs extends TranslationsWebNoteEditorEn {
	_TranslationsWebNoteEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando…';
	@override String get source => 'Origen';
	@override String get preview => 'Vista previa';
	@override String tagTitle({required Object tag}) => 'etiqueta #${tag}';
	@override String get emptyNote => 'Nota vacía. Cambia a Origen para empezar a escribir.';
	@override String get saveFailedToast => 'Error al guardar';
	@override late final _TranslationsWebNoteEditorStatusEs status = _TranslationsWebNoteEditorStatusEs._(_root);
}

// Path: web.export
class _TranslationsWebExportEs extends TranslationsWebExportEn {
	_TranslationsWebExportEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Exportar datos';
	@override String get subtitle => 'Genera un paquete zip puntual de las entidades lógicas seleccionadas. Los paquetes se conservan en el servidor durante 24 horas y luego se eliminan automáticamente.';
	@override String get backToBackups => '← Backups';
	@override late final _TranslationsWebExportSectionsEs sections = _TranslationsWebExportSectionsEs._(_root);
	@override late final _TranslationsWebExportFormEs form = _TranslationsWebExportFormEs._(_root);
	@override late final _TranslationsWebExportHistoryEs history = _TranslationsWebExportHistoryEs._(_root);
	@override late final _TranslationsWebExportImportEs import = _TranslationsWebExportImportEs._(_root);
	@override late final _TranslationsWebExportImportsEs imports = _TranslationsWebExportImportsEs._(_root);
}

// Path: web.knowledge
class _TranslationsWebKnowledgeEs extends TranslationsWebKnowledgeEn {
	_TranslationsWebKnowledgeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Conocimiento';
	@override String get subtitle => 'Lo que sabemos en todos los proyectos: infraestructura y reglas fundacionales, más lecciones y funciones reutilizables destiladas del trabajo previo. Se inyecta para arrancar cada proyecto nuevo.';
	@override String get searchPlaceholder => 'Buscar conocimiento…';
	@override String get search => 'Buscar';
	@override String get browse => 'Explorar';
	@override String get cwdPlaceholder => 'Ruta del proyecto (cwd) para búsqueda con ámbito';
	@override String get noResults => 'Sin resultados.';
	@override String get empty => 'Aún no hay nada. El conocimiento se destila automáticamente mientras trabajas.';
	@override String get neighbors => 'Conexiones';
	@override String get promote => 'Promover a global';
	@override String get skillify => 'Crear habilidad';
	@override String get promoted => 'Promovido a global';
	@override String skillified({required Object title}) => 'Habilidad creada: ${title}';
	@override String get actionFailed => 'La acción falló';
	@override String get selectHint => 'Selecciona un nodo para ver los detalles.';
	@override String get scope => 'Ámbito';
	@override String get delete => 'Eliminar';
	@override String get deleted => 'Eliminado';
	@override String get deleteConfirm => '¿Eliminar este nodo? Las habilidades quedan eliminadas; los hechos/entidades derivados automáticamente pueden reaparecer en el próximo barrido.';
	@override late final _TranslationsWebKnowledgeScopesEs scopes = _TranslationsWebKnowledgeScopesEs._(_root);
	@override late final _TranslationsWebKnowledgeKbEs kb = _TranslationsWebKnowledgeKbEs._(_root);
	@override late final _TranslationsWebKnowledgeKindsEs kinds = _TranslationsWebKnowledgeKindsEs._(_root);
	@override late final _TranslationsWebKnowledgeDistillEs distill = _TranslationsWebKnowledgeDistillEs._(_root);
	@override late final _TranslationsWebKnowledgeGraphEs graph = _TranslationsWebKnowledgeGraphEs._(_root);
}

// Path: web.cortex
class _TranslationsWebCortexEs extends TranslationsWebCortexEn {
	_TranslationsWebCortexEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebCortexHomeEs home = _TranslationsWebCortexHomeEs._(_root);
	@override late final _TranslationsWebCortexChatEs chat = _TranslationsWebCortexChatEs._(_root);
	@override late final _TranslationsWebCortexBlueprintEs blueprint = _TranslationsWebCortexBlueprintEs._(_root);
	@override late final _TranslationsWebCortexQuarantineEs quarantine = _TranslationsWebCortexQuarantineEs._(_root);
	@override late final _TranslationsWebCortexSettingsEs settings = _TranslationsWebCortexSettingsEs._(_root);
}

// Path: more.identity
class _TranslationsMoreIdentityEs extends TranslationsMoreIdentityEn {
	_TranslationsMoreIdentityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get signedInAs => 'Sesión iniciada como';
	@override String get server => 'Servidor';
	@override String get tokenExpires => 'El token caduca';
}

// Path: more.sections
class _TranslationsMoreSectionsEs extends TranslationsMoreSectionsEn {
	_TranslationsMoreSectionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get gateway => 'Gateway';
	@override String get plugins => 'Complementos';
	@override String get memory => 'Memoria';
	@override String get system => 'Sistema';
}

// Path: more.items
class _TranslationsMoreItemsEs extends TranslationsMoreItemsEn {
	_TranslationsMoreItemsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsMoreItemsIntegrationsEs integrations = _TranslationsMoreItemsIntegrationsEs._(_root);
	@override late final _TranslationsMoreItemsActivityEs activity = _TranslationsMoreItemsActivityEs._(_root);
	@override late final _TranslationsMoreItemsMemoryAmbientEs memoryAmbient = _TranslationsMoreItemsMemoryAmbientEs._(_root);
	@override late final _TranslationsMoreItemsChannelsEs channels = _TranslationsMoreItemsChannelsEs._(_root);
	@override late final _TranslationsMoreItemsProvidersEs providers = _TranslationsMoreItemsProvidersEs._(_root);
	@override late final _TranslationsMoreItemsMcpEs mcp = _TranslationsMoreItemsMcpEs._(_root);
	@override late final _TranslationsMoreItemsSkillsEs skills = _TranslationsMoreItemsSkillsEs._(_root);
	@override late final _TranslationsMoreItemsGitHostsEs gitHosts = _TranslationsMoreItemsGitHostsEs._(_root);
	@override late final _TranslationsMoreItemsCustomTasksEs customTasks = _TranslationsMoreItemsCustomTasksEs._(_root);
	@override late final _TranslationsMoreItemsCortexHubEs cortexHub = _TranslationsMoreItemsCortexHubEs._(_root);
	@override late final _TranslationsMoreItemsProjectMemoryEs projectMemory = _TranslationsMoreItemsProjectMemoryEs._(_root);
	@override late final _TranslationsMoreItemsArchivedEs archived = _TranslationsMoreItemsArchivedEs._(_root);
	@override late final _TranslationsMoreItemsQuarantineEs quarantine = _TranslationsMoreItemsQuarantineEs._(_root);
	@override late final _TranslationsMoreItemsBackupsEs backups = _TranslationsMoreItemsBackupsEs._(_root);
	@override late final _TranslationsMoreItemsDataExportEs dataExport = _TranslationsMoreItemsDataExportEs._(_root);
	@override late final _TranslationsMoreItemsSettingsEs settings = _TranslationsMoreItemsSettingsEs._(_root);
	@override late final _TranslationsMoreItemsAboutEs about = _TranslationsMoreItemsAboutEs._(_root);
	@override late final _TranslationsMoreItemsVaultEs vault = _TranslationsMoreItemsVaultEs._(_root);
}

// Path: activity.filter
class _TranslationsActivityFilterEs extends TranslationsActivityFilterEn {
	_TranslationsActivityFilterEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Filtrar llamadas';
	@override String get direction => 'Dirección';
	@override String get directionAll => 'Todas';
	@override String get status => 'Estado';
	@override String get statusAll => 'Todos';
	@override String get integration => 'Integración';
	@override String get integrationAll => 'Todas las integraciones';
	@override String get apply => 'Aplicar';
	@override String get clear => 'Limpiar';
	@override String activeCount({required Object count}) => '${count} activos';
}

// Path: activity.detail
class _TranslationsActivityDetailEs extends TranslationsActivityDetailEn {
	_TranslationsActivityDetailEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Detalle de llamada';
	@override String get integration => 'Integración';
	@override String get direction => 'Dirección';
	@override String get status => 'Estado';
	@override String get duration => 'Duración';
	@override String get bytes => 'Bytes';
	@override String get requestId => 'ID de solicitud';
	@override String get resource => 'Recurso';
	@override String get timestamp => 'Marca de tiempo';
}

// Path: sessions.filters
class _TranslationsSessionsFiltersEs extends TranslationsSessionsFiltersEn {
	_TranslationsSessionsFiltersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get all => 'Todas';
	@override String get running => 'En ejecución';
	@override String get idle => 'Inactivas';
	@override String get ended => 'Finalizadas';
}

// Path: sessions.card
class _TranslationsSessionsCardEs extends TranslationsSessionsCardEn {
	_TranslationsSessionsCardEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String startedRelative({required Object provider, required Object when}) => '${provider} · iniciada ${when}';
}

// Path: sessions.empty
class _TranslationsSessionsEmptyEs extends TranslationsSessionsEmptyEn {
	_TranslationsSessionsEmptyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get titleAll => 'Aún no hay sesiones';
	@override String titleFiltered({required Object filter}) => 'Ninguna sesión coincide con el filtro "${filter}".';
	@override String get subtitleAll => 'Toca el botón Crear para crear una.';
	@override String get subtitleFiltered => 'Prueba con otro filtro o desliza para actualizar.';
}

// Path: sessions.relative
class _TranslationsSessionsRelativeEs extends TranslationsSessionsRelativeEn {
	_TranslationsSessionsRelativeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String secondsAgo({required Object n}) => 'hace ${n}s';
	@override String minutesAgo({required Object n}) => 'hace ${n}m';
	@override String hoursAgo({required Object n}) => 'hace ${n}h';
	@override String daysAgo({required Object n}) => 'hace ${n}d';
}

// Path: sessions.detail
class _TranslationsSessionsDetailEs extends TranslationsSessionsDetailEn {
	_TranslationsSessionsDetailEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get fallbackTitle => 'Sesión';
	@override String get refreshMetadata => 'Actualizar metadatos';
	@override String get inspector => 'Inspector (Archivos / Git / Tareas / Historial / Notas)';
	@override String get projectMemory => 'Memoria del proyecto (objetivo / plan / diario / bandeja de entrada)';
	@override String get actions => 'Acciones';
	@override String started({required Object when}) => 'iniciada ${when}';
	@override String startedEnded({required Object started, required Object ended}) => 'iniciada ${started}  ·  finalizada ${ended}';
	@override String idPrefix({required Object id}) => 'id: ${id}';
	@override String get errorTitle => 'No se pudo cargar la sesión';
	@override late final _TranslationsSessionsDetailAccountSwitcherEs accountSwitcher = _TranslationsSessionsDetailAccountSwitcherEs._(_root);
}

// Path: sessions.terminal
class _TranslationsSessionsTerminalEs extends TranslationsSessionsTerminalEn {
	_TranslationsSessionsTerminalEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsSessionsTerminalSnackbarEs snackbar = _TranslationsSessionsTerminalSnackbarEs._(_root);
	@override late final _TranslationsSessionsTerminalImageSourceEs imageSource = _TranslationsSessionsTerminalImageSourceEs._(_root);
	@override late final _TranslationsSessionsTerminalKeyboardEs keyboard = _TranslationsSessionsTerminalKeyboardEs._(_root);
	@override late final _TranslationsSessionsTerminalConnectionEs connection = _TranslationsSessionsTerminalConnectionEs._(_root);
}

// Path: sessions.action
class _TranslationsSessionsActionEs extends TranslationsSessionsActionEn {
	_TranslationsSessionsActionEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get stop => 'Detener';
	@override String get stopping => 'Deteniendo…';
	@override String get stopDescription => 'Envía SIGTERM, conserva el historial';
	@override String get restart => 'Reiniciar';
	@override String get restarting => 'Reiniciando…';
	@override String get restartDescription => 'Vuelve a crear el proceso del CLI';
	@override String get delete => 'Eliminar';
	@override String get deleteDescription => 'Elimina la session y su historial';
	@override String get deleteConfirm => '¿Eliminar esta session de forma permanente? Su ring buffer y su historial desaparecerán.';
	@override late final _TranslationsSessionsActionErrorsEs errors = _TranslationsSessionsActionErrorsEs._(_root);
}

// Path: sessions.dirPicker
class _TranslationsSessionsDirPickerEs extends TranslationsSessionsDirPickerEn {
	_TranslationsSessionsDirPickerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get parent => 'Superior';
	@override String get newFolder => 'Nueva carpeta';
	@override String get useThisFolder => 'Usar esta carpeta';
	@override String get loading => 'Cargando…';
	@override String get empty => 'No hay subcarpetas aquí.\nElige esta carpeta o crea una nueva.';
	@override String createdSnack({required Object path}) => 'Creada ${path}';
	@override String mkdirFailedSnack({required Object error}) => 'Falló mkdir: ${error}';
	@override late final _TranslationsSessionsDirPickerDialogEs dialog = _TranslationsSessionsDirPickerDialogEs._(_root);
}

// Path: sessions.inspector
class _TranslationsSessionsInspectorEs extends TranslationsSessionsInspectorEn {
	_TranslationsSessionsInspectorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsSessionsInspectorShellEs shell = _TranslationsSessionsInspectorShellEs._(_root);
	@override late final _TranslationsSessionsInspectorCortexEs cortex = _TranslationsSessionsInspectorCortexEs._(_root);
	@override late final _TranslationsSessionsInspectorSharedEs shared = _TranslationsSessionsInspectorSharedEs._(_root);
	@override late final _TranslationsSessionsInspectorHistoryEs history = _TranslationsSessionsInspectorHistoryEs._(_root);
	@override late final _TranslationsSessionsInspectorFilesEs files = _TranslationsSessionsInspectorFilesEs._(_root);
	@override late final _TranslationsSessionsInspectorGitEs git = _TranslationsSessionsInspectorGitEs._(_root);
	@override late final _TranslationsSessionsInspectorTasksEs tasks = _TranslationsSessionsInspectorTasksEs._(_root);
	@override late final _TranslationsSessionsInspectorNotesEs notes = _TranslationsSessionsInspectorNotesEs._(_root);
}

// Path: sessions.spawnSheet
class _TranslationsSessionsSpawnSheetEs extends TranslationsSessionsSpawnSheetEn {
	_TranslationsSessionsSpawnSheetEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Nueva session';
	@override String get errorRequired => 'El proveedor y el directorio de trabajo son obligatorios';
	@override String errorGeneric({required Object error}) => 'No se pudo crear la session: ${error}';
	@override String get cancel => 'Cancelar';
	@override String get spawn => 'Crear';
	@override String get providerLabel => 'Proveedor';
	@override String get disabledSuffix => ' (desactivado)';
	@override String get cwdLabel => 'Directorio de trabajo';
	@override String get cwdHint => '/Users/you/projects/foo';
	@override String get cwdHelper => 'Ruta absoluta en el host del gateway.';
	@override String get browse => 'Examinar';
	@override String get nameLabel => 'Nombre (opcional)';
	@override String get nameHint => 'p. ej. backend-refactor';
	@override String get argsLabel => 'Argumentos adicionales (opcional)';
	@override String get argsHint => '--continue --verbose';
	@override String get argsHelper => 'Separados por espacios; en blanco usa los valores predeterminados del proveedor.';
	@override late final _TranslationsSessionsSpawnSheetBypassEs bypass = _TranslationsSessionsSpawnSheetBypassEs._(_root);
	@override late final _TranslationsSessionsSpawnSheetNoProvidersEs noProviders = _TranslationsSessionsSpawnSheetNoProvidersEs._(_root);
	@override late final _TranslationsSessionsSpawnSheetProviderLoadErrorEs providerLoadError = _TranslationsSessionsSpawnSheetProviderLoadErrorEs._(_root);
	@override late final _TranslationsSessionsSpawnSheetClaudeAccountEs claudeAccount = _TranslationsSessionsSpawnSheetClaudeAccountEs._(_root);
}

// Path: mcp.errorPrefix
class _TranslationsMcpErrorPrefixEs extends TranslationsMcpErrorPrefixEn {
	_TranslationsMcpErrorPrefixEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get delete => 'Error al eliminar';
	@override String get add => 'Error al añadir';
	@override String get update => 'Error al actualizar';
	@override String get toggle => 'Error al alternar';
}

// Path: mcp.editor
class _TranslationsMcpEditorEs extends TranslationsMcpEditorEn {
	_TranslationsMcpEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get nameHint => 'my-mcp-server';
	@override String get jsonHint => 'Configuración JSON, nombre, transport: stdio, command, args…';
	@override String get descriptionPlaceholder => 'Descripción opcional de una línea';
	@override String get validateJsonObject => 'El cuerpo debe ser un objeto JSON';
	@override String validateJsonInvalid({required Object error}) => 'JSON no válido: ${error}';
	@override String get appBarEdit => 'Editar servidor MCP';
	@override String get appBarNew => 'Nuevo servidor MCP';
	@override String get idLockedHint => 'Bloqueado en modo edición, elimínalo y vuelve a crearlo para cambiarlo.';
	@override String get jsonLabel => 'JSON del servidor';
	@override String get jsonSchemaHelp => 'Esquema: transport debe ser stdio, http o sse. Para stdio define command + args. Para http/sse define url + headers. Usa \$secret:KEY para referenciar secretos del vault.';
	@override String get idLabel => 'id (segmento de URL, alfanumérico en minúsculas / guion / guion bajo)';
	@override String get idRequired => 'el id es obligatorio';
	@override String get saving => 'Guardando…';
	@override String get save => 'Guardar';
	@override String get create => 'Crear';
}

// Path: mcp.secret
class _TranslationsMcpSecretEs extends TranslationsMcpSecretEn {
	_TranslationsMcpSecretEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get keyLabel => 'Clave';
	@override String get keyHint => 'GITHUB_TOKEN, OPENAI_KEY, …';
	@override String get valueLabel => 'Valor';
	@override String get keyRequired => 'La clave es obligatoria.';
	@override String get keyInvalid => 'La clave debe coincidir con [A-Za-z_][A-Za-z0-9_]*, las mismas reglas que una variable de entorno de shell.';
	@override String get valueRequired => 'El valor es obligatorio.';
	@override String get replaceTitle => 'Reemplazar valor del secreto';
	@override String get addTitle => 'Añadir secreto';
	@override String get saveButton => 'Guardar';
	@override String get addButton => 'Añadir';
	@override String get helpRules => 'Reglas de variable de entorno de shell: empieza por una letra o _, después solo letras / dígitos / _.';
	@override String get replaceHint => 'Pega el nuevo valor (el anterior se borra)';
	@override String get addHint => 'Pega el valor del secreto';
	@override String addedSnack({required Object key}) => 'Secreto ${key} añadido.';
	@override String updatedSnack({required Object key}) => 'Secreto ${key} actualizado.';
	@override String deletedSnack({required Object key}) => 'Eliminado ${key}.';
	@override String get deleteBody => 'Elimina el valor del vault cifrado. Cualquier servidor MCP que lo referencie fallará hasta que se restaure.';
}

// Path: mcp.popup
class _TranslationsMcpPopupEs extends TranslationsMcpPopupEn {
	_TranslationsMcpPopupEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get editConfigSubtitle => 'Editor JSON completo, solo servidores respaldados por el vault';
	@override String get viewRawSubtitle => 'Inspector de solo lectura para el JSON del servidor';
	@override String get deleteLabel => 'Eliminar';
}

// Path: mcp.kv
class _TranslationsMcpKvEs extends TranslationsMcpKvEn {
	_TranslationsMcpKvEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get transport => 'Transport';
	@override String get description => 'Descripción';
	@override String get command => 'Command';
	@override String get args => 'Args';
	@override String get headers => 'Headers';
}

// Path: providers.errorPrefix
class _TranslationsProvidersErrorPrefixEs extends TranslationsProvidersErrorPrefixEn {
	_TranslationsProvidersErrorPrefixEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get toggle => 'Error al alternar';
	@override String get rename => 'Error al renombrar';
	@override String get delete => 'Error al eliminar';
}

// Path: providers.updateCheck
class _TranslationsProvidersUpdateCheckEs extends TranslationsProvidersUpdateCheckEn {
	_TranslationsProvidersUpdateCheckEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get sectionTitle => 'Versión del CLI';
	@override String get checking => 'Buscando actualizaciones…';
	@override String get checkFailed => 'No se pudo buscar actualizaciones';
	@override String get notInstalled => 'No instalado en el host del gateway';
	@override String installed({required Object version}) => 'Instalado: ${version}';
	@override String get upToDate => 'Actualizado';
	@override String updateAvailable({required Object version}) => 'Actualización disponible: ${version}';
	@override String latest({required Object version}) => 'última ${version}';
	@override String get updateButton => 'Actualizar CLI';
	@override String get updating => 'Actualizando…';
	@override String updatedSnack({required Object version}) => 'Actualizado a ${version}.';
	@override String get noChangeSnack => 'Ya está en la última versión.';
	@override String updateFailed({required Object error}) => 'Actualización fallida: ${error}';
	@override String notAvailableHere({required Object reason}) => 'La actualización en la app no está disponible en este host: ${reason}';
	@override String activeSessionsWarning({required Object n}) => '${n} sesión(es) activa(s) usan este CLI — actualizar no las interrumpe, pero mantienen la versión anterior hasta reiniciarse.';
}

// Path: providers.accounts
class _TranslationsProvidersAccountsEs extends TranslationsProvidersAccountsEn {
	_TranslationsProvidersAccountsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get rename => 'Renombrar';
	@override String renameTitle({required Object name}) => 'Renombrar ${name}';
	@override String get displayNameLabel => 'Nombre visible';
	@override String get displayNameHint => 'Cuenta de trabajo';
	@override String get deleteTitle => '¿Eliminar la cuenta?';
	@override String importFailedApi({required Object error}) => 'Error al importar: ${error}';
	@override String importFailedGeneric({required Object error}) => 'Error al importar: ${error}';
	@override String get enable => 'Activar';
	@override String get disable => 'Desactivar';
	@override String get deleteLabel => 'Eliminar';
	@override String get deleteBody => 'Elimina la cuenta y su token OAuth almacenado. Las sessions que ya usan esta cuenta siguen funcionando, pero la reautenticación fallará.';
	@override String deletedSnack({required Object name}) => '${name} eliminada.';
	@override String get importSyncedSnack => 'Ya está sincronizado, el gateway no tiene cuentas nuevas.';
	@override String importedSnackOne({required Object n}) => 'Se importó ${n} cuenta.';
	@override String importedSnackOther({required Object n}) => 'Se importaron ${n} cuentas.';
	@override String get importing => 'Sincronizando…';
	@override String get importLocal => 'Importar local';
	@override String get addHint => 'Añadir una cuenta nueva solo se puede hacer en el host del gateway.';
	@override String get addBody => 'El nuevo directorio aparece aquí automáticamente. Consulta la documentación para los pasos del flujo OAuth.';
	@override String loadFailed({required Object error}) => 'Error al cargar las cuentas: ${error}';
	@override String get intro => 'Las sessions creadas con el proveedor Claude eligen entre estas cuentas (o recurren a las variables de entorno).';
	@override String enabledSnack({required Object name}) => '${name} activada.';
	@override String disabledSnack({required Object name}) => '${name} desactivada.';
	@override String renamedSnack({required Object name}) => 'Renombrada a ${name}.';
	@override String activeSessions({required Object count}) => '${count} activas';
	@override String usedAgo({required Object when}) => 'usada ${when}';
	@override String get identityChanged => 'La identidad cambió';
	@override String identityWas({required Object email}) => 'era ${email}';
	@override String get acceptIdentity => 'Aceptar';
	@override String get identityAcceptedSnack => 'Cambio de identidad aceptado';
	@override String get identityAcceptFailed => 'Error al aceptar';
}

// Path: integrations.form
class _TranslationsIntegrationsFormEs extends TranslationsIntegrationsFormEn {
	_TranslationsIntegrationsFormEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get validateRequired => 'El nombre, la URL base y el prefijo de ruta son obligatorios.';
	@override String get fieldName => 'Nombre';
	@override String get fieldNameHint => 'Mi Bot';
	@override String get fieldBaseUrl => 'URL base';
	@override String get fieldRoutePrefix => 'Prefijo de ruta';
	@override String get routePrefixHelper => 'Accesible como /api/v1/<prefix>/...';
	@override String get fieldScopes => 'Ámbitos (opcional)';
	@override String get scopesHelper => 'Separados por comas. Vacío = valores por defecto del servidor.';
	@override String get fieldVersion => 'Versión (opcional)';
	@override String get validateBaseUrl => 'La URL base es obligatoria.';
	@override String get editFieldScopes => 'Ámbitos';
	@override String get editScopesHelper => 'Separados por comas.';
	@override String get editFieldVersion => 'Versión';
	@override String get apiKeyWarn => 'No volverás a ver esta key.';
	@override String get copyCopied => 'Copiado';
	@override String get copyCopy => 'Copiar';
}

// Path: memoryWorkers.tasks
class _TranslationsMemoryWorkersTasksEs extends TranslationsMemoryWorkersTasksEn {
	_TranslationsMemoryWorkersTasksEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsMemoryWorkersTasksGatekeeperEs gatekeeper = _TranslationsMemoryWorkersTasksGatekeeperEs._(_root);
	@override late final _TranslationsMemoryWorkersTasksCleanerEs cleaner = _TranslationsMemoryWorkersTasksCleanerEs._(_root);
	@override late final _TranslationsMemoryWorkersTasksGitactivityEs gitactivity = _TranslationsMemoryWorkersTasksGitactivityEs._(_root);
	@override late final _TranslationsMemoryWorkersTasksTranscriptEs transcript = _TranslationsMemoryWorkersTasksTranscriptEs._(_root);
	@override late final _TranslationsMemoryWorkersTasksPlanDriftEs planDrift = _TranslationsMemoryWorkersTasksPlanDriftEs._(_root);
	@override late final _TranslationsMemoryWorkersTasksConflictDetectorEs conflictDetector = _TranslationsMemoryWorkersTasksConflictDetectorEs._(_root);
	@override late final _TranslationsMemoryWorkersTasksCaptureEs capture = _TranslationsMemoryWorkersTasksCaptureEs._(_root);
}

// Path: project.health
class _TranslationsProjectHealthEs extends TranslationsProjectHealthEn {
	_TranslationsProjectHealthEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object days}) => 'Salud de la memoria, últimos ${days} días';
	@override String get subtitle => 'Señales agregadas de ambos subsistemas de memoria para este proyecto.';
	@override String get newFacts => 'Hechos nuevos';
	@override String newFactsHint({required Object total}) => '${total} almacenados en total';
	@override String get captureFires => 'Capturas disparadas';
	@override String captureFiresHint({required Object stored, required Object deduped}) => '${stored} almacenados · ${deduped} deduplicados';
	@override String get newJournal => 'Entradas de diario';
	@override String newJournalHint({required Object total}) => '${total} en total';
	@override String get planAge => 'Última actualización del plan';
	@override String planAgeHint({required Object count}) => '${count} propuesta(s) de desvío del plan pendiente(s)';
	@override String get planAgeHintNone => 'No hay propuestas de desvío del plan pendientes';
	@override String get goalAge => 'Última actualización del objetivo';
	@override String get pending => 'Propuestas pendientes';
	@override String pendingHint({required Object days}) => 'la más antigua tiene ${days}d';
	@override String topHit({required Object hits}) => 'Más consultado · ${hits} recuperaciones';
	@override String zeroHit({required Object count}) => '${count} hechos con más de 7d y cero recuperaciones, candidatos para limpieza.';
	@override String get never => 'nunca';
	@override String get today => 'hoy';
	@override String daysAgo({required Object count}) => 'hace ${count}d';
}

// Path: project.conflicts
class _TranslationsProjectConflictsEs extends TranslationsProjectConflictsEn {
	_TranslationsProjectConflictsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get subtitle => 'Contradicciones que el detector diario encontró entre hechos, plan, objetivo y entradas de diario.';
	@override String get empty => 'No hay conflictos pendientes. Pulsa Detectar ahora para un barrido bajo demanda.';
	@override String get detectNow => 'Detectar ahora';
	@override String detected({required Object count}) => '${count} conflicto(s) nuevo(s) encontrado(s)';
	@override String get accept => 'Aceptar';
	@override String get dismiss => 'Descartar';
	@override String deleteFact({required Object side}) => 'Eliminar hecho ${side}';
	@override String deleteConfirmTitle({required Object side}) => '¿Eliminar hecho ${side}?';
	@override String get deleteConfirmBody => 'Esto elimina el hecho de forma permanente y acepta el conflicto. El otro lado permanece como la afirmación superviviente.';
	@override String deleteWillDelete({required Object side}) => 'Se eliminará (lado ${side}):';
	@override String deleteWillKeep({required Object side}) => 'Se conservará (lado ${side}):';
	@override String deleteNonFactOther({required Object layer}) => '(entrada de ${layer}, abre la pestaña correspondiente para inspeccionar)';
	@override String get deleteLoading => 'Cargando el texto del hecho…';
	@override String deleteFactLabel({required Object side}) => 'Eliminar ${side}';
	@override String get deletedFact => 'Hecho eliminado y conflicto aceptado';
	@override String get openPlanEditor => 'Abrir el editor del plan';
	@override String get openGoalEditor => 'Abrir el editor del objetivo';
	@override late final _TranslationsProjectConflictsSeverityEs severity = _TranslationsProjectConflictsSeverityEs._(_root);
}

// Path: project.journalPrune
class _TranslationsProjectJournalPruneEs extends TranslationsProjectJournalPruneEn {
	_TranslationsProjectJournalPruneEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Purgar entradas de diario obsoletas';
	@override String subtitle({required Object days}) => 'Con más de ${days} días, sin conflictos pendientes.';
	@override String get daysLabel => 'Con más de (días):';
	@override String get empty => 'No hay nada obsoleto que purgar.';
	@override String get selectAll => 'Seleccionar todo';
	@override String get deselectAll => 'Deseleccionar todo';
	@override String deleteSelected({required Object count}) => 'Eliminar (${count})';
	@override String deleted({required Object count}) => '${count} entrada(s) eliminada(s)';
}

// Path: project.archived
class _TranslationsProjectArchivedEs extends TranslationsProjectArchivedEn {
	_TranslationsProjectArchivedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get emptyTitle => 'Nada archivado';
	@override String get emptyBody => 'No hay memorias archivadas para este proyecto. El limpiador automático archiva aquí los hechos obsoletos y duplicados automáticamente; todavía ninguno.';
	@override String restoreFailed({required Object error}) => 'Error al restaurar: ${error}';
	@override String get restore => 'Restaurar';
}

// Path: backups.kv
class _TranslationsBackupsKvEs extends TranslationsBackupsKvEn {
	_TranslationsBackupsKvEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get status => 'Estado';
	@override String get verified => 'Verificada';
	@override String get kind => 'Tipo';
	@override String get target => 'Destino';
	@override String get dedup => 'Deduplicación';
	@override String get fanout => 'Grupo de difusión';
	@override String get triggeredBy => 'Lanzado por';
	@override String get started => 'Iniciado';
	@override String get finished => 'Finalizado';
	@override String get size => 'Tamaño';
	@override String get encrypted => 'Cifrado';
	@override String get targetPath => 'Ruta de destino';
	@override String get error => 'Error';
	@override String get yes => 'sí';
	@override String get no => 'no';
}

// Path: backups.recoveryKit
class _TranslationsBackupsRecoveryKitEs extends TranslationsBackupsRecoveryKitEn {
	_TranslationsBackupsRecoveryKitEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get menuLabel => 'Kit de recuperación';
	@override String get title => 'Kit de recuperación';
	@override String get warning => 'La frase de la copia de seguridad nunca se guarda en una copia. Este kit la envuelve con una frase de recuperación que tú eliges. Guarda TANTO el kit COMO la frase de recuperación en un lugar seguro y separado: sin ellos, perder el host significa copias irrecuperables.';
	@override String get passphraseLabel => 'Frase de recuperación (mín. 8)';
	@override String get confirmLabel => 'Confirmar frase de recuperación';
	@override String get generate => 'Generar';
	@override String get copy => 'Copiar kit';
	@override String get copied => 'Kit de recuperación copiado: guárdalo de forma segura';
	@override String failed({required Object error}) => 'No se pudo generar el kit de recuperación: ${error}';
}

// Path: backups.emptyMissingDeps
class _TranslationsBackupsEmptyMissingDepsEs extends TranslationsBackupsEmptyMissingDepsEn {
	_TranslationsBackupsEmptyMissingDepsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get headline => 'Las copias de seguridad aún no pueden ejecutarse';
	@override String get body => 'Instala postgresql-client y reinicia opendray.';
}

// Path: backups.emptyNoTargets
class _TranslationsBackupsEmptyNoTargetsEs extends TranslationsBackupsEmptyNoTargetsEn {
	_TranslationsBackupsEmptyNoTargetsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get headline => 'No hay destinos de copia de seguridad configurados';
	@override String get body => 'Abre el menú Más → Destinos para añadir un destino (local / S3 / SMB / SFTP / WebDAV / rclone). Luego vuelve y toca "Ejecutar ahora".';
}

// Path: backups.emptyNoBackups
class _TranslationsBackupsEmptyNoBackupsEs extends TranslationsBackupsEmptyNoBackupsEn {
	_TranslationsBackupsEmptyNoBackupsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get headline => 'Aún no hay copias de seguridad';
	@override String get body => 'Toca "Ejecutar ahora" para tomar una nueva instantánea, o abre Programaciones para configurar ejecuciones periódicas.';
}

// Path: backups.wizard
class _TranslationsBackupsWizardEs extends TranslationsBackupsWizardEn {
	_TranslationsBackupsWizardEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Configurar copias de seguridad';
	@override String get intro => 'Elige una passphrase maestra. opendray la usa para cifrar cada blob de copia de seguridad con AES-256-GCM. Si pierdes la passphrase, pierdes los datos: no hay forma de recuperarlos.';
	@override String get saving => 'Guardando…';
	@override String get generateAndSave => 'Generar y guardar';
	@override String get savePassphrase => 'Guardar passphrase';
	@override String get generateHint => 'El servidor genera una passphrase criptográficamente aleatoria, tú la copias a un gestor de contraseñas y luego confirmas.';
	@override String get helperRecommended => 'Recomendado: más de 40 caracteres desde un gestor de contraseñas';
	@override String get saveNowHeader => 'Guarda esta passphrase AHORA';
	@override String get saveNowBody => 'Se muestra UNA SOLA VEZ. Después no podrás recuperarla desde opendray.';
}

// Path: backups.health
class _TranslationsBackupsHealthEs extends TranslationsBackupsHealthEn {
	_TranslationsBackupsHealthEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get headlineHealthy => 'Copias correctas';
	@override String get headlineAttention => 'Requiere atención';
	@override String get headlineNever => 'Aún sin copias';
	@override String get lastSuccess => 'Última copia correcta';
	@override String get never => 'nunca';
	@override late final _TranslationsBackupsHealthTilesEs tiles = _TranslationsBackupsHealthTilesEs._(_root);
}

// Path: backups.encryption
class _TranslationsBackupsEncryptionEs extends TranslationsBackupsEncryptionEn {
	_TranslationsBackupsEncryptionEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get checkAgain => 'Volver a comprobar';
	@override String get generate => 'Generar';
	@override String get paste => 'Pegar';
	@override String get random256bit => 'Clave aleatoria de 256 bits';
	@override String get passphraseLabel => 'Tu passphrase';
	@override String get passphraseHint => 'Al menos 20 caracteres';
	@override String get passphraseCopied => 'Passphrase copiada al portapapeles';
}

// Path: backups.restore
class _TranslationsBackupsRestoreEs extends TranslationsBackupsRestoreEn {
	_TranslationsBackupsRestoreEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Restaurar desde paquete';
	@override String get subtitle => 'Reproduce un paquete cifrado .tar.gz.enc en una base de datos Postgres. El paquete se sube desde este teléfono: elige un archivo generado por una copia de seguridad anterior.';
	@override String get bundleLabel => 'Archivo de paquete (.tar.gz.enc)';
	@override String get pickFile => 'Elegir archivo';
	@override String fileSelected({required Object name, required Object size}) => '${name} · ${size}';
	@override String get noFile => 'Ningún archivo seleccionado';
	@override String get targetDsnLabel => 'DSN de Postgres de destino';
	@override String get targetDsnHint => 'Déjalo vacío para restaurar en la propia base de datos de opendray.';
	@override String get targetDsnPlaceholder => 'postgres://user:pass@host:5432/dbname';
	@override String get cleanLabel => 'pg_restore --clean --if-exists';
	@override String get cleanHint => 'Elimina los objetos existentes antes de volver a crearlos.';
	@override String get auditNoteLabel => 'Nota de auditoría (opcional)';
	@override String get auditNotePlaceholder => 'p. ej. recuperando de #INC-481';
	@override String get ownDbWarning => 'Restaurar en la PROPIA base de datos de opendray reescribirá las filas que este gateway está sirviendo actualmente. Escribe "I understand" para confirmar.';
	@override String get confirmPlaceholder => 'Escribe "I understand"';
	@override String get confirmSentinel => 'I understand';
	@override String get restoring => 'Restaurando…';
	@override String get preview => 'Vista previa (simulación)';
	@override String get previewing => 'Generando vista previa…';
	@override String get previewFirstHint => 'Ejecuta primero una vista previa en simulación';
	@override String get applyRestore => 'Aplicar restauración';
	@override String get dryRunToast => 'Simulación completada — revisa el plan y luego aplícalo';
	@override String get planTitle => 'Plan de restauración (simulación — no se cambió nada)';
	@override String planDump({required Object size}) => 'Volcado de base de datos: ${size}';
	@override String planConfig({required Object path}) => 'config.toml → ${path}';
	@override String planSecrets({required Object path}) => 'secrets.env → ${path}';
	@override String planVault({required Object files, required Object roots}) => 'vault: ${files} archivos (${roots})';
	@override String get planApplyHint => 'Aplicar toma primero una instantánea de seguridad de toda la instancia, luego sobrescribe lo anterior y ejecuta pg_restore.';
	@override String get succeededTitle => 'Restauración completada';
	@override String succeededBody({required Object bytes, required Object id}) => 'Se reprodujeron ${bytes} de la copia de seguridad ${id}.';
	@override String get failedTitle => 'Error en la restauración';
	@override String get pickFileToast => 'Primero elige un archivo de paquete.';
	@override String get outputTitle => 'Salida de pg_restore';
	@override String get noPgRestoreOutput => '(vacío: la restauración se completó sin salida)';
	@override String get manifestTitle => 'Manifiesto';
	@override String get manifestBackupId => 'ID de copia de seguridad';
	@override String get manifestVersion => 'Versión del manifiesto';
	@override String get manifestCreatedAt => 'Creado';
	@override String get manifestPgVersion => 'pg_version';
	@override String get manifestOpendrayVersion => 'versión de opendray';
	@override String get fingerprint => 'Huella de la clave';
	@override String get fingerprintOk => 'coincide';
	@override String get fingerprintMismatch => 'NO COINCIDE';
	@override String get encryptionAlgo => 'Cifrado';
	@override String get bytesRead => 'Bytes leídos';
	@override String get targetDsnUsed => 'DSN de destino';
	@override String get targetDsnSelfLabel => '(la propia base de datos de opendray)';
	@override String get done => 'Hecho';
}

// Path: backups.inventory
class _TranslationsBackupsInventoryEs extends TranslationsBackupsInventoryEn {
	_TranslationsBackupsInventoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Qué contiene una copia de seguridad';
	@override String summary({required Object rows, required Object tables}) => '${rows} filas · ${tables} tablas';
	@override String get description => 'Recuentos de filas en vivo desde la base de datos Postgres de opendray. Las copias de seguridad capturan todas las filas de abajo; los artefactos binarios en disco no se incluyen.';
	@override String get rowsLabel => 'filas';
	@override String get loadFailedToast => 'Error al cargar el inventario';
	@override String get loading => 'Cargando…';
	@override String get tap => 'Toca para expandir';
}

// Path: backupTargetEditor.kinds
class _TranslationsBackupTargetEditorKindsEs extends TranslationsBackupTargetEditorKindsEn {
	_TranslationsBackupTargetEditorKindsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsBackupTargetEditorKindsLocalEs local = _TranslationsBackupTargetEditorKindsLocalEs._(_root);
	@override late final _TranslationsBackupTargetEditorKindsSmbEs smb = _TranslationsBackupTargetEditorKindsSmbEs._(_root);
	@override late final _TranslationsBackupTargetEditorKindsWebdavEs webdav = _TranslationsBackupTargetEditorKindsWebdavEs._(_root);
	@override late final _TranslationsBackupTargetEditorKindsSftpEs sftp = _TranslationsBackupTargetEditorKindsSftpEs._(_root);
	@override late final _TranslationsBackupTargetEditorKindsS3Es s3 = _TranslationsBackupTargetEditorKindsS3Es._(_root);
	@override late final _TranslationsBackupTargetEditorKindsRcloneEs rclone = _TranslationsBackupTargetEditorKindsRcloneEs._(_root);
}

// Path: githosts.errorPrefix
class _TranslationsGithostsErrorPrefixEs extends TranslationsGithostsErrorPrefixEn {
	_TranslationsGithostsErrorPrefixEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get toggle => 'Error al alternar';
	@override String get delete => 'Error al eliminar';
}

// Path: githosts.form
class _TranslationsGithostsFormEs extends TranslationsGithostsFormEn {
	_TranslationsGithostsFormEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get kindLabel => 'Tipo';
	@override String get hostLabel => 'Host';
	@override String get nameLabel => 'Nombre';
	@override String get nameHint => 'work-github, personal-gitlab, …';
	@override late final _TranslationsGithostsFormKindsEs kinds = _TranslationsGithostsFormKindsEs._(_root);
	@override String get validateHost => 'El host es obligatorio.';
	@override String get validateName => 'El nombre es obligatorio.';
	@override String get snackAdded => 'Host añadido.';
	@override String get snackUpdated => 'Host actualizado.';
	@override String saveFailedApi({required Object error}) => 'Error al guardar: ${error}';
	@override String saveFailedGeneric({required Object error}) => 'Error al guardar: ${error}';
	@override String get saving => 'Guardando…';
	@override String get save => 'Guardar';
	@override String get add => 'Añadir';
	@override String get nameHelper => 'Nombre visible que se muestra en las listas de PR.';
	@override String get tokenLabelKeep => 'Token (déjalo en blanco para conservar el actual)';
	@override String get tokenLabel => 'Token';
	@override String get tokenHintKeep => 'Déjalo en blanco para conservar el actual.';
	@override String get tokenHintNew => 'Pega el token de acceso personal.';
	@override String get enabledHelper => 'Disponible para las sessions en búsquedas de PR / remotos.';
	@override String get validateTokenRequired => 'El token es obligatorio al añadir un host.';
	@override String appBarEdit({required Object name}) => 'Editar ${name}';
	@override String get appBarNew => 'Añadir host de Git';
	@override String tokenPreviewHint({required Object preview}) => 'Vista previa actual: ${preview}';
	@override String get tokenPreviewNone => '(ninguno)';
	@override String get pausedSubtitle => 'En pausa. Las sessions omiten este host.';
}

// Path: channels.configDialog
class _TranslationsChannelsConfigDialogEs extends TranslationsChannelsConfigDialogEn {
	_TranslationsChannelsConfigDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object kind}) => 'Configuración de ${kind}';
}

// Path: channels.webhookDialog
class _TranslationsChannelsWebhookDialogEs extends TranslationsChannelsWebhookDialogEn {
	_TranslationsChannelsWebhookDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object kind}) => 'URL del webhook de ${kind}';
	@override String get copiedSnack => 'URL del webhook copiada.';
}

// Path: channels.notifications
class _TranslationsChannelsNotificationsEs extends TranslationsChannelsNotificationsEn {
	_TranslationsChannelsNotificationsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Preferencias de notificación';
	@override String get repeatPolicy => 'Política de repetición';
	@override String get cooldownWindow => 'Ventana de cooldown';
	@override String get includeSnippet => 'Incluir fragmento del terminal';
	@override String get snippetLengthCap => 'Límite de longitud del fragmento';
	@override String get snippetHelper => 'Incrusta el final reciente del terminal en cada notificación.';
	@override String get snippetNoCap => 'sin límite';
	@override String snippetChars({required Object n}) => '${n} caracteres';
	@override String get updatedSnack => 'Preferencias de notificación actualizadas.';
	@override late final _TranslationsChannelsNotificationsModesEs modes = _TranslationsChannelsNotificationsModesEs._(_root);
}

// Path: channels.popup
class _TranslationsChannelsPopupEs extends TranslationsChannelsPopupEn {
	_TranslationsChannelsPopupEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get enable => 'Activar';
	@override String get disable => 'Desactivar';
	@override String get mute => 'Silenciar';
	@override String get unmute => 'Reactivar sonido';
	@override String get deleteLabel => 'Eliminar';
}

// Path: channels.badges
class _TranslationsChannelsBadgesEs extends TranslationsChannelsBadgesEn {
	_TranslationsChannelsBadgesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get running => 'en ejecución';
	@override String get starting => 'iniciando…';
	@override String get disabled => 'desactivado';
	@override String get muted => 'silenciado';
}

// Path: channels.snacks
class _TranslationsChannelsSnacksEs extends TranslationsChannelsSnacksEn {
	_TranslationsChannelsSnacksEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get testDispatched => 'Mensaje de prueba enviado.';
	@override String get channelEnabled => 'Canal activado.';
	@override String get channelDisabled => 'Canal desactivado.';
	@override String get channelMuted => 'Canal silenciado.';
	@override String get channelUnmuted => 'Sonido del canal reactivado.';
	@override String get configUpdated => 'Configuración del canal actualizada.';
	@override String get channelDeleted => 'Canal eliminado.';
}

// Path: channels.errorPrefix
class _TranslationsChannelsErrorPrefixEs extends TranslationsChannelsErrorPrefixEn {
	_TranslationsChannelsErrorPrefixEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get test => 'Error en la prueba';
	@override String get toggle => 'Error al alternar';
	@override String get muteToggle => 'Error al alternar el silencio';
	@override String get update => 'Error al actualizar';
	@override String get delete => 'Error al eliminar';
}

// Path: channels.kinds
class _TranslationsChannelsKindsEs extends TranslationsChannelsKindsEn {
	_TranslationsChannelsKindsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsChannelsKindsTelegramEs telegram = _TranslationsChannelsKindsTelegramEs._(_root);
	@override late final _TranslationsChannelsKindsSlackEs slack = _TranslationsChannelsKindsSlackEs._(_root);
	@override late final _TranslationsChannelsKindsDiscordEs discord = _TranslationsChannelsKindsDiscordEs._(_root);
	@override late final _TranslationsChannelsKindsFeishuEs feishu = _TranslationsChannelsKindsFeishuEs._(_root);
	@override late final _TranslationsChannelsKindsDingtalkEs dingtalk = _TranslationsChannelsKindsDingtalkEs._(_root);
	@override late final _TranslationsChannelsKindsWecomEs wecom = _TranslationsChannelsKindsWecomEs._(_root);
}

// Path: notesPage.editor
class _TranslationsNotesPageEditorEs extends TranslationsNotesPageEditorEn {
	_TranslationsNotesPageEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get markdownHint => 'Markdown…';
	@override String get saving => 'Guardando…';
	@override String get autosave => 'Se guarda automáticamente mientras escribes';
	@override String loadFailedApi({required Object error}) => 'Error al cargar: ${error}';
	@override String loadFailedGeneric({required Object error}) => 'Error al cargar: ${error}';
	@override String saveFailedApi({required Object error}) => 'Error al guardar: ${error}';
	@override String saveFailedGeneric({required Object error}) => 'Error al guardar: ${error}';
	@override String savedAt({required Object time}) => 'Guardado ${time}';
}

// Path: dataExport.sections
class _TranslationsDataExportSectionsEs extends TranslationsDataExportSectionsEn {
	_TranslationsDataExportSectionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get export => 'Exportar';
	@override String get import => 'Importar';
}

// Path: dataExport.form
class _TranslationsDataExportFormEs extends TranslationsDataExportFormEn {
	_TranslationsDataExportFormEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get scope => 'Alcance';
	@override String get memories => 'Memorias';
	@override String get memoriesHint => 'Todas las memorias persistidas y sus embeddings.';
	@override String get integrations => 'Integraciones';
	@override late final _TranslationsDataExportFormIntegrationOptionsEs integrationOptions = _TranslationsDataExportFormIntegrationOptionsEs._(_root);
	@override String get confirmWarning => 'La exportación de claves en texto plano contiene secretos descifrables. Escribe "Lo entiendo" para confirmar.';
	@override String get confirmPlaceholder => 'Escribe "Lo entiendo"';
	@override String get confirmSentinel => 'Lo entiendo';
	@override String get customTasks => 'Tareas personalizadas';
	@override String get customTasksHint => 'Definiciones de tareas por usuario (programaciones cron y cuerpos de script).';
	@override String get footnote => 'Los paquetes caducan 7 días después de su creación. El enlace de descarga es de un solo uso.';
	@override String get create => 'Crear paquete';
	@override String get building => 'Creando…';
	@override String get readyToast => 'Paquete listo';
	@override String readyDescription({required Object bytes}) => '${bytes} bytes, descárgalo desde el historial de abajo.';
	@override String failedToast({required Object error}) => 'Error al crear el paquete: ${error}';
}

// Path: dataExport.history
class _TranslationsDataExportHistoryEs extends TranslationsDataExportHistoryEn {
	_TranslationsDataExportHistoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Historial de exportaciones';
	@override String get loading => 'Cargando…';
	@override String get empty => 'Aún no hay exportaciones.';
	@override String listFailedToast({required Object error}) => 'Error al cargar las exportaciones: ${error}';
	@override String downloadFailedToast({required Object error}) => 'Error al obtener el token de descarga: ${error}';
	@override String get noTokenToast => 'Esta exportación no tiene un token de descarga utilizable (ya consumido o caducado).';
	@override String get deletedToast => 'Exportación eliminada.';
	@override String deleteFailedToast({required Object error}) => 'Error al eliminar la exportación: ${error}';
	@override String get deleteConfirmTitle => '¿Eliminar la exportación?';
	@override String deleteConfirmBody({required Object id}) => 'Elimina el paquete y revoca el token de descarga. ${id}';
	@override String get download => 'Descargar';
	@override String get delete => 'Eliminar';
	@override String get downloadCopiedToast => 'URL de descarga copiada al portapapeles. Pégala en un navegador para obtenerla (un solo uso).';
	@override late final _TranslationsDataExportHistoryColumnsEs columns = _TranslationsDataExportHistoryColumnsEs._(_root);
	@override String get scopeEmpty => '(vacío)';
	@override String get scopeMemories => 'memorias';
	@override String scopeIntegrations({required Object mode}) => 'integraciones(${mode})';
	@override String get scopeCustomTasks => 'custom_tasks';
}

// Path: dataExport.import
class _TranslationsDataExportImportEs extends TranslationsDataExportImportEn {
	_TranslationsDataExportImportEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get intro => 'Reproduce un paquete generado previamente por Exportar. Solo se importan las entidades que marques abajo; todo lo demás en el paquete se ignora.';
	@override String get bundleLabel => 'Archivo de paquete (.zip)';
	@override String get pickFile => 'Elegir archivo';
	@override String fileSelected({required Object name, required Object size}) => '${name} · ${size}';
	@override String get noFile => 'Ningún archivo seleccionado';
	@override String get memoriesLabel => 'Memorias';
	@override String get integrationsLabel => 'Integraciones';
	@override String get customTasksLabel => 'Tareas personalizadas';
	@override String get importBundle => 'Importar paquete';
	@override String get importing => 'Importando…';
	@override String get pickFileToast => 'Elige primero un archivo de paquete.';
	@override String get doneToast => 'Importación completada';
	@override String get finishedWithErrors => 'Importación finalizada con errores';
	@override String failedToast({required Object error}) => 'Error en la importación: ${error}';
	@override late final _TranslationsDataExportImportSummaryCardEs summaryCard = _TranslationsDataExportImportSummaryCardEs._(_root);
}

// Path: dataExport.imports
class _TranslationsDataExportImportsEs extends TranslationsDataExportImportsEn {
	_TranslationsDataExportImportsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Historial de importaciones';
	@override String get loading => 'Cargando…';
	@override String get empty => 'Aún no hay importaciones.';
	@override String listFailedToast({required Object error}) => 'Error al cargar las importaciones: ${error}';
	@override String get noneCounts => '(sin recuentos)';
	@override String get sourceUnknown => '(origen desconocido)';
	@override late final _TranslationsDataExportImportsColumnsEs columns = _TranslationsDataExportImportsColumnsEs._(_root);
}

// Path: dataExport.relative
class _TranslationsDataExportRelativeEs extends TranslationsDataExportRelativeEn {
	_TranslationsDataExportRelativeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String inSeconds({required Object n}) => 'en ${n}s';
	@override String inMinutes({required Object n}) => 'en ${n}m';
	@override String inHours({required Object n}) => 'en ${n}h';
	@override String inDays({required Object n}) => 'en ${n}d';
	@override String secondsAgo({required Object n}) => 'hace ${n}s';
	@override String minutesAgo({required Object n}) => 'hace ${n}m';
	@override String hoursAgo({required Object n}) => 'hace ${n}h';
}

// Path: dataExport.status
class _TranslationsDataExportStatusEs extends TranslationsDataExportStatusEn {
	_TranslationsDataExportStatusEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get pending => 'pendiente';
	@override String get running => 'en ejecución';
	@override String get ready => 'listo';
	@override String get failed => 'fallido';
	@override String get expired => 'caducado';
	@override String get succeeded => 'completado';
}

// Path: memory.status
class _TranslationsMemoryStatusEs extends TranslationsMemoryStatusEn {
	_TranslationsMemoryStatusEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Embedder activo';
	@override String dimensions({required Object dim, required Object state}) => '${dim}-dim · ${state}';
	@override String get enabled => 'habilitado';
	@override String get disabled => 'deshabilitado';
	@override String get floorNoModel => 'Solo recuperación por palabras clave (BM25) — no hay modelo de embedding configurado. Configura un endpoint denso en Settings para habilitar la memoria semántica.';
	@override String denseConfiguredPendingRestart({required Object model}) => 'Configurado ${model} (denso) — reinicia el gateway para activar la memoria semántica y re-embeber las memorias existentes.';
	@override String denseUnreachableFloor({required Object model}) => 'Configurado ${model} (denso) pero el endpoint está inalcanzable — se usa el piso de palabras clave hasta que responda (se actualiza al reiniciar).';
	@override String get denseDegraded => 'Embedder denso activo pero su endpoint está inalcanzable ahora — los vectores existentes se conservan; las nuevas escrituras y la búsqueda por similitud se pausan hasta que responda.';
}

// Path: memory.rank
class _TranslationsMemoryRankEs extends TranslationsMemoryRankEn {
	_TranslationsMemoryRankEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Desglose del ranking';
	@override String effective({required Object value}) => 'Puntuación efectiva: ${value}';
	@override String get similarity => 'Similitud del coseno';
	@override String ageMultiplier({required Object days}) => 'Multiplicador por antigüedad (${days}d de antigüedad)';
	@override String hitMultiplier({required Object hits}) => 'Multiplicador por número de hits (${hits} hits)';
	@override String get confidenceMultiplier => 'Multiplicador por confianza';
	@override String get formula => 'effective = similarity × age × hits × confidence';
	@override String get close => 'Cerrar';
}

// Path: memory.deleteAllConfirm
class _TranslationsMemoryDeleteAllConfirmEs extends TranslationsMemoryDeleteAllConfirmEn {
	_TranslationsMemoryDeleteAllConfirmEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => '¿Eliminar todas las memorias de este ámbito?';
	@override String get deleteAll => 'Eliminar todas';
}

// Path: memory.deleteOne
class _TranslationsMemoryDeleteOneEs extends TranslationsMemoryDeleteOneEn {
	_TranslationsMemoryDeleteOneEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => '¿Eliminar memoria?';
	@override String get body => 'Esto no se puede deshacer.';
}

// Path: memory.scope
class _TranslationsMemoryScopeEs extends TranslationsMemoryScopeEn {
	_TranslationsMemoryScopeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get project => 'Proyecto';
	@override String get global => 'Global';
}

// Path: memory.create
class _TranslationsMemoryCreateEs extends TranslationsMemoryCreateEn {
	_TranslationsMemoryCreateEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get textLabel => 'Texto';
	@override String get scopeKeyLabel => 'Clave de ámbito (cwd del proyecto)';
	@override String get scopeKeyHint => '/Users/you/projects/foo';
	@override String get submit => 'Crear';
}

// Path: memory.reembed
class _TranslationsMemoryReembedEs extends TranslationsMemoryReembedEn {
	_TranslationsMemoryReembedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get menuItem => 'Reincrustar todo';
	@override String get confirmTitle => '¿Reincrustar todas las memorias?';
	@override String get confirmBody => 'Recodifica cada memoria y página de KB con el modelo de embedding actual. Necesario tras cambiar de modelo, ya que la dimensión del vector cambia. Puede tardar un rato.';
	@override String get confirmButton => 'Reincrustar';
	@override String get running => 'Reincrustando… esto puede tardar.';
	@override String done({required Object count}) => 'Se reincrustaron ${count} memorias.';
	@override String failed({required Object error}) => 'Falló la reincrustación: ${error}';
}

// Path: about.sections
class _TranslationsAboutSectionsEs extends TranslationsAboutSectionsEn {
	_TranslationsAboutSectionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get app => 'App';
	@override String get server => 'Servidor';
	@override String get gateway => 'Gateway';
}

// Path: about.fields
class _TranslationsAboutFieldsEs extends TranslationsAboutFieldsEn {
	_TranslationsAboutFieldsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get app => 'App';
	@override String get version => 'Versión';
	@override String versionFormat({required Object version, required Object build}) => '${version} (build ${build})';
	@override String get package => 'Paquete';
	@override String get url => 'URL';
	@override String get signedInAs => 'Sesión iniciada como';
	@override String get tokenExpires => 'El token caduca';
}

// Path: about.copyLabels
class _TranslationsAboutCopyLabelsEs extends TranslationsAboutCopyLabelsEn {
	_TranslationsAboutCopyLabelsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get version => 'versión';
	@override String get serverUrl => 'URL del servidor';
}

// Path: about.gateway
class _TranslationsAboutGatewayEs extends TranslationsAboutGatewayEn {
	_TranslationsAboutGatewayEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get version => 'Versión';
	@override String get commit => 'Commit';
	@override String get checking => 'Buscando actualizaciones…';
	@override String get upToDate => 'Actualizado';
	@override String updateAvailable({required Object version}) => 'Actualización disponible: ${version}';
	@override String get releaseNotes => 'Notas de la versión';
	@override String get checkFailed => 'Comprobación de actualizaciones no disponible';
}

// Path: settings.language
class _TranslationsSettingsLanguageEs extends TranslationsSettingsLanguageEn {
	_TranslationsSettingsLanguageEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get section => 'Idioma';
	@override String get system => 'Sistema';
	@override String get systemSubtitle => 'Sigue la configuración de idioma de tu teléfono';
	@override String get english => 'English';
	@override String get chinese => '中文';
	@override String get spanish => 'Español';
}

// Path: settings.appearance
class _TranslationsSettingsAppearanceEs extends TranslationsSettingsAppearanceEn {
	_TranslationsSettingsAppearanceEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get section => 'Apariencia';
	@override String get system => 'Sistema';
	@override String get systemSubtitle => 'Sigue la configuración de apariencia de tu teléfono';
	@override String get light => 'Claro';
	@override String get lightSubtitle => 'Usa siempre la paleta clara';
	@override String get dark => 'Oscuro';
	@override String get darkSubtitle => 'Usa siempre la paleta oscura';
}

// Path: settings.account
class _TranslationsSettingsAccountEs extends TranslationsSettingsAccountEn {
	_TranslationsSettingsAccountEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get section => 'Cuenta';
	@override String get changeCredentials => 'Cambiar credenciales';
	@override String get changeCredentialsSubtitle => 'Usuario y contraseña';
}

// Path: settings.gateway
class _TranslationsSettingsGatewayEs extends TranslationsSettingsGatewayEn {
	_TranslationsSettingsGatewayEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get section => 'Gateway';
	@override String get serverSettings => 'Ajustes del servidor';
	@override String get serverSettingsSubtitle => 'Dirección de escucha, registro, vault, memoria, rutas de almacenamiento…';
	@override String get liveLogs => 'Logs en vivo';
	@override String get liveLogsSubtitle => 'Sigue el búfer de logs del gateway (la misma fuente que el panel web)';
}

// Path: settings.changeCredentials
class _TranslationsSettingsChangeCredentialsEs extends TranslationsSettingsChangeCredentialsEn {
	_TranslationsSettingsChangeCredentialsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cambiar credenciales';
	@override String get explanation => 'Verifica tu contraseña actual y luego elige nuevas credenciales. Todas las demás sessions con sesión iniciada se revocarán.';
	@override String get currentPassword => 'Contraseña actual';
	@override String get newUsername => 'Nuevo usuario';
	@override String get newPassword => 'Nueva contraseña';
	@override String get confirmPassword => 'Confirma la nueva contraseña';
	@override String get validatorRequired => 'Obligatorio';
	@override String get passwordHelper => 'Al menos 8 caracteres';
	@override String get passwordTooShort => 'Debe tener al menos 8 caracteres';
	@override String get passwordMismatch => 'No coincide con la nueva contraseña';
	@override String get updatedSnack => 'Credenciales actualizadas.';
	@override String get wrongCurrent => 'La contraseña actual es incorrecta.';
	@override String get saving => 'Guardando…';
	@override String get update => 'Actualizar';
}

// Path: settings.logViewer
class _TranslationsSettingsLogViewerEs extends TranslationsSettingsLogViewerEn {
	_TranslationsSettingsLogViewerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Logs en vivo';
	@override String get reconnect => 'Reconectar';
	@override String get copyBuffer => 'Copiar búfer';
	@override String get clearLocal => 'Borrar vista local';
	@override String get copiedSnack => 'Búfer copiado al portapapeles';
	@override String get filterHint => 'Filtrar subcadena…';
	@override late final _TranslationsSettingsLogViewerLevelsEs levels = _TranslationsSettingsLogViewerLevelsEs._(_root);
}

// Path: settings.serverSettings
class _TranslationsSettingsServerSettingsEs extends TranslationsSettingsServerSettingsEn {
	_TranslationsSettingsServerSettingsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Ajustes del servidor';
	@override String get reloadTooltip => 'Recargar desde el servidor';
	@override String get restartTooltip => 'Reiniciar gateway';
	@override String get restartConfirmTitle => '¿Reiniciar opendray?';
	@override String get restartConfirmBody => 'El gateway se ejecutará de nuevo a sí mismo. La app móvil puede perder la conexión brevemente.';
	@override String get restart => 'Reiniciar';
	@override String get restartQueuedSnack => 'Reinicio solicitado. Desliza para actualizar en un momento.';
	@override String restartFailedApi({required Object error}) => 'Falló el reinicio: ${error}';
	@override String restartFailedGeneric({required Object error}) => 'Falló el reinicio: ${error}';
	@override String loadedFrom({required Object path}) => 'Cargado desde: ${path}';
	@override String get restartHint => 'La mayoría de las secciones necesitan un reinicio del gateway para surtir efecto. El botón de reinicio está en la AppBar.';
	@override String get savedNeedsRestart => 'Guardado. Reinicia el gateway para aplicar.';
	@override String get savedSimple => 'Guardado.';
	@override String get changesNeedRestart => 'Los cambios en esta sección necesitan un reinicio del gateway.';
	@override String get loadFailed => 'No se pudieron cargar los ajustes del servidor';
	@override late final _TranslationsSettingsServerSettingsSectionsEs sections = _TranslationsSettingsServerSettingsSectionsEs._(_root);
	@override late final _TranslationsSettingsServerSettingsSectionDescriptionsEs sectionDescriptions = _TranslationsSettingsServerSettingsSectionDescriptionsEs._(_root);
	@override late final _TranslationsSettingsServerSettingsFieldsEs fields = _TranslationsSettingsServerSettingsFieldsEs._(_root);
	@override String validateInteger({required Object field}) => '"${field}" debe ser un entero';
	@override String validateNumber({required Object field}) => '"${field}" debe ser un número';
	@override late final _TranslationsSettingsServerSettingsEmbedderModelEs embedderModel = _TranslationsSettingsServerSettingsEmbedderModelEs._(_root);
}

// Path: web.sessions.list
class _TranslationsWebSessionsListEs extends TranslationsWebSessionsListEn {
	_TranslationsWebSessionsListEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Sesiones';
	@override String get countSeparator => '·';
	@override String get newAria => 'Crear nueva session';
	@override String get newTooltip => 'Nueva session';
	@override String get loading => 'Cargando…';
	@override String get emptyTitle => 'Aún no hay sesiones.';
	@override String emptyHint({required Object kbd}) => 'Pulsa ${kbd} para crear una.';
	@override String endedHeader({required Object count}) => 'Finalizadas (${count})';
	@override String get clearAll => 'Borrar todas';
	@override String confirmClearAll({required Object count}) => '¿Eliminar las ${count} sesiones finalizadas?';
	@override String confirmTerminate({required Object name}) => '¿Terminar y eliminar ${name}?';
	@override String childPromoted({required Object count}) => ' ${count} session de tarea secundaria pasará al nivel superior.';
	@override String childPromotedPlural({required Object count}) => ' ${count} sesiones de tarea secundaria pasarán al nivel superior.';
	@override String footer({required Object live, required Object ended}) => '${live} activas · ${ended} finalizadas';
	@override late final _TranslationsWebSessionsListRowEs row = _TranslationsWebSessionsListRowEs._(_root);
	@override String get deleteFailedToast => 'Error al eliminar';
}

// Path: web.sessions.tabs
class _TranslationsWebSessionsTabsEs extends TranslationsWebSessionsTabsEn {
	_TranslationsWebSessionsTabsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get closeAria => 'Cerrar pestaña y eliminar session';
	@override String get closeTitle => 'Cerrar pestaña y eliminar session';
}

// Path: web.sessions.page
class _TranslationsWebSessionsPageEs extends TranslationsWebSessionsPageEn {
	_TranslationsWebSessionsPageEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get removedToast => 'Session eliminada';
	@override String get removeFailedToast => 'Error al eliminar';
	@override String get stoppedToast => 'Session detenida';
	@override String get stopFailedToast => 'Error al detener';
	@override String get restartedToast => 'Session reiniciada';
	@override String get restartFailedToast => 'Error al reiniciar';
	@override String confirmCloseTabTitle({required Object name}) => '¿Detener y eliminar "${name}"?';
	@override String get confirmCloseTabDescription => 'El proceso de la CLI se terminará y la fila se eliminará.';
	@override String get confirmCloseTabConfirm => 'Detener y eliminar';
	@override String confirmRemoveTitle({required Object name}) => '¿Eliminar ${name}?';
	@override String get confirmRemoveTitleFallback => '¿Eliminar session?';
	@override String get confirmRemoveDescription => 'Esto elimina la fila.';
	@override String get confirmRemoveConfirm => 'Eliminar';
}

// Path: web.sessions.empty
class _TranslationsWebSessionsEmptyEs extends TranslationsWebSessionsEmptyEn {
	_TranslationsWebSessionsEmptyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Ninguna session abierta';
	@override String hint({required Object kbdN, required Object kbdW, required Object kbdRange}) => 'Elige una session de la lista o crea una nueva. Teclado: ${kbdN} nueva, ${kbdW} cerrar, ${kbdRange} cambiar.';
	@override String get spawn => 'Crear session';
}

// Path: web.sessions.header
class _TranslationsWebSessionsHeaderEs extends TranslationsWebSessionsHeaderEn {
	_TranslationsWebSessionsHeaderEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loadingSession => 'Cargando session…';
	@override String get showList => 'Mostrar lista de sesiones';
	@override String get hideList => 'Ocultar lista de sesiones';
	@override String get showInspector => 'Mostrar inspector';
	@override String get hideInspector => 'Ocultar inspector';
	@override String get attachImage => 'Adjuntar imagen';
	@override String get attachImageTooltip => 'Adjuntar imagen (o pega / suelta en el terminal)';
	@override String get copyOutput => 'Copiar salida';
	@override String get copyOutputTooltip => 'Copia la salida del terminal (la selección si hay alguna, si no, todo)';
	@override String get restart => 'Reiniciar';
	@override String get restarting => 'Reiniciando…';
	@override String get remove => 'Eliminar';
	@override String get removing => 'Eliminando…';
	@override String get stop => 'Detener';
	@override String get stopping => 'Deteniendo…';
	@override String pid({required Object pid}) => 'pid ${pid}';
}

// Path: web.sessions.terminal
class _TranslationsWebSessionsTerminalEs extends TranslationsWebSessionsTerminalEn {
	_TranslationsWebSessionsTerminalEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get uploadingToast => 'Subiendo imagen…';
	@override String get uploadedToast => 'Imagen adjuntada';
	@override String get uploadFailedToast => 'Error al subir';
	@override String get uploadInvalidTypeToast => 'Solo se pueden adjuntar archivos de imagen';
	@override String get dropToAttach => 'Suelta la imagen para adjuntarla';
	@override String get copyButton => 'Copiar';
	@override String get copyAllTooltip => 'Copia la salida del terminal al portapapeles (la selección si hay alguna, si no, todo)';
	@override String get copySelection => 'Copiar';
	@override String get copySelectionTooltip => 'Copiar el texto seleccionado';
	@override String get copiedToast => 'Copiado al portapapeles';
	@override String get copyEmptyToast => 'Aún no hay nada que copiar';
	@override String get copyFailedToast => 'No se pudo copiar al portapapeles';
	@override late final _TranslationsWebSessionsTerminalUrlsEs urls = _TranslationsWebSessionsTerminalUrlsEs._(_root);
}

// Path: web.sessions.spawn
class _TranslationsWebSessionsSpawnEs extends TranslationsWebSessionsSpawnEn {
	_TranslationsWebSessionsSpawnEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Crear session';
	@override String get description => 'Inicia una session de la CLI con un proveedor registrado.';
	@override String get provider => 'Proveedor';
	@override String get claudeAccount => 'Cuenta de Claude';
	@override String get loadingAccounts => 'Cargando cuentas…';
	@override String get noAccounts => 'No se han encontrado cuentas de Claude. Crea esta session y ejecuta <1>claude login</1> en el terminal. Las credenciales acaban en <3>~/.claude</3> en el gateway y aparecen automáticamente la próxima vez.';
	@override String get kDefault => 'Predeterminada';
	@override String get defaultTooltip => 'Usar el keychain del sistema / env';
	@override String get tokenEmptyBadge => '·vacío';
	@override String get tokenMissingTooltip => 'No hay token configurado. Configura el token primero en el panel de Proveedores';
	@override String get multiAccountHint => 'Hay varias cuentas configuradas. Elige una para esta session.';
	@override String get cwd => 'Directorio de trabajo';
	@override String get cwdPlaceholder => '/Users/you/projects/foo';
	@override String get browse => 'Examinar';
	@override String get nameLabel => 'Nombre (opcional)';
	@override String get namePlaceholder => 'claude in pet-tracker';
	@override String get argsLabel => 'Argumentos de la CLI (uno por línea)';
	@override String get bypassClaude => 'Omitir las confirmaciones de permisos';
	@override String get bypassCodex => 'Omitir aprobaciones y sandbox (--dangerously-bypass-approvals-and-sandbox)';
	@override String get bypassGemini => 'Modo YOLO (--yolo)';
	@override String get bypassAntigravity => 'Omitir permisos / YOLO (--dangerously-skip-permissions)';
	@override String get bypassOnHint => 'Esta session se ejecutará con autonomía elevada.';
	@override String get bypassOffHint => 'Desactivado. Las confirmaciones y los prompts se comportan con normalidad.';
	@override String get errorPickProvider => 'Elige un proveedor.';
	@override String get errorCwdRequired => 'cwd es obligatorio.';
	@override String get cancel => 'Cancelar';
	@override String get submit => 'Crear';
	@override String get submitting => 'Creando…';
	@override String get spawnedToast => 'Session creada';
	@override String spawnedDescription({required Object provider, required Object pid}) => '${provider} · pid ${pid}';
	@override String get pidFallback => '—';
}

// Path: web.sessions.accountSwitcher
class _TranslationsWebSessionsAccountSwitcherEs extends TranslationsWebSessionsAccountSwitcherEn {
	_TranslationsWebSessionsAccountSwitcherEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get tooltip => 'Cambiar de cuenta de Claude (reinicia el proceso de la CLI)';
	@override String get currentDefault => 'predeterminada';
	@override String get menuTitle => 'Cambiar de cuenta de Claude';
	@override String get defaultName => 'Predeterminada';
	@override String get defaultSubtitle => 'keychain del sistema / env de la CLI';
	@override String get tokenEmpty => '·vacío';
	@override String get confirmSwitch => 'Cambiar de cuenta reiniciará la CLI de Claude. El historial de la conversación se conserva (el transcript se migra y --resume lo reproduce con la nueva cuenta), pero se perderá cualquier ejecución de herramienta en curso o entrada sin enviar. ¿Continuar?';
	@override String get switchedToast => 'Cuenta cambiada';
	@override String switchedDescription({required Object account, required Object pid}) => 'Ahora usando @${account} · pid ${pid}';
	@override String get switchedDefault => 'predeterminada';
	@override String get switchFailedToast => 'Error al cambiar';
}

// Path: web.sessions.inspector
class _TranslationsWebSessionsInspectorEs extends TranslationsWebSessionsInspectorEn {
	_TranslationsWebSessionsInspectorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebSessionsInspectorTabsEs tabs = _TranslationsWebSessionsInspectorTabsEs._(_root);
	@override late final _TranslationsWebSessionsInspectorVaultPanelEs vaultPanel = _TranslationsWebSessionsInspectorVaultPanelEs._(_root);
	@override late final _TranslationsWebSessionsInspectorCortexPanelEs cortexPanel = _TranslationsWebSessionsInspectorCortexPanelEs._(_root);
}

// Path: web.sessions.ended
class _TranslationsWebSessionsEndedEs extends TranslationsWebSessionsEndedEn {
	_TranslationsWebSessionsEndedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get bufferUnavailable => '[búfer no disponible]';
	@override String get readOnlyBanner => '[session finalizada. búfer de solo lectura]';
}

// Path: web.sessions.fileBrowser
class _TranslationsWebSessionsFileBrowserEs extends TranslationsWebSessionsFileBrowserEn {
	_TranslationsWebSessionsFileBrowserEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Elige el directorio de trabajo';
	@override String get description => 'Examina el sistema de archivos del host del gateway y elige una carpeta.';
	@override String get parent => 'Directorio superior';
	@override String get home => 'Directorio personal';
	@override String get refresh => 'Actualizar';
	@override String get pathPlaceholder => '/Users/you/projects';
	@override String get loading => 'Cargando…';
	@override String get empty => 'Directorio vacío.';
	@override String get newFolder => 'Nueva carpeta';
	@override String get newFolderPlaceholder => 'nombre-de-la-carpeta';
	@override String get create => 'Crear';
	@override String get cancel => 'Cancelar';
	@override String get useThisFolder => 'Usar esta carpeta';
	@override String get createdToast => 'Directorio creado';
	@override String get mkdirFailedToast => 'Error al crear el directorio';
	@override String get homeFailedToast => 'Error al leer el directorio personal';
}

// Path: web.conflicts.confirmDelete
class _TranslationsWebConflictsConfirmDeleteEs extends TranslationsWebConflictsConfirmDeleteEn {
	_TranslationsWebConflictsConfirmDeleteEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object side}) => '¿Eliminar el hecho ${side}?';
	@override String get description => 'Esto elimina el hecho de forma permanente y acepta el conflicto. El otro lado se conserva como la afirmación superviviente.';
	@override String targetLabel({required Object side}) => 'Se eliminará (lado ${side}):';
	@override String keepLabel({required Object side}) => 'Se conservará (lado ${side}):';
	@override String nonFactOther({required Object layer}) => '(entrada de ${layer}, abre la pestaña correspondiente para inspeccionar)';
	@override String get evidenceLabel => 'Evidencia del detector:';
	@override String get loading => 'Cargando texto del hecho…';
	@override String get loadError => 'No se pudo cargar el texto del hecho. Inspecciónalo en la página de Memoria.';
	@override String get cancel => 'Cancelar';
	@override String confirm({required Object side}) => 'Eliminar ${side}';
}

// Path: web.conflicts.openLayer
class _TranslationsWebConflictsOpenLayerEs extends TranslationsWebConflictsOpenLayerEn {
	_TranslationsWebConflictsOpenLayerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get plan => 'Abrir editor de plan';
	@override String get goal => 'Abrir editor de objetivo';
}

// Path: web.conflicts.severity
class _TranslationsWebConflictsSeverityEs extends TranslationsWebConflictsSeverityEn {
	_TranslationsWebConflictsSeverityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get low => 'baja';
	@override String get medium => 'media';
	@override String get high => 'alta';
}

// Path: web.memoryConfig.sections
class _TranslationsWebMemoryConfigSectionsEs extends TranslationsWebMemoryConfigSectionsEn {
	_TranslationsWebMemoryConfigSectionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get providers => 'Providers';
	@override String get workers => 'Workers';
	@override String get rules => 'Reglas de captura';
	@override String get profiles => 'Perfiles de inyección';
	@override String get costs => 'Coste en tokens';
}

// Path: web.memoryConfig.sectionHints
class _TranslationsWebMemoryConfigSectionHintsEs extends TranslationsWebMemoryConfigSectionHintsEn {
	_TranslationsWebMemoryConfigSectionHintsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get providers => 'Endpoints HTTP registrados (Ollama / LM Studio / Anthropic / OpenAI / Integration) a los que cualquier tarea puede despachar.';
	@override String get workers => 'Para cada punto de contacto elige un provider HTTP (barato, local) o un Agent headless de Claude / Gemini (mayor calidad, consume tokens de CLI).';
	@override String get rules => 'Cuándo se activa el motor de captura en cada session (tras N mensajes / al estar inactivo / K caracteres / manual). Las reglas sin un provider fijado siguen el ajuste del worker de Captura de arriba.';
	@override String get profiles => 'Cómo se inyectan las memorias previas en el prompt del sistema del agente al iniciar la session (recencia, relevancia, híbrido o desactivado).';
	@override String get costs => 'Gasto agregado reconstruido a partir de memory_summarizer_calls. Los providers locales (Ollama, LM Studio, Integration) son gratuitos; los providers en la nube muestran el coste real.';
}

// Path: web.memoryConfig.moveBanner
class _TranslationsWebMemoryConfigMoveBannerEs extends TranslationsWebMemoryConfigMoveBannerEn {
	_TranslationsWebMemoryConfigMoveBannerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'La configuración de memoria se ha movido';
	@override String get body => 'Todos los ajustes relacionados con la memoria (providers / reglas de captura / perfiles de inyección / coste) ahora conviven con Workers en una sola página para que los ajustes relacionados estén juntos.';
	@override String get openButton => 'Abrir Configuración de memoria →';
}

// Path: web.memoryConfig.infra
class _TranslationsWebMemoryConfigInfraEs extends TranslationsWebMemoryConfigInfraEn {
	_TranslationsWebMemoryConfigInfraEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Almacenamiento y embedder (infraestructura)';
	@override String get hint => 'La otra mitad de la config de memoria — backend de embeddings, ajuste de recuperación, puertas de gatekeeper/cleaner y el flag del grafo de conocimiento — vive en Server Settings y requiere reinicio.';
	@override String get openSettings => 'Server Settings →';
	@override String get embedder => 'embedder';
	@override String get gatekeeper => 'gatekeeper';
	@override String get cleaner => 'cleaner';
	@override String get knowledge => 'grafo de conocimiento';
	@override String get on => 'on';
	@override String get off => 'off';
}

// Path: web.memoryWorkers.tasks
class _TranslationsWebMemoryWorkersTasksEs extends TranslationsWebMemoryWorkersTasksEn {
	_TranslationsWebMemoryWorkersTasksEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebMemoryWorkersTasksGatekeeperEs gatekeeper = _TranslationsWebMemoryWorkersTasksGatekeeperEs._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksCleanerEs cleaner = _TranslationsWebMemoryWorkersTasksCleanerEs._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksGitactivityEs gitactivity = _TranslationsWebMemoryWorkersTasksGitactivityEs._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksTranscriptEs transcript = _TranslationsWebMemoryWorkersTasksTranscriptEs._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksPlanDriftEs plan_drift = _TranslationsWebMemoryWorkersTasksPlanDriftEs._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksConflictDetectorEs conflict_detector = _TranslationsWebMemoryWorkersTasksConflictDetectorEs._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksCaptureEs capture = _TranslationsWebMemoryWorkersTasksCaptureEs._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksBlueprintEs blueprint = _TranslationsWebMemoryWorkersTasksBlueprintEs._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksCurationEs curation = _TranslationsWebMemoryWorkersTasksCurationEs._(_root);
}

// Path: web.project.picker
class _TranslationsWebProjectPickerEs extends TranslationsWebProjectPickerEn {
	_TranslationsWebProjectPickerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Elige un proyecto';
	@override String get subtitle => 'La memoria del proyecto se delimita por el directorio de trabajo. Elige uno para gestionar su objetivo, plan, diario y cola de limpieza.';
	@override String get pathPlaceholder => '/path/to/your/project';
	@override String get browse => 'Examinar';
	@override String get browseTooltip => 'Examina el sistema de archivos del host del gateway';
	@override String get open => 'Abrir';
	@override String get recentLabel => 'Proyectos recientes (desde la memoria almacenada):';
	@override String get orphanTooltip => 'Parece un scope_key truncado (antiguo error de importación del mirror). Puede que no tenga documentos del proyecto.';
	@override String get orphanBadge => 'huérfano';
}

// Path: web.project.header
class _TranslationsWebProjectHeaderEs extends TranslationsWebProjectHeaderEn {
	_TranslationsWebProjectHeaderEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String docsCount_one({required Object count}) => '${count} documento';
	@override String docsCount_other({required Object count}) => '${count} documentos';
	@override String journalEntries_one({required Object count}) => '${count} entrada del diario';
	@override String journalEntries_other({required Object count}) => '${count} entradas del diario';
	@override String pendingProposals_one({required Object count}) => '${count} propuesta pendiente';
	@override String pendingProposals_other({required Object count}) => '${count} propuestas pendientes';
	@override String archivedCount({required Object count}) => '${count} archivadas';
}

// Path: web.project.tabs
class _TranslationsWebProjectTabsEs extends TranslationsWebProjectTabsEn {
	_TranslationsWebProjectTabsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get health => 'Estado';
	@override String get goal => 'Objetivo';
	@override String get plan => 'Plan';
	@override String get tech => 'Tecnología';
	@override String get activity => 'Actividad';
	@override String get journal => 'Diario';
	@override String get inbox => 'Bandeja de entrada';
	@override String get conflicts => 'Conflictos';
	@override String get archived => 'Archivadas';
	@override String get overview => 'Resumen';
	@override String get hygiene => 'Higiene';
}

// Path: web.project.docLabel
class _TranslationsWebProjectDocLabelEs extends TranslationsWebProjectDocLabelEn {
	_TranslationsWebProjectDocLabelEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get goal => 'Objetivo';
	@override String get plan => 'Plan';
	@override String get tech_stack => 'Stack tecnológico';
	@override String get recent_activity => 'Actividad reciente';
}

// Path: web.project.editor
class _TranslationsWebProjectEditorEs extends TranslationsWebProjectEditorEn {
	_TranslationsWebProjectEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get updatedBy => 'Actualizado por';
	@override String noDocSet({required Object label}) => 'Aún no se ha definido ningún ${label}.';
	@override String get save => 'Guardar';
	@override String get saveFailedToast => 'Error al guardar';
	@override String savedToast({required Object label}) => '${label} guardado';
	@override String get goalPlaceholder => '¿Qué estamos construyendo? Un párrafo. Lo lee cada agente al iniciarse.';
	@override String get planPlaceholder => 'Plan activo: qué estamos haciendo ahora mismo y qué viene después. Se actualiza a medida que avanza el trabajo.';
	@override String get sectionPlaceholder => 'Escribe esta sección en markdown…';
}

// Path: web.project.readonly
class _TranslationsWebProjectReadonlyEs extends TranslationsWebProjectReadonlyEn {
	_TranslationsWebProjectReadonlyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebProjectReadonlyTechStackEs tech_stack = _TranslationsWebProjectReadonlyTechStackEs._(_root);
	@override late final _TranslationsWebProjectReadonlyRecentActivityEs recent_activity = _TranslationsWebProjectReadonlyRecentActivityEs._(_root);
	@override String noneCaptured({required Object label}) => 'Aún no se ha capturado ningún ${label}.';
	@override String get generatedBy => 'Generado por';
	@override String get lastRefresh => 'última actualización';
	@override String get customEmpty => 'Sección gestionada por el escáner; se rellena cuando éste corre.';
}

// Path: web.project.journal
class _TranslationsWebProjectJournalEs extends TranslationsWebProjectJournalEn {
	_TranslationsWebProjectJournalEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando…';
	@override String get empty => 'Aún no hay entradas en el diario. Cada fin de session añade una automáticamente.';
}

// Path: web.project.inbox
class _TranslationsWebProjectInboxEs extends TranslationsWebProjectInboxEn {
	_TranslationsWebProjectInboxEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando…';
	@override String get emptyTitle => 'Bandeja de entrada vacía.';
	@override String get emptyHint => 'Los agentes presentan propuestas aquí mediante las herramientas MCP `project_goal_set` / `project_plan_set`.';
	@override String approvedToast({required Object label}) => '${label} actualizado';
	@override String get approveFailedToast => 'Error al aprobar';
	@override String get rejectedToast => 'Rechazado';
	@override String get rejectFailedToast => 'Error al rechazar';
	@override String get sessionPrefix => 'ses';
	@override String warning({required Object label}) => 'Aprobar REEMPLAZARÁ por completo el ${label} actual.';
	@override String get warningSuffix => 'Revisa el diff de abajo; esto no es una fusión.';
	@override String get current => 'Actual';
	@override String get proposed => 'Propuesto';
	@override String get emptyBody => '(vacío)';
	@override String get approve => 'Aprobar';
	@override String get reject => 'Rechazar';
	@override String confirmDialogTitle({required Object label}) => '¿Reemplazar ${label}?';
	@override String confirmDialogDescription({required Object label}) => 'El ${label} actual se sobrescribirá con el contenido propuesto. Esto no se puede deshacer desde esta interfaz (puedes volver a editarlo manualmente).';
	@override String get confirmCancel => 'Cancelar';
	@override String get confirmReplace => 'Confirmar reemplazo';
}

// Path: web.project.archived
class _TranslationsWebProjectArchivedEs extends TranslationsWebProjectArchivedEn {
	_TranslationsWebProjectArchivedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get hint => 'Memorias que el limpiador automático archivó para este proyecto. Se excluyen de la recuperación pero son restaurables hasta que la ventana de gracia de 30 días las purgue.';
	@override String get empty => 'Nada archivado para este proyecto. El limpiador archiva aquí los hechos obsoletos y duplicados automáticamente; todavía ninguno.';
	@override String get archivedAtPrefix => 'Archivado';
	@override String get restoreButton => 'Restaurar';
	@override String get restoredToast => 'Restaurado';
	@override String get restoreFailedToast => 'Error al restaurar';
}

// Path: web.project.reset
class _TranslationsWebProjectResetEs extends TranslationsWebProjectResetEn {
	_TranslationsWebProjectResetEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get button => 'Restablecer';
	@override String get dialogTitle => '¿Restablecer la memoria del proyecto?';
	@override String get dialogDescription => 'Elimina todo el contexto de proyecto almacenado para este cwd. Esto no se puede deshacer.';
	@override String get alwaysDeleted => 'Siempre se elimina: objetivo, plan, propuestas, diario, decisiones de limpieza.';
	@override String get alsoDeleteScannerLabel => 'Eliminar también los documentos del escáner';
	@override String get alsoDeleteScannerSuffix => '(tech_stack + recent_activity).';
	@override String get alsoDeleteScannerHint => 'De todos modos se reconstruyen automáticamente en el siguiente inicio; dejarlo sin marcar suele estar bien.';
	@override String get alsoDeleteMemoriesLabel => 'Eliminar también las memorias de pgvector';
	@override String get alsoDeleteMemoriesSuffix => 'para este scope_key.';
	@override String get alsoDeleteMemoriesHint => 'Hechos a largo plazo que el agente almacenó (preferencias del usuario, datos del proyecto). No se pueden recuperar.';
	@override String get cancel => 'Cancelar';
	@override String get deleteForever => 'Eliminar para siempre';
	@override String successToast({required Object summary}) => 'Restablecido: se eliminó ${summary}';
	@override late final _TranslationsWebProjectResetSummaryEs summary = _TranslationsWebProjectResetSummaryEs._(_root);
	@override String get failedToast => 'Error al restablecer';
}

// Path: web.project.lifecycle
class _TranslationsWebProjectLifecycleEs extends TranslationsWebProjectLifecycleEn {
	_TranslationsWebProjectLifecycleEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebProjectLifecycleStatusEs status = _TranslationsWebProjectLifecycleStatusEs._(_root);
	@override String get activate => 'Activar';
	@override String get pause => 'Pausar';
	@override String get archive => 'Archivar';
	@override String get idleSuggest => 'Inactivo — considera archivar';
	@override String idleHint({required Object days}) => 'Sin actividad durante ${days} días';
	@override String get failedToast => 'No se pudo cambiar el estado del proyecto';
	@override late final _TranslationsWebProjectLifecycleAppliedEs applied = _TranslationsWebProjectLifecycleAppliedEs._(_root);
	@override late final _TranslationsWebProjectLifecycleTooltipEs tooltip = _TranslationsWebProjectLifecycleTooltipEs._(_root);
}

// Path: web.project.docMeta
class _TranslationsWebProjectDocMetaEs extends TranslationsWebProjectDocMetaEn {
	_TranslationsWebProjectDocMetaEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebProjectDocMetaMaintainerEs maintainer = _TranslationsWebProjectDocMetaMaintainerEs._(_root);
	@override late final _TranslationsWebProjectDocMetaPurposeEs purpose = _TranslationsWebProjectDocMetaPurposeEs._(_root);
}

// Path: web.project.proposalBanner
class _TranslationsWebProjectProposalBannerEs extends TranslationsWebProjectProposalBannerEn {
	_TranslationsWebProjectProposalBannerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get text => 'La IA ha propuesto una actualización de este documento, a la espera de tu aprobación.';
	@override String get button => 'Revisar en la Bandeja';
}

// Path: web.project.overview
class _TranslationsWebProjectOverviewEs extends TranslationsWebProjectOverviewEn {
	_TranslationsWebProjectOverviewEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get aiManaged => 'Mantenido por IA (se actualiza desde el proyecto)';
	@override String get locked => 'Bloqueado — lo editaste; las actualizaciones de IA llegan como propuestas';
	@override String get edit => 'Editar';
	@override String get save => 'Guardar (bloquea)';
	@override String get cancel => 'Cancelar';
	@override String get unlock => 'Desbloquear (devolver a la IA)';
	@override String get regenerate => 'Regenerar';
	@override String get generate => 'Generar ahora';
	@override String get regenerateHint => 'Pide a la IA que redacte el resumen con el estado más reciente';
	@override String get editHint => 'Guardar bloquea la página; desbloquear deja que la IA la redacte.';
	@override String get empty => 'Aún no hay resumen. El motor en segundo plano lo redacta desde el objetivo/plan, el escaneo de stack, el registro y la memoria — o genéralo ahora.';
	@override String get saved => 'Resumen guardado';
	@override String get unlocked => 'Desbloqueado — la IA volverá a gestionarlo';
	@override String get regenerating => 'Regenerando el resumen…';
}

// Path: web.memoryInspector.status
class _TranslationsWebMemoryInspectorStatusEs extends TranslationsWebMemoryInspectorStatusEn {
	_TranslationsWebMemoryInspectorStatusEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Embedder activo';
	@override String get unavailable => 'no disponible';
	@override String get probing => 'sondeando…';
	@override String dimensions({required Object dim, required Object state}) => '${dim}-dim · ${state}';
	@override String get enabled => 'habilitado';
	@override String get disabled => 'deshabilitado';
	@override String get floorNoModel => 'Solo recuperación por palabras clave (BM25) — no hay modelo de embedding configurado. Añade un endpoint denso [memory.http] en Settings para habilitar la memoria semántica.';
	@override String denseConfiguredPendingRestart({required Object model}) => 'Configurado ${model} (denso) — reinicia el gateway para activar la memoria semántica y re-embeber las memorias existentes.';
	@override String denseUnreachableFloor({required Object model}) => 'Configurado ${model} (denso) pero el endpoint está inalcanzable — se usa el piso de palabras clave hasta que responda (se actualiza al reiniciar).';
	@override String get denseDegraded => 'Embedder denso activo pero su endpoint está inalcanzable ahora — los vectores existentes se conservan; las nuevas escrituras y la búsqueda por similitud se pausan hasta que responda.';
}

// Path: web.memoryInspector.scope
class _TranslationsWebMemoryInspectorScopeEs extends TranslationsWebMemoryInspectorScopeEn {
	_TranslationsWebMemoryInspectorScopeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Scope';
	@override String get scopeKey => 'Clave de scope';
	@override String get scopeKeyIgnored => '(ignorado para global)';
	@override String get scopeKeyCwd => '(cwd del proyecto)';
	@override String get placeholderProject => '/path/to/project (cwd)';
	@override String get syncMd => 'Sincronizar .md';
	@override String get syncTooltip => 'Reimportar los archivos <cwd>/.claude/memory/*.md de Claude a pgvector';
	@override String get browse => 'Explorar';
	@override String get browseTooltip => 'Explora el sistema de archivos del host del gateway para elegir cualquier directorio de proyecto';
	@override late final _TranslationsWebMemoryInspectorScopeValuesEs values = _TranslationsWebMemoryInspectorScopeValuesEs._(_root);
}

// Path: web.memoryInspector.search
class _TranslationsWebMemoryInspectorSearchEs extends TranslationsWebMemoryInspectorSearchEn {
	_TranslationsWebMemoryInspectorSearchEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get placeholder => 'Consulta de búsqueda semántica (Enter para ejecutar; vacío = explorar)';
	@override String get run => 'Buscar';
	@override String get clear => 'Limpiar';
	@override String get failedToast => 'La búsqueda falló';
}

// Path: web.memoryInspector.records
class _TranslationsWebMemoryInspectorRecordsEs extends TranslationsWebMemoryInspectorRecordsEn {
	_TranslationsWebMemoryInspectorRecordsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get noMemories => 'Aún no hay memorias';
	@override String matches_one({required Object count}) => '${count} coincidencia';
	@override String matches_other({required Object count}) => '${count} coincidencias';
	@override String memories_one({required Object count}) => '${count} memoria';
	@override String memories_other({required Object count}) => '${count} memorias';
	@override String get scopeGlobalSuffix => ' (global)';
	@override String scopeInSuffix({required Object scope}) => ' en ${scope}: ';
	@override String get addButton => 'Añadir memoria';
	@override String get addTooltip => 'Crear manualmente una memoria en este scope';
	@override String get deleteAll => 'Eliminar todo';
	@override String get deleteAllTooltip => 'Eliminar todas las memorias de este scope/scope_key';
	@override String get loading => 'Cargando…';
	@override String get enterScopeKeyHint => 'Introduce una clave de scope para explorar las memorias.';
	@override String noMatchesForQuery({required Object query}) => 'No hay coincidencias para "${query}"';
	@override String get noMemoriesInScope => 'Aún no hay memorias en este scope.';
}

// Path: web.memoryInspector.row
class _TranslationsWebMemoryInspectorRowEs extends TranslationsWebMemoryInspectorRowEn {
	_TranslationsWebMemoryInspectorRowEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String simBadge({required Object value}) => 'sim ${value}';
	@override String rankBadge({required Object value}) => 'rango ${value}';
	@override String rankTooltip({required Object effective, required Object similarity, required Object age, required Object days, required Object hits, required Object confidence}) => 'efectivo ${effective} = sim ${similarity} × antigüedad ${age} (${days}d) × hits ${hits} × conf ${confidence}';
	@override String hits_one({required Object count}) => '${count} hit';
	@override String hits_other({required Object count}) => '${count} hits';
	@override String lastHitTooltip({required Object relative}) => 'Último hit ${relative}';
	@override String get editPlaceholder => 'Texto de la memoria. Cmd/Ctrl+Enter para guardar · Esc para cancelar';
	@override String get saveTooltip => 'Guardar (Cmd/Ctrl+Enter)';
	@override String get cancelTooltip => 'Cancelar (Esc)';
	@override String get editTooltip => 'Editar esta memoria';
	@override String get deleteTooltip => 'Eliminar esta memoria';
	@override String get emptyError => 'El texto de la memoria no puede estar vacío';
	@override String deleteConfirm({required Object id}) => '¿Eliminar la memoria ${id}? Esto es permanente.';
	@override String get archiveTooltip => 'Archivar (reversible) — va a la vista Archivado';
	@override String get quarantineTooltip => 'Cuarentena — va a la cola de revisión hasta promoverla o que expire';
}

// Path: web.memoryInspector.toasts
class _TranslationsWebMemoryInspectorToastsEs extends TranslationsWebMemoryInspectorToastsEn {
	_TranslationsWebMemoryInspectorToastsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get deleted => 'Memoria eliminada';
	@override String get deleteFailed => 'La eliminación falló';
	@override String bulkDeleted_one({required Object count}) => 'Se eliminó ${count} memoria de este scope';
	@override String bulkDeleted_other({required Object count}) => 'Se eliminaron ${count} memorias de este scope';
	@override String get bulkDeleteFailed => 'La eliminación masiva falló';
	@override String get created => 'Memoria creada';
	@override String get createFailed => 'La creación falló';
	@override String get updated => 'Memoria actualizada';
	@override String get updateFailed => 'La actualización falló';
	@override String migrated({required Object reembed, required Object examined, required Object to}) => 'Se migraron ${reembed}/${examined} memorias a ${to}';
	@override String get migrationFailed => 'La migración falló';
	@override String syncIngested_one({required Object count}) => 'Se importó ${count} nuevo archivo de memoria';
	@override String syncIngested_other({required Object count}) => 'Se importaron ${count} nuevos archivos de memoria';
	@override String get syncEmpty => 'No hay nuevos archivos .md que sincronizar';
	@override String get syncEmptyDescription => 'Ya está sincronizado, o no hay directorio de memoria de Claude para este cwd.';
	@override String get syncFailed => 'La sincronización falló';
	@override String get archived => 'Memoria archivada — restaurable desde la vista Archivado';
	@override String get archiveFailed => 'Fallo al archivar';
	@override String get quarantined => 'Memoria en cuarentena — revísala en Cortex → Cuarentena';
	@override String get quarantineFailed => 'Fallo al poner en cuarentena';
}

// Path: web.memoryInspector.bulkDelete
class _TranslationsWebMemoryInspectorBulkDeleteEs extends TranslationsWebMemoryInspectorBulkDeleteEn {
	_TranslationsWebMemoryInspectorBulkDeleteEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => '¿Eliminar todas las memorias de este scope?';
	@override String get description => 'Esto es una única operación SQL: todas las memorias del scope especificado se eliminan de forma atómica. Las memorias que se importaron a través del mirror de Claude reaparecen en la siguiente ejecución de <1>Sincronizar .md</1>; todo lo demás se pierde para siempre.';
	@override String get scope => 'Scope';
	@override String get scopeKey => 'Clave de scope';
	@override String get currentlyVisible => 'Visibles actualmente';
	@override String items_one({required Object count}) => '${count} elemento de memoria';
	@override String items_other({required Object count}) => '${count} elementos de memoria';
	@override String get cancel => 'Cancelar';
	@override String get deleteAll => 'Eliminar todo';
}

// Path: web.memoryInspector.addMem
class _TranslationsWebMemoryInspectorAddMemEs extends TranslationsWebMemoryInspectorAddMemEn {
	_TranslationsWebMemoryInspectorAddMemEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Añadir memoria';
	@override String get description => 'Crea manualmente una memoria. Los agentes las crean automáticamente mediante la herramienta MCP <1>memory_store</1>. Este formulario es para los casos en que el operador quiere insertar un hecho sin pasar por un agente.';
	@override String get textLabel => 'Texto';
	@override String get textPlaceholder => 'Prosa simple. El embedder lo convierte en un vector en el momento de almacenarlo; los agentes lo recuperarán mediante memory_search.';
	@override String get cancel => 'Cancelar';
	@override String get create => 'Crear';
}

// Path: web.memoryInspector.picker
class _TranslationsWebMemoryInspectorPickerEs extends TranslationsWebMemoryInspectorPickerEn {
	_TranslationsWebMemoryInspectorPickerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get button => 'Elegir';
	@override String get buttonTooltip => 'Elige entre las claves de scope guardadas o las sessions activas';
	@override String get loading => 'Cargando…';
	@override String empty({required Object scope}) => 'No hay claves guardadas ni sessions activas para ${scope}.';
	@override String get savedHeader => 'Memorias guardadas';
	@override String get activeHeader => 'Sessions activas';
}

// Path: web.memoryInspector.migrationBanner
class _TranslationsWebMemoryInspectorMigrationBannerEs extends TranslationsWebMemoryInspectorMigrationBannerEn {
	_TranslationsWebMemoryInspectorMigrationBannerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String headline_one({required Object count}) => '${count} memoria no aparecerá en las búsquedas';
	@override String headline_other({required Object count}) => '${count} memorias no aparecerán en las búsquedas';
	@override String subtitle({required Object summary, required Object current}) => '${summary}. El embedder actual es <1>${current}</1>. pgvector particiona su índice de similitud por embedder, así que las entradas más antiguas permanecen silenciosas hasta que se vuelven a embeber.';
	@override String summaryItem({required Object count, required Object name}) => '${count} en ${name}';
	@override String get migrateButton => 'Migrar';
}

// Path: web.memoryInspector.reembed
class _TranslationsWebMemoryInspectorReembedEs extends TranslationsWebMemoryInspectorReembedEn {
	_TranslationsWebMemoryInspectorReembedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Volver a embeber memorias';
	@override String get description => 'Recalcula los vectores de las memorias almacenadas con un embedder diferente para que vuelvan a ser buscables.';
	@override String get targetEmbedder => 'Embedder de destino';
	@override String get onName => 'en';
	@override String get totalToReembed => 'Total a volver a embeber';
	@override String get explainer => 'El texto de cada memoria se vuelve a enviar al embedder actual; el nuevo vector reemplaza al antiguo en su lugar. Se conservan el ID, el scope, el scope_key, los metadatos y las marcas de tiempo. Los resultados de búsqueda surten efecto de inmediato, sin necesidad de reiniciar.';
	@override String get reportExamined => 'Examinadas';
	@override String get reportReembedded => 'Vueltas a embeber';
	@override String get reportFailed => 'Fallidas';
	@override String get reportFrom => 'Desde';
	@override String errors_one({required Object count}) => '${count} error';
	@override String errors_other({required Object count}) => '${count} errores';
	@override String get done => 'Hecho';
	@override String get cancel => 'Cancelar';
	@override String get reembedding => 'Volviendo a embeber…';
	@override String reembedTotal({required Object total}) => 'Volver a embeber ${total}';
}

// Path: web.notes.header
class _TranslationsWebNotesHeaderEs extends TranslationsWebNotesHeaderEn {
	_TranslationsWebNotesHeaderEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get outline => 'Esquema';
	@override String get showOutline => 'Mostrar esquema';
	@override String get hideOutline => 'Ocultar esquema';
	@override String get today => 'Hoy';
	@override String get todayTooltip => 'Abre o crea la nota diaria de hoy';
	@override String get kNew => 'Nueva';
}

// Path: web.notes.left
class _TranslationsWebNotesLeftEs extends TranslationsWebNotesLeftEn {
	_TranslationsWebNotesLeftEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get tree => 'Árbol';
	@override String get tags => 'Etiquetas';
	@override String get filterNotes => 'Filtrar notas…';
	@override String get filterTags => 'Filtrar etiquetas…';
	@override String get filteredBy => 'filtrado por';
	@override String get clearTagTooltip => 'Limpiar filtro de etiquetas';
	@override String get expandAll => 'Expandir todo';
	@override String get expandAllTooltip => 'Expandir todas las carpetas';
	@override String get collapseAll => 'Contraer todo';
	@override String get collapseAllTooltip => 'Contraer todas las carpetas';
	@override String get loading => 'Cargando…';
	@override String footer({required Object visible, required Object total}) => '${visible} / ${total} notas';
}

// Path: web.notes.tags
class _TranslationsWebNotesTagsEs extends TranslationsWebNotesTagsEn {
	_TranslationsWebNotesTagsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get emptyVault => 'Aún no hay etiquetas en el vault.';
	@override String noMatches({required Object query}) => 'No hay coincidencias para "${query}".';
}

// Path: web.notes.tree
class _TranslationsWebNotesTreeEs extends TranslationsWebNotesTreeEn {
	_TranslationsWebNotesTreeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get empty => 'El vault está vacío.';
}

// Path: web.notes.outline
class _TranslationsWebNotesOutlineEs extends TranslationsWebNotesOutlineEn {
	_TranslationsWebNotesOutlineEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Esquema';
	@override String get empty => 'No hay encabezados en esta nota. Añade líneas <1>## Título</1> para rellenar el esquema.';
}

// Path: web.notes.newNote
class _TranslationsWebNotesNewNoteEs extends TranslationsWebNotesNewNoteEn {
	_TranslationsWebNotesNewNoteEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get prompt => 'Ruta de la nueva nota (relativa al vault, debe terminar en .md)';
	@override String defaultPath({required Object date}) => 'library/notes-${date}.md';
	@override String get errorMustEndMd => 'La ruta debe terminar en .md';
	@override String get createdToast => 'Nota creada';
	@override String get createFailedToast => 'Error al crear';
}

// Path: web.notes.empty
class _TranslationsWebNotesEmptyEs extends TranslationsWebNotesEmptyEn {
	_TranslationsWebNotesEmptyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Ninguna nota seleccionada';
	@override String get hint => 'Elige una nota del árbol de la izquierda, ve directo al registro diario de hoy o crea una nueva. Los docs de proyecto escritos por la IA viven en <1>projects/</1>; tus borradores personales en <3>personal/</3>.';
	@override String get today => 'Nota diaria de hoy';
	@override String get kNew => 'Nueva nota';
}

// Path: web.notes.picker
class _TranslationsWebNotesPickerEs extends TranslationsWebNotesPickerEn {
	_TranslationsWebNotesPickerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get browseAria => 'Explorar carpetas';
	@override String matches_one({required Object count}) => '${count} coincidencia';
	@override String matches_other({required Object count}) => '${count} coincidencias';
	@override String foldersInVault({required Object count}) => '${count} carpetas en el vault';
	@override String noMatch({required Object value}) => 'Ninguna carpeta existente coincide. Guarda igualmente para usar <1>${value}</1> (se crea de forma diferida en la primera escritura).';
}

// Path: web.notes.vaultSync
class _TranslationsWebNotesVaultSyncEs extends TranslationsWebNotesVaultSyncEn {
	_TranslationsWebNotesVaultSyncEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Sincronización del vault';
	@override String get description => 'Haz commit, pull y push del vault de notas como un repositorio git. La autenticación usa las credenciales de git del host de tu gateway (agente SSH / asistente de credenciales).';
	@override String get reading => 'Leyendo el estado del vault…';
	@override late final _TranslationsWebNotesVaultSyncInitEs init = _TranslationsWebNotesVaultSyncInitEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncBranchEs branch = _TranslationsWebNotesVaultSyncBranchEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncActionEs action = _TranslationsWebNotesVaultSyncActionEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncCommitEs commit = _TranslationsWebNotesVaultSyncCommitEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncFileListEs fileList = _TranslationsWebNotesVaultSyncFileListEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncRemoteEs remote = _TranslationsWebNotesVaultSyncRemoteEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncHistoryEs history = _TranslationsWebNotesVaultSyncHistoryEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncConflictEs conflict = _TranslationsWebNotesVaultSyncConflictEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncAuthEs auth = _TranslationsWebNotesVaultSyncAuthEs._(_root);
	@override late final _TranslationsWebNotesVaultSyncAutoSyncEs autoSync = _TranslationsWebNotesVaultSyncAutoSyncEs._(_root);
}

// Path: web.notes.syncBadge
class _TranslationsWebNotesSyncBadgeEs extends TranslationsWebNotesSyncBadgeEn {
	_TranslationsWebNotesSyncBadgeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando…';
	@override String get syncLabel => 'Sincronizar';
	@override String get initLabel => 'Init';
	@override String get initTooltip => 'El vault aún no es un repo git';
	@override String get conflictLabel => 'Conflicto';
	@override String get conflictTooltip => 'El vault tiene conflictos sin resolver: haz clic para recuperar';
	@override String get syncFallback => 'sincronizar';
	@override String tooltip({required Object branch, required Object files, required Object ahead, required Object behind}) => 'rama ${branch} · ${files} cambios · ${ahead} por delante · ${behind} por detrás';
	@override String get tooltipAutoOn => ' · sincronización automática activada';
	@override String tooltipLastError({required Object error}) => ' · último error: ${error}';
	@override String get branchPlaceholder => '—';
}

// Path: web.activity.filters
class _TranslationsWebActivityFiltersEs extends TranslationsWebActivityFiltersEn {
	_TranslationsWebActivityFiltersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get integration => 'Integración';
	@override String get direction => 'Dirección';
	@override String get status => 'Estado';
	@override String get allIntegrations => 'Todas las integraciones';
	@override String get all => 'Todas';
	@override String get inbound => 'Entrante';
	@override String get outbound => 'Saliente';
	@override String get allStatuses => 'Todos los estados';
	@override String get status2 => '2xx correcto';
	@override String get status3 => '3xx redirección';
	@override String get status4 => '4xx error de cliente';
	@override String get status5 => '5xx error de servidor';
}

// Path: web.activity.table
class _TranslationsWebActivityTableEs extends TranslationsWebActivityTableEn {
	_TranslationsWebActivityTableEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get time => 'Hora';
	@override String get integration => 'Integración';
	@override String get directionTitle => 'Dirección';
	@override String get method => 'Método';
	@override String get path => 'Ruta';
	@override String get status => 'Estado';
	@override String get duration => 'Duración';
	@override String get inboundAria => 'entrante';
	@override String get outboundAria => 'saliente';
}

// Path: web.activity.empty
class _TranslationsWebActivityEmptyEs extends TranslationsWebActivityEmptyEn {
	_TranslationsWebActivityEmptyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get filtered => 'Ninguna llamada coincide con estos filtros.';
	@override String get title => 'Aún no se ha registrado ninguna llamada API';
	@override String get description => 'Cuando una app de terceros llama a opendray con su clave de API de integración, cada solicitud se registra aquí.';
	@override String get stepWithIntegrations => 'Usa la clave de API de una integración existente en tu app de terceros';
	@override String get stepRegister => 'Registra una integración en Integraciones → Nueva';
	@override String get stepCallEndpoint => 'Llama a cualquier endpoint, p. ej. <1>POST /api/v1/sessions</1>';
	@override String get stepAppears => 'Las llamadas aparecen aquí en cuestión de segundos';
	@override String get footnote => 'Las llamadas que haces desde esta UI de administración no se registran; solo se registra el tráfico atribuido a integraciones.';
}

// Path: web.activity.events
class _TranslationsWebActivityEventsEs extends TranslationsWebActivityEventsEn {
	_TranslationsWebActivityEventsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando eventos…';
	@override String get empty => 'Aún no hay eventos.';
	@override String get emptyFiltered => 'No hay eventos coincidentes.';
	@override String get loadOlder => 'Cargar eventos anteriores';
	@override String get today => 'Hoy';
	@override String get yesterday => 'Ayer';
}

// Path: web.providers.list
class _TranslationsWebProvidersListEs extends TranslationsWebProvidersListEn {
	_TranslationsWebProvidersListEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Proveedores';
	@override String get loading => 'Cargando…';
	@override String get disabledBadge => 'deshabilitado';
	@override String get noneSelected => 'Ningún proveedor seleccionado.';
}

// Path: web.providers.detail
class _TranslationsWebProvidersDetailEs extends TranslationsWebProvidersDetailEn {
	_TranslationsWebProvidersDetailEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get enabled => 'Habilitado';
	@override String get disabled => 'Deshabilitado';
	@override String toggleAria({required Object name}) => 'Alternar ${name}';
	@override String get configuration => 'Configuración';
	@override String get noConfig => 'Este proveedor no tiene campos configurables por el usuario.';
	@override String get executable => 'executable:';
	@override String get manifestHash => 'manifest_hash:';
	@override String get reset => 'Restablecer';
	@override String get save => 'Guardar cambios';
	@override String get saving => 'Guardando…';
	@override String get savedToast => 'Configuración del proveedor guardada';
	@override String get saveFailedToast => 'Error al guardar';
	@override String get toggleFailedToast => 'Error al alternar';
	@override late final _TranslationsWebProvidersDetailCapsEs caps = _TranslationsWebProvidersDetailCapsEs._(_root);
	@override String get notInstalled => 'no instalado';
	@override String updateAvailable({required Object version}) => 'actualización disponible → ${version}';
	@override String get upToDate => 'actualizado';
	@override String update({required Object version}) => 'Actualizar a ${version}';
	@override String get updating => 'Actualizando…';
	@override String updatedToast({required Object from, required Object to}) => 'Actualizado ${from} → ${to}';
	@override String get alreadyLatestToast => 'Ya está actualizado';
	@override String get updateFailedToast => 'Error al actualizar';
	@override String get updateUnavailable => 'La actualización dentro de la app no está disponible aquí';
}

// Path: web.providers.configForm
class _TranslationsWebProvidersConfigFormEs extends TranslationsWebProvidersConfigFormEn {
	_TranslationsWebProvidersConfigFormEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get selectPlaceholder => 'Selecciona…';
	@override String get defaultOption => '(predeterminado)';
	@override String get switchOn => 'Activado';
	@override String get switchOff => 'Desactivado';
	@override String get showSecret => 'Mostrar secreto';
	@override String get hideSecret => 'Ocultar secreto';
}

// Path: web.providers.claudeAccounts
class _TranslationsWebProvidersClaudeAccountsEs extends TranslationsWebProvidersClaudeAccountsEn {
	_TranslationsWebProvidersClaudeAccountsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cuentas de Claude';
	@override String get importLocal => 'Importar local';
	@override String get importLocalTooltip => 'Escanea ~/.claude-accounts/ en el host del gateway y registra cualquier directorio nuevo. El botón solo funciona en el host del gateway.';
	@override String get importedNothingToast => 'Nada que importar, las cuentas ya están sincronizadas.';
	@override String importedToast_one({required Object count}) => 'Se importó ${count} cuenta desde ~/.claude-accounts';
	@override String importedToast_other({required Object count}) => 'Se importaron ${count} cuentas desde ~/.claude-accounts';
	@override String get importFailedToast => 'Error al importar';
	@override String get addingTitle => 'Añadiendo una cuenta nueva.';
	@override String get addingBodyPrefix => 'Ejecuta en el host del gateway:';
	@override String get addingBodySuffix => 'el monitor del sistema de archivos de opendray registrará el directorio nuevo automáticamente, o haz clic en <1>Importar local</1> para escanear de inmediato.';
	@override String get architectureLink => 'Arquitectura y guía completa →';
	@override String get loading => 'Cargando…';
	@override String get empty => 'Aún no hay cuentas de Claude. La forma más sencilla: abre Sessions, inicia una session de Claude y ejecuta <1>claude login</1> en la terminal. Tus credenciales de OAuth se guardan en <3>~/.claude</3> en el gateway y aparecen aquí automáticamente. Los usuarios avanzados que gestionan varias identidades pueden usar el flujo de shell anterior en su lugar.';
	@override String get noTokenYet => 'aún no hay token';
	@override String get configDir => 'config_dir:';
	@override String get tokenPath => 'token_path:';
	@override String get toggleFailedToast => 'Error al alternar';
	@override String removeConfirm({required Object name}) => '¿Quitar la cuenta "${name}"?';
	@override String get removedToast => 'Cuenta eliminada';
	@override String get removeFailedToast => 'Error al eliminar';
	@override String toggleAria({required Object name}) => 'Alternar ${name}';
	@override String removeAria({required Object name}) => 'Quitar ${name}';
	@override String get identityAcceptedToast => 'Nueva identidad registrada';
	@override String get identityAcceptFailedToast => 'No se pudo aceptar la identidad';
}

// Path: web.providers.models
class _TranslationsWebProvidersModelsEs extends TranslationsWebProvidersModelsEn {
	_TranslationsWebProvidersModelsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Modelos';
	@override String get help => 'Modelos ofrecidos para este proveedor. El predeterminado se pasa a cada session mediante el flag de modelo; las sessions aún pueden sobrescribirlo.';
	@override String get empty => 'Aún no hay modelos configurados.';
	@override String get add => 'Añadir';
	@override String get addPlaceholder => 'id del modelo (p. ej. sonnet)';
	@override String suggested({required Object count}) => 'Sugeridos (${count})';
	@override String get kDefault => 'predeterminado';
	@override String get makeDefault => 'establecer como predeterminado';
	@override String get setDefault => 'Usar como modelo predeterminado';
	@override String remove({required Object model}) => 'Quitar ${model}';
}

// Path: web.channels.empty
class _TranslationsWebChannelsEmptyEs extends TranslationsWebChannelsEmptyEn {
	_TranslationsWebChannelsEmptyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Aún no hay canales';
	@override String get description => 'Tipos incluidos: Telegram · Slack · Discord · Feishu · DingTalk · WeCom. Elige uno y pega las credenciales, o usa <1>bridge</1> para una plataforma personalizada vía WebSocket.';
}

// Path: web.channels.card
class _TranslationsWebChannelsCardEs extends TranslationsWebChannelsCardEn {
	_TranslationsWebChannelsCardEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get running => 'en ejecución';
	@override String get starting => 'iniciando…';
	@override String get disabled => 'desactivado';
	@override String get muted => 'silenciado';
	@override String get tokenLabel => 'token:';
	@override String get chatIdLabel => 'chat_id:';
	@override String get channelIdLabel => 'channel_id:';
	@override String get webhookLabel => 'webhook:';
	@override String get copyWebhookTooltip => 'Copiar la URL del webhook';
	@override String get webhookCopiedToast => 'URL del webhook copiada';
	@override String get setup => 'Configuración';
	@override String get setupTooltip => 'Mostrar los detalles de conexión del adaptador y código de ejemplo';
	@override String get test => 'Probar';
	@override String get testNotRunningTooltip => 'El canal debe estar en ejecución';
	@override String get testBridgeTooltip => 'Los canales bridge no se pueden probar desde el panel de administración, conecta primero un adaptador';
	@override String get editAria => 'Editar canal';
	@override String get editTooltip => 'Editar la configuración del canal';
	@override String get deleteAria => 'Eliminar canal';
	@override String get muteAria => 'Silenciar o reactivar el canal';
	@override String get muteTooltip => 'Silenciar notificaciones (el chat bidireccional sigue funcionando)';
	@override String get unmuteTooltip => 'Reactivar notificaciones';
	@override String get bridgeSuffix => '(bridge)';
}

// Path: web.channels.toasts
class _TranslationsWebChannelsToastsEs extends TranslationsWebChannelsToastsEn {
	_TranslationsWebChannelsToastsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get testSent => 'Mensaje de prueba enviado';
	@override String get testFailed => 'La prueba falló';
	@override String deleteConfirm({required Object id}) => '¿Eliminar el canal ${id}?';
	@override String get deleted => 'Canal eliminado';
	@override String get created => 'Canal creado';
	@override String get updated => 'Canal actualizado';
	@override String get muted => 'Canal silenciado';
	@override String get unmuted => 'Canal reactivado';
}

// Path: web.channels.dialog
class _TranslationsWebChannelsDialogEs extends TranslationsWebChannelsDialogEn {
	_TranslationsWebChannelsDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get editTitle => 'Editar canal';
	@override String get createTitle => 'Registrar canal';
	@override String get descriptionBridge => 'Un adaptador externo (Python/Node/...) se conecta vía WebSocket y presenta este token.';
	@override String get descriptionDefault => 'Configura la integración de mensajería.';
	@override String get kindLabel => 'Tipo';
	@override String get kindImmutable => '(inmutable, elimina y vuelve a crear para cambiar el tipo)';
	@override String get enabledLabel => 'Activado';
	@override String get enabledBridgeHint => ' (acepta conexiones de adaptadores de inmediato)';
	@override String get enabledWebhookHint => ' (empieza a recibir webhooks de inmediato)';
	@override String get enabledDefaultHint => ' (empieza de inmediato)';
	@override String get cancel => 'Cancelar';
	@override String get save => 'Guardar';
	@override String get saving => 'Guardando…';
	@override String get create => 'Crear';
	@override String get creating => 'Creando…';
	@override String unknownKind({required Object kind}) => 'Tipo desconocido: ${kind}';
	@override String get nameRequired => 'el nombre es obligatorio';
	@override String get tokenRequired => 'el token es obligatorio';
	@override String topicIdsNumeric({required Object value}) => 'Los ID de tema deben ser numéricos (se recibió ${value})';
	@override String fieldRequired({required Object label}) => '${label} es obligatorio';
	@override String get cooldownInvalid => 'El cooldown debe ser un número de segundos no negativo';
	@override String get snippetCapInvalid => 'El límite del fragmento debe ser un número no negativo';
}

// Path: web.channels.notifications
class _TranslationsWebChannelsNotificationsEs extends TranslationsWebChannelsNotificationsEn {
	_TranslationsWebChannelsNotificationsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get sectionTitle => 'Notificaciones de session';
	@override String get repeatPolicyLabel => 'Política de repetición';
	@override String get cooldownLabel => 'Duración del cooldown';
	@override String get onceReplyHint => 'Responder con texto que no sea un comando en este chat restablece la supresión, opendray reenvía tu respuesta al stdin de la session y rearma el notificador.';
	@override String get terminalSnippetLabel => 'Fragmento de terminal';
	@override String get embedSnippetLabel => 'Incrustar la pantalla reciente del terminal en las notificaciones de inactividad';
	@override String get snippetExplainer => 'Cuando está activado, la tarjeta de inactividad incluye un fragmento en bloque de código de lo que el usuario vería en el terminal web en vivo, los elementos de la interfaz del TUI de Claude (indicador de estado, aviso de "bypass permissions", líneas separadoras) se filtran automáticamente.';
	@override late final _TranslationsWebChannelsNotificationsModesEs modes = _TranslationsWebChannelsNotificationsModesEs._(_root);
	@override late final _TranslationsWebChannelsNotificationsCooldownsEs cooldowns = _TranslationsWebChannelsNotificationsCooldownsEs._(_root);
	@override late final _TranslationsWebChannelsNotificationsSnippetCapsEs snippetCaps = _TranslationsWebChannelsNotificationsSnippetCapsEs._(_root);
}

// Path: web.channels.bridge
class _TranslationsWebChannelsBridgeEs extends TranslationsWebChannelsBridgeEn {
	_TranslationsWebChannelsBridgeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get nameLabel => 'Nombre del bridge';
	@override String get namePlaceholder => 'wechat / discord-custom / whatsapp...';
	@override String get nameHint => 'Etiqueta legible para el adaptador. Se muestra en la lista de canales.';
	@override String get tokenLabel => 'Token del adaptador';
	@override String get regenerateTooltip => 'Regenerar';
	@override String get copyTooltip => 'Copiar';
	@override String get tokenCopiedToast => 'Token copiado';
	@override String get tokenHint => 'El adaptador se autentica enviándolo en el frame de registro de WS (o como cabecera <1>X-Bridge-Token</1>).';
	@override String get capsLabel => 'Aceptar capacidades (lista blanca opcional)';
	@override String get capsHint => 'Vacío = aceptar lo que declare el adaptador. Seleccionado = permitir solo estas capacidades aunque el adaptador ofrezca más.';
	@override String get afterCreate => 'Tras <1>Crear</1>, el diálogo de configuración del adaptador se abre automáticamente con la URL del WebSocket y código de inicio en Python / Node / wscat listo para copiar y pegar.';
}

// Path: web.channels.setup
class _TranslationsWebChannelsSetupEs extends TranslationsWebChannelsSetupEn {
	_TranslationsWebChannelsSetupEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object name}) => 'Configuración del adaptador, ${name}';
	@override String get description => 'Ejecuta un adaptador (en cualquier lenguaje) que se conecte a opendray vía WebSocket usando estas credenciales. opendray enrutará las notificaciones de session y las acciones de los comandos a través de él.';
	@override String get wsUrlLabel => 'URL del WebSocket';
	@override String get tokenLabel => 'Token del adaptador';
	@override String authInfo({required Object frame}) => '<1>Auth:</1> envía el token como cabecera <3>X-Bridge-Token</3>, parámetro de consulta <5>?token=</5> o <7>Authorization: Bearer …</7>. El primer frame de WS debe ser <9>${frame}</9>. Especificación completa: <11>docs/bridge-protocol.md</11> en el repo.';
	@override String get pythonInstall => 'Instalar: <1>pip install websockets</1>. Ejecutar: <3>python adapter.py</3>.';
	@override String get nodeInstall => 'Instalar: <1>npm i ws</1>. Ejecutar: <3>node adapter.mjs</3>.';
	@override String get wscatInstall => 'Instalar: <1>npm i -g wscat</1>. Una vez conectado, pega la línea JSON mostrada arriba para registrarte, luego envía más frames manualmente.';
	@override String get close => 'Cerrar';
	@override String get copyHide => 'Ocultar';
	@override String get copyShow => 'Mostrar';
	@override String copyLabelToast({required Object label}) => '${label} copiado';
	@override String get copyCode => 'Copiar';
	@override String get copied => 'Copiado';
	@override String get codeCopiedToast => 'Código copiado';
}

// Path: web.integrations.tabs
class _TranslationsWebIntegrationsTabsEs extends TranslationsWebIntegrationsTabsEn {
	_TranslationsWebIntegrationsTabsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get registered => 'Registradas';
	@override String get console => 'Reverse proxy';
}

// Path: web.integrations.empty
class _TranslationsWebIntegrationsEmptyEs extends TranslationsWebIntegrationsEmptyEn {
	_TranslationsWebIntegrationsEmptyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Aún no hay integraciones';
	@override String get description => 'Registra una aplicación externa para darle una API key con alcance limitado. Su código se queda fuera de este repositorio.';
	@override String get register => 'Registrar integración';
}

// Path: web.integrations.card
class _TranslationsWebIntegrationsCardEs extends TranslationsWebIntegrationsCardEn {
	_TranslationsWebIntegrationsCardEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get managedBadge => 'gestionada';
	@override String get managedTooltip => 'opendray gestiona esta integración. Editar o rotar su key dejaría huérfanas las sessions en ejecución cuyo mcp.json contiene el bearer anterior.';
	@override String get consumerBadge => 'consumidora';
	@override String get consumerTooltip => 'Integración solo consumidora. No hay servicio HTTP que sondear';
	@override String get disabledBadge => 'deshabilitada';
	@override String get consumerOnlyHint => 'Consume la API de opendray. No tiene reverse proxy montado.';
	@override String lastProbed({required Object relative}) => 'último sondeo ${relative}';
	@override String rotated({required Object relative}) => 'rotada ${relative}';
	@override String get managedReadOnly => 'solo lectura. opendray incrusta su key en el mcp.json de cada spawn';
	@override String get managedReadOnlyTooltip => 'opendray gestiona esta fila. Para restablecerla: borra ~/.opendray/memory.key y reinicia, o borra esta fila directamente mediante SQL (se volverá a inicializar en el siguiente arranque).';
	@override String get editAria => 'Editar integración';
	@override String get editTooltip => 'Editar scopes / URL base / versión';
	@override String get rotateKey => 'Rotar key';
	@override String get deleteAria => 'Eliminar integración';
	@override String rotateConfirm({required Object name}) => '¿Rotar la API key de "${name}"? La key actual dejará de funcionar de inmediato.';
	@override String deleteConfirm({required Object name}) => '¿Eliminar la integración ${name}?';
	@override String get removedToast => 'Integración eliminada';
}

// Path: web.integrations.register_dialog
class _TranslationsWebIntegrationsRegisterDialogEs extends TranslationsWebIntegrationsRegisterDialogEn {
	_TranslationsWebIntegrationsRegisterDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Registrar integración';
	@override String get description => 'Emite una API key de un solo uso. Cópiala antes de cerrar: opendray nunca vuelve a mostrar el texto en claro.';
	@override String get nameLabel => 'Nombre';
	@override String get namePlaceholder => 'PetTracker';
	@override String get modeHint => 'Deja en blanco los dos campos siguientes para una integración <1>solo consumidora</1> (aplicación de terceros que llama a la API de opendray pero no expone su propio servicio). Rellena ambos para una integración con <3>reverse-proxy</3>.';
	@override String get baseUrlLabel => 'URL base';
	@override String get optionalSuffix => '(opcional)';
	@override String get baseUrlPlaceholder => 'http://192.168.1.10:8080';
	@override String get routePrefixLabel => 'Prefijo de ruta';
	@override String get routePrefixPlaceholder => 'pet-tracker';
	@override String routePrefixHint({required Object prefix}) => 'Accesible en <1>/api/v1/proxy/${prefix}/*</1>.';
	@override String get routePrefixPlaceholderToken => '<prefix>';
	@override String get versionLabel => 'Versión (opcional)';
	@override String get versionPlaceholder => '0.1.0';
	@override String get scopesLabel => 'Scopes';
	@override String get scopesIntro => 'Elige la superficie de API que esta integración tiene permitido llamar. Cada interruptor se corresponde con un claim del Bearer-token: opendray rechaza las peticiones que tocan endpoints fuera del conjunto concedido.';
	@override String get errorNameRequired => 'El nombre es obligatorio.';
	@override String get errorBothOrNeither => 'base_url y route_prefix van juntos. Configura ambos para una integración con reverse-proxy, o deja ambos en blanco para una integración solo consumidora.';
	@override String get cancel => 'Cancelar';
	@override String get submit => 'Registrar';
	@override String get submitting => 'Registrando…';
}

// Path: web.integrations.reveal
class _TranslationsWebIntegrationsRevealEs extends TranslationsWebIntegrationsRevealEn {
	_TranslationsWebIntegrationsRevealEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get titleIssued => 'API key emitida';
	@override String get titleRotated => 'API key rotada';
	@override String get description => 'Esta es la única vez que se mostrará la key en texto en claro. Cópiala ahora y actualiza todas las aplicaciones consumidoras: la key anterior (si la había) ya no autentica.';
	@override String get discardAria => 'Descartar la nueva key';
	@override String get discardTooltip => 'Descartar la nueva key (la rotación ya ha ocurrido, la key antigua también desapareció)';
	@override String get discardConfirm => '¿Descartar la nueva key? La rotación ya ha invalidado la key antigua: descartarla significa que NO tendrás ninguna key funcional para esta integración hasta que vuelvas a rotar.';
	@override String get copy => 'Copiar';
	@override String get copied => 'Copiada';
	@override String get updateHint => '<1>Actualiza todas las aplicaciones consumidoras con esta nueva key.</1> La key anterior se ha invalidado en el servidor y devolverá <3>401 unauthorized</3> en la siguiente petición.';
	@override String get acknowledge => 'He copiado la key y actualizaré mis aplicaciones consumidoras. Entiendo que opendray no la volverá a mostrar.';
	@override String get discard => 'Descartar';
	@override String get done => 'Hecho';
}

// Path: web.integrations.edit_dialog
class _TranslationsWebIntegrationsEditDialogEs extends TranslationsWebIntegrationsEditDialogEn {
	_TranslationsWebIntegrationsEditDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object name}) => 'Editar integración · ${name}';
	@override String get description => 'Cambia los scopes, la versión o la URL base. El nombre y el prefijo de ruta son inmutables: elimina y vuelve a registrar si necesitas cambiarlos.';
	@override String get nameLabel => 'Nombre';
	@override String get routePrefixLabel => 'Prefijo de ruta';
	@override String get consumerOnlyLabel => '(solo consumidora)';
	@override String get baseUrlLabel => 'URL base';
	@override String get baseUrlConsumerSuffix => '(solo consumidora, deja en blanco)';
	@override String get baseUrlProxySuffix => '(destino del reverse-proxy)';
	@override String get baseUrlConsumerPlaceholder => '(en blanco: esta integración consume la API de opendray)';
	@override String get baseUrlProxyPlaceholder => 'http://127.0.0.1:8080';
	@override String get consumerHint => 'Esta es una integración solo consumidora. Cambiar la URL base aquí también requeriría un prefijo de ruta; hazlo eliminando y volviendo a registrar.';
	@override String get versionLabel => 'Versión';
	@override String get versionPlaceholder => '0.1.0';
	@override String get scopesLabel => 'Scopes';
	@override String get scopesIntro => 'Reduce o amplía la superficie de API que autoriza la API key de esta integración. Los tokens activos no se ven afectados: el nuevo conjunto de scopes surte efecto en la siguiente petición.';
	@override String get errorModeSwitch => 'Cambiar entre modo solo consumidora y reverse-proxy requiere eliminar la integración y volver a registrarla: el nombre y route_prefix no pueden cambiarse sobre la marcha.';
	@override String get updatedToast => 'Integración actualizada';
	@override String get cancel => 'Cancelar';
	@override String get save => 'Guardar cambios';
}

// Path: web.integrations.proxy
class _TranslationsWebIntegrationsProxyEs extends TranslationsWebIntegrationsProxyEn {
	_TranslationsWebIntegrationsProxyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get emptyTitle => 'No hay integraciones registradas';
	@override String emptyDescription({required Object prefix}) => 'Registra primero una integración; la consola hace de proxy a través de /api/v1/proxy/${prefix}/* usando el token de administrador.';
	@override String get targetLabel => 'Destino';
	@override String get selectPlaceholder => 'Selecciona una integración…';
	@override String get baseLabel => 'base:';
	@override String get history => 'Historial';
	@override String get historyEmpty => 'no hay peticiones anteriores para esta integración';
	@override String get send => 'Enviar';
	@override String get sending => 'Enviando…';
	@override String get extraHeadersLabel => 'Headers adicionales (uno por línea, Nombre: Valor)';
	@override String get bodyLabel => 'Body';
	@override String get headers => 'Headers';
	@override String get body => 'Body';
	@override String get emptyBody => '(vacío)';
	@override String get requestFailed => 'la petición falló';
	@override String get stubText => 'Envía una petición para ver la respuesta del upstream.';
	@override String get stubInjects => 'opendray inyecta <1>X-Integration-ID</1> y elimina tu header <3>Authorization</3>.';
	@override String get prefixPlaceholder => '<prefix>';
}

// Path: web.plugins.common
class _TranslationsWebPluginsCommonEs extends TranslationsWebPluginsCommonEn {
	_TranslationsWebPluginsCommonEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando…';
	@override String get cancel => 'Cancelar';
	@override String get edit => 'Editar';
	@override String get add => 'Añadir';
	@override String get save => 'Guardar';
	@override String get create => 'Crear';
}

// Path: web.plugins.mcp
class _TranslationsWebPluginsMcpEs extends TranslationsWebPluginsMcpEn {
	_TranslationsWebPluginsMcpEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Servidores MCP';
	@override String description({required Object KEY}) => 'Servidores Model Context Protocol inyectados en cada spawn (claude / codex). Las entradas del vault están en <1>~/.opendray/vault/mcp/&lt;id&gt;/mcp.json</1>; los secretos (referenciados como <3>\$${KEY}</3> en env / headers) provienen de la sección <5>secretos MCP</5> de abajo.';
	@override String get newServer => 'Nuevo servidor';
	@override String get empty => 'Aún no hay servidores MCP. Añade uno para exponer herramientas adicionales a tus sessions de agente.';
	@override late final _TranslationsWebPluginsMcpColumnsEs columns = _TranslationsWebPluginsMcpColumnsEs._(_root);
	@override String get noUrl => 'sin url';
	@override String get noCommand => 'sin comando';
	@override String deleteConfirm({required Object id}) => '¿Eliminar el servidor MCP "${id}"?';
	@override String get removedToast => 'Servidor MCP eliminado';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override String get toggleFailedToast => 'Error al alternar';
	@override String get codexUnsupportedBadge => 'Codex: no compatible';
	@override String get codexUnsupportedTooltip => 'El CLI de codex solo admite el transport stdio. Este servidor se omitirá en las sessions de codex; claude y gemini lo seguirán usando.';
	@override String get builtinBadge => 'Integrado';
	@override String get builtinTooltip => 'Provisto por el propio opendray — se adjunta automáticamente a cada session que admite MCP. No se puede editar ni eliminar.';
	@override String get builtinDescription => 'El servidor compartido de memoria y conocimiento de opendray: memory_search / memory_store, project_goal y project_plan get/set, session_log_append, decision_record, doc_read, skill_distill, project_search. Se adjunta automáticamente a cada session de Claude / Codex / Gemini.';
	@override String get builtinAutoAttach => 'siempre activo';
	@override late final _TranslationsWebPluginsMcpEditorEs editor = _TranslationsWebPluginsMcpEditorEs._(_root);
	@override late final _TranslationsWebPluginsMcpTestEs test = _TranslationsWebPluginsMcpTestEs._(_root);
}

// Path: web.plugins.mcpSecrets
class _TranslationsWebPluginsMcpSecretsEs extends TranslationsWebPluginsMcpSecretsEn {
	_TranslationsWebPluginsMcpSecretsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Secretos MCP';
	@override String get encryptedBadge => 'cifrado';
	@override String get plaintextBadge => 'texto plano';
	@override String get encryptedTooltip => 'Cifrado AES-GCM en disco; la clave se almacena en el llavero del SO';
	@override String get plaintextTooltip => 'Llavero del SO no disponible. El archivo está en texto plano en disco. Revisa el log del gateway.';
	@override String description({required Object KEY}) => 'Los valores referenciados desde los marcadores <1>\$${KEY}</1> en cualquier <3>mcp.json</3> se sustituyen en el momento del spawn. <5>Los valores guardados nunca se devuelven a través de la API</5>, puedes sobrescribirlos o eliminarlos, pero no volver a leerlos.';
	@override String descriptionStored({required Object path}) => ' Almacenado en <1>${path}</1>.';
	@override String get addSecret => 'Añadir secreto';
	@override String empty({required Object KEY}) => 'No hay secretos almacenados. Añade uno para empezar a referenciarlo como <1>\$${KEY}</1> en las configuraciones de tus servidores MCP.';
	@override late final _TranslationsWebPluginsMcpSecretsColumnsEs columns = _TranslationsWebPluginsMcpSecretsColumnsEs._(_root);
	@override String get editTooltip => 'Sobrescribir el valor almacenado';
	@override String deleteConfirm({required Object key}) => '¿Eliminar el secreto "${key}"? Cualquier mcp.json que referencie \$${key} recurrirá al marcador literal hasta que establezcas un nuevo valor.';
	@override String get removedToast => 'Secreto eliminado';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override late final _TranslationsWebPluginsMcpSecretsEditorEs editor = _TranslationsWebPluginsMcpSecretsEditorEs._(_root);
}

// Path: web.plugins.skills
class _TranslationsWebPluginsSkillsEs extends TranslationsWebPluginsSkillsEn {
	_TranslationsWebPluginsSkillsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Habilidades del agente';
	@override String get description => 'Capacidades reutilizables inyectadas en las sessions de Claude como un índice de Tier 1, el agente carga el SKILL.md completo bajo demanda mediante <1>opendray skill describe &lt;id&gt;</1>. Las integradas vienen en el binario pero se pueden <3>personalizar</3>, tus ediciones se guardan en <5>~/.opendray/vault/skills/&lt;id&gt;/SKILL.md</5> y anulan la versión incorporada. Usa Restablecer para revertir.';
	@override String get newSkill => 'Nueva habilidad';
	@override String get empty => 'No se encontraron habilidades.';
	@override late final _TranslationsWebPluginsSkillsColumnsEs columns = _TranslationsWebPluginsSkillsColumnsEs._(_root);
	@override String get noDescription => 'sin descripción';
	@override String get builtinBadge => 'integrada';
	@override String get builtinTooltip => 'Incorporada en el binario de opendray, haz clic en Personalizar para anularla en tu vault';
	@override String get vaultBadge => 'vault';
	@override String get overridesBuiltin => 'anula la integrada';
	@override String get overridesBuiltinTooltip => 'Esta habilidad del vault anula la versión integrada del mismo id';
	@override String get customize => 'Personalizar';
	@override String get customizeTooltip => 'Abre el SKILL.md y guarda los cambios como una anulación del vault';
	@override String get editTooltip => 'Editar esta habilidad del vault';
	@override String get resetTooltip => 'Eliminar la anulación del vault y volver a la versión integrada';
	@override String get reset => 'Restablecer';
	@override String resetConfirm({required Object id}) => '¿Restablecer "${id}" a la versión integrada? Esto elimina tu SKILL.md del vault y vuelve a la copia incorporada.';
	@override String deleteConfirm({required Object id}) => '¿Eliminar la habilidad "${id}" de tu vault? Esto elimina el archivo SKILL.md.';
	@override String get removedToast => 'Habilidad eliminada';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override late final _TranslationsWebPluginsSkillsEditorEs editor = _TranslationsWebPluginsSkillsEditorEs._(_root);
	@override String get dropHint => 'O suelta un SKILL.md aquí para instalarlo.';
	@override String get dropToInstall => 'Suelta el SKILL.md para instalar';
	@override String get uploading => 'Instalando habilidad…';
	@override String uploadedToast({required Object id}) => 'Habilidad "${id}" instalada';
	@override String get uploadFailedToast => 'Error al subir la habilidad';
	@override String get uploadInvalidTypeToast => 'Solo se pueden instalar archivos SKILL.md por arrastre';
}

// Path: web.plugins.customTasks
class _TranslationsWebPluginsCustomTasksEs extends TranslationsWebPluginsCustomTasksEn {
	_TranslationsWebPluginsCustomTasksEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Tareas personalizadas';
	@override String get description => 'Atajos de ejecución con un clic que se muestran en la pestaña Tareas. Deja cwd en blanco para tareas globales visibles en todas las sessions, o fíjalo a una ruta absoluta para acotarlo.';
	@override String get addTask => 'Añadir tarea';
	@override String get empty => 'Aún no hay tareas personalizadas.';
	@override late final _TranslationsWebPluginsCustomTasksColumnsEs columns = _TranslationsWebPluginsCustomTasksColumnsEs._(_root);
	@override String get globalScope => 'global';
	@override String deleteConfirm({required Object name}) => '¿Eliminar la tarea personalizada "${name}"?';
	@override String get removedToast => 'Tarea eliminada';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override late final _TranslationsWebPluginsCustomTasksDialogEs dialog = _TranslationsWebPluginsCustomTasksDialogEs._(_root);
}

// Path: web.plugins.gitHosts
class _TranslationsWebPluginsGitHostsEs extends TranslationsWebPluginsGitHostsEn {
	_TranslationsWebPluginsGitHostsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Hosts de git';
	@override String get description => 'Un token por host, usado por la pestaña Git para obtener los pull requests <1>y por la sincronización del vault de Notas</1> cuando su remoto usa HTTPS hacia un repo privado en el mismo host. Se admiten GitHub.com, GitHub Enterprise autoalojado, Gitea y GitLab.';
	@override String get addHost => 'Añadir host';
	@override String get empty => 'No hay hosts de git configurados.\nAñade uno para habilitar la lista de PR en la pestaña Git del inspector.';
	@override late final _TranslationsWebPluginsGitHostsColumnsEs columns = _TranslationsWebPluginsGitHostsColumnsEs._(_root);
	@override String get statusEnabled => 'habilitado';
	@override String get statusDisabled => 'deshabilitado';
	@override String deleteConfirm({required Object host}) => '¿Eliminar el host de git ${host}? Las consultas de PR contra este host dejarán de funcionar.';
	@override String get removedToast => 'Host de git eliminado';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override late final _TranslationsWebPluginsGitHostsDialogEs dialog = _TranslationsWebPluginsGitHostsDialogEs._(_root);
}

// Path: web.backups.tabs
class _TranslationsWebBackupsTabsEs extends TranslationsWebBackupsTabsEn {
	_TranslationsWebBackupsTabsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get backups => 'Copias de seguridad';
	@override String get schedules => 'Programaciones';
	@override String get targets => 'Destinos';
}

// Path: web.backups.inventory
class _TranslationsWebBackupsInventoryEs extends TranslationsWebBackupsInventoryEn {
	_TranslationsWebBackupsInventoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => '¿Qué hay en una copia de seguridad?';
	@override String summary({required Object rows, required Object tables}) => '${rows} filas en ${tables} tablas';
	@override String get description => 'Cada copia de seguridad es un <1>pg_dump --format=custom</1> de cada tabla de abajo, más <3>manifest.json</3> y (opcionalmente) <5>config.toml</5>. Los recuentos son en vivo; el paquete captura lo que haya en el momento de la copia.';
	@override String get loadFailedToast => 'No se pudo cargar el inventario';
	@override String get rowsLabel => 'filas';
}

// Path: web.backups.restart
class _TranslationsWebBackupsRestartEs extends TranslationsWebBackupsRestartEn {
	_TranslationsWebBackupsRestartEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Reinicia opendray para activar las copias de seguridad';
	@override String get description => 'Tu frase de contraseña está guardada. El gateway solo la carga al iniciarse, así que la función permanece desactivada hasta que reinicies el proceso.';
	@override String get keyFile => 'Archivo de clave:';
	@override String get configuredVia => 'Configurado mediante:';
	@override String get envVar => 'variable de entorno OPENDRAY_BACKUP_KEY';
	@override String get checkAgain => 'Comprobar de nuevo';
}

// Path: web.backups.setup
class _TranslationsWebBackupsSetupEs extends TranslationsWebBackupsSetupEn {
	_TranslationsWebBackupsSetupEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Configurar copias de seguridad';
	@override String get description => 'Elige una frase de contraseña maestra. opendray la usa para cifrar cada blob de copia de seguridad. <1>Si la pierdes, tus copias de seguridad serán irrecuperables</1>, así que guárdala en un gestor de contraseñas (Vaultwarden, 1Password, …) antes de continuar.';
	@override String get generate => 'Generar';
	@override String get pasteOwn => 'Pegar la mía';
	@override String get generateTitle => 'Clave aleatoria de 256 bits';
	@override String get generateHint => 'El servidor genera una frase de contraseña criptográficamente aleatoria y la muestra una sola vez. Debes copiarla antes de continuar, no hay forma de recuperarla.';
	@override String get pasteLabel => 'Tu frase de contraseña';
	@override String get pastePlaceholder => 'Al menos 20 caracteres';
	@override String get pasteHint => 'Recomendado: más de 40 caracteres desde un gestor de contraseñas.';
	@override String get savesTo => 'Se guarda en:';
	@override String get saving => 'Guardando…';
	@override String get generateAndSave => 'Generar y guardar';
	@override String get save => 'Guardar';
}

// Path: web.backups.generated
class _TranslationsWebBackupsGeneratedEs extends TranslationsWebBackupsGeneratedEn {
	_TranslationsWebBackupsGeneratedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Guarda esta frase de contraseña AHORA';
	@override String get description => 'Esto se muestra <1>una sola vez</1>. No se podrá recuperar desde opendray ni desde ningún otro sitio. Cópiala en un gestor de contraseñas antes de continuar.';
	@override String get copy => 'Copiar';
	@override String get copiedToast => 'Frase de contraseña copiada al portapapeles';
	@override String get copyFailedToast => 'Error al copiar, selecciónala y cópiala manualmente';
	@override String get savedTo => 'Guardada en:';
	@override String get ack => 'He guardado esta frase de contraseña en mi gestor de contraseñas';
	@override String get kContinue => 'Continuar';
}

// Path: web.backups.status
class _TranslationsWebBackupsStatusEs extends TranslationsWebBackupsStatusEn {
	_TranslationsWebBackupsStatusEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get pgDump => 'pg_dump';
	@override String get pgRestore => 'pg_restore';
	@override String get pgDumpUnavailable => 'no disponible';
	@override String get pgDumpHint => 'Las copias de seguridad no pueden ejecutarse hasta que pg_dump esté en PATH (o se haya definido su ruta absoluta en <1>backup.pg_dump_path</1>). Instala <3>postgresql-client</3> de la misma versión mayor que tu servidor y reinicia.';
}

// Path: web.backups.backupsTab
class _TranslationsWebBackupsBackupsTabEs extends TranslationsWebBackupsBackupsTabEn {
	_TranslationsWebBackupsBackupsTabEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get backupNow => 'Hacer copia ahora';
	@override String get triggering => 'Lanzando…';
	@override String get includeConfig => 'incluir config.toml';
	@override String get fullInstance => 'Instancia completa';
	@override String get fullInstanceHint => 'Incluye también el vault (notes/skills/mcp), secrets.env y config.toml: todo lo necesario para reconstruir una instancia funcional, no solo su base de datos.';
	@override String get restoreFromFile => 'Restaurar desde archivo';
	@override String get refresh => 'Actualizar';
	@override String get queuedToast => 'Copia de seguridad en cola';
	@override String get triggerFailedToast => 'Error al lanzar';
	@override String get listFailedToast => 'No se pudieron listar las copias de seguridad';
	@override String deleteConfirm({required Object id}) => '¿Eliminar la copia de seguridad ${id}? El blob se elimina de su destino.';
	@override String get deletedToast => 'Copia de seguridad eliminada';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override String get empty => 'Aún no hay copias de seguridad. Haz clic en "Hacer copia ahora" arriba para crear la primera.';
	@override late final _TranslationsWebBackupsBackupsTabColumnsEs columns = _TranslationsWebBackupsBackupsTabColumnsEs._(_root);
	@override String get downloadTooltip => 'Descargar';
	@override String get deleteTooltip => 'Eliminar';
}

// Path: web.backups.restore
class _TranslationsWebBackupsRestoreEs extends TranslationsWebBackupsRestoreEn {
	_TranslationsWebBackupsRestoreEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Restaurar desde un paquete de copia de seguridad';
	@override String get bundleLabel => 'Paquete cifrado (.tar.gz.enc)';
	@override String get targetDsnLabel => 'DSN de la base de datos de destino';
	@override String get targetDsnHint => '(en blanco = la propia base de datos de opendray, PELIGROSO)';
	@override String get targetDsnPlaceholder => 'postgres://user:pass@host:5432/dbname';
	@override String get cleanLabel => '--clean --if-exists (eliminar primero el esquema existente; obligatorio al restaurar sobre una base de datos con datos)';
	@override String get auditNoteLabel => 'Nota de auditoría (opcional)';
	@override String get auditNotePlaceholder => 'Motivo de la restauración, aparece en el slog';
	@override String get ownDbWarning => 'Estás restaurando en <1>la propia base de datos de opendray</1>. Con "--clean" activado, esto elimina todas las tablas y reproduce la copia de seguridad tal cual, es irreversible. Escribe <3>I understand</3> para continuar.';
	@override String get confirmPlaceholder => 'I understand';
	@override String get confirmSentinel => 'I understand';
	@override String get pgRestoreOutput => 'Salida de pg_restore (últimos 8 KiB)';
	@override String get noPgRestoreOutput => '(sin salida de pg_restore)';
	@override String get pickFileToast => 'Elige primero un archivo de paquete';
	@override String get succeededToast => 'Restauración correcta';
	@override String replayedDescription({required Object bytes, required Object id}) => '${bytes} reproducidos desde el manifest ${id}';
	@override String get failedToast => 'Error en la restauración';
	@override String get restoring => 'Restaurando…';
	@override String get dryRunToast => 'Simulación completa: revisa el plan y luego aplícalo';
	@override String get planTitle => 'Plan de restauración (simulación: nada cambió)';
	@override String planDump({required Object size}) => 'Volcado de base de datos: ${size}';
	@override String planConfig({required Object path}) => 'config.toml → ${path}';
	@override String planSecrets({required Object path}) => 'secrets.env → ${path}';
	@override String planVault({required Object files, required Object roots}) => 'vault: ${files} archivos (${roots})';
	@override String get planApplyHint => 'Aplicar toma primero una instantánea de seguridad de instancia completa, luego sobrescribe lo anterior y ejecuta pg_restore.';
	@override String get preview => 'Previsualizar (simulación)';
	@override String get previewing => 'Previsualizando…';
	@override String get previewFirstHint => 'Ejecuta primero una simulación';
	@override String get applyRestore => 'Aplicar restauración';
}

// Path: web.backups.kind
class _TranslationsWebBackupsKindEs extends TranslationsWebBackupsKindEn {
	_TranslationsWebBackupsKindEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get dbOnly => 'Solo BD';
	@override String get fullInstance => 'Instancia completa';
	@override String get fullInstanceHint => 'Incluye el vault, secrets.env y config.toml';
}

// Path: web.backups.verify
class _TranslationsWebBackupsVerifyEs extends TranslationsWebBackupsVerifyEn {
	_TranslationsWebBackupsVerifyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get ok => 'verificada';
	@override String get okHint => 'Descifrada y confirmada como restaurable (pg_restore --list)';
	@override String get failed => 'sin verificar';
}

// Path: web.backups.health
class _TranslationsWebBackupsHealthEs extends TranslationsWebBackupsHealthEn {
	_TranslationsWebBackupsHealthEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get headlineHealthy => 'Copias correctas';
	@override String get headlineAttention => 'Requiere atención';
	@override String get headlineNever => 'Aún sin copias';
	@override String get lastSuccess => 'Última copia correcta';
	@override String get never => 'nunca';
	@override late final _TranslationsWebBackupsHealthTilesEs tiles = _TranslationsWebBackupsHealthTilesEs._(_root);
	@override String get loadFailedToast => 'No se pudo cargar el estado de las copias';
}

// Path: web.backups.trigger
class _TranslationsWebBackupsTriggerEs extends TranslationsWebBackupsTriggerEn {
	_TranslationsWebBackupsTriggerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get preMigrate => 'pre-migración';
	@override String get preMigrateHint => 'Instantánea automática tomada antes de ejecutar las migraciones de esquema';
	@override String get preRestore => 'pre-restauración';
	@override String get preRestoreHint => 'Instantánea de seguridad automática tomada antes de aplicar una restauración';
}

// Path: web.backups.recoveryKit
class _TranslationsWebBackupsRecoveryKitEs extends TranslationsWebBackupsRecoveryKitEn {
	_TranslationsWebBackupsRecoveryKitEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get button => 'Kit de recuperación';
	@override String get title => 'Descargar kit de recuperación';
	@override String get warning => 'La frase de la copia de seguridad nunca se guarda en una copia. Este kit es esa frase envuelta con una frase de recuperación que tú eliges. Guarda TANTO el archivo COMO la frase de recuperación en un lugar seguro y separado: sin ellos, perder el host significa copias irrecuperables.';
	@override String get passphraseLabel => 'Frase de recuperación (mín. 8 caracteres)';
	@override String get passphrasePlaceholder => 'una frase fuerte que no perderás';
	@override String get confirmLabel => 'Confirmar frase de recuperación';
	@override String get mismatch => 'Las frases no coinciden';
	@override String get generating => 'Generando…';
	@override String get download => 'Descargar kit';
	@override String get downloadedToast => 'Kit de recuperación descargado: guárdalo de forma segura';
	@override String get failedToast => 'No se pudo generar el kit de recuperación';
}

// Path: web.backups.schedulesTab
class _TranslationsWebBackupsSchedulesTabEs extends TranslationsWebBackupsSchedulesTabEn {
	_TranslationsWebBackupsSchedulesTabEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get description => 'Copias de seguridad periódicas. El programador consulta cada 30 s y ejecuta la programación pendiente más antigua.';
	@override String get newSchedule => 'Nueva programación';
	@override String get loadFailedToast => 'No se pudieron cargar las programaciones';
	@override String deleteConfirm({required Object id}) => '¿Eliminar la programación ${id}?';
	@override String get deletedToast => 'Programación eliminada';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override String get toggleFailedToast => 'Error al alternar';
	@override String get empty => 'No hay programaciones. Añade una para hacer copias de seguridad periódicas automáticas.';
	@override late final _TranslationsWebBackupsSchedulesTabColumnsEs columns = _TranslationsWebBackupsSchedulesTabColumnsEs._(_root);
	@override String keepCount({required Object count}) => '${count} copias de seguridad';
	@override String get deleteTooltip => 'Eliminar';
}

// Path: web.backups.newSchedule
class _TranslationsWebBackupsNewScheduleEs extends TranslationsWebBackupsNewScheduleEn {
	_TranslationsWebBackupsNewScheduleEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Nueva programación de copia de seguridad';
	@override String get targetLabel => 'Destinos';
	@override String get targetsHint => 'Elige uno o más: la misma copia se escribe en cada destino (3-2-1).';
	@override String get everyHoursLabel => 'Cada (horas)';
	@override String get keepLastNLabel => 'Conservar las últimas N';
	@override String get enableImmediately => 'Habilitar inmediatamente';
	@override String get createdToast => 'Programación creada';
	@override String get createFailedToast => 'Error al crear';
	@override String get creating => 'Creando…';
	@override String get create => 'Crear';
}

// Path: web.backups.fanout
class _TranslationsWebBackupsFanoutEs extends TranslationsWebBackupsFanoutEn {
	_TranslationsWebBackupsFanoutEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get badge => 'difusión';
	@override String hint({required Object group}) => 'Parte de una difusión a varios destinos (grupo ${group})';
}

// Path: web.backups.dedup
class _TranslationsWebBackupsDedupEs extends TranslationsWebBackupsDedupEn {
	_TranslationsWebBackupsDedupEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get badge => 'deduplicada';
	@override String get hint => 'Idéntica a una copia anterior: reutilizó el blob existente en lugar de subir una copia';
}

// Path: web.backups.targetsTab
class _TranslationsWebBackupsTargetsTabEs extends TranslationsWebBackupsTargetsTabEn {
	_TranslationsWebBackupsTargetsTabEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get description => 'Destinos de almacenamiento. v1 admite <1>local</1> (disco en el host de opendray) y <3>smb</3> (cualquier recurso compartido SMB / CIFS, p. ej. UNAS o Synology).';
	@override String get newTarget => 'Nuevo destino';
	@override String get listFailedToast => 'No se pudieron listar los destinos';
	@override String deleteConfirm({required Object id}) => '¿Eliminar el destino ${id}? Las programaciones que lo referencien bloquearán la eliminación.';
	@override String get deletedToast => 'Destino eliminado';
	@override String get deleteFailedToast => 'Error al eliminar';
	@override String get connectionOkToast => 'Conexión correcta';
	@override String get connectionFailedToast => 'Error de conexión';
	@override String get testFailedToast => 'Error en la prueba';
	@override late final _TranslationsWebBackupsTargetsTabColumnsEs columns = _TranslationsWebBackupsTargetsTabColumnsEs._(_root);
	@override String get on => 'activado';
	@override String get off => 'desactivado';
	@override String get test => 'Probar';
	@override String get testing => 'Probando…';
	@override String get deleteTooltip => 'Eliminar';
}

// Path: web.backups.targetEditor
class _TranslationsWebBackupsTargetEditorEs extends TranslationsWebBackupsTargetEditorEn {
	_TranslationsWebBackupsTargetEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Nuevo destino de copia de seguridad';
	@override String get kindPicker => '¿Dónde quieres hacer la copia de seguridad?';
	@override String get idLabel => 'ID (opcional)';
	@override String get idPlaceholder => 'se genera automáticamente si se deja en blanco, p. ej. tgt_xxx';
	@override String get createdToast => 'Destino creado';
	@override String get createFailedToast => 'Error al crear';
	@override String get creating => 'Creando…';
	@override String get create => 'Crear destino';
	@override String get enableImmediately => 'Habilitar inmediatamente (de lo contrario se guarda como deshabilitado, útil para "configurar ahora, activar más tarde")';
	@override late final _TranslationsWebBackupsTargetEditorLocalEs local = _TranslationsWebBackupsTargetEditorLocalEs._(_root);
	@override late final _TranslationsWebBackupsTargetEditorSmbEs smb = _TranslationsWebBackupsTargetEditorSmbEs._(_root);
	@override late final _TranslationsWebBackupsTargetEditorS3Es s3 = _TranslationsWebBackupsTargetEditorS3Es._(_root);
	@override late final _TranslationsWebBackupsTargetEditorWebdavEs webdav = _TranslationsWebBackupsTargetEditorWebdavEs._(_root);
	@override late final _TranslationsWebBackupsTargetEditorSftpEs sftp = _TranslationsWebBackupsTargetEditorSftpEs._(_root);
	@override late final _TranslationsWebBackupsTargetEditorRcloneEs rclone = _TranslationsWebBackupsTargetEditorRcloneEs._(_root);
}

// Path: web.serverSettings.sections
class _TranslationsWebServerSettingsSectionsEs extends TranslationsWebServerSettingsSectionsEn {
	_TranslationsWebServerSettingsSectionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebServerSettingsSectionsGeneralEs general = _TranslationsWebServerSettingsSectionsGeneralEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsLoggingEs logging = _TranslationsWebServerSettingsSectionsLoggingEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsSessionsEs sessions = _TranslationsWebServerSettingsSectionsSessionsEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsVaultEs vault = _TranslationsWebServerSettingsSectionsVaultEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsMcpEs mcp = _TranslationsWebServerSettingsSectionsMcpEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsMemoryEs memory = _TranslationsWebServerSettingsSectionsMemoryEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsBackupEs backup = _TranslationsWebServerSettingsSectionsBackupEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsClaudeEs claude = _TranslationsWebServerSettingsSectionsClaudeEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsCodexEs codex = _TranslationsWebServerSettingsSectionsCodexEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsGeminiEs gemini = _TranslationsWebServerSettingsSectionsGeminiEs._(_root);
	@override late final _TranslationsWebServerSettingsSectionsAntigravityEs antigravity = _TranslationsWebServerSettingsSectionsAntigravityEs._(_root);
}

// Path: web.serverSettings.restart
class _TranslationsWebServerSettingsRestartEs extends TranslationsWebServerSettingsRestartEn {
	_TranslationsWebServerSettingsRestartEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get button => 'Reiniciar servidor';
	@override String get buttonTitle => 'Auto-ejecutar el proceso del gateway';
	@override String get dirtyConfirm => 'Tienes cambios sin guardar. El reinicio usará la ÚLTIMA configuración GUARDADA. ¿Continuar?';
	@override String get confirm => '¿Reiniciar el gateway de opendray? Todas las sesiones de terminal abiertas se reconectarán automáticamente.';
	@override String get overlay => 'Reiniciando servidor…';
	@override String waiting({required Object tick}) => 'Esperando a /health · ${tick}s';
	@override String get timedOutTitle => 'Se agotó el tiempo del reinicio';
	@override String get timedOutDesc => 'El endpoint de salud nunca respondió. Revisa los logs del servidor.';
	@override String get successToast => 'Servidor reiniciado';
}

// Path: web.serverSettings.formGroups
class _TranslationsWebServerSettingsFormGroupsEs extends TranslationsWebServerSettingsFormGroupsEn {
	_TranslationsWebServerSettingsFormGroupsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get network => 'Red';
	@override String get operatorAccount => 'Cuenta de operador';
	@override String get memoryConfiguration => 'Configuración';
	@override String get memoryHttp => 'Backend HTTP (se usa cuando backend=http)';
	@override String get memoryLocal => 'ONNX local (se usa cuando backend=local)';
	@override String get backupStatus => 'Estado';
	@override String get backupWhere => 'Dónde van las copias de seguridad';
	@override String get backupSchedules => 'Programaciones';
	@override String get backupWhatsInside => '¿Qué contiene una copia de seguridad?';
	@override String get memoryGovernance => 'Gobernanza de fondo (gatekeeper / cleaner)';
	@override String get knowledgeGraph => 'Grafo de conocimiento';
	@override String get database => 'Base de datos';
}

// Path: web.serverSettings.fields
class _TranslationsWebServerSettingsFieldsEs extends TranslationsWebServerSettingsFieldsEn {
	_TranslationsWebServerSettingsFieldsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebServerSettingsFieldsListenAddressEs listenAddress = _TranslationsWebServerSettingsFieldsListenAddressEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsUsernameEs username = _TranslationsWebServerSettingsFieldsUsernameEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsPasswordEs password = _TranslationsWebServerSettingsFieldsPasswordEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsTokenTTLEs tokenTTL = _TranslationsWebServerSettingsFieldsTokenTTLEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsLogLevelEs logLevel = _TranslationsWebServerSettingsFieldsLogLevelEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsLogFormatEs logFormat = _TranslationsWebServerSettingsFieldsLogFormatEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsLogFileEs logFile = _TranslationsWebServerSettingsFieldsLogFileEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsIdleThresholdEs idleThreshold = _TranslationsWebServerSettingsFieldsIdleThresholdEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsIdlePollIntervalEs idlePollInterval = _TranslationsWebServerSettingsFieldsIdlePollIntervalEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsVaultRootEs vaultRoot = _TranslationsWebServerSettingsFieldsVaultRootEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsNotesDirectoryEs notesDirectory = _TranslationsWebServerSettingsFieldsNotesDirectoryEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsSkillsDirectoryEs skillsDirectory = _TranslationsWebServerSettingsFieldsSkillsDirectoryEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsGitRootEs gitRoot = _TranslationsWebServerSettingsFieldsGitRootEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsPersonalPrefixEs personalPrefix = _TranslationsWebServerSettingsFieldsPersonalPrefixEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsProjectsPrefixEs projectsPrefix = _TranslationsWebServerSettingsFieldsProjectsPrefixEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsRegistryRootEs registryRoot = _TranslationsWebServerSettingsFieldsRegistryRootEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsSecretsFileEs secretsFile = _TranslationsWebServerSettingsFieldsSecretsFileEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryBackendEs memoryBackend = _TranslationsWebServerSettingsFieldsMemoryBackendEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryStoreEs memoryStore = _TranslationsWebServerSettingsFieldsMemoryStoreEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryTopKEs memoryTopK = _TranslationsWebServerSettingsFieldsMemoryTopKEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryThresholdEs memoryThreshold = _TranslationsWebServerSettingsFieldsMemoryThresholdEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryScopeEs memoryScope = _TranslationsWebServerSettingsFieldsMemoryScopeEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryBaseUrlEs memoryBaseUrl = _TranslationsWebServerSettingsFieldsMemoryBaseUrlEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryModelEs memoryModel = _TranslationsWebServerSettingsFieldsMemoryModelEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryApiKeyEs memoryApiKey = _TranslationsWebServerSettingsFieldsMemoryApiKeyEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryLocalModelEs memoryLocalModel = _TranslationsWebServerSettingsFieldsMemoryLocalModelEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryLibraryPathEs memoryLibraryPath = _TranslationsWebServerSettingsFieldsMemoryLibraryPathEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryModelPathEs memoryModelPath = _TranslationsWebServerSettingsFieldsMemoryModelPathEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryTokenizerPathEs memoryTokenizerPath = _TranslationsWebServerSettingsFieldsMemoryTokenizerPathEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEs memoryMaxSeqLen = _TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsClaudeHistoryRootsEs claudeHistoryRoots = _TranslationsWebServerSettingsFieldsClaudeHistoryRootsEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsClaudeAccountsDirEs claudeAccountsDir = _TranslationsWebServerSettingsFieldsClaudeAccountsDirEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsCodexSessionsRootEs codexSessionsRoot = _TranslationsWebServerSettingsFieldsCodexSessionsRootEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsGeminiTmpRootEs geminiTmpRoot = _TranslationsWebServerSettingsFieldsGeminiTmpRootEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsGeminiProjectsFileEs geminiProjectsFile = _TranslationsWebServerSettingsFieldsGeminiProjectsFileEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsAntigravityConversationsRootEs antigravityConversationsRoot = _TranslationsWebServerSettingsFieldsAntigravityConversationsRootEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsBackupLocalDirEs backupLocalDir = _TranslationsWebServerSettingsFieldsBackupLocalDirEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsBackupExportDirEs backupExportDir = _TranslationsWebServerSettingsFieldsBackupExportDirEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsBackupPgDumpPathEs backupPgDumpPath = _TranslationsWebServerSettingsFieldsBackupPgDumpPathEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsBackupPgRestorePathEs backupPgRestorePath = _TranslationsWebServerSettingsFieldsBackupPgRestorePathEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryDedupEs memoryDedup = _TranslationsWebServerSettingsFieldsMemoryDedupEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsGatekeeperEnabledEs gatekeeperEnabled = _TranslationsWebServerSettingsFieldsGatekeeperEnabledEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsGatekeeperLatencyEs gatekeeperLatency = _TranslationsWebServerSettingsFieldsGatekeeperLatencyEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsCleanerEnabledEs cleanerEnabled = _TranslationsWebServerSettingsFieldsCleanerEnabledEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsCleanerIntervalEs cleanerInterval = _TranslationsWebServerSettingsFieldsCleanerIntervalEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsCleanerGlobalScopeEs cleanerGlobalScope = _TranslationsWebServerSettingsFieldsCleanerGlobalScopeEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsKnowledgeEnabledEs knowledgeEnabled = _TranslationsWebServerSettingsFieldsKnowledgeEnabledEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsClaudeWatcherEs claudeWatcher = _TranslationsWebServerSettingsFieldsClaudeWatcherEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsClaudeAutoFailoverEs claudeAutoFailover = _TranslationsWebServerSettingsFieldsClaudeAutoFailoverEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMobileTokenTTLEs mobileTokenTTL = _TranslationsWebServerSettingsFieldsMobileTokenTTLEs._(_root);
	@override late final _TranslationsWebServerSettingsFieldsDbMaxConnsEs dbMaxConns = _TranslationsWebServerSettingsFieldsDbMaxConnsEs._(_root);
}

// Path: web.serverSettings.liveTail
class _TranslationsWebServerSettingsLiveTailEs extends TranslationsWebServerSettingsLiveTailEn {
	_TranslationsWebServerSettingsLiveTailEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get heading => 'Seguimiento en vivo';
	@override String get description => 'Búfer circular en memoria (los últimos ~2.000 registros). Se reinicia al reiniciar.';
}

// Path: web.serverSettings.memoryInspectorCard
class _TranslationsWebServerSettingsMemoryInspectorCardEs extends TranslationsWebServerSettingsMemoryInspectorCardEn {
	_TranslationsWebServerSettingsMemoryInspectorCardEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get heading => 'Inspector';
	@override String get description => 'Explora, busca y edita las memorias almacenadas en la página dedicada.';
	@override String get openButton => 'Abrir Memory →';
}

// Path: web.serverSettings.stringList
class _TranslationsWebServerSettingsStringListEs extends TranslationsWebServerSettingsStringListEn {
	_TranslationsWebServerSettingsStringListEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get noneDefault => '(ninguno, usando los valores por defecto integrados)';
	@override String get addPath => 'Añadir ruta';
	@override String get removeTitle => 'Eliminar';
}

// Path: web.serverSettings.httpHelpers
class _TranslationsWebServerSettingsHttpHelpersEs extends TranslationsWebServerSettingsHttpHelpersEn {
	_TranslationsWebServerSettingsHttpHelpersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get autoDetected => 'Detectado automáticamente al arrancar';
	@override String modelCount({required Object count}) => '${count} modelo(s), haz clic para usar';
	@override String get presets => 'Preajustes:';
	@override String get testConnection => 'Probar conexión';
	@override late final _TranslationsWebServerSettingsHttpHelpersPresetTipEs presetTip = _TranslationsWebServerSettingsHttpHelpersPresetTipEs._(_root);
}

// Path: web.serverSettings.probe
class _TranslationsWebServerSettingsProbeEs extends TranslationsWebServerSettingsProbeEn {
	_TranslationsWebServerSettingsProbeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String unreachable({required Object error}) => '✗ inaccesible: ${error}';
	@override String get connectionFailed => 'conexión fallida';
	@override String reachable({required Object detected, required Object total, required Object embedding}) => '✓ accesible ${detected}· ${total} modelo(s) en total · ${embedding} embedding';
	@override String modelMissing({required Object model}) => '⚠ El modelo configurado ${model} no está en la lista. Elige uno de los modelos de embedding de abajo o corrige el nombre.';
	@override String get embeddingModelsLabel => 'modelos de embedding:';
	@override String moreModels({required Object count}) => '+${count} más';
	@override String get noEmbeddingFound => '⚠ Ningún nombre de modelo contiene "embed". Puede que el endpoint no tenga cargado un modelo de embedding, revisa tu servidor local.';
	@override String get configuredTitle => 'Configurado actualmente';
	@override String get applyTitle => 'Haz clic para aplicar';
}

// Path: web.serverSettings.backup
class _TranslationsWebServerSettingsBackupEs extends TranslationsWebServerSettingsBackupEn {
	_TranslationsWebServerSettingsBackupEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get featureDisabledTitle => 'Función deshabilitada';
	@override String get featureDisabledHint => 'Define <1>OPENDRAY_BACKUP_ENABLED=1</1> + <3>OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;</3> en el entorno de opendray y luego reinicia. La passphrase maestra es solo de entorno, nunca toca config.toml.';
	@override String get statusRowLabel => 'Estado';
	@override String get enabledHealthy => 'habilitado · correcto';
	@override String get enabledDegraded => 'habilitado · degradado';
	@override String get keyFingerprintLabel => 'Huella de la clave';
	@override String get keyFingerprintHint => 'guárdala en Vaultwarden, perderla bloquea todas las copias de seguridad anteriores';
	@override String get pgDumpLabel => 'pg_dump';
	@override String get pgDumpUnavailable => 'no disponible';
	@override String get pgRestoreLabel => 'pg_restore';
	@override String get pgRestoreNotResolved => '(no resuelto)';
	@override String get openBackups => 'Abrir la página de Backups →';
	@override String get openExport => 'Abrir Exportar / Importar →';
	@override String get whereDesc => 'Cada destino es un lugar donde puede escribirse un blob de copia de seguridad. opendray admite <1>disco local</1>, <3>SMB/CIFS</3> (Windows / NAS), <5>compatible con S3</5> (AWS, R2, B2, MinIO, Alibaba Cloud OSS, Tencent Cloud COS, ...), <7>WebDAV</7> (Nextcloud, Synology, Jianguoyun), <9>SFTP</9>, además de un puente <11>rclone</11> que conecta con más de 70 backends adicionales (Google Drive, OneDrive, Dropbox, Baidu Pan, Aliyun Drive, ...).';
	@override String get loading => 'Cargando…';
	@override String get noTargets => 'Aún no hay destinos. Añade uno para empezar a hacer copias de seguridad.';
	@override String get addTarget => 'Añadir destino';
	@override String get noSchedulesHint => 'No hay programaciones recurrentes. Añade una en <1>/backups → Programaciones</1> para hacer copias de seguridad automáticamente.';
	@override late final _TranslationsWebServerSettingsBackupScheduleHeadersEs scheduleHeaders = _TranslationsWebServerSettingsBackupScheduleHeadersEs._(_root);
	@override String every({required Object interval}) => 'cada ${interval}';
	@override String backupsKeep({required Object count}) => '${count} copias de seguridad';
	@override String get stateEnabled => 'habilitado';
	@override String get statePaused => 'pausado';
	@override String get manageSchedules => 'Gestionar en /backups → Programaciones →';
	@override String get whatsInsideDesc => 'Cada copia de seguridad es un <1>pg_dump --format=custom</1> de cada tabla de opendray (sessions, integraciones, memorias, audit_log, etc.) más un <3>manifest.json</3> y (opcionalmente) el <5>config.toml</5> en vivo. Abre el panel "¿Qué contiene una copia de seguridad?" en la <7>página de Backups</7> para ver el inventario en vivo con el número de filas.';
	@override String get advancedToggle => 'Avanzado (rutas y binarios cliente), requiere reinicio';
}

// Path: web.serverSettings.targetRow
class _TranslationsWebServerSettingsTargetRowEs extends TranslationsWebServerSettingsTargetRowEn {
	_TranslationsWebServerSettingsTargetRowEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get on => 'activado';
	@override String get off => 'desactivado';
	@override String get test => 'Probar';
	@override String get testing => 'Probando…';
	@override String get delete => 'Eliminar';
	@override String connectionOk({required Object id}) => '${id}: conexión correcta';
	@override String get connectionFailedTitle => 'Conexión fallida';
	@override String get testFailedTitle => 'Prueba fallida';
	@override String deleteConfirm({required Object id}) => '¿Eliminar el destino "${id}"? Las programaciones que lo referencien bloquearán la eliminación.';
	@override String get deleteSuccess => 'Destino eliminado';
	@override String get deleteFailedTitle => 'Error al eliminar';
	@override String get unknownError => 'Error desconocido';
}

// Path: web.serverSettings.toggle
class _TranslationsWebServerSettingsToggleEs extends TranslationsWebServerSettingsToggleEn {
	_TranslationsWebServerSettingsToggleEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get on => 'Activado';
	@override String get off => 'Desactivado';
	@override String get defaultOn => 'Por defecto (on)';
	@override String get defaultOff => 'Por defecto (off)';
}

// Path: web.settings.groups
class _TranslationsWebSettingsGroupsEs extends TranslationsWebSettingsGroupsEn {
	_TranslationsWebSettingsGroupsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get workspace => 'Espacio de trabajo';
	@override String get server => 'Servidor';
	@override String get system => 'Sistema';
}

// Path: web.settings.items
class _TranslationsWebSettingsItemsEs extends TranslationsWebSettingsItemsEn {
	_TranslationsWebSettingsItemsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get appearance => 'Apariencia';
	@override String get font => 'Tamaño de fuente';
	@override String get account => 'Cuenta';
	@override String get status => 'Estado';
	@override String get about => 'Acerca de';
}

// Path: web.settings.health
class _TranslationsWebSettingsHealthEs extends TranslationsWebSettingsHealthEn {
	_TranslationsWebSettingsHealthEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get connecting => 'conectando…';
	@override String get dbOk => 'db ok';
	@override String get dbDown => 'db caída';
}

// Path: web.settings.breadcrumb
class _TranslationsWebSettingsBreadcrumbEs extends TranslationsWebSettingsBreadcrumbEn {
	_TranslationsWebSettingsBreadcrumbEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get server => 'Servidor';
}

// Path: web.settings.appearance
class _TranslationsWebSettingsAppearanceEs extends TranslationsWebSettingsAppearanceEn {
	_TranslationsWebSettingsAppearanceEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Apariencia';
	@override String get description => 'Elige el aspecto de opendray.';
	@override late final _TranslationsWebSettingsAppearanceOptionsEs options = _TranslationsWebSettingsAppearanceOptionsEs._(_root);
}

// Path: web.settings.font
class _TranslationsWebSettingsFontEs extends TranslationsWebSettingsFontEn {
	_TranslationsWebSettingsFontEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Tamaño de fuente';
	@override String get description => 'Escala toda la interfaz. Se guarda por navegador.';
	@override late final _TranslationsWebSettingsFontOptionsEs options = _TranslationsWebSettingsFontOptionsEs._(_root);
}

// Path: web.settings.account
class _TranslationsWebSettingsAccountEs extends TranslationsWebSettingsAccountEn {
	_TranslationsWebSettingsAccountEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cuenta';
	@override String get description => 'Operador y token de portador actual.';
	@override String get username => 'Nombre de usuario';
	@override String get tokenExpires => 'El token caduca';
	@override String get changeCredentials => 'Cambiar credenciales';
}

// Path: web.settings.changeCredentials
class _TranslationsWebSettingsChangeCredentialsEs extends TranslationsWebSettingsChangeCredentialsEn {
	_TranslationsWebSettingsChangeCredentialsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cambiar credenciales';
	@override String get description => 'Verifica tu contraseña actual y luego elige nuevas credenciales. Se revocarán todas las demás sesiones con sesión iniciada.';
	@override String get currentPassword => 'Contraseña actual';
	@override String get newUsername => 'Nuevo nombre de usuario';
	@override String get newPassword => 'Nueva contraseña';
	@override String get newPasswordHint => 'Al menos 8 caracteres.';
	@override String get confirm => 'Confirmar nueva contraseña';
	@override String get errorTooShort => 'La nueva contraseña debe tener al menos 8 caracteres.';
	@override String get errorMismatch => 'La nueva contraseña y la confirmación no coinciden.';
	@override String get errorWrongPassword => 'La contraseña actual es incorrecta.';
	@override String get cancel => 'Cancelar';
	@override String get update => 'Actualizar';
	@override String get saving => 'Guardando…';
}

// Path: web.settings.system
class _TranslationsWebSettingsSystemEs extends TranslationsWebSettingsSystemEn {
	_TranslationsWebSettingsSystemEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Estado del sistema';
	@override String get description => 'Estado en vivo desde el endpoint /health del gateway.';
	@override String get status => 'Estado';
	@override String get version => 'Versión';
	@override String get uptime => 'Tiempo de actividad';
	@override String get database => 'Base de datos';
	@override String get reachable => 'accesible';
	@override String get unreachable => 'no accesible';
}

// Path: web.settings.about
class _TranslationsWebSettingsAboutEs extends TranslationsWebSettingsAboutEn {
	_TranslationsWebSettingsAboutEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Acerca de';
	@override String get description => 'opendray v2: el multiplexor + gateway de integración para CLIs de agentes de IA. Código bajo Apache 2.0.';
	@override String get version => 'Versión';
	@override String get commit => 'Commit';
	@override String updateAvailable({required Object version}) => 'Actualización disponible: ${version}';
	@override String get releaseNotes => 'Notas de la versión ↗';
	@override String get updateNow => 'Actualizar ahora';
	@override String get upgradingShort => 'Actualizando…';
	@override String get confirmRestart => 'Esto reinicia el servicio; las sesiones en ejecución se reconectan.';
	@override String get confirmUpgrade => 'Actualizar y reiniciar';
	@override String upgrading({required Object version}) => 'Actualizando a ${version}…';
	@override String upgraded({required Object version}) => 'Actualizado a ${version}.';
	@override String get upgradeSlow => 'La actualización está tardando un poco. Revisa los logs del servicio si no vuelve.';
	@override String get guidedHint => 'La actualización dentro de la app no está disponible aquí. Ejecuta en el servidor:';
	@override String get checkFailed => 'No se pudieron comprobar las actualizaciones (sin conexión o con límite de frecuencia).';
	@override String get upToDate => 'Tienes la última versión.';
	@override String get checkUpdates => 'Comprobar actualizaciones';
	@override String get checking => 'Comprobando…';
	@override String get reinstall => 'Reinstalar';
}

// Path: web.memoryAmbient.header
class _TranslationsWebMemoryAmbientHeaderEs extends TranslationsWebMemoryAmbientHeaderEn {
	_TranslationsWebMemoryAmbientHeaderEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Memoria ambiental: captura e inyección automáticas';
	@override String get body => 'opendray sondea cada session de agente activa cada 10 segundos, extrae hechos duraderos mediante un LLM configurable y los deduplica antes de almacenarlos en el pool de memoria compartida. Configura qué LLM realiza la extracción (Proveedor), cuándo se activa la extracción (Regla de captura) y qué (si es que algo) se antepone al system prompt del agente al arrancar (Perfil de inyección).';
}

// Path: web.memoryAmbient.providers
class _TranslationsWebMemoryAmbientProvidersEs extends TranslationsWebMemoryAmbientProvidersEn {
	_TranslationsWebMemoryAmbientProvidersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Proveedores de resumen';
	@override String get addButton => 'Añadir proveedor';
	@override String get intro => 'Se requiere al menos un proveedor habilitado para que la captura se active realmente. Las opciones locales (Ollama, LM Studio, Integración) mantienen tus transcripts fuera de redes externas.';
	@override String get empty => 'Aún no hay proveedores configurados.';
	@override late final _TranslationsWebMemoryAmbientProvidersRowEs row = _TranslationsWebMemoryAmbientProvidersRowEs._(_root);
	@override late final _TranslationsWebMemoryAmbientProvidersDialogEs dialog = _TranslationsWebMemoryAmbientProvidersDialogEs._(_root);
	@override late final _TranslationsWebMemoryAmbientProvidersModelSelectEs modelSelect = _TranslationsWebMemoryAmbientProvidersModelSelectEs._(_root);
}

// Path: web.memoryAmbient.rules
class _TranslationsWebMemoryAmbientRulesEs extends TranslationsWebMemoryAmbientRulesEn {
	_TranslationsWebMemoryAmbientRulesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Reglas de captura';
	@override String get addButton => 'Añadir regla';
	@override String get intro => 'Cada regla dice "cuando se active este disparador, resume los nuevos mensajes del transcript y almacena los hechos duraderos." Las reglas por session prevalecen sobre el valor predeterminado global. La v1 incluye 4 tipos de disparador.';
	@override String get empty => 'Aún no hay reglas de captura. Añade una para habilitar la captura automática.';
	@override late final _TranslationsWebMemoryAmbientRulesRowEs row = _TranslationsWebMemoryAmbientRulesRowEs._(_root);
	@override late final _TranslationsWebMemoryAmbientRulesDialogEs dialog = _TranslationsWebMemoryAmbientRulesDialogEs._(_root);
}

// Path: web.memoryAmbient.profiles
class _TranslationsWebMemoryAmbientProfilesEs extends TranslationsWebMemoryAmbientProfilesEn {
	_TranslationsWebMemoryAmbientProfilesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Perfiles de inyección';
	@override String get addButton => 'Añadir perfil';
	@override String get intro => 'Al arrancar, opendray antepone un banner en markdown con las memorias recientes del proyecto al system prompt del agente, SI hay un perfil configurado. Sin un perfil, el modelo sigue usando memory_search bajo demanda.';
	@override String get empty => 'No hay perfil de inyección. Las memorias no se inyectan automáticamente al arrancar; el modelo sigue usando memory_search.';
	@override late final _TranslationsWebMemoryAmbientProfilesRowEs row = _TranslationsWebMemoryAmbientProfilesRowEs._(_root);
	@override late final _TranslationsWebMemoryAmbientProfilesDialogEs dialog = _TranslationsWebMemoryAmbientProfilesDialogEs._(_root);
}

// Path: web.memoryAmbient.cost
class _TranslationsWebMemoryAmbientCostEs extends TranslationsWebMemoryAmbientCostEn {
	_TranslationsWebMemoryAmbientCostEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Coste de tokens (histórico total)';
	@override String get intro => 'Resumen por proveedor agregado a partir de <1>memory_summarizer_calls</1>. Los proveedores locales (Ollama, LM Studio, Integración) tienen precio de \$0: el operador asume el coste del hardware.';
	@override String get empty => 'No hay proveedores habilitados: no hay datos de coste.';
	@override late final _TranslationsWebMemoryAmbientCostColumnsEs columns = _TranslationsWebMemoryAmbientCostColumnsEs._(_root);
}

// Path: web.noteEditor.status
class _TranslationsWebNoteEditorStatusEs extends TranslationsWebNoteEditorStatusEn {
	_TranslationsWebNoteEditorStatusEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get saveFailed => 'error al guardar';
	@override String get saving => 'guardando…';
	@override String get unsaved => 'sin guardar';
	@override String get newNote => 'nota nueva';
	@override String get saved => 'guardada';
}

// Path: web.export.sections
class _TranslationsWebExportSectionsEs extends TranslationsWebExportSectionsEn {
	_TranslationsWebExportSectionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get export => 'Exportar';
	@override String get import => 'Importar';
}

// Path: web.export.form
class _TranslationsWebExportFormEs extends TranslationsWebExportFormEn {
	_TranslationsWebExportFormEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get scope => 'Alcance';
	@override String get memories => 'Memorias';
	@override String get memoriesHint => 'Filas de memoria persistente entre CLI (texto + alcance + metadatos). Los vectores de embedding se omiten; el importador vuelve a generarlos.';
	@override String get integrations => 'Integraciones';
	@override String get customTasks => 'Tareas personalizadas';
	@override String get customTasksHint => 'Tareas definidas por el operador que se muestran en la pestaña Tareas del Inspector.';
	@override late final _TranslationsWebExportFormIntegrationOptionsEs integrationOptions = _TranslationsWebExportFormIntegrationOptionsEs._(_root);
	@override String get confirmWarning => 'Escribe <1>Lo entiendo</1> para confirmar. opendray actualmente almacena solo hashes bcrypt, así que seleccionar texto plano NO exporta ningún texto plano (la función está reservada para una versión futura que mantenga cachés de texto plano).';
	@override String get confirmPlaceholder => 'Lo entiendo';
	@override String get confirmSentinel => 'lo entiendo';
	@override String get footnote => 'Los logs de auditoría y los transcripts de session quedan fuera del alcance; en su lugar los cubre /backups (volcado del operador).';
	@override String get building => 'Generando…';
	@override String get create => 'Crear exportación';
	@override String get readyToast => 'Exportación lista';
	@override String readyDescription({required Object bytes}) => '${bytes} bytes';
	@override String get failedToast => 'Falló la exportación';
}

// Path: web.export.history
class _TranslationsWebExportHistoryEs extends TranslationsWebExportHistoryEn {
	_TranslationsWebExportHistoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando…';
	@override String get empty => 'Aún no hay exportaciones. Usa el formulario de arriba para crear una.';
	@override String get title => 'Historial';
	@override late final _TranslationsWebExportHistoryColumnsEs columns = _TranslationsWebExportHistoryColumnsEs._(_root);
	@override String get download => 'Descargar';
	@override String get deleteTooltip => 'Eliminar';
	@override String get listFailedToast => 'No se pudieron listar las exportaciones';
	@override String get downloadFailedToast => 'Falló la descarga';
	@override String get noTokenToast => 'Sin token de descarga (¿caducado?)';
	@override String deleteConfirm({required Object id}) => '¿Eliminar la exportación ${id}?';
	@override String get deletedToast => 'Exportación eliminada';
	@override String get deleteFailedToast => 'Falló la eliminación';
	@override String get scopeEmpty => '(vacío)';
}

// Path: web.export.import
class _TranslationsWebExportImportEs extends TranslationsWebExportImportEn {
	_TranslationsWebExportImportEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get intro => 'Reproduce un paquete de exportación (zip) en la base de datos en vivo. Los conflictos (id coincidente, o route_prefix único para integraciones) se <1>omiten</1> de forma predeterminada. Las memorias se etiquetan con <3>embedder=imported_v1</3> y necesitan una pasada de re-embedding antes de que la búsqueda las devuelva; activa el re-embedding en <5>Memory → Maintenance</5>. Las integraciones se importan con <7>enabled=false</7> y una clave de marcador de posición sin bcrypt; el operador debe rotarla antes de usarla.';
	@override String get memoryLink => 'Memory → Maintenance';
	@override String get bundleLabel => 'Paquete (.zip)';
	@override String get memoriesLabel => 'Memorias';
	@override String get integrationsLabel => 'Integraciones (solo metadatos, las claves nunca se importan)';
	@override String get customTasksLabel => 'Tareas personalizadas';
	@override String get importing => 'Importando…';
	@override String get importBundle => 'Importar paquete';
	@override String get pickFileToast => 'Selecciona primero un archivo de paquete';
	@override String get doneToast => 'Importación completada';
	@override String get finishedWithErrors => 'La importación terminó con errores';
	@override String get failedToast => 'Falló la importación';
	@override late final _TranslationsWebExportImportSummaryCardEs summaryCard = _TranslationsWebExportImportSummaryCardEs._(_root);
}

// Path: web.export.imports
class _TranslationsWebExportImportsEs extends TranslationsWebExportImportsEn {
	_TranslationsWebExportImportsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando…';
	@override String get empty => 'Aún no hay importaciones.';
	@override String get title => 'Historial';
	@override late final _TranslationsWebExportImportsColumnsEs columns = _TranslationsWebExportImportsColumnsEs._(_root);
	@override String get noneCounts => '(ninguno)';
	@override String get listFailedToast => 'No se pudieron listar las importaciones';
}

// Path: web.knowledge.scopes
class _TranslationsWebKnowledgeScopesEs extends TranslationsWebKnowledgeScopesEn {
	_TranslationsWebKnowledgeScopesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get all => 'Todos';
	@override String get global => 'Global';
	@override String get project => 'Proyecto';
	@override String get domain => 'Dominio';
}

// Path: web.knowledge.kb
class _TranslationsWebKnowledgeKbEs extends TranslationsWebKnowledgeKbEn {
	_TranslationsWebKnowledgeKbEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get tab => 'Base de conocimiento';
	@override String get graphTab => 'Grafo';
	@override String graphCounts({required Object nodes, required Object edges}) => '${nodes} nodos · ${edges} enlaces';
	@override String get global => 'Global';
	@override String get projectHandbook => 'Manual del proyecto';
	@override String get locked => 'Editado por ti';
	@override String get aiDrafted => 'Redactado por IA';
	@override String get edit => 'Editar';
	@override String get unlock => 'Desbloquear (que lo gestione la IA)';
	@override String get regenerate => 'Regenerar';
	@override String get save => 'Guardar';
	@override String get cancel => 'Cancelar';
	@override String get editHint => 'Guardar bloquea esta página para que la IA no la sobrescriba.';
	@override String get empty => 'Aún no generada. Pulsa Regenerar, o se construye automáticamente mientras trabajas.';
	@override String get saved => 'Guardado';
	@override String get unlocked => 'Desbloqueada — la IA volverá a gestionar esta página';
	@override String get regenerating => 'Regenerando en segundo plano…';
	@override late final _TranslationsWebKnowledgeKbKindsEs kinds = _TranslationsWebKnowledgeKbKindsEs._(_root);
	@override String get foundational => 'Fundacional';
	@override String get foundationalHint => 'Infraestructura y convenciones — reglas vinculantes inyectadas en cada proyecto.';
	@override String get emergent => 'Emergente';
	@override String get emergentHint => 'Lecciones y funciones reutilizables destiladas del trabajo previo — orientación.';
	@override String get bindingBadge => 'Vinculante · obligatorio';
	@override String get referenceBadge => 'Referencia';
	@override late final _TranslationsWebKnowledgeKbProposalEs proposal = _TranslationsWebKnowledgeKbProposalEs._(_root);
	@override String get discuss => 'Hablar con la IA';
	@override String get discussHint => 'Redacta de nuevo esta política conversando con la IA — las páginas bloqueadas reciben propuestas, nunca sobrescrituras';
	@override String get onDemand => 'bajo demanda';
	@override String get removePage => 'Quitar página';
	@override String get removePageHint => 'Quita esta página de la base de conocimiento (su contenido se conserva y vuelve si se re-añade el slug)';
	@override String get pageRemovedToast => 'Página quitada';
	@override late final _TranslationsWebKnowledgeKbNewPageEs newPage = _TranslationsWebKnowledgeKbNewPageEs._(_root);
}

// Path: web.knowledge.kinds
class _TranslationsWebKnowledgeKindsEs extends TranslationsWebKnowledgeKindsEn {
	_TranslationsWebKnowledgeKindsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get all => 'Todos';
	@override String get entity => 'Entidades';
	@override String get fact => 'Hechos';
	@override String get playbook => 'Guías';
	@override String get skill => 'Habilidades';
}

// Path: web.knowledge.distill
class _TranslationsWebKnowledgeDistillEs extends TranslationsWebKnowledgeDistillEn {
	_TranslationsWebKnowledgeDistillEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get tab => 'Destilación';
	@override String get intro => 'Un SKILL es un PROCEDIMIENTO probado y repetible destilado de tu trabajo real. El compilador de experiencia mina los diarios de sesión de TODOS los proyectos, agrupa trabajo similar y solo redacta un candidato cuando el mismo procedimiento TUVO ÉXITO en 2+ sesiones — cada cita de evidencia se verifica literalmente contra el diario. Los candidatos se ordenan por recurrencia × el coste en tiempo del procedimiento manual; los procedimientos totalmente mecánicos también se compilan a un run.sh ejecutable con paso de validación.';
	@override String get playbooks => 'Playbooks — destilados, pendientes de revisión';
	@override String get playbooksHint => 'Cada candidato pasó las puertas: ≥2 sesiones exitosas, citas de evidencia verificadas, ≥3 pasos concretos. Ordenados por tiempo ahorrado (recurrencia × minutos manuales). Promueve lo reutilizable, descarta el resto.';
	@override String get playbooksEmpty => 'Nada minado aún — los candidatos aparecen cuando el mismo procedimiento tiene éxito en dos o más sesiones.';
	@override String get skills => 'Skills — activos, inyectados al arranque';
	@override String get skillsHint => 'Playbooks promovidos. Cada sesión nueva los recibe como skills.';
	@override String get skillsEmpty => 'Sin skills aún — promueve un playbook para crear el primero.';
	@override String get skillify => 'Promover a skill';
	@override String get skillifyHint => 'Renderizar como skill e inyectar en cada arranque';
	@override String get discard => 'Descartar';
	@override String get retire => 'Retirar skill';
	@override String get injectedBadge => 'inyectado';
	@override String get skillifiedToast => 'Promovido — publicado en Plugins → Agent Skills; las nuevas sessions reciben esta skill';
	@override String get removedToast => 'Eliminado';
	@override String usage({required Object count}) => 'usado en ${count} sesiones';
	@override String lastUsed({required Object date}) => 'último ${date}';
	@override String get enabledToast => 'Skill activado — las sesiones nuevas lo cargan';
	@override String get disabledToast => 'Skill desactivado — fuera del conjunto cargado';
	@override String get disabledBadge => 'off';
	@override String get toggleHint => 'Solo los skills activados se cargan; desactiva lo que esta etapa no necesita';
	@override String get viewHint => 'Clic para ver el procedimiento completo';
	@override String get inAgentSkills => 'en Plugins → Agent Skills';
	@override String get agentSkillsHint => 'El SKILL.md renderizado vive en el vault de skills — míralo o gestiónalo en Plugins → Agent Skills.';
	@override String get notInVault => 'desactivado — SKILL.md retirado del vault';
	@override String get compiledBadge => 'compilado';
	@override String get compiledHint => 'Incluye un run.sh ejecutable con paso de validación; al promover también se registra como tarea personalizada';
	@override String recurrence({required Object count}) => 'exitoso ×${count}';
	@override String timeCost({required Object minutes}) => '~${minutes} min manual';
	@override String projectSpan({required Object count}) => '${count} proyectos';
	@override String get scoreHint => 'Ordenado por recurrencia × coste de tiempo manual — lo que más tiempo ahorra se destila primero';
	@override String outcomes({required Object ok, required Object failed}) => '${ok} ok / ${failed} fallidas tras cargarlo';
	@override late final _TranslationsWebKnowledgeDistillRetirementEs retirement = _TranslationsWebKnowledgeDistillRetirementEs._(_root);
	@override String get retirementEmpty => 'Sin candidatos a retiro: todas las habilidades aportan.';
	@override String get retirementHint => 'Habilidades que el bucle de resultados propone descartar; desactiva las que consideres.';
	@override String get retirementTitle => 'Candidatos a retiro';
}

// Path: web.knowledge.graph
class _TranslationsWebKnowledgeGraphEs extends TranslationsWebKnowledgeGraphEn {
	_TranslationsWebKnowledgeGraphEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get tab => 'Grafo';
	@override String get intro => 'El mapa de relaciones de todo lo que la IA ha aprendido: qué proyectos comparten tecnología, qué skills y trampas se asocian a qué entidades. Comprueba aquí el radio de impacto de un nodo ANTES de tocar infraestructura compartida.';
	@override String get empty => 'Sin conocimiento aún — el grafo se construye solo mientras corren las sessions: el barrido de anclaje extrae entidades del trabajo de proyecto y la destilación añade playbooks y skills. Vuelve tras unas cuantas sesiones de trabajo.';
	@override String get hint => 'Rueda para zoom · arrastra el fondo para desplazarte · arrastra un nodo para desenredar · clic en un nodo para inspeccionarlo';
	@override late final _TranslationsWebKnowledgeGraphLegendEs legend = _TranslationsWebKnowledgeGraphLegendEs._(_root);
	@override String connections({required Object count}) => '${count} nodos conectados';
	@override String get noLinks => 'Nada enlaza con este nodo todavía.';
}

// Path: web.cortex.home
class _TranslationsWebCortexHomeEs extends TranslationsWebCortexHomeEn {
	_TranslationsWebCortexHomeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cortex';
	@override String get subtitle => 'Un módulo, tres peldaños, un ciclo: la memoria bruta cristaliza en el documento oficial de cada proyecto, se destila en conocimiento entre proyectos y se inyecta en cada nueva sesión.';
	@override String get disabled => 'desactivado';
	@override String pendingProposals({required Object count}) => '${count} pendientes';
	@override String get loopHint => 'Memoria → Notas → Conocimiento → inyectado en cada arranque. Ascender es transformar, nunca copiar.';
	@override String get activeProjects => 'Proyectos activos';
	@override String idle({required Object days}) => 'inactivo ${days}d';
	@override late final _TranslationsWebCortexHomeMemoryEs memory = _TranslationsWebCortexHomeMemoryEs._(_root);
	@override late final _TranslationsWebCortexHomeNotesEs notes = _TranslationsWebCortexHomeNotesEs._(_root);
	@override late final _TranslationsWebCortexHomeKnowledgeEs knowledge = _TranslationsWebCortexHomeKnowledgeEs._(_root);
	@override String get settings => 'Ajustes';
	@override late final _TranslationsWebCortexHomeProposalsEs proposals = _TranslationsWebCortexHomeProposalsEs._(_root);
}

// Path: web.cortex.chat
class _TranslationsWebCortexChatEs extends TranslationsWebCortexChatEn {
	_TranslationsWebCortexChatEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Chat de curación';
	@override String get show => 'Hablar con la IA';
	@override String get hide => 'Ocultar chat';
	@override String get emptyHint => 'Pide a la IA actualizar, reestructurar o reescribir este documento. Los cambios se aplican directamente si lo mantiene la IA, o llegan a la bandeja si lo bloqueaste.';
	@override String get placeholder => 'p. ej. actualiza esto con el trabajo reciente · ⌘↵ para enviar';
	@override String get thinking => 'La IA está trabajando…';
	@override String get sendFailed => 'Error al enviar';
	@override String get escalate => 'Escalar a sesión';
	@override String get escalated => 'Escalado';
	@override String get escalateHint => 'Lanza una sesión de agente completa, fundamentada en el código, con esta conversación';
	@override String get escalateFailed => 'Error al escalar';
	@override String get escalatedToast => 'Sesión de agente lanzada';
	@override String get closeHint => 'Cerrar esta conversación';
	@override String get revisionApplied => 'Documento actualizado';
	@override String get revisionProposed => 'Propuesta creada — revísala en la bandeja';
}

// Path: web.cortex.blueprint
class _TranslationsWebCortexBlueprintEs extends TranslationsWebCortexBlueprintEn {
	_TranslationsWebCortexBlueprintEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get open => 'Plano';
	@override String get openHint => 'Edita qué secciones documenta este proyecto';
	@override String get title => 'Plano del documento';
	@override String get description => 'El conjunto de secciones del documento oficial. Cada tipo de proyecto merece secciones distintas — dale forma tú o deja que la IA proponga.';
	@override String get propose => 'Proponer (IA)';
	@override String get proposeHint => 'Clasifica el proyecto y propone un conjunto de secciones a medida';
	@override String get proposeFailed => 'Propuesta fallida';
	@override String proposalNote({required Object type, required Object reason}) => 'La IA lo clasificó como: ${type} — ${reason} Revísalo, edítalo y aplica.';
	@override String get addSection => 'Añadir sección';
	@override String get slugPlaceholder => 'slug';
	@override String get titlePlaceholder => 'Título';
	@override String get hintPlaceholder => 'Pista para el mantenedor — una frase que guíe a la IA (opcional)';
	@override late final _TranslationsWebCortexBlueprintModeEs mode = _TranslationsWebCortexBlueprintModeEs._(_root);
	@override String get inject => 'inyectar';
	@override String get reserved => 'reservada';
	@override String get deleteNote => 'Quitar una sección la oculta sin borrar su contenido — vuelve a añadir el mismo slug para recuperarla.';
	@override String get cancel => 'Cancelar';
	@override String get apply => 'Aplicar plano';
	@override String get applyFailed => 'Error al aplicar';
	@override String get appliedToast => 'Plano aplicado';
}

// Path: web.cortex.quarantine
class _TranslationsWebCortexQuarantineEs extends TranslationsWebCortexQuarantineEn {
	_TranslationsWebCortexQuarantineEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cuarentena';
	@override String get subtitle => 'Hechos que necesitan revisión antes de contar como memoria durable: las capturas de integraciones de terceros llegan aquí por política, y puedes poner cualquier memoria en cuarentena a mano desde el inspector de Memoria. Promueve lo verdadero; descarta el resto — las filas sin revisar expiran solas.';
	@override String get empty => 'Nada en cuarentena. Las filas llegan desde sessions de origen integración (política “quarantine”) o cuando pones una memoria en cuarentena manualmente en el inspector de Memoria.';
	@override String get promote => 'Promocionar';
	@override String get promoteHint => 'Mover a memoria duradera (entra en la recuperación y consolidación)';
	@override String get discard => 'Descartar';
	@override String get promotedToast => 'Promocionada a memoria duradera';
	@override String get discardedToast => 'Descartada';
	@override String get actionFailed => 'Acción fallida';
	@override String expires({required Object date}) => 'expira ${date}';
}

// Path: web.cortex.settings
class _TranslationsWebCortexSettingsEs extends TranslationsWebCortexSettingsEn {
	_TranslationsWebCortexSettingsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebCortexSettingsInjectionEs injection = _TranslationsWebCortexSettingsInjectionEs._(_root);
}

// Path: more.items.integrations
class _TranslationsMoreItemsIntegrationsEs extends TranslationsMoreItemsIntegrationsEn {
	_TranslationsMoreItemsIntegrationsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Integraciones';
	@override String get subtitle => 'Llamadores de la API: actividad reciente y tasas de error';
}

// Path: more.items.activity
class _TranslationsMoreItemsActivityEs extends TranslationsMoreItemsActivityEn {
	_TranslationsMoreItemsActivityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Actividad';
	@override String get subtitle => 'Auditoría de llamadas API de integraciones';
}

// Path: more.items.memoryAmbient
class _TranslationsMoreItemsMemoryAmbientEs extends TranslationsMoreItemsMemoryAmbientEn {
	_TranslationsMoreItemsMemoryAmbientEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Captura e inyección';
	@override String get subtitle => 'Reglas de captura + perfiles de inyección';
}

// Path: more.items.channels
class _TranslationsMoreItemsChannelsEs extends TranslationsMoreItemsChannelsEn {
	_TranslationsMoreItemsChannelsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Canales';
	@override String get subtitle => 'Destinos de notificaciones';
}

// Path: more.items.providers
class _TranslationsMoreItemsProvidersEs extends TranslationsMoreItemsProvidersEn {
	_TranslationsMoreItemsProvidersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Proveedores';
	@override String get subtitle => 'Estado de los CLI de Claude / Codex / Gemini';
}

// Path: more.items.mcp
class _TranslationsMoreItemsMcpEs extends TranslationsMoreItemsMcpEn {
	_TranslationsMoreItemsMcpEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'MCP';
	@override String get subtitle => 'Servidores y secretos de Model Context Protocol';
}

// Path: more.items.skills
class _TranslationsMoreItemsSkillsEs extends TranslationsMoreItemsSkillsEn {
	_TranslationsMoreItemsSkillsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Skills';
	@override String get subtitle => 'Biblioteca de SKILL.md del agente (integrados + vault)';
}

// Path: more.items.gitHosts
class _TranslationsMoreItemsGitHostsEs extends TranslationsMoreItemsGitHostsEn {
	_TranslationsMoreItemsGitHostsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Hosts de Git';
	@override String get subtitle => 'Credenciales PAT para GitHub / GitLab / etc.';
}

// Path: more.items.customTasks
class _TranslationsMoreItemsCustomTasksEs extends TranslationsMoreItemsCustomTasksEn {
	_TranslationsMoreItemsCustomTasksEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Tareas personalizadas';
	@override String get subtitle => 'Comandos slash que se muestran en el selector de tareas de la session';
}

// Path: more.items.cortexHub
class _TranslationsMoreItemsCortexHubEs extends TranslationsMoreItemsCortexHubEn {
	_TranslationsMoreItemsCortexHubEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cortex';
	@override String get subtitle => 'Hub Memoria → Notas → Conocimiento + propuestas pendientes';
}

// Path: more.items.projectMemory
class _TranslationsMoreItemsProjectMemoryEs extends TranslationsMoreItemsProjectMemoryEn {
	_TranslationsMoreItemsProjectMemoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Objetivo / plan / diario del proyecto';
	@override String get subtitle => 'Capas de memoria 2-4 por cwd + propuestas del agente';
}

// Path: more.items.archived
class _TranslationsMoreItemsArchivedEs extends TranslationsMoreItemsArchivedEn {
	_TranslationsMoreItemsArchivedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Memorias archivadas';
	@override String get subtitle => 'Restaura memorias que el limpiador automático archivó (gracia de 30 días)';
}

// Path: more.items.quarantine
class _TranslationsMoreItemsQuarantineEs extends TranslationsMoreItemsQuarantineEn {
	_TranslationsMoreItemsQuarantineEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Cuarentena';
	@override String get subtitle => 'Revisa memorias capturadas antes de que sean durables';
}

// Path: more.items.backups
class _TranslationsMoreItemsBackupsEs extends TranslationsMoreItemsBackupsEn {
	_TranslationsMoreItemsBackupsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Copias de seguridad';
	@override String get subtitle => 'Estado de la última copia de seguridad y ejecución inmediata';
}

// Path: more.items.dataExport
class _TranslationsMoreItemsDataExportEs extends TranslationsMoreItemsDataExportEn {
	_TranslationsMoreItemsDataExportEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Exportación e importación de datos';
	@override String get subtitle => 'Paquetes de datos a nivel de usuario (memorias / integraciones / tareas personalizadas)';
}

// Path: more.items.settings
class _TranslationsMoreItemsSettingsEs extends TranslationsMoreItemsSettingsEn {
	_TranslationsMoreItemsSettingsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Ajustes';
	@override String get subtitle => 'Idioma, apariencia, cuenta';
}

// Path: more.items.about
class _TranslationsMoreItemsAboutEs extends TranslationsMoreItemsAboutEn {
	_TranslationsMoreItemsAboutEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Acerca de';
	@override String get subtitle => 'Versión de compilación e información del servidor';
}

// Path: more.items.vault
class _TranslationsMoreItemsVaultEs extends TranslationsMoreItemsVaultEn {
	_TranslationsMoreItemsVaultEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Bóveda';
	@override String get subtitle => 'Notas markdown libres (sincronización Obsidian)';
}

// Path: sessions.detail.accountSwitcher
class _TranslationsSessionsDetailAccountSwitcherEs extends TranslationsSessionsDetailAccountSwitcherEn {
	_TranslationsSessionsDetailAccountSwitcherEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get tooltip => 'Cambiar de cuenta de Claude';
	@override String get sheetTitle => 'Cambiar de cuenta de Claude';
	@override String current({required Object account}) => 'Actual: ${account}';
	@override String get defaultName => 'Predeterminada (credencial del sistema)';
	@override String get defaultSubtitle => 'Usa el propio inicio de sesión del CLI, sin cuenta específica';
	@override String get defaultShort => 'predeterminada';
	@override String get tokenEmpty => 'sin token';
	@override String get confirmTitle => '¿Cambiar de cuenta?';
	@override String get confirmBody => 'Esto reinicia el CLI con la nueva cuenta — se pierde el contexto de conversación actual dentro del CLI (la pestaña de la sesión se conserva).';
	@override String get confirmAction => 'Cambiar';
	@override String get cancel => 'Cancelar';
	@override String switchedSnack({required Object account}) => 'Cambiado a ${account}';
	@override String switchFailed({required Object error}) => 'Cambio fallido: ${error}';
	@override String get noneHint => 'No hay cuentas de Claude configuradas. Agrégalas en Más → Providers → Claude.';
}

// Path: sessions.terminal.snackbar
class _TranslationsSessionsTerminalSnackbarEs extends TranslationsSessionsTerminalSnackbarEn {
	_TranslationsSessionsTerminalSnackbarEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String imagePickerFailed({required Object error}) => 'Falló el selector de imágenes: ${error}';
	@override String get uploadingImage => 'Subiendo imagen…';
	@override String imageAttached({required Object path}) => 'Imagen adjuntada: ${path}';
	@override String uploadFailed({required Object status, required Object message}) => 'Falló la subida (${status}): ${message}';
	@override String uploadFailedGeneric({required Object error}) => 'Falló la subida: ${error}';
}

// Path: sessions.terminal.imageSource
class _TranslationsSessionsTerminalImageSourceEs extends TranslationsSessionsTerminalImageSourceEn {
	_TranslationsSessionsTerminalImageSourceEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get photoLibrary => 'Biblioteca de fotos';
	@override String get takePhoto => 'Tomar una foto';
}

// Path: sessions.terminal.keyboard
class _TranslationsSessionsTerminalKeyboardEs extends TranslationsSessionsTerminalKeyboardEn {
	_TranslationsSessionsTerminalKeyboardEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get copyBuffer => 'Copiar búfer';
	@override String get paste => 'Pegar';
	@override String get attachImage => 'Adjuntar imagen';
	@override String get enter => 'Intro';
}

// Path: sessions.terminal.connection
class _TranslationsSessionsTerminalConnectionEs extends TranslationsSessionsTerminalConnectionEn {
	_TranslationsSessionsTerminalConnectionEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get connecting => 'Conectando…';
	@override String get connected => 'Conectado';
	@override String get reconnecting => 'Reconectando…';
	@override String reconnectingWithError({required Object error}) => 'Reconectando (${error})…';
	@override String get disconnected => 'Desconectado';
	@override String disconnectedWithError({required Object error}) => 'Desconectado (${error})';
	@override String get ended => 'Sesión finalizada';
}

// Path: sessions.action.errors
class _TranslationsSessionsActionErrorsEs extends TranslationsSessionsActionErrorsEn {
	_TranslationsSessionsActionErrorsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String stop({required Object error}) => 'Falló al detener: ${error}';
	@override String start({required Object error}) => 'Falló al reiniciar: ${error}';
	@override String delete({required Object error}) => 'Falló al eliminar: ${error}';
}

// Path: sessions.dirPicker.dialog
class _TranslationsSessionsDirPickerDialogEs extends TranslationsSessionsDirPickerDialogEn {
	_TranslationsSessionsDirPickerDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Nueva carpeta';
	@override String get hint => 'Nombre de la carpeta';
	@override String get create => 'Crear';
}

// Path: sessions.inspector.shell
class _TranslationsSessionsInspectorShellEs extends TranslationsSessionsInspectorShellEn {
	_TranslationsSessionsInspectorShellEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Inspector';
	@override String loadError({required Object error}) => 'No se pudo cargar la sesión: ${error}';
	@override late final _TranslationsSessionsInspectorShellTabsEs tabs = _TranslationsSessionsInspectorShellTabsEs._(_root);
}

// Path: sessions.inspector.cortex
class _TranslationsSessionsInspectorCortexEs extends TranslationsSessionsInspectorCortexEn {
	_TranslationsSessionsInspectorCortexEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Espacio Cortex';
	@override String get blurb => 'Objetivo, plan, diario, bandeja y limpieza de memoria de este proyecto — el Cortex mantenido por IA.';
	@override String get open => 'Abrir espacio Cortex';
}

// Path: sessions.inspector.shared
class _TranslationsSessionsInspectorSharedEs extends TranslationsSessionsInspectorSharedEn {
	_TranslationsSessionsInspectorSharedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get refresh => 'Actualizar';
	@override String inserted({required Object text}) => 'Insertado: ${text}';
	@override String insertFailedApi({required Object status, required Object message}) => 'Falló la inserción (${status}): ${message}';
	@override String insertFailedGeneric({required Object error}) => 'Falló la inserción: ${error}';
	@override String insertFailedShort({required Object error}) => 'Falló la inserción: ${error}';
}

// Path: sessions.inspector.history
class _TranslationsSessionsInspectorHistoryEs extends TranslationsSessionsInspectorHistoryEn {
	_TranslationsSessionsInspectorHistoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get insertIntoTerminal => 'Insertar en el terminal';
	@override String get searchHint => 'Buscar prompts…';
}

// Path: sessions.inspector.files
class _TranslationsSessionsInspectorFilesEs extends TranslationsSessionsInspectorFilesEn {
	_TranslationsSessionsInspectorFilesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get insertAtRef => 'Insertar como @referencia';
	@override String get insertPath => 'Insertar ruta';
	@override String get insertPathSubtitle => 'Pega la ruta absoluta tal cual';
	@override String get readContent => 'Leer contenido';
	@override String get readContentSubtitle => 'Hasta 256 KiB de texto plano';
	@override String readFailedApi({required Object status, required Object message}) => 'Falló la lectura (${status}): ${message}';
	@override String readFailedGeneric({required Object error}) => 'Falló la lectura: ${error}';
	@override String get parent => 'Superior';
	@override String get backToCwd => 'Volver al cwd de la session';
}

// Path: sessions.inspector.git
class _TranslationsSessionsInspectorGitEs extends TranslationsSessionsInspectorGitEn {
	_TranslationsSessionsInspectorGitEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get insertAtRef => 'Insertar como @referencia';
	@override String get insertPath => 'Insertar ruta';
	@override String get showDiff => 'Mostrar diff';
	@override String diffFailedApi({required Object status, required Object message}) => 'Falló el diff (${status}): ${message}';
	@override String diffFailedGeneric({required Object error}) => 'Falló el diff: ${error}';
	@override String get insertHash => 'Insertar hash';
	@override String get showFullPatch => 'Mostrar el parche completo';
	@override String showFailedApi({required Object status, required Object message}) => 'Falló al mostrar (${status}): ${message}';
	@override String showFailedGeneric({required Object error}) => 'Falló al mostrar: ${error}';
	@override String get tabStatus => 'Estado';
	@override String get tabLog => 'Log';
}

// Path: sessions.inspector.tasks
class _TranslationsSessionsInspectorTasksEs extends TranslationsSessionsInspectorTasksEn {
	_TranslationsSessionsInspectorTasksEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get runCommand => 'Ejecutar comando';
	@override String get runCommandSubtitle => 'Se ejecuta en una nueva session de shell y cambia a ella';
	@override String get filterHint => 'Filtrar tareas…';
	@override String noMatch({required Object query}) => 'Ninguna tarea coincide con "${query}"';
	@override String get emptyTitle => 'No hay tareas en esta carpeta';
	@override String get emptyHint => 'Buscando package.json, Makefile, Taskfile, justfile, Cargo.toml, go.mod, pyproject.toml o scripts de shell';
}

// Path: sessions.inspector.notes
class _TranslationsSessionsInspectorNotesEs extends TranslationsSessionsInspectorNotesEn {
	_TranslationsSessionsInspectorNotesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String insertedAt({required Object path}) => 'Insertado: @${path}';
	@override String get myNotes => 'Mis notas';
	@override String get projectDocs => 'Documentos del proyecto';
	@override String get insertAtRefTooltip => 'Insertar como @referencia';
	@override String get insertAtRefShort => 'Insertar @referencia';
	@override String draftHint({required Object project}) => '# ${project}\n\nIdeas, tareas pendientes, contexto para el agente…';
	@override String createFailed({required Object error}) => 'Falló al crear: ${error}';
	@override String saveFailed({required Object error}) => 'Falló al guardar: ${error}';
	@override String get changeLocationTooltip => 'Cambiar la ubicación de los documentos del proyecto';
	@override String get filenameHint => 'nombre de archivo (p. ej. spec o design.md)';
	@override String get create => 'Crear';
	@override String get filterHint => 'Filtrar…';
	@override String get locationDialogTitle => 'Ubicación de los documentos del proyecto';
	@override String loadFailedApi({required Object error}) => 'Falló la carga: ${error}';
	@override String loadFailedGeneric({required Object error}) => 'Falló la carga: ${error}';
	@override String saveFailedApi({required Object error}) => 'Falló al guardar: ${error}';
	@override String saveFailedGeneric({required Object error}) => 'Falló al guardar: ${error}';
	@override String insertFailedApi({required Object error}) => 'Falló la inserción: ${error}';
	@override String insertFailedGeneric({required Object error}) => 'Falló la inserción: ${error}';
	@override String createFailedApi({required Object error}) => 'Falló al crear: ${error}';
	@override String createFailedGeneric({required Object error}) => 'Falló al crear: ${error}';
	@override String get personalHint => 'Bloc de notas personal. Se guarda automáticamente mientras escribes. Los agentes de IA no escriben aquí.';
	@override String get projectDocsHint => 'Arquitectura / spec / decisiones / plan / retrospectivas. Normalmente redactados o mantenidos por un agente.';
	@override String get mappingCleared => 'Asignación borrada. Usando el valor predeterminado';
	@override String mappedTo({required Object path}) => 'Asignado a ${path}';
	@override String get cancelTooltip => 'Cancelar';
	@override String get newDocTooltip => 'Nuevo documento';
	@override String get noProjectMapping => 'No se pudo resolver una asignación de proyecto para esta session. Comprueba que el gateway tenga configurado un almacén de notas y que el cwd de la session esté establecido.';
	@override String get emptyProjectDocs => 'Aún no hay documentos del proyecto. Toca + para crear uno o deja que un agente de IA lo genere a partir de un prompt.';
	@override String emptyFilterMatch({required Object query}) => 'No hay coincidencias para "${query}".';
	@override String get locationDialogHelp => 'Fija el cwd de esta session a una carpeta específica dentro de tu almacén de notas. Déjalo en blanco para restablecer.';
	@override String get sessionCwd => 'cwd de la session';
	@override String get projectDocsPath => 'Ruta de los documentos del proyecto relativa al almacén';
	@override String get locationStoredHint => 'Almacenado en <vault>/.opendray-projects.json. Se sincroniza con git junto con el resto del almacén.';
	@override String pinnedHint({required Object path, required Object defaultPath}) => 'Fijado a ${path}/ (anula ${defaultPath}). Los agentes de IA también redactan documentos aquí.';
	@override String get noProjectMapping2 => '(sin asignación de proyecto)';
	@override String get clearOverride => 'Borrar anulación';
	@override String get save => 'Guardar';
}

// Path: sessions.spawnSheet.bypass
class _TranslationsSessionsSpawnSheetBypassEs extends TranslationsSessionsSpawnSheetBypassEn {
	_TranslationsSessionsSpawnSheetBypassEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get labelClaude => 'Omitir permisos';
	@override String get labelCodex => 'Omitir aprobaciones y sandbox';
	@override String get labelGemini => 'Modo YOLO';
	@override String get labelAntigravity => 'Omitir permisos / YOLO';
	@override String get subtitleOn => 'Esta session se ejecutará con autonomía elevada.';
	@override String get subtitleOff => 'Desactivado. Las confirmaciones y los prompts se comportan de forma normal.';
}

// Path: sessions.spawnSheet.noProviders
class _TranslationsSessionsSpawnSheetNoProvidersEs extends TranslationsSessionsSpawnSheetNoProvidersEn {
	_TranslationsSessionsSpawnSheetNoProvidersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'No hay proveedores configurados';
	@override String get message => 'El gateway no tiene proveedores de CLI habilitados. Configura uno en Proveedores (admin web) o en [providers] en config.toml, y luego toca Recargar.';
	@override String get reload => 'Recargar';
}

// Path: sessions.spawnSheet.providerLoadError
class _TranslationsSessionsSpawnSheetProviderLoadErrorEs extends TranslationsSessionsSpawnSheetProviderLoadErrorEn {
	_TranslationsSessionsSpawnSheetProviderLoadErrorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'No se pudieron cargar los proveedores';
	@override String get networkError => 'Error de red';
	@override String serverPrefix({required Object code}) => 'Servidor ${code}';
	@override String format({required Object prefix, required Object message}) => '${prefix}: ${message}';
}

// Path: sessions.spawnSheet.claudeAccount
class _TranslationsSessionsSpawnSheetClaudeAccountEs extends TranslationsSessionsSpawnSheetClaudeAccountEn {
	_TranslationsSessionsSpawnSheetClaudeAccountEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Cuenta de Claude';
	@override String get helperMulti => 'Hay varias cuentas configuradas. Elige una para esta session.';
	@override String get helperSingle => 'Elige una cuenta configurada o usa la predeterminada (env / sistema).';
	@override String get kDefault => 'Predeterminada (env / sistema)';
	@override String get disabledSuffix => ' (desactivada)';
	@override String get noTokenSuffix => ' (sin token)';
	@override String get noneHint => 'No hay cuentas de Claude configuradas. El gateway usará la ANTHROPIC_API_KEY del sistema. Añade cuentas en Ajustes → Cuentas en el admin web.';
	@override String errorHint({required Object error}) => 'No se pudieron cargar las cuentas de Claude (${error}). La session se creará con el valor predeterminado del gateway.';
}

// Path: memoryWorkers.tasks.gatekeeper
class _TranslationsMemoryWorkersTasksGatekeeperEs extends TranslationsMemoryWorkersTasksGatekeeperEn {
	_TranslationsMemoryWorkersTasksGatekeeperEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Gatekeeper';
	@override String get description => 'Filtro previo a la escritura en cada memory_store. Alta frecuencia (objetivo <500ms): solo resumidor.';
}

// Path: memoryWorkers.tasks.cleaner
class _TranslationsMemoryWorkersTasksCleanerEs extends TranslationsMemoryWorkersTasksCleanerEn {
	_TranslationsMemoryWorkersTasksCleanerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Bibliotecario de limpieza';
	@override String get description => 'Bibliotecario LLM periódico. Juzga las memorias antiguas como conservar / obsoleta / duplicada.';
}

// Path: memoryWorkers.tasks.gitactivity
class _TranslationsMemoryWorkersTasksGitactivityEs extends TranslationsMemoryWorkersTasksGitactivityEn {
	_TranslationsMemoryWorkersTasksGitactivityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Resumidor de actividad de git';
	@override String get description => 'git log a narrativa de 2-3 párrafos cada 24h. Encaja de forma natural con un worker de agente.';
}

// Path: memoryWorkers.tasks.transcript
class _TranslationsMemoryWorkersTasksTranscriptEs extends TranslationsMemoryWorkersTasksTranscriptEn {
	_TranslationsMemoryWorkersTasksTranscriptEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Resumidor de transcript de sesión';
	@override String get description => 'Resumen al final de la sesión de \'qué hizo el agente\'. Encaja de forma natural con un worker de agente.';
}

// Path: memoryWorkers.tasks.planDrift
class _TranslationsMemoryWorkersTasksPlanDriftEs extends TranslationsMemoryWorkersTasksPlanDriftEn {
	_TranslationsMemoryWorkersTasksPlanDriftEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Detector de deriva del plan';
	@override String get description => 'Después de que termina cada sesión, comprueba si el plan del proyecto necesita actualizarse y presenta una propuesta. Encaja con un worker de agente para un razonamiento más rico.';
}

// Path: memoryWorkers.tasks.conflictDetector
class _TranslationsMemoryWorkersTasksConflictDetectorEs extends TranslationsMemoryWorkersTasksConflictDetectorEn {
	_TranslationsMemoryWorkersTasksConflictDetectorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Detector de conflictos entre capas';
	@override String get description => 'Escaneo diario que encuentra contradicciones entre hechos / plan / objetivo / journal. Un modelo de mayor calidad implica menos falsos positivos.';
}

// Path: memoryWorkers.tasks.capture
class _TranslationsMemoryWorkersTasksCaptureEs extends TranslationsMemoryWorkersTasksCaptureEn {
	_TranslationsMemoryWorkersTasksCaptureEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Motor de captura';
	@override String get description => 'Extracción de hechos por cada trigger desde las transcripciones de sesión. El modo agente da hechos notablemente mejores en sesiones largas; el modo resumidor es barato y local.';
}

// Path: project.conflicts.severity
class _TranslationsProjectConflictsSeverityEs extends TranslationsProjectConflictsSeverityEn {
	_TranslationsProjectConflictsSeverityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get low => 'baja';
	@override String get medium => 'media';
	@override String get high => 'alta';
}

// Path: backups.health.tiles
class _TranslationsBackupsHealthTilesEs extends TranslationsBackupsHealthTilesEn {
	_TranslationsBackupsHealthTilesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get recentFailures => 'Fallos recientes';
	@override String get verifyFailures => 'Verificación fallida';
	@override String get overdue => 'Atrasadas';
	@override String get schedules => 'Programaciones';
}

// Path: backupTargetEditor.kinds.local
class _TranslationsBackupTargetEditorKindsLocalEs extends TranslationsBackupTargetEditorKindsLocalEn {
	_TranslationsBackupTargetEditorKindsLocalEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Disco local';
	@override String get description => 'Carpeta en la máquina que ejecuta opendray';
}

// Path: backupTargetEditor.kinds.smb
class _TranslationsBackupTargetEditorKindsSmbEs extends TranslationsBackupTargetEditorKindsSmbEn {
	_TranslationsBackupTargetEditorKindsSmbEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Recurso compartido SMB';
	@override String get description => 'Recursos compartidos de Windows y la mayoría de los NAS domésticos';
}

// Path: backupTargetEditor.kinds.webdav
class _TranslationsBackupTargetEditorKindsWebdavEs extends TranslationsBackupTargetEditorKindsWebdavEn {
	_TranslationsBackupTargetEditorKindsWebdavEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'WebDAV';
	@override String get description => 'Nubes autoalojadas y servicios para compartir archivos';
}

// Path: backupTargetEditor.kinds.sftp
class _TranslationsBackupTargetEditorKindsSftpEs extends TranslationsBackupTargetEditorKindsSftpEn {
	_TranslationsBackupTargetEditorKindsSftpEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'SFTP';
	@override String get description => 'Cualquier servidor accesible por SSH';
}

// Path: backupTargetEditor.kinds.s3
class _TranslationsBackupTargetEditorKindsS3Es extends TranslationsBackupTargetEditorKindsS3En {
	_TranslationsBackupTargetEditorKindsS3Es._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'S3 / compatible';
	@override String get description => 'Amazon S3 y buckets compatibles con S3 (MinIO, R2, B2)';
}

// Path: backupTargetEditor.kinds.rclone
class _TranslationsBackupTargetEditorKindsRcloneEs extends TranslationsBackupTargetEditorKindsRcloneEn {
	_TranslationsBackupTargetEditorKindsRcloneEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'rclone (cualquiera)';
	@override String get description => 'OneDrive, Google Drive, Dropbox a través de la CLI de rclone';
}

// Path: githosts.form.kinds
class _TranslationsGithostsFormKindsEs extends TranslationsGithostsFormKindsEn {
	_TranslationsGithostsFormKindsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get github => 'GitHub';
	@override String get gitlab => 'GitLab';
	@override String get bitbucket => 'Bitbucket';
	@override String get gitea => 'Gitea';
	@override String get custom => 'Personalizado';
}

// Path: channels.notifications.modes
class _TranslationsChannelsNotificationsModesEs extends TranslationsChannelsNotificationsModesEn {
	_TranslationsChannelsNotificationsModesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get onceLabel => 'Una vez por session';
	@override String get onceDescription => 'Se dispara una vez al quedar inactiva, permanece en silencio hasta la respuesta o el fin.';
	@override String get cooldownLabel => 'Cooldown por ventana de tiempo';
	@override String get cooldownDescription => 'Suprime las repeticiones dentro de la ventana elegida.';
	@override String get everyLabel => 'Cada evento (ruidoso)';
	@override String get everyDescription => 'Sin supresión. Solo para canales de baja frecuencia.';
}

// Path: channels.kinds.telegram
class _TranslationsChannelsKindsTelegramEs extends TranslationsChannelsKindsTelegramEn {
	_TranslationsChannelsKindsTelegramEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get description => 'Bot mediante @BotFather. opendray hace long-polling de getUpdates y envía vía REST. Los botones y reply_to_message funcionan de forma nativa.';
	@override String get botTokenLabel => 'Token del bot';
	@override String get botTokenHint => 'De @BotFather. Se guarda en la configuración del canal; API solo para administradores.';
	@override String get chatIdLabel => 'Chat ID por defecto';
	@override String get chatIdPlaceholder => '42 (opcional, se usa cuando no hay ReplyCtx)';
	@override String get ownerUserIdsLabel => 'ID(s) de usuario de Telegram del propietario';
	@override String get ownerUserIdsPlaceholder => '123456789 (separados por comas para más de uno)';
	@override String get ownerUserIdsHint => 'Solo estos IDs numéricos de usuario de Telegram pueden controlar sessions, ejecutar comandos o pulsar botones; el resto se ignora. Déjalo en blanco para permitir a cualquiera (no recomendado para chat bidireccional). Obtén el tuyo enviando un DM a @userinfobot.';
	@override String get chatEnabledLabel => 'Chat bidireccional (enrutar mensajes a la session)';
	@override String get chatEnabledHint => 'Cuando está activado, tus mensajes se escriben en la session seleccionada y el agente responde aquí. Desactívalo solo para notificaciones.';
	@override String get chatTypingLabel => 'Mostrar “escribiendo…” mientras el agente trabaja';
	@override String get chatTypingHint => 'Muestra un indicador de escritura hasta que la respuesta del agente se asienta.';
	@override String get replyMaxCharsLabel => 'Longitud máxima de la respuesta (caracteres)';
	@override String get replyMaxCharsPlaceholder => '3500 (en blanco = 3500, 0 = sin límite)';
	@override String get replyMaxCharsHint => 'Limita cuánto de la respuesta del agente se envía antes de recortarla con una nota “…(truncado)”. En blanco usa el valor por defecto de 3500 (~un mensaje); pon 0 para enviar la respuesta completa, dividida en varios mensajes.';
}

// Path: channels.kinds.slack
class _TranslationsChannelsKindsSlackEs extends TranslationsChannelsKindsSlackEn {
	_TranslationsChannelsKindsSlackEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get description => 'Socket Mode, no necesita un webhook público. Requiere un token OAuth de bot (xoxb-) y un token a nivel de app (xapp-) con connections:write.';
	@override String get botTokenLabel => 'Token del bot (xoxb-…)';
	@override String get botTokenHint => 'OAuth & Permissions → Bot User OAuth Token. Necesita chat:write.';
	@override String get appTokenLabel => 'Token a nivel de app (xapp-…)';
	@override String get appTokenHint => 'Settings → Basic Information → App-Level Tokens. Scope: connections:write.';
	@override String get channelIdLabel => 'ID de canal por defecto';
	@override String get channelIdPlaceholder => 'C0123ABC456 (opcional)';
}

// Path: channels.kinds.discord
class _TranslationsChannelsKindsDiscordEs extends TranslationsChannelsKindsDiscordEn {
	_TranslationsChannelsKindsDiscordEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get description => 'Bot mediante el Discord Developer Portal con MESSAGE CONTENT INTENT activado. Se conecta al Gateway WS, no requiere URL pública.';
	@override String get botTokenLabel => 'Token del bot';
	@override String get botTokenPlaceholder => 'Token del bot del Discord Developer Portal';
	@override String get botTokenHint => 'Application → Bot → Reset Token. Invita al bot con send_messages + embed_links.';
	@override String get channelIdLabel => 'ID de canal por defecto';
	@override String get channelIdPlaceholder => '123456789012345678 (opcional)';
}

// Path: channels.kinds.feishu
class _TranslationsChannelsKindsFeishuEs extends TranslationsChannelsKindsFeishuEn {
	_TranslationsChannelsKindsFeishuEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get description => 'Credenciales a nivel de app. Usa el webhook de suscripción de eventos para la entrada. La URL pública del webhook se genera abajo: pégala en la consola de desarrollo de Feishu.';
	@override String get afterCreateHint => 'Abre la URL del webhook desde la tarjeta del canal y pégala en Feishu Open Platform → Event Subscriptions → Request URL.';
	@override String get appIdLabel => 'App ID';
	@override String get appSecretLabel => 'App secret';
	@override String get appSecretPlaceholder => 'Secret de la credencial de la aplicación';
	@override String get verificationTokenLabel => 'Token de verificación';
	@override String get verificationTokenHint => 'De Event Subscriptions → Verification Token. Cuando se establece, opendray rechaza los webhooks con un token diferente.';
	@override String get chatIdLabel => 'Chat ID por defecto (oc_…)';
	@override String get chatIdPlaceholder => 'oc_xxxxxxxxxx (opcional)';
}

// Path: channels.kinds.dingtalk
class _TranslationsChannelsKindsDingtalkEs extends TranslationsChannelsKindsDingtalkEn {
	_TranslationsChannelsKindsDingtalkEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get description => 'Robot de grupo personalizado. Solo saliente. Chat de grupo → Robots → Añadir → Modo de firma → copia el webhook + secret.';
	@override String get webhookUrlLabel => 'URL del webhook';
	@override String get secretLabel => 'Secret de firma';
	@override String get secretHint => 'Cuando el robot está configurado en modo de seguridad "Sign", copia el secret aquí. opendray añade los parámetros timestamp + sign automáticamente.';
}

// Path: channels.kinds.wecom
class _TranslationsChannelsKindsWecomEs extends TranslationsChannelsKindsWecomEn {
	_TranslationsChannelsKindsWecomEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get description => 'Webhook de robot de grupo. Solo saliente (texto + markdown). Configuración del grupo → Robots de grupo → Añadir → copia la URL del webhook.';
	@override String get webhookKeyLabel => 'Clave del webhook';
	@override String get webhookKeyPlaceholder => 'El valor de la consulta "key="';
	@override String get webhookKeyHint => 'O pega la URL completa del webhook en el campo de abajo: cualquiera de las dos opciones es suficiente.';
	@override String get webhookUrlLabel => 'O la URL completa del webhook';
}

// Path: dataExport.form.integrationOptions
class _TranslationsDataExportFormIntegrationOptionsEs extends TranslationsDataExportFormIntegrationOptionsEn {
	_TranslationsDataExportFormIntegrationOptionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get none => 'Omitir';
	@override String get noneHint => 'No incluir el registro de /integrations.';
	@override String get metadata => 'Solo metadatos (predeterminado)';
	@override String get metadataHint => 'Nombre y endpoint por integración, sin API keys.';
	@override String get plaintext => 'Claves en texto plano';
	@override String get plaintextHint => 'PELIGROSO: incluye los API tokens en bruto. v1 solo almacena hashes bcrypt, así que hoy esto es efectivamente una operación nula; muéstralo de todos modos.';
}

// Path: dataExport.history.columns
class _TranslationsDataExportHistoryColumnsEs extends TranslationsDataExportHistoryColumnsEn {
	_TranslationsDataExportHistoryColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get scope => 'Alcance';
	@override String get size => 'Tamaño';
	@override String get expires => 'Caduca';
	@override String get actions => 'Acciones';
}

// Path: dataExport.import.summaryCard
class _TranslationsDataExportImportSummaryCardEs extends TranslationsDataExportImportSummaryCardEn {
	_TranslationsDataExportImportSummaryCardEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get memories => 'Memorias';
	@override String get integrations => 'Integraciones';
	@override String get customTasks => 'Tareas personalizadas';
	@override String get created => 'creadas';
	@override String get skipped => 'omitidas';
	@override String get failed => 'fallidas';
}

// Path: dataExport.imports.columns
class _TranslationsDataExportImportsColumnsEs extends TranslationsDataExportImportsColumnsEn {
	_TranslationsDataExportImportsColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get status => 'Estado';
	@override String get source => 'Origen';
	@override String get counts => 'Recuentos';
	@override String get when => 'Cuándo';
}

// Path: settings.logViewer.levels
class _TranslationsSettingsLogViewerLevelsEs extends TranslationsSettingsLogViewerLevelsEn {
	_TranslationsSettingsLogViewerLevelsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get all => 'Todos';
	@override String get debug => 'Debug';
	@override String get info => 'Info';
	@override String get warn => 'Warn';
	@override String get error => 'Error';
}

// Path: settings.serverSettings.sections
class _TranslationsSettingsServerSettingsSectionsEs extends TranslationsSettingsServerSettingsSectionsEn {
	_TranslationsSettingsServerSettingsSectionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get general => 'General';
	@override String get logging => 'Registro';
	@override String get sessions => 'Sessions';
	@override String get vault => 'Vault';
	@override String get mcpRegistry => 'Registro de MCP';
	@override String get memory => 'Memoria';
	@override String get backup => 'Copia de seguridad';
	@override String get storageClaude => 'Almacenamiento · Claude';
	@override String get storageCodex => 'Almacenamiento · Codex';
	@override String get storageGemini => 'Almacenamiento · Gemini';
	@override String get storageAntigravity => 'Almacenamiento · Antigravity';
}

// Path: settings.serverSettings.sectionDescriptions
class _TranslationsSettingsServerSettingsSectionDescriptionsEs extends TranslationsSettingsServerSettingsSectionDescriptionsEn {
	_TranslationsSettingsServerSettingsSectionDescriptionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get general => 'Dirección de escucha, cuenta del operador, TTL del token.';
	@override String get logging => 'Verbosidad, formato y ruta del log en disco.';
	@override String get sessions => 'Umbrales de detección de inactividad.';
	@override String get vault => 'Notas, skills y raíz versionada con git.';
	@override String get mcpRegistry => 'Rutas del vault para servidores MCP + archivo de secretos.';
	@override String get memory => 'Subsistema de memoria persistente entre CLIs.';
	@override String get backup => 'Copias de seguridad cifradas de la BD + exportaciones de datos de admin. La frase de contraseña vive en el keyfile (Ajustes → Copias de seguridad).';
	@override String get storageClaude => 'Dónde viven los transcripts de Claude en disco.';
	@override String get storageCodex => 'Raíz de las sessions de Codex.';
	@override String get storageGemini => 'Rutas de tmp por proyecto + projects.json.';
	@override String get storageAntigravity => 'Almacén SQLite por conversación de agy.';
}

// Path: settings.serverSettings.fields
class _TranslationsSettingsServerSettingsFieldsEs extends TranslationsSettingsServerSettingsFieldsEn {
	_TranslationsSettingsServerSettingsFieldsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get listenAddress => 'Dirección de escucha';
	@override String get adminUser => 'Usuario admin';
	@override String get adminUserHelper => 'Efectivo cuando no hay keyfile ni variable de entorno configurada. Si no, consulta Ajustes → Cuenta.';
	@override String get adminPassword => 'Contraseña admin';
	@override String get adminPasswordHelper => 'Envíalo en blanco para conservarlo. Para rotaciones continuas usa Ajustes → Cuenta (respaldado por keyfile, sin reinicio).';
	@override String get tokenTtlWeb => 'TTL del token (web)';
	@override String get tokenTtlHelper => 'Cadena de duración de Go, p. ej. 24h, 30m.';
	@override String get level => 'Nivel';
	@override String get format => 'Formato';
	@override String get filePath => 'Ruta del archivo';
	@override String get filePathHelper => 'Vacío = solo stdout.';
	@override String get idleThreshold => 'Umbral de inactividad';
	@override String get idleThresholdHelper => 'Periodo de silencio antes de marcar una session como inactiva. Duración de Go.';
	@override String get idleCheckInterval => 'Intervalo de comprobación de inactividad';
	@override String get idleCheckHelper => 'Con qué frecuencia se ejecuta el reaper de inactividad.';
	@override String get root => 'Raíz';
	@override String get rootHelper => 'Padre de las sub-rutas notes / skills / git_root.';
	@override String get notesPath => 'Ruta de notas';
	@override String get skillsPath => 'Ruta de skills';
	@override String get gitRoot => 'Raíz de git';
	@override String get personalPrefix => 'Prefijo personal';
	@override String get projectsPrefix => 'Prefijo de proyectos';
	@override String get registryRoot => 'Raíz del registro';
	@override String get secretsFile => 'Archivo de secretos';
	@override String get backend => 'Backend';
	@override String get store => 'Almacén';
	@override String get defaultTopK => 'Top-k por defecto';
	@override String get similarityThreshold => 'Umbral de similitud';
	@override String get defaultScope => 'Ámbito por defecto';
	@override String get preserveHelper => 'En blanco para conservar el actual.';
	@override String get localModelName => 'Nombre del modelo local';
	@override String get localLibraryPath => 'Ruta de la biblioteca local';
	@override String get localModelPath => 'Ruta del modelo local';
	@override String get localTokenizerPath => 'Ruta del tokenizador local';
	@override String get localMaxSeqLen => 'Longitud máx. de secuencia local';
	@override String get backupEnabled => 'Habilitado';
	@override String get backupEnabledHelper => 'Aunque esto esté activado, el subsistema de copias de seguridad permanece apagado hasta que se configure OPENDRAY_BACKUP_KEY o el keyfile.';
	@override String get backupLocalDir => 'Directorio local';
	@override String get backupExportDir => 'Directorio de exportación';
	@override String get pathHelper => 'Vacío = resolver desde PATH al arrancar.';
	@override String get accountsDir => 'Directorio de cuentas';
	@override String get accountsHelper => 'Padre de los subdirectorios .claude/ por cuenta. Vacío = ~/.claude-accounts.';
	@override String get sessionsRoot => 'Raíz de sessions';
	@override String get sessionsRootHelper => 'Vacío = ~/.codex/sessions.';
	@override String get listenHelper => 'host:port al que se vincula el gateway. Requiere reinicio.';
	@override String get secretsHelper => 'Vault de secretos cifrado con AES-256-GCM.';
	@override String get backendHelper => 'auto elige el mejor disponible; local necesita ONNX.';
	@override String get similarityHelper => '0.0-1.0; los resultados por debajo de esto se filtran.';
	@override String defaultFallback({required Object value}) => 'Por defecto: ${value}';
	@override String get httpBaseUrl => 'URL base HTTP';
	@override String get httpModel => 'Modelo HTTP';
	@override String get httpApiKey => 'API key HTTP';
	@override String get httpDimensions => 'Dimensiones HTTP';
	@override String get pgDumpPath => 'Ruta de pg_dump';
	@override String get pgRestorePath => 'Ruta de pg_restore';
	@override String get tmpRoot => 'Raíz de tmp';
	@override String get projectsJson => 'projects.json';
	@override String get conversationsRoot => 'Directorio de conversaciones';
	@override String get dedupThreshold => 'Umbral de dedup';
	@override String get dedupHelper => 'Umbral de plegado al escribir; 0 = por defecto, negativo desactiva.';
	@override String get gatekeeperEnabled => 'Gatekeeper';
	@override String get gatekeeperHelper => 'Juez LLM pre-escritura para memory_store. Provider en ajustes de Cortex.';
	@override String get cleanerEnabled => 'Cleaner';
	@override String get cleanerHelper => 'Auto-bibliotecario periódico que archiva memorias obsoletas / duplicadas.';
	@override String get knowledgeEnabled => 'Grafo de conocimiento';
	@override String get knowledgeHelper => 'La capa estructurada de entidades/playbooks/skills sobre la memoria.';
}

// Path: settings.serverSettings.embedderModel
class _TranslationsSettingsServerSettingsEmbedderModelEs extends TranslationsSettingsServerSettingsEmbedderModelEn {
	_TranslationsSettingsServerSettingsEmbedderModelEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get reprobe => 'Volver a comprobar el endpoint';
	@override String get unreachable => 'Endpoint no accesible — escribe el id del modelo a mano.';
	@override String get pickHint => 'Selecciona un modelo';
	@override String get manual => 'Escribir manualmente';
	@override String get pickFromList => 'Elegir de la lista';
}

// Path: web.sessions.list.row
class _TranslationsWebSessionsListRowEs extends TranslationsWebSessionsListRowEn {
	_TranslationsWebSessionsListRowEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get deleteAria => 'Eliminar session';
	@override String get titleRemoveHistory => 'Quitar del historial';
	@override String get titleTerminate => 'Terminar y eliminar';
	@override String get titleRemove => 'Eliminar';
	@override String claudeAccountTitle({required Object label}) => 'Cuenta de Claude: ${label}';
}

// Path: web.sessions.terminal.urls
class _TranslationsWebSessionsTerminalUrlsEs extends TranslationsWebSessionsTerminalUrlsEn {
	_TranslationsWebSessionsTerminalUrlsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get tooltip => 'Abre el último enlace detectado en esta session';
	@override String get tapToOpenLatest => 'Toca para abrir el último enlace (la URL de OAuth más reciente)';
	@override String get openListTooltip => 'Mostrar todos los enlaces';
	@override String buttonLabel({required Object count}) => '${count} enlace';
	@override String buttonLabel_plural({required Object count}) => '${count} enlaces';
	@override String get dialogTitle => 'Enlaces detectados';
	@override String get dialogDesc => 'URLs impresas en la salida de esta session, las más recientes primero. Toca Abrir para lanzarla en tu navegador predeterminado. Funciona incluso cuando la URL se divide en varias líneas en el terminal.';
	@override String get openButton => 'Abrir';
	@override String get copyButton => 'Copiar';
	@override String get copiedToast => 'URL copiada';
	@override String get copyFailedToast => 'No se pudo copiar. Mantén pulsada la URL y cópiala manualmente';
	@override String get noneHint => 'Aún no se han detectado enlaces.';
}

// Path: web.sessions.inspector.tabs
class _TranslationsWebSessionsInspectorTabsEs extends TranslationsWebSessionsInspectorTabsEn {
	_TranslationsWebSessionsInspectorTabsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get files => 'Archivos';
	@override String get git => 'Git';
	@override String get search => 'Buscar';
	@override String get tasks => 'Tareas';
	@override String get history => 'Historial';
	@override String get vault => 'Bóveda';
	@override String get cortex => 'Cortex';
}

// Path: web.sessions.inspector.vaultPanel
class _TranslationsWebSessionsInspectorVaultPanelEs extends TranslationsWebSessionsInspectorVaultPanelEn {
	_TranslationsWebSessionsInspectorVaultPanelEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get open => 'Abrir Bóveda';
	@override String get projectDocs => 'Docs del proyecto';
	@override String get projectDocsHint => 'Docs del proyecto escritos por el agente en la bóveda. Revincula la carpeta si las notas de este proyecto viven en otro sitio.';
	@override String get pinnedHint => 'Vinculado a una carpeta de bóveda personalizada para este proyecto.';
	@override String get bind => 'Vincular';
	@override String get changeLocation => 'Cambiar la carpeta de bóveda vinculada a este proyecto';
	@override String get newDoc => 'Nuevo doc';
	@override String get cancel => 'Cancelar';
	@override String get create => 'Crear';
	@override String get filenamePlaceholder => 'archivo.md';
	@override String get noDocs => 'Aún no hay docs del proyecto en esta carpeta de la bóveda.';
	@override String get createFailed => 'No se pudo crear el doc';
	@override String get mappingTitle => 'Vincular carpeta de bóveda del proyecto';
	@override String get mappingHelp => 'Elige la carpeta de la bóveda que contiene las notas de este proyecto. Relativa a la bóveda, p. ej. projects/my-app. Déjalo vacío para usar el valor por defecto.';
	@override String get sessionCwd => 'cwd de la sesión';
	@override String get folderLabel => 'Carpeta de la bóveda';
	@override String get mappingStoredHint => 'Se guarda en la bóveda en .opendray-projects.json, así se sincroniza con tus notas.';
	@override String get save => 'Guardar';
	@override String get clearOverride => 'Borrar anulación';
	@override String get boundToast => 'Carpeta de bóveda del proyecto vinculada';
	@override String get clearedToast => 'Anulación borrada — usando la carpeta por defecto';
	@override String get saveFailed => 'No se pudo guardar el mapeo';
}

// Path: web.sessions.inspector.cortexPanel
class _TranslationsWebSessionsInspectorCortexPanelEs extends TranslationsWebSessionsInspectorCortexPanelEn {
	_TranslationsWebSessionsInspectorCortexPanelEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get noCwd => 'La sesión no tiene cwd — las funciones de Cortex necesitan un directorio de trabajo.';
	@override String get open => 'Abrir espacio Cortex';
	@override String get docs => 'Docs';
	@override String get journal => 'Diario';
	@override String get inbox => 'Entrada';
	@override String get archived => 'Archivados';
	@override String get pending => 'pendiente';
	@override String get goal => 'Objetivo';
	@override String get plan => 'Plan';
	@override String get latestJournal => 'Último diario';
	@override String get empty => 'Aún no se ha capturado memoria de Cortex para este proyecto. Inicia una sesión o define un objetivo para poblarla.';
}

// Path: web.memoryWorkers.tasks.gatekeeper
class _TranslationsWebMemoryWorkersTasksGatekeeperEs extends TranslationsWebMemoryWorkersTasksGatekeeperEn {
	_TranslationsWebMemoryWorkersTasksGatekeeperEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Gatekeeper';
	@override String get description => 'Filtro previo a la escritura en cada memory_store. Alta frecuencia (objetivo <500ms), solo-summarizer.';
	@override String get modelAdvice => 'Juicio sí/no de alta frecuencia — un modelo ligero (haiku / flash-lite / codex-mini / local) basta.';
}

// Path: web.memoryWorkers.tasks.cleaner
class _TranslationsWebMemoryWorkersTasksCleanerEs extends TranslationsWebMemoryWorkersTasksCleanerEn {
	_TranslationsWebMemoryWorkersTasksCleanerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Bibliotecario de limpieza';
	@override String get description => 'Bibliotecario LLM periódico. Evalúa los recuerdos antiguos como conservar / obsoleto / duplicado.';
	@override String get modelAdvice => 'Veredictos por lotes sobre hechos viejos — modelo ligero recomendado; corre programado.';
}

// Path: web.memoryWorkers.tasks.gitactivity
class _TranslationsWebMemoryWorkersTasksGitactivityEs extends TranslationsWebMemoryWorkersTasksGitactivityEn {
	_TranslationsWebMemoryWorkersTasksGitactivityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Resumidor de actividad de git';
	@override String get description => 'git log → narrativa de 2-3 párrafos cada 24h. Encaja de forma natural con un worker de agente.';
	@override String get modelAdvice => 'Resumen narrativo del historial git — un modelo equilibrado (sonnet / flash) se lee mejor.';
}

// Path: web.memoryWorkers.tasks.transcript
class _TranslationsWebMemoryWorkersTasksTranscriptEs extends TranslationsWebMemoryWorkersTasksTranscriptEn {
	_TranslationsWebMemoryWorkersTasksTranscriptEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Resumidor de transcript de sesión';
	@override String get description => 'Resumen al final de la sesión sobre "qué hizo el agente". Encaja de forma natural con un worker de agente.';
	@override String get modelAdvice => 'Resúmenes de sesión — modelo equilibrado recomendado; alimenta el diario y la detección de deriva.';
}

// Path: web.memoryWorkers.tasks.plan_drift
class _TranslationsWebMemoryWorkersTasksPlanDriftEs extends TranslationsWebMemoryWorkersTasksPlanDriftEn {
	_TranslationsWebMemoryWorkersTasksPlanDriftEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Detector de desviación del plan';
	@override String get description => 'Al terminar cada sesión, comprueba si el plan del proyecto necesita actualizarse y presenta una propuesta. Encaja con un worker de agente para un razonamiento más completo.';
	@override String get modelAdvice => 'Reescribe goal/plan/secciones — exige criterio; un modelo fuerte (sonnet/opus) evita malas actualizaciones.';
}

// Path: web.memoryWorkers.tasks.conflict_detector
class _TranslationsWebMemoryWorkersTasksConflictDetectorEs extends TranslationsWebMemoryWorkersTasksConflictDetectorEn {
	_TranslationsWebMemoryWorkersTasksConflictDetectorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Detector de conflictos entre capas';
	@override String get description => 'Escaneo diario que encuentra contradicciones entre hechos / plan / objetivo / diario. Un modelo de mayor calidad = menos falsos positivos.';
	@override String get modelAdvice => 'Escaneo diario de contradicciones — un modelo equilibrado basta.';
}

// Path: web.memoryWorkers.tasks.capture
class _TranslationsWebMemoryWorkersTasksCaptureEs extends TranslationsWebMemoryWorkersTasksCaptureEn {
	_TranslationsWebMemoryWorkersTasksCaptureEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Motor de captura';
	@override String get description => 'Extracción de hechos por cada trigger a partir de los transcripts de sesión. El modo agente ofrece hechos notablemente mejores en sesiones largas; el modo summarizer es barato y local.';
	@override String get modelAdvice => 'La tarea más frecuente: extracción de hechos — usa el modelo MÁS BARATO que funcione (haiku / local).';
}

// Path: web.memoryWorkers.tasks.blueprint
class _TranslationsWebMemoryWorkersTasksBlueprintEs extends TranslationsWebMemoryWorkersTasksBlueprintEn {
	_TranslationsWebMemoryWorkersTasksBlueprintEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get modelAdvice => 'Clasificación ocasional del proyecto — modelo equilibrado; aquí la calidad importa más que el costo.';
	@override String get label => 'Proponedor de planos';
	@override String get description => 'Clasifica un proyecto y propone su conjunto de secciones. Disparado por el operador.';
}

// Path: web.memoryWorkers.tasks.curation
class _TranslationsWebMemoryWorkersTasksCurationEs extends TranslationsWebMemoryWorkersTasksCurationEn {
	_TranslationsWebMemoryWorkersTasksCurationEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get modelAdvice => 'Tu editor conversacional de docs/políticas — modelo fuerte recomendado (sonnet/opus).';
	@override String get label => 'Chat de curación';
	@override String get description => 'Impulsa el canal conversacional que actualiza secciones y re-redacta páginas de conocimiento.';
}

// Path: web.project.readonly.tech_stack
class _TranslationsWebProjectReadonlyTechStackEs extends TranslationsWebProjectReadonlyTechStackEn {
	_TranslationsWebProjectReadonlyTechStackEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Stack tecnológico y estructura';
	@override String get empty => 'Ejecuta una session de Claude en este proyecto. El escáner se actualiza en cada inicio.';
}

// Path: web.project.readonly.recent_activity
class _TranslationsWebProjectReadonlyRecentActivityEs extends TranslationsWebProjectReadonlyRecentActivityEn {
	_TranslationsWebProjectReadonlyRecentActivityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Actividad reciente (git → LLM)';
	@override String get empty => 'El resumidor de actividad de git se ejecuta cada 24 h; vuelve a comprobarlo tras el siguiente ciclo del planificador.';
}

// Path: web.project.reset.summary
class _TranslationsWebProjectResetSummaryEs extends TranslationsWebProjectResetSummaryEn {
	_TranslationsWebProjectResetSummaryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String docs_one({required Object count}) => '${count} documento';
	@override String docs_other({required Object count}) => '${count} documentos';
	@override String journal({required Object count}) => '${count} diario';
	@override String proposals_one({required Object count}) => '${count} propuesta';
	@override String proposals_other({required Object count}) => '${count} propuestas';
	@override String cleanup({required Object count}) => '${count} limpieza';
	@override String memories({required Object count}) => '${count} memorias';
}

// Path: web.project.lifecycle.status
class _TranslationsWebProjectLifecycleStatusEs extends TranslationsWebProjectLifecycleStatusEn {
	_TranslationsWebProjectLifecycleStatusEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get active => 'Activo';
	@override String get paused => 'Pausado';
	@override String get archived => 'Archivado';
}

// Path: web.project.lifecycle.applied
class _TranslationsWebProjectLifecycleAppliedEs extends TranslationsWebProjectLifecycleAppliedEn {
	_TranslationsWebProjectLifecycleAppliedEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get active => 'Proyecto reactivado';
	@override String get paused => 'Proyecto pausado';
	@override String get archived => 'Proyecto archivado';
}

// Path: web.project.lifecycle.tooltip
class _TranslationsWebProjectLifecycleTooltipEs extends TranslationsWebProjectLifecycleTooltipEn {
	_TranslationsWebProjectLifecycleTooltipEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get badge => 'Ciclo de vida del proyecto. Los proyectos congelados (pausados/archivados) se excluyen de la inyección en nuevas sesiones y de la destilación por IA.';
	@override String get activate => 'Reactivar: inyectar en nuevas sesiones y reanudar el mantenimiento por IA.';
	@override String get pause => 'Pausar: congelar este proyecto — omitir inyección y destilación, pero mantenerlo en la lista activa.';
	@override String get archive => 'Archivar: archivar este proyecto — congelado y oculto de las vistas habituales.';
}

// Path: web.project.docMeta.maintainer
class _TranslationsWebProjectDocMetaMaintainerEs extends TranslationsWebProjectDocMetaMaintainerEn {
	_TranslationsWebProjectDocMetaMaintainerEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get coauthored => 'Tú mantienes · IA propone';
	@override String get auto => 'Autogenerado · solo lectura';
	@override String get human => 'Autoría humana';
}

// Path: web.project.docMeta.purpose
class _TranslationsWebProjectDocMetaPurposeEs extends TranslationsWebProjectDocMetaPurposeEn {
	_TranslationsWebProjectDocMetaPurposeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get goal => 'La intención a largo plazo del proyecto: qué construimos y por qué. Cuando una sesión cambia el rumbo, la IA propone una actualización en tu Bandeja para que la apruebes.';
	@override String get plan => 'La hoja de ruta actual / trabajo en curso. La IA propone una actualización en tu Bandeja tras avanzar una sesión; tú la apruebas.';
	@override String get tech_stack => 'Stack y estructura, autogenerado por el escáner del proyecto (se actualiza cada 6 h).';
	@override String get recent_activity => 'Resumen por IA de la actividad reciente de Git, actualizado automáticamente (cada 12 h).';
	@override String get overview => 'El documento oficial del proyecto: qué es, sus funciones, arquitectura, cómo construir/ejecutar y las bases en que se apoya. Redactado por IA desde las señales del propio proyecto; puedes editarlo (lo bloquea) o regenerarlo.';
}

// Path: web.memoryInspector.scope.values
class _TranslationsWebMemoryInspectorScopeValuesEs extends TranslationsWebMemoryInspectorScopeValuesEn {
	_TranslationsWebMemoryInspectorScopeValuesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get project => 'proyecto';
	@override String get global => 'global';
}

// Path: web.notes.vaultSync.init
class _TranslationsWebNotesVaultSyncInitEs extends TranslationsWebNotesVaultSyncInitEn {
	_TranslationsWebNotesVaultSyncInitEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'El vault aún no es un repo git';
	@override String get body => 'Al inicializarlo se ejecutará <1>git init -b main</1> en la raíz de tu vault y se añadirá un <3>.gitignore</3> sensato. Después podrás hacer commit de tus notas y configurar un remoto (GitHub / Gitea / GitLab) para la sincronización entre máquinas.';
	@override String get button => 'Inicializar el vault como repo git';
	@override String get initToast => 'Vault inicializado como repo git';
	@override String get initFailedToast => 'Error al inicializar';
}

// Path: web.notes.vaultSync.branch
class _TranslationsWebNotesVaultSyncBranchEs extends TranslationsWebNotesVaultSyncBranchEn {
	_TranslationsWebNotesVaultSyncBranchEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get clean => 'limpio';
	@override String staged({required Object count}) => '${count} en stage';
	@override String modified({required Object count}) => '${count} modificados';
	@override String untracked({required Object count}) => '${count} sin seguimiento';
}

// Path: web.notes.vaultSync.action
class _TranslationsWebNotesVaultSyncActionEs extends TranslationsWebNotesVaultSyncActionEn {
	_TranslationsWebNotesVaultSyncActionEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get pull => 'Pull';
	@override String get push => 'Push';
	@override String get pullTitleNoRemote => 'Configura primero un remoto';
	@override String get pullTitleHasUpstream => 'git pull --rebase --autostash';
	@override String get pullTitleNoUpstream => 'Hace pull del HEAD de origin; configura el seguimiento de forma implícita';
	@override String get pushTitleNoRemote => 'Configura primero un remoto';
	@override String get pushTitleHasUpstream => 'git push -u origin HEAD';
	@override String get pushTitleNoUpstream => 'Primer push: configurará el upstream a origin/HEAD';
	@override String get noRemote => 'Sin remoto configurado: pull/push deshabilitados';
	@override String get noUpstream => 'Aún no hay seguimiento de upstream: el primer push lo configurará.';
	@override String get pulledToast => 'Pull realizado';
	@override String get pullFailedToast => 'Error en el pull';
	@override String get pushedToast => 'Push realizado';
	@override String get pushFailedToast => 'Error en el push';
}

// Path: web.notes.vaultSync.commit
class _TranslationsWebNotesVaultSyncCommitEs extends TranslationsWebNotesVaultSyncCommitEn {
	_TranslationsWebNotesVaultSyncCommitEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Commit';
	@override String placeholder({required Object date}) => 'Notas: ${date} (predeterminado)';
	@override String get commitAll => 'Hacer commit de todo';
	@override String get hint => 'Pone en stage todos los cambios (<1>git add .</1>) y luego hace commit con este mensaje. Un mensaje vacío usa de forma predeterminada un asunto con marca de tiempo.';
	@override String committedToast({required Object hash}) => 'Commit ${hash} realizado';
	@override String get commitFailedToast => 'Error en el commit';
}

// Path: web.notes.vaultSync.fileList
class _TranslationsWebNotesVaultSyncFileListEs extends TranslationsWebNotesVaultSyncFileListEn {
	_TranslationsWebNotesVaultSyncFileListEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object count}) => 'Árbol de trabajo · ${count}';
	@override String moreSuffix({required Object count}) => '+${count} más';
}

// Path: web.notes.vaultSync.remote
class _TranslationsWebNotesVaultSyncRemoteEs extends TranslationsWebNotesVaultSyncRemoteEn {
	_TranslationsWebNotesVaultSyncRemoteEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Remoto (origin)';
	@override String get cancel => 'Cancelar';
	@override String get change => 'Cambiar';
	@override String get configure => 'Configurar';
	@override String get empty => 'No hay remoto configurado. Añade una URL HTTPS o SSH (p. ej. <1>git@github.com:you/notes.git</1> o <3>https://gitea.example.com/you/notes.git</3>) para habilitar push / pull.';
	@override String get urlLabel => 'URL (HTTPS o SSH)';
	@override String get urlPlaceholder => 'git@host:owner/notes.git';
	@override String get save => 'Guardar';
	@override String get savedToast => 'Remoto guardado';
	@override String get saveFailedToast => 'Error al configurar el remoto';
}

// Path: web.notes.vaultSync.history
class _TranslationsWebNotesVaultSyncHistoryEs extends TranslationsWebNotesVaultSyncHistoryEn {
	_TranslationsWebNotesVaultSyncHistoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Commits recientes';
	@override String get loading => 'Cargando…';
	@override String get empty => 'Aún no hay commits.';
}

// Path: web.notes.vaultSync.conflict
class _TranslationsWebNotesVaultSyncConflictEs extends TranslationsWebNotesVaultSyncConflictEn {
	_TranslationsWebNotesVaultSyncConflictEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebNotesVaultSyncConflictKindsEs kinds = _TranslationsWebNotesVaultSyncConflictKindsEs._(_root);
	@override String headline({required Object kind}) => 'El vault tiene un ${kind} en pausa con conflictos sin resolver';
	@override String explainer({required Object kind}) => 'Pull, push y commit están bloqueados hasta que termine el ${kind}. Puedes hacer <1>abort</1> (restaurar el árbol de trabajo a su estado anterior al ${kind}, conserva tus commits locales y descarta los remotos) o <3>forzar reset al remoto</3> (descartar TODOS los commits locales y los cambios sin confirmar; el vault se convierte en una copia exacta de origin).';
	@override String conflictedHeader({required Object count}) => 'Archivos en conflicto · ${count}';
	@override String abort({required Object kind}) => 'Abortar ${kind}';
	@override String abortTitle({required Object kind}) => 'git ${kind} --abort';
	@override String get forceReset => 'Forzar reset al remoto';
	@override String get forceResetTitle => 'git fetch && git reset --hard origin/<branch> && git clean -fd';
	@override String forceResetConfirm({required Object kind}) => 'DESTRUCTIVO: esto va a\n  • abortar el ${kind} en curso\n  • ejecutar git fetch origin\n  • hacer reset --hard a origin/<branch>\n  • ejecutar clean -fd (descartar archivos sin seguimiento)\n\nCualquier commit local no enviado a origin Y cualquier edición sin confirmar se PERDERÁ DE FORMA PERMANENTE.\n\n¿Continuar?';
	@override String abortedToast({required Object kind}) => '${kind} abortado';
	@override String get abortedDescription => 'Árbol de trabajo restaurado al estado anterior a la operación.';
	@override String get abortFailedToast => 'Error al abortar';
	@override String resetToast({required Object branch}) => 'Reset a ${branch}';
	@override String get resetDescription => 'Cambios locales descartados; el vault coincide con el remoto.';
	@override String get resetFailedToast => 'Error en el reset';
}

// Path: web.notes.vaultSync.auth
class _TranslationsWebNotesVaultSyncAuthEs extends TranslationsWebNotesVaultSyncAuthEn {
	_TranslationsWebNotesVaultSyncAuthEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Autenticación';
	@override String httpsTokenOk({required Object host}) => 'Usará el token guardado para <1>${host}</1> en Plugins → Hosts de git. ✓';
	@override String httpsTokenMissing({required Object host}) => 'Remoto HTTPS en <1>${host}</1> sin ningún token de opendray configurado. Es probable que push / pull fallen en repos privados hasta que añadas uno.';
	@override String ssh({required Object host}) => 'Remoto SSH en <1>${host}</1>. La autenticación usa el <3>~/.ssh/</3> del host del gateway (ssh-agent, archivo de identidad, configuración de host). Verifícalo con <5>ssh -T git@${host}</5> desde la shell del host.';
	@override String get configureTokenLink => '→ Configurar token de host de git';
}

// Path: web.notes.vaultSync.autoSync
class _TranslationsWebNotesVaultSyncAutoSyncEs extends TranslationsWebNotesVaultSyncAutoSyncEn {
	_TranslationsWebNotesVaultSyncAutoSyncEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get loading => 'Cargando ajustes de sincronización automática…';
	@override String get title => 'Sincronización automática';
	@override String get on => 'activada';
	@override String get runNow => 'Ejecutar ahora';
	@override String get runNowTooltip => 'Despierta el bucle de sincronización ahora (omite la espera y luego ejecuta los pasos pendientes)';
	@override String get configure => 'Configurar';
	@override String get hide => 'Ocultar';
	@override String get enabled => 'Habilitada';
	@override String get enabledTooltipNoRemote => 'Configura primero un remoto para habilitar la sincronización automática';
	@override String get noRemoteHint => 'Sin remoto: se omitirán push/pull.';
	@override String get commitEvery => 'Hacer commit cada';
	@override String get commitEveryExamples => 'Ejemplos: <1>30s</1>, <3>10m</3>, <5>2h</5>. Mínimo 30s.';
	@override String get pullEvery => 'Hacer pull cada';
	@override String get pullEveryHint => 'Solo se usa cuando Pull está habilitado.';
	@override String get pushAfterCommit => 'Hacer push tras el commit';
	@override String get pullPeriodically => 'Hacer pull periódicamente';
	@override String get commitTemplateLabel => 'Plantilla del mensaje de commit';
	@override String commitTemplatePlaceholder({required Object date}) => 'Sincronización automática: ${date}  (predeterminado si está vacío)';
	@override String get saveSettings => 'Guardar ajustes';
	@override String get discard => 'Descartar';
	@override String get lastCommit => 'último commit';
	@override String get lastPush => 'último push';
	@override String get lastPull => 'último pull';
	@override String get never => 'nunca';
	@override String get savedToast => 'Ajustes de sincronización automática guardados';
	@override String get saveFailedToast => 'Error al guardar';
	@override String get triggeredToast => 'Sincronización automática iniciada';
	@override String get runFailedToast => 'Error en la ejecución';
}

// Path: web.providers.detail.caps
class _TranslationsWebProvidersDetailCapsEs extends TranslationsWebProvidersDetailCapsEn {
	_TranslationsWebProvidersDetailCapsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get resume => 'resume';
	@override String get stream => 'stream';
	@override String get images => 'images';
	@override String get mcp => 'mcp';
}

// Path: web.channels.notifications.modes
class _TranslationsWebChannelsNotificationsModesEs extends TranslationsWebChannelsNotificationsModesEn {
	_TranslationsWebChannelsNotificationsModesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get onceLabel => 'Una vez por session (recomendado)';
	@override String get onceHint => 'Se dispara una vez cuando una session queda inactiva, luego permanece en silencio hasta que la session termine o respondas por este canal.';
	@override String get cooldownLabel => 'Cooldown por ventana de tiempo';
	@override String get cooldownHint => 'Suprime las repeticiones del mismo par (session, evento) dentro de la ventana elegida.';
	@override String get everyLabel => 'Cada evento (ruidoso)';
	@override String get everyHint => 'Sin supresión. Úsalo solo para canales de baja frecuencia.';
}

// Path: web.channels.notifications.cooldowns
class _TranslationsWebChannelsNotificationsCooldownsEs extends TranslationsWebChannelsNotificationsCooldownsEn {
	_TranslationsWebChannelsNotificationsCooldownsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get k60 => '1 minuto';
	@override String get k300 => '5 minutos';
	@override String get k900 => '15 minutos';
	@override String get k1800 => '30 minutos';
	@override String get k3600 => '1 hora';
}

// Path: web.channels.notifications.snippetCaps
class _TranslationsWebChannelsNotificationsSnippetCapsEs extends TranslationsWebChannelsNotificationsSnippetCapsEn {
	_TranslationsWebChannelsNotificationsSnippetCapsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get k0 => 'Sin límite, dividir en varios mensajes (predeterminado)';
	@override String get k1000 => '1000 caracteres (conciso)';
	@override String get k3000 => '3000 caracteres';
	@override String get k6000 => '6000 caracteres';
	@override String get k12000 => '12000 caracteres';
}

// Path: web.plugins.mcp.columns
class _TranslationsWebPluginsMcpColumnsEs extends TranslationsWebPluginsMcpColumnsEn {
	_TranslationsWebPluginsMcpColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get name => 'Nombre';
	@override String get transport => 'Transport';
	@override String get spec => 'Spec';
	@override String get enabled => 'Habilitado';
}

// Path: web.plugins.mcp.editor
class _TranslationsWebPluginsMcpEditorEs extends TranslationsWebPluginsMcpEditorEn {
	_TranslationsWebPluginsMcpEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get createTitle => 'Nuevo servidor MCP';
	@override String editTitle({required Object id}) => 'Editar MCP: ${id}';
	@override String description({required Object API_KEY}) => 'Forma del JSON: <1>command</1>+<3>args</3>+<5>env</5> para stdio (predeterminado), o <7>transport</7> +<9> url</9>+<11>headers</11> para sse / http. Referencia los secretos como <13>\$${API_KEY}</13>, se sustituyen en el momento del spawn desde el archivo de secretos.';
	@override String get idLabel => 'ID';
	@override String get idPlaceholder => 'filesystem';
	@override String get idHint => 'Minúsculas / dígitos / guion / guion bajo. Se convierte tanto en el nombre del directorio como en el <1>name</1> predeterminado.';
	@override String get bodyLabel => 'mcp.json';
	@override String invalidJson({required Object error}) => 'JSON no válido: ${error}';
	@override String get createdToast => 'Servidor MCP creado';
	@override String get savedToast => 'Servidor MCP guardado';
	@override String get createFailedToast => 'Error al crear';
	@override String get saveFailedToast => 'Error al guardar';
	@override String get transportLabel => 'Transport';
	@override String get transportHint => 'Cambiar el transport reemplaza la plantilla JSON por una forma inicial adecuada para el nuevo transport.';
	@override String get transportStdio => 'stdio (subproceso local)';
	@override String get transportSse => 'sse (servidor remoto)';
	@override String get transportHttp => 'http (servidor remoto)';
}

// Path: web.plugins.mcp.test
class _TranslationsWebPluginsMcpTestEs extends TranslationsWebPluginsMcpTestEn {
	_TranslationsWebPluginsMcpTestEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get button => 'Probar';
	@override String get title => 'Validar este servidor MCP desde el daemon';
	@override String connected({required Object count}) => 'conectado · ${count} herramientas';
	@override String get reachable => 'accesible';
	@override String get failed => 'prueba fallida';
}

// Path: web.plugins.mcpSecrets.columns
class _TranslationsWebPluginsMcpSecretsColumnsEs extends TranslationsWebPluginsMcpSecretsColumnsEn {
	_TranslationsWebPluginsMcpSecretsColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get key => 'Clave';
	@override String get value => 'Valor';
}

// Path: web.plugins.mcpSecrets.editor
class _TranslationsWebPluginsMcpSecretsEditorEs extends TranslationsWebPluginsMcpSecretsEditorEn {
	_TranslationsWebPluginsMcpSecretsEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get addTitle => 'Añadir secreto';
	@override String updateTitle({required Object key}) => 'Actualizar ${key}';
	@override String addDescription({required Object KEY}) => 'Se almacena cifrado en disco si el llavero del SO está disponible. Referéncialo desde el env / headers / args / url de cualquier mcp.json con \$${KEY}.';
	@override String get editDescription => 'Introduce el nuevo valor para sobrescribir. El valor anterior no se puede recuperar.';
	@override String get keyLabel => 'Clave';
	@override String get keyPlaceholder => 'BRAVE_API_KEY';
	@override String get keyPattern => 'Debe coincidir con <1>[A-Za-z_][A-Za-z0-9_]*</1>';
	@override String get keyCollision => 'Ya existe. Usa Editar en su lugar, o elige un nombre diferente.';
	@override String get valueLabel => 'Valor';
	@override String get valueHint => 'Oculto mientras escribes. El valor guardado nunca se devuelve a través de la API.';
	@override String get addedToast => 'Secreto añadido';
	@override String get updatedToast => 'Secreto actualizado';
	@override String get saveFailedToast => 'Error al guardar';
}

// Path: web.plugins.skills.columns
class _TranslationsWebPluginsSkillsColumnsEs extends TranslationsWebPluginsSkillsColumnsEn {
	_TranslationsWebPluginsSkillsColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get description => 'Descripción';
	@override String get source => 'Origen';
}

// Path: web.plugins.skills.editor
class _TranslationsWebPluginsSkillsEditorEs extends TranslationsWebPluginsSkillsEditorEn {
	_TranslationsWebPluginsSkillsEditorEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get createTitle => 'Nueva habilidad';
	@override String customizeTitle({required Object id}) => 'Personalizar la integrada: ${id}';
	@override String editTitle({required Object id}) => 'Editar habilidad: ${id}';
	@override String get customizeDescription => 'Estás viendo una habilidad integrada incorporada en opendray. Al guardar se creará una anulación del vault con el mismo id, tus ediciones se guardan en ~/.opendray/vault/skills/<id>/SKILL.md y ocultan la integrada hasta que la Restablezcas.';
	@override String get editDescription => 'Formato SKILL.md: frontmatter con name + description, luego instrucciones en markdown. La descripción aparece en el índice de Tier 1 del agente.';
	@override String get idLabel => 'ID';
	@override String get idPlaceholder => 'my-helper';
	@override String get idHint => 'Minúsculas / dígitos / guion / guion bajo. Se convierte en el nombre del directorio bajo <1>~/.opendray/vault/skills/&lt;id&gt;/</1>.';
	@override String get bodyLabel => 'SKILL.md';
	@override String get createdToast => 'Habilidad creada';
	@override String get savedToast => 'Habilidad guardada';
	@override String get savedOverrideToast => 'Guardada como anulación del vault';
	@override String get createFailedToast => 'Error al crear';
	@override String get saveFailedToast => 'Error al guardar';
	@override String get saveAsOverride => 'Guardar como anulación del vault';
}

// Path: web.plugins.customTasks.columns
class _TranslationsWebPluginsCustomTasksColumnsEs extends TranslationsWebPluginsCustomTasksColumnsEn {
	_TranslationsWebPluginsCustomTasksColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get name => 'Nombre';
	@override String get command => 'Comando';
	@override String get scope => 'Ámbito';
}

// Path: web.plugins.customTasks.dialog
class _TranslationsWebPluginsCustomTasksDialogEs extends TranslationsWebPluginsCustomTasksDialogEn {
	_TranslationsWebPluginsCustomTasksDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get addTitle => 'Añadir tarea personalizada';
	@override String editTitle({required Object name}) => 'Editar ${name}';
	@override String get description => 'El comando se envía textualmente a la terminal de la session. Es lo mismo que escribirlo en el prompt y pulsar Enter.';
	@override String get nameLabel => 'Nombre';
	@override String get namePlaceholder => 'dev';
	@override String get commandLabel => 'Comando';
	@override String get commandPlaceholder => 'docker compose up --build';
	@override String get descLabel => 'Descripción (opcional)';
	@override String get descPlaceholder => 'Arranca la infraestructura de desarrollo y sigue los logs';
	@override String get cwdLabel => 'Ámbito de cwd (opcional)';
	@override String get cwdPlaceholder => '/Users/me/projects/foo  (en blanco = global)';
	@override String get cwdHint => 'En blanco = visible en todas las sessions. De lo contrario, la tarea solo se muestra cuando el cwd de la session coincide con esta ruta absoluta.';
	@override String get addedToast => 'Tarea añadida';
	@override String get updatedToast => 'Tarea actualizada';
	@override String get addFailedToast => 'Error al añadir';
	@override String get updateFailedToast => 'Error al actualizar';
}

// Path: web.plugins.gitHosts.columns
class _TranslationsWebPluginsGitHostsColumnsEs extends TranslationsWebPluginsGitHostsColumnsEn {
	_TranslationsWebPluginsGitHostsColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get host => 'Host';
	@override String get kind => 'Tipo';
	@override String get token => 'Token';
	@override String get enabled => 'Habilitado';
}

// Path: web.plugins.gitHosts.dialog
class _TranslationsWebPluginsGitHostsDialogEs extends TranslationsWebPluginsGitHostsDialogEn {
	_TranslationsWebPluginsGitHostsDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get addTitle => 'Añadir host de git';
	@override String editTitle({required Object host}) => 'Editar ${host}';
	@override String get description => 'El token se almacena en el gateway. Se usa solo para llamadas de solo lectura a la API (listar PR, etc.).';
	@override String get kindLabel => 'Tipo';
	@override String get kindGitHub => 'GitHub';
	@override String get kindGitea => 'Gitea';
	@override String get kindGitLab => 'GitLab';
	@override String get hostLabel => 'Host';
	@override String get hostPlaceholder => 'github.com';
	@override String get displayNameLabel => 'Nombre visible (opcional)';
	@override String get displayNamePlaceholder => 'Personal';
	@override String get tokenLabel => 'Token';
	@override String get newTokenLabel => 'Nuevo token (déjalo en blanco para conservarlo)';
	@override String get tokenPlaceholder => 'ghp_… / gho_… / glpat-…';
	@override String get tokenPlaceholderEdit => '…';
	@override String get tokenHint => 'GitHub: PAT con scope <1>repo</1>. Gitea: token con <3>read:repository</3>. GitLab: PAT con <5>read_api</5>.';
	@override String get enabledLabel => 'Habilitado';
	@override String get addedToast => 'Host de git añadido';
	@override String get updatedToast => 'Host de git actualizado';
	@override String get addFailedToast => 'Error al añadir';
	@override String get updateFailedToast => 'Error al actualizar';
}

// Path: web.backups.backupsTab.columns
class _TranslationsWebBackupsBackupsTabColumnsEs extends TranslationsWebBackupsBackupsTabColumnsEn {
	_TranslationsWebBackupsBackupsTabColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get type => 'Tipo';
	@override String get target => 'Destino';
	@override String get status => 'Estado';
	@override String get started => 'Iniciada';
	@override String get size => 'Tamaño';
	@override String get actions => 'Acciones';
}

// Path: web.backups.health.tiles
class _TranslationsWebBackupsHealthTilesEs extends TranslationsWebBackupsHealthTilesEn {
	_TranslationsWebBackupsHealthTilesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get recentFailures => 'Fallos recientes';
	@override String get verifyFailures => 'Verificación fallida';
	@override String get overdue => 'Atrasadas';
	@override String get schedules => 'Programaciones';
}

// Path: web.backups.schedulesTab.columns
class _TranslationsWebBackupsSchedulesTabColumnsEs extends TranslationsWebBackupsSchedulesTabColumnsEn {
	_TranslationsWebBackupsSchedulesTabColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get target => 'Destino';
	@override String get interval => 'Intervalo';
	@override String get keep => 'Conservar';
	@override String get nextRun => 'Próxima ejecución';
	@override String get enabled => 'Habilitada';
	@override String get actions => 'Acciones';
}

// Path: web.backups.targetsTab.columns
class _TranslationsWebBackupsTargetsTabColumnsEs extends TranslationsWebBackupsTargetsTabColumnsEn {
	_TranslationsWebBackupsTargetsTabColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get kind => 'Tipo';
	@override String get config => 'Config';
	@override String get enabled => 'Habilitado';
	@override String get actions => 'Acciones';
}

// Path: web.backups.targetEditor.local
class _TranslationsWebBackupsTargetEditorLocalEs extends TranslationsWebBackupsTargetEditorLocalEn {
	_TranslationsWebBackupsTargetEditorLocalEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get rootLabel => 'Directorio raíz';
	@override String get rootHint => 'Vacío = cfg.backup.local_dir (~/.opendray/backups)';
	@override String get rootPlaceholder => '~/backups/opendray  o  /mnt/external-hdd/opendray';
}

// Path: web.backups.targetEditor.smb
class _TranslationsWebBackupsTargetEditorSmbEs extends TranslationsWebBackupsTargetEditorSmbEn {
	_TranslationsWebBackupsTargetEditorSmbEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get hostLabel => 'Host';
	@override String get hostPlaceholder => '192.168.1.20';
	@override String get portLabel => 'Puerto';
	@override String get shareLabel => 'Recurso compartido';
	@override String get shareHint => 'Nombre del recurso compartido de nivel superior en el servidor SMB';
	@override String get sharePlaceholder => 'Claude_Workspace';
	@override String get userLabel => 'Usuario';
	@override String get passwordLabel => 'Contraseña';
	@override String get pathPrefixLabel => 'Prefijo de ruta';
	@override String get pathPrefixHint => 'Subcarpeta bajo la raíz del recurso compartido (opcional)';
	@override String get pathPrefixPlaceholder => 'opendray/backups';
}

// Path: web.backups.targetEditor.s3
class _TranslationsWebBackupsTargetEditorS3Es extends TranslationsWebBackupsTargetEditorS3En {
	_TranslationsWebBackupsTargetEditorS3Es._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get endpointLabel => 'Endpoint';
	@override String get endpointHint => 'Host (sin protocolo). AWS: s3.amazonaws.com · R2: <accountid>.r2.cloudflarestorage.com · MinIO: minio.local:9000';
	@override String get endpointPlaceholder => 's3.amazonaws.com';
	@override String get regionLabel => 'Región';
	@override String get regionHint => 'Solo AWS; en R2 usa \'auto\'';
	@override String get regionPlaceholder => 'us-east-1 / auto';
	@override String get bucketLabel => 'Bucket';
	@override String get bucketPlaceholder => 'opendray-backups';
	@override String get accessKeyLabel => 'Clave de acceso';
	@override String get secretKeyLabel => 'Clave secreta';
	@override String get secretKeyHint => 'Se almacena cifrada con AES-256-GCM; nunca se devuelve';
	@override String get pathPrefixLabel => 'Prefijo de ruta';
	@override String get pathPrefixHint => 'Prefijo de clave de objeto (opcional)';
	@override String get pathPrefixPlaceholder => 'opendray/backups';
	@override String get useHttps => 'Usar HTTPS';
	@override String get pathStyle => 'Direccionamiento de tipo ruta (heredado / MinIO)';
}

// Path: web.backups.targetEditor.webdav
class _TranslationsWebBackupsTargetEditorWebdavEs extends TranslationsWebBackupsTargetEditorWebdavEn {
	_TranslationsWebBackupsTargetEditorWebdavEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get baseUrlLabel => 'URL base';
	@override String get baseUrlHint => 'URL completa incluyendo cualquier ruta. Ejemplos: https://cloud.example.com/remote.php/dav/files/me/ (Nextcloud), https://nas.local:5006/ (Synology), https://dav.jianguoyun.com/dav/ (Jianguoyun / 坚果云)';
	@override String get baseUrlPlaceholder => 'https://cloud.example.com/remote.php/dav/files/<user>/';
	@override String get userLabel => 'Usuario';
	@override String get passwordLabel => 'Contraseña';
	@override String get pathPrefixLabel => 'Prefijo de ruta';
	@override String get pathPrefixHint => 'Subcarpeta bajo la URL base (opcional)';
	@override String get pathPrefixPlaceholder => 'opendray/backups';
}

// Path: web.backups.targetEditor.sftp
class _TranslationsWebBackupsTargetEditorSftpEs extends TranslationsWebBackupsTargetEditorSftpEn {
	_TranslationsWebBackupsTargetEditorSftpEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get hostLabel => 'Host';
	@override String get hostPlaceholder => 'vps.example.com';
	@override String get portLabel => 'Puerto';
	@override String get userLabel => 'Usuario';
	@override String get passwordLabel => 'Contraseña';
	@override String get passwordHint => 'Se requiere contraseña O clave privada. Si se indican ambas, la contraseña se trata como la frase de contraseña de la clave.';
	@override String get privateKeyLabel => 'Clave privada (PEM)';
	@override String get privateKeyHint => 'Pega el contenido de una clave privada OpenSSH/PEM (p. ej. ~/.ssh/id_ed25519). Déjalo en blanco para autenticación solo con contraseña.';
	@override String get privateKeyPlaceholder => '-----BEGIN OPENSSH PRIVATE KEY-----...';
	@override String get hostKeyLabel => 'Clave de host (fijación)';
	@override String get hostKeyHint => 'Clave pública del servidor en formato OpenSSH (ejecuta `ssh-keyscan host` para obtenerla). Déjalo en blanco para desactivar la fijación (NO recomendado fuera de la LAN).';
	@override String get hostKeyPlaceholder => 'ssh-ed25519 AAAA...';
	@override String get pathPrefixLabel => 'Prefijo de ruta';
	@override String get pathPrefixHint => 'Absoluta o relativa al directorio personal del usuario (opcional)';
	@override String get pathPrefixPlaceholder => '/var/backups/opendray  o  opendray-backups';
}

// Path: web.backups.targetEditor.rclone
class _TranslationsWebBackupsTargetEditorRcloneEs extends TranslationsWebBackupsTargetEditorRcloneEn {
	_TranslationsWebBackupsTargetEditorRcloneEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get rcloneHint => 'Requiere tener instalada la CLI de <1>rclone</1> en el host de opendray. Primero configura tu remoto con <3>rclone config</3>, luego usa el nombre del remoto de abajo. opendray invoca <5>rclone rcat / cat / lsd</5> internamente.';
	@override String get remoteLabel => 'Nombre del remoto';
	@override String get remoteHint => 'Nombre de `rclone config` (sin dos puntos). Ejemplo: gdrive, onedrive, dropbox-personal, baidu-pan';
	@override String get remotePlaceholder => 'gdrive';
	@override String get pathPrefixLabel => 'Prefijo de ruta';
	@override String get pathPrefixHint => 'Subcarpeta bajo la raíz del remoto (opcional)';
	@override String get pathPrefixPlaceholder => 'opendray/backups';
	@override String get binaryPathLabel => 'Ruta del binario';
	@override String get binaryPathHint => 'Anula `which rclone`. Si está vacío, usa la búsqueda en PATH.';
	@override String get binaryPathPlaceholder => '/opt/homebrew/bin/rclone';
	@override String get configPathLabel => 'Ruta de configuración';
	@override String get configPathHint => 'Anula --config (por defecto ~/.config/rclone/rclone.conf o ~/.rclone.conf)';
	@override String get configPathPlaceholder => 'déjalo en blanco para el valor por defecto de rclone';
}

// Path: web.serverSettings.sections.general
class _TranslationsWebServerSettingsSectionsGeneralEs extends TranslationsWebServerSettingsSectionsGeneralEn {
	_TranslationsWebServerSettingsSectionsGeneralEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'General';
	@override String get desc => 'Dirección de escucha, cuenta de operador, TTL del token.';
}

// Path: web.serverSettings.sections.logging
class _TranslationsWebServerSettingsSectionsLoggingEs extends TranslationsWebServerSettingsSectionsLoggingEn {
	_TranslationsWebServerSettingsSectionsLoggingEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Registro';
	@override String get desc => 'Verbosidad, formato y seguimiento en vivo.';
}

// Path: web.serverSettings.sections.sessions
class _TranslationsWebServerSettingsSectionsSessionsEs extends TranslationsWebServerSettingsSectionsSessionsEn {
	_TranslationsWebServerSettingsSectionsSessionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Sesiones';
	@override String get desc => 'Umbrales de detección de inactividad.';
}

// Path: web.serverSettings.sections.vault
class _TranslationsWebServerSettingsSectionsVaultEs extends TranslationsWebServerSettingsSectionsVaultEn {
	_TranslationsWebServerSettingsSectionsVaultEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Vault';
	@override String get desc => 'Notas, skills y raíz versionada con git.';
}

// Path: web.serverSettings.sections.mcp
class _TranslationsWebServerSettingsSectionsMcpEs extends TranslationsWebServerSettingsSectionsMcpEn {
	_TranslationsWebServerSettingsSectionsMcpEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Registro de MCP';
	@override String get desc => 'Registro de servidores + secretos.';
}

// Path: web.serverSettings.sections.memory
class _TranslationsWebServerSettingsSectionsMemoryEs extends TranslationsWebServerSettingsSectionsMemoryEn {
	_TranslationsWebServerSettingsSectionsMemoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Memoria · almacenamiento y embedder';
	@override String get desc => 'La mitad de infraestructura del subsistema de memoria: backend de embeddings, ajuste de recuperación y gobernanza de fondo. Reinicia para aplicar. El comportamiento en runtime (workers, captura, inyección) vive en los ajustes de Cortex.';
}

// Path: web.serverSettings.sections.backup
class _TranslationsWebServerSettingsSectionsBackupEs extends TranslationsWebServerSettingsSectionsBackupEn {
	_TranslationsWebServerSettingsSectionsBackupEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Backup';
	@override String get desc => 'Copias de seguridad cifradas de la DB, restauración y exportaciones de datos de administración.';
}

// Path: web.serverSettings.sections.claude
class _TranslationsWebServerSettingsSectionsClaudeEs extends TranslationsWebServerSettingsSectionsClaudeEn {
	_TranslationsWebServerSettingsSectionsClaudeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Almacenamiento · Claude';
	@override String get desc => 'Dónde se guardan los transcripts de Claude en disco.';
}

// Path: web.serverSettings.sections.codex
class _TranslationsWebServerSettingsSectionsCodexEs extends TranslationsWebServerSettingsSectionsCodexEn {
	_TranslationsWebServerSettingsSectionsCodexEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Almacenamiento · Codex';
	@override String get desc => 'Raíz de sesiones de Codex.';
}

// Path: web.serverSettings.sections.gemini
class _TranslationsWebServerSettingsSectionsGeminiEs extends TranslationsWebServerSettingsSectionsGeminiEn {
	_TranslationsWebServerSettingsSectionsGeminiEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Almacenamiento · Gemini';
	@override String get desc => 'Carpeta tmp por proyecto de Gemini + projects.json.';
}

// Path: web.serverSettings.sections.antigravity
class _TranslationsWebServerSettingsSectionsAntigravityEs extends TranslationsWebServerSettingsSectionsAntigravityEn {
	_TranslationsWebServerSettingsSectionsAntigravityEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Almacenamiento · Antigravity';
	@override String get desc => 'Almacén SQLite por conversación de Antigravity (agy).';
}

// Path: web.serverSettings.fields.listenAddress
class _TranslationsWebServerSettingsFieldsListenAddressEs extends TranslationsWebServerSettingsFieldsListenAddressEn {
	_TranslationsWebServerSettingsFieldsListenAddressEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Dirección de escucha';
	@override String get hint => 'El host:port al que se vincula el servidor HTTP. Ejemplo: 0.0.0.0:8770.';
}

// Path: web.serverSettings.fields.username
class _TranslationsWebServerSettingsFieldsUsernameEs extends TranslationsWebServerSettingsFieldsUsernameEn {
	_TranslationsWebServerSettingsFieldsUsernameEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Nombre de usuario';
	@override String get hint => 'Nombre de inicio de sesión usado en el formulario de acceso. Cambiarlo obliga a volver a iniciar sesión en la siguiente solicitud.';
}

// Path: web.serverSettings.fields.password
class _TranslationsWebServerSettingsFieldsPasswordEs extends TranslationsWebServerSettingsFieldsPasswordEn {
	_TranslationsWebServerSettingsFieldsPasswordEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Contraseña';
	@override String get hint => 'Déjalo en blanco para mantener la contraseña actual. Enviar un valor la sobrescribe.';
	@override String get hideTitle => 'Ocultar';
	@override String get revealTitle => 'Mostrar';
}

// Path: web.serverSettings.fields.tokenTTL
class _TranslationsWebServerSettingsFieldsTokenTTLEs extends TranslationsWebServerSettingsFieldsTokenTTLEn {
	_TranslationsWebServerSettingsFieldsTokenTTLEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'TTL del token';
	@override String get hint => 'Tiempo de vida del bearer-token como duración de Go, p. ej. "24h", "30m". Vacío = nunca expira.';
}

// Path: web.serverSettings.fields.logLevel
class _TranslationsWebServerSettingsFieldsLogLevelEs extends TranslationsWebServerSettingsFieldsLogLevelEn {
	_TranslationsWebServerSettingsFieldsLogLevelEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Nivel de log';
	@override String get hint => 'Las líneas por debajo de este nivel se descartan.';
}

// Path: web.serverSettings.fields.logFormat
class _TranslationsWebServerSettingsFieldsLogFormatEs extends TranslationsWebServerSettingsFieldsLogFormatEn {
	_TranslationsWebServerSettingsFieldsLogFormatEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Formato';
	@override String get hint => '"text" es legible para humanos; "json" es analizable por máquinas.';
}

// Path: web.serverSettings.fields.logFile
class _TranslationsWebServerSettingsFieldsLogFileEs extends TranslationsWebServerSettingsFieldsLogFileEn {
	_TranslationsWebServerSettingsFieldsLogFileEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Archivo de log';
	@override String get hint => 'Ruta de archivo opcional. Rota automáticamente a los 10 MB, conserva 5 copias. Vacío = solo stderr.';
}

// Path: web.serverSettings.fields.idleThreshold
class _TranslationsWebServerSettingsFieldsIdleThresholdEs extends TranslationsWebServerSettingsFieldsIdleThresholdEn {
	_TranslationsWebServerSettingsFieldsIdleThresholdEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Umbral de inactividad';
	@override String get hint => 'Una session permanece en silencio este tiempo antes de que se dispare session.idle. Vacío = 30s.';
}

// Path: web.serverSettings.fields.idlePollInterval
class _TranslationsWebServerSettingsFieldsIdlePollIntervalEs extends TranslationsWebServerSettingsFieldsIdlePollIntervalEn {
	_TranslationsWebServerSettingsFieldsIdlePollIntervalEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Intervalo de sondeo de inactividad';
	@override String get hint => 'Con qué frecuencia se activa el detector de inactividad. Más bajo = menor latencia, más activaciones. Vacío = 5s.';
}

// Path: web.serverSettings.fields.vaultRoot
class _TranslationsWebServerSettingsFieldsVaultRootEs extends TranslationsWebServerSettingsFieldsVaultRootEn {
	_TranslationsWebServerSettingsFieldsVaultRootEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Raíz del Vault';
	@override String get hint => 'Directorio de nivel superior para notas, skills y el registro de MCP.';
}

// Path: web.serverSettings.fields.notesDirectory
class _TranslationsWebServerSettingsFieldsNotesDirectoryEs extends TranslationsWebServerSettingsFieldsNotesDirectoryEn {
	_TranslationsWebServerSettingsFieldsNotesDirectoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Directorio de notas';
	@override String get hint => 'Anula la ubicación de las notas. Por defecto <vault root>/notes.';
}

// Path: web.serverSettings.fields.skillsDirectory
class _TranslationsWebServerSettingsFieldsSkillsDirectoryEs extends TranslationsWebServerSettingsFieldsSkillsDirectoryEn {
	_TranslationsWebServerSettingsFieldsSkillsDirectoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Directorio de skills';
	@override String get hint => 'Anula la ubicación de los skills. Por defecto <vault root>/skills.';
}

// Path: web.serverSettings.fields.gitRoot
class _TranslationsWebServerSettingsFieldsGitRootEs extends TranslationsWebServerSettingsFieldsGitRootEn {
	_TranslationsWebServerSettingsFieldsGitRootEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Raíz de git';
	@override String get hint => 'Árbol de trabajo al que confirma la función Vault Sync.';
}

// Path: web.serverSettings.fields.personalPrefix
class _TranslationsWebServerSettingsFieldsPersonalPrefixEs extends TranslationsWebServerSettingsFieldsPersonalPrefixEn {
	_TranslationsWebServerSettingsFieldsPersonalPrefixEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Prefijo personal';
	@override String get hint => 'Nombre de carpeta usado para las notas personales al derivar rutas automáticamente. Por defecto "personal".';
}

// Path: web.serverSettings.fields.projectsPrefix
class _TranslationsWebServerSettingsFieldsProjectsPrefixEs extends TranslationsWebServerSettingsFieldsProjectsPrefixEn {
	_TranslationsWebServerSettingsFieldsProjectsPrefixEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Prefijo de proyectos';
	@override String get hint => 'Nombre de carpeta usado para las notas de proyecto. Por defecto "projects".';
}

// Path: web.serverSettings.fields.registryRoot
class _TranslationsWebServerSettingsFieldsRegistryRootEs extends TranslationsWebServerSettingsFieldsRegistryRootEn {
	_TranslationsWebServerSettingsFieldsRegistryRootEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Raíz del registro';
	@override String get hint => 'Directorio que contiene las definiciones JSON de los servidores MCP. Por defecto <vault>/mcp.';
}

// Path: web.serverSettings.fields.secretsFile
class _TranslationsWebServerSettingsFieldsSecretsFileEs extends TranslationsWebServerSettingsFieldsSecretsFileEn {
	_TranslationsWebServerSettingsFieldsSecretsFileEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Archivo de secretos';
	@override String get hint => 'Archivo key=value que se sustituye en los comandos del servidor MCP en el momento del arranque.';
}

// Path: web.serverSettings.fields.memoryBackend
class _TranslationsWebServerSettingsFieldsMemoryBackendEs extends TranslationsWebServerSettingsFieldsMemoryBackendEn {
	_TranslationsWebServerSettingsFieldsMemoryBackendEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Backend del embedder';
	@override String get hint => '"auto" / "bm25" usan la ruta de palabras clave en Go puro, sin cgo. "http" llama a cualquier /v1/embeddings compatible con OpenAI (ollama / OpenAI / LocalAI). "local" ejecuta un sentence-transformer ONNX en el proceso, requiere un binario compilado con `-tags local_onnx`.';
}

// Path: web.serverSettings.fields.memoryStore
class _TranslationsWebServerSettingsFieldsMemoryStoreEs extends TranslationsWebServerSettingsFieldsMemoryStoreEn {
	_TranslationsWebServerSettingsFieldsMemoryStoreEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Almacén';
	@override String get hint => '"pgvector" reutiliza la PG existente de opendray con la extensión vector; es la única opción en la v1.';
}

// Path: web.serverSettings.fields.memoryTopK
class _TranslationsWebServerSettingsFieldsMemoryTopKEs extends TranslationsWebServerSettingsFieldsMemoryTopKEn {
	_TranslationsWebServerSettingsFieldsMemoryTopKEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Top-K por defecto';
	@override String get hint => 'Cuántos resultados devuelve memory_search cuando el agente no lo especifica. Vacío = 5.';
}

// Path: web.serverSettings.fields.memoryThreshold
class _TranslationsWebServerSettingsFieldsMemoryThresholdEs extends TranslationsWebServerSettingsFieldsMemoryThresholdEn {
	_TranslationsWebServerSettingsFieldsMemoryThresholdEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Umbral de similitud';
	@override String get hint => 'Los resultados por debajo de esta puntuación se descartan. Vacío = 0.1 (permisivo, los vectores dispersos de BM25 rara vez superan 0.5).';
}

// Path: web.serverSettings.fields.memoryScope
class _TranslationsWebServerSettingsFieldsMemoryScopeEs extends TranslationsWebServerSettingsFieldsMemoryScopeEn {
	_TranslationsWebServerSettingsFieldsMemoryScopeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Ámbito por defecto';
	@override String get hint => 'Lo que usa memory_store cuando el agente no lo especifica. "project" (recomendado) agrupa por cwd; "global" comparte entre cwds.';
}

// Path: web.serverSettings.fields.memoryBaseUrl
class _TranslationsWebServerSettingsFieldsMemoryBaseUrlEs extends TranslationsWebServerSettingsFieldsMemoryBaseUrlEn {
	_TranslationsWebServerSettingsFieldsMemoryBaseUrlEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'URL base';
	@override String get hint => 'p. ej. "http://localhost:11434/v1" para ollama, "https://api.openai.com/v1" para OpenAI.';
}

// Path: web.serverSettings.fields.memoryModel
class _TranslationsWebServerSettingsFieldsMemoryModelEs extends TranslationsWebServerSettingsFieldsMemoryModelEn {
	_TranslationsWebServerSettingsFieldsMemoryModelEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Modelo';
	@override String get hint => 'p. ej. "nomic-embed-text" para ollama, "text-embedding-3-small" para OpenAI.';
}

// Path: web.serverSettings.fields.memoryApiKey
class _TranslationsWebServerSettingsFieldsMemoryApiKeyEs extends TranslationsWebServerSettingsFieldsMemoryApiKeyEn {
	_TranslationsWebServerSettingsFieldsMemoryApiKeyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'API key';
	@override String get hint => 'Vacío para ollama / servidores locales. Obligatorio para OpenAI / Voyage / servicios alojados.';
}

// Path: web.serverSettings.fields.memoryLocalModel
class _TranslationsWebServerSettingsFieldsMemoryLocalModelEs extends TranslationsWebServerSettingsFieldsMemoryLocalModelEn {
	_TranslationsWebServerSettingsFieldsMemoryLocalModelEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Nombre del modelo';
	@override String get hint => 'Cosmético, aparece en los logs / el Inspector. p. ej. "bge-m3", "bge-small-en-v1.5".';
}

// Path: web.serverSettings.fields.memoryLibraryPath
class _TranslationsWebServerSettingsFieldsMemoryLibraryPathEs extends TranslationsWebServerSettingsFieldsMemoryLibraryPathEn {
	_TranslationsWebServerSettingsFieldsMemoryLibraryPathEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Ruta de la biblioteca';
	@override String get hint => 'Directorio que contiene libonnxruntime.dylib (macOS) / libonnxruntime.so (Linux). Tras `brew install onnxruntime`, es /opt/homebrew/opt/onnxruntime/lib.';
}

// Path: web.serverSettings.fields.memoryModelPath
class _TranslationsWebServerSettingsFieldsMemoryModelPathEs extends TranslationsWebServerSettingsFieldsMemoryModelPathEn {
	_TranslationsWebServerSettingsFieldsMemoryModelPathEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Ruta del modelo';
	@override String get hint => 'Ruta absoluta a los pesos .onnx. Descárgalos de HuggingFace, p. ej. Xenova/bge-m3 o Xenova/bge-small-en-v1.5.';
}

// Path: web.serverSettings.fields.memoryTokenizerPath
class _TranslationsWebServerSettingsFieldsMemoryTokenizerPathEs extends TranslationsWebServerSettingsFieldsMemoryTokenizerPathEn {
	_TranslationsWebServerSettingsFieldsMemoryTokenizerPathEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Ruta del tokenizador';
	@override String get hint => 'Ruta absoluta a tokenizer.json (formato estándar de HuggingFace), normalmente justo al lado del modelo.';
}

// Path: web.serverSettings.fields.memoryMaxSeqLen
class _TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEs extends TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEn {
	_TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Longitud máxima de secuencia';
	@override String get hint => 'Los tokens que superan este límite se truncan. El valor por defecto de bge-m3 es 512. Vacío = 512.';
}

// Path: web.serverSettings.fields.claudeHistoryRoots
class _TranslationsWebServerSettingsFieldsClaudeHistoryRootsEs extends TranslationsWebServerSettingsFieldsClaudeHistoryRootsEn {
	_TranslationsWebServerSettingsFieldsClaudeHistoryRootsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Raíces de historial';
	@override String get hint => 'Directorios que se escanean en busca de los transcripts JSONL por proyecto de Claude. Vacío = escanear ~/.claude/projects + cada ~/.claude-accounts/*/projects.';
}

// Path: web.serverSettings.fields.claudeAccountsDir
class _TranslationsWebServerSettingsFieldsClaudeAccountsDirEs extends TranslationsWebServerSettingsFieldsClaudeAccountsDirEn {
	_TranslationsWebServerSettingsFieldsClaudeAccountsDirEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Directorio de cuentas';
	@override String get hint => 'Raíz usada para los ConfigDirs de las cuentas de Claude gestionadas por opendray. Por defecto ~/.claude-accounts.';
}

// Path: web.serverSettings.fields.codexSessionsRoot
class _TranslationsWebServerSettingsFieldsCodexSessionsRootEs extends TranslationsWebServerSettingsFieldsCodexSessionsRootEn {
	_TranslationsWebServerSettingsFieldsCodexSessionsRootEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Raíz de sesiones';
	@override String get hint => 'Directorio que se recorre en busca de los archivos JSONL de rollout de Codex. Por defecto ~/.codex/sessions.';
}

// Path: web.serverSettings.fields.geminiTmpRoot
class _TranslationsWebServerSettingsFieldsGeminiTmpRootEs extends TranslationsWebServerSettingsFieldsGeminiTmpRootEn {
	_TranslationsWebServerSettingsFieldsGeminiTmpRootEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Directorio tmp';
	@override String get hint => 'Raíz que contiene las carpetas tmp por proyecto de Gemini. Por defecto ~/.gemini/tmp.';
}

// Path: web.serverSettings.fields.geminiProjectsFile
class _TranslationsWebServerSettingsFieldsGeminiProjectsFileEs extends TranslationsWebServerSettingsFieldsGeminiProjectsFileEn {
	_TranslationsWebServerSettingsFieldsGeminiProjectsFileEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'projects.json';
	@override String get hint => 'Ruta al archivo de mapeo cwd→nombre-corto de Gemini. Por defecto ~/.gemini/projects.json.';
}

// Path: web.serverSettings.fields.antigravityConversationsRoot
class _TranslationsWebServerSettingsFieldsAntigravityConversationsRootEs extends TranslationsWebServerSettingsFieldsAntigravityConversationsRootEn {
	_TranslationsWebServerSettingsFieldsAntigravityConversationsRootEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Directorio de conversaciones';
	@override String get hint => 'Raíz con los archivos .db por conversación de agy. Por defecto ~/.gemini/antigravity-cli/conversations.';
}

// Path: web.serverSettings.fields.backupLocalDir
class _TranslationsWebServerSettingsFieldsBackupLocalDirEs extends TranslationsWebServerSettingsFieldsBackupLocalDirEn {
	_TranslationsWebServerSettingsFieldsBackupLocalDirEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Directorio local de copias de seguridad';
	@override String get hint => 'Raíz por defecto para el destino `local` creado automáticamente. Vacío = ~/.opendray/backups. Requiere reinicio.';
}

// Path: web.serverSettings.fields.backupExportDir
class _TranslationsWebServerSettingsFieldsBackupExportDirEs extends TranslationsWebServerSettingsFieldsBackupExportDirEn {
	_TranslationsWebServerSettingsFieldsBackupExportDirEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Directorio de exportación';
	@override String get hint => 'Dónde se preparan en disco los zips de exportación puntual. Vacío = ~/.opendray/exports. Los paquetes expiran automáticamente tras 24h. Requiere reinicio.';
}

// Path: web.serverSettings.fields.backupPgDumpPath
class _TranslationsWebServerSettingsFieldsBackupPgDumpPathEs extends TranslationsWebServerSettingsFieldsBackupPgDumpPathEn {
	_TranslationsWebServerSettingsFieldsBackupPgDumpPathEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Ruta de pg_dump';
	@override String get hint => 'Ruta absoluta a pg_dump. La versión mayor debe ser ≥ la del servidor. Vacío = el primer pg_dump en el PATH.';
}

// Path: web.serverSettings.fields.backupPgRestorePath
class _TranslationsWebServerSettingsFieldsBackupPgRestorePathEs extends TranslationsWebServerSettingsFieldsBackupPgRestorePathEn {
	_TranslationsWebServerSettingsFieldsBackupPgRestorePathEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Ruta de pg_restore';
	@override String get hint => 'Ruta absoluta a pg_restore para el flujo /backups/restore. Misma regla de versión mayor.';
}

// Path: web.serverSettings.fields.memoryDedup
class _TranslationsWebServerSettingsFieldsMemoryDedupEs extends TranslationsWebServerSettingsFieldsMemoryDedupEn {
	_TranslationsWebServerSettingsFieldsMemoryDedupEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Umbral de dedup';
	@override String get hint => 'Umbral de plegado al escribir: una similitud superior actualiza la memoria existente en vez de insertar un casi-duplicado. 0 = valor por defecto relativo al embedder; negativo desactiva el plegado.';
}

// Path: web.serverSettings.fields.gatekeeperEnabled
class _TranslationsWebServerSettingsFieldsGatekeeperEnabledEs extends TranslationsWebServerSettingsFieldsGatekeeperEnabledEn {
	_TranslationsWebServerSettingsFieldsGatekeeperEnabledEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Gatekeeper';
	@override String get hint => 'Juez LLM pre-escritura: decide si un memory_store trae un hecho durable o ruido. Qué LLM lo ejecuta se enruta en ajustes de Cortex → Workers.';
}

// Path: web.serverSettings.fields.gatekeeperLatency
class _TranslationsWebServerSettingsFieldsGatekeeperLatencyEs extends TranslationsWebServerSettingsFieldsGatekeeperLatencyEn {
	_TranslationsWebServerSettingsFieldsGatekeeperLatencyEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Latencia máx. del gatekeeper (ms)';
	@override String get hint => 'Por encima de esto el gatekeeper degrada a "permitir" en vez de bloquear la escritura por un LLM lento. Por defecto 2000.';
}

// Path: web.serverSettings.fields.cleanerEnabled
class _TranslationsWebServerSettingsFieldsCleanerEnabledEs extends TranslationsWebServerSettingsFieldsCleanerEnabledEn {
	_TranslationsWebServerSettingsFieldsCleanerEnabledEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Cleaner (auto-bibliotecario)';
	@override String get hint => 'Barrido periódico que archiva (reversible, con periodo de gracia) memorias obsoletas o duplicadas. Qué LLM lo ejecuta se enruta en ajustes de Cortex → Workers.';
}

// Path: web.serverSettings.fields.cleanerInterval
class _TranslationsWebServerSettingsFieldsCleanerIntervalEs extends TranslationsWebServerSettingsFieldsCleanerIntervalEn {
	_TranslationsWebServerSettingsFieldsCleanerIntervalEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Intervalo del cleaner (s)';
	@override String get hint => 'Segundos entre barridos automáticos. Por defecto 86400 (24h).';
}

// Path: web.serverSettings.fields.cleanerGlobalScope
class _TranslationsWebServerSettingsFieldsCleanerGlobalScopeEs extends TranslationsWebServerSettingsFieldsCleanerGlobalScopeEn {
	_TranslationsWebServerSettingsFieldsCleanerGlobalScopeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Cleaner barre el scope global';
	@override String get hint => 'Revisar también memorias de scope global (normalmente curadas por el operador). Por defecto off hasta que confíes en el cleaner.';
}

// Path: web.serverSettings.fields.knowledgeEnabled
class _TranslationsWebServerSettingsFieldsKnowledgeEnabledEs extends TranslationsWebServerSettingsFieldsKnowledgeEnabledEn {
	_TranslationsWebServerSettingsFieldsKnowledgeEnabledEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Grafo de conocimiento';
	@override String get hint => 'La capa de conocimiento estructurada y auto-evolutiva (entidades / playbooks / skills) sobre la memoria episódica. Alimenta la pestaña Cortex → Knowledge.';
}

// Path: web.serverSettings.fields.claudeWatcher
class _TranslationsWebServerSettingsFieldsClaudeWatcherEs extends TranslationsWebServerSettingsFieldsClaudeWatcherEn {
	_TranslationsWebServerSettingsFieldsClaudeWatcherEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Watcher de cuentas';
	@override String get hint => 'Vigila accounts_dir y auto-registra una cuenta nueva cuando aparece un .credentials.json (resultado de CLAUDE_CONFIG_DIR=<dir> claude login).';
}

// Path: web.serverSettings.fields.claudeAutoFailover
class _TranslationsWebServerSettingsFieldsClaudeAutoFailoverEs extends TranslationsWebServerSettingsFieldsClaudeAutoFailoverEn {
	_TranslationsWebServerSettingsFieldsClaudeAutoFailoverEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Auto-failover por rate limit';
	@override String get hint => 'Cambia una session viva a otra cuenta de Claude al chocar con un rate limit. Opt-in: cambia la atribución de facturación sin un clic.';
}

// Path: web.serverSettings.fields.mobileTokenTTL
class _TranslationsWebServerSettingsFieldsMobileTokenTTLEs extends TranslationsWebServerSettingsFieldsMobileTokenTTLEn {
	_TranslationsWebServerSettingsFieldsMobileTokenTTLEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'TTL del token móvil';
	@override String get hint => 'Vida de los tokens emitidos a la app móvil. Por defecto 720h (30 días).';
}

// Path: web.serverSettings.fields.dbMaxConns
class _TranslationsWebServerSettingsFieldsDbMaxConnsEs extends TranslationsWebServerSettingsFieldsDbMaxConnsEn {
	_TranslationsWebServerSettingsFieldsDbMaxConnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Conexiones máx.';
	@override String get hint => 'Tope del pool de conexiones pgx. 0 = valor por defecto (16).';
}

// Path: web.serverSettings.httpHelpers.presetTip
class _TranslationsWebServerSettingsHttpHelpersPresetTipEs extends TranslationsWebServerSettingsHttpHelpersPresetTipEn {
	_TranslationsWebServerSettingsHttpHelpersPresetTipEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get ollama => 'Daemon local de ollama';
	@override String get lmStudio => 'Servidor local de LM Studio';
	@override String get openai => 'Nube de OpenAI (necesita API key)';
}

// Path: web.serverSettings.backup.scheduleHeaders
class _TranslationsWebServerSettingsBackupScheduleHeadersEs extends TranslationsWebServerSettingsBackupScheduleHeadersEn {
	_TranslationsWebServerSettingsBackupScheduleHeadersEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get schedule => 'Programación';
	@override String get target => 'Destino';
	@override String get cadence => 'Cadencia';
	@override String get keep => 'Conservar';
	@override String get state => 'Estado';
}

// Path: web.settings.appearance.options
class _TranslationsWebSettingsAppearanceOptionsEs extends TranslationsWebSettingsAppearanceOptionsEn {
	_TranslationsWebSettingsAppearanceOptionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get light => 'Claro';
	@override String get lightDesc => 'Siempre claro';
	@override String get dark => 'Oscuro';
	@override String get darkDesc => 'Siempre oscuro';
	@override String get system => 'Sistema';
	@override String get systemDesc => 'Seguir la configuración del SO';
}

// Path: web.settings.font.options
class _TranslationsWebSettingsFontOptionsEs extends TranslationsWebSettingsFontOptionsEn {
	_TranslationsWebSettingsFontOptionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get compact => 'Compacto';
	@override String get kDefault => 'Predeterminado';
	@override String get comfy => 'Cómodo';
	@override String get large => 'Grande';
}

// Path: web.memoryAmbient.providers.row
class _TranslationsWebMemoryAmbientProvidersRowEs extends TranslationsWebMemoryAmbientProvidersRowEn {
	_TranslationsWebMemoryAmbientProvidersRowEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get defaultBadge => '★ predeterminado';
	@override String get makeDefault => 'Hacer predeterminado';
	@override String get test => 'Probar';
	@override String get testing => 'Probando…';
	@override String get delete => 'Eliminar';
	@override String testOk({required Object name}) => '${name}: conexión correcta';
	@override String get testFailedToast => 'La prueba falló';
	@override String deleteConfirm({required Object name}) => '¿Eliminar el proveedor "${name}"?';
	@override String get deletedToast => 'Proveedor eliminado';
	@override String get deleteFailedToast => 'La eliminación falló';
	@override String get updateFailedToast => 'La actualización falló';
	@override String madeDefaultToast({required Object name}) => '${name} ahora es el predeterminado';
}

// Path: web.memoryAmbient.providers.dialog
class _TranslationsWebMemoryAmbientProvidersDialogEs extends TranslationsWebMemoryAmbientProvidersDialogEn {
	_TranslationsWebMemoryAmbientProvidersDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Añadir proveedor de resumen';
	@override String get kindLabel => 'Tipo';
	@override String get nameLabel => 'Nombre';
	@override String get namePlaceholder => 'p. ej. lmstudio-qwen';
	@override String get modelLabel => 'Modelo';
	@override String get baseUrlLabel => 'URL base';
	@override String get integrationNote => 'Los proveedores de integración resuelven su URL base a partir de una integración registrada. Configúrala primero en Integraciones; el cableado avanzado (extra_config) es solo de DB en esta versión.';
	@override String get apiKeyLabel => 'Clave de API';
	@override String get apiKeyHint => 'Se almacena cifrada (AES-GCM con la frase de contraseña maestra de copia de seguridad). Nunca se devuelve; tras guardar solo se muestra la huella digital.';
	@override String get makeDefaultLabel => 'Hacer de este el proveedor predeterminado';
	@override String get create => 'Crear';
	@override String get nameRequiredToast => 'El nombre es obligatorio';
	@override String createdToast({required Object name}) => 'Proveedor ${name} creado';
	@override String get createFailedToast => 'La creación falló';
}

// Path: web.memoryAmbient.providers.modelSelect
class _TranslationsWebMemoryAmbientProvidersModelSelectEs extends TranslationsWebMemoryAmbientProvidersModelSelectEn {
	_TranslationsWebMemoryAmbientProvidersModelSelectEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get editTitle => 'Cambiar modelo';
	@override String dialogTitle({required Object name}) => 'Cambiar modelo — ${name}';
	@override String get custom => 'Personalizado…';
	@override String get backToList => 'Elegir de la lista';
	@override String get refresh => 'Volver a escanear los modelos del endpoint';
	@override String get unreachable => 'Endpoint no alcanzable — escribe el nombre del modelo a mano; la lista aparece cuando el servicio esté arriba.';
	@override String get none => 'El endpoint responde pero no anuncia modelos — carga uno en LM Studio / haz pull en Ollama y vuelve a escanear.';
	@override String get notOnEndpoint => 'no está en el endpoint';
	@override String get save => 'Guardar modelo';
	@override String savedToast({required Object name, required Object model}) => '${name} ahora usa ${model}';
}

// Path: web.memoryAmbient.rules.row
class _TranslationsWebMemoryAmbientRulesRowEs extends TranslationsWebMemoryAmbientRulesRowEn {
	_TranslationsWebMemoryAmbientRulesRowEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get globalDefault => 'predeterminado global';
	@override String get scopeLabel => 'ámbito:';
	@override String get dedupLabel => 'dedup:';
	@override String get runNow => 'Ejecutar ahora';
	@override String get running => 'Ejecutando…';
	@override String get delete => 'Eliminar';
	@override String firedToast({required Object sessions}) => 'La regla se activó en ${sessions} session(es)';
	@override String get runNowFailedToast => 'La ejecución inmediata falló';
	@override String deleteConfirm({required Object name}) => '¿Eliminar la regla "${name}"?';
	@override String get deletedToast => 'Regla eliminada';
	@override String get deleteFailedToast => 'La eliminación falló';
	@override late final _TranslationsWebMemoryAmbientRulesRowSummaryEs summary = _TranslationsWebMemoryAmbientRulesRowSummaryEs._(_root);
}

// Path: web.memoryAmbient.rules.dialog
class _TranslationsWebMemoryAmbientRulesDialogEs extends TranslationsWebMemoryAmbientRulesDialogEn {
	_TranslationsWebMemoryAmbientRulesDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Añadir regla de captura';
	@override String get nameLabel => 'Nombre';
	@override String get triggerLabel => 'Disparador';
	@override String get nLabel => 'N (mensajes)';
	@override String get idleLabel => 'Segundos de inactividad';
	@override String get kLabel => 'K (caracteres)';
	@override String get scopeLabel => 'Ámbito objetivo';
	@override String get scopeProject => 'proyecto (recomendado)';
	@override String get scopeGlobal => 'global';
	@override String get dedupLabel => 'Umbral de dedup (0.0 a 1.0)';
	@override String get dedupHint => 'Más alto = deduplicación más estricta. 0.85 es el punto óptimo recomendado.';
	@override String get create => 'Crear';
	@override String get nameRequiredToast => 'El nombre es obligatorio';
	@override String createdToast({required Object name}) => 'Regla ${name} creada';
	@override String get createFailedToast => 'La creación falló';
}

// Path: web.memoryAmbient.profiles.row
class _TranslationsWebMemoryAmbientProfilesRowEs extends TranslationsWebMemoryAmbientProfilesRowEn {
	_TranslationsWebMemoryAmbientProfilesRowEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get globalDefault => 'predeterminado global';
	@override String get delete => 'Eliminar';
	@override String get deleteConfirm => '¿Eliminar este perfil de inyección?';
	@override String get deletedToast => 'Perfil eliminado';
	@override String get deleteFailedToast => 'La eliminación falló';
}

// Path: web.memoryAmbient.profiles.dialog
class _TranslationsWebMemoryAmbientProfilesDialogEs extends TranslationsWebMemoryAmbientProfilesDialogEn {
	_TranslationsWebMemoryAmbientProfilesDialogEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Añadir perfil de inyección';
	@override String get strategyLabel => 'Estrategia';
	@override String get kLabel => 'K (principales memorias a inyectar)';
	@override String get hint => 'Un perfil por session_id (o predeterminado global). Los perfiles por session se pueden añadir más tarde mediante API; la UI actualmente solo gestiona el predeterminado global.';
	@override String get create => 'Crear';
	@override String get createdToast => 'Perfil creado';
	@override String get createFailedToast => 'La creación falló';
}

// Path: web.memoryAmbient.cost.columns
class _TranslationsWebMemoryAmbientCostColumnsEs extends TranslationsWebMemoryAmbientCostColumnsEn {
	_TranslationsWebMemoryAmbientCostColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get provider => 'Proveedor';
	@override String get calls => 'Llamadas';
	@override String get inTokens => 'Tokens de entrada';
	@override String get outTokens => 'Tokens de salida';
	@override String get usdEst => 'USD est.';
}

// Path: web.export.form.integrationOptions
class _TranslationsWebExportFormIntegrationOptionsEs extends TranslationsWebExportFormIntegrationOptionsEn {
	_TranslationsWebExportFormIntegrationOptionsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get none => 'Ninguna';
	@override String get noneHint => 'Omitir por completo la tabla de integraciones.';
	@override String get metadata => 'Solo metadatos (recomendado)';
	@override String get metadataHint => 'ID, nombre, prefijo de ruta, alcances. Sin material de API key.';
	@override String get plaintext => 'Incluir las API keys en texto plano';
	@override String get plaintextHint => 'v1 solo con bcrypt: no existe texto plano recuperable. El manifiesto lo documenta; no se filtra nada.';
}

// Path: web.export.history.columns
class _TranslationsWebExportHistoryColumnsEs extends TranslationsWebExportHistoryColumnsEn {
	_TranslationsWebExportHistoryColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get status => 'Estado';
	@override String get scope => 'Alcance';
	@override String get size => 'Tamaño';
	@override String get expires => 'Caduca';
	@override String get actions => 'Acciones';
}

// Path: web.export.import.summaryCard
class _TranslationsWebExportImportSummaryCardEs extends TranslationsWebExportImportSummaryCardEn {
	_TranslationsWebExportImportSummaryCardEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get memories => 'Memorias';
	@override String get integrations => 'Integraciones';
	@override String get customTasks => 'Tareas personalizadas';
	@override String get created => 'creadas';
	@override String get skipped => 'omitidas';
	@override String get failed => 'fallidas';
}

// Path: web.export.imports.columns
class _TranslationsWebExportImportsColumnsEs extends TranslationsWebExportImportsColumnsEn {
	_TranslationsWebExportImportsColumnsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get status => 'Estado';
	@override String get source => 'Origen';
	@override String get counts => 'Recuentos';
	@override String get when => 'Cuándo';
}

// Path: web.knowledge.kb.kinds
class _TranslationsWebKnowledgeKbKindsEs extends TranslationsWebKnowledgeKbKindsEn {
	_TranslationsWebKnowledgeKbKindsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get kb_infrastructure => 'Infraestructura';
	@override String get kb_conventions => 'Convenciones';
	@override String get kb_lessons => 'Lecciones';
	@override String get kb_reusable => 'Funciones reutilizables';
}

// Path: web.knowledge.kb.proposal
class _TranslationsWebKnowledgeKbProposalEs extends TranslationsWebKnowledgeKbProposalEn {
	_TranslationsWebKnowledgeKbProposalEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get text => 'La IA propuso una actualización de esta página (evidencia nueva divergente).';
	@override String get preview => 'Vista previa';
	@override String get hide => 'Ocultar';
	@override String get approve => 'Aprobar';
	@override String get reject => 'Rechazar';
	@override String get approved => 'Actualización aprobada';
	@override String get rejected => 'Propuesta rechazada';
}

// Path: web.knowledge.kb.newPage
class _TranslationsWebKnowledgeKbNewPageEs extends TranslationsWebKnowledgeKbNewPageEn {
	_TranslationsWebKnowledgeKbNewPageEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get button => 'Nueva página de conocimiento';
	@override String get title => 'Nueva página de conocimiento';
	@override String get description => 'Da a cada dominio de conocimiento su propia página de grano fino en vez de engordar las clásicas — los agentes indexan páginas individualmente y recuperan solo lo que una tarea necesita.';
	@override String get slugPlaceholder => 'network_topology';
	@override String get titlePlaceholder => 'Título (p. ej. Topología de red)';
	@override String get descPlaceholder => 'Una frase: qué va en esta página';
	@override String get inject => 'inyectar en cada arranque';
	@override String get injectHint => 'Apagado (recomendado): la página queda fuera del banner de arranque y los agentes la alcanzan bajo demanda vía búsqueda. Encendido: las fundacionales se inyectan como reglas vinculantes, las emergentes como referencia.';
	@override String get create => 'Crear página';
	@override String get createdToast => 'Página de conocimiento creada';
}

// Path: web.knowledge.distill.retirement
class _TranslationsWebKnowledgeDistillRetirementEs extends TranslationsWebKnowledgeDistillRetirementEn {
	_TranslationsWebKnowledgeDistillRetirementEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get never_used => 'nunca usado';
	@override String get never_usedHint => 'Inyectado 14+ días sin que ninguna sesión lo refiera — el bucle de resultados propone retirarlo';
	@override String get low_success => 'poco éxito';
	@override String get low_successHint => 'Las sesiones que cargan este skill siguen terminando en fallo — el bucle de resultados propone retirarlo';
	@override String get dormant => 'inactivo';
	@override String get dormantHint => 'Se usó alguna vez, pero lleva 45+ días sin referencias — el bucle de resultados propone retirarlo';
}

// Path: web.knowledge.graph.legend
class _TranslationsWebKnowledgeGraphLegendEs extends TranslationsWebKnowledgeGraphLegendEn {
	_TranslationsWebKnowledgeGraphLegendEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get project => 'Proyecto';
	@override String get entity => 'Entidad';
	@override String get playbook => 'Playbook';
	@override String get skill => 'Skill';
}

// Path: web.cortex.home.memory
class _TranslationsWebCortexHomeMemoryEs extends TranslationsWebCortexHomeMemoryEn {
	_TranslationsWebCortexHomeMemoryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Memoria';
	@override String get description => 'Hechos episódicos capturados de tus sesiones — recuperados por relevancia, en cuarentena si vienen de terceros.';
	@override String quarantine({required Object count}) => '${count} en cuarentena';
}

// Path: web.cortex.home.notes
class _TranslationsWebCortexHomeNotesEs extends TranslationsWebCortexHomeNotesEn {
	_TranslationsWebCortexHomeNotesEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Notas';
	@override String get description => 'El documento oficial de cada proyecto — secciones según su plano, mantenidas por la IA mientras trabajas.';
	@override String projects({required Object count}) => '${count} activos';
}

// Path: web.cortex.home.knowledge
class _TranslationsWebCortexHomeKnowledgeEs extends TranslationsWebCortexHomeKnowledgeEn {
	_TranslationsWebCortexHomeKnowledgeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Conocimiento';
	@override String get description => 'Experiencia iterable entre proyectos: reglas fundacionales vinculantes + lecciones emergentes, inyectadas en cada arranque.';
}

// Path: web.cortex.home.proposals
class _TranslationsWebCortexHomeProposalsEs extends TranslationsWebCortexHomeProposalsEn {
	_TranslationsWebCortexHomeProposalsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String title({required Object count}) => 'Propuestas pendientes (${count})';
	@override String get hint => 'Actualizaciones propuestas por la IA para notas de proyecto y páginas KB, a la espera de tu veredicto. Aprueba para publicar, rechaza para descartar.';
	@override String get kbLabel => 'Base de conocimiento';
	@override String get preview => 'Vista previa';
	@override String get hide => 'Ocultar';
	@override String get approve => 'Aprobar';
	@override String get reject => 'Rechazar';
	@override String get open => 'Abrir la página correspondiente';
	@override String get approvedToast => 'Propuesta aprobada — documento actualizado';
	@override String get rejectedToast => 'Propuesta rechazada';
	@override String get failedToast => 'La acción falló';
}

// Path: web.cortex.blueprint.mode
class _TranslationsWebCortexBlueprintModeEs extends TranslationsWebCortexBlueprintModeEn {
	_TranslationsWebCortexBlueprintModeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get ai => 'IA';
	@override String get human => 'Humano';
	@override String get scanner => 'Escáner';
}

// Path: web.cortex.settings.injection
class _TranslationsWebCortexSettingsInjectionEs extends TranslationsWebCortexSettingsInjectionEn {
	_TranslationsWebCortexSettingsInjectionEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get title => 'Inyección al arranque';
	@override String get hint => 'Cuánto contexto de Cortex carga cada SESIÓN NUEVA por adelantado. El cambio aplica de inmediato a las sesiones creadas después — el backend nunca necesita reiniciarse.';
	@override String get active => 'activo';
	@override late final _TranslationsWebCortexSettingsInjectionModeEs mode = _TranslationsWebCortexSettingsInjectionModeEs._(_root);
	@override String get savedToast => 'Modo guardado — las sesiones nuevas lo usan de inmediato (sin reiniciar el backend)';
	@override String get saveFailed => 'Error al guardar';
	@override String get note => 'En modo completo siguen aplicando los flags de inyección por sección/página; en modo ligero las reglas fundacionales siempre se inyectan y el resto va al índice.';
}

// Path: sessions.inspector.shell.tabs
class _TranslationsSessionsInspectorShellTabsEs extends TranslationsSessionsInspectorShellTabsEn {
	_TranslationsSessionsInspectorShellTabsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get files => 'Archivos';
	@override String get git => 'Git';
	@override String get tasks => 'Tareas';
	@override String get history => 'Historial';
	@override String get vault => 'Bóveda';
	@override String get cortex => 'Cortex';
}

// Path: web.notes.vaultSync.conflict.kinds
class _TranslationsWebNotesVaultSyncConflictKindsEs extends TranslationsWebNotesVaultSyncConflictKindsEn {
	_TranslationsWebNotesVaultSyncConflictKindsEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get rebase => 'rebase';
	@override String get merge => 'merge';
	@override String get cherryPick => 'cherry-pick';
	@override String get operation => 'operación';
}

// Path: web.memoryAmbient.rules.row.summary
class _TranslationsWebMemoryAmbientRulesRowSummaryEs extends TranslationsWebMemoryAmbientRulesRowSummaryEn {
	_TranslationsWebMemoryAmbientRulesRowSummaryEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String afterMessages({required Object n}) => 'cada ${n} mensajes';
	@override String onIdle({required Object seconds}) => 'inactivo ≥ ${seconds}s';
	@override String kChars({required Object k}) => '≥ ${k} caracteres';
	@override String get manual => 'solo manual';
}

// Path: web.cortex.settings.injection.mode
class _TranslationsWebCortexSettingsInjectionModeEs extends TranslationsWebCortexSettingsInjectionModeEn {
	_TranslationsWebCortexSettingsInjectionModeEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebCortexSettingsInjectionModeLeanEs lean = _TranslationsWebCortexSettingsInjectionModeLeanEs._(_root);
	@override late final _TranslationsWebCortexSettingsInjectionModeFullEs full = _TranslationsWebCortexSettingsInjectionModeFullEs._(_root);
}

// Path: web.cortex.settings.injection.mode.lean
class _TranslationsWebCortexSettingsInjectionModeLeanEs extends TranslationsWebCortexSettingsInjectionModeLeanEn {
	_TranslationsWebCortexSettingsInjectionModeLeanEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Ligero — índice + bajo demanda (recomendado)';
	@override String get description => 'Inyecta solo las reglas fundacionales vinculantes más un índice compacto de secciones y páginas de conocimiento. Los agentes recuperan exactamente lo que necesitan vía doc_read / project_search. Ahorra tokens y evita ahogar las sesiones largas.';
}

// Path: web.cortex.settings.injection.mode.full
class _TranslationsWebCortexSettingsInjectionModeFullEs extends TranslationsWebCortexSettingsInjectionModeFullEn {
	_TranslationsWebCortexSettingsInjectionModeFullEs._(TranslationsEs root) : this._root = root, super.internal(root);

	final TranslationsEs _root; // ignore: unused_field

	// Translations
	@override String get label => 'Completo — inyectar todo';
	@override String get description => 'Inyecta al arranque cada sección y página marcada para inyección, completa (comportamiento clásico). Simple, pero cuesta tokens en cada sesión y satura la ventana de contexto.';
}

/// The flat map containing all translations for locale <es>.
/// Only for edge cases! For simple maps, use the map function of this library.
///
/// The Dart AOT compiler has issues with very large switch statements,
/// so the map is split into smaller functions (512 entries each).
extension on TranslationsEs {
	dynamic _flatMapFunction(String path) {
		return switch (path) {
			'common.ok' => 'OK',
			'common.cancel' => 'Cancelar',
			'common.save' => 'Guardar',
			'common.delete' => 'Eliminar',
			'common.edit' => 'Editar',
			'common.back' => 'Atrás',
			'common.done' => 'Hecho',
			'common.close' => 'Cerrar',
			'common.retry' => 'Reintentar',
			'common.copy' => 'Copiar',
			'common.enabled' => 'Activado',
			'common.refresh' => 'Actualizar',
			'common.clear' => 'Limpiar',
			'auth.signInTitle' => 'Iniciar sesión',
			'auth.changeServer' => 'Cambiar',
			'auth.username' => 'Usuario',
			'auth.password' => 'Contraseña',
			'auth.signIn' => 'Iniciar sesión',
			'auth.signingIn' => 'Iniciando sesión…',
			'auth.subtitle' => 'Usa tus credenciales de operador.',
			'auth.errorRequired' => 'El usuario y la contraseña son obligatorios',
			'auth.errorGeneric' => ({required Object error}) => 'Error al iniciar sesión: ${error}',
			'auth.errorFallback' => 'Error al iniciar sesión',
			'nav.sessions' => 'Sessions',
			'nav.memory' => 'Memoria',
			'nav.notes' => 'Notas',
			'nav.more' => 'Más',
			'nav.activity' => 'Actividad',
			'nav.providers' => 'Proveedores',
			'nav.channels' => 'Canales',
			'nav.integrations' => 'Integraciones',
			'nav.plugins' => 'Plugins',
			'nav.backups' => 'Copias de seguridad',
			'nav.settings' => 'Ajustes',
			'nav.workspace' => 'Espacio de trabajo',
			'nav.knowledge' => 'Conocimiento',
			'nav.vault' => 'Bóveda',
			'nav.cortex' => 'Cortex',
			'nav.updateAvailable' => 'Actualización disponible',
			'web.brand' => 'opendray',
			'web.loading' => 'Cargando…',
			'web.topbar.expandSidebar' => 'Expandir barra lateral',
			'web.topbar.collapseSidebar' => 'Contraer barra lateral',
			'web.topbar.search' => 'Buscar',
			'web.topbar.openPalette' => 'Abrir paleta de comandos',
			'web.topbar.theme' => 'Tema',
			'web.topbar.themeLabel' => ({required Object mode}) => 'Tema: ${mode}',
			'web.topbar.appearance' => 'Apariencia',
			'web.topbar.themeLight' => 'Claro',
			'web.topbar.themeDark' => 'Oscuro',
			'web.topbar.themeSystem' => 'Sistema',
			'web.topbar.language' => 'Idioma',
			'web.topbar.languageEnglish' => 'English',
			'web.topbar.languageChinese' => '中文',
			'web.topbar.languageSpanish' => 'Español',
			'web.topbar.signedInAs' => 'Sesión iniciada como',
			'web.topbar.tokenExpires' => 'El token caduca',
			'web.topbar.signOut' => 'Cerrar sesión',
			'web.sessions.list.title' => 'Sesiones',
			'web.sessions.list.countSeparator' => '·',
			'web.sessions.list.newAria' => 'Crear nueva session',
			'web.sessions.list.newTooltip' => 'Nueva session',
			'web.sessions.list.loading' => 'Cargando…',
			'web.sessions.list.emptyTitle' => 'Aún no hay sesiones.',
			'web.sessions.list.emptyHint' => ({required Object kbd}) => 'Pulsa ${kbd} para crear una.',
			'web.sessions.list.endedHeader' => ({required Object count}) => 'Finalizadas (${count})',
			'web.sessions.list.clearAll' => 'Borrar todas',
			'web.sessions.list.confirmClearAll' => ({required Object count}) => '¿Eliminar las ${count} sesiones finalizadas?',
			'web.sessions.list.confirmTerminate' => ({required Object name}) => '¿Terminar y eliminar ${name}?',
			'web.sessions.list.childPromoted' => ({required Object count}) => ' ${count} session de tarea secundaria pasará al nivel superior.',
			'web.sessions.list.childPromotedPlural' => ({required Object count}) => ' ${count} sesiones de tarea secundaria pasarán al nivel superior.',
			'web.sessions.list.footer' => ({required Object live, required Object ended}) => '${live} activas · ${ended} finalizadas',
			'web.sessions.list.row.deleteAria' => 'Eliminar session',
			'web.sessions.list.row.titleRemoveHistory' => 'Quitar del historial',
			'web.sessions.list.row.titleTerminate' => 'Terminar y eliminar',
			'web.sessions.list.row.titleRemove' => 'Eliminar',
			'web.sessions.list.row.claudeAccountTitle' => ({required Object label}) => 'Cuenta de Claude: ${label}',
			'web.sessions.list.deleteFailedToast' => 'Error al eliminar',
			'web.sessions.tabs.closeAria' => 'Cerrar pestaña y eliminar session',
			'web.sessions.tabs.closeTitle' => 'Cerrar pestaña y eliminar session',
			'web.sessions.page.removedToast' => 'Session eliminada',
			'web.sessions.page.removeFailedToast' => 'Error al eliminar',
			'web.sessions.page.stoppedToast' => 'Session detenida',
			'web.sessions.page.stopFailedToast' => 'Error al detener',
			'web.sessions.page.restartedToast' => 'Session reiniciada',
			'web.sessions.page.restartFailedToast' => 'Error al reiniciar',
			'web.sessions.page.confirmCloseTabTitle' => ({required Object name}) => '¿Detener y eliminar "${name}"?',
			'web.sessions.page.confirmCloseTabDescription' => 'El proceso de la CLI se terminará y la fila se eliminará.',
			'web.sessions.page.confirmCloseTabConfirm' => 'Detener y eliminar',
			'web.sessions.page.confirmRemoveTitle' => ({required Object name}) => '¿Eliminar ${name}?',
			'web.sessions.page.confirmRemoveTitleFallback' => '¿Eliminar session?',
			'web.sessions.page.confirmRemoveDescription' => 'Esto elimina la fila.',
			'web.sessions.page.confirmRemoveConfirm' => 'Eliminar',
			'web.sessions.empty.title' => 'Ninguna session abierta',
			'web.sessions.empty.hint' => ({required Object kbdN, required Object kbdW, required Object kbdRange}) => 'Elige una session de la lista o crea una nueva. Teclado: ${kbdN} nueva, ${kbdW} cerrar, ${kbdRange} cambiar.',
			'web.sessions.empty.spawn' => 'Crear session',
			'web.sessions.header.loadingSession' => 'Cargando session…',
			'web.sessions.header.showList' => 'Mostrar lista de sesiones',
			'web.sessions.header.hideList' => 'Ocultar lista de sesiones',
			'web.sessions.header.showInspector' => 'Mostrar inspector',
			'web.sessions.header.hideInspector' => 'Ocultar inspector',
			'web.sessions.header.attachImage' => 'Adjuntar imagen',
			'web.sessions.header.attachImageTooltip' => 'Adjuntar imagen (o pega / suelta en el terminal)',
			'web.sessions.header.copyOutput' => 'Copiar salida',
			'web.sessions.header.copyOutputTooltip' => 'Copia la salida del terminal (la selección si hay alguna, si no, todo)',
			'web.sessions.header.restart' => 'Reiniciar',
			'web.sessions.header.restarting' => 'Reiniciando…',
			'web.sessions.header.remove' => 'Eliminar',
			'web.sessions.header.removing' => 'Eliminando…',
			'web.sessions.header.stop' => 'Detener',
			'web.sessions.header.stopping' => 'Deteniendo…',
			'web.sessions.header.pid' => ({required Object pid}) => 'pid ${pid}',
			'web.sessions.terminal.uploadingToast' => 'Subiendo imagen…',
			'web.sessions.terminal.uploadedToast' => 'Imagen adjuntada',
			'web.sessions.terminal.uploadFailedToast' => 'Error al subir',
			'web.sessions.terminal.uploadInvalidTypeToast' => 'Solo se pueden adjuntar archivos de imagen',
			'web.sessions.terminal.dropToAttach' => 'Suelta la imagen para adjuntarla',
			'web.sessions.terminal.copyButton' => 'Copiar',
			'web.sessions.terminal.copyAllTooltip' => 'Copia la salida del terminal al portapapeles (la selección si hay alguna, si no, todo)',
			'web.sessions.terminal.copySelection' => 'Copiar',
			'web.sessions.terminal.copySelectionTooltip' => 'Copiar el texto seleccionado',
			'web.sessions.terminal.copiedToast' => 'Copiado al portapapeles',
			'web.sessions.terminal.copyEmptyToast' => 'Aún no hay nada que copiar',
			'web.sessions.terminal.copyFailedToast' => 'No se pudo copiar al portapapeles',
			'web.sessions.terminal.urls.tooltip' => 'Abre el último enlace detectado en esta session',
			'web.sessions.terminal.urls.tapToOpenLatest' => 'Toca para abrir el último enlace (la URL de OAuth más reciente)',
			'web.sessions.terminal.urls.openListTooltip' => 'Mostrar todos los enlaces',
			'web.sessions.terminal.urls.buttonLabel' => ({required Object count}) => '${count} enlace',
			'web.sessions.terminal.urls.buttonLabel_plural' => ({required Object count}) => '${count} enlaces',
			'web.sessions.terminal.urls.dialogTitle' => 'Enlaces detectados',
			'web.sessions.terminal.urls.dialogDesc' => 'URLs impresas en la salida de esta session, las más recientes primero. Toca Abrir para lanzarla en tu navegador predeterminado. Funciona incluso cuando la URL se divide en varias líneas en el terminal.',
			'web.sessions.terminal.urls.openButton' => 'Abrir',
			'web.sessions.terminal.urls.copyButton' => 'Copiar',
			'web.sessions.terminal.urls.copiedToast' => 'URL copiada',
			'web.sessions.terminal.urls.copyFailedToast' => 'No se pudo copiar. Mantén pulsada la URL y cópiala manualmente',
			'web.sessions.terminal.urls.noneHint' => 'Aún no se han detectado enlaces.',
			'web.sessions.spawn.title' => 'Crear session',
			'web.sessions.spawn.description' => 'Inicia una session de la CLI con un proveedor registrado.',
			'web.sessions.spawn.provider' => 'Proveedor',
			'web.sessions.spawn.claudeAccount' => 'Cuenta de Claude',
			'web.sessions.spawn.loadingAccounts' => 'Cargando cuentas…',
			'web.sessions.spawn.noAccounts' => 'No se han encontrado cuentas de Claude. Crea esta session y ejecuta <1>claude login</1> en el terminal. Las credenciales acaban en <3>~/.claude</3> en el gateway y aparecen automáticamente la próxima vez.',
			'web.sessions.spawn.kDefault' => 'Predeterminada',
			'web.sessions.spawn.defaultTooltip' => 'Usar el keychain del sistema / env',
			'web.sessions.spawn.tokenEmptyBadge' => '·vacío',
			'web.sessions.spawn.tokenMissingTooltip' => 'No hay token configurado. Configura el token primero en el panel de Proveedores',
			'web.sessions.spawn.multiAccountHint' => 'Hay varias cuentas configuradas. Elige una para esta session.',
			'web.sessions.spawn.cwd' => 'Directorio de trabajo',
			'web.sessions.spawn.cwdPlaceholder' => '/Users/you/projects/foo',
			'web.sessions.spawn.browse' => 'Examinar',
			'web.sessions.spawn.nameLabel' => 'Nombre (opcional)',
			'web.sessions.spawn.namePlaceholder' => 'claude in pet-tracker',
			'web.sessions.spawn.argsLabel' => 'Argumentos de la CLI (uno por línea)',
			'web.sessions.spawn.bypassClaude' => 'Omitir las confirmaciones de permisos',
			'web.sessions.spawn.bypassCodex' => 'Omitir aprobaciones y sandbox (--dangerously-bypass-approvals-and-sandbox)',
			'web.sessions.spawn.bypassGemini' => 'Modo YOLO (--yolo)',
			'web.sessions.spawn.bypassAntigravity' => 'Omitir permisos / YOLO (--dangerously-skip-permissions)',
			'web.sessions.spawn.bypassOnHint' => 'Esta session se ejecutará con autonomía elevada.',
			'web.sessions.spawn.bypassOffHint' => 'Desactivado. Las confirmaciones y los prompts se comportan con normalidad.',
			'web.sessions.spawn.errorPickProvider' => 'Elige un proveedor.',
			'web.sessions.spawn.errorCwdRequired' => 'cwd es obligatorio.',
			'web.sessions.spawn.cancel' => 'Cancelar',
			'web.sessions.spawn.submit' => 'Crear',
			'web.sessions.spawn.submitting' => 'Creando…',
			'web.sessions.spawn.spawnedToast' => 'Session creada',
			'web.sessions.spawn.spawnedDescription' => ({required Object provider, required Object pid}) => '${provider} · pid ${pid}',
			'web.sessions.spawn.pidFallback' => '—',
			'web.sessions.accountSwitcher.tooltip' => 'Cambiar de cuenta de Claude (reinicia el proceso de la CLI)',
			'web.sessions.accountSwitcher.currentDefault' => 'predeterminada',
			'web.sessions.accountSwitcher.menuTitle' => 'Cambiar de cuenta de Claude',
			'web.sessions.accountSwitcher.defaultName' => 'Predeterminada',
			'web.sessions.accountSwitcher.defaultSubtitle' => 'keychain del sistema / env de la CLI',
			'web.sessions.accountSwitcher.tokenEmpty' => '·vacío',
			'web.sessions.accountSwitcher.confirmSwitch' => 'Cambiar de cuenta reiniciará la CLI de Claude. El historial de la conversación se conserva (el transcript se migra y --resume lo reproduce con la nueva cuenta), pero se perderá cualquier ejecución de herramienta en curso o entrada sin enviar. ¿Continuar?',
			'web.sessions.accountSwitcher.switchedToast' => 'Cuenta cambiada',
			'web.sessions.accountSwitcher.switchedDescription' => ({required Object account, required Object pid}) => 'Ahora usando @${account} · pid ${pid}',
			'web.sessions.accountSwitcher.switchedDefault' => 'predeterminada',
			'web.sessions.accountSwitcher.switchFailedToast' => 'Error al cambiar',
			'web.sessions.inspector.tabs.files' => 'Archivos',
			'web.sessions.inspector.tabs.git' => 'Git',
			'web.sessions.inspector.tabs.search' => 'Buscar',
			'web.sessions.inspector.tabs.tasks' => 'Tareas',
			'web.sessions.inspector.tabs.history' => 'Historial',
			'web.sessions.inspector.tabs.vault' => 'Bóveda',
			'web.sessions.inspector.tabs.cortex' => 'Cortex',
			'web.sessions.inspector.vaultPanel.open' => 'Abrir Bóveda',
			'web.sessions.inspector.vaultPanel.projectDocs' => 'Docs del proyecto',
			'web.sessions.inspector.vaultPanel.projectDocsHint' => 'Docs del proyecto escritos por el agente en la bóveda. Revincula la carpeta si las notas de este proyecto viven en otro sitio.',
			'web.sessions.inspector.vaultPanel.pinnedHint' => 'Vinculado a una carpeta de bóveda personalizada para este proyecto.',
			'web.sessions.inspector.vaultPanel.bind' => 'Vincular',
			'web.sessions.inspector.vaultPanel.changeLocation' => 'Cambiar la carpeta de bóveda vinculada a este proyecto',
			'web.sessions.inspector.vaultPanel.newDoc' => 'Nuevo doc',
			'web.sessions.inspector.vaultPanel.cancel' => 'Cancelar',
			'web.sessions.inspector.vaultPanel.create' => 'Crear',
			'web.sessions.inspector.vaultPanel.filenamePlaceholder' => 'archivo.md',
			'web.sessions.inspector.vaultPanel.noDocs' => 'Aún no hay docs del proyecto en esta carpeta de la bóveda.',
			'web.sessions.inspector.vaultPanel.createFailed' => 'No se pudo crear el doc',
			'web.sessions.inspector.vaultPanel.mappingTitle' => 'Vincular carpeta de bóveda del proyecto',
			'web.sessions.inspector.vaultPanel.mappingHelp' => 'Elige la carpeta de la bóveda que contiene las notas de este proyecto. Relativa a la bóveda, p. ej. projects/my-app. Déjalo vacío para usar el valor por defecto.',
			'web.sessions.inspector.vaultPanel.sessionCwd' => 'cwd de la sesión',
			'web.sessions.inspector.vaultPanel.folderLabel' => 'Carpeta de la bóveda',
			'web.sessions.inspector.vaultPanel.mappingStoredHint' => 'Se guarda en la bóveda en .opendray-projects.json, así se sincroniza con tus notas.',
			'web.sessions.inspector.vaultPanel.save' => 'Guardar',
			'web.sessions.inspector.vaultPanel.clearOverride' => 'Borrar anulación',
			'web.sessions.inspector.vaultPanel.boundToast' => 'Carpeta de bóveda del proyecto vinculada',
			'web.sessions.inspector.vaultPanel.clearedToast' => 'Anulación borrada — usando la carpeta por defecto',
			'web.sessions.inspector.vaultPanel.saveFailed' => 'No se pudo guardar el mapeo',
			'web.sessions.inspector.cortexPanel.noCwd' => 'La sesión no tiene cwd — las funciones de Cortex necesitan un directorio de trabajo.',
			'web.sessions.inspector.cortexPanel.open' => 'Abrir espacio Cortex',
			'web.sessions.inspector.cortexPanel.docs' => 'Docs',
			'web.sessions.inspector.cortexPanel.journal' => 'Diario',
			'web.sessions.inspector.cortexPanel.inbox' => 'Entrada',
			'web.sessions.inspector.cortexPanel.archived' => 'Archivados',
			'web.sessions.inspector.cortexPanel.pending' => 'pendiente',
			'web.sessions.inspector.cortexPanel.goal' => 'Objetivo',
			'web.sessions.inspector.cortexPanel.plan' => 'Plan',
			'web.sessions.inspector.cortexPanel.latestJournal' => 'Último diario',
			'web.sessions.inspector.cortexPanel.empty' => 'Aún no se ha capturado memoria de Cortex para este proyecto. Inicia una sesión o define un objetivo para poblarla.',
			'web.sessions.ended.bufferUnavailable' => '[búfer no disponible]',
			'web.sessions.ended.readOnlyBanner' => '[session finalizada. búfer de solo lectura]',
			'web.sessions.fileBrowser.title' => 'Elige el directorio de trabajo',
			'web.sessions.fileBrowser.description' => 'Examina el sistema de archivos del host del gateway y elige una carpeta.',
			'web.sessions.fileBrowser.parent' => 'Directorio superior',
			'web.sessions.fileBrowser.home' => 'Directorio personal',
			'web.sessions.fileBrowser.refresh' => 'Actualizar',
			'web.sessions.fileBrowser.pathPlaceholder' => '/Users/you/projects',
			'web.sessions.fileBrowser.loading' => 'Cargando…',
			'web.sessions.fileBrowser.empty' => 'Directorio vacío.',
			'web.sessions.fileBrowser.newFolder' => 'Nueva carpeta',
			'web.sessions.fileBrowser.newFolderPlaceholder' => 'nombre-de-la-carpeta',
			'web.sessions.fileBrowser.create' => 'Crear',
			'web.sessions.fileBrowser.cancel' => 'Cancelar',
			'web.sessions.fileBrowser.useThisFolder' => 'Usar esta carpeta',
			'web.sessions.fileBrowser.createdToast' => 'Directorio creado',
			'web.sessions.fileBrowser.mkdirFailedToast' => 'Error al crear el directorio',
			'web.sessions.fileBrowser.homeFailedToast' => 'Error al leer el directorio personal',
			'web.memory.title' => 'Memoria',
			'web.memory.subtitle' => 'Explora, busca y edita las memorias que los agentes han almacenado a través del servidor MCP opendray-memory.',
			'web.memory.navProject' => 'Proyecto',
			'web.memory.navArchived' => 'Archivadas',
			'web.memory.navWorkers' => 'Ajustes de Cortex',
			'web.memory.navConfiguration' => 'Almacenamiento y embedder →',
			'web.memory.navQuarantine' => 'Cuarentena',
			'web.journalStale.title' => 'Purgar entradas obsoletas',
			'web.journalStale.subtitle' => ({required Object days}) => '(con más de ${days} días, sin conflictos pendientes)',
			'web.journalStale.daysLabel' => 'Con más de (días):',
			'web.journalStale.loading' => 'Escaneando…',
			'web.journalStale.empty' => 'No hay nada obsoleto que purgar.',
			'web.journalStale.selectAll' => 'Seleccionar todo',
			'web.journalStale.deselectAll' => 'Deseleccionar todo',
			'web.journalStale.deleteSelected' => ({required Object count}) => 'Eliminar (${count})',
			'web.journalStale.deleted_one' => ({required Object count}) => '${count} entrada eliminada',
			'web.journalStale.deleted_other' => ({required Object count}) => '${count} entradas eliminadas',
			'web.conflicts.title' => 'Conflictos entre capas',
			'web.conflicts.subtitle' => 'Contradicciones que el detector diario encontró entre hechos, plan, objetivo y entradas del diario.',
			'web.conflicts.loading' => 'Cargando conflictos…',
			'web.conflicts.empty' => 'No hay conflictos pendientes. Haz clic en "Detectar ahora" para ejecutar un análisis bajo demanda.',
			'web.conflicts.pickCwd' => 'Elige un proyecto para ver sus conflictos.',
			'web.conflicts.detectNow' => 'Detectar ahora',
			'web.conflicts.detected' => ({required Object count}) => 'Se encontraron ${count} conflicto(s) nuevo(s)',
			'web.conflicts.accept' => 'Aceptar',
			'web.conflicts.dismiss' => 'Descartar',
			'web.conflicts.accepted' => 'Conflicto aceptado. Recuerda aplicar la corrección',
			'web.conflicts.dismissed' => 'Conflicto descartado',
			'web.conflicts.deletedFact' => 'Hecho eliminado y conflicto aceptado',
			'web.conflicts.quickActions' => 'Corrección:',
			'web.conflicts.deleteFact' => 'Eliminar hecho',
			'web.conflicts.deleteFactSide' => ({required Object side, required Object ref}) => 'Eliminar ${side}: ${ref}',
			'web.conflicts.confirmDelete.title' => ({required Object side}) => '¿Eliminar el hecho ${side}?',
			'web.conflicts.confirmDelete.description' => 'Esto elimina el hecho de forma permanente y acepta el conflicto. El otro lado se conserva como la afirmación superviviente.',
			'web.conflicts.confirmDelete.targetLabel' => ({required Object side}) => 'Se eliminará (lado ${side}):',
			'web.conflicts.confirmDelete.keepLabel' => ({required Object side}) => 'Se conservará (lado ${side}):',
			'web.conflicts.confirmDelete.nonFactOther' => ({required Object layer}) => '(entrada de ${layer}, abre la pestaña correspondiente para inspeccionar)',
			'web.conflicts.confirmDelete.evidenceLabel' => 'Evidencia del detector:',
			'web.conflicts.confirmDelete.loading' => 'Cargando texto del hecho…',
			'web.conflicts.confirmDelete.loadError' => 'No se pudo cargar el texto del hecho. Inspecciónalo en la página de Memoria.',
			'web.conflicts.confirmDelete.cancel' => 'Cancelar',
			'web.conflicts.confirmDelete.confirm' => ({required Object side}) => 'Eliminar ${side}',
			'web.conflicts.openLayer.plan' => 'Abrir editor de plan',
			'web.conflicts.openLayer.goal' => 'Abrir editor de objetivo',
			'web.conflicts.severity.low' => 'baja',
			'web.conflicts.severity.medium' => 'media',
			'web.conflicts.severity.high' => 'alta',
			'web.memoryHealth.title' => ({required Object days}) => 'Estado de la memoria, últimos ${days} días',
			'web.memoryHealth.subtitle' => 'Señales agregadas de ambos subsistemas de memoria de este proyecto.',
			'web.memoryHealth.loading' => 'Cargando instantánea de estado…',
			'web.memoryHealth.errorLoading' => 'No se pudo cargar la instantánea de estado.',
			'web.memoryHealth.pickCwd' => 'Elige un proyecto para ver el estado de su memoria.',
			'web.memoryHealth.newFacts' => 'Datos nuevos',
			'web.memoryHealth.newFactsHint' => ({required Object total}) => '${total} almacenados en total',
			'web.memoryHealth.captureFires' => 'Capturas activadas',
			'web.memoryHealth.captureFiresHint' => ({required Object stored, required Object deduped}) => '${stored} almacenadas · ${deduped} deduplicadas',
			'web.memoryHealth.newJournal' => 'Entradas de diario',
			'web.memoryHealth.newJournalHint' => ({required Object total}) => '${total} en total',
			'web.memoryHealth.planAge' => 'Plan actualizado por última vez',
			'web.memoryHealth.planAgeHint' => ({required Object count}) => '${count} propuesta(s) de desvío de plan pendiente(s)',
			'web.memoryHealth.planAgeHintNone' => 'No hay propuestas de desvío de plan pendientes',
			'web.memoryHealth.goalAge' => 'Objetivo actualizado por última vez',
			'web.memoryHealth.pending' => 'Propuestas pendientes',
			'web.memoryHealth.pendingHint' => ({required Object days}) => 'la más antigua, ${days}d de antigüedad',
			'web.memoryHealth.topHit' => ({required Object hits}) => 'Más consultado · ${hits} recuperaciones',
			'web.memoryHealth.zeroHit' => ({required Object count}) => '${count} datos con más de 7d sin ninguna recuperación, candidatos para limpieza.',
			'web.memoryHealth.never' => 'nunca',
			'web.memoryHealth.today' => 'hoy',
			'web.memoryHealth.daysAgo_one' => ({required Object count}) => 'hace ${count} día',
			'web.memoryHealth.daysAgo_other' => ({required Object count}) => 'hace ${count} días',
			'web.memoryConfig.title' => 'Ajustes de Cortex',
			'web.memoryConfig.subtitle' => 'Todos los mandos de runtime del ciclo de IA en un solo lugar — inyección de spawn, providers LLM, workers por tarea, triggers de captura, perfiles de inyección, costes de tokens. Los cambios aplican al instante; sin reinicio.',
			'web.memoryConfig.sections.providers' => 'Providers',
			'web.memoryConfig.sections.workers' => 'Workers',
			'web.memoryConfig.sections.rules' => 'Reglas de captura',
			'web.memoryConfig.sections.profiles' => 'Perfiles de inyección',
			'web.memoryConfig.sections.costs' => 'Coste en tokens',
			'web.memoryConfig.sectionHints.providers' => 'Endpoints HTTP registrados (Ollama / LM Studio / Anthropic / OpenAI / Integration) a los que cualquier tarea puede despachar.',
			'web.memoryConfig.sectionHints.workers' => 'Para cada punto de contacto elige un provider HTTP (barato, local) o un Agent headless de Claude / Gemini (mayor calidad, consume tokens de CLI).',
			'web.memoryConfig.sectionHints.rules' => 'Cuándo se activa el motor de captura en cada session (tras N mensajes / al estar inactivo / K caracteres / manual). Las reglas sin un provider fijado siguen el ajuste del worker de Captura de arriba.',
			'web.memoryConfig.sectionHints.profiles' => 'Cómo se inyectan las memorias previas en el prompt del sistema del agente al iniciar la session (recencia, relevancia, híbrido o desactivado).',
			'web.memoryConfig.sectionHints.costs' => 'Gasto agregado reconstruido a partir de memory_summarizer_calls. Los providers locales (Ollama, LM Studio, Integration) son gratuitos; los providers en la nube muestran el coste real.',
			'web.memoryConfig.moveBanner.title' => 'La configuración de memoria se ha movido',
			'web.memoryConfig.moveBanner.body' => 'Todos los ajustes relacionados con la memoria (providers / reglas de captura / perfiles de inyección / coste) ahora conviven con Workers en una sola página para que los ajustes relacionados estén juntos.',
			'web.memoryConfig.moveBanner.openButton' => 'Abrir Configuración de memoria →',
			'web.memoryConfig.infra.title' => 'Almacenamiento y embedder (infraestructura)',
			'web.memoryConfig.infra.hint' => 'La otra mitad de la config de memoria — backend de embeddings, ajuste de recuperación, puertas de gatekeeper/cleaner y el flag del grafo de conocimiento — vive en Server Settings y requiere reinicio.',
			'web.memoryConfig.infra.openSettings' => 'Server Settings →',
			'web.memoryConfig.infra.embedder' => 'embedder',
			'web.memoryConfig.infra.gatekeeper' => 'gatekeeper',
			'web.memoryConfig.infra.cleaner' => 'cleaner',
			'web.memoryConfig.infra.knowledge' => 'grafo de conocimiento',
			'web.memoryConfig.infra.on' => 'on',
			'web.memoryConfig.infra.off' => 'off',
			'web.memoryWorkers.title' => 'Workers de memoria',
			'web.memoryWorkers.loading' => 'Cargando configuración de workers…',
			'web.memoryWorkers.errorTitle' => 'No se puede acceder al endpoint.',
			'web.memoryWorkers.errorDescription' => 'Las rutas /api/v1/memory/workers son nuevas en M25. Puede que el binario de opendray necesite un reinicio para montarlas y ejecutar la migración 0029.',
			'web.memoryWorkers.intro' => 'Cada punto de contacto del LLM con el sistema de memoria puede atenderse de forma independiente mediante el endpoint local <1>summarizer</1> (LM Studio / compatible con OpenAI) o lanzando un <3>agente Claude / Gemini</3> sin interfaz en modo <5>--print</5>. Las tareas narrativas de alta calidad (gitactivity, transcript) se benefician de los workers de agente; las tareas de alta frecuencia (gatekeeper) permanecen en el endpoint local por diseño.',
			'web.memoryWorkers.enabledBadge' => 'habilitado',
			'web.memoryWorkers.disabledBadge' => 'deshabilitado',
			'web.memoryWorkers.summarizerOnlyBadge' => 'solo-summarizer',
			'web.memoryWorkers.callsCount' => ({required Object count}) => '${count} llamadas · 24h',
			'web.memoryWorkers.avgMs' => ({required Object ms}) => 'media ${ms}ms',
			'web.memoryWorkers.errorsCount' => ({required Object count}) => '${count} errores',
			'web.memoryWorkers.workerLabel' => 'Worker',
			'web.memoryWorkers.summarizerHttp' => 'Summarizer (HTTP)',
			'web.memoryWorkers.agentCliPrint' => 'Agente (CLI --print)',
			'web.memoryWorkers.summarizerProviderLabel' => 'Proveedor del summarizer',
			'web.memoryWorkers.registryDefault' => 'Predeterminado del registro',
			'web.memoryWorkers.cliLabel' => 'CLI',
			'web.memoryWorkers.selectPlaceholder' => 'Seleccionar',
			'web.memoryWorkers.cliClaude' => 'Claude',
			'web.memoryWorkers.cliGemini' => 'Gemini',
			'web.memoryWorkers.claudeAccountLabel' => 'Cuenta de Claude',
			'web.memoryWorkers.claudeAccountDefault' => 'Predeterminada',
			'web.memoryWorkers.agentWarning' => 'El modo agente lanza un CLI sin interfaz por cada llamada. La latencia sube de <1>~1s</1> (summarizer) a <3>~5-15s</3>; el coste pasa de la CPU a tu quota de Claude/Gemini.',
			'web.memoryWorkers.enabledCheckbox' => 'Habilitado',
			'web.memoryWorkers.testButton' => 'Probar',
			'web.memoryWorkers.saveButton' => 'Guardar',
			'web.memoryWorkers.recentCalls' => ({required Object count}) => 'Llamadas recientes (${count})',
			'web.memoryWorkers.tableWhen' => 'cuándo',
			'web.memoryWorkers.tableWorker' => 'worker',
			'web.memoryWorkers.tableMs' => 'ms',
			'web.memoryWorkers.tableOk' => 'ok',
			'web.memoryWorkers.savedToast' => ({required Object label}) => '${label} actualizado',
			'web.memoryWorkers.saveFailedToast' => 'Error al guardar',
			'web.memoryWorkers.testOkToast' => ({required Object label, required Object ms}) => '${label} OK. ${ms}ms',
			'web.memoryWorkers.testFailedToast' => ({required Object label}) => '${label} falló',
			'web.memoryWorkers.testCallFailedToast' => 'La llamada de prueba falló',
			'web.memoryWorkers.unknownError' => 'error desconocido',
			'web.memoryWorkers.tasks.gatekeeper.label' => 'Gatekeeper',
			'web.memoryWorkers.tasks.gatekeeper.description' => 'Filtro previo a la escritura en cada memory_store. Alta frecuencia (objetivo <500ms), solo-summarizer.',
			'web.memoryWorkers.tasks.gatekeeper.modelAdvice' => 'Juicio sí/no de alta frecuencia — un modelo ligero (haiku / flash-lite / codex-mini / local) basta.',
			'web.memoryWorkers.tasks.cleaner.label' => 'Bibliotecario de limpieza',
			'web.memoryWorkers.tasks.cleaner.description' => 'Bibliotecario LLM periódico. Evalúa los recuerdos antiguos como conservar / obsoleto / duplicado.',
			'web.memoryWorkers.tasks.cleaner.modelAdvice' => 'Veredictos por lotes sobre hechos viejos — modelo ligero recomendado; corre programado.',
			'web.memoryWorkers.tasks.gitactivity.label' => 'Resumidor de actividad de git',
			'web.memoryWorkers.tasks.gitactivity.description' => 'git log → narrativa de 2-3 párrafos cada 24h. Encaja de forma natural con un worker de agente.',
			'web.memoryWorkers.tasks.gitactivity.modelAdvice' => 'Resumen narrativo del historial git — un modelo equilibrado (sonnet / flash) se lee mejor.',
			'web.memoryWorkers.tasks.transcript.label' => 'Resumidor de transcript de sesión',
			'web.memoryWorkers.tasks.transcript.description' => 'Resumen al final de la sesión sobre "qué hizo el agente". Encaja de forma natural con un worker de agente.',
			'web.memoryWorkers.tasks.transcript.modelAdvice' => 'Resúmenes de sesión — modelo equilibrado recomendado; alimenta el diario y la detección de deriva.',
			'web.memoryWorkers.tasks.plan_drift.label' => 'Detector de desviación del plan',
			'web.memoryWorkers.tasks.plan_drift.description' => 'Al terminar cada sesión, comprueba si el plan del proyecto necesita actualizarse y presenta una propuesta. Encaja con un worker de agente para un razonamiento más completo.',
			'web.memoryWorkers.tasks.plan_drift.modelAdvice' => 'Reescribe goal/plan/secciones — exige criterio; un modelo fuerte (sonnet/opus) evita malas actualizaciones.',
			'web.memoryWorkers.tasks.conflict_detector.label' => 'Detector de conflictos entre capas',
			'web.memoryWorkers.tasks.conflict_detector.description' => 'Escaneo diario que encuentra contradicciones entre hechos / plan / objetivo / diario. Un modelo de mayor calidad = menos falsos positivos.',
			'web.memoryWorkers.tasks.conflict_detector.modelAdvice' => 'Escaneo diario de contradicciones — un modelo equilibrado basta.',
			'web.memoryWorkers.tasks.capture.label' => 'Motor de captura',
			'web.memoryWorkers.tasks.capture.description' => 'Extracción de hechos por cada trigger a partir de los transcripts de sesión. El modo agente ofrece hechos notablemente mejores en sesiones largas; el modo summarizer es barato y local.',
			'web.memoryWorkers.tasks.capture.modelAdvice' => 'La tarea más frecuente: extracción de hechos — usa el modelo MÁS BARATO que funcione (haiku / local).',
			'web.memoryWorkers.tasks.blueprint.modelAdvice' => 'Clasificación ocasional del proyecto — modelo equilibrado; aquí la calidad importa más que el costo.',
			'web.memoryWorkers.tasks.blueprint.label' => 'Proponedor de planos',
			'web.memoryWorkers.tasks.blueprint.description' => 'Clasifica un proyecto y propone su conjunto de secciones. Disparado por el operador.',
			'web.memoryWorkers.tasks.curation.modelAdvice' => 'Tu editor conversacional de docs/políticas — modelo fuerte recomendado (sonnet/opus).',
			'web.memoryWorkers.tasks.curation.label' => 'Chat de curación',
			'web.memoryWorkers.tasks.curation.description' => 'Impulsa el canal conversacional que actualiza secciones y re-redacta páginas de conocimiento.',
			'web.memoryWorkers.modelLabel' => 'Modelo',
			'web.memoryWorkers.modelHint' => 'Fija el modelo del CLI para esta tarea (p. ej. haiku para tareas básicas). Vacío = predeterminado del CLI.',
			'web.memoryWorkers.modelCliDefault' => 'Predeterminado del CLI (último)',
			'web.memoryWorkers.modelCustom' => 'Personalizado…',
			'web.memoryWorkers.modelCustomPlaceholder' => 'id exacto del modelo',
			'web.memoryWorkers.modelBackToList' => 'Lista',
			'web.memoryWorkers.cliCodex' => 'Codex (codex exec)',
			'web.memoryWorkers.cliAntigravity' => 'Antigravity (agy --print)',
			'web.memoryWorkers.infraGateOff' => ({required Object label}) => 'El enrutado de ${label} está guardado, pero su puerta de función está APAGADA en Server Settings — no se ejecutará nada hasta que la actives allí.',
			'web.memoryWorkers.infraGateOpen' => 'Activarla',
			'web.memoryWorkers.providerModel' => 'modelo:',
			'web.archived.loading' => 'Cargando…',
			'web.archived.emptyTitle' => 'Nada archivado',
			'web.archived.emptyDescription' => 'No hay memorias archivadas en ningún proyecto. El limpiador automático archiva aquí los hechos obsoletos y duplicados (restaurables durante 30 días); todavía no se ha eliminado nada.',
			'web.archived.title' => 'Memorias archivadas',
			'web.archived.subtitle' => 'Memorias archivadas (reversibles) hasta que la ventana de gracia de 30 días las purgue. Fuentes: los veredictos del auto-cleaner, el archivado manual por memoria, y los proyectos que archivas — las memorias de un proyecto llegan juntas y vuelven juntas al desarchivarlo (los archivados de proyecto están exentos de la purga).',
			'web.archived.globalScope' => '(global)',
			'web.archived.summary' => ({required Object projects, required Object memories}) => '${projects} proyectos · ${memories} memorias archivadas',
			'web.archived.memCount' => ({required Object count}) => '${count} memorias',
			'web.archived.restoreAll' => 'Restaurar todo',
			'web.archived.restoreAllTooltip' => 'Restaurar todas las memorias archivadas de este proyecto',
			'web.archived.restoreAllConfirm' => ({required Object count, required Object project}) => '¿Restaurar las ${count} memorias archivadas de ${project}?',
			'web.archived.restoredAllToast' => ({required Object count}) => '${count} memorias restauradas',
			'web.archived.deleteButton' => 'Eliminar',
			'web.archived.deleteTooltip' => 'Eliminar permanentemente ahora — omite la ventana de gracia de 30 días, no se puede deshacer',
			'web.archived.deleteConfirm' => '¿Eliminar permanentemente esta memoria ahora? Omite la ventana de gracia de 30 días y no se puede deshacer.',
			'web.archived.deletedToast' => 'Eliminada permanentemente',
			'web.archived.deleteFailedToast' => 'Error al eliminar',
			'web.archived.deleteAll' => 'Eliminar todo',
			'web.archived.deleteAllTooltip' => 'Eliminar permanentemente ahora todas las memorias archivadas de este proyecto',
			'web.archived.deleteAllConfirm' => ({required Object count, required Object project}) => '¿Eliminar permanentemente ahora las ${count} memorias archivadas de ${project}? Omite la ventana de gracia de 30 días y no se puede deshacer.',
			'web.archived.deletedAllToast' => ({required Object count}) => '${count} memorias eliminadas',
			'web.archived.openProject' => 'Abrir proyecto',
			'web.archived.archivedAtPrefix' => 'Archivado',
			'web.archived.restoreButton' => 'Restaurar',
			'web.archived.restoredToast' => 'Restaurado',
			'web.archived.restoreFailedToast' => 'Error al restaurar',
			'web.project.picker.title' => 'Elige un proyecto',
			'web.project.picker.subtitle' => 'La memoria del proyecto se delimita por el directorio de trabajo. Elige uno para gestionar su objetivo, plan, diario y cola de limpieza.',
			'web.project.picker.pathPlaceholder' => '/path/to/your/project',
			'web.project.picker.browse' => 'Examinar',
			'web.project.picker.browseTooltip' => 'Examina el sistema de archivos del host del gateway',
			'web.project.picker.open' => 'Abrir',
			'web.project.picker.recentLabel' => 'Proyectos recientes (desde la memoria almacenada):',
			'web.project.picker.orphanTooltip' => 'Parece un scope_key truncado (antiguo error de importación del mirror). Puede que no tenga documentos del proyecto.',
			'web.project.picker.orphanBadge' => 'huérfano',
			'web.project.noCwd' => 'Elige un proyecto para gestionar su memoria.',
			'web.project.header.docsCount_one' => ({required Object count}) => '${count} documento',
			'web.project.header.docsCount_other' => ({required Object count}) => '${count} documentos',
			'web.project.header.journalEntries_one' => ({required Object count}) => '${count} entrada del diario',
			'web.project.header.journalEntries_other' => ({required Object count}) => '${count} entradas del diario',
			'web.project.header.pendingProposals_one' => ({required Object count}) => '${count} propuesta pendiente',
			'web.project.header.pendingProposals_other' => ({required Object count}) => '${count} propuestas pendientes',
			'web.project.header.archivedCount' => ({required Object count}) => '${count} archivadas',
			'web.project.tabs.health' => 'Estado',
			'web.project.tabs.goal' => 'Objetivo',
			'web.project.tabs.plan' => 'Plan',
			'web.project.tabs.tech' => 'Tecnología',
			'web.project.tabs.activity' => 'Actividad',
			'web.project.tabs.journal' => 'Diario',
			'web.project.tabs.inbox' => 'Bandeja de entrada',
			'web.project.tabs.conflicts' => 'Conflictos',
			'web.project.tabs.archived' => 'Archivadas',
			'web.project.tabs.overview' => 'Resumen',
			'web.project.tabs.hygiene' => 'Higiene',
			'web.project.docLabel.goal' => 'Objetivo',
			'web.project.docLabel.plan' => 'Plan',
			'web.project.docLabel.tech_stack' => 'Stack tecnológico',
			'web.project.docLabel.recent_activity' => 'Actividad reciente',
			'web.project.editor.updatedBy' => 'Actualizado por',
			'web.project.editor.noDocSet' => ({required Object label}) => 'Aún no se ha definido ningún ${label}.',
			'web.project.editor.save' => 'Guardar',
			'web.project.editor.saveFailedToast' => 'Error al guardar',
			'web.project.editor.savedToast' => ({required Object label}) => '${label} guardado',
			'web.project.editor.goalPlaceholder' => '¿Qué estamos construyendo? Un párrafo. Lo lee cada agente al iniciarse.',
			'web.project.editor.planPlaceholder' => 'Plan activo: qué estamos haciendo ahora mismo y qué viene después. Se actualiza a medida que avanza el trabajo.',
			'web.project.editor.sectionPlaceholder' => 'Escribe esta sección en markdown…',
			'web.project.readonly.tech_stack.label' => 'Stack tecnológico y estructura',
			'web.project.readonly.tech_stack.empty' => 'Ejecuta una session de Claude en este proyecto. El escáner se actualiza en cada inicio.',
			'web.project.readonly.recent_activity.label' => 'Actividad reciente (git → LLM)',
			'web.project.readonly.recent_activity.empty' => 'El resumidor de actividad de git se ejecuta cada 24 h; vuelve a comprobarlo tras el siguiente ciclo del planificador.',
			'web.project.readonly.noneCaptured' => ({required Object label}) => 'Aún no se ha capturado ningún ${label}.',
			'web.project.readonly.generatedBy' => 'Generado por',
			'web.project.readonly.lastRefresh' => 'última actualización',
			'web.project.readonly.customEmpty' => 'Sección gestionada por el escáner; se rellena cuando éste corre.',
			'web.project.journal.loading' => 'Cargando…',
			'web.project.journal.empty' => 'Aún no hay entradas en el diario. Cada fin de session añade una automáticamente.',
			'web.project.inbox.loading' => 'Cargando…',
			'web.project.inbox.emptyTitle' => 'Bandeja de entrada vacía.',
			'web.project.inbox.emptyHint' => 'Los agentes presentan propuestas aquí mediante las herramientas MCP `project_goal_set` / `project_plan_set`.',
			'web.project.inbox.approvedToast' => ({required Object label}) => '${label} actualizado',
			'web.project.inbox.approveFailedToast' => 'Error al aprobar',
			'web.project.inbox.rejectedToast' => 'Rechazado',
			'web.project.inbox.rejectFailedToast' => 'Error al rechazar',
			'web.project.inbox.sessionPrefix' => 'ses',
			'web.project.inbox.warning' => ({required Object label}) => 'Aprobar REEMPLAZARÁ por completo el ${label} actual.',
			'web.project.inbox.warningSuffix' => 'Revisa el diff de abajo; esto no es una fusión.',
			'web.project.inbox.current' => 'Actual',
			'web.project.inbox.proposed' => 'Propuesto',
			'web.project.inbox.emptyBody' => '(vacío)',
			'web.project.inbox.approve' => 'Aprobar',
			'web.project.inbox.reject' => 'Rechazar',
			'web.project.inbox.confirmDialogTitle' => ({required Object label}) => '¿Reemplazar ${label}?',
			'web.project.inbox.confirmDialogDescription' => ({required Object label}) => 'El ${label} actual se sobrescribirá con el contenido propuesto. Esto no se puede deshacer desde esta interfaz (puedes volver a editarlo manualmente).',
			'web.project.inbox.confirmCancel' => 'Cancelar',
			'web.project.inbox.confirmReplace' => 'Confirmar reemplazo',
			'web.project.archived.hint' => 'Memorias que el limpiador automático archivó para este proyecto. Se excluyen de la recuperación pero son restaurables hasta que la ventana de gracia de 30 días las purgue.',
			'web.project.archived.empty' => 'Nada archivado para este proyecto. El limpiador archiva aquí los hechos obsoletos y duplicados automáticamente; todavía ninguno.',
			'web.project.archived.archivedAtPrefix' => 'Archivado',
			'web.project.archived.restoreButton' => 'Restaurar',
			'web.project.archived.restoredToast' => 'Restaurado',
			'web.project.archived.restoreFailedToast' => 'Error al restaurar',
			'web.project.reset.button' => 'Restablecer',
			'web.project.reset.dialogTitle' => '¿Restablecer la memoria del proyecto?',
			'web.project.reset.dialogDescription' => 'Elimina todo el contexto de proyecto almacenado para este cwd. Esto no se puede deshacer.',
			'web.project.reset.alwaysDeleted' => 'Siempre se elimina: objetivo, plan, propuestas, diario, decisiones de limpieza.',
			'web.project.reset.alsoDeleteScannerLabel' => 'Eliminar también los documentos del escáner',
			'web.project.reset.alsoDeleteScannerSuffix' => '(tech_stack + recent_activity).',
			_ => null,
		} ?? switch (path) {
			'web.project.reset.alsoDeleteScannerHint' => 'De todos modos se reconstruyen automáticamente en el siguiente inicio; dejarlo sin marcar suele estar bien.',
			'web.project.reset.alsoDeleteMemoriesLabel' => 'Eliminar también las memorias de pgvector',
			'web.project.reset.alsoDeleteMemoriesSuffix' => 'para este scope_key.',
			'web.project.reset.alsoDeleteMemoriesHint' => 'Hechos a largo plazo que el agente almacenó (preferencias del usuario, datos del proyecto). No se pueden recuperar.',
			'web.project.reset.cancel' => 'Cancelar',
			'web.project.reset.deleteForever' => 'Eliminar para siempre',
			'web.project.reset.successToast' => ({required Object summary}) => 'Restablecido: se eliminó ${summary}',
			'web.project.reset.summary.docs_one' => ({required Object count}) => '${count} documento',
			'web.project.reset.summary.docs_other' => ({required Object count}) => '${count} documentos',
			'web.project.reset.summary.journal' => ({required Object count}) => '${count} diario',
			'web.project.reset.summary.proposals_one' => ({required Object count}) => '${count} propuesta',
			'web.project.reset.summary.proposals_other' => ({required Object count}) => '${count} propuestas',
			'web.project.reset.summary.cleanup' => ({required Object count}) => '${count} limpieza',
			'web.project.reset.summary.memories' => ({required Object count}) => '${count} memorias',
			'web.project.reset.failedToast' => 'Error al restablecer',
			'web.project.lifecycle.status.active' => 'Activo',
			'web.project.lifecycle.status.paused' => 'Pausado',
			'web.project.lifecycle.status.archived' => 'Archivado',
			'web.project.lifecycle.activate' => 'Activar',
			'web.project.lifecycle.pause' => 'Pausar',
			'web.project.lifecycle.archive' => 'Archivar',
			'web.project.lifecycle.idleSuggest' => 'Inactivo — considera archivar',
			'web.project.lifecycle.idleHint' => ({required Object days}) => 'Sin actividad durante ${days} días',
			'web.project.lifecycle.failedToast' => 'No se pudo cambiar el estado del proyecto',
			'web.project.lifecycle.applied.active' => 'Proyecto reactivado',
			'web.project.lifecycle.applied.paused' => 'Proyecto pausado',
			'web.project.lifecycle.applied.archived' => 'Proyecto archivado',
			'web.project.lifecycle.tooltip.badge' => 'Ciclo de vida del proyecto. Los proyectos congelados (pausados/archivados) se excluyen de la inyección en nuevas sesiones y de la destilación por IA.',
			'web.project.lifecycle.tooltip.activate' => 'Reactivar: inyectar en nuevas sesiones y reanudar el mantenimiento por IA.',
			'web.project.lifecycle.tooltip.pause' => 'Pausar: congelar este proyecto — omitir inyección y destilación, pero mantenerlo en la lista activa.',
			'web.project.lifecycle.tooltip.archive' => 'Archivar: archivar este proyecto — congelado y oculto de las vistas habituales.',
			'web.project.docMeta.maintainer.coauthored' => 'Tú mantienes · IA propone',
			'web.project.docMeta.maintainer.auto' => 'Autogenerado · solo lectura',
			'web.project.docMeta.maintainer.human' => 'Autoría humana',
			'web.project.docMeta.purpose.goal' => 'La intención a largo plazo del proyecto: qué construimos y por qué. Cuando una sesión cambia el rumbo, la IA propone una actualización en tu Bandeja para que la apruebes.',
			'web.project.docMeta.purpose.plan' => 'La hoja de ruta actual / trabajo en curso. La IA propone una actualización en tu Bandeja tras avanzar una sesión; tú la apruebas.',
			'web.project.docMeta.purpose.tech_stack' => 'Stack y estructura, autogenerado por el escáner del proyecto (se actualiza cada 6 h).',
			'web.project.docMeta.purpose.recent_activity' => 'Resumen por IA de la actividad reciente de Git, actualizado automáticamente (cada 12 h).',
			'web.project.docMeta.purpose.overview' => 'El documento oficial del proyecto: qué es, sus funciones, arquitectura, cómo construir/ejecutar y las bases en que se apoya. Redactado por IA desde las señales del propio proyecto; puedes editarlo (lo bloquea) o regenerarlo.',
			'web.project.proposalBanner.text' => 'La IA ha propuesto una actualización de este documento, a la espera de tu aprobación.',
			'web.project.proposalBanner.button' => 'Revisar en la Bandeja',
			'web.project.overview.aiManaged' => 'Mantenido por IA (se actualiza desde el proyecto)',
			'web.project.overview.locked' => 'Bloqueado — lo editaste; las actualizaciones de IA llegan como propuestas',
			'web.project.overview.edit' => 'Editar',
			'web.project.overview.save' => 'Guardar (bloquea)',
			'web.project.overview.cancel' => 'Cancelar',
			'web.project.overview.unlock' => 'Desbloquear (devolver a la IA)',
			'web.project.overview.regenerate' => 'Regenerar',
			'web.project.overview.generate' => 'Generar ahora',
			'web.project.overview.regenerateHint' => 'Pide a la IA que redacte el resumen con el estado más reciente',
			'web.project.overview.editHint' => 'Guardar bloquea la página; desbloquear deja que la IA la redacte.',
			'web.project.overview.empty' => 'Aún no hay resumen. El motor en segundo plano lo redacta desde el objetivo/plan, el escaneo de stack, el registro y la memoria — o genéralo ahora.',
			'web.project.overview.saved' => 'Resumen guardado',
			'web.project.overview.unlocked' => 'Desbloqueado — la IA volverá a gestionarlo',
			'web.project.overview.regenerating' => 'Regenerando el resumen…',
			'web.memoryInspector.status.label' => 'Embedder activo',
			'web.memoryInspector.status.unavailable' => 'no disponible',
			'web.memoryInspector.status.probing' => 'sondeando…',
			'web.memoryInspector.status.dimensions' => ({required Object dim, required Object state}) => '${dim}-dim · ${state}',
			'web.memoryInspector.status.enabled' => 'habilitado',
			'web.memoryInspector.status.disabled' => 'deshabilitado',
			'web.memoryInspector.status.floorNoModel' => 'Solo recuperación por palabras clave (BM25) — no hay modelo de embedding configurado. Añade un endpoint denso [memory.http] en Settings para habilitar la memoria semántica.',
			'web.memoryInspector.status.denseConfiguredPendingRestart' => ({required Object model}) => 'Configurado ${model} (denso) — reinicia el gateway para activar la memoria semántica y re-embeber las memorias existentes.',
			'web.memoryInspector.status.denseUnreachableFloor' => ({required Object model}) => 'Configurado ${model} (denso) pero el endpoint está inalcanzable — se usa el piso de palabras clave hasta que responda (se actualiza al reiniciar).',
			'web.memoryInspector.status.denseDegraded' => 'Embedder denso activo pero su endpoint está inalcanzable ahora — los vectores existentes se conservan; las nuevas escrituras y la búsqueda por similitud se pausan hasta que responda.',
			'web.memoryInspector.scope.label' => 'Scope',
			'web.memoryInspector.scope.scopeKey' => 'Clave de scope',
			'web.memoryInspector.scope.scopeKeyIgnored' => '(ignorado para global)',
			'web.memoryInspector.scope.scopeKeyCwd' => '(cwd del proyecto)',
			'web.memoryInspector.scope.placeholderProject' => '/path/to/project (cwd)',
			'web.memoryInspector.scope.syncMd' => 'Sincronizar .md',
			'web.memoryInspector.scope.syncTooltip' => 'Reimportar los archivos <cwd>/.claude/memory/*.md de Claude a pgvector',
			'web.memoryInspector.scope.browse' => 'Explorar',
			'web.memoryInspector.scope.browseTooltip' => 'Explora el sistema de archivos del host del gateway para elegir cualquier directorio de proyecto',
			'web.memoryInspector.scope.values.project' => 'proyecto',
			'web.memoryInspector.scope.values.global' => 'global',
			'web.memoryInspector.search.placeholder' => 'Consulta de búsqueda semántica (Enter para ejecutar; vacío = explorar)',
			'web.memoryInspector.search.run' => 'Buscar',
			'web.memoryInspector.search.clear' => 'Limpiar',
			'web.memoryInspector.search.failedToast' => 'La búsqueda falló',
			'web.memoryInspector.records.noMemories' => 'Aún no hay memorias',
			'web.memoryInspector.records.matches_one' => ({required Object count}) => '${count} coincidencia',
			'web.memoryInspector.records.matches_other' => ({required Object count}) => '${count} coincidencias',
			'web.memoryInspector.records.memories_one' => ({required Object count}) => '${count} memoria',
			'web.memoryInspector.records.memories_other' => ({required Object count}) => '${count} memorias',
			'web.memoryInspector.records.scopeGlobalSuffix' => ' (global)',
			'web.memoryInspector.records.scopeInSuffix' => ({required Object scope}) => ' en ${scope}: ',
			'web.memoryInspector.records.addButton' => 'Añadir memoria',
			'web.memoryInspector.records.addTooltip' => 'Crear manualmente una memoria en este scope',
			'web.memoryInspector.records.deleteAll' => 'Eliminar todo',
			'web.memoryInspector.records.deleteAllTooltip' => 'Eliminar todas las memorias de este scope/scope_key',
			'web.memoryInspector.records.loading' => 'Cargando…',
			'web.memoryInspector.records.enterScopeKeyHint' => 'Introduce una clave de scope para explorar las memorias.',
			'web.memoryInspector.records.noMatchesForQuery' => ({required Object query}) => 'No hay coincidencias para "${query}"',
			'web.memoryInspector.records.noMemoriesInScope' => 'Aún no hay memorias en este scope.',
			'web.memoryInspector.row.simBadge' => ({required Object value}) => 'sim ${value}',
			'web.memoryInspector.row.rankBadge' => ({required Object value}) => 'rango ${value}',
			'web.memoryInspector.row.rankTooltip' => ({required Object effective, required Object similarity, required Object age, required Object days, required Object hits, required Object confidence}) => 'efectivo ${effective} = sim ${similarity} × antigüedad ${age} (${days}d) × hits ${hits} × conf ${confidence}',
			'web.memoryInspector.row.hits_one' => ({required Object count}) => '${count} hit',
			'web.memoryInspector.row.hits_other' => ({required Object count}) => '${count} hits',
			'web.memoryInspector.row.lastHitTooltip' => ({required Object relative}) => 'Último hit ${relative}',
			'web.memoryInspector.row.editPlaceholder' => 'Texto de la memoria. Cmd/Ctrl+Enter para guardar · Esc para cancelar',
			'web.memoryInspector.row.saveTooltip' => 'Guardar (Cmd/Ctrl+Enter)',
			'web.memoryInspector.row.cancelTooltip' => 'Cancelar (Esc)',
			'web.memoryInspector.row.editTooltip' => 'Editar esta memoria',
			'web.memoryInspector.row.deleteTooltip' => 'Eliminar esta memoria',
			'web.memoryInspector.row.emptyError' => 'El texto de la memoria no puede estar vacío',
			'web.memoryInspector.row.deleteConfirm' => ({required Object id}) => '¿Eliminar la memoria ${id}? Esto es permanente.',
			'web.memoryInspector.row.archiveTooltip' => 'Archivar (reversible) — va a la vista Archivado',
			'web.memoryInspector.row.quarantineTooltip' => 'Cuarentena — va a la cola de revisión hasta promoverla o que expire',
			'web.memoryInspector.toasts.deleted' => 'Memoria eliminada',
			'web.memoryInspector.toasts.deleteFailed' => 'La eliminación falló',
			'web.memoryInspector.toasts.bulkDeleted_one' => ({required Object count}) => 'Se eliminó ${count} memoria de este scope',
			'web.memoryInspector.toasts.bulkDeleted_other' => ({required Object count}) => 'Se eliminaron ${count} memorias de este scope',
			'web.memoryInspector.toasts.bulkDeleteFailed' => 'La eliminación masiva falló',
			'web.memoryInspector.toasts.created' => 'Memoria creada',
			'web.memoryInspector.toasts.createFailed' => 'La creación falló',
			'web.memoryInspector.toasts.updated' => 'Memoria actualizada',
			'web.memoryInspector.toasts.updateFailed' => 'La actualización falló',
			'web.memoryInspector.toasts.migrated' => ({required Object reembed, required Object examined, required Object to}) => 'Se migraron ${reembed}/${examined} memorias a ${to}',
			'web.memoryInspector.toasts.migrationFailed' => 'La migración falló',
			'web.memoryInspector.toasts.syncIngested_one' => ({required Object count}) => 'Se importó ${count} nuevo archivo de memoria',
			'web.memoryInspector.toasts.syncIngested_other' => ({required Object count}) => 'Se importaron ${count} nuevos archivos de memoria',
			'web.memoryInspector.toasts.syncEmpty' => 'No hay nuevos archivos .md que sincronizar',
			'web.memoryInspector.toasts.syncEmptyDescription' => 'Ya está sincronizado, o no hay directorio de memoria de Claude para este cwd.',
			'web.memoryInspector.toasts.syncFailed' => 'La sincronización falló',
			'web.memoryInspector.toasts.archived' => 'Memoria archivada — restaurable desde la vista Archivado',
			'web.memoryInspector.toasts.archiveFailed' => 'Fallo al archivar',
			'web.memoryInspector.toasts.quarantined' => 'Memoria en cuarentena — revísala en Cortex → Cuarentena',
			'web.memoryInspector.toasts.quarantineFailed' => 'Fallo al poner en cuarentena',
			'web.memoryInspector.bulkDelete.title' => '¿Eliminar todas las memorias de este scope?',
			'web.memoryInspector.bulkDelete.description' => 'Esto es una única operación SQL: todas las memorias del scope especificado se eliminan de forma atómica. Las memorias que se importaron a través del mirror de Claude reaparecen en la siguiente ejecución de <1>Sincronizar .md</1>; todo lo demás se pierde para siempre.',
			'web.memoryInspector.bulkDelete.scope' => 'Scope',
			'web.memoryInspector.bulkDelete.scopeKey' => 'Clave de scope',
			'web.memoryInspector.bulkDelete.currentlyVisible' => 'Visibles actualmente',
			'web.memoryInspector.bulkDelete.items_one' => ({required Object count}) => '${count} elemento de memoria',
			'web.memoryInspector.bulkDelete.items_other' => ({required Object count}) => '${count} elementos de memoria',
			'web.memoryInspector.bulkDelete.cancel' => 'Cancelar',
			'web.memoryInspector.bulkDelete.deleteAll' => 'Eliminar todo',
			'web.memoryInspector.addMem.title' => 'Añadir memoria',
			'web.memoryInspector.addMem.description' => 'Crea manualmente una memoria. Los agentes las crean automáticamente mediante la herramienta MCP <1>memory_store</1>. Este formulario es para los casos en que el operador quiere insertar un hecho sin pasar por un agente.',
			'web.memoryInspector.addMem.textLabel' => 'Texto',
			'web.memoryInspector.addMem.textPlaceholder' => 'Prosa simple. El embedder lo convierte en un vector en el momento de almacenarlo; los agentes lo recuperarán mediante memory_search.',
			'web.memoryInspector.addMem.cancel' => 'Cancelar',
			'web.memoryInspector.addMem.create' => 'Crear',
			'web.memoryInspector.picker.button' => 'Elegir',
			'web.memoryInspector.picker.buttonTooltip' => 'Elige entre las claves de scope guardadas o las sessions activas',
			'web.memoryInspector.picker.loading' => 'Cargando…',
			'web.memoryInspector.picker.empty' => ({required Object scope}) => 'No hay claves guardadas ni sessions activas para ${scope}.',
			'web.memoryInspector.picker.savedHeader' => 'Memorias guardadas',
			'web.memoryInspector.picker.activeHeader' => 'Sessions activas',
			'web.memoryInspector.migrationBanner.headline_one' => ({required Object count}) => '${count} memoria no aparecerá en las búsquedas',
			'web.memoryInspector.migrationBanner.headline_other' => ({required Object count}) => '${count} memorias no aparecerán en las búsquedas',
			'web.memoryInspector.migrationBanner.subtitle' => ({required Object summary, required Object current}) => '${summary}. El embedder actual es <1>${current}</1>. pgvector particiona su índice de similitud por embedder, así que las entradas más antiguas permanecen silenciosas hasta que se vuelven a embeber.',
			'web.memoryInspector.migrationBanner.summaryItem' => ({required Object count, required Object name}) => '${count} en ${name}',
			'web.memoryInspector.migrationBanner.migrateButton' => 'Migrar',
			'web.memoryInspector.reembed.title' => 'Volver a embeber memorias',
			'web.memoryInspector.reembed.description' => 'Recalcula los vectores de las memorias almacenadas con un embedder diferente para que vuelvan a ser buscables.',
			'web.memoryInspector.reembed.targetEmbedder' => 'Embedder de destino',
			'web.memoryInspector.reembed.onName' => 'en',
			'web.memoryInspector.reembed.totalToReembed' => 'Total a volver a embeber',
			'web.memoryInspector.reembed.explainer' => 'El texto de cada memoria se vuelve a enviar al embedder actual; el nuevo vector reemplaza al antiguo en su lugar. Se conservan el ID, el scope, el scope_key, los metadatos y las marcas de tiempo. Los resultados de búsqueda surten efecto de inmediato, sin necesidad de reiniciar.',
			'web.memoryInspector.reembed.reportExamined' => 'Examinadas',
			'web.memoryInspector.reembed.reportReembedded' => 'Vueltas a embeber',
			'web.memoryInspector.reembed.reportFailed' => 'Fallidas',
			'web.memoryInspector.reembed.reportFrom' => 'Desde',
			'web.memoryInspector.reembed.errors_one' => ({required Object count}) => '${count} error',
			'web.memoryInspector.reembed.errors_other' => ({required Object count}) => '${count} errores',
			'web.memoryInspector.reembed.done' => 'Hecho',
			'web.memoryInspector.reembed.cancel' => 'Cancelar',
			'web.memoryInspector.reembed.reembedding' => 'Volviendo a embeber…',
			'web.memoryInspector.reembed.reembedTotal' => ({required Object total}) => 'Volver a embeber ${total}',
			'web.notes.title' => 'Notas',
			'web.notes.header.outline' => 'Esquema',
			'web.notes.header.showOutline' => 'Mostrar esquema',
			'web.notes.header.hideOutline' => 'Ocultar esquema',
			'web.notes.header.today' => 'Hoy',
			'web.notes.header.todayTooltip' => 'Abre o crea la nota diaria de hoy',
			'web.notes.header.kNew' => 'Nueva',
			'web.notes.left.tree' => 'Árbol',
			'web.notes.left.tags' => 'Etiquetas',
			'web.notes.left.filterNotes' => 'Filtrar notas…',
			'web.notes.left.filterTags' => 'Filtrar etiquetas…',
			'web.notes.left.filteredBy' => 'filtrado por',
			'web.notes.left.clearTagTooltip' => 'Limpiar filtro de etiquetas',
			'web.notes.left.expandAll' => 'Expandir todo',
			'web.notes.left.expandAllTooltip' => 'Expandir todas las carpetas',
			'web.notes.left.collapseAll' => 'Contraer todo',
			'web.notes.left.collapseAllTooltip' => 'Contraer todas las carpetas',
			'web.notes.left.loading' => 'Cargando…',
			'web.notes.left.footer' => ({required Object visible, required Object total}) => '${visible} / ${total} notas',
			'web.notes.tags.emptyVault' => 'Aún no hay etiquetas en el vault.',
			'web.notes.tags.noMatches' => ({required Object query}) => 'No hay coincidencias para "${query}".',
			'web.notes.tree.empty' => 'El vault está vacío.',
			'web.notes.outline.label' => 'Esquema',
			'web.notes.outline.empty' => 'No hay encabezados en esta nota. Añade líneas <1>## Título</1> para rellenar el esquema.',
			'web.notes.newNote.prompt' => 'Ruta de la nueva nota (relativa al vault, debe terminar en .md)',
			'web.notes.newNote.defaultPath' => ({required Object date}) => 'library/notes-${date}.md',
			'web.notes.newNote.errorMustEndMd' => 'La ruta debe terminar en .md',
			'web.notes.newNote.createdToast' => 'Nota creada',
			'web.notes.newNote.createFailedToast' => 'Error al crear',
			'web.notes.empty.title' => 'Ninguna nota seleccionada',
			'web.notes.empty.hint' => 'Elige una nota del árbol de la izquierda, ve directo al registro diario de hoy o crea una nueva. Los docs de proyecto escritos por la IA viven en <1>projects/</1>; tus borradores personales en <3>personal/</3>.',
			'web.notes.empty.today' => 'Nota diaria de hoy',
			'web.notes.empty.kNew' => 'Nueva nota',
			'web.notes.picker.browseAria' => 'Explorar carpetas',
			'web.notes.picker.matches_one' => ({required Object count}) => '${count} coincidencia',
			'web.notes.picker.matches_other' => ({required Object count}) => '${count} coincidencias',
			'web.notes.picker.foldersInVault' => ({required Object count}) => '${count} carpetas en el vault',
			'web.notes.picker.noMatch' => ({required Object value}) => 'Ninguna carpeta existente coincide. Guarda igualmente para usar <1>${value}</1> (se crea de forma diferida en la primera escritura).',
			'web.notes.vaultSync.title' => 'Sincronización del vault',
			'web.notes.vaultSync.description' => 'Haz commit, pull y push del vault de notas como un repositorio git. La autenticación usa las credenciales de git del host de tu gateway (agente SSH / asistente de credenciales).',
			'web.notes.vaultSync.reading' => 'Leyendo el estado del vault…',
			'web.notes.vaultSync.init.title' => 'El vault aún no es un repo git',
			'web.notes.vaultSync.init.body' => 'Al inicializarlo se ejecutará <1>git init -b main</1> en la raíz de tu vault y se añadirá un <3>.gitignore</3> sensato. Después podrás hacer commit de tus notas y configurar un remoto (GitHub / Gitea / GitLab) para la sincronización entre máquinas.',
			'web.notes.vaultSync.init.button' => 'Inicializar el vault como repo git',
			'web.notes.vaultSync.init.initToast' => 'Vault inicializado como repo git',
			'web.notes.vaultSync.init.initFailedToast' => 'Error al inicializar',
			'web.notes.vaultSync.branch.clean' => 'limpio',
			'web.notes.vaultSync.branch.staged' => ({required Object count}) => '${count} en stage',
			'web.notes.vaultSync.branch.modified' => ({required Object count}) => '${count} modificados',
			'web.notes.vaultSync.branch.untracked' => ({required Object count}) => '${count} sin seguimiento',
			'web.notes.vaultSync.action.pull' => 'Pull',
			'web.notes.vaultSync.action.push' => 'Push',
			'web.notes.vaultSync.action.pullTitleNoRemote' => 'Configura primero un remoto',
			'web.notes.vaultSync.action.pullTitleHasUpstream' => 'git pull --rebase --autostash',
			'web.notes.vaultSync.action.pullTitleNoUpstream' => 'Hace pull del HEAD de origin; configura el seguimiento de forma implícita',
			'web.notes.vaultSync.action.pushTitleNoRemote' => 'Configura primero un remoto',
			'web.notes.vaultSync.action.pushTitleHasUpstream' => 'git push -u origin HEAD',
			'web.notes.vaultSync.action.pushTitleNoUpstream' => 'Primer push: configurará el upstream a origin/HEAD',
			'web.notes.vaultSync.action.noRemote' => 'Sin remoto configurado: pull/push deshabilitados',
			'web.notes.vaultSync.action.noUpstream' => 'Aún no hay seguimiento de upstream: el primer push lo configurará.',
			'web.notes.vaultSync.action.pulledToast' => 'Pull realizado',
			'web.notes.vaultSync.action.pullFailedToast' => 'Error en el pull',
			'web.notes.vaultSync.action.pushedToast' => 'Push realizado',
			'web.notes.vaultSync.action.pushFailedToast' => 'Error en el push',
			'web.notes.vaultSync.commit.title' => 'Commit',
			'web.notes.vaultSync.commit.placeholder' => ({required Object date}) => 'Notas: ${date} (predeterminado)',
			'web.notes.vaultSync.commit.commitAll' => 'Hacer commit de todo',
			'web.notes.vaultSync.commit.hint' => 'Pone en stage todos los cambios (<1>git add .</1>) y luego hace commit con este mensaje. Un mensaje vacío usa de forma predeterminada un asunto con marca de tiempo.',
			'web.notes.vaultSync.commit.committedToast' => ({required Object hash}) => 'Commit ${hash} realizado',
			'web.notes.vaultSync.commit.commitFailedToast' => 'Error en el commit',
			'web.notes.vaultSync.fileList.title' => ({required Object count}) => 'Árbol de trabajo · ${count}',
			'web.notes.vaultSync.fileList.moreSuffix' => ({required Object count}) => '+${count} más',
			'web.notes.vaultSync.remote.title' => 'Remoto (origin)',
			'web.notes.vaultSync.remote.cancel' => 'Cancelar',
			'web.notes.vaultSync.remote.change' => 'Cambiar',
			'web.notes.vaultSync.remote.configure' => 'Configurar',
			'web.notes.vaultSync.remote.empty' => 'No hay remoto configurado. Añade una URL HTTPS o SSH (p. ej. <1>git@github.com:you/notes.git</1> o <3>https://gitea.example.com/you/notes.git</3>) para habilitar push / pull.',
			'web.notes.vaultSync.remote.urlLabel' => 'URL (HTTPS o SSH)',
			'web.notes.vaultSync.remote.urlPlaceholder' => 'git@host:owner/notes.git',
			'web.notes.vaultSync.remote.save' => 'Guardar',
			'web.notes.vaultSync.remote.savedToast' => 'Remoto guardado',
			'web.notes.vaultSync.remote.saveFailedToast' => 'Error al configurar el remoto',
			'web.notes.vaultSync.history.title' => 'Commits recientes',
			'web.notes.vaultSync.history.loading' => 'Cargando…',
			'web.notes.vaultSync.history.empty' => 'Aún no hay commits.',
			'web.notes.vaultSync.conflict.kinds.rebase' => 'rebase',
			'web.notes.vaultSync.conflict.kinds.merge' => 'merge',
			'web.notes.vaultSync.conflict.kinds.cherryPick' => 'cherry-pick',
			'web.notes.vaultSync.conflict.kinds.operation' => 'operación',
			'web.notes.vaultSync.conflict.headline' => ({required Object kind}) => 'El vault tiene un ${kind} en pausa con conflictos sin resolver',
			'web.notes.vaultSync.conflict.explainer' => ({required Object kind}) => 'Pull, push y commit están bloqueados hasta que termine el ${kind}. Puedes hacer <1>abort</1> (restaurar el árbol de trabajo a su estado anterior al ${kind}, conserva tus commits locales y descarta los remotos) o <3>forzar reset al remoto</3> (descartar TODOS los commits locales y los cambios sin confirmar; el vault se convierte en una copia exacta de origin).',
			'web.notes.vaultSync.conflict.conflictedHeader' => ({required Object count}) => 'Archivos en conflicto · ${count}',
			'web.notes.vaultSync.conflict.abort' => ({required Object kind}) => 'Abortar ${kind}',
			'web.notes.vaultSync.conflict.abortTitle' => ({required Object kind}) => 'git ${kind} --abort',
			'web.notes.vaultSync.conflict.forceReset' => 'Forzar reset al remoto',
			'web.notes.vaultSync.conflict.forceResetTitle' => 'git fetch && git reset --hard origin/<branch> && git clean -fd',
			'web.notes.vaultSync.conflict.forceResetConfirm' => ({required Object kind}) => 'DESTRUCTIVO: esto va a\n  • abortar el ${kind} en curso\n  • ejecutar git fetch origin\n  • hacer reset --hard a origin/<branch>\n  • ejecutar clean -fd (descartar archivos sin seguimiento)\n\nCualquier commit local no enviado a origin Y cualquier edición sin confirmar se PERDERÁ DE FORMA PERMANENTE.\n\n¿Continuar?',
			'web.notes.vaultSync.conflict.abortedToast' => ({required Object kind}) => '${kind} abortado',
			'web.notes.vaultSync.conflict.abortedDescription' => 'Árbol de trabajo restaurado al estado anterior a la operación.',
			'web.notes.vaultSync.conflict.abortFailedToast' => 'Error al abortar',
			'web.notes.vaultSync.conflict.resetToast' => ({required Object branch}) => 'Reset a ${branch}',
			'web.notes.vaultSync.conflict.resetDescription' => 'Cambios locales descartados; el vault coincide con el remoto.',
			'web.notes.vaultSync.conflict.resetFailedToast' => 'Error en el reset',
			'web.notes.vaultSync.auth.title' => 'Autenticación',
			'web.notes.vaultSync.auth.httpsTokenOk' => ({required Object host}) => 'Usará el token guardado para <1>${host}</1> en Plugins → Hosts de git. ✓',
			'web.notes.vaultSync.auth.httpsTokenMissing' => ({required Object host}) => 'Remoto HTTPS en <1>${host}</1> sin ningún token de opendray configurado. Es probable que push / pull fallen en repos privados hasta que añadas uno.',
			'web.notes.vaultSync.auth.ssh' => ({required Object host}) => 'Remoto SSH en <1>${host}</1>. La autenticación usa el <3>~/.ssh/</3> del host del gateway (ssh-agent, archivo de identidad, configuración de host). Verifícalo con <5>ssh -T git@${host}</5> desde la shell del host.',
			'web.notes.vaultSync.auth.configureTokenLink' => '→ Configurar token de host de git',
			'web.notes.vaultSync.autoSync.loading' => 'Cargando ajustes de sincronización automática…',
			'web.notes.vaultSync.autoSync.title' => 'Sincronización automática',
			'web.notes.vaultSync.autoSync.on' => 'activada',
			'web.notes.vaultSync.autoSync.runNow' => 'Ejecutar ahora',
			'web.notes.vaultSync.autoSync.runNowTooltip' => 'Despierta el bucle de sincronización ahora (omite la espera y luego ejecuta los pasos pendientes)',
			'web.notes.vaultSync.autoSync.configure' => 'Configurar',
			'web.notes.vaultSync.autoSync.hide' => 'Ocultar',
			'web.notes.vaultSync.autoSync.enabled' => 'Habilitada',
			'web.notes.vaultSync.autoSync.enabledTooltipNoRemote' => 'Configura primero un remoto para habilitar la sincronización automática',
			'web.notes.vaultSync.autoSync.noRemoteHint' => 'Sin remoto: se omitirán push/pull.',
			'web.notes.vaultSync.autoSync.commitEvery' => 'Hacer commit cada',
			'web.notes.vaultSync.autoSync.commitEveryExamples' => 'Ejemplos: <1>30s</1>, <3>10m</3>, <5>2h</5>. Mínimo 30s.',
			'web.notes.vaultSync.autoSync.pullEvery' => 'Hacer pull cada',
			'web.notes.vaultSync.autoSync.pullEveryHint' => 'Solo se usa cuando Pull está habilitado.',
			'web.notes.vaultSync.autoSync.pushAfterCommit' => 'Hacer push tras el commit',
			'web.notes.vaultSync.autoSync.pullPeriodically' => 'Hacer pull periódicamente',
			'web.notes.vaultSync.autoSync.commitTemplateLabel' => 'Plantilla del mensaje de commit',
			'web.notes.vaultSync.autoSync.commitTemplatePlaceholder' => ({required Object date}) => 'Sincronización automática: ${date}  (predeterminado si está vacío)',
			'web.notes.vaultSync.autoSync.saveSettings' => 'Guardar ajustes',
			'web.notes.vaultSync.autoSync.discard' => 'Descartar',
			'web.notes.vaultSync.autoSync.lastCommit' => 'último commit',
			'web.notes.vaultSync.autoSync.lastPush' => 'último push',
			'web.notes.vaultSync.autoSync.lastPull' => 'último pull',
			'web.notes.vaultSync.autoSync.never' => 'nunca',
			'web.notes.vaultSync.autoSync.savedToast' => 'Ajustes de sincronización automática guardados',
			'web.notes.vaultSync.autoSync.saveFailedToast' => 'Error al guardar',
			'web.notes.vaultSync.autoSync.triggeredToast' => 'Sincronización automática iniciada',
			'web.notes.vaultSync.autoSync.runFailedToast' => 'Error en la ejecución',
			'web.notes.syncBadge.loading' => 'Cargando…',
			'web.notes.syncBadge.syncLabel' => 'Sincronizar',
			'web.notes.syncBadge.initLabel' => 'Init',
			'web.notes.syncBadge.initTooltip' => 'El vault aún no es un repo git',
			'web.notes.syncBadge.conflictLabel' => 'Conflicto',
			'web.notes.syncBadge.conflictTooltip' => 'El vault tiene conflictos sin resolver: haz clic para recuperar',
			'web.notes.syncBadge.syncFallback' => 'sincronizar',
			'web.notes.syncBadge.tooltip' => ({required Object branch, required Object files, required Object ahead, required Object behind}) => 'rama ${branch} · ${files} cambios · ${ahead} por delante · ${behind} por detrás',
			'web.notes.syncBadge.tooltipAutoOn' => ' · sincronización automática activada',
			'web.notes.syncBadge.tooltipLastError' => ({required Object error}) => ' · último error: ${error}',
			'web.notes.syncBadge.branchPlaceholder' => '—',
			'web.activity.title' => 'Actividad',
			'web.activity.subtitle' => 'Auditoría por llamada de las solicitudes API realizadas por las integraciones registradas. Incluye tanto las llamadas entrantes (una app de terceros que llama a opendray con su clave de API) como las llamadas salientes a través del proxy (admin → proxy de opendray → integración). Las llamadas hechas directamente por esta UI de administración no se registran.',
			'web.activity.refresh' => 'Actualizar',
			'web.activity.refreshTooltip' => 'Actualizar',
			'web.activity.filters.integration' => 'Integración',
			'web.activity.filters.direction' => 'Dirección',
			'web.activity.filters.status' => 'Estado',
			'web.activity.filters.allIntegrations' => 'Todas las integraciones',
			'web.activity.filters.all' => 'Todas',
			'web.activity.filters.inbound' => 'Entrante',
			'web.activity.filters.outbound' => 'Saliente',
			'web.activity.filters.allStatuses' => 'Todos los estados',
			'web.activity.filters.status2' => '2xx correcto',
			'web.activity.filters.status3' => '3xx redirección',
			'web.activity.filters.status4' => '4xx error de cliente',
			'web.activity.filters.status5' => '5xx error de servidor',
			'web.activity.callsCount_one' => ({required Object count}) => '${count} llamada',
			'web.activity.callsCount_other' => ({required Object count}) => '${count} llamadas',
			'web.activity.loading' => 'Cargando…',
			'web.activity.table.time' => 'Hora',
			'web.activity.table.integration' => 'Integración',
			'web.activity.table.directionTitle' => 'Dirección',
			'web.activity.table.method' => 'Método',
			'web.activity.table.path' => 'Ruta',
			'web.activity.table.status' => 'Estado',
			'web.activity.table.duration' => 'Duración',
			'web.activity.table.inboundAria' => 'entrante',
			'web.activity.table.outboundAria' => 'saliente',
			'web.activity.empty.filtered' => 'Ninguna llamada coincide con estos filtros.',
			'web.activity.empty.title' => 'Aún no se ha registrado ninguna llamada API',
			'web.activity.empty.description' => 'Cuando una app de terceros llama a opendray con su clave de API de integración, cada solicitud se registra aquí.',
			'web.activity.empty.stepWithIntegrations' => 'Usa la clave de API de una integración existente en tu app de terceros',
			'web.activity.empty.stepRegister' => 'Registra una integración en Integraciones → Nueva',
			'web.activity.empty.stepCallEndpoint' => 'Llama a cualquier endpoint, p. ej. <1>POST /api/v1/sessions</1>',
			'web.activity.empty.stepAppears' => 'Las llamadas aparecen aquí en cuestión de segundos',
			'web.activity.empty.footnote' => 'Las llamadas que haces desde esta UI de administración no se registran; solo se registra el tráfico atribuido a integraciones.',
			'web.activity.events.loading' => 'Cargando eventos…',
			'web.activity.events.empty' => 'Aún no hay eventos.',
			'web.activity.events.emptyFiltered' => 'No hay eventos coincidentes.',
			'web.activity.events.loadOlder' => 'Cargar eventos anteriores',
			'web.activity.events.today' => 'Hoy',
			'web.activity.events.yesterday' => 'Ayer',
			'web.providers.list.title' => 'Proveedores',
			'web.providers.list.loading' => 'Cargando…',
			'web.providers.list.disabledBadge' => 'deshabilitado',
			'web.providers.list.noneSelected' => 'Ningún proveedor seleccionado.',
			'web.providers.detail.enabled' => 'Habilitado',
			'web.providers.detail.disabled' => 'Deshabilitado',
			'web.providers.detail.toggleAria' => ({required Object name}) => 'Alternar ${name}',
			'web.providers.detail.configuration' => 'Configuración',
			'web.providers.detail.noConfig' => 'Este proveedor no tiene campos configurables por el usuario.',
			'web.providers.detail.executable' => 'executable:',
			'web.providers.detail.manifestHash' => 'manifest_hash:',
			'web.providers.detail.reset' => 'Restablecer',
			'web.providers.detail.save' => 'Guardar cambios',
			'web.providers.detail.saving' => 'Guardando…',
			'web.providers.detail.savedToast' => 'Configuración del proveedor guardada',
			'web.providers.detail.saveFailedToast' => 'Error al guardar',
			'web.providers.detail.toggleFailedToast' => 'Error al alternar',
			'web.providers.detail.caps.resume' => 'resume',
			'web.providers.detail.caps.stream' => 'stream',
			'web.providers.detail.caps.images' => 'images',
			'web.providers.detail.caps.mcp' => 'mcp',
			'web.providers.detail.notInstalled' => 'no instalado',
			'web.providers.detail.updateAvailable' => ({required Object version}) => 'actualización disponible → ${version}',
			'web.providers.detail.upToDate' => 'actualizado',
			'web.providers.detail.update' => ({required Object version}) => 'Actualizar a ${version}',
			'web.providers.detail.updating' => 'Actualizando…',
			'web.providers.detail.updatedToast' => ({required Object from, required Object to}) => 'Actualizado ${from} → ${to}',
			'web.providers.detail.alreadyLatestToast' => 'Ya está actualizado',
			'web.providers.detail.updateFailedToast' => 'Error al actualizar',
			'web.providers.detail.updateUnavailable' => 'La actualización dentro de la app no está disponible aquí',
			'web.providers.configForm.selectPlaceholder' => 'Selecciona…',
			'web.providers.configForm.defaultOption' => '(predeterminado)',
			'web.providers.configForm.switchOn' => 'Activado',
			'web.providers.configForm.switchOff' => 'Desactivado',
			'web.providers.configForm.showSecret' => 'Mostrar secreto',
			'web.providers.configForm.hideSecret' => 'Ocultar secreto',
			'web.providers.claudeAccounts.title' => 'Cuentas de Claude',
			'web.providers.claudeAccounts.importLocal' => 'Importar local',
			'web.providers.claudeAccounts.importLocalTooltip' => 'Escanea ~/.claude-accounts/ en el host del gateway y registra cualquier directorio nuevo. El botón solo funciona en el host del gateway.',
			'web.providers.claudeAccounts.importedNothingToast' => 'Nada que importar, las cuentas ya están sincronizadas.',
			'web.providers.claudeAccounts.importedToast_one' => ({required Object count}) => 'Se importó ${count} cuenta desde ~/.claude-accounts',
			'web.providers.claudeAccounts.importedToast_other' => ({required Object count}) => 'Se importaron ${count} cuentas desde ~/.claude-accounts',
			'web.providers.claudeAccounts.importFailedToast' => 'Error al importar',
			'web.providers.claudeAccounts.addingTitle' => 'Añadiendo una cuenta nueva.',
			'web.providers.claudeAccounts.addingBodyPrefix' => 'Ejecuta en el host del gateway:',
			'web.providers.claudeAccounts.addingBodySuffix' => 'el monitor del sistema de archivos de opendray registrará el directorio nuevo automáticamente, o haz clic en <1>Importar local</1> para escanear de inmediato.',
			'web.providers.claudeAccounts.architectureLink' => 'Arquitectura y guía completa →',
			'web.providers.claudeAccounts.loading' => 'Cargando…',
			'web.providers.claudeAccounts.empty' => 'Aún no hay cuentas de Claude. La forma más sencilla: abre Sessions, inicia una session de Claude y ejecuta <1>claude login</1> en la terminal. Tus credenciales de OAuth se guardan en <3>~/.claude</3> en el gateway y aparecen aquí automáticamente. Los usuarios avanzados que gestionan varias identidades pueden usar el flujo de shell anterior en su lugar.',
			'web.providers.claudeAccounts.noTokenYet' => 'aún no hay token',
			'web.providers.claudeAccounts.configDir' => 'config_dir:',
			'web.providers.claudeAccounts.tokenPath' => 'token_path:',
			'web.providers.claudeAccounts.toggleFailedToast' => 'Error al alternar',
			'web.providers.claudeAccounts.removeConfirm' => ({required Object name}) => '¿Quitar la cuenta "${name}"?',
			'web.providers.claudeAccounts.removedToast' => 'Cuenta eliminada',
			'web.providers.claudeAccounts.removeFailedToast' => 'Error al eliminar',
			'web.providers.claudeAccounts.toggleAria' => ({required Object name}) => 'Alternar ${name}',
			'web.providers.claudeAccounts.removeAria' => ({required Object name}) => 'Quitar ${name}',
			'web.providers.claudeAccounts.identityAcceptedToast' => 'Nueva identidad registrada',
			'web.providers.claudeAccounts.identityAcceptFailedToast' => 'No se pudo aceptar la identidad',
			'web.providers.models.title' => 'Modelos',
			'web.providers.models.help' => 'Modelos ofrecidos para este proveedor. El predeterminado se pasa a cada session mediante el flag de modelo; las sessions aún pueden sobrescribirlo.',
			'web.providers.models.empty' => 'Aún no hay modelos configurados.',
			'web.providers.models.add' => 'Añadir',
			'web.providers.models.addPlaceholder' => 'id del modelo (p. ej. sonnet)',
			'web.providers.models.suggested' => ({required Object count}) => 'Sugeridos (${count})',
			'web.providers.models.kDefault' => 'predeterminado',
			'web.providers.models.makeDefault' => 'establecer como predeterminado',
			'web.providers.models.setDefault' => 'Usar como modelo predeterminado',
			'web.providers.models.remove' => ({required Object model}) => 'Quitar ${model}',
			'web.channels.title' => 'Canales',
			'web.channels.subtitle' => 'Integraciones de mensajería bidireccional. Cada canal habilitado y no silenciado recibe notificaciones de sesión.',
			'web.channels.newButton' => 'Nuevo canal',
			'web.channels.loading' => 'Cargando…',
			'web.channels.empty.title' => 'Aún no hay canales',
			'web.channels.empty.description' => 'Tipos incluidos: Telegram · Slack · Discord · Feishu · DingTalk · WeCom. Elige uno y pega las credenciales, o usa <1>bridge</1> para una plataforma personalizada vía WebSocket.',
			'web.channels.card.running' => 'en ejecución',
			'web.channels.card.starting' => 'iniciando…',
			'web.channels.card.disabled' => 'desactivado',
			'web.channels.card.muted' => 'silenciado',
			'web.channels.card.tokenLabel' => 'token:',
			'web.channels.card.chatIdLabel' => 'chat_id:',
			'web.channels.card.channelIdLabel' => 'channel_id:',
			'web.channels.card.webhookLabel' => 'webhook:',
			'web.channels.card.copyWebhookTooltip' => 'Copiar la URL del webhook',
			'web.channels.card.webhookCopiedToast' => 'URL del webhook copiada',
			'web.channels.card.setup' => 'Configuración',
			'web.channels.card.setupTooltip' => 'Mostrar los detalles de conexión del adaptador y código de ejemplo',
			'web.channels.card.test' => 'Probar',
			'web.channels.card.testNotRunningTooltip' => 'El canal debe estar en ejecución',
			'web.channels.card.testBridgeTooltip' => 'Los canales bridge no se pueden probar desde el panel de administración, conecta primero un adaptador',
			'web.channels.card.editAria' => 'Editar canal',
			'web.channels.card.editTooltip' => 'Editar la configuración del canal',
			'web.channels.card.deleteAria' => 'Eliminar canal',
			'web.channels.card.muteAria' => 'Silenciar o reactivar el canal',
			'web.channels.card.muteTooltip' => 'Silenciar notificaciones (el chat bidireccional sigue funcionando)',
			'web.channels.card.unmuteTooltip' => 'Reactivar notificaciones',
			'web.channels.card.bridgeSuffix' => '(bridge)',
			'web.channels.toasts.testSent' => 'Mensaje de prueba enviado',
			'web.channels.toasts.testFailed' => 'La prueba falló',
			'web.channels.toasts.deleteConfirm' => ({required Object id}) => '¿Eliminar el canal ${id}?',
			'web.channels.toasts.deleted' => 'Canal eliminado',
			'web.channels.toasts.created' => 'Canal creado',
			'web.channels.toasts.updated' => 'Canal actualizado',
			'web.channels.toasts.muted' => 'Canal silenciado',
			'web.channels.toasts.unmuted' => 'Canal reactivado',
			'web.channels.dialog.editTitle' => 'Editar canal',
			'web.channels.dialog.createTitle' => 'Registrar canal',
			'web.channels.dialog.descriptionBridge' => 'Un adaptador externo (Python/Node/...) se conecta vía WebSocket y presenta este token.',
			'web.channels.dialog.descriptionDefault' => 'Configura la integración de mensajería.',
			'web.channels.dialog.kindLabel' => 'Tipo',
			'web.channels.dialog.kindImmutable' => '(inmutable, elimina y vuelve a crear para cambiar el tipo)',
			'web.channels.dialog.enabledLabel' => 'Activado',
			'web.channels.dialog.enabledBridgeHint' => ' (acepta conexiones de adaptadores de inmediato)',
			'web.channels.dialog.enabledWebhookHint' => ' (empieza a recibir webhooks de inmediato)',
			'web.channels.dialog.enabledDefaultHint' => ' (empieza de inmediato)',
			'web.channels.dialog.cancel' => 'Cancelar',
			'web.channels.dialog.save' => 'Guardar',
			'web.channels.dialog.saving' => 'Guardando…',
			'web.channels.dialog.create' => 'Crear',
			'web.channels.dialog.creating' => 'Creando…',
			'web.channels.dialog.unknownKind' => ({required Object kind}) => 'Tipo desconocido: ${kind}',
			'web.channels.dialog.nameRequired' => 'el nombre es obligatorio',
			'web.channels.dialog.tokenRequired' => 'el token es obligatorio',
			'web.channels.dialog.topicIdsNumeric' => ({required Object value}) => 'Los ID de tema deben ser numéricos (se recibió ${value})',
			'web.channels.dialog.fieldRequired' => ({required Object label}) => '${label} es obligatorio',
			'web.channels.dialog.cooldownInvalid' => 'El cooldown debe ser un número de segundos no negativo',
			'web.channels.dialog.snippetCapInvalid' => 'El límite del fragmento debe ser un número no negativo',
			'web.channels.notifications.sectionTitle' => 'Notificaciones de session',
			'web.channels.notifications.repeatPolicyLabel' => 'Política de repetición',
			'web.channels.notifications.cooldownLabel' => 'Duración del cooldown',
			'web.channels.notifications.onceReplyHint' => 'Responder con texto que no sea un comando en este chat restablece la supresión, opendray reenvía tu respuesta al stdin de la session y rearma el notificador.',
			'web.channels.notifications.terminalSnippetLabel' => 'Fragmento de terminal',
			'web.channels.notifications.embedSnippetLabel' => 'Incrustar la pantalla reciente del terminal en las notificaciones de inactividad',
			'web.channels.notifications.snippetExplainer' => 'Cuando está activado, la tarjeta de inactividad incluye un fragmento en bloque de código de lo que el usuario vería en el terminal web en vivo, los elementos de la interfaz del TUI de Claude (indicador de estado, aviso de "bypass permissions", líneas separadoras) se filtran automáticamente.',
			'web.channels.notifications.modes.onceLabel' => 'Una vez por session (recomendado)',
			'web.channels.notifications.modes.onceHint' => 'Se dispara una vez cuando una session queda inactiva, luego permanece en silencio hasta que la session termine o respondas por este canal.',
			'web.channels.notifications.modes.cooldownLabel' => 'Cooldown por ventana de tiempo',
			'web.channels.notifications.modes.cooldownHint' => 'Suprime las repeticiones del mismo par (session, evento) dentro de la ventana elegida.',
			'web.channels.notifications.modes.everyLabel' => 'Cada evento (ruidoso)',
			'web.channels.notifications.modes.everyHint' => 'Sin supresión. Úsalo solo para canales de baja frecuencia.',
			'web.channels.notifications.cooldowns.k60' => '1 minuto',
			'web.channels.notifications.cooldowns.k300' => '5 minutos',
			'web.channels.notifications.cooldowns.k900' => '15 minutos',
			'web.channels.notifications.cooldowns.k1800' => '30 minutos',
			'web.channels.notifications.cooldowns.k3600' => '1 hora',
			'web.channels.notifications.snippetCaps.k0' => 'Sin límite, dividir en varios mensajes (predeterminado)',
			'web.channels.notifications.snippetCaps.k1000' => '1000 caracteres (conciso)',
			'web.channels.notifications.snippetCaps.k3000' => '3000 caracteres',
			'web.channels.notifications.snippetCaps.k6000' => '6000 caracteres',
			'web.channels.notifications.snippetCaps.k12000' => '12000 caracteres',
			_ => null,
		} ?? switch (path) {
			'web.channels.bridge.nameLabel' => 'Nombre del bridge',
			'web.channels.bridge.namePlaceholder' => 'wechat / discord-custom / whatsapp...',
			'web.channels.bridge.nameHint' => 'Etiqueta legible para el adaptador. Se muestra en la lista de canales.',
			'web.channels.bridge.tokenLabel' => 'Token del adaptador',
			'web.channels.bridge.regenerateTooltip' => 'Regenerar',
			'web.channels.bridge.copyTooltip' => 'Copiar',
			'web.channels.bridge.tokenCopiedToast' => 'Token copiado',
			'web.channels.bridge.tokenHint' => 'El adaptador se autentica enviándolo en el frame de registro de WS (o como cabecera <1>X-Bridge-Token</1>).',
			'web.channels.bridge.capsLabel' => 'Aceptar capacidades (lista blanca opcional)',
			'web.channels.bridge.capsHint' => 'Vacío = aceptar lo que declare el adaptador. Seleccionado = permitir solo estas capacidades aunque el adaptador ofrezca más.',
			'web.channels.bridge.afterCreate' => 'Tras <1>Crear</1>, el diálogo de configuración del adaptador se abre automáticamente con la URL del WebSocket y código de inicio en Python / Node / wscat listo para copiar y pegar.',
			'web.channels.setup.title' => ({required Object name}) => 'Configuración del adaptador, ${name}',
			'web.channels.setup.description' => 'Ejecuta un adaptador (en cualquier lenguaje) que se conecte a opendray vía WebSocket usando estas credenciales. opendray enrutará las notificaciones de session y las acciones de los comandos a través de él.',
			'web.channels.setup.wsUrlLabel' => 'URL del WebSocket',
			'web.channels.setup.tokenLabel' => 'Token del adaptador',
			'web.channels.setup.authInfo' => ({required Object frame}) => '<1>Auth:</1> envía el token como cabecera <3>X-Bridge-Token</3>, parámetro de consulta <5>?token=</5> o <7>Authorization: Bearer …</7>. El primer frame de WS debe ser <9>${frame}</9>. Especificación completa: <11>docs/bridge-protocol.md</11> en el repo.',
			'web.channels.setup.pythonInstall' => 'Instalar: <1>pip install websockets</1>. Ejecutar: <3>python adapter.py</3>.',
			'web.channels.setup.nodeInstall' => 'Instalar: <1>npm i ws</1>. Ejecutar: <3>node adapter.mjs</3>.',
			'web.channels.setup.wscatInstall' => 'Instalar: <1>npm i -g wscat</1>. Una vez conectado, pega la línea JSON mostrada arriba para registrarte, luego envía más frames manualmente.',
			'web.channels.setup.close' => 'Cerrar',
			'web.channels.setup.copyHide' => 'Ocultar',
			'web.channels.setup.copyShow' => 'Mostrar',
			'web.channels.setup.copyLabelToast' => ({required Object label}) => '${label} copiado',
			'web.channels.setup.copyCode' => 'Copiar',
			'web.channels.setup.copied' => 'Copiado',
			'web.channels.setup.codeCopiedToast' => 'Código copiado',
			'web.integrations.title' => 'Integraciones',
			'web.integrations.subtitle' => 'Aplicaciones externas que consumen opendray. Reenvían mediante reverse-proxy a través de <1>/api/v1/proxy/&lt;prefix&gt;/…</1> y se suscriben a eventos a través del endpoint WS.',
			'web.integrations.register' => 'Registrar',
			'web.integrations.loading' => 'Cargando…',
			'web.integrations.tabs.registered' => 'Registradas',
			'web.integrations.tabs.console' => 'Reverse proxy',
			'web.integrations.empty.title' => 'Aún no hay integraciones',
			'web.integrations.empty.description' => 'Registra una aplicación externa para darle una API key con alcance limitado. Su código se queda fuera de este repositorio.',
			'web.integrations.empty.register' => 'Registrar integración',
			'web.integrations.groupSystem' => 'Sistema (gestionado por opendray)',
			'web.integrations.groupOperator' => 'Registradas por el operador',
			'web.integrations.card.managedBadge' => 'gestionada',
			'web.integrations.card.managedTooltip' => 'opendray gestiona esta integración. Editar o rotar su key dejaría huérfanas las sessions en ejecución cuyo mcp.json contiene el bearer anterior.',
			'web.integrations.card.consumerBadge' => 'consumidora',
			'web.integrations.card.consumerTooltip' => 'Integración solo consumidora. No hay servicio HTTP que sondear',
			'web.integrations.card.disabledBadge' => 'deshabilitada',
			'web.integrations.card.consumerOnlyHint' => 'Consume la API de opendray. No tiene reverse proxy montado.',
			'web.integrations.card.lastProbed' => ({required Object relative}) => 'último sondeo ${relative}',
			'web.integrations.card.rotated' => ({required Object relative}) => 'rotada ${relative}',
			'web.integrations.card.managedReadOnly' => 'solo lectura. opendray incrusta su key en el mcp.json de cada spawn',
			'web.integrations.card.managedReadOnlyTooltip' => 'opendray gestiona esta fila. Para restablecerla: borra ~/.opendray/memory.key y reinicia, o borra esta fila directamente mediante SQL (se volverá a inicializar en el siguiente arranque).',
			'web.integrations.card.editAria' => 'Editar integración',
			'web.integrations.card.editTooltip' => 'Editar scopes / URL base / versión',
			'web.integrations.card.rotateKey' => 'Rotar key',
			'web.integrations.card.deleteAria' => 'Eliminar integración',
			'web.integrations.card.rotateConfirm' => ({required Object name}) => '¿Rotar la API key de "${name}"? La key actual dejará de funcionar de inmediato.',
			'web.integrations.card.deleteConfirm' => ({required Object name}) => '¿Eliminar la integración ${name}?',
			'web.integrations.card.removedToast' => 'Integración eliminada',
			'web.integrations.register_dialog.title' => 'Registrar integración',
			'web.integrations.register_dialog.description' => 'Emite una API key de un solo uso. Cópiala antes de cerrar: opendray nunca vuelve a mostrar el texto en claro.',
			'web.integrations.register_dialog.nameLabel' => 'Nombre',
			'web.integrations.register_dialog.namePlaceholder' => 'PetTracker',
			'web.integrations.register_dialog.modeHint' => 'Deja en blanco los dos campos siguientes para una integración <1>solo consumidora</1> (aplicación de terceros que llama a la API de opendray pero no expone su propio servicio). Rellena ambos para una integración con <3>reverse-proxy</3>.',
			'web.integrations.register_dialog.baseUrlLabel' => 'URL base',
			'web.integrations.register_dialog.optionalSuffix' => '(opcional)',
			'web.integrations.register_dialog.baseUrlPlaceholder' => 'http://192.168.1.10:8080',
			'web.integrations.register_dialog.routePrefixLabel' => 'Prefijo de ruta',
			'web.integrations.register_dialog.routePrefixPlaceholder' => 'pet-tracker',
			'web.integrations.register_dialog.routePrefixHint' => ({required Object prefix}) => 'Accesible en <1>/api/v1/proxy/${prefix}/*</1>.',
			'web.integrations.register_dialog.routePrefixPlaceholderToken' => '<prefix>',
			'web.integrations.register_dialog.versionLabel' => 'Versión (opcional)',
			'web.integrations.register_dialog.versionPlaceholder' => '0.1.0',
			'web.integrations.register_dialog.scopesLabel' => 'Scopes',
			'web.integrations.register_dialog.scopesIntro' => 'Elige la superficie de API que esta integración tiene permitido llamar. Cada interruptor se corresponde con un claim del Bearer-token: opendray rechaza las peticiones que tocan endpoints fuera del conjunto concedido.',
			'web.integrations.register_dialog.errorNameRequired' => 'El nombre es obligatorio.',
			'web.integrations.register_dialog.errorBothOrNeither' => 'base_url y route_prefix van juntos. Configura ambos para una integración con reverse-proxy, o deja ambos en blanco para una integración solo consumidora.',
			'web.integrations.register_dialog.cancel' => 'Cancelar',
			'web.integrations.register_dialog.submit' => 'Registrar',
			'web.integrations.register_dialog.submitting' => 'Registrando…',
			'web.integrations.reveal.titleIssued' => 'API key emitida',
			'web.integrations.reveal.titleRotated' => 'API key rotada',
			'web.integrations.reveal.description' => 'Esta es la única vez que se mostrará la key en texto en claro. Cópiala ahora y actualiza todas las aplicaciones consumidoras: la key anterior (si la había) ya no autentica.',
			'web.integrations.reveal.discardAria' => 'Descartar la nueva key',
			'web.integrations.reveal.discardTooltip' => 'Descartar la nueva key (la rotación ya ha ocurrido, la key antigua también desapareció)',
			'web.integrations.reveal.discardConfirm' => '¿Descartar la nueva key? La rotación ya ha invalidado la key antigua: descartarla significa que NO tendrás ninguna key funcional para esta integración hasta que vuelvas a rotar.',
			'web.integrations.reveal.copy' => 'Copiar',
			'web.integrations.reveal.copied' => 'Copiada',
			'web.integrations.reveal.updateHint' => '<1>Actualiza todas las aplicaciones consumidoras con esta nueva key.</1> La key anterior se ha invalidado en el servidor y devolverá <3>401 unauthorized</3> en la siguiente petición.',
			'web.integrations.reveal.acknowledge' => 'He copiado la key y actualizaré mis aplicaciones consumidoras. Entiendo que opendray no la volverá a mostrar.',
			'web.integrations.reveal.discard' => 'Descartar',
			'web.integrations.reveal.done' => 'Hecho',
			'web.integrations.edit_dialog.title' => ({required Object name}) => 'Editar integración · ${name}',
			'web.integrations.edit_dialog.description' => 'Cambia los scopes, la versión o la URL base. El nombre y el prefijo de ruta son inmutables: elimina y vuelve a registrar si necesitas cambiarlos.',
			'web.integrations.edit_dialog.nameLabel' => 'Nombre',
			'web.integrations.edit_dialog.routePrefixLabel' => 'Prefijo de ruta',
			'web.integrations.edit_dialog.consumerOnlyLabel' => '(solo consumidora)',
			'web.integrations.edit_dialog.baseUrlLabel' => 'URL base',
			'web.integrations.edit_dialog.baseUrlConsumerSuffix' => '(solo consumidora, deja en blanco)',
			'web.integrations.edit_dialog.baseUrlProxySuffix' => '(destino del reverse-proxy)',
			'web.integrations.edit_dialog.baseUrlConsumerPlaceholder' => '(en blanco: esta integración consume la API de opendray)',
			'web.integrations.edit_dialog.baseUrlProxyPlaceholder' => 'http://127.0.0.1:8080',
			'web.integrations.edit_dialog.consumerHint' => 'Esta es una integración solo consumidora. Cambiar la URL base aquí también requeriría un prefijo de ruta; hazlo eliminando y volviendo a registrar.',
			'web.integrations.edit_dialog.versionLabel' => 'Versión',
			'web.integrations.edit_dialog.versionPlaceholder' => '0.1.0',
			'web.integrations.edit_dialog.scopesLabel' => 'Scopes',
			'web.integrations.edit_dialog.scopesIntro' => 'Reduce o amplía la superficie de API que autoriza la API key de esta integración. Los tokens activos no se ven afectados: el nuevo conjunto de scopes surte efecto en la siguiente petición.',
			'web.integrations.edit_dialog.errorModeSwitch' => 'Cambiar entre modo solo consumidora y reverse-proxy requiere eliminar la integración y volver a registrarla: el nombre y route_prefix no pueden cambiarse sobre la marcha.',
			'web.integrations.edit_dialog.updatedToast' => 'Integración actualizada',
			'web.integrations.edit_dialog.cancel' => 'Cancelar',
			'web.integrations.edit_dialog.save' => 'Guardar cambios',
			'web.integrations.proxy.emptyTitle' => 'No hay integraciones registradas',
			'web.integrations.proxy.emptyDescription' => ({required Object prefix}) => 'Registra primero una integración; la consola hace de proxy a través de /api/v1/proxy/${prefix}/* usando el token de administrador.',
			'web.integrations.proxy.targetLabel' => 'Destino',
			'web.integrations.proxy.selectPlaceholder' => 'Selecciona una integración…',
			'web.integrations.proxy.baseLabel' => 'base:',
			'web.integrations.proxy.history' => 'Historial',
			'web.integrations.proxy.historyEmpty' => 'no hay peticiones anteriores para esta integración',
			'web.integrations.proxy.send' => 'Enviar',
			'web.integrations.proxy.sending' => 'Enviando…',
			'web.integrations.proxy.extraHeadersLabel' => 'Headers adicionales (uno por línea, Nombre: Valor)',
			'web.integrations.proxy.bodyLabel' => 'Body',
			'web.integrations.proxy.headers' => 'Headers',
			'web.integrations.proxy.body' => 'Body',
			'web.integrations.proxy.emptyBody' => '(vacío)',
			'web.integrations.proxy.requestFailed' => 'la petición falló',
			'web.integrations.proxy.stubText' => 'Envía una petición para ver la respuesta del upstream.',
			'web.integrations.proxy.stubInjects' => 'opendray inyecta <1>X-Integration-ID</1> y elimina tu header <3>Authorization</3>.',
			'web.integrations.proxy.prefixPlaceholder' => '<prefix>',
			'web.plugins.title' => 'Plugins del Inspector',
			'web.plugins.subtitle' => 'Configura las fuentes de datos que se muestran en el panel Inspector de la derecha cuando hay una session abierta. Cada plugin es solo para administradores y se comparte entre todas las sessions. Haz clic en el encabezado de una sección para contraerla.',
			'web.plugins.common.loading' => 'Cargando…',
			'web.plugins.common.cancel' => 'Cancelar',
			'web.plugins.common.edit' => 'Editar',
			'web.plugins.common.add' => 'Añadir',
			'web.plugins.common.save' => 'Guardar',
			'web.plugins.common.create' => 'Crear',
			'web.plugins.mcp.title' => 'Servidores MCP',
			'web.plugins.mcp.description' => ({required Object KEY}) => 'Servidores Model Context Protocol inyectados en cada spawn (claude / codex). Las entradas del vault están en <1>~/.opendray/vault/mcp/&lt;id&gt;/mcp.json</1>; los secretos (referenciados como <3>\$${KEY}</3> en env / headers) provienen de la sección <5>secretos MCP</5> de abajo.',
			'web.plugins.mcp.newServer' => 'Nuevo servidor',
			'web.plugins.mcp.empty' => 'Aún no hay servidores MCP. Añade uno para exponer herramientas adicionales a tus sessions de agente.',
			'web.plugins.mcp.columns.name' => 'Nombre',
			'web.plugins.mcp.columns.transport' => 'Transport',
			'web.plugins.mcp.columns.spec' => 'Spec',
			'web.plugins.mcp.columns.enabled' => 'Habilitado',
			'web.plugins.mcp.noUrl' => 'sin url',
			'web.plugins.mcp.noCommand' => 'sin comando',
			'web.plugins.mcp.deleteConfirm' => ({required Object id}) => '¿Eliminar el servidor MCP "${id}"?',
			'web.plugins.mcp.removedToast' => 'Servidor MCP eliminado',
			'web.plugins.mcp.deleteFailedToast' => 'Error al eliminar',
			'web.plugins.mcp.toggleFailedToast' => 'Error al alternar',
			'web.plugins.mcp.codexUnsupportedBadge' => 'Codex: no compatible',
			'web.plugins.mcp.codexUnsupportedTooltip' => 'El CLI de codex solo admite el transport stdio. Este servidor se omitirá en las sessions de codex; claude y gemini lo seguirán usando.',
			'web.plugins.mcp.builtinBadge' => 'Integrado',
			'web.plugins.mcp.builtinTooltip' => 'Provisto por el propio opendray — se adjunta automáticamente a cada session que admite MCP. No se puede editar ni eliminar.',
			'web.plugins.mcp.builtinDescription' => 'El servidor compartido de memoria y conocimiento de opendray: memory_search / memory_store, project_goal y project_plan get/set, session_log_append, decision_record, doc_read, skill_distill, project_search. Se adjunta automáticamente a cada session de Claude / Codex / Gemini.',
			'web.plugins.mcp.builtinAutoAttach' => 'siempre activo',
			'web.plugins.mcp.editor.createTitle' => 'Nuevo servidor MCP',
			'web.plugins.mcp.editor.editTitle' => ({required Object id}) => 'Editar MCP: ${id}',
			'web.plugins.mcp.editor.description' => ({required Object API_KEY}) => 'Forma del JSON: <1>command</1>+<3>args</3>+<5>env</5> para stdio (predeterminado), o <7>transport</7> +<9> url</9>+<11>headers</11> para sse / http. Referencia los secretos como <13>\$${API_KEY}</13>, se sustituyen en el momento del spawn desde el archivo de secretos.',
			'web.plugins.mcp.editor.idLabel' => 'ID',
			'web.plugins.mcp.editor.idPlaceholder' => 'filesystem',
			'web.plugins.mcp.editor.idHint' => 'Minúsculas / dígitos / guion / guion bajo. Se convierte tanto en el nombre del directorio como en el <1>name</1> predeterminado.',
			'web.plugins.mcp.editor.bodyLabel' => 'mcp.json',
			'web.plugins.mcp.editor.invalidJson' => ({required Object error}) => 'JSON no válido: ${error}',
			'web.plugins.mcp.editor.createdToast' => 'Servidor MCP creado',
			'web.plugins.mcp.editor.savedToast' => 'Servidor MCP guardado',
			'web.plugins.mcp.editor.createFailedToast' => 'Error al crear',
			'web.plugins.mcp.editor.saveFailedToast' => 'Error al guardar',
			'web.plugins.mcp.editor.transportLabel' => 'Transport',
			'web.plugins.mcp.editor.transportHint' => 'Cambiar el transport reemplaza la plantilla JSON por una forma inicial adecuada para el nuevo transport.',
			'web.plugins.mcp.editor.transportStdio' => 'stdio (subproceso local)',
			'web.plugins.mcp.editor.transportSse' => 'sse (servidor remoto)',
			'web.plugins.mcp.editor.transportHttp' => 'http (servidor remoto)',
			'web.plugins.mcp.test.button' => 'Probar',
			'web.plugins.mcp.test.title' => 'Validar este servidor MCP desde el daemon',
			'web.plugins.mcp.test.connected' => ({required Object count}) => 'conectado · ${count} herramientas',
			'web.plugins.mcp.test.reachable' => 'accesible',
			'web.plugins.mcp.test.failed' => 'prueba fallida',
			'web.plugins.mcpSecrets.title' => 'Secretos MCP',
			'web.plugins.mcpSecrets.encryptedBadge' => 'cifrado',
			'web.plugins.mcpSecrets.plaintextBadge' => 'texto plano',
			'web.plugins.mcpSecrets.encryptedTooltip' => 'Cifrado AES-GCM en disco; la clave se almacena en el llavero del SO',
			'web.plugins.mcpSecrets.plaintextTooltip' => 'Llavero del SO no disponible. El archivo está en texto plano en disco. Revisa el log del gateway.',
			'web.plugins.mcpSecrets.description' => ({required Object KEY}) => 'Los valores referenciados desde los marcadores <1>\$${KEY}</1> en cualquier <3>mcp.json</3> se sustituyen en el momento del spawn. <5>Los valores guardados nunca se devuelven a través de la API</5>, puedes sobrescribirlos o eliminarlos, pero no volver a leerlos.',
			'web.plugins.mcpSecrets.descriptionStored' => ({required Object path}) => ' Almacenado en <1>${path}</1>.',
			'web.plugins.mcpSecrets.addSecret' => 'Añadir secreto',
			'web.plugins.mcpSecrets.empty' => ({required Object KEY}) => 'No hay secretos almacenados. Añade uno para empezar a referenciarlo como <1>\$${KEY}</1> en las configuraciones de tus servidores MCP.',
			'web.plugins.mcpSecrets.columns.key' => 'Clave',
			'web.plugins.mcpSecrets.columns.value' => 'Valor',
			'web.plugins.mcpSecrets.editTooltip' => 'Sobrescribir el valor almacenado',
			'web.plugins.mcpSecrets.deleteConfirm' => ({required Object key}) => '¿Eliminar el secreto "${key}"? Cualquier mcp.json que referencie \$${key} recurrirá al marcador literal hasta que establezcas un nuevo valor.',
			'web.plugins.mcpSecrets.removedToast' => 'Secreto eliminado',
			'web.plugins.mcpSecrets.deleteFailedToast' => 'Error al eliminar',
			'web.plugins.mcpSecrets.editor.addTitle' => 'Añadir secreto',
			'web.plugins.mcpSecrets.editor.updateTitle' => ({required Object key}) => 'Actualizar ${key}',
			'web.plugins.mcpSecrets.editor.addDescription' => ({required Object KEY}) => 'Se almacena cifrado en disco si el llavero del SO está disponible. Referéncialo desde el env / headers / args / url de cualquier mcp.json con \$${KEY}.',
			'web.plugins.mcpSecrets.editor.editDescription' => 'Introduce el nuevo valor para sobrescribir. El valor anterior no se puede recuperar.',
			'web.plugins.mcpSecrets.editor.keyLabel' => 'Clave',
			'web.plugins.mcpSecrets.editor.keyPlaceholder' => 'BRAVE_API_KEY',
			'web.plugins.mcpSecrets.editor.keyPattern' => 'Debe coincidir con <1>[A-Za-z_][A-Za-z0-9_]*</1>',
			'web.plugins.mcpSecrets.editor.keyCollision' => 'Ya existe. Usa Editar en su lugar, o elige un nombre diferente.',
			'web.plugins.mcpSecrets.editor.valueLabel' => 'Valor',
			'web.plugins.mcpSecrets.editor.valueHint' => 'Oculto mientras escribes. El valor guardado nunca se devuelve a través de la API.',
			'web.plugins.mcpSecrets.editor.addedToast' => 'Secreto añadido',
			'web.plugins.mcpSecrets.editor.updatedToast' => 'Secreto actualizado',
			'web.plugins.mcpSecrets.editor.saveFailedToast' => 'Error al guardar',
			'web.plugins.skills.title' => 'Habilidades del agente',
			'web.plugins.skills.description' => 'Capacidades reutilizables inyectadas en las sessions de Claude como un índice de Tier 1, el agente carga el SKILL.md completo bajo demanda mediante <1>opendray skill describe &lt;id&gt;</1>. Las integradas vienen en el binario pero se pueden <3>personalizar</3>, tus ediciones se guardan en <5>~/.opendray/vault/skills/&lt;id&gt;/SKILL.md</5> y anulan la versión incorporada. Usa Restablecer para revertir.',
			'web.plugins.skills.newSkill' => 'Nueva habilidad',
			'web.plugins.skills.empty' => 'No se encontraron habilidades.',
			'web.plugins.skills.columns.id' => 'ID',
			'web.plugins.skills.columns.description' => 'Descripción',
			'web.plugins.skills.columns.source' => 'Origen',
			'web.plugins.skills.noDescription' => 'sin descripción',
			'web.plugins.skills.builtinBadge' => 'integrada',
			'web.plugins.skills.builtinTooltip' => 'Incorporada en el binario de opendray, haz clic en Personalizar para anularla en tu vault',
			'web.plugins.skills.vaultBadge' => 'vault',
			'web.plugins.skills.overridesBuiltin' => 'anula la integrada',
			'web.plugins.skills.overridesBuiltinTooltip' => 'Esta habilidad del vault anula la versión integrada del mismo id',
			'web.plugins.skills.customize' => 'Personalizar',
			'web.plugins.skills.customizeTooltip' => 'Abre el SKILL.md y guarda los cambios como una anulación del vault',
			'web.plugins.skills.editTooltip' => 'Editar esta habilidad del vault',
			'web.plugins.skills.resetTooltip' => 'Eliminar la anulación del vault y volver a la versión integrada',
			'web.plugins.skills.reset' => 'Restablecer',
			'web.plugins.skills.resetConfirm' => ({required Object id}) => '¿Restablecer "${id}" a la versión integrada? Esto elimina tu SKILL.md del vault y vuelve a la copia incorporada.',
			'web.plugins.skills.deleteConfirm' => ({required Object id}) => '¿Eliminar la habilidad "${id}" de tu vault? Esto elimina el archivo SKILL.md.',
			'web.plugins.skills.removedToast' => 'Habilidad eliminada',
			'web.plugins.skills.deleteFailedToast' => 'Error al eliminar',
			'web.plugins.skills.editor.createTitle' => 'Nueva habilidad',
			'web.plugins.skills.editor.customizeTitle' => ({required Object id}) => 'Personalizar la integrada: ${id}',
			'web.plugins.skills.editor.editTitle' => ({required Object id}) => 'Editar habilidad: ${id}',
			'web.plugins.skills.editor.customizeDescription' => 'Estás viendo una habilidad integrada incorporada en opendray. Al guardar se creará una anulación del vault con el mismo id, tus ediciones se guardan en ~/.opendray/vault/skills/<id>/SKILL.md y ocultan la integrada hasta que la Restablezcas.',
			'web.plugins.skills.editor.editDescription' => 'Formato SKILL.md: frontmatter con name + description, luego instrucciones en markdown. La descripción aparece en el índice de Tier 1 del agente.',
			'web.plugins.skills.editor.idLabel' => 'ID',
			'web.plugins.skills.editor.idPlaceholder' => 'my-helper',
			'web.plugins.skills.editor.idHint' => 'Minúsculas / dígitos / guion / guion bajo. Se convierte en el nombre del directorio bajo <1>~/.opendray/vault/skills/&lt;id&gt;/</1>.',
			'web.plugins.skills.editor.bodyLabel' => 'SKILL.md',
			'web.plugins.skills.editor.createdToast' => 'Habilidad creada',
			'web.plugins.skills.editor.savedToast' => 'Habilidad guardada',
			'web.plugins.skills.editor.savedOverrideToast' => 'Guardada como anulación del vault',
			'web.plugins.skills.editor.createFailedToast' => 'Error al crear',
			'web.plugins.skills.editor.saveFailedToast' => 'Error al guardar',
			'web.plugins.skills.editor.saveAsOverride' => 'Guardar como anulación del vault',
			'web.plugins.skills.dropHint' => 'O suelta un SKILL.md aquí para instalarlo.',
			'web.plugins.skills.dropToInstall' => 'Suelta el SKILL.md para instalar',
			'web.plugins.skills.uploading' => 'Instalando habilidad…',
			'web.plugins.skills.uploadedToast' => ({required Object id}) => 'Habilidad "${id}" instalada',
			'web.plugins.skills.uploadFailedToast' => 'Error al subir la habilidad',
			'web.plugins.skills.uploadInvalidTypeToast' => 'Solo se pueden instalar archivos SKILL.md por arrastre',
			'web.plugins.customTasks.title' => 'Tareas personalizadas',
			'web.plugins.customTasks.description' => 'Atajos de ejecución con un clic que se muestran en la pestaña Tareas. Deja cwd en blanco para tareas globales visibles en todas las sessions, o fíjalo a una ruta absoluta para acotarlo.',
			'web.plugins.customTasks.addTask' => 'Añadir tarea',
			'web.plugins.customTasks.empty' => 'Aún no hay tareas personalizadas.',
			'web.plugins.customTasks.columns.name' => 'Nombre',
			'web.plugins.customTasks.columns.command' => 'Comando',
			'web.plugins.customTasks.columns.scope' => 'Ámbito',
			'web.plugins.customTasks.globalScope' => 'global',
			'web.plugins.customTasks.deleteConfirm' => ({required Object name}) => '¿Eliminar la tarea personalizada "${name}"?',
			'web.plugins.customTasks.removedToast' => 'Tarea eliminada',
			'web.plugins.customTasks.deleteFailedToast' => 'Error al eliminar',
			'web.plugins.customTasks.dialog.addTitle' => 'Añadir tarea personalizada',
			'web.plugins.customTasks.dialog.editTitle' => ({required Object name}) => 'Editar ${name}',
			'web.plugins.customTasks.dialog.description' => 'El comando se envía textualmente a la terminal de la session. Es lo mismo que escribirlo en el prompt y pulsar Enter.',
			'web.plugins.customTasks.dialog.nameLabel' => 'Nombre',
			'web.plugins.customTasks.dialog.namePlaceholder' => 'dev',
			'web.plugins.customTasks.dialog.commandLabel' => 'Comando',
			'web.plugins.customTasks.dialog.commandPlaceholder' => 'docker compose up --build',
			'web.plugins.customTasks.dialog.descLabel' => 'Descripción (opcional)',
			'web.plugins.customTasks.dialog.descPlaceholder' => 'Arranca la infraestructura de desarrollo y sigue los logs',
			'web.plugins.customTasks.dialog.cwdLabel' => 'Ámbito de cwd (opcional)',
			'web.plugins.customTasks.dialog.cwdPlaceholder' => '/Users/me/projects/foo  (en blanco = global)',
			'web.plugins.customTasks.dialog.cwdHint' => 'En blanco = visible en todas las sessions. De lo contrario, la tarea solo se muestra cuando el cwd de la session coincide con esta ruta absoluta.',
			'web.plugins.customTasks.dialog.addedToast' => 'Tarea añadida',
			'web.plugins.customTasks.dialog.updatedToast' => 'Tarea actualizada',
			'web.plugins.customTasks.dialog.addFailedToast' => 'Error al añadir',
			'web.plugins.customTasks.dialog.updateFailedToast' => 'Error al actualizar',
			'web.plugins.gitHosts.title' => 'Hosts de git',
			'web.plugins.gitHosts.description' => 'Un token por host, usado por la pestaña Git para obtener los pull requests <1>y por la sincronización del vault de Notas</1> cuando su remoto usa HTTPS hacia un repo privado en el mismo host. Se admiten GitHub.com, GitHub Enterprise autoalojado, Gitea y GitLab.',
			'web.plugins.gitHosts.addHost' => 'Añadir host',
			'web.plugins.gitHosts.empty' => 'No hay hosts de git configurados.\nAñade uno para habilitar la lista de PR en la pestaña Git del inspector.',
			'web.plugins.gitHosts.columns.host' => 'Host',
			'web.plugins.gitHosts.columns.kind' => 'Tipo',
			'web.plugins.gitHosts.columns.token' => 'Token',
			'web.plugins.gitHosts.columns.enabled' => 'Habilitado',
			'web.plugins.gitHosts.statusEnabled' => 'habilitado',
			'web.plugins.gitHosts.statusDisabled' => 'deshabilitado',
			'web.plugins.gitHosts.deleteConfirm' => ({required Object host}) => '¿Eliminar el host de git ${host}? Las consultas de PR contra este host dejarán de funcionar.',
			'web.plugins.gitHosts.removedToast' => 'Host de git eliminado',
			'web.plugins.gitHosts.deleteFailedToast' => 'Error al eliminar',
			'web.plugins.gitHosts.dialog.addTitle' => 'Añadir host de git',
			'web.plugins.gitHosts.dialog.editTitle' => ({required Object host}) => 'Editar ${host}',
			'web.plugins.gitHosts.dialog.description' => 'El token se almacena en el gateway. Se usa solo para llamadas de solo lectura a la API (listar PR, etc.).',
			'web.plugins.gitHosts.dialog.kindLabel' => 'Tipo',
			'web.plugins.gitHosts.dialog.kindGitHub' => 'GitHub',
			'web.plugins.gitHosts.dialog.kindGitea' => 'Gitea',
			'web.plugins.gitHosts.dialog.kindGitLab' => 'GitLab',
			'web.plugins.gitHosts.dialog.hostLabel' => 'Host',
			'web.plugins.gitHosts.dialog.hostPlaceholder' => 'github.com',
			'web.plugins.gitHosts.dialog.displayNameLabel' => 'Nombre visible (opcional)',
			'web.plugins.gitHosts.dialog.displayNamePlaceholder' => 'Personal',
			'web.plugins.gitHosts.dialog.tokenLabel' => 'Token',
			'web.plugins.gitHosts.dialog.newTokenLabel' => 'Nuevo token (déjalo en blanco para conservarlo)',
			'web.plugins.gitHosts.dialog.tokenPlaceholder' => 'ghp_… / gho_… / glpat-…',
			'web.plugins.gitHosts.dialog.tokenPlaceholderEdit' => '…',
			'web.plugins.gitHosts.dialog.tokenHint' => 'GitHub: PAT con scope <1>repo</1>. Gitea: token con <3>read:repository</3>. GitLab: PAT con <5>read_api</5>.',
			'web.plugins.gitHosts.dialog.enabledLabel' => 'Habilitado',
			'web.plugins.gitHosts.dialog.addedToast' => 'Host de git añadido',
			'web.plugins.gitHosts.dialog.updatedToast' => 'Host de git actualizado',
			'web.plugins.gitHosts.dialog.addFailedToast' => 'Error al añadir',
			'web.plugins.gitHosts.dialog.updateFailedToast' => 'Error al actualizar',
			'web.backups.title' => 'Copias de seguridad',
			'web.backups.subtitle' => 'Volcados cifrados de PostgreSQL escritos en un destino conectable. Configura programaciones y retención, o lanza copias puntuales para tener una red de seguridad rápida.',
			'web.backups.exportData' => 'Exportar datos',
			'web.backups.loading' => 'Cargando…',
			'web.backups.loadStatusFailedToast' => 'No se pudo cargar el estado de la copia de seguridad',
			'web.backups.tabs.backups' => 'Copias de seguridad',
			'web.backups.tabs.schedules' => 'Programaciones',
			'web.backups.tabs.targets' => 'Destinos',
			'web.backups.inventory.title' => '¿Qué hay en una copia de seguridad?',
			'web.backups.inventory.summary' => ({required Object rows, required Object tables}) => '${rows} filas en ${tables} tablas',
			'web.backups.inventory.description' => 'Cada copia de seguridad es un <1>pg_dump --format=custom</1> de cada tabla de abajo, más <3>manifest.json</3> y (opcionalmente) <5>config.toml</5>. Los recuentos son en vivo; el paquete captura lo que haya en el momento de la copia.',
			'web.backups.inventory.loadFailedToast' => 'No se pudo cargar el inventario',
			'web.backups.inventory.rowsLabel' => 'filas',
			'web.backups.restart.title' => 'Reinicia opendray para activar las copias de seguridad',
			'web.backups.restart.description' => 'Tu frase de contraseña está guardada. El gateway solo la carga al iniciarse, así que la función permanece desactivada hasta que reinicies el proceso.',
			'web.backups.restart.keyFile' => 'Archivo de clave:',
			'web.backups.restart.configuredVia' => 'Configurado mediante:',
			'web.backups.restart.envVar' => 'variable de entorno OPENDRAY_BACKUP_KEY',
			'web.backups.restart.checkAgain' => 'Comprobar de nuevo',
			'web.backups.setup.title' => 'Configurar copias de seguridad',
			'web.backups.setup.description' => 'Elige una frase de contraseña maestra. opendray la usa para cifrar cada blob de copia de seguridad. <1>Si la pierdes, tus copias de seguridad serán irrecuperables</1>, así que guárdala en un gestor de contraseñas (Vaultwarden, 1Password, …) antes de continuar.',
			'web.backups.setup.generate' => 'Generar',
			'web.backups.setup.pasteOwn' => 'Pegar la mía',
			'web.backups.setup.generateTitle' => 'Clave aleatoria de 256 bits',
			'web.backups.setup.generateHint' => 'El servidor genera una frase de contraseña criptográficamente aleatoria y la muestra una sola vez. Debes copiarla antes de continuar, no hay forma de recuperarla.',
			'web.backups.setup.pasteLabel' => 'Tu frase de contraseña',
			'web.backups.setup.pastePlaceholder' => 'Al menos 20 caracteres',
			'web.backups.setup.pasteHint' => 'Recomendado: más de 40 caracteres desde un gestor de contraseñas.',
			'web.backups.setup.savesTo' => 'Se guarda en:',
			'web.backups.setup.saving' => 'Guardando…',
			'web.backups.setup.generateAndSave' => 'Generar y guardar',
			'web.backups.setup.save' => 'Guardar',
			'web.backups.generated.title' => 'Guarda esta frase de contraseña AHORA',
			'web.backups.generated.description' => 'Esto se muestra <1>una sola vez</1>. No se podrá recuperar desde opendray ni desde ningún otro sitio. Cópiala en un gestor de contraseñas antes de continuar.',
			'web.backups.generated.copy' => 'Copiar',
			'web.backups.generated.copiedToast' => 'Frase de contraseña copiada al portapapeles',
			'web.backups.generated.copyFailedToast' => 'Error al copiar, selecciónala y cópiala manualmente',
			'web.backups.generated.savedTo' => 'Guardada en:',
			'web.backups.generated.ack' => 'He guardado esta frase de contraseña en mi gestor de contraseñas',
			'web.backups.generated.kContinue' => 'Continuar',
			'web.backups.status.pgDump' => 'pg_dump',
			'web.backups.status.pgRestore' => 'pg_restore',
			'web.backups.status.pgDumpUnavailable' => 'no disponible',
			'web.backups.status.pgDumpHint' => 'Las copias de seguridad no pueden ejecutarse hasta que pg_dump esté en PATH (o se haya definido su ruta absoluta en <1>backup.pg_dump_path</1>). Instala <3>postgresql-client</3> de la misma versión mayor que tu servidor y reinicia.',
			'web.backups.backupsTab.backupNow' => 'Hacer copia ahora',
			'web.backups.backupsTab.triggering' => 'Lanzando…',
			'web.backups.backupsTab.includeConfig' => 'incluir config.toml',
			'web.backups.backupsTab.fullInstance' => 'Instancia completa',
			'web.backups.backupsTab.fullInstanceHint' => 'Incluye también el vault (notes/skills/mcp), secrets.env y config.toml: todo lo necesario para reconstruir una instancia funcional, no solo su base de datos.',
			'web.backups.backupsTab.restoreFromFile' => 'Restaurar desde archivo',
			'web.backups.backupsTab.refresh' => 'Actualizar',
			'web.backups.backupsTab.queuedToast' => 'Copia de seguridad en cola',
			'web.backups.backupsTab.triggerFailedToast' => 'Error al lanzar',
			'web.backups.backupsTab.listFailedToast' => 'No se pudieron listar las copias de seguridad',
			'web.backups.backupsTab.deleteConfirm' => ({required Object id}) => '¿Eliminar la copia de seguridad ${id}? El blob se elimina de su destino.',
			'web.backups.backupsTab.deletedToast' => 'Copia de seguridad eliminada',
			'web.backups.backupsTab.deleteFailedToast' => 'Error al eliminar',
			'web.backups.backupsTab.empty' => 'Aún no hay copias de seguridad. Haz clic en "Hacer copia ahora" arriba para crear la primera.',
			'web.backups.backupsTab.columns.id' => 'ID',
			'web.backups.backupsTab.columns.type' => 'Tipo',
			'web.backups.backupsTab.columns.target' => 'Destino',
			'web.backups.backupsTab.columns.status' => 'Estado',
			'web.backups.backupsTab.columns.started' => 'Iniciada',
			'web.backups.backupsTab.columns.size' => 'Tamaño',
			'web.backups.backupsTab.columns.actions' => 'Acciones',
			'web.backups.backupsTab.downloadTooltip' => 'Descargar',
			'web.backups.backupsTab.deleteTooltip' => 'Eliminar',
			'web.backups.restore.title' => 'Restaurar desde un paquete de copia de seguridad',
			'web.backups.restore.bundleLabel' => 'Paquete cifrado (.tar.gz.enc)',
			'web.backups.restore.targetDsnLabel' => 'DSN de la base de datos de destino',
			'web.backups.restore.targetDsnHint' => '(en blanco = la propia base de datos de opendray, PELIGROSO)',
			'web.backups.restore.targetDsnPlaceholder' => 'postgres://user:pass@host:5432/dbname',
			'web.backups.restore.cleanLabel' => '--clean --if-exists (eliminar primero el esquema existente; obligatorio al restaurar sobre una base de datos con datos)',
			'web.backups.restore.auditNoteLabel' => 'Nota de auditoría (opcional)',
			'web.backups.restore.auditNotePlaceholder' => 'Motivo de la restauración, aparece en el slog',
			'web.backups.restore.ownDbWarning' => 'Estás restaurando en <1>la propia base de datos de opendray</1>. Con "--clean" activado, esto elimina todas las tablas y reproduce la copia de seguridad tal cual, es irreversible. Escribe <3>I understand</3> para continuar.',
			'web.backups.restore.confirmPlaceholder' => 'I understand',
			'web.backups.restore.confirmSentinel' => 'I understand',
			'web.backups.restore.pgRestoreOutput' => 'Salida de pg_restore (últimos 8 KiB)',
			'web.backups.restore.noPgRestoreOutput' => '(sin salida de pg_restore)',
			'web.backups.restore.pickFileToast' => 'Elige primero un archivo de paquete',
			'web.backups.restore.succeededToast' => 'Restauración correcta',
			'web.backups.restore.replayedDescription' => ({required Object bytes, required Object id}) => '${bytes} reproducidos desde el manifest ${id}',
			'web.backups.restore.failedToast' => 'Error en la restauración',
			'web.backups.restore.restoring' => 'Restaurando…',
			'web.backups.restore.dryRunToast' => 'Simulación completa: revisa el plan y luego aplícalo',
			'web.backups.restore.planTitle' => 'Plan de restauración (simulación: nada cambió)',
			'web.backups.restore.planDump' => ({required Object size}) => 'Volcado de base de datos: ${size}',
			'web.backups.restore.planConfig' => ({required Object path}) => 'config.toml → ${path}',
			'web.backups.restore.planSecrets' => ({required Object path}) => 'secrets.env → ${path}',
			'web.backups.restore.planVault' => ({required Object files, required Object roots}) => 'vault: ${files} archivos (${roots})',
			'web.backups.restore.planApplyHint' => 'Aplicar toma primero una instantánea de seguridad de instancia completa, luego sobrescribe lo anterior y ejecuta pg_restore.',
			'web.backups.restore.preview' => 'Previsualizar (simulación)',
			'web.backups.restore.previewing' => 'Previsualizando…',
			'web.backups.restore.previewFirstHint' => 'Ejecuta primero una simulación',
			'web.backups.restore.applyRestore' => 'Aplicar restauración',
			'web.backups.kind.dbOnly' => 'Solo BD',
			'web.backups.kind.fullInstance' => 'Instancia completa',
			'web.backups.kind.fullInstanceHint' => 'Incluye el vault, secrets.env y config.toml',
			'web.backups.verify.ok' => 'verificada',
			'web.backups.verify.okHint' => 'Descifrada y confirmada como restaurable (pg_restore --list)',
			'web.backups.verify.failed' => 'sin verificar',
			'web.backups.health.headlineHealthy' => 'Copias correctas',
			'web.backups.health.headlineAttention' => 'Requiere atención',
			'web.backups.health.headlineNever' => 'Aún sin copias',
			'web.backups.health.lastSuccess' => 'Última copia correcta',
			'web.backups.health.never' => 'nunca',
			'web.backups.health.tiles.recentFailures' => 'Fallos recientes',
			'web.backups.health.tiles.verifyFailures' => 'Verificación fallida',
			'web.backups.health.tiles.overdue' => 'Atrasadas',
			'web.backups.health.tiles.schedules' => 'Programaciones',
			'web.backups.health.loadFailedToast' => 'No se pudo cargar el estado de las copias',
			'web.backups.trigger.preMigrate' => 'pre-migración',
			'web.backups.trigger.preMigrateHint' => 'Instantánea automática tomada antes de ejecutar las migraciones de esquema',
			'web.backups.trigger.preRestore' => 'pre-restauración',
			'web.backups.trigger.preRestoreHint' => 'Instantánea de seguridad automática tomada antes de aplicar una restauración',
			'web.backups.recoveryKit.button' => 'Kit de recuperación',
			'web.backups.recoveryKit.title' => 'Descargar kit de recuperación',
			'web.backups.recoveryKit.warning' => 'La frase de la copia de seguridad nunca se guarda en una copia. Este kit es esa frase envuelta con una frase de recuperación que tú eliges. Guarda TANTO el archivo COMO la frase de recuperación en un lugar seguro y separado: sin ellos, perder el host significa copias irrecuperables.',
			'web.backups.recoveryKit.passphraseLabel' => 'Frase de recuperación (mín. 8 caracteres)',
			'web.backups.recoveryKit.passphrasePlaceholder' => 'una frase fuerte que no perderás',
			'web.backups.recoveryKit.confirmLabel' => 'Confirmar frase de recuperación',
			'web.backups.recoveryKit.mismatch' => 'Las frases no coinciden',
			'web.backups.recoveryKit.generating' => 'Generando…',
			'web.backups.recoveryKit.download' => 'Descargar kit',
			'web.backups.recoveryKit.downloadedToast' => 'Kit de recuperación descargado: guárdalo de forma segura',
			'web.backups.recoveryKit.failedToast' => 'No se pudo generar el kit de recuperación',
			'web.backups.schedulesTab.description' => 'Copias de seguridad periódicas. El programador consulta cada 30 s y ejecuta la programación pendiente más antigua.',
			'web.backups.schedulesTab.newSchedule' => 'Nueva programación',
			'web.backups.schedulesTab.loadFailedToast' => 'No se pudieron cargar las programaciones',
			'web.backups.schedulesTab.deleteConfirm' => ({required Object id}) => '¿Eliminar la programación ${id}?',
			'web.backups.schedulesTab.deletedToast' => 'Programación eliminada',
			'web.backups.schedulesTab.deleteFailedToast' => 'Error al eliminar',
			'web.backups.schedulesTab.toggleFailedToast' => 'Error al alternar',
			'web.backups.schedulesTab.empty' => 'No hay programaciones. Añade una para hacer copias de seguridad periódicas automáticas.',
			'web.backups.schedulesTab.columns.id' => 'ID',
			'web.backups.schedulesTab.columns.target' => 'Destino',
			'web.backups.schedulesTab.columns.interval' => 'Intervalo',
			'web.backups.schedulesTab.columns.keep' => 'Conservar',
			'web.backups.schedulesTab.columns.nextRun' => 'Próxima ejecución',
			'web.backups.schedulesTab.columns.enabled' => 'Habilitada',
			'web.backups.schedulesTab.columns.actions' => 'Acciones',
			'web.backups.schedulesTab.keepCount' => ({required Object count}) => '${count} copias de seguridad',
			'web.backups.schedulesTab.deleteTooltip' => 'Eliminar',
			'web.backups.newSchedule.title' => 'Nueva programación de copia de seguridad',
			'web.backups.newSchedule.targetLabel' => 'Destinos',
			'web.backups.newSchedule.targetsHint' => 'Elige uno o más: la misma copia se escribe en cada destino (3-2-1).',
			'web.backups.newSchedule.everyHoursLabel' => 'Cada (horas)',
			'web.backups.newSchedule.keepLastNLabel' => 'Conservar las últimas N',
			'web.backups.newSchedule.enableImmediately' => 'Habilitar inmediatamente',
			'web.backups.newSchedule.createdToast' => 'Programación creada',
			'web.backups.newSchedule.createFailedToast' => 'Error al crear',
			'web.backups.newSchedule.creating' => 'Creando…',
			'web.backups.newSchedule.create' => 'Crear',
			'web.backups.fanout.badge' => 'difusión',
			'web.backups.fanout.hint' => ({required Object group}) => 'Parte de una difusión a varios destinos (grupo ${group})',
			'web.backups.dedup.badge' => 'deduplicada',
			'web.backups.dedup.hint' => 'Idéntica a una copia anterior: reutilizó el blob existente en lugar de subir una copia',
			'web.backups.targetsTab.description' => 'Destinos de almacenamiento. v1 admite <1>local</1> (disco en el host de opendray) y <3>smb</3> (cualquier recurso compartido SMB / CIFS, p. ej. UNAS o Synology).',
			'web.backups.targetsTab.newTarget' => 'Nuevo destino',
			'web.backups.targetsTab.listFailedToast' => 'No se pudieron listar los destinos',
			'web.backups.targetsTab.deleteConfirm' => ({required Object id}) => '¿Eliminar el destino ${id}? Las programaciones que lo referencien bloquearán la eliminación.',
			'web.backups.targetsTab.deletedToast' => 'Destino eliminado',
			'web.backups.targetsTab.deleteFailedToast' => 'Error al eliminar',
			'web.backups.targetsTab.connectionOkToast' => 'Conexión correcta',
			'web.backups.targetsTab.connectionFailedToast' => 'Error de conexión',
			'web.backups.targetsTab.testFailedToast' => 'Error en la prueba',
			'web.backups.targetsTab.columns.id' => 'ID',
			'web.backups.targetsTab.columns.kind' => 'Tipo',
			'web.backups.targetsTab.columns.config' => 'Config',
			'web.backups.targetsTab.columns.enabled' => 'Habilitado',
			'web.backups.targetsTab.columns.actions' => 'Acciones',
			'web.backups.targetsTab.on' => 'activado',
			'web.backups.targetsTab.off' => 'desactivado',
			'web.backups.targetsTab.test' => 'Probar',
			'web.backups.targetsTab.testing' => 'Probando…',
			'web.backups.targetsTab.deleteTooltip' => 'Eliminar',
			'web.backups.targetEditor.title' => 'Nuevo destino de copia de seguridad',
			'web.backups.targetEditor.kindPicker' => '¿Dónde quieres hacer la copia de seguridad?',
			'web.backups.targetEditor.idLabel' => 'ID (opcional)',
			'web.backups.targetEditor.idPlaceholder' => 'se genera automáticamente si se deja en blanco, p. ej. tgt_xxx',
			'web.backups.targetEditor.createdToast' => 'Destino creado',
			'web.backups.targetEditor.createFailedToast' => 'Error al crear',
			'web.backups.targetEditor.creating' => 'Creando…',
			'web.backups.targetEditor.create' => 'Crear destino',
			'web.backups.targetEditor.enableImmediately' => 'Habilitar inmediatamente (de lo contrario se guarda como deshabilitado, útil para "configurar ahora, activar más tarde")',
			'web.backups.targetEditor.local.rootLabel' => 'Directorio raíz',
			'web.backups.targetEditor.local.rootHint' => 'Vacío = cfg.backup.local_dir (~/.opendray/backups)',
			'web.backups.targetEditor.local.rootPlaceholder' => '~/backups/opendray  o  /mnt/external-hdd/opendray',
			'web.backups.targetEditor.smb.hostLabel' => 'Host',
			'web.backups.targetEditor.smb.hostPlaceholder' => '192.168.1.20',
			'web.backups.targetEditor.smb.portLabel' => 'Puerto',
			'web.backups.targetEditor.smb.shareLabel' => 'Recurso compartido',
			'web.backups.targetEditor.smb.shareHint' => 'Nombre del recurso compartido de nivel superior en el servidor SMB',
			'web.backups.targetEditor.smb.sharePlaceholder' => 'Claude_Workspace',
			'web.backups.targetEditor.smb.userLabel' => 'Usuario',
			'web.backups.targetEditor.smb.passwordLabel' => 'Contraseña',
			'web.backups.targetEditor.smb.pathPrefixLabel' => 'Prefijo de ruta',
			'web.backups.targetEditor.smb.pathPrefixHint' => 'Subcarpeta bajo la raíz del recurso compartido (opcional)',
			'web.backups.targetEditor.smb.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.s3.endpointLabel' => 'Endpoint',
			'web.backups.targetEditor.s3.endpointHint' => 'Host (sin protocolo). AWS: s3.amazonaws.com · R2: <accountid>.r2.cloudflarestorage.com · MinIO: minio.local:9000',
			'web.backups.targetEditor.s3.endpointPlaceholder' => 's3.amazonaws.com',
			'web.backups.targetEditor.s3.regionLabel' => 'Región',
			'web.backups.targetEditor.s3.regionHint' => 'Solo AWS; en R2 usa \'auto\'',
			'web.backups.targetEditor.s3.regionPlaceholder' => 'us-east-1 / auto',
			_ => null,
		} ?? switch (path) {
			'web.backups.targetEditor.s3.bucketLabel' => 'Bucket',
			'web.backups.targetEditor.s3.bucketPlaceholder' => 'opendray-backups',
			'web.backups.targetEditor.s3.accessKeyLabel' => 'Clave de acceso',
			'web.backups.targetEditor.s3.secretKeyLabel' => 'Clave secreta',
			'web.backups.targetEditor.s3.secretKeyHint' => 'Se almacena cifrada con AES-256-GCM; nunca se devuelve',
			'web.backups.targetEditor.s3.pathPrefixLabel' => 'Prefijo de ruta',
			'web.backups.targetEditor.s3.pathPrefixHint' => 'Prefijo de clave de objeto (opcional)',
			'web.backups.targetEditor.s3.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.s3.useHttps' => 'Usar HTTPS',
			'web.backups.targetEditor.s3.pathStyle' => 'Direccionamiento de tipo ruta (heredado / MinIO)',
			'web.backups.targetEditor.webdav.baseUrlLabel' => 'URL base',
			'web.backups.targetEditor.webdav.baseUrlHint' => 'URL completa incluyendo cualquier ruta. Ejemplos: https://cloud.example.com/remote.php/dav/files/me/ (Nextcloud), https://nas.local:5006/ (Synology), https://dav.jianguoyun.com/dav/ (Jianguoyun / 坚果云)',
			'web.backups.targetEditor.webdav.baseUrlPlaceholder' => 'https://cloud.example.com/remote.php/dav/files/<user>/',
			'web.backups.targetEditor.webdav.userLabel' => 'Usuario',
			'web.backups.targetEditor.webdav.passwordLabel' => 'Contraseña',
			'web.backups.targetEditor.webdav.pathPrefixLabel' => 'Prefijo de ruta',
			'web.backups.targetEditor.webdav.pathPrefixHint' => 'Subcarpeta bajo la URL base (opcional)',
			'web.backups.targetEditor.webdav.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.sftp.hostLabel' => 'Host',
			'web.backups.targetEditor.sftp.hostPlaceholder' => 'vps.example.com',
			'web.backups.targetEditor.sftp.portLabel' => 'Puerto',
			'web.backups.targetEditor.sftp.userLabel' => 'Usuario',
			'web.backups.targetEditor.sftp.passwordLabel' => 'Contraseña',
			'web.backups.targetEditor.sftp.passwordHint' => 'Se requiere contraseña O clave privada. Si se indican ambas, la contraseña se trata como la frase de contraseña de la clave.',
			'web.backups.targetEditor.sftp.privateKeyLabel' => 'Clave privada (PEM)',
			'web.backups.targetEditor.sftp.privateKeyHint' => 'Pega el contenido de una clave privada OpenSSH/PEM (p. ej. ~/.ssh/id_ed25519). Déjalo en blanco para autenticación solo con contraseña.',
			'web.backups.targetEditor.sftp.privateKeyPlaceholder' => '-----BEGIN OPENSSH PRIVATE KEY-----...',
			'web.backups.targetEditor.sftp.hostKeyLabel' => 'Clave de host (fijación)',
			'web.backups.targetEditor.sftp.hostKeyHint' => 'Clave pública del servidor en formato OpenSSH (ejecuta `ssh-keyscan host` para obtenerla). Déjalo en blanco para desactivar la fijación (NO recomendado fuera de la LAN).',
			'web.backups.targetEditor.sftp.hostKeyPlaceholder' => 'ssh-ed25519 AAAA...',
			'web.backups.targetEditor.sftp.pathPrefixLabel' => 'Prefijo de ruta',
			'web.backups.targetEditor.sftp.pathPrefixHint' => 'Absoluta o relativa al directorio personal del usuario (opcional)',
			'web.backups.targetEditor.sftp.pathPrefixPlaceholder' => '/var/backups/opendray  o  opendray-backups',
			'web.backups.targetEditor.rclone.rcloneHint' => 'Requiere tener instalada la CLI de <1>rclone</1> en el host de opendray. Primero configura tu remoto con <3>rclone config</3>, luego usa el nombre del remoto de abajo. opendray invoca <5>rclone rcat / cat / lsd</5> internamente.',
			'web.backups.targetEditor.rclone.remoteLabel' => 'Nombre del remoto',
			'web.backups.targetEditor.rclone.remoteHint' => 'Nombre de `rclone config` (sin dos puntos). Ejemplo: gdrive, onedrive, dropbox-personal, baidu-pan',
			'web.backups.targetEditor.rclone.remotePlaceholder' => 'gdrive',
			'web.backups.targetEditor.rclone.pathPrefixLabel' => 'Prefijo de ruta',
			'web.backups.targetEditor.rclone.pathPrefixHint' => 'Subcarpeta bajo la raíz del remoto (opcional)',
			'web.backups.targetEditor.rclone.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.rclone.binaryPathLabel' => 'Ruta del binario',
			'web.backups.targetEditor.rclone.binaryPathHint' => 'Anula `which rclone`. Si está vacío, usa la búsqueda en PATH.',
			'web.backups.targetEditor.rclone.binaryPathPlaceholder' => '/opt/homebrew/bin/rclone',
			'web.backups.targetEditor.rclone.configPathLabel' => 'Ruta de configuración',
			'web.backups.targetEditor.rclone.configPathHint' => 'Anula --config (por defecto ~/.config/rclone/rclone.conf o ~/.rclone.conf)',
			'web.backups.targetEditor.rclone.configPathPlaceholder' => 'déjalo en blanco para el valor por defecto de rclone',
			'web.serverSettings.sections.general.title' => 'General',
			'web.serverSettings.sections.general.desc' => 'Dirección de escucha, cuenta de operador, TTL del token.',
			'web.serverSettings.sections.logging.title' => 'Registro',
			'web.serverSettings.sections.logging.desc' => 'Verbosidad, formato y seguimiento en vivo.',
			'web.serverSettings.sections.sessions.title' => 'Sesiones',
			'web.serverSettings.sections.sessions.desc' => 'Umbrales de detección de inactividad.',
			'web.serverSettings.sections.vault.title' => 'Vault',
			'web.serverSettings.sections.vault.desc' => 'Notas, skills y raíz versionada con git.',
			'web.serverSettings.sections.mcp.title' => 'Registro de MCP',
			'web.serverSettings.sections.mcp.desc' => 'Registro de servidores + secretos.',
			'web.serverSettings.sections.memory.title' => 'Memoria · almacenamiento y embedder',
			'web.serverSettings.sections.memory.desc' => 'La mitad de infraestructura del subsistema de memoria: backend de embeddings, ajuste de recuperación y gobernanza de fondo. Reinicia para aplicar. El comportamiento en runtime (workers, captura, inyección) vive en los ajustes de Cortex.',
			'web.serverSettings.sections.backup.title' => 'Backup',
			'web.serverSettings.sections.backup.desc' => 'Copias de seguridad cifradas de la DB, restauración y exportaciones de datos de administración.',
			'web.serverSettings.sections.claude.title' => 'Almacenamiento · Claude',
			'web.serverSettings.sections.claude.desc' => 'Dónde se guardan los transcripts de Claude en disco.',
			'web.serverSettings.sections.codex.title' => 'Almacenamiento · Codex',
			'web.serverSettings.sections.codex.desc' => 'Raíz de sesiones de Codex.',
			'web.serverSettings.sections.gemini.title' => 'Almacenamiento · Gemini',
			'web.serverSettings.sections.gemini.desc' => 'Carpeta tmp por proyecto de Gemini + projects.json.',
			'web.serverSettings.sections.antigravity.title' => 'Almacenamiento · Antigravity',
			'web.serverSettings.sections.antigravity.desc' => 'Almacén SQLite por conversación de Antigravity (agy).',
			'web.serverSettings.loading' => 'Cargando ajustes del servidor…',
			'web.serverSettings.loadFailed' => ({required Object message}) => 'Error al cargar: ${message}',
			'web.serverSettings.noConfigFlag' => 'opendray se inició sin la opción -config. Los ajustes se cargan únicamente desde variables de entorno y no pueden editarse aquí.',
			'web.serverSettings.resetButton' => 'Restablecer',
			'web.serverSettings.resetButtonTitle' => 'Descartar los cambios sin guardar de esta sección',
			'web.serverSettings.resetConfirm' => ({required Object section}) => '¿Restablecer "${section}" a los últimos valores guardados?',
			'web.serverSettings.badgeRestartRequired' => 'requiere reinicio',
			'web.serverSettings.badgeUnsaved' => 'sin guardar',
			'web.serverSettings.saveButton' => 'Guardar cambios',
			'web.serverSettings.saveToastTitle' => 'Ajustes guardados',
			'web.serverSettings.saveToastDesc' => 'Haz clic en Reiniciar para aplicarlos.',
			'web.serverSettings.saveErrorTitle' => 'Error al guardar',
			'web.serverSettings.dangerousConfirm' => 'Has cambiado la dirección de escucha, el usuario de administración o la contraseña de administración. Tras reiniciar puede que necesites volver a autenticarte o usar la nueva dirección. ¿Continuar?',
			'web.serverSettings.unsavedHint' => 'Tienes cambios sin guardar',
			'web.serverSettings.savedHint' => 'Todos los cambios guardados',
			'web.serverSettings.searchPlaceholder' => 'Filtrar campos…',
			'web.serverSettings.restart.button' => 'Reiniciar servidor',
			'web.serverSettings.restart.buttonTitle' => 'Auto-ejecutar el proceso del gateway',
			'web.serverSettings.restart.dirtyConfirm' => 'Tienes cambios sin guardar. El reinicio usará la ÚLTIMA configuración GUARDADA. ¿Continuar?',
			'web.serverSettings.restart.confirm' => '¿Reiniciar el gateway de opendray? Todas las sesiones de terminal abiertas se reconectarán automáticamente.',
			'web.serverSettings.restart.overlay' => 'Reiniciando servidor…',
			'web.serverSettings.restart.waiting' => ({required Object tick}) => 'Esperando a /health · ${tick}s',
			'web.serverSettings.restart.timedOutTitle' => 'Se agotó el tiempo del reinicio',
			'web.serverSettings.restart.timedOutDesc' => 'El endpoint de salud nunca respondió. Revisa los logs del servidor.',
			'web.serverSettings.restart.successToast' => 'Servidor reiniciado',
			'web.serverSettings.formGroups.network' => 'Red',
			'web.serverSettings.formGroups.operatorAccount' => 'Cuenta de operador',
			'web.serverSettings.formGroups.memoryConfiguration' => 'Configuración',
			'web.serverSettings.formGroups.memoryHttp' => 'Backend HTTP (se usa cuando backend=http)',
			'web.serverSettings.formGroups.memoryLocal' => 'ONNX local (se usa cuando backend=local)',
			'web.serverSettings.formGroups.backupStatus' => 'Estado',
			'web.serverSettings.formGroups.backupWhere' => 'Dónde van las copias de seguridad',
			'web.serverSettings.formGroups.backupSchedules' => 'Programaciones',
			'web.serverSettings.formGroups.backupWhatsInside' => '¿Qué contiene una copia de seguridad?',
			'web.serverSettings.formGroups.memoryGovernance' => 'Gobernanza de fondo (gatekeeper / cleaner)',
			'web.serverSettings.formGroups.knowledgeGraph' => 'Grafo de conocimiento',
			'web.serverSettings.formGroups.database' => 'Base de datos',
			'web.serverSettings.fields.listenAddress.label' => 'Dirección de escucha',
			'web.serverSettings.fields.listenAddress.hint' => 'El host:port al que se vincula el servidor HTTP. Ejemplo: 0.0.0.0:8770.',
			'web.serverSettings.fields.username.label' => 'Nombre de usuario',
			'web.serverSettings.fields.username.hint' => 'Nombre de inicio de sesión usado en el formulario de acceso. Cambiarlo obliga a volver a iniciar sesión en la siguiente solicitud.',
			'web.serverSettings.fields.password.label' => 'Contraseña',
			'web.serverSettings.fields.password.hint' => 'Déjalo en blanco para mantener la contraseña actual. Enviar un valor la sobrescribe.',
			'web.serverSettings.fields.password.hideTitle' => 'Ocultar',
			'web.serverSettings.fields.password.revealTitle' => 'Mostrar',
			'web.serverSettings.fields.tokenTTL.label' => 'TTL del token',
			'web.serverSettings.fields.tokenTTL.hint' => 'Tiempo de vida del bearer-token como duración de Go, p. ej. "24h", "30m". Vacío = nunca expira.',
			'web.serverSettings.fields.logLevel.label' => 'Nivel de log',
			'web.serverSettings.fields.logLevel.hint' => 'Las líneas por debajo de este nivel se descartan.',
			'web.serverSettings.fields.logFormat.label' => 'Formato',
			'web.serverSettings.fields.logFormat.hint' => '"text" es legible para humanos; "json" es analizable por máquinas.',
			'web.serverSettings.fields.logFile.label' => 'Archivo de log',
			'web.serverSettings.fields.logFile.hint' => 'Ruta de archivo opcional. Rota automáticamente a los 10 MB, conserva 5 copias. Vacío = solo stderr.',
			'web.serverSettings.fields.idleThreshold.label' => 'Umbral de inactividad',
			'web.serverSettings.fields.idleThreshold.hint' => 'Una session permanece en silencio este tiempo antes de que se dispare session.idle. Vacío = 30s.',
			'web.serverSettings.fields.idlePollInterval.label' => 'Intervalo de sondeo de inactividad',
			'web.serverSettings.fields.idlePollInterval.hint' => 'Con qué frecuencia se activa el detector de inactividad. Más bajo = menor latencia, más activaciones. Vacío = 5s.',
			'web.serverSettings.fields.vaultRoot.label' => 'Raíz del Vault',
			'web.serverSettings.fields.vaultRoot.hint' => 'Directorio de nivel superior para notas, skills y el registro de MCP.',
			'web.serverSettings.fields.notesDirectory.label' => 'Directorio de notas',
			'web.serverSettings.fields.notesDirectory.hint' => 'Anula la ubicación de las notas. Por defecto <vault root>/notes.',
			'web.serverSettings.fields.skillsDirectory.label' => 'Directorio de skills',
			'web.serverSettings.fields.skillsDirectory.hint' => 'Anula la ubicación de los skills. Por defecto <vault root>/skills.',
			'web.serverSettings.fields.gitRoot.label' => 'Raíz de git',
			'web.serverSettings.fields.gitRoot.hint' => 'Árbol de trabajo al que confirma la función Vault Sync.',
			'web.serverSettings.fields.personalPrefix.label' => 'Prefijo personal',
			'web.serverSettings.fields.personalPrefix.hint' => 'Nombre de carpeta usado para las notas personales al derivar rutas automáticamente. Por defecto "personal".',
			'web.serverSettings.fields.projectsPrefix.label' => 'Prefijo de proyectos',
			'web.serverSettings.fields.projectsPrefix.hint' => 'Nombre de carpeta usado para las notas de proyecto. Por defecto "projects".',
			'web.serverSettings.fields.registryRoot.label' => 'Raíz del registro',
			'web.serverSettings.fields.registryRoot.hint' => 'Directorio que contiene las definiciones JSON de los servidores MCP. Por defecto <vault>/mcp.',
			'web.serverSettings.fields.secretsFile.label' => 'Archivo de secretos',
			'web.serverSettings.fields.secretsFile.hint' => 'Archivo key=value que se sustituye en los comandos del servidor MCP en el momento del arranque.',
			'web.serverSettings.fields.memoryBackend.label' => 'Backend del embedder',
			'web.serverSettings.fields.memoryBackend.hint' => '"auto" / "bm25" usan la ruta de palabras clave en Go puro, sin cgo. "http" llama a cualquier /v1/embeddings compatible con OpenAI (ollama / OpenAI / LocalAI). "local" ejecuta un sentence-transformer ONNX en el proceso, requiere un binario compilado con `-tags local_onnx`.',
			'web.serverSettings.fields.memoryStore.label' => 'Almacén',
			'web.serverSettings.fields.memoryStore.hint' => '"pgvector" reutiliza la PG existente de opendray con la extensión vector; es la única opción en la v1.',
			'web.serverSettings.fields.memoryTopK.label' => 'Top-K por defecto',
			'web.serverSettings.fields.memoryTopK.hint' => 'Cuántos resultados devuelve memory_search cuando el agente no lo especifica. Vacío = 5.',
			'web.serverSettings.fields.memoryThreshold.label' => 'Umbral de similitud',
			'web.serverSettings.fields.memoryThreshold.hint' => 'Los resultados por debajo de esta puntuación se descartan. Vacío = 0.1 (permisivo, los vectores dispersos de BM25 rara vez superan 0.5).',
			'web.serverSettings.fields.memoryScope.label' => 'Ámbito por defecto',
			'web.serverSettings.fields.memoryScope.hint' => 'Lo que usa memory_store cuando el agente no lo especifica. "project" (recomendado) agrupa por cwd; "global" comparte entre cwds.',
			'web.serverSettings.fields.memoryBaseUrl.label' => 'URL base',
			'web.serverSettings.fields.memoryBaseUrl.hint' => 'p. ej. "http://localhost:11434/v1" para ollama, "https://api.openai.com/v1" para OpenAI.',
			'web.serverSettings.fields.memoryModel.label' => 'Modelo',
			'web.serverSettings.fields.memoryModel.hint' => 'p. ej. "nomic-embed-text" para ollama, "text-embedding-3-small" para OpenAI.',
			'web.serverSettings.fields.memoryApiKey.label' => 'API key',
			'web.serverSettings.fields.memoryApiKey.hint' => 'Vacío para ollama / servidores locales. Obligatorio para OpenAI / Voyage / servicios alojados.',
			'web.serverSettings.fields.memoryLocalModel.label' => 'Nombre del modelo',
			'web.serverSettings.fields.memoryLocalModel.hint' => 'Cosmético, aparece en los logs / el Inspector. p. ej. "bge-m3", "bge-small-en-v1.5".',
			'web.serverSettings.fields.memoryLibraryPath.label' => 'Ruta de la biblioteca',
			'web.serverSettings.fields.memoryLibraryPath.hint' => 'Directorio que contiene libonnxruntime.dylib (macOS) / libonnxruntime.so (Linux). Tras `brew install onnxruntime`, es /opt/homebrew/opt/onnxruntime/lib.',
			'web.serverSettings.fields.memoryModelPath.label' => 'Ruta del modelo',
			'web.serverSettings.fields.memoryModelPath.hint' => 'Ruta absoluta a los pesos .onnx. Descárgalos de HuggingFace, p. ej. Xenova/bge-m3 o Xenova/bge-small-en-v1.5.',
			'web.serverSettings.fields.memoryTokenizerPath.label' => 'Ruta del tokenizador',
			'web.serverSettings.fields.memoryTokenizerPath.hint' => 'Ruta absoluta a tokenizer.json (formato estándar de HuggingFace), normalmente justo al lado del modelo.',
			'web.serverSettings.fields.memoryMaxSeqLen.label' => 'Longitud máxima de secuencia',
			'web.serverSettings.fields.memoryMaxSeqLen.hint' => 'Los tokens que superan este límite se truncan. El valor por defecto de bge-m3 es 512. Vacío = 512.',
			'web.serverSettings.fields.claudeHistoryRoots.label' => 'Raíces de historial',
			'web.serverSettings.fields.claudeHistoryRoots.hint' => 'Directorios que se escanean en busca de los transcripts JSONL por proyecto de Claude. Vacío = escanear ~/.claude/projects + cada ~/.claude-accounts/*/projects.',
			'web.serverSettings.fields.claudeAccountsDir.label' => 'Directorio de cuentas',
			'web.serverSettings.fields.claudeAccountsDir.hint' => 'Raíz usada para los ConfigDirs de las cuentas de Claude gestionadas por opendray. Por defecto ~/.claude-accounts.',
			'web.serverSettings.fields.codexSessionsRoot.label' => 'Raíz de sesiones',
			'web.serverSettings.fields.codexSessionsRoot.hint' => 'Directorio que se recorre en busca de los archivos JSONL de rollout de Codex. Por defecto ~/.codex/sessions.',
			'web.serverSettings.fields.geminiTmpRoot.label' => 'Directorio tmp',
			'web.serverSettings.fields.geminiTmpRoot.hint' => 'Raíz que contiene las carpetas tmp por proyecto de Gemini. Por defecto ~/.gemini/tmp.',
			'web.serverSettings.fields.geminiProjectsFile.label' => 'projects.json',
			'web.serverSettings.fields.geminiProjectsFile.hint' => 'Ruta al archivo de mapeo cwd→nombre-corto de Gemini. Por defecto ~/.gemini/projects.json.',
			'web.serverSettings.fields.antigravityConversationsRoot.label' => 'Directorio de conversaciones',
			'web.serverSettings.fields.antigravityConversationsRoot.hint' => 'Raíz con los archivos .db por conversación de agy. Por defecto ~/.gemini/antigravity-cli/conversations.',
			'web.serverSettings.fields.backupLocalDir.label' => 'Directorio local de copias de seguridad',
			'web.serverSettings.fields.backupLocalDir.hint' => 'Raíz por defecto para el destino `local` creado automáticamente. Vacío = ~/.opendray/backups. Requiere reinicio.',
			'web.serverSettings.fields.backupExportDir.label' => 'Directorio de exportación',
			'web.serverSettings.fields.backupExportDir.hint' => 'Dónde se preparan en disco los zips de exportación puntual. Vacío = ~/.opendray/exports. Los paquetes expiran automáticamente tras 24h. Requiere reinicio.',
			'web.serverSettings.fields.backupPgDumpPath.label' => 'Ruta de pg_dump',
			'web.serverSettings.fields.backupPgDumpPath.hint' => 'Ruta absoluta a pg_dump. La versión mayor debe ser ≥ la del servidor. Vacío = el primer pg_dump en el PATH.',
			'web.serverSettings.fields.backupPgRestorePath.label' => 'Ruta de pg_restore',
			'web.serverSettings.fields.backupPgRestorePath.hint' => 'Ruta absoluta a pg_restore para el flujo /backups/restore. Misma regla de versión mayor.',
			'web.serverSettings.fields.memoryDedup.label' => 'Umbral de dedup',
			'web.serverSettings.fields.memoryDedup.hint' => 'Umbral de plegado al escribir: una similitud superior actualiza la memoria existente en vez de insertar un casi-duplicado. 0 = valor por defecto relativo al embedder; negativo desactiva el plegado.',
			'web.serverSettings.fields.gatekeeperEnabled.label' => 'Gatekeeper',
			'web.serverSettings.fields.gatekeeperEnabled.hint' => 'Juez LLM pre-escritura: decide si un memory_store trae un hecho durable o ruido. Qué LLM lo ejecuta se enruta en ajustes de Cortex → Workers.',
			'web.serverSettings.fields.gatekeeperLatency.label' => 'Latencia máx. del gatekeeper (ms)',
			'web.serverSettings.fields.gatekeeperLatency.hint' => 'Por encima de esto el gatekeeper degrada a "permitir" en vez de bloquear la escritura por un LLM lento. Por defecto 2000.',
			'web.serverSettings.fields.cleanerEnabled.label' => 'Cleaner (auto-bibliotecario)',
			'web.serverSettings.fields.cleanerEnabled.hint' => 'Barrido periódico que archiva (reversible, con periodo de gracia) memorias obsoletas o duplicadas. Qué LLM lo ejecuta se enruta en ajustes de Cortex → Workers.',
			'web.serverSettings.fields.cleanerInterval.label' => 'Intervalo del cleaner (s)',
			'web.serverSettings.fields.cleanerInterval.hint' => 'Segundos entre barridos automáticos. Por defecto 86400 (24h).',
			'web.serverSettings.fields.cleanerGlobalScope.label' => 'Cleaner barre el scope global',
			'web.serverSettings.fields.cleanerGlobalScope.hint' => 'Revisar también memorias de scope global (normalmente curadas por el operador). Por defecto off hasta que confíes en el cleaner.',
			'web.serverSettings.fields.knowledgeEnabled.label' => 'Grafo de conocimiento',
			'web.serverSettings.fields.knowledgeEnabled.hint' => 'La capa de conocimiento estructurada y auto-evolutiva (entidades / playbooks / skills) sobre la memoria episódica. Alimenta la pestaña Cortex → Knowledge.',
			'web.serverSettings.fields.claudeWatcher.label' => 'Watcher de cuentas',
			'web.serverSettings.fields.claudeWatcher.hint' => 'Vigila accounts_dir y auto-registra una cuenta nueva cuando aparece un .credentials.json (resultado de CLAUDE_CONFIG_DIR=<dir> claude login).',
			'web.serverSettings.fields.claudeAutoFailover.label' => 'Auto-failover por rate limit',
			'web.serverSettings.fields.claudeAutoFailover.hint' => 'Cambia una session viva a otra cuenta de Claude al chocar con un rate limit. Opt-in: cambia la atribución de facturación sin un clic.',
			'web.serverSettings.fields.mobileTokenTTL.label' => 'TTL del token móvil',
			'web.serverSettings.fields.mobileTokenTTL.hint' => 'Vida de los tokens emitidos a la app móvil. Por defecto 720h (30 días).',
			'web.serverSettings.fields.dbMaxConns.label' => 'Conexiones máx.',
			'web.serverSettings.fields.dbMaxConns.hint' => 'Tope del pool de conexiones pgx. 0 = valor por defecto (16).',
			'web.serverSettings.liveTail.heading' => 'Seguimiento en vivo',
			'web.serverSettings.liveTail.description' => 'Búfer circular en memoria (los últimos ~2.000 registros). Se reinicia al reiniciar.',
			'web.serverSettings.memoryInspectorCard.heading' => 'Inspector',
			'web.serverSettings.memoryInspectorCard.description' => 'Explora, busca y edita las memorias almacenadas en la página dedicada.',
			'web.serverSettings.memoryInspectorCard.openButton' => 'Abrir Memory →',
			'web.serverSettings.localOnnxBanner' => 'Requiere que el binario se compile con <1>-tags local_onnx</1>. La compilación estándar devuelve un error de stub claro cuando se selecciona este backend. Consulta el tutorial <3>Memory → ONNX local</3> para los pasos de configuración.',
			'web.serverSettings.stringList.noneDefault' => '(ninguno, usando los valores por defecto integrados)',
			'web.serverSettings.stringList.addPath' => 'Añadir ruta',
			'web.serverSettings.stringList.removeTitle' => 'Eliminar',
			'web.serverSettings.httpHelpers.autoDetected' => 'Detectado automáticamente al arrancar',
			'web.serverSettings.httpHelpers.modelCount' => ({required Object count}) => '${count} modelo(s), haz clic para usar',
			'web.serverSettings.httpHelpers.presets' => 'Preajustes:',
			'web.serverSettings.httpHelpers.testConnection' => 'Probar conexión',
			'web.serverSettings.httpHelpers.presetTip.ollama' => 'Daemon local de ollama',
			'web.serverSettings.httpHelpers.presetTip.lmStudio' => 'Servidor local de LM Studio',
			'web.serverSettings.httpHelpers.presetTip.openai' => 'Nube de OpenAI (necesita API key)',
			'web.serverSettings.probe.unreachable' => ({required Object error}) => '✗ inaccesible: ${error}',
			'web.serverSettings.probe.connectionFailed' => 'conexión fallida',
			'web.serverSettings.probe.reachable' => ({required Object detected, required Object total, required Object embedding}) => '✓ accesible ${detected}· ${total} modelo(s) en total · ${embedding} embedding',
			'web.serverSettings.probe.modelMissing' => ({required Object model}) => '⚠ El modelo configurado ${model} no está en la lista. Elige uno de los modelos de embedding de abajo o corrige el nombre.',
			'web.serverSettings.probe.embeddingModelsLabel' => 'modelos de embedding:',
			'web.serverSettings.probe.moreModels' => ({required Object count}) => '+${count} más',
			'web.serverSettings.probe.noEmbeddingFound' => '⚠ Ningún nombre de modelo contiene "embed". Puede que el endpoint no tenga cargado un modelo de embedding, revisa tu servidor local.',
			'web.serverSettings.probe.configuredTitle' => 'Configurado actualmente',
			'web.serverSettings.probe.applyTitle' => 'Haz clic para aplicar',
			'web.serverSettings.backup.featureDisabledTitle' => 'Función deshabilitada',
			'web.serverSettings.backup.featureDisabledHint' => 'Define <1>OPENDRAY_BACKUP_ENABLED=1</1> + <3>OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;</3> en el entorno de opendray y luego reinicia. La passphrase maestra es solo de entorno, nunca toca config.toml.',
			'web.serverSettings.backup.statusRowLabel' => 'Estado',
			'web.serverSettings.backup.enabledHealthy' => 'habilitado · correcto',
			'web.serverSettings.backup.enabledDegraded' => 'habilitado · degradado',
			'web.serverSettings.backup.keyFingerprintLabel' => 'Huella de la clave',
			'web.serverSettings.backup.keyFingerprintHint' => 'guárdala en Vaultwarden, perderla bloquea todas las copias de seguridad anteriores',
			'web.serverSettings.backup.pgDumpLabel' => 'pg_dump',
			'web.serverSettings.backup.pgDumpUnavailable' => 'no disponible',
			'web.serverSettings.backup.pgRestoreLabel' => 'pg_restore',
			'web.serverSettings.backup.pgRestoreNotResolved' => '(no resuelto)',
			'web.serverSettings.backup.openBackups' => 'Abrir la página de Backups →',
			'web.serverSettings.backup.openExport' => 'Abrir Exportar / Importar →',
			'web.serverSettings.backup.whereDesc' => 'Cada destino es un lugar donde puede escribirse un blob de copia de seguridad. opendray admite <1>disco local</1>, <3>SMB/CIFS</3> (Windows / NAS), <5>compatible con S3</5> (AWS, R2, B2, MinIO, Alibaba Cloud OSS, Tencent Cloud COS, ...), <7>WebDAV</7> (Nextcloud, Synology, Jianguoyun), <9>SFTP</9>, además de un puente <11>rclone</11> que conecta con más de 70 backends adicionales (Google Drive, OneDrive, Dropbox, Baidu Pan, Aliyun Drive, ...).',
			'web.serverSettings.backup.loading' => 'Cargando…',
			'web.serverSettings.backup.noTargets' => 'Aún no hay destinos. Añade uno para empezar a hacer copias de seguridad.',
			'web.serverSettings.backup.addTarget' => 'Añadir destino',
			'web.serverSettings.backup.noSchedulesHint' => 'No hay programaciones recurrentes. Añade una en <1>/backups → Programaciones</1> para hacer copias de seguridad automáticamente.',
			'web.serverSettings.backup.scheduleHeaders.schedule' => 'Programación',
			'web.serverSettings.backup.scheduleHeaders.target' => 'Destino',
			'web.serverSettings.backup.scheduleHeaders.cadence' => 'Cadencia',
			'web.serverSettings.backup.scheduleHeaders.keep' => 'Conservar',
			'web.serverSettings.backup.scheduleHeaders.state' => 'Estado',
			'web.serverSettings.backup.every' => ({required Object interval}) => 'cada ${interval}',
			'web.serverSettings.backup.backupsKeep' => ({required Object count}) => '${count} copias de seguridad',
			'web.serverSettings.backup.stateEnabled' => 'habilitado',
			'web.serverSettings.backup.statePaused' => 'pausado',
			'web.serverSettings.backup.manageSchedules' => 'Gestionar en /backups → Programaciones →',
			'web.serverSettings.backup.whatsInsideDesc' => 'Cada copia de seguridad es un <1>pg_dump --format=custom</1> de cada tabla de opendray (sessions, integraciones, memorias, audit_log, etc.) más un <3>manifest.json</3> y (opcionalmente) el <5>config.toml</5> en vivo. Abre el panel "¿Qué contiene una copia de seguridad?" en la <7>página de Backups</7> para ver el inventario en vivo con el número de filas.',
			'web.serverSettings.backup.advancedToggle' => 'Avanzado (rutas y binarios cliente), requiere reinicio',
			'web.serverSettings.targetRow.on' => 'activado',
			'web.serverSettings.targetRow.off' => 'desactivado',
			'web.serverSettings.targetRow.test' => 'Probar',
			'web.serverSettings.targetRow.testing' => 'Probando…',
			'web.serverSettings.targetRow.delete' => 'Eliminar',
			'web.serverSettings.targetRow.connectionOk' => ({required Object id}) => '${id}: conexión correcta',
			'web.serverSettings.targetRow.connectionFailedTitle' => 'Conexión fallida',
			'web.serverSettings.targetRow.testFailedTitle' => 'Prueba fallida',
			'web.serverSettings.targetRow.deleteConfirm' => ({required Object id}) => '¿Eliminar el destino "${id}"? Las programaciones que lo referencien bloquearán la eliminación.',
			'web.serverSettings.targetRow.deleteSuccess' => 'Destino eliminado',
			'web.serverSettings.targetRow.deleteFailedTitle' => 'Error al eliminar',
			'web.serverSettings.targetRow.unknownError' => 'Error desconocido',
			'web.serverSettings.toggle.on' => 'Activado',
			'web.serverSettings.toggle.off' => 'Desactivado',
			'web.serverSettings.toggle.defaultOn' => 'Por defecto (on)',
			'web.serverSettings.toggle.defaultOff' => 'Por defecto (off)',
			'web.serverSettings.memoryRuntimeBanner' => 'El comportamiento de IA en runtime — workers, reglas de captura, perfiles de inyección y modo de spawn — vive en los ajustes de Cortex y se aplica al instante. Esta sección es la mitad de infraestructura: embedder, almacenamiento y gobernanza de fondo (requiere reinicio).',
			'web.serverSettings.memoryRuntimeBannerButton' => 'Abrir ajustes de Cortex',
			'web.settings.title' => 'Ajustes',
			'web.settings.subtitle' => 'Configuración del espacio de trabajo, la cuenta y el gateway.',
			'web.settings.groups.workspace' => 'Espacio de trabajo',
			'web.settings.groups.server' => 'Servidor',
			'web.settings.groups.system' => 'Sistema',
			'web.settings.items.appearance' => 'Apariencia',
			'web.settings.items.font' => 'Tamaño de fuente',
			'web.settings.items.account' => 'Cuenta',
			'web.settings.items.status' => 'Estado',
			'web.settings.items.about' => 'Acerca de',
			'web.settings.health.connecting' => 'conectando…',
			'web.settings.health.dbOk' => 'db ok',
			'web.settings.health.dbDown' => 'db caída',
			'web.settings.breadcrumb.server' => 'Servidor',
			'web.settings.appearance.title' => 'Apariencia',
			'web.settings.appearance.description' => 'Elige el aspecto de opendray.',
			'web.settings.appearance.options.light' => 'Claro',
			'web.settings.appearance.options.lightDesc' => 'Siempre claro',
			'web.settings.appearance.options.dark' => 'Oscuro',
			'web.settings.appearance.options.darkDesc' => 'Siempre oscuro',
			'web.settings.appearance.options.system' => 'Sistema',
			'web.settings.appearance.options.systemDesc' => 'Seguir la configuración del SO',
			'web.settings.font.title' => 'Tamaño de fuente',
			'web.settings.font.description' => 'Escala toda la interfaz. Se guarda por navegador.',
			'web.settings.font.options.compact' => 'Compacto',
			'web.settings.font.options.kDefault' => 'Predeterminado',
			'web.settings.font.options.comfy' => 'Cómodo',
			'web.settings.font.options.large' => 'Grande',
			'web.settings.account.title' => 'Cuenta',
			'web.settings.account.description' => 'Operador y token de portador actual.',
			'web.settings.account.username' => 'Nombre de usuario',
			'web.settings.account.tokenExpires' => 'El token caduca',
			'web.settings.account.changeCredentials' => 'Cambiar credenciales',
			'web.settings.changeCredentials.title' => 'Cambiar credenciales',
			'web.settings.changeCredentials.description' => 'Verifica tu contraseña actual y luego elige nuevas credenciales. Se revocarán todas las demás sesiones con sesión iniciada.',
			'web.settings.changeCredentials.currentPassword' => 'Contraseña actual',
			'web.settings.changeCredentials.newUsername' => 'Nuevo nombre de usuario',
			'web.settings.changeCredentials.newPassword' => 'Nueva contraseña',
			'web.settings.changeCredentials.newPasswordHint' => 'Al menos 8 caracteres.',
			'web.settings.changeCredentials.confirm' => 'Confirmar nueva contraseña',
			'web.settings.changeCredentials.errorTooShort' => 'La nueva contraseña debe tener al menos 8 caracteres.',
			'web.settings.changeCredentials.errorMismatch' => 'La nueva contraseña y la confirmación no coinciden.',
			'web.settings.changeCredentials.errorWrongPassword' => 'La contraseña actual es incorrecta.',
			'web.settings.changeCredentials.cancel' => 'Cancelar',
			'web.settings.changeCredentials.update' => 'Actualizar',
			'web.settings.changeCredentials.saving' => 'Guardando…',
			'web.settings.system.title' => 'Estado del sistema',
			'web.settings.system.description' => 'Estado en vivo desde el endpoint /health del gateway.',
			'web.settings.system.status' => 'Estado',
			'web.settings.system.version' => 'Versión',
			'web.settings.system.uptime' => 'Tiempo de actividad',
			'web.settings.system.database' => 'Base de datos',
			'web.settings.system.reachable' => 'accesible',
			'web.settings.system.unreachable' => 'no accesible',
			'web.settings.about.title' => 'Acerca de',
			'web.settings.about.description' => 'opendray v2: el multiplexor + gateway de integración para CLIs de agentes de IA. Código bajo Apache 2.0.',
			'web.settings.about.version' => 'Versión',
			'web.settings.about.commit' => 'Commit',
			'web.settings.about.updateAvailable' => ({required Object version}) => 'Actualización disponible: ${version}',
			'web.settings.about.releaseNotes' => 'Notas de la versión ↗',
			'web.settings.about.updateNow' => 'Actualizar ahora',
			'web.settings.about.upgradingShort' => 'Actualizando…',
			'web.settings.about.confirmRestart' => 'Esto reinicia el servicio; las sesiones en ejecución se reconectan.',
			'web.settings.about.confirmUpgrade' => 'Actualizar y reiniciar',
			'web.settings.about.upgrading' => ({required Object version}) => 'Actualizando a ${version}…',
			'web.settings.about.upgraded' => ({required Object version}) => 'Actualizado a ${version}.',
			'web.settings.about.upgradeSlow' => 'La actualización está tardando un poco. Revisa los logs del servicio si no vuelve.',
			'web.settings.about.guidedHint' => 'La actualización dentro de la app no está disponible aquí. Ejecuta en el servidor:',
			'web.settings.about.checkFailed' => 'No se pudieron comprobar las actualizaciones (sin conexión o con límite de frecuencia).',
			'web.settings.about.upToDate' => 'Tienes la última versión.',
			'web.settings.about.checkUpdates' => 'Comprobar actualizaciones',
			'web.settings.about.checking' => 'Comprobando…',
			'web.settings.about.reinstall' => 'Reinstalar',
			'web.logViewer.filterPlaceholder' => 'Filtrar…',
			'web.logViewer.debugTooltip' => 'Recuento de debug',
			'web.logViewer.infoTooltip' => 'Recuento de info',
			'web.logViewer.warnTooltip' => 'Recuento de advertencias',
			'web.logViewer.errorTooltip' => 'Recuento de errores',
			'web.logViewer.streaming' => 'Transmitiendo',
			'web.logViewer.disconnected' => 'Desconectado',
			'web.logViewer.live' => 'en directo',
			'web.logViewer.offline' => 'sin conexión',
			'web.logViewer.pauseTooltip' => 'Pausar el desplazamiento automático',
			'web.logViewer.resumeTooltip' => 'Reanudar el desplazamiento automático',
			'web.logViewer.clearTooltip' => 'Limpiar la vista local (el ring del servidor no se toca)',
			'web.logViewer.downloadTooltip' => 'Descargar el ring completo como archivo .log',
			'web.logViewer.emptyWaiting' => 'Esperando registros de log…',
			'web.logViewer.emptyFiltered' => ({required Object query}) => 'Ningún registro coincide con "${query}"',
			'web.pathInput.testButton' => 'Probar',
			'web.pathInput.testTooltip' => 'Resolver y comprobar esta ruta',
			'web.pathInput.notFound' => 'no encontrado ·',
			'web.pathInput.childrenSuffix' => 'elementos',
			'web.pathInput.expectedDirectory' => '· se esperaba un directorio',
			'web.memoryAmbient.header.title' => 'Memoria ambiental: captura e inyección automáticas',
			'web.memoryAmbient.header.body' => 'opendray sondea cada session de agente activa cada 10 segundos, extrae hechos duraderos mediante un LLM configurable y los deduplica antes de almacenarlos en el pool de memoria compartida. Configura qué LLM realiza la extracción (Proveedor), cuándo se activa la extracción (Regla de captura) y qué (si es que algo) se antepone al system prompt del agente al arrancar (Perfil de inyección).',
			'web.memoryAmbient.loading' => 'Cargando…',
			'web.memoryAmbient.providers.title' => 'Proveedores de resumen',
			'web.memoryAmbient.providers.addButton' => 'Añadir proveedor',
			'web.memoryAmbient.providers.intro' => 'Se requiere al menos un proveedor habilitado para que la captura se active realmente. Las opciones locales (Ollama, LM Studio, Integración) mantienen tus transcripts fuera de redes externas.',
			'web.memoryAmbient.providers.empty' => 'Aún no hay proveedores configurados.',
			'web.memoryAmbient.providers.row.defaultBadge' => '★ predeterminado',
			'web.memoryAmbient.providers.row.makeDefault' => 'Hacer predeterminado',
			'web.memoryAmbient.providers.row.test' => 'Probar',
			'web.memoryAmbient.providers.row.testing' => 'Probando…',
			'web.memoryAmbient.providers.row.delete' => 'Eliminar',
			'web.memoryAmbient.providers.row.testOk' => ({required Object name}) => '${name}: conexión correcta',
			'web.memoryAmbient.providers.row.testFailedToast' => 'La prueba falló',
			'web.memoryAmbient.providers.row.deleteConfirm' => ({required Object name}) => '¿Eliminar el proveedor "${name}"?',
			'web.memoryAmbient.providers.row.deletedToast' => 'Proveedor eliminado',
			'web.memoryAmbient.providers.row.deleteFailedToast' => 'La eliminación falló',
			'web.memoryAmbient.providers.row.updateFailedToast' => 'La actualización falló',
			'web.memoryAmbient.providers.row.madeDefaultToast' => ({required Object name}) => '${name} ahora es el predeterminado',
			'web.memoryAmbient.providers.dialog.title' => 'Añadir proveedor de resumen',
			'web.memoryAmbient.providers.dialog.kindLabel' => 'Tipo',
			'web.memoryAmbient.providers.dialog.nameLabel' => 'Nombre',
			'web.memoryAmbient.providers.dialog.namePlaceholder' => 'p. ej. lmstudio-qwen',
			'web.memoryAmbient.providers.dialog.modelLabel' => 'Modelo',
			'web.memoryAmbient.providers.dialog.baseUrlLabel' => 'URL base',
			'web.memoryAmbient.providers.dialog.integrationNote' => 'Los proveedores de integración resuelven su URL base a partir de una integración registrada. Configúrala primero en Integraciones; el cableado avanzado (extra_config) es solo de DB en esta versión.',
			'web.memoryAmbient.providers.dialog.apiKeyLabel' => 'Clave de API',
			'web.memoryAmbient.providers.dialog.apiKeyHint' => 'Se almacena cifrada (AES-GCM con la frase de contraseña maestra de copia de seguridad). Nunca se devuelve; tras guardar solo se muestra la huella digital.',
			'web.memoryAmbient.providers.dialog.makeDefaultLabel' => 'Hacer de este el proveedor predeterminado',
			'web.memoryAmbient.providers.dialog.create' => 'Crear',
			'web.memoryAmbient.providers.dialog.nameRequiredToast' => 'El nombre es obligatorio',
			'web.memoryAmbient.providers.dialog.createdToast' => ({required Object name}) => 'Proveedor ${name} creado',
			'web.memoryAmbient.providers.dialog.createFailedToast' => 'La creación falló',
			'web.memoryAmbient.providers.modelSelect.editTitle' => 'Cambiar modelo',
			'web.memoryAmbient.providers.modelSelect.dialogTitle' => ({required Object name}) => 'Cambiar modelo — ${name}',
			'web.memoryAmbient.providers.modelSelect.custom' => 'Personalizado…',
			'web.memoryAmbient.providers.modelSelect.backToList' => 'Elegir de la lista',
			'web.memoryAmbient.providers.modelSelect.refresh' => 'Volver a escanear los modelos del endpoint',
			'web.memoryAmbient.providers.modelSelect.unreachable' => 'Endpoint no alcanzable — escribe el nombre del modelo a mano; la lista aparece cuando el servicio esté arriba.',
			'web.memoryAmbient.providers.modelSelect.none' => 'El endpoint responde pero no anuncia modelos — carga uno en LM Studio / haz pull en Ollama y vuelve a escanear.',
			'web.memoryAmbient.providers.modelSelect.notOnEndpoint' => 'no está en el endpoint',
			'web.memoryAmbient.providers.modelSelect.save' => 'Guardar modelo',
			'web.memoryAmbient.providers.modelSelect.savedToast' => ({required Object name, required Object model}) => '${name} ahora usa ${model}',
			'web.memoryAmbient.rules.title' => 'Reglas de captura',
			'web.memoryAmbient.rules.addButton' => 'Añadir regla',
			'web.memoryAmbient.rules.intro' => 'Cada regla dice "cuando se active este disparador, resume los nuevos mensajes del transcript y almacena los hechos duraderos." Las reglas por session prevalecen sobre el valor predeterminado global. La v1 incluye 4 tipos de disparador.',
			'web.memoryAmbient.rules.empty' => 'Aún no hay reglas de captura. Añade una para habilitar la captura automática.',
			'web.memoryAmbient.rules.row.globalDefault' => 'predeterminado global',
			'web.memoryAmbient.rules.row.scopeLabel' => 'ámbito:',
			'web.memoryAmbient.rules.row.dedupLabel' => 'dedup:',
			'web.memoryAmbient.rules.row.runNow' => 'Ejecutar ahora',
			'web.memoryAmbient.rules.row.running' => 'Ejecutando…',
			'web.memoryAmbient.rules.row.delete' => 'Eliminar',
			'web.memoryAmbient.rules.row.firedToast' => ({required Object sessions}) => 'La regla se activó en ${sessions} session(es)',
			'web.memoryAmbient.rules.row.runNowFailedToast' => 'La ejecución inmediata falló',
			'web.memoryAmbient.rules.row.deleteConfirm' => ({required Object name}) => '¿Eliminar la regla "${name}"?',
			'web.memoryAmbient.rules.row.deletedToast' => 'Regla eliminada',
			'web.memoryAmbient.rules.row.deleteFailedToast' => 'La eliminación falló',
			'web.memoryAmbient.rules.row.summary.afterMessages' => ({required Object n}) => 'cada ${n} mensajes',
			'web.memoryAmbient.rules.row.summary.onIdle' => ({required Object seconds}) => 'inactivo ≥ ${seconds}s',
			'web.memoryAmbient.rules.row.summary.kChars' => ({required Object k}) => '≥ ${k} caracteres',
			'web.memoryAmbient.rules.row.summary.manual' => 'solo manual',
			'web.memoryAmbient.rules.dialog.title' => 'Añadir regla de captura',
			'web.memoryAmbient.rules.dialog.nameLabel' => 'Nombre',
			'web.memoryAmbient.rules.dialog.triggerLabel' => 'Disparador',
			'web.memoryAmbient.rules.dialog.nLabel' => 'N (mensajes)',
			'web.memoryAmbient.rules.dialog.idleLabel' => 'Segundos de inactividad',
			'web.memoryAmbient.rules.dialog.kLabel' => 'K (caracteres)',
			'web.memoryAmbient.rules.dialog.scopeLabel' => 'Ámbito objetivo',
			'web.memoryAmbient.rules.dialog.scopeProject' => 'proyecto (recomendado)',
			'web.memoryAmbient.rules.dialog.scopeGlobal' => 'global',
			'web.memoryAmbient.rules.dialog.dedupLabel' => 'Umbral de dedup (0.0 a 1.0)',
			'web.memoryAmbient.rules.dialog.dedupHint' => 'Más alto = deduplicación más estricta. 0.85 es el punto óptimo recomendado.',
			'web.memoryAmbient.rules.dialog.create' => 'Crear',
			'web.memoryAmbient.rules.dialog.nameRequiredToast' => 'El nombre es obligatorio',
			'web.memoryAmbient.rules.dialog.createdToast' => ({required Object name}) => 'Regla ${name} creada',
			'web.memoryAmbient.rules.dialog.createFailedToast' => 'La creación falló',
			'web.memoryAmbient.profiles.title' => 'Perfiles de inyección',
			'web.memoryAmbient.profiles.addButton' => 'Añadir perfil',
			'web.memoryAmbient.profiles.intro' => 'Al arrancar, opendray antepone un banner en markdown con las memorias recientes del proyecto al system prompt del agente, SI hay un perfil configurado. Sin un perfil, el modelo sigue usando memory_search bajo demanda.',
			'web.memoryAmbient.profiles.empty' => 'No hay perfil de inyección. Las memorias no se inyectan automáticamente al arrancar; el modelo sigue usando memory_search.',
			'web.memoryAmbient.profiles.row.globalDefault' => 'predeterminado global',
			'web.memoryAmbient.profiles.row.delete' => 'Eliminar',
			'web.memoryAmbient.profiles.row.deleteConfirm' => '¿Eliminar este perfil de inyección?',
			'web.memoryAmbient.profiles.row.deletedToast' => 'Perfil eliminado',
			'web.memoryAmbient.profiles.row.deleteFailedToast' => 'La eliminación falló',
			'web.memoryAmbient.profiles.dialog.title' => 'Añadir perfil de inyección',
			'web.memoryAmbient.profiles.dialog.strategyLabel' => 'Estrategia',
			'web.memoryAmbient.profiles.dialog.kLabel' => 'K (principales memorias a inyectar)',
			'web.memoryAmbient.profiles.dialog.hint' => 'Un perfil por session_id (o predeterminado global). Los perfiles por session se pueden añadir más tarde mediante API; la UI actualmente solo gestiona el predeterminado global.',
			'web.memoryAmbient.profiles.dialog.create' => 'Crear',
			'web.memoryAmbient.profiles.dialog.createdToast' => 'Perfil creado',
			'web.memoryAmbient.profiles.dialog.createFailedToast' => 'La creación falló',
			'web.memoryAmbient.cost.title' => 'Coste de tokens (histórico total)',
			'web.memoryAmbient.cost.intro' => 'Resumen por proveedor agregado a partir de <1>memory_summarizer_calls</1>. Los proveedores locales (Ollama, LM Studio, Integración) tienen precio de \$0: el operador asume el coste del hardware.',
			'web.memoryAmbient.cost.empty' => 'No hay proveedores habilitados: no hay datos de coste.',
			'web.memoryAmbient.cost.columns.provider' => 'Proveedor',
			'web.memoryAmbient.cost.columns.calls' => 'Llamadas',
			'web.memoryAmbient.cost.columns.inTokens' => 'Tokens de entrada',
			'web.memoryAmbient.cost.columns.outTokens' => 'Tokens de salida',
			'web.memoryAmbient.cost.columns.usdEst' => 'USD est.',
			'web.noteEditor.loading' => 'Cargando…',
			'web.noteEditor.source' => 'Origen',
			'web.noteEditor.preview' => 'Vista previa',
			'web.noteEditor.tagTitle' => ({required Object tag}) => 'etiqueta #${tag}',
			'web.noteEditor.emptyNote' => 'Nota vacía. Cambia a Origen para empezar a escribir.',
			'web.noteEditor.saveFailedToast' => 'Error al guardar',
			'web.noteEditor.status.saveFailed' => 'error al guardar',
			'web.noteEditor.status.saving' => 'guardando…',
			'web.noteEditor.status.unsaved' => 'sin guardar',
			'web.noteEditor.status.newNote' => 'nota nueva',
			'web.noteEditor.status.saved' => 'guardada',
			'web.export.title' => 'Exportar datos',
			'web.export.subtitle' => 'Genera un paquete zip puntual de las entidades lógicas seleccionadas. Los paquetes se conservan en el servidor durante 24 horas y luego se eliminan automáticamente.',
			'web.export.backToBackups' => '← Backups',
			'web.export.sections.export' => 'Exportar',
			'web.export.sections.import' => 'Importar',
			'web.export.form.scope' => 'Alcance',
			'web.export.form.memories' => 'Memorias',
			'web.export.form.memoriesHint' => 'Filas de memoria persistente entre CLI (texto + alcance + metadatos). Los vectores de embedding se omiten; el importador vuelve a generarlos.',
			'web.export.form.integrations' => 'Integraciones',
			'web.export.form.customTasks' => 'Tareas personalizadas',
			'web.export.form.customTasksHint' => 'Tareas definidas por el operador que se muestran en la pestaña Tareas del Inspector.',
			'web.export.form.integrationOptions.none' => 'Ninguna',
			'web.export.form.integrationOptions.noneHint' => 'Omitir por completo la tabla de integraciones.',
			'web.export.form.integrationOptions.metadata' => 'Solo metadatos (recomendado)',
			'web.export.form.integrationOptions.metadataHint' => 'ID, nombre, prefijo de ruta, alcances. Sin material de API key.',
			'web.export.form.integrationOptions.plaintext' => 'Incluir las API keys en texto plano',
			'web.export.form.integrationOptions.plaintextHint' => 'v1 solo con bcrypt: no existe texto plano recuperable. El manifiesto lo documenta; no se filtra nada.',
			'web.export.form.confirmWarning' => 'Escribe <1>Lo entiendo</1> para confirmar. opendray actualmente almacena solo hashes bcrypt, así que seleccionar texto plano NO exporta ningún texto plano (la función está reservada para una versión futura que mantenga cachés de texto plano).',
			'web.export.form.confirmPlaceholder' => 'Lo entiendo',
			'web.export.form.confirmSentinel' => 'lo entiendo',
			'web.export.form.footnote' => 'Los logs de auditoría y los transcripts de session quedan fuera del alcance; en su lugar los cubre /backups (volcado del operador).',
			'web.export.form.building' => 'Generando…',
			'web.export.form.create' => 'Crear exportación',
			'web.export.form.readyToast' => 'Exportación lista',
			'web.export.form.readyDescription' => ({required Object bytes}) => '${bytes} bytes',
			_ => null,
		} ?? switch (path) {
			'web.export.form.failedToast' => 'Falló la exportación',
			'web.export.history.loading' => 'Cargando…',
			'web.export.history.empty' => 'Aún no hay exportaciones. Usa el formulario de arriba para crear una.',
			'web.export.history.title' => 'Historial',
			'web.export.history.columns.id' => 'ID',
			'web.export.history.columns.status' => 'Estado',
			'web.export.history.columns.scope' => 'Alcance',
			'web.export.history.columns.size' => 'Tamaño',
			'web.export.history.columns.expires' => 'Caduca',
			'web.export.history.columns.actions' => 'Acciones',
			'web.export.history.download' => 'Descargar',
			'web.export.history.deleteTooltip' => 'Eliminar',
			'web.export.history.listFailedToast' => 'No se pudieron listar las exportaciones',
			'web.export.history.downloadFailedToast' => 'Falló la descarga',
			'web.export.history.noTokenToast' => 'Sin token de descarga (¿caducado?)',
			'web.export.history.deleteConfirm' => ({required Object id}) => '¿Eliminar la exportación ${id}?',
			'web.export.history.deletedToast' => 'Exportación eliminada',
			'web.export.history.deleteFailedToast' => 'Falló la eliminación',
			'web.export.history.scopeEmpty' => '(vacío)',
			'web.export.import.intro' => 'Reproduce un paquete de exportación (zip) en la base de datos en vivo. Los conflictos (id coincidente, o route_prefix único para integraciones) se <1>omiten</1> de forma predeterminada. Las memorias se etiquetan con <3>embedder=imported_v1</3> y necesitan una pasada de re-embedding antes de que la búsqueda las devuelva; activa el re-embedding en <5>Memory → Maintenance</5>. Las integraciones se importan con <7>enabled=false</7> y una clave de marcador de posición sin bcrypt; el operador debe rotarla antes de usarla.',
			'web.export.import.memoryLink' => 'Memory → Maintenance',
			'web.export.import.bundleLabel' => 'Paquete (.zip)',
			'web.export.import.memoriesLabel' => 'Memorias',
			'web.export.import.integrationsLabel' => 'Integraciones (solo metadatos, las claves nunca se importan)',
			'web.export.import.customTasksLabel' => 'Tareas personalizadas',
			'web.export.import.importing' => 'Importando…',
			'web.export.import.importBundle' => 'Importar paquete',
			'web.export.import.pickFileToast' => 'Selecciona primero un archivo de paquete',
			'web.export.import.doneToast' => 'Importación completada',
			'web.export.import.finishedWithErrors' => 'La importación terminó con errores',
			'web.export.import.failedToast' => 'Falló la importación',
			'web.export.import.summaryCard.memories' => 'Memorias',
			'web.export.import.summaryCard.integrations' => 'Integraciones',
			'web.export.import.summaryCard.customTasks' => 'Tareas personalizadas',
			'web.export.import.summaryCard.created' => 'creadas',
			'web.export.import.summaryCard.skipped' => 'omitidas',
			'web.export.import.summaryCard.failed' => 'fallidas',
			'web.export.imports.loading' => 'Cargando…',
			'web.export.imports.empty' => 'Aún no hay importaciones.',
			'web.export.imports.title' => 'Historial',
			'web.export.imports.columns.id' => 'ID',
			'web.export.imports.columns.status' => 'Estado',
			'web.export.imports.columns.source' => 'Origen',
			'web.export.imports.columns.counts' => 'Recuentos',
			'web.export.imports.columns.when' => 'Cuándo',
			'web.export.imports.noneCounts' => '(ninguno)',
			'web.export.imports.listFailedToast' => 'No se pudieron listar las importaciones',
			'web.knowledge.title' => 'Conocimiento',
			'web.knowledge.subtitle' => 'Lo que sabemos en todos los proyectos: infraestructura y reglas fundacionales, más lecciones y funciones reutilizables destiladas del trabajo previo. Se inyecta para arrancar cada proyecto nuevo.',
			'web.knowledge.searchPlaceholder' => 'Buscar conocimiento…',
			'web.knowledge.search' => 'Buscar',
			'web.knowledge.browse' => 'Explorar',
			'web.knowledge.cwdPlaceholder' => 'Ruta del proyecto (cwd) para búsqueda con ámbito',
			'web.knowledge.noResults' => 'Sin resultados.',
			'web.knowledge.empty' => 'Aún no hay nada. El conocimiento se destila automáticamente mientras trabajas.',
			'web.knowledge.neighbors' => 'Conexiones',
			'web.knowledge.promote' => 'Promover a global',
			'web.knowledge.skillify' => 'Crear habilidad',
			'web.knowledge.promoted' => 'Promovido a global',
			'web.knowledge.skillified' => ({required Object title}) => 'Habilidad creada: ${title}',
			'web.knowledge.actionFailed' => 'La acción falló',
			'web.knowledge.selectHint' => 'Selecciona un nodo para ver los detalles.',
			'web.knowledge.scope' => 'Ámbito',
			'web.knowledge.delete' => 'Eliminar',
			'web.knowledge.deleted' => 'Eliminado',
			'web.knowledge.deleteConfirm' => '¿Eliminar este nodo? Las habilidades quedan eliminadas; los hechos/entidades derivados automáticamente pueden reaparecer en el próximo barrido.',
			'web.knowledge.scopes.all' => 'Todos',
			'web.knowledge.scopes.global' => 'Global',
			'web.knowledge.scopes.project' => 'Proyecto',
			'web.knowledge.scopes.domain' => 'Dominio',
			'web.knowledge.kb.tab' => 'Base de conocimiento',
			'web.knowledge.kb.graphTab' => 'Grafo',
			'web.knowledge.kb.graphCounts' => ({required Object nodes, required Object edges}) => '${nodes} nodos · ${edges} enlaces',
			'web.knowledge.kb.global' => 'Global',
			'web.knowledge.kb.projectHandbook' => 'Manual del proyecto',
			'web.knowledge.kb.locked' => 'Editado por ti',
			'web.knowledge.kb.aiDrafted' => 'Redactado por IA',
			'web.knowledge.kb.edit' => 'Editar',
			'web.knowledge.kb.unlock' => 'Desbloquear (que lo gestione la IA)',
			'web.knowledge.kb.regenerate' => 'Regenerar',
			'web.knowledge.kb.save' => 'Guardar',
			'web.knowledge.kb.cancel' => 'Cancelar',
			'web.knowledge.kb.editHint' => 'Guardar bloquea esta página para que la IA no la sobrescriba.',
			'web.knowledge.kb.empty' => 'Aún no generada. Pulsa Regenerar, o se construye automáticamente mientras trabajas.',
			'web.knowledge.kb.saved' => 'Guardado',
			'web.knowledge.kb.unlocked' => 'Desbloqueada — la IA volverá a gestionar esta página',
			'web.knowledge.kb.regenerating' => 'Regenerando en segundo plano…',
			'web.knowledge.kb.kinds.kb_infrastructure' => 'Infraestructura',
			'web.knowledge.kb.kinds.kb_conventions' => 'Convenciones',
			'web.knowledge.kb.kinds.kb_lessons' => 'Lecciones',
			'web.knowledge.kb.kinds.kb_reusable' => 'Funciones reutilizables',
			'web.knowledge.kb.foundational' => 'Fundacional',
			'web.knowledge.kb.foundationalHint' => 'Infraestructura y convenciones — reglas vinculantes inyectadas en cada proyecto.',
			'web.knowledge.kb.emergent' => 'Emergente',
			'web.knowledge.kb.emergentHint' => 'Lecciones y funciones reutilizables destiladas del trabajo previo — orientación.',
			'web.knowledge.kb.bindingBadge' => 'Vinculante · obligatorio',
			'web.knowledge.kb.referenceBadge' => 'Referencia',
			'web.knowledge.kb.proposal.text' => 'La IA propuso una actualización de esta página (evidencia nueva divergente).',
			'web.knowledge.kb.proposal.preview' => 'Vista previa',
			'web.knowledge.kb.proposal.hide' => 'Ocultar',
			'web.knowledge.kb.proposal.approve' => 'Aprobar',
			'web.knowledge.kb.proposal.reject' => 'Rechazar',
			'web.knowledge.kb.proposal.approved' => 'Actualización aprobada',
			'web.knowledge.kb.proposal.rejected' => 'Propuesta rechazada',
			'web.knowledge.kb.discuss' => 'Hablar con la IA',
			'web.knowledge.kb.discussHint' => 'Redacta de nuevo esta política conversando con la IA — las páginas bloqueadas reciben propuestas, nunca sobrescrituras',
			'web.knowledge.kb.onDemand' => 'bajo demanda',
			'web.knowledge.kb.removePage' => 'Quitar página',
			'web.knowledge.kb.removePageHint' => 'Quita esta página de la base de conocimiento (su contenido se conserva y vuelve si se re-añade el slug)',
			'web.knowledge.kb.pageRemovedToast' => 'Página quitada',
			'web.knowledge.kb.newPage.button' => 'Nueva página de conocimiento',
			'web.knowledge.kb.newPage.title' => 'Nueva página de conocimiento',
			'web.knowledge.kb.newPage.description' => 'Da a cada dominio de conocimiento su propia página de grano fino en vez de engordar las clásicas — los agentes indexan páginas individualmente y recuperan solo lo que una tarea necesita.',
			'web.knowledge.kb.newPage.slugPlaceholder' => 'network_topology',
			'web.knowledge.kb.newPage.titlePlaceholder' => 'Título (p. ej. Topología de red)',
			'web.knowledge.kb.newPage.descPlaceholder' => 'Una frase: qué va en esta página',
			'web.knowledge.kb.newPage.inject' => 'inyectar en cada arranque',
			'web.knowledge.kb.newPage.injectHint' => 'Apagado (recomendado): la página queda fuera del banner de arranque y los agentes la alcanzan bajo demanda vía búsqueda. Encendido: las fundacionales se inyectan como reglas vinculantes, las emergentes como referencia.',
			'web.knowledge.kb.newPage.create' => 'Crear página',
			'web.knowledge.kb.newPage.createdToast' => 'Página de conocimiento creada',
			'web.knowledge.kinds.all' => 'Todos',
			'web.knowledge.kinds.entity' => 'Entidades',
			'web.knowledge.kinds.fact' => 'Hechos',
			'web.knowledge.kinds.playbook' => 'Guías',
			'web.knowledge.kinds.skill' => 'Habilidades',
			'web.knowledge.distill.tab' => 'Destilación',
			'web.knowledge.distill.intro' => 'Un SKILL es un PROCEDIMIENTO probado y repetible destilado de tu trabajo real. El compilador de experiencia mina los diarios de sesión de TODOS los proyectos, agrupa trabajo similar y solo redacta un candidato cuando el mismo procedimiento TUVO ÉXITO en 2+ sesiones — cada cita de evidencia se verifica literalmente contra el diario. Los candidatos se ordenan por recurrencia × el coste en tiempo del procedimiento manual; los procedimientos totalmente mecánicos también se compilan a un run.sh ejecutable con paso de validación.',
			'web.knowledge.distill.playbooks' => 'Playbooks — destilados, pendientes de revisión',
			'web.knowledge.distill.playbooksHint' => 'Cada candidato pasó las puertas: ≥2 sesiones exitosas, citas de evidencia verificadas, ≥3 pasos concretos. Ordenados por tiempo ahorrado (recurrencia × minutos manuales). Promueve lo reutilizable, descarta el resto.',
			'web.knowledge.distill.playbooksEmpty' => 'Nada minado aún — los candidatos aparecen cuando el mismo procedimiento tiene éxito en dos o más sesiones.',
			'web.knowledge.distill.skills' => 'Skills — activos, inyectados al arranque',
			'web.knowledge.distill.skillsHint' => 'Playbooks promovidos. Cada sesión nueva los recibe como skills.',
			'web.knowledge.distill.skillsEmpty' => 'Sin skills aún — promueve un playbook para crear el primero.',
			'web.knowledge.distill.skillify' => 'Promover a skill',
			'web.knowledge.distill.skillifyHint' => 'Renderizar como skill e inyectar en cada arranque',
			'web.knowledge.distill.discard' => 'Descartar',
			'web.knowledge.distill.retire' => 'Retirar skill',
			'web.knowledge.distill.injectedBadge' => 'inyectado',
			'web.knowledge.distill.skillifiedToast' => 'Promovido — publicado en Plugins → Agent Skills; las nuevas sessions reciben esta skill',
			'web.knowledge.distill.removedToast' => 'Eliminado',
			'web.knowledge.distill.usage' => ({required Object count}) => 'usado en ${count} sesiones',
			'web.knowledge.distill.lastUsed' => ({required Object date}) => 'último ${date}',
			'web.knowledge.distill.enabledToast' => 'Skill activado — las sesiones nuevas lo cargan',
			'web.knowledge.distill.disabledToast' => 'Skill desactivado — fuera del conjunto cargado',
			'web.knowledge.distill.disabledBadge' => 'off',
			'web.knowledge.distill.toggleHint' => 'Solo los skills activados se cargan; desactiva lo que esta etapa no necesita',
			'web.knowledge.distill.viewHint' => 'Clic para ver el procedimiento completo',
			'web.knowledge.distill.inAgentSkills' => 'en Plugins → Agent Skills',
			'web.knowledge.distill.agentSkillsHint' => 'El SKILL.md renderizado vive en el vault de skills — míralo o gestiónalo en Plugins → Agent Skills.',
			'web.knowledge.distill.notInVault' => 'desactivado — SKILL.md retirado del vault',
			'web.knowledge.distill.compiledBadge' => 'compilado',
			'web.knowledge.distill.compiledHint' => 'Incluye un run.sh ejecutable con paso de validación; al promover también se registra como tarea personalizada',
			'web.knowledge.distill.recurrence' => ({required Object count}) => 'exitoso ×${count}',
			'web.knowledge.distill.timeCost' => ({required Object minutes}) => '~${minutes} min manual',
			'web.knowledge.distill.projectSpan' => ({required Object count}) => '${count} proyectos',
			'web.knowledge.distill.scoreHint' => 'Ordenado por recurrencia × coste de tiempo manual — lo que más tiempo ahorra se destila primero',
			'web.knowledge.distill.outcomes' => ({required Object ok, required Object failed}) => '${ok} ok / ${failed} fallidas tras cargarlo',
			'web.knowledge.distill.retirement.never_used' => 'nunca usado',
			'web.knowledge.distill.retirement.never_usedHint' => 'Inyectado 14+ días sin que ninguna sesión lo refiera — el bucle de resultados propone retirarlo',
			'web.knowledge.distill.retirement.low_success' => 'poco éxito',
			'web.knowledge.distill.retirement.low_successHint' => 'Las sesiones que cargan este skill siguen terminando en fallo — el bucle de resultados propone retirarlo',
			'web.knowledge.distill.retirement.dormant' => 'inactivo',
			'web.knowledge.distill.retirement.dormantHint' => 'Se usó alguna vez, pero lleva 45+ días sin referencias — el bucle de resultados propone retirarlo',
			'web.knowledge.distill.retirementEmpty' => 'Sin candidatos a retiro: todas las habilidades aportan.',
			'web.knowledge.distill.retirementHint' => 'Habilidades que el bucle de resultados propone descartar; desactiva las que consideres.',
			'web.knowledge.distill.retirementTitle' => 'Candidatos a retiro',
			'web.knowledge.graph.tab' => 'Grafo',
			'web.knowledge.graph.intro' => 'El mapa de relaciones de todo lo que la IA ha aprendido: qué proyectos comparten tecnología, qué skills y trampas se asocian a qué entidades. Comprueba aquí el radio de impacto de un nodo ANTES de tocar infraestructura compartida.',
			'web.knowledge.graph.empty' => 'Sin conocimiento aún — el grafo se construye solo mientras corren las sessions: el barrido de anclaje extrae entidades del trabajo de proyecto y la destilación añade playbooks y skills. Vuelve tras unas cuantas sesiones de trabajo.',
			'web.knowledge.graph.hint' => 'Rueda para zoom · arrastra el fondo para desplazarte · arrastra un nodo para desenredar · clic en un nodo para inspeccionarlo',
			'web.knowledge.graph.legend.project' => 'Proyecto',
			'web.knowledge.graph.legend.entity' => 'Entidad',
			'web.knowledge.graph.legend.playbook' => 'Playbook',
			'web.knowledge.graph.legend.skill' => 'Skill',
			'web.knowledge.graph.connections' => ({required Object count}) => '${count} nodos conectados',
			'web.knowledge.graph.noLinks' => 'Nada enlaza con este nodo todavía.',
			'web.cortex.home.title' => 'Cortex',
			'web.cortex.home.subtitle' => 'Un módulo, tres peldaños, un ciclo: la memoria bruta cristaliza en el documento oficial de cada proyecto, se destila en conocimiento entre proyectos y se inyecta en cada nueva sesión.',
			'web.cortex.home.disabled' => 'desactivado',
			'web.cortex.home.pendingProposals' => ({required Object count}) => '${count} pendientes',
			'web.cortex.home.loopHint' => 'Memoria → Notas → Conocimiento → inyectado en cada arranque. Ascender es transformar, nunca copiar.',
			'web.cortex.home.activeProjects' => 'Proyectos activos',
			'web.cortex.home.idle' => ({required Object days}) => 'inactivo ${days}d',
			'web.cortex.home.memory.title' => 'Memoria',
			'web.cortex.home.memory.description' => 'Hechos episódicos capturados de tus sesiones — recuperados por relevancia, en cuarentena si vienen de terceros.',
			'web.cortex.home.memory.quarantine' => ({required Object count}) => '${count} en cuarentena',
			'web.cortex.home.notes.title' => 'Notas',
			'web.cortex.home.notes.description' => 'El documento oficial de cada proyecto — secciones según su plano, mantenidas por la IA mientras trabajas.',
			'web.cortex.home.notes.projects' => ({required Object count}) => '${count} activos',
			'web.cortex.home.knowledge.title' => 'Conocimiento',
			'web.cortex.home.knowledge.description' => 'Experiencia iterable entre proyectos: reglas fundacionales vinculantes + lecciones emergentes, inyectadas en cada arranque.',
			'web.cortex.home.settings' => 'Ajustes',
			'web.cortex.home.proposals.title' => ({required Object count}) => 'Propuestas pendientes (${count})',
			'web.cortex.home.proposals.hint' => 'Actualizaciones propuestas por la IA para notas de proyecto y páginas KB, a la espera de tu veredicto. Aprueba para publicar, rechaza para descartar.',
			'web.cortex.home.proposals.kbLabel' => 'Base de conocimiento',
			'web.cortex.home.proposals.preview' => 'Vista previa',
			'web.cortex.home.proposals.hide' => 'Ocultar',
			'web.cortex.home.proposals.approve' => 'Aprobar',
			'web.cortex.home.proposals.reject' => 'Rechazar',
			'web.cortex.home.proposals.open' => 'Abrir la página correspondiente',
			'web.cortex.home.proposals.approvedToast' => 'Propuesta aprobada — documento actualizado',
			'web.cortex.home.proposals.rejectedToast' => 'Propuesta rechazada',
			'web.cortex.home.proposals.failedToast' => 'La acción falló',
			'web.cortex.chat.title' => 'Chat de curación',
			'web.cortex.chat.show' => 'Hablar con la IA',
			'web.cortex.chat.hide' => 'Ocultar chat',
			'web.cortex.chat.emptyHint' => 'Pide a la IA actualizar, reestructurar o reescribir este documento. Los cambios se aplican directamente si lo mantiene la IA, o llegan a la bandeja si lo bloqueaste.',
			'web.cortex.chat.placeholder' => 'p. ej. actualiza esto con el trabajo reciente · ⌘↵ para enviar',
			'web.cortex.chat.thinking' => 'La IA está trabajando…',
			'web.cortex.chat.sendFailed' => 'Error al enviar',
			'web.cortex.chat.escalate' => 'Escalar a sesión',
			'web.cortex.chat.escalated' => 'Escalado',
			'web.cortex.chat.escalateHint' => 'Lanza una sesión de agente completa, fundamentada en el código, con esta conversación',
			'web.cortex.chat.escalateFailed' => 'Error al escalar',
			'web.cortex.chat.escalatedToast' => 'Sesión de agente lanzada',
			'web.cortex.chat.closeHint' => 'Cerrar esta conversación',
			'web.cortex.chat.revisionApplied' => 'Documento actualizado',
			'web.cortex.chat.revisionProposed' => 'Propuesta creada — revísala en la bandeja',
			'web.cortex.blueprint.open' => 'Plano',
			'web.cortex.blueprint.openHint' => 'Edita qué secciones documenta este proyecto',
			'web.cortex.blueprint.title' => 'Plano del documento',
			'web.cortex.blueprint.description' => 'El conjunto de secciones del documento oficial. Cada tipo de proyecto merece secciones distintas — dale forma tú o deja que la IA proponga.',
			'web.cortex.blueprint.propose' => 'Proponer (IA)',
			'web.cortex.blueprint.proposeHint' => 'Clasifica el proyecto y propone un conjunto de secciones a medida',
			'web.cortex.blueprint.proposeFailed' => 'Propuesta fallida',
			'web.cortex.blueprint.proposalNote' => ({required Object type, required Object reason}) => 'La IA lo clasificó como: ${type} — ${reason} Revísalo, edítalo y aplica.',
			'web.cortex.blueprint.addSection' => 'Añadir sección',
			'web.cortex.blueprint.slugPlaceholder' => 'slug',
			'web.cortex.blueprint.titlePlaceholder' => 'Título',
			'web.cortex.blueprint.hintPlaceholder' => 'Pista para el mantenedor — una frase que guíe a la IA (opcional)',
			'web.cortex.blueprint.mode.ai' => 'IA',
			'web.cortex.blueprint.mode.human' => 'Humano',
			'web.cortex.blueprint.mode.scanner' => 'Escáner',
			'web.cortex.blueprint.inject' => 'inyectar',
			'web.cortex.blueprint.reserved' => 'reservada',
			'web.cortex.blueprint.deleteNote' => 'Quitar una sección la oculta sin borrar su contenido — vuelve a añadir el mismo slug para recuperarla.',
			'web.cortex.blueprint.cancel' => 'Cancelar',
			'web.cortex.blueprint.apply' => 'Aplicar plano',
			'web.cortex.blueprint.applyFailed' => 'Error al aplicar',
			'web.cortex.blueprint.appliedToast' => 'Plano aplicado',
			'web.cortex.quarantine.title' => 'Cuarentena',
			'web.cortex.quarantine.subtitle' => 'Hechos que necesitan revisión antes de contar como memoria durable: las capturas de integraciones de terceros llegan aquí por política, y puedes poner cualquier memoria en cuarentena a mano desde el inspector de Memoria. Promueve lo verdadero; descarta el resto — las filas sin revisar expiran solas.',
			'web.cortex.quarantine.empty' => 'Nada en cuarentena. Las filas llegan desde sessions de origen integración (política “quarantine”) o cuando pones una memoria en cuarentena manualmente en el inspector de Memoria.',
			'web.cortex.quarantine.promote' => 'Promocionar',
			'web.cortex.quarantine.promoteHint' => 'Mover a memoria duradera (entra en la recuperación y consolidación)',
			'web.cortex.quarantine.discard' => 'Descartar',
			'web.cortex.quarantine.promotedToast' => 'Promocionada a memoria duradera',
			'web.cortex.quarantine.discardedToast' => 'Descartada',
			'web.cortex.quarantine.actionFailed' => 'Acción fallida',
			'web.cortex.quarantine.expires' => ({required Object date}) => 'expira ${date}',
			'web.cortex.settings.injection.title' => 'Inyección al arranque',
			'web.cortex.settings.injection.hint' => 'Cuánto contexto de Cortex carga cada SESIÓN NUEVA por adelantado. El cambio aplica de inmediato a las sesiones creadas después — el backend nunca necesita reiniciarse.',
			'web.cortex.settings.injection.active' => 'activo',
			'web.cortex.settings.injection.mode.lean.label' => 'Ligero — índice + bajo demanda (recomendado)',
			'web.cortex.settings.injection.mode.lean.description' => 'Inyecta solo las reglas fundacionales vinculantes más un índice compacto de secciones y páginas de conocimiento. Los agentes recuperan exactamente lo que necesitan vía doc_read / project_search. Ahorra tokens y evita ahogar las sesiones largas.',
			'web.cortex.settings.injection.mode.full.label' => 'Completo — inyectar todo',
			'web.cortex.settings.injection.mode.full.description' => 'Inyecta al arranque cada sección y página marcada para inyección, completa (comportamiento clásico). Simple, pero cuesta tokens en cada sesión y satura la ventana de contexto.',
			'web.cortex.settings.injection.savedToast' => 'Modo guardado — las sesiones nuevas lo usan de inmediato (sin reiniciar el backend)',
			'web.cortex.settings.injection.saveFailed' => 'Error al guardar',
			'web.cortex.settings.injection.note' => 'En modo completo siguen aplicando los flags de inyección por sección/página; en modo ligero las reglas fundacionales siempre se inyectan y el resto va al índice.',
			'more.title' => 'Más',
			'more.identity.signedInAs' => 'Sesión iniciada como',
			'more.identity.server' => 'Servidor',
			'more.identity.tokenExpires' => 'El token caduca',
			'more.sections.gateway' => 'Gateway',
			'more.sections.plugins' => 'Complementos',
			'more.sections.memory' => 'Memoria',
			'more.sections.system' => 'Sistema',
			'more.items.integrations.title' => 'Integraciones',
			'more.items.integrations.subtitle' => 'Llamadores de la API: actividad reciente y tasas de error',
			'more.items.activity.title' => 'Actividad',
			'more.items.activity.subtitle' => 'Auditoría de llamadas API de integraciones',
			'more.items.memoryAmbient.title' => 'Captura e inyección',
			'more.items.memoryAmbient.subtitle' => 'Reglas de captura + perfiles de inyección',
			'more.items.channels.title' => 'Canales',
			'more.items.channels.subtitle' => 'Destinos de notificaciones',
			'more.items.providers.title' => 'Proveedores',
			'more.items.providers.subtitle' => 'Estado de los CLI de Claude / Codex / Gemini',
			'more.items.mcp.title' => 'MCP',
			'more.items.mcp.subtitle' => 'Servidores y secretos de Model Context Protocol',
			'more.items.skills.title' => 'Skills',
			'more.items.skills.subtitle' => 'Biblioteca de SKILL.md del agente (integrados + vault)',
			'more.items.gitHosts.title' => 'Hosts de Git',
			'more.items.gitHosts.subtitle' => 'Credenciales PAT para GitHub / GitLab / etc.',
			'more.items.customTasks.title' => 'Tareas personalizadas',
			'more.items.customTasks.subtitle' => 'Comandos slash que se muestran en el selector de tareas de la session',
			'more.items.cortexHub.title' => 'Cortex',
			'more.items.cortexHub.subtitle' => 'Hub Memoria → Notas → Conocimiento + propuestas pendientes',
			'more.items.projectMemory.title' => 'Objetivo / plan / diario del proyecto',
			'more.items.projectMemory.subtitle' => 'Capas de memoria 2-4 por cwd + propuestas del agente',
			'more.items.archived.title' => 'Memorias archivadas',
			'more.items.archived.subtitle' => 'Restaura memorias que el limpiador automático archivó (gracia de 30 días)',
			'more.items.quarantine.title' => 'Cuarentena',
			'more.items.quarantine.subtitle' => 'Revisa memorias capturadas antes de que sean durables',
			'more.items.backups.title' => 'Copias de seguridad',
			'more.items.backups.subtitle' => 'Estado de la última copia de seguridad y ejecución inmediata',
			'more.items.dataExport.title' => 'Exportación e importación de datos',
			'more.items.dataExport.subtitle' => 'Paquetes de datos a nivel de usuario (memorias / integraciones / tareas personalizadas)',
			'more.items.settings.title' => 'Ajustes',
			'more.items.settings.subtitle' => 'Idioma, apariencia, cuenta',
			'more.items.about.title' => 'Acerca de',
			'more.items.about.subtitle' => 'Versión de compilación e información del servidor',
			'more.items.vault.title' => 'Bóveda',
			'more.items.vault.subtitle' => 'Notas markdown libres (sincronización Obsidian)',
			'more.signOut' => 'Cerrar sesión',
			'activity.title' => 'Actividad',
			'activity.empty' => 'Aún no hay llamadas de integración registradas.',
			'activity.loadFailed' => 'Error al cargar la actividad',
			'activity.callsCount' => ({required Object count}) => '${count} llamadas',
			'activity.directionInbound' => 'entrante',
			'activity.directionOutbound' => 'saliente',
			'activity.filter.title' => 'Filtrar llamadas',
			'activity.filter.direction' => 'Dirección',
			'activity.filter.directionAll' => 'Todas',
			'activity.filter.status' => 'Estado',
			'activity.filter.statusAll' => 'Todos',
			'activity.filter.integration' => 'Integración',
			'activity.filter.integrationAll' => 'Todas las integraciones',
			'activity.filter.apply' => 'Aplicar',
			'activity.filter.clear' => 'Limpiar',
			'activity.filter.activeCount' => ({required Object count}) => '${count} activos',
			'activity.detail.title' => 'Detalle de llamada',
			'activity.detail.integration' => 'Integración',
			'activity.detail.direction' => 'Dirección',
			'activity.detail.status' => 'Estado',
			'activity.detail.duration' => 'Duración',
			'activity.detail.bytes' => 'Bytes',
			'activity.detail.requestId' => 'ID de solicitud',
			'activity.detail.resource' => 'Recurso',
			'activity.detail.timestamp' => 'Marca de tiempo',
			'memoryAmbient.title' => 'Captura e inyección',
			'memoryAmbient.intro' => 'Cómo se resumen las sesiones en memoria y qué contexto se precarga. La creación de reglas y la edición detallada están en el panel web.',
			'memoryAmbient.captureSection' => 'Reglas de captura',
			'memoryAmbient.injectionSection' => 'Perfiles de inyección',
			'memoryAmbient.empty' => 'Nada configurado aún.',
			'memoryAmbient.loadFailed' => 'Error al cargar',
			'memoryAmbient.runNow' => 'Ejecutar ahora',
			'memoryAmbient.ranSnack' => ({required Object count}) => 'Ejecutado en ${count} sesión(es)',
			'memoryAmbient.actionFailed' => ({required Object error}) => 'Acción fallida: ${error}',
			'memoryAmbient.strategyLabel' => 'Estrategia',
			'memoryAmbient.scopeProject' => 'proyecto',
			'memoryAmbient.scopeGlobal' => 'global',
			'memoryAmbient.triggerAfterMessages' => 'Tras N mensajes',
			'memoryAmbient.triggerOnIdle' => 'En inactividad',
			'memoryAmbient.triggerKChars' => 'Tras K caracteres',
			'memoryAmbient.triggerManual' => 'Manual',
			'memoryAmbient.triggerUnknown' => 'Desconocido',
			'memoryAmbient.strategyNone' => 'Ninguna (búsqueda bajo demanda)',
			'memoryAmbient.strategyTopKRecent' => 'Top-K recientes',
			'memoryAmbient.strategyTopKRelevant' => 'Top-K relevantes',
			'memoryAmbient.strategyOnKeyword' => 'Por palabra clave',
			'memoryAmbient.strategyManualOnly' => 'Solo manual',
			'memoryAmbient.strategyHybrid' => 'Resumen híbrido',
			'memoryAmbient.strategyUnknown' => 'Desconocido',
			'sessions.title' => 'Sesiones',
			'sessions.refresh' => 'Actualizar',
			'sessions.actions' => 'Acciones',
			'sessions.spawn' => 'Crear',
			'sessions.filters.all' => 'Todas',
			'sessions.filters.running' => 'En ejecución',
			'sessions.filters.idle' => 'Inactivas',
			'sessions.filters.ended' => 'Finalizadas',
			'sessions.card.startedRelative' => ({required Object provider, required Object when}) => '${provider} · iniciada ${when}',
			'sessions.empty.titleAll' => 'Aún no hay sesiones',
			'sessions.empty.titleFiltered' => ({required Object filter}) => 'Ninguna sesión coincide con el filtro "${filter}".',
			'sessions.empty.subtitleAll' => 'Toca el botón Crear para crear una.',
			'sessions.empty.subtitleFiltered' => 'Prueba con otro filtro o desliza para actualizar.',
			'sessions.errorTitle' => 'No se pudieron cargar las sesiones',
			'sessions.relative.secondsAgo' => ({required Object n}) => 'hace ${n}s',
			'sessions.relative.minutesAgo' => ({required Object n}) => 'hace ${n}m',
			'sessions.relative.hoursAgo' => ({required Object n}) => 'hace ${n}h',
			'sessions.relative.daysAgo' => ({required Object n}) => 'hace ${n}d',
			'sessions.detail.fallbackTitle' => 'Sesión',
			'sessions.detail.refreshMetadata' => 'Actualizar metadatos',
			'sessions.detail.inspector' => 'Inspector (Archivos / Git / Tareas / Historial / Notas)',
			'sessions.detail.projectMemory' => 'Memoria del proyecto (objetivo / plan / diario / bandeja de entrada)',
			'sessions.detail.actions' => 'Acciones',
			'sessions.detail.started' => ({required Object when}) => 'iniciada ${when}',
			'sessions.detail.startedEnded' => ({required Object started, required Object ended}) => 'iniciada ${started}  ·  finalizada ${ended}',
			'sessions.detail.idPrefix' => ({required Object id}) => 'id: ${id}',
			'sessions.detail.errorTitle' => 'No se pudo cargar la sesión',
			'sessions.detail.accountSwitcher.tooltip' => 'Cambiar de cuenta de Claude',
			'sessions.detail.accountSwitcher.sheetTitle' => 'Cambiar de cuenta de Claude',
			'sessions.detail.accountSwitcher.current' => ({required Object account}) => 'Actual: ${account}',
			'sessions.detail.accountSwitcher.defaultName' => 'Predeterminada (credencial del sistema)',
			'sessions.detail.accountSwitcher.defaultSubtitle' => 'Usa el propio inicio de sesión del CLI, sin cuenta específica',
			'sessions.detail.accountSwitcher.defaultShort' => 'predeterminada',
			'sessions.detail.accountSwitcher.tokenEmpty' => 'sin token',
			'sessions.detail.accountSwitcher.confirmTitle' => '¿Cambiar de cuenta?',
			'sessions.detail.accountSwitcher.confirmBody' => 'Esto reinicia el CLI con la nueva cuenta — se pierde el contexto de conversación actual dentro del CLI (la pestaña de la sesión se conserva).',
			'sessions.detail.accountSwitcher.confirmAction' => 'Cambiar',
			'sessions.detail.accountSwitcher.cancel' => 'Cancelar',
			'sessions.detail.accountSwitcher.switchedSnack' => ({required Object account}) => 'Cambiado a ${account}',
			'sessions.detail.accountSwitcher.switchFailed' => ({required Object error}) => 'Cambio fallido: ${error}',
			'sessions.detail.accountSwitcher.noneHint' => 'No hay cuentas de Claude configuradas. Agrégalas en Más → Providers → Claude.',
			'sessions.terminal.snackbar.imagePickerFailed' => ({required Object error}) => 'Falló el selector de imágenes: ${error}',
			'sessions.terminal.snackbar.uploadingImage' => 'Subiendo imagen…',
			'sessions.terminal.snackbar.imageAttached' => ({required Object path}) => 'Imagen adjuntada: ${path}',
			'sessions.terminal.snackbar.uploadFailed' => ({required Object status, required Object message}) => 'Falló la subida (${status}): ${message}',
			'sessions.terminal.snackbar.uploadFailedGeneric' => ({required Object error}) => 'Falló la subida: ${error}',
			'sessions.terminal.imageSource.photoLibrary' => 'Biblioteca de fotos',
			'sessions.terminal.imageSource.takePhoto' => 'Tomar una foto',
			'sessions.terminal.keyboard.copyBuffer' => 'Copiar búfer',
			'sessions.terminal.keyboard.paste' => 'Pegar',
			'sessions.terminal.keyboard.attachImage' => 'Adjuntar imagen',
			'sessions.terminal.keyboard.enter' => 'Intro',
			'sessions.terminal.connection.connecting' => 'Conectando…',
			'sessions.terminal.connection.connected' => 'Conectado',
			'sessions.terminal.connection.reconnecting' => 'Reconectando…',
			'sessions.terminal.connection.reconnectingWithError' => ({required Object error}) => 'Reconectando (${error})…',
			'sessions.terminal.connection.disconnected' => 'Desconectado',
			'sessions.terminal.connection.disconnectedWithError' => ({required Object error}) => 'Desconectado (${error})',
			'sessions.terminal.connection.ended' => 'Sesión finalizada',
			'sessions.action.stop' => 'Detener',
			'sessions.action.stopping' => 'Deteniendo…',
			'sessions.action.stopDescription' => 'Envía SIGTERM, conserva el historial',
			'sessions.action.restart' => 'Reiniciar',
			'sessions.action.restarting' => 'Reiniciando…',
			'sessions.action.restartDescription' => 'Vuelve a crear el proceso del CLI',
			'sessions.action.delete' => 'Eliminar',
			'sessions.action.deleteDescription' => 'Elimina la session y su historial',
			'sessions.action.deleteConfirm' => '¿Eliminar esta session de forma permanente? Su ring buffer y su historial desaparecerán.',
			'sessions.action.errors.stop' => ({required Object error}) => 'Falló al detener: ${error}',
			'sessions.action.errors.start' => ({required Object error}) => 'Falló al reiniciar: ${error}',
			'sessions.action.errors.delete' => ({required Object error}) => 'Falló al eliminar: ${error}',
			'sessions.dirPicker.parent' => 'Superior',
			'sessions.dirPicker.newFolder' => 'Nueva carpeta',
			'sessions.dirPicker.useThisFolder' => 'Usar esta carpeta',
			'sessions.dirPicker.loading' => 'Cargando…',
			'sessions.dirPicker.empty' => 'No hay subcarpetas aquí.\nElige esta carpeta o crea una nueva.',
			'sessions.dirPicker.createdSnack' => ({required Object path}) => 'Creada ${path}',
			'sessions.dirPicker.mkdirFailedSnack' => ({required Object error}) => 'Falló mkdir: ${error}',
			'sessions.dirPicker.dialog.title' => 'Nueva carpeta',
			'sessions.dirPicker.dialog.hint' => 'Nombre de la carpeta',
			'sessions.dirPicker.dialog.create' => 'Crear',
			'sessions.inspector.shell.title' => 'Inspector',
			'sessions.inspector.shell.loadError' => ({required Object error}) => 'No se pudo cargar la sesión: ${error}',
			'sessions.inspector.shell.tabs.files' => 'Archivos',
			'sessions.inspector.shell.tabs.git' => 'Git',
			'sessions.inspector.shell.tabs.tasks' => 'Tareas',
			'sessions.inspector.shell.tabs.history' => 'Historial',
			'sessions.inspector.shell.tabs.vault' => 'Bóveda',
			'sessions.inspector.shell.tabs.cortex' => 'Cortex',
			'sessions.inspector.cortex.title' => 'Espacio Cortex',
			'sessions.inspector.cortex.blurb' => 'Objetivo, plan, diario, bandeja y limpieza de memoria de este proyecto — el Cortex mantenido por IA.',
			'sessions.inspector.cortex.open' => 'Abrir espacio Cortex',
			'sessions.inspector.shared.refresh' => 'Actualizar',
			'sessions.inspector.shared.inserted' => ({required Object text}) => 'Insertado: ${text}',
			'sessions.inspector.shared.insertFailedApi' => ({required Object status, required Object message}) => 'Falló la inserción (${status}): ${message}',
			'sessions.inspector.shared.insertFailedGeneric' => ({required Object error}) => 'Falló la inserción: ${error}',
			'sessions.inspector.shared.insertFailedShort' => ({required Object error}) => 'Falló la inserción: ${error}',
			'sessions.inspector.history.insertIntoTerminal' => 'Insertar en el terminal',
			'sessions.inspector.history.searchHint' => 'Buscar prompts…',
			'sessions.inspector.files.insertAtRef' => 'Insertar como @referencia',
			'sessions.inspector.files.insertPath' => 'Insertar ruta',
			'sessions.inspector.files.insertPathSubtitle' => 'Pega la ruta absoluta tal cual',
			'sessions.inspector.files.readContent' => 'Leer contenido',
			'sessions.inspector.files.readContentSubtitle' => 'Hasta 256 KiB de texto plano',
			'sessions.inspector.files.readFailedApi' => ({required Object status, required Object message}) => 'Falló la lectura (${status}): ${message}',
			'sessions.inspector.files.readFailedGeneric' => ({required Object error}) => 'Falló la lectura: ${error}',
			'sessions.inspector.files.parent' => 'Superior',
			'sessions.inspector.files.backToCwd' => 'Volver al cwd de la session',
			'sessions.inspector.git.insertAtRef' => 'Insertar como @referencia',
			'sessions.inspector.git.insertPath' => 'Insertar ruta',
			'sessions.inspector.git.showDiff' => 'Mostrar diff',
			'sessions.inspector.git.diffFailedApi' => ({required Object status, required Object message}) => 'Falló el diff (${status}): ${message}',
			'sessions.inspector.git.diffFailedGeneric' => ({required Object error}) => 'Falló el diff: ${error}',
			'sessions.inspector.git.insertHash' => 'Insertar hash',
			'sessions.inspector.git.showFullPatch' => 'Mostrar el parche completo',
			'sessions.inspector.git.showFailedApi' => ({required Object status, required Object message}) => 'Falló al mostrar (${status}): ${message}',
			'sessions.inspector.git.showFailedGeneric' => ({required Object error}) => 'Falló al mostrar: ${error}',
			'sessions.inspector.git.tabStatus' => 'Estado',
			'sessions.inspector.git.tabLog' => 'Log',
			'sessions.inspector.tasks.runCommand' => 'Ejecutar comando',
			'sessions.inspector.tasks.runCommandSubtitle' => 'Se ejecuta en una nueva session de shell y cambia a ella',
			'sessions.inspector.tasks.filterHint' => 'Filtrar tareas…',
			'sessions.inspector.tasks.noMatch' => ({required Object query}) => 'Ninguna tarea coincide con "${query}"',
			'sessions.inspector.tasks.emptyTitle' => 'No hay tareas en esta carpeta',
			'sessions.inspector.tasks.emptyHint' => 'Buscando package.json, Makefile, Taskfile, justfile, Cargo.toml, go.mod, pyproject.toml o scripts de shell',
			'sessions.inspector.notes.insertedAt' => ({required Object path}) => 'Insertado: @${path}',
			'sessions.inspector.notes.myNotes' => 'Mis notas',
			'sessions.inspector.notes.projectDocs' => 'Documentos del proyecto',
			'sessions.inspector.notes.insertAtRefTooltip' => 'Insertar como @referencia',
			'sessions.inspector.notes.insertAtRefShort' => 'Insertar @referencia',
			'sessions.inspector.notes.draftHint' => ({required Object project}) => '# ${project}\n\nIdeas, tareas pendientes, contexto para el agente…',
			'sessions.inspector.notes.createFailed' => ({required Object error}) => 'Falló al crear: ${error}',
			'sessions.inspector.notes.saveFailed' => ({required Object error}) => 'Falló al guardar: ${error}',
			'sessions.inspector.notes.changeLocationTooltip' => 'Cambiar la ubicación de los documentos del proyecto',
			'sessions.inspector.notes.filenameHint' => 'nombre de archivo (p. ej. spec o design.md)',
			'sessions.inspector.notes.create' => 'Crear',
			'sessions.inspector.notes.filterHint' => 'Filtrar…',
			'sessions.inspector.notes.locationDialogTitle' => 'Ubicación de los documentos del proyecto',
			'sessions.inspector.notes.loadFailedApi' => ({required Object error}) => 'Falló la carga: ${error}',
			'sessions.inspector.notes.loadFailedGeneric' => ({required Object error}) => 'Falló la carga: ${error}',
			'sessions.inspector.notes.saveFailedApi' => ({required Object error}) => 'Falló al guardar: ${error}',
			'sessions.inspector.notes.saveFailedGeneric' => ({required Object error}) => 'Falló al guardar: ${error}',
			'sessions.inspector.notes.insertFailedApi' => ({required Object error}) => 'Falló la inserción: ${error}',
			'sessions.inspector.notes.insertFailedGeneric' => ({required Object error}) => 'Falló la inserción: ${error}',
			'sessions.inspector.notes.createFailedApi' => ({required Object error}) => 'Falló al crear: ${error}',
			'sessions.inspector.notes.createFailedGeneric' => ({required Object error}) => 'Falló al crear: ${error}',
			'sessions.inspector.notes.personalHint' => 'Bloc de notas personal. Se guarda automáticamente mientras escribes. Los agentes de IA no escriben aquí.',
			'sessions.inspector.notes.projectDocsHint' => 'Arquitectura / spec / decisiones / plan / retrospectivas. Normalmente redactados o mantenidos por un agente.',
			'sessions.inspector.notes.mappingCleared' => 'Asignación borrada. Usando el valor predeterminado',
			'sessions.inspector.notes.mappedTo' => ({required Object path}) => 'Asignado a ${path}',
			'sessions.inspector.notes.cancelTooltip' => 'Cancelar',
			'sessions.inspector.notes.newDocTooltip' => 'Nuevo documento',
			'sessions.inspector.notes.noProjectMapping' => 'No se pudo resolver una asignación de proyecto para esta session. Comprueba que el gateway tenga configurado un almacén de notas y que el cwd de la session esté establecido.',
			'sessions.inspector.notes.emptyProjectDocs' => 'Aún no hay documentos del proyecto. Toca + para crear uno o deja que un agente de IA lo genere a partir de un prompt.',
			'sessions.inspector.notes.emptyFilterMatch' => ({required Object query}) => 'No hay coincidencias para "${query}".',
			'sessions.inspector.notes.locationDialogHelp' => 'Fija el cwd de esta session a una carpeta específica dentro de tu almacén de notas. Déjalo en blanco para restablecer.',
			'sessions.inspector.notes.sessionCwd' => 'cwd de la session',
			'sessions.inspector.notes.projectDocsPath' => 'Ruta de los documentos del proyecto relativa al almacén',
			_ => null,
		} ?? switch (path) {
			'sessions.inspector.notes.locationStoredHint' => 'Almacenado en <vault>/.opendray-projects.json. Se sincroniza con git junto con el resto del almacén.',
			'sessions.inspector.notes.pinnedHint' => ({required Object path, required Object defaultPath}) => 'Fijado a ${path}/ (anula ${defaultPath}). Los agentes de IA también redactan documentos aquí.',
			'sessions.inspector.notes.noProjectMapping2' => '(sin asignación de proyecto)',
			'sessions.inspector.notes.clearOverride' => 'Borrar anulación',
			'sessions.inspector.notes.save' => 'Guardar',
			'sessions.spawnSheet.title' => 'Nueva session',
			'sessions.spawnSheet.errorRequired' => 'El proveedor y el directorio de trabajo son obligatorios',
			'sessions.spawnSheet.errorGeneric' => ({required Object error}) => 'No se pudo crear la session: ${error}',
			'sessions.spawnSheet.cancel' => 'Cancelar',
			'sessions.spawnSheet.spawn' => 'Crear',
			'sessions.spawnSheet.providerLabel' => 'Proveedor',
			'sessions.spawnSheet.disabledSuffix' => ' (desactivado)',
			'sessions.spawnSheet.cwdLabel' => 'Directorio de trabajo',
			'sessions.spawnSheet.cwdHint' => '/Users/you/projects/foo',
			'sessions.spawnSheet.cwdHelper' => 'Ruta absoluta en el host del gateway.',
			'sessions.spawnSheet.browse' => 'Examinar',
			'sessions.spawnSheet.nameLabel' => 'Nombre (opcional)',
			'sessions.spawnSheet.nameHint' => 'p. ej. backend-refactor',
			'sessions.spawnSheet.argsLabel' => 'Argumentos adicionales (opcional)',
			'sessions.spawnSheet.argsHint' => '--continue --verbose',
			'sessions.spawnSheet.argsHelper' => 'Separados por espacios; en blanco usa los valores predeterminados del proveedor.',
			'sessions.spawnSheet.bypass.labelClaude' => 'Omitir permisos',
			'sessions.spawnSheet.bypass.labelCodex' => 'Omitir aprobaciones y sandbox',
			'sessions.spawnSheet.bypass.labelGemini' => 'Modo YOLO',
			'sessions.spawnSheet.bypass.labelAntigravity' => 'Omitir permisos / YOLO',
			'sessions.spawnSheet.bypass.subtitleOn' => 'Esta session se ejecutará con autonomía elevada.',
			'sessions.spawnSheet.bypass.subtitleOff' => 'Desactivado. Las confirmaciones y los prompts se comportan de forma normal.',
			'sessions.spawnSheet.noProviders.title' => 'No hay proveedores configurados',
			'sessions.spawnSheet.noProviders.message' => 'El gateway no tiene proveedores de CLI habilitados. Configura uno en Proveedores (admin web) o en [providers] en config.toml, y luego toca Recargar.',
			'sessions.spawnSheet.noProviders.reload' => 'Recargar',
			'sessions.spawnSheet.providerLoadError.title' => 'No se pudieron cargar los proveedores',
			'sessions.spawnSheet.providerLoadError.networkError' => 'Error de red',
			'sessions.spawnSheet.providerLoadError.serverPrefix' => ({required Object code}) => 'Servidor ${code}',
			'sessions.spawnSheet.providerLoadError.format' => ({required Object prefix, required Object message}) => '${prefix}: ${message}',
			'sessions.spawnSheet.claudeAccount.label' => 'Cuenta de Claude',
			'sessions.spawnSheet.claudeAccount.helperMulti' => 'Hay varias cuentas configuradas. Elige una para esta session.',
			'sessions.spawnSheet.claudeAccount.helperSingle' => 'Elige una cuenta configurada o usa la predeterminada (env / sistema).',
			'sessions.spawnSheet.claudeAccount.kDefault' => 'Predeterminada (env / sistema)',
			'sessions.spawnSheet.claudeAccount.disabledSuffix' => ' (desactivada)',
			'sessions.spawnSheet.claudeAccount.noTokenSuffix' => ' (sin token)',
			'sessions.spawnSheet.claudeAccount.noneHint' => 'No hay cuentas de Claude configuradas. El gateway usará la ANTHROPIC_API_KEY del sistema. Añade cuentas en Ajustes → Cuentas en el admin web.',
			'sessions.spawnSheet.claudeAccount.errorHint' => ({required Object error}) => 'No se pudieron cargar las cuentas de Claude (${error}). La session se creará con el valor predeterminado del gateway.',
			'mcp.title' => 'MCP',
			'mcp.newServer' => 'Nuevo servidor',
			'mcp.addSecret' => 'Añadir secreto',
			'mcp.editConfig' => 'Editar configuración',
			'mcp.viewRawConfig' => 'Ver configuración en bruto',
			'mcp.copyId' => 'Copiar id',
			'mcp.copiedSnack' => ({required Object id}) => 'Copiado ${id}',
			'mcp.deleteServerTitle' => '¿Eliminar servidor MCP?',
			'mcp.deleteSecretTitle' => '¿Eliminar secreto?',
			'mcp.errorPrefix.delete' => 'Error al eliminar',
			'mcp.errorPrefix.add' => 'Error al añadir',
			'mcp.errorPrefix.update' => 'Error al actualizar',
			'mcp.errorPrefix.toggle' => 'Error al alternar',
			'mcp.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'mcp.editor.nameHint' => 'my-mcp-server',
			'mcp.editor.jsonHint' => 'Configuración JSON, nombre, transport: stdio, command, args…',
			'mcp.editor.descriptionPlaceholder' => 'Descripción opcional de una línea',
			'mcp.editor.validateJsonObject' => 'El cuerpo debe ser un objeto JSON',
			'mcp.editor.validateJsonInvalid' => ({required Object error}) => 'JSON no válido: ${error}',
			'mcp.editor.appBarEdit' => 'Editar servidor MCP',
			'mcp.editor.appBarNew' => 'Nuevo servidor MCP',
			'mcp.editor.idLockedHint' => 'Bloqueado en modo edición, elimínalo y vuelve a crearlo para cambiarlo.',
			'mcp.editor.jsonLabel' => 'JSON del servidor',
			'mcp.editor.jsonSchemaHelp' => 'Esquema: transport debe ser stdio, http o sse. Para stdio define command + args. Para http/sse define url + headers. Usa \$secret:KEY para referenciar secretos del vault.',
			'mcp.editor.idLabel' => 'id (segmento de URL, alfanumérico en minúsculas / guion / guion bajo)',
			'mcp.editor.idRequired' => 'el id es obligatorio',
			'mcp.editor.saving' => 'Guardando…',
			'mcp.editor.save' => 'Guardar',
			'mcp.editor.create' => 'Crear',
			'mcp.secret.keyLabel' => 'Clave',
			'mcp.secret.keyHint' => 'GITHUB_TOKEN, OPENAI_KEY, …',
			'mcp.secret.valueLabel' => 'Valor',
			'mcp.secret.keyRequired' => 'La clave es obligatoria.',
			'mcp.secret.keyInvalid' => 'La clave debe coincidir con [A-Za-z_][A-Za-z0-9_]*, las mismas reglas que una variable de entorno de shell.',
			'mcp.secret.valueRequired' => 'El valor es obligatorio.',
			'mcp.secret.replaceTitle' => 'Reemplazar valor del secreto',
			'mcp.secret.addTitle' => 'Añadir secreto',
			'mcp.secret.saveButton' => 'Guardar',
			'mcp.secret.addButton' => 'Añadir',
			'mcp.secret.helpRules' => 'Reglas de variable de entorno de shell: empieza por una letra o _, después solo letras / dígitos / _.',
			'mcp.secret.replaceHint' => 'Pega el nuevo valor (el anterior se borra)',
			'mcp.secret.addHint' => 'Pega el valor del secreto',
			'mcp.secret.addedSnack' => ({required Object key}) => 'Secreto ${key} añadido.',
			'mcp.secret.updatedSnack' => ({required Object key}) => 'Secreto ${key} actualizado.',
			'mcp.secret.deletedSnack' => ({required Object key}) => 'Eliminado ${key}.',
			'mcp.secret.deleteBody' => 'Elimina el valor del vault cifrado. Cualquier servidor MCP que lo referencie fallará hasta que se restaure.',
			'mcp.popup.editConfigSubtitle' => 'Editor JSON completo, solo servidores respaldados por el vault',
			'mcp.popup.viewRawSubtitle' => 'Inspector de solo lectura para el JSON del servidor',
			'mcp.popup.deleteLabel' => 'Eliminar',
			'mcp.kv.transport' => 'Transport',
			'mcp.kv.description' => 'Descripción',
			'mcp.kv.command' => 'Command',
			'mcp.kv.args' => 'Args',
			'mcp.kv.headers' => 'Headers',
			'mcp.deleteServerBody' => ({required Object id}) => 'Elimina el directorio del vault para ${id}. Las sesiones que referencian este servidor dejan de poder iniciarlo.',
			'mcp.deleteServerSnack' => ({required Object id}) => 'Eliminado ${id}.',
			'mcp.serversCount' => ({required Object count}) => 'Servidores (${count})',
			'mcp.secretsCount' => ({required Object count}) => 'Secretos (${count})',
			'mcp.emptyServers' => 'No hay servidores MCP registrados. Toca "Nuevo servidor" para añadir uno.',
			'mcp.emptySecrets' => 'No hay secretos almacenados. Añade uno para pasar variables de entorno / headers sensibles a los servidores MCP sin ponerlos en el JSON.',
			'mcp.noVaultFileYet' => 'Aún no hay archivo de vault, los secretos añadidos lo crean.',
			'mcp.tapToReplaceHint' => 'Toca para reemplazar · mantén pulsado / papelera para eliminar',
			'mcp.failedToLoad' => 'No se pudo cargar el estado de MCP',
			'mcp.serverCreatedSnack' => 'Servidor MCP creado.',
			'mcp.serverUpdatedSnack' => 'Servidor MCP actualizado.',
			'mcp.envHeading' => 'Env',
			'mcp.encryptionAes' => 'Cifrado AES-GCM (clave en el keychain del SO)',
			'mcp.encryptionPlaintext' => 'PLAINTEXT, keychain no disponible',
			'mcp.toggleEnabledSnack' => ({required Object name}) => '${name} activado.',
			'mcp.toggleDisabledSnack' => ({required Object name}) => '${name} desactivado.',
			'mcp.builtinBadge' => 'integrado',
			'mcp.builtinAlwaysOn' => 'siempre activo',
			'mcp.builtinHint' => 'Provisto por opendray — se adjunta a cada session. No se puede editar ni eliminar.',
			'providers.title' => 'Proveedores',
			'providers.configSaved' => 'Configuración del proveedor actualizada.',
			'providers.saveFailedApi' => ({required Object error}) => 'Error al guardar: ${error}',
			'providers.saveFailedGeneric' => ({required Object error}) => 'Error al guardar: ${error}',
			'providers.reload' => 'Recargar',
			'providers.errorPrefix.toggle' => 'Error al alternar',
			'providers.errorPrefix.rename' => 'Error al renombrar',
			'providers.errorPrefix.delete' => 'Error al eliminar',
			'providers.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'providers.updateCheck.sectionTitle' => 'Versión del CLI',
			'providers.updateCheck.checking' => 'Buscando actualizaciones…',
			'providers.updateCheck.checkFailed' => 'No se pudo buscar actualizaciones',
			'providers.updateCheck.notInstalled' => 'No instalado en el host del gateway',
			'providers.updateCheck.installed' => ({required Object version}) => 'Instalado: ${version}',
			'providers.updateCheck.upToDate' => 'Actualizado',
			'providers.updateCheck.updateAvailable' => ({required Object version}) => 'Actualización disponible: ${version}',
			'providers.updateCheck.latest' => ({required Object version}) => 'última ${version}',
			'providers.updateCheck.updateButton' => 'Actualizar CLI',
			'providers.updateCheck.updating' => 'Actualizando…',
			'providers.updateCheck.updatedSnack' => ({required Object version}) => 'Actualizado a ${version}.',
			'providers.updateCheck.noChangeSnack' => 'Ya está en la última versión.',
			'providers.updateCheck.updateFailed' => ({required Object error}) => 'Actualización fallida: ${error}',
			'providers.updateCheck.notAvailableHere' => ({required Object reason}) => 'La actualización en la app no está disponible en este host: ${reason}',
			'providers.updateCheck.activeSessionsWarning' => ({required Object n}) => '${n} sesión(es) activa(s) usan este CLI — actualizar no las interrumpe, pero mantienen la versión anterior hasta reiniciarse.',
			'providers.accounts.rename' => 'Renombrar',
			'providers.accounts.renameTitle' => ({required Object name}) => 'Renombrar ${name}',
			'providers.accounts.displayNameLabel' => 'Nombre visible',
			'providers.accounts.displayNameHint' => 'Cuenta de trabajo',
			'providers.accounts.deleteTitle' => '¿Eliminar la cuenta?',
			'providers.accounts.importFailedApi' => ({required Object error}) => 'Error al importar: ${error}',
			'providers.accounts.importFailedGeneric' => ({required Object error}) => 'Error al importar: ${error}',
			'providers.accounts.enable' => 'Activar',
			'providers.accounts.disable' => 'Desactivar',
			'providers.accounts.deleteLabel' => 'Eliminar',
			'providers.accounts.deleteBody' => 'Elimina la cuenta y su token OAuth almacenado. Las sessions que ya usan esta cuenta siguen funcionando, pero la reautenticación fallará.',
			'providers.accounts.deletedSnack' => ({required Object name}) => '${name} eliminada.',
			'providers.accounts.importSyncedSnack' => 'Ya está sincronizado, el gateway no tiene cuentas nuevas.',
			'providers.accounts.importedSnackOne' => ({required Object n}) => 'Se importó ${n} cuenta.',
			'providers.accounts.importedSnackOther' => ({required Object n}) => 'Se importaron ${n} cuentas.',
			'providers.accounts.importing' => 'Sincronizando…',
			'providers.accounts.importLocal' => 'Importar local',
			'providers.accounts.addHint' => 'Añadir una cuenta nueva solo se puede hacer en el host del gateway.',
			'providers.accounts.addBody' => 'El nuevo directorio aparece aquí automáticamente. Consulta la documentación para los pasos del flujo OAuth.',
			'providers.accounts.loadFailed' => ({required Object error}) => 'Error al cargar las cuentas: ${error}',
			'providers.accounts.intro' => 'Las sessions creadas con el proveedor Claude eligen entre estas cuentas (o recurren a las variables de entorno).',
			'providers.accounts.enabledSnack' => ({required Object name}) => '${name} activada.',
			'providers.accounts.disabledSnack' => ({required Object name}) => '${name} desactivada.',
			'providers.accounts.renamedSnack' => ({required Object name}) => 'Renombrada a ${name}.',
			'providers.accounts.activeSessions' => ({required Object count}) => '${count} activas',
			'providers.accounts.usedAgo' => ({required Object when}) => 'usada ${when}',
			'providers.accounts.identityChanged' => 'La identidad cambió',
			'providers.accounts.identityWas' => ({required Object email}) => 'era ${email}',
			'providers.accounts.acceptIdentity' => 'Aceptar',
			'providers.accounts.identityAcceptedSnack' => 'Cambio de identidad aceptado',
			'providers.accounts.identityAcceptFailed' => 'Error al aceptar',
			'providers.configFallbackTitle' => 'Configuración del proveedor',
			'providers.saving' => 'Guardando…',
			'providers.save' => 'Guardar',
			'providers.configLoadFailed' => 'Error al cargar el proveedor',
			'providers.argsHelper' => 'Argumentos de CLI separados por espacios.',
			'providers.listEmptyHeadline' => 'No hay proveedores cargados.',
			'providers.listEmptyBody' => 'El gateway resuelve los proveedores desde su directorio de plugins al arrancar. Revisa los logs si esperabas alguno.',
			'providers.listLoadFailed' => 'Error al cargar los proveedores',
			'providers.cliSectionHeader' => 'Proveedores de CLI',
			'providers.enabledSnack' => ({required Object name}) => '${name} activada.',
			'providers.disabledSnack' => ({required Object name}) => '${name} desactivada.',
			'integrations.title' => 'Integraciones',
			'integrations.register' => 'Registrar',
			'integrations.registerDialogTitle' => 'Registrar integración',
			'integrations.edit' => 'Editar',
			'integrations.editTitle' => ({required Object name}) => 'Editar ${name}',
			'integrations.enabledLabel' => 'Habilitada',
			'integrations.iSavedIt' => 'Ya la he guardado',
			'integrations.apiKeyForName' => ({required Object name}) => 'API key de ${name}',
			'integrations.apiKeySubtitleRegister' => ({required Object routePrefix}) => 'Entrégasela a la integración para que pueda autenticarse contra /api/v1/${routePrefix}/…',
			'integrations.copiedRequestId' => ({required Object id}) => 'request_id ${id} copiado',
			'integrations.updateOk' => 'Integración actualizada.',
			'integrations.registerFailedApi' => ({required Object error}) => 'Error al registrar: ${error}',
			'integrations.registerFailedGeneric' => ({required Object error}) => 'Error al registrar: ${error}',
			'integrations.updateFailedApi' => ({required Object error}) => 'Error al actualizar: ${error}',
			'integrations.updateFailedGeneric' => ({required Object error}) => 'Error al actualizar: ${error}',
			'integrations.deleteTitle' => '¿Eliminar integración?',
			'integrations.deletedSnack' => ({required Object name}) => '${name} eliminada.',
			'integrations.deleteFailedApi' => ({required Object error}) => 'Error al eliminar: ${error}',
			'integrations.deleteFailedGeneric' => ({required Object error}) => 'Error al eliminar: ${error}',
			'integrations.rotateKey' => 'Rotar key',
			'integrations.rotateConfirmTitle' => '¿Rotar la API key?',
			'integrations.rotate' => 'Rotar',
			'integrations.newApiKeyTitle' => ({required Object name}) => 'Nueva API key de ${name}',
			'integrations.newApiKeySubtitle' => 'Entrégasela a la integración. La key anterior acaba de quedar invalidada.',
			'integrations.rotateFailedApi' => ({required Object error}) => 'Error al rotar: ${error}',
			'integrations.rotateFailedGeneric' => ({required Object error}) => 'Error al rotar: ${error}',
			'integrations.deleteBody' => 'Elimina el registro y revoca la API key. Las solicitudes en curso que usen la key antigua empezarán a fallar.',
			'integrations.rotateBody' => ({required Object name}) => 'Genera una nueva API key para ${name} e invalida la antigua de inmediato.',
			'integrations.appBarFallback' => 'Integración',
			'integrations.tooltipMore' => 'Más',
			'integrations.tooltipReadOnly' => 'Integración del sistema (solo lectura)',
			'integrations.kvRoutePrefix' => 'Prefijo de ruta',
			'integrations.kvBaseUrl' => 'URL base',
			'integrations.kvScopes' => 'Ámbitos',
			'integrations.kvVersion' => 'Versión',
			'integrations.kvLastHealthPing' => 'Último ping de estado',
			'integrations.kvCreated' => 'Creada',
			'integrations.kvKeyRotated' => 'Key rotada',
			'integrations.detailLoadFailed' => ({required Object error}) => 'Error al cargar la integración: ${error}',
			'integrations.callsLoadFailed' => 'Error al cargar las llamadas',
			'integrations.noMatchingCalls' => 'Aún no hay llamadas coincidentes en el registro.',
			'integrations.directionAll' => 'Todas',
			'integrations.directionInbound' => 'Entrantes',
			'integrations.directionOutbound' => 'Salientes',
			'integrations.form.validateRequired' => 'El nombre, la URL base y el prefijo de ruta son obligatorios.',
			'integrations.form.fieldName' => 'Nombre',
			'integrations.form.fieldNameHint' => 'Mi Bot',
			'integrations.form.fieldBaseUrl' => 'URL base',
			'integrations.form.fieldRoutePrefix' => 'Prefijo de ruta',
			'integrations.form.routePrefixHelper' => 'Accesible como /api/v1/<prefix>/...',
			'integrations.form.fieldScopes' => 'Ámbitos (opcional)',
			'integrations.form.scopesHelper' => 'Separados por comas. Vacío = valores por defecto del servidor.',
			'integrations.form.fieldVersion' => 'Versión (opcional)',
			'integrations.form.validateBaseUrl' => 'La URL base es obligatoria.',
			'integrations.form.editFieldScopes' => 'Ámbitos',
			'integrations.form.editScopesHelper' => 'Separados por comas.',
			'integrations.form.editFieldVersion' => 'Versión',
			'integrations.form.apiKeyWarn' => 'No volverás a ver esta key.',
			'integrations.form.copyCopied' => 'Copiado',
			'integrations.form.copyCopy' => 'Copiar',
			'integrations.emptyState' => 'Regístrala desde el admin web: Integraciones → Nueva.',
			'integrations.sectionRegistered' => 'Registradas',
			'integrations.sectionSystem' => 'Sistema',
			'integrations.listLoadFailed' => 'Error al cargar las integraciones',
			'memoryWorkers.title' => 'Workers de memoria',
			'memoryWorkers.savedSnack' => ({required Object label}) => '${label} guardado',
			'memoryWorkers.saveFailed' => ({required Object error}) => 'Error al guardar: ${error}',
			'memoryWorkers.testFailed' => ({required Object error}) => 'La llamada de prueba falló: ${error}',
			'memoryWorkers.workerLabel' => 'Worker',
			'memoryWorkers.summarizerHttp' => 'Resumidor (HTTP)',
			'memoryWorkers.agentCliPrint' => 'Agente (CLI --print)',
			'memoryWorkers.cliLabel' => 'CLI',
			'memoryWorkers.cliClaude' => 'Claude',
			'memoryWorkers.cliGemini' => 'Gemini',
			'memoryWorkers.cliCodex' => 'Codex (codex exec)',
			'memoryWorkers.cliAntigravity' => 'Antigravity (agy --print)',
			'memoryWorkers.modelLabel' => 'Modelo',
			'memoryWorkers.modelCliDefault' => 'Predeterminado del CLI (último)',
			'memoryWorkers.modelCustom' => 'Personalizado…',
			'memoryWorkers.modelCustomPlaceholder' => 'id de modelo exacto',
			'memoryWorkers.modelBackToList' => 'Lista',
			'memoryWorkers.claudeAccountLabel' => 'Cuenta de Claude',
			'memoryWorkers.claudeAccountDefault' => 'Predeterminada',
			'memoryWorkers.test' => 'Probar',
			'memoryWorkers.intro' => 'Cada punto de contacto con el LLM del sistema de memoria puede atenderse de forma independiente mediante el endpoint del resumidor local (LM Studio / compatible con OpenAI) o lanzando un agente headless de Claude / Gemini en modo --print. Las tareas narrativas de alta calidad (gitactivity, transcript) se benefician de los workers de agente; las tareas de alta frecuencia (gatekeeper) permanecen en el endpoint local por diseño.',
			'memoryWorkers.errorTitle' => 'Endpoint no accesible',
			'memoryWorkers.errorDetail' => 'Las rutas /api/v1/memory/workers son nuevas en M25. Puede que el binario de opendray necesite un reinicio para montarlas y ejecutar la migración 0029.',
			'memoryWorkers.summarizerOnlyBadge' => 'solo resumidor',
			'memoryWorkers.summarizerProviderLabel' => 'Proveedor de resumidor',
			'memoryWorkers.registryDefault' => 'Predeterminado del registro',
			'memoryWorkers.agentWarning' => 'El modo agente lanza un CLI headless por cada llamada. Latencia ~5-15s (frente a ~1s del resumidor); el coste pasa de la CPU a tu quota de Claude/Gemini.',
			'memoryWorkers.noCalls24h' => 'Sin llamadas en las últimas 24h.',
			'memoryWorkers.testOkSnack' => ({required Object label, required Object duration}) => '${label} OK, ${duration}ms',
			'memoryWorkers.testFailedReturnedSnack' => ({required Object label, required Object error}) => '${label} falló: ${error}',
			'memoryWorkers.unknownError' => 'desconocido',
			'memoryWorkers.tasks.gatekeeper.label' => 'Gatekeeper',
			'memoryWorkers.tasks.gatekeeper.description' => 'Filtro previo a la escritura en cada memory_store. Alta frecuencia (objetivo <500ms): solo resumidor.',
			'memoryWorkers.tasks.cleaner.label' => 'Bibliotecario de limpieza',
			'memoryWorkers.tasks.cleaner.description' => 'Bibliotecario LLM periódico. Juzga las memorias antiguas como conservar / obsoleta / duplicada.',
			'memoryWorkers.tasks.gitactivity.label' => 'Resumidor de actividad de git',
			'memoryWorkers.tasks.gitactivity.description' => 'git log a narrativa de 2-3 párrafos cada 24h. Encaja de forma natural con un worker de agente.',
			'memoryWorkers.tasks.transcript.label' => 'Resumidor de transcript de sesión',
			'memoryWorkers.tasks.transcript.description' => 'Resumen al final de la sesión de \'qué hizo el agente\'. Encaja de forma natural con un worker de agente.',
			'memoryWorkers.tasks.planDrift.label' => 'Detector de deriva del plan',
			'memoryWorkers.tasks.planDrift.description' => 'Después de que termina cada sesión, comprueba si el plan del proyecto necesita actualizarse y presenta una propuesta. Encaja con un worker de agente para un razonamiento más rico.',
			'memoryWorkers.tasks.conflictDetector.label' => 'Detector de conflictos entre capas',
			'memoryWorkers.tasks.conflictDetector.description' => 'Escaneo diario que encuentra contradicciones entre hechos / plan / objetivo / journal. Un modelo de mayor calidad implica menos falsos positivos.',
			'memoryWorkers.tasks.capture.label' => 'Motor de captura',
			'memoryWorkers.tasks.capture.description' => 'Extracción de hechos por cada trigger desde las transcripciones de sesión. El modo agente da hechos notablemente mejores en sesiones largas; el modo resumidor es barato y local.',
			'memoryArchived.title' => 'Memorias archivadas',
			'memoryArchived.loadFailed' => ({required Object error}) => 'Error al cargar: ${error}',
			'memoryArchived.restoreFailed' => ({required Object error}) => 'Error al restaurar: ${error}',
			'memoryArchived.emptyTitle' => 'Nada archivado',
			'memoryArchived.emptyBody' => 'No hay memorias archivadas en ningún proyecto. El limpiador automático archiva aquí los hechos obsoletos y duplicados (restaurables durante 30 días); todavía no se ha eliminado nada.',
			'memoryArchived.globalScope' => '(global)',
			'memoryArchived.countBadge' => ({required Object count}) => '${count} archivadas',
			'memoryArchived.restore' => 'Restaurar',
			'memoryArchived.restoreAll' => 'Restaurar todo',
			'memoryArchived.deleteAll' => 'Eliminar todo',
			'memoryArchived.restoreAllConfirm' => ({required Object count, required Object project}) => '¿Restaurar las ${count} memorias archivadas de ${project}?',
			'memoryArchived.deleteAllConfirm' => ({required Object count, required Object project}) => '¿Eliminar permanentemente las ${count} memorias archivadas de ${project}? Omite la ventana de gracia de 30 días y no se puede deshacer.',
			'memoryArchived.deletePermanently' => 'Eliminar',
			'memoryArchived.deleteConfirm' => '¿Eliminar permanentemente esta memoria ahora? Omite la ventana de gracia de 30 días y no se puede deshacer.',
			'memoryArchived.restoredToast' => 'Restaurada',
			'memoryArchived.restoredAllToast' => ({required Object count}) => '${count} memorias restauradas',
			'memoryArchived.deletedToast' => 'Eliminada permanentemente',
			'memoryArchived.deletedAllToast' => ({required Object count}) => '${count} memorias eliminadas',
			'memoryArchived.deleteFailed' => ({required Object error}) => 'Error al eliminar: ${error}',
			'memoryArchived.summary' => ({required Object projects, required Object memories}) => '${projects} proyectos · ${memories} archivadas',
			'project.title' => 'Proyecto',
			'project.pickFirst' => 'Elige primero un proyecto.',
			'project.health.title' => ({required Object days}) => 'Salud de la memoria, últimos ${days} días',
			'project.health.subtitle' => 'Señales agregadas de ambos subsistemas de memoria para este proyecto.',
			'project.health.newFacts' => 'Hechos nuevos',
			'project.health.newFactsHint' => ({required Object total}) => '${total} almacenados en total',
			'project.health.captureFires' => 'Capturas disparadas',
			'project.health.captureFiresHint' => ({required Object stored, required Object deduped}) => '${stored} almacenados · ${deduped} deduplicados',
			'project.health.newJournal' => 'Entradas de diario',
			'project.health.newJournalHint' => ({required Object total}) => '${total} en total',
			'project.health.planAge' => 'Última actualización del plan',
			'project.health.planAgeHint' => ({required Object count}) => '${count} propuesta(s) de desvío del plan pendiente(s)',
			'project.health.planAgeHintNone' => 'No hay propuestas de desvío del plan pendientes',
			'project.health.goalAge' => 'Última actualización del objetivo',
			'project.health.pending' => 'Propuestas pendientes',
			'project.health.pendingHint' => ({required Object days}) => 'la más antigua tiene ${days}d',
			'project.health.topHit' => ({required Object hits}) => 'Más consultado · ${hits} recuperaciones',
			'project.health.zeroHit' => ({required Object count}) => '${count} hechos con más de 7d y cero recuperaciones, candidatos para limpieza.',
			'project.health.never' => 'nunca',
			'project.health.today' => 'hoy',
			'project.health.daysAgo' => ({required Object count}) => 'hace ${count}d',
			'project.conflicts.subtitle' => 'Contradicciones que el detector diario encontró entre hechos, plan, objetivo y entradas de diario.',
			'project.conflicts.empty' => 'No hay conflictos pendientes. Pulsa Detectar ahora para un barrido bajo demanda.',
			'project.conflicts.detectNow' => 'Detectar ahora',
			'project.conflicts.detected' => ({required Object count}) => '${count} conflicto(s) nuevo(s) encontrado(s)',
			'project.conflicts.accept' => 'Aceptar',
			'project.conflicts.dismiss' => 'Descartar',
			'project.conflicts.deleteFact' => ({required Object side}) => 'Eliminar hecho ${side}',
			'project.conflicts.deleteConfirmTitle' => ({required Object side}) => '¿Eliminar hecho ${side}?',
			'project.conflicts.deleteConfirmBody' => 'Esto elimina el hecho de forma permanente y acepta el conflicto. El otro lado permanece como la afirmación superviviente.',
			'project.conflicts.deleteWillDelete' => ({required Object side}) => 'Se eliminará (lado ${side}):',
			'project.conflicts.deleteWillKeep' => ({required Object side}) => 'Se conservará (lado ${side}):',
			'project.conflicts.deleteNonFactOther' => ({required Object layer}) => '(entrada de ${layer}, abre la pestaña correspondiente para inspeccionar)',
			'project.conflicts.deleteLoading' => 'Cargando el texto del hecho…',
			'project.conflicts.deleteFactLabel' => ({required Object side}) => 'Eliminar ${side}',
			'project.conflicts.deletedFact' => 'Hecho eliminado y conflicto aceptado',
			'project.conflicts.openPlanEditor' => 'Abrir el editor del plan',
			'project.conflicts.openGoalEditor' => 'Abrir el editor del objetivo',
			'project.conflicts.severity.low' => 'baja',
			'project.conflicts.severity.medium' => 'media',
			'project.conflicts.severity.high' => 'alta',
			'project.journalPrune.title' => 'Purgar entradas de diario obsoletas',
			'project.journalPrune.subtitle' => ({required Object days}) => 'Con más de ${days} días, sin conflictos pendientes.',
			'project.journalPrune.daysLabel' => 'Con más de (días):',
			'project.journalPrune.empty' => 'No hay nada obsoleto que purgar.',
			'project.journalPrune.selectAll' => 'Seleccionar todo',
			'project.journalPrune.deselectAll' => 'Deseleccionar todo',
			'project.journalPrune.deleteSelected' => ({required Object count}) => 'Eliminar (${count})',
			'project.journalPrune.deleted' => ({required Object count}) => '${count} entrada(s) eliminada(s)',
			'project.loadFailed' => ({required Object error}) => 'Error al cargar: ${error}',
			'project.projectsLoadFailed' => ({required Object error}) => 'Error al cargar los proyectos: ${error}',
			'project.projectLabel' => 'Proyecto',
			'project.browseFolder' => 'Explorar carpeta…',
			'project.resetTooltip' => 'Restablecer la memoria del proyecto',
			'project.append' => 'Añadir',
			'project.appendDialogTitle' => 'Añadir entrada de diario',
			'project.titleFieldLabel' => 'Título (opcional)',
			'project.contentFieldLabel' => 'Contenido (markdown)',
			'project.appendFailed' => ({required Object error}) => 'Error: ${error}',
			'project.approveFailed' => ({required Object error}) => 'Error al aprobar: ${error}',
			'project.rejectFailed' => ({required Object error}) => 'Error al rechazar: ${error}',
			'project.resetConfirmTitle' => '¿Restablecer la memoria del proyecto?',
			'project.alsoDeleteScanner' => 'Eliminar también los documentos del scanner',
			'project.alsoDeletePgvector' => 'Eliminar también las memorias de pgvector',
			'project.deleteForever' => 'Eliminar para siempre',
			'project.resetDoneSnack' => ({required Object parts}) => 'Restablecido: ${parts}',
			'project.resetFailed' => ({required Object error}) => 'Error al restablecer: ${error}',
			'project.docSavedSnack' => ({required Object kind}) => '${kind} guardado',
			'project.docSaveFailed' => ({required Object error}) => 'Error al guardar: ${error}',
			'project.docHintTemplate' => ({required Object kind}) => 'Escribe el ${kind} como markdown…',
			'project.deleteEntryTooltip' => 'Eliminar entrada',
			'project.agentReason' => 'Motivo del agente',
			'project.reject' => 'Rechazar',
			'project.approve' => 'Aprobar',
			'project.replaceConfirmTitle' => ({required Object kind}) => '¿Reemplazar el ${kind} actual?',
			'project.replaceKind' => ({required Object kind}) => 'Reemplazar ${kind}',
			'project.archived.emptyTitle' => 'Nada archivado',
			'project.archived.emptyBody' => 'No hay memorias archivadas para este proyecto. El limpiador automático archiva aquí los hechos obsoletos y duplicados automáticamente; todavía ninguno.',
			'project.archived.restoreFailed' => ({required Object error}) => 'Error al restaurar: ${error}',
			'project.archived.restore' => 'Restaurar',
			'backups.title' => 'Copias de seguridad',
			'backups.runConfirmTitle' => '¿Ejecutar copia de seguridad ahora?',
			'backups.runConfirmBody' => 'Lanza un nuevo volcado contra el destino local. El trabajo se ejecuta en el servidor; esta lista se actualizará a medida que avance.',
			'backups.runFullInstance' => 'Instancia completa',
			'backups.runFullInstanceHint' => 'Incluye también el vault, secrets.env y config.toml, no solo la base de datos.',
			'backups.kindDbOnly' => 'Solo BD',
			'backups.kindFullInstance' => 'Instancia completa',
			'backups.dedupValue' => 'reutilizó el blob existente (contenido idéntico)',
			'backups.verifyOk' => 'verificada',
			'backups.verifyFailed' => 'sin verificar (falló la comprobación)',
			'backups.verifyPending' => 'sin verificar',
			'backups.run' => 'Ejecutar',
			'backups.runNow' => 'Ejecutar ahora',
			'backups.queueing' => 'Encolando…',
			'backups.queuedSnack' => ({required Object id}) => 'Copia de seguridad encolada (${id}). Esperando el progreso…',
			'backups.runFailedApi' => ({required Object error}) => 'Error al ejecutar: ${error}',
			'backups.runFailedGeneric' => ({required Object error}) => 'Error al ejecutar: ${error}',
			'backups.rowSucceededSnack' => ({required Object bytes}) => 'Copia de seguridad completada (${bytes}).',
			'backups.rowFailedSnack' => ({required Object error}) => 'Error en la copia de seguridad: ${error}',
			'backups.unknownError' => 'error desconocido',
			'backups.detailTitle' => 'Detalle de la copia de seguridad',
			'backups.deleteTitle' => '¿Eliminar copia de seguridad?',
			'backups.deleteBody' => ({required Object target}) => 'Elimina el blob de ${target} y marca la fila como eliminada en el índice.',
			'backups.deletedSnack' => ({required Object id}) => 'Eliminado ${id}.',
			'backups.deleteFailedApi' => ({required Object error}) => 'Error al eliminar: ${error}',
			'backups.deleteFailedGeneric' => ({required Object error}) => 'Error al eliminar: ${error}',
			'backups.menuSchedules' => 'Programaciones',
			'backups.menuTargets' => 'Destinos',
			'backups.kv.status' => 'Estado',
			'backups.kv.verified' => 'Verificada',
			'backups.kv.kind' => 'Tipo',
			'backups.kv.target' => 'Destino',
			'backups.kv.dedup' => 'Deduplicación',
			'backups.kv.fanout' => 'Grupo de difusión',
			'backups.kv.triggeredBy' => 'Lanzado por',
			'backups.kv.started' => 'Iniciado',
			'backups.kv.finished' => 'Finalizado',
			'backups.kv.size' => 'Tamaño',
			'backups.kv.encrypted' => 'Cifrado',
			'backups.kv.targetPath' => 'Ruta de destino',
			'backups.kv.error' => 'Error',
			'backups.kv.yes' => 'sí',
			'backups.kv.no' => 'no',
			'backups.recoveryKit.menuLabel' => 'Kit de recuperación',
			'backups.recoveryKit.title' => 'Kit de recuperación',
			'backups.recoveryKit.warning' => 'La frase de la copia de seguridad nunca se guarda en una copia. Este kit la envuelve con una frase de recuperación que tú eliges. Guarda TANTO el kit COMO la frase de recuperación en un lugar seguro y separado: sin ellos, perder el host significa copias irrecuperables.',
			'backups.recoveryKit.passphraseLabel' => 'Frase de recuperación (mín. 8)',
			'backups.recoveryKit.confirmLabel' => 'Confirmar frase de recuperación',
			'backups.recoveryKit.generate' => 'Generar',
			'backups.recoveryKit.copy' => 'Copiar kit',
			'backups.recoveryKit.copied' => 'Kit de recuperación copiado: guárdalo de forma segura',
			'backups.recoveryKit.failed' => ({required Object error}) => 'No se pudo generar el kit de recuperación: ${error}',
			'backups.emptyMissingDeps.headline' => 'Las copias de seguridad aún no pueden ejecutarse',
			'backups.emptyMissingDeps.body' => 'Instala postgresql-client y reinicia opendray.',
			'backups.emptyNoTargets.headline' => 'No hay destinos de copia de seguridad configurados',
			'backups.emptyNoTargets.body' => 'Abre el menú Más → Destinos para añadir un destino (local / S3 / SMB / SFTP / WebDAV / rclone). Luego vuelve y toca "Ejecutar ahora".',
			'backups.emptyNoBackups.headline' => 'Aún no hay copias de seguridad',
			'backups.emptyNoBackups.body' => 'Toca "Ejecutar ahora" para tomar una nueva instantánea, o abre Programaciones para configurar ejecuciones periódicas.',
			'backups.restartToActivate' => 'Reinicia opendray para activar las copias de seguridad',
			'backups.passphraseSaved' => 'Tu passphrase está guardada. El gateway solo la carga al iniciarse, así que los cambios solo surten efecto tras un reinicio.',
			'backups.keyFileLabel' => 'Archivo de clave',
			'backups.configuredViaLabel' => 'Configurado mediante',
			'backups.wizard.title' => 'Configurar copias de seguridad',
			'backups.wizard.intro' => 'Elige una passphrase maestra. opendray la usa para cifrar cada blob de copia de seguridad con AES-256-GCM. Si pierdes la passphrase, pierdes los datos: no hay forma de recuperarlos.',
			'backups.wizard.saving' => 'Guardando…',
			'backups.wizard.generateAndSave' => 'Generar y guardar',
			'backups.wizard.savePassphrase' => 'Guardar passphrase',
			'backups.wizard.generateHint' => 'El servidor genera una passphrase criptográficamente aleatoria, tú la copias a un gestor de contraseñas y luego confirmas.',
			'backups.wizard.helperRecommended' => 'Recomendado: más de 40 caracteres desde un gestor de contraseñas',
			'backups.wizard.saveNowHeader' => 'Guarda esta passphrase AHORA',
			'backups.wizard.saveNowBody' => 'Se muestra UNA SOLA VEZ. Después no podrás recuperarla desde opendray.',
			'backups.overviewTargets' => 'Destinos',
			'backups.overviewSchedules' => 'Programaciones',
			'backups.overviewBackups' => 'Copias de seguridad',
			'backups.health.headlineHealthy' => 'Copias correctas',
			'backups.health.headlineAttention' => 'Requiere atención',
			'backups.health.headlineNever' => 'Aún sin copias',
			'backups.health.lastSuccess' => 'Última copia correcta',
			'backups.health.never' => 'nunca',
			'backups.health.tiles.recentFailures' => 'Fallos recientes',
			'backups.health.tiles.verifyFailures' => 'Verificación fallida',
			'backups.health.tiles.overdue' => 'Atrasadas',
			'backups.health.tiles.schedules' => 'Programaciones',
			'backups.failedToLoad' => 'Error al cargar las copias de seguridad',
			'backups.envVarConfigured' => 'variable de entorno OPENDRAY_BACKUP_KEY',
			'backups.savedConfirmCheckbox' => 'He guardado esta passphrase en mi gestor de contraseñas',
			'backups.pgDumpMissing' => 'pg_dump no está en el PATH. Instala postgresql-client y reinicia opendray.',
			'backups.encryption.checkAgain' => 'Volver a comprobar',
			'backups.encryption.generate' => 'Generar',
			'backups.encryption.paste' => 'Pegar',
			'backups.encryption.random256bit' => 'Clave aleatoria de 256 bits',
			'backups.encryption.passphraseLabel' => 'Tu passphrase',
			'backups.encryption.passphraseHint' => 'Al menos 20 caracteres',
			'backups.encryption.passphraseCopied' => 'Passphrase copiada al portapapeles',
			'backups.restoreFromFile' => 'Restaurar desde archivo',
			'backups.restore.title' => 'Restaurar desde paquete',
			'backups.restore.subtitle' => 'Reproduce un paquete cifrado .tar.gz.enc en una base de datos Postgres. El paquete se sube desde este teléfono: elige un archivo generado por una copia de seguridad anterior.',
			'backups.restore.bundleLabel' => 'Archivo de paquete (.tar.gz.enc)',
			'backups.restore.pickFile' => 'Elegir archivo',
			'backups.restore.fileSelected' => ({required Object name, required Object size}) => '${name} · ${size}',
			'backups.restore.noFile' => 'Ningún archivo seleccionado',
			'backups.restore.targetDsnLabel' => 'DSN de Postgres de destino',
			'backups.restore.targetDsnHint' => 'Déjalo vacío para restaurar en la propia base de datos de opendray.',
			'backups.restore.targetDsnPlaceholder' => 'postgres://user:pass@host:5432/dbname',
			'backups.restore.cleanLabel' => 'pg_restore --clean --if-exists',
			'backups.restore.cleanHint' => 'Elimina los objetos existentes antes de volver a crearlos.',
			'backups.restore.auditNoteLabel' => 'Nota de auditoría (opcional)',
			'backups.restore.auditNotePlaceholder' => 'p. ej. recuperando de #INC-481',
			'backups.restore.ownDbWarning' => 'Restaurar en la PROPIA base de datos de opendray reescribirá las filas que este gateway está sirviendo actualmente. Escribe "I understand" para confirmar.',
			'backups.restore.confirmPlaceholder' => 'Escribe "I understand"',
			'backups.restore.confirmSentinel' => 'I understand',
			'backups.restore.restoring' => 'Restaurando…',
			'backups.restore.preview' => 'Vista previa (simulación)',
			'backups.restore.previewing' => 'Generando vista previa…',
			'backups.restore.previewFirstHint' => 'Ejecuta primero una vista previa en simulación',
			'backups.restore.applyRestore' => 'Aplicar restauración',
			'backups.restore.dryRunToast' => 'Simulación completada — revisa el plan y luego aplícalo',
			'backups.restore.planTitle' => 'Plan de restauración (simulación — no se cambió nada)',
			'backups.restore.planDump' => ({required Object size}) => 'Volcado de base de datos: ${size}',
			'backups.restore.planConfig' => ({required Object path}) => 'config.toml → ${path}',
			'backups.restore.planSecrets' => ({required Object path}) => 'secrets.env → ${path}',
			'backups.restore.planVault' => ({required Object files, required Object roots}) => 'vault: ${files} archivos (${roots})',
			_ => null,
		} ?? switch (path) {
			'backups.restore.planApplyHint' => 'Aplicar toma primero una instantánea de seguridad de toda la instancia, luego sobrescribe lo anterior y ejecuta pg_restore.',
			'backups.restore.succeededTitle' => 'Restauración completada',
			'backups.restore.succeededBody' => ({required Object bytes, required Object id}) => 'Se reprodujeron ${bytes} de la copia de seguridad ${id}.',
			'backups.restore.failedTitle' => 'Error en la restauración',
			'backups.restore.pickFileToast' => 'Primero elige un archivo de paquete.',
			'backups.restore.outputTitle' => 'Salida de pg_restore',
			'backups.restore.noPgRestoreOutput' => '(vacío: la restauración se completó sin salida)',
			'backups.restore.manifestTitle' => 'Manifiesto',
			'backups.restore.manifestBackupId' => 'ID de copia de seguridad',
			'backups.restore.manifestVersion' => 'Versión del manifiesto',
			'backups.restore.manifestCreatedAt' => 'Creado',
			'backups.restore.manifestPgVersion' => 'pg_version',
			'backups.restore.manifestOpendrayVersion' => 'versión de opendray',
			'backups.restore.fingerprint' => 'Huella de la clave',
			'backups.restore.fingerprintOk' => 'coincide',
			'backups.restore.fingerprintMismatch' => 'NO COINCIDE',
			'backups.restore.encryptionAlgo' => 'Cifrado',
			'backups.restore.bytesRead' => 'Bytes leídos',
			'backups.restore.targetDsnUsed' => 'DSN de destino',
			'backups.restore.targetDsnSelfLabel' => '(la propia base de datos de opendray)',
			'backups.restore.done' => 'Hecho',
			'backups.inventory.title' => 'Qué contiene una copia de seguridad',
			'backups.inventory.summary' => ({required Object rows, required Object tables}) => '${rows} filas · ${tables} tablas',
			'backups.inventory.description' => 'Recuentos de filas en vivo desde la base de datos Postgres de opendray. Las copias de seguridad capturan todas las filas de abajo; los artefactos binarios en disco no se incluyen.',
			'backups.inventory.rowsLabel' => 'filas',
			'backups.inventory.loadFailedToast' => 'Error al cargar el inventario',
			'backups.inventory.loading' => 'Cargando…',
			'backups.inventory.tap' => 'Toca para expandir',
			'backupTargets.title' => 'Destinos de copia de seguridad',
			'backupTargets.newTarget' => 'Nuevo destino',
			'backupTargets.testConnection' => 'Probar conexión',
			'backupTargets.editConfig' => 'Editar configuración',
			'backupTargets.viewRawConfig' => 'Ver configuración sin procesar',
			'backupTargets.configDialogTitle' => ({required Object kind}) => 'Configuración de ${kind}',
			'backupTargets.deleteTitle' => '¿Eliminar destino?',
			'backupTargets.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'backupSchedules.title' => 'Programaciones de copia de seguridad',
			'backupSchedules.newButton' => 'Nueva',
			'backupSchedules.deleteTitle' => '¿Eliminar programación?',
			'backupSchedules.targetLabel' => 'Destinos',
			'backupSchedules.targetsHint' => 'Elige uno o más: la misma copia se escribe en cada destino (3-2-1).',
			'backupSchedules.intervalLabel' => 'Intervalo',
			'backupSchedules.retentionLabel' => 'Retención (conservar las N más recientes)',
			'backupSchedules.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'backupSchedules.noTargets' => 'No hay destinos de copia de seguridad configurados. Añade uno desde el panel de administración web o la pantalla de Destinos.',
			'backupSchedules.okMsgCreate' => 'Programación creada.',
			'backupSchedules.okMsgUpdate' => 'Programación actualizada.',
			'backupSchedules.okMsgDelete' => 'Programación eliminada.',
			'backupSchedules.errorPrefixCreate' => 'Error al crear',
			'backupSchedules.errorPrefixUpdate' => 'Error al actualizar',
			'backupSchedules.errorPrefixDelete' => 'Error al eliminar',
			'backupSchedules.deleteBody' => ({required Object targetId}) => 'Elimina la especificación recurrente para el destino ${targetId}. Los blobs de copia de seguridad existentes no se modifican.',
			'backupSchedules.emptyList' => 'Aún no hay programaciones.\nToca "Nueva" para crear una.',
			'backupSchedules.validatePickTarget' => 'Elige un destino.',
			'backupSchedules.validateInterval' => 'El intervalo debe ser > 0.',
			'backupSchedules.formTitleEdit' => 'Editar programación',
			'backupSchedules.formTitleNew' => 'Nueva programación',
			'backupSchedules.saveButtonEdit' => 'Guardar',
			'backupSchedules.saveButtonNew' => 'Crear',
			'backupSchedules.targetFixedHint' => 'El destino queda fijado una vez creado.',
			'backupSchedules.enabledOn' => 'El programador la ejecutará según la cadencia.',
			'backupSchedules.enabledOff' => 'En pausa. No habrá ejecuciones automáticas hasta volver a activarla.',
			'backupSchedules.loadFailedTitle' => 'Error al cargar las programaciones',
			'backupSchedules.pausedBadge' => 'en pausa',
			'backupSchedules.everyInterval' => ({required Object interval}) => 'cada ${interval}',
			'backupSchedules.keepRetention' => ({required Object n}) => '· conservar ${n}',
			'backupSchedules.nextRun' => ({required Object when}) => '· siguiente ${when}',
			'backupSchedules.lastRun' => ({required Object when}) => '· última ${when}',
			'backupTargetEditor.useHttps' => 'Usar HTTPS',
			'backupTargetEditor.pathStyle' => 'Direccionamiento por ruta (path-style)',
			'backupTargetEditor.pathStyleSubtitle' => 'Heredado / MinIO',
			'backupTargetEditor.kinds.local.label' => 'Disco local',
			'backupTargetEditor.kinds.local.description' => 'Carpeta en la máquina que ejecuta opendray',
			'backupTargetEditor.kinds.smb.label' => 'Recurso compartido SMB',
			'backupTargetEditor.kinds.smb.description' => 'Recursos compartidos de Windows y la mayoría de los NAS domésticos',
			'backupTargetEditor.kinds.webdav.label' => 'WebDAV',
			'backupTargetEditor.kinds.webdav.description' => 'Nubes autoalojadas y servicios para compartir archivos',
			'backupTargetEditor.kinds.sftp.label' => 'SFTP',
			'backupTargetEditor.kinds.sftp.description' => 'Cualquier servidor accesible por SSH',
			'backupTargetEditor.kinds.s3.label' => 'S3 / compatible',
			'backupTargetEditor.kinds.s3.description' => 'Amazon S3 y buckets compatibles con S3 (MinIO, R2, B2)',
			'backupTargetEditor.kinds.rclone.label' => 'rclone (cualquiera)',
			'backupTargetEditor.kinds.rclone.description' => 'OneDrive, Google Drive, Dropbox a través de la CLI de rclone',
			'backupTargetEditor.formTitleEdit' => 'Editar destino',
			'backupTargetEditor.formTitleNew' => 'Nuevo destino de backup',
			'backupTargetEditor.idHintAuto' => ({required Object prefix}) => 'Automático: ${prefix}-1',
			'backupTargetEditor.idHelper' => 'Letras minúsculas, dígitos, guiones. Por defecto, la siguiente ranura disponible.',
			'backupTargetEditor.enabledOn' => 'Los backups programados y puntuales pueden usar este destino.',
			'backupTargetEditor.enabledOff' => 'El servidor se negará a escribir backups aquí.',
			'backupTargetEditor.saving' => 'Guardando…',
			'backupTargetEditor.create' => 'Crear',
			'backupTargetEditor.rootDirLabel' => 'Directorio raíz',
			'backupTargetEditor.rootDirHint' => 'Vacío = cfg.backup.local_dir (~/.opendray/backups)',
			'backupTargetEditor.hostLabel' => 'Host',
			'backupTargetEditor.portLabel' => 'Puerto',
			'backupTargetEditor.shareLabel' => 'Recurso compartido',
			'backupTargetEditor.shareHint' => 'Nombre del recurso compartido de nivel superior',
			'backupTargetEditor.shareSampleHint' => 'Claude_Workspace',
			'backupTargetEditor.userLabel' => 'Usuario',
			'backupTargetEditor.passwordLabel' => 'Contraseña',
			'backupTargetEditor.passwordHintKeepCurrent' => 'Déjalo en blanco para conservar la actual',
			'backupTargetEditor.passwordHintKeep' => 'Déjalo en blanco para conservarla',
			'backupTargetEditor.pathPrefixLabel' => 'Prefijo de ruta',
			'backupTargetEditor.pathPrefixHintShareRoot' => 'Subcarpeta bajo la raíz del recurso compartido (opcional)',
			'backupTargetEditor.pathPrefixHintBaseUrl' => 'Subcarpeta bajo la URL base (opcional)',
			'backupTargetEditor.pathPrefixHintObjectKey' => 'Prefijo de clave de objeto (opcional)',
			'backupTargetEditor.pathPrefixHintSshFolder' => 'Absoluta o relativa al home del usuario (opcional)',
			'backupTargetEditor.pathPrefixHintRemoteRoot' => 'Subcarpeta bajo la raíz remota (opcional)',
			'backupTargetEditor.endpointLabel' => 'Endpoint',
			'backupTargetEditor.regionLabel' => 'Región',
			'backupTargetEditor.bucketLabel' => 'Bucket',
			'backupTargetEditor.accessKeyLabel' => 'Clave de acceso',
			'backupTargetEditor.secretKeyLabel' => 'Clave secreta',
			'backupTargetEditor.secretKeyHintEdit' => 'Déjalo en blanco para conservar la actual. Se almacena cifrada con AES-256-GCM.',
			'backupTargetEditor.secretKeyHintNew' => 'Se almacena cifrada con AES-256-GCM; nunca se devuelve.',
			'backupTargetEditor.baseUrlLabel' => 'URL base',
			'backupTargetEditor.baseUrlHint' => 'URL completa incluyendo la ruta. Nextcloud: https://cloud.example/remote.php/dav/files/<user>',
			'backupTargetEditor.sftpPasswordHintEdit' => 'Déjalo en blanco para conservarla. Si están presentes tanto la contraseña como la clave privada, prevalece la clave privada.',
			'backupTargetEditor.sftpPasswordHintNew' => 'Contraseña O clave privada. Si están ambas, la contraseña pasa a ser solo un respaldo.',
			'backupTargetEditor.privateKeyLabel' => 'Clave privada (PEM)',
			'backupTargetEditor.privateKeyHintEdit' => 'Déjalo en blanco para conservarla. Pega el contenido OpenSSH/PEM.',
			'backupTargetEditor.privateKeyHintNew' => 'Pega el contenido de una clave privada OpenSSH/PEM. Entrada de varias líneas: conserva los marcadores BEGIN/END.',
			'backupTargetEditor.hostKeyLabel' => 'Clave de host (fijación)',
			'backupTargetEditor.hostKeyHint' => 'Clave pública del servidor en formato OpenSSH. Usa `ssh-keyscan <host>` para obtenerla. En blanco = sin fijación (NO recomendado fuera de la LAN).',
			'backupTargetEditor.rcloneNote' => 'Requiere la CLI de rclone en el host de opendray. Ejecuta primero `rclone config` una vez de forma interactiva para autenticar las cuentas en la nube.',
			'backupTargetEditor.rcloneRemoteLabel' => 'Nombre del remoto',
			'backupTargetEditor.rcloneRemoteHint' => 'Nombre de `rclone config` (sin los dos puntos).',
			'backupTargetEditor.rcloneBinaryLabel' => 'Ruta del binario',
			'backupTargetEditor.rcloneBinaryHint' => 'Anula `which rclone`. Vacío = búsqueda en PATH.',
			'backupTargetEditor.rcloneConfigLabel' => 'Ruta de configuración',
			'backupTargetEditor.rcloneConfigHint' => 'Anula --config. Vacío = valor por defecto de rclone.',
			'githosts.title' => 'Hosts de Git',
			'githosts.addHost' => 'Añadir host',
			'githosts.deleteTitle' => '¿Eliminar host de Git?',
			'githosts.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'githosts.errorPrefix.toggle' => 'Error al alternar',
			'githosts.errorPrefix.delete' => 'Error al eliminar',
			'githosts.form.kindLabel' => 'Tipo',
			'githosts.form.hostLabel' => 'Host',
			'githosts.form.nameLabel' => 'Nombre',
			'githosts.form.nameHint' => 'work-github, personal-gitlab, …',
			'githosts.form.kinds.github' => 'GitHub',
			'githosts.form.kinds.gitlab' => 'GitLab',
			'githosts.form.kinds.bitbucket' => 'Bitbucket',
			'githosts.form.kinds.gitea' => 'Gitea',
			'githosts.form.kinds.custom' => 'Personalizado',
			'githosts.form.validateHost' => 'El host es obligatorio.',
			'githosts.form.validateName' => 'El nombre es obligatorio.',
			'githosts.form.snackAdded' => 'Host añadido.',
			'githosts.form.snackUpdated' => 'Host actualizado.',
			'githosts.form.saveFailedApi' => ({required Object error}) => 'Error al guardar: ${error}',
			'githosts.form.saveFailedGeneric' => ({required Object error}) => 'Error al guardar: ${error}',
			'githosts.form.saving' => 'Guardando…',
			'githosts.form.save' => 'Guardar',
			'githosts.form.add' => 'Añadir',
			'githosts.form.nameHelper' => 'Nombre visible que se muestra en las listas de PR.',
			'githosts.form.tokenLabelKeep' => 'Token (déjalo en blanco para conservar el actual)',
			'githosts.form.tokenLabel' => 'Token',
			'githosts.form.tokenHintKeep' => 'Déjalo en blanco para conservar el actual.',
			'githosts.form.tokenHintNew' => 'Pega el token de acceso personal.',
			'githosts.form.enabledHelper' => 'Disponible para las sessions en búsquedas de PR / remotos.',
			'githosts.form.validateTokenRequired' => 'El token es obligatorio al añadir un host.',
			'githosts.form.appBarEdit' => ({required Object name}) => 'Editar ${name}',
			'githosts.form.appBarNew' => 'Añadir host de Git',
			'githosts.form.tokenPreviewHint' => ({required Object preview}) => 'Vista previa actual: ${preview}',
			'githosts.form.tokenPreviewNone' => '(ninguno)',
			'githosts.form.pausedSubtitle' => 'En pausa. Las sessions omiten este host.',
			'githosts.deleteBody' => ({required Object host}) => 'Elimina la credencial. Las sessions que intenten listar PRs de ${host} recurrirán a la API sin autenticar.',
			'githosts.deletedSnack' => ({required Object name}) => '${name} eliminado.',
			'githosts.enabledSnack' => ({required Object name}) => '${name} habilitado.',
			'githosts.disabledSnack' => ({required Object name}) => '${name} deshabilitado.',
			'githosts.emptyList' => 'No hay hosts de Git configurados.\n\nAñade una credencial para que el gateway pueda listar pull requests en todos tus repos.',
			'githosts.failedToLoad' => 'Error al cargar los hosts de Git',
			'channels.title' => 'Canales',
			'channels.kNew' => 'Nuevo',
			'channels.sendTest' => 'Enviar mensaje de prueba',
			'channels.editConfig' => 'Editar configuración',
			'channels.editNotifications' => 'Editar notificaciones',
			'channels.viewRawConfig' => 'Ver configuración sin procesar',
			'channels.copyChannelId' => 'Copiar id del canal',
			'channels.copiedSnack' => ({required Object id}) => 'Copiado ${id}',
			'channels.createdSnack' => ({required Object kind}) => 'Canal ${kind} creado.',
			'channels.createFailedApi' => ({required Object error}) => 'Error al crear: ${error}',
			'channels.createFailedGeneric' => ({required Object error}) => 'Error al crear: ${error}',
			'channels.deleteTitle' => '¿Eliminar canal?',
			'channels.configDialog.title' => ({required Object kind}) => 'Configuración de ${kind}',
			'channels.webhookDialog.title' => ({required Object kind}) => 'URL del webhook de ${kind}',
			'channels.webhookDialog.copiedSnack' => 'URL del webhook copiada.',
			'channels.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}: ${error}',
			'channels.notifications.title' => 'Preferencias de notificación',
			'channels.notifications.repeatPolicy' => 'Política de repetición',
			'channels.notifications.cooldownWindow' => 'Ventana de cooldown',
			'channels.notifications.includeSnippet' => 'Incluir fragmento del terminal',
			'channels.notifications.snippetLengthCap' => 'Límite de longitud del fragmento',
			'channels.notifications.snippetHelper' => 'Incrusta el final reciente del terminal en cada notificación.',
			'channels.notifications.snippetNoCap' => 'sin límite',
			'channels.notifications.snippetChars' => ({required Object n}) => '${n} caracteres',
			'channels.notifications.updatedSnack' => 'Preferencias de notificación actualizadas.',
			'channels.notifications.modes.onceLabel' => 'Una vez por session',
			'channels.notifications.modes.onceDescription' => 'Se dispara una vez al quedar inactiva, permanece en silencio hasta la respuesta o el fin.',
			'channels.notifications.modes.cooldownLabel' => 'Cooldown por ventana de tiempo',
			'channels.notifications.modes.cooldownDescription' => 'Suprime las repeticiones dentro de la ventana elegida.',
			'channels.notifications.modes.everyLabel' => 'Cada evento (ruidoso)',
			'channels.notifications.modes.everyDescription' => 'Sin supresión. Solo para canales de baja frecuencia.',
			'channels.popup.enable' => 'Activar',
			'channels.popup.disable' => 'Desactivar',
			'channels.popup.mute' => 'Silenciar',
			'channels.popup.unmute' => 'Reactivar sonido',
			'channels.popup.deleteLabel' => 'Eliminar',
			'channels.badges.running' => 'en ejecución',
			'channels.badges.starting' => 'iniciando…',
			'channels.badges.disabled' => 'desactivado',
			'channels.badges.muted' => 'silenciado',
			'channels.capsLabel' => ({required Object list}) => '· caps: ${list}',
			'channels.bridgeWebOnly' => 'Los canales bridge solo están disponibles en la web',
			'channels.bridgeEmptyAdd' => 'Añade uno desde el admin web: Canales → Nuevo.',
			'channels.deleteBody' => 'Detiene el canal y elimina su configuración. Las notificaciones en curso dirigidas a él se descartarán de forma silenciosa.',
			'channels.snacks.testDispatched' => 'Mensaje de prueba enviado.',
			'channels.snacks.channelEnabled' => 'Canal activado.',
			'channels.snacks.channelDisabled' => 'Canal desactivado.',
			'channels.snacks.channelMuted' => 'Canal silenciado.',
			'channels.snacks.channelUnmuted' => 'Sonido del canal reactivado.',
			'channels.snacks.configUpdated' => 'Configuración del canal actualizada.',
			'channels.snacks.channelDeleted' => 'Canal eliminado.',
			'channels.errorPrefix.test' => 'Error en la prueba',
			'channels.errorPrefix.toggle' => 'Error al alternar',
			'channels.errorPrefix.muteToggle' => 'Error al alternar el silencio',
			'channels.errorPrefix.update' => 'Error al actualizar',
			'channels.errorPrefix.delete' => 'Error al eliminar',
			'channels.failedToLoad' => 'Error al cargar los canales',
			'channels.kinds.telegram.description' => 'Bot mediante @BotFather. opendray hace long-polling de getUpdates y envía vía REST. Los botones y reply_to_message funcionan de forma nativa.',
			'channels.kinds.telegram.botTokenLabel' => 'Token del bot',
			'channels.kinds.telegram.botTokenHint' => 'De @BotFather. Se guarda en la configuración del canal; API solo para administradores.',
			'channels.kinds.telegram.chatIdLabel' => 'Chat ID por defecto',
			'channels.kinds.telegram.chatIdPlaceholder' => '42 (opcional, se usa cuando no hay ReplyCtx)',
			'channels.kinds.telegram.ownerUserIdsLabel' => 'ID(s) de usuario de Telegram del propietario',
			'channels.kinds.telegram.ownerUserIdsPlaceholder' => '123456789 (separados por comas para más de uno)',
			'channels.kinds.telegram.ownerUserIdsHint' => 'Solo estos IDs numéricos de usuario de Telegram pueden controlar sessions, ejecutar comandos o pulsar botones; el resto se ignora. Déjalo en blanco para permitir a cualquiera (no recomendado para chat bidireccional). Obtén el tuyo enviando un DM a @userinfobot.',
			'channels.kinds.telegram.chatEnabledLabel' => 'Chat bidireccional (enrutar mensajes a la session)',
			'channels.kinds.telegram.chatEnabledHint' => 'Cuando está activado, tus mensajes se escriben en la session seleccionada y el agente responde aquí. Desactívalo solo para notificaciones.',
			'channels.kinds.telegram.chatTypingLabel' => 'Mostrar “escribiendo…” mientras el agente trabaja',
			'channels.kinds.telegram.chatTypingHint' => 'Muestra un indicador de escritura hasta que la respuesta del agente se asienta.',
			'channels.kinds.telegram.replyMaxCharsLabel' => 'Longitud máxima de la respuesta (caracteres)',
			'channels.kinds.telegram.replyMaxCharsPlaceholder' => '3500 (en blanco = 3500, 0 = sin límite)',
			'channels.kinds.telegram.replyMaxCharsHint' => 'Limita cuánto de la respuesta del agente se envía antes de recortarla con una nota “…(truncado)”. En blanco usa el valor por defecto de 3500 (~un mensaje); pon 0 para enviar la respuesta completa, dividida en varios mensajes.',
			'channels.kinds.slack.description' => 'Socket Mode, no necesita un webhook público. Requiere un token OAuth de bot (xoxb-) y un token a nivel de app (xapp-) con connections:write.',
			'channels.kinds.slack.botTokenLabel' => 'Token del bot (xoxb-…)',
			'channels.kinds.slack.botTokenHint' => 'OAuth & Permissions → Bot User OAuth Token. Necesita chat:write.',
			'channels.kinds.slack.appTokenLabel' => 'Token a nivel de app (xapp-…)',
			'channels.kinds.slack.appTokenHint' => 'Settings → Basic Information → App-Level Tokens. Scope: connections:write.',
			'channels.kinds.slack.channelIdLabel' => 'ID de canal por defecto',
			'channels.kinds.slack.channelIdPlaceholder' => 'C0123ABC456 (opcional)',
			'channels.kinds.discord.description' => 'Bot mediante el Discord Developer Portal con MESSAGE CONTENT INTENT activado. Se conecta al Gateway WS, no requiere URL pública.',
			'channels.kinds.discord.botTokenLabel' => 'Token del bot',
			'channels.kinds.discord.botTokenPlaceholder' => 'Token del bot del Discord Developer Portal',
			'channels.kinds.discord.botTokenHint' => 'Application → Bot → Reset Token. Invita al bot con send_messages + embed_links.',
			'channels.kinds.discord.channelIdLabel' => 'ID de canal por defecto',
			'channels.kinds.discord.channelIdPlaceholder' => '123456789012345678 (opcional)',
			'channels.kinds.feishu.description' => 'Credenciales a nivel de app. Usa el webhook de suscripción de eventos para la entrada. La URL pública del webhook se genera abajo: pégala en la consola de desarrollo de Feishu.',
			'channels.kinds.feishu.afterCreateHint' => 'Abre la URL del webhook desde la tarjeta del canal y pégala en Feishu Open Platform → Event Subscriptions → Request URL.',
			'channels.kinds.feishu.appIdLabel' => 'App ID',
			'channels.kinds.feishu.appSecretLabel' => 'App secret',
			'channels.kinds.feishu.appSecretPlaceholder' => 'Secret de la credencial de la aplicación',
			'channels.kinds.feishu.verificationTokenLabel' => 'Token de verificación',
			'channels.kinds.feishu.verificationTokenHint' => 'De Event Subscriptions → Verification Token. Cuando se establece, opendray rechaza los webhooks con un token diferente.',
			'channels.kinds.feishu.chatIdLabel' => 'Chat ID por defecto (oc_…)',
			'channels.kinds.feishu.chatIdPlaceholder' => 'oc_xxxxxxxxxx (opcional)',
			'channels.kinds.dingtalk.description' => 'Robot de grupo personalizado. Solo saliente. Chat de grupo → Robots → Añadir → Modo de firma → copia el webhook + secret.',
			'channels.kinds.dingtalk.webhookUrlLabel' => 'URL del webhook',
			'channels.kinds.dingtalk.secretLabel' => 'Secret de firma',
			'channels.kinds.dingtalk.secretHint' => 'Cuando el robot está configurado en modo de seguridad "Sign", copia el secret aquí. opendray añade los parámetros timestamp + sign automáticamente.',
			'channels.kinds.wecom.description' => 'Webhook de robot de grupo. Solo saliente (texto + markdown). Configuración del grupo → Robots de grupo → Añadir → copia la URL del webhook.',
			'channels.kinds.wecom.webhookKeyLabel' => 'Clave del webhook',
			'channels.kinds.wecom.webhookKeyPlaceholder' => 'El valor de la consulta "key="',
			'channels.kinds.wecom.webhookKeyHint' => 'O pega la URL completa del webhook en el campo de abajo: cualquiera de las dos opciones es suficiente.',
			'channels.kinds.wecom.webhookUrlLabel' => 'O la URL completa del webhook',
			'onboarding.gatewayLabel' => 'URL del gateway',
			'onboarding.gatewayHint' => 'https://opendray.example.com',
			'onboarding.kContinue' => 'Continuar',
			'skills.title' => 'Skills',
			'skills.newSkill' => 'Nuevo skill',
			'skills.install' => 'Instalar SKILL.md',
			'skills.installedSnack' => ({required Object name}) => 'Instalado ${name}',
			'skills.installFailed' => ({required Object error}) => 'Instalación fallida: ${error}',
			'skills.customizingBuiltin' => ({required Object id}) => 'Personalizando ${id} integrado',
			'skills.idLabel' => 'Id (slug)',
			'skills.idHint' => 'p. ej. tdd-guide',
			'skills.bodyLabel' => 'Cuerpo (markdown)',
			'skills.loadFailedApi' => ({required Object error}) => 'Error al cargar: ${error}',
			'skills.loadFailedGeneric' => ({required Object error}) => 'Error al cargar: ${error}',
			'skills.idRequired' => 'El Id es obligatorio.',
			'skills.bodyRequired' => 'El cuerpo no puede estar vacío.',
			'skills.snackCreated' => 'Skill creado.',
			'skills.snackOverride' => 'Guardado como override del vault.',
			'skills.snackUpdated' => 'Skill actualizado.',
			'skills.saveFailedApi' => ({required Object error}) => 'Error al guardar: ${error}',
			'skills.saveFailedGeneric' => ({required Object error}) => 'Error al guardar: ${error}',
			'skills.resetTitle' => '¿Restablecer al integrado?',
			'skills.deleteTitle' => '¿Eliminar skill?',
			'skills.resetBody' => ({required Object id}) => 'Elimina el override del vault para ${id}. Las sesiones recurrirán al cuerpo integrado.',
			'skills.resetButton' => 'Restablecer',
			'skills.resetSnack' => ({required Object id}) => '${id} restablecido al integrado.',
			'skills.deletedSnack' => ({required Object id}) => '${id} eliminado.',
			'skills.deleteFailedApi' => ({required Object error}) => 'Error al eliminar: ${error}',
			'skills.deleteFailedGeneric' => ({required Object error}) => 'Error al eliminar: ${error}',
			'skills.deleteBody' => ({required Object id}) => 'Elimina ${id} del vault. Las sesiones que lo referencian fallarán hasta que se restaure.',
			'skills.newSkillTitle' => 'Nuevo skill',
			'skills.customizeTitle' => ({required Object id}) => 'Personalizar ${id}',
			'skills.editTitle' => ({required Object id}) => 'Editar ${id}',
			'skills.resetTooltip' => 'Restablecer al integrado',
			'skills.deleteTooltip' => 'Eliminar',
			'skills.saving' => 'Guardando…',
			'skills.saveOverride' => 'Guardar override',
			'skills.overrideBanner' => 'Al guardar se crea un override del vault con el mismo id. Las sesiones usarán este cuerpo en lugar del integrado hasta que lo restablezcas.',
			'skills.idHelper' => 'Letras minúsculas / dígitos / guion. Se bloquea una vez creado.',
			'skills.emptyList' => 'No hay skills configurados. El gateway incluye integrados (planner, code-reviewer, etc.).',
			'skills.failedToLoad' => 'Error al cargar los skills',
			'customTasks.title' => 'Tareas personalizadas',
			'customTasks.newTask' => 'Nueva tarea',
			'customTasks.deleteTitle' => '¿Eliminar tarea?',
			'customTasks.deletedSnack' => ({required Object name}) => '${name} eliminada.',
			'customTasks.deleteFailedApi' => ({required Object error}) => 'Error al eliminar: ${error}',
			'customTasks.deleteFailedGeneric' => ({required Object error}) => 'Error al eliminar: ${error}',
			'customTasks.popupEdit' => 'Editar',
			'customTasks.popupDelete' => 'Eliminar',
			'customTasks.nameHint' => 'p. ej. backend-tests',
			'customTasks.commandHint' => '/run pnpm test --filter backend',
			'customTasks.descriptionHint' => 'Frase breve que aparece bajo el nombre de la tarea.',
			'customTasks.scopeGlobal' => 'Global',
			'customTasks.scopeProject' => 'Proyecto',
			'customTasks.cwdHint' => '/Users/you/projects/backend',
			'customTasks.snackCreated' => 'Tarea creada.',
			'customTasks.snackUpdated' => 'Tarea actualizada.',
			'customTasks.deleteBody' => 'Quita la tarea del catálogo. Las sessions que ya la insertaron no se ven afectadas.',
			'customTasks.introBanner' => 'Define tus propios slash commands. Aparecen en el selector de tareas de la session junto a los integrados.',
			'customTasks.validateNameRequired' => 'El nombre es obligatorio',
			'customTasks.validateCommandRequired' => 'El comando es obligatorio',
			'customTasks.validateProjectCwd' => 'Las tareas con ámbito de proyecto necesitan una ruta cwd absoluta',
			'customTasks.appBarEdit' => 'Editar tarea personalizada',
			'customTasks.appBarNew' => 'Nueva tarea personalizada',
			'customTasks.fieldName' => 'Nombre',
			'customTasks.nameHelper' => 'Aparece en el selector de tareas del inspector.',
			'customTasks.fieldCommand' => 'Comando',
			'customTasks.commandHelper' => 'El texto que se inserta en la session al elegirlo. Puede ser un comando de CLI o un slash command de Claude.',
			'customTasks.fieldDescription' => 'Descripción (opcional)',
			'customTasks.fieldScope' => 'Ámbito',
			'customTasks.globalScopeHint' => 'Visible desde cualquier session, sin importar el cwd.',
			'customTasks.projectScopeHint' => 'Visible solo cuando el cwd de una session coincide con la ruta de abajo.',
			'customTasks.fieldProjectCwd' => 'cwd del proyecto',
			'customTasks.cwdHelper' => 'Ruta absoluta. Las sessions creadas con este cwd exacto verán la tarea.',
			'customTasks.saving' => 'Guardando…',
			'customTasks.save' => 'Guardar',
			'customTasks.create' => 'Crear',
			'customTasks.failedToLoad' => 'Error al cargar las tareas personalizadas',
			'notesPage.title' => 'Notas',
			'notesPage.newButton' => 'Nueva',
			'notesPage.newNoteDialogTitle' => 'Nueva nota',
			'notesPage.searchHint' => 'Busca en todo el vault…',
			'notesPage.up' => 'Arriba',
			'notesPage.copyPath' => 'Copiar ruta',
			'notesPage.open' => 'Abrir',
			'notesPage.copiedSnack' => ({required Object path}) => 'Copiado ${path}',
			'notesPage.deleteTitle' => '¿Eliminar nota?',
			'notesPage.deletedSnack' => ({required Object path}) => 'Eliminado ${path}',
			'notesPage.deleteFailedApi' => ({required Object error}) => 'Error al eliminar: ${error}',
			'notesPage.deleteFailedGeneric' => ({required Object error}) => 'Error al eliminar: ${error}',
			'notesPage.createFailedApi' => ({required Object error}) => 'Error al crear: ${error}',
			'notesPage.createFailedGeneric' => ({required Object error}) => 'Error al crear: ${error}',
			'notesPage.pathLabel' => 'Ruta relativa al vault',
			'notesPage.pathHint' => 'personal/scratch.md',
			'notesPage.create' => 'Crear',
			'notesPage.popupDelete' => 'Eliminar',
			'notesPage.deleteBody' => 'Esto es irreversible. La sincronización git del vault también eliminará el archivo en el host del gateway.',
			'notesPage.emptyFilterMatch' => ({required Object query}) => 'Ninguna nota coincide con "${query}".',
			'notesPage.emptyVault' => 'El vault está vacío. Toca + para crear tu primera nota.',
			'notesPage.emptyFolder' => ({required Object path}) => 'La carpeta "${path}" está vacía.',
			'notesPage.validatePath' => 'La ruta es obligatoria',
			'notesPage.validatePathDots' => 'La ruta no puede contener ".."',
			'notesPage.pathHelper' => 'Añade .md automáticamente si falta.',
			'notesPage.editor.markdownHint' => 'Markdown…',
			'notesPage.editor.saving' => 'Guardando…',
			'notesPage.editor.autosave' => 'Se guarda automáticamente mientras escribes',
			'notesPage.editor.loadFailedApi' => ({required Object error}) => 'Error al cargar: ${error}',
			'notesPage.editor.loadFailedGeneric' => ({required Object error}) => 'Error al cargar: ${error}',
			'notesPage.editor.saveFailedApi' => ({required Object error}) => 'Error al guardar: ${error}',
			'notesPage.editor.saveFailedGeneric' => ({required Object error}) => 'Error al guardar: ${error}',
			'notesPage.editor.savedAt' => ({required Object time}) => 'Guardado ${time}',
			'dataExport.title' => 'Exportación e importación de datos',
			'dataExport.subtitle' => 'Paquetes a nivel de usuario para migración o verificación, independientes de /backups (recuperación ante desastres).',
			'dataExport.sections.export' => 'Exportar',
			'dataExport.sections.import' => 'Importar',
			'dataExport.form.scope' => 'Alcance',
			'dataExport.form.memories' => 'Memorias',
			'dataExport.form.memoriesHint' => 'Todas las memorias persistidas y sus embeddings.',
			'dataExport.form.integrations' => 'Integraciones',
			'dataExport.form.integrationOptions.none' => 'Omitir',
			'dataExport.form.integrationOptions.noneHint' => 'No incluir el registro de /integrations.',
			'dataExport.form.integrationOptions.metadata' => 'Solo metadatos (predeterminado)',
			'dataExport.form.integrationOptions.metadataHint' => 'Nombre y endpoint por integración, sin API keys.',
			'dataExport.form.integrationOptions.plaintext' => 'Claves en texto plano',
			'dataExport.form.integrationOptions.plaintextHint' => 'PELIGROSO: incluye los API tokens en bruto. v1 solo almacena hashes bcrypt, así que hoy esto es efectivamente una operación nula; muéstralo de todos modos.',
			'dataExport.form.confirmWarning' => 'La exportación de claves en texto plano contiene secretos descifrables. Escribe "Lo entiendo" para confirmar.',
			'dataExport.form.confirmPlaceholder' => 'Escribe "Lo entiendo"',
			'dataExport.form.confirmSentinel' => 'Lo entiendo',
			'dataExport.form.customTasks' => 'Tareas personalizadas',
			'dataExport.form.customTasksHint' => 'Definiciones de tareas por usuario (programaciones cron y cuerpos de script).',
			'dataExport.form.footnote' => 'Los paquetes caducan 7 días después de su creación. El enlace de descarga es de un solo uso.',
			'dataExport.form.create' => 'Crear paquete',
			'dataExport.form.building' => 'Creando…',
			'dataExport.form.readyToast' => 'Paquete listo',
			'dataExport.form.readyDescription' => ({required Object bytes}) => '${bytes} bytes, descárgalo desde el historial de abajo.',
			'dataExport.form.failedToast' => ({required Object error}) => 'Error al crear el paquete: ${error}',
			'dataExport.history.title' => 'Historial de exportaciones',
			'dataExport.history.loading' => 'Cargando…',
			'dataExport.history.empty' => 'Aún no hay exportaciones.',
			'dataExport.history.listFailedToast' => ({required Object error}) => 'Error al cargar las exportaciones: ${error}',
			'dataExport.history.downloadFailedToast' => ({required Object error}) => 'Error al obtener el token de descarga: ${error}',
			'dataExport.history.noTokenToast' => 'Esta exportación no tiene un token de descarga utilizable (ya consumido o caducado).',
			'dataExport.history.deletedToast' => 'Exportación eliminada.',
			'dataExport.history.deleteFailedToast' => ({required Object error}) => 'Error al eliminar la exportación: ${error}',
			'dataExport.history.deleteConfirmTitle' => '¿Eliminar la exportación?',
			'dataExport.history.deleteConfirmBody' => ({required Object id}) => 'Elimina el paquete y revoca el token de descarga. ${id}',
			'dataExport.history.download' => 'Descargar',
			'dataExport.history.delete' => 'Eliminar',
			'dataExport.history.downloadCopiedToast' => 'URL de descarga copiada al portapapeles. Pégala en un navegador para obtenerla (un solo uso).',
			'dataExport.history.columns.scope' => 'Alcance',
			'dataExport.history.columns.size' => 'Tamaño',
			'dataExport.history.columns.expires' => 'Caduca',
			'dataExport.history.columns.actions' => 'Acciones',
			'dataExport.history.scopeEmpty' => '(vacío)',
			'dataExport.history.scopeMemories' => 'memorias',
			'dataExport.history.scopeIntegrations' => ({required Object mode}) => 'integraciones(${mode})',
			'dataExport.history.scopeCustomTasks' => 'custom_tasks',
			'dataExport.import.intro' => 'Reproduce un paquete generado previamente por Exportar. Solo se importan las entidades que marques abajo; todo lo demás en el paquete se ignora.',
			'dataExport.import.bundleLabel' => 'Archivo de paquete (.zip)',
			'dataExport.import.pickFile' => 'Elegir archivo',
			'dataExport.import.fileSelected' => ({required Object name, required Object size}) => '${name} · ${size}',
			'dataExport.import.noFile' => 'Ningún archivo seleccionado',
			'dataExport.import.memoriesLabel' => 'Memorias',
			'dataExport.import.integrationsLabel' => 'Integraciones',
			'dataExport.import.customTasksLabel' => 'Tareas personalizadas',
			'dataExport.import.importBundle' => 'Importar paquete',
			'dataExport.import.importing' => 'Importando…',
			'dataExport.import.pickFileToast' => 'Elige primero un archivo de paquete.',
			'dataExport.import.doneToast' => 'Importación completada',
			'dataExport.import.finishedWithErrors' => 'Importación finalizada con errores',
			'dataExport.import.failedToast' => ({required Object error}) => 'Error en la importación: ${error}',
			'dataExport.import.summaryCard.memories' => 'Memorias',
			'dataExport.import.summaryCard.integrations' => 'Integraciones',
			'dataExport.import.summaryCard.customTasks' => 'Tareas personalizadas',
			'dataExport.import.summaryCard.created' => 'creadas',
			'dataExport.import.summaryCard.skipped' => 'omitidas',
			'dataExport.import.summaryCard.failed' => 'fallidas',
			'dataExport.imports.title' => 'Historial de importaciones',
			'dataExport.imports.loading' => 'Cargando…',
			'dataExport.imports.empty' => 'Aún no hay importaciones.',
			'dataExport.imports.listFailedToast' => ({required Object error}) => 'Error al cargar las importaciones: ${error}',
			'dataExport.imports.noneCounts' => '(sin recuentos)',
			'dataExport.imports.sourceUnknown' => '(origen desconocido)',
			'dataExport.imports.columns.id' => 'ID',
			'dataExport.imports.columns.status' => 'Estado',
			'dataExport.imports.columns.source' => 'Origen',
			'dataExport.imports.columns.counts' => 'Recuentos',
			'dataExport.imports.columns.when' => 'Cuándo',
			'dataExport.relative.inSeconds' => ({required Object n}) => 'en ${n}s',
			'dataExport.relative.inMinutes' => ({required Object n}) => 'en ${n}m',
			'dataExport.relative.inHours' => ({required Object n}) => 'en ${n}h',
			'dataExport.relative.inDays' => ({required Object n}) => 'en ${n}d',
			'dataExport.relative.secondsAgo' => ({required Object n}) => 'hace ${n}s',
			'dataExport.relative.minutesAgo' => ({required Object n}) => 'hace ${n}m',
			'dataExport.relative.hoursAgo' => ({required Object n}) => 'hace ${n}h',
			'dataExport.status.pending' => 'pendiente',
			'dataExport.status.running' => 'en ejecución',
			'dataExport.status.ready' => 'listo',
			'dataExport.status.failed' => 'fallido',
			'dataExport.status.expired' => 'caducado',
			'dataExport.status.succeeded' => 'completado',
			'memory.status.label' => 'Embedder activo',
			'memory.status.dimensions' => ({required Object dim, required Object state}) => '${dim}-dim · ${state}',
			'memory.status.enabled' => 'habilitado',
			'memory.status.disabled' => 'deshabilitado',
			'memory.status.floorNoModel' => 'Solo recuperación por palabras clave (BM25) — no hay modelo de embedding configurado. Configura un endpoint denso en Settings para habilitar la memoria semántica.',
			'memory.status.denseConfiguredPendingRestart' => ({required Object model}) => 'Configurado ${model} (denso) — reinicia el gateway para activar la memoria semántica y re-embeber las memorias existentes.',
			'memory.status.denseUnreachableFloor' => ({required Object model}) => 'Configurado ${model} (denso) pero el endpoint está inalcanzable — se usa el piso de palabras clave hasta que responda (se actualiza al reiniciar).',
			'memory.status.denseDegraded' => 'Embedder denso activo pero su endpoint está inalcanzable ahora — los vectores existentes se conservan; las nuevas escrituras y la búsqueda por similitud se pausan hasta que responda.',
			'memory.title' => 'Memoria',
			'memory.more' => 'Más',
			'memory.workers' => 'Workers de memoria',
			'memory.rank.title' => 'Desglose del ranking',
			'memory.rank.effective' => ({required Object value}) => 'Puntuación efectiva: ${value}',
			'memory.rank.similarity' => 'Similitud del coseno',
			'memory.rank.ageMultiplier' => ({required Object days}) => 'Multiplicador por antigüedad (${days}d de antigüedad)',
			'memory.rank.hitMultiplier' => ({required Object hits}) => 'Multiplicador por número de hits (${hits} hits)',
			'memory.rank.confidenceMultiplier' => 'Multiplicador por confianza',
			'memory.rank.formula' => 'effective = similarity × age × hits × confidence',
			'memory.rank.close' => 'Cerrar',
			'memory.kNew' => 'Nuevo',
			'memory.searchHint' => 'Buscar…',
			'memory.projectLabel' => 'Proyecto',
			'memory.filterHint' => 'Filtrar por nombre o ruta…',
			'memory.copied' => 'Copiado',
			'memory.copyTooltip' => 'Copiar texto',
			'memory.deleteAllConfirm.title' => '¿Eliminar todas las memorias de este ámbito?',
			'memory.deleteAllConfirm.deleteAll' => 'Eliminar todas',
			'memory.deletedSnackOne' => ({required Object n}) => 'Se eliminó ${n} elemento de memoria',
			'memory.deletedSnackOther' => ({required Object n}) => 'Se eliminaron ${n} elementos de memoria',
			'memory.bulkDeleteFailedApi' => ({required Object error}) => 'Error al eliminar en bloque: ${error}',
			'memory.bulkDeleteFailedGeneric' => ({required Object error}) => 'Error al eliminar en bloque: ${error}',
			'memory.deleteOne.title' => '¿Eliminar memoria?',
			'memory.deleteOne.body' => 'Esto no se puede deshacer.',
			'memory.scope.project' => 'Proyecto',
			'memory.scope.global' => 'Global',
			_ => null,
		} ?? switch (path) {
			'memory.create.textLabel' => 'Texto',
			'memory.create.scopeKeyLabel' => 'Clave de ámbito (cwd del proyecto)',
			'memory.create.scopeKeyHint' => '/Users/you/projects/foo',
			'memory.create.submit' => 'Crear',
			'memory.archive' => 'Archivar',
			'memory.quarantine' => 'Cuarentena',
			'memory.archivedToast' => 'Memoria archivada — restaurable desde Archivado',
			'memory.quarantinedToast' => 'Memoria en cuarentena — revísala en Cortex → Cuarentena',
			'memory.archiveFailed' => ({required Object error}) => 'Error al archivar: ${error}',
			'memory.quarantineFailed' => ({required Object error}) => 'Error al poner en cuarentena: ${error}',
			'memory.reembed.menuItem' => 'Reincrustar todo',
			'memory.reembed.confirmTitle' => '¿Reincrustar todas las memorias?',
			'memory.reembed.confirmBody' => 'Recodifica cada memoria y página de KB con el modelo de embedding actual. Necesario tras cambiar de modelo, ya que la dimensión del vector cambia. Puede tardar un rato.',
			'memory.reembed.confirmButton' => 'Reincrustar',
			'memory.reembed.running' => 'Reincrustando… esto puede tardar.',
			'memory.reembed.done' => ({required Object count}) => 'Se reincrustaron ${count} memorias.',
			'memory.reembed.failed' => ({required Object error}) => 'Falló la reincrustación: ${error}',
			'about.title' => 'Acerca de',
			'about.loading' => 'Cargando…',
			'about.sections.app' => 'App',
			'about.sections.server' => 'Servidor',
			'about.sections.gateway' => 'Gateway',
			'about.fields.app' => 'App',
			'about.fields.version' => 'Versión',
			'about.fields.versionFormat' => ({required Object version, required Object build}) => '${version} (build ${build})',
			'about.fields.package' => 'Paquete',
			'about.fields.url' => 'URL',
			'about.fields.signedInAs' => 'Sesión iniciada como',
			'about.fields.tokenExpires' => 'El token caduca',
			'about.copied' => ({required Object label}) => '${label} copiado',
			'about.copyTooltip' => 'Copiar',
			'about.copyLabels.version' => 'versión',
			'about.copyLabels.serverUrl' => 'URL del servidor',
			'about.tagline' => 'opendray móvil, control del gateway multi-CLI.\nFuente: github.com/Opendray/opendray',
			'about.gateway.version' => 'Versión',
			'about.gateway.commit' => 'Commit',
			'about.gateway.checking' => 'Buscando actualizaciones…',
			'about.gateway.upToDate' => 'Actualizado',
			'about.gateway.updateAvailable' => ({required Object version}) => 'Actualización disponible: ${version}',
			'about.gateway.releaseNotes' => 'Notas de la versión',
			'about.gateway.checkFailed' => 'Comprobación de actualizaciones no disponible',
			'settings.title' => 'Ajustes',
			'settings.language.section' => 'Idioma',
			'settings.language.system' => 'Sistema',
			'settings.language.systemSubtitle' => 'Sigue la configuración de idioma de tu teléfono',
			'settings.language.english' => 'English',
			'settings.language.chinese' => '中文',
			'settings.language.spanish' => 'Español',
			'settings.appearance.section' => 'Apariencia',
			'settings.appearance.system' => 'Sistema',
			'settings.appearance.systemSubtitle' => 'Sigue la configuración de apariencia de tu teléfono',
			'settings.appearance.light' => 'Claro',
			'settings.appearance.lightSubtitle' => 'Usa siempre la paleta clara',
			'settings.appearance.dark' => 'Oscuro',
			'settings.appearance.darkSubtitle' => 'Usa siempre la paleta oscura',
			'settings.account.section' => 'Cuenta',
			'settings.account.changeCredentials' => 'Cambiar credenciales',
			'settings.account.changeCredentialsSubtitle' => 'Usuario y contraseña',
			'settings.gateway.section' => 'Gateway',
			'settings.gateway.serverSettings' => 'Ajustes del servidor',
			'settings.gateway.serverSettingsSubtitle' => 'Dirección de escucha, registro, vault, memoria, rutas de almacenamiento…',
			'settings.gateway.liveLogs' => 'Logs en vivo',
			'settings.gateway.liveLogsSubtitle' => 'Sigue el búfer de logs del gateway (la misma fuente que el panel web)',
			'settings.changeCredentials.title' => 'Cambiar credenciales',
			'settings.changeCredentials.explanation' => 'Verifica tu contraseña actual y luego elige nuevas credenciales. Todas las demás sessions con sesión iniciada se revocarán.',
			'settings.changeCredentials.currentPassword' => 'Contraseña actual',
			'settings.changeCredentials.newUsername' => 'Nuevo usuario',
			'settings.changeCredentials.newPassword' => 'Nueva contraseña',
			'settings.changeCredentials.confirmPassword' => 'Confirma la nueva contraseña',
			'settings.changeCredentials.validatorRequired' => 'Obligatorio',
			'settings.changeCredentials.passwordHelper' => 'Al menos 8 caracteres',
			'settings.changeCredentials.passwordTooShort' => 'Debe tener al menos 8 caracteres',
			'settings.changeCredentials.passwordMismatch' => 'No coincide con la nueva contraseña',
			'settings.changeCredentials.updatedSnack' => 'Credenciales actualizadas.',
			'settings.changeCredentials.wrongCurrent' => 'La contraseña actual es incorrecta.',
			'settings.changeCredentials.saving' => 'Guardando…',
			'settings.changeCredentials.update' => 'Actualizar',
			'settings.logViewer.title' => 'Logs en vivo',
			'settings.logViewer.reconnect' => 'Reconectar',
			'settings.logViewer.copyBuffer' => 'Copiar búfer',
			'settings.logViewer.clearLocal' => 'Borrar vista local',
			'settings.logViewer.copiedSnack' => 'Búfer copiado al portapapeles',
			'settings.logViewer.filterHint' => 'Filtrar subcadena…',
			'settings.logViewer.levels.all' => 'Todos',
			'settings.logViewer.levels.debug' => 'Debug',
			'settings.logViewer.levels.info' => 'Info',
			'settings.logViewer.levels.warn' => 'Warn',
			'settings.logViewer.levels.error' => 'Error',
			'settings.serverSettings.title' => 'Ajustes del servidor',
			'settings.serverSettings.reloadTooltip' => 'Recargar desde el servidor',
			'settings.serverSettings.restartTooltip' => 'Reiniciar gateway',
			'settings.serverSettings.restartConfirmTitle' => '¿Reiniciar opendray?',
			'settings.serverSettings.restartConfirmBody' => 'El gateway se ejecutará de nuevo a sí mismo. La app móvil puede perder la conexión brevemente.',
			'settings.serverSettings.restart' => 'Reiniciar',
			'settings.serverSettings.restartQueuedSnack' => 'Reinicio solicitado. Desliza para actualizar en un momento.',
			'settings.serverSettings.restartFailedApi' => ({required Object error}) => 'Falló el reinicio: ${error}',
			'settings.serverSettings.restartFailedGeneric' => ({required Object error}) => 'Falló el reinicio: ${error}',
			'settings.serverSettings.loadedFrom' => ({required Object path}) => 'Cargado desde: ${path}',
			'settings.serverSettings.restartHint' => 'La mayoría de las secciones necesitan un reinicio del gateway para surtir efecto. El botón de reinicio está en la AppBar.',
			'settings.serverSettings.savedNeedsRestart' => 'Guardado. Reinicia el gateway para aplicar.',
			'settings.serverSettings.savedSimple' => 'Guardado.',
			'settings.serverSettings.changesNeedRestart' => 'Los cambios en esta sección necesitan un reinicio del gateway.',
			'settings.serverSettings.loadFailed' => 'No se pudieron cargar los ajustes del servidor',
			'settings.serverSettings.sections.general' => 'General',
			'settings.serverSettings.sections.logging' => 'Registro',
			'settings.serverSettings.sections.sessions' => 'Sessions',
			'settings.serverSettings.sections.vault' => 'Vault',
			'settings.serverSettings.sections.mcpRegistry' => 'Registro de MCP',
			'settings.serverSettings.sections.memory' => 'Memoria',
			'settings.serverSettings.sections.backup' => 'Copia de seguridad',
			'settings.serverSettings.sections.storageClaude' => 'Almacenamiento · Claude',
			'settings.serverSettings.sections.storageCodex' => 'Almacenamiento · Codex',
			'settings.serverSettings.sections.storageGemini' => 'Almacenamiento · Gemini',
			'settings.serverSettings.sections.storageAntigravity' => 'Almacenamiento · Antigravity',
			'settings.serverSettings.sectionDescriptions.general' => 'Dirección de escucha, cuenta del operador, TTL del token.',
			'settings.serverSettings.sectionDescriptions.logging' => 'Verbosidad, formato y ruta del log en disco.',
			'settings.serverSettings.sectionDescriptions.sessions' => 'Umbrales de detección de inactividad.',
			'settings.serverSettings.sectionDescriptions.vault' => 'Notas, skills y raíz versionada con git.',
			'settings.serverSettings.sectionDescriptions.mcpRegistry' => 'Rutas del vault para servidores MCP + archivo de secretos.',
			'settings.serverSettings.sectionDescriptions.memory' => 'Subsistema de memoria persistente entre CLIs.',
			'settings.serverSettings.sectionDescriptions.backup' => 'Copias de seguridad cifradas de la BD + exportaciones de datos de admin. La frase de contraseña vive en el keyfile (Ajustes → Copias de seguridad).',
			'settings.serverSettings.sectionDescriptions.storageClaude' => 'Dónde viven los transcripts de Claude en disco.',
			'settings.serverSettings.sectionDescriptions.storageCodex' => 'Raíz de las sessions de Codex.',
			'settings.serverSettings.sectionDescriptions.storageGemini' => 'Rutas de tmp por proyecto + projects.json.',
			'settings.serverSettings.sectionDescriptions.storageAntigravity' => 'Almacén SQLite por conversación de agy.',
			'settings.serverSettings.fields.listenAddress' => 'Dirección de escucha',
			'settings.serverSettings.fields.adminUser' => 'Usuario admin',
			'settings.serverSettings.fields.adminUserHelper' => 'Efectivo cuando no hay keyfile ni variable de entorno configurada. Si no, consulta Ajustes → Cuenta.',
			'settings.serverSettings.fields.adminPassword' => 'Contraseña admin',
			'settings.serverSettings.fields.adminPasswordHelper' => 'Envíalo en blanco para conservarlo. Para rotaciones continuas usa Ajustes → Cuenta (respaldado por keyfile, sin reinicio).',
			'settings.serverSettings.fields.tokenTtlWeb' => 'TTL del token (web)',
			'settings.serverSettings.fields.tokenTtlHelper' => 'Cadena de duración de Go, p. ej. 24h, 30m.',
			'settings.serverSettings.fields.level' => 'Nivel',
			'settings.serverSettings.fields.format' => 'Formato',
			'settings.serverSettings.fields.filePath' => 'Ruta del archivo',
			'settings.serverSettings.fields.filePathHelper' => 'Vacío = solo stdout.',
			'settings.serverSettings.fields.idleThreshold' => 'Umbral de inactividad',
			'settings.serverSettings.fields.idleThresholdHelper' => 'Periodo de silencio antes de marcar una session como inactiva. Duración de Go.',
			'settings.serverSettings.fields.idleCheckInterval' => 'Intervalo de comprobación de inactividad',
			'settings.serverSettings.fields.idleCheckHelper' => 'Con qué frecuencia se ejecuta el reaper de inactividad.',
			'settings.serverSettings.fields.root' => 'Raíz',
			'settings.serverSettings.fields.rootHelper' => 'Padre de las sub-rutas notes / skills / git_root.',
			'settings.serverSettings.fields.notesPath' => 'Ruta de notas',
			'settings.serverSettings.fields.skillsPath' => 'Ruta de skills',
			'settings.serverSettings.fields.gitRoot' => 'Raíz de git',
			'settings.serverSettings.fields.personalPrefix' => 'Prefijo personal',
			'settings.serverSettings.fields.projectsPrefix' => 'Prefijo de proyectos',
			'settings.serverSettings.fields.registryRoot' => 'Raíz del registro',
			'settings.serverSettings.fields.secretsFile' => 'Archivo de secretos',
			'settings.serverSettings.fields.backend' => 'Backend',
			'settings.serverSettings.fields.store' => 'Almacén',
			'settings.serverSettings.fields.defaultTopK' => 'Top-k por defecto',
			'settings.serverSettings.fields.similarityThreshold' => 'Umbral de similitud',
			'settings.serverSettings.fields.defaultScope' => 'Ámbito por defecto',
			'settings.serverSettings.fields.preserveHelper' => 'En blanco para conservar el actual.',
			'settings.serverSettings.fields.localModelName' => 'Nombre del modelo local',
			'settings.serverSettings.fields.localLibraryPath' => 'Ruta de la biblioteca local',
			'settings.serverSettings.fields.localModelPath' => 'Ruta del modelo local',
			'settings.serverSettings.fields.localTokenizerPath' => 'Ruta del tokenizador local',
			'settings.serverSettings.fields.localMaxSeqLen' => 'Longitud máx. de secuencia local',
			'settings.serverSettings.fields.backupEnabled' => 'Habilitado',
			'settings.serverSettings.fields.backupEnabledHelper' => 'Aunque esto esté activado, el subsistema de copias de seguridad permanece apagado hasta que se configure OPENDRAY_BACKUP_KEY o el keyfile.',
			'settings.serverSettings.fields.backupLocalDir' => 'Directorio local',
			'settings.serverSettings.fields.backupExportDir' => 'Directorio de exportación',
			'settings.serverSettings.fields.pathHelper' => 'Vacío = resolver desde PATH al arrancar.',
			'settings.serverSettings.fields.accountsDir' => 'Directorio de cuentas',
			'settings.serverSettings.fields.accountsHelper' => 'Padre de los subdirectorios .claude/ por cuenta. Vacío = ~/.claude-accounts.',
			'settings.serverSettings.fields.sessionsRoot' => 'Raíz de sessions',
			'settings.serverSettings.fields.sessionsRootHelper' => 'Vacío = ~/.codex/sessions.',
			'settings.serverSettings.fields.listenHelper' => 'host:port al que se vincula el gateway. Requiere reinicio.',
			'settings.serverSettings.fields.secretsHelper' => 'Vault de secretos cifrado con AES-256-GCM.',
			'settings.serverSettings.fields.backendHelper' => 'auto elige el mejor disponible; local necesita ONNX.',
			'settings.serverSettings.fields.similarityHelper' => '0.0-1.0; los resultados por debajo de esto se filtran.',
			'settings.serverSettings.fields.defaultFallback' => ({required Object value}) => 'Por defecto: ${value}',
			'settings.serverSettings.fields.httpBaseUrl' => 'URL base HTTP',
			'settings.serverSettings.fields.httpModel' => 'Modelo HTTP',
			'settings.serverSettings.fields.httpApiKey' => 'API key HTTP',
			'settings.serverSettings.fields.httpDimensions' => 'Dimensiones HTTP',
			'settings.serverSettings.fields.pgDumpPath' => 'Ruta de pg_dump',
			'settings.serverSettings.fields.pgRestorePath' => 'Ruta de pg_restore',
			'settings.serverSettings.fields.tmpRoot' => 'Raíz de tmp',
			'settings.serverSettings.fields.projectsJson' => 'projects.json',
			'settings.serverSettings.fields.conversationsRoot' => 'Directorio de conversaciones',
			'settings.serverSettings.fields.dedupThreshold' => 'Umbral de dedup',
			'settings.serverSettings.fields.dedupHelper' => 'Umbral de plegado al escribir; 0 = por defecto, negativo desactiva.',
			'settings.serverSettings.fields.gatekeeperEnabled' => 'Gatekeeper',
			'settings.serverSettings.fields.gatekeeperHelper' => 'Juez LLM pre-escritura para memory_store. Provider en ajustes de Cortex.',
			'settings.serverSettings.fields.cleanerEnabled' => 'Cleaner',
			'settings.serverSettings.fields.cleanerHelper' => 'Auto-bibliotecario periódico que archiva memorias obsoletas / duplicadas.',
			'settings.serverSettings.fields.knowledgeEnabled' => 'Grafo de conocimiento',
			'settings.serverSettings.fields.knowledgeHelper' => 'La capa estructurada de entidades/playbooks/skills sobre la memoria.',
			'settings.serverSettings.validateInteger' => ({required Object field}) => '"${field}" debe ser un entero',
			'settings.serverSettings.validateNumber' => ({required Object field}) => '"${field}" debe ser un número',
			'settings.serverSettings.embedderModel.reprobe' => 'Volver a comprobar el endpoint',
			'settings.serverSettings.embedderModel.unreachable' => 'Endpoint no accesible — escribe el id del modelo a mano.',
			'settings.serverSettings.embedderModel.pickHint' => 'Selecciona un modelo',
			'settings.serverSettings.embedderModel.manual' => 'Escribir manualmente',
			'settings.serverSettings.embedderModel.pickFromList' => 'Elegir de la lista',
			'memoryQuarantine.title' => 'Cuarentena',
			'memoryQuarantine.subtitle' => 'Hechos que necesitan revisión antes de contar como memoria durable: las capturas de integraciones llegan aquí por política, y puedes poner cualquier memoria en cuarentena a mano. Promueve lo verdadero; descarta el resto — las filas sin revisar expiran solas.',
			'memoryQuarantine.empty' => 'Nada en cuarentena.',
			'memoryQuarantine.loadFailed' => ({required Object error}) => 'Error al cargar: ${error}',
			'memoryQuarantine.promote' => 'Promover',
			'memoryQuarantine.discard' => 'Descartar',
			'memoryQuarantine.promotedToast' => 'Promovida a memoria durable',
			'memoryQuarantine.discardedToast' => 'Descartada',
			'memoryQuarantine.actionFailed' => ({required Object error}) => 'La acción falló: ${error}',
			'memoryQuarantine.expires' => ({required Object date}) => 'expira ${date}',
			'memoryQuarantine.countBadge' => ({required Object count}) => '${count} pendientes',
			'cortexHub.title' => 'Cortex',
			'cortexHub.subtitle' => 'El volante de experiencia: Memoria → Notas → Conocimiento, realimentado en cada session.',
			'cortexHub.idleBadge' => ({required Object days}) => 'inactivo ${days}d',
			'cortexHub.activeProjectsBadge' => ({required Object count}) => '${count} activos',
			'cortexHub.activeProjectsTitle' => 'Proyectos activos',
			'cortexHub.loopHint' => 'Las sesiones alimentan la Memoria → la Memoria se destila en Notas → las Notas se compilan en Conocimiento → el Conocimiento guía cada nueva sesión.',
			'cortexHub.settings' => 'Ajustes',
			'cortexHub.memory' => 'Memoria',
			'cortexHub.memoryDesc' => 'Hechos crudos entre sessions que los agentes guardan y recuerdan.',
			'cortexHub.notes' => 'Notas',
			'cortexHub.notesDesc' => 'El objetivo / plan / diario oficial de cada proyecto.',
			'cortexHub.knowledge' => 'Conocimiento',
			'cortexHub.knowledgeDesc' => 'Experiencia destilada entre proyectos.',
			'cortexHub.quarantineBadge' => ({required Object count}) => '${count} por revisar',
			'cortexHub.pendingBadge' => ({required Object count}) => '${count} pendientes',
			'cortexHub.disabled' => 'desactivado',
			'cortexHub.inboxTitle' => ({required Object count}) => 'Propuestas pendientes (${count})',
			'cortexHub.inboxHint' => 'Actualizaciones propuestas por la IA para notas y páginas KB. Aprueba para publicar, rechaza para descartar.',
			'cortexHub.kbLabel' => 'Base de conocimiento',
			'cortexHub.preview' => 'Vista previa',
			'cortexHub.hide' => 'Ocultar',
			'cortexHub.approve' => 'Aprobar',
			'cortexHub.reject' => 'Rechazar',
			'cortexHub.approvedToast' => 'Propuesta aprobada',
			'cortexHub.rejectedToast' => 'Propuesta rechazada',
			'cortexHub.actionFailed' => ({required Object error}) => 'La acción falló: ${error}',
			'cortexHub.loadFailed' => ({required Object error}) => 'Error al cargar: ${error}',
			'cortexSettings.title' => 'Ajustes de Cortex',
			'cortexSettings.tabWorkers' => 'Workers',
			'cortexSettings.tabCapture' => 'Captura e inyección',
			'cortexSettings.tabProviders' => 'Proveedores',
			'cortexSettings.providersHint' => 'Endpoints LLM a los que enrutan los workers de resumen/agente.',
			'cortexSettings.providersEmpty' => 'Sin proveedores configurados.',
			'cortexSettings.providersManageOnWeb' => 'Añade o edita proveedores en el panel web.',
			'cortexSettings.providersLoadFailed' => 'Error al cargar proveedores',
			'cortexSettings.defaultBadge' => 'predeterminado',
			_ => null,
		};
	}
}
