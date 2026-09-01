class Transaction {
  final String id;
  final String userId;
  final String type;   // 'ingreso' o 'gasto'  — campo interno Flutter
  final String source; // 'manual' o 'order'    — campo interno Flutter
  final double amount;
  final String description;
  final DateTime date;
  final String? orderId;
  final DateTime createdAt;

  Transaction({
    required this.id,
    required this.userId,
    required this.type,
    required this.source,
    required this.amount,
    required this.description,
    required this.date,
    this.orderId,
    required this.createdAt,
  });

  factory Transaction.fromJson(Map<String, dynamic> json) {
    // El backend (Go) responde con claves en ESPAÑOL según los json tags del struct:
    //   tipo, monto (int64 CLP), descripcion, fecha, source, user_id, created_at
    return Transaction(
      id: json['id'],
      userId: json['user_id'],
      type: json['tipo'],                            // ← era 'type'
      source: json['source'],
      amount: (json['monto'] as num).toDouble(),     // ← era 'amount'
      description: json['descripcion'],              // ← era 'description'
      date: DateTime.parse(json['fecha']),           // ← era 'date'
      orderId: json['order_id'],
      createdAt: DateTime.parse(json['created_at']),
    );
  }
}

class TransactionBalance {
  final double ingresos;
  final double gastos;
  final double balance;

  TransactionBalance({
    required this.ingresos,
    required this.gastos,
    required this.balance,
  });

  factory TransactionBalance.fromJson(Map<String, dynamic> json) {
    return TransactionBalance(
      ingresos: (json['total_ingresos'] as num?)?.toDouble() ?? 0.0,
      gastos: (json['total_gastos'] as num?)?.toDouble() ?? 0.0,
      balance: (json['balance'] as num?)?.toDouble() ?? 0.0,
    );
  }
}
