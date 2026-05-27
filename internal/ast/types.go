package ast

import tok "comp/internal/token"

type TypeKind int

const (
	TypeKindUnknown TypeKind = iota
	TypeKindInt
	TypeKindBool
	TypeKindString
	TypeKindArray
)

type ValueType struct {
	Kind    TypeKind
	Element *ValueType
}

var (
	TypeUnknown = ValueType{Kind: TypeKindUnknown}
	TypeInt     = ValueType{Kind: TypeKindInt}
	TypeBool    = ValueType{Kind: TypeKindBool}
	TypeString  = ValueType{Kind: TypeKindString}
)

func ArrayOf(element ValueType) ValueType {
	elementCopy := element
	return ValueType{
		Kind:    TypeKindArray,
		Element: &elementCopy,
	}
}

func (t ValueType) IsUnknown() bool {
	return t.Kind == TypeKindUnknown
}

func (t ValueType) IsArray() bool {
	return t.Kind == TypeKindArray && t.Element != nil
}

func (t ValueType) Equals(other ValueType) bool {
	if t.Kind != other.Kind {
		return false
	}
	if t.Kind != TypeKindArray {
		return true
	}
	if t.Element == nil || other.Element == nil {
		return t.Element == nil && other.Element == nil
	}
	return t.Element.Equals(*other.Element)
}

func (t ValueType) String() string {
	switch t.Kind {
	case TypeKindInt:
		return "int"
	case TypeKindBool:
		return "bool"
	case TypeKindString:
		return "string"
	case TypeKindArray:
		if t.Element == nil {
			return "unknown[]"
		}
		return t.Element.String() + "[]"
	default:
		return "unknown"
	}
}

type TypeAnnotation struct {
	Name tok.Token
	Kind ValueType
}
