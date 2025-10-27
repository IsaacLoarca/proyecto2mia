package instrucciones

import (
	"bytes"
	"fmt"
	estructuras "godisk/Estructuras"
	global "godisk/Global"
	"os"
	"regexp"
	"strings"
)

func AnalizarLoss(tokens []string) (string, error) {
	var output bytes.Buffer
	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-id=[^\s]+`)
	matches := re.FindAllString(args, -1)
	if len(matches) != len(tokens) {
		for _, token := range tokens {
			if !re.MatchString(token) {
				return "", fmt.Errorf("parámetro inválido: %s", token)
			}
		}
	}
	var id string
	for _, m := range matches {
		parts := strings.SplitN(m, "=", 2)
		if strings.ToLower(parts[0]) == "-id" && len(parts) == 2 {
			id = strings.Trim(parts[1], `"`)
		}
	}
	if id == "" {
		return "", fmt.Errorf("falta parámetro requerido: -id")
	}
	if err := LossPartition(id); err != nil {
		return "", err
	}
	output.WriteString(fmt.Sprintf("Simulación de pérdida completada en partición %s", id))
	return output.String(), nil
}

func LossPartition(id string) error {
	sb, part, path, err := global.GetMountedPartitionSuperblock(id)
	if err != nil {
		return fmt.Errorf("no existe montaje %s: %w", id, err)
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	if err = sb.Decodificar(f, int64(part.Part_start)); err != nil {
		return fmt.Errorf("leer superbloque: %w", err)
	}

	if err = estructuras.CleanLossAreas(f, sb); err != nil {
		return err
	}
	return nil
}
