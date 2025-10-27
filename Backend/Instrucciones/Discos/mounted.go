package instrucciones

import (
	"fmt"
	globales "godisk/Global"
	"strings"
)

func Mounted(parametros []string) (string, error) {
	var resultado strings.Builder
	resultado.WriteString("================ MOUNTED =================\n")

	var contador = 0
	for id := range globales.ParticionesMontadas {
		contador++
		if len(globales.ParticionesMontadas) == contador {
			resultado.WriteString(fmt.Sprintf("%s\n", id))
		} else {
			resultado.WriteString(fmt.Sprintf("%s,", id))
		}
	}

	resultado.WriteString("==================== FIN MOUNTED ====================\n")

	if contador == 0 {
		resultado.WriteString("No hay particiones montadas\n")
	}

	return resultado.String(), nil

}
