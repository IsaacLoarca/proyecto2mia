package instrucciones

import (
	"bytes"
	"errors"
	"fmt"
	estructuras "godisk/Estructuras"
	globales "godisk/Global"
	utilidades "godisk/Utilidades"
	"os"
	"regexp"
	"strings"
)

type MKDIR struct {
	ruta string
	p    bool
}

func AnalizarMkdir(parametros []string) (string, error) {
	cmd := &MKDIR{}
	var outputBuffer bytes.Buffer
	args := strings.Join(parametros, " ")
	re := regexp.MustCompile(`-path=[^\s]+|-p`)
	matches := re.FindAllString(args, -1)

	if len(matches) != len(parametros) {
		for _, token := range parametros {
			if !re.MatchString(token) {
				return "", fmt.Errorf("parámetro inválido: %s", token)
			}
		}
	}

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		key := strings.ToLower(kv[0])

		switch key {
		case "-path":
			if len(kv) != 2 {
				return "", fmt.Errorf("formato de parámetro inválido: %s", match)
			}
			value := kv[1]
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
			cmd.ruta = value
		case "-p":
			cmd.p = true
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.ruta == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	err := ejecutarMkdir(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

func ejecutarMkdir(mkdir *MKDIR, outputBuffer *bytes.Buffer) error {

	if !globales.EstaLogueado() {
		fmt.Println("No hay ninguna sesión activa")
		return fmt.Errorf("no hay ninguna sesión activa")
	}

	idParticion := globales.UsuarioActual.Id

	particionSuperbloque, particionMontada, rutaPartition, err := globales.GetMountedPartitionSuperblock(idParticion)

	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	archivo, err := os.OpenFile(rutaPartition, os.O_RDWR, 0666)

	if err != nil {
		return fmt.Errorf("error al abrir el archivo de partición: %w", err)
	}

	defer archivo.Close()

	fmt.Printf("Creando directorio: %s\n", mkdir.ruta)

	err = CrearDirectorio(mkdir.ruta, mkdir.p, particionSuperbloque, archivo, particionMontada)

	if err != nil {
		return fmt.Errorf("error al crear el directorio: %w", err)
	}

	return nil
}

func CrearDirectorio(dirRuta string, crearPadres bool, superbloque *estructuras.Superbloque, archivo *os.File, particionMontada *estructuras.Partition) error {
	directorios, _ := utilidades.ObtenerDirectoriosPadre(dirRuta)
	var dirPadres []string
	if len(directorios) > 1 {
		dirPadres = directorios[:len(directorios)-1]
	} else {
		dirPadres = []string{}
	}
	siExistenTodosLosPadres := false
	contador := 0
	if len(directorios) == 1 {
		siExistenTodosLosPadres = true
	} else if len(dirPadres) > 0 {
		for _, parentDir := range dirPadres {
			for i := int32(0); i < superbloque.S_inodes_count; i++ {
				bandera, err := superbloque.ValidarExistenciaDeDirectorio(archivo, i, parentDir)
				if err != nil {
					return err
				}
				if bandera {
					contador++
				}
				if contador == len(dirPadres) {
					siExistenTodosLosPadres = true
					break
				}
			}
		}
	}

	if siExistenTodosLosPadres || crearPadres {
		for _, parentDir := range directorios {
			err := superbloque.CrearCarpeta(archivo, directorios, parentDir, crearPadres)
			if err != nil {
				fmt.Println("Error al crear las carpetas")
				return err
			}
		}
	} else {
		fmt.Println("No se pudieron crear los directorios ya que hacen falta carpeta padre")
	}

	err := superbloque.Codificar(archivo, int64(particionMontada.Part_start))

	if err != nil {
		return fmt.Errorf("error al serializar el superbloque: %w", err)
	}

	return nil
}
