# 🚀 Despliegue de GODISK en AWS

Este proyecto incluye scripts automatizados para desplegar GODISK en Amazon Web Services usando EC2 y S3.

## 📋 Prerrequisitos

### 1. AWS CLI Instalado
```bash
# Descargar e instalar AWS CLI v2
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# Verificar instalación
aws --version
```

### 2. Configurar Credenciales AWS
```bash
aws configure
# AWS Access Key ID: tu-access-key
# AWS Secret Access Key: tu-secret-key  
# Default region: us-east-1
# Default output format: json
```

### 3. Permisos IAM Requeridos
Tu usuario AWS debe tener permisos para:
- EC2 (crear/eliminar instancias, security groups, key pairs)
- S3 (crear/eliminar buckets, objetos)
- IAM (para roles si es necesario)

## 🎯 Despliegue Paso a Paso

### Paso 1: Crear Infraestructura AWS
```bash
./deploy-aws.sh
```

Este script:
- ✅ Crea bucket S3 para almacenamiento
- ✅ Crea security group con reglas de firewall
- ✅ Genera key pair para acceso SSH
- ✅ Lanza instancia EC2 t3.micro (Free Tier)
- ✅ Instala Docker y dependencias

**Tiempo estimado:** 3-5 minutos

### Paso 2: Desplegar Aplicación
```bash
./deploy-app.sh
```

Este script:
- ✅ Copia código fuente a EC2
- ✅ Configura variables de entorno
- ✅ Construye containers Docker
- ✅ Despliega frontend y backend

**Tiempo estimado:** 5-10 minutos

## 🌐 Acceso a la Aplicación

Una vez completado el despliegue:

```
🌐 Frontend: http://TU-IP-PUBLICA
🔗 Backend API: http://TU-IP-PUBLICA:8080
❤️  Health Check: http://TU-IP-PUBLICA:8080/health
```

## 🔐 Acceso SSH a la Instancia

```bash
ssh -i godisk-key.pem ec2-user@TU-IP-PUBLICA
```

## 📊 Monitoreo y Logs

### Ver logs de la aplicación:
```bash
ssh -i godisk-key.pem ec2-user@TU-IP-PUBLICA
cd godisk
docker-compose logs -f
```

### Estado de los containers:
```bash
docker-compose ps
```

### Reiniciar servicios:
```bash
docker-compose restart
```

## 🔧 Configuración Personalizada

### Variables de Entorno (Backend)
Edita `Backend/.env` antes del despliegue:

```env
PORT=8080
HOST=0.0.0.0
GIN_MODE=release
AWS_REGION=us-east-1
S3_BUCKET_NAME=tu-bucket
CORS_ORIGIN=http://tu-dominio.com
```

### Variables de Entorno (Frontend)
Edita `Frontend/.env.production`:

```env
VITE_API_URL=http://tu-dominio.com:8080
VITE_APP_TITLE=GODISK
VITE_ENVIRONMENT=production
```

## 💰 Costos Estimados AWS

### Instancia EC2 t3.micro:
- **Free Tier**: 750 horas/mes gratis (primer año)
- **Después**: ~$8.50/mes

### Almacenamiento S3:
- **Free Tier**: 5GB gratis (primer año)  
- **Después**: ~$0.023/GB/mes

### Transferencia de Datos:
- **Free Tier**: 100GB/mes gratis
- **Después**: ~$0.09/GB

**Total estimado**: $0-15/mes dependiendo del uso

## 🛠️ Comandos Útiles

### Actualizar aplicación:
```bash
# En tu máquina local
./deploy-app.sh
```

### Ver recursos creados:
```bash
# Listar instancias
aws ec2 describe-instances --filters "Name=tag:Name,Values=GODISK-Server"

# Listar buckets S3  
aws s3 ls | grep godisk

# Ver security groups
aws ec2 describe-security-groups --filters "Name=group-name,Values=godisk-sg"
```

### Backup de discos virtuales:
```bash
# En la instancia EC2
cd /home/ec2-user/godisk
tar -czf backup-$(date +%Y%m%d).tar.gz disks/
aws s3 cp backup-*.tar.gz s3://tu-bucket/backups/
```

## 🧹 Limpieza y Eliminación

### ⚠️  Eliminar TODOS los recursos:
```bash
./cleanup-aws.sh
```

**CUIDADO**: Este comando eliminará permanentemente:
- ❌ Instancia EC2
- ❌ Security groups  
- ❌ Key pairs
- ❌ Bucket S3 y todos sus archivos
- ❌ Todos los discos virtuales creados

## 🔒 Consideraciones de Seguridad

### Producción:
1. **Cambiar credenciales por defecto**
2. **Configurar HTTPS con certificado SSL**
3. **Restringir acceso SSH por IP**
4. **Configurar backup automático**
5. **Monitoreo con CloudWatch**

### SSL/HTTPS Setup:
```bash
# En la instancia EC2
sudo yum install certbot python3-certbot-nginx
sudo certbot --nginx -d tu-dominio.com
```

## 🐛 Solución de Problemas

### Error: "No se puede conectar a la instancia"
```bash
# Verificar estado de la instancia
aws ec2 describe-instances --instance-ids i-tu-instance-id

# Verificar security group permite puerto 22
aws ec2 describe-security-groups --group-ids sg-tu-security-group
```

### Error: "Aplicación no responde"
```bash
ssh -i godisk-key.pem ec2-user@TU-IP-PUBLICA
sudo docker-compose logs
sudo docker-compose restart
```

### Error: "Sin permisos AWS"
```bash
# Verificar credenciales
aws sts get-caller-identity

# Verificar permisos
aws iam get-user
```

## 📞 Soporte

Para problemas específicos del despliegue:
1. Verificar logs: `docker-compose logs`
2. Revisar security groups y puertos
3. Comprobar variables de entorno
4. Validar conectividad de red

## 📚 Recursos Adicionales

- [AWS EC2 Documentation](https://docs.aws.amazon.com/ec2/)
- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Nginx Configuration](https://nginx.org/en/docs/)