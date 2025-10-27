#!/bin/bash

# Script de despliegue en AWS EC2
# Este script configura y despliega GODISK en una instancia EC2

set -e

echo "🚀 Iniciando despliegue de GODISK en AWS EC2..."

# Variables de configuración
REGION="us-east-1"
INSTANCE_TYPE="t3.micro"  # Free tier eligible
KEY_NAME="godisk-key"
SECURITY_GROUP="godisk-sg"
AMI_ID="ami-0c55b159cbfafe1d0"  # Amazon Linux 2
S3_BUCKET="godisk-storage-$(date +%s)"

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Función para verificar si AWS CLI está instalado
check_aws_cli() {
    if ! command -v aws &> /dev/null; then
        print_error "AWS CLI no está instalado. Por favor instálalo primero:"
        echo "curl 'https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip' -o 'awscliv2.zip'"
        echo "unzip awscliv2.zip"
        echo "sudo ./aws/install"
        exit 1
    fi
    print_status "AWS CLI encontrado"
}

# Función para verificar credenciales AWS
check_aws_credentials() {
    if ! aws sts get-caller-identity &> /dev/null; then
        print_error "No hay credenciales AWS configuradas. Ejecuta: aws configure"
        exit 1
    fi
    print_status "Credenciales AWS verificadas"
}

# Función para crear S3 bucket
create_s3_bucket() {
    print_status "Creando bucket S3: $S3_BUCKET"
    
    # Crear bucket
    aws s3 mb "s3://$S3_BUCKET" --region "$REGION"
    
    # Configurar policy del bucket
    cat > bucket-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowGodiskAccess",
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):root"
            },
            "Action": [
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject"
            ],
            "Resource": "arn:aws:s3:::$S3_BUCKET/*"
        }
    ]
}
EOF
    
    aws s3api put-bucket-policy --bucket "$S3_BUCKET" --policy file://bucket-policy.json
    rm bucket-policy.json
    
    print_status "Bucket S3 creado correctamente"
}

# Función para crear security group
create_security_group() {
    print_status "Creando security group: $SECURITY_GROUP"
    
    # Crear security group
    SECURITY_GROUP_ID=$(aws ec2 create-security-group \
        --group-name "$SECURITY_GROUP" \
        --description "Security group for GODISK application" \
        --query 'GroupId' --output text)
    
    # Agregar reglas de firewall
    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp --port 22 --cidr 0.0.0.0/0  # SSH
    
    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp --port 80 --cidr 0.0.0.0/0  # HTTP
    
    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp --port 443 --cidr 0.0.0.0/0  # HTTPS
    
    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp --port 8080 --cidr 0.0.0.0/0  # Backend API
    
    print_status "Security group creado: $SECURITY_GROUP_ID"
    echo "$SECURITY_GROUP_ID" > security-group-id.txt
}

# Función para crear key pair
create_key_pair() {
    print_status "Creando key pair: $KEY_NAME"
    
    aws ec2 create-key-pair \
        --key-name "$KEY_NAME" \
        --query 'KeyMaterial' --output text > "${KEY_NAME}.pem"
    
    chmod 400 "${KEY_NAME}.pem"
    print_status "Key pair creado: ${KEY_NAME}.pem"
}

# Función para crear instancia EC2
create_ec2_instance() {
    print_status "Creando instancia EC2..."
    
    # User data script para configurar la instancia
    cat > user-data.sh << 'EOF'
#!/bin/bash
yum update -y

# Instalar Docker
amazon-linux-extras install docker -y
systemctl start docker
systemctl enable docker
usermod -a -G docker ec2-user

# Instalar Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Instalar Git
yum install git -y

# Crear directorio para la aplicación
mkdir -p /home/ec2-user/godisk
chown ec2-user:ec2-user /home/ec2-user/godisk

# Instalar Node.js (para builds si es necesario)
curl -fsSL https://rpm.nodesource.com/setup_18.x | bash -
yum install -y nodejs

# Configurar firewall
yum install -y iptables-services
systemctl start iptables
systemctl enable iptables

echo "✓ Configuración inicial completada" > /var/log/user-data.log
EOF

    # Crear instancia
    INSTANCE_ID=$(aws ec2 run-instances \
        --image-id "$AMI_ID" \
        --count 1 \
        --instance-type "$INSTANCE_TYPE" \
        --key-name "$KEY_NAME" \
        --security-group-ids "$(cat security-group-id.txt)" \
        --user-data file://user-data.sh \
        --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=GODISK-Server}]' \
        --query 'Instances[0].InstanceId' --output text)
    
    print_status "Instancia EC2 creada: $INSTANCE_ID"
    echo "$INSTANCE_ID" > instance-id.txt
    
    # Esperar a que la instancia esté corriendo
    print_status "Esperando a que la instancia esté lista..."
    aws ec2 wait instance-running --instance-ids "$INSTANCE_ID"
    
    # Obtener IP pública
    PUBLIC_IP=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' --output text)
    
    print_status "Instancia lista. IP pública: $PUBLIC_IP"
    echo "$PUBLIC_IP" > public-ip.txt
    
    rm user-data.sh
}

# Función para preparar archivos de despliegue
prepare_deployment_files() {
    print_status "Preparando archivos de despliegue..."
    
    # Crear script de despliegue remoto
    cat > deploy-remote.sh << EOF
#!/bin/bash
cd /home/ec2-user/godisk

# Clonar repositorio (o copiar archivos)
echo "📥 Descargando código fuente..."

# Crear archivo .env para producción
cat > Backend/.env << EOL
PORT=8080
HOST=0.0.0.0
GIN_MODE=release
AWS_REGION=$REGION
S3_BUCKET_NAME=$S3_BUCKET
CORS_ORIGIN=http://$(cat /home/ec2-user/public-ip.txt)
DISK_STORAGE_PATH=/home/ec2-user/godisk/disks
REPORTS_PATH=/home/ec2-user/godisk/reports
EOL

# Crear directorios necesarios
mkdir -p disks reports

# Construir y ejecutar con Docker Compose
echo "🏗️  Construyendo aplicación..."
docker-compose up --build -d

echo "✅ Despliegue completado!"
echo "🌐 Frontend: http://\$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)"
echo "🔗 Backend API: http://\$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4):8080"
EOF

    chmod +x deploy-remote.sh
}

# Función principal
main() {
    echo "🚀 GODISK AWS Deployment Script"
    echo "================================"
    
    # Verificaciones previas
    check_aws_cli
    check_aws_credentials
    
    # Crear recursos AWS
    create_s3_bucket
    create_security_group
    create_key_pair
    create_ec2_instance
    prepare_deployment_files
    
    PUBLIC_IP=$(cat public-ip.txt)
    
    echo ""
    echo "🎉 ¡Despliegue inicial completado!"
    echo "=================================="
    echo "📋 Recursos creados:"
    echo "   • Instancia EC2: $(cat instance-id.txt)"
    echo "   • IP Pública: $PUBLIC_IP"
    echo "   • Security Group: $(cat security-group-id.txt)"
    echo "   • S3 Bucket: $S3_BUCKET"
    echo "   • Key Pair: ${KEY_NAME}.pem"
    echo ""
    echo "📝 Próximos pasos:"
    echo "1. Espera 2-3 minutos para que la instancia termine de configurarse"
    echo "2. Ejecuta: ./deploy-app.sh"
    echo "3. Accede a tu aplicación en: http://$PUBLIC_IP"
    echo ""
    echo "🔐 Para conectarte via SSH:"
    echo "   ssh -i ${KEY_NAME}.pem ec2-user@$PUBLIC_IP"
}

# Ejecutar script principal
main "$@"