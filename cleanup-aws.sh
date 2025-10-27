#!/bin/bash

# Script para limpiar recursos AWS creados por el despliegue
# ⚠️  CUIDADO: Este script eliminará todos los recursos creados

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

echo "🗑️  GODISK AWS Cleanup Script"
echo "============================"
print_warning "Este script eliminará TODOS los recursos de AWS creados para GODISK"
read -p "¿Estás seguro? (escribe 'DELETE' para confirmar): " confirmation

if [ "$confirmation" != "DELETE" ]; then
    echo "Operación cancelada."
    exit 0
fi

# Terminar instancia EC2
if [ -f "instance-id.txt" ]; then
    INSTANCE_ID=$(cat instance-id.txt)
    print_status "Terminando instancia EC2: $INSTANCE_ID"
    aws ec2 terminate-instances --instance-ids "$INSTANCE_ID"
    
    # Esperar a que la instancia se termine
    print_status "Esperando a que la instancia se termine..."
    aws ec2 wait instance-terminated --instance-ids "$INSTANCE_ID"
    rm instance-id.txt
fi

# Eliminar security group
if [ -f "security-group-id.txt" ]; then
    SECURITY_GROUP_ID=$(cat security-group-id.txt)
    print_status "Eliminando security group: $SECURITY_GROUP_ID"
    
    # Esperar un poco para que la instancia se termine completamente
    sleep 30
    
    aws ec2 delete-security-group --group-id "$SECURITY_GROUP_ID" || true
    rm security-group-id.txt
fi

# Eliminar key pair
if [ -f "godisk-key.pem" ]; then
    print_status "Eliminando key pair: godisk-key"
    aws ec2 delete-key-pair --key-name "godisk-key" || true
    rm godisk-key.pem
fi

# Vaciar y eliminar bucket S3
S3_BUCKET=$(aws s3 ls | grep godisk-storage | awk '{print $3}' | head -1)
if [ ! -z "$S3_BUCKET" ]; then
    print_status "Vaciando bucket S3: $S3_BUCKET"
    aws s3 rm "s3://$S3_BUCKET" --recursive || true
    
    print_status "Eliminando bucket S3: $S3_BUCKET"
    aws s3 rb "s3://$S3_BUCKET" || true
fi

# Limpiar archivos locales
rm -f public-ip.txt security-group-id.txt deploy-remote.sh

print_status "✅ Limpieza completada"
echo "Todos los recursos de AWS han sido eliminados."