import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../../../data/repositories/material_repository.dart';
import '../providers/inventory_provider.dart';

class MaterialFormScreen extends ConsumerStatefulWidget {
  final String? materialId;

  const MaterialFormScreen({super.key, this.materialId});

  @override
  ConsumerState<MaterialFormScreen> createState() => _MaterialFormScreenState();
}

class _MaterialFormScreenState extends ConsumerState<MaterialFormScreen> {
  final _formKey = GlobalKey<FormState>();
  final _nombreCtrl = TextEditingController();
  final _categoriaCtrl = TextEditingController();
  final _unidadCtrl = TextEditingController();
  final _stockMinimoCtrl = TextEditingController();
  final _costoUnitarioCtrl = TextEditingController();
  
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    if (widget.materialId != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _loadData();
      });
    }
  }

  Future<void> _loadData() async {
    setState(() => _isLoading = true);
    try {
      final repo = ref.read(materialRepositoryProvider);
      final material = await repo.getMaterial(widget.materialId!);
      _nombreCtrl.text = material.nombre;
      _categoriaCtrl.text = material.categoria ?? '';
      _unidadCtrl.text = material.unidadMedida;
      _stockMinimoCtrl.text = material.stockMinimo.toString();
      _costoUnitarioCtrl.text = material.costoUnitario.toString();
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.errorRed),
      );
    } finally {
      setState(() => _isLoading = false);
    }
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _isLoading = true);
    try {
      final repo = ref.read(materialRepositoryProvider);
      final stockMinimo = double.tryParse(_stockMinimoCtrl.text) ?? 0;
      final costoUnitario = double.tryParse(_costoUnitarioCtrl.text) ?? 0;

      if (widget.materialId == null) {
        await repo.createMaterial(
          _nombreCtrl.text.trim(),
          _categoriaCtrl.text.trim(),
          _unidadCtrl.text.trim(),
          stockMinimo,
          costoUnitario,
        );
      } else {
        await repo.updateMaterial(
          widget.materialId!,
          _nombreCtrl.text.trim(),
          _categoriaCtrl.text.trim(),
          _unidadCtrl.text.trim(),
          stockMinimo,
        );
      }
      
      ref.invalidate(inventoryListProvider);
      if (widget.materialId != null) {
        ref.invalidate(materialDetailProvider(widget.materialId!));
      }
      
      if (mounted) context.pop();
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.errorRed),
      );
    } finally {
      setState(() => _isLoading = false);
    }
  }

  @override
  void dispose() {
    _nombreCtrl.dispose();
    _categoriaCtrl.dispose();
    _unidadCtrl.dispose();
    _stockMinimoCtrl.dispose();
    _costoUnitarioCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final isEditing = widget.materialId != null;

    return Scaffold(
      appBar: AppBar(
        title: Text(isEditing ? 'Editar Material' : 'Nuevo Material'),
      ),
      body: _isLoading && !isEditing 
          ? const Center(child: CircularProgressIndicator())
          : SingleChildScrollView(
              padding: const EdgeInsets.all(16),
              child: Form(
                key: _formKey,
                child: Column(
                  children: [
                    TextFormField(
                      controller: _nombreCtrl,
                      decoration: const InputDecoration(
                        labelText: 'Nombre del Material *',
                        prefixIcon: Icon(Icons.inventory_2),
                      ),
                      validator: (value) => value == null || value.isEmpty ? 'Requerido' : null,
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _categoriaCtrl,
                      decoration: const InputDecoration(
                        labelText: 'Categoría (ej. telas, hilos) *',
                        prefixIcon: Icon(Icons.category),
                      ),
                      validator: (value) => value == null || value.isEmpty ? 'Requerido' : null,
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _unidadCtrl,
                      decoration: const InputDecoration(
                        labelText: 'Unidad de Medida (ej. metros, botones) *',
                        prefixIcon: Icon(Icons.straighten),
                      ),
                      validator: (value) => value == null || value.isEmpty ? 'Requerido' : null,
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _stockMinimoCtrl,
                      keyboardType: const TextInputType.numberWithOptions(decimal: true),
                      decoration: const InputDecoration(
                        labelText: 'Stock Mínimo de Alerta',
                        prefixIcon: Icon(Icons.warning),
                      ),
                      validator: (value) {
                        if (value == null || value.isEmpty) return 'Requerido';
                        if (double.tryParse(value) == null) return 'Debe ser un número';
                        return null;
                      },
                    ),
                    if (!isEditing) ...[
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _costoUnitarioCtrl,
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: const InputDecoration(
                          labelText: 'Costo Unitario (Valor por unidad) *',
                          prefixIcon: Icon(Icons.attach_money),
                        ),
                        validator: (value) {
                          if (value == null || value.isEmpty) return 'Requerido';
                          if (double.tryParse(value) == null) return 'Debe ser un número';
                          return null;
                        },
                      ),
                    ],
                    const SizedBox(height: 32),
                    SizedBox(
                      width: double.infinity,
                      child: ElevatedButton(
                        onPressed: _isLoading ? null : _submit,
                        child: _isLoading 
                            ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(color: Colors.black, strokeWidth: 2))
                            : const Text('Guardar'),
                      ),
                    )
                  ],
                ),
              ),
            ),
    );
  }
}
