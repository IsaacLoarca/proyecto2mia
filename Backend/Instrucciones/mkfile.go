package instrucciones

import (
	"bytes"
	"errors"
	"fmt"
	estructuras "godisk/Estructuras"
	global "godisk/Global"
	utilidades "godisk/Utilidades"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type MKFILE struct {
	path string
	r    bool
	size int
	cont string
}

func AnalizarMkfile(tokens []string) (string, error) {
	cmd := &MKFILE{}
	var outputBuffer bytes.Buffer

	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-r|-size=\d+|-cont="[^"]+"|-cont=[^\s]+`)
	matches := re.FindAllString(args, -1)

	if len(matches) != len(tokens) {
		for _, token := range tokens {
			if !re.MatchString(token) {
				return "", fmt.Errorf("parámetro inválido: %s", token)
			}
		}
	}

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		key := strings.ToLower(kv[0])
		var value string
		if len(kv) == 2 {
			value = kv[1]
		}

		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		switch key {
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-r":
			cmd.r = true
		case "-size":
			size, err := strconv.Atoi(value)
			if err != nil || size < 0 {
				return "", errors.New("el tamaño debe ser un número entero no negativo")
			}
			cmd.size = size
		case "-cont":
			if value == "" {
				return "", errors.New("el contenido no puede estar vacío")
			}
			cmd.cont = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	if cmd.size == 0 {
		cmd.size = 0
	}

	if cmd.cont == "" {
		cmd.cont = ""
	}

	err := commandMkfile(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandMkfile(mkfile *MKFILE, outputBuffer *bytes.Buffer) error {
	if !global.EstaLogueado() {
		return fmt.Errorf("no hay un usuario logueado")
	}

	idPartition := global.UsuarioActual.Id

	partitionSuperblock, mountedPartition, partitionPath, err := global.GetMountedPartitionSuperblock(idPartition)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	if mkfile.cont == "" {
		mkfile.cont = generateContent(mkfile.size)
	}

	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de partición: %w", err)
	}
	defer file.Close()

	fmt.Fprintln(outputBuffer, "======================= MKFILE =======================")
	fmt.Fprintf(outputBuffer, "Creando archivo: %s\n", mkfile.path)

	dirPath, _ := GetDirectoryAndFile(mkfile.path)

	fmt.Fprintf(outputBuffer, "Verificando la existencia del directorio: %s\n", dirPath)
	exists, _, err := directoryExists(partitionSuperblock, file, 0, dirPath)
	if err != nil {
		return fmt.Errorf("error al verificar directorio: %w", err)
	}

	if mkfile.r && !exists {
		err = CrearDirectorio(dirPath, mkfile.r, partitionSuperblock, file, mountedPartition)
		if err != nil {
			return fmt.Errorf("error al crear directorios intermedios: %w", err)
		}
	}

	err = createFile(mkfile.path, mkfile.size, mkfile.cont, partitionSuperblock, file, mountedPartition, outputBuffer, mkfile.r)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %w", err)
	}

	fmt.Fprintf(outputBuffer, "Archivo %s creado exitosamente\n", mkfile.path)
	fmt.Fprintln(outputBuffer, "==================== FIN MKFILE ==================")

	return nil
}

func generateContent(size int) string {
	content := ""
	for len(content) < size {
		content += "0124656789"
	}
	return content[:size]
}

func createFile(filePath string, size int, content string, sb *estructuras.Superbloque, file *os.File, mountedPartition *estructuras.Partition, outputBuffer *bytes.Buffer, r bool) error {
	fmt.Fprintf(outputBuffer, "Creando archivo en la ruta: %s\n", filePath)

	parentDirs, destDir := utilidades.ObtenerDirectoriosPadre(filePath)
	chunks := utilidades.DividirCadenaEnTrozos(content)
	fmt.Fprintf(outputBuffer, "Contenido generado: %v\n", chunks)

	err := sb.CrearArchivo(file, parentDirs, destDir, size, chunks, r)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %w", err)
	}

	err = sb.Codificar(file, int64(mountedPartition.Part_start))
	if err != nil {
		return fmt.Errorf("error al serializar el superbloque: %w", err)
	}

	fmt.Println("\nInodos:")
	sb.PrintInodes(file.Name())
	fmt.Println("\nBloques de datos:")
	sb.PrintBlocks(file.Name())

	return nil
}

func GetDirectoryAndFile(path string) (string, string) {
	dir := filepath.Dir(path)
	file := filepath.Base(path)
	return dir, file
}
