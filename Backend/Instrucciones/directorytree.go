package instrucciones

import (
	"fmt"
	"os"
	"strings"

	estructuras "godisk/Estructuras"
	globals "godisk/Global"
	utilidades "godisk/Utilidades"
)

type DirectoryTree struct {
	Name     string           `json:"name"`
	Children []*DirectoryTree `json:"children,omitempty"`
	IsDir    bool             `json:"isDir"`
}

type DirectoryTreeService struct {
	partitionSuperblock *estructuras.Superbloque
	partitionPath       string
	file                *os.File
}

func NewDirectoryTreeService() (*DirectoryTreeService, error) {
	if !globals.EstaLogueado() {
		return nil, fmt.Errorf("operación denegada: no se ha iniciado sesión")
	}
	if err := globals.ValidarAcceso(globals.UsuarioActual.Id); err != nil {
		return nil, fmt.Errorf("permisos insuficientes para acceder a la partición: %w", err)
	}
	idPartition := globals.UsuarioActual.Id
	partitionSuperblock, _, partitionPath, err := globals.GetMountedPartitionSuperblock(idPartition)
	if err != nil {
		return nil, fmt.Errorf("imposible obtener la partición montada (ID: %s): %w", idPartition, err)
	}
	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("fallo al abrir el archivo de la partición en '%s': %w", partitionPath, err)
	}
	return &DirectoryTreeService{
		partitionSuperblock: partitionSuperblock,
		partitionPath:       partitionPath,
		file:                file,
	}, nil
}

func (dts *DirectoryTreeService) Close() {
	dts.file.Close()
}

func (dts *DirectoryTreeService) GetDirectoryTree(path string) (*DirectoryTree, error) {
	var rootInodeIndex int32
	var err error

	if path == "/" {
		rootInodeIndex = 0
	} else {
		parentDirs, dirName := utilidades.ObtenerDirectoriosPadre(path)
		rootInodeIndex, err = findFileInode(dts.file, dts.partitionSuperblock, parentDirs, dirName)
		if err != nil {
			return nil, fmt.Errorf("imposible localizar el directorio inicial '%s': %w", path, err)
		}
	}
	tree, err := dts.buildDirectoryTree(rootInodeIndex, path)
	if err != nil {
		return nil, fmt.Errorf("fallo al construir el árbol de directorios para '%s': %w", path, err)
	}

	return tree, nil
}

func (dts *DirectoryTreeService) buildDirectoryTree(inodeIndex int32, currentPath string) (*DirectoryTree, error) {
	inode := &estructuras.Inodo{}
	offset := int64(dts.partitionSuperblock.S_inode_start) + int64(inodeIndex*dts.partitionSuperblock.S_inode_size)
	err := inode.Decode(dts.file, offset)
	if err != nil {
		return nil, fmt.Errorf("fallo al deserializar el inodo %d (offset %d) para '%s': %w", inodeIndex, offset, currentPath, err)
	}
	var currentName string
	if currentPath == "/" {
		currentName = "/"
	} else {
		pathSegments := strings.Split(strings.Trim(currentPath, "/"), "/")
		currentName = pathSegments[len(pathSegments)-1]
	}

	tree := &DirectoryTree{
		Name:     currentName,
		IsDir:    inode.I_type[0] == '0',
		Children: []*DirectoryTree{}, // Inicializar siempre como slice vacío
	}

	if !tree.IsDir {
		return tree, nil
	}

	for _, blockIndex := range inode.I_block {
		if blockIndex == -1 {
			break
		}

		block := &estructuras.FolderBlock{}
		blockOffset := int64(dts.partitionSuperblock.S_block_start) + int64(blockIndex*dts.partitionSuperblock.S_block_size)
		err := block.Decode(dts.file, blockOffset)
		if err != nil {
			return nil, fmt.Errorf("fallo al deserializar el bloque %d (offset %d): %w", blockIndex, blockOffset, err)
		}
		for _, content := range block.B_content {
			if content.B_inodo == -1 {
				continue
			}

			contentName := strings.Trim(string(content.B_name[:]), "\x00 ")
			if contentName == "." || contentName == ".." {
				continue
			}
			var childPath string
			if currentPath == "/" {
				childPath = "/" + contentName
			} else {
				childPath = currentPath + "/" + contentName
			}

			childNode, err := dts.buildDirectoryTree(content.B_inodo, childPath)
			if err != nil {
				// Log el error pero continúa con otros hijos
				fmt.Printf("Error building child '%s': %v\n", childPath, err)
				continue
			}
			tree.Children = append(tree.Children, childNode)
		}
	}

	return tree, nil
}

func (dts *DirectoryTreeService) GenerateDotGraph() (string, error) {
	tree, err := dts.GetDirectoryTree("/")
	if err != nil {
		return "", fmt.Errorf("imposible obtener el árbol de directorios: %w", err)
	}

	const header = `digraph DirectoryTree {
    rankdir=TB;
    bgcolor="#f8f9fa";
    node [
        fontname="JetBrains Mono, Monaco, 'Courier New'",
        fontsize=11,
        style="rounded,filled",
        margin=0.15
    ];
    edge [
        arrowhead=vee,
        color="#6c757d",
        penwidth=1.5,
        arrowsize=0.8
    ];
    graph [
        dpi=300,
        pad=0.5,
        nodesep=0.8,
        ranksep=1.2
    ];
`
	const footer = `
}
`
	var lines []string
	lines = append(lines, header)

	nodeCounter := 0
	nodeIDs := make(map[*DirectoryTree]string)

	var buildDot func(node *DirectoryTree, parentID string, depth int)
	buildDot = func(node *DirectoryTree, parentID string, depth int) {
		id := fmt.Sprintf("node%d", nodeCounter)
		nodeCounter++
		nodeIDs[node] = id

		var fill, font, border, shape string
		if node.IsDir {
			fill = "#667eea"
			font = "#ffffff"
			border = "#4f46e5"
			shape = "folder"
		} else {
			fill = "#10b981"
			font = "#ffffff"
			border = "#059669"
			shape = "note"
		}

		label := node.Name
		if node.IsDir && label == "/" {
			label = "ROOT"
			fill = "#f59e0b"
			border = "#d97706"
		}

		lines = append(lines, fmt.Sprintf(
			"    %s [label=%q fillcolor=%q fontcolor=%q color=%q shape=%s penwidth=2];",
			id, label, fill, font, border, shape,
		))

		if parentID != "" {
			lines = append(lines, fmt.Sprintf("    %s -> %s;", parentID, id))
		}

		for _, c := range node.Children {
			buildDot(c, id, depth+1)
		}
	}
	buildDot(tree, "", 0)
	lines = append(lines, footer)

	return strings.Join(lines, "\n"), nil
}
