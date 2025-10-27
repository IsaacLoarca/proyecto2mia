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

type RMGRP struct {
	Name string
}

func AnalizarRmgrp(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer

	cmd := &RMGRP{}

	re := regexp.MustCompile(`-name=[^\s]+`)
	matches := re.FindString(strings.Join(tokens, " "))

	if matches == "" {
		return "", fmt.Errorf("falta el parámetro -name")
	}

	param := strings.SplitN(matches, "=", 2)
	if len(param) != 2 {
		return "", fmt.Errorf("formato incorrecto para -name")
	}
	cmd.Name = param[1]

	err := commandRmgrp(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandRmgrp(rmgrp *RMGRP, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "======================= RMGRP =======================")
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

	_, err = globals.FindInUsersFile(file, sb, &usersInode, rmgrp.Name, "G")
	if err != nil {
		return fmt.Errorf("el grupo '%s' no existe", rmgrp.Name)
	}

	err = UpdateEntityStateOrRemoveUsers(file, sb, &usersInode, rmgrp.Name, "G", "0")
	if err != nil {
		return fmt.Errorf("error eliminando el grupo y usuarios asociados: %v", err)
	}

	err = usersInode.Encode(file, inodeOffset)
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %v", err)
	}

	err = sb.Codificar(file, int64(partition.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el Superblock: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Grupo '%s' eliminado exitosamente, junto con sus usuarios.\n", rmgrp.Name)
	fmt.Println("\nInodos actualizados:")
	sb.PrintInodes(file.Name())
	fmt.Println("\nBloques de datos actualizados:")
	sb.PrintBlocks(file.Name())

	fmt.Fprintln(outputBuffer, "======================= FIN RMGRP =======================")

	return nil
}

func UpdateEntityStateOrRemoveUsers(file *os.File, sb *estructuras.Superbloque, usersInode *estructuras.Inodo, name string, entityType string, newState string) error {
	contenido, err := globals.ReadFileBlocks(file, sb, usersInode)
	if err != nil {
		return fmt.Errorf("error leyendo el contenido de users.txt: %v", err)
	}

	lineas := strings.Split(contenido, "\n")
	modificado := false

	var groupName string
	if entityType == "G" {
		groupName = name
	}

	for i, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		partes := strings.Split(linea, ",")
		if len(partes) < 3 {
			continue
		}

		tipo := partes[1]
		nombre := partes[2]

		if tipo == entityType && nombre == name {
			partes[0] = newState
			lineas[i] = strings.Join(partes, ",")
			modificado = true

			if entityType == "G" {
				for j, lineaUsuario := range lineas {
					lineaUsuario = strings.TrimSpace(lineaUsuario)
					if lineaUsuario == "" {
						continue
					}
					partesUsuario := strings.Split(lineaUsuario, ",")
					if len(partesUsuario) == 5 && partesUsuario[2] == groupName {
						partesUsuario[0] = "0"
						lineas[j] = strings.Join(partesUsuario, ",")
					}
				}
			}
			break
		}
	}

	if modificado {
		contenidoActualizado := strings.Join(lineas, "\n")

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

		err = globals.WriteUsersBlocks(file, sb, usersInode, contenidoActualizado)
		if err != nil {
			return fmt.Errorf("error guardando los cambios en users.txt: %v", err)
		}
	} else {
		return fmt.Errorf("%s '%s' no encontrado en users.txt", entityType, name)
	}

	return nil
}
