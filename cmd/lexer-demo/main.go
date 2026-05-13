package main

import (
	"comp/internal/driver"
	"comp/internal/lexer"
	"fmt"
	"os"
)

func main() {
	filePath, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	input, err := driver.ReadInput(filePath, "var x: int = 123; print x + 5;")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lex := lexer.NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, token := range tokens {
		fmt.Println(driver.FormatToken(token))
	}
}

func parseArgs(args []string) (string, error) {
	var filePath string
	for _, arg := range args {
		if len(arg) > 0 && arg[0] == '-' {
			return "", fmt.Errorf("unknown option: %s", arg)
		}
		if filePath != "" {
			return "", fmt.Errorf("multiple input files are not supported")
		}
		filePath = arg
	}
	return filePath, nil
}
