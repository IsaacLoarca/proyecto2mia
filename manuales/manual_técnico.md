# Manual Técnico - Sistema de Gestión de Discos (GODISK)

## Descripción General

GODISK es un sistema completo de gestión de discos virtuales que simula el comportamiento de un sistema de archivos real. El proyecto implementa un backend en Go y un frontend en React+TypeScript, proporcionando funcionalidades completas de administración de discos, particiones, sistemas de archivos y gestión de usuarios.

## Arquitectura del Sistema

### Arquitectura General
```
┌─────────────────┐    HTTP/JSON    ┌──────────────────┐
│   Frontend      │ ◄─────────────► │    Backend       │
│   React + TS    │                 │    Go + Gin      │
└─────────────────┘                 └──────────────────┘
         │                                   │
         │                                   │
         ▼                                   ▼
┌─────────────────┐                 ┌──────────────────┐
│  Navegador Web  │                 │  Sistema Archivos│
│  Monaco Editor  │                 │  Archivos .mia   │
└─────────────────┘                 └──────────────────┘
```

### Tecnologías Utilizadas

#### Backend
- **Lenguaje**: Go 1.21+
- **Framework Web**: Gin-Gonic v1.11.0
- **Arquitectura**: Modular con separación de responsabilidades

#### Frontend
- **Framework**: React 18+ con TypeScript
- **Build Tool**: Vite 7.1.7
- **Editor**: Monaco Editor para entrada de comandos
- **Comunicación**: HTTP REST con JSON

## Estructura del Proyecto

### Backend (/Backend)

```
Backend/
├── Servidor.go                 # Servidor principal y endpoints
├── go.mod                     # Dependencias Go
├── Analizador/               # Parser de comandos
│   └── Analizador.go
├── Estructuras/              # Estructuras de datos del sistema
│   ├── mbr.go               # Master Boot Record
│   ├── partition.go         # Particiones
│   ├── super_bloque.go      # Superbloque ext2/ext3
│   ├── inodo.go            # Inodos del sistema
│   ├── bitmap.go           # Mapas de bits
│   ├── usuario.go          # Usuarios del sistema
│   ├── grupo.go            # Grupos de usuarios
│   └── ...                 # Otras estructuras
├── Global/                  # Variables y estado global
│   ├── particiones_montadas.go
│   └── bloquesglobales.go
├── Instrucciones/           # Comandos implementados
│   ├── Discos/             # Gestión de discos
│   │   ├── mkdisk.go      # Crear discos
│   │   ├── fdisk.go       # Gestión particiones
│   │   ├── mount.go       # Montar particiones
│   │   └── ...
│   ├── Usuarios/           # Gestión de usuarios
│   │   ├── login.go       # Autenticación
│   │   ├── mkusr.go       # Crear usuarios
│   │   └── ...
│   ├── mkdir.go           # Crear directorios
│   ├── mkfile.go          # Crear archivos
│   ├── directorytree.go   # Árbol de directorios
│   └── ...
├── Reportes/               # Generación de reportes
│   ├── reporte_mbr.go     # Reporte MBR
│   ├── reporte_disk.go    # Reporte disco
│   └── ...
└── Utilidades/             # Funciones auxiliares
    └── Utilidades.go
```

### Frontend (/Frontend)

```
Frontend/
├── src/
│   ├── App.tsx                    # Componente principal
│   ├── main.tsx                  # Punto de entrada
│   ├── components/
│   │   ├── CodeEditor.tsx        # Editor Monaco
│   │   ├── FileSystemViewer.tsx  # Visor sistema archivos
│   │   ├── FileManagerActions.tsx # Acciones archivos
│   │   ├── LoginModal.tsx        # Modal de login
│   │   ├── Toolbar.tsx          # Barra de herramientas
│   │   └── FileContentViewer.tsx # Visor contenido
│   └── assets/                   # Recursos estáticos
├── public/                      # Archivos públicos
└── package.json                 # Dependencias Node.js
```

## Estructuras de Datos Principales

### Master Boot Record (MBR)
```go
type MBR struct {
    Mbr_size           int32      // Tamaño total del disco
    Mbr_creation_date  [19]byte   // Fecha de creación
    Mbr_disk_signature int32      // Firma única del disco
    Mbr_disk_fit       [1]byte    // Tipo de ajuste (F, B, W)
    Mbr_partition_1    Partition  // Partición primaria 1
    Mbr_partition_2    Partition  // Partición primaria 2
    Mbr_partition_3    Partition  // Partición primaria 3
    Mbr_partition_4    Partition  // Partición primaria 4
}
```

### Partición
```go
type Partition struct {
    Part_status [1]byte   // Estado (A=activa, I=inactiva)
    Part_type   [1]byte   // Tipo (P=primaria, E=extendida)
    Part_fit    [1]byte   // Ajuste (F, B, W)
    Part_start  int32     // Byte inicio partición
    Part_size   int32     // Tamaño en bytes
    Part_name   [16]byte  // Nombre de la partición
}
```

### Superbloque
```go
type Superbloque struct {
    S_filesystem_type   int32  // Tipo sistema archivos
    S_inodes_count      int32  // Total inodos
    S_blocks_count      int32  // Total bloques
    S_free_blocks_count int32  // Bloques libres
    S_free_inodes_count int32  // Inodos libres
    S_mtime            [19]byte // Fecha montaje
    S_umtime           [19]byte // Fecha desmontaje
    S_mnt_count         int32   // Contador montajes
    S_magic             int32   // Número mágico
    S_inode_size        int32   // Tamaño inodo
    S_block_size        int32   // Tamaño bloque
    S_first_ino         int32   // Primer inodo libre
    S_first_blo         int32   // Primer bloque libre
    S_bm_inode_start    int32   // Inicio bitmap inodos
    S_bm_block_start    int32   // Inicio bitmap bloques
    S_inode_start       int32   // Inicio tabla inodos
    S_block_start       int32   // Inicio bloques datos
}
```

### Inodo
```go
type Inodo struct {
    I_uid   int32     // ID usuario propietario
    I_gid   int32     // ID grupo propietario
    I_size  int32     // Tamaño archivo/directorio
    I_atime [19]byte  // Último acceso
    I_ctime [19]byte  // Creación
    I_mtime [19]byte  // Modificación
    I_block [15]int32 // Bloques datos (12 directos + 3 indirectos)
    I_type  [1]byte   // Tipo (0=archivo, 1=directorio)
    I_perm  [3]byte   // Permisos
}
```

## API Endpoints

### Autenticación
- **POST /login**: Autenticación de usuario
  ```json
  Request: {"user": "root", "pass": "123", "id": "461A"}
  Response: {"status": "success", "user": {...}}
  ```
- **POST /logout**: Cerrar sesión
- **GET /session**: Verificar sesión activa

### Sistema de Archivos
- **GET /directory-tree**: Obtener árbol de directorios
  ```json
  Response: {"success": true, "tree": {...}}
  ```
- **POST /analizar**: Ejecutar comandos del sistema
  ```
  Body: "mkdisk -size=5 -unit=M -path=/disco.mia"
  ```

### Salud del Sistema
- **GET /health**: Estado del servidor

## Comandos Implementados

### Gestión de Discos
- **mkdisk**: Crear disco virtual
  ```
  mkdisk -size=5 -unit=M -path=/ruta/disco.mia -fit=FF
  ```
- **rmdisk**: Eliminar disco virtual
- **fdisk**: Gestionar particiones
  ```
  fdisk -size=1024 -path=/disco.mia -name=Particion1 -unit=K -type=P -fit=BF -add
  ```
- **mount**: Montar particiones
  ```
  mount -path=/disco.mia -name=Particion1
  ```
- **unmount**: Desmontar particiones
- **mkfs**: Crear sistema de archivos
  ```
  mkfs -type=full -id=461A -fs=3fs
  ```

### Gestión de Usuarios
- **login**: Iniciar sesión
  ```
  login -user=root -pass=123 -id=461A
  ```
- **logout**: Cerrar sesión
- **mkusr**: Crear usuario
- **rmusr**: Eliminar usuario
- **mkgrp**: Crear grupo
- **rmgrp**: Eliminar grupo
- **chgrp**: Cambiar grupo

### Sistema de Archivos
- **mkdir**: Crear directorios
  ```
  mkdir -path=/nueva/carpeta -p
  ```
- **mkfile**: Crear archivos
  ```
  mkfile -path=/archivo.txt -size=100 -cont="contenido"
  ```
- **cat**: Mostrar contenido archivo
- **remove**: Eliminar archivo/directorio
- **rename**: Renombrar archivo/directorio
- **edit**: Editar archivo
- **find**: Buscar archivos/directorios

### Reportes
- **rep**: Generar reportes
  ```
  rep -id=461A -path=/reporte.jpg -name=mbr
  rep -id=461A -path=/reporte.jpg -name=tree
  ```

## Patrones de Diseño Utilizados

### Backend
- **Patrón Singleton**: Para gestión de estado global
- **Patrón Factory**: Para creación de estructuras
- **Patrón Command**: Para procesamiento de comandos
- **Patrón Repository**: Para acceso a datos del disco

### Frontend
- **Patrón Observer**: Para actualizaciones de estado
- **Patrón Component**: Arquitectura basada en componentes React
- **Patrón Hook**: Para gestión de estado local

## Flujo de Datos

### Creación de Archivos/Directorios
```
Frontend → POST /analizar → Analizador → Comando específico → 
Validaciones → Escritura en disco → Respuesta JSON → Frontend
```

### Visualización Sistema de Archivos
```
Frontend → GET /directory-tree → DirectoryTreeService → 
Lectura inodos → Construcción árbol → JSON → Frontend
```

## Algoritmos de Ajuste de Particiones

### First Fit (FF)
Busca el primer espacio disponible que sea suficiente para la partición.

### Best Fit (BF)
Busca el espacio disponible más pequeño que sea suficiente para la partición.

### Worst Fit (WF)
Busca el espacio disponible más grande para la partición.

## Sistema de Archivos Ext2/Ext3

### Estructura
- **Superbloque**: Metadatos del sistema de archivos
- **Bitmap de Inodos**: Marca inodos libres/ocupados
- **Bitmap de Bloques**: Marca bloques libres/ocupados
- **Tabla de Inodos**: Array de inodos
- **Bloques de Datos**: Contenido real de archivos/directorios

### Journaling (Ext3)
- **Journal**: Registro de transacciones
- **Recovery**: Recuperación ante fallos
- **Consistencia**: Garantía de integridad

## Consideraciones de Seguridad

- **Autenticación**: Sistema de login obligatorio
- **Validación de entrada**: Sanitización de comandos
- **Permisos**: Sistema de permisos usuario/grupo
- **CORS**: Configurado para desarrollo local

## Configuración de Desarrollo

### Backend
```bash
cd Backend
go mod tidy
go run Servidor.go
```

### Frontend
```bash
cd Frontend
npm install
npm run dev
```

## Despliegue en la Nube

### Preparación Backend
1. Compilar binario: `go build -o godisk Servidor.go`
2. Configurar variables de entorno
3. Exponer puerto 8080

### Preparación Frontend
1. Build de producción: `npm run build`
2. Servir archivos estáticos
3. Configurar proxy para API

### Consideraciones Cloud
- **Persistencia**: Almacenar archivos .mia en volumen persistente
- **Escalabilidad**: Configurar load balancer si necesario
- **Monitoreo**: Implementar logs y métricas
- **Backup**: Sistema de respaldo automático de discos virtuales

## Manejo de Errores

### Backend
- Validación de parámetros de entrada
- Manejo de errores de E/O de archivos
- Validación de permisos de usuario
- Gestión de memoria y recursos

### Frontend
- Validación de formularios
- Manejo de errores de red
- Feedback visual al usuario
- Recuperación de estados de error

## Optimizaciones

### Performance
- Caching de estructuras frecuentemente accedidas
- Lectura/escritura en bloque
- Índices para búsqueda rápida
- Compresión de datos donde sea posible

### Memoria
- Liberación explícita de recursos
- Reutilización de buffers
- Lazy loading de estructuras grandes
- Garbage collection eficiente
