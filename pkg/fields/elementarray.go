package fields

// ElementArray is an interface which represents all values that could be turned into an ElementMap
type ElementArray interface {
	AsElementMap() ElementMap
}
