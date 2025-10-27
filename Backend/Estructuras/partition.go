package estructuras

import (
	"fmt"
	"os"
	"strings"
)

type Partition struct {
	Part_status      [1]byte
	Part_type        [1]byte
	Part_fit         [1]byte
	Part_start       int32
	Part_s           int32
	Part_name        [16]byte
	Part_correlative int32
	Part_id          [4]byte
}

func (p *Partition) CrearParticion(partStart, partSize int, partType, partFit, partName string) {
	p.Part_status[0] = '0'
	p.Part_start = int32(partStart)
	p.Part_s = int32(partSize)
	if len(partType) > 0 {
		p.Part_type[0] = partType[0]
	}
	if len(partFit) > 0 {
		p.Part_fit[0] = partFit[0]
	}
	copy(p.Part_name[:], partName)
}

func (p *Partition) ImprimirParticion() {
	fmt.Printf("Status: %c | Type: %c | Fit: %c | Start: %d | Size: %d | Name: %s | Correlative: %d | ID: %s\n",
		p.Part_status[0], p.Part_type[0], p.Part_fit[0], p.Part_start, p.Part_s,
		string(p.Part_name[:]), p.Part_correlative, string(p.Part_id[:]))
}

func (p *Partition) MontarParticion(correlative int, id string) error {
	p.Part_correlative = int32(correlative)
	copy(p.Part_id[:], id)
	return nil
}

func (p *Partition) ModifySize(addSize int32, availableSpace int32) error {
	newSize := p.Part_s + addSize

	if newSize < 0 {
		return fmt.Errorf("el tamaño de la partición no puede ser negativo")
	}

	if addSize > 0 && availableSpace < addSize {
		return fmt.Errorf("no hay suficiente espacio disponible para agregar a la partición")
	}
	p.Part_s = newSize

	fmt.Printf("El tamaño de la partición '%s' ha sido modificado. Nuevo tamaño: %d bytes.\n", string(p.Part_name[:]), p.Part_s)
	return nil
}

func (p *Partition) Delete(deleteType string, file *os.File, isExtended bool) error {
	if isExtended {
		err := p.deleteLogicalPartitions(file)
		if err != nil {
			return fmt.Errorf("error al eliminar las particiones lógicas dentro de la partición extendida: %v", err)
		}
	}

	if deleteType == "full" {
		err := p.Overwrite(file)
		if err != nil {
			return fmt.Errorf("error al sobrescribir la partición: %v", err)
		}
	}

	p.Part_start = -1
	p.Part_s = -1
	p.Part_name = [16]byte{}

	fmt.Printf("La partición '%s' ha sido eliminada (%s).\n", strings.TrimSpace(string(p.Part_name[:])), deleteType)
	return nil
}

func (p *Partition) Overwrite(file *os.File) error {
	_, err := file.Seek(int64(p.Part_start), 0)
	if err != nil {
		return err
	}
	zeroes := make([]byte, p.Part_s)
	_, err = file.Write(zeroes)
	if err != nil {
		return fmt.Errorf("error al sobrescribir el espacio de la partición: %v", err)
	}

	fmt.Printf("Espacio de la partición sobrescrito con ceros.\n")
	return nil
}

func (p *Partition) deleteLogicalPartitions(file *os.File) error {
	fmt.Println("Eliminando particiones lógicas dentro de la partición extendida...")
	var currentEBR Ebr
	start := p.Part_start
	for {
		err := currentEBR.Decodificar(file, int64(start))
		if err != nil {
			return fmt.Errorf("error al leer el EBR: %v", err)
		}
		if currentEBR.Part_start == -1 {
			break
		}
		currentEBR.Part_start = -1
		currentEBR.Part_s = -1
		copy(currentEBR.Part_name[:], "")

		err = currentEBR.Overwrite(file)
		if err != nil {
			return fmt.Errorf("error al sobrescribir el EBR: %v", err)
		}

		start = currentEBR.Part_next
	}

	fmt.Println("Particiones lógicas eliminadas exitosamente.")
	return nil
}
