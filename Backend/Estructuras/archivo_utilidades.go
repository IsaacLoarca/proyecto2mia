package estructuras

import (
	"fmt"
	utilidades "godisk/Utilidades"
	"os"
	"strings"
	"time"
)

func (sb *Superbloque) crearArchivoEnInodo(archivo *os.File, indiceInodo int32, padresDir []string, destArchivo string, tamanioArchivo int, contenidoArchivo []string, r bool) error {
	fmt.Printf("Intentando crear archivo '%s' en inodo con índice %d\n", destArchivo, indiceInodo)
	inodo := &Inodo{}
	err := inodo.Decode(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}
	if inodo.I_type[0] == '1' {
		fmt.Printf("El inodo %d es una carpeta, omitiendo.\n", indiceInodo)
		return nil
	}
	seCreoArchivo := false
	for _, indiceBloque := range inodo.I_block {
		if indiceBloque == -1 {
			if !seCreoArchivo {
				fmt.Printf("Creando nuevo Bloque Carpeta para poder crear el archivo\n")
				err := sb.CrearNuevoBloqueCarpeta(archivo, int32(indiceInodo), padresDir, destArchivo, tamanioArchivo, contenidoArchivo, r, 1)
				if err != nil {
					fmt.Printf("Error con la creación de la carpeta que contendrá el archivo: %s", destArchivo)
					return err
				}
			}
			return nil
		}
		bloque := &FolderBlock{}
		err := bloque.Decode(archivo, int64(sb.S_block_start+(indiceBloque*sb.S_block_size)))
		if err != nil {
			return fmt.Errorf("error al deserializar bloque %d: %v", indiceBloque, err)
		}
		for indiceContenido := 2; indiceContenido < len(bloque.B_content); indiceContenido++ {
			contenido := bloque.B_content[indiceContenido]
			if contenido.B_inodo != -1 {
				fmt.Printf("El inodo %d ya está ocupado, continuando.\n", contenido.B_inodo)
				continue
			}
			copy(contenido.B_name[:], []byte(destArchivo))
			contenido.B_inodo = sb.S_inodes_count
			bloque.B_content[indiceContenido] = contenido
			err = bloque.Encode(archivo, int64(sb.S_block_start+(indiceBloque*sb.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al serializar bloque %d: %v", indiceBloque, err)
			}
			fmt.Printf("Bloque actualizado para el archivo '%s' en el inodo %d\n", destArchivo, sb.S_inodes_count)
			inodoArchivo := &Inodo{
				I_uid:   1,
				I_gid:   1,
				I_size:  int32(tamanioArchivo),
				I_atime: float32(time.Now().Unix()),
				I_ctime: float32(time.Now().Unix()),
				I_mtime: float32(time.Now().Unix()),
				I_block: [15]int32{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
				I_type:  [1]byte{'1'},
				I_perm:  [3]byte{'6', '6', '4'},
			}
			for i := 0; i < len(contenidoArchivo); i++ {
				inodoArchivo.I_block[i] = sb.S_blocks_count
				bloqueArchivo := &ArchivoBloque{
					B_content: [64]byte{},
				}
				copy(bloqueArchivo.B_content[:], contenidoArchivo[i])
				err = bloqueArchivo.Encode(archivo, int64(sb.S_first_blo))
				if err != nil {
					return fmt.Errorf("error al serializar bloque de archivo: %v", err)
				}
				fmt.Printf("Bloque de archivo '%s' serializado correctamente.\n", destArchivo)
				err = sb.UpdateBitmapBlock(archivo, sb.S_blocks_count, true)
				if err != nil {
					return fmt.Errorf("error al actualizar bitmap de bloque: %v", err)
				}
				sb.UpdateSuperblockAfterBlockAllocation()
			}
			err = inodoArchivo.Encode(archivo, int64(sb.S_first_ino))
			if err != nil {
				return fmt.Errorf("error al serializar inodo del archivo: %v", err)
			}
			fmt.Printf("Inodo del archivo '%s' serializado correctamente.\n", destArchivo)
			err = sb.UpdateBitmapInode(archivo, sb.S_inodes_count, true)
			if err != nil {
				return fmt.Errorf("error al actualizar bitmap de inodo: %v", err)
			}
			sb.UpdateSuperblockAfterInodeAllocation()
			fmt.Printf("Archivo '%s' creado correctamente en el inodo %d.\n", destArchivo, sb.S_inodes_count)
			return nil
		}
	}
	fmt.Println("Ya no hubo espacio en el inodo para crear el archivo")
	return nil
}

func (sb *Superbloque) CrearArchivo(archivo *os.File, padresDir []string, destCarpeta string, tamanio int, contenido []string, r bool) error {
	fmt.Printf("Creando archivo '%s' con tamaño %d\n", destCarpeta, tamanio)
	var padreEncontrado bool
	var indiceInodoEncontrado int32
	for i := int32(0); i < sb.S_inodes_count; i++ {
		indiceInodo, encontrado, err := sb.ValidarPadreDirectorio(archivo, i, padresDir, destCarpeta, r)
		if encontrado {
			padreEncontrado = true
			indiceInodoEncontrado = int32(indiceInodo)
			break
		}
		if err != nil {
			return err
		}
	}
	partes := utilidades.DefinirCarpetaArchivo(destCarpeta)
	if len(partes) == 1 {
		fmt.Printf("Se está validando una carpeta de nombre: %s\n", destCarpeta)
		return nil
	}
	if padreEncontrado {
		err := sb.crearArchivoEnInodo(archivo, int32(indiceInodoEncontrado), padresDir, destCarpeta, tamanio, contenido, r)
		if err != nil {
			return err
		}
	} else if len(padresDir) == 0 {
		err := sb.crearArchivoEnInodo(archivo, 0, padresDir, destCarpeta, tamanio, contenido, r)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("no se encontró la carpeta padre de: %s\n", destCarpeta)
		return fmt.Errorf("no se encontró la carpeta padre de: %s", destCarpeta)
	}
	return nil
}

func (sb *Superbloque) CrearNuevoBloqueCarpeta(archivo *os.File, indiceInodo int32, padresDir []string, destArchivo string, tamanioArchivo int, contenidoArchivo []string, r bool, tipoInodo int) error {
	inodo := &Inodo{}
	err := inodo.Decode(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}
	if inodo.I_type[0] == '1' {
		fmt.Printf("El inodo %d es una carpeta, omitiendo.\n", indiceInodo)
		return nil
	}
	for i, indiceBloque := range inodo.I_block {
		if indiceBloque == -1 {
			inodo.I_block[i] = sb.S_blocks_count
			err = inodo.Encode(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
			if err != nil {
				return fmt.Errorf("error al serializar inodo del archivo: %v", err)
			}
			fmt.Printf("Inodo del archivo '%s' serializado correctamente.\n", destArchivo)
			err = sb.UpdateBitmapInode(archivo, sb.S_inodes_count, true)
			if err != nil {
				return fmt.Errorf("error al actualizar bitmap de inodo: %v", err)
			}
			sb.UpdateSuperblockAfterInodeAllocation()
			indiceInodoPadre, _, err := sb.ValidarPadreDirectorio(archivo, indiceInodo, padresDir, destArchivo, true)
			if err != nil {
				fmt.Printf("Error al conseguir el id del padre para crear la carpeta que contendrá el archivo: %s", destArchivo)
				return err
			}
			bloqueCarpeta := &FolderBlock{
				B_content: [4]FolderContent{
					{B_name: [12]byte{'.'}, B_inodo: int32(indiceInodoPadre)},
					{B_name: [12]byte{'.', '.'}, B_inodo: indiceInodo},
					{B_name: [12]byte{'-'}, B_inodo: -1},
					{B_name: [12]byte{'-'}, B_inodo: -1},
				},
			}
			fmt.Printf("Serializando el bloque de la carpeta que va a contener el archivo '%s'\n", destArchivo)
			err = bloqueCarpeta.Encode(archivo, int64(sb.S_first_blo))
			if err != nil {
				return fmt.Errorf("error al serializar el bloque del directorio '%s': %v", destArchivo, err)
			}
			err = sb.UpdateBitmapBlock(archivo, sb.S_blocks_count, true)
			if err != nil {
				return fmt.Errorf("error al actualizar el bitmap de bloques para el directorio '%s': %v", destArchivo, err)
			}
			sb.UpdateSuperblockAfterBlockAllocation()
			if tipoInodo == 0 {
				err = sb.CrearCarpetaEnInodo(archivo, int32(indiceInodo), padresDir, destArchivo, r)
				if err != nil {
					return err
				}
			} else if tipoInodo == 1 {
				err = sb.crearArchivoEnInodo(archivo, int32(indiceInodo), padresDir, destArchivo, tamanioArchivo, contenidoArchivo, r)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
	return nil
}
func (sb *Superbloque) deleteFileInInode(file *os.File, inodeIndex int32, fileName string, parentPath ...string) error {
	dirInode := &Inodo{}
	err := dirInode.Decode(file, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar inodo %d: %w", inodeIndex, err)
	}

	if dirInode.I_type[0] != '0' {
		return fmt.Errorf("el inodo %d no es una carpeta", inodeIndex)
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

		for i, content := range block.B_content {
			contentName := strings.Trim(string(content.B_name[:]), "\x00 ")

			if content.B_inodo != -1 && strings.EqualFold(contentName, fileName) {
				fileInodeIndex := content.B_inodo
				fmt.Printf("Archivo '%s' encontrado en inodo %d, eliminando.\n", fileName, fileInodeIndex)

				fileInode := &Inodo{}
				fileInodeOffset := int64(sb.S_inode_start + (fileInodeIndex * sb.S_inode_size))
				if err := fileInode.Decode(file, fileInodeOffset); err != nil {
					return fmt.Errorf("error deserializando inodo del archivo %d: %w", fileInodeIndex, err)
				}

				if fileInode.I_type[0] != '1' {
					return fmt.Errorf("el inodo %d no es un archivo sino de tipo %c", fileInodeIndex, fileInode.I_type[0])
				}

				if sb.S_filesystem_type == 3 {
					var fullPath string
					if len(parentPath) > 0 && parentPath[0] != "" {
						fullPath = parentPath[0] + "/" + fileName
					} else {
						fullPath = "/" + fileName
					}

					journaling_start := int64(sb.JournalStart())

					fileData, err := fileInode.ReadData(file, sb)
					fileContent := ""
					if err != nil {
						fmt.Printf("Error leyendo contenido del archivo para journal: %v\n", err)
					} else {
						fileContent = string(fileData)
					}

					if err := AddJournalEntry(
						file,
						journaling_start,
						JOURNAL_ENTRIES,
						"rm",
						fullPath,
						fileContent,
						sb,
					); err != nil {
						fmt.Printf("Advertencia: error registrando operación en journal: %v\n", err)
					} else {
						fmt.Printf("Operación 'rm %s' registrada en journal correctamente\n", fullPath)
					}
				}

				if err := fileInode.FreeAllBlocks(file, sb); err != nil {
					return fmt.Errorf("error liberando bloques del archivo: %w", err)
				}

				if err := sb.UpdateBitmapInode(file, fileInodeIndex, false); err != nil {
					return fmt.Errorf("error liberando inodo %d: %w", fileInodeIndex, err)
				}
				sb.UpdateSuperblockAfterInodeDeallocation()

				block.B_content[i] = FolderContent{B_name: [12]byte{'-'}, B_inodo: -1}
				if err := block.Encode(file, blockOffset); err != nil {
					return fmt.Errorf("error actualizando bloque de directorio: %w", err)
				}

				if err := dirInode.CheckAndFreeEmptyIndirectBlocks(file, sb); err != nil {
					fmt.Printf("Advertencia: error al verificar bloques indirectos vacíos: %v\n", err)
				}

				fmt.Printf("Archivo '%s' eliminado correctamente.\n", fileName)
				return nil
			}
		}
	}

	return fmt.Errorf("archivo '%s' no encontrado en directorio (inodo %d)", fileName, inodeIndex)
}

func (sb *Superbloque) DeleteFile(file *os.File, parentsDir []string, fileName string) error {
	fmt.Printf("Intentando eliminar archivo '%s'\n", fileName)

	if len(parentsDir) == 0 {
		return sb.deleteFileInInode(file, 0, fileName)
	}

	currentInodeIndex := int32(0)
	for _, dirName := range parentsDir {
		found := false

		currentInode := &Inodo{}
		if err := currentInode.Decode(file, int64(sb.S_inode_start+currentInodeIndex*sb.S_inode_size)); err != nil {
			return fmt.Errorf("error cargando directorio actual (inodo %d): %w", currentInodeIndex, err)
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

	return sb.deleteFileInInode(file, currentInodeIndex, fileName)
}
