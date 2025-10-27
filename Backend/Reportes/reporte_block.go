package reportes

import (
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"html"
	"os"
	"os/exec"
	"strings"
)

func ReporteBloque(superbloque *estructuras.Superbloque, rutaDisco string, ruta string) error {
	err := utilidades.CrearDirectoriosPadre(ruta)

	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}

	archivo, err := os.Open(rutaDisco)

	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}

	defer archivo.Close()

	dotFileName, outputImage := utilidades.ObtenerNombresArchivo(ruta)

	dotContent := iniciarDotGraph()

	dotContent, conexiones, err := generarGrafoBloque(dotContent, superbloque, archivo)

	if err != nil {
		return err
	}

	dotContent += conexiones
	dotContent += "}"

	err = escribirDotFile(dotFileName, dotContent)
	if err != nil {
		return err
	}

	err = ejecutarGraphviz(dotFileName, outputImage)
	if err != nil {
		return err
	}

	fmt.Println("Imagen de los bloques generada:", outputImage)
	return nil
}

func GenerarImagenBloque(dotFileName string, outputImage string) error {
	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	return nil
}

func generarGrafoBloque(dotContent string, superbloque *estructuras.Superbloque, archivo *os.File) (string, string, error) {
	visitedBlocks := make(map[int32]bool)
	var conexiones string

	for i := int32(0); i < superbloque.S_inodes_count; i++ {
		inodo := &estructuras.Inodo{}
		err := inodo.Decode(archivo, int64(superbloque.S_inode_start+(i*superbloque.S_inode_size)))
		if err != nil {
			return "", "", fmt.Errorf("error al deserializar el inodo %d: %v", i, err)
		}

		if inodo.I_uid == -1 || inodo.I_uid == 0 {
			continue
		}

		for _, block := range inodo.I_block {
			if block != -1 {
				if !visitedBlocks[block] {
					dotContent, conexiones, err = generarEtiquetaBloque(dotContent, conexiones, block, inodo, superbloque, archivo, visitedBlocks)
					if err != nil {
						return "", "", err
					}
					visitedBlocks[block] = true
				}
			}
		}
	}
	return dotContent, conexiones, nil
}

func generarEtiquetaBloque(dotContent, conexiones string, indiceBloque int32, inodo *estructuras.Inodo, superbloque *estructuras.Superbloque, archivo *os.File, visitedBlocks map[int32]bool) (string, string, error) {
	bloqueOffset := int64(superbloque.S_block_start + (indiceBloque * superbloque.S_block_size))

	if inodo.I_type[0] == '0' {
		bloqueFolder := &estructuras.FolderBlock{}
		err := bloqueFolder.Decode(archivo, bloqueOffset)
		if err != nil {
			return "", "", fmt.Errorf("error al decodificar bloque de carpeta %d: %w", indiceBloque, err)
		}

		label := fmt.Sprintf("BLOQUE DE CARPETA %d", indiceBloque)
		hasValidConnections := false

		for i, content := range bloqueFolder.B_content {
			name := limpiarNombreBloque(content.B_name)

			name = html.EscapeString(name)

			if content.B_inodo != -1 && !(i == 0 || i == 1) {
				label += fmt.Sprintf("\\nContenido %d: %s (Inodo %d)", i+1, name, content.B_inodo)
				if content.B_inodo != indiceBloque {
					conexiones += fmt.Sprintf("block%d -> block%d [color=\"#FF7043\"];\n", indiceBloque, content.B_inodo)
				}
				hasValidConnections = true
			} else {
				if i > 1 {
					label += fmt.Sprintf("\\nContenido %d: %s (Inodo no asignado)", i+1, name)
				}
			}
		}

		if hasValidConnections {
			dotContent += fmt.Sprintf("block%d [label=\"%s\", shape=box, style=filled, fillcolor=\"#FFFDE7\", color=\"#EEEEEE\"];\n", indiceBloque, label)
		}

	} else if inodo.I_type[0] == '1' {
		bloqueFile := &estructuras.ArchivoBloque{}
		err := bloqueFile.Decode(archivo, bloqueOffset)
		if err != nil {
			return "", "", fmt.Errorf("error al decodificar bloque de archivo %d: %w", indiceBloque, err)
		}

		content := limpiarContenidoBloque(bloqueFile.GetContent())

		if len(strings.TrimSpace(content)) > 0 {
			label := fmt.Sprintf("BLOQUE DE ARCHIVO %d\\n%s", indiceBloque, content)
			dotContent += fmt.Sprintf("block%d [label=\"%s\", shape=box, style=filled, fillcolor=\"#FFFDE7\", color=\"#EEEEEE\"];\n", indiceBloque, label)

			nextBlock := encontrarSiguienteBloqueValido(inodo, indiceBloque)
			if nextBlock != -1 {
				conexiones += fmt.Sprintf("block%d -> block%d [color=\"#FF7043\"];\n", indiceBloque, nextBlock)
			}
		}
	}

	parentBlock := encontrarBloquePadre(inodo, indiceBloque)
	if parentBlock != -1 {
		conexiones += fmt.Sprintf("block%d -> block%d [color=\"#FF7043\"];\n", parentBlock, indiceBloque)
	}

	return dotContent, conexiones, nil
}

func encontrarBloquePadre(inodo *estructuras.Inodo, bloqueActual int32) int32 {
	for i := 0; i < len(inodo.I_block); i++ {
		if inodo.I_block[i] == bloqueActual && i > 0 {
			return inodo.I_block[i-1]
		}
	}
	return -1
}

func encontrarSiguienteBloqueValido(inodo *estructuras.Inodo, bloqueActual int32) int32 {
	for i := 0; i < len(inodo.I_block); i++ {
		if inodo.I_block[i] == bloqueActual {
			for j := i + 1; j < len(inodo.I_block); j++ {
				if inodo.I_block[j] != -1 {
					return inodo.I_block[j]
				}
			}
		}
	}
	return -1
}

func limpiarNombreBloque(nameArray [12]byte) string {
	return strings.TrimRight(string(nameArray[:]), "\x00")
}

func limpiarContenidoBloque(content string) string {
	return strings.ReplaceAll(content, "\n", "\\n")
}

func ejecutarGraphviz(dotFileName string, outputImage string) error {
	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error executing Graphviz: %v", err)
	}
	return nil
}
