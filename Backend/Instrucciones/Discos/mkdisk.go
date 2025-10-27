package instrucciones

import (
	"bytes"
	"errors"
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	UnitK = "K"
	UnitM = "M"
	FitBF = "BF"
	FitFF = "FF"
	FitWF = "WF"
)

type Mkdisk struct {
	Size int
	Fit  string
	Unit string
	Path string
}

func AnalizarMkdisk(tokens []string) (string, error) {
	cmd := &Mkdisk{}
	var outputBuffer bytes.Buffer

	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-size=\d+|-unit=[kKmM]|-fit=[bBfFwW]{2}|-path="[^"]+"|-path=[^\s]+`)
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
		case "-size":
			size, err := strconv.Atoi(value)
			if err != nil || size <= 0 {
				return "", errors.New("el tamaño debe ser un número entero positivo")
			}
			cmd.Size = size
		case "-unit":
			value = strings.ToUpper(value)
			if value != UnitK && value != UnitM {
				return "", errors.New("la unidad debe ser K o M")
			}
			cmd.Unit = value
		case "-fit":
			value = strings.ToUpper(value)
			if value != FitBF && value != FitFF && value != FitWF {
				return "", errors.New("el ajuste debe ser BF, FF o WF")
			}
			cmd.Fit = value
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			if !strings.HasSuffix(value, ".mia") {
				return "", errors.New("el archivo debe tener la extensión .mia")
			}
			cmd.Path = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.Size == 0 {
		return "", errors.New("faltan parámetros requeridos: -size")
	}
	if cmd.Path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}
	if cmd.Unit == "" {
		cmd.Unit = UnitM
	}
	if cmd.Fit == "" {
		cmd.Fit = FitFF
	}

	err := ejecutarMkdisk(cmd, &outputBuffer)
	if err != nil {
		return "", fmt.Errorf("error al crear el disco: %v", err)
	}

	return outputBuffer.String(), nil
}

func ejecutarMkdisk(mkdisk *Mkdisk, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "================== MKDISK ================= ")
	fmt.Fprintf(outputBuffer, "Creando disco con tamaño: %d %s\n", mkdisk.Size, mkdisk.Unit)

	sizeBytes, err := utilidades.ConvertirABytes(mkdisk.Size, mkdisk.Unit)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error convirtiendo el tamaño:", err)
		return err
	}

	err = crearDisco(mkdisk, sizeBytes, outputBuffer)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error creando el disco:", err)
		return err
	}

	err = creacionMBR(mkdisk, sizeBytes, outputBuffer)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error creando el MBR:", err)
		return err
	}

	fmt.Fprintln(outputBuffer, "================= FIN MKDISK ================= ")
	return nil
}

func crearDisco(mkdisk *Mkdisk, sizeBytes int, outputBuffer *bytes.Buffer) error {
	err := os.MkdirAll(filepath.Dir(mkdisk.Path), os.ModePerm)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error creando directorios:", err)
		return err
	}

	file, err := os.Create(mkdisk.Path)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error creando archivo:", err)
		return err
	}
	defer file.Close()

	buffer := make([]byte, 1024*1024)
	for sizeBytes > 0 {
		writeSize := len(buffer)
		if sizeBytes < writeSize {
			writeSize = sizeBytes
		}
		if _, err := file.Write(buffer[:writeSize]); err != nil {
			return err
		}
		sizeBytes -= writeSize
	}
	fmt.Fprintln(outputBuffer, "Disco creado exitosamente:", mkdisk.Path)
	return nil
}

func creacionMBR(mkdisk *Mkdisk, sizeBytes int, outputBuffer *bytes.Buffer) error {
	file, err := os.OpenFile(mkdisk.Path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error abriendo el archivo:", err)
		return err
	}
	defer file.Close()

	mbr := &estructuras.Mbr{
		Mbr_tamano:         int32(sizeBytes),
		Mbr_fecha_creacion: float32(time.Now().Unix()),
		Mbr_dsk_signature:  rand.Int31(),
		Dsk_fit:            [1]byte{mkdisk.Fit[0]},
		Mbr_partitions: [4]estructuras.Partition{
			{Part_status: [1]byte{'9'}, Part_type: [1]byte{'0'}, Part_fit: [1]byte{'0'}, Part_start: -1, Part_s: -1, Part_name: [16]byte{'0'}, Part_correlative: -1, Part_id: [4]byte{'0'}},
			{Part_status: [1]byte{'9'}, Part_type: [1]byte{'0'}, Part_fit: [1]byte{'0'}, Part_start: -1, Part_s: -1, Part_name: [16]byte{'0'}, Part_correlative: -1, Part_id: [4]byte{'0'}},
			{Part_status: [1]byte{'9'}, Part_type: [1]byte{'0'}, Part_fit: [1]byte{'0'}, Part_start: -1, Part_s: -1, Part_name: [16]byte{'0'}, Part_correlative: -1, Part_id: [4]byte{'0'}},
			{Part_status: [1]byte{'9'}, Part_type: [1]byte{'0'}, Part_fit: [1]byte{'0'}, Part_start: -1, Part_s: -1, Part_name: [16]byte{'0'}, Part_correlative: -1, Part_id: [4]byte{'0'}},
		},
	}
	err = mbr.Codificar(file)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error serializando el MBR en el archivo:", err)
		return err
	}

	fmt.Fprintln(outputBuffer, "MBR creado exitosamente en el disco.")
	mbr.Imprimir()
	fmt.Println("===========================================================")

	return nil
}
