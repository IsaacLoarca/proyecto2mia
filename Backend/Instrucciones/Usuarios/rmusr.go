package instrucciones

import (
	"bytes"
	"encoding/binary"
	"fmt"
	estructuras "godisk/Estructuras"
	globals "godisk/Global"
	"os"
	"regexp"
	"strings"
)

type RMUSR struct {
	User string
}

func AnalizarRmusr(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer

	cmd := &RMUSR{}

	re := regexp.MustCompile(`-usr=[^\s]+`)
	matches := re.FindString(strings.Join(tokens, " "))

	if matches == "" {
		return "", fmt.Errorf("falta el parámetro -usr")
	}

	param := strings.SplitN(matches, "=", 2)
	if len(param) != 2 {
		return "", fmt.Errorf("formato incorrecto para -user")
	}
	cmd.User = param[1]

	err := commandRmusr(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandRmusr(rmusr *RMUSR, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "======================= RMUSR =======================")
	if !globals.EstaLogueado() {
		return fmt.Errorf("no hay ninguna sesión activa")
	}
	if globals.UsuarioActual.Name != "root" {
		return fmt.Errorf("solo el usuario root puede ejecutar este comando")
	}

	_, path, err := globals.ObtenerParticionMontada(globals.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se puede encontrar la partición montada: %v", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0755)
	if err != nil {
		return fmt.Errorf("no se puede abrir el archivo de la partición: %v", err)
	}
	defer file.Close()

	mbr, sb, _, err := globals.GetMountedPartitionRep(globals.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo cargar el Superblock: %v", err)
	}

	partition, err := mbr.ObtenerParticionPorID(globals.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo obtener la partición: %v", err)
	}

	var usersInode estructuras.Inodo
	inodeOffset := int64(sb.S_inode_start + int32(binary.Size(usersInode)))
	err = usersInode.Decode(file, inodeOffset)
	if err != nil {
		return fmt.Errorf("error leyendo el inodo de users.txt: %v", err)
	}

	_, err = globals.FindInUsersFile(file, sb, &usersInode, rmusr.User, "U")
	if err != nil {
		return fmt.Errorf("el usuario '%s' no existe", rmusr.User)
	}

	err = UpdateUserState(file, sb, &usersInode, rmusr.User)
	if err != nil {
		return fmt.Errorf("error eliminando el usuario '%s': %v", rmusr.User, err)
	}

	err = usersInode.Encode(file, inodeOffset)
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %v", err)
	}

	err = sb.Codificar(file, int64(partition.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el Superblock: %v", err)
	} else {
		fmt.Println("Superbloque guardado correctamente")

	}

	sb.Print()
	fmt.Println("------")
	fmt.Fprintf(outputBuffer, "Usuario '%s' eliminado exitosamente.\n", rmusr.User)
	fmt.Println("\nBloques:")
	sb.PrintBlocks(file.Name())
	fmt.Println("\nInodos:")
	sb.PrintInodes(file.Name())
	fmt.Fprintf(outputBuffer, "====================== FIN RMUSR =====================")
	return nil
}

func UpdateUserState(file *os.File, sb *estructuras.Superbloque, usersInode *estructuras.Inodo, userName string) error {
	contenido, err := globals.ReadFileBlocks(file, sb, usersInode)
	if err != nil {
		return fmt.Errorf("error leyendo el contenido de users.txt: %v", err)
	}

	lineas := strings.Split(contenido, "\n")
	modificado := false

	for i, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		usuario := crearUsuarioDesdeLinea(linea)

		if usuario != nil && usuario.Name == userName {
			usuario.Eliminar()

			lineas[i] = usuario.ToString()
			modificado = true
			break
		}
	}

	if !modificado {
		return fmt.Errorf("usuario '%s' no encontrado en users.txt", userName)
	}

	contenidoActualizado := limpiarYActualizarContenido(lineas)

	return escribirCambiosEnArchivo(file, sb, usersInode, contenidoActualizado)
}

func crearUsuarioDesdeLinea(linea string) *estructuras.Usuario {
	partes := strings.Split(linea, ",")
	if len(partes) >= 5 && partes[1] == "U" {
		return estructuras.NewUser(partes[0], partes[2], partes[3], partes[4])
	}
	return nil
}

func limpiarYActualizarContenido(lineas []string) string {
	var contenidoActualizado []string
	for _, linea := range lineas {
		if strings.TrimSpace(linea) != "" {
			contenidoActualizado = append(contenidoActualizado, linea)
		}
	}
	return strings.Join(contenidoActualizado, "\n") + "\n"
}

func escribirCambiosEnArchivo(file *os.File, sb *estructuras.Superbloque, usersInode *estructuras.Inodo, contenido string) error {
	for _, blockIndex := range usersInode.I_block {
		if blockIndex == -1 {
			break
		}

		blockOffset := int64(sb.S_block_start + blockIndex*sb.S_block_size)
		var fileBlock estructuras.ArchivoBloque

		fileBlock.ClearContent()

		err := fileBlock.Encode(file, blockOffset)
		if err != nil {
			return fmt.Errorf("error escribiendo bloque limpio %d: %w", blockIndex, err)
		}
	}

	err := globals.WriteUsersBlocks(file, sb, usersInode, contenido)
	if err != nil {
		return fmt.Errorf("error guardando los cambios en users.txt: %v", err)
	}

	return nil
}
