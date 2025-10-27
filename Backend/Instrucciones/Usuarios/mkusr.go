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

type MKUSR struct {
	User string
	Pass string
	Grp  string
}

func validateParamLength(param string, maxLength int, paramName string) error {
	if len(param) > maxLength {
		return fmt.Errorf("%s debe tener un máximo de %d caracteres", paramName, maxLength)
	}
	return nil
}

func AnalizarMkusr(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer

	cmd := &MKUSR{}

	reUser := regexp.MustCompile(`-user=[^\s]+`)
	rePass := regexp.MustCompile(`-pass=[^\s]+`)
	reGrp := regexp.MustCompile(`-grp=[^\s]+`)

	matchesUser := reUser.FindString(strings.Join(tokens, " "))
	matchesPass := rePass.FindString(strings.Join(tokens, " "))
	matchesGrp := reGrp.FindString(strings.Join(tokens, " "))

	if matchesUser == "" {
		return "", fmt.Errorf("falta el parámetro -user")
	}
	if matchesPass == "" {
		return "", fmt.Errorf("falta el parámetro -pass")
	}
	if matchesGrp == "" {
		return "", fmt.Errorf("falta el parámetro -grp")
	}

	cmd.User = strings.SplitN(matchesUser, "=", 2)[1]
	cmd.Pass = strings.SplitN(matchesPass, "=", 2)[1]
	cmd.Grp = strings.SplitN(matchesGrp, "=", 2)[1]

	if err := validateParamLength(cmd.User, 10, "Usuario"); err != nil {
		return "", err
	}
	if err := validateParamLength(cmd.Pass, 10, "Contraseña"); err != nil {
		return "", err
	}
	if err := validateParamLength(cmd.Grp, 10, "Grupo"); err != nil {
		return "", err
	}

	err := commandMkusr(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandMkusr(mkusr *MKUSR, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "======================= MKUSR =======================")
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

	_, err = globals.FindInUsersFile(file, sb, &usersInode, mkusr.Grp, "G")
	if err != nil {
		return fmt.Errorf("el grupo '%s' no existe", mkusr.Grp)
	}

	_, err = globals.FindInUsersFile(file, sb, &usersInode, mkusr.User, "U")
	if err == nil {
		return fmt.Errorf("el usuario '%s' ya existe", mkusr.User)
	}

	usuario := estructuras.NewUser(fmt.Sprintf("%d", sb.S_inodes_count+1), mkusr.Grp, mkusr.User, mkusr.Pass)
	fmt.Println(usuario.ToString())

	err = globals.InsertIntoUsersFile(file, sb, &usersInode, usuario.ToString())
	if err != nil {
		return fmt.Errorf("error insertando el usuario '%s': %v", mkusr.User, err)
	}

	err = usersInode.Encode(file, inodeOffset)
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %v", err)
	}

	err = sb.Codificar(file, int64(partition.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el Superblock: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Usuario '%s' agregado exitosamente al grupo '%s'\n", mkusr.User, mkusr.Grp)
	fmt.Println("\nSuperblock")
	sb.Print()
	fmt.Println("\nInodos")
	sb.PrintInodes(file.Name())
	fmt.Println("\nBloques")
	sb.PrintBlocks(file.Name())
	fmt.Fprintf(outputBuffer, "======================= FIN MKUSR =======================\n")

	return nil
}
