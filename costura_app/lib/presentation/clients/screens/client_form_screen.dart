import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../../../data/repositories/client_repository.dart';
import '../../../domain/models/client.dart';
import '../providers/client_provider.dart';

class ClientFormScreen extends ConsumerStatefulWidget {
  final String? clientId; // Si es null, estamos creando. Si tiene valor, editando.

  const ClientFormScreen({super.key, this.clientId});

  @override
  ConsumerState<ClientFormScreen> createState() => _ClientFormScreenState();
}

class _ClientFormScreenState extends ConsumerState<ClientFormScreen> {
  final _formKey = GlobalKey<FormState>();
  final _nombreCtrl = TextEditingController();
  final _telefonoCtrl = TextEditingController();
  final _emailCtrl = TextEditingController();
  
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    if (widget.clientId != null) {
      // Cargar datos iniciales
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _loadClientData();
      });
    }
  }

  Future<void> _loadClientData() async {
    setState(() => _isLoading = true);
    try {
      final repo = ref.read(clientRepositoryProvider);
      final client = await repo.getClient(widget.clientId!);
      _nombreCtrl.text = client.nombre;
      _telefonoCtrl.text = client.telefono ?? '';
      _emailCtrl.text = client.email ?? '';
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
      final repo = ref.read(clientRepositoryProvider);
      if (widget.clientId == null) {
        await repo.createClient(
          _nombreCtrl.text.trim(),
          _telefonoCtrl.text.trim(),
          _emailCtrl.text.trim(),
        );
      } else {
        await repo.updateClient(
          widget.clientId!,
          _nombreCtrl.text.trim(),
          _telefonoCtrl.text.trim(),
          _emailCtrl.text.trim(),
        );
      }
      
      // Invalida la lista para que se recargue al volver
      ref.invalidate(clientsListProvider);
      if (widget.clientId != null) {
        ref.invalidate(clientDetailProvider(widget.clientId!));
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
    _telefonoCtrl.dispose();
    _emailCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final isEditing = widget.clientId != null;

    return Scaffold(
      appBar: AppBar(
        title: Text(isEditing ? 'Editar Cliente' : 'Nuevo Cliente'),
      ),
      body: _isLoading && !isEditing 
          // Solo mostramos loader bloqueante inicial si no estamos editando o si estamos esperando peticiones pesadas
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
                        labelText: 'Nombre Completo *',
                        prefixIcon: Icon(Icons.person),
                      ),
                      validator: (value) => value == null || value.isEmpty ? 'Requerido' : null,
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _telefonoCtrl,
                      keyboardType: TextInputType.phone,
                      decoration: const InputDecoration(
                        labelText: 'Teléfono',
                        prefixIcon: Icon(Icons.phone),
                      ),
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _emailCtrl,
                      keyboardType: TextInputType.emailAddress,
                      decoration: const InputDecoration(
                        labelText: 'Correo Electrónico',
                        prefixIcon: Icon(Icons.email),
                      ),
                    ),
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
