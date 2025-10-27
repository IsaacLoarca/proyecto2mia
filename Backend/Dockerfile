# ===== STAGE 1: build =====
FROM golang:1.25.1-alpine AS builder

# Dependencias mínimas para compilar
RUN apk add --no-cache git

WORKDIR /app

# Paso 1: copiar solo mod/sum para aprovechar caché de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Paso 2: copiar el resto del código
COPY . .

# Paso 3: compilar binario estático
RUN CGO_ENABLED=0 GOOS=linux go build -p 1 -o godisk .

# ===== STAGE 2: runtime =====
FROM alpine:latest

# Certificados SSL y zona horaria (útil si hacés HTTPS o logs con hora local)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Directorios que tu app usa en runtime
RUN mkdir -p /app/disks /app/reports

# Copiar el binario ya compilado desde la etapa builder
COPY --from=builder /app/godisk ./godisk

# Exponer el puerto donde escucha tu servidor Go
EXPOSE 8080

# Comando final
CMD ["./godisk"]
