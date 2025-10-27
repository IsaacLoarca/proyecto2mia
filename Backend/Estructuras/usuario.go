package estructuras

import "fmt"

type Usuario struct {
	Id       string
	Tipo     string
	Group    string
	Name     string
	Password string
	Status   bool
}

func NewUser(id, group, name, password string) *Usuario {
	return &Usuario{id, "U", group, name, password, true}
}

func (u *Usuario) ToString() string {
	return fmt.Sprintf("%s,%s,%s,%s,%s", u.Id, u.Tipo, u.Group, u.Name, u.Password)
}

func (u *Usuario) Eliminar() {
	u.Id = "0"
	u.Status = false
}
