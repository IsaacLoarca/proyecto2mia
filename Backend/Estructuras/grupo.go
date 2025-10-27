package estructuras

import "fmt"

type Group struct {
	GID   string
	Tipo  string
	Group string
}

func NewGroup(gid, group string) *Group {
	return &Group{gid, "G", group}
}

func (g *Group) ToString() string {
	return fmt.Sprintf("%s,%s,%s", g.GID, g.Tipo, g.Group)
}

func (g *Group) Eliminar() {
	g.GID = "0"
}
