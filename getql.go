package getql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Suffixes
const (
	Sel = "SEL" // SELECT
	Frm = "FRM" // FROM

	// Column  Operator  Value/Values
	// name    =         'bob'
	// name    IN        ('bob', 'alice')
	col = "COL" // Column
	opr = "OPR" // Operator
	val = "VAL" // Value

	ord = "ORD" // ORDER BY
	Lim = "LIM" // LIMIT
	Off = "OFF" // OFFSET

	aor    = "AOR" // And/Or
	Page   = "PAGE"
	Filter = "FILTER"
)

func IsValidSuffix(suffix string) bool {
	suffixes := map[string]bool{
		Sel:  true,
		Frm:  true,
		col:  true,
		opr:  true,
		val:  true,
		ord:  true,
		Lim:  true,
		aor:  true,
		Page: true,
	}
	return suffixes[suffix]
}

func Col(strs ...string) string { return strings.Join(append(strs, col), Sep) }
func Opr(strs ...string) string { return strings.Join(append(strs, opr), Sep) }
func Val(strs ...string) string { return strings.Join(append(strs, val), Sep) }
func Ord(strs ...string) string { return strings.Join(append(strs, ord), Sep) }
func Aor(strs ...string) string { return strings.Join(append(strs, aor), Sep) }

// Notice all the suffixes have the same length, which is so that a fixed
// number of characters can be chopped off the end of a string to be used in a
// switch statement
const suffixLen = 3

// The separator between the prefixes and suffix
const Sep = "."

// just a helper variable
const space = " "

// SQL Keywords
const (
	Count = "COUNT(*)"
	Asc   = "ASC"
	Desc  = "DESC"
	And   = "AND"
	Or    = "OR"
)

// Operators
const (
	Eq      = "EQ"
	Ne      = "NE"
	In      = "IN"
	Gt      = "GT"
	Ge      = "GE"
	Lt      = "LT"
	Le      = "LE"
	Null    = "NULL"
	NotNull = "NOTNULL"
	Like    = "LIKE"
	ILike   = "ILIKE"
	Between = "BETWEEN"
	Ignore  = "IGNORE"
)

type OrderBy struct {
	Column string
	Order  string // "ASC" or "DESC"
}

func (orderby OrderBy) String() string {
	if orderby.Column == "" || orderby.Order == "" {
		return ""
	}
	return orderby.Column + " " + orderby.Order
}

func ScrubForm(form url.Values) url.Values {
	for key, _ := range form {
		strs := strings.Split(key, Sep)
		suffix := strs[len(strs)-1]
		if !IsValidSuffix(suffix) {
			form.Del(key)
		}
	}
	return form
}

func ScrubRequest(r *http.Request) *http.Request {
	r.ParseForm()
	for key, _ := range r.Form {
		strs := strings.Split(key, Sep)
		suffix := strs[len(strs)-1]
		if !IsValidSuffix(suffix) {
			r.Form.Del(key)
		}
	}
	return r
}

func ScrubUrl(url string, form url.Values) string {
	form = ScrubForm(form)
	if len(form) > 0 {
		url += "?" + form.Encode()
	}
	return url
}

type SelectQuery struct {
	Select   []string
	From     string
	Where    *PredGrp
	OrderBys []OrderBy
	Limit    int
	Offset   int
}

type PredGrp struct {
	Or    bool
	Preds map[string]*Pred
}

// Column  Operator  Value/Values
// name    =         'bob'
// name    IN        ('bob', 'alice')
type Pred struct {
	Column   string
	Operator string
	Value    string
	Values   []string
	Nested   bool
	PredGrp  *PredGrp
}

func ParseSelect(params map[string][]string) (query SelectQuery) {
	// Return first string from params[name], or empty string
	paramvalue := func(name string) (value string) {
		if values := params[name]; len(values) > 0 {
			value = values[0]
		}
		return value
	}
	// Return first string from params[name] converted into int, or 0
	paramvalueInt := func(name string) (value int) {
		if values := params[name]; len(values) > 0 {
			value, _ = strconv.Atoi(values[0]) // Don't care if it fails, zero value is fine
		}
		return value
	}
	query.Select = dedup(removeEmptyStrings(params[Sel]))
	query.From = paramvalue(Frm)
	query.Where = &PredGrp{}
	query.Limit = paramvalueInt(Lim)
	query.Offset = paramvalueInt(Off)
	orderbyMap := make(map[string]OrderBy)
	orderbyKeys := make([]string, 0)
	var ref *PredGrp
	for name, values := range params {
		strs := strings.Split(name, Sep)
		if len(strs) == 0 {
			continue
		}
		prefixes := strs[:len(strs)-1]
		suffix := strs[len(strs)-1]
		if !IsValidSuffix(suffix) {
			continue
		}
		value := paramvalue(name)
		values = dedup(values)
		ref = query.Where
		switch suffix {
		case ord:
			var orderby OrderBy
			for _, value := range values {
				if value == Ignore {
					break
				}
				if value == Asc || value == Desc {
					orderby.Order = value
				} else {
					orderby.Column = value
				}
				if orderby.String() != "" {
					orderbyMap[name] = orderby
					orderbyKeys = append(orderbyKeys, name)
					break
				}
			}
		case col, opr, val, aor:
			for i, prefix := range prefixes {
				if ref == nil {
					ref = &PredGrp{}
				}
				if ref.Preds == nil {
					ref.Preds = make(map[string]*Pred)
				}
				if ref.Preds[prefix] == nil {
					ref.Preds[prefix] = &Pred{}
				}
				if i == len(prefixes)-1 {
					switch suffix {
					case col:
						ref.Preds[prefix].Column = value
					case opr:
						ref.Preds[prefix].Operator = value
					case val:
						ref.Preds[prefix].Value = value
						ref.Preds[prefix].Values = values
					case aor:
						if ref.Preds[prefix].PredGrp == nil {
							ref.Preds[prefix].PredGrp = &PredGrp{}
						}
						ref.Preds[prefix].PredGrp.Or = value == Or
					}
					break
				}
				if ref.Preds[prefix].PredGrp == nil {
					ref.Preds[prefix].PredGrp = &PredGrp{}
				}
				ref.Preds[prefix].Nested = true
				ref = ref.Preds[prefix].PredGrp
			}
		}
	}
	sort.Strings(orderbyKeys)
	for _, key := range orderbyKeys {
		orderby := orderbyMap[key]
		if orderby.String() != "" {
			query.OrderBys = append(query.OrderBys, orderby)
		}
	}
	return query
}

type SelectOption func(SelectQuery) SelectQuery

var SelectCount SelectOption = func(sq SelectQuery) SelectQuery {
	sq.Select = []string{Count}
	sq.OrderBys = nil
	sq.Limit = 0
	sq.Offset = 0
	return sq
}

var SelectAll SelectOption = func(sq SelectQuery) SelectQuery {
	sq.Select = []string{"*"}
	return sq
}

var WhereOnly SelectOption = func(sq SelectQuery) SelectQuery {
	sq.Select = nil
	sq.From = ""
	sq.OrderBys = nil
	sq.Limit = 0
	sq.Offset = 0
	return sq
}

func (sq SelectQuery) Sql(options ...SelectOption) (query string, args []interface{}) {
	for _, option := range options {
		sq = option(sq)
	}
	var selectStr, whereStr, orderByStr string
	selectStr = strings.Join(dedup(removeEmptyStrings(sq.Select)), ","+space)
	whereStr, args = stringifyWhere(sq.Where)
	orderByStr = stringifyOrder(sq.OrderBys)
	buf := &strings.Builder{}
	if selectStr != "" {
		if buf.Len() > 0 {
			buf.WriteString(space)
		}
		buf.WriteString("SELECT" + space + selectStr)
	}
	if sq.From != "" {
		if buf.Len() > 0 {
			buf.WriteString(space)
		}
		buf.WriteString("FROM" + space + sq.From)
	}
	if whereStr != "" {
		if buf.Len() > 0 {
			buf.WriteString(space)
		}
		buf.WriteString("WHERE" + space + whereStr)
	}
	if orderByStr != "" {
		if buf.Len() > 0 {
			buf.WriteString(space)
		}
		buf.WriteString("ORDER BY" + space + orderByStr)
	}
	if sq.Limit != 0 {
		if buf.Len() > 0 {
			buf.WriteString(space)
		}
		buf.WriteString("LIMIT" + space + strconv.Itoa(sq.Limit))
	}
	if sq.Offset != 0 {
		if buf.Len() > 0 {
			buf.WriteString(space)
		}
		buf.WriteString("OFFSET" + space + strconv.Itoa(sq.Offset))
	}
	query = buf.String()
	query = ReplacePlaceholders(query)
	return query, args
}

func stringifyWhere(where *PredGrp) (whereStr string, args []interface{}) {
	if where == nil {
		return whereStr, args
	}
	buf := &strings.Builder{}
	conjuctor := And
	if where.Or {
		conjuctor = Or
	}
	for _, pred := range where.Preds {
		predStr, argsTemp := stringifyPred(pred)
		if predStr != "" {
			if buf.Len() > 0 {
				buf.WriteString(space + conjuctor + space)
			}
			buf.WriteString(predStr)
			args = append(args, argsTemp...)
		}
	}
	whereStr = buf.String()
	return whereStr, args
}

func stringifyPred(pred *Pred) (predStr string, args []interface{}) {
	if pred == nil {
		return predStr, args
	}
	if pred.Nested {
		whereStr, argsTemp := stringifyWhere(pred.PredGrp)
		args = append(args, argsTemp...)
		return "(" + whereStr + ")", args
	}
	// Because we are going to be using ? as placeholders, we need to escape any existing ? into ??
	pred.Column = strings.ReplaceAll(pred.Column, "?", "??")
	pred.Operator = strings.TrimSpace(pred.Operator)
	pred.Values = dedup(pred.Values)
	if pred.Column == "" {
		return predStr, args
	}
	switch pred.Operator {
	case Eq:
		return fmt.Sprintf("%s = ?", pred.Column), []interface{}{pred.Value}
	case Ne:
		return fmt.Sprintf("%s <> ?", pred.Column), []interface{}{pred.Value}
	case In:
		var placeholders []string
		for _, val := range pred.Values {
			placeholders = append(placeholders, "?")
			args = append(args, val)
		}
		return fmt.Sprintf("%s IN (%s)", pred.Column, strings.Join(placeholders, ","+space)), args
	case Gt:
		return fmt.Sprintf("%s > ?", pred.Column), []interface{}{pred.Value}
	case Ge:
		return fmt.Sprintf("%s >= ?", pred.Column), []interface{}{pred.Value}
	case Lt:
		return fmt.Sprintf("%s < ?", pred.Column), []interface{}{pred.Value}
	case Le:
		return fmt.Sprintf("%s <= ?", pred.Column), []interface{}{pred.Value}
	case Null:
		return fmt.Sprintf("%s IS NULL", pred.Column), []interface{}{}
	case NotNull:
		return fmt.Sprintf("%s IS NOT NULL", pred.Column), []interface{}{}
	case Like:
		return fmt.Sprintf("%s LIKE ?", pred.Column), []interface{}{pred.Value}
	case ILike:
		return fmt.Sprintf("%s ILIKE ?", pred.Column), []interface{}{pred.Value}
	case Between:
		if len(pred.Values) < 2 {
			return "", []interface{}{}
		}
		smaller, greater := pred.Values[0], pred.Values[1]
		return fmt.Sprintf("%s BETWEEN ? AND ?", pred.Column), []interface{}{smaller, greater}
	default:
		return predStr, args
	}
}

func stringifyOrder(orderBys []OrderBy) (order string) {
	buf := &strings.Builder{}
	for _, o := range orderBys {
		if o.String() != "" {
			if buf.Len() > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(o.String())
		}
	}
	order = buf.String()
	return order
}

func Subst(query string, args ...interface{}) string {
	query = regexp.MustCompile(`(?m)--.*$`).ReplaceAllString(query, " ") // Remove comments
	query = regexp.MustCompile(`\\n|\\t`).ReplaceAllString(query, " ")   // Remove newlines/tabs
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")       // Replace multiple spaces with one space
	query = strings.TrimSpace(query)
	for i, arg := range args {
		var val string
		switch v := arg.(type) {
		case string:
			val = fmt.Sprintf("'%s'", v)
		case int64:
			val = strconv.FormatInt(v, 10)
		case int32:
			val = strconv.FormatInt(int64(v), 10)
		case int:
			val = strconv.Itoa(v)
		case time.Time:
			val = fmt.Sprintf("'%s'", v.Format(time.RFC3339))
		case nil:
			val = "NULL"
		default:
			// Try to unmarshal arg into a json string. If that fails, then give up and return
			b, err := json.Marshal(arg)
			if err != nil {
				return query + space + err.Error()
			}
			val = fmt.Sprintf("'%s'", string(b))
		}
		query = strings.ReplaceAll(query, "$"+strconv.Itoa(i+1), val)
	}
	if !strings.HasSuffix(query, ";") {
		query = query + ";"
	}
	return query
}

// Replace all ? with postgres placeholders $<number>
func ReplacePlaceholders(query string) string {
	buf := &bytes.Buffer{}
	i := 0
	for {
		p := strings.Index(query, "?")
		if p < 0 {
			break
		}
		if len(query[p:]) > 1 && query[p:p+2] == "??" { // Unescape ?? -> ?
			buf.WriteString(query[:p])
			buf.WriteString("?")
			query = query[p+2:]
		} else { // Replace ? -> $<number>
			i++
			buf.WriteString(query[:p])
			buf.WriteString("$" + strconv.Itoa(i))
			query = query[p+1:]
		}
	}
	buf.WriteString(query)
	return buf.String()
}

// Deduplicate slice, maintaining order
func dedup(values []string) (deduped []string) {
	uniq := make(map[string]bool)
	for _, value := range values {
		if !uniq[value] {
			deduped = append(deduped, value)
			uniq[value] = true
		}
	}
	return deduped
}

// Return new slice with empty strings removed
func removeEmptyStrings(values []string) (purged []string) {
	for _, value := range values {
		if value == "" {
			continue
		}
		purged = append(purged, value)
	}
	return purged
}

type SelectStats struct {
	Query      string
	Total      int
	Limit      int
	Page       int
	TotalPages int
}

type SelectStatsConfig struct {
	MinimumLimit int
	QueryOptions []SelectOption
}

type SelectStatsOption func(SelectStatsConfig) SelectStatsConfig

func SelectStatsMinimumLimit(limit int) SelectStatsOption {
	return func(config SelectStatsConfig) SelectStatsConfig {
		config.MinimumLimit = limit
		return config
	}
}

var SelectStatsQueryAll SelectStatsOption = func(config SelectStatsConfig) SelectStatsConfig {
	config.QueryOptions = append(config.QueryOptions, SelectAll)
	return config
}

func DBSelectWithStats(db *sqlx.DB, params map[string][]string, options ...SelectStatsOption) (rows *sqlx.Rows, stats SelectStats, err error) {
	config := SelectStatsConfig{MinimumLimit: 5}
	for _, option := range options {
		config = option(config)
	}
	sq := ParseSelect(params)
	// stats.Total
	query, args := sq.Sql(SelectCount)
	err = db.QueryRowx(query, args...).Scan(&stats.Total)
	if err != nil {
		return rows, stats, err
	}
	// stats.Limit
	if sq.Limit < config.MinimumLimit {
		sq.Limit = config.MinimumLimit
	}
	stats.Limit = sq.Limit
	// stats.Page
	stats.Page = 1
	if params[Page] != nil {
		page, _ := strconv.Atoi(params[Page][0])
		if page > stats.Page {
			stats.Page = page
		}
	}
	sq.Offset = stats.Limit * (stats.Page - 1)
	// stats.TotalPages
	stats.TotalPages = int(math.Ceil(float64(stats.Total) / float64(stats.Limit)))
	// stats.Query
	query, args = sq.Sql(config.QueryOptions...)
	stats.Query = Subst(query, args...)
	// rows
	query, args = sq.Sql()
	rows, err = db.Queryx(query, args...)
	return rows, stats, err
}

func ResolvePage(params map[string][]string) map[string][]string {
	limit := 5
	page := 1
	if params[Lim] != nil {
		l, _ := strconv.Atoi(params[Lim][0])
		if l > limit {
			limit = l
		}
	}
	if params[Page] != nil {
		p, _ := strconv.Atoi(params[Page][0])
		if p > page {
			page = p
		}
	}
	offset := limit * (page - 1)
	params[Off] = []string{strconv.Itoa(offset)}
	return params
}

func PaginateHandlerFunc(url string, delta int, errorHandler func(http.ResponseWriter, *http.Request, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			errorHandler(w, r, err)
			return
		}
		params := ScrubForm(r.Form)
		page, _ := strconv.Atoi(r.FormValue(Page))
		if page != 0 {
			page += delta
			params[Page] = []string{strconv.Itoa(page)}
		}
		if page == 0 {
			params[Page] = []string{"1"}
		}
		newUrl := ScrubUrl(url, params)
		http.Redirect(w, r, newUrl, http.StatusMovedPermanently)
	}
}
