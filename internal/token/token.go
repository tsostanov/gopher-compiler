package token

import "fmt"

type TokenType int

const (
	TokenNumber TokenType = iota
	TokenID
	TokenString
	TokenTrue
	TokenFalse
	TokenVar
	TokenFunc
	TokenReturn
	TokenPrint
	TokenIf
	TokenElse
	TokenWhile
	TokenInt
	TokenBool
	TokenStringType
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenEq
	TokenEqEq
	TokenExcl
	TokenNeq
	TokenLt
	TokenGt
	TokenLtEq
	TokenGtEq
	TokenAnd
	TokenOr
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenLBrace
	TokenRBrace
	TokenColon
	TokenComma
	TokenSemicolon
	TokenEOF
)

var tokenTypeNames = []string{
	"NUMBER",
	"ID",
	"STRING",
	"TRUE",
	"FALSE",
	"VAR",
	"FUNC",
	"RETURN",
	"PRINT",
	"IF",
	"ELSE",
	"WHILE",
	"INT",
	"BOOL",
	"STRING_TYPE",
	"PLUS",
	"MINUS",
	"STAR",
	"SLASH",
	"EQ",
	"EQEQ",
	"EXCL",
	"NEQ",
	"LT",
	"GT",
	"LTEQ",
	"GTEQ",
	"AND",
	"OR",
	"LPAREN",
	"RPAREN",
	"LBRACKET",
	"RBRACKET",
	"LBRACE",
	"RBRACE",
	"COLON",
	"COMMA",
	"SEMICOLON",
	"EOF",
}

func (t TokenType) String() string {
	if int(t) < len(tokenTypeNames) {
		return tokenTypeNames[t]
	}
	return "UNKNOWN"
}

type Token struct {
	Type     TokenType
	Value    string
	Position int
	Line     int
	Column   int
}

func (t Token) String() string {
	return fmt.Sprintf("Token(Type: %s, Value: %q) at %d:%d", t.Type.String(), t.Value, t.Line, t.Column)
}
