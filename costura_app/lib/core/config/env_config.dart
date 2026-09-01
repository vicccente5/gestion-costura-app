class EnvConfig {
  static const String baseUrl = String.fromEnvironment(
    'BASE_URL',
    // Valor por defecto para Android Emulator. Si usas iOS, cambiar a http://localhost:8080
    // O mejor aún, pasarlo al compilar: flutter run --dart-define=BASE_URL=http://...
    defaultValue: 'http://10.0.2.2:8080', 
  );
}
