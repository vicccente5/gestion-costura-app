import 'client.dart';
import 'material.dart';

class Order {
  final String id;
  final String clientId;
  final String descripcion;
  final String estado;
  final DateTime? fechaEntrega;   // nullable — el backend lo envía como omitempty
  final double precioVenta;       // reemplaza margenPorcentaje — mapeado desde 'precio_venta'
  final double horas;
  final double tarifaHora;
  final String? notas;
  final DateTime createdAt;

  final Client? client;
  final List<OrderMaterial>? materiales;

  Order({
    required this.id,
    required this.clientId,
    required this.descripcion,
    required this.estado,
    this.fechaEntrega,
    required this.precioVenta,
    required this.horas,
    required this.tarifaHora,
    this.notas,
    required this.createdAt,
    this.client,
    this.materiales,
  });

  factory Order.fromJson(Map<String, dynamic> json) {
    // El backend (Go) responde con los json tags del domain.Order:
    //   descripcion, estado, horas, tarifa_hora, precio_venta, fecha_entrega (omitempty),
    //   notas (omitempty), client_id, client (omitempty), materials (omitempty), created_at
    return Order(
      id: json['id'],
      clientId: json['client_id'],
      descripcion: json['descripcion'],
      estado: json['estado'],
      // fecha_entrega es omitempty → puede llegar como null
      fechaEntrega: json['fecha_entrega'] != null
          ? DateTime.tryParse(json['fecha_entrega'])
          : null,
      // precio_venta es int64 en Go
      precioVenta: json['precio_venta'] != null
          ? (json['precio_venta'] as num).toDouble()
          : 0.0,
      horas: json['horas'] != null ? (json['horas'] as num).toDouble() : 0.0,
      tarifaHora: json['tarifa_hora'] != null
          ? (json['tarifa_hora'] as num).toDouble()
          : 0.0,
      notas: json['notas'],
      createdAt: DateTime.parse(json['created_at']),
      client: json['client'] != null ? Client.fromJson(json['client']) : null,
      // IMPORTANTE: el backend envía 'materials' (no 'materiales')
      materiales: json['materials'] != null
          ? (json['materials'] as List)
              .map((i) => OrderMaterial.fromJson(i))
              .toList()
          : null,
    );
  }

  /// Costo total calculado en cliente: materiales + mano de obra
  double get costoTotal {
    final costoMateriales = materiales?.fold<double>(
            0, (sum, m) => sum + (m.cantidad * m.costoUnitarioSnapshot)) ??
        0.0;
    final costoManoObra = horas * tarifaHora;
    return costoMateriales + costoManoObra;
  }

  /// Ganancia neta: precio de venta - costo total
  double get gananciaNeta => precioVenta - costoTotal;
}

class OrderMaterial {
  final String id;
  final String orderId;
  final String materialId;
  final double cantidad;                  // era 'cantidadUsada' — backend envía 'cantidad'
  final double costoUnitarioSnapshot;    // era 'costoCalculado' — backend envía 'costo_unitario_snapshot'

  final MaterialModel? material;

  OrderMaterial({
    required this.id,
    required this.orderId,
    required this.materialId,
    required this.cantidad,
    required this.costoUnitarioSnapshot,
    this.material,
  });

  factory OrderMaterial.fromJson(Map<String, dynamic> json) {
    return OrderMaterial(
      id: json['id'],
      orderId: json['order_id'],
      materialId: json['material_id'],
      // IMPORTANTE: el backend envía 'cantidad' (no 'cantidad_usada')
      cantidad: (json['cantidad'] as num).toDouble(),
      // IMPORTANTE: el backend envía 'costo_unitario_snapshot' (no 'costo_calculado')
      costoUnitarioSnapshot: (json['costo_unitario_snapshot'] as num).toDouble(),
      material: json['material'] != null ? MaterialModel.fromJson(json['material']) : null,
    );
  }

  /// Costo total de este material en el encargo
  double get costoTotal => cantidad * costoUnitarioSnapshot;
}
