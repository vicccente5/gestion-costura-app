import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../presentation/auth/providers/auth_provider.dart';
import '../../presentation/auth/screens/login_screen.dart';
import '../../presentation/auth/screens/register_screen.dart';
import '../../presentation/home/screens/home_screen.dart';
import '../../presentation/finance/screens/finance_screen.dart';
import '../../presentation/finance/screens/transaction_form_screen.dart';
import '../../presentation/home/screens/main_navigation_screen.dart';
import '../../presentation/splash/screens/splash_screen.dart';
import '../../presentation/clients/screens/clients_list_screen.dart';
import '../../presentation/clients/screens/client_form_screen.dart';
import '../../presentation/clients/screens/client_detail_screen.dart';
import '../../presentation/inventory/screens/inventory_screen.dart';
import '../../presentation/inventory/screens/material_form_screen.dart';
import '../../presentation/inventory/screens/material_detail_screen.dart';
import '../../presentation/orders/screens/orders_screen.dart';
import '../../presentation/orders/screens/order_form_screen.dart';
import '../../presentation/orders/screens/order_detail_screen.dart';

final _rootNavigatorKey = GlobalKey<NavigatorState>();
final _shellNavigatorInicioKey = GlobalKey<NavigatorState>(debugLabel: 'inicio');
final _shellNavigatorInventarioKey = GlobalKey<NavigatorState>(debugLabel: 'inventario');
final _shellNavigatorEncargosKey = GlobalKey<NavigatorState>(debugLabel: 'encargos');
final _shellNavigatorFinanzasKey = GlobalKey<NavigatorState>(debugLabel: 'finanzas');

final routerProvider = Provider<GoRouter>((ref) {
  // Un ValueNotifier para forzar la re-evaluación del router cuando cambie el estado
  final listenable = ValueNotifier<bool>(false);

  ref.listen<AuthState>(authProvider, (previous, next) {
    if (previous?.status != next.status) {
      listenable.value = !listenable.value;
    }
  });

  return GoRouter(
    navigatorKey: _rootNavigatorKey,
    initialLocation: '/splash',
    refreshListenable: listenable,
    redirect: (context, state) {
      final authState = ref.read(authProvider);

      final isAuth = authState.status == AuthStatus.authenticated;
      final isChecking = authState.status == AuthStatus.checking;
      final isSplash = state.uri.path == '/splash';
      
      final isGoingToLogin = state.uri.path == '/login';
      final isGoingToRegister = state.uri.path == '/register';

      if (isChecking) return isSplash ? null : '/splash';

      if (!isAuth) {
        if (isGoingToLogin || isGoingToRegister) return null;
        return '/login';
      }

      if (isAuth && (isGoingToLogin || isGoingToRegister || isSplash)) {
        return '/';
      }

      return null;
    },
    routes: [
      GoRoute(
        path: '/splash',
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/register',
        builder: (context, state) => const RegisterScreen(),
      ),
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) {
          return MainNavigationScreen(navigationShell: navigationShell);
        },
        branches: [
          StatefulShellBranch(
            navigatorKey: _shellNavigatorInicioKey,
            routes: [
              GoRoute(
                path: '/',
                builder: (context, state) => const HomeScreen(),
                routes: [
                  GoRoute(
                    path: 'clients',
                    builder: (context, state) => const ClientsListScreen(),
                    routes: [
                      GoRoute(
                        path: 'new',
                        builder: (context, state) => const ClientFormScreen(),
                      ),
                      GoRoute(
                        path: ':id',
                        builder: (context, state) => ClientDetailScreen(id: state.pathParameters['id']!),
                        routes: [
                          GoRoute(
                            path: 'edit',
                            builder: (context, state) => ClientFormScreen(clientId: state.pathParameters['id']),
                          ),
                        ],
                      ),
                    ],
                  ),
                ],
              ),
            ],
          ),
          StatefulShellBranch(
            navigatorKey: _shellNavigatorInventarioKey,
            routes: [
              GoRoute(
                path: '/inventory',
                builder: (context, state) => const InventoryScreen(),
                routes: [
                  GoRoute(
                    path: 'new',
                    builder: (context, state) => const MaterialFormScreen(),
                  ),
                  GoRoute(
                    path: ':id',
                    builder: (context, state) => MaterialDetailScreen(id: state.pathParameters['id']!),
                    routes: [
                      GoRoute(
                        path: 'edit',
                        builder: (context, state) => MaterialFormScreen(materialId: state.pathParameters['id']),
                      ),
                    ],
                  ),
                ],
              ),
            ],
          ),
          StatefulShellBranch(
            navigatorKey: _shellNavigatorEncargosKey,
            routes: [
              GoRoute(
                path: '/orders',
                builder: (context, state) => const OrdersScreen(),
                routes: [
                  GoRoute(
                    path: 'new',
                    builder: (context, state) => const OrderFormScreen(),
                  ),
                  GoRoute(
                    path: ':id',
                    builder: (context, state) => OrderDetailScreen(id: state.pathParameters['id']!),
                    routes: [
                      GoRoute(
                        path: 'edit',
                        builder: (context, state) => OrderFormScreen(orderId: state.pathParameters['id']),
                      ),
                    ],
                  ),
                ],
              ),
            ],
          ),
          StatefulShellBranch(
            navigatorKey: _shellNavigatorFinanzasKey,
            routes: [
              GoRoute(
                path: '/finance',
                builder: (context, state) => const FinanceScreen(),
                routes: [
                  GoRoute(
                    path: 'new',
                    builder: (context, state) => const TransactionFormScreen(),
                  ),
                  GoRoute(
                    path: ':id/edit',
                    builder: (context, state) => TransactionFormScreen(transactionId: state.pathParameters['id']),
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
    ],
  );
});
