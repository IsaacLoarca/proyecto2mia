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

func ReporteTree(superbloque *estructuras.Superbloque, rutaDisco string, ruta string) error {
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

	dotContent := iniciarDotTreeGraph()

	dotContent, err = generarGrafoTree(dotContent, superbloque, archivo)
	if err != nil {
		return err
	}

	dotContent += "}"

	err = escribirArchivoTexto(dotFileName, dotContent)
	if err != nil {
		return err
	}

	err = generarImagenTree(dotFileName, outputImage)
	if err != nil {
		return err
	}

	fmt.Println("Imagen del árbol EXT2 generada:", outputImage)
	return nil
}

func iniciarDotTreeGraph() string {
	return `digraph EXT2Tree {
		fontname="Helvetica,Arial,sans-serif"
		node [fontname="Helvetica,Arial,sans-serif", shape=plain, fontsize=12];
		edge [fontname="Helvetica,Arial,sans-serif", color="#4676D2", arrowsize=0.8];
		rankdir=TB;
		bgcolor="#FAFAFA";
		node [shape=plaintext];
		inodeHeaderColor="#388E3C";
		blockHeaderColor="#FBC02D";
		cellBackgroundColor="#FFFDE7";
		cellBorderColor="#EEEEEE";
		textColor="#263238";
	`
}

func generarGrafoTree(dotContent string, superbloque *estructuras.Superbloque, archivo *os.File) (string, error) {
	conexiones := ""
	inodos := make(map[int32]*estructuras.Inodo)
	for i := int32(0); i < superbloque.S_inodes_count; i++ {
		inodo := &estructuras.Inodo{}
		err := inodo.Decode(archivo, int64(superbloque.S_inode_start+(i*superbloque.S_inode_size)))
		if err == nil && inodo.I_uid != -1 && inodo.I_uid != 0 {
			inodos[i] = inodo
			dotContent += generarTablaInodoTree(i, inodo)
		}
	}
	for i, inodo := range inodos {
		for _, block := range inodo.I_block {
			if block != -1 {
				dotContent += generarNodoBloqueTree(block, inodo, superbloque, archivo)
				conexiones += fmt.Sprintf("inodo%d -> block%d [color=\"#4676D2\"]\n", i, block)
				if inodo.I_type[0] == '0' {
					bloqueOffset := int64(superbloque.S_block_start + (block * superbloque.S_block_size))
					bloqueFolder := &estructuras.FolderBlock{}
					err := bloqueFolder.Decode(archivo, bloqueOffset)
					if err == nil {
						for j, content := range bloqueFolder.B_content {
							if j > 1 && content.B_inodo != -1 {
								conexiones += fmt.Sprintf("block%d -> inodo%d [color=\"#388E3C\"]\n", block, content.B_inodo)
							}
						}
					}
				}
			}
		}
	}
	for i, inodo := range inodos {
		if inodo.I_type[0] == '0' {
			for _, block := range inodo.I_block {
				if block != -1 {
					bloqueOffset := int64(superbloque.S_block_start + (block * superbloque.S_block_size))
					bloqueFolder := &estructuras.FolderBlock{}
					err := bloqueFolder.Decode(archivo, bloqueOffset)
					if err == nil {
						for j, content := range bloqueFolder.B_content {
							// Se evitan . y ..
							if j > 1 && content.B_inodo != -1 {
								conexiones += fmt.Sprintf("inodo%d -> inodo%d [style=dashed, color=\"#FBC02D\", arrowhead=none]\n", i, content.B_inodo)
							}
						}
					}
				}
			}
		}
	}
	dotContent += conexiones
	return dotContent, nil
}

func generarTablaInodoTree(indiceInodo int32, inodo *estructuras.Inodo) string {
	atime := time.Unix(int64(inodo.I_atime), 0).Format(time.RFC3339)
	ctime := time.Unix(int64(inodo.I_ctime), 0).Format(time.RFC3339)
	mtime := time.Unix(int64(inodo.I_mtime), 0).Format(time.RFC3339)
	table := fmt.Sprintf("inodo%d [label=<\n        <table border='0' cellborder='1' cellspacing='0' cellpadding='4' bgcolor='#FFFDE7' style='rounded'>\n            <tr><td colspan='2' bgcolor='#388E3C' align='center'><b>INODO %d</b></td></tr>\n            <tr><td><b>i_uid</b></td><td>%d</td></tr>\n            <tr><td><b>i_gid</b></td><td>%d</td></tr>\n            <tr><td><b>i_size</b></td><td>%d</td></tr>\n            <tr><td><b>i_atime</b></td><td>%s</td></tr>\n            <tr><td><b>i_ctime</b></td><td>%s</td></tr>\n            <tr><td><b>i_mtime</b></td><td>%s</td></tr>\n            <tr><td><b>i_type</b></td><td>%c</td></tr>\n            <tr><td><b>i_perm</b></td><td>%s</td></tr>\n        </table>>];\n",
		indiceInodo, indiceInodo, inodo.I_uid, inodo.I_gid, inodo.I_size, atime, ctime, mtime, rune(inodo.I_type[0]), string(inodo.I_perm[:]))
	return table
}

func generarNodoBloqueTree(indiceBloque int32, inodo *estructuras.Inodo, superbloque *estructuras.Superbloque, archivo *os.File) string {
	bloqueOffset := int64(superbloque.S_block_start + (indiceBloque * superbloque.S_block_size))
	var dot string
	if inodo.I_type[0] == '0' {
		bloqueFolder := &estructuras.FolderBlock{}
		err := bloqueFolder.Decode(archivo, bloqueOffset)
		if err != nil {
			return ""
		}
		label := fmt.Sprintf("BLOQUE DE CARPETA %d", indiceBloque)
		for i, content := range bloqueFolder.B_content {
			name := limpiarNombreBloqueTree(content.B_name)
			if content.B_inodo != -1 {
				label += fmt.Sprintf("\\nContenido %d: %s (Inodo %d)", i+1, name, content.B_inodo)
			} else {
				label += fmt.Sprintf("\\nContenido %d: %s (Sin inodo)", i+1, name)
			}
		}
		label = reemplazarNuevasLineas(label)
		dot += fmt.Sprintf("block%d [label=\"%s\", shape=box, style=filled, fillcolor=\"#FFFDE7\", color=\"#EEEEEE\"]\n", indiceBloque, label)
	} else if inodo.I_type[0] == '1' {
		bloqueFile := &estructuras.ArchivoBloque{}
		err := bloqueFile.Decode(archivo, bloqueOffset)
		if err != nil {
			return ""
		}
		content := limpiarContenidoBloqueTree(bloqueFile.GetContent())
		if len(content) > 0 {
			content = reemplazarNuevasLineas(content)
			label := fmt.Sprintf("BLOQUE DE ARCHIVO %d\\n%s", indiceBloque, content)
			dot += fmt.Sprintf("block%d [label=\"%s\", shape=box, style=filled, fillcolor=\"#FFFDE7\", color=\"#EEEEEE\"]\n", indiceBloque, label)
		}
	}
	return dot
}

func reemplazarNuevasLineas(s string) string {
	return strings.ReplaceAll(s, "\n", "\\n")
}

func limpiarNombreBloqueTree(nameArray [12]byte) string {
	name := string(nameArray[:])
	for i := range name {
		if name[i] == '\x00' {
			return name[:i]
		}
	}
	return name
}

func limpiarContenidoBloqueTree(content string) string {
	return content
}

func generarImagenTree(dotFileName string, outputImage string) error {
	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}
	return nil
}

func escribirArchivoTexto(nombreArchivo string, contenido string) error {
	archivo, err := os.Create(nombreArchivo)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %v", err)
	}
	defer archivo.Close()

	_, err = archivo.WriteString(contenido)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo: %v", err)
	}

	return nil
}
