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

func cleanCString(buf []byte) string {
	return strings.TrimSpace(string(bytes.TrimRight(buf, "\x00")))
}
func splitPath(p string) ([]string, string) {
	p = strings.Trim(p, "/")
	if p == "" {
		return []string{}, ""
	}
	parts := strings.Split(p, "/")
	return parts[:len(parts)-1], parts[len(parts)-1]
}

func ensureRoot(f *os.File, sb *Superbloque) error {
	in0 := &Inodo{}
	if err := in0.Decode(f, int64(sb.S_inode_start)); err == nil &&
		in0.I_type[0] == '0' {
		return nil
	}

	if err := sb.UpdateBitmapInode(f, 0, true); err != nil {
		return err
	}
	if err := sb.UpdateBitmapBlock(f, 0, true); err != nil {
		return err
	}

	root := NewEmptyInode()
	root.I_type[0] = '0'
	root.I_perm = [3]byte{'7', '7', '7'}
	root.I_block[0] = 0
	if err := root.Encode(f, int64(sb.S_inode_start)); err != nil {
		return err
	}

	b0 := NewFolderBlock(0, 0, map[string]int32{})
	if err := b0.Encode(f, int64(sb.S_block_start)); err != nil {
		return err
	}

	sb.S_inodes_count = 1
	sb.S_blocks_count = 1
	sb.S_free_inodes_count--
	sb.S_free_blocks_count--
	sb.S_first_ino += sb.S_inode_size
	sb.S_first_blo += sb.S_block_size
	return nil
}

func wipeStructures(f *os.File, sb *Superbloque) error {
	return CleanLossAreas(f, sb)
}

func replayJournal(f *os.File, sb *Superbloque, partStart int32) error {
	jStart := int64(partStart) + int64(binary.Size(Superbloque{}))

	entries, err := FindValidJournalEntries(f, jStart, JOURNAL_ENTRIES)
	if err != nil {
		return err
	}

	for _, e := range entries {
		op := cleanCString(e.J_content.I_operation[:])
		path := cleanCString(e.J_content.I_path[:])
		data := cleanCString(e.J_content.I_content[:])

		if path == "" {
			continue
		}

		if op == "mkdir" && (path == "/" || path == "") {
			continue
		}

		parentDirs, name := splitPath(path)

		switch op {
		case "mkdir":
			if err := sb.CrearCarpeta(f, parentDirs, name, false); err != nil {
				return fmt.Errorf("replay mkdir %s: %w", path, err)
			}

		case "mkfile":
			chunks := utilidades.DividirCadenaEnTrozos(data)
			if err := sb.CrearArchivo(f, parentDirs, name,
				len(data), chunks, false); err != nil {
				return fmt.Errorf("replay mkfile %s: %w", path, err)
			}

		default:
		}
	}
	return nil
}

func RecoverFileSystem(f *os.File, sb *Superbloque, partStart int32) error {
	if err := wipeStructures(f, sb); err != nil {
		return err
	}

	if err := ensureRoot(f, sb); err != nil {
		return err
	}

	if err := replayJournal(f, sb, partStart); err != nil {
		return err
	}

	sb.S_mtime = float64(time.Now().Unix())
	if err := sb.Codificar(f, int64(partStart)); err != nil {
		return err
	}

	return f.Sync()
}
