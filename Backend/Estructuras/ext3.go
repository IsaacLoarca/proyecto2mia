package estructuras

import (
	"fmt"
	"os"
)

func (sb *Superbloque) CreateUsersFileExt3(file *os.File, journaling_start int64) error {
	fmt.Println("Inicializando área de journaling para EXT3...")
	err := InitializeJournalArea(file, journaling_start, JOURNAL_ENTRIES)
	if err != nil {
		return fmt.Errorf("error al inicializar el área de journaling: %w", err)
	}

	nextJournalIndex, err := GetNextEmptyJournalIndex(file, journaling_start, JOURNAL_ENTRIES)
	if err != nil {
		return fmt.Errorf("error obteniendo el siguiente índice de journal: %w", err)
	}
	fmt.Printf("Siguiente índice de journal disponible: %d\n", nextJournalIndex)

	err = AddJournalEntry(
		file,
		journaling_start,
		JOURNAL_ENTRIES,
		"mkdir",
		"/",
		"",
		sb,
	)
	if err != nil {
		return fmt.Errorf("error al guardar la entrada de la raíz en el journal: %w", err)
	}

	rootBlockIndex, err := sb.FindNextFreeBlock(file)
	if err != nil {
		return fmt.Errorf("error al encontrar el primer bloque libre para la raíz: %w", err)
	}

	rootBlocks := [15]int32{rootBlockIndex, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}

	rootInode := &Inodo{}
	err = rootInode.CreateInode(
		file,
		sb,
		'0',
		0,
		rootBlocks,
		[3]byte{'7', '7', '7'},
	)
	if err != nil {
		return fmt.Errorf("error al crear el inodo raíz: %w", err)
	}

	rootBlock := &FolderBlock{
		B_content: [4]FolderContent{
			{B_name: [12]byte{'.'}, B_inodo: 0},
			{B_name: [12]byte{'.', '.'}, B_inodo: 0},
			{B_name: [12]byte{'u', 's', 'e', 'r', 's', '.', 't', 'x', 't'}, B_inodo: sb.S_inodes_count},
			{B_name: [12]byte{'-'}, B_inodo: -1},
		},
	}

	err = sb.UpdateBitmapBlock(file, rootBlockIndex, true)
	if err != nil {
		return fmt.Errorf("error actualizando el bitmap de bloques: %w", err)
	}

	err = rootBlock.Encode(file, int64(sb.S_first_blo))
	if err != nil {
		return fmt.Errorf("error serializando el bloque raíz: %w", err)
	}

	sb.UpdateSuperblockAfterBlockAllocation()

	rootGroup := NewGroup("1", "root")
	rootUser := NewUser("1", "root", "root", "123")
	usersText := fmt.Sprintf("%s\n%s\n", rootGroup.ToString(), rootUser.ToString())

	err = AddJournalEntry(
		file,
		journaling_start,
		JOURNAL_ENTRIES,
		"mkfile",
		"/users.txt",
		usersText,
		sb,
	)
	if err != nil {
		return fmt.Errorf("error al guardar la entrada del archivo /users.txt en el journal: %w", err)
	}

	usersBlockIndex, err := sb.FindNextFreeBlock(file)
	if err != nil {
		return fmt.Errorf("error al encontrar el primer bloque libre para /users.txt: %w", err)
	}
	fileBlocks := [15]int32{usersBlockIndex, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}

	usersInode := &Inodo{}
	err = usersInode.CreateInode(
		file,
		sb,
		'1',
		int32(len(usersText)),
		fileBlocks,
		[3]byte{'7', '7', '7'},
	)
	if err != nil {
		return fmt.Errorf("error al crear el inodo de /users.txt: %w", err)
	}
	usersBlock := &ArchivoBloque{
		B_content: [64]byte{},
	}
	usersBlock.AppendContent(usersText)
	err = usersBlock.Encode(file, int64(sb.S_first_blo))
	if err != nil {
		return fmt.Errorf("error serializando el bloque de /users.txt: %w", err)
	}
	err = sb.UpdateBitmapBlock(file, usersBlockIndex, true)
	if err != nil {
		return fmt.Errorf("error actualizando el bitmap de bloques para /users.txt: %w", err)
	}

	sb.UpdateSuperblockAfterBlockAllocation()

	fmt.Println("Bloques")
	sb.PrintBlocks(file.Name())

	fmt.Println("Journal Entries:")
	entries, err := FindValidJournalEntries(file, journaling_start, JOURNAL_ENTRIES)
	if err != nil {
		fmt.Printf("Error leyendo entradas de journal: %v\n", err)
	} else {
		for i, entry := range entries {
			fmt.Printf("-- Entrada %d --\n", i)
			entry.Print()
		}
	}

	fmt.Println("Sistema de archivos EXT3 inicializado correctamente con journaling")
	return nil
}
