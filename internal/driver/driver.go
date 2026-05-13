package driver

import (
	tok "comp/internal/token"
	"fmt"
	"os"
)

func ReadInput(filePath, fallback string) (string, error) {
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	return fallback, nil
}

func FormatToken(token tok.Token) string {
	return fmt.Sprintf("[%d:%d] Token(%s, %q)", token.Line, token.Column, token.Type.String(), token.Value)
}
