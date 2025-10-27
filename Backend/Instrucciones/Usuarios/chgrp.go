package instrucciones

import (
	"encoding/binary"
	"fmt"
	estructuras "godisk/Estructuras"
	globals "godisk/Global"
	"os"
	"regexp"
	"strings"
)

type CHGRP struct {
	User string
	Grp  string
}

func AnalizarChgrp(tokens []string) (string, error) {
	var outputBuffer strings.Builder
	cmd := &CHGRP{}

	reUser := regexp.MustCompile(`-usr=[^\s]+`)
	reGrp := regexp.MustCompile(`-grp=[^\s]+`)

	matchesUser := reUser.FindString(strings.Join(tokens, " "))
	matchesGrp := reGrp.FindString(strings.Join(tokens, " "))

	if matchesUser == "" {
		return "", fmt.Errorf("falta el parámetro -usr")
	}
	if matchesGrp == "" {
		return "", fmt.Errorf("falta el parámetro -grp")
	}

	cmd.User = strings.SplitN(matchesUser, "=", 2)[1]
	cmd.Grp = strings.SplitN(matchesGrp, "=", 2)[1]

	err := commandChgrp(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandChgrp(chgrp *CHGRP, outputBuffer *strings.Builder) error {
	fmt.Fprintln(outputBuffer, "======================= CHGRP =======================")
	if !globals.EstaLogueado() {
		return fmt.Errorf("no hay ninguna sesión activa")
	}
	if globals.UsuarioActual.Name != "root" {
		return fmt.Errorf("solo el usuario root puede ejecutar este comando")
	}

	partition, path, err := globals.ObtenerParticionMontada(globals.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se puede encontrar la partición montada: %v", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0755)
	if err != nil {
		return fmt.Errorf("no se puede abrir el archivo de la partición: %v", err)
	}
	defer file.Close()

	_, sb, _, err := globals.GetMountedPartitionRep(globals.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo cargar el Superblock: %v", err)
	}

	var usersInode estructuras.Inodo
	inodeOffset := int64(sb.S_inode_start + int32(binary.Size(usersInode)))
	err = usersInode.Decode(file, inodeOffset)
	if err != nil {
		return fmt.Errorf("error leyendo el inodo de users.txt: %v", err)
	}

	err = ChangeUserGroup(file, sb, &usersInode, chgrp.User, chgrp.Grp)
	if err != nil {
		return fmt.Errorf("error cambiando el grupo del usuario '%s': %v", chgrp.User, err)
	}

	err = sb.Codificar(file, int64(partition.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el superbloque: %v", err)
	}

	fmt.Fprintf(outputBuffer, "El grupo del usuario '%s' ha sido cambiado exitosamente a '%s'\n", chgrp.User, chgrp.Grp)
	fmt.Println("\nInodos")
	sb.PrintInodes(file.Name())
	fmt.Println("\nBloques")
	sb.PrintBlocks(file.Name())
	fmt.Fprintln(outputBuffer, "==================== FIN CHGRP ====================")
	return nil
}

func ChangeUserGroup(file *os.File, sb *estructuras.Superbloque, usersInode *estructuras.Inodo, userName, newGroup string) error {
	contenidoActual, err := globals.ReadFileBlocks(file, sb, usersInode)
	if err != nil {
		return fmt.Errorf("error leyendo el contenido de users.txt: %w", err)
	}

	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")
	var nuevoContenido []string
	var usuarioModificado bool
	var grupoEncontrado bool

	var usuarios []estructuras.Usuario
	var grupos []estructuras.Group

	for _, linea := range lineas {
		partes := strings.Split(linea, ",")
		if len(partes) < 3 {
			continue
		}

		tipo := strings.TrimSpace(partes[1])
		if tipo == "G" {
			group := estructuras.NewGroup(partes[0], partes[2])
			grupos = append(grupos, *group)
		} else if tipo == "U" && len(partes) >= 5 {
			user := estructuras.NewUser(partes[0], partes[2], partes[3], partes[4])
			usuarios = append(usuarios, *user)
		}
	}

	var nuevoIDGrupo string
	for _, group := range grupos {
		if group.Group == newGroup && group.GID != "0" {
			nuevoIDGrupo = group.GID
			grupoEncontrado = true
			break
		}
	}

	if !grupoEncontrado {
		return fmt.Errorf("el grupo '%s' no existe o está eliminado", newGroup)
	}

	for i, usuario := range usuarios {
		if usuario.Name == userName && usuario.Id != "0" {
			fmt.Printf("Cambiando el grupo del usuario '%s' al grupo '%s' (ID grupo: %s)\n", usuario.Name, newGroup, nuevoIDGrupo)
			usuarios[i].Group = newGroup
			usuarios[i].Id = nuevoIDGrupo
			fmt.Printf("Nuevo estado del usuario: %s\n", usuarios[i].ToString())
			usuarioModificado = true
		}
	}

	if !usuarioModificado {
		return fmt.Errorf("el usuario '%s' no existe o está eliminado", userName)
	}

	for _, group := range grupos {
		nuevoContenido = append(nuevoContenido, group.ToString())

		for _, usuario := range usuarios {
			if usuario.Group == group.Group {
				nuevoContenido = append(nuevoContenido, usuario.ToString())
			}
		}
	}

	for _, blockIndex := range usersInode.I_block {
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

	err = WriteContentToBlocks(file, sb, usersInode, nuevoContenido)
	if err != nil {
		return fmt.Errorf("error guardando los cambios en users.txt: %v", err)
	}

	usersInode.I_size = int32(len(strings.Join(nuevoContenido, "\n")))

	usersInode.ActualizarMtime()
	usersInode.ActualizarCtime()

	inodeOffset := int64(sb.S_inode_start + int32(binary.Size(*usersInode)))
	err = usersInode.Encode(file, inodeOffset)
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %w", err)
	}

	return nil
}

func WriteContentToBlocks(file *os.File, sb *estructuras.Superbloque, usersInode *estructuras.Inodo, contenido []string) error {
	contenidoFinal := strings.Join(contenido, "\n") + "\n"
	data := []byte(contenidoFinal)

	blockSize := int(sb.S_block_size)

	for i, blockIndex := range usersInode.I_block {
		if blockIndex == -1 {
			break
		}

		start := i * blockSize
		end := start + blockSize
		if end > len(data) {
			end = len(data)
		}

		var fileBlock estructuras.ArchivoBloque
		copy(fileBlock.B_content[:], data[start:end])

		fmt.Printf("Escribiendo bloque %d: %s\n", blockIndex, string(fileBlock.B_content[:]))
	}

	return nil
}
