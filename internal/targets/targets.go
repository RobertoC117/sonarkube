package targets

import (
	"bufio"
	"io"
)

func ParseTargets(r io.Reader) ([]string, error) {

	scanner := bufio.NewScanner(r)
	parsedTargets := make([]string, 0)

	for scanner.Scan() {
		target := scanner.Text()
		parsedTargets = append(parsedTargets, target)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return parsedTargets, nil
}
