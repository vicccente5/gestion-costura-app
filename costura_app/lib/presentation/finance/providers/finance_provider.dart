import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../data/repositories/transaction_repository.dart';
import '../../../domain/models/transaction.dart';

// Current month/year filter
final financeDateFilterProvider = StateProvider<({int month, int year})>((ref) {
  final now = DateTime.now();
  return (month: now.month, year: now.year);
});

// Balance Provider
final balanceProvider = FutureProvider.autoDispose<TransactionBalance>((ref) async {
  final repo = ref.watch(transactionRepositoryProvider);
  final filter = ref.watch(financeDateFilterProvider);
  return await repo.getBalance(month: filter.month, year: filter.year);
});

// Transactions List Provider
final transactionsProvider = FutureProvider.autoDispose<List<Transaction>>((ref) async {
  final repo = ref.watch(transactionRepositoryProvider);
  // Get all for the sake of the list, or we could pass filters. The backend returns 100 limit by default.
  // Actually, we can get all transactions and group them in UI.
  return await repo.getTransactions();
});
