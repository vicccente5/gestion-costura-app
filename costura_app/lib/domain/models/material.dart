class MaterialModel {
  final String id;
  final String nombre;
  final String? categoria;
  final String unidadMedida;
  final double stockActual;
  final double stockMinimo;
  final double costoUnitario;

  MaterialModel({
    required this.id,
    required this.nombre,
    this.categoria,
    required this.unidadMedida,
    required this.stockActual,
    required this.stockMinimo,
    required this.costoUnitario,
  });

  factory MaterialModel.fromJson(Map<String, dynamic> json) {
    return MaterialModel(
      id: json['id'],
      nombre: json['nombre'],
      categoria: json['categoria'],
      unidadMedida: json['unidad'],
      stockActual: (json['stock_actual'] as num).toDouble(),
      stockMinimo: (json['stock_minimo'] as num).toDouble(),
      costoUnitario: json['costo_unitario'] != null ? (json['costo_unitario'] as num).toDouble() : 0.0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'nombre': nombre,
      'categoria': categoria,
      'unidad': unidadMedida,
      'stock_actual': stockActual,
      'stock_minimo': stockMinimo,
      'costo_unitario': costoUnitario,
    };
  }

  bool get isLowStock => stockActual <= stockMinimo;
}
