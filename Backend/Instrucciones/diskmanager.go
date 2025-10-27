package instrucciones

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	estructuras "godisk/Estructuras"
	globals "godisk/Global"
)

type DiskManager struct {
	disks         map[string]*os.File
	PartitionMBRs map[string]*estructuras.Mbr
}

func NewDiskManager() *DiskManager {
	return &DiskManager{
		disks:         make(map[string]*os.File),
		PartitionMBRs: make(map[string]*estructuras.Mbr),
	}
}

func (dm *DiskManager) LoadDisk(diskPath string) error {
	if !globals.EstaLogueado() {
		return fmt.Errorf("no hay un usuario logueado")
	}
	if err := globals.ValidarAcceso(globals.UsuarioActual.Id); err != nil {
		return fmt.Errorf("acceso denegado: %w", err)
	}

	file, err := os.OpenFile(diskPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %w", err)
	}

	mbr := &estructuras.Mbr{}
	err = mbr.Decodificar(file)
	if err != nil {
		file.Close()
		return fmt.Errorf("error al leer el MBR del disco: %w", err)
	}

	dm.disks[diskPath] = file
	dm.PartitionMBRs[diskPath] = mbr
	fmt.Printf("Disco '%s' cargado exitosamente.\n", diskPath)
	return nil
}

func (dm *DiskManager) CloseDisk(diskPath string) error {
	if file, exists := dm.disks[diskPath]; exists {
		file.Close()
		delete(dm.disks, diskPath)
		delete(dm.PartitionMBRs, diskPath)
		fmt.Printf("Disco '%s' cerrado exitosamente.\n", diskPath)
		return nil
	}
	return fmt.Errorf("disco no encontrado: %s", diskPath)
}

func (dm *DiskManager) MountPartition(diskPath string, partitionName string) (*estructuras.Partition, error) {
	partition, path, err := globals.ObtenerParticionMontada(partitionName)
	if err != nil {
		return nil, fmt.Errorf("la partición '%s' no está montada en el disco '%s': %v", partitionName, diskPath, err)
	}
	if path != diskPath {
		return nil, fmt.Errorf("la partición '%s' no está montada en el disco '%s'", partitionName, diskPath)
	}
	return partition, nil
}

func (dm *DiskManager) PrintPartitionTree(diskPath string, partitionName string, outputBuffer *bytes.Buffer) error {
	tree, err := dm.GetPartitionTree(diskPath, partitionName)
	if err != nil {
		return fmt.Errorf("error obteniendo el árbol de directorios: %v", err)
	}

	treeJSON, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return fmt.Errorf("error al serializar el árbol de directorios a JSON: %v", err)
	}

	outputBuffer.WriteString(string(treeJSON))
	return nil
}

func (dm *DiskManager) GetPartitionTree(diskPath string, partitionName string) (*DirectoryTree, error) {
	_, exists := dm.disks[diskPath]
	if !exists {
		return nil, fmt.Errorf("disco '%s' no está cargado", diskPath)
	}
	partition, err := dm.MountPartition(diskPath, partitionName)
	if err != nil {
		return nil, err
	}

	treeService, err := NewDirectoryTreeService()
	if err != nil {
		return nil, fmt.Errorf("error inicializando el servicio de árbol de directorios: %v", err)
	}
	defer treeService.Close()

	tree, err := treeService.GetDirectoryTree(fmt.Sprintf("/partition/%s", partition.Part_name))
	if err != nil {
		return nil, fmt.Errorf("error obteniendo el árbol de directorios: %v", err)
	}

	return tree, nil
}
