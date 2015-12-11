package yaml

import (
	"reflect"
)

type Marshaller struct {
	MaxWidth      int
	IndentSpaces  int
	CommentStart  int
}

func (m *Marshaller) AlignComments() {
	m.CommentStart = -1
}

func (m *Marshaller) Marshal(in interface{}) (out []byte, err error) {
	defer handleErr(&err)
	e := newEncoder(m.IndentSpaces, m.MaxWidth, m.CommentStart)
	e.marshal("", reflect.ValueOf(in))
	e.finish()
	out = e.out
	return
}
