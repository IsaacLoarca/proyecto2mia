#!/bin/bash

# Script para desplegar la aplicación en la instancia EC2 ya creada
# Este script copia el código y ejecuta el despliegue

set -e

# Colores para output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Verificar que existen los archivos necesarios
if [ ! -f "public-ip.txt" ] || [ ! -f "godisk-key.pem" ]; then
    echo "❌ Error: Ejecuta primero ./deploy-aws.sh para crear la infraestructura"
    exit 1
fi

PUBLIC_IP=$(cat public-ip.txt)
KEY_FILE="godisk-key.pem"

print_status "Desplegando aplicación en $PUBLIC_IP"

# Esperar a que la instancia esté completamente lista
print_status "Verificando conectividad..."
while ! ssh -i "$KEY_FILE" -o ConnectTimeout=5 -o StrictHostKeyChecking=no ec2-user@"$PUBLIC_IP" echo "Conexión exitosa" 2>/dev/null; do
    print_warning "Esperando a que la instancia esté lista..."
    sleep 10
done

print_status "Instancia accesible, continuando con el despliegue..."

# Crear directorio temporal para el despliegue
TEMP_DIR=$(mktemp -d)
print_status "Creando paquete de despliegue en $TEMP_DIR"

# Copiar archivos necesarios excluyendo directorios innecesarios
rsync -av \
    --exclude='node_modules' \
    --exclude='.git' \
    --exclude='*.log' \
    --exclude='dist' \
    --exclude='build' \
    --exclude='Pruebas' \
    --exclude='Enunciado' \
    --exclude='manuales' \
    --exclude='*.pem' \
    --exclude='*.txt' \
    ./ "$TEMP_DIR/godisk/"

# Copiar docker-compose con IP real
sed "s/yourdomain.com/$PUBLIC_IP/g" docker-compose.yml > "$TEMP_DIR/godisk/docker-compose.yml"

# Crear archivo de configuración de producción
S3_BUCKET=$(aws s3 ls | grep godisk-storage | awk '{print $3}' | head -1)

cat > "$TEMP_DIR/godisk/Backend/.env" << EOF
PORT=8080
HOST=0.0.0.0
GIN_MODE=release
AWS_REGION=us-east-1
S3_BUCKET_NAME=$S3_BUCKET
CORS_ORIGIN=http://$PUBLIC_IP
DISK_STORAGE_PATH=/home/ec2-user/godisk/disks
REPORTS_PATH=/home/ec2-user/godisk/reports
EOF

# Crear archivo de configuración para frontend
cat > "$TEMP_DIR/godisk/Frontend/.env.production" << EOF
VITE_API_URL=http://$PUBLIC_IP:8080
VITE_APP_TITLE=GODISK - Sistema de Gestión de Discos
VITE_APP_VERSION=1.0.0
VITE_ENVIRONMENT=production
EOF

# Copiar archivos a la instancia EC2
print_status "Copiando archivos a la instancia EC2..."
scp -i "$KEY_FILE" -o StrictHostKeyChecking=no -r "$TEMP_DIR/godisk" ec2-user@"$PUBLIC_IP":/home/ec2-user/

# Crear script de instalación remota
cat > "$TEMP_DIR/install-remote.sh" << 'EOF'
#!/bin/bash
cd /home/ec2-user/godisk

echo "🏗️  Configurando aplicación..."

# Crear directorios necesarios
mkdir -p disks reports

# Asegurar permisos correctos
sudo chown -R ec2-user:ec2-user /home/ec2-user/godisk

# Detener containers existentes si los hay
docker-compose down 2>/dev/null || true

echo "🏗️  Construyendo y desplegando aplicación..."

# Construir y ejecutar
docker-compose up --build -d

# Esperar a que los servicios estén listos
echo "⏳ Esperando a que los servicios estén listos..."
sleep 30

# Verificar estado de los contenedores
docker-compose ps

# Obtener IP pública de la instancia
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

echo ""
echo "✅ ¡Despliegue completado exitosamente!"
echo "🌐 Frontend disponible en: http://$PUBLIC_IP"
echo "🔗 API Backend disponible en: http://$PUBLIC_IP:8080"
echo "❤️  Health check: http://$PUBLIC_IP:8080/health"
echo ""
echo "📋 Para ver logs:"
echo "   docker-compose logs -f"
echo ""
echo "🔄 Para reiniciar:"
echo "   docker-compose restart"
EOF

# Copiar y ejecutar script de instalación
scp -i "$KEY_FILE" -o StrictHostKeyChecking=no "$TEMP_DIR/install-remote.sh" ec2-user@"$PUBLIC_IP":/home/ec2-user/
ssh -i "$KEY_FILE" -o StrictHostKeyChecking=no ec2-user@"$PUBLIC_IP" "chmod +x /home/ec2-user/install-remote.sh && /home/ec2-user/install-remote.sh"

# Limpiar archivos temporales
rm -rf "$TEMP_DIR"

print_status "¡Despliegue completado!"
echo ""
echo "🎉 ¡GODISK está ahora ejecutándose en AWS!"
echo "============================================="
echo "🌐 Accede a tu aplicación en: http://$PUBLIC_IP"
echo "🔗 API disponible en: http://$PUBLIC_IP:8080"
echo "❤️  Health check: http://$PUBLIC_IP:8080/health"
echo ""
echo "🔐 Para acceso SSH:"
echo "   ssh -i $KEY_FILE ec2-user@$PUBLIC_IP"
echo ""
echo "📊 Para monitorear:"
echo "   ssh -i $KEY_FILE ec2-user@$PUBLIC_IP 'cd godisk && docker-compose logs -f'"