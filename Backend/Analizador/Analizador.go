package analizador

import (
	"errors"
	"fmt"
	comandos "godisk/Instrucciones"
	instrucciones "godisk/Instrucciones/Discos"
	usuarios "godisk/Instrucciones/Usuarios"
	reportes "godisk/Reportes"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var mapaComandos = map[string]func([]string) (string, error){
	"mkdisk": func(args []string) (string, error) {
		resultado, err := instrucciones.AnalizarMkdisk(args)
		return fmt.Sprintf("%v", resultado), err
	},
	"fdisk": func(args []string) (string, error) {
		resultado, err := instrucciones.AnalizarFdisk(args)
		return fmt.Sprintf("%v", resultado), err
	},
	"mount": func(args []string) (string, error) {
		resultado, err := instrucciones.AnalizarMount(args)
		return fmt.Sprintf("%v", resultado), err
	},
	"mounted": func(args []string) (string, error) {
		resultado, err := instrucciones.Mounted(args)
		return fmt.Sprintf("%v", resultado), err
	},
	"unmount": func(args []string) (string, error) {
		result, err := instrucciones.AnalizarUnmount(args)
		return fmt.Sprintf("%v", result), err
	},
	"mkfs": func(args []string) (string, error) {
		result, err := instrucciones.AnalizarMkfs(args)
		return fmt.Sprintf("%v", result), err
	},
	"rep": func(args []string) (string, error) {
		result, err := reportes.AnalizarRep(args)
		return fmt.Sprintf("%v", result), err
	},
	"login": func(args []string) (string, error) {
		result, err := usuarios.AnalizarLogin(args)
		return fmt.Sprintf("%v", result), err
	},
	"logout": func(args []string) (string, error) {
		result, err := usuarios.AnalizarLogout(args)
		return fmt.Sprintf("%v", result), err
	},
	"mkgrp": func(args []string) (string, error) {
		result, err := usuarios.AnalizarMkgrp(args)
		return fmt.Sprintf("%v", result), err
	},
	"rmgrp": func(args []string) (string, error) {
		result, err := usuarios.AnalizarRmgrp(args)
		return fmt.Sprintf("%v", result), err
	},
	"mkusr": func(args []string) (string, error) {
		result, err := usuarios.AnalizarMkusr(args)
		return fmt.Sprintf("%v", result), err
	},
	"rmusr": func(args []string) (string, error) {
		result, err := usuarios.AnalizarRmusr(args)
		return fmt.Sprintf("%v", result), err
	},
	"chgrp": func(args []string) (string, error) {
		result, err := usuarios.AnalizarChgrp(args)
		return fmt.Sprintf("%v", result), err
	},
	"mkfile": func(args []string) (string, error) {
		result, err := comandos.AnalizarMkfile(args)
		return fmt.Sprintf("%v", result), err
	},
	"mkdir": func(args []string) (string, error) {
		result, err := comandos.AnalizarMkdir(args)
		return fmt.Sprintf("%v", result), err
	},
	"cat": func(args []string) (string, error) {
		result, err := comandos.AnalizarCat(args)
		return fmt.Sprintf("%v", result), err
	},
	"rmdisk": func(args []string) (string, error) {
		result, err := instrucciones.AnalizarRmdisk(args)
		return fmt.Sprintf("%v", result), err
	},
	"rename": func(args []string) (string, error) {
		result, err := comandos.AnalizarRename(args)
		return fmt.Sprintf("%v", result), err
	},
	"edit": func(args []string) (string, error) {
		result, err := comandos.AnalizarEdit(args)
		return fmt.Sprintf("%v", result), err
	},
	"find": func(args []string) (string, error) {
		result, err := comandos.AnalizarFind(args)
		return fmt.Sprintf("%v", result), err
	},
	"remove": func(args []string) (string, error) {
		result, err := comandos.AnalizarRemove(args)
		return fmt.Sprintf("%v", result), err
	},
	"lsblk": func(args []string) (string, error) {
		result, err := instrucciones.AnalizarListPartitions(args)
		return fmt.Sprintf("%v", result), err
	},
	"journaling": func(args []string) (string, error) {
		result, err := comandos.AnalizarJournaling(args)
		return fmt.Sprintf("%v", result), err
	},
	"loss": func(args []string) (string, error) {
		result, err := comandos.AnalizarLoss(args)
		return result, err
	},
	"recovery": func(args []string) (string, error) {
		result, err := comandos.AnalizarRecovery(args)
		return result, err
	},
}

func Analizador(entrada string) (string, error) {
	entrada = strings.TrimSpace(entrada)

	if entrada == "" {
		return "", nil
	}

	if strings.HasPrefix(entrada, "#") {
		return "", nil
	}

	tokens := strings.Fields(entrada)
	if len(tokens) == 0 {
		return "", errors.New("no se proporcionó ningún comando")
	}

	funcionComando, existe := mapaComandos[tokens[0]]
	if !existe {
		switch tokens[0] {
		case "clear":
			return limpiarTerminal()
		case "exit":
			os.Exit(0)
		default:
			return "", fmt.Errorf("comando desconocido: %s", tokens[0])
		}
	}

	return funcionComando(tokens[1:])
}

func limpiarTerminal() (string, error) {
	var comando *exec.Cmd
	if runtime.GOOS == "windows" {
		comando = exec.Command("cmd", "/c", "cls")
	} else {
		comando = exec.Command("clear")
	}
	comando.Stdout = os.Stdout
	if err := comando.Run(); err != nil {
		return "", errors.New("no se pudo limpiar la terminal")
	}
	return "Terminal limpiada", nil
}
