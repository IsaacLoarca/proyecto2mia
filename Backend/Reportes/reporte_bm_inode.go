package reportes

import (
	"encoding/binary"
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"os"
	"strings"
)

func ReporteBMInodo(superbloque *estructuras.Superbloque, rutaDisco string, rutaSalida string) error {
	err := utilidades.CrearDirectoriosPadre(rutaSalida)

	if err != nil {
		return fmt.Errorf("error creando carpetas padre: %v", err)
	}

	archivo, err := os.Open(rutaDisco)

	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}

	defer archivo.Close()

	totalInodos := superbloque.S_inodes_count + superbloque.S_free_inodes_count

	conteoBytes := (totalInodos + 7) / 8

	var contenidoBitmap strings.Builder

	for indiceByte := int32(0); indiceByte < conteoBytes; indiceByte++ {
		_, err := archivo.Seek(int64(superbloque.S_bm_inode_start+indiceByte), 0)
		if err != nil {
			return fmt.Errorf("error al posicionar el archivo: %v", err)
		}

		var byteVal byte
		err = binary.Read(archivo, binary.LittleEndian, &byteVal)
		if err != nil {
			return fmt.Errorf("error al leer el byte del bitmap: %v", err)
		}

		for bitOffset := 0; bitOffset < 8; bitOffset++ {
			if indiceByte*8+int32(bitOffset) >= totalInodos {
				break
			}

			if (byteVal & (1 << bitOffset)) != 0 {
				contenidoBitmap.WriteByte('1')
			} else {
				contenidoBitmap.WriteByte('0')
			}

			if (indiceByte*8+int32(bitOffset)+1)%20 == 0 {
				contenidoBitmap.WriteString("\n")
			}
		}
	}

	archivoTXT, err := os.Create(rutaSalida)
	if err != nil {
		return fmt.Errorf("error al crear el archivo de reporte: %v", err)
	}
	defer archivoTXT.Close()

	_, err = archivoTXT.WriteString(contenidoBitmap.String())
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo de reporte: %v", err)
	}

	fmt.Println("Reporte del bitmap de inodos generado correctamente:", rutaSalida)

	return nil
}
