import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../core/theme/app_theme.dart';
import '../../../data/repositories/order_repository.dart';
import '../../inventory/providers/inventory_provider.dart';
import '../providers/order_provider.dart';
import 'order_material_modal.dart';

class OrderDetailScreen extends ConsumerStatefulWidget {
  final String id;

  const OrderDetailScreen({super.key, required this.id});

  @override
  ConsumerState<OrderDetailScreen> createState() => _OrderDetailScreenState();
}

class _OrderDetailScreenState extends ConsumerState<OrderDetailScreen> {
  bool _changingStatus = false;

  // Colores y labels por estado
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

  String _getStatusLabel(String status) {
    switch (status.toLowerCase()) {
      case 'pendiente':
        return 'Pendiente';
      case 'en_progreso':
        return 'En Proceso';
      case 'completado':
        return 'Completado';
      case 'entregado':
        return 'Entregado';
      case 'cancelado':
        return 'Cancelado';
      default:
        return status.toUpperCase();
    }
  }

  // Siempre se muestran los 4 estados principales — el usuario elige libremente
  // Solo excepción: si ya está cancelado, no se puede cambiar
  List<String> _nextStates(String current) {
    if (current.toLowerCase() == 'cancelado') return [];
    final states = ['pendiente', 'en_progreso', 'completado', 'entregado'];
    states.remove(current.toLowerCase());
    return states;
  }

  void _mostrarCambioEstado(BuildContext context, String estadoActual) {
    final posiblesEstados = _nextStates(estadoActual);
    if (posiblesEstados.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Este encargo está cancelado y no puede cambiar de estado.'),
          backgroundColor: Colors.grey,
        ),
      );
      return;
    }

    showModalBottomSheet(
      context: context,
      backgroundColor: AppTheme.surfaceDark,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return Padding(
          padding: const EdgeInsets.fromLTRB(24, 24, 24, 40),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Cambiar Estado del Encargo',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.bold,
                  color: AppTheme.primaryGold,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                'Estado actual: ${_getStatusLabel(estadoActual)}',
                style: const TextStyle(color: Colors.white54),
              ),
              const SizedBox(height: 24),
              ...posiblesEstados.map((estado) {
                final color = _getStatusColor(estado);
                IconData icon;
                switch (estado) {
                  case 'en_progreso':
                    icon = Icons.play_circle_outline;
                    break;
                  case 'completado':
                    icon = Icons.check_circle_outline;
                    break;
                  case 'entregado':
                    icon = Icons.local_shipping_outlined;
                    break;
                  case 'cancelado':
                    icon = Icons.cancel_outlined;
                    break;
                  default:
                    icon = Icons.circle_outlined;
                }

                return Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: SizedBox(
                    width: double.infinity,
                    child: OutlinedButton.icon(
                      icon: Icon(icon, color: color),
                      label: Text(
                        _getStatusLabel(estado),
                        style: TextStyle(color: color, fontSize: 16),
                      ),
                      style: OutlinedButton.styleFrom(
                        padding: const EdgeInsets.symmetric(vertical: 14),
                        side: BorderSide(color: color),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(12),
                        ),
                      ),
                      onPressed: () {
                        Navigator.of(ctx).pop();
                        _cambiarEstado(estado);
                      },
                    ),
                  ),
                );
              }),
            ],
          ),
        );
      },
    );
  }

  Future<void> _cambiarEstado(String nuevoEstado) async {
    setState(() => _changingStatus = true);
    try {
      final repo = ref.read(orderRepositoryProvider);
      await repo.changeOrderStatus(widget.id, nuevoEstado);
      ref.invalidate(orderDetailProvider(widget.id));
      ref.invalidate(ordersListProvider);
      // Si se entregó, también actualizamos finanzas
      if (nuevoEstado == 'entregado') {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('✅ Encargo entregado — se registró el ingreso automáticamente'),
            backgroundColor: Colors.green,
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.errorRed),
        );
      }
    } finally {
      if (mounted) setState(() => _changingStatus = false);
    }
  }

  void _mostrarModalMaterial(BuildContext context) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: AppTheme.surfaceDark,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => OrderMaterialModal(orderId: widget.id),
    );
  }

  Future<void> _eliminarMaterial(BuildContext context, String materialId) async {
    try {
      final repo = ref.read(orderRepositoryProvider);
      await repo.removeMaterialFromOrder(widget.id, materialId);
      ref.invalidate(orderDetailProvider(widget.id));
      ref.invalidate(inventoryListProvider);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.errorRed),
        );
      }
    }
  }

  Future<void> _deleteOrder(BuildContext context) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Eliminar encargo'),
        content: const Text('¿Estás seguro de eliminar este encargo? Esta acción no se puede deshacer y restaurará el stock de los materiales.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancelar')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Eliminar', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );

    if (confirm != true) return;

    if (mounted) setState(() => _changingStatus = true);
    try {
      final repo = ref.read(orderRepositoryProvider);
      await repo.deleteOrder(widget.id);
      
      ref.invalidate(ordersListProvider);
      ref.invalidate(inventoryListProvider);

      if (mounted) {
        context.pop(); // volver a la lista
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Encargo eliminado exitosamente')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.errorRed),
        );
      }
    } finally {
      if (mounted) setState(() => _changingStatus = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final orderAsync = ref.watch(orderDetailProvider(widget.id));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Detalles del Encargo'),
        actions: [
          IconButton(
            icon: const Icon(Icons.delete, color: Colors.red),
            onPressed: () => _deleteOrder(context),
          ),
          IconButton(
            icon: const Icon(Icons.edit),
            onPressed: () => context.go('/orders/${widget.id}/edit'),
          )
        ],
      ),
      body: orderAsync.when(
        data: (order) {
          final costoTotal = order.costoTotal;
          final precioVenta = order.precioVenta;
          final gananciaNeta = order.gananciaNeta;
          final hasPrecio = precioVenta > 0;
          final statusColor = _getStatusColor(order.estado);
          final isEstadoFinal = order.estado == 'cancelado';

          return Column(
            children: [
              // ── Botón cambio de estado (sticky arriba) ──────────────
              if (!isEstadoFinal)
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                  color: AppTheme.surfaceDark,
                  child: _changingStatus
                      ? const Center(child: CircularProgressIndicator())
                      : ElevatedButton.icon(
                          icon: const Icon(Icons.swap_horiz, color: Colors.black),
                          label: const Text('Cambiar Estado del Encargo'),
                          style: ElevatedButton.styleFrom(
                            backgroundColor: statusColor,
                            foregroundColor: Colors.black,
                            padding: const EdgeInsets.symmetric(vertical: 12),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(12),
                            ),
                          ),
                          onPressed: () => _mostrarCambioEstado(context, order.estado),
                        ),
                ),

              // ── Contenido scrolleable ────────────────────────────────
              Expanded(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // Encabezado
                      Text(
                        order.descripcion,
                        style: Theme.of(context).textTheme.displayLarge?.copyWith(fontSize: 22),
                      ),
                      const SizedBox(height: 8),
                      Row(
                        children: [
                          const Icon(Icons.person, color: AppTheme.primaryGold),
                          const SizedBox(width: 8),
                          Text(
                            order.client?.nombre ?? 'Cliente eliminado',
                            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                          ),
                        ],
                      ),
                      const SizedBox(height: 16),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          // Badge de estado con color dinámico
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                            decoration: BoxDecoration(
                              color: statusColor.withValues(alpha: 0.15),
                              borderRadius: BorderRadius.circular(16),
                              border: Border.all(color: statusColor),
                            ),
                            child: Text(
                              _getStatusLabel(order.estado).toUpperCase(),
                              style: TextStyle(
                                fontWeight: FontWeight.bold,
                                color: statusColor,
                              ),
                            ),
                          ),
                          Text(
                            order.fechaEntrega != null
                                ? 'Entrega: ${DateFormat('dd/MM/yyyy').format(order.fechaEntrega!)}'
                                : 'Sin fecha acordada',
                            style: const TextStyle(color: Colors.white70),
                          ),
                        ],
                      ),
                      const SizedBox(height: 32),

                      // Panel de Rentabilidad
                      const Text('Análisis de Rentabilidad', style: TextStyle(color: AppTheme.primaryGold, fontWeight: FontWeight.bold)),
                      const SizedBox(height: 8),
                      Card(
                        child: Padding(
                          padding: const EdgeInsets.all(16.0),
                          child: Column(
                            children: [
                              Row(
                                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                                children: [
                                  const Text('Costo Total (Materiales + Horas):'),
                                  Text('\$${costoTotal.toStringAsFixed(0)}',
                                      style: const TextStyle(fontWeight: FontWeight.bold)),
                                ],
                              ),
                              const Divider(height: 24),
                              Row(
                                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                                children: [
                                  const Text('Precio de Venta:', style: TextStyle(fontSize: 16)),
                                  Text(
                                    '\$${precioVenta.toStringAsFixed(0)}',
                                    style: const TextStyle(
                                        fontSize: 18,
                                        fontWeight: FontWeight.bold,
                                        color: AppTheme.primaryGold),
                                  ),
                                ],
                              ),
                              if (hasPrecio) ...[
                                const SizedBox(height: 8),
                                Row(
                                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                                  children: [
                                    const Text('Ganancia Neta:'),
                                    Text(
                                      '${gananciaNeta >= 0 ? '+' : ''}\$${gananciaNeta.toStringAsFixed(0)}',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                        color: gananciaNeta >= 0 ? Colors.green : Colors.red,
                                      ),
                                    ),
                                  ],
                                ),
                              ]
                            ],
                          ),
                        ),
                      ),
                      const SizedBox(height: 32),

                      // Lista de Materiales
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          const Text('Materiales Usados',
                              style: TextStyle(color: AppTheme.primaryGold, fontWeight: FontWeight.bold)),
                          TextButton.icon(
                            icon: const Icon(Icons.add, size: 18),
                            label: const Text('Asignar'),
                            onPressed: () => _mostrarModalMaterial(context),
                          ),
                        ],
                      ),
                      if (order.materiales == null || order.materiales!.isEmpty)
                        const Padding(
                          padding: EdgeInsets.only(top: 16.0),
                          child: Center(
                            child: Text(
                              'No hay materiales asignados a este encargo.',
                              style: TextStyle(color: Colors.white54),
                            ),
                          ),
                        )
                      else
                        ListView.builder(
                          shrinkWrap: true,
                          physics: const NeverScrollableScrollPhysics(),
                          itemCount: order.materiales!.length,
                          itemBuilder: (context, index) {
                            final item = order.materiales![index];
                            return Card(
                              margin: const EdgeInsets.only(bottom: 8),
                              child: ListTile(
                                leading: const Icon(Icons.inventory_2, color: AppTheme.primaryGold),
                                title: Text(item.material?.nombre ?? 'Material Desconocido'),
                                subtitle: Text(
                                    'Cantidad: ${item.cantidad} | Costo: \$${item.costoTotal.toStringAsFixed(0)}'),
                                trailing: IconButton(
                                  icon: const Icon(Icons.delete, color: AppTheme.errorRed),
                                  onPressed: () => _eliminarMaterial(context, item.materialId),
                                ),
                              ),
                            );
                          },
                        ),
                      const SizedBox(height: 24),

                      // Mensaje si el encargo está en estado final
                      if (isEstadoFinal)
                        Container(
                          width: double.infinity,
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: statusColor.withValues(alpha: 0.1),
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(color: statusColor.withValues(alpha: 0.4)),
                          ),
                          child: Row(
                            children: [
                              Icon(
                                order.estado == 'entregado'
                                    ? Icons.check_circle
                                    : Icons.cancel,
                                color: statusColor,
                              ),
                              const SizedBox(width: 10),
                              Text(
                                order.estado == 'entregado'
                                    ? 'Encargo entregado — ingreso registrado en finanzas.'
                                    : 'Encargo cancelado.',
                                style: TextStyle(color: statusColor, fontWeight: FontWeight.w500),
                              ),
                            ],
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ],
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
