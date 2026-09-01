import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../data/repositories/order_repository.dart';
import '../../../domain/models/order.dart';

final orderSearchQueryProvider = StateProvider<String>((ref) => '');
final orderStatusFilterProvider = StateProvider<String>((ref) => '');

final ordersListProvider = FutureProvider.autoDispose<List<Order>>((ref) async {
  final repository = ref.watch(orderRepositoryProvider);
  final query = ref.watch(orderSearchQueryProvider);
  final estado = ref.watch(orderStatusFilterProvider);
  
  return repository.getOrders(q: query, estado: estado);
});

final orderDetailProvider = FutureProvider.family.autoDispose<Order, String>((ref, id) async {
  final repository = ref.watch(orderRepositoryProvider);
  return repository.getOrder(id);
});
