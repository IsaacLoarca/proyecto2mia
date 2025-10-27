package global

import (
	"errors"
	estructuras "godisk/Estructuras"
	"os"
)

const Carnet string = "46"

var (
	UsuarioActual       *estructuras.Usuario = nil
	ParticionesMontadas map[string]string    = make(map[string]string)
)

func GetMountedPartitionSuperblock(id string) (*estructuras.Superbloque, *estructuras.Partition, string, error) {
	path := ParticionesMontadas[id]
	if path == "" {
		return nil, nil, "", errors.New("la partición no está montada")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	var mbr estructuras.Mbr

	err = mbr.Decodificar(file)
	if err != nil {
		return nil, nil, "", err
	}

	partition, err := mbr.ObtenerParticionPorID(id)
	if partition == nil {
		return nil, nil, "", err
	}

	var sb estructuras.Superbloque

	err = sb.Decodificar(file, int64(partition.Part_start))
	if err != nil {
		return nil, nil, "", err
	}

	return &sb, partition, path, nil
}

func ObtenerParticionMontada(id string) (*estructuras.Partition, string, error) {
	path := ParticionesMontadas[id]
	if path == "" {
		return nil, "", errors.New("la partición no está montada")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var mbr estructuras.Mbr

	err = mbr.Decodificar(file)
	if err != nil {
		return nil, "", err
	}

	partition, err := mbr.ObtenerParticionPorID(id)
	if partition == nil {
		return nil, "", err
	}

	return partition, path, nil
}

func GetMountedPartitionRep(id string) (*estructuras.Mbr, *estructuras.Superbloque, string, error) {
	path := ParticionesMontadas[id]
	if path == "" {
		return nil, nil, "", errors.New("la partición no está montada")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	defer file.Close()

	var mbr estructuras.Mbr

	err = mbr.Decodificar(file)
	if err != nil {
		return nil, nil, "", err
	}

	partition, err := mbr.ObtenerParticionPorID(id)
	if err != nil {
		return nil, nil, "", err
	}

	var sb estructuras.Superbloque

	err = sb.Decodificar(file, int64(partition.Part_start))
	if err != nil {
		return nil, nil, "", err
	}

	return &mbr, &sb, path, nil
}

func EstaLogueado() bool {
	return UsuarioActual != nil && UsuarioActual.Status
}

func CerrarSesion() {
	if UsuarioActual != nil {
		UsuarioActual.Status = false
		UsuarioActual = nil
	}
}

func ValidarAcceso(partitionId string) error {
	if !EstaLogueado() {
		return errors.New("no hay un usuario logueado")
	}
	_, _, err := ObtenerParticionMontada(partitionId)
	if err != nil {
		return errors.New("la partición no está montada")
	}
	return nil
}
