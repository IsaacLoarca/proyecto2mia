package reportes

import (
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ReporteMBR(mbr *estructuras.Mbr, path string, file *os.File) error {
	err := utilidades.CrearDirectoriosPadre(path)
	if err != nil {
		return err
	}

	dotFileName, outputImage := utilidades.ObtenerNombresArchivo(path)

	primaryColor := "#FFDDC1"
	extendedColor := "#C1E1C1"
	logicalColor := "#C1D1FF"
	ebrColor := "#FFD1DC"
	unallocatedColor := "#FFFFFF"

	dotContent := fmt.Sprintf(`digraph G {
        node [shape=plaintext]
        tabla [label=<
            <table border="0" cellborder="1" cellspacing="0">
                <tr><td colspan="2" bgcolor="#F8D7DA"><b>REPORTE MBR</b></td></tr>
                <tr><td bgcolor="#F5B7B1">mbr_tamano</td><td bgcolor="#F5B7B1">%d</td></tr>
                <tr><td bgcolor="#F5B7B1">mbr_fecha_creacion</td><td bgcolor="#F5B7B1">%s</td></tr>
                <tr><td bgcolor="#F5B7B1">mbr_disk_signature</td><td bgcolor="#F5B7B1">%d</td></tr>
            `, mbr.Mbr_tamano, time.Unix(int64(mbr.Mbr_fecha_creacion), 0), mbr.Mbr_dsk_signature)

	totalSize := mbr.Mbr_tamano
	allocatedSize := int32(0)

	for i, part := range mbr.Mbr_partitions {
		if part.Part_s > 0 && part.Part_start > 0 {
			if part.Part_start > allocatedSize {
				unallocatedSize := part.Part_start - allocatedSize
				dotContent += fmt.Sprintf(`
                    <tr><td colspan="2" bgcolor="%s"><b>ESPACIO NO ASIGNADO (Tamaño: %d bytes)</b></td></tr>
                `, unallocatedColor, unallocatedSize)
				allocatedSize += unallocatedSize
			}

			partName := strings.TrimRight(string(part.Part_name[:]), "\x00")
			partStatus := rune(part.Part_status[0])
			partType := rune(part.Part_type[0])
			partFit := rune(part.Part_fit[0])

			rowColor := ""
			switch partType {
			case 'P':
				rowColor = primaryColor
			case 'E':
				rowColor = extendedColor
			}

			dotContent += fmt.Sprintf(`
                <tr><td colspan="2" bgcolor="%s"><b>PARTICIÓN %d</b></td></tr>
                <tr><td bgcolor="%s">part_status</td><td bgcolor="%s">%c</td></tr>
                <tr><td bgcolor="%s">part_type</td><td bgcolor="%s">%c</td></tr>
                <tr><td bgcolor="%s">part_fit</td><td bgcolor="%s">%c</td></tr>
                <tr><td bgcolor="%s">part_start</td><td bgcolor="%s">%d</td></tr>
                <tr><td bgcolor="%s">part_size</td><td bgcolor="%s">%d</td></tr>
                <tr><td bgcolor="%s">part_name</td><td bgcolor="%s">%s</td></tr>
            `, rowColor, i+1,
				rowColor, rowColor, partStatus,
				rowColor, rowColor, partType,
				rowColor, rowColor, partFit,
				rowColor, rowColor, part.Part_start,
				rowColor, rowColor, part.Part_s,
				rowColor, rowColor, partName)

			allocatedSize += part.Part_s

			if partType == 'E' {
				ebrStart := part.Part_start
				dotContent += fmt.Sprintf(`
                    <tr><td colspan="2" bgcolor="%s"><b>PART. EXTENDIDA (Inicio: %d)</b></td></tr>
                `, extendedColor, ebrStart)

				for ebrStart != -1 {
					ebr := &estructuras.Ebr{}
					err := ebr.Decodificar(file, int64(ebrStart))
					if err != nil {
						return fmt.Errorf("error al decodificar EBR: %v", err)
					}

					ebrName := strings.TrimRight(string(ebr.Part_name[:]), "\x00")
					ebrFit := rune(ebr.Part_fit[0])

					dotContent += fmt.Sprintf(`
                        <tr><td colspan="2" bgcolor="%s"><b>EBR (Inicio: %d)</b></td></tr>
                        <tr><td bgcolor="%s">ebr_fit</td><td bgcolor="%s">%c</td></tr>
                        <tr><td bgcolor="%s">ebr_start</td><td bgcolor="%s">%d</td></tr>
                        <tr><td bgcolor="%s">ebr_size</td><td bgcolor="%s">%d</td></tr>
                        <tr><td bgcolor="%s">ebr_next</td><td bgcolor="%s">%d</td></tr>
                        <tr><td bgcolor="%s">ebr_name</td><td bgcolor="%s">%s</td></tr>
                    `, ebrColor, ebrStart,
						ebrColor, ebrColor, ebrFit,
						ebrColor, ebrColor, ebr.Part_start,
						ebrColor, ebrColor, ebr.Part_s,
						ebrColor, ebrColor, ebr.Part_next,
						ebrColor, ebrColor, ebrName)

					if ebr.Part_s > 0 {
						dotContent += fmt.Sprintf(`
                            <tr><td colspan="2" bgcolor="%s"><b>PART. LÓGICA (Inicio: %d)</b></td></tr>
                        `, logicalColor, ebr.Part_start)
					}

					allocatedSize += ebr.Part_s
					ebrStart = ebr.Part_next
				}
			}
		}
	}

	if allocatedSize < totalSize {
		unallocatedSize := totalSize - allocatedSize
		dotContent += fmt.Sprintf(`
            <tr><td colspan="2" bgcolor="%s"><b>ESPACIO NO ASIGNADO (Tamaño: %d bytes)</b></td></tr>
        `, unallocatedColor, unallocatedSize)
	}

	dotContent += "</table>>] }"

	file, err = os.Create(dotFileName)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(dotContent)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo: %v", err)
	}

	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Imagen de la tabla generada:", outputImage)
	return nil
}
