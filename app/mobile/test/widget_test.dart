import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:opendray/core/theme/app_theme.dart';

// F1 smoke test: the OpendrayApp shell needs secure storage at
// boot, which is hard to mock in a widget test without diverting
// the F2 schedule into mocks. For F1 we just verify the theme
// builder produces a usable ThemeData.
void main() {
  testWidgets('AppTheme.dark builds and applies', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.dark(),
          home: const Scaffold(body: Center(child: Text('opendray'))),
        ),
      ),
    );
    expect(find.text('opendray'), findsOneWidget);
  });
}
