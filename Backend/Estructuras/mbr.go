package estructuras

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	utilidades "godisk/Utilidades"
)

type Mbr struct {
	Mbr_tamano         int32
	Mbr_fecha_creacion float32
	Mbr_dsk_signature  int32
	Dsk_fit            [1]byte
	Mbr_partitions     [4]Partition
}

func (mbr *Mbr) Codificar(file *os.File) error {
	return utilidades.EscribirEnArchivo(file, 0, mbr)
}

func (mbr *Mbr) Decodificar(file *os.File) error {
	return utilidades.LeerDesdeArchivo(file, 0, mbr)
}

func (mbr *Mbr) ObtenerPrimeraParticionDisponible() (*Partition, int, int) {
	offset := binary.Size(mbr)
	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		if mbr.Mbr_partitions[i].Part_start == -1 {
			return &mbr.Mbr_partitions[i], offset, i
		} else {
			offset += int(mbr.Mbr_partitions[i].Part_s)
		}
	}
	return nil, -1, -1
}

func (mbr *Mbr) ObtenerParticionPorNombre(name string) (*Partition, int) {
	for i, partition := range mbr.Mbr_partitions {
		partitionName := strings.Trim(string(partition.Part_name[:]), "\x00 ")
		inputName := strings.Trim(name, "\x00 ")
		if strings.EqualFold(partitionName, inputName) {
			return &partition, i
		}
	}
	return nil, -1
}

func (mbr *Mbr) ObtenerParticionPorID(id string) (*Partition, error) {
	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		partitionID := strings.Trim(string(mbr.Mbr_partitions[i].Part_id[:]), "\x00 ")
		inputID := strings.Trim(id, "\x00 ")
		if strings.EqualFold(partitionID, inputID) {
			return &mbr.Mbr_partitions[i], nil
		}
	}
	return nil, errors.New("partición no encontrada")
}

func (mbr *Mbr) TieneParticionExtendida() bool {
	for _, partition := range mbr.Mbr_partitions {
		if partition.Part_type[0] == 'E' {
			return true
		}
	}
	return false
}

func (mbr *Mbr) CalcularEspacioDisponible() (int32, error) {
	totalSize := mbr.Mbr_tamano
	usedSpace := int32(binary.Size(Mbr{}))

	partitions := mbr.Mbr_partitions[:]
	for _, part := range partitions {
		// Solo contar particiones que están activas (Part_start != -1 y tienen tamaño > 0)
		if part.Part_start != -1 && part.Part_s > 0 {
			usedSpace += part.Part_s
		}
	}

	if usedSpace >= totalSize {
		return 0, fmt.Errorf("there is no available space on the disk")
	}

	return totalSize - usedSpace, nil
}

func (mbr *Mbr) ListarParticiones() []map[string]interface{} {
	partitions := []map[string]interface{}{}
	for _, partition := range mbr.Mbr_partitions {
		if partition.Part_start != -1 {
			partitionData := map[string]interface{}{
				"name": strings.Trim(string(partition.Part_name[:]), "\x00 "),
			}
			partitions = append(partitions, partitionData)
		}
	}

	return partitions
}

func (mbr *Mbr) Imprimir() {
	creationTime := time.Unix(int64(mbr.Mbr_fecha_creacion), 0)
	diskFit := rune(mbr.Dsk_fit[0])
	fmt.Printf("MBR Size: %d | Creation Date: %s | Disk Signature: %d | Disk Fit: %c\n",
		mbr.Mbr_tamano, creationTime.Format(time.RFC3339), mbr.Mbr_dsk_signature, diskFit)
}

func (mbr *Mbr) ImprimirParticiones() {
	for i, partition := range mbr.Mbr_partitions {
		partStatus := rune(partition.Part_status[0])
		partType := rune(partition.Part_type[0])
		partFit := rune(partition.Part_fit[0])
		partName := strings.TrimSpace(string(partition.Part_name[:]))
		partID := strings.TrimSpace(string(partition.Part_id[:]))
		fmt.Printf("Partition %d: Status: %c | Type: %c | Fit: %c | Start: %d | Size: %d | Name: %s | Correlative: %d | ID: %s\n",
			i+1, partStatus, partType, partFit, partition.Part_start, partition.Part_s, partName, partition.Part_correlative, partID)
	}
}

func (mbr *Mbr) AplicarFit(partitionSize int32) (*Partition, error) {
	availableSpace, err := mbr.CalcularEspacioDisponible()
	if err != nil {
		return nil, err
	}

	if availableSpace < partitionSize {
		return nil, fmt.Errorf("no hay suficiente espacio en el disco")
	}

	switch rune(mbr.Dsk_fit[0]) {
	case 'F':
		return mbr.AplicarFirstFit(partitionSize)
	case 'B':
		return mbr.AplicarMejorFit(partitionSize)
	case 'W':
		return mbr.AplicarPeorFit(partitionSize)
	default:
		return nil, fmt.Errorf("tipo de ajuste inválido")
	}
}

func (mbr *Mbr) CalcularEspacioDisponibleParaParticion(partition *Partition) (int32, error) {
	startOfPartition := partition.Part_start
	endOfPartition := startOfPartition + partition.Part_s
	var nextPartitionStart int32 = -1
	for _, p := range mbr.Mbr_partitions {
		if p.Part_start > endOfPartition && (nextPartitionStart == -1 || p.Part_start < nextPartitionStart) {
			nextPartitionStart = p.Part_start
		}
	}
	if nextPartitionStart == -1 {
		nextPartitionStart = mbr.Mbr_tamano
	}

	availableSpace := nextPartitionStart - endOfPartition
	if availableSpace < 0 {
		return 0, fmt.Errorf("el cálculo de espacio disponible resultó en un valor negativo")
	}

	return availableSpace, nil
}

func (mbr *Mbr) AplicarFirstFit(partitionSize int32) (*Partition, error) {
	fmt.Println("Iniciando First Fit...")

	// Primero buscar una partición eliminada (Part_start == -1)
	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		partition := &mbr.Mbr_partitions[i]
		fmt.Printf("Evaluando partición %d: Inicio %d, Tamaño %d\n", i, partition.Part_start, partition.Part_s)

		// Si la partición está eliminada (Part_start == -1), podemos usarla
		if partition.Part_start == -1 {
			// Calcular la posición donde debería empezar esta partición
			offset := mbr.CalcularOffsetParaParticion(i)
			fmt.Printf("Partición %d está vacía, asignando en offset %d\n", i, offset)
			partition.Part_start = int32(offset)
			return partition, nil
		}
	}

	fmt.Println("No se encontró espacio suficiente con First Fit.")
	return nil, fmt.Errorf("no se encontró espacio suficiente con First Fit")
}

func (mbr *Mbr) AplicarMejorFit(partitionSize int32) (*Partition, error) {
	fmt.Println("Iniciando Best Fit...")

	bestFit := int32(-1)
	bestPartition := -1
	offset := binary.Size(mbr)

	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		partition := &mbr.Mbr_partitions[i]

		fmt.Printf("Evaluando partición %d: Inicio %d, Tamaño %d, Estado %c\n", i, partition.Part_start, partition.Part_s, partition.Part_status[0])

		if partition.Part_start == -1 && (partition.Part_s == -1 || partition.Part_s >= partitionSize) {
			fmt.Printf("Partición %d es candidata para Best Fit: Tamaño actual %d\n", i, partition.Part_s)

			if bestFit == -1 || partition.Part_s < bestFit {
				bestFit = partition.Part_s
				bestPartition = i
				fmt.Printf("Partición %d es la mejor opción actual para Best Fit con tamaño %d\n", i, bestFit)
			}
		}
		offset += int(partition.Part_s)
	}

	if bestPartition == -1 {
		fmt.Println("No se encontró espacio suficiente con Best Fit.")
		return nil, fmt.Errorf("no se encontró espacio suficiente con Best Fit")
	}

	partition := &mbr.Mbr_partitions[bestPartition]
	partition.Part_start = int32(offset)
	partition.Part_s = partitionSize
	fmt.Printf("Partición %d seleccionada para Best Fit: Inicio en %d, Tamaño %d\n", bestPartition, offset, partitionSize)
	return partition, nil
}

func (mbr *Mbr) AplicarPeorFit(partitionSize int32) (*Partition, error) {
	fmt.Println("Iniciando Worst Fit...")

	worstFit := int32(-1)
	worstPartition := -1
	offset := binary.Size(mbr)

	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		partition := &mbr.Mbr_partitions[i]

		fmt.Printf("Evaluando partición %d: Inicio %d, Tamaño %d, Estado %c\n", i, partition.Part_start, partition.Part_s, partition.Part_status[0])

		if partition.Part_start == -1 && (partition.Part_s == -1 || partition.Part_s >= partitionSize) {
			fmt.Printf("Partición %d es candidata para Worst Fit: Tamaño actual %d\n", i, partition.Part_s)

			if worstFit == -1 || partition.Part_s > worstFit {
				worstFit = partition.Part_s
				worstPartition = i
				fmt.Printf("Partición %d es la peor opción actual para Worst Fit con tamaño %d\n", i, worstFit)
			}
		}
		offset += int(partition.Part_s)
	}

	if worstPartition == -1 {
		fmt.Println("No se encontró espacio suficiente con Worst Fit.")
		return nil, fmt.Errorf("no se encontró espacio suficiente con Worst Fit")
	}

	partition := &mbr.Mbr_partitions[worstPartition]
	partition.Part_start = int32(offset)
	partition.Part_s = partitionSize
	fmt.Printf("Partición %d seleccionada para Worst Fit: Inicio en %d, Tamaño %d\n", worstPartition, offset, partitionSize)
	return partition, nil
}

func (mbr *Mbr) CreatePartitionWithFit(partSize int32, partType, partName string) error {
	availableSpace, err := mbr.CalcularEspacioDisponible()
	if err != nil {
		return fmt.Errorf("error calculando el espacio disponible: %v", err)
	}
	if availableSpace < partSize {
		return fmt.Errorf("no hay suficiente espacio en el disco para la nueva partición")
	}
	partition, err := mbr.AplicarFirstFit(partSize)
	if err != nil {
		return fmt.Errorf("error al aplicar el ajuste: %v", err)
	}
	partition.Part_status[0] = '1' // Activar partición (1 = Activa)
	partition.Part_s = partSize
	if len(partType) > 0 {
		partition.Part_type[0] = partType[0]
	}

	switch mbr.Dsk_fit[0] {
	case 'B', 'F', 'W':
		partition.Part_fit[0] = mbr.Dsk_fit[0]
	default:
		return fmt.Errorf("ajuste inválido en el MBR: %c. Debe ser BF (Best Fit), FF (First Fit) o WF (Worst Fit)", mbr.Dsk_fit[0])
	}
	copy(partition.Part_name[:], partName)

	fmt.Printf("Partición '%s' creada exitosamente con el ajuste '%c'.\n", partName, mbr.Dsk_fit[0])
	return nil
}

func (mbr *Mbr) CalcularOffsetParaParticion(index int) int {
	offset := binary.Size(mbr)

	// Calcular el offset basándose solo en las particiones activas anteriores a este índice
	for i := 0; i < index; i++ {
		partition := &mbr.Mbr_partitions[i]
		// Solo contar particiones que están activas (Part_start != -1 y tienen tamaño > 0)
		if partition.Part_start != -1 && partition.Part_s > 0 {
			offset += int(partition.Part_s)
		}
	}

	return offset
}
