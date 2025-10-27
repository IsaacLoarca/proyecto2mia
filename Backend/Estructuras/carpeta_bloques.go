package estructuras

import (
	"fmt"
	utilidades "godisk/Utilidades"
	"os"
	"strings"
)

type FolderBlock struct {
	B_content [4]FolderContent
}

type FolderContent struct {
	B_name  [12]byte
	B_inodo int32
}

func (fb *FolderBlock) Encode(file *os.File, offset int64) error {
	err := utilidades.EscribirEnArchivo(file, offset, fb)
	if err != nil {
		return fmt.Errorf("error writing FolderBlock to file: %w", err)
	}
	return nil
}

func (fb *FolderBlock) Decode(file *os.File, offset int64) error {
	err := utilidades.LeerDesdeArchivo(file, offset, fb)
	if err != nil {
		return fmt.Errorf("error reading FolderBlock from file: %w", err)
	}
	return nil
}

func (fb *FolderBlock) Print() {
	for i, content := range fb.B_content {
		name := string(content.B_name[:])
		fmt.Printf("Content %d:\n", i+1)
		fmt.Printf("  B_name: %s\n", name)
		fmt.Printf("  B_inodo: %d\n", content.B_inodo)
	}
}

func NewFolderBlock(selfInodo, parentInodo int32, additionalContents map[string]int32) *FolderBlock {
	fb := &FolderBlock{}

	copy(fb.B_content[0].B_name[:], ".")
	fb.B_content[0].B_inodo = selfInodo
	copy(fb.B_content[1].B_name[:], "..")
	fb.B_content[1].B_inodo = parentInodo
	i := 2
	for name, inodo := range additionalContents {
		if i >= len(fb.B_content) {
			break
		}

		copy(fb.B_content[i].B_name[:], name)
		fb.B_content[i].B_inodo = inodo
		i++
	}

	for ; i < len(fb.B_content); i++ {
		copy(fb.B_content[i].B_name[:], "-")
		fb.B_content[i].B_inodo = -1
	}

	return fb
}

func (fb *FolderBlock) IsFull() bool {
	for _, content := range fb.B_content {
		if content.B_inodo == -1 {
			return false
		}
	}
	return true
}

func (fb *FolderBlock) RenameInFolderBlock(oldName string, newName string) error {
	for i := 2; i < len(fb.B_content); i++ {
		content := &fb.B_content[i]
		currentName := strings.Trim(string(content.B_name[:]), "\x00 ")

		if strings.EqualFold(currentName, oldName) && content.B_inodo != -1 {
			if len(newName) > 12 {
				return fmt.Errorf("el nuevo nombre '%s' es demasiado largo, máximo 12 caracteres", newName)
			}
			copy(content.B_name[:], newName)
			for j := len(newName); j < 12; j++ {
				content.B_name[j] = 0
			}

			return nil
		}
	}

	return fmt.Errorf("el nombre '%s' no fue encontrado en los inodos 3 o 4", oldName)
}

func (fb *FolderBlock) RemoveEntry(file *os.File, name string, blockOffset int64) error {
	for i := 2; i < len(fb.B_content); i++ {
		content := &fb.B_content[i]
		currentName := strings.Trim(string(content.B_name[:]), "\x00 ")

		if strings.EqualFold(currentName, name) && content.B_inodo != -1 {
			content.B_inodo = -1
			copy(content.B_name[:], strings.Repeat("\x00", len(content.B_name)))
			err := fb.Encode(file, blockOffset)
			if err != nil {
				return fmt.Errorf("error al serializar el FolderBlock después de eliminar la entrada '%s': %w", name, err)
			}

			return nil
		}

	}

	return fmt.Errorf("la entrada '%s' no fue encontrada en el FolderBlock", name)
}
