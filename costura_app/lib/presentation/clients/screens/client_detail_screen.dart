import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../providers/client_provider.dart';

class ClientDetailScreen extends ConsumerWidget {
  final String id;

  const ClientDetailScreen({super.key, required this.id});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final clientAsync = ref.watch(clientDetailProvider(id));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Detalle de Cliente'),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit),
            onPressed: () => context.go('/clients/$id/edit'),
          )
        ],
      ),
      body: clientAsync.when(
        data: (client) {
          return SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Center(
                  child: CircleAvatar(
                    radius: 40,
                    backgroundColor: AppTheme.primaryGold.withOpacity(0.2),
                    child: Text(
                      client.nombre[0].toUpperCase(),
                      style: const TextStyle(
                        fontSize: 32,
                        color: AppTheme.primaryGold,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                Center(
                  child: Text(
                    client.nombre,
                    style: Theme.of(context).textTheme.displayLarge?.copyWith(fontSize: 24),
                  ),
                ),
                const SizedBox(height: 32),
                const Text('Información de Contacto', style: TextStyle(color: AppTheme.primaryGold, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                Card(
                  child: Column(
                    children: [
                      ListTile(
                        leading: const Icon(Icons.phone),
                        title: Text(client.telefono ?? 'No especificado'),
                      ),
                      const Divider(height: 0),
                      ListTile(
                        leading: const Icon(Icons.email),
                        title: Text(client.email ?? 'No especificado'),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 32),
                const Text('Historial de Encargos', style: TextStyle(color: AppTheme.primaryGold, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(32.0),
                    child: Center(
                      child: Text(
                        'Aún no hay encargos para este cliente',
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: Colors.white54),
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
