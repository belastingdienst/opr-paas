package fields

// ElementArray is an interface which represents all values that could be truned into a string map (a.o.)
type ElementArray interface {
	AsElementMap() (ElementMap, error)
}
