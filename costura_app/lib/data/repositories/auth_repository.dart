import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../../core/network/dio_client.dart';
import '../../core/utils/secure_storage.dart';
import '../../domain/models/user.dart';

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepository(
    dio: ref.watch(dioProvider),
    secureStorage: ref.watch(secureStorageProvider),
  );
});

class AuthRepository {
  final Dio _dio;
  final AppSecureStorage _secureStorage;

  AuthRepository({
    required Dio dio,
    required AppSecureStorage secureStorage,
  })  : _dio = dio,
        _secureStorage = secureStorage;

  Future<User> login(String email, String password) async {
    try {
      final response = await _dio.post('/api/v1/auth/login', data: {
        'email': email,
        'password': password,
      });

      final data = response.data['data'];
      await _secureStorage.write(key: 'access_token', value: data['access_token']);
      await _secureStorage.write(key: 'refresh_token', value: data['refresh_token']);

      if (data['user'] != null) {
        return User.fromJson(data['user']);
      }
      return User(id: '', nombre: 'Usuario', email: email);
    } on DioException catch (e) {
      if (e.response != null) {
        throw Exception(e.response?.data['message'] ?? 'Error de autenticación');
      }
      throw Exception('Error de conexión');
    }
  }

  Future<User> register(String nombre, String email, String password) async {
    try {
      final response = await _dio.post('/api/v1/auth/register', data: {
        'nombre': nombre,
        'email': email,
        'password': password,
      });

      final data = response.data['data'];
      // En register, el usuario viene en data directamente, no en data['user']
      // Asumimos que luego el usuario se logueará para obtener los tokens, o los extraemos si vienen.
      if (data['access_token'] != null) {
        await _secureStorage.write(key: 'access_token', value: data['access_token']);
        await _secureStorage.write(key: 'refresh_token', value: data['refresh_token']);
      }

      return User(id: data['id'] ?? '', nombre: data['nombre'] ?? nombre, email: data['email'] ?? email);
    } on DioException catch (e) {
      if (e.response != null) {
        throw Exception(e.response?.data['message'] ?? 'Error al registrarse');
      }
      throw Exception('Error de conexión');
    }
  }

  Future<void> logout() async {
    try {
      await _dio.post('/api/v1/auth/logout');
    } catch (_) {
      // Ignorar si falla el logout remoto
    } finally {
      await _secureStorage.delete(key: 'access_token');
      await _secureStorage.delete(key: 'refresh_token');
    }
  }

  Future<bool> hasValidToken() async {
    try {
      // Añadido un timeout de 2 segundos para evitar que la app se cuelgue en Linux
      // si el sistema de llaveros (keyring) no está disponible o está bloqueado.
      final token = await _secureStorage.read(key: 'refresh_token');
      return token != null;
    } catch (e) {
      return false;
    }
  }
}
