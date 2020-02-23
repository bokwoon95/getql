package getql

import (
	"fmt"
	"testing"
)

func TestSelect(t *testing.T) {
	values := []KV{
		KV{"EQ", "="},
		KV{"NE", "<>"},
		KV{"IN", "IN"},
		KV{"", ""},
	}
	params := map[string][]string{
		"nama": []string{"EQ", "NE", "EQ"},
	}
	x := Select(params)("nama", values, "IN", "ml2")
	fmt.Println(x)
}
