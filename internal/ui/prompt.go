package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

// Prompt asks the user for text input with a default value.
// Returns the default if the user presses Enter without typing.
func Prompt(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

// Confirm asks a yes/no question. Returns the default if user presses Enter.
func Confirm(label string, defaultYes bool) bool {
	suffix := " [Y/n]: "
	if !defaultYes {
		suffix = " [y/N]: "
	}
	fmt.Print(label + suffix)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}

// Select shows a numbered list and returns the selected index.
// Returns -1 if the user cancels.
func Select(label string, options []string) int {
	fmt.Println(label)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Printf("Choose [1-%d]: ", len(options))
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(options) {
		return -1
	}
	return n - 1
}

// WaitForEnter pauses until the user presses Enter.
func WaitForEnter(label string) {
	fmt.Print(label)
	reader.ReadString('\n')
}
