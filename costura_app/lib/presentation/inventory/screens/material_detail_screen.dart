import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../providers/inventory_provider.dart';
import 'purchase_material_modal.dart';

class MaterialDetailScreen extends ConsumerWidget {
  final String id;

  const MaterialDetailScreen({super.key, required this.id});

  void _mostrarModalCompra(BuildContext context, String materialId, String unidadMedida) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: AppTheme.surfaceDark,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => PurchaseMaterialModal(
        materialId: materialId,
        unidadMedida: unidadMedida,
      ),
    );
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final materialAsync = ref.watch(materialDetailProvider(id));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Detalle de Material'),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit),
            onPressed: () => context.go('/inventory/$id/edit'),
          )
        ],
      ),
      body: materialAsync.when(
        data: (material) {
          return SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Center(
                  child: CircleAvatar(
                    radius: 40,
                    backgroundColor: material.isLowStock ? AppTheme.errorRed.withOpacity(0.2) : AppTheme.primaryGold.withOpacity(0.2),
                    child: Icon(
                      Icons.inventory,
                      size: 40,
                      color: material.isLowStock ? AppTheme.errorRed : AppTheme.primaryGold,
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                Center(
                  child: Text(
                    material.nombre,
                    style: Theme.of(context).textTheme.displayLarge?.copyWith(fontSize: 24),
                  ),
                ),
                const SizedBox(height: 32),
                if (material.isLowStock)
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(12),
                    margin: const EdgeInsets.only(bottom: 16),
                    decoration: BoxDecoration(
                      color: AppTheme.errorRed.withOpacity(0.2),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: AppTheme.errorRed),
                    ),
                    child: const Row(
                      children: [
                        Icon(Icons.warning, color: AppTheme.errorRed),
                        SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            '¡Atención! El stock está por debajo del nivel mínimo.',
                            style: TextStyle(color: AppTheme.errorRed, fontWeight: FontWeight.bold),
                          ),
                        ),
                      ],
                    ),
                  ),
                const Text('Información de Stock', style: TextStyle(color: AppTheme.primaryGold, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                Card(
                  child: Column(
                    children: [
                      ListTile(
                        leading: const Icon(Icons.layers),
                        title: const Text('Stock Actual'),
                        trailing: Text(
                          '${material.stockActual} ${material.unidadMedida}',
                          style: TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: 16,
                            color: material.isLowStock ? AppTheme.errorRed : Colors.white,
                          ),
                        ),
                      ),
                      const Divider(height: 0),
                      ListTile(
                        leading: const Icon(Icons.security),
                        title: const Text('Stock Mínimo (Alerta)'),
                        trailing: Text(
                          '${material.stockMinimo} ${material.unidadMedida}',
                          style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 32),
                SizedBox(
                  width: double.infinity,
                  child: ElevatedButton.icon(
                    icon: const Icon(Icons.add_shopping_cart, color: Colors.black),
                    label: const Text('Registrar Compra / Ingreso'),
                    onPressed: () => _mostrarModalCompra(context, material.id, material.unidadMedida),
                  ),
                ),
                const SizedBox(height: 32),
                const Text('Historial de Compras', style: TextStyle(color: AppTheme.primaryGold, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(32.0),
                    child: Center(
                      child: Text(
                        'El historial detallado se implementará próximamente',
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: Colors.white54),
                        textAlign: TextAlign.center,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(
          child: Text('Error: $error', style: const TextStyle(color: AppTheme.errorRed)),
        ),
      ),
    );
  }
}
