package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Confirm(message string) (bool, error) {
	fmt.Printf("%s [y/N]: ", message)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read input: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}
