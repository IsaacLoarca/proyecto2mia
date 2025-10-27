package instrucciones

import (
	"bytes"
	"errors"
	"fmt"
	estructuras "godisk/Estructuras"
	globals "godisk/Global"
	"os"
	"strings"
)

type Unmount struct {
	id string
}

func AnalizarUnmount(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer
	cmd := &Unmount{}

	for _, token := range tokens {
		if strings.HasPrefix(token, "-id=") {
			cmd.id = strings.TrimPrefix(token, "-id=")
		}
	}

	if cmd.id == "" {
		return "", errors.New("faltan parámetros requeridos: -id")
	}

	err := commandUnmount(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandUnmount(unmount *Unmount, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "==================== DESMONTAJE DE PARTICIÓN ====================")

	mountedPath, exists := globals.ParticionesMontadas[unmount.id]
	if !exists {
		return fmt.Errorf("error: la partición con ID '%s' no se encuentra montada", unmount.id)
	}

	file, err := os.OpenFile(mountedPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al acceder al archivo del disco: %v", err)
	}
	defer file.Close()

	var mbr estructuras.Mbr
	err = mbr.Decodificar(file)
	if err != nil {
		return fmt.Errorf("error al leer el MBR del disco: %v", err)
	}

	found := false
	for i := range mbr.Mbr_partitions {
		partition := &mbr.Mbr_partitions[i]
		partitionID := strings.TrimSpace(string(partition.Part_id[:]))
		if partitionID == unmount.id {
			err = partition.MontarParticion(0, "")
			if err != nil {
				return fmt.Errorf("error al desmontar la partición: %v", err)
			}

			err = mbr.Codificar(file)
			if err != nil {
				return fmt.Errorf("error al guardar cambios en el MBR: %v", err)
			}

			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("error: partición con ID '%s' no localizada en el disco", unmount.id)
	}

	delete(globals.ParticionesMontadas, unmount.id)

	fmt.Fprintf(outputBuffer, "✓ Partición '%s' ha sido desmontada correctamente.\n", unmount.id)
	fmt.Fprintln(outputBuffer, "\n=== Estado Actual de Particiones Montadas ===")
	for id, path := range globals.ParticionesMontadas {
		fmt.Fprintf(outputBuffer, "ID: %s | Ruta: %s\n", id, path)
	}
	fmt.Fprintln(outputBuffer, "=============================================================")

	return nil
}
