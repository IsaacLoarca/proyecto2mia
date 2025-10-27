package instrucciones

import (
	"bytes"
	"errors"
	"fmt"
	estructuras "godisk/Estructuras"
	utilidades "godisk/Utilidades"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Fdisk struct {
	size   int
	unit   string
	path   string
	typpe  string
	fit    string
	name   string
	add    int
	delete string
}

func AnalizarFdisk(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer
	cmd := &Fdisk{}

	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-size=\d+|-unit=[bBkKmM]|-fit=[bBfFwfW]{2}|-path="[^"]+"|-path=[^\s]+|-type=[pPeElL]|-name="[^"]+"|-name=[^\s]+|-add=[+-]?\d+|-delete=(fast|full)`)
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
			cmd.size = size
		case "-unit":
			value = strings.ToUpper(value)
			if value != "B" && value != "K" && value != "M" {
				return "", errors.New("la unidad debe ser B, K o M")
			}
			cmd.unit = value
		case "-fit":
			value = strings.ToUpper(value)
			if value != "BF" && value != "FF" && value != "WF" {
				return "", errors.New("el ajuste debe ser BF, FF o WF")
			}
			cmd.fit = value
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-type":
			value = strings.ToUpper(value)
			if value != "P" && value != "E" && value != "L" {
				return "", errors.New("el tipo debe ser P, E o L")
			}
			cmd.typpe = value
		case "-name":
			if value == "" {
				return "", errors.New("el nombre no puede estar vacío")
			}
			cmd.name = value
		case "-add":
			add, err := strconv.Atoi(value)
			if err != nil {
				return "", errors.New("el valor de -add debe ser un número entero")
			}
			cmd.add = add
		case "-delete":
			value = strings.ToLower(value)
			if value != "fast" && value != "full" {
				return "", errors.New("el valor de -delete debe ser 'fast' o 'full'")
			}
			cmd.delete = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.delete != "" {
		if cmd.path == "" {
			return "", errors.New("falta el parámetro requerido: -path")
		}
		if cmd.name == "" {
			return "", errors.New("falta el parámetro requerido: -name")
		}
		return processDeletePartition(cmd, &outputBuffer)
	}

	if cmd.add != 0 {
		if cmd.path == "" {
			return "", errors.New("falta el parámetro requerido: -path")
		}
		if cmd.name == "" {
			return "", errors.New("falta el parámetro requerido: -name")
		}
		return processAddPartition(cmd, &outputBuffer)
	}

	if cmd.size == 0 {
		return "", errors.New("faltan parámetros requeridos: -size")
	}
	if cmd.path == "" {
		return "", errors.New("falta el parámetro requerido: -path")
	}
	if cmd.name == "" {
		return "", errors.New("falta el parámetro requerido: -name")
	}

	if cmd.unit == "" {
		cmd.unit = "K"
	}

	if cmd.fit == "" {
		cmd.fit = "WF"
	}

	if cmd.typpe == "" {
		cmd.typpe = "P"
	}

	err := commandFdisk(cmd, &outputBuffer)
	if err != nil {
		return "", fmt.Errorf("error al crear la partición: %v", err)
	}

	return outputBuffer.String(), nil
}

func processDeletePartition(cmd *Fdisk, outputBuffer *bytes.Buffer) (string, error) {
	fmt.Fprintf(outputBuffer, "========================== DELETE ==========================\n")
	fmt.Fprintf(outputBuffer, "Eliminando partición con nombre '%s' usando el método %s...\n", cmd.name, cmd.delete)

	file, err := os.OpenFile(cmd.path, os.O_RDWR, 0644)
	if err != nil {
		return "", fmt.Errorf("error abriendo el archivo del disco: %v", err)
	}
	defer file.Close()

	var mbr estructuras.Mbr
	err = mbr.Decodificar(file)
	if err != nil {
		return "", fmt.Errorf("error al deserializar el MBR: %v", err)
	}

	partition, _ := mbr.ObtenerParticionPorNombre(cmd.name)
	if partition == nil {
		return "", fmt.Errorf("la partición '%s' no existe", cmd.name)
	}

	isExtended := partition.Part_type[0] == 'E'
	err = partition.Delete(cmd.delete, file, isExtended)
	if err != nil {
		return "", fmt.Errorf("error al eliminar la partición: %v", err)
	}

	for i := range mbr.Mbr_partitions {
		nombreParticion := mbr.Mbr_partitions[i].Part_name[:]
		nombreLimpio := string(bytes.TrimRight(nombreParticion, "\x00"))
		if nombreLimpio == cmd.name {
			// Marcar la partición como eliminada estableciendo Part_start en -1
			mbr.Mbr_partitions[i].Part_status[0] = '0'
			mbr.Mbr_partitions[i].Part_type = [1]byte{0}
			mbr.Mbr_partitions[i].Part_fit = [1]byte{0}
			mbr.Mbr_partitions[i].Part_start = -1 // Cambio crítico: -1 indica partición vacía
			mbr.Mbr_partitions[i].Part_s = 0
			mbr.Mbr_partitions[i].Part_name = [16]byte{}
			mbr.Mbr_partitions[i].Part_correlative = 0
			mbr.Mbr_partitions[i].Part_id = [4]byte{}
			break
		}
	}

	err = mbr.Codificar(file)
	if err != nil {
		return "", fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Partición '%s' eliminada exitosamente.\n", cmd.name)
	fmt.Fprintf(outputBuffer, "===========================================================\n")
	fmt.Fprintf(outputBuffer, "========================== PARTICIONES ==========================\n")
	printPartitions(&mbr, outputBuffer)
	fmt.Fprintf(outputBuffer, "===========================================================\n")

	return outputBuffer.String(), nil
}

func processAddPartition(cmd *Fdisk, outputBuffer *bytes.Buffer) (string, error) {
	fmt.Fprintf(outputBuffer, "========================== ADD ==========================\n")
	fmt.Fprintf(outputBuffer, "Modificando partición '%s', ajustando %d unidades...\n", cmd.name, cmd.add)

	file, err := os.OpenFile(cmd.path, os.O_RDWR, 0644)
	if err != nil {
		return "", fmt.Errorf("error abriendo el archivo del disco: %v", err)
	}
	defer file.Close()

	var mbr estructuras.Mbr
	err = mbr.Decodificar(file)
	if err != nil {
		return "", fmt.Errorf("error al deserializar el MBR: %v", err)
	}

	partition, _ := mbr.ObtenerParticionPorNombre(cmd.name)
	if partition == nil {
		return "", fmt.Errorf("la partición '%s' no existe", cmd.name)
	}

	addBytes, err := utilidades.ConvertirABytes(cmd.add, cmd.unit)
	if err != nil {
		return "", fmt.Errorf("error al convertir las unidades de -add: %v", err)
	}

	var availableSpace int32 = 0
	if addBytes > 0 {
		availableSpace, err = mbr.CalcularEspacioDisponibleParaParticion(partition)
		if err != nil {
			return "", fmt.Errorf("error al calcular el espacio disponible para la partición '%s': %v", cmd.name, err)
		}
	}

	err = partition.ModifySize(int32(addBytes), availableSpace)
	if err != nil {
		return "", fmt.Errorf("error al modificar el tamaño de la partición: %v", err)
	}

	err = mbr.Codificar(file)
	if err != nil {
		return "", fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
	}

	fmt.Fprintf(outputBuffer, "Espacio en la partición '%s' modificado exitosamente.\n", cmd.name)
	fmt.Fprintf(outputBuffer, "===========================================================\n")

	fmt.Fprintf(outputBuffer, "========================== PARTICIONES ==========================\n")
	printPartitions(&mbr, outputBuffer)
	fmt.Fprintf(outputBuffer, "===========================================================\n")

	return outputBuffer.String(), nil
}

func printPartitions(mbr *estructuras.Mbr, outputBuffer *bytes.Buffer) {
	for i, partition := range mbr.Mbr_partitions {
		if partition.Part_start != -1 && partition.Part_s > 0 {
			// Partición activa con datos válidos
			partitionName := strings.TrimSpace(string(bytes.TrimRight(partition.Part_name[:], "\x00")))
			partitionType := partition.Part_type[0]
			partitionStatus := partition.Part_status[0]

			// Solo mostrar si tiene nombre válido
			if partitionName != "" {
				fmt.Fprintf(outputBuffer, "Partición %d: Nombre: %s | Inicio: %d | Tamaño: %d bytes | Tipo: %c | Estado: %c\n",
					i+1,
					partitionName,
					partition.Part_start,
					partition.Part_s,
					partitionType,
					partitionStatus,
				)
			} else {
				fmt.Fprintf(outputBuffer, "Partición %d: (Vacía)\n", i+1)
			}
		} else {
			fmt.Fprintf(outputBuffer, "Partición %d: (Vacía)\n", i+1)
		}
	}
}

func commandFdisk(fdisk *Fdisk, outputBuffer *bytes.Buffer) error {
	fmt.Fprintf(outputBuffer, "========================== FDISK ==========================\n")
	fmt.Fprintf(outputBuffer, "Creando partición con nombre '%s' y tamaño %d %s...\n", fdisk.name, fdisk.size, fdisk.unit)
	fmt.Println("Detalles internos de la creación de partición:", fdisk.size, fdisk.unit, fdisk.fit, fdisk.path, fdisk.typpe, fdisk.name)

	file, err := os.OpenFile(fdisk.path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo el archivo del disco: %v", err)
	}
	defer file.Close()

	sizeBytes, err := utilidades.ConvertirABytes(fdisk.size, fdisk.unit)
	if err != nil {
		fmt.Println("Error converting size:", err)
		return err
	}

	if fdisk.typpe == "P" {
		err = createPrimaryPartition(file, fdisk, sizeBytes, outputBuffer)
		if err != nil {
			fmt.Println("Error creando partición primaria:", err)
			return err
		}
	} else if fdisk.typpe == "E" {
		fmt.Println("Creando partición extendida...")
		err = createExtendedPartition(file, fdisk, sizeBytes, outputBuffer)
		if err != nil {
			fmt.Println("Error creando partición extendida:", err)
			return err
		}
	} else if fdisk.typpe == "L" {
		fmt.Println("Creando partición lógica...")
		err = createLogicalPartition(file, fdisk, sizeBytes, outputBuffer)
		if err != nil {
			fmt.Println("Error creando partición lógica:", err)
			return err
		}
	}

	fmt.Fprintln(outputBuffer, "Partición creada exitosamente.")
	fmt.Fprintln(outputBuffer, "===========================================================")
	return nil
}

func createPrimaryPartition(file *os.File, fdisk *Fdisk, sizeBytes int, outputBuffer *bytes.Buffer) error {
	fmt.Fprintf(outputBuffer, "Creando partición primaria con tamaño %d %s...\n", fdisk.size, fdisk.unit)

	var mbr estructuras.Mbr
	err := mbr.Decodificar(file)
	if err != nil {
		return fmt.Errorf("error al deserializar el MBR: %v", err)
	}
	availableSpace, err := mbr.CalcularEspacioDisponible()
	if err != nil {
		fmt.Println("Error calculando el espacio disponible:", err)
	} else {
		fmt.Println("Espacio disponible en el disco:", availableSpace)
	}
	err = mbr.CreatePartitionWithFit(int32(sizeBytes), fdisk.typpe, fdisk.name)
	if err != nil {
		return fmt.Errorf("error al crear la partición primaria: %v", err)
	}

	err = mbr.Codificar(file)
	if err != nil {
		return fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
	}

	fmt.Fprintln(outputBuffer, "Partición primaria creada exitosamente.")
	return nil
}

func createExtendedPartition(file *os.File, fdisk *Fdisk, sizeBytes int, outputBuffer *bytes.Buffer) error {
	fmt.Fprintf(outputBuffer, "Creando partición extendida con tamaño %d %s...\n", fdisk.size, fdisk.unit)

	var mbr estructuras.Mbr
	err := mbr.Decodificar(file)
	if err != nil {
		return fmt.Errorf("error al deserializar el MBR: %v", err)
	}

	if mbr.TieneParticionExtendida() {
		return errors.New("ya existe una partición extendida en este disco")
	}

	err = mbr.CreatePartitionWithFit(int32(sizeBytes), "E", fdisk.name)
	if err != nil {
		return fmt.Errorf("error al crear la partición extendida: %v", err)
	}

	extendedPartition, _ := mbr.ObtenerParticionPorNombre(fdisk.name)
	err = estructuras.CrearYEscribirEBR(extendedPartition.Part_start, 0, fdisk.fit[0], fdisk.name, file)
	if err != nil {
		return fmt.Errorf("error al crear el primer EBR en la partición extendida: %v", err)
	}

	err = mbr.Codificar(file)
	if err != nil {
		return fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
	}

	fmt.Fprintln(outputBuffer, "Partición extendida creada exitosamente.")
	return nil
}

func createLogicalPartition(file *os.File, fdisk *Fdisk, sizeBytes int, outputBuffer *bytes.Buffer) error {
	fmt.Fprintf(outputBuffer, "Creando partición lógica con tamaño %d %s...\n", fdisk.size, fdisk.unit)

	var mbr estructuras.Mbr
	err := mbr.Decodificar(file)
	if err != nil {
		return fmt.Errorf("error al deserializar el MBR: %v", err)
	}

	if !mbr.TieneParticionExtendida() {
		return errors.New("no se encontró una partición extendida en el disco")
	}

	var extendedPartition *estructuras.Partition
	for i := range mbr.Mbr_partitions {
		if mbr.Mbr_partitions[i].Part_type[0] == 'E' {
			extendedPartition = &mbr.Mbr_partitions[i]
			break
		}
	}

	lastEBR, err := estructuras.EncontrarUltimoEBR(extendedPartition.Part_start, file)
	if err != nil {
		return fmt.Errorf("error al buscar el último EBR: %v", err)
	}

	if lastEBR.Part_s == 0 {
		fmt.Println("Detectado EBR inicial vacío, asignando tamaño a la nueva partición lógica.")
		lastEBR.Part_s = int32(sizeBytes)
		copy(lastEBR.Part_name[:], fdisk.name)

		err = lastEBR.Codificar(file, int64(lastEBR.Part_start))
		if err != nil {
			return fmt.Errorf("error al escribir el primer EBR con la nueva partición lógica: %v", err)
		}

		fmt.Fprintln(outputBuffer, "Primera partición lógica creada exitosamente.")
		return nil
	}

	newEBRStart, err := lastEBR.CalcularInicioSiguienteEBR(extendedPartition.Part_start, extendedPartition.Part_s)
	if err != nil {
		return fmt.Errorf("error calculando el inicio del nuevo EBR: %v", err)
	}

	availableSize := extendedPartition.Part_s - (newEBRStart - extendedPartition.Part_start)
	if availableSize < int32(sizeBytes) {
		return errors.New("no hay suficiente espacio en la partición extendida para una nueva partición lógica")
	}

	newEBR := estructuras.Ebr{}
	newEBR.EstablecerEBR(fdisk.fit[0], int32(sizeBytes), newEBRStart, -1, fdisk.name)

	err = newEBR.Codificar(file, int64(newEBRStart))
	if err != nil {
		return fmt.Errorf("error al escribir el nuevo EBR en el disco: %v", err)
	}

	lastEBR.EstablecerSiguienteEBR(newEBRStart)
	err = lastEBR.Codificar(file, int64(lastEBR.Part_start))
	if err != nil {
		return fmt.Errorf("error al actualizar el EBR anterior: %v", err)
	}

	fmt.Fprintln(outputBuffer, "Partición lógica creada exitosamente.")
	return nil
}
