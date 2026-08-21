package parser

type FieldInfo struct {
	Name     string
	Type     string
	Tag      string
	Embedded bool
	Parent   string
}
