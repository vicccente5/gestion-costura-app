import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:http_certificate_pinning/http_certificate_pinning.dart';

class SecurityInterceptor extends Interceptor {
  // Lista de SHA-256 fingerprints de las claves públicas del servidor.
  // IMPORTANTE: Estos fingerprints deben pertenecer a la CLAVE PÚBLICA, no al certificado completo.
  // Ejemplo ficticio, en producción reemplazar con el fingerprint real del dominio.
  final List<String> allowedFingerprints = [
    '00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF',
  ];

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) async {
    // Si estamos en entorno de depuración (local), o la URL es localhost, ignorar pinning.
    if (kDebugMode || options.baseUrl.contains('localhost') || options.baseUrl.contains('10.0.2.2')) {
      return handler.next(options);
    }

    try {
      // Validar el certificado usando Public Key Pinning (SHA-256)
      // Esta verificación abortará la conexión si el fingerprint no coincide (ej. proxy MitM falso)
      final secure = await HttpCertificatePinning.check(
        serverURL: options.baseUrl,
        headerHttp: options.headers.map((key, value) => MapEntry(key, value.toString())),
        sha: SHA.SHA256,
        allowedSHAFingerprints: allowedFingerprints,
        timeout: 50,
      );

      if (secure.contains('CONNECTION_SECURE')) {
        return handler.next(options);
      } else {
        return handler.reject(
          DioException(
            requestOptions: options,
            type: DioExceptionType.connectionError,
            error: 'Insecure Connection: Pinning Failed',
          ),
        );
      }
    } catch (e) {
      return handler.reject(
        DioException(
          requestOptions: options,
          type: DioExceptionType.connectionError,
          error: 'Certificate Pinning Exception: $e',
        ),
      );
    }
  }
}
