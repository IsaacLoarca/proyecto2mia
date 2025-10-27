package instrucciones

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type DiskCommand struct {
	DiskManager *DiskManager
}

func NewDiskCommand() *DiskCommand {
	return &DiskCommand{
		DiskManager: NewDiskManager(),
	}
}

func (dc *DiskCommand) ShowDisk(diskPath string) (string, error) {
	var outputBuffer bytes.Buffer

	err := dc.DiskManager.LoadDisk(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al cargar el disco: %v", err)
	}

	mbr, exists := dc.DiskManager.PartitionMBRs[diskPath]
	if !exists {
		return "", fmt.Errorf("error: no se pudo encontrar el MBR para el disco en la ruta '%s'", diskPath)
	}

	partitions := mbr.ListarParticiones()
	partitionsJSON, err := json.MarshalIndent(partitions, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error serializando las particiones a JSON: %v", err)
	}

	outputBuffer.WriteString(fmt.Sprintf("Disco: %s\n", diskPath))
	outputBuffer.WriteString("Particiones:\n")
	outputBuffer.WriteString(string(partitionsJSON))
	outputBuffer.WriteString("\n")

	return outputBuffer.String(), nil
}
