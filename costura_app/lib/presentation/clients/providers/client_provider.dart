import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../data/repositories/client_repository.dart';
import '../../../domain/models/client.dart';

// Proveedor para la búsqueda
final clientSearchQueryProvider = StateProvider<String>((ref) => '');

// Proveedor de la lista de clientes (usa el FutureProvider para cache y loading states)
final clientsListProvider = FutureProvider.autoDispose<List<Client>>((ref) async {
  final repository = ref.watch(clientRepositoryProvider);
  final query = ref.watch(clientSearchQueryProvider);
  return repository.getClients(q: query);
});

// Proveedor para obtener un cliente específico por ID
final clientDetailProvider = FutureProvider.family.autoDispose<Client, String>((ref, id) async {
  final repository = ref.watch(clientRepositoryProvider);
  return repository.getClient(id);
});
