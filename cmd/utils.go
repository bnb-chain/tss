// Most utilitie functions are borrowed from cosmos/cosmos-sdk/client/input.go

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
)

func GetInt(prompt string, buf *bufio.Reader) (int, error) {
	s, err := GetString(prompt, buf)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(s)
}

// GetString simply returns the trimmed string output of a given reader.
func GetString(prompt string, buf *bufio.Reader) (string, error) {
	if inputIsTty() && prompt != "" {
		PrintPrefixed(prompt)
	}

	out, err := readLineFromBuf(buf)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func GetBool(prompt string, defaultValue bool, buf *bufio.Reader) (bool, error) {
	answer, err := GetString(prompt, buf)
	if err != nil {
		return false, err
	}
	if answer == "y" || answer == "Y" || answer == "Yes" || answer == "YES" {
		return true, nil
	} else if answer == "n" || answer == "N" || answer == "No" || answer == "NO" {
		return false, nil
	} else if strings.TrimSpace(answer) == "" {
		return defaultValue, nil
	} else {
		return false, fmt.Errorf("input does not make sense, please input 'y' or 'n'")
	}
}

// inputIsTty returns true iff we have an interactive prompt,
// where we can disable echo and request to repeat the password.
// If false, we can optimize for piped input from another command
func inputIsTty() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// readLineFromBuf reads one line from stdin.
// Subsequent calls reuse the same buffer, so we don't lose
// any input when reading a password twice (to verify)
func readLineFromBuf(buf *bufio.Reader) (string, error) {
	pass, err := buf.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(pass), nil
}

// PrintPrefixed prints a string with > prefixed for use in prompts.
func PrintPrefixed(msg string) {
	msg = fmt.Sprintf("> %s\n", msg)
	fmt.Fprint(os.Stderr, msg)
}
