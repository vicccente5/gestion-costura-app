import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../core/theme/app_theme.dart';
import '../../../data/repositories/order_repository.dart';
import '../../clients/providers/client_provider.dart';
import '../../inventory/providers/inventory_provider.dart';
import '../providers/order_provider.dart';

class OrderFormScreen extends ConsumerStatefulWidget {
  final String? orderId;

  const OrderFormScreen({super.key, this.orderId});

  @override
  ConsumerState<OrderFormScreen> createState() => _OrderFormScreenState();
}

class _OrderFormScreenState extends ConsumerState<OrderFormScreen> {
  final _formKey = GlobalKey<FormState>();
  
  String? _selectedClientId;
  final _descripcionCtrl = TextEditingController();
  final _precioVentaCtrl = TextEditingController();
  final _horasCtrl = TextEditingController();
  final _tarifaHoraCtrl = TextEditingController();
  final _notasCtrl = TextEditingController();
  DateTime? _fechaEntrega;
  String _estado = 'pendiente';
  
  final List<Map<String, dynamic>> _selectedMaterials = [];
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    if (widget.orderId != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _loadData();
      });
    }
  }

  Future<void> _loadData() async {
    setState(() => _isLoading = true);
    try {
      final repo = ref.read(orderRepositoryProvider);
      final order = await repo.getOrder(widget.orderId!);
      
      _selectedClientId = order.clientId;
      _descripcionCtrl.text = order.descripcion;
      _precioVentaCtrl.text = order.precioVenta.toStringAsFixed(0);
      _horasCtrl.text = order.horas.toString();
      _tarifaHoraCtrl.text = order.tarifaHora.toStringAsFixed(0);
      _fechaEntrega = order.fechaEntrega;
      _estado = order.estado;
      
      // Load existing materials
      if (order.materiales != null) {
        for (final m in order.materiales!) {
          _selectedMaterials.add({
            'material_id': m.materialId,
            'cantidad': m.cantidad,             // era cantidadUsada
            'nombre': m.material?.nombre ?? 'Material',
          });
        }
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.errorRed),
        );
      }
    } finally {
      setState(() => _isLoading = false);
    }
  }

  Future<void> _selectDate(BuildContext context) async {
    final DateTime? picked = await showDatePicker(
      context: context,
      initialDate: _fechaEntrega ?? DateTime.now().add(const Duration(days: 7)),
      firstDate: DateTime.now(),
      lastDate: DateTime.now().add(const Duration(days: 365)),
    );
    if (picked != null && picked != _fechaEntrega) {
      setState(() {
        _fechaEntrega = picked;
      });
    }
  }

  void _showAddMaterialDialog() {
    String? selectedMaterialId;
    String? selectedMaterialName;
    final cantCtrl = TextEditingController();

    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          title: const Text('Agregar Material'),
          content: Consumer(
            builder: (context, ref, child) {
              final materialsAsync = ref.watch(inventoryListProvider);
              return materialsAsync.when(
                data: (materials) {
                  return Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      DropdownButtonFormField<String>(
                        decoration: const InputDecoration(labelText: 'Material'),
                        items: materials.map((m) => DropdownMenuItem(
                          value: m.id,
                          child: Text('${m.nombre} (Disp: ${m.stockActual} ${m.unidadMedida})'),
                        )).toList(),
                        onChanged: (val) {
                          selectedMaterialId = val;
                          selectedMaterialName = materials.firstWhere((m) => m.id == val).nombre;
                        },
                      ),
                      const SizedBox(height: 16),
                      TextField(
                        controller: cantCtrl,
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: const InputDecoration(labelText: 'Cantidad a utilizar'),
                      ),
                    ],
                  );
                },
                loading: () => const CircularProgressIndicator(),
                error: (e, _) => Text('Error: $e'),
              );
            },
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancelar')),
            ElevatedButton(
              onPressed: () {
                if (selectedMaterialId != null && cantCtrl.text.isNotEmpty) {
                  final cant = double.tryParse(cantCtrl.text);
                  if (cant != null && cant > 0) {
                    setState(() {
                      _selectedMaterials.add({
                        'material_id': selectedMaterialId,
                        'cantidad': cant,
                        'nombre': selectedMaterialName,
                      });
                    });
                    Navigator.pop(ctx);
                  }
                }
              },
              child: const Text('Agregar'),
            ),
          ],
        );
      }
    );
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_selectedClientId == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Debe seleccionar un cliente'), backgroundColor: AppTheme.errorRed),
      );
      return;
    }
    if (_fechaEntrega == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Debe seleccionar una fecha de entrega'), backgroundColor: AppTheme.errorRed),
      );
      return;
    }

    setState(() => _isLoading = true);
    try {
      final repo = ref.read(orderRepositoryProvider);
      
      final precioVenta = double.tryParse(_precioVentaCtrl.text) ?? 0;
      final horas = double.tryParse(_horasCtrl.text) ?? 0;
      final tarifaHora = double.tryParse(_tarifaHoraCtrl.text) ?? 0;
      final fechaStr = _fechaEntrega!.toIso8601String().substring(0, 10); // YYYY-MM-DD
      
      // Map materials for API
      final apiMaterials = _selectedMaterials.map((m) => {
        'material_id': m['material_id'],
        'cantidad': m['cantidad'],
      }).toList();

      if (widget.orderId == null) {
        // Crear nuevo encargo
        await repo.createOrder(
          _selectedClientId!,
          _descripcionCtrl.text.trim(),
          fechaStr,
          precioVenta,
          horas,
          tarifaHora,
          _notasCtrl.text.trim(),
          apiMaterials,
        );
      } else {
        // Actualizar metadatos
        await repo.updateOrder(
          widget.orderId!,
          _descripcionCtrl.text.trim(),
          fechaStr,
          precioVenta,
          horas,
          tarifaHora,
          _notasCtrl.text.trim(),
        );
        // Si el estado cambió, usar el endpoint dedicado
        final originalOrder = await repo.getOrder(widget.orderId!);
        if (originalOrder.estado != _estado) {
          await repo.changeOrderStatus(widget.orderId!, _estado);
        }
      }
      
      ref.invalidate(ordersListProvider);
      if (widget.orderId != null) {
        ref.invalidate(orderDetailProvider(widget.orderId!));
      }
      
      if (mounted) context.pop();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.errorRed),
        );
      }
    } finally {
      setState(() => _isLoading = false);
    }
  }

  @override
  void dispose() {
    _descripcionCtrl.dispose();
    _precioVentaCtrl.dispose();
    _horasCtrl.dispose();
    _tarifaHoraCtrl.dispose();
    _notasCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final isEditing = widget.orderId != null;
    final clientsAsync = ref.watch(clientsListProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text(isEditing ? 'Editar Encargo' : 'Nuevo Encargo'),
      ),
      body: _isLoading && !isEditing 
          ? const Center(child: CircularProgressIndicator())
          : SingleChildScrollView(
              padding: const EdgeInsets.all(16),
              child: Form(
                key: _formKey,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('Seleccionar Cliente', style: TextStyle(fontWeight: FontWeight.bold, color: AppTheme.primaryGold)),
                    const SizedBox(height: 8),
                    clientsAsync.when(
                      data: (clients) {
                        return DropdownButtonFormField<String>(
                          value: _selectedClientId,
                          decoration: const InputDecoration(
                            prefixIcon: Icon(Icons.person),
                          ),
                          hint: const Text('Elija un cliente'),
                          items: clients.map((c) {
                            return DropdownMenuItem(
                              value: c.id,
                              child: Text(c.nombre),
                            );
                          }).toList(),
                          onChanged: (val) => setState(() => _selectedClientId = val),
                          validator: (val) => val == null ? 'Requerido' : null,
                        );
                      },
                      loading: () => const CircularProgressIndicator(),
                      error: (err, _) => Text('Error al cargar clientes: $err'),
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _descripcionCtrl,
                      maxLines: 3,
                      decoration: const InputDecoration(
                        labelText: 'Descripción del Encargo *',
                        prefixIcon: Icon(Icons.description),
                      ),
                      validator: (value) => value == null || value.isEmpty ? 'Requerido' : null,
                    ),
                    const SizedBox(height: 16),
                    ListTile(
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                        side: const BorderSide(color: Colors.white24),
                      ),
                      leading: const Icon(Icons.calendar_today, color: AppTheme.primaryGold),
                      title: Text(
                        _fechaEntrega == null 
                          ? 'Seleccionar Fecha de Entrega *' 
                          : 'Entrega: ${DateFormat('dd/MM/yyyy').format(_fechaEntrega!)}'
                      ),
                      onTap: () => _selectDate(context),
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _precioVentaCtrl,
                      keyboardType: const TextInputType.numberWithOptions(decimal: true),
                      decoration: const InputDecoration(
                        labelText: 'Precio de Venta (\$)',
                        prefixIcon: Icon(Icons.attach_money),
                      ),
                      validator: (val) => val == null || val.isEmpty ? 'Requerido' : null,
                    ),
                    const SizedBox(height: 16),
                    Row(
                      children: [
                        Expanded(
                          child: TextFormField(
                            controller: _horasCtrl,
                            keyboardType: const TextInputType.numberWithOptions(decimal: true),
                            decoration: const InputDecoration(
                              labelText: 'Horas Trabajo',
                              prefixIcon: Icon(Icons.timer),
                            ),
                          ),
                        ),
                        const SizedBox(width: 16),
                        Expanded(
                          child: TextFormField(
                            controller: _tarifaHoraCtrl,
                            keyboardType: const TextInputType.numberWithOptions(decimal: true),
                            decoration: const InputDecoration(
                              labelText: 'Tarifa/Hora (\$)',
                              prefixIcon: Icon(Icons.monetization_on),
                            ),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 24),
                    const Text('Materiales a Utilizar', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16, color: AppTheme.primaryGold)),
                    const SizedBox(height: 8),
                    if (_selectedMaterials.isEmpty)
                      const Text('No se han asignado materiales.')
                    else
                      ListView.builder(
                        shrinkWrap: true,
                        physics: const NeverScrollableScrollPhysics(),
                        itemCount: _selectedMaterials.length,
                        itemBuilder: (ctx, i) {
                          final mat = _selectedMaterials[i];
                          return ListTile(
                            dense: true,
                            title: Text(mat['nombre'] ?? ''),
                            subtitle: Text('Cantidad: ${mat['cantidad']}'),
                            trailing: IconButton(
                              icon: const Icon(Icons.delete, color: Colors.red),
                              onPressed: () {
                                setState(() {
                                  _selectedMaterials.removeAt(i);
                                });
                              },
                            ),
                          );
                        },
                      ),
                    if (!isEditing)
                      TextButton.icon(
                        onPressed: _showAddMaterialDialog,
                        icon: const Icon(Icons.add),
                        label: const Text('Agregar Material'),
                      ),
                    
                    if (isEditing) ...[
                      const SizedBox(height: 16),
                      DropdownButtonFormField<String>(
                        value: _estado,
                        decoration: const InputDecoration(
                          labelText: 'Estado del Encargo',
                          prefixIcon: Icon(Icons.info_outline),
                        ),
                        items: const [
                          DropdownMenuItem(value: 'pendiente', child: Text('Pendiente')),
                          DropdownMenuItem(value: 'en_progreso', child: Text('En Proceso')),
                          DropdownMenuItem(value: 'completado', child: Text('Completado')),
                          DropdownMenuItem(value: 'entregado', child: Text('Entregado')),
                          DropdownMenuItem(value: 'cancelado', child: Text('Cancelado')),
                        ],
                        onChanged: (val) => setState(() => _estado = val!),
                      ),
                    ],
                    const SizedBox(height: 32),
                    SizedBox(
                      width: double.infinity,
                      child: ElevatedButton(
                        onPressed: _isLoading ? null : _submit,
                        child: _isLoading 
                            ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(color: Colors.black, strokeWidth: 2))
                            : const Text('Guardar Encargo'),
                      ),
                    )
                  ],
                ),
              ),
            ),
    );
  }
}
