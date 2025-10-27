package estructuras

import (
	"fmt"
	utilidades "godisk/Utilidades"
	"os"
	"time"
)

type Inodo struct {
	I_uid   int32
	I_gid   int32
	I_size  int32
	I_atime float32
	I_ctime float32
	I_mtime float32
	I_block [15]int32
	I_type  [1]byte
	I_perm  [3]byte
}

func (inodo *Inodo) Encode(file *os.File, offset int64) error {
	err := utilidades.EscribirEnArchivo(file, offset, inodo)
	if err != nil {
		return fmt.Errorf("error al escribir el inodo: %w", err)
	}
	return nil
}

func (inodo *Inodo) Decode(file *os.File, offset int64) error {
	err := utilidades.LeerDesdeArchivo(file, offset, inodo)
	if err != nil {
		return fmt.Errorf("error reading Inode from file: %w", err)
	}
	return nil
}

func (inodo *Inodo) ActualizarAtime() {
	inodo.I_atime = float32(time.Now().Unix())
}

func (inodo *Inodo) ActualizarMtime() {
	inodo.I_mtime = float32(time.Now().Unix())
}

func (inodo *Inodo) ActualizarCtime() {
	inodo.I_ctime = float32(time.Now().Unix())
}

func (inodo *Inodo) Print() {
	atime := time.Unix(int64(inodo.I_atime), 0)
	ctime := time.Unix(int64(inodo.I_ctime), 0)
	mtime := time.Unix(int64(inodo.I_mtime), 0)

	fmt.Printf("I_uid: %d\n", inodo.I_uid)
	fmt.Printf("I_gid: %d\n", inodo.I_gid)
	fmt.Printf("I_size: %d\n", inodo.I_size)
	fmt.Printf("I_atime: %s\n", atime.Format(time.RFC3339))
	fmt.Printf("I_ctime: %s\n", ctime.Format(time.RFC3339))
	fmt.Printf("I_mtime: %s\n", mtime.Format(time.RFC3339))
	fmt.Printf("I_block: %v\n", inodo.I_block)
	fmt.Printf("I_type: %s\n", string(inodo.I_type[:]))
	fmt.Printf("I_perm: %s\n", string(inodo.I_perm[:]))
}

func (inode *Inodo) GetAllBlockIndexes(file *os.File, sb *Superbloque) ([]int32, error) {
	var blockIndexes []int32

	for i := 0; i < 12; i++ {
		if inode.I_block[i] != -1 {
			blockIndexes = append(blockIndexes, inode.I_block[i])
		}
	}

	if inode.I_block[12] != -1 {
		blockIndexes = append(blockIndexes, inode.I_block[12])

		pb := &PointerBlock{}
		pbOffset := int64(sb.S_block_start + inode.I_block[12]*sb.S_block_size)
		err := pb.Decode(file, pbOffset)
		if err != nil {
			return nil, fmt.Errorf("error leyendo bloque indirecto simple: %w", err)
		}

		for _, pointer := range pb.B_pointers {
			if pointer != -1 {
				blockIndexes = append(blockIndexes, int32(pointer))
			}
		}
	}

	if inode.I_block[13] != -1 {
		blockIndexes = append(blockIndexes, inode.I_block[13])

		primPB := &PointerBlock{}
		primOffset := int64(sb.S_block_start + inode.I_block[13]*sb.S_block_size)
		err := primPB.Decode(file, primOffset)
		if err != nil {
			return nil, fmt.Errorf("error leyendo bloque indirecto doble: %w", err)
		}

		for _, primPointer := range primPB.B_pointers {
			if primPointer != -1 {
				blockIndexes = append(blockIndexes, int32(primPointer))

				secPB := &PointerBlock{}
				secOffset := int64(sb.S_block_start + int32(primPointer)*sb.S_block_size)
				err := secPB.Decode(file, secOffset)
				if err != nil {
					return nil, fmt.Errorf("error leyendo bloque secundario: %w", err)
				}

				for _, secPointer := range secPB.B_pointers {
					if secPointer != -1 {
						blockIndexes = append(blockIndexes, int32(secPointer))
					}
				}
			}
		}
	}

	if inode.I_block[14] != -1 {
		blockIndexes = append(blockIndexes, inode.I_block[14])
		primPB := &PointerBlock{}
		primOffset := int64(sb.S_block_start + inode.I_block[14]*sb.S_block_size)
		err := primPB.Decode(file, primOffset)
		if err != nil {
			return nil, fmt.Errorf("error leyendo bloque indirecto triple: %w", err)
		}

		for _, primPointer := range primPB.B_pointers {
			if primPointer != -1 {
				blockIndexes = append(blockIndexes, int32(primPointer))

				secPB := &PointerBlock{}
				secOffset := int64(sb.S_block_start + int32(primPointer)*sb.S_block_size)
				err := secPB.Decode(file, secOffset)
				if err != nil {
					return nil, fmt.Errorf("error leyendo bloque secundario en triple: %w", err)
				}

				for _, secPointer := range secPB.B_pointers {
					if secPointer != -1 {
						blockIndexes = append(blockIndexes, int32(secPointer))
						tercPB := &PointerBlock{}
						tercOffset := int64(sb.S_block_start + int32(secPointer)*sb.S_block_size)
						err := tercPB.Decode(file, tercOffset)
						if err != nil {
							return nil, fmt.Errorf("error leyendo bloque terciario: %w", err)
						}

						for _, tercPointer := range tercPB.B_pointers {
							if tercPointer != -1 {
								blockIndexes = append(blockIndexes, int32(tercPointer))
							}
						}
					}
				}
			}
		}
	}

	return blockIndexes, nil
}

func (inode *Inodo) AddBlock(file *os.File, sb *Superbloque) (int32, error) {
	for i := 0; i < 12; i++ {
		if inode.I_block[i] == -1 {
			newBlock, err := sb.AssignNewBlock(file, inode, i)
			if err != nil {
				return -1, fmt.Errorf("error al asignar bloque directo: %w", err)
			}

			inode.ActualizarMtime()
			return newBlock, nil
		}
	}

	return inode.AddBlockWithIndirection(file, sb)
}

func (inode *Inodo) AddBlockWithIndirection(file *os.File, sb *Superbloque) (int32, error) {
	if inode.I_block[12] == -1 {
		pointerBlockIndex, err := sb.AssignNewBlock(file, inode, 12)
		if err != nil {
			return -1, fmt.Errorf("error al crear bloque de apuntadores simple: %w", err)
		}

		pb := &PointerBlock{}
		for i := range pb.B_pointers {
			pb.B_pointers[i] = -1
		}

		pbOffset := int64(sb.S_block_start + pointerBlockIndex*sb.S_block_size)
		if err := pb.Encode(file, pbOffset); err != nil {
			return -1, fmt.Errorf("error al escribir bloque de apuntadores simple: %w", err)
		}
	}

	dataBlockIndex, err := inode.AddBlockToSimpleIndirect(file, sb)
	if err == nil {
		return dataBlockIndex, nil
	}

	if inode.I_block[13] == -1 {
		doublePointerBlockIndex, err := sb.AssignNewBlock(file, inode, 13)
		if err != nil {
			return -1, fmt.Errorf("error al crear bloque de apuntadores doble: %w", err)
		}

		dpb := &PointerBlock{}
		for i := range dpb.B_pointers {
			dpb.B_pointers[i] = -1
		}

		dpbOffset := int64(sb.S_block_start + doublePointerBlockIndex*sb.S_block_size)
		if err := dpb.Encode(file, dpbOffset); err != nil {
			return -1, fmt.Errorf("error al escribir bloque de apuntadores doble: %w", err)
		}
	}

	dataBlockIndex, err = inode.AddBlockToDoubleIndirect(file, sb)
	if err == nil {
		return dataBlockIndex, nil
	}

	if inode.I_block[14] == -1 {
		triplePointerBlockIndex, err := sb.AssignNewBlock(file, inode, 14)
		if err != nil {
			return -1, fmt.Errorf("error al crear bloque de apuntadores triple: %w", err)
		}

		tpb := &PointerBlock{}
		for i := range tpb.B_pointers {
			tpb.B_pointers[i] = -1
		}

		tpbOffset := int64(sb.S_block_start + triplePointerBlockIndex*sb.S_block_size)
		if err := tpb.Encode(file, tpbOffset); err != nil {
			return -1, fmt.Errorf("error al escribir bloque de apuntadores triple: %w", err)
		}
	}

	dataBlockIndex, err = inode.AddBlockToTripleIndirect(file, sb)
	if err == nil {
		return dataBlockIndex, nil
	}

	return -1, fmt.Errorf("no hay espacio disponible para agregar más bloques al inodo")
}

func (inode *Inodo) AddBlockToSimpleIndirect(file *os.File, sb *Superbloque) (int32, error) {
	if inode.I_block[12] == -1 {
		return -1, fmt.Errorf("no existe bloque indirecto simple")
	}

	pb := &PointerBlock{}
	pbOffset := int64(sb.S_block_start + inode.I_block[12]*sb.S_block_size)
	if err := pb.Decode(file, pbOffset); err != nil {
		return -1, fmt.Errorf("error al leer bloque de apuntadores: %w", err)
	}

	freeIndex, err := pb.FindFreePointer()
	if err != nil {
		return -1, fmt.Errorf("bloque indirecto simple lleno: %w", err)
	}

	newBlockIndex, err := sb.FindNextFreeBlock(file)
	if err != nil {
		return -1, fmt.Errorf("error buscando bloque libre: %w", err)
	}

	if err := sb.UpdateBitmapBlock(file, newBlockIndex, true); err != nil {
		return -1, fmt.Errorf("error actualizando bitmap: %w", err)
	}

	zeroBuffer := make([]byte, sb.S_block_size)
	blockOffset := int64(sb.S_block_start + newBlockIndex*sb.S_block_size)
	if _, err := file.WriteAt(zeroBuffer, blockOffset); err != nil {
		return -1, fmt.Errorf("error inicializando bloque nuevo: %w", err)
	}

	pb.B_pointers[freeIndex] = int32(newBlockIndex)

	if err := pb.Encode(file, pbOffset); err != nil {
		return -1, fmt.Errorf("error escribiendo bloque de apuntadores: %w", err)
	}

	sb.UpdateSuperblockAfterBlockAllocation()

	return newBlockIndex, nil
}

func (inode *Inodo) AddBlockToDoubleIndirect(file *os.File, sb *Superbloque) (int32, error) {
	if inode.I_block[13] == -1 {
		return -1, fmt.Errorf("no existe bloque indirecto doble")
	}

	primPB := &PointerBlock{}
	primOffset := int64(sb.S_block_start + inode.I_block[13]*sb.S_block_size)
	if err := primPB.Decode(file, primOffset); err != nil {
		return -1, fmt.Errorf("error al leer bloque de apuntadores primario: %w", err)
	}

	for i, primPointer := range primPB.B_pointers {
		if primPointer == -1 {
			newSecondaryBlockIndex, err := sb.FindNextFreeBlock(file)
			if err != nil {
				return -1, fmt.Errorf("error buscando bloque libre para apuntadores secundario: %w", err)
			}

			if err := sb.UpdateBitmapBlock(file, newSecondaryBlockIndex, true); err != nil {
				return -1, fmt.Errorf("error actualizando bitmap: %w", err)
			}

			secPB := &PointerBlock{}
			for j := range secPB.B_pointers {
				secPB.B_pointers[j] = -1
			}
			secOffset := int64(sb.S_block_start + newSecondaryBlockIndex*sb.S_block_size)
			if err := secPB.Encode(file, secOffset); err != nil {
				return -1, fmt.Errorf("error escribiendo bloque de apuntadores secundario: %w", err)
			}

			primPB.B_pointers[i] = int32(newSecondaryBlockIndex)
			if err := primPB.Encode(file, primOffset); err != nil {
				return -1, fmt.Errorf("error actualizando bloque de apuntadores primario: %w", err)
			}

			newDataBlockIndex, err := sb.FindNextFreeBlock(file)
			if err != nil {
				return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
			}

			if err := sb.UpdateBitmapBlock(file, newDataBlockIndex, true); err != nil {
				return -1, fmt.Errorf("error actualizando bitmap: %w", err)
			}

			secPB.B_pointers[0] = int32(newDataBlockIndex)
			if err := secPB.Encode(file, secOffset); err != nil {
				return -1, fmt.Errorf("error actualizando bloque de apuntadores secundario: %w", err)
			}

			sb.UpdateSuperblockAfterBlockAllocation()
			sb.UpdateSuperblockAfterBlockAllocation()

			return newDataBlockIndex, nil
		} else {
			secPB := &PointerBlock{}
			secOffset := int64(sb.S_block_start + int32(primPointer)*sb.S_block_size)
			if err := secPB.Decode(file, secOffset); err != nil {
				return -1, fmt.Errorf("error leyendo bloque de apuntadores secundario: %w", err)
			}

			freeSecIndex, err := secPB.FindFreePointer()
			if err != nil {
				continue
			}

			newDataBlockIndex, err := sb.FindNextFreeBlock(file)
			if err != nil {
				return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
			}

			if err := sb.UpdateBitmapBlock(file, newDataBlockIndex, true); err != nil {
				return -1, fmt.Errorf("error actualizando bitmap: %w", err)
			}

			secPB.B_pointers[freeSecIndex] = int32(newDataBlockIndex)
			if err := secPB.Encode(file, secOffset); err != nil {
				return -1, fmt.Errorf("error actualizando bloque de apuntadores secundario: %w", err)
			}

			sb.UpdateSuperblockAfterBlockAllocation()

			return newDataBlockIndex, nil
		}
	}

	return -1, fmt.Errorf("bloque de apuntadores primario lleno")
}

func (inode *Inodo) AddBlockToTripleIndirect(file *os.File, sb *Superbloque) (int32, error) {
	if inode.I_block[14] == -1 {
		return -1, fmt.Errorf("no existe bloque indirecto triple")
	}

	primPB := &PointerBlock{}
	primOffset := int64(sb.S_block_start + inode.I_block[14]*sb.S_block_size)
	if err := primPB.Decode(file, primOffset); err != nil {
		return -1, fmt.Errorf("error al leer bloque de apuntadores primario: %w", err)
	}

	for primIndex, primPointer := range primPB.B_pointers {
		if primPointer == -1 {
			newSecBlockIndex, err := sb.FindNextFreeBlock(file)
			if err != nil {
				return -1, fmt.Errorf("error buscando bloque libre para apuntadores secundario: %w", err)
			}

			if err := sb.UpdateBitmapBlock(file, newSecBlockIndex, true); err != nil {
				return -1, fmt.Errorf("error actualizando bitmap: %w", err)
			}

			secPB := &PointerBlock{}
			for j := range secPB.B_pointers {
				secPB.B_pointers[j] = -1
			}

			newTercBlockIndex, err := sb.FindNextFreeBlock(file)
			if err != nil {
				return -1, fmt.Errorf("error buscando bloque libre para apuntadores terciario: %w", err)
			}

			if err := sb.UpdateBitmapBlock(file, newTercBlockIndex, true); err != nil {
				return -1, fmt.Errorf("error actualizando bitmap: %w", err)
			}

			tercPB := &PointerBlock{}
			for j := range tercPB.B_pointers {
				tercPB.B_pointers[j] = -1
			}

			newDataBlockIndex, err := sb.FindNextFreeBlock(file)
			if err != nil {
				return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
			}

			if err := sb.UpdateBitmapBlock(file, newDataBlockIndex, true); err != nil {
				return -1, fmt.Errorf("error actualizando bitmap: %w", err)
			}

			tercPB.B_pointers[0] = int32(newDataBlockIndex)

			tercOffset := int64(sb.S_block_start + newTercBlockIndex*sb.S_block_size)
			if err := tercPB.Encode(file, tercOffset); err != nil {
				return -1, fmt.Errorf("error escribiendo bloque de apuntadores terciario: %w", err)
			}

			secPB.B_pointers[0] = int32(newTercBlockIndex)

			secOffset := int64(sb.S_block_start + newSecBlockIndex*sb.S_block_size)
			if err := secPB.Encode(file, secOffset); err != nil {
				return -1, fmt.Errorf("error escribiendo bloque de apuntadores secundario: %w", err)
			}

			primPB.B_pointers[primIndex] = int32(newSecBlockIndex)

			if err := primPB.Encode(file, primOffset); err != nil {
				return -1, fmt.Errorf("error actualizando bloque de apuntadores primario: %w", err)
			}

			sb.UpdateSuperblockAfterBlockAllocation()
			sb.UpdateSuperblockAfterBlockAllocation()
			sb.UpdateSuperblockAfterBlockAllocation()

			return newDataBlockIndex, nil
		} else {
			secPB := &PointerBlock{}
			secOffset := int64(sb.S_block_start + int32(primPointer)*sb.S_block_size)
			if err := secPB.Decode(file, secOffset); err != nil {
				return -1, fmt.Errorf("error leyendo bloque de apuntadores secundario: %w", err)
			}

			for secIndex, secPointer := range secPB.B_pointers {
				if secPointer == -1 {
					newTercBlockIndex, err := sb.FindNextFreeBlock(file)
					if err != nil {
						return -1, fmt.Errorf("error buscando bloque libre para apuntadores terciario: %w", err)
					}

					if err := sb.UpdateBitmapBlock(file, newTercBlockIndex, true); err != nil {
						return -1, fmt.Errorf("error actualizando bitmap: %w", err)
					}

					tercPB := &PointerBlock{}
					for j := range tercPB.B_pointers {
						tercPB.B_pointers[j] = -1
					}

					newDataBlockIndex, err := sb.FindNextFreeBlock(file)
					if err != nil {
						return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
					}

					if err := sb.UpdateBitmapBlock(file, newDataBlockIndex, true); err != nil {
						return -1, fmt.Errorf("error actualizando bitmap: %w", err)
					}

					tercPB.B_pointers[0] = int32(newDataBlockIndex)

					tercOffset := int64(sb.S_block_start + newTercBlockIndex*sb.S_block_size)
					if err := tercPB.Encode(file, tercOffset); err != nil {
						return -1, fmt.Errorf("error escribiendo bloque de apuntadores terciario: %w", err)
					}

					secPB.B_pointers[secIndex] = int32(newTercBlockIndex)

					if err := secPB.Encode(file, secOffset); err != nil {
						return -1, fmt.Errorf("error actualizando bloque de apuntadores secundario: %w", err)
					}

					sb.UpdateSuperblockAfterBlockAllocation()
					sb.UpdateSuperblockAfterBlockAllocation()

					return newDataBlockIndex, nil
				} else {
					tercPB := &PointerBlock{}
					tercOffset := int64(sb.S_block_start + int32(secPointer)*sb.S_block_size)
					if err := tercPB.Decode(file, tercOffset); err != nil {
						return -1, fmt.Errorf("error leyendo bloque de apuntadores terciario: %w", err)
					}

					freeTercIndex, err := tercPB.FindFreePointer()
					if err != nil {
						continue
					}

					newDataBlockIndex, err := sb.FindNextFreeBlock(file)
					if err != nil {
						return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
					}

					if err := sb.UpdateBitmapBlock(file, newDataBlockIndex, true); err != nil {
						return -1, fmt.Errorf("error actualizando bitmap: %w", err)
					}

					tercPB.B_pointers[freeTercIndex] = int32(newDataBlockIndex)

					if err := tercPB.Encode(file, tercOffset); err != nil {
						return -1, fmt.Errorf("error actualizando bloque de apuntadores terciario: %w", err)
					}
					sb.UpdateSuperblockAfterBlockAllocation()

					return newDataBlockIndex, nil
				}
			}
		}
	}

	return -1, fmt.Errorf("todos los bloques de apuntadores de indirección triple están llenos")
}

func (inode *Inodo) FreeBlock(file *os.File, sb *Superbloque, blockIndex int32) error {
	if err := sb.UpdateBitmapBlock(file, blockIndex, false); err != nil {
		return fmt.Errorf("error liberando bloque %d: %w", blockIndex, err)
	}
	sb.UpdateSuperblockAfterBlockDeallocation()
	return nil
}

func (inode *Inodo) FreeAllBlocks(file *os.File, sb *Superbloque) error {
	// Obtener todos los bloques
	blocks, err := inode.GetAllBlockIndexes(file, sb)
	if err != nil {
		return err
	}

	for _, blockIndex := range blocks {
		if err := inode.FreeBlock(file, sb, blockIndex); err != nil {
			return err
		}
	}

	for i := range inode.I_block {
		inode.I_block[i] = -1
	}

	inode.I_size = 0
	inode.ActualizarMtime()

	return nil
}

func (inode *Inodo) CheckAndFreeEmptyIndirectBlocks(file *os.File, sb *Superbloque) error {
	if inode.I_block[12] != -1 {
		pb := &PointerBlock{}
		pbOffset := int64(sb.S_block_start + inode.I_block[12]*sb.S_block_size)
		if err := pb.Decode(file, pbOffset); err != nil {
			return fmt.Errorf("error leyendo bloque indirecto simple: %w", err)
		}

		isEmpty := true
		for _, pointer := range pb.B_pointers {
			if pointer != -1 {
				isEmpty = false
				break
			}
		}

		if isEmpty {
			fmt.Printf("Liberando bloque de apuntadores simple %d (vacío)\n", inode.I_block[12])
			if err := inode.FreeBlock(file, sb, inode.I_block[12]); err != nil {
				return err
			}
			inode.I_block[12] = -1
		}
	}

	if inode.I_block[13] != -1 {
		primPB := &PointerBlock{}
		primOffset := int64(sb.S_block_start + inode.I_block[13]*sb.S_block_size)
		if err := primPB.Decode(file, primOffset); err != nil {
			return fmt.Errorf("error leyendo bloque indirecto doble: %w", err)
		}

		emptySecondaryBlocks := make([]int, 0)
		allEmpty := true

		for i, primPointer := range primPB.B_pointers {
			if primPointer != -1 {
				secPB := &PointerBlock{}
				secOffset := int64(sb.S_block_start + int32(primPointer)*sb.S_block_size)
				if err := secPB.Decode(file, secOffset); err != nil {
					return fmt.Errorf("error leyendo bloque secundario: %w", err)
				}

				isEmpty := true
				for _, secPointer := range secPB.B_pointers {
					if secPointer != -1 {
						isEmpty = false
						break
					}
				}

				if isEmpty {
					emptySecondaryBlocks = append(emptySecondaryBlocks, i)
					fmt.Printf("Marcando bloque secundario %d para liberación (vacío)\n", primPointer)
				} else {
					allEmpty = false
				}
			}
		}

		for _, idx := range emptySecondaryBlocks {
			secBlockIndex := primPB.B_pointers[idx]
			// Liberar el bloque
			if err := inode.FreeBlock(file, sb, secBlockIndex); err != nil {
				return err
			}
			primPB.B_pointers[idx] = -1
		}

		if len(emptySecondaryBlocks) > 0 {
			if err := primPB.Encode(file, primOffset); err != nil {
				return fmt.Errorf("error actualizando bloque primario: %w", err)
			}
		}

		if allEmpty {
			fmt.Printf("Liberando bloque de apuntadores doble %d (vacío)\n", inode.I_block[13])
			if err := inode.FreeBlock(file, sb, inode.I_block[13]); err != nil {
				return err
			}
			inode.I_block[13] = -1
		}
	}

	if inode.I_block[14] != -1 {
		primPB := &PointerBlock{}
		primOffset := int64(sb.S_block_start + inode.I_block[14]*sb.S_block_size)
		if err := primPB.Decode(file, primOffset); err != nil {
			return fmt.Errorf("error leyendo bloque indirecto triple: %w", err)
		}

		primEmptyCount := 0
		allPrimEmpty := true

		for primIdx, primPointer := range primPB.B_pointers {
			if primPointer == -1 {
				primEmptyCount++
				continue
			}

			secPB := &PointerBlock{}
			secOffset := int64(sb.S_block_start + int32(primPointer)*sb.S_block_size)
			if err := secPB.Decode(file, secOffset); err != nil {
				return fmt.Errorf("error leyendo bloque secundario en triple: %w", err)
			}

			secEmptyCount := 0
			allSecEmpty := true

			for secIdx, secPointer := range secPB.B_pointers {
				if secPointer == -1 {
					secEmptyCount++
					continue
				}

				tercPB := &PointerBlock{}
				tercOffset := int64(sb.S_block_start + int32(secPointer)*sb.S_block_size)
				if err := tercPB.Decode(file, tercOffset); err != nil {
					return fmt.Errorf("error leyendo bloque terciario: %w", err)
				}

				isEmpty := true
				for _, tercPointer := range tercPB.B_pointers {
					if tercPointer != -1 {
						isEmpty = false
						break
					}
				}

				if isEmpty {
					fmt.Printf("Liberando bloque terciario %d (vacío)\n", secPointer)
					if err := inode.FreeBlock(file, sb, secPointer); err != nil {
						return err
					}
					secPB.B_pointers[secIdx] = -1
				} else {
					allSecEmpty = false
				}
			}

			if allSecEmpty {
				fmt.Printf("Liberando bloque secundario %d en triple (vacío)\n", primPointer)
				if err := inode.FreeBlock(file, sb, primPointer); err != nil {
					return err
				}
				primPB.B_pointers[primIdx] = -1
			} else {
				if secEmptyCount > 0 && secEmptyCount < len(secPB.B_pointers) {
					if err := secPB.Encode(file, secOffset); err != nil {
						return fmt.Errorf("error actualizando bloque secundario: %w", err)
					}
				}
				allPrimEmpty = false
			}
		}

		if !allPrimEmpty && primEmptyCount > 0 {
			if err := primPB.Encode(file, primOffset); err != nil {
				return fmt.Errorf("error actualizando bloque primario triple: %w", err)
			}
		}

		if allPrimEmpty {
			fmt.Printf("Liberando bloque de apuntadores triple %d (vacío)\n", inode.I_block[14])
			if err := inode.FreeBlock(file, sb, inode.I_block[14]); err != nil {
				return err
			}
			inode.I_block[14] = -1
		}
	}

	return nil
}

func (inode *Inodo) ReadData(file *os.File, sb *Superbloque) ([]byte, error) {
	blockIndexes, err := inode.GetDataBlockIndexes(file, sb)
	if err != nil {
		return nil, err
	}

	bytesToRead := int(inode.I_size)
	result := make([]byte, 0, bytesToRead)

	for _, blockIndex := range blockIndexes {
		if bytesToRead <= 0 {
			break
		}

		fileBlock := &ArchivoBloque{}
		blockOffset := int64(sb.S_block_start + blockIndex*sb.S_block_size)
		if err := fileBlock.Decode(file, blockOffset); err != nil {
			return nil, err
		}
		bytesFromBlock := BlockSize
		if bytesFromBlock > bytesToRead {
			bytesFromBlock = bytesToRead
		}

		result = append(result, fileBlock.B_content[:bytesFromBlock]...)
		bytesToRead -= bytesFromBlock
	}

	return result, nil
}

func (inode *Inodo) WriteData(file *os.File, sb *Superbloque, data []byte) error {
	oldSize := inode.I_size
	newSize := int32(len(data))

	if oldSize != newSize {
		if oldSize > 0 {
			if err := inode.FreeAllBlocks(file, sb); err != nil {
				return fmt.Errorf("error liberando bloques existentes: %w", err)
			}
		}

		if newSize == 0 {
			inode.I_size = 0
			inode.ActualizarMtime()
			return nil
		}

		blocksNeeded := (newSize + sb.S_block_size - 1) / sb.S_block_size

		for i := int32(0); i < blocksNeeded; i++ {
			_, err := inode.AddBlock(file, sb)
			if err != nil {
				return fmt.Errorf("error asignando bloque %d al redimensionar: %w", i, err)
			}
		}
	}

	if len(data) == 0 {
		return nil
	}

	blockIndexes, err := inode.GetDataBlockIndexes(file, sb)
	if err != nil {
		return fmt.Errorf("error obteniendo bloques de datos: %w", err)
	}

	expectedBlocks := (newSize + sb.S_block_size - 1) / sb.S_block_size
	if int32(len(blockIndexes)) < expectedBlocks {
		return fmt.Errorf("faltan bloques para escribir datos: tiene %d, necesita %d",
			len(blockIndexes), expectedBlocks)
	}

	dataOffset := 0
	for _, blockIndex := range blockIndexes {
		blockOffset := int64(sb.S_block_start + blockIndex*sb.S_block_size)

		bytesToWrite := int(sb.S_block_size)
		if dataOffset+bytesToWrite > len(data) {
			bytesToWrite = len(data) - dataOffset
		}

		if bytesToWrite <= 0 {
			break
		}

		if _, err := file.WriteAt(data[dataOffset:dataOffset+bytesToWrite], blockOffset); err != nil {
			return fmt.Errorf("error escribiendo datos al bloque %d: %w", blockIndex, err)
		}

		dataOffset += bytesToWrite
		if dataOffset >= len(data) {
			break
		}
	}

	inode.I_size = newSize
	inode.ActualizarMtime()

	return nil
}

func (inode *Inodo) GetDataBlockIndexes(file *os.File, sb *Superbloque) ([]int32, error) {
	dataBlocks := []int32{}
	for i := 0; i < 12; i++ {
		if inode.I_block[i] != -1 {
			dataBlocks = append(dataBlocks, inode.I_block[i])
		}
	}

	if inode.I_block[12] != -1 {
		pb := &PointerBlock{}
		pbOffset := int64(sb.S_block_start + inode.I_block[12]*sb.S_block_size)
		if err := pb.Decode(file, pbOffset); err != nil {
			return nil, fmt.Errorf("error leyendo bloque indirecto simple: %w", err)
		}

		for _, pointer := range pb.B_pointers {
			if pointer != -1 {
				dataBlocks = append(dataBlocks, pointer)
			}
		}
	}

	if inode.I_block[13] != -1 {
		primPB := &PointerBlock{}
		primOffset := int64(sb.S_block_start + inode.I_block[13]*sb.S_block_size)
		if err := primPB.Decode(file, primOffset); err != nil {
			return nil, fmt.Errorf("error leyendo bloque indirecto doble: %w", err)
		}

		for _, primPointer := range primPB.B_pointers {
			if primPointer != -1 {
				secPB := &PointerBlock{}
				secOffset := int64(sb.S_block_start + primPointer*sb.S_block_size)
				if err := secPB.Decode(file, secOffset); err != nil {
					return nil, fmt.Errorf("error leyendo bloque secundario: %w", err)
				}

				for _, secPointer := range secPB.B_pointers {
					if secPointer != -1 {
						dataBlocks = append(dataBlocks, secPointer)
					}
				}
			}
		}
	}

	if inode.I_block[14] != -1 {
		primPB := &PointerBlock{}
		primOffset := int64(sb.S_block_start + inode.I_block[14]*sb.S_block_size)
		if err := primPB.Decode(file, primOffset); err != nil {
			return nil, fmt.Errorf("error leyendo bloque indirecto triple: %w", err)
		}

		for _, primPointer := range primPB.B_pointers {
			if primPointer != -1 {
				secPB := &PointerBlock{}
				secOffset := int64(sb.S_block_start + primPointer*sb.S_block_size)
				if err := secPB.Decode(file, secOffset); err != nil {
					return nil, fmt.Errorf("error leyendo bloque secundario en indirección triple: %w", err)
				}

				for _, secPointer := range secPB.B_pointers {
					if secPointer != -1 {
						tercPB := &PointerBlock{}
						tercOffset := int64(sb.S_block_start + secPointer*sb.S_block_size)
						if err := tercPB.Decode(file, tercOffset); err != nil {
							return nil, fmt.Errorf("error leyendo bloque terciario: %w", err)
						}

						for _, tercPointer := range tercPB.B_pointers {
							if tercPointer != -1 {
								dataBlocks = append(dataBlocks, tercPointer)
							}
						}
					}
				}
			}
		}
	}

	return dataBlocks, nil
}

func NewEmptyInode() *Inodo {
	in := &Inodo{}

	in.I_uid = 1
	in.I_gid = 1
	in.I_size = 0
	now := float32(time.Now().Unix())
	in.I_atime = now
	in.I_ctime = now
	in.I_mtime = now

	in.I_type[0] = 0
	in.I_perm = [3]byte{'0', '0', '0'}

	for i := range in.I_block {
		in.I_block[i] = -1
	}
	return in
}

func (inode *Inodo) CreateInode(
	file *os.File,
	sb *Superbloque,
	inodeType byte,
	size int32,
	blocks [15]int32,
	permissions [3]byte,
) error {
	inodeIndex, err := sb.AssignNewInode(file)
	if err != nil {
		return fmt.Errorf("error asignando nuevo inodo: %w", err)
	}

	inode.I_uid = 1
	inode.I_gid = 1
	inode.I_size = size
	inode.I_atime = float32(time.Now().Unix())
	inode.I_ctime = float32(time.Now().Unix())
	inode.I_mtime = float32(time.Now().Unix())
	inode.I_block = blocks
	inode.I_type = [1]byte{inodeType}
	inode.I_perm = permissions

	err = sb.UpdateBitmapInode(file, inodeIndex, true)
	if err != nil {
		return fmt.Errorf("error actualizando el bitmap de inodos: %w", err)
	}

	inodeOffset := int64(sb.S_inode_start + (inodeIndex * sb.S_inode_size))
	err = inode.Encode(file, inodeOffset)
	if err != nil {
		return fmt.Errorf("error serializando el inodo en la ubicación %d: %w", inodeOffset, err)
	}

	return nil
}
