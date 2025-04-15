package secrets

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func PromptForSecureInput(prompt string) (string, error) {
	fmt.Print(prompt)
	// Disable input echoing
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println() // Print a newline after input
	return string(bytePassword), nil
}
