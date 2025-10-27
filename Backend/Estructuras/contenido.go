package estructuras

type Content struct {
	b_name  [12]byte
	b_inodo int64
}

func NewContent() Content {
	var cont Content
	cont.b_inodo = -1
	return cont
}
