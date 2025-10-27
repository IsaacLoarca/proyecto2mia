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

type REMOVE struct {
	path string
}

func AnalizarRemove(tokens []string) (string, error) {
	cmd := &REMOVE{}
	var outputBuffer bytes.Buffer

	re := regexp.MustCompile(`-path=("[^"]+"|[^\s]+)`)
	matches := re.FindAllString(strings.Join(tokens, " "), -1)
	if len(matches) == 0 {
		return "", errors.New("no se especificó una ruta para eliminar")
	}

	kv := strings.SplitN(matches[0], "=", 2)
	if len(kv) == 2 {
		cmd.path = kv[1]
		if strings.HasPrefix(cmd.path, "\"") && strings.HasSuffix(cmd.path, "\"") {
			cmd.path = strings.Trim(cmd.path, "\"")
		}
	}

	err := commandRemove(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}
func commandRemove(removeCmd *REMOVE, outputBuffer *bytes.Buffer) error {
	fmt.Fprint(outputBuffer, "====================== REMOVE ======================\n")

	if !global.EstaLogueado() {
		return fmt.Errorf("no hay un usuario logueado")
	}

	idPartition := global.UsuarioActual.Id
	partitionSuperblock, mountedPartition, partitionPath, err := global.GetMountedPartitionSuperblock(idPartition)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de partición: %w", err)
	}
	defer file.Close()

	err = removeFileOrDirectory(removeCmd.path, partitionSuperblock, file)
	if err != nil {
		return fmt.Errorf("error al eliminar archivo o carpeta: %v", err)
	}

	err = partitionSuperblock.Codificar(file, int64(mountedPartition.Part_start))
	if err != nil {
		return fmt.Errorf("error al serializar el superbloque después de la eliminación: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Archivo o carpeta '%s' eliminado exitosamente.\n", removeCmd.path)
	fmt.Fprint(outputBuffer, "====================================================\n")
	return nil
}

func removeFileOrDirectory(path string, sb *estructuras.Superbloque, file *os.File) error {
	parentDirs, fileName := utilidades.ObtenerDirectoriosPadre(path)

	err := removeFile(sb, file, parentDirs, fileName)
	if err == nil {
		return nil
	}

	err = removeDirectory(sb, file, parentDirs, fileName)
	if err != nil {
		return fmt.Errorf("error al eliminar archivo o carpeta '%s': %v", path, err)
	}

	return nil
}

func removeFile(sb *estructuras.Superbloque, file *os.File, parentDirs []string, fileName string) error {
	_, err := findFileInode(file, sb, parentDirs, fileName)
	if err != nil {
		return fmt.Errorf("archivo '%s' no encontrado: %v", fileName, err)
	}

	err = sb.DeleteFile(file, parentDirs, fileName)
	if err != nil {
		return fmt.Errorf("error al eliminar el archivo '%s': %v", fileName, err)
	}

	fmt.Printf("Archivo '%s' eliminado correctamente.\n", fileName)
	return nil
}

func removeDirectory(sb *estructuras.Superbloque, file *os.File, parentDirs []string, dirName string) error {
	_, err := findFolderInode(file, sb, parentDirs)
	if err != nil {
		return fmt.Errorf("carpeta '%s' no encontrada: %v", dirName, err)
	}

	err = sb.DeleteFolder(file, parentDirs, dirName)
	if err != nil {
		return fmt.Errorf("error al eliminar la carpeta '%s': %v", dirName, err)
	}

	fmt.Printf("Carpeta '%s' eliminada correctamente.\n", dirName)
	return nil
}
