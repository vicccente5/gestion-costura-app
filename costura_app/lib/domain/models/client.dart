class Client {
  final String id;
  final String nombre;
  final String? telefono;
  final String? email;
  final DateTime createdAt;

  Client({
    required this.id,
    required this.nombre,
    this.telefono,
    this.email,
    required this.createdAt,
  });

  factory Client.fromJson(Map<String, dynamic> json) {
    return Client(
      id: json['id'],
      nombre: json['nombre'],
      telefono: json['telefono'],
      email: json['email'],
      createdAt: DateTime.parse(json['created_at']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'nombre': nombre,
      'telefono': telefono,
      'email': email,
    };
  }
}
