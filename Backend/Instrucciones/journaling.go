package instrucciones

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	estructuras "godisk/Estructuras"
	global "godisk/Global"
	"os"
	"strings"
	"time"
)

type JournalingCommand struct {
	Id string `json:"id"`
}

type JournalEntry struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
	Content   string `json:"content"`
	Date      string `json:"date"`
}

func cleanCString(buf []byte) string {
	return strings.TrimSpace(
		string(bytes.TrimRight(buf, "\x00")),
	)
}

func (cmd *JournalingCommand) Execute() (interface{}, error) {
	if cmd.Id == "" {
		return nil, errors.New("el parámetro id es obligatorio")
	}

	sb, partition, path, err := global.GetMountedPartitionSuperblock(cmd.Id)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo la partición: %w", err)
	}

	if sb.S_filesystem_type != 3 {
		return nil, errors.New("la partición no es de tipo EXT3, no tiene journaling")
	}

	file, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("error abriendo el archivo: %w", err)
	}
	defer file.Close()

	journalStart := int64(partition.Part_start) + int64(binary.Size(estructuras.Superbloque{}))

	fmt.Printf("Leyendo journal desde posición %d\n", journalStart)

	entries, err := estructuras.FindValidJournalEntries(file, journalStart, estructuras.JOURNAL_ENTRIES)
	if err != nil {
		return nil, fmt.Errorf("error buscando entradas de journal: %w", err)
	}

	if len(entries) == 0 {
		return "No hay entradas de journal para mostrar", nil
	}

	var result []JournalEntry
	for _, entry := range entries {
		operation := cleanCString(entry.J_content.I_operation[:])
		path := cleanCString(entry.J_content.I_path[:])
		content := cleanCString(entry.J_content.I_content[:])

		date := time.Unix(int64(entry.J_content.I_date), 0)
		dateStr := date.Format(time.RFC3339)

		result = append(result, JournalEntry{
			Operation: operation,
			Path:      path,
			Content:   content,
			Date:      dateStr,
		})
	}

	fmt.Printf("Se encontraron %d entradas válidas de journal\n", len(result))
	return cmd.GenerateJournalingTable(result)
}

func AnalizarJournaling(args []string) (interface{}, error) {
	cmd := &JournalingCommand{}

	if len(args) == 0 {
		return nil, errors.New("no se proporcionaron parámetros para el comando journaling")
	}

	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			continue
		}

		parts := strings.SplitN(strings.TrimPrefix(arg, "-"), "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("formato de parámetro incorrecto: %s", arg)
		}

		param, value := strings.ToLower(parts[0]), parts[1]
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = value[1 : len(value)-1]
		}

		switch param {
		case "id":
			cmd.Id = value
		default:
			return nil, fmt.Errorf("parámetro desconocido: %s", param)
		}
	}

	if cmd.Id == "" {
		return nil, errors.New("el parámetro id es obligatorio")
	}

	return cmd.Execute()
}

func (cmd *JournalingCommand) GenerateJournalingTable(entries []JournalEntry) (string, error) {
	if len(entries) == 0 {
		return "No hay entradas válidas de journal para mostrar", nil
	}

	const (
		pathWidth = 28
		dateWidth = 46
	)

	header := fmt.Sprintf("%-5s | %-10s | %-*s | %-*s | %s\n",
		"NO.", "OPERACIÓN", pathWidth, "RUTA", dateWidth, "FECHA", "CONTENIDO")
	divider := strings.Repeat("-", len(header)-1) + "\n"

	var tb strings.Builder
	tb.WriteString("\nREGISTRO DE TRANSACCIONES (JOURNAL)\n")
	tb.WriteString(strings.Repeat("=", 34) + "\n\n")
	tb.WriteString(header)
	tb.WriteString(divider)

	for i, e := range entries {
		content := e.Content
		if len(content) > 40 {
			content = content[:37] + "..."
		}

		date := e.Date
		if t, err := time.Parse(time.RFC3339, date); err == nil {
			date = t.Format("02/01/2006 15:04:05")
		}

		row := fmt.Sprintf("%-5d | %-10s | %-*s | %-*s | %s\n",
			i+1,
			e.Operation,
			pathWidth, e.Path,
			dateWidth, date,
			content)

		tb.WriteString(row)
	}

	return tb.String(), nil
}
