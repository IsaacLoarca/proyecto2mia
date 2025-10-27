package estructuras

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	FreeBlockBit     = 0
	OccupiedBlockBit = 1
)

func (sb *Superbloque) CreateBitMaps(file *os.File) error {
	err := sb.createBitmap(file, sb.S_bm_inode_start, sb.S_inodes_count+sb.S_free_inodes_count, false)
	if err != nil {
		return fmt.Errorf("error creando bitmap de inodos: %w", err)
	}

	err = sb.createBitmap(file, sb.S_bm_block_start, sb.S_blocks_count+sb.S_free_blocks_count, false)
	if err != nil {
		return fmt.Errorf("error creando bitmap de bloques: %w", err)
	}

	return nil
}

func (sb *Superbloque) createBitmap(file *os.File, start int32, count int32, occupied bool) error {
	_, err := file.Seek(int64(start), 0)
	if err != nil {
		return fmt.Errorf("error buscando el inicio del bitmap: %w", err)
	}

	byteCount := (count + 7) / 8

	fillByte := byte(0x00)
	if occupied {
		fillByte = 0xFF
	}

	buffer := make([]byte, byteCount)
	for i := range buffer {
		buffer[i] = fillByte
	}

	err = binary.Write(file, binary.LittleEndian, buffer)
	if err != nil {
		return fmt.Errorf("error escribiendo el bitmap: %w", err)
	}

	return nil
}

func (sb *Superbloque) UpdateBitmapInode(file *os.File, position int32, occupied bool) error {
	return sb.updateBitmap(file, sb.S_bm_inode_start, position, occupied)
}

func (sb *Superbloque) UpdateBitmapBlock(file *os.File, position int32, occupied bool) error {
	return sb.updateBitmap(file, sb.S_bm_block_start, position, occupied)
}

func (sb *Superbloque) updateBitmap(file *os.File, start int32, position int32, occupied bool) error {
	byteIndex := position / 8
	bitOffset := position % 8

	_, err := file.Seek(int64(start)+int64(byteIndex), 0)
	if err != nil {
		return fmt.Errorf("error buscando la posición en el bitmap: %w", err)
	}

	var byteVal byte
	err = binary.Read(file, binary.LittleEndian, &byteVal)
	if err != nil {
		return fmt.Errorf("error leyendo el byte del bitmap: %w", err)
	}

	if occupied {
		byteVal |= (1 << bitOffset)
	} else {
		byteVal &= ^(1 << bitOffset)
	}

	_, err = file.Seek(int64(start)+int64(byteIndex), 0)
	if err != nil {
		return fmt.Errorf("error buscando la posición en el bitmap para escribir: %w", err)
	}

	err = binary.Write(file, binary.LittleEndian, &byteVal)
	if err != nil {
		return fmt.Errorf("error escribiendo el byte actualizado del bitmap: %w", err)
	}

	return nil
}

func (sb *Superbloque) isBlockFree(file *os.File, start int32, position int32) (bool, error) {
	byteIndex := position / 8
	bitOffset := position % 8

	_, err := file.Seek(int64(start)+int64(byteIndex), 0)
	if err != nil {
		return false, fmt.Errorf("error buscando la posición en el bitmap: %w", err)
	}

	var byteVal byte
	err = binary.Read(file, binary.LittleEndian, &byteVal)
	if err != nil {
		return false, fmt.Errorf("error leyendo el byte del bitmap: %w", err)
	}

	return (byteVal & (1 << bitOffset)) == 0, nil
}

func (sb *Superbloque) isInodeFree(file *os.File, start int32, position int32) (bool, error) {
	byteIndex := position / 8
	bitOffset := position % 8

	_, err := file.Seek(int64(start)+int64(byteIndex), 0)
	if err != nil {
		return false, fmt.Errorf("error buscando el byte en el bitmap de inodos: %w", err)
	}

	var byteVal byte
	err = binary.Read(file, binary.LittleEndian, &byteVal)
	if err != nil {
		return false, fmt.Errorf("error leyendo el byte del bitmap de inodos: %w", err)
	}

	return (byteVal & (1 << bitOffset)) == 0, nil
}
