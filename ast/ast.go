package ast

//go:generate sumgen Node = *File | *Header | *Directive | *List | ListItem | *Paragraph | Text
type Node interface {
	node()
}

//go:generate sumgen Stmt = *Header | *Directive | *List | *Paragraph | *Citation
type Stmt interface {
	Node
	stmt()
}

type File struct {
	List   []Stmt
	Errors []error
	Cite   map[string]string
}

type Header struct {
	NThorpe int
	Text    Text
}

type Directive struct {
	Command string
	Raw     string
}

type List struct {
	Items []ListItem
}

type ListItem struct {
	NTab  int
	Label string
	Text  Text
}

type Paragraph struct {
	Format []Format
	Body   string
}

type Format struct {
	Kind DKind
	Beg  int
	End  int
}

type Text Paragraph

type Citation struct {
	Label string
	Src   string
}

type DKind int

const (
	Cite DKind = iota
	Italic
	Bold
	BoldItalic
	Underline
	Strikethrough
	Raw
)

func Walk(n Node, f Walker) (Node, error) {
	if n != nil {
		nn, e := f(n)
		if e != nil {
			return n, e
		}
		n = nn
		switch t := n.(type) {
		case *File:
			for i := range t.List {
				s, e := f(t.List[i])
				if e != nil {
					return n, e
				}
				if s == nil {
					t.List = append(t.List[:i], t.List[i+1:]...)
				} else {
					t.List[i] = s.(Stmt)
				}
			}
		case *Header:
			s, e := f(t.Text)
			if e != nil {
				return n, e
			}
			t.Text = s.(Text)
		case *List:
			for i := range t.Items {
				s, e := f(t.Items[i])
				if e != nil {
					return n, e
				}
				if s == nil {
					t.Items = append(t.Items[:i], t.Items[i+1:]...)
				} else {
					t.Items[i] = s.(ListItem)
				}
			}
		case ListItem:
			s, e := f(t.Text)
			if e != nil {
				return n, e
			}
			t.Text = s.(Text)
		}
	}
	return n, nil
}

type Walker func(Node) (Node, error)
