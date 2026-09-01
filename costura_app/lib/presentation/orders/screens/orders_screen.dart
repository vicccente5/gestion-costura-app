import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../core/theme/app_theme.dart';
import '../providers/order_provider.dart';

class OrdersScreen extends ConsumerWidget {
  const OrdersScreen({super.key});

  Color _getStatusColor(String status) {
    switch (status.toLowerCase()) {
      case 'pendiente':
        return Colors.orange;
      case 'en_progreso':
        return Colors.blue;
      case 'completado':
        return Colors.green;
      case 'entregado':
        return AppTheme.primaryGold;
      case 'cancelado':
        return Colors.red;
      default:
        return Colors.grey;
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ordersAsync = ref.watch(ordersListProvider);
    final currentStatus = ref.watch(orderStatusFilterProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Encargos'),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(110),
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 8.0),
                child: TextField(
                  decoration: InputDecoration(
                    hintText: 'Buscar encargo...',
                    prefixIcon: const Icon(Icons.search),
                    contentPadding: const EdgeInsets.symmetric(vertical: 0, horizontal: 16),
                    filled: true,
                    fillColor: AppTheme.backgroundDark,
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(30),
                      borderSide: BorderSide.none,
                    ),
                  ),
                  onChanged: (value) => ref.read(orderSearchQueryProvider.notifier).state = value,
                ),
              ),
              SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 8.0),
                child: Row(
                  children: [
                    _buildFilterChip(context, ref, 'Todos', '', currentStatus),
                    _buildFilterChip(context, ref, 'Pendientes', 'pendiente', currentStatus),
                    _buildFilterChip(context, ref, 'En Proceso', 'en_progreso', currentStatus),
                    _buildFilterChip(context, ref, 'Completados', 'completado', currentStatus),
                    _buildFilterChip(context, ref, 'Entregados', 'entregado', currentStatus),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
      body: ordersAsync.when(
        data: (orders) {
          if (orders.isEmpty) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.inventory_2_outlined, size: 80, color: Colors.white24),
                  const SizedBox(height: 16),
                  const Text('No tienes encargos aquí', style: TextStyle(fontSize: 18, color: Colors.white54)),
                  const SizedBox(height: 8),
                  TextButton.icon(
                    onPressed: () => context.go('/orders/new'),
                    icon: const Icon(Icons.add, color: AppTheme.primaryGold),
                    label: const Text('Crear Encargo', style: TextStyle(color: AppTheme.primaryGold)),
                  ),
                ],
              ),
            );
          }
          return RefreshIndicator(
            onRefresh: () async => ref.invalidate(ordersListProvider),
            color: AppTheme.primaryGold,
            child: ListView.builder(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.all(16),
              itemCount: orders.length,
              itemBuilder: (context, index) {
                final order = orders[index];
                return Card(
                  margin: const EdgeInsets.only(bottom: 12),
                  child: ListTile(
                    contentPadding: const EdgeInsets.all(16),
                    title: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Expanded(
                          child: Text(
                            order.descripcion,
                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                          decoration: BoxDecoration(
                            color: _getStatusColor(order.estado).withOpacity(0.2),
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(color: _getStatusColor(order.estado)),
                          ),
                          child: Text(
                            order.estado.toUpperCase(),
                            style: TextStyle(
                              fontSize: 10,
                              fontWeight: FontWeight.bold,
                              color: _getStatusColor(order.estado),
                            ),
                          ),
                        ),
                      ],
                    ),
                    subtitle: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const SizedBox(height: 8),
                        Row(
                          children: [
                            const Icon(Icons.person, size: 16, color: Colors.white70),
                            const SizedBox(width: 4),
                            Text(order.client?.nombre ?? 'Cliente Desconocido', style: const TextStyle(color: Colors.white70)),
                          ],
                        ),
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            const Icon(Icons.event, size: 16, color: AppTheme.primaryGold),
                            const SizedBox(width: 4),
                            Text(
                              order.fechaEntrega != null
                                  ? 'Entrega: ${DateFormat('dd/MM/yyyy').format(order.fechaEntrega!)}'
                                  : 'Sin fecha',
                              style: const TextStyle(color: AppTheme.primaryGold),
                            ),
                          ],
                        ),
                      ],
                    ),
                    onTap: () => context.go('/orders/${order.id}'),
                  ),
                );
              },
            ),
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(child: Text('Error: $error')),
      ),
      floatingActionButton: FloatingActionButton(
        backgroundColor: AppTheme.primaryGold,
        foregroundColor: Colors.black,
        child: const Icon(Icons.add),
        onPressed: () => context.go('/orders/new'),
      ),
    );
  }

  Widget _buildFilterChip(BuildContext context, WidgetRef ref, String label, String value, String currentValue) {
    final isSelected = currentValue == value;
    return Padding(
      padding: const EdgeInsets.only(right: 8.0),
      child: FilterChip(
        label: Text(label),
        selected: isSelected,
        onSelected: (_) => ref.read(orderStatusFilterProvider.notifier).state = value,
        selectedColor: AppTheme.primaryGold.withOpacity(0.3),
        checkmarkColor: AppTheme.primaryGold,
      ),
    );
  }
}

