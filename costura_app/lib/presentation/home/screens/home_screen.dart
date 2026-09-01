import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../../auth/providers/auth_provider.dart';
import '../../finance/providers/finance_provider.dart';
import '../../orders/providers/order_provider.dart';
import '../widgets/dashboard_card.dart';

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).user;
    final balanceAsync = ref.watch(balanceProvider);
    final ordersAsync = ref.watch(ordersListProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text('Hola, ${user?.nombre ?? 'Costurera'}'),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: () {
              ref.read(authProvider.notifier).logout();
            },
          )
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Resumen del Mes',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: DashboardCard(
                    title: 'Ingresos',
                    value: balanceAsync.when(
                      data: (b) => '\$${b.ingresos.toStringAsFixed(0)}',
                      loading: () => '...',
                      error: (_, __) => '\$0',
                    ),
                    icon: Icons.trending_up,
                    color: Colors.green,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: DashboardCard(
                    title: 'Encargos',
                    value: ordersAsync.when(
                      data: (orders) => '${orders.length}',
                      loading: () => '...',
                      error: (_, __) => '0',
                    ),
                    icon: Icons.cut,
                    color: AppTheme.primaryGold,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 32),
            Text(
              'Accesos Rápidos',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 16),
            ListTile(
              leading: const CircleAvatar(
                backgroundColor: AppTheme.surfaceDark,
                child: Icon(Icons.people_outline, color: AppTheme.primaryGold),
              ),
              title: const Text('Mis Clientes'),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => context.go('/clients'),
            ),
            ListTile(
              leading: const CircleAvatar(
                backgroundColor: AppTheme.surfaceDark,
                child: Icon(Icons.add_shopping_cart, color: AppTheme.primaryGold),
              ),
              title: const Text('Nuevo Encargo'),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => context.go('/orders/new'),
            ),
            ListTile(
              leading: const CircleAvatar(
                backgroundColor: AppTheme.surfaceDark,
                child: Icon(Icons.inventory_2_outlined, color: AppTheme.primaryGold),
              ),
              title: const Text('Ver Inventario'),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => context.go('/inventory'),
            ),
          ],
        ),
      ),
    );
  }
}

