import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../data/repositories/material_repository.dart';
import '../../../domain/models/material.dart';

final materialSearchQueryProvider = StateProvider<String>((ref) => '');

final inventoryListProvider = FutureProvider.autoDispose<List<MaterialModel>>((ref) async {
  final repository = ref.watch(materialRepositoryProvider);
  final query = ref.watch(materialSearchQueryProvider);
  return repository.getMaterials(q: query);
});

final materialDetailProvider = FutureProvider.family.autoDispose<MaterialModel, String>((ref, id) async {
  final repository = ref.watch(materialRepositoryProvider);
  return repository.getMaterial(id);
});
