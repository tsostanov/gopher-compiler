package lexer

import (
	"fmt"

	tok "comp/internal/token"
)

type lexerState int

const (
	stateDefault lexerState = iota
	stateNumber
	stateWord
	stateString
)

type Lexer struct {
	input    string
	length   int
	position int
	line     int
	column   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		length: len(input),
		line:   1,
		column: 1,
	}
}

func (l *Lexer) Tokenize() ([]tok.Token, error) {
	var result []tok.Token
	err := l.TokenizeEach(func(token tok.Token) bool {
		result = append(result, token)
		return true
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (l *Lexer) TokenizeEach(yield func(tok.Token) bool) error {
	state := stateDefault
	for l.position < l.length {
		switch state {
		case stateDefault:
			current := l.Peek()

			if isWhitespace(current) {
				l.Next()
				continue
			}

			if current == '/' && l.PeekNext() == '/' {
				l.skipLineComment()
				continue
			}

			switch {
			case isDigit(current):
				state = stateNumber
			case isAlpha(current):
				state = stateWord
			case current == '"':
				state = stateString
			default:
				token, err := l.readOperatorOrPunctuation()
				if err != nil {
					return err
				}
				if !yield(token) {
					return nil
				}
			}
		case stateNumber:
			if !yield(l.readNumber()) {
				return nil
			}
			state = stateDefault
		case stateWord:
			if !yield(l.readWord()) {
				return nil
			}
			state = stateDefault
		case stateString:
			token, err := l.readString()
			if err != nil {
				return err
			}
			if !yield(token) {
				return nil
			}
			state = stateDefault
		}
	}

	eofToken := tok.Token{
		Type:     tok.TokenEOF,
		Value:    "",
		Position: l.position,
		Line:     l.line,
		Column:   l.column,
	}
	yield(eofToken)
	return nil
}

var keywords = map[string]tok.TokenType{
	"var":    tok.TokenVar,
	"fun":    tok.TokenFunc,
	"func":   tok.TokenFunc,
	"return": tok.TokenReturn,
	"print":  tok.TokenPrint,
	"if":     tok.TokenIf,
	"else":   tok.TokenElse,
	"while":  tok.TokenWhile,
	"and":    tok.TokenAnd,
	"or":     tok.TokenOr,
	"int":    tok.TokenInt,
	"bool":   tok.TokenBool,
	"string": tok.TokenStringType,
	"true":   tok.TokenTrue,
	"false":  tok.TokenFalse,
}

var operators = map[string]tok.TokenType{
	"==": tok.TokenEqEq,
	"!=": tok.TokenNeq,
	"<=": tok.TokenLtEq,
	">=": tok.TokenGtEq,
	"&&": tok.TokenAnd,
	"||": tok.TokenOr,
	"+":  tok.TokenPlus,
	"-":  tok.TokenMinus,
	"*":  tok.TokenStar,
	"/":  tok.TokenSlash,
	"=":  tok.TokenEq,
	"<":  tok.TokenLt,
	">":  tok.TokenGt,
	"!":  tok.TokenExcl,
	"(":  tok.TokenLParen,
	")":  tok.TokenRParen,
	"{":  tok.TokenLBrace,
	"}":  tok.TokenRBrace,
	":":  tok.TokenColon,
	",":  tok.TokenComma,
	";":  tok.TokenSemicolon,
}

func (l *Lexer) readNumber() tok.Token {
	start := l.position
	startLine := l.line
	startCol := l.column

	for isDigit(l.Peek()) {
		l.Next()
	}

	numberStr := l.input[start:l.position]
	return tok.Token{
		Type:     tok.TokenNumber,
		Value:    numberStr,
		Position: start,
		Line:     startLine,
		Column:   startCol,
	}
}

func (l *Lexer) readWord() tok.Token {
	start := l.position
	startLine := l.line
	startCol := l.column

	for isAlphaNumeric(l.Peek()) {
		l.Next()
	}

	word := l.input[start:l.position]
	tokenType := tok.TokenID
	if kwType, ok := keywords[word]; ok {
		tokenType = kwType
	}

	return tok.Token{
		Type:     tokenType,
		Value:    word,
		Position: start,
		Line:     startLine,
		Column:   startCol,
	}
}

func (l *Lexer) readString() (tok.Token, error) {
	start := l.position
	startLine := l.line
	startCol := l.column
	l.Next()

	for l.position < l.length && l.Peek() != '"' {
		l.Next()
	}

	if l.position >= l.length {
		return tok.Token{}, fmt.Errorf("unterminated string at %d:%d", startLine, startCol)
	}

	l.Next()
	value := l.input[start+1 : l.position-1]
	return tok.Token{
		Type:     tok.TokenString,
		Value:    value,
		Position: start,
		Line:     startLine,
		Column:   startCol,
	}, nil
}

func (l *Lexer) skipLineComment() {
	for l.position < l.length && l.Peek() != '\n' {
		l.Next()
	}
}

func (l *Lexer) readOperatorOrPunctuation() (tok.Token, error) {
	start := l.position
	startLine := l.line
	startCol := l.column

	if l.position+1 < l.length {
		twoChars := l.input[l.position : l.position+2]
		if tokenType, ok := operators[twoChars]; ok {
			l.Next()
			l.Next()
			return tok.Token{
				Type:     tokenType,
				Value:    twoChars,
				Position: start,
				Line:     startLine,
				Column:   startCol,
			}, nil
		}
	}

	oneChar := l.input[l.position : l.position+1]
	if tokenType, ok := operators[oneChar]; ok {
		l.Next()
		return tok.Token{
			Type:     tokenType,
			Value:    oneChar,
			Position: start,
			Line:     startLine,
			Column:   startCol,
		}, nil
	}

	return tok.Token{}, fmt.Errorf("unexpected character '%c' at %d:%d", l.Peek(), startLine, startCol)
}

func (l *Lexer) Peek() byte {
	if l.position >= l.length {
		return 0
	}
	return l.input[l.position]
}

func (l *Lexer) PeekNext() byte {
	if l.position+1 >= l.length {
		return 0
	}
	return l.input[l.position+1]
}

func (l *Lexer) Next() byte {
	if l.position >= l.length {
		return 0
	}
	ch := l.input[l.position]
	l.position++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\r' || ch == '\t' || ch == '\n'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isAlphaNumeric(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}
