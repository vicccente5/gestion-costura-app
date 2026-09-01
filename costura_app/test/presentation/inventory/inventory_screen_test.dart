import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:costura_app/presentation/inventory/screens/inventory_screen.dart';
import 'package:costura_app/presentation/inventory/providers/inventory_provider.dart';
import 'package:costura_app/domain/models/material.dart' as domain;

void main() {
  testWidgets('InventoryScreen muestra badge ROJO cuando el stock es bajo', (WidgetTester tester) async {
    // 1. Arrange: Datos de prueba (1 material con stock bajo)
    final List<domain.MaterialModel> mockMaterials = [
      domain.MaterialModel(
        id: '1',
        nombre: 'Tela de Algodón',
        unidadMedida: 'metros',
        stockActual: 5,  // Stock actual
        stockMinimo: 10, // Stock mínimo (es mayor al actual, entonces es bajo)
        costoUnitario: 0,
      )
    ];

    // Override del provider para devolver los datos mockeados
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          inventoryListProvider.overrideWith((ref) => Future.value(mockMaterials)),
        ],
        child: const MaterialApp(
          home: InventoryScreen(),
        ),
      ),
    );

    // Esperar a que el FutureProvider resuelva
    await tester.pumpAndSettle();

    // 2. Act & Assert: Verificar que aparece "STOCK BAJO"
    expect(find.text('Tela de Algodón'), findsOneWidget);
    expect(find.text('STOCK BAJO'), findsOneWidget);
    expect(find.text('Stock: 5.0 metros (Min: 10.0)'), findsOneWidget);
  });

  testWidgets('InventoryScreen NO muestra badge cuando el stock es suficiente', (WidgetTester tester) async {
    // 1. Arrange: Datos de prueba (1 material con buen stock)
    final List<domain.MaterialModel> mockMaterials = [
      domain.MaterialModel(
        id: '2',
        nombre: 'Hilo Negro',
        unidadMedida: 'conos',
        stockActual: 50,  // Stock actual alto
        stockMinimo: 5,   // Stock mínimo bajo
        costoUnitario: 0,
      )
    ];

    // Override del provider para devolver los datos mockeados
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          inventoryListProvider.overrideWith((ref) => Future.value(mockMaterials)),
        ],
        child: const MaterialApp(
          home: InventoryScreen(),
        ),
      ),
    );

    // Esperar a que el FutureProvider resuelva
    await tester.pumpAndSettle();

    // 2. Act & Assert: Verificar que NO aparece "STOCK BAJO"
    expect(find.text('Hilo Negro'), findsOneWidget);
    expect(find.text('STOCK BAJO'), findsNothing);
    expect(find.text('Stock: 50.0 conos (Min: 5.0)'), findsOneWidget);
  });
}
