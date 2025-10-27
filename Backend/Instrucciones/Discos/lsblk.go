package instrucciones

import (
	"bytes"
	"errors"
	"fmt"
	estructuras "godisk/Estructuras"
	"os"
	"strings"
)

type ListPartitions struct {
	path string
}

func AnalizarListPartitions(tokens []string) (string, error) {
	cmd := &ListPartitions{}
	var outputBuffer bytes.Buffer

	for _, token := range tokens {
		if strings.HasPrefix(strings.ToLower(token), "-path=") {
			cmd.path = strings.Trim(strings.SplitN(token, "=", 2)[1], "\"")
		}
	}

	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	err := commandListPartitions(cmd, &outputBuffer)
	if err != nil {
		return "", fmt.Errorf("error al listar las particiones: %v", err)
	}

	return outputBuffer.String(), nil
}

func commandListPartitions(listCmd *ListPartitions, outputBuffer *bytes.Buffer) error {
	file, err := os.Open(listCmd.path)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer file.Close()

	mbr := &estructuras.Mbr{}
	err = mbr.Decodificar(file)
	if err != nil {
		return fmt.Errorf("error al leer el MBR del disco: %v", err)
	}

	fmt.Fprintln(outputBuffer, "===================== LISTA DE PARTICIONES =====================")
	fmt.Fprintf(outputBuffer, "	Disco: %s 	(Tamaño: %d 	bytes)\n", listCmd.path, mbr.Mbr_tamano)
	fmt.Fprintln(outputBuffer, "-----------------------------------------------------------------")
	fmt.Fprintln(outputBuffer, "Tipo     	Nombre      	Inicio       	Tamaño       	Estado")

	for _, part := range mbr.Mbr_partitions {
		if part.Part_s > 0 {
			partName := strings.TrimRight(string(part.Part_name[:]), "\x00")
			partType := "Desconocido"
			if part.Part_type[0] == 'P' {
				partType = "Primaria"
			} else if part.Part_type[0] == 'E' {
				partType = "Extendida"
			}

			partStatus := "Libre"
			if part.Part_status[0] != '9' {
				partStatus = "Ocupado"
			}

			fmt.Fprintf(outputBuffer, "%-8s %-10s %-12d %-12d %s\n", partType, partName, part.Part_start, part.Part_s, partStatus)

			if part.Part_type[0] == 'E' {
				listLogicalPartitions(file, part.Part_start, outputBuffer)
			}
		}
	}

	fmt.Fprintln(outputBuffer, "=================================================================")
	return nil
}

func listLogicalPartitions(file *os.File, start int32, outputBuffer *bytes.Buffer) {
	ebrStart := start

	fmt.Fprintln(outputBuffer, "  Particiones lógicas dentro de la extendida:")
	for ebrStart != -1 {
		ebr := &estructuras.Ebr{}
		err := ebr.Codificar(file, int64(ebrStart))
		if err != nil {
			fmt.Fprintf(outputBuffer, "  Error al leer EBR en la posición %d: %v\n", ebrStart, err)
			return
		}

		ebrName := strings.TrimRight(string(ebr.Part_name[:]), "\x00")
		ebrFit := string(ebr.Part_fit[:])
		ebrMount := "No Montada"
		if ebr.Part_mount[0] == '1' {
			ebrMount = "Montada"
		}

		fmt.Fprintf(outputBuffer, "  Lógica  %-10s %-12d %-12d %-6s %-10s Next: %d\n",
			ebrName, ebr.Part_start, ebr.Part_s, ebrFit, ebrMount, ebr.Part_next)

		ebrStart = ebr.Part_next
	}
}
