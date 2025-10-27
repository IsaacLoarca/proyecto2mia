package reportes

import (
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"os"
	"os/exec"
	"time"
)

func ReporteInodo(superbloque *estructuras.Superbloque, rutaDisco string, ruta string) error {
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

	if superbloque.S_inodes_count == 0 {
		return fmt.Errorf("no hay inodos en el sistema")
	}

	dotContent, err = generarGrafoInodo(dotContent, superbloque, archivo)
	if err != nil {
		return err
	}

	dotContent += "}"

	err = escribirDotFile(dotFileName, dotContent)
	if err != nil {
		return err
	}

	err = generarImagenInodo(dotFileName, outputImage)
	if err != nil {
		return err
	}

	fmt.Println("Imagen de los inodos generada:", outputImage)
	return nil
}

func iniciarDotGraph() string {
	return `digraph G {
		fontname="Helvetica,Arial,sans-serif"
		node [fontname="Helvetica,Arial,sans-serif", shape=plain, fontsize=12];
		edge [fontname="Helvetica,Arial,sans-serif", color="#FF7043", arrowsize=0.8];
		rankdir=LR;
		bgcolor="#FAFAFA";
		node [shape=plaintext];
		inodeHeaderColor="#4CAF50"; 
		blockHeaderColor="#FF9800"; 
		cellBackgroundColor="#FFFDE7";
		cellBorderColor="#EEEEEE";
		textColor="#263238";
	`
}

func generarGrafoInodo(dotContent string, superbloque *estructuras.Superbloque, archivo *os.File) (string, error) {
	for i := int32(0); i < superbloque.S_inodes_count; i++ {
		inodo := &estructuras.Inodo{}
		err := inodo.Decode(archivo, int64(superbloque.S_inode_start+(i*superbloque.S_inode_size)))
		if err != nil {
			return "", fmt.Errorf("error al deserializar el inodo %d: %v", i, err)
		}

		if inodo.I_uid == -1 || inodo.I_uid == 0 {
			continue
		}

		dotContent += generarTablaInodo(i, inodo)

		if i < superbloque.S_inodes_count-1 {
			dotContent += fmt.Sprintf("inodo%d -> inodo%d [color=\"#FF7043\"];\n", i, i+1)
		}
	}
	return dotContent, nil
}

func generarTablaInodo(indiceInodo int32, inodo *estructuras.Inodo) string {
	atime := time.Unix(int64(inodo.I_atime), 0).Format(time.RFC3339)
	ctime := time.Unix(int64(inodo.I_ctime), 0).Format(time.RFC3339)
	mtime := time.Unix(int64(inodo.I_mtime), 0).Format(time.RFC3339)

	table := fmt.Sprintf(`inodo%d [label=<
		<table border="0" cellborder="1" cellspacing="0" cellpadding="4" bgcolor="#FFFDE7" style="rounded">
			<tr><td colspan="2" bgcolor="#4CAF50" align="center"><b>INODO %d</b></td></tr>
			<tr><td><b>i_uid</b></td><td>%d</td></tr>
			<tr><td><b>i_gid</b></td><td>%d</td></tr>
			<tr><td><b>i_size</b></td><td>%d</td></tr>
			<tr><td><b>i_atime</b></td><td>%s</td></tr>
			<tr><td><b>i_ctime</b></td><td>%s</td></tr>
			<tr><td><b>i_mtime</b></td><td>%s</td></tr>
			<tr><td><b>i_type</b></td><td>%c</td></tr>
			<tr><td><b>i_perm</b></td><td>%s</td></tr>
			<tr><td colspan="2" bgcolor="#FF9800"><b>BLOQUES DIRECTOS</b></td></tr>
	`, indiceInodo, indiceInodo, inodo.I_uid, inodo.I_gid, inodo.I_size, atime, ctime, mtime, rune(inodo.I_type[0]), string(inodo.I_perm[:]))

	for j, block := range inodo.I_block[:12] {
		if block != -1 {
			table += fmt.Sprintf("<tr><td><b>%d</b></td><td>%d</td></tr>", j+1, block)
		}
	}

	table += generarBloquesIndirectos(inodo)

	table += "</table>>];"
	return table
}

func generarBloquesIndirectos(inodo *estructuras.Inodo) string {
	result := ""
	if inodo.I_block[12] != -1 {
		result += fmt.Sprintf(`
			<tr><td colspan="2" bgcolor="#FF9800"><b>BLOQUE INDIRECTO SIMPLE</b></td></tr>
			<tr><td><b>13</b></td><td>%d</td></tr>
		`, inodo.I_block[12])
	}

	if inodo.I_block[13] != -1 {
		result += fmt.Sprintf(`
			<tr><td colspan="2" bgcolor="#FF9800"><b>BLOQUE INDIRECTO DOBLE</b></td></tr>
			<tr><td><b>14</b></td><td>%d</td></tr>
		`, inodo.I_block[13])
	}

	if inodo.I_block[14] != -1 {
		result += fmt.Sprintf(`
			<tr><td colspan="2" bgcolor="#FF9800"><b>BLOQUE INDIRECTO TRIPLE</b></td></tr>
			<tr><td><b>15</b></td><td>%d</td></tr>
		`, inodo.I_block[14])
	}

	return result
}

func escribirDotFile(dotFileName string, dotContent string) error {
	dotFile, err := os.Create(dotFileName)
	if err != nil {
		return fmt.Errorf("error al crear el archivo DOT: %v", err)
	}
	defer dotFile.Close()

	_, err = dotFile.WriteString(dotContent)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo DOT: %v", err)
	}

	return nil
}

func generarImagenInodo(dotFileName string, outputImage string) error {
	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	return nil
}
