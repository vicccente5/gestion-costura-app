import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/dio_client.dart';
import '../../domain/models/material.dart';

final materialRepositoryProvider = Provider<MaterialRepository>((ref) {
  return MaterialRepository(dio: ref.watch(dioProvider));
});

class MaterialRepository {
  final Dio _dio;

  MaterialRepository({required Dio dio}) : _dio = dio;

  Future<List<MaterialModel>> getMaterials({int page = 1, int limit = 100, String q = ''}) async {
    try {
      final response = await _dio.get('/api/v1/materials', queryParameters: {
        'page': page,
        'limit': limit,
        if (q.isNotEmpty) 'q': q,
      });

      final List data = response.data['data']['data'];
      return data.map((json) => MaterialModel.fromJson(json)).toList();
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener materiales');
    }
  }

  Future<MaterialModel> getMaterial(String id) async {
    try {
      final response = await _dio.get('/api/v1/materials/$id');
      return MaterialModel.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener el material');
    }
  }

  Future<MaterialModel> createMaterial(String nombre, String categoria, String unidad, double stockMinimo, double costoUnitario) async {
    try {
      final response = await _dio.post('/api/v1/materials', data: {
        'nombre': nombre,
        'categoria': categoria,
        'unidad': unidad,
        'stock_minimo': stockMinimo,
        'costo_unitario': costoUnitario.toInt(), // El backend espera int64
      });
      return MaterialModel.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al crear material');
    }
  }

  Future<MaterialModel> updateMaterial(String id, String nombre, String categoria, String unidad, double stockMinimo) async {
    try {
      final response = await _dio.put('/api/v1/materials/$id', data: {
        'nombre': nombre,
        'categoria': categoria,
        'unidad': unidad,
        'stock_minimo': stockMinimo,
      });
      return MaterialModel.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al actualizar material');
    }
  }

  Future<void> deleteMaterial(String id) async {
    try {
      await _dio.delete('/api/v1/materials/$id');
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al eliminar material');
    }
  }

  // Compras de material — El backend espera: cantidad, precio_unitario (por unidad), fecha (YYYY-MM-DD)
  Future<void> registrarCompra(String materialId, double cantidad, double precioUnitario, String fecha) async {
    try {
      await _dio.post('/api/v1/materials/$materialId/purchases', data: {
        'cantidad': cantidad,
        'precio_unitario': precioUnitario.toInt(), // El backend espera int64 (CLP)
        'fecha': fecha,
      });
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al registrar compra');
    }
  }
}
