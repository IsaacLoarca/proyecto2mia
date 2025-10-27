package reportes

import (
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"os"
	"os/exec"
	"strings"
)

func ReporteDisk(mbr *estructuras.Mbr, path string, diskPath string) error {

	err := utilidades.CrearDirectoriosPadre(path)
	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}

	file, err := os.Open(diskPath)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer file.Close()

	dotFileName, outputImage := utilidades.ObtenerNombresArchivo(path)

	dotContent := `digraph G {
		fontname="Helvetica,Arial,sans-serif"
		node [fontname="Helvetica,Arial,sans-serif"]
		edge [fontname="Helvetica,Arial,sans-serif"]
		concentrate=True;
		rankdir=TB;
		node [shape=record];

		title [label="Reporte DISK" shape=plaintext fontname="Helvetica,Arial,sans-serif"];

		dsk [label="`

	totalSize := mbr.Mbr_tamano
	usedSize := int32(0)

	dotContent += "{MBR}"

	for _, part := range mbr.Mbr_partitions {
		if part.Part_s > 0 {
			percentage := (float64(part.Part_s) / float64(totalSize)) * 100
			usedSize += part.Part_s

			partName := strings.TrimRight(string(part.Part_name[:]), "\x00")
			if part.Part_type[0] == 'P' {
				dotContent += fmt.Sprintf("|{Primaria %s\\n%.2f%%}", partName, percentage)
			} else if part.Part_type[0] == 'E' {
				dotContent += fmt.Sprintf("|{Extendida %.2f%%|{", percentage)
				ebrStart := part.Part_start
				ebrCount := 0
				ebrUsedSize := int32(0)
				for ebrStart != -1 {
					ebr := &estructuras.Ebr{}
					err := ebr.Decodificar(file, int64(ebrStart))
					if err != nil {
						return fmt.Errorf("error al decodificar EBR: %v", err)
					}

					ebrName := strings.TrimRight(string(ebr.Part_name[:]), "\x00")
					ebrPercentage := (float64(ebr.Part_s) / float64(totalSize)) * 100
					ebrUsedSize += ebr.Part_s

					if ebrCount > 0 {
						dotContent += "|"
					}
					dotContent += fmt.Sprintf("{EBR|Lógica %s\\n%.2f%%}", ebrName, ebrPercentage)

					ebrStart = ebr.Part_next
					ebrCount++
				}

				extendedFreeSize := part.Part_s - ebrUsedSize
				if extendedFreeSize > 0 {
					extendedFreePercentage := (float64(extendedFreeSize) / float64(totalSize)) * 100
					dotContent += fmt.Sprintf("|Libre %.2f%%", extendedFreePercentage)
				}

				dotContent += "}}"
			}
		}
	}

	freeSize := totalSize - usedSize
	if freeSize > 0 {
		freePercentage := (float64(freeSize) / float64(totalSize)) * 100
		dotContent += fmt.Sprintf("|Libre %.2f%%", freePercentage)
	}

	dotContent += `"];

		title -> dsk [style=invis];
	}`

	dotFile, err := os.Create(dotFileName)
	if err != nil {
		return fmt.Errorf("error al crear el archivo DOT: %v", err)
	}
	defer dotFile.Close()

	_, err = dotFile.WriteString(dotContent)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo DOT: %v", err)
	}

	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Reporte de disco generado:", outputImage)
	return nil
}
