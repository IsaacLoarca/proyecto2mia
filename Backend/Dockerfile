# Dockerfile para Backend
FROM golang:1.25.1-alpine AS builder

# Instalar dependencias necesarias
RUN apk add --no-cache git

WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o godisk .

# Imagen final
FROM alpine:latest

# Instalar certificados SSL
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Crear directorios necesarios
RUN mkdir -p /app/disks /app/reports

# Copiar el binario
COPY --from=builder /app/godisk .
COPY --from=builder /app/.env.example ./.env

# Exponer puerto
EXPOSE 8080

# Comando para ejecutar
CMD ["./godisk"]