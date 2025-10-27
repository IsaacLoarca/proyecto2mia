package estructuras

import (
	"bytes"
	"encoding/binary"
	"fmt"
	utilidades "godisk/Utilidades"
	"os"
	"strings"
	"time"
)

const JOURNAL_ENTRIES = 50

type Journal struct {
	J_count   int32
	J_content Information
}

type Information struct {
	I_operation [10]byte
	I_path      [32]byte
	I_content   [64]byte
	I_date      uint32
}

func (journal *Journal) Encode(file *os.File, offset int64) error {
	err := utilidades.EscribirEnArchivo(file, offset, journal)
	if err != nil {
		return fmt.Errorf("error al escribir el journal en el archivo: %w", err)
	}

	return nil
}

func (journal *Journal) Decode(file *os.File, offset int64) error {
	err := utilidades.LeerDesdeArchivo(file, offset, journal)
	if err != nil {
		return fmt.Errorf("error al leer el journal del archivo: %w", err)
	}

	return nil
}

func (journal *Journal) Print() {
	date := time.Unix(int64(journal.J_content.I_date), 0)
	fmt.Println("Journal:")
	fmt.Printf("J_count: %d\n", journal.J_count)
	fmt.Println("Information:")
	fmt.Printf("I_operation: %s\n", strings.TrimSpace(string(journal.J_content.I_operation[:])))
	fmt.Printf("I_path: %s\n", strings.TrimSpace(string(journal.J_content.I_path[:])))
	fmt.Printf("I_content: %s\n", strings.TrimSpace(string(journal.J_content.I_content[:])))
	fmt.Printf("I_date: %s\n", date.Format(time.RFC3339))
}

func (j *Journal) CreateJournalEntry(op, path, content string) {
	*j = Journal{}
	copy(j.J_content.I_operation[:], op)
	copy(j.J_content.I_path[:], path)
	copy(j.J_content.I_content[:], content)
	j.J_content.I_date = uint32(time.Now().Unix())
}

func (journal *Journal) GenerateJournalTable(journalIndex int32) string {
	date := time.Unix(int64(journal.J_content.I_date), 0).Format(time.RFC3339)
	operation := strings.TrimSpace(string(journal.J_content.I_operation[:]))
	path := strings.TrimSpace(string(journal.J_content.I_path[:]))
	content := strings.TrimSpace(string(journal.J_content.I_content[:]))

	table := fmt.Sprintf(`journal_table_%d [label=<
        <TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4">
            <TR>
                <TD COLSPAN="2" BGCOLOR="#4CAF50"><FONT COLOR="#FFFFFF">Journal Entry %d</FONT></TD>
            </TR>
            <TR>
                <TD BGCOLOR="#FF7043">Operation:</TD>
                <TD>%s</TD>
            </TR>
            <TR>
                <TD BGCOLOR="#FF7043">Path:</TD>
                <TD>%s</TD>
            </TR>
            <TR>
                <TD BGCOLOR="#FF7043">Content:</TD>
                <TD>%s</TD>
            </TR>
            <TR>
                <TD BGCOLOR="#FF7043">Date:</TD>
                <TD>%s</TD>
            </TR>
        </TABLE>
    >];`, journalIndex, journalIndex, operation, path, content, date)

	return table
}

func (journal *Journal) GenerateGraph(journalStart int64, journalCount int32, file *os.File) (string, error) {
	dotContent := ""
	entrySize := int64(binary.Size(Journal{}))

	for i := int32(0); i < journalCount; i++ {
		offset := journalStart + int64(i)*entrySize
		err := journal.Decode(file, offset)
		if err != nil {
			return "", fmt.Errorf("error al deserializar el journal %d en offset %d: %v", i, offset, err)
		}
		operation := strings.TrimSpace(string(journal.J_content.I_operation[:]))
		if operation == "" {
			break
		}
		dotContent += journal.GenerateJournalTable(i)
	}

	return dotContent, nil
}

func (journal *Journal) SaveJournalEntry(file *os.File, journaling_start int64, operation string, path string, content string) error {
	journal.CreateJournalEntry(operation, path, content)
	entrySize := int64(binary.Size(Journal{}))
	offset := journaling_start + int64(journal.J_count)*entrySize

	err := journal.Encode(file, offset)
	if err != nil {
		return fmt.Errorf("error al guardar la entrada de journal: %w", err)
	}
	return nil
}

func CalculateJournalingSpace(n int32) int64 {
	return int64(n) * int64(binary.Size(Journal{}))
}

func InitializeJournalArea(file *os.File, journalStart int64, n int32) error {
	entrySize := int64(binary.Size(Journal{}))

	nullJournal := &Journal{
		J_content: Information{
			I_operation: [10]byte{},
			I_path:      [32]byte{},
			I_content:   [64]byte{},
			I_date:      0,
		},
	}

	for i := int32(0); i < n; i++ {
		nullJournal.J_count = i
		offset := journalStart + entrySize*int64(i)

		if err := utilidades.EscribirEnArchivo(file, offset, nullJournal); err != nil {
			return fmt.Errorf("error inicializando journal slot %d (off %d): %w", i, offset, err)
		}
	}

	return nil
}

func FindValidJournalEntries(file *os.File, journalStart int64, maxEntries int32) ([]Journal, error) {
	var entries []Journal
	entrySize := int64(binary.Size(Journal{}))

	validOps := map[string]bool{
		"mkdir": true, "mkfile": true, "rm": true, "rmdir": true,
		"edit": true, "cat": true, "rename": true, "copy": true,
	}

	for i := int32(0); i < maxEntries; i++ {
		offset := journalStart + int64(i)*entrySize
		journal := &Journal{}

		if err := journal.Decode(file, offset); err != nil {
			break
		}

		rawOp := journal.J_content.I_operation[:]
		nullPos := 0
		for ; nullPos < len(rawOp); nullPos++ {
			if rawOp[nullPos] == 0 {
				break
			}
		}
		operation := string(rawOp[:nullPos])
		operation = strings.TrimSpace(operation)

		if operation == "" {
			break
		}

		if _, ok := validOps[operation]; !ok {
			continue
		}
		entries = append(entries, *journal)
	}

	return entries, nil
}

func IsEmptyJournal(j *Journal) bool {
	op := strings.TrimSpace(
		string(bytes.TrimRight(j.J_content.I_operation[:], "\x00")),
	)
	return op == ""
}

func AddJournalEntry(file *os.File, journalStart int64, maxEntries int32, operation string, path string, content string, sb *Superbloque) error {
	expectedStart := int64(sb.JournalStart())
	if journalStart != expectedStart {
		journalStart = expectedStart
	}

	nextIndex, err := GetNextEmptyJournalIndex(file, journalStart, maxEntries)
	if err != nil {
		return fmt.Errorf("error buscando el siguiente índice disponible: %w", err)
	}

	if nextIndex >= maxEntries {
		nextIndex = 0
	}

	journal := &Journal{
		J_count: nextIndex,
	}

	journal.CreateJournalEntry(operation, path, content)
	offset := journalStart + int64(nextIndex)*int64(binary.Size(Journal{}))

	journalEnd := int64(sb.JournalEnd())
	if offset >= journalEnd {
		return fmt.Errorf("error: intento de escritura fuera del área de journal (%d >= %d)",
			offset, journalEnd)
	}

	if err := journal.Encode(file, offset); err != nil {
		return fmt.Errorf("error escribiendo nueva entrada de journal: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("error sincronizando archivo: %w", err)
	}

	return nil
}

func GetNextEmptyJournalIndex(file *os.File, journalStart int64, maxEntries int32) (int32, error) {
	entrySize := int64(binary.Size(Journal{}))

	for i := int32(0); i < maxEntries; i++ {
		offset := journalStart + entrySize*int64(i)

		j := &Journal{}
		if err := j.Decode(file, offset); err != nil {
			return -1, fmt.Errorf("leer journal[%d] en off %d: %w", i, offset, err)
		}

		if IsEmptyJournal(j) {
			return i, nil
		}
	}

	return 0, nil
}
