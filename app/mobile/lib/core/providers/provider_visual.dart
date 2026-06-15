// Provider visual identity — single source of truth so the
// sessions list, session detail, and any future surface (push
// notifications, activity feed) show the same brand mark, tint
// and fallback letter for a given provider id.
//
// Mirrors the web admin's app/shared/src/lib/providers.ts +
// providerIcons.ts. Keep the maps in sync when a new provider
// ships so the two clients stay visually consistent.

import 'package:flutter/material.dart';

class ProviderVisual {
  const ProviderVisual({
    required this.iconAsset,
    required this.brandColor,
    required this.label,
    required this.fallbackLetter,
  });

  /// Asset path under `assets/provider_icons/`, or null if no
  /// curated SVG is bundled for this provider (caller renders the
  /// fallback letter inside a neutral tile).
  final String? iconAsset;

  /// Canonical brand colour. Used as the tile background tint and
  /// the SVG colorFilter (when the source SVG is a single-fill
  /// monochrome mark like Claude / Shell).
  final Color brandColor;

  /// Human-readable provider name (matches the web admin's
  /// providerVisual.name).
  final String label;

  /// Single uppercase character used inside the tile when no
  /// brand SVG is registered.
  final String fallbackLetter;
}

// Brand mark + colour for every provider opendray currently
// ships. The keys are provider ids as returned by the gateway
// (`session.provider_id`).
const _palette = <String, ProviderVisual>{
  'claude': ProviderVisual(
    iconAsset: 'assets/provider_icons/claude.svg',
    brandColor: Color(0xFFD97757),
    label: 'Claude Code',
    fallbackLetter: 'C',
  ),
  'codex': ProviderVisual(
    // Codex CLI is OpenAI-branded; reuse the OpenAI mark like the
    // web admin does.
    iconAsset: 'assets/provider_icons/openai.svg',
    brandColor: Color(0xFF10A37F),
    label: 'Codex',
    fallbackLetter: 'C',
  ),
  'gemini': ProviderVisual(
    iconAsset: 'assets/provider_icons/gemini.svg',
    brandColor: Color(0xFF3186FF),
    label: 'Gemini',
    fallbackLetter: 'G',
  ),
  'antigravity': ProviderVisual(
    // Antigravity (agy) ships its own multi-colour Google mark. Tint
    // uses the lighter brand blue so the tile reads distinct from
    // Gemini's. Keep in sync with web's PROVIDER_ICON_MAP.
    iconAsset: 'assets/provider_icons/antigravity.svg',
    brandColor: Color(0xFF749BFF),
    label: 'Antigravity',
    fallbackLetter: 'A',
  ),
  'shell': ProviderVisual(
    iconAsset: 'assets/provider_icons/shell.svg',
    brandColor: Color(0xFF4D4D4D),
    label: 'Shell',
    fallbackLetter: 'S',
  ),
};

const _fallback = ProviderVisual(
  iconAsset: null,
  brandColor: Color(0xFF6B7280),
  label: 'Provider',
  fallbackLetter: '?',
);

ProviderVisual providerVisualFor(String providerId) {
  if (providerId.isEmpty) return _fallback;
  final entry = _palette[providerId.toLowerCase()];
  if (entry != null) return entry;
  // Unknown provider → neutral tile with the provider id's first
  // letter (matches web admin's letter-avatar fallback).
  final letter = providerId.substring(0, 1).toUpperCase();
  return ProviderVisual(
    iconAsset: null,
    brandColor: _fallback.brandColor,
    label: providerId,
    fallbackLetter: letter,
  );
}
