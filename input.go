package getql

import (
	"bytes"
	"html/template"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func AddKeywords(funcs map[string]interface{}) map[string]interface{} {
	funcs["GetqlSel"] = func() string { return Sel }
	funcs["GetqlFrm"] = func() string { return Frm }
	funcs["GetqlLim"] = func() string { return Lim }
	funcs["GetqlPage"] = func() string { return Page }
	funcs["GetqlCol"] = Col
	funcs["GetqlOpr"] = Opr
	funcs["GetqlVal"] = Val
	funcs["GetqlOrd"] = Ord
	funcs["GetqlAor"] = Aor
	funcs["GetqlJoin"] = Join

	funcs["GetqlEq"] = func() string { return Eq }
	funcs["GetqlNe"] = func() string { return Ne }
	funcs["GetqlIn"] = func() string { return In }
	funcs["GetqlGt"] = func() string { return Gt }
	funcs["GetqlGe"] = func() string { return Ge }
	funcs["GetqlLt"] = func() string { return Lt }
	funcs["GetqlLe"] = func() string { return Le }
	funcs["GetqlNull"] = func() string { return Null }
	funcs["GetqlNotNull"] = func() string { return NotNull }
	funcs["GetqlLike"] = func() string { return Like }
	funcs["GetqlILike"] = func() string { return ILike }
	funcs["GetqlBetween"] = func() string { return Between }
	funcs["GetqlIgnore"] = func() string { return Ignore }
	funcs = AddOperatorKV(funcs)

	funcs["GetqlAsc"] = func() string { return Asc }
	funcs["GetqlDesc"] = func() string { return Desc }
	funcs["GetqlAscDescKV"] = func() []KV {
		return []KV{
			KV{Key: Asc, Value: "Ascending"},
			KV{Key: Desc, Value: "Descending"},
			KV{Key: Ignore, Value: "IGNORE"},
		}
	}

	funcs["GetqlAnd"] = func() string { return And }
	funcs["GetqlOr"] = func() string { return Or }
	funcs["GetqlAndOrKV"] = func() []KV {
		return []KV{
			KV{Key: And, Value: "AND"},
			KV{Key: Or, Value: "OR"},
			KV{Key: Ignore, Value: "IGNORE"},
		}
	}
	return funcs
}

func Funcs(funcs map[string]interface{}, params map[string][]string) map[string]interface{} {
	funcs = AddKeywords(funcs)
	funcs["GetqlFilter"] = func() string { return "getql-filter" }
	funcs["GetqlFilterCheckbox"] = FilterCheckbox(params)
	funcs["GetqlFilterCheckboxLabel"] = func() string { return "getql-filter-checkbox" }
	funcs["Input_Select"] = Select(params)
	funcs["Input_Multiselect"] = Multiselect(params)
	funcs["Input_SelectOptional"] = SelectOptional(params)
	funcs["Input_MultiselectOptional"] = MultiselectOptional(params)
	funcs["Input_Text"] = Text(params)
	funcs["Input_Multitext"] = Multitext(params)
	funcs["Input_Number"] = Number(params)
	funcs["Input_Multinumber"] = Multinumber(params)
	funcs["Input_Date"] = Date(params)
	return funcs
}

type KV struct {
	Key   string
	Value string
}

func Multiselect(params map[string][]string) func(string, []KV, string, string) template.HTML {
	return func(name string, values []KV, defaultValue, class string) template.HTML {
		buf := &bytes.Buffer{}
		paramvals := params[name]
		if len(paramvals) == 0 {
			node := SelectNode(name, values, defaultValue, class)
			html.Render(buf, node)
		}
		for _, paramval := range paramvals {
			node := SelectNode(name, values, paramval, class)
			html.Render(buf, node)
		}
		return template.HTML(buf.String())
	}
}

func Select(params map[string][]string) func(string, []KV, string, string) template.HTML {
	return func(name string, values []KV, defaultValue, class string) template.HTML {
		buf := &bytes.Buffer{}
		paramvals := params[name]
		if len(paramvals) == 0 {
			node := SelectNode(name, values, defaultValue, class)
			html.Render(buf, node)
		} else {
			selected := defaultValue
			for _, value := range values {
				if value.Key == paramvals[0] {
					selected = paramvals[0]
					break
				}
			}
			node := SelectNode(name, values, selected, class)
			html.Render(buf, node)
		}
		return template.HTML(buf.String())
	}
}

func SelectOptional(params map[string][]string) func(string, []KV, string, string) template.HTML {
	return func(name string, values []KV, defaultValue, class string) template.HTML {
		var newValues []KV
		newValues = append(newValues, KV{Key: Ignore, Value: "(IGNORE)"})
		newValues = append(newValues, values...)
		return Select(params)(name, newValues, defaultValue, class)
	}
}

func MultiselectOptional(params map[string][]string) func(string, []KV, string, string) template.HTML {
	return func(name string, values []KV, defaultValue, class string) template.HTML {
		var newValues []KV
		newValues = append(newValues, KV{Key: Ignore, Value: "(IGNORE)"})
		newValues = append(newValues, values...)
		return Multiselect(params)(name, newValues, defaultValue, class)
	}
}

func SelectNode(name string, values []KV, selected, class string) (node *html.Node) {
	node = &html.Node{
		Type: html.ElementNode,
		Data: "select",
		Attr: []html.Attribute{
			html.Attribute{Key: "name", Val: name},
			html.Attribute{Key: "class", Val: class},
		},
	}
	for _, v := range values {
		option := &html.Node{
			Type: html.ElementNode,
			Data: "option",
			Attr: []html.Attribute{
				html.Attribute{Key: "value", Val: v.Key},
			},
			FirstChild: &html.Node{Type: html.TextNode, Data: v.Value},
		}
		if v.Key == selected {
			option.Attr = append(option.Attr, html.Attribute{Key: "selected", Val: "selected"})
		}
		node.AppendChild(option)
	}
	return node
}

func Multitext(params map[string][]string) func(string, string, string) template.HTML {
	return func(name, defaultValue, class string) template.HTML {
		buf := &bytes.Buffer{}
		paramvals := params[name]
		if len(paramvals) == 0 {
			node := TextNode(name, defaultValue, class)
			html.Render(buf, node)
		}
		for _, paramval := range paramvals {
			node := TextNode(name, paramval, class)
			html.Render(buf, node)
		}
		return template.HTML(buf.String())
	}
}

func Text(params map[string][]string) func(string, string, string) template.HTML {
	return func(name, defaultValue, class string) template.HTML {
		buf := &bytes.Buffer{}
		paramvals := params[name]
		if len(paramvals) == 0 {
			node := TextNode(name, defaultValue, class)
			html.Render(buf, node)
		} else {
			node := TextNode(name, paramvals[0], class)
			html.Render(buf, node)
		}
		return template.HTML(buf.String())
	}
}

func TextNode(name, value, class string) (node *html.Node) {
	return &html.Node{
		Type: html.ElementNode,
		Data: "input",
		Attr: []html.Attribute{
			html.Attribute{Key: "type", Val: "text"},
			html.Attribute{Key: "name", Val: name},
			html.Attribute{Key: "value", Val: value},
			html.Attribute{Key: "class", Val: class},
			html.Attribute{Key: "autocomplete", Val: "off"},
		},
	}
}

func Date(params map[string][]string) func(string, string, string) template.HTML {
	return func(name, defaultValue, class string) template.HTML {
		buf := &bytes.Buffer{}
		paramvals := params[name]
		if len(paramvals) == 0 {
			node := DateNode(name, defaultValue, class)
			html.Render(buf, node)
		}
		for _, paramval := range paramvals {
			node := DateNode(name, paramval, class)
			html.Render(buf, node)
		}
		return template.HTML(buf.String())
	}
}

func DateNode(name, value, class string) (node *html.Node) {
	return &html.Node{
		Type: html.ElementNode,
		Data: "input",
		Attr: []html.Attribute{
			html.Attribute{Key: "type", Val: "date"},
			html.Attribute{Key: "name", Val: name},
			html.Attribute{Key: "value", Val: value},
			html.Attribute{Key: "class", Val: class},
		},
	}
}

func FilterCheckbox(params map[string][]string) func() template.HTML {
	buf := &strings.Builder{}
	node := &html.Node{
		Type: html.ElementNode,
		Data: "input",
		Attr: []html.Attribute{
			html.Attribute{Key: "type", Val: "checkbox"},
			html.Attribute{Key: "id", Val: "getql-filter-checkbox"},
		},
	}
	paramvals := params[Filter]
	if len(paramvals) != 0 {
		node.Attr = append(node.Attr, html.Attribute{Key: "checked", Val: "checked"})
	}
	html.Render(buf, node)
	str := buf.String()
	return func() template.HTML {
		return template.HTML(str)
	}
}

func Number(params map[string][]string) func(string, string, string) template.HTML {
	return func(name, defaultValue, class string) template.HTML {
		buf := &bytes.Buffer{}
		paramvals := params[name]
		if len(paramvals) == 0 {
			node := NumberNode(name, defaultValue, class)
			html.Render(buf, node)
		} else {
			node := NumberNode(name, paramvals[0], class)
			html.Render(buf, node)
		}
		return template.HTML(buf.String())
	}
}

func Multinumber(params map[string][]string) func(string, string, string) template.HTML {
	return func(name, defaultValue, class string) template.HTML {
		buf := &bytes.Buffer{}
		paramvals := params[name]
		if len(paramvals) == 0 {
			node := NumberNode(name, defaultValue, class)
			html.Render(buf, node)
		}
		for _, paramval := range paramvals {
			node := NumberNode(name, paramval, class)
			html.Render(buf, node)
		}
		return template.HTML(buf.String())
	}
}

func NumberNode(name, value, class string) (node *html.Node) {
	return &html.Node{
		Type: html.ElementNode,
		Data: "input",
		Attr: []html.Attribute{
			html.Attribute{Key: "type", Val: "number"},
			html.Attribute{Key: "name", Val: name},
			html.Attribute{Key: "value", Val: value},
			html.Attribute{Key: "class", Val: class},
		},
	}
}

func AddOperatorKV(funcs map[string]interface{}) map[string]interface{} {
	funcs["GetqlOprKV"] = func() []KV {
		return []KV{
			KV{Eq, "is equal to"},
			KV{Ne, "is not equal to"},
			KV{In, "is one of"},
			KV{Gt, "is greater than"},
			KV{Ge, "is greater or equal to"},
			KV{Lt, "is less than"},
			KV{Le, "is less or equal to"},
			KV{Null, "is null"},
			KV{NotNull, "is not null"},
			KV{Between, "is between"},
			KV{Like, "is like"},
			KV{ILike, "is ilike"},
			KV{Ignore, "(IGNORE)"},
		}
	}
	funcs["GetqlTextOprKV"] = func() []KV {
		return []KV{
			KV{Eq, "is equal to"},
			KV{Ne, "is not equal to"},
			KV{In, "is one of"},
			KV{Null, "is null"},
			KV{NotNull, "is not null"},
			KV{Like, "is like"},
			KV{ILike, "is ilike"},
			KV{Ignore, "(IGNORE)"},
		}
	}
	funcs["GetqlNumOprKV"] = func() []KV {
		return []KV{
			KV{Eq, "is equal to"},
			KV{Ne, "is not equal to"},
			KV{In, "is one of"},
			KV{Gt, "is greater than"},
			KV{Ge, "is greater or equal to"},
			KV{Lt, "is less than"},
			KV{Le, "is less or equal to"},
			KV{Null, "is null"},
			KV{NotNull, "is not null"},
			KV{Between, "is between"},
			KV{Ignore, "(IGNORE)"},
		}
	}
	funcs["GetqlEnumOprKV"] = func() []KV {
		return []KV{
			KV{Eq, "is equal to"},
			KV{Ne, "is not equal to"},
			KV{In, "is one of"},
			KV{Null, "is null"},
			KV{NotNull, "is not null"},
			KV{Ignore, "(IGNORE)"},
		}
	}
	return funcs
}

func Join(items ...interface{}) string {
	var strs []string
	for _, item := range items {
		switch v := item.(type) {
		case string:
			strs = append(strs, v)
		case int:
			strs = append(strs, strconv.Itoa(v))
		case int64:
			strs = append(strs, strconv.FormatInt(v, 10))
		case int32:
			strs = append(strs, strconv.FormatInt(int64(v), 10))
		}
	}
	return strings.Join(strs, Sep)
}
