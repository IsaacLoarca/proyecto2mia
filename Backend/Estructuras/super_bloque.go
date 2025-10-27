package estructuras

import (
	"encoding/binary"
	"fmt"
	utilidades "godisk/Utilidades"
	"os"
	"time"
)

type Superbloque struct {
	S_filesystem_type   int32
	S_inodes_count      int32
	S_blocks_count      int32
	S_free_blocks_count int32
	S_free_inodes_count int32
	S_mtime             float64
	S_umtime            float64
	S_mnt_count         int32
	S_magic             int32
	S_inode_size        int32
	S_block_size        int32
	S_first_ino         int32
	S_first_blo         int32
	S_bm_inode_start    int32
	S_bm_block_start    int32
	S_inode_start       int32
	S_block_start       int32
}

func (sb *Superbloque) Codificar(file *os.File, offset int64) error {
	return utilidades.EscribirEnArchivo(file, offset, sb)
}

func (sb *Superbloque) Decodificar(file *os.File, offset int64) error {
	return utilidades.LeerDesdeArchivo(file, offset, sb)
}

func (sb *Superbloque) CrearArchivoUsuarios(file *os.File) error {
	rootInode := &Inodo{
		I_uid:   1,
		I_gid:   1,
		I_size:  0,
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{sb.S_blocks_count, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		I_type:  [1]byte{'0'}, // Tipo carpeta
		I_perm:  [3]byte{'7', '7', '7'},
	}

	err := utilidades.EscribirEnArchivo(file, int64(sb.S_inode_start), rootInode)
	if err != nil {
		return fmt.Errorf("error al escribir el inodo raíz: %w", err)
	}

	err = sb.UpdateBitmapInode(file, 0, true)
	if err != nil {
		return fmt.Errorf("error al actualizar bitmap de inodos: %w", err)
	}

	sb.UpdateSuperblockAfterInodeAllocation()

	rootBlock := &FolderBlock{
		B_content: [4]FolderContent{
			{B_name: [12]byte{'.'}, B_inodo: 0},                                                         // Apunta a sí mismo
			{B_name: [12]byte{'.', '.'}, B_inodo: 0},                                                    // Apunta al padre
			{B_name: [12]byte{'u', 's', 'e', 'r', 's', '.', 't', 'x', 't'}, B_inodo: sb.S_inodes_count}, // Apunta a users.txt
			{B_name: [12]byte{'-'}, B_inodo: -1},                                                        // Vacío
		},
	}

	err = utilidades.EscribirEnArchivo(file, int64(sb.S_block_start), rootBlock)
	if err != nil {
		return fmt.Errorf("error al escribir el bloque raíz: %w", err)
	}

	err = sb.UpdateBitmapBlock(file, 0, true)
	if err != nil {
		return fmt.Errorf("error al actualizar el bitmap de bloques: %w", err)
	}

	sb.UpdateSuperblockAfterBlockAllocation()

	rootGroup := NewGroup("1", "root")
	rootUser := NewUser("1", "root", "root", "123")
	usersText := fmt.Sprintf("%s\n%s\n", rootGroup.ToString(), rootUser.ToString())

	usersInode := &Inodo{
		I_uid:   1,
		I_gid:   1,
		I_size:  int32(len(usersText)),
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // Apunta al bloque 1 (users.txt)
		I_type:  [1]byte{'1'},                                                         // Tipo archivo
		I_perm:  [3]byte{'7', '7', '7'},
	}

	err = utilidades.EscribirEnArchivo(file, int64(sb.S_inode_start+int32(binary.Size(usersInode))), usersInode)
	if err != nil {
		return fmt.Errorf("error al escribir el inodo de users.txt: %w", err)
	}

	err = sb.UpdateBitmapInode(file, 1, true)
	if err != nil {
		return fmt.Errorf("error al actualizar bitmap de inodos para users.txt: %w", err)
	}

	sb.UpdateSuperblockAfterInodeAllocation()

	usersBlock := &ArchivoBloque{}
	copy(usersBlock.B_content[:], usersText)

	err = utilidades.EscribirEnArchivo(file, int64(sb.S_block_start+int32(binary.Size(usersBlock))), usersBlock)
	if err != nil {
		return fmt.Errorf("error al escribir el bloque de users.txt: %w", err)
	}

	err = sb.UpdateBitmapBlock(file, 1, true)
	if err != nil {
		return fmt.Errorf("error al actualizar el bitmap de bloques para users.txt: %w", err)
	}

	sb.UpdateSuperblockAfterBlockAllocation()

	fmt.Println("Archivo users.txt creado correctamente.")
	fmt.Println("Superbloque después de la creación de users.txt:")
	sb.Print()
	fmt.Println("\nBloques:")
	sb.PrintBlocks(file.Name())
	fmt.Println("\nInodos:")
	sb.PrintInodes(file.Name())
	return nil
}

func (sb *Superbloque) Print() {
	fmt.Printf("%-25s %-10s\n", "Campo", "Valor")
	fmt.Printf("%-25s %-10s\n", "-------------------------", "----------")
	fmt.Printf("%-25s %-10d\n", "S_filesystem_type:", sb.S_filesystem_type)
	fmt.Printf("%-25s %-10d\n", "S_inodes_count:", sb.S_inodes_count)
	fmt.Printf("%-25s %-10d\n", "S_blocks_count:", sb.S_blocks_count)
	fmt.Printf("%-25s %-10d\n", "S_free_blocks_count:", sb.S_free_blocks_count)
	fmt.Printf("%-25s %-10d\n", "S_free_inodes_count:", sb.S_free_inodes_count)
	fmt.Printf("%-25s %-10s\n", "S_mtime:", time.Unix(int64(sb.S_mtime), 0).Format("02/01/2006 15:04"))
	fmt.Printf("%-25s %-10s\n", "S_umtime:", time.Unix(int64(sb.S_umtime), 0).Format("02/01/2006 15:04"))
	fmt.Printf("%-25s %-10d\n", "S_mnt_count:", sb.S_mnt_count)
	fmt.Printf("%-25s %-10x\n", "S_magic:", sb.S_magic)
	fmt.Printf("%-25s %-10d\n", "S_inode_size:", sb.S_inode_size)
	fmt.Printf("%-25s %-10d\n", "S_block_size:", sb.S_block_size)
	fmt.Printf("%-25s %-10d\n", "S_first_ino:", sb.S_first_ino)
	fmt.Printf("%-25s %-10d\n", "S_first_blo:", sb.S_first_blo)
	fmt.Printf("%-25s %-10d\n", "S_bm_inode_start:", sb.S_bm_inode_start)
	fmt.Printf("%-25s %-10d\n", "S_bm_block_start:", sb.S_bm_block_start)
	fmt.Printf("%-25s %-10d\n", "S_inode_start:", sb.S_inode_start)
	fmt.Printf("%-25s %-10d\n", "S_block_start:", sb.S_block_start)
}

func (sb *Superbloque) PrintInodes(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	fmt.Println("\nInodos\n----------------")
	inodes := make([]Inodo, sb.S_inodes_count)

	for i := int32(0); i < sb.S_inodes_count; i++ {
		inode := &inodes[i]
		err := utilidades.LeerDesdeArchivo(file, int64(sb.S_inode_start+(i*int32(binary.Size(Inodo{})))), inode)
		if err != nil {
			return fmt.Errorf("failed to decode inode %d: %w", i, err)
		}
	}

	for i, inode := range inodes {
		fmt.Printf("\nInodo %d:\n", i)
		inode.Print()
	}

	return nil
}

func (sb *Superbloque) PrintBlocks(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	fmt.Println("\nBloques\n----------------")
	inodes := make([]Inodo, sb.S_inodes_count)

	for i := int32(0); i < sb.S_inodes_count; i++ {
		inode := &inodes[i]
		err := utilidades.LeerDesdeArchivo(file, int64(sb.S_inode_start+(i*int32(binary.Size(Inodo{})))), inode)
		if err != nil {
			return fmt.Errorf("failed to decode inode %d: %w", i, err)
		}
	}

	for _, inode := range inodes {
		for _, blockIndex := range inode.I_block {
			if blockIndex == -1 {
				break
			}
			if inode.I_type[0] == '0' {
				block := &FolderBlock{}
				err := utilidades.LeerDesdeArchivo(file, int64(sb.S_block_start+(blockIndex*sb.S_block_size)), block)
				if err != nil {
					return fmt.Errorf("failed to decode folder block %d: %w", blockIndex, err)
				}
				fmt.Printf("\nBloque %d:\n", blockIndex)
				block.Print()
			} else if inode.I_type[0] == '1' {
				block := &ArchivoBloque{}
				err := utilidades.LeerDesdeArchivo(file, int64(sb.S_block_start+(blockIndex*sb.S_block_size)), block)
				if err != nil {
					return fmt.Errorf("failed to decode file block %d: %w", blockIndex, err)
				}
				fmt.Printf("\nBloque %d:\n", blockIndex)
				block.Print()
			}
		}
	}

	return nil
}

func (sb *Superbloque) FindNextFreeBlock(file *os.File) (int32, error) {
	totalBlocks := sb.S_blocks_count + sb.S_free_blocks_count // Número total de bloques

	for position := int32(0); position < totalBlocks; position++ {
		isFree, err := sb.isBlockFree(file, sb.S_bm_block_start, position)
		if err != nil {
			return -1, fmt.Errorf("error buscando bloque libre: %w", err)
		}

		if isFree {
			err = sb.UpdateBitmapBlock(file, position, true)
			if err != nil {
				return -1, fmt.Errorf("error actualizando el bitmap del bloque: %w", err)
			}

			fmt.Println("Indice encontrado:", position)
			return position, nil
		}
	}

	return -1, fmt.Errorf("no hay bloques disponibles")
}

func (sb *Superbloque) FindNextFreeInode(file *os.File) (int32, error) {
	totalInodes := sb.S_inodes_count + sb.S_free_inodes_count // Número total de inodos

	for position := int32(0); position < totalInodes; position++ {
		isFree, err := sb.isInodeFree(file, sb.S_bm_inode_start, position)
		if err != nil {
			return -1, fmt.Errorf("error buscando inodo libre en la posición %d: %w", position, err)
		}

		if isFree {
			err = sb.UpdateBitmapInode(file, position, true)
			if err != nil {
				return -1, fmt.Errorf("error actualizando el bitmap del inodo en la posición %d: %w", position, err)
			}
			fmt.Printf("Inodo libre encontrado y asignado: %d\n", position)
			return position, nil
		}
	}

	return -1, fmt.Errorf("no hay inodos disponibles")
}

func (sb *Superbloque) AssignNewBlock(file *os.File, inode *Inodo, index int) (int32, error) {
	fmt.Println("=== Iniciando la asignación de un nuevo bloque ===")

	if index < 0 || index >= len(inode.I_block) {
		return -1, fmt.Errorf("índice de bloque fuera de rango: %d", index)
	}

	if inode.I_block[index] != -1 {
		return -1, fmt.Errorf("bloque en el índice %d ya está asignado: %d", index, inode.I_block[index])
	}

	newBlock, err := sb.FindNextFreeBlock(file)
	if err != nil {
		return -1, fmt.Errorf("error buscando nuevo bloque libre: %w", err)
	}

	if newBlock == -1 {
		return -1, fmt.Errorf("no hay bloques libres disponibles")
	}

	inode.I_block[index] = newBlock
	fmt.Printf("Nuevo bloque asignado: %d en I_block[%d]\n", newBlock, index)

	sb.UpdateSuperblockAfterBlockAllocation()

	return newBlock, nil
}

func (sb *Superbloque) AssignNewInode(file *os.File) (int32, error) {
	// Intentar encontrar un inodo libre
	newInode, err := sb.FindNextFreeInode(file)
	if err != nil {
		return -1, fmt.Errorf("error buscando nuevo inodo libre: %w", err)
	}
	// Verificar si se encontró un inodo libre
	if newInode == -1 {
		return -1, fmt.Errorf("no hay inodos libres disponibles")
	}

	// Actualizar el Superblock después de asignar el inodo
	sb.UpdateSuperblockAfterInodeAllocation()

	// Retornar el nuevo inodo asignado
	return newInode, nil
}

func WriteInodeToFile(file *os.File, offset int64, inode *Inodo) error {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return fmt.Errorf("error buscando la posición para escribir el inodo: %w", err)
	}

	err = binary.Write(file, binary.LittleEndian, inode)
	if err != nil {
		return fmt.Errorf("error escribiendo el inodo en el archivo: %w", err)
	}

	return nil
}

func (sb *Superbloque) CalculateInodeOffset(inodeIndex int32) int64 {
	return int64(sb.S_inode_start) + int64(inodeIndex)*int64(sb.S_inode_size)
}

func (sb *Superbloque) UpdateSuperblockAfterBlockAllocation() {
	sb.S_blocks_count++

	sb.S_free_blocks_count--

	sb.S_first_blo += sb.S_block_size
}

func (sb *Superbloque) UpdateSuperblockAfterInodeAllocation() {
	sb.S_inodes_count++

	sb.S_free_inodes_count--

	sb.S_first_ino += sb.S_inode_size
}

func (sb *Superbloque) UpdateSuperblockAfterBlockDeallocation() {
	sb.S_blocks_count--

	sb.S_free_blocks_count++

	sb.S_first_blo -= sb.S_block_size
}

func (sb *Superbloque) UpdateSuperblockAfterInodeDeallocation() {
	sb.S_inodes_count--

	sb.S_free_inodes_count++

	sb.S_first_ino -= sb.S_inode_size
}

func (sb *Superbloque) JournalStart() int32 {
	journalSize := int32(binary.Size(Journal{}))
	start := sb.S_bm_inode_start - JOURNAL_ENTRIES*journalSize
	return start
}

func (sb *Superbloque) JournalEnd() int32 {
	end := sb.S_bm_inode_start
	return end
}
