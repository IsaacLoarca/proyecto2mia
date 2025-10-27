package estructuras

import (
	"fmt"
	"os"
	"time"
)

func (sb *Superbloque) CreateUsersFile(file *os.File) error {
	rootInode := &Inodo{
		I_uid:   1,
		I_gid:   1,
		I_size:  0,
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{sb.S_blocks_count, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		I_type:  [1]byte{'0'},
		I_perm:  [3]byte{'7', '7', '7'},
	}

	err := rootInode.Encode(file, int64(sb.S_inode_start+0))
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
			{B_name: [12]byte{'.'}, B_inodo: 0},
			{B_name: [12]byte{'.', '.'}, B_inodo: 0},
			{B_name: [12]byte{'u', 's', 'e', 'r', 's', '.', 't', 'x', 't'}, B_inodo: sb.S_inodes_count},
			{B_name: [12]byte{'-'}, B_inodo: -1},
		},
	}

	err = rootBlock.Encode(file, int64(sb.S_block_start))
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

	err = usersInode.Encode(file, int64(sb.S_first_ino))
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

	err = usersBlock.Encode(file, int64(sb.S_first_blo))
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
