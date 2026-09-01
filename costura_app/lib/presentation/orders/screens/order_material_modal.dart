import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/app_theme.dart';
import '../../../data/repositories/order_repository.dart';
import '../../inventory/providers/inventory_provider.dart';
import '../providers/order_provider.dart';

class OrderMaterialModal extends ConsumerStatefulWidget {
  final String orderId;

  const OrderMaterialModal({super.key, required this.orderId});

  @override
  ConsumerState<OrderMaterialModal> createState() => _OrderMaterialModalState();
}

class _OrderMaterialModalState extends ConsumerState<OrderMaterialModal> {
  final _formKey = GlobalKey<FormState>();
  
  String? _selectedMaterialId;
  final _cantidadCtrl = TextEditingController();
  
  bool _isLoading = false;

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_selectedMaterialId == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Seleccione un material'), backgroundColor: AppTheme.errorRed),
      );
      return;
    }

    setState(() => _isLoading = true);
    try {
      final repo = ref.read(orderRepositoryProvider);
      final cantidad = double.parse(_cantidadCtrl.text);

      await repo.addMaterialToOrder(widget.orderId, _selectedMaterialId!, cantidad);
      
      ref.invalidate(orderDetailProvider(widget.orderId));
      ref.invalidate(inventoryListProvider); // Stock was reduced
      
      if (mounted) Navigator.of(context).pop();
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.errorRed),
      );
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  @override
  void dispose() {
    _cantidadCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final inventoryAsync = ref.watch(inventoryListProvider);

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
        left: 16,
        right: 16,
        top: 24,
      ),
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Asignar Material',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: AppTheme.primaryGold),
            ),
            const SizedBox(height: 16),
            inventoryAsync.when(
              data: (materials) {
                return DropdownButtonFormField<String>(
                  value: _selectedMaterialId,
                  decoration: const InputDecoration(
                    prefixIcon: Icon(Icons.inventory),
                  ),
                  hint: const Text('Seleccionar Material'),
                  items: materials.map((m) {
                    return DropdownMenuItem(
                      value: m.id,
                      child: Text('${m.nombre} (Disp: ${m.stockActual} ${m.unidadMedida})'),
                    );
                  }).toList(),
                  onChanged: (val) => setState(() => _selectedMaterialId = val),
                  validator: (val) => val == null ? 'Requerido' : null,
                );
              },
              loading: () => const CircularProgressIndicator(),
              error: (err, _) => Text('Error: $err'),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _cantidadCtrl,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: const InputDecoration(
                labelText: 'Cantidad a utilizar',
                prefixIcon: Icon(Icons.straighten),
              ),
              validator: (v) => (v == null || v.isEmpty || double.tryParse(v) == null) ? 'Inválido' : null,
            ),
            const SizedBox(height: 24),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: _isLoading ? null : _submit,
                child: _isLoading 
                    ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(color: Colors.black))
                    : const Text('Asignar al Encargo'),
              ),
            ),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}
