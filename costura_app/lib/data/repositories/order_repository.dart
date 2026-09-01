import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/dio_client.dart';
import '../../domain/models/order.dart';

final orderRepositoryProvider = Provider<OrderRepository>((ref) {
  return OrderRepository(dio: ref.watch(dioProvider));
});

class OrderRepository {
  final Dio _dio;

  OrderRepository({required Dio dio}) : _dio = dio;

  Future<List<Order>> getOrders({int page = 1, int limit = 100, String q = '', String estado = ''}) async {
    try {
      final response = await _dio.get('/api/v1/orders', queryParameters: {
        'page': page,
        'limit': limit,
        if (q.isNotEmpty) 'q': q,
        if (estado.isNotEmpty) 'estado': estado,
      });

      final List data = response.data['data']['data'];
      return data.map((json) => Order.fromJson(json)).toList();
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener encargos');
    }
  }

  Future<Order> getOrder(String id) async {
    try {
      final response = await _dio.get('/api/v1/orders/$id');
      return Order.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener el encargo');
    }
  }

  Future<Order> createOrder(
      String clientId,
      String descripcion,
      String fechaEntrega,
      double precioVenta,
      double horas,
      double tarifaHora,
      String notas,
      List<Map<String, dynamic>> materials) async {
    try {
      final response = await _dio.post('/api/v1/orders', data: {
        'client_id': clientId,
        'descripcion': descripcion,
        'fecha_entrega': fechaEntrega,
        'precio_venta': precioVenta.toInt(),
        'horas': horas,
        'tarifa_hora': tarifaHora.toInt(),
        'notas': notas.isEmpty ? null : notas,
        'materials': materials, // [{'material_id': '...', 'cantidad': 1.5}]
      });
      return Order.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al crear encargo');
    }
  }

  Future<Order> updateOrder(
      String id,
      String descripcion,
      String fechaEntrega,
      double precioVenta,
      double horas,
      double tarifaHora,
      String notas) async {
    try {
      final response = await _dio.put('/api/v1/orders/$id', data: {
        // NOTA: el PUT endpoint (orderUpdateRequest) solo acepta estos campos.
        // Para cambiar estado se usa el endpoint PATCH /orders/:id/status
        'descripcion': descripcion,
        'fecha_entrega': fechaEntrega,
        'precio_venta': precioVenta.toInt(),
        'horas': horas,
        'tarifa_hora': tarifaHora.toInt(),
        'notas': notas.isEmpty ? null : notas,
      });
      return Order.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al actualizar encargo');
    }
  }

  /// Cambia el estado de un encargo usando el endpoint dedicado PATCH /orders/:id/status
  Future<Order> changeOrderStatus(String id, String nuevoEstado) async {
    try {
      final response = await _dio.patch('/api/v1/orders/$id/status', data: {
        'estado': nuevoEstado,
      });
      return Order.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al cambiar estado del encargo');
    }
  }

  Future<void> deleteOrder(String id) async {
    try {
      await _dio.delete('/api/v1/orders/$id');
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al eliminar encargo');
    }
  }

  // Materiales del Encargo
  Future<void> addMaterialToOrder(String orderId, String materialId, double cantidad) async {
    try {
      await _dio.post('/api/v1/orders/$orderId/materials', data: {
        'material_id': materialId,
        'cantidad': cantidad, // el backend (orderMaterialRequest) espera 'cantidad' no 'cantidad_usada'
      });
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al asignar material');
    }
  }

  Future<void> removeMaterialFromOrder(String orderId, String materialId) async {
    try {
      await _dio.delete('/api/v1/orders/$orderId/materials/$materialId');
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al remover material');
    }
  }
}
