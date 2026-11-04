package pangolin

import (
	"bytes"
	"fmt"
	"strconv"
)

// O Cloud serializa COUNT(*) como texto em algumas rotas e como número noutras.
type jsonCount int

func (c *jsonCount) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 1 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	value, err := strconv.Atoi(string(raw))
	if err != nil || value < 0 {
		return fmt.Errorf("total de paginação inválido")
	}
	*c = jsonCount(value)
	return nil
}
