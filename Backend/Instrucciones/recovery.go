package instrucciones

import (
	"errors"
	estructuras "godisk/Estructuras"
	global "godisk/Global"
	"os"
	"strings"
)

type RecoveryCmd struct{ Id string }

func AnalizarRecovery(args []string) (string, error) {
	cmd := &RecoveryCmd{}
	for _, a := range args {
		if strings.HasPrefix(strings.ToLower(a), "-id=") {
			cmd.Id = strings.SplitN(a, "=", 2)[1]
		}
	}
	if cmd.Id == "" {
		return "", errors.New("falta parámetro -id")
	}
	return cmd.Execute()
}

func (c *RecoveryCmd) Execute() (string, error) {
	sb, part, path, err := global.GetMountedPartitionSuperblock(c.Id)
	if err != nil {
		return "", err
	}
	if sb.S_filesystem_type != 3 {
		return "", errors.New("la partición no es EXT3 (sisn journaling)")
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := estructuras.RecoverFileSystem(f, sb, part.Part_start); err != nil {
		return "", err
	}

	return "Recuperación exitosa: el sistema se restauró usando el journal", nil
}
