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

type Mount struct {
	path string
	name string
}

func AnalizarMount(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer
	cmd := &Mount{}

	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-name="[^"]+"|-name=[^\s]+`)
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
		case "-name":
			if value == "" {
				return "", errors.New("el nombre no puede estar vacío")
			}
			cmd.name = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}
	if cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -name")
	}

	err := commandMount(cmd, &outputBuffer)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandMount(mount *Mount, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "========================== MOUNT ==========================")

	file, err := os.OpenFile(mount.path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo el archivo del disco en el path: %s: %v", mount.path, err)
	}
	defer file.Close()

	var mbr estructuras.Mbr
	err = mbr.Decodificar(file)
	if err != nil {
		return fmt.Errorf("error deserializando el MBR: %v", err)
	}

	partition, indexPartition := mbr.ObtenerParticionPorNombre(mount.name)
	if partition == nil {
		return fmt.Errorf("error: la partición '%s' no existe en el disco", mount.name)
	}

	for id, mountedPath := range global.ParticionesMontadas {
		if mountedPath == mount.path && strings.Contains(id, mount.name) {
			return fmt.Errorf("error: la partición '%s' ya está montada con ID: %s", mount.name, id)
		}
	}

	idPartition, err := GenerateIdPartition(mount, indexPartition)
	if err != nil {
		return fmt.Errorf("error generando el ID de la partición: %v", err)
	}

	global.ParticionesMontadas[idPartition] = mount.path

	partition.MontarParticion(indexPartition, idPartition)
	mbr.Mbr_partitions[indexPartition] = *partition

	err = mbr.Codificar(file)
	if err != nil {
		return fmt.Errorf("error serializando el MBR de vuelta al disco: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Partición '%s' montada correctamente con ID: %s\n", mount.name, idPartition)
	fmt.Fprintln(outputBuffer, "\n=== Particiones Montadas ===")
	for id, path := range global.ParticionesMontadas {
		fmt.Fprintf(outputBuffer, "ID: %s | Path: %s\n", id, path)
	}
	fmt.Fprintln(outputBuffer, "============================ FIN MOUNT ========================")

	return nil
}

func GenerateIdPartition(mount *Mount, indexPartition int) (string, error) {
	lastTwoDigits := global.Carnet[len(global.Carnet)-2:]
	letter, err := utilidades.ObtenerLetra(mount.path)
	if err != nil {
		return "", err
	}

	idPartition := fmt.Sprintf("%s%d%s", lastTwoDigits, indexPartition+1, letter)
	return idPartition, nil
}
