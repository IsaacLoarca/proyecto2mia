package reportes

import (
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"os/exec"
	"time"
)

func ReporteSb(superbloque *estructuras.Superbloque, rutaDisco string, ruta string) error {
	err := utilidades.CrearDirectoriosPadre(ruta)
	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}

	dotFileName, outputImage := utilidades.ObtenerNombresArchivo(ruta)

	dotContent := iniciarGrafoDotParaSuperbloque(superbloque)

	err = escribirDotFile(dotFileName, dotContent)
	if err != nil {
		return err
	}

	err = generarImagenSuperbloque(dotFileName, outputImage)
	if err != nil {
		return err
	}

	fmt.Println("Imagen del Superbloque generada:", outputImage)

	return nil
}

func generarImagenSuperbloque(dotFileName string, outputImage string) error {
	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz para generar la imagen del Superbloque: %v", err)
	}

	return nil
}

func iniciarGrafoDotParaSuperbloque(superbloque *estructuras.Superbloque) string {
	mtime := time.Unix(int64(superbloque.S_mtime), 0).Format(time.RFC3339)
	umtime := time.Unix(int64(superbloque.S_umtime), 0).Format(time.RFC3339)

	dotContent := `
		digraph G {
			fontname="Helvetica,Arial,sans-serif"
			node [fontname="Helvetica,Arial,sans-serif", shape=plain, fontsize=12];
			edge [fontname="Helvetica,Arial,sans-serif", color="#FF7043", arrowsize=0.8];
			bgcolor="#FAFAFA";
			rankdir=TB;

			superblockTable [label=<
				<table border="0" cellborder="1" cellspacing="0" cellpadding="10" bgcolor="#FFF9C4" style="rounded">
					<tr><td colspan="2" bgcolor="#4CAF50" align="center"><b>REPORTE DEL SUPERBLOQUE</b></td></tr>
					<tr><td><b>Cantidad de Inodos</b></td><td>%d</td></tr>
					<tr><td><b>Cantidad de Bloques</b></td><td>%d</td></tr>
					<tr><td><b>Inodos Libres</b></td><td>%d</td></tr>
					<tr><td><b>Bloques Libres</b></td><td>%d</td></tr>
					<tr><td><b>Tamaño de Inodo</b></td><td>%d bytes</td></tr>
					<tr><td><b>Tamaño de Bloque</b></td><td>%d bytes</td></tr>
					<tr><td><b>Primer Inodo Libre</b></td><td>%d</td></tr>
					<tr><td><b>Primer Bloque Libre</b></td><td>%d</td></tr>
					<tr><td><b>Inicio Bitmap de Inodos</b></td><td>%d</td></tr>
					<tr><td><b>Inicio Bitmap de Bloques</b></td><td>%d</td></tr>
					<tr><td><b>Última Modificación</b></td><td>%s</td></tr>
					<tr><td><b>Último Montaje</b></td><td>%s</td></tr>
				</table>>];
		}
	`

	dotContent = fmt.Sprintf(dotContent,
		superbloque.S_inodes_count,
		superbloque.S_blocks_count,
		superbloque.S_free_inodes_count,
		superbloque.S_free_blocks_count,
		superbloque.S_inode_size,
		superbloque.S_block_size,
		superbloque.S_first_ino,
		superbloque.S_first_blo,
		superbloque.S_bm_inode_start,
		superbloque.S_bm_block_start,
		mtime,
		umtime,
	)

	return dotContent
}
