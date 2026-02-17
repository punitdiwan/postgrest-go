package main

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/jackc/pgx/v5"
)

type SQLQuery struct {
	Query  string
	Values []interface{}
}

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type JoinConfig struct {
	Table   string
	Alias   string
	OnLeft  string
	OnRight string
	Columns []string
	Filters map[string][]string
	OrderBy string
	Limit   int
	Offset  int
}

func BuildQuery(ctx context.Context, db Querier, table string, params url.Values) SQLQuery {
	dialect := goqu.Dialect("postgres")
	query := dialect.From(table)

	// Handle SELECT columns (support nested embedding like directors(id,last_name))
	if s := params.Get("select"); s != "" {
		// Parse comma-separated fields but keep parentheses groups together
		fields := splitSelectFields(s)
		selectCols := make([]interface{}, 0, len(fields))
		for _, f := range fields {
			f = strings.TrimSpace(f)
			// nested like: related_table(col1,col2)
			if strings.Contains(f, "(") && strings.Contains(f, ")") {
				// extract table and columns
				re := regexp.MustCompile(`^(\w+)\((.*)\)$`)
				m := re.FindStringSubmatch(f)
				if len(m) == 3 {
					relTable := m[1]
					colsStr := m[2]
					cols := []string{}
					if colsStr != "" {
						for _, c := range strings.Split(colsStr, ",") {
							cols = append(cols, strings.TrimSpace(c))
						}
					}

					// Determine relationship type
					mainSingular := singularize(table)
					relSingular := singularize(relTable)

					// Pattern 1: Many-to-one - main_table.{related_table_singular}_id = related_table.id
					// Pattern 2: One-to-many - related_table.{main_table_singular}_id = main_table.id
					manyToOneFK := fmt.Sprintf("%s_%s", relSingular, "id")  // e.g., "author_id"
					oneToManyFK := fmt.Sprintf("%s_%s", mainSingular, "id") // e.g., "post_id"

					// Build column selection for related table
					colList := ""
					if len(cols) == 0 {
						colList = "*"
					} else {
						colList = strings.Join(cols, ",")
					}

					// Check which foreign key exists to determine relationship type
					isManyToOne := columnExists(ctx, db, table, manyToOneFK)

					var sub string
					if isManyToOne {
						// Many-to-one relationship: main_table.{related_table_singular}_id = related_table.id
						// Return a single JSON object (not an array)
						sub = fmt.Sprintf(
							"(SELECT row_to_json(%s.*) FROM %s WHERE %s.id = %s.%s LIMIT 1) AS %s",
							relTable, relTable, relTable, table, manyToOneFK, relTable,
						)
					} else {
						// One-to-many relationship: related_table.{main_table_singular}_id = main_table.id
						// Return an array of JSON objects
						sub = fmt.Sprintf(
							"(SELECT COALESCE(json_agg(row_to_json(arr)), '[]'::json) FROM (SELECT %s FROM %s WHERE %s.%s = %s.id) arr) AS %s",
							colList, relTable, relTable, oneToManyFK, table, relTable,
						)
					}

					selectCols = append(selectCols, goqu.L(sub))
					continue
				}
			}

			// regular column
			selectCols = append(selectCols, goqu.C(f))
		}

		if len(selectCols) > 0 {
			query = query.Select(selectCols...)
		} else {
			query = query.Select("*")
		}
	} else {
		query = query.Select("*")
	}

	// Handle WHERE conditions for main table
	for key, val := range params {
		if key == "select" || key == "order" || key == "limit" || key == "offset" || strings.Contains(key, ".") {
			continue
		}
		// Skip if this is a related resource (contains parentheses or is a table reference)
		if strings.Contains(key, "(") || isRelatedResource(key, params) {
			continue
		}

		parts := strings.SplitN(val[0], ".", 2)
		if len(parts) != 2 {
			continue
		}

		op, v := parts[0], parts[1]
		col := goqu.C(key)

		switch op {
		case "eq":
			query = query.Where(col.Eq(v))
		case "gt":
			query = query.Where(col.Gt(v))
		case "lt":
			query = query.Where(col.Lt(v))
		case "gte":
			query = query.Where(col.Gte(v))
		case "lte":
			query = query.Where(col.Lte(v))
		case "like":
			// Use goqu's Like to safely build LIKE expressions with identifiers
			query = query.Where(goqu.C(key).Like(v))
		case "in":
			values := strings.Split(v, ",")
			query = query.Where(col.In(values))
		}
	}

	// Handle dynamic joins
	joins := parseJoins(params, table)
	for _, join := range joins {
		query = applyJoin(query, join, table)
	}

	// Handle ORDER BY
	if order := params.Get("order"); order != "" {
		parts := strings.SplitN(order, ".", 2)
		col := goqu.C(parts[0])
		if len(parts) > 1 && parts[1] == "desc" {
			query = query.Order(col.Desc())
		} else {
			query = query.Order(col.Asc())
		}
	}

	// Handle LIMIT
	if limit := params.Get("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil {
			query = query.Limit(uint(n))
		}
	}

	// Handle OFFSET
	if offset := params.Get("offset"); offset != "" {
		if n, err := strconv.Atoi(offset); err == nil {
			query = query.Offset(uint(n))
		}
	}

	sql, values, _ := query.ToSQL()

	return SQLQuery{
		Query:  sql,
		Values: values,
	}
}

// isRelatedResource checks if a key is a related resource (like a table reference)
func isRelatedResource(key string, params url.Values) bool {
	// Check if there are any dot-notation params for this key (like "posts.select")
	for paramKey := range params {
		if strings.HasPrefix(paramKey, key+".") {
			return true
		}
	}
	return false
}

// parseJoins extracts join configurations from query parameters
func parseJoins(params url.Values, mainTable string) []JoinConfig {
	var joins []JoinConfig

	// List of filter operators that should not be treated as join configurations
	filterOperators := map[string]bool{
		"eq": true, "gt": true, "lt": true, "gte": true, "lte": true,
		"like": true, "in": true,
	}

	// Check if embedded relations are already in the select parameter (to avoid duplicate handling)
	embeddedRelations := extractEmbeddedRelations(params.Get("select"))

	// Look for patterns like: related_table=fk_column.pk_column
	// Or shorthand: related_table where we infer the foreign key
	for key, val := range params {
		if strings.Contains(key, ".") || key == "select" || key == "order" || key == "limit" || key == "offset" {
			continue
		}

		// Check if value is in format "fk.pk" (explicit join config)
		if len(val) > 0 && strings.Contains(val[0], ".") && !strings.Contains(val[0], "=") {
			parts := strings.SplitN(val[0], ".", 2)
			if len(parts) == 2 {
				// Skip if the first part is a filter operator (e.g., "eq", "gt", etc.)
				if filterOperators[parts[0]] {
					continue
				}

				join := JoinConfig{
					Table:   key,
					OnLeft:  parts[0],
					OnRight: parts[1],
				}

				// Extract columns for related table (select parameter)
				selectKey := key + ".select"
				if selectVal, ok := params[selectKey]; ok && len(selectVal) > 0 {
					join.Columns = strings.Split(selectVal[0], ",")
				}

				joins = append(joins, join)
			}
		}

		// Check for parentheses notation like: posts(id,title)
		// Skip if this relation is already embedded in the select parameter
		if strings.Contains(key, "(") && strings.Contains(key, ")") {
			relName := strings.SplitN(key, "(", 2)[0]
			if !embeddedRelations[relName] {
				join := parseParenthesesJoin(key)
				if join != nil {
					joins = append(joins, *join)
				}
			}
		}
	}

	return joins
}

// parseParenthesesJoin parses join in format: tablename(col1,col2)
func parseParenthesesJoin(param string) *JoinConfig {
	re := regexp.MustCompile(`^(\w+)\((.*)\)$`)
	matches := re.FindStringSubmatch(param)
	if len(matches) < 3 {
		return nil
	}

	tableName := matches[1]
	columnStr := matches[2]

	join := &JoinConfig{
		Table: tableName,
	}

	if columnStr != "" {
		cols := strings.Split(columnStr, ",")
		for i := range cols {
			cols[i] = strings.TrimSpace(cols[i])
		}
		join.Columns = cols
	}

	// Infer foreign key (assumption: fk column is {related_table}_id)
	join.OnLeft = fmt.Sprintf("%s_id", tableName)
	join.OnRight = "id"

	return join
}

// splitSelectFields splits top-level comma-separated select fields, keeping parenthesis groups intact
func splitSelectFields(s string) []string {
	var fields []string
	var cur strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case ',':
			if depth == 0 {
				fields = append(fields, cur.String())
				cur.Reset()
				continue
			}
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		fields = append(fields, cur.String())
	}
	return fields
}

// singularize makes a naive singular form for a plural table name
func singularize(name string) string {
	if strings.HasSuffix(name, "ies") {
		return strings.TrimSuffix(name, "ies") + "y"
	}
	if strings.HasSuffix(name, "s") {
		return strings.TrimSuffix(name, "s")
	}
	return name
}

// extractEmbeddedRelations extracts table names from embedded select fields like "posts(id,content)"
func extractEmbeddedRelations(selectStr string) map[string]bool {
	embedded := make(map[string]bool)
	if selectStr == "" {
		return embedded
	}
	fields := splitSelectFields(selectStr)
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if strings.Contains(f, "(") && strings.Contains(f, ")") {
			re := regexp.MustCompile(`^(\w+)\(`)
			m := re.FindStringSubmatch(f)
			if len(m) > 1 {
				embedded[m[1]] = true
			}
		}
	}
	return embedded
}

// applyJoin applies a JOIN to the query
func applyJoin(query *goqu.SelectDataset, join JoinConfig, mainTable string) *goqu.SelectDataset {
	// Build the join condition
	leftCol := goqu.T(mainTable).Col(join.OnLeft)
	rightCol := goqu.T(join.Table).Col(join.OnRight)

	// Apply left join with On condition
	query = query.LeftJoin(
		goqu.T(join.Table),
		goqu.On(leftCol.Eq(rightCol)),
	)

	// Add selected columns from joined table
	if len(join.Columns) > 0 {
		for _, col := range join.Columns {
			query = query.SelectAppend(goqu.T(join.Table).Col(strings.TrimSpace(col)))
		}
	} else {
		// Select all columns from joined table
		query = query.SelectAppend(goqu.T(join.Table).All())
	}

	return query
}

// columnExists checks if a column exists in a table by querying the database
func columnExists(ctx context.Context, db Querier, table string, column string) bool {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = $1 AND column_name = $2
		)
	`
	var exists bool
	err := db.QueryRow(ctx, query, table, column).Scan(&exists)
	if err != nil {
		// If there's an error, assume column doesn't exist
		return false
	}
	return exists
}
