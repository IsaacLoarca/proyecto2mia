package instrucciones

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type rmDisk struct {
	path string
}

func AnalizarRmdisk(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer

	cmd := &rmDisk{}

	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+`)
	matches := re.FindAllString(args, -1)

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}
		key, value := strings.ToLower(kv[0]), kv[1]

		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		switch key {
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	err := commandRmdisk(cmd, &outputBuffer)
	if err != nil {
		return "", fmt.Errorf("error al eliminar el disco: %v", err)
	}

	return outputBuffer.String(), nil
}

func commandRmdisk(rmdisk *rmDisk, outputBuffer *bytes.Buffer) error {
	// Redirigir las salidas de fmt a outputBuffer
	fmt.Fprintln(outputBuffer, "============================= RMDISK ===============================")
	fmt.Fprintf(outputBuffer, "Eliminando disco en %s...\n", rmdisk.path)

	// Verificar si el archivo existe
	if _, err := os.Stat(rmdisk.path); os.IsNotExist(err) {
		return fmt.Errorf("el archivo %s no existe", rmdisk.path)
	}

	// Eliminar el archivo inmediatamente, sin preguntar
	err := os.Remove(rmdisk.path)
	if err != nil {
		return fmt.Errorf("error al eliminar el archivo: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Disco en %s eliminado exitosamente.\n", rmdisk.path)
	fmt.Fprintln(outputBuffer, "============================== FIN RMDISK ===============================")
	return nil
}
