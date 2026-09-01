import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../config/env_config.dart';
import '../utils/secure_storage.dart';

class JwtInterceptor extends Interceptor {
  final AppSecureStorage secureStorage;
  final Dio dio; // Referencia al cliente principal
  final Dio tokenDio; // Cliente independiente para refresh token (sin interceptores)
  final ProviderRef ref;

  bool _isRefreshing = false;
  final List<RequestOptions> _failedRequests = [];

  JwtInterceptor({
    required this.secureStorage,
    required this.dio,
    required this.tokenDio,
    required this.ref,
  });

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) async {
    // Si la ruta no necesita autenticación, continua.
    if (options.path.contains('/auth/login') || options.path.contains('/auth/register')) {
      return handler.next(options);
    }

    final accessToken = await secureStorage.read(key: 'access_token');
    if (accessToken != null) {
      options.headers['Authorization'] = 'Bearer $accessToken';
    }

    return handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    // Si no es un 401 o la petición original fue de refresh, devolvemos el error.
    if (err.response?.statusCode != 401 || err.requestOptions.path.contains('/auth/refresh')) {
      return handler.next(err);
    }

    final refreshToken = await secureStorage.read(key: 'refresh_token');
    if (refreshToken == null) {
      _logout();
      return handler.next(err);
    }

    if (!_isRefreshing) {
      _isRefreshing = true;
      _failedRequests.add(err.requestOptions);

      try {
        final response = await tokenDio.post(
          '${EnvConfig.baseUrl}/api/v1/auth/refresh',
          data: {'refresh_token': refreshToken},
        );

        if (response.statusCode == 200) {
          final newAccessToken = response.data['data']['access_token'];
          final newRefreshToken = response.data['data']['refresh_token'];

          await secureStorage.write(key: 'access_token', value: newAccessToken);
          await secureStorage.write(key: 'refresh_token', value: newRefreshToken);

          _isRefreshing = false;

          // Reintentar las peticiones fallidas
          for (final options in _failedRequests) {
            options.headers['Authorization'] = 'Bearer $newAccessToken';
            final retryResponse = await dio.fetch(options);
            // Esto solo reintenta la última, en una app real se necesita un Completer para cada una
            // Para simplificar, asumimos que no hay concurrencia extrema
            handler.resolve(retryResponse);
          }
          _failedRequests.clear();
          return;
        } else {
          _logout();
          return handler.next(err);
        }
      } catch (e) {
        _isRefreshing = false;
        _logout();
        return handler.next(err);
      }
    } else {
      // Si ya se está refrescando, no implementamos encolamiento complejo aquí,
      // simplemente fallamos. Una implementación pro encolaría y resolvería después.
      return handler.next(err);
    }
  }

  void _logout() async {
    await secureStorage.delete(key: 'access_token');
    await secureStorage.delete(key: 'refresh_token');
    // TODO: Invalidad auth provider state para forzar redirección
  }
}
