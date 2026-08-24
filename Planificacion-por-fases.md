# 🧵 Planificación Detallada por Fases — App Gestión de Taller de Costura

> **v2.0 — Auditada y corregida.** 14 problemas identificados y corregidos. Ver tabla al final.

---

## 📋 Análisis del Prompt Original

### ✅ Puntos Fuertes
- Define claramente el stack: **Go, PostgreSQL, GORM**.
- Describe bien el dominio del negocio.
- Divide el trabajo en fases ordenadas.

### ⚠️ Debilidades — Ya Corregidas en v2.0

| Problema | Solución Aplicada |
|---|---|
| Sin tipo de app móvil definido | Flutter especificado |
| Sin autenticación ni usuarios | JWT + bcrypt en Fase 2 |
| Sin entorno de desarrollo | Docker desde Fase 1 |
| Fase de concurrencia vaga | Reemplazada por Testing + Despliegue |
| Sin manejo de errores | Requisito transversal explícito |
| Sin API versioning | /api/v1/ + estrategia en Fase 8 |
| Sin logging | Fase 7 dedicada |
| Sin módulo de reportes | Fase 6 dedicada |
| Sin definición de moneda | CLP en enteros desde Fase 1 |

---

## 🚀 Prompt Mejorado v2.0

Actúa como un Tech Lead experto en backend con Go (Golang).

Quiero construir el backend completo de una app móvil Flutter para gestión de un taller de costura. El backend expone una API REST segura.

**Stack:** Go 1.21+, PostgreSQL, GORM (queries) + golang-migrate (migraciones), Gin, JWT+bcrypt, Docker, zerolog, testify.

**Arquitectura:** cmd/, internal/domain/, internal/repository/, internal/service/, internal/handler/, internal/middleware/, config/, migrations/

**Módulos:**
1. Autenticación: JWT (access 15min + refresh 7d), todos los endpoints filtran por user_id del JWT
2. Clientes del taller: CRUD (nombre, teléfono, email), historial de encargos
3. Inventario: CRUD materiales, compras con promedio ponderado móvil, alertas bajo stock
4. Encargos: vinculados a cliente, asignación/edición/eliminación de materiales atómica, snapshot de costo, margen solo si precio_venta > 0, transacción automática al marcar entregado
5. Flujo de Caja: campo source (manual/order) para evitar duplicados en balance
6. Reportes: ganancias por período, materiales más usados, encargos más rentables

**Requisitos Transversales:** golang-migrate (NO AutoMigrate), errores centralizados, validación con go-playground/validator, paginación, /api/v1/, Rate Limiting en login, CLP en enteros, tests unitarios en cada fase.

**Metodología:** Fase a fase. NO avances hasta que yo lo indique. Código real, comentado y con decisiones explicadas. Al final de cada fase dime qué probar. Comienza con la Fase 1.

---

## 🗺️ Planificación Detallada por Fases

---

### FASE 0 — Prerrequisitos: Instalación y Configuración del Entorno

**Objetivo:** Tener todas las herramientas instaladas, configuradas y verificadas antes de escribir una sola línea de código del proyecto. Sin esta fase completa, las fases siguientes fallarán.

> Esta fase se hace UNA SOLA VEZ en la máquina de desarrollo.
> No es necesario repetirla al reanudar el proyecto.

---

**0A — Herramientas del Sistema Base**

- [ ] **Git** — control de versiones
  - Windows: https://git-scm.com/download/win
  - Verificar: `git --version`

- [ ] **Docker Desktop** — entorno de contenedores (PostgreSQL corre aquí)
  - Windows/Mac/Linux: https://www.docker.com/products/docker-desktop
  - Verificar: `docker --version` y `docker compose version`
  - ⚠️ En Windows: asegurarse de que WSL2 está habilitado (Docker lo requiere)

- [ ] **Make** (opcional pero muy recomendado para automatizar comandos Go)
  - Windows: instalar via Chocolatey `choco install make` o usar Git Bash
  - Linux/Mac: ya viene instalado
  - Verificar: `make --version`

---

**0B — Entorno de Backend: Go**

- [ ] **Go SDK 1.21+**
  - Descargar: https://go.dev/dl/
  - Windows: instalar el .msi y seguir el asistente
  - Verificar: `go version` (debe mostrar go1.21 o superior)
  - Verificar que el PATH incluye `$GOPATH/bin` (donde se instalan los binarios de Go)

- [ ] **Air** — hot reload para Go en desarrollo (se usa en docker-compose dev)
  - Instalar: `go install github.com/air-verse/air@latest`
  - Verificar: `air -v`

- [ ] **golang-migrate CLI** — herramienta de migraciones de la base de datos
  - Windows: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
  - Linux/Mac: `brew install golang-migrate` o el comando go install anterior
  - Verificar: `migrate -version`

- [ ] **swag CLI** — genera la documentación Swagger desde el código Go (se usa en Fase 8)
  - Instalar: `go install github.com/swaggo/swag/cmd/swag@latest`
  - Verificar: `swag --version`

- [ ] **golangci-lint** — linter para Go (se usa en Fase 7)
  - Windows/Linux/Mac: https://golangci-lint.run/usage/install/
  - Instalar recomendado: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
  - Verificar: `golangci-lint --version`

---

**0C — Entorno de Frontend: Flutter y Dart**

- [ ] **Flutter SDK** — incluye Dart automáticamente
  - Descargar: https://docs.flutter.dev/get-started/install
  - Windows: extraer el ZIP y agregar `flutter/bin` al PATH del sistema
  - Verificar instalación completa: `flutter doctor` (debe mostrar todo en verde)
  - ⚠️ `flutter doctor` indica exactamente qué falta configurar

- [ ] **Android Studio** (necesario para el emulador Android y el SDK de Android)
  - Descargar: https://developer.android.com/studio
  - Durante la instalación: instalar el **Android SDK** y crear un **AVD (emulador)**
  - Verificar en flutter doctor: `[✓] Android toolchain`

- [ ] **Xcode** (solo macOS — necesario para correr en simulador iOS)
  - Instalar desde la Mac App Store
  - Verificar en flutter doctor: `[✓] Xcode`
  - En Windows esta línea no aparece, es normal

- [ ] Aceptar licencias de Android SDK:
  - Ejecutar: `flutter doctor --android-licenses` y aceptar todo con `y`

---

**0D — IDE: Visual Studio Code y Extensiones**

- [ ] **Visual Studio Code**
  - Descargar: https://code.visualstudio.com/
  - Es el IDE recomendado para este proyecto (soporta Go y Flutter en el mismo editor)

- [ ] Extensiones obligatorias en VS Code:
  - `Go` (by Google) — soporte completo para Go: autocompletado, debugging, tests
  - `Flutter` (by Dart Code) — soporte completo para Flutter y Dart
  - `Dart` (by Dart Code) — viene junto con la extensión Flutter
  - `Docker` (by Microsoft) — manejo de contenedores desde el editor
  - `REST Client` (by Huachao Mao) — probar endpoints de la API desde VS Code sin Postman
  - `GitLens` (by GitKraken) — visualización avanzada de Git (opcional pero recomendado)

---

**0E — Herramientas de Base de Datos**

- [ ] **DBeaver Community** — cliente gráfico para inspeccionar PostgreSQL
  - Descargar: https://dbeaver.io/download/ (versión Community es gratuita)
  - Alternativa más ligera: **TablePlus** (https://tableplus.com/)
  - Se usa en los checkpoints de Fase 1 en adelante para verificar tablas y datos

- [ ] **Postman** (alternativa a REST Client de VS Code) — para probar endpoints de la API
  - Descargar: https://www.postman.com/downloads/
  - O usar directamente la extensión **REST Client** de VS Code (sin instalación adicional)

---

**0F — Herramientas de Seguridad (para Fase 10)**

> Estas herramientas solo se necesitan en la Fase 10. Se pueden instalar cuando llegue ese momento.

- [ ] **OWASP ZAP** — escáner de seguridad para la API (modo pasivo)
  - Descargar: https://www.zaproxy.org/download/
  - Gratuito y open source

- [ ] **certbot** — generación de certificados TLS con Let's Encrypt
  - Solo se instala en el servidor de producción (Linux), no en la máquina de desarrollo
  - Instalar en el servidor: `sudo apt install certbot python3-certbot-nginx`

---

**0G — Configuración de Git (primera vez)**

- [ ] Configurar identidad de Git (solo si es la primera vez):
  ```
  git config --global user.name "Tu Nombre"
  git config --global user.email "tu@email.com"
  ```
- [ ] Crear repositorio en GitHub/GitLab para el proyecto
- [ ] Clonar el repositorio localmente o inicializarlo:
  ```
  git init
  git remote add origin https://github.com/tu-usuario/costura-app.git
  ```

---

**Resumen de versiones mínimas requeridas:**

| Herramienta | Versión mínima | Comando de verificación |
|---|---|---|
| Go | 1.21 | `go version` |
| Flutter | 3.16+ | `flutter --version` |
| Dart | 3.2+ | `dart --version` (incluido en Flutter) |
| Docker | 24.0+ | `docker --version` |
| Docker Compose | 2.20+ | `docker compose version` |
| Git | 2.40+ | `git --version` |
| Air | latest | `air -v` |
| golang-migrate | v4 | `migrate -version` |
| swag | latest | `swag --version` |
| golangci-lint | 1.55+ | `golangci-lint --version` |

---

**Checkpoint de validación ✅**
- `go version` → Go 1.21 o superior
- `flutter doctor` → todos los ítems en verde (o solo advertencias menores)
- `docker --version` + `docker compose version` → ambos responden
- `migrate -version` → golang-migrate instalado
- `air -v` → Air instalado
- `swag --version` → swag CLI instalado
- `golangci-lint --version` → linter instalado
- VS Code abre un archivo .go con autocompletado funcionando
- VS Code abre un archivo .dart con autocompletado funcionando
- DBeaver o TablePlus puede conectarse a una base de datos PostgreSQL de prueba

**⚠️ Solo cuando TODOS los checkpoints estén en verde, pasar a la Fase 1.**

---

### FASE 1 — Fundamentos: Arquitectura, Modelos y Migraciones

**Objetivo:** Estructura del proyecto, modelos de la DB y sistema de migraciones correcto desde el inicio.

**Entregables:**
- [ ] Estructura de carpetas completa
- [ ] go.mod con todas las dependencias
- [ ] docker-compose.yml (desarrollo: PostgreSQL + Air para hot reload)
- [ ] docker-compose.prod.yml (producción: solo binario + DB, sin puertos innecesarios)
- [ ] .env.example con todas las variables
- [ ] .gitignore con .env incluido desde el primer commit ⚠️
- [ ] Modelos GORM en internal/domain/:
  - User (costurera — dueña de todos los datos)
  - Client (nombre, teléfono, email, user_id FK)
  - Material (nombre, categoría, unidad, stock_actual, stock_minimo, costo_unitario, user_id FK)
  - MaterialPurchase (material_id FK, cantidad, precio_total, fecha)
  - Order (client_id FK, descripción, estado, horas, tarifa_hora, precio_venta, user_id FK)
  - OrderMaterial (order_id FK, material_id FK, cantidad, costo_unitario_snapshot)
  - Transaction (tipo, monto, categoría, source, order_id FK opcional, user_id FK)
- [ ] Migraciones con golang-migrate/migrate:
  - 000001_create_users.up/down.sql
  - 000002_create_clients.up/down.sql
  - 000003_create_materials.up/down.sql
  - 000004_create_orders.up/down.sql
  - 000005_create_transactions.up/down.sql
- [ ] config/database.go — conexión GORM + ejecución de migraciones
- [ ] main.go inicial

> ⚠️ Por qué golang-migrate y NO AutoMigrate:
> AutoMigrate no elimina columnas ni altera tipos al cambiar un modelo.
> Puede dejar la DB inconsistente sin avisar.
> golang-migrate usa SQL versionado — explícito, reversible y trazable en Git.

> ⚠️ Por qué costo_unitario_snapshot en OrderMaterial:
> Si el costo del material cambia después, la rentabilidad histórica debe
> usar el precio que tenía cuando se asignó, no el precio actual.

**Checkpoint ✅**
- go build ./... compila sin errores
- docker-compose up -d levanta PostgreSQL
- Las 7 tablas existen en DB con FK correctas
- .env NO aparece en git status

**Errores a corregir antes de Fase 2:**
- Foreign keys en orden incorrecto en migraciones
- Precios como FLOAT en lugar de INTEGER
- user_id faltante en alguna tabla
- costo_unitario_snapshot faltante en OrderMaterial


---

### FASE 2 — Autenticación: Registro, Login y JWT

**Objetivo:** Sistema de autenticación completo. Toda la API protegida y todos los queries filtran por user_id.

**Entregables:**
- [ ] internal/repository/user_repository.go — interfaz + implementación
- [ ] internal/service/auth_service.go — registro y login
- [ ] internal/handler/auth_handler.go
- [ ] internal/middleware/auth_middleware.go — valida JWT, inyecta user_id en contexto Gin
- [ ] internal/middleware/rate_limit_middleware.go — máx. 5 intentos de login por IP/minuto
- [ ] bcrypt para contraseñas (costo mínimo 12)
- [ ] Access Token JWT: expira en 15 minutos
- [ ] Refresh Token JWT: expira en 7 días, guardado en tabla refresh_tokens en DB
- [ ] internal/router/router.go — rutas públicas vs. protegidas claramente separadas
- [ ] main.go actualizado para levantar el servidor HTTP
- [ ] Endpoints:
  - POST /api/v1/auth/register
  - POST /api/v1/auth/login (con rate limiting)
  - POST /api/v1/auth/refresh
  - POST /api/v1/auth/logout — invalida el refresh token en DB
  - GET /api/version (público, sin auth)
- [ ] Tests unitarios de auth_service.go

> ⚠️ REGLA DE ORO para todos los services de aquí en adelante:
> Cada query DEBE incluir WHERE user_id = ? usando el user_id del JWT.
> Nunca confiar en un ID del body o URL sin cruzarlo con el JWT.
> Ejemplo correcto: db.Where("id = ? AND user_id = ?", id, userID).First(&order)

**Checkpoint ✅**
- POST /register → crea usuario con password hasheada en DB
- POST /login → devuelve access token (15min) + refresh token (7d)
- POST /refresh → devuelve nuevo access token
- POST /logout → refresh token queda inválido en DB
- Ruta protegida sin token → 401; con token → 200
- 6 intentos fallidos → 429 Too Many Requests
- go test ./internal/service/... pasa

**Errores a corregir antes de Fase 3:**
- JWT secret hardcodeado (debe venir de .env)
- bcrypt cost < 12 (vulnerable a fuerza bruta)
- CORS no configurado (falla desde Flutter)
- Refresh token no guardado en DB (logout seguro imposible)

---

### FASE 3 — Módulo de Clientes del Taller

**Objetivo:** CRUD de clientes antes de los encargos, ya que Order depende de Client.

**Entregables:**
- [ ] internal/repository/client_repository.go
- [ ] internal/service/client_service.go
- [ ] internal/handler/client_handler.go
- [ ] Endpoints (todos protegidos con JWT, todos filtran por user_id):
  - GET /api/v1/clients — listado paginado con búsqueda por nombre
  - POST /api/v1/clients — crear cliente
  - GET /api/v1/clients/:id — detalle
  - PUT /api/v1/clients/:id — editar
  - DELETE /api/v1/clients/:id — eliminar (solo si no tiene encargos activos)
  - GET /api/v1/clients/:id/orders — historial de encargos de un cliente
- [ ] Tests unitarios de client_service.go

**Checkpoint ✅**
- CRUD completo funciona
- No se puede eliminar cliente con encargos activos (retorna 409 Conflict)
- Historial de encargos por cliente correcto
- Un usuario NO puede ver clientes de otro usuario

**Errores a corregir antes de Fase 4:**
- No validar unicidad de email de cliente por usuario
- Eliminar cliente con encargos sin avisar (pérdida de datos)

---

### FASE 4 — Módulo de Inventario de Materiales

**Objetivo:** CRUD de materiales y sistema de compras con costo unitario por promedio ponderado móvil.

**Entregables:**
- [ ] internal/repository/material_repository.go
- [ ] internal/service/material_service.go
- [ ] internal/handler/material_handler.go
- [ ] Endpoints:

> ⚠️ ORDEN DE RUTAS EN GIN — CRÍTICO:
> Las rutas estáticas DEBEN declararse ANTES que las rutas dinámicas.
> Si /:id se declara antes que /alerts/low-stock, Gin interpreta "low-stock" como un ID
> y el endpoint NUNCA será alcanzado. Orden correcto obligatorio:

  1. GET  /api/v1/materials/alerts/low-stock  (PRIMERO — ruta estática)
  2. GET  /api/v1/materials
  3. POST /api/v1/materials
  4. GET  /api/v1/materials/:id               (DESPUÉS — ruta dinámica)
  5. PUT  /api/v1/materials/:id
  6. DELETE /api/v1/materials/:id
  7. POST /api/v1/materials/:id/purchases
  8. GET  /api/v1/materials/:id/purchases

- [ ] Lógica de costo unitario — PROMEDIO PONDERADO MÓVIL:
  nuevo = (stock_actual x costo_anterior + cantidad_comprada x precio_unitario) / (stock_actual + cantidad_comprada)
  Más preciso que simple división porque considera el stock previo.

- [ ] Validaciones: stock no negativo, stock_minimo > 0, cantidad_comprada > 0
- [ ] Tests unitarios de material_service.go (especialmente el promedio ponderado)

**Checkpoint ✅**
- CRUD de materiales completo
- Al registrar compra: stock aumenta, costo unitario se recalcula por promedio ponderado
- GET /alerts/low-stock devuelve solo materiales con stock_actual <= stock_minimo
- No se puede registrar compra con cantidad 0 o negativa

**Errores a corregir antes de Fase 5:**
- Promedio ponderado mal calculado (usar enteros con precisión, no floats)
- Ruta /alerts/low-stock no alcanzable por conflicto con /:id
- costo_unitario guardado como float (debe ser entero en CLP)

---

### FASE 5 — Módulo de Gestión de Trabajos / Encargos

**Objetivo:** Módulo central — encargos con materiales, rentabilidad, control de estado y generación automática de transacciones al completar.

**Entregables:**
- [ ] internal/repository/order_repository.go
- [ ] internal/service/order_service.go — lógica:
  - Al asignar materiales: descuento de stock EN TRANSACCIÓN DB ATÓMICA
  - Al editar materiales: restaura stock anterior + descuenta nuevo, EN TRANSACCIÓN DB ATÓMICA
  - Al eliminar material de encargo: restaura stock, EN TRANSACCIÓN DB ATÓMICA
  - Al eliminar encargo: restaura todo el stock de materiales asignados
  - Guardar costo_unitario_snapshot al momento de asignar el material
  - Cálculo de rentabilidad:
    - costo_materiales = suma(cantidad × costo_unitario_snapshot)
    - costo_mano_obra = horas × tarifa_hora
    - ganancia_neta = precio_venta - costo_materiales - costo_mano_obra
    - margen_porcentaje = calculado SOLO si precio_venta > 0, sino devuelve null
  - Al cambiar estado a "entregado": crear AUTOMÁTICAMENTE una Transaction de tipo ingreso
    con source="order" y order_id. Verificar idempotencia (no duplicar si ya está en entregado)
- [ ] internal/handler/order_handler.go
- [ ] Endpoints:
  - GET    /api/v1/orders
  - POST   /api/v1/orders
  - GET    /api/v1/orders/:id
  - PUT    /api/v1/orders/:id
  - DELETE /api/v1/orders/:id
  - PATCH  /api/v1/orders/:id/status
  - POST   /api/v1/orders/:id/materials         (asignar material)
  - PUT    /api/v1/orders/:id/materials/:mid     (editar cantidad)
  - DELETE /api/v1/orders/:id/materials/:mid     (quitar material)
- [ ] Estados: pendiente → en_progreso → completado → entregado (aquí se genera la transacción)
- [ ] Tests unitarios de order_service.go

**Checkpoint ✅**
- Crear encargo con materiales → stock se descuenta
- Editar cantidad de material → stock se ajusta correctamente (atómico)
- Eliminar material de encargo → stock se restaura
- Eliminar encargo → todo el inventario se restaura
- precio_venta = 0 → margen_porcentaje devuelve null, NO error 500
- Cambiar estado a entregado → aparece automáticamente transacción de ingreso en flujo de caja
- Llamar entregado dos veces → NO se duplica la transacción (idempotencia)
- Stock insuficiente al asignar → retorna 400 con mensaje descriptivo

**Errores a corregir antes de Fase 6:**
- No usar db.Transaction() en operaciones que modifican inventario (inconsistencia)
- No guardar costo_unitario_snapshot (rentabilidad histórica incorrecta)
- División por cero en margen_porcentaje cuando precio_venta = 0
- Transacción automática duplicada si se llama entregado dos veces


---

### FASE 6 — Módulo de Flujo de Caja y Reportes

**Objetivo:** Flujo financiero completo y estadísticas sin duplicados ni inconsistencias con los encargos.

**Entregables — 6A: Flujo de Caja:**
- [ ] internal/repository/transaction_repository.go
- [ ] internal/service/transaction_service.go
- [ ] internal/handler/transaction_handler.go
- [ ] Modelo Transaction con campo source:
  - source = "manual" → ingresado directamente por la costurera
  - source = "order" → generado automáticamente al completar un encargo
  - order_id → FK opcional, presente cuando source = "order"
- [ ] Endpoints:
  - POST /api/v1/transactions — SOLO crea transacciones source="manual"
  - GET /api/v1/transactions — listado paginado (filtros: tipo, fecha, categoría, source)
  - PUT /api/v1/transactions/:id — editar (SOLO las source="manual")
  - DELETE /api/v1/transactions/:id — eliminar (SOLO las source="manual")
  - GET /api/v1/transactions/balance — balance del mes actual
  - GET /api/v1/transactions/balance?month=YYYY-MM — balance por mes específico

> ⚠️ Por qué el campo source:
> Sin este campo, si la costurera registra manualmente el cobro de un encargo,
> el balance mensual lo contaría dos veces (automático + manual).
> Con source, el sistema puede filtrar y la UI puede mostrar de dónde vino cada ingreso.

**Entregables — 6B: Reportes:**
- [ ] internal/service/report_service.go
- [ ] internal/handler/report_handler.go
- [ ] Endpoints:
  - GET /api/v1/reports/summary
  - GET /api/v1/reports/earnings?period=monthly&year=2025
  - GET /api/v1/reports/top-materials
  - GET /api/v1/reports/top-orders
- [ ] Tests unitarios de transaction_service.go

**Checkpoint ✅**
- Balance mensual correcto: ingresos - gastos (sin duplicados)
- Las transacciones source="order" NO se pueden crear/editar/eliminar manualmente desde la API
- Los reportes son consistentes con los datos reales
- No hay queries N+1 (verificar con db.Debug() de GORM)
- Filtros por fecha funcionan correctamente con timezone

**Errores a corregir antes de Fase 7:**
- Balance duplicado si source no se implementa correctamente
- Queries N+1 en reportes de materiales más usados
- Timezone no manejado en filtros por mes

---

### FASE 7 — Testing, Logging y Calidad de Código

**Objetivo:** Completar cobertura de tests, añadir observabilidad y asegurar calidad antes del despliegue.

> Nota: Los tests unitarios de cada service ya se escribieron en sus fases (2, 3, 4, 5, 6).
> Aquí se completan los tests de integración y se añade logging y linting.

**Entregables:**
- [ ] Tests de integración para handlers principales (usando httptest)
- [ ] go test ./... -coverprofile=coverage.out para revisar cobertura total
- [ ] zerolog para logging estructurado (JSON en producción, legible en dev)
- [ ] Middleware de logging de requests (método, ruta, status, latencia, user_id)
- [ ] Error handler global centralizado en Gin
- [ ] Respuestas JSON estandarizadas:
  - Éxito: { "success": true, "data": {...}, "message": "..." }
  - Error: { "success": false, "error": "descripcion", "code": 400 }
- [ ] golangci-lint configurado con .golangci.yml
- [ ] Verificar que ningún log expone passwords, tokens o datos sensibles

**Checkpoint ✅**
- go test ./... pasa sin errores
- Cobertura > 70% en capa de services
- golangci-lint run sin errores críticos
- Los logs incluyen: método, ruta, status, latencia y user_id
- Ningún log imprime password ni JWT en texto plano

---

### FASE 8 — Despliegue y Documentación

**Objetivo:** Backend listo para producción con Docker separado por entorno, documentación y estrategia de versionado.

**Entregables:**
- [ ] Dockerfile con multi-stage build (build Go + runtime Alpine mínimo)
- [ ] docker-compose.yml — DESARROLLO: PostgreSQL + Air, puertos expuestos para debug
- [ ] docker-compose.prod.yml — PRODUCCIÓN: solo binario + DB + volúmenes persistentes, SIN puertos innecesarios
- [ ] Documentación de la API con swaggo/swag (Swagger/OpenAPI)
- [ ] README.md completo (instalación dev y prod, cómo correr migraciones, variables de entorno)
- [ ] GET /health — health check (público, sin auth)
- [ ] GET /api/version — versión de la API y versión mínima de app compatible:
  { "api_version": "1.0.0", "min_app_version": "1.0.0" }
- [ ] Estrategia de versionado: mantener /api/v1/ funcional al introducir /api/v2/
- [ ] Nginx como proxy inverso con HTTPS

**Checkpoint ✅**
- docker-compose up (dev) levanta todo sin errores
- docker-compose -f docker-compose.prod.yml up (prod) levanta todo
- Swagger accesible en /swagger/index.html
- GET /health responde 200 desde fuera del contenedor
- GET /api/version responde correctamente
- Puerto de PostgreSQL NO expuesto en docker-compose.prod.yml

---

### FASE 9 — Frontend Móvil: Diseño y Desarrollo con Flutter

**Objetivo:** Interfaz visual completa de la app en Flutter, conectada al backend, con UX intuitiva y diseño premium.

**Stack de Frontend:**
- Riverpod 2.x (gestor de estado)
- dio + interceptor JWT automático (cliente HTTP)
- go_router (navegación)
- Material 3 con tema personalizado (diseño)
- flutter_secure_storage + shared_preferences (almacenamiento)
- fl_chart (gráficos)
- Material Icons nativos de Flutter (íconos — ya incluidos, sin dependencia extra)

---

**9A — Configuración Base y Entorno**

- [ ] flutter create costura_app
- [ ] Estructura: lib/core/, lib/core/config/env_config.dart, lib/data/, lib/domain/, lib/presentation/, lib/shared/widgets/
- [ ] env_config.dart para URL base por entorno:
  ```dart
  class EnvConfig {
    static const String baseUrl = String.fromEnvironment(
      'BASE_URL',
      defaultValue: 'http://10.0.2.2:8080', // Android emulator apunta al host local
      // iOS simulator usa: http://localhost:8080
      // Dispositivo físico: http://192.168.1.X:8080
      // Producción: https://tudominio.com
    );
  }
  // Compilar con: flutter run --dart-define=BASE_URL=https://tudominio.com
  ```
- [ ] dio con interceptor: inyecta JWT, renueva automáticamente con refresh token al recibir 401, redirige a login si refresh falla
- [ ] go_router con rutas protegidas (guard que verifica JWT en flutter_secure_storage)
- [ ] Sistema de diseño: colores, tipografía, espaciados, tema oscuro y claro

---

**9B — Pantallas de Autenticación**

- [ ] Splash Screen: animación del logo + verificación de sesión activa
- [ ] Login: email, contraseña, errores inline, mensaje "Demasiados intentos" al recibir 429
- [ ] Registro: nombre, email, contraseña, confirmación
- [ ] Guardar access token + refresh token en flutter_secure_storage

---

**9C — Pantalla Principal y Navegación**

- [ ] Bottom Navigation Bar con 4 secciones: Inicio, Inventario, Encargos, Finanzas
- [ ] Dashboard: tarjetas de resumen, accesos rápidos, gráfico de barras con fl_chart

---

**9D — Módulo de Clientes (UI)**

- [ ] Listado con búsqueda, formulario crear/editar, detalle con historial de encargos

---

**9E — Módulo de Inventario (UI)**

- [ ] Listado con badge rojo si bajo stock, formulario crear/editar, detalle con historial de compras
- [ ] Formulario de compra: costo unitario calculado en tiempo real
- [ ] Sección de alertas de bajo stock

---

**9F — Módulo de Encargos (UI)**

- [ ] Listado con chips de estado con color, filtros por estado/cliente/fecha
- [ ] Formulario crear: selector de cliente (no texto libre)
- [ ] Pantalla detalle: materiales con editar/eliminar, panel de rentabilidad
- [ ] margen_porcentaje = null → mostrar "Sin precio de venta asignado" (NO crash)
- [ ] Modal de asignación de materiales con preview del stock restante

---

**9G — Módulo de Finanzas (UI)**

- [ ] Flujo de caja: balance del mes, lista agrupada por día
- [ ] Distinción visual: source="manual" (azul) vs source="order" (verde)
- [ ] FAB para agregar ingreso/gasto manual
- [ ] Reportes: selector de período, gráficos fl_chart, listas de más rentables/utilizados

---

**9H — Pulido Final y UX**

- [ ] Animaciones (Hero, fade), pull-to-refresh, empty states, estados de error
- [ ] Modo oscuro completo
- [ ] Íconos de app y splash screen nativo
- [ ] Pantalla de privacidad en App Switcher
- [ ] Pruebas en emulador Android Y dispositivo físico

**Checkpoint ✅**
- flutter run sin errores en emulador Android
- Login funciona y guarda JWT; refresh token renueva automáticamente
- CRUD completo funciona desde la app
- margen_porcentaje = null → texto amigable, no crash
- Transacciones de encargos diferenciadas visualmente de las manuales
- Pantalla se oscurece en App Switcher
- URL base cambiable con --dart-define sin tocar el código

**Errores a corregir:**
- Emulador Android no conecta → verificar que BASE_URL usa 10.0.2.2
- JWT expirado no redirige al login → verificar interceptor dio
- Datos no se refrescan → invalidar provider de Riverpod


---

### FASE 10 — Endurecimiento de Seguridad (Security Hardening)

**Objetivo:** Auditar y reforzar todos los puntos vulnerables antes de usar la app en producción.

---

**10A — Backend (Go)**

- [ ] Rate Limiting en login: verificar que funciona en producción detrás de Nginx
- [ ] Refresh Token en DB: verificar que tokens invalidados retornan 401 correctamente
- [ ] SQL Injection: auditar todos los db.Raw(); GORM parametriza automáticamente los métodos estándar
- [ ] IDOR: auditar que TODOS los endpoints incluyen WHERE user_id = ? en sus queries
- [ ] Sanitización de inputs en campos de texto libre
- [ ] Headers de seguridad HTTP (agregar middleware en Gin):
  - X-Content-Type-Options: nosniff
  - X-Frame-Options: DENY
  - Strict-Transport-Security: max-age=31536000 (solo en HTTPS)
- [ ] Auditar: git log --all -- .env (verificar que .env nunca fue commiteado)
- [ ] Body size máximo configurado en Gin

---

**10B — Base de Datos (PostgreSQL)**

- [ ] Usuario de DB con mínimos privilegios: crear usuario costura_app solo con SELECT/INSERT/UPDATE/DELETE (no superusuario)
- [ ] Backups automáticos: cron diario de pg_dump guardado en almacenamiento externo
- [ ] Tabla de auditoría: registrar acciones críticas (eliminaciones, cambios de estado)
- [ ] Connection pool limitado: db.SetMaxOpenConns(25) para evitar DoS
- [ ] Puerto de DB no expuesto en docker-compose.prod.yml

---

**10C — Frontend Flutter**

- [ ] Almacenamiento: todo en flutter_secure_storage, nada sensible en SharedPreferences
- [ ] PUBLIC KEY PINNING (NO Certificate Pinning completo):
  ⚠️ IMPORTANTE: Usar Public Key Pinning, NO Certificate Pinning completo.
  Let's Encrypt renueva certificados cada 90 días. Si se fija el certificado completo,
  la app deja de funcionar con cada renovación hasta publicar una actualización.
  Con Public Key Pinning se fija la clave pública del servidor (más estable),
  que solo cambia si se genera un nuevo par de claves.
  Librería recomendada: http_certificate_pinning en pub.dev
- [ ] Logout limpio: elimina access token, refresh token y caché de Riverpod
- [ ] Ofuscación en release: flutter build apk --obfuscate --split-debug-info=./debug-info
- [ ] Timeout de sesión: si la app está en segundo plano > 10 minutos, pedir biometría o PIN
- [ ] No loguear datos sensibles: auditar todos los print() y debugPrint()

---

**10D — Comunicación Segura (Red)**

- [ ] HTTPS obligatorio: Nginx + certificado Let's Encrypt (certbot --nginx)
- [ ] CORS restrictivo: solo aceptar el origen del cliente conocido, NO usar *
- [ ] Firewall: solo puerto 443 expuesto a internet; DB (5432) solo accesible internamente

**Checkpoint ✅**
- 6+ intentos de login fallidos → 429 Too Many Requests
- Acceder a recurso de otro usuario con JWT propio → 403 Forbidden
- Logout → refresh token inválido → 401 al intentar renovar
- APK de release descompilado no revela rutas ni secretos legibles
- Conexión solo funciona por HTTPS
- App Switcher oculta la pantalla
- Auditar con OWASP ZAP o Burp Suite en modo pasivo contra la API

**Errores a corregir:**
- CORS con * habilitado → revisar middleware de Gin
- Algún endpoint sin WHERE user_id = ? → revisión manual de todos los services
- JWT secret débil (menos de 32 caracteres aleatorios)
- print() con datos sensibles en Flutter (buscar con: grep -rn "print" lib/)

---

## 📊 Resumen del Roadmap

```
Fase 1  → Arquitectura, Modelos y Migraciones (golang-migrate + 7 tablas)
    ↓ (verificar: tablas en DB, .env en .gitignore, proyecto compila)
Fase 2  → Autenticación JWT + Refresh Token + Rate Limiting
    ↓ (verificar: login/refresh/logout + rutas protegidas + user_id en queries)
Fase 3  → Módulo de Clientes del Taller
    ↓ (verificar: CRUD + historial de encargos por cliente)
Fase 4  → Inventario de Materiales (promedio ponderado + orden de rutas Gin)
    ↓ (verificar: compras, costo unitario correcto, /alerts/low-stock accesible)
Fase 5  → Gestión de Encargos (transacciones atómicas + snapshot + auto-transacción)
    ↓ (verificar: rentabilidad, inventario consistente, transacción al entregar, null safety)
Fase 6  → Flujo de Caja + Reportes (campo source, sin duplicados)
    ↓ (verificar: balance correcto, fuente de transacciones diferenciada)
Fase 7  → Testing completo + Logging + Linting
    ↓ (verificar: tests pasan, cobertura > 70%, linter OK)
Fase 8  → Despliegue: Docker dev/prod separados + Swagger + Versionado de API
    ↓ (verificar: Docker prod funciona, Swagger OK, /health y /api/version OK)
Fase 9  → Frontend Flutter (URL por entorno, interceptor JWT, UI completa)
    ↓ (verificar: app conecta al backend, flujos completos, null safety en margen)
Fase 10 → Endurecimiento de Seguridad (Public Key Pinning, IDOR audit, HTTPS)
    ↓ (verificar: OWASP ZAP, rate limit, logout seguro, APK ofuscado)
```

---

## 🔑 Decisiones de Diseño Importantes

| Decisión | Justificación |
|---|---|
| golang-migrate en lugar de AutoMigrate | Migraciones explícitas, reversibles y seguras en producción |
| Modelo Client separado | Historial por cliente, sin datos duplicados, búsqueda y filtrado real |
| costo_unitario_snapshot en OrderMaterial | Rentabilidad histórica no cambia si el precio del material cambia después |
| Promedio ponderado móvil para costo unitario | Más preciso que simple división cuando hay múltiples compras históricas |
| margen_porcentaje = null cuando precio = 0 | Evita división por cero; frontend muestra texto amigable en lugar de error |
| Transacción automática al marcar entregado | Consistencia entre encargos y flujo de caja sin doble registro manual |
| Campo source en transacciones | Distingue ingresos manuales de ingresos por encargos; evita balance duplicado |
| Go + Gin | Alto rendimiento, compilación estática, ideal para APIs REST |
| Access Token 15min + Refresh Token 7d | Seguridad sin fricción: tokens cortos, renovación automática transparente |
| docker-compose dev/prod separados | Entornos distintos, sin exponer puertos innecesarios en producción |
| Flutter + Riverpod | Un código para Android e iOS; estado reactivo y testeable |
| --dart-define para URL base | Sin hardcodear la URL; cambia por entorno sin tocar el código |
| Public Key Pinning (NO Certificate Pinning) | Renovaciones de Let's Encrypt cada 90d no rompen la app |
| WHERE user_id = ? en todos los queries | Protección IDOR desde el inicio; escala a multi-usuario sin refactoring |
| Rutas estáticas antes que dinámicas en Gin | Evita que /alerts/low-stock sea interpretado como /:id = "low-stock" |
| Tests por fase (no al final) | Los bugs de diseño se detectan en su propia fase, no al final del proyecto |

---

## ⚠️ Riesgos Identificados

- **Atomicidad en Fase 5:** Usar db.Transaction() de GORM en toda operación que toque inventario + encargo. Sin esto, una falla deja el inventario inconsistente.
- **Secretos en el repositorio:** .env en .gitignore desde el primer commit. Un secreto commiteado es muy difícil de eliminar del historial de Git.
- **Escalabilidad futura:** El user_id ya está en todas las tablas. Si se agrega multi-usuario (varios talleres), no es necesario refactorizar el esquema.
- **Timezone en reportes:** Definir desde el inicio si la DB guarda en UTC y el backend convierte, o si guarda en zona local. Mezclar los dos esquemas genera reportes incorrectos.
- **Renovación de certificado TLS:** Con Public Key Pinning se mitiga. Documentar el proceso de rotación de claves.
- **Idempotencia del estado entregado:** Validar en el service que el encargo no esté ya en estado entregado antes de crear la transacción automática. Llamar dos veces al endpoint duplicaría el ingreso.

---

## 📝 Correcciones Aplicadas (v1.0 → v2.0)

| # | Problema Original | Corrección Aplicada | Fase |
|---|---|---|---|
| 1 | GORM AutoMigrate en producción | Reemplazado por golang-migrate desde Fase 1 | Fase 1 |
| 2 | Ruta /low-stock bloqueada por /:id en Gin | Renombrada a /alerts/low-stock, orden documentado | Fase 4 |
| 3 | Sin endpoint para editar/quitar materiales de encargo | Agregados PUT y DELETE /orders/:id/materials/:mid | Fase 5 |
| 4 | División por cero en margen_porcentaje | Calculado solo si precio_venta > 0, sino devuelve null | Fase 5 |
| 5 | Sin modelo Client (cliente en texto libre) | Agregado modelo Client y Fase 3 dedicada | Fase 1+3 |
| 6 | Sin validación user_id en queries | Regla de oro en Fase 2, aplicada en todos los services | Fase 2 |
| 7 | Encargo entregado no genera transacción automática | Lógica en order_service + idempotencia | Fase 5 |
| 8 | Balance duplicado por mezcla de transacciones | Campo source y order_id FK en Transaction | Fase 6 |
| 9 | Tests solo al final (Fase 7) | Tests unitarios de services en cada fase | Todas |
| 10 | Un solo docker-compose.yml para dev y prod | Separados en docker-compose.yml y docker-compose.prod.yml | Fase 8 |
| 11 | lucide_icons no existe en pub.dev | Reemplazado por Material Icons nativos de Flutter | Fase 9 |
| 12 | URL base del backend hardcodeada | env_config.dart con --dart-define por entorno | Fase 9 |
| 13 | Certificate Pinning rompe app en renovación TLS | Cambiado a Public Key Pinning con justificación | Fase 10 |
| 14 | Sin estrategia de versionado de API | Agregado GET /api/version y política de deprecación | Fase 8 |

---

*Documento v2.0 — Planificación auditada y lista para desarrollo.*
