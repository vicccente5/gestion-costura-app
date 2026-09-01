import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/dio_client.dart';
import '../../domain/models/transaction.dart';

final transactionRepositoryProvider = Provider<TransactionRepository>((ref) {
  return TransactionRepository(dio: ref.watch(dioProvider));
});

class TransactionRepository {
  final Dio _dio;

  TransactionRepository({required Dio dio}) : _dio = dio;

  Future<List<Transaction>> getTransactions({int page = 1, int limit = 100, String type = '', String source = ''}) async {
    try {
      final response = await _dio.get('/api/v1/transactions', queryParameters: {
        'page': page,
        'limit': limit,
        if (type.isNotEmpty) 'type': type,
        if (source.isNotEmpty) 'source': source,
      });

      final List data = response.data['data']['data'];
      return data.map((json) => Transaction.fromJson(json)).toList();
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener transacciones');
    }
  }

  Future<TransactionBalance> getBalance({int year = 0, int month = 0}) async {
    try {
      final query = <String, dynamic>{};
      if (year > 0 && month > 0) {
        final formattedMonth = month.toString().padLeft(2, '0');
        query['month'] = '$year-$formattedMonth';
      }

      final response = await _dio.get('/api/v1/transactions/balance', queryParameters: query);
      return TransactionBalance.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener balance');
    }
  }

  Future<Transaction> createTransaction(String type, double amount, String description, String date) async {
    try {
      final response = await _dio.post('/api/v1/transactions', data: {
        'tipo': type,
        'monto': amount.toInt(), // El backend espera int64
        'descripcion': description,
        'fecha': date,
      });
      return Transaction.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al crear transacción');
    }
  }

  Future<Transaction> getTransaction(String id) async {
    try {
      final response = await _dio.get('/api/v1/transactions/$id');
      return Transaction.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al obtener transacción');
    }
  }

  Future<Transaction> updateTransaction(String id, String type, double amount, String description, String date) async {
    try {
      final response = await _dio.put('/api/v1/transactions/$id', data: {
        'tipo': type,
        'monto': amount.toInt(),
        'descripcion': description,
        'fecha': date,
      });
      return Transaction.fromJson(response.data['data']);
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al actualizar transacción');
    }
  }

  Future<void> deleteTransaction(String id) async {
    try {
      await _dio.delete('/api/v1/transactions/$id');
    } on DioException catch (e) {
      throw Exception(e.response?.data['message'] ?? 'Error al eliminar transacción');
    }
  }
}
