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

type CAT struct {
	files []string
}

func AnalizarCat(tokens []string) (string, error) {
	cmd := &CAT{}
	var outputBuffer bytes.Buffer

	re := regexp.MustCompile(`-file\d+=("[^"]+"|[^\s]+)`)
	matches := re.FindAllString(strings.Join(tokens, " "), -1)

	if len(matches) == 0 {
		return "", errors.New("no se especificaron archivos para leer")
	}

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) == 2 {
			filePath := kv[1]
			if strings.HasPrefix(filePath, "\"") && strings.HasSuffix(filePath, "\"") {
				filePath = strings.Trim(filePath, "\"")
			}
			cmd.files = append(cmd.files, filePath)
		}
	}

	err := commandCat(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandCat(cat *CAT, outputBuffer *bytes.Buffer) error {
	fmt.Fprint(outputBuffer, "======================= CAT =======================\n")
	if !global.EstaLogueado() {
		return fmt.Errorf("no hay un usuario logueado")
	}

	idPartition := global.UsuarioActual.Id

	_, _, partitionPath, err := global.GetMountedPartitionSuperblock(idPartition)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de partición: %w", err)
	}
	defer file.Close()

	for _, filePath := range cat.files {
		fmt.Fprintf(outputBuffer, "Leyendo archivo: %s\n", filePath)

		content, err := readFileContent(filePath)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error al leer el archivo %s: %v\n", filePath, err)
			continue
		}

		outputBuffer.WriteString(content)
		outputBuffer.WriteString("\n")
		fmt.Fprint(outputBuffer, "=================== FIN CAT ==================\n")
	}

	return nil
}

func readFileContent(filePath string) (string, error) {
	idPartition := global.UsuarioActual.Id
	partitionSuperblock, _, partitionPath, err := global.GetMountedPartitionSuperblock(idPartition)
	if err != nil {
		return "", fmt.Errorf("error al obtener la partición montada: %v", err)
	}

	file, err := os.OpenFile(partitionPath, os.O_RDONLY, 0666)
	if err != nil {
		return "", fmt.Errorf("error al abrir el archivo de partición: %v", err)
	}
	defer file.Close()

	parentDirs, fileName := utilidades.ObtenerDirectoriosPadre(filePath)

	inodeIndex, err := findFileInode(file, partitionSuperblock, parentDirs, fileName)
	if err != nil {
		return "", fmt.Errorf("error al encontrar el archivo: %v", err)
	}

	content, err := readFileFromInode(file, partitionSuperblock, inodeIndex)
	if err != nil {
		return "", fmt.Errorf("error al leer el contenido del archivo: %v", err)
	}

	return content, nil
}

func directoryExists(sb *estructuras.Superbloque, file *os.File, inodeIndex int32, dirName string) (bool, int32, error) {
	fmt.Printf("Verificando si el directorio o archivo '%s' existe en el inodo %d\n", dirName, inodeIndex)

	inode := &estructuras.Inodo{}
	err := inode.Decode(file, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return false, -1, fmt.Errorf("error al deserializar inodo %d: %v", inodeIndex, err)
	}

	if inode.I_type[0] != '0' {
		return false, -1, fmt.Errorf("el inodo %d no es una carpeta", inodeIndex)
	}

	for _, blockIndex := range inode.I_block {
		if blockIndex == -1 {
			break
		}

		block := &estructuras.FolderBlock{}
		err := block.Decode(file, int64(sb.S_block_start+(blockIndex*sb.S_block_size)))
		if err != nil {
			return false, -1, fmt.Errorf("error al deserializar bloque %d: %v", blockIndex, err)
		}

		for _, content := range block.B_content {
			contentName := strings.Trim(string(content.B_name[:]), "\x00 ")
			if strings.EqualFold(contentName, dirName) && content.B_inodo != -1 {
				fmt.Printf("Directorio o archivo '%s' encontrado en inodo %d\n", dirName, content.B_inodo)
				return true, content.B_inodo, nil
			}
		}
	}

	fmt.Printf("Directorio o archivo '%s' no encontrado en inodo %d\n", dirName, inodeIndex)
	return false, -1, nil
}

func findFileInode(file *os.File, sb *estructuras.Superbloque, parentsDir []string, fileName string) (int32, error) {
	inodeIndex := int32(0)

	for len(parentsDir) > 0 {
		dirName := parentsDir[0]
		found, newInodeIndex, err := directoryExists(sb, file, inodeIndex, dirName)
		if err != nil {
			return -1, err
		}
		if !found {
			return -1, fmt.Errorf("directorio '%s' no encontrado", dirName)
		}
		inodeIndex = newInodeIndex
		parentsDir = parentsDir[1:]
	}

	found, fileInodeIndex, err := directoryExists(sb, file, inodeIndex, fileName)
	if err != nil {
		return -1, err
	}
	if !found {
		return -1, fmt.Errorf("archivo '%s' no encontrado", fileName)
	}

	return fileInodeIndex, nil
}

func readFileFromInode(file *os.File, sb *estructuras.Superbloque, inodeIndex int32) (string, error) {
	inode := &estructuras.Inodo{}
	err := inode.Decode(file, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return "", fmt.Errorf("error al deserializar el inodo %d: %v", inodeIndex, err)
	}

	if inode.I_type[0] != '1' {
		return "", fmt.Errorf("el inodo %d no corresponde a un archivo", inodeIndex)
	}

	var contentBuilder strings.Builder
	for _, blockIndex := range inode.I_block {
		if blockIndex == -1 {
			break
		}

		fileBlock := &estructuras.ArchivoBloque{}
		err := fileBlock.Decode(file, int64(sb.S_block_start+(blockIndex*sb.S_block_size)))
		if err != nil {
			return "", fmt.Errorf("error al deserializar el bloque %d: %v", blockIndex, err)
		}

		contentBuilder.WriteString(string(fileBlock.B_content[:]))
	}

	return contentBuilder.String(), nil
}

func findFolderInode(file *os.File, sb *estructuras.Superbloque, parentsDir []string) (int32, error) {
	inodeIndex := int32(0)

	for len(parentsDir) > 0 {
		dirName := parentsDir[0]

		found, newInodeIndex, err := directoryExists(sb, file, inodeIndex, dirName)
		if err != nil {
			return -1, err
		}
		if !found {
			return -1, fmt.Errorf("directorio '%s' no encontrado", dirName)
		}

		inodeIndex = newInodeIndex
		parentsDir = parentsDir[1:]
	}

	return inodeIndex, nil
}
