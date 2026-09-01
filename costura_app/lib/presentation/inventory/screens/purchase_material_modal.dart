import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/app_theme.dart';
import '../../../data/repositories/material_repository.dart';
import '../providers/inventory_provider.dart';

class PurchaseMaterialModal extends ConsumerStatefulWidget {
  final String materialId;
  final String unidadMedida;

  const PurchaseMaterialModal({super.key, required this.materialId, required this.unidadMedida});

  @override
  ConsumerState<PurchaseMaterialModal> createState() => _PurchaseMaterialModalState();
}

class _PurchaseMaterialModalState extends ConsumerState<PurchaseMaterialModal> {
  final _formKey = GlobalKey<FormState>();
  final _cantidadCtrl = TextEditingController();
  final _precioUnitarioCtrl = TextEditingController(); // precio por unidad, no total

  bool _isLoading = false;


  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _isLoading = true);
    try {
      final repo = ref.read(materialRepositoryProvider);
      final cantidad = double.parse(_cantidadCtrl.text);
      final precioUnitario = double.parse(_precioUnitarioCtrl.text);
      final fecha = DateTime.now().toIso8601String().substring(0, 10); // YYYY-MM-DD

      await repo.registrarCompra(widget.materialId, cantidad, precioUnitario, fecha);
      
      ref.invalidate(inventoryListProvider);
      ref.invalidate(materialDetailProvider(widget.materialId));
      
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
  void initState() {
    super.initState();
  }

  @override
  void dispose() {
    _cantidadCtrl.dispose();
    _precioUnitarioCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
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
              'Registrar Compra',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: AppTheme.primaryGold),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _cantidadCtrl,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: InputDecoration(
                labelText: 'Cantidad Comprada (${widget.unidadMedida})',
                prefixIcon: const Icon(Icons.add_box),
              ),
              validator: (v) => (v == null || v.isEmpty || double.tryParse(v) == null) ? 'Inválido' : null,
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _precioUnitarioCtrl,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: InputDecoration(
                labelText: 'Precio por ${widget.unidadMedida} (\$)',
                prefixIcon: const Icon(Icons.attach_money),
                hintText: 'Ej: 1500',
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
                    : const Text('Confirmar Compra'),
              ),
            ),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}
