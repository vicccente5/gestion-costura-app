import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final secureStorageProvider = Provider<AppSecureStorage>((ref) {
  return AppSecureStorage();
});

class AppSecureStorage {
  final _storage = const FlutterSecureStorage();
  final Map<String, String> _memoryFallback = {};

  Future<void> write({required String key, required String? value}) async {
    if (value == null) {
      await delete(key: key);
      return;
    }
    _memoryFallback[key] = value;
    try {
      await _storage.write(key: key, value: value).timeout(const Duration(seconds: 2));
    } on PlatformException catch (_) {
      // Ignorar, ya guardado en memoria
    } on Exception catch (_) {
      // Ignorar
    }
  }

  Future<String?> read({required String key}) async {
    try {
      final value = await _storage.read(key: key).timeout(const Duration(seconds: 2));
      return value ?? _memoryFallback[key];
    } on PlatformException catch (_) {
      return _memoryFallback[key];
    } on Exception catch (_) {
      return _memoryFallback[key];
    }
  }

  Future<void> delete({required String key}) async {
    _memoryFallback.remove(key);
    try {
      await _storage.delete(key: key).timeout(const Duration(seconds: 2));
    } on PlatformException catch (_) {
      // Ignorar
    } on Exception catch (_) {
      // Ignorar
    }
  }
}
