package utilidades

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ConvertirABytes(tamano int, unidad string) (int, error) {
	switch unidad {
	case "B":
		return tamano, nil
	case "K":
		return tamano * 1024, nil
	case "M":
		return tamano * 1024 * 1024, nil
	default:
		return 0, errors.New("unidad inválida")
	}
}

var abecedario = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
}

var rutaALetra = make(map[string]string)

var siguienteIndiceLetra = 0

func ObtenerLetra(ruta string) (string, error) {
	if _, existe := rutaALetra[ruta]; !existe {
		if siguienteIndiceLetra < len(abecedario) {
			rutaALetra[ruta] = abecedario[siguienteIndiceLetra]
			siguienteIndiceLetra++
		} else {
			return "", errors.New("no hay más letras disponibles para asignar")
		}
	}

	return rutaALetra[ruta], nil
}

func EliminarLetra(ruta string) {
	delete(rutaALetra, ruta)
}

func LeerDesdeArchivo(archivo *os.File, desplazamiento int64, datos interface{}) error {
	_, err := archivo.Seek(desplazamiento, 0)
	if err != nil {
		return fmt.Errorf("falló buscar el desplazamiento %d: %w", desplazamiento, err)
	}

	err = binary.Read(archivo, binary.LittleEndian, datos)
	if err != nil {
		return fmt.Errorf("falló leer datos del archivo: %w", err)
	}

	return nil
}

func EscribirEnArchivo(archivo *os.File, desplazamiento int64, datos interface{}) error {
	_, err := archivo.Seek(desplazamiento, 0)
	if err != nil {
		return fmt.Errorf("falló buscar el desplazamiento %d: %w", desplazamiento, err)
	}

	err = binary.Write(archivo, binary.LittleEndian, datos)
	if err != nil {
		return fmt.Errorf("falló escribir datos en el archivo: %w", err)
	}

	return nil
}

func CrearDirectoriosPadre(ruta string) error {
	directorio := filepath.Dir(ruta)
	err := os.MkdirAll(directorio, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error al crear las carpetas padre: %v", err)
	}
	return nil
}

func ObtenerNombresArchivo(ruta string) (string, string) {
	directorio := filepath.Dir(ruta)
	nombreBase := strings.TrimSuffix(filepath.Base(ruta), filepath.Ext(ruta))
	nombreArchivoDot := filepath.Join(directorio, nombreBase+".dot")
	imagenSalida := ruta
	return nombreArchivoDot, imagenSalida
}

func Primero[T any](slice []T) (T, error) {
	if len(slice) == 0 {
		var cero T
		return cero, errors.New("el slice está vacío")
	}
	return slice[0], nil
}

func EliminarElemento[T any](slice []T, indice int) []T {
	if indice < 0 || indice >= len(slice) {
		return slice
	}
	return append(slice[:indice], slice[indice+1:]...)
}

func DividirCadenaEnTrozos(cadena string) []string {
	var trozos []string
	for i := 0; i < len(cadena); i += 64 {
		fin := i + 64
		if fin > len(cadena) {
			fin = len(cadena)
		}
		trozos = append(trozos, cadena[i:fin])
	}
	return trozos
}

func ObtenerDirectoriosPadre(ruta string) ([]string, string) {
	ruta = filepath.Clean(ruta)
	componentes := strings.Split(ruta, string(filepath.Separator))

	var directoriosPadre []string
	for i := 1; i < len(componentes)-1; i++ {
		directoriosPadre = append(directoriosPadre, componentes[i])
	}

	directorioDestino := componentes[len(componentes)-1]
	return directoriosPadre, directorioDestino
}

func PadreCarpeta(slice []string, valor string) (string, bool) {
	for i, v := range slice {
		if v == valor {
			if i > 0 {
				return slice[i-1], true
			}
			return "", false
		}
	}
	return "", false
}

func DefinirCarpetaArchivo(directorio string) []string {
	partes := strings.Split(directorio, ".")
	resultado := make([]string, 0)

	for _, parte := range partes {
		if parte != "" {
			resultado = append(resultado, parte)
		}
	}

	if len(resultado) == 0 {
		return []string{directorio}
	}

	return resultado
}
