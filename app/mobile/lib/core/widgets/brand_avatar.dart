// BrandAvatar — circular tile that shows the provider's official
// mark (Claude / Antigravity / OpenAI / Shell) on top of a brand-tinted
// background. Falls back to a single uppercase letter when the
// provider has no bundled SVG.
//
// Mirrors app/web/src/components/BrandAvatar.tsx — both surfaces
// render the same mark so a Claude session looks like a Claude
// session whether the operator opens the web admin or the iOS app.

import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:opendray/core/providers/provider_visual.dart';

class BrandAvatar extends StatelessWidget {
  const BrandAvatar({
    required this.providerId,
    this.size = 36,
    super.key,
  });

  final String providerId;
  final double size;

  @override
  Widget build(BuildContext context) {
    final visual = providerVisualFor(providerId);
    final inner = size * 0.6;
    // 18% alpha tint matches the web admin so the brand stays
    // legible without overwhelming the card.
    final tint = visual.brandColor.withValues(alpha: 0.18);
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: tint,
        shape: BoxShape.circle,
      ),
      alignment: Alignment.center,
      child: visual.iconAsset != null
          ? SvgPicture.asset(
              visual.iconAsset!,
              width: inner,
              height: inner,
              semanticsLabel: visual.label,
              // Single-fill black marks (OpenAI/Codex, Shell, OpenCode)
              // render in the theme foreground so they read on the dark
              // tile — matching the web admin, which inverts them to white.
              // Multi-colour marks keep their own ink.
              colorFilter: visual.monochrome
                  ? ColorFilter.mode(
                      Theme.of(context).colorScheme.onSurface,
                      BlendMode.srcIn,
                    )
                  : null,
            )
          : Text(
              visual.fallbackLetter,
              style: TextStyle(
                color: visual.brandColor,
                fontSize: size * 0.45,
                fontWeight: FontWeight.w600,
                height: 1,
              ),
            ),
    );
  }
}
