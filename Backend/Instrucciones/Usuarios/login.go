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

type LOGIN struct {
	User string
	Pass string
	ID   string
}

func ParserLogin(tokens []string) (map[string]interface{}, error) {
	var outputBuffer bytes.Buffer
	cmd := &LOGIN{}
	args := strings.Join(tokens, " ")

	// Expresión regular para los parámetros
	re := regexp.MustCompile(`-user=[^\s]+|-pass=[^\s]+|-id=[^\s]+`)
	matches := re.FindAllString(args, -1)

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			switch strings.ToLower(kv[0]) {
			case "-user":
				return map[string]interface{}{
					"status":  "error",
					"message": "El formato del parámetro 'user' es incorrecto. Asegúrese de escribir '-user=nombre_usuario'.",
				}, fmt.Errorf("formato de parámetro 'user' incorrecto")
			case "-pass":
				return map[string]interface{}{
					"status":  "error",
					"message": "El formato del parámetro 'pass' es incorrecto. Asegúrese de escribir '-pass=contraseña'.",
				}, fmt.Errorf("formato de parámetro 'pass' incorrecto")
			case "-id":
				return map[string]interface{}{
					"status":  "error",
					"message": "El formato del parámetro 'id' es incorrecto. Asegúrese de escribir '-id=id_partición'.",
				}, fmt.Errorf("formato de parámetro 'id' incorrecto")
			default:
				return map[string]interface{}{
					"status":  "error",
					"message": fmt.Sprintf("Parámetro desconocido: %s. Use solo '-user', '-pass' y '-id'.", kv[0]),
				}, fmt.Errorf("parámetro desconocido: %s", kv[0])
			}
		}

		key, value := strings.ToLower(kv[0]), kv[1]
		switch key {
		case "-user":
			cmd.User = value
		case "-pass":
			cmd.Pass = value
		case "-id":
			cmd.ID = value
		}
	}

	// Validar que se hayan proporcionado todos los parámetros
	missingParams := []string{}
	if cmd.User == "" {
		missingParams = append(missingParams, "El parámetro 'user' es requerido.")
	}
	if cmd.Pass == "" {
		missingParams = append(missingParams, "El parámetro 'pass' es requerido.")
	}
	if cmd.ID == "" {
		missingParams = append(missingParams, "El parámetro 'id' es requerido.")
	}
	if len(missingParams) > 0 {
		return map[string]interface{}{
			"status":  "error",
			"message": strings.Join(missingParams, " "),
		}, fmt.Errorf("faltan parámetros requeridos: %s", strings.Join(missingParams, ", "))
	}

	// Ejecutar el comando login
	err := commandLogin(cmd, &outputBuffer)
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}, err
	}

	return map[string]interface{}{
		"status":  "success",
		"message": outputBuffer.String(),
	}, nil
}

// Lógica para ejecutar el login con respuesta estructurada
func commandLogin(login *LOGIN, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "===== INICIO DE LOGIN =====")
	fmt.Fprintf(outputBuffer, "Intentando iniciar sesión con ID: %s, Usuario: %s\n", login.ID, login.User)

	// Verificar si ya hay un usuario logueado
	if globals.UsuarioActual != nil && globals.UsuarioActual.Status {
		return fmt.Errorf("ya hay un usuario logueado, debe cerrar sesión primero")
	}

	// Ver las particiones montadas
	for id, path := range globals.ParticionesMontadas {
		fmt.Fprintf(outputBuffer, "Partición montada con ID: %s | Path: %s\n", id, path)
	}

	// Verificar si la partición está montada
	_, path, err := globals.ObtenerParticionMontada(login.ID)
	if err != nil {
		return fmt.Errorf("no se puede encontrar la partición: %v", err)
	}

	// Cargar el Superblock de la partición montada
	_, sb, _, err := globals.GetMountedPartitionRep(login.ID)
	if err != nil {
		return fmt.Errorf("no se pudo cargar el Superblock: %v", err)
	}
	fmt.Fprintln(outputBuffer, "Superblock cargado correctamente")

	// Leer el archivo users.txt (inodo 1)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("no se puede abrir el archivo de partición: %v", err)
	}
	defer file.Close()

	var usersInode estructuras.Inodo
	inodeOffset := int64(sb.S_inode_start + int32(binary.Size(usersInode)))
	err = usersInode.Decode(file, inodeOffset)
	if err != nil {
		return fmt.Errorf("error leyendo inodo de users.txt: %v", err)
	}
	usersInode.ActualizarAtime()

	var contenido string
	for _, blockIndex := range usersInode.I_block {
		if blockIndex == -1 {
			continue
		}

		blockOffset := int64(sb.S_block_start + blockIndex*int32(binary.Size(estructuras.ArchivoBloque{})))
		var fileBlock estructuras.ArchivoBloque
		err = fileBlock.Decode(file, blockOffset)
		if err != nil {
			return fmt.Errorf("error leyendo bloque de users.txt: %v", err)
		}
		contenido += string(fileBlock.B_content[:])
	}

	// Validar usuario y contraseña
	encontrado := false
	for _, linea := range strings.Split(strings.TrimSpace(contenido), "\n") {
		if linea == "" {
			continue
		}

		datos := strings.Split(linea, ",")
		if len(datos) == 5 && datos[1] == "U" {
			usuario := estructuras.NewUser(datos[0], datos[2], datos[3], datos[4])
			if usuario.Name == login.User && usuario.Password == login.Pass {
				encontrado = true
				globals.UsuarioActual = usuario
				globals.UsuarioActual.Status = true
				fmt.Fprintf(outputBuffer, "Bienvenido %s, inicio de sesión exitoso.\n", usuario.Name)
				globals.UsuarioActual.Id = login.ID
				break
			}
		}
	}

	if !encontrado {
		return fmt.Errorf("usuario o contraseña incorrectos")
	}

	fmt.Fprintln(outputBuffer, "======================================================")
	return nil
}

func AnalizarLogin(tokens []string) (string, error) {
	result, err := ParserLogin(tokens)
	if err != nil {
		return "", err
	}
	return result["message"].(string), nil
}
