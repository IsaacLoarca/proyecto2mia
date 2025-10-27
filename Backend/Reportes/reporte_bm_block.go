package reportes

import (
	"encoding/binary"
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"os"
	"strings"
)

func ReporteBMBloque(superbloque *estructuras.Superbloque, rutaDisco string, rutaSalida string) error {
	err := utilidades.CrearDirectoriosPadre(rutaSalida)
	if err != nil {
		return fmt.Errorf("error creando carpetas padre: %v", err)
	}

	archivo, err := os.Open(rutaDisco)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer archivo.Close()

	totalBlocks := superbloque.S_blocks_count + superbloque.S_free_blocks_count

	byteCount := (totalBlocks + 7) / 8

	var bitmapContent strings.Builder

	for byteIndex := int32(0); byteIndex < byteCount; byteIndex++ {
		_, err := archivo.Seek(int64(superbloque.S_bm_block_start+byteIndex), 0)
		if err != nil {
			return fmt.Errorf("error al posicionar el archivo: %v", err)
		}

		var byteVal byte
		err = binary.Read(archivo, binary.LittleEndian, &byteVal)
		if err != nil {
			return fmt.Errorf("error al leer el byte del bitmap: %v", err)
		}

		for bitOffset := 0; bitOffset < 8; bitOffset++ {
			if byteIndex*8+int32(bitOffset) >= totalBlocks {
				break
			}

			if (byteVal & (1 << bitOffset)) != 0 {
				bitmapContent.WriteByte('1')
			} else {
				bitmapContent.WriteByte('0')
			}

			if (byteIndex*8+int32(bitOffset)+1)%20 == 0 {
				bitmapContent.WriteString("\n")
			}
		}
	}

	archivoTXT, err := os.Create(rutaSalida)
	if err != nil {
		return fmt.Errorf("error al crear el archivo de reporte: %v", err)
	}
	defer archivoTXT.Close()

	_, err = archivoTXT.WriteString(bitmapContent.String())
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo de reporte: %v", err)
	}

	fmt.Println("Reporte del bitmap de bloques generado correctamente:", rutaSalida)
	return nil
}
