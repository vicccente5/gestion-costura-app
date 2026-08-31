<div align="center">

# 🧵 Gestión Costura App

### La herramienta definitiva para administrar tu taller de costura

*Control total de tu inventario, encargos, finanzas y rentabilidad — en la palma de tu mano.*

---

![Estado del proyecto](https://img.shields.io/badge/Estado-En%20Desarrollo-yellow?style=for-the-badge)
![Backend](https://img.shields.io/badge/Backend-Go%20%2B%20Gin-00ADD8?style=for-the-badge&logo=go)
![Frontend](https://img.shields.io/badge/Frontend-Flutter-02569B?style=for-the-badge&logo=flutter)
![Base de datos](https://img.shields.io/badge/Base%20de%20Datos-PostgreSQL-336791?style=for-the-badge&logo=postgresql)
![Plataformas](https://img.shields.io/badge/Plataformas-Android%20%7C%20iOS-black?style=for-the-badge)

</div>

---

## 📖 ¿Qué es Gestión Costura App?

**Gestión Costura App** es una aplicación móvil diseñada especialmente para costureras y talleres de costura artesanal. Permite administrar de manera integral todos los aspectos del negocio: desde el control del inventario de materiales hasta el cálculo exacto de la rentabilidad de cada encargo.

Olvídate de las planillas de Excel, los cuadernos con anotaciones y las calculadoras manuales. Con esta app, tienes toda la información de tu taller organizada, actualizada y disponible en tiempo real desde tu teléfono.

---

## ✨ Funcionalidades Principales

### 👥 Gestión de Clientes
- Registro completo de clientes: nombre, teléfono y correo electrónico
- Historial completo de encargos por cliente
- Búsqueda y filtrado rápido de clientes

### 📦 Control de Inventario
- Registro de materiales: telas, hilos, botones, encajes y cualquier insumo
- Control de stock en tiempo real con alertas de bajo stock
- Registro de compras con cálculo automático del **costo unitario por promedio ponderado móvil** (el más preciso para negocios con múltiples compras del mismo material)
- Historial de compras por cada material

### ✂️ Gestión de Encargos y Trabajos
- Creación de encargos vinculados directamente al cliente
- Asignación de materiales con descuento automático del inventario
- Registro de horas trabajadas y tarifa de mano de obra
- **Cálculo automático de rentabilidad:**
  - Costo total de materiales
  - Costo de mano de obra
  - Ganancia neta
  - Margen de ganancia en porcentaje
- Control de estados: Pendiente → En Progreso → Completado → Entregado
- Al marcar un encargo como **Entregado**, el ingreso se registra automáticamente en el flujo de caja

### 💰 Flujo de Caja y Finanzas
- Registro de ingresos y gastos generales del taller
- Balance mensual automático: ingresos − gastos = ganancia real
- Diferenciación entre ingresos manuales e ingresos generados por encargos (sin duplicados)
- Filtros por período, categoría y tipo de transacción

### 📊 Reportes y Estadísticas
- Ganancias por período: semanal, mensual y anual
- Encargos más rentables
- Materiales más utilizados
- Resumen general del negocio
- Gráficos visuales para entender el rendimiento de un vistazo

---

## 🛠️ Stack Tecnológico

### Backend
| Tecnología | Uso |
|---|---|
| **Go (Golang) 1.21+** | Lenguaje principal del servidor — alto rendimiento y tipado estático |
| **Gin** | Framework HTTP para la API REST |
| **PostgreSQL** | Base de datos relacional principal |
| **GORM** | ORM para operaciones de base de datos |
| **golang-migrate** | Migraciones de base de datos versionadas y reversibles |
| **JWT (golang-jwt)** | Autenticación y autorización mediante tokens |
| **bcrypt** | Hash seguro de contraseñas |
| **zerolog** | Logging estructurado de alto rendimiento |
| **go-playground/validator** | Validación de datos de entrada |
| **Docker + Docker Compose** | Contenedores para desarrollo y producción |
| **Nginx** | Proxy inverso y terminación TLS en producción |

### Frontend
| Tecnología | Uso |
|---|---|
| **Flutter (Dart) 3.16+** | Framework UI multiplataforma — Android e iOS desde un solo código |
| **Riverpod 2.x** | Gestión de estado reactiva y testeable |
| **dio** | Cliente HTTP con interceptor automático de JWT |
| **go_router** | Navegación con rutas protegidas |
| **flutter_secure_storage** | Almacenamiento encriptado del token de sesión |
| **fl_chart** | Gráficos interactivos para reportes |
| **Material 3** | Sistema de diseño con modo oscuro y claro |

---

## 🔐 Seguridad y Manejo de Datos

La aplicación fue diseñada con seguridad en todas sus capas desde el primer día:

### Autenticación y Acceso
- 🔑 **JWT con tokens de corta duración:** el access token expira en 15 minutos; el refresh token en 7 días, minimizando el impacto de un token robado
- 🔄 **Renovación automática:** la app renueva el token de forma transparente sin interrumpir al usuario
- 🚪 **Logout seguro:** al cerrar sesión, el refresh token se invalida en la base de datos, sin posibilidad de reutilización
- 🛡️ **Rate Limiting en el login:** máximo 5 intentos fallidos por minuto por IP, bloqueando ataques de fuerza bruta

### Protección de Datos
- 🔒 **Contraseñas hasheadas con bcrypt** (costo 12) — nunca se almacenan en texto plano, ni siquiera el administrador puede leerlas
- 📱 **Almacenamiento seguro en el dispositivo:** el token de sesión se guarda en `flutter_secure_storage`, un almacén encriptado del sistema operativo (no en texto plano)
- 👁️ **Pantalla de privacidad:** cuando la app pasa a segundo plano, la pantalla se oscurece automáticamente para que los datos no sean visibles en el selector de apps
- ⏱️ **Timeout de sesión:** si la app está inactiva en segundo plano por más de 10 minutos, solicita autenticación biométrica (huella o Face ID) para reanudar

### Seguridad en la Comunicación
- 🌐 **HTTPS obligatorio:** toda la comunicación entre la app y el servidor está encriptada con TLS (certificado Let's Encrypt)
- 📌 **Public Key Pinning:** la app solo acepta el certificado del servidor propio, protegiendo contra ataques de interceptación en redes WiFi públicas
- 🔧 **Headers de seguridad HTTP:** el servidor incluye `X-Content-Type-Options`, `X-Frame-Options` y `Strict-Transport-Security`
- 🚫 **CORS restrictivo:** la API solo acepta peticiones del cliente autorizado

### Integridad de los Datos
- ⚛️ **Transacciones atómicas en la base de datos:** operaciones críticas (como asignar materiales a un encargo) se ejecutan como una sola transacción; si algo falla, todo se revierte sin dejar datos a medias
- 🏷️ **Snapshot de costos:** al asignar un material a un encargo, se guarda el precio que tenía en ese momento. Si el costo del material cambia después, la rentabilidad histórica del encargo permanece intacta
- 🔍 **Aislamiento de datos por usuario:** cada registro en la base de datos está vinculado al usuario que lo creó; es imposible acceder a datos de otro usuario aunque se tenga una sesión válida
- 💾 **Backups automáticos diarios** de la base de datos

---

## 📱 Interfaz de la Aplicación

> *Las imágenes de la interfaz serán agregadas a medida que avance el desarrollo.*

### Capturas de Pantalla

#### Pantalla de Inicio y Autenticación
| | | |
|:---:|:---:|:---:|
| *Splash Screen* | *Login* | *Registro* |
| `[imagen próximamente]` | `[imagen próximamente]` | `[imagen próximamente]` |

---

#### Dashboard Principal
| | |
|:---:|:---:|
| *Vista general del taller* | *Modo oscuro* |
| `[imagen próximamente]` | `[imagen próximamente]` |

---

#### Módulo de Inventario
| | | |
|:---:|:---:|:---:|
| *Lista de materiales* | *Detalle del material* | *Alertas de bajo stock* |
| `[imagen próximamente]` | `[imagen próximamente]` | `[imagen próximamente]` |

---

#### Módulo de Encargos
| | | |
|:---:|:---:|:---:|
| *Lista de encargos* | *Detalle del encargo* | *Panel de rentabilidad* |
| `[imagen próximamente]` | `[imagen próximamente]` | `[imagen próximamente]` |

---

#### Módulo de Finanzas y Reportes
| | | |
|:---:|:---:|:---:|
| *Flujo de caja* | *Balance mensual* | *Gráficos de ganancias* |
| `[imagen próximamente]` | `[imagen próximamente]` | `[imagen próximamente]` |

---

## 🚀 Instalación y Desarrollo

### Requisitos previos
- Go 1.21+
- Docker y Docker Compose
- Flutter 3.16+ (con Dart 3.2+) para el cliente móvil

### Variables de Entorno (.env)
Crea un archivo `.env` en la raíz del proyecto basado en `.env.example`:
```env
SERVER_PORT=8080
GIN_MODE=debug
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=costuradb
DB_SSLMODE=disable
JWT_SECRET=supersecretkey
JWT_ACCESS_EXPIRY_MINUTES=15
JWT_REFRESH_EXPIRY_DAYS=7
```

### Levantar el entorno de Desarrollo (con Hot Reload)
Utilizamos `docker-compose.yml` para levantar la base de datos, y recomendamos usar `air` para compilar automáticamente los cambios de Go.

```bash
# 1. Levantar solo la base de datos en Docker
docker-compose up -d

# 2. Descargar dependencias
go mod download

# 3. Correr la aplicación localmente (las migraciones se ejecutan solas al iniciar)
go run cmd/main.go

# Alternativa: Usar Air para Hot Reload
go install github.com/air-verse/air@latest
air
```

### Documentación de la API (Swagger)
La API está documentada usando Swagger. Una vez que el servidor esté corriendo, visita:
👉 **[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)**

Para regenerar la documentación si cambias los handlers:
```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/main.go --parseDependency --parseInternal
```

### Levantar el entorno de Producción
El entorno de producción utiliza un Dockerfile multi-stage sumamente liviano y asegura la aplicación detrás de Nginx como proxy inverso. No se expone el puerto de la base de datos.

```bash
# Compilar y levantar los contenedores de producción en segundo plano
docker-compose -f docker-compose.prod.yml up -d --build
```
La API estará disponible en el puerto `80` a través de Nginx.

---

## 🛤 Estrategia de Versionado de API

La API sigue una estrategia de versionado mediante la ruta (URL path versioning).

- **`GET /api/version`**: Endpoint de utilidad para el frontend. Retorna la versión actual de la API y la versión mínima requerida de la aplicación móvil compatible.
- **Rutas `/api/v1/*`**: Toda la funcionalidad actual se encuentra bajo `v1`.
- **Evolución a `v2`**: Cuando se requieran **cambios que rompan compatibilidad** (breaking changes) en los payloads, se creará un grupo `/api/v2/*` mientras se mantiene funcionando `v1`. De este modo, los clientes con versiones antiguas de la app no dejarán de funcionar repentinamente hasta que decidas descontinuar `v1` por completo.

---

## 📋 Roadmap del Proyecto

- [x] Planificación y diseño de arquitectura
- [x] **Fase 0 a 6:** Core (Auth, Clientes, Inventario, Encargos, Flujo de Caja)
- [x] **Fase 7:** Testing y calidad de código
- [x] **Fase 8:** Despliegue y documentación API (Swagger, Docker Multi-stage, Nginx)
- [ ] **Fase 9:** Desarrollo del frontend Flutter
- [ ] **Fase 10:** Endurecimiento de seguridad

---

## 👩‍💻 Desarrollado para

Costureras y talleres de costura artesanal que quieren llevar su negocio al siguiente nivel con tecnología real, sin necesidad de ser expertas en informática.

---

<div align="center">

*Hecho con ❤️ para el mundo de la costura artesanal*

</div>
