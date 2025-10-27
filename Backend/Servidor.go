package main

import (
	"fmt"
	analizador "godisk/Analizador"
	globals "godisk/Global"
	instrucciones_gen "godisk/Instrucciones"
	instrucciones "godisk/Instrucciones/Usuarios"
	"log"
	"net/http"
	"os"
	"strings"

	ginCors "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func analizar(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se ha proveído ningún comando"})
		return
	}

	command := string(body)

	lines := strings.Split(command, "\n")

	var results []string
	var errors []string

	for i, line := range lines {
		lineNumber := i + 1
		result, err := analizador.Analizador(line)

		if err != nil {
			if err.Error() != "" {
				errors = append(errors, fmt.Sprintf("Línea %d: %s", lineNumber, err.Error()))
			}
		} else if result != "" {
			results = append(results, fmt.Sprintf("Línea %d: %s", lineNumber, result))
		}
	}

	response := gin.H{
		"Lineas en total":     len(lines),
		"Lineas procesadas":   len(results),
		"Errores encontrados": len(errors),
	}

	if len(results) > 0 {
		response["Resultados"] = results
	}

	if len(errors) > 0 {
		response["Errores"] = errors
	}

	statusCode := http.StatusOK
	if len(errors) > 0 {
		statusCode = http.StatusMultiStatus
	}

	c.JSON(statusCode, response)
}

// LoginRequest estructura para la petición de login
type LoginRequest struct {
	User string `json:"user" binding:"required"`
	Pass string `json:"pass" binding:"required"`
	ID   string `json:"id" binding:"required"`
}

// LoginResponse estructura para la respuesta de login
type LoginResponse struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	User    *UserData `json:"user,omitempty"`
}

// UserData estructura para datos del usuario logueado
type UserData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Group       string `json:"group"`
	Status      bool   `json:"status"`
	PartitionID string `json:"partition_id"`
}

// Handler para login
func loginHandler(c *gin.Context) {
	var loginReq LoginRequest
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{
			Status:  "error",
			Message: "Datos de login inválidos: " + err.Error(),
		})
		return
	}

	// Verificar si ya hay un usuario logueado
	if globals.UsuarioActual != nil && globals.UsuarioActual.Status {
		c.JSON(http.StatusConflict, LoginResponse{
			Status:  "error",
			Message: "Ya hay un usuario logueado. Debe cerrar sesión primero.",
		})
		return
	}

	// Crear el comando LOGIN usando las tokens como el parser original
	tokens := []string{
		fmt.Sprintf("-user=%s", loginReq.User),
		fmt.Sprintf("-pass=%s", loginReq.Pass),
		fmt.Sprintf("-id=%s", loginReq.ID),
	}

	// Usar el parser existente
	result, err := instrucciones.ParserLogin(tokens)
	if err != nil {
		errorMessage := err.Error()
		if result != nil && result["message"] != nil {
			errorMessage = result["message"].(string)
		}
		c.JSON(http.StatusUnauthorized, LoginResponse{
			Status:  "error",
			Message: errorMessage,
		})
		return
	}

	// Si el login fue exitoso, devolver los datos del usuario
	if globals.UsuarioActual != nil && globals.UsuarioActual.Status {
		c.JSON(http.StatusOK, LoginResponse{
			Status:  "success",
			Message: "Login exitoso",
			User: &UserData{
				ID:          globals.UsuarioActual.Id,
				Name:        globals.UsuarioActual.Name,
				Group:       globals.UsuarioActual.Group,
				Status:      globals.UsuarioActual.Status,
				PartitionID: loginReq.ID,
			},
		})
	} else {
		c.JSON(http.StatusInternalServerError, LoginResponse{
			Status:  "error",
			Message: "Error interno del servidor durante el login",
		})
	}
}

// Handler para logout
func logoutHandler(c *gin.Context) {
	if globals.UsuarioActual == nil || !globals.UsuarioActual.Status {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "No hay un usuario logueado",
		})
		return
	}

	// Cerrar sesión
	globals.CerrarSesion()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Sesión cerrada exitosamente",
	})
}

// Handler para verificar la sesión actual
func sessionHandler(c *gin.Context) {
	if globals.EstaLogueado() {
		c.JSON(http.StatusOK, gin.H{
			"logged_in": true,
			"user":      globals.UsuarioActual.Name,
			"id":        globals.UsuarioActual.Id,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"logged_in": false,
		})
	}
}

func directoryTreeHandler(c *gin.Context) {
	if !globals.EstaLogueado() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Usuario no logueado",
		})
		return
	}

	// Crear el servicio de árbol de directorios
	dirService, err := instrucciones_gen.NewDirectoryTreeService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error al acceder al sistema de archivos: " + err.Error(),
		})
		return
	}
	defer dirService.Close()

	// Obtener el árbol de directorios desde la raíz
	tree, err := dirService.GetDirectoryTree("/")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error al obtener el árbol de directorios: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tree":    tree,
	})
}

func main() {
	// Configuración fija: usar localhost:8080 (no depender de .env)

	// Configurar Gin para producción si está en producción
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Usar middleware de CORS oficial de Gin (configuración directa similar al ejemplo de Fiber)
	// CORS: cuando AllowCredentials = true no se puede usar "*" como AllowOrigins
	// Usamos AllowOriginFunc para reflejar el Origin (aceptar cualquier origen) pero
	// podrías restringirlo a dominios concretos por seguridad.
	router.Use(ginCors.New(ginCors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		// Permitir dinámicamente cualquier origen (en producción restringir a orígenes confiables)
		AllowOriginFunc: func(origin string) bool {
			return true
		},
	}))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "GODISK API",
			"version": "1.0.0",
			"env":     os.Getenv("GIN_MODE"),
		})
	})

	// Authentication endpoints
	router.POST("/login", loginHandler)
	router.POST("/logout", logoutHandler)
	router.GET("/session", sessionHandler)

	// File system endpoints
	router.GET("/directory-tree", directoryTreeHandler)

	router.POST("/analizar", analizar)

	// Lanzamiento directo de la API: escuchar en todas las interfaces en el puerto 8080
	bindAddr := "0.0.0.0:8080"
	log.Printf("Servidor iniciando en %s", bindAddr)
	router.Run(bindAddr)
}
