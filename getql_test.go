package getql

import (
	"fmt"
	"testing"
)

func TestParseSelect(t *testing.T) {
	params := map[string][]string{
		Sel: []string{"a", "b", "c"},
		Frm: []string{"tabel"},

		Col("1"): []string{"A"},
		Opr("1"): []string{Eq},
		Val("1"): []string{"bullymong"},

		Col("2"): []string{"B"},
		Opr("2"): []string{Between},
		Val("2"): []string{"9", "10"},

		Col("3"): []string{"C"},
		Opr("3"): []string{In},
		Val("3"): []string{"x", "y", "z"},

		Aor("4"): []string{Or},

		Col("4", "1"): []string{"student1"},
		Opr("4", "1"): []string{Eq},
		Val("4", "1"): []string{"john"},

		Col("4", "2"): []string{"student2"},
		Opr("4", "2"): []string{Eq},
		Val("4", "2"): []string{"john"},

		Ord("1"): []string{"nyeh"},
		Ord("2"): []string{"esc", "ASC"},
		Ord("3"): []string{"DESC", "dasc"},

		Lim: []string{"1"},
		Off: []string{"2"},
	}
	sq := ParseSelect(params)
	query, args := sq.Sql()
	fmt.Println(query, args)
	fmt.Println(Subst(query, args...))
	fmt.Println(sq.Sql(SelectCount))
	fmt.Println(sq.Sql(WhereOnly))
}
