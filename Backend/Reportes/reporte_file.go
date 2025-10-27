package reportes

import (
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"os"
	"path/filepath"
	"strings"
)

func ReporteFile(superbloque *estructuras.Superbloque, rutaDisco string, ruta string, rutaArchivo string) error {
	err := utilidades.CrearDirectoriosPadre(ruta)
	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}

	archivo, err := os.Open(rutaDisco)

	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}

	defer archivo.Close()

	inodoIndice, err := encontrarArchivoInodo(superbloque, archivo, rutaArchivo)

	if err != nil {
		return fmt.Errorf("error al buscar el inodo del archivo: %v", err)
	}

	fileContent, err := leerContenidoArchivo(superbloque, archivo, inodoIndice)

	if err != nil {
		return fmt.Errorf("error al leer el contenido del archivo: %v", err)
	}

	reporteArchivo, err := os.Create(ruta)

	if err != nil {
		return fmt.Errorf("error al crear el archivo de reporte: %v", err)
	}

	defer reporteArchivo.Close()

	_, nombreArchivo := filepath.Split(rutaArchivo)
	reportContent := fmt.Sprintf("Nombre del archivo: %s\n\nContenido del archivo:\n%s", nombreArchivo, fileContent)

	_, err = reporteArchivo.WriteString(reportContent)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo de reporte: %v", err)
	}

	fmt.Println("Reporte del archivo generado:", ruta)
	return nil
}

func encontrarArchivoInodo(superbloque *estructuras.Superbloque, archivoDisco *os.File, rutaArchivo string) (int32, error) {
	indiceInodoActual := int32(0)

	directories, nombreArchivo := utilidades.ObtenerDirectoriosPadre(rutaArchivo)

	for _, directorio := range directories {
		inodo, err := leerInodo(superbloque, archivoDisco, indiceInodoActual)

		if err != nil {
			fmt.Printf("Error al leer el inodo: %v\n", err)
			return -1, err
		}

		encontrado, siguienteIndiceInodo := encontrarInodoEnDirectorio(inodo, archivoDisco, directorio, superbloque)

		if !encontrado {
			fmt.Printf("Directorio '%s' no encontrado\n", directorio)
			return -1, err
		}

		indiceInodoActual = siguienteIndiceInodo
	}

	inodo, err := leerInodo(superbloque, archivoDisco, indiceInodoActual)
	if err != nil {
		fmt.Printf("Error al leer el inodo del directorio final: %v\n", err)
		return -1, err
	}

	encontrado, archivoIndiceInodo := encontrarInodoEnDirectorio(inodo, archivoDisco, nombreArchivo, superbloque)
	if !encontrado {
		fmt.Printf("Archivo '%s' no encontrado\n", nombreArchivo)
		return -1, err
	}

	return archivoIndiceInodo, nil
}

func leerContenidoArchivo(superbloque *estructuras.Superbloque, archivoDisco *os.File, indiceInodo int32) (string, error) {
	inodo, err := leerInodo(superbloque, archivoDisco, indiceInodo)
	if err != nil {
		return "", fmt.Errorf("error al leer el inodo del archivo: %v", err)
	}

	var contenido string
	for _, indiceBloque := range inodo.I_block {
		if indiceBloque == -1 {
			continue
		}

		bloque, err := leerArchivoBloque(superbloque, archivoDisco, indiceBloque)
		if err != nil {
			return "", fmt.Errorf("error al leer el bloque de archivo: %v", err)
		}

		contenido += string(bloque.B_content[:])
	}

	return contenido, nil
}

func leerInodo(superblock *estructuras.Superbloque, archivoDisco *os.File, inodeIndex int32) (*estructuras.Inodo, error) {
	inodo := &estructuras.Inodo{}
	offset := int64(superblock.S_inode_start + inodeIndex*superblock.S_inode_size)
	err := inodo.Decode(archivoDisco, offset)

	if err != nil {
		return nil, fmt.Errorf("error al decodificar el inodo: %v", err)
	}

	return inodo, nil
}

func leerArchivoBloque(superblock *estructuras.Superbloque, archivoDisco *os.File, indiceBloque int32) (*estructuras.ArchivoBloque, error) {
	bloque := &estructuras.ArchivoBloque{}
	offset := int64(superblock.S_block_start + indiceBloque*superblock.S_block_size)
	err := bloque.Decode(archivoDisco, offset)
	if err != nil {
		return nil, fmt.Errorf("error al decodificar el bloque de archivo: %v", err)
	}
	return bloque, nil
}

func encontrarInodoEnDirectorio(inodo *estructuras.Inodo, archivoDisco *os.File, nombre string, superbloque *estructuras.Superbloque) (bool, int32) {
	for _, indiceBloque := range inodo.I_block {
		if indiceBloque == -1 {
			continue
		}

		bloque := &estructuras.FolderBlock{}
		offset := int64(superbloque.S_block_start + indiceBloque*superbloque.S_block_size)
		err := bloque.Decode(archivoDisco, offset)
		if err != nil {
			continue
		}

		for _, contenido := range bloque.B_content {
			nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
			if nombreContenido == nombre {
				return true, contenido.B_inodo
			}
		}
	}
	return false, -1
}
