import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/features/auth/login_screen.dart';
import 'package:opendray/features/home/home_shell.dart';
import 'package:opendray/features/onboarding/onboarding_screen.dart';

// Top-level route map. The redirect callback funnels every
// request through the AuthState gate so the user can never sit
// on a screen they're not authorized for: bootstrap → onboarding
// → login → home.
final routerProvider = Provider<GoRouter>((ref) {
  final authNotifier = ValueNotifier<AuthState>(
    ref.read(authControllerProvider),
  );
  ref.listen<AuthState>(authControllerProvider, (_, next) {
    authNotifier.value = next;
  });

  return GoRouter(
    refreshListenable: authNotifier,
    initialLocation: '/bootstrap',
    routes: [
      GoRoute(
        path: '/bootstrap',
        builder: (_, __) => const _SplashScreen(),
      ),
      GoRoute(
        path: '/onboarding',
        builder: (_, __) => const OnboardingScreen(),
      ),
      GoRoute(
        path: '/login',
        builder: (_, __) => const LoginScreen(),
      ),
      GoRoute(
        path: '/home',
        builder: (_, __) => const HomeShell(),
      ),
    ],
    redirect: (context, state) {
      final auth = authNotifier.value;
      final loc = state.matchedLocation;
      switch (auth) {
        case AuthBootstrapping():
          return loc == '/bootstrap' ? null : '/bootstrap';
        case AuthOnboarding():
          return loc == '/onboarding' ? null : '/onboarding';
        case AuthLoggedOut():
          return loc == '/login' ? null : '/login';
        case AuthLoggedIn():
          if (loc == '/bootstrap' ||
              loc == '/onboarding' ||
              loc == '/login') {
            return '/home';
          }
          return null;
      }
    },
  );
});

class _SplashScreen extends StatelessWidget {
  const _SplashScreen();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(child: CircularProgressIndicator()),
    );
  }
}
