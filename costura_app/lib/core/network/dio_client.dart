import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../config/env_config.dart';
import '../utils/secure_storage.dart';
import 'jwt_interceptor.dart';
import 'security_interceptor.dart';

final dioProvider = Provider<Dio>((ref) {
  final secureStorage = ref.watch(secureStorageProvider);

  // Cliente para refescar tokens sin interceptores cíclicos
  final tokenDio = Dio(BaseOptions(
    baseUrl: EnvConfig.baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 10),
  ));

  final dio = Dio(BaseOptions(
    baseUrl: EnvConfig.baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 10),
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
  ));

  dio.interceptors.addAll([
    SecurityInterceptor(), // Public Key Pinning en producción
    JwtInterceptor(
      secureStorage: secureStorage,
      dio: dio,
      tokenDio: tokenDio,
      ref: ref,
    ),
  ]);

  return dio;
});
