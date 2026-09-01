import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/theme/app_theme.dart';

class SplashScreen extends ConsumerWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return const Scaffold(
      backgroundColor: AppTheme.backgroundDark,
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.cut_outlined,
              size: 80,
              color: AppTheme.primaryGold,
            ),
            SizedBox(height: 24),
            CircularProgressIndicator(
              color: AppTheme.primaryGold,
            ),
          ],
        ),
      ),
    );
  }
}
