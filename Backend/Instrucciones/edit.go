package instrucciones

import (
	"bytes"
	"errors"
	"fmt"
	estructuras "godisk/Estructuras"
	global "godisk/Global"
	utilidades "godisk/Utilidades"
	"os"
	"regexp"
	"strings"
)

type EDIT struct {
	path      string
	contenido string
}

func AnalizarEdit(tokens []string) (string, error) {
	cmd := &EDIT{}
	var outputBuffer bytes.Buffer

	re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-contenido="[^"]+"|-contenido=[^\s]+`)
	matches := re.FindAllString(strings.Join(tokens, " "), -1)

	if len(matches) != len(tokens) || len(matches) < 2 {
		return "", errors.New("faltan parámetros requeridos: -path o -contenido")
	}

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		key := strings.ToLower(kv[0])
		value := strings.Trim(kv[1], "\"")

		switch key {
		case "-path":
			cmd.path = value
		case "-contenido":
			cmd.contenido = value
		}
	}

	if cmd.path == "" || cmd.contenido == "" {
		return "", errors.New("los parámetros -path y -contenido son obligatorios")
	}

	err := commandEdit(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandEdit(editCmd *EDIT, outputBuffer *bytes.Buffer) error {
	fmt.Fprint(outputBuffer, "======================= EDIT =======================\n")

	if !global.EstaLogueado() {
		return fmt.Errorf("no hay un usuario logueado")
	}

	idPartition := global.UsuarioActual.Id

	partitionSuperblock, _, partitionPath, err := global.GetMountedPartitionSuperblock(idPartition)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de partición: %w", err)
	}
	defer file.Close()

	parentDirs, fileName := utilidades.ObtenerDirectoriosPadre(editCmd.path)

	inodeIndex, err := findFileInode(file, partitionSuperblock, parentDirs, fileName)
	if err != nil {
		return fmt.Errorf("error al encontrar el archivo: %v", err)
	}

	newContent, err := os.ReadFile(editCmd.contenido)
	if err != nil {
		return fmt.Errorf("error al leer el archivo de contenido '%s': %v", editCmd.contenido, err)
	}

	err = editFileContent(file, partitionSuperblock, inodeIndex, newContent)
	if err != nil {
		return fmt.Errorf("error al editar el contenido del archivo: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Contenido del archivo '%s' editado exitosamente\n", fileName)
	fmt.Fprint(outputBuffer, "=================================================\n")

	return nil
}

func editFileContent(file *os.File, sb *estructuras.Superbloque, inodeIndex int32, newContent []byte) error {
	inode := &estructuras.Inodo{}
	err := inode.Decode(file, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar el inodo %d: %v", inodeIndex, err)
	}

	if inode.I_type[0] != '1' {
		return fmt.Errorf("el inodo %d no corresponde a un archivo", inodeIndex)
	}

	for _, blockIndex := range inode.I_block {
		if blockIndex != -1 {
			fileBlock := &estructuras.ArchivoBloque{}
			fileBlock.ClearContent()
			err := fileBlock.Encode(file, int64(sb.S_block_start+(blockIndex*sb.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al limpiar el bloque %d: %v", blockIndex, err)
			}
		}
	}

	blocks, err := estructuras.SplitContent(string(newContent))
	if err != nil {
		return fmt.Errorf("error al dividir el contenido en bloques: %v", err)
	}

	blockCount := len(blocks)
	for i := 0; i < blockCount; i++ {
		if i < len(inode.I_block) {
			blockIndex := inode.I_block[i]
			if blockIndex == -1 {
				blockIndex, err = sb.AssignNewBlock(file, inode, i)
				if err != nil {
					return fmt.Errorf("error asignando un nuevo bloque: %v", err)
				}
			}

			err := blocks[i].Encode(file, int64(sb.S_block_start+(blockIndex*sb.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al escribir el bloque %d: %v", blockIndex, err)
			}
		} else {
			pointerBlockIndex := inode.I_block[len(inode.I_block)-1]
			if pointerBlockIndex == -1 {
				pointerBlockIndex, err = sb.AssignNewBlock(file, inode, len(inode.I_block)-1)
				if err != nil {
					return fmt.Errorf("error asignando un nuevo bloque de apuntadores: %v", err)
				}
			}

			pointerBlock := &estructuras.PointerBlock{}
			err := pointerBlock.Decode(file, int64(sb.S_block_start+(pointerBlockIndex*sb.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al decodificar el bloque de apuntadores: %v", err)
			}

			freeIndex, err := pointerBlock.FindFreePointer()
			if err != nil {
				return fmt.Errorf("no hay apuntadores libres en el bloque de apuntadores: %v", err)
			}

			newBlockIndex, err := sb.AssignNewBlock(file, inode, freeIndex)
			if err != nil {
				return fmt.Errorf("error asignando un nuevo bloque: %v", err)
			}

			err = pointerBlock.SetPointer(freeIndex, int64(newBlockIndex))
			if err != nil {
				return fmt.Errorf("error actualizando el bloque de apuntadores: %v", err)
			}

			err = pointerBlock.Encode(file, int64(sb.S_block_start+(pointerBlockIndex*sb.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al guardar el bloque de apuntadores: %v", err)
			}

			err = blocks[i].Encode(file, int64(sb.S_block_start+(newBlockIndex*sb.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al escribir el nuevo bloque %d: %v", newBlockIndex, err)
			}
		}
	}

	inode.I_size = int32(len(newContent))
	err = inode.Encode(file, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al actualizar el inodo %d: %v", inodeIndex, err)
	}

	return nil
}
