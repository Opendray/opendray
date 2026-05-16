// Channel kind visual identity — single source of truth so the
// channels list, kind picker, and any future surface render the
// same brand mark + tint for a given kind id.
//
// Mirrors the web admin's <BrandIcon> component (which sources
// from simple-icons + curated SVGs under app/web/public/icons/).
// The SVG assets we bundle here are byte-identical to the repo's
// canonical store at /assets/icons/ — Flutter can only bundle
// files inside the package, so we keep a normalised mirror under
// app/mobile/assets/channel_icons/.

import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

class ChannelVisual {
  const ChannelVisual({
    required this.iconAsset,
    required this.brandColor,
    required this.label,
    required this.fallbackLetter,
  });

  /// Asset path under `assets/channel_icons/`, or null when no
  /// curated SVG is bundled (caller renders the fallback letter
  /// in a neutral tile).
  final String? iconAsset;

  /// Canonical brand colour. Used as the tile background tint.
  final Color brandColor;

  /// Human-readable channel name (matches the kind's `label`).
  final String label;

  /// Single uppercase character used inside the tile when no
  /// brand SVG is registered.
  final String fallbackLetter;
}

const _palette = <String, ChannelVisual>{
  'telegram': ChannelVisual(
    iconAsset: 'assets/channel_icons/telegram.svg',
    brandColor: Color(0xFF26A5E4),
    label: 'Telegram',
    fallbackLetter: 'T',
  ),
  'slack': ChannelVisual(
    // The bundled SVG is the 4-colour Slack mark, so we tint the
    // tile with their primary purple but let the SVG carry its
    // own ink.
    iconAsset: 'assets/channel_icons/slack.svg',
    brandColor: Color(0xFF4A154B),
    label: 'Slack',
    fallbackLetter: 'S',
  ),
  'discord': ChannelVisual(
    iconAsset: 'assets/channel_icons/discord.svg',
    brandColor: Color(0xFF5865F2),
    label: 'Discord',
    fallbackLetter: 'D',
  ),
  'feishu': ChannelVisual(
    iconAsset: 'assets/channel_icons/feishu.svg',
    brandColor: Color(0xFF00D6B9),
    label: 'Feishu',
    fallbackLetter: 'F',
  ),
  'dingtalk': ChannelVisual(
    iconAsset: 'assets/channel_icons/dingtalk.svg',
    brandColor: Color(0xFF0089FF),
    label: 'DingTalk',
    fallbackLetter: 'D',
  ),
  'wecom': ChannelVisual(
    iconAsset: 'assets/channel_icons/wecom.svg',
    brandColor: Color(0xFF0082EF),
    label: 'WeCom',
    fallbackLetter: 'W',
  ),
};

const _fallback = ChannelVisual(
  iconAsset: null,
  brandColor: Color(0xFF6B7280),
  label: 'Channel',
  fallbackLetter: '?',
);

ChannelVisual channelVisualFor(String kind) {
  if (kind.isEmpty) return _fallback;
  final entry = _palette[kind.toLowerCase()];
  if (entry != null) return entry;
  // Unknown kind → neutral tile with the kind's first letter so
  // we don't lose all signal (e.g. for legacy `wechat` rows that
  // predate the entry removal).
  final letter = kind.substring(0, 1).toUpperCase();
  return ChannelVisual(
    iconAsset: null,
    brandColor: _fallback.brandColor,
    label: kind,
    fallbackLetter: letter,
  );
}

/// ChannelBrandIcon — the 36×36 tile used in the channel list +
/// the kind-picker sheet. Renders the bundled SVG when available;
/// falls back to a single-letter mark in a brand-tinted disc when
/// the kind is unknown.
class ChannelBrandIcon extends StatelessWidget {
  const ChannelBrandIcon({required this.kind, super.key, this.size = 36});

  final String kind;
  final double size;

  @override
  Widget build(BuildContext context) {
    final visual = channelVisualFor(kind);
    return Container(
      width: size,
      height: size,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: visual.brandColor.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(8),
      ),
      child: visual.iconAsset == null
          ? Text(
              visual.fallbackLetter,
              style: TextStyle(
                color: visual.brandColor,
                fontWeight: FontWeight.w600,
              ),
            )
          : Padding(
              padding: EdgeInsets.all(size * 0.18),
              child: SvgPicture.asset(
                visual.iconAsset!,
                fit: BoxFit.contain,
                semanticsLabel: visual.label,
              ),
            ),
    );
  }
}
