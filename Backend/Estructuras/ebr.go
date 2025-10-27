package estructuras

import (
	"fmt"
	utilidades "godisk/Utilidades"
	"os"
)

type Ebr struct {
	Part_mount [1]byte
	Part_fit   [1]byte
	Part_start int32
	Part_s     int32
	Part_next  int32
	Part_name  [16]byte
}

func (e *Ebr) Codificar(archivo *os.File, posicion int64) error {
	return utilidades.EscribirEnArchivo(archivo, posicion, e)
}

func (e *Ebr) Decodificar(file *os.File, position int64) error {
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error al obtener información del archivo: %v", err)
	}
	if position < 0 || position >= fileInfo.Size() {
		return fmt.Errorf("posición inválida para EBR: %d", position)
	}

	err = utilidades.LeerDesdeArchivo(file, position, e)
	if err != nil {
		return err
	}

	fmt.Printf("EBR decodificado con éxito desde la posición %d.\n", position)
	return nil
}

func (e *Ebr) EstablecerEBR(ajuste byte, tamano int32, inicio int32, siguiente int32, nombre string) {
	fmt.Println("Estableciendo valores del EBR:")
	fmt.Printf("Fit: %c | Size: %d | Start: %d | Next: %d | Name: %s\n", ajuste, tamano, inicio, siguiente, nombre)

	e.Part_mount[0] = '1' // Created
	e.Part_fit[0] = ajuste
	e.Part_start = inicio
	e.Part_s = tamano
	e.Part_next = siguiente

	copy(e.Part_name[:], nombre)
	for i := len(nombre); i < len(e.Part_name); i++ {
		e.Part_name[i] = 0
	}
}

func CrearYEscribirEBR(inicio int32, tamano int32, ajuste byte, nombre string, archivo *os.File) error {
	fmt.Printf("Creando y escribiendo Ebr en la posición: %d\n", inicio)
	ebr := &Ebr{}
	ebr.EstablecerEBR(ajuste, tamano, inicio, -1, nombre)
	return ebr.Codificar(archivo, int64(inicio))
}

func (e *Ebr) Imprimir() {
	fmt.Printf("Mount: %c | Fit: %c | Start: %d | Size: %d | Next: %d | Name: %s\n",
		e.Part_mount[0], e.Part_fit[0], e.Part_start, e.Part_s, e.Part_next, string(e.Part_name[:]))
}

func (e *Ebr) CalcularInicioSiguienteEBR(inicioParticionExtendida int32, tamanoParticionExtendida int32) (int32, error) {

	if e.Part_s <= 0 {
		return -1, fmt.Errorf("tamaño del EBR inválido")
	}

	if e.Part_start < inicioParticionExtendida {
		return -1, fmt.Errorf("posición de inicio del EBR inválida")
	}

	siguienteInicio := e.Part_start + e.Part_s

	if siguienteInicio <= e.Part_start || siguienteInicio >= inicioParticionExtendida+tamanoParticionExtendida {
		return -1, fmt.Errorf("el siguiente EBR está fuera de los límites de la partición extendida")
	}

	return siguienteInicio, nil
}

func EncontrarUltimoEBR(inicio int32, archivo *os.File) (*Ebr, error) {
	currentEBR := &Ebr{}

	// Decodificar el EBR en la posición inicial
	err := currentEBR.Decodificar(archivo, int64(inicio))
	if err != nil {
		return nil, err
	}

	// Recorrer la cadena de EBRs hasta encontrar el último
	for currentEBR.Part_next != -1 {
		if currentEBR.Part_next < 0 {
			// Evitar leer una posición negativa
			return currentEBR, nil
		}
		fmt.Printf("EBR encontrado - Start: %d, Next: %d\n", currentEBR.Part_start, currentEBR.Part_next)

		// Crear una nueva instancia de EBR para el siguiente
		nextEBR := &Ebr{}
		err = nextEBR.Decodificar(archivo, int64(currentEBR.Part_next))
		if err != nil {
			return nil, err
		}
		currentEBR = nextEBR
	}

	fmt.Printf("Último EBR encontrado en la posición: %d\n", currentEBR.Part_start)
	return currentEBR, nil
}

func (e *Ebr) EstablecerSiguienteEBR(nuevoSiguiente int32) {
	e.Part_next = nuevoSiguiente
}

func (e *Ebr) Overwrite(file *os.File) error {
	// Verificar si el EBR tiene un tamaño válido
	if e.Part_s <= 0 {
		return fmt.Errorf("el tamaño del EBR es inválido o cero")
	}

	// Posicionarse en el inicio del EBR (donde comienza la partición lógica)
	_, err := file.Seek(int64(e.Part_start), 0)
	if err != nil {
		return fmt.Errorf("error al mover el puntero del archivo a la posición del EBR: %v", err)
	}

	// Crear un buffer de ceros del tamaño de la partición lógica
	zeroes := make([]byte, e.Part_s)

	// Escribir los ceros en el archivo
	_, err = file.Write(zeroes)
	if err != nil {
		return fmt.Errorf("error al sobrescribir el espacio del EBR: %v", err)
	}

	fmt.Printf("Espacio de la partición lógica (EBR) en posición %d sobrescrito con ceros.\n", e.Part_start)
	return nil
}
