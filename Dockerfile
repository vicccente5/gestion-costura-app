# 1. Fase de construcción (Builder)
FROM golang:alpine AS builder

# Instalar dependencias necesarias para compilar y probar
RUN apk add --no-cache git tzdata

WORKDIR /app

# Descargar módulos (caché)
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Construir el binario minimizado (-s -w elimina tabla de símbolos e info debug)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api-server ./cmd/main.go

# 2. Fase de ejecución (Runtime) - Mínima y segura
FROM alpine:latest

# Añadir zona horaria y certificados SSL
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiar el binario compilado desde la fase anterior
COPY --from=builder /app/api-server .

# Definir variables de entorno por defecto
ENV SERVER_PORT=8080
ENV GIN_MODE=release

# Exponer el puerto
EXPOSE 8080

# Ejecutar la API
CMD ["./api-server"]
