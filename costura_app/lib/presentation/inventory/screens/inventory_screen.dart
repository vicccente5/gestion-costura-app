import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../../inventory/providers/inventory_provider.dart';

class InventoryScreen extends ConsumerWidget {
  const InventoryScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final inventoryAsync = ref.watch(inventoryListProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Inventario'),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(60),
          child: Padding(
            padding: const EdgeInsets.all(8.0),
            child: TextField(
              decoration: InputDecoration(
                hintText: 'Buscar material...',
                prefixIcon: const Icon(Icons.search),
                contentPadding: const EdgeInsets.symmetric(vertical: 0, horizontal: 16),
                filled: true,
                fillColor: AppTheme.backgroundDark,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(30),
                  borderSide: BorderSide.none,
                ),
              ),
              onChanged: (value) {
                ref.read(materialSearchQueryProvider.notifier).state = value;
              },
            ),
          ),
        ),
      ),
      body: inventoryAsync.when(
        data: (materials) {
          if (materials.isEmpty) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(Icons.inventory_2_outlined, size: 80, color: Colors.white24),
                  const SizedBox(height: 16),
                  const Text('Inventario vacío', style: TextStyle(fontSize: 18, color: Colors.white54)),
                  const SizedBox(height: 8),
                  TextButton.icon(
                    onPressed: () => context.go('/inventory/new'),
                    icon: const Icon(Icons.add, color: AppTheme.primaryGold),
                    label: const Text('Añadir Material', style: TextStyle(color: AppTheme.primaryGold)),
                  ),
                ],
              ),
            );
          }
          return RefreshIndicator(
            onRefresh: () async => ref.invalidate(inventoryListProvider),
            color: AppTheme.primaryGold,
            child: ListView.builder(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.all(16),
              itemCount: materials.length,
              itemBuilder: (context, index) {
                final material = materials[index];
                final hasLowStock = material.stockActual <= material.stockMinimo;
                
                return Card(
                  margin: const EdgeInsets.only(bottom: 12),
                  child: ListTile(
                    contentPadding: const EdgeInsets.all(16),
                    leading: Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: hasLowStock ? AppTheme.errorRed.withOpacity(0.1) : AppTheme.primaryGold.withOpacity(0.1),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(
                        Icons.category,
                        color: hasLowStock ? AppTheme.errorRed : AppTheme.primaryGold,
                      ),
                    ),
                    title: Row(
                      children: [
                        Expanded(
                          child: Text(
                            material.nombre,
                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                          ),
                        ),
                        if (hasLowStock)
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                            decoration: BoxDecoration(
                              color: AppTheme.errorRed.withOpacity(0.2),
                              borderRadius: BorderRadius.circular(12),
                              border: Border.all(color: AppTheme.errorRed),
                            ),
                            child: const Text(
                              'STOCK BAJO',
                              style: TextStyle(
                                fontSize: 10,
                                fontWeight: FontWeight.bold,
                                color: AppTheme.errorRed,
                              ),
                            ),
                          ),
                      ],
                    ),
                    subtitle: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const SizedBox(height: 8),
                        Text(
                          'Stock: ${material.stockActual} ${material.unidadMedida} (Min: ${material.stockMinimo})',
                          style: TextStyle(
                            color: hasLowStock ? AppTheme.errorRed : Colors.white70,
                          ),
                        ),
                      ],
                    ),
                    onTap: () => context.go('/inventory/${material.id}'),
                  ),
                );
              },
            ),
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, stack) => Center(
          child: Text(
            'Error al cargar inventario:\n$error',
            textAlign: TextAlign.center,
            style: const TextStyle(color: AppTheme.errorRed),
          ),
        ),
      ),
      floatingActionButton: FloatingActionButton(
        backgroundColor: AppTheme.primaryGold,
        foregroundColor: Colors.black,
        child: const Icon(Icons.add),
        onPressed: () => context.go('/inventory/new'),
      ),
    );
  }
}

