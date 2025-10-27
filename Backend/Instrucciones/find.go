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

type FIND struct {
	path string
	name string
}

func AnalizarFind(tokens []string) (string, error) {
	cmd := &FIND{}
	var outputBuffer bytes.Buffer

	re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-name="[^"]+"|-name=[^\s]+`)
	matches := re.FindAllString(strings.Join(tokens, " "), -1)

	if len(matches) != len(tokens) || len(matches) < 2 {
		return "", errors.New("faltan parámetros requeridos: -path o -name")
	}

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		key := strings.ToLower(kv[0])
		value := strings.Trim(kv[1], "\"")
		switch key {
		case "-path":
			cmd.path = value
		case "-name":
			cmd.name = value
		}
	}

	if cmd.path == "" || cmd.name == "" {
		return "", errors.New("los parámetros -path y -name son obligatorios")
	}

	err := commandFind(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandFind(findCmd *FIND, outputBuffer *bytes.Buffer) error {
	fmt.Fprint(outputBuffer, "======================= FIND =======================\n")

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

	var rootInodeIndex int32
	if findCmd.path == "/" {
		rootInodeIndex = 0
	} else {
		parentDirs, dirName := utilidades.ObtenerDirectoriosPadre(findCmd.path)
		rootInodeIndex, err = findFileInode(file, partitionSuperblock, parentDirs, dirName)
		if err != nil {
			return fmt.Errorf("error al encontrar el directorio inicial: %v", err)
		}
	}

	pattern, err := wildcardToRegex(findCmd.name)
	if err != nil {
		return fmt.Errorf("error al convertir el patrón de búsqueda: %v", err)
	}

	err = searchRecursive(file, partitionSuperblock, rootInodeIndex, pattern, findCmd.path, outputBuffer)
	if err != nil {
		return fmt.Errorf("error durante la búsqueda: %v", err)
	}

	fmt.Fprint(outputBuffer, "=================================================\n")
	return nil
}

func searchRecursive(file *os.File, sb *estructuras.Superbloque, inodeIndex int32, pattern *regexp.Regexp, currentPath string, outputBuffer *bytes.Buffer) error {
	inode := &estructuras.Inodo{}
	err := inode.Decode(file, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar el inodo %d: %v", inodeIndex, err)
	}

	if inode.I_type[0] != '0' {
		return nil
	}

	for _, blockIndex := range inode.I_block {
		if blockIndex == -1 {
			break
		}

		block := &estructuras.FolderBlock{}
		err := block.Decode(file, int64(sb.S_block_start+(blockIndex*sb.S_block_size)))
		if err != nil {
			return fmt.Errorf("error al deserializar el bloque %d: %v", blockIndex, err)
		}

		for _, content := range block.B_content {
			if content.B_inodo == -1 {
				continue
			}

			contentName := strings.Trim(string(content.B_name[:]), "\x00 ")
			if contentName == "." || contentName == ".." {
				continue
			}
			if pattern.MatchString(contentName) {
				fmt.Fprintf(outputBuffer, "%s/%s\n", currentPath, contentName)
			}
			newInodeIndex := content.B_inodo
			err = searchRecursive(file, sb, newInodeIndex, pattern, currentPath+"/"+contentName, outputBuffer)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func wildcardToRegex(pattern string) (*regexp.Regexp, error) {
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "?", ".")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	return regexp.Compile("^" + pattern + "$")
}
