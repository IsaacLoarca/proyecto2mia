package instrucciones

import (
	"bytes"
	"fmt"
	estructuras "godisk/Estructuras"
	globals "godisk/Global"
)

type LOGOUT struct{}

func AnalizarLogout(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer

	if len(tokens) > 1 {
		return "", fmt.Errorf("el comando Logout no acepta parámetros")
	}

	err := commandLogout(&outputBuffer)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	return outputBuffer.String(), nil
}

func commandLogout(outputBuffer *bytes.Buffer) error {
	if globals.UsuarioActual == nil || !globals.UsuarioActual.Status {
		return fmt.Errorf("no hay ninguna sesión activa")
	}

	fmt.Fprintf(outputBuffer, "Cerrando sesión de usuario: %s\n", globals.UsuarioActual.Name)

	fmt.Printf("Cerrando sesión de usuario: %s\n", globals.UsuarioActual.Name)

	globals.UsuarioActual = &estructuras.Usuario{}

	fmt.Fprintln(outputBuffer, "Sesión cerrada correctamente.")
	fmt.Println("Sesión cerrada correctamente.")

	return nil
}
