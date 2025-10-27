package estructuras

import (
	"fmt"
	utilidades "godisk/Utilidades"
	"os"
	"strings"
	"time"
)

func (sb *Superbloque) CrearCarpeta(archivo *os.File, directorios []string, destDir string, p bool) error {
	var padreEncontrado bool
	var indiceInodoEncontrado int32
	for i := int32(0); i < sb.S_inodes_count; i++ {
		indiceInodo, encontrado, err := sb.ValidarPadreDirectorio(archivo, i, directorios, destDir, p)
		if encontrado {
			padreEncontrado = true
			indiceInodoEncontrado = int32(indiceInodo)
			break
		}

		if err != nil {
			return err
		}
	}

	var siExisteCarpeta bool
	for i := int32(0); i < sb.S_inodes_count; i++ {
		bandera, err := sb.ValidarExistenciaCarpeta(archivo, i, directorios, destDir, p)

		if bandera {
			siExisteCarpeta = true
			break
		}

		if err != nil {
			return err
		}
	}

	if len(directorios) == 1 {
		padreEncontrado = true
	}

	if siExisteCarpeta {
		return nil
	} else if padreEncontrado {
		err := sb.CrearCarpetaEnInodo(archivo, int32(indiceInodoEncontrado), directorios, destDir, p)
		if err != nil {
			return err
		}
	} else if len(directorios) == 0 {
		err := sb.CrearCarpetaEnInodo(archivo, 0, directorios, destDir, p)
		if err != nil {
			return err
		}
	} else if p {
		err := sb.CrearCarpetaEnInodo(archivo, 0, directorios, destDir, p)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("no se encontró la carpeta padre de: %s\n", destDir)
		return fmt.Errorf("no se encontró la carpeta padre de: %s", destDir)
	}

	return nil
}

func (sb *Superbloque) ValidarExistenciaDeDirectorio(archivo *os.File, indiceInodo int32, destDir string) (bool, error) {
	inodo := &Inodo{}
	fmt.Printf("Deserializando inodo %d\n", indiceInodo)

	err := inodo.Decode(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))

	if err != nil {
		return false, fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}
	fmt.Printf("Inodo %d deserializado. Tipo: %c\n", indiceInodo, inodo.I_type[0])

	if inodo.I_type[0] != '0' {
		fmt.Printf("Inodo %d no es una carpeta, es de tipo: %c\n", indiceInodo, inodo.I_type[0])
		return false, nil
	}

	for _, indiceBloque := range inodo.I_block {
		if indiceBloque == -1 {
			fmt.Printf("Inodo %d no tiene más bloques asignados, terminando la búsqueda.\n", indiceInodo)
			break
		}

		fmt.Printf("Deserializando bloque %d del inodo %d\n", indiceBloque, indiceInodo)
		bloque := &FolderBlock{}

		err := bloque.Decode(archivo, int64(sb.S_block_start+(indiceBloque*sb.S_block_size)))

		if err != nil {
			return false, fmt.Errorf("error al deserializar bloque %d: %v", indiceBloque, err)
		}

		fmt.Printf("Bloque %d del inodo %d deserializado correctamente\n", indiceBloque, indiceInodo)

		for indiceContenido := 2; indiceContenido < len(bloque.B_content); indiceContenido++ {
			contenido := bloque.B_content[indiceContenido]
			fmt.Printf("Verificando contenido en índice %d del bloque %d\n", indiceContenido, indiceBloque)

			if contenido.B_inodo == -1 {
				fmt.Printf("No se encontró carpeta padre en inodo %d en la posición %d, terminando.\n", indiceInodo, indiceContenido)
				break
			}

			nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
			nombreDirectorio := strings.Trim(destDir, "\x00 ")
			fmt.Printf("Comparando '%s' con el nombre de la carpeta padre '%s'\n", nombreContenido, nombreDirectorio)

			if strings.EqualFold(nombreContenido, nombreDirectorio) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (sb *Superbloque) ValidarExistenciaCarpeta(archivo *os.File, indiceInodo int32, padresDir []string, destDir string, p bool) (bool, error) {

	inodo := &Inodo{}
	fmt.Printf("Deserializando inodo %d\n", indiceInodo)

	err := inodo.Decode(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))

	if err != nil {
		return false, fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}
	fmt.Printf("Inodo %d deserializado. Tipo: %c\n", indiceInodo, inodo.I_type[0])

	if inodo.I_type[0] != '0' {
		fmt.Printf("Inodo %d no es una carpeta, es de tipo: %c\n", indiceInodo, inodo.I_type[0])
		return false, nil
	}

	for _, indiceBloque := range inodo.I_block {
		if indiceBloque == -1 {
			fmt.Printf("Inodo %d no tiene más bloques asignados, terminando la búsqueda.\n", indiceInodo)
			break
		}

		fmt.Printf("Deserializando bloque %d del inodo %d\n", indiceBloque, indiceInodo)
		bloque := &FolderBlock{}

		err := bloque.Decode(archivo, int64(sb.S_block_start+(indiceBloque*sb.S_block_size)))

		if err != nil {
			return false, fmt.Errorf("error al deserializar bloque %d: %v", indiceBloque, err)
		}

		fmt.Printf("Bloque %d del inodo %d deserializado correctamente\n", indiceBloque, indiceInodo)

		for indiceContenido := 2; indiceContenido < len(bloque.B_content); indiceContenido++ {
			contenido := bloque.B_content[indiceContenido]
			fmt.Printf("Verificando contenido en índice %d del bloque %d\n", indiceContenido, indiceBloque)

			if contenido.B_inodo == -1 {
				fmt.Printf("No se encontró carpeta padre en inodo %d en la posición %d, terminando.\n", indiceInodo, indiceContenido)
				break
			}

			nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
			fmt.Printf("Comparando '%s' con el nombre de la carpeta existente '%s'\n", nombreContenido, destDir)
			if strings.EqualFold(nombreContenido, destDir) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (sb *Superbloque) ValidarPadreDirectorio(archivo *os.File, indiceInodo int32, padresDir []string, destDir string, p bool) (int, bool, error) {
	inodo := &Inodo{}
	fmt.Printf("Deserializando inodo %d\n", indiceInodo)

	err := inodo.Decode(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))

	if err != nil {
		return 0, false, fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}
	fmt.Printf("Inodo %d deserializado. Tipo: %c\n", indiceInodo, inodo.I_type[0])

	if inodo.I_type[0] != '0' {
		fmt.Printf("Inodo %d no es una carpeta, es de tipo: %c\n", indiceInodo, inodo.I_type[0])
		return 0, false, nil
	}

	for _, indiceBloque := range inodo.I_block {
		if indiceBloque == -1 {
			fmt.Printf("Inodo %d no tiene más bloques asignados, terminando la búsqueda.\n", indiceInodo)
			break
		}

		fmt.Printf("Deserializando bloque %d del inodo %d\n", indiceBloque, indiceInodo)
		bloque := &FolderBlock{}
		err := bloque.Decode(archivo, int64(sb.S_block_start+(indiceBloque*sb.S_block_size)))

		if err != nil {
			return 0, false, fmt.Errorf("error al deserializar bloque %d: %v", indiceBloque, err)
		}

		fmt.Printf("Bloque %d del inodo %d deserializado correctamente\n", indiceBloque, indiceInodo)

		for indiceContenido := 2; indiceContenido < len(bloque.B_content); indiceContenido++ {
			contenido := bloque.B_content[indiceContenido]
			fmt.Printf("Verificando contenido en índice %d del bloque %d\n", indiceContenido, indiceBloque)

			if contenido.B_inodo == -1 {
				break
			}

			padreDir, _ := utilidades.PadreCarpeta(padresDir, destDir)

			if padreDir == "" {
				padreDir = padresDir[0]
			}

			nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
			nombreDirPadre := strings.Trim(padreDir, "\x00 ")
			fmt.Printf("Comparando '%s' con el nombre de la carpeta padre '%s'\n", nombreContenido, nombreDirPadre)

			if strings.EqualFold(nombreContenido, nombreDirPadre) {
				return int(contenido.B_inodo), true, nil
			}
		}
	}

	return 0, false, nil
}

func (sb *Superbloque) CrearCarpetaEnInodo(archivo *os.File, indiceInodo int32, padresDir []string, destDir string, p bool) error {
	inodo := &Inodo{}
	fmt.Printf("Deserializando inodo %d\n", indiceInodo)

	err := inodo.Decode(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}
	fmt.Printf("Inodo %d deserializado. Tipo: %c\n", indiceInodo, inodo.I_type[0])

	if inodo.I_type[0] != '0' {
		fmt.Printf("Inodo %d no es una carpeta, es de tipo: %c\n", indiceInodo, inodo.I_type[0])
		return nil
	}

	seCreoArchivo := false
	for _, indiceBloque := range inodo.I_block {
		if indiceBloque == -1 {
			if !seCreoArchivo {
				fmt.Printf("Creando nuevo Bloque Carpeta para poder crear el directorio\n")
				err := sb.CrearNuevoBloqueCarpeta(archivo, int32(indiceInodo), padresDir, destDir, 0, nil, p, 0)
				if err != nil {
					fmt.Printf("Error con la creación de la carpeta que contendrá el directorio: %s", destDir)
					return err
				}
			}
			return nil
		}

		fmt.Printf("Deserializando bloque %d del inodo %d\n", indiceBloque, indiceInodo)
		bloque := &FolderBlock{}

		err := bloque.Decode(archivo, int64(sb.S_block_start+(indiceBloque*sb.S_block_size)))

		if err != nil {
			return fmt.Errorf("error al deserializar bloque %d: %v", indiceBloque, err)
		}

		fmt.Printf("Bloque %d del inodo %d deserializado correctamente\n", indiceBloque, indiceInodo)

		for indiceContenido := 2; indiceContenido < len(bloque.B_content); indiceContenido++ {
			contenido := bloque.B_content[indiceContenido]
			fmt.Printf("Verificando contenido en índice %d del bloque %d\n", indiceContenido, indiceBloque)

			if contenido.B_inodo != -1 {
				fmt.Printf("El inodo %d ya está ocupado con otro contenido, saltando al siguiente.\n", contenido.B_inodo)
				continue
			}

			fmt.Printf("Asignando el nombre del directorio '%s' al bloque en la posición %d\n", destDir, indiceContenido)

			copy(contenido.B_name[:], destDir)
			contenido.B_inodo = sb.S_inodes_count

			bloque.B_content[indiceContenido] = contenido

			err = bloque.Encode(archivo, int64(sb.S_block_start+(indiceBloque*sb.S_block_size)))

			if err != nil {
				return fmt.Errorf("error al serializar el bloque %d: %v", indiceBloque, err)
			}

			fmt.Printf("Bloque %d actualizado con éxito.\n", indiceBloque)

			inodoCarpeta := &Inodo{
				I_uid:   1,
				I_gid:   1,
				I_size:  0,
				I_atime: float32(time.Now().Unix()),
				I_ctime: float32(time.Now().Unix()),
				I_mtime: float32(time.Now().Unix()),
				I_block: [15]int32{sb.S_blocks_count, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
				I_type:  [1]byte{'0'},
				I_perm:  [3]byte{'6', '6', '4'},
			}

			fmt.Printf("Serializando el inodo de la carpeta '%s' (inodo %d)\n", destDir, sb.S_inodes_count)

			err = inodoCarpeta.Encode(archivo, int64(sb.S_first_ino))

			if err != nil {
				return fmt.Errorf("error al serializar el inodo del directorio '%s': %v", destDir, err)
			}

			err = sb.UpdateBitmapInode(archivo, sb.S_inodes_count, true)

			if err != nil {
				return fmt.Errorf("error al actualizar el bitmap de inodos para el directorio '%s': %v", destDir, err)
			}

			sb.UpdateSuperblockAfterInodeAllocation()

			bloqueCarpeta := &FolderBlock{
				B_content: [4]FolderContent{
					{B_name: [12]byte{'.'}, B_inodo: contenido.B_inodo},
					{B_name: [12]byte{'.', '.'}, B_inodo: indiceInodo},
					{B_name: [12]byte{'-'}, B_inodo: -1},
					{B_name: [12]byte{'-'}, B_inodo: -1},
				},
			}

			fmt.Printf("Serializando el bloque de la carpeta '%s'\n", destDir)

			err = bloqueCarpeta.Encode(archivo, int64(sb.S_first_blo))

			if err != nil {
				return fmt.Errorf("error al serializar el bloque del directorio '%s': %v", destDir, err)
			}

			err = sb.UpdateBitmapBlock(archivo, sb.S_blocks_count, true)

			if err != nil {
				return fmt.Errorf("error al actualizar el bitmap de bloques para el directorio '%s': %v", destDir, err)
			}

			sb.UpdateSuperblockAfterBlockAllocation()

			fmt.Printf("Directorio '%s' creado correctamente en inodo %d.\n", destDir, sb.S_inodes_count)
			return nil
		}
	}

	fmt.Printf("No se encontraron bloques disponibles para crear la carpeta '%s' en inodo %d\n", destDir, indiceInodo)
	return nil
}

func (sb *Superbloque) deleteFolderInInode(file *os.File, inodeIndex int32, dirPath ...string) error {
	dirInode := &Inodo{}
	err := dirInode.Decode(file, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar inodo %d: %w", inodeIndex, err)
	}

	if dirInode.I_type[0] != '0' {
		return fmt.Errorf("el inodo %d no es una carpeta", inodeIndex)
	}

	fullPath := "/"
	if len(dirPath) > 0 && dirPath[0] != "" {
		fullPath = dirPath[0]
	}

	journaling_start := int64(sb.JournalStart())
	if sb.S_filesystem_type == 3 {
		if err := AddJournalEntry(
			file,
			journaling_start,
			JOURNAL_ENTRIES,
			"rmdir",
			fullPath,
			"",
			sb,
		); err != nil {
			fmt.Printf("Advertencia: error registrando operación en journal: %v\n", err)
		} else {
			fmt.Printf("Operación 'rmdir %s' registrada en journal correctamente\n", fullPath)
		}
	}

	blockIndexes, err := dirInode.GetDataBlockIndexes(file, sb)
	if err != nil {
		return fmt.Errorf("error obteniendo bloques de datos del directorio: %w", err)
	}

	for _, blockIndex := range blockIndexes {
		block := &FolderBlock{}
		blockOffset := int64(sb.S_block_start + blockIndex*sb.S_block_size)
		if err := block.Decode(file, blockOffset); err != nil {
			return fmt.Errorf("error deserializando bloque %d: %w", blockIndex, err)
		}

		for _, content := range block.B_content {
			if content.B_inodo == -1 ||
				string(content.B_name[:1]) == "." ||
				string(content.B_name[:2]) == ".." {
				continue
			}

			contentName := strings.Trim(string(content.B_name[:]), "\x00 ")
			fmt.Printf("Eliminando contenido '%s' en inodo %d\n", contentName, content.B_inodo)

			childPath := fullPath
			if !strings.HasSuffix(childPath, "/") {
				childPath += "/"
			}
			childPath += contentName

			childInode := &Inodo{}
			childInodeOffset := int64(sb.S_inode_start + (content.B_inodo * sb.S_inode_size))
			if err := childInode.Decode(file, childInodeOffset); err != nil {
				return fmt.Errorf("error deserializando inodo hijo %d: %w", content.B_inodo, err)
			}

			if childInode.I_type[0] == '0' {
				if sb.S_filesystem_type == 3 {
					if err := AddJournalEntry(
						file,
						journaling_start,
						JOURNAL_ENTRIES,
						"rmdir",
						childPath,
						"",
						sb,
					); err != nil {
						fmt.Printf("Advertencia: error registrando eliminación de subcarpeta en journal: %v\n", err)
					}
				}

				if err := sb.deleteFolderInInode(file, content.B_inodo, childPath); err != nil {
					return fmt.Errorf("error eliminando subcarpeta '%s': %w", contentName, err)
				}
			} else {
				if sb.S_filesystem_type == 3 {
					fileData, err := childInode.ReadData(file, sb)
					fileContent := ""
					if err == nil {
						fileContent = string(fileData)
					}

					if err := AddJournalEntry(
						file,
						journaling_start,
						JOURNAL_ENTRIES,
						"rm",
						childPath,
						fileContent,
						sb,
					); err != nil {
						fmt.Printf("Advertencia: error registrando eliminación de archivo '%s' en journal: %v\n", contentName, err)
					} else {
						fmt.Printf("Operación 'rm %s' registrada en journal correctamente\n", childPath)
					}
				}

				if err := childInode.FreeAllBlocks(file, sb); err != nil {
					return fmt.Errorf("error liberando bloques del archivo '%s': %w", contentName, err)
				}

				if err := sb.UpdateBitmapInode(file, content.B_inodo, false); err != nil {
					return fmt.Errorf("error liberando inodo %d: %w", content.B_inodo, err)
				}
				sb.UpdateSuperblockAfterInodeDeallocation()
				fmt.Printf("Archivo '%s' eliminado (inodo %d)\n", contentName, content.B_inodo)
			}
		}
	}

	if err := dirInode.FreeAllBlocks(file, sb); err != nil {
		return fmt.Errorf("error liberando bloques del directorio: %w", err)
	}

	if err := dirInode.CheckAndFreeEmptyIndirectBlocks(file, sb); err != nil {
		fmt.Printf("Advertencia: error al verificar bloques indirectos vacíos: %v\n", err)
	}

	if err := sb.UpdateBitmapInode(file, inodeIndex, false); err != nil {
		return fmt.Errorf("error liberando inodo del directorio %d: %w", inodeIndex, err)
	}
	sb.UpdateSuperblockAfterInodeDeallocation()

	fmt.Printf("Carpeta en inodo %d eliminada correctamente.\n", inodeIndex)
	return nil
}

func (sb *Superbloque) deleteFolderFromDirectory(file *os.File, parentInodeIndex int32, folderName string, fullPath string) error {
	parentInode := &Inodo{}
	if err := parentInode.Decode(file, int64(sb.S_inode_start+parentInodeIndex*sb.S_inode_size)); err != nil {
		return fmt.Errorf("error deserializando inodo del directorio padre %d: %w", parentInodeIndex, err)
	}

	if parentInode.I_type[0] != '0' {
		return fmt.Errorf("el inodo %d no es un directorio", parentInodeIndex)
	}

	blockIndexes, err := parentInode.GetDataBlockIndexes(file, sb)
	if err != nil {
		return fmt.Errorf("error obteniendo bloques de datos: %w", err)
	}

	for _, blockIndex := range blockIndexes {
		block := &FolderBlock{}
		blockOffset := int64(sb.S_block_start + blockIndex*sb.S_block_size)

		if err := block.Decode(file, blockOffset); err != nil {
			return fmt.Errorf("error deserializando bloque %d: %w", blockIndex, err)
		}

		for i, content := range block.B_content {
			contentName := strings.Trim(string(content.B_name[:]), "\x00 ")

			if content.B_inodo != -1 && strings.EqualFold(contentName, folderName) {
				folderInode := &Inodo{}
				if err := folderInode.Decode(file, int64(sb.S_inode_start+content.B_inodo*sb.S_inode_size)); err != nil {
					return fmt.Errorf("error deserializando inodo %d: %w", content.B_inodo, err)
				}

				if folderInode.I_type[0] != '0' {
					return fmt.Errorf("'%s' no es un directorio", folderName)
				}

				if err := sb.deleteFolderInInode(file, content.B_inodo, fullPath); err != nil {
					return fmt.Errorf("error eliminando carpeta '%s': %w", folderName, err)
				}

				block.B_content[i] = FolderContent{
					B_name:  [12]byte{'-'},
					B_inodo: -1,
				}

				if err := block.Encode(file, blockOffset); err != nil {
					return fmt.Errorf("error actualizando bloque de directorio: %w", err)
				}

				fmt.Printf("Carpeta '%s' eliminada correctamente\n", folderName)
				return nil
			}
		}
	}

	return fmt.Errorf("carpeta '%s' no encontrada en el directorio", folderName)
}

func (sb *Superbloque) DeleteFolder(file *os.File, parentsDir []string, folderName string) error {
	fmt.Printf("Intentando eliminar carpeta '%s'\n", folderName)

	var fullPath string
	if len(parentsDir) > 0 {
		fullPath = "/" + strings.Join(parentsDir, "/") + "/" + folderName
	} else {
		fullPath = "/" + folderName
	}
	fmt.Printf("Ruta completa: %s\n", fullPath)

	if len(parentsDir) == 0 {
		return sb.deleteFolderFromDirectory(file, 0, folderName, fullPath)
	}

	currentInodeIndex := int32(0)
	for _, dirName := range parentsDir {
		found := false

		currentInode := &Inodo{}
		if err := currentInode.Decode(file, int64(sb.S_inode_start+currentInodeIndex*sb.S_inode_size)); err != nil {
			return fmt.Errorf("error cargando directorio actual (inodo %d): %w", currentInodeIndex, err)
		}

		if currentInode.I_type[0] != '0' {
			return fmt.Errorf("el inodo %d no es un directorio", currentInodeIndex)
		}

		blockIndexes, err := currentInode.GetDataBlockIndexes(file, sb)
		if err != nil {
			return fmt.Errorf("error obteniendo bloques de directorio: %w", err)
		}

		for _, blockIndex := range blockIndexes {
			if found {
				break
			}

			block := &FolderBlock{}
			if err := block.Decode(file, int64(sb.S_block_start+blockIndex*sb.S_block_size)); err != nil {
				return fmt.Errorf("error deserializando bloque %d: %w", blockIndex, err)
			}

			for _, content := range block.B_content {
				contentName := strings.Trim(string(content.B_name[:]), "\x00 ")

				if content.B_inodo != -1 && strings.EqualFold(contentName, dirName) {
					subDirInode := &Inodo{}
					if err := subDirInode.Decode(file, int64(sb.S_inode_start+content.B_inodo*sb.S_inode_size)); err != nil {
						return fmt.Errorf("error cargando inodo %d: %w", content.B_inodo, err)
					}

					if subDirInode.I_type[0] != '0' {
						return fmt.Errorf("la entrada '%s' no es un directorio", dirName)
					}

					currentInodeIndex = content.B_inodo
					found = true
					break
				}
			}
		}

		if !found {
			return fmt.Errorf("no se encontró el directorio '%s' en la ruta", dirName)
		}
	}

	return sb.deleteFolderFromDirectory(file, currentInodeIndex, folderName, fullPath)
}
