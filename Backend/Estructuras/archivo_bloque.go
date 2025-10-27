package estructuras

import (
	"fmt"
	utilidades "godisk/Utilidades"
	"os"
	"strings"
)

const BlockSize = 64

type ArchivoBloque struct {
	B_content [BlockSize]byte
}

func (fb *ArchivoBloque) Encode(file *os.File, offset int64) error {
	err := utilidades.EscribirEnArchivo(file, offset, fb.B_content)
	if err != nil {
		return fmt.Errorf("error writing ArchivoBloque to file: %w", err)
	}
	return nil
}

func (fb *ArchivoBloque) Decode(file *os.File, offset int64) error {
	err := utilidades.LeerDesdeArchivo(file, offset, &fb.B_content)
	if err != nil {
		return fmt.Errorf("error reading FileBlock from file: %w", err)
	}
	return nil
}

func (fb *ArchivoBloque) EspacioUsado() int {
	content := fb.GetContent()
	return len(content)
}

func (fb *ArchivoBloque) GetContent() string {
	content := string(fb.B_content[:])
	content = strings.TrimRight(content, "\x00")
	return content
}

func (fb *ArchivoBloque) SetContent(content string) error {
	if len(content) > BlockSize {
		return fmt.Errorf("el tamaño del contenido excede el tamaño del bloque de %d bytes", BlockSize)
	}
	fb.ClearContent()
	copy(fb.B_content[:], content)
	return nil
}

func (fb *ArchivoBloque) EspacioDisponible() int {
	return BlockSize - fb.EspacioUsado()
}

func (fb *ArchivoBloque) TieneEspacio() bool {
	return fb.EspacioDisponible() > 0
}

func (fb *ArchivoBloque) Print() {
	fmt.Print(fb.GetContent())
}

func (fb *ArchivoBloque) AppendContent(content string) error {
	espacioDisponible := fb.EspacioDisponible()

	if len(content) > espacioDisponible {
		return fmt.Errorf("no hay suficiente espacio para agregar el contenido, se requieren %d bytes pero solo hay %d bytes disponibles", len(content), espacioDisponible)
	}

	espacioUsado := fb.EspacioUsado()

	copy(fb.B_content[espacioUsado:], content)

	return nil
}

func (fb *ArchivoBloque) ClearContent() {
	for i := range fb.B_content {
		fb.B_content[i] = 0
	}
}

func NewArchivoBloque(content string) (*ArchivoBloque, error) {
	fb := &ArchivoBloque{}
	err := fb.SetContent(content)
	if err != nil {
		return nil, err
	}
	return fb, nil
}

func SplitContent(content string) ([]*ArchivoBloque, error) {
	var blocks []*ArchivoBloque
	for len(content) > 0 {
		end := BlockSize
		if len(content) < BlockSize {
			end = len(content)
		}
		fb, err := NewArchivoBloque(content[:end])
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, fb)
		content = content[end:]
	}
	return blocks, nil
}
