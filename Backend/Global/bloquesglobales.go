package global

import (
	"fmt"
	estructuras "godisk/Estructuras"
	"os"
	"strings"
)

func ReadFileBlocks(file *os.File, sb *estructuras.Superbloque, inode *estructuras.Inodo) (string, error) {
	var contenido string

	for _, blockIndex := range inode.I_block {
		if blockIndex == -1 {
			break
		}

		blockOffset := int64(sb.S_block_start + blockIndex*int32(sb.S_block_size))
		var fileBlock estructuras.ArchivoBloque

		err := fileBlock.Decode(file, blockOffset)
		if err != nil {
			return "", fmt.Errorf("error leyendo bloque %d: %w", blockIndex, err)
		}

		contenido += string(fileBlock.B_content[:])
	}

	inode.ActualizarAtime()

	return strings.TrimRight(contenido, "\x00"), nil
}

func WriteUsersBlocks(file *os.File, sb *estructuras.Superbloque, inode *estructuras.Inodo, nuevoContenido string) error {
	contenidoExistente, err := ReadFileBlocks(file, sb, inode)
	if err != nil {
		return fmt.Errorf("error leyendo contenido existente de users.txt: %w", err)
	}

	contenidoTotal := contenidoExistente + nuevoContenido

	blocks, err := estructuras.SplitContent(contenidoTotal)
	if err != nil {
		return fmt.Errorf("error al dividir el contenido en bloques: %w", err)
	}

	index := 0

	for _, block := range blocks {
		if index >= len(inode.I_block) {
			return fmt.Errorf("se alcanzó el límite máximo de bloques del inodo")
		}

		if inode.I_block[index] == -1 {
			newBlockIndex, err := sb.AssignNewBlock(file, inode, index)
			if err != nil {
				return fmt.Errorf("error asignando nuevo bloque: %w", err)
			}
			inode.I_block[index] = newBlockIndex
		}

		blockOffset := int64(sb.S_block_start + inode.I_block[index]*int32(sb.S_block_size))

		err = block.Encode(file, blockOffset)
		if err != nil {
			return fmt.Errorf("error escribiendo el bloque %d: %w", inode.I_block[index], err)
		}

		index++
	}

	nuevoTamano := len(contenidoTotal)
	inode.I_size = int32(nuevoTamano)

	inode.ActualizarMtime()
	inode.ActualizarCtime()

	return nil
}

func InsertIntoUsersFile(file *os.File, sb *estructuras.Superbloque, inode *estructuras.Inodo, entry string) error {
	contenidoActual, err := ReadFileBlocks(file, sb, inode)
	if err != nil {
		return fmt.Errorf("error leyendo el contenido de users.txt: %w", err)
	}

	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")

	partesEntry := strings.Split(entry, ",")
	if len(partesEntry) < 4 {
		return fmt.Errorf("entrada de usuario inválida: %s", entry)
	}
	userGrupo := partesEntry[2]

	var groupID string
	var nuevoContenido []string
	usuarioInsertado := false

	for _, linea := range lineas {
		partes := strings.Split(linea, ",")
		nuevoContenido = append(nuevoContenido, strings.TrimSpace(linea))

		if len(partes) > 2 && partes[1] == "G" && partes[2] == userGrupo {
			groupID = partes[0]
			if groupID != "" && !usuarioInsertado {
				usuarioConGrupo := fmt.Sprintf("%s,U,%s,%s,%s", groupID, partesEntry[2], partesEntry[3], partesEntry[4])
				nuevoContenido = append(nuevoContenido, usuarioConGrupo)
				usuarioInsertado = true
			}
		}
	}

	if groupID == "" {
		return fmt.Errorf("el grupo '%s' no existe", userGrupo)
	}

	contenidoNuevo := strings.Join(nuevoContenido, "\n") + "\n"
	fmt.Println("=== Escribiendo nuevo contenido en users.txt ===")
	fmt.Println(contenidoNuevo)

	for _, blockIndex := range inode.I_block {
		if blockIndex == -1 {
			break
		}

		blockOffset := int64(sb.S_block_start + blockIndex*sb.S_block_size)
		var fileBlock estructuras.ArchivoBloque

		fileBlock.ClearContent()

		err = fileBlock.Encode(file, blockOffset)
		if err != nil {
			return fmt.Errorf("error escribiendo bloque limpio %d: %w", blockIndex, err)
		}
	}

	err = WriteUsersBlocks(file, sb, inode, contenidoNuevo)
	if err != nil {
		return fmt.Errorf("error escribiendo el nuevo contenido en users.txt: %w", err)
	}

	inode.I_size = int32(len(contenidoNuevo))

	inode.ActualizarMtime()
	inode.ActualizarCtime()

	return nil
}

func AddEntryToUsersFile(file *os.File, sb *estructuras.Superbloque, inode *estructuras.Inodo, entry, name, entityType string) error {
	contenidoActual, err := ReadFileBlocks(file, sb, inode)
	if err != nil {
		return fmt.Errorf("error leyendo blocks de users.txt: %w", err)
	}

	_, _, err = findLineInUsersFile(contenidoActual, name, entityType)
	if err == nil {
		fmt.Printf("El %s '%s' ya existe en users.txt\n", entityType, name)
		return nil
	}
	fmt.Println("=== Escribiendo nuevo contenido en users.txt ===")
	fmt.Println(entry)

	err = WriteUsersBlocks(file, sb, inode, entry+"\n")
	if err != nil {
		return fmt.Errorf("error agregando entrada a users.txt: %w", err)
	}

	fmt.Println("\n=== Estado del inodo después de la modificación ===")
	sb.PrintInodes(file.Name())

	fmt.Println("\n=== Estado de los bloques después de la modificación ===")
	sb.PrintBlocks(file.Name())

	return nil
}

func CreateGroup(file *os.File, sb *estructuras.Superbloque, inode *estructuras.Inodo, groupName string) error {
	groupEntry := fmt.Sprintf("%d,G,%s", sb.S_inodes_count+1, groupName)
	return AddEntryToUsersFile(file, sb, inode, groupEntry, groupName, "G")
}

func CreateUser(file *os.File, sb *estructuras.Superbloque, inode *estructuras.Inodo, userName, userPassword, groupName string) error {
	userEntry := fmt.Sprintf("%d,U,%s,%s,%s", sb.S_inodes_count+1, userName, groupName, userPassword)
	return AddEntryToUsersFile(file, sb, inode, userEntry, userName, "U")
}

func FindInUsersFile(file *os.File, sb *estructuras.Superbloque, inode *estructuras.Inodo, name, entityType string) (string, error) {
	contenido, err := ReadFileBlocks(file, sb, inode)
	if err != nil {
		return "", err
	}

	linea, _, err := findLineInUsersFile(contenido, name, entityType)
	if err != nil {
		return "", err
	}

	return linea, nil
}

func findLineInUsersFile(contenido string, name, entityType string) (string, int, error) {
	lineas := strings.Split(contenido, "\n")

	for i, linea := range lineas {
		campos := strings.Split(linea, ",")
		if len(campos) < 3 {
			continue
		}

		if entityType == "G" && len(campos) == 3 {
			grupo := estructuras.NewGroup(campos[0], campos[2])
			if grupo.Tipo == entityType && grupo.Group == name {
				return grupo.ToString(), i, nil
			}
		} else if entityType == "U" && len(campos) == 5 {
			usuario := estructuras.NewUser(campos[0], campos[2], campos[3], campos[4])
			if usuario.Tipo == entityType && usuario.Name == name {
				return usuario.ToString(), i, nil
			}
		}
	}

	return "", -1, fmt.Errorf("%s '%s' no encontrado en users.txt", entityType, name)
}
