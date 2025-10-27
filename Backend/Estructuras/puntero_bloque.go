package estructuras

import (
	"encoding/binary"
	"fmt"
	"os"
)

type PointerBlock struct {
	B_pointers [16]int32
}

func (pb *PointerBlock) ReadSimpleIndirect(file *os.File, sb *Superbloque) ([]int32, error) {
	var blocks []int32
	for _, pointer := range pb.B_pointers {
		if pointer != -1 {
			blocks = append(blocks, int32(pointer))
		}
	}
	return blocks, nil
}

func (pb *PointerBlock) ReadDoubleIndirect(file *os.File, sb *Superbloque) ([]int32, error) {
	var blocks []int32
	for _, pointer := range pb.B_pointers {
		if pointer != -1 {
			secondaryPB := &PointerBlock{}
			err := secondaryPB.Decode(file, int64(sb.S_block_start+int32(pointer)*sb.S_block_size))
			if err != nil {
				return nil, err
			}
			secondaryBlocks, err := secondaryPB.ReadSimpleIndirect(file, sb)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, secondaryBlocks...)
		}
	}
	return blocks, nil
}

func (pb *PointerBlock) FindFreePointer() (int, error) {
	for i, pointer := range pb.B_pointers {
		if pointer == -1 {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no hay apuntadores libres en el bloque de apuntadores")
}

func (pb *PointerBlock) IsEmpty() bool {
	return pb.CountFreePointers() == len(pb.B_pointers)
}

func (pb *PointerBlock) FreeIfEmpty(file *os.File, sb *Superbloque, blockIndex int32, parentInode *Inodo, pointerIndex int) error {
	if pb.IsEmpty() {
		if err := sb.UpdateBitmapBlock(file, blockIndex, false); err != nil {
			return err
		}

		if parentInode != nil && pointerIndex >= 0 {
			parentInode.I_block[pointerIndex] = -1
			return parentInode.Encode(file, sb.CalculateInodeOffset(parentInode.I_uid))
		}

		sb.UpdateSuperblockAfterBlockDeallocation()
	}
	return nil
}

func (pb *PointerBlock) SetPointer(index int, value int64) error {
	if index < 0 || index >= len(pb.B_pointers) {
		return fmt.Errorf("índice fuera de rango")
	}
	pb.B_pointers[index] = int32(value)
	return nil
}

func (pb *PointerBlock) GetPointer(index int) (int64, error) {
	if index < 0 || index >= len(pb.B_pointers) {
		return -1, fmt.Errorf("índice fuera de rango")
	}
	return int64(pb.B_pointers[index]), nil
}

func (pb *PointerBlock) IsFull() bool {
	for _, pointer := range pb.B_pointers {
		if pointer == -1 {
			return false
		}
	}
	return true
}

func (pb *PointerBlock) CountFreePointers() int {
	count := 0
	for _, pointer := range pb.B_pointers {
		if pointer == -1 {
			count++
		}
	}
	return count
}

func (pb *PointerBlock) Encode(file *os.File, offset int64) error {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return fmt.Errorf("error buscando la posición en el archivo: %w", err)
	}
	err = binary.Write(file, binary.BigEndian, *pb)
	if err != nil {
		return fmt.Errorf("error escribiendo el PointerBlock: %w", err)
	}
	return nil
}

func (pb *PointerBlock) Decode(file *os.File, offset int64) error {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return fmt.Errorf("error buscando la posición en el archivo: %w", err)
	}
	err = binary.Read(file, binary.BigEndian, pb)
	if err != nil {
		return fmt.Errorf("error leyendo el PointerBlock: %w", err)
	}
	return nil
}

func (pb *PointerBlock) ReadTripleIndirect(file *os.File, sb *Superbloque) ([]int32, error) {
	var blocks []int32
	for _, primPointer := range pb.B_pointers {
		if primPointer != -1 {
			secPB := &PointerBlock{}
			secOffset := int64(sb.S_block_start + int32(primPointer)*sb.S_block_size)
			if err := secPB.Decode(file, secOffset); err != nil {
				return nil, fmt.Errorf("error leyendo bloque secundario: %w", err)
			}

			for _, secPointer := range secPB.B_pointers {
				if secPointer != -1 {
					tercPB := &PointerBlock{}
					tercOffset := int64(sb.S_block_start + int32(secPointer)*sb.S_block_size)
					if err := tercPB.Decode(file, tercOffset); err != nil {
						return nil, fmt.Errorf("error leyendo bloque terciario: %w", err)
					}

					terBlocks, err := tercPB.ReadSimpleIndirect(file, sb)
					if err != nil {
						return nil, err
					}
					blocks = append(blocks, terBlocks...)
				}
			}
		}
	}
	return blocks, nil
}
