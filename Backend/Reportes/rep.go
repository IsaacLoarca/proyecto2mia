package reportes

import (
	"bytes"
	"errors"
	"fmt"
	global "godisk/Global"
	"os"
	"regexp"
	"strings"
)

type REP struct {
	id           string
	path         string
	name         string
	path_file_ls string
}

func AnalizarRep(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer

	cmd := &REP{}
	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-id=[^\s]+|-path="[^"]+"|-path=[^\s]+|-name=[^\s]+|-path_file_ls="[^"]+"|-path_file_ls=[^\s]+`)
	matches := re.FindAllString(args, -1)

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}
		key, value := strings.ToLower(kv[0]), kv[1]
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		switch key {
		case "-id":
			if value == "" {
				return "", errors.New("el id no puede estar vacío")
			}
			cmd.id = value
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-name":
			validNames := []string{"mbr", "disk", "inode", "block", "bm_inode", "bm_block", "sb", "file", "ls", "tree"}
			if !contains(validNames, value) {
				return "", errors.New("nombre inválido, debe ser uno de los siguientes: mbr, disk, inode, block, bm_inode, bm_block, sb, file, ls")
			}
			cmd.name = value
		case "-path_file_ls":
			cmd.path_file_ls = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.id == "" || cmd.path == "" || cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -id, -path, -name")
	}

	err := commandRep(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func commandRep(rep *REP, outputBuffer *bytes.Buffer) error {
	mountedMbr, mountedSb, mountedDiskPath, err := global.GetMountedPartitionRep(rep.id)
	if err != nil {
		return err
	}

	file, err := os.Open(mountedDiskPath)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer file.Close()

	fmt.Fprintf(outputBuffer, "Generando reporte '%s'...\n", rep.name)
	fmt.Printf("Generando reporte '%s'...\n", rep.name)

	switch rep.name {
	case "mbr":
		err = ReporteMBR(mountedMbr, rep.path, file)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte MBR: %v\n", err)
			fmt.Printf("Error generando reporte MBR: %v\n", err)
			return err
		}
	case "disk":
		err = ReporteDisk(mountedMbr, rep.path, mountedDiskPath)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte del disco: %v\n", err)
			fmt.Printf("Error generando reporte del disco: %v\n", err)
			return err
		}
	case "inode":
		err = ReporteInodo(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte de inodos: %v\n", err)
			fmt.Printf("Error generando reporte de inodos: %v\n", err)
			return err
		}
	case "block":
		err = ReporteBloque(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte de bloques: %v\n", err)
			fmt.Printf("Error generando reporte de bloques: %v\n", err)
			return err
		}
	case "bm_inode":
		err = ReporteBMInodo(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte de bitmap de inodos: %v\n", err)
			fmt.Printf("Error generando reporte de bitmap de inodos: %v\n", err)
			return err
		}
	case "bm_block":
		err = ReporteBMBloque(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte de bitmap de bloques: %v\n", err)
			fmt.Printf("Error generando reporte de bitmap de bloques: %v\n", err)
		}
	case "sb":
		err = ReporteSb(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte del superbloque: %v\n", err)
			fmt.Printf("Error generando reporte del superbloque: %v\n", err)
			return err
		}
	case "file":
		err = ReporteFile(mountedSb, mountedDiskPath, rep.path, rep.path_file_ls)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte de archivo: %v\n", err)
			fmt.Printf("Error generando reporte de archivo: %v\n", err)
			return err
		}
	case "tree":
		err = ReporteTree(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte de árbol: %v\n", err)
			fmt.Printf("Error generando reporte de árbol: %v\n", err)
			return err
		}
	case "ls":
		err = ReporteLS(mountedSb, mountedDiskPath, rep.path, rep.path_file_ls)
		if err != nil {
			fmt.Fprintf(outputBuffer, "Error generando reporte LS: %v\n", err)
			fmt.Printf("Error generando reporte LS: %v\n", err)
			return err
		}
	default:
		return fmt.Errorf("tipo de reporte no soportado: %s", rep.name)
	}

	fmt.Fprintf(outputBuffer, "Reporte '%s' generado exitosamente en la ruta: %s\n", rep.name, rep.path)
	fmt.Printf("Reporte '%s' generado exitosamente en la ruta: %s\n", rep.name, rep.path)

	return nil
}
