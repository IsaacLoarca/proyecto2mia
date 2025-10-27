package instrucciones

import (
	"bytes"
	"encoding/binary"
	"fmt"
	estructuras "godisk/Estructuras"
	globals "godisk/Global"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type MKGRP struct {
	Name string
}

func AnalizarMkgrp(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer

	cmd := &MKGRP{}

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

	err := commandMkgrp(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandMkgrp(mkgrp *MKGRP, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "======================= MKGRP =======================")
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
	usersInode.ActualizarAtime()
	if err != nil {
		return fmt.Errorf("error leyendo el inodo de users.txt: %v", err)
	}

	_, err = globals.FindInUsersFile(file, sb, &usersInode, mkgrp.Name, "G")
	if err == nil {
		return fmt.Errorf("el grupo '%s' ya existe", mkgrp.Name)
	}

	nextGroupID, err := calculateNextID(file, sb, &usersInode)
	if err != nil {
		return fmt.Errorf("error calculando el siguiente ID: %v", err)
	}

	newGroupEntry := fmt.Sprintf("%d,G,%s", nextGroupID, mkgrp.Name)

	err = globals.AddEntryToUsersFile(file, sb, &usersInode, newGroupEntry, mkgrp.Name, "G")
	if err != nil {
		return fmt.Errorf("error creando el grupo '%s': %v", mkgrp.Name, err)
	}

	err = usersInode.Encode(file, inodeOffset)
	usersInode.ActualizarAtime()
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %v", err)
	}

	err = sb.Codificar(file, int64(partition.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el Superblock: %v", err)
	} else {
		fmt.Println("\nSuperbloque guardado correctamente")
		sb.Print()
		fmt.Println("\nInodos")
		sb.PrintInodes(file.Name())
		sb.PrintBlocks(file.Name())

	}

	fmt.Fprintf(outputBuffer, "Grupo creado exitosamente: %s\n", mkgrp.Name)
	fmt.Fprintf(outputBuffer, "============ FIN DE MKGRP ==========\n")
	return nil
}

func calculateNextID(file *os.File, sb *estructuras.Superbloque, inode *estructuras.Inodo) (int, error) {
	contenido, err := globals.ReadFileBlocks(file, sb, inode)
	if err != nil {
		return -1, fmt.Errorf("error leyendo el contenido de users.txt: %v", err)
	}

	lineas := strings.Split(contenido, "\n")
	maxID := 0
	for _, linea := range lineas {
		if linea == "" {
			continue
		}

		campos := strings.Split(linea, ",")
		if len(campos) < 3 {
			continue
		}

		id, err := strconv.Atoi(campos[0])
		if err != nil {
			continue
		}

		if id > maxID {
			maxID = id
		}
	}

	return maxID + 1, nil
}
