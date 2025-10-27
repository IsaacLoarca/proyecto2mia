package estructuras

import (
	"fmt"
	"os"
)

func ZeroRegion(f *os.File, offset, length int64) error {
	const chunk = 4096
	buf := make([]byte, chunk)
	var written int64
	for written < length {
		n := length - written
		if n > chunk {
			n = chunk
		}
		if _, err := f.WriteAt(buf[:n], offset+written); err != nil {
			return fmt.Errorf("escribir ceros en off=%d: %w", offset+written, err)
		}
		written += n
	}
	return nil
}

func CleanLossAreas(f *os.File, sb *Superbloque) error {
	totalInodes := int64(sb.S_inodes_count + sb.S_free_inodes_count) // n
	totalBlocks := int64(sb.S_blocks_count + sb.S_free_blocks_count) // 3n
	inodeSize := int64(sb.S_inode_size)
	blockSize := int64(sb.S_block_size)

	bmpInodeLen := (totalInodes + 7) / 8
	if err := ZeroRegion(f, int64(sb.S_bm_inode_start), bmpInodeLen); err != nil {
		return err
	}
	bmpBlockLen := (totalBlocks + 7) / 8
	if err := ZeroRegion(f, int64(sb.S_bm_block_start), bmpBlockLen); err != nil {
		return err
	}
	if err := ZeroRegion(f, int64(sb.S_inode_start), totalInodes*inodeSize); err != nil {
		return err
	}
	if err := ZeroRegion(f, int64(sb.S_block_start), totalBlocks*blockSize); err != nil {
		return err
	}
	return f.Sync()
}
