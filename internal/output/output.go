package output

import (
	"encoding/json"
	"fmt"
)

type Renderer interface {
	RenderTable() (string, error)
}

func Print(format string, r Renderer) error {
	switch format {
	case "json":
		b, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	default:
		str, err := r.RenderTable()
		if err != nil {
			return err
		}
		fmt.Println(str)
	}
	return nil
}
