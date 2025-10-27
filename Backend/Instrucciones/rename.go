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

type RENAME struct {
	path string
	name string
}

func AnalizarRename(tokens []string) (string, error) {
	cmd := &RENAME{}
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

	err := commandRename(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandRename(renameCmd *RENAME, outputBuffer *bytes.Buffer) error {
	fmt.Fprint(outputBuffer, "======================= RENAME =======================\n")

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

	parentDirs, oldName := utilidades.ObtenerDirectoriosPadre(renameCmd.path)

	inodeIndex, err := findFolderInode(file, partitionSuperblock, parentDirs)
	if err != nil {
		return fmt.Errorf("error al encontrar el directorio padre: %v", err)
	}
	folderBlock := &estructuras.FolderBlock{}
	err = folderBlock.Decode(file, int64(partitionSuperblock.S_block_start+(inodeIndex*partitionSuperblock.S_block_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar el bloque de carpeta: %v", err)
	}

	for _, content := range folderBlock.B_content {
		if strings.EqualFold(strings.Trim(string(content.B_name[:]), "\x00 "), renameCmd.name) {
			return fmt.Errorf("ya existe un archivo o carpeta con el nombre '%s'", renameCmd.name)
		}
	}

	err = folderBlock.RenameInFolderBlock(oldName, renameCmd.name)
	if err != nil {
		return fmt.Errorf("error al renombrar el archivo o carpeta: %v", err)
	}

	err = folderBlock.Encode(file, int64(partitionSuperblock.S_block_start+(inodeIndex*partitionSuperblock.S_block_size)))
	if err != nil {
		return fmt.Errorf("error al guardar el bloque de carpeta modificado: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Nombre cambiado exitosamente de '%s' a '%s'\n", oldName, renameCmd.name)
	fmt.Fprint(outputBuffer, "=====================================================\n")

	return nil
}
