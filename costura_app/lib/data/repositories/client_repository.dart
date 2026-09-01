import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/dio_client.dart';
import '../../domain/models/client.dart';

final clientRepositoryProvider = Provider<ClientRepository>((ref) {
  return ClientRepository(dio: ref.watch(dioProvider));
});

class ClientRepository {
  final Dio _dio;

  ClientRepository({required Dio dio}) : _dio = dio;

  Future<List<Client>> getClients({int page = 1, int limit = 100, String q = ''}) async {
    try {
      final response = await _dio.get('/api/v1/clients', queryParameters: {
        'page': page,
        'limit': limit,
        if (q.isNotEmpty) 'q': q,
      });

      final List data = response.data['data']['data'];
      return data.map((json) => Client.fromJson(json)).toList();
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener clientes');
    }
  }

  Future<Client> getClient(String id) async {
    try {
      final response = await _dio.get('/api/v1/clients/$id');
      return Client.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener el cliente');
    }
  }

  Future<Client> createClient(String nombre, String? telefono, String? email) async {
    try {
      final response = await _dio.post('/api/v1/clients', data: {
        'nombre': nombre,
        if (telefono != null && telefono.isNotEmpty) 'telefono': telefono,
        if (email != null && email.isNotEmpty) 'email': email,
      });
      return Client.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al crear cliente');
    }
  }

  Future<Client> updateClient(String id, String nombre, String? telefono, String? email) async {
    try {
      final response = await _dio.put('/api/v1/clients/$id', data: {
        'nombre': nombre,
        'telefono': telefono,
        'email': email,
      });
      return Client.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al actualizar cliente');
    }
  }

  Future<void> deleteClient(String id) async {
    try {
      await _dio.delete('/api/v1/clients/$id');
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al eliminar cliente');
    }
  }
}
