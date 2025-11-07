package webdav

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SQLBuilder SQL查询构建器
type SQLBuilder struct {
	table      string
	selectCols []string
	joins      []string
	whereConds []string
	groupBy    []string
	havingCond []string
	orderBy    []string
	limitVal   int
	offsetVal  int
	args       []interface{}
}

// NewSelectBuilder 创建新的SELECT查询构建器
func NewSelectBuilder(table string, cols ...string) *SQLBuilder {
	var selectCols []string
	if len(cols) == 0 {
		selectCols = []string{"*"}
	} else {
		selectCols = cols
	}
	
	return &SQLBuilder{
		table:      table,
		selectCols: selectCols,
		args:       make([]interface{}, 0),
	}
}

// AddColumn 添加列
func (b *SQLBuilder) AddColumn(col ...string) *SQLBuilder {
	b.selectCols = append(b.selectCols, col...)
	return b
}

// Join 添加JOIN
func (b *SQLBuilder) Join(joinType, table, onCondition string) *SQLBuilder {
	b.joins = append(b.joins, fmt.Sprintf("%s %s ON %s", joinType, table, onCondition))
	return b
}

// Where 添加WHERE条件
func (b *SQLBuilder) Where(condition string, args ...interface{}) *SQLBuilder {
	b.whereConds = append(b.whereConds, condition)
	b.args = append(b.args, args...)
	return b
}

// And AND连接
func (b *SQLBuilder) And(condition string, args ...interface{}) *SQLBuilder {
	b.whereConds = append(b.whereConds, "AND "+condition)
	b.args = append(b.args, args...)
	return b
}

// Or OR连接
func (b *SQLBuilder) Or(condition string, args ...interface{}) *SQLBuilder {
	b.whereConds = append(b.whereConds, "OR "+condition)
	b.args = append(b.args, args...)
	return b
}

// GroupBy 添加GROUP BY
func (b *SQLBuilder) GroupBy(cols ...string) *SQLBuilder {
	b.groupBy = append(b.groupBy, cols...)
	return b
}

// Having 添加HAVING条件
func (b *SQLBuilder) Having(condition string, args ...interface{}) *SQLBuilder {
	b.havingCond = append(b.havingCond, condition)
	b.args = append(b.args, args...)
	return b
}

// OrderBy 添加ORDER BY
func (b *SQLBuilder) OrderBy(cols ...string) *SQLBuilder {
	b.orderBy = append(b.orderBy, cols...)
	return b
}

// Limit 设置LIMIT
func (b *SQLBuilder) Limit(n int) *SQLBuilder {
	b.limitVal = n
	return b
}

// Offset 设置OFFSET
func (b *SQLBuilder) Offset(n int) *SQLBuilder {
	b.offsetVal = n
	return b
}

// Args 获取参数
func (b *SQLBuilder) Args() []interface{} {
	return b.args
}

// Build 构建SQL语句
func (b *SQLBuilder) Build() string {
	var query strings.Builder
	
	// SELECT子句
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(b.selectCols, ", "))
	query.WriteString(" FROM " + b.table)
	
	// JOIN子句
	for _, join := range b.joins {
		query.WriteString(" " + join)
	}
	
	// WHERE子句
	if len(b.whereConds) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(b.whereConds, " "))
	}
	
	// GROUP BY子句
	if len(b.groupBy) > 0 {
		query.WriteString(" GROUP BY ")
		query.WriteString(strings.Join(b.groupBy, ", "))
	}
	
	// HAVING子句
	if len(b.havingCond) > 0 {
		query.WriteString(" HAVING ")
		query.WriteString(strings.Join(b.havingCond, " "))
	}
	
	// ORDER BY子句
	if len(b.orderBy) > 0 {
		query.WriteString(" ORDER BY ")
		query.WriteString(strings.Join(b.orderBy, ", "))
	}
	
	// LIMIT和OFFSET
	if b.limitVal > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", b.limitVal))
	}
	if b.offsetVal > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", b.offsetVal))
	}
	
	return query.String()
}

// ExecuteQuery 执行查询
func (b *SQLBuilder) ExecuteQuery(ctx context.Context, db *sql.DB) (*sql.Rows, error) {
	return db.QueryContext(ctx, b.Build(), b.args...)
}

// ExecuteQueryRow 执行单行查询
func (b *SQLBuilder) ExecuteQueryRow(ctx context.Context, db *sql.DB) *sql.Row {
	return db.QueryRowContext(ctx, b.Build(), b.args...)
}

// ExecuteExec 执行执行操作
func (b *SQLBuilder) ExecuteExec(ctx context.Context, db *sql.DB) (sql.Result, error) {
	return db.ExecContext(ctx, b.Build(), b.args...)
}

// NewInsertBuilder 创建新的INSERT查询构建器
type InsertBuilder struct {
	table      string
	cols       []string
	values     [][]interface{}
	args       []interface{}
	onConflict []string
}

// NewInsertBuilder 创建INSERT构建器
func NewInsertBuilder(table string) *InsertBuilder {
	return &InsertBuilder{
		table:  table,
		cols:   make([]string, 0),
		values: make([][]interface{}, 0),
		args:   make([]interface{}, 0),
	}
}

// Columns 设置列
func (i *InsertBuilder) Columns(cols ...string) *InsertBuilder {
	i.cols = append(i.cols, cols...)
	return i
}

// Values 添加值
func (i *InsertBuilder) Values(vals ...interface{}) *InsertBuilder {
	i.values = append(i.values, vals)
	i.args = append(i.args, vals...)
	return i
}

// OnConflict 添加ON CONFLICT子句
func (i *InsertBuilder) OnConflict(cols ...string) *InsertBuilder {
	i.onConflict = append(i.onConflict, cols...)
	return i
}

// Build 构建INSERT语句
func (i *InsertBuilder) Build() string {
	var query strings.Builder
	
	query.WriteString("INSERT INTO " + i.table)
	
	if len(i.cols) > 0 {
		query.WriteString(" (" + strings.Join(i.cols, ", ") + ")")
	}
	
	if len(i.values) > 0 {
		query.WriteString(" VALUES ")
		valuePlaceholders := make([]string, len(i.values[0]))
		for j := range valuePlaceholders {
			valuePlaceholders[j] = "?"
		}
		
		for idx, values := range i.values {
			if idx > 0 {
				query.WriteString(", ")
			}
			query.WriteString("(" + strings.Join(valuePlaceholders, ", ") + ")")
			// 将值添加到args中
			i.args = append(i.args, values...)
		}
	}
	
	if len(i.onConflict) > 0 {
		query.WriteString(" ON CONFLICT (" + strings.Join(i.onConflict, ", ") + ") DO NOTHING")
	}
	
	return query.String()
}

// Args 返回参数列表
func (i *InsertBuilder) Args() []interface{} {
	return i.args
}

// Execute 执行INSERT
func (i *InsertBuilder) Execute(ctx context.Context, db *sql.DB) (sql.Result, error) {
	return db.ExecContext(ctx, i.Build(), i.args...)
}

// NewUpdateBuilder 创建新的UPDATE查询构建器
type UpdateBuilder struct {
	table      string
	sets       map[string]interface{}
	conditions []string
	args       []interface{}
	orderBy    []string
	limitVal   int
}

// NewUpdateBuilder 创建UPDATE构建器
func NewUpdateBuilder(table string) *UpdateBuilder {
	return &UpdateBuilder{
		table:      table,
		sets:       make(map[string]interface{}),
		conditions: make([]string, 0),
		args:       make([]interface{}, 0),
		orderBy:    make([]string, 0),
	}
}

// Set 设置更新列
func (u *UpdateBuilder) Set(col string, val interface{}) *UpdateBuilder {
	u.sets[col] = val
	u.args = append(u.args, val)
	return u
}

// Where 设置WHERE条件
func (u *UpdateBuilder) Where(condition string, args ...interface{}) *UpdateBuilder {
	u.conditions = append(u.conditions, condition)
	u.args = append(u.args, args...)
	return u
}

// OrderBy 设置ORDER BY
func (u *UpdateBuilder) OrderBy(cols ...string) *UpdateBuilder {
	u.orderBy = append(u.orderBy, cols...)
	return u
}

// Limit 设置LIMIT
func (u *UpdateBuilder) Limit(n int) *UpdateBuilder {
	u.limitVal = n
	return u
}

// Build 构建UPDATE语句
func (u *UpdateBuilder) Build() string {
	var query strings.Builder
	
	query.WriteString("UPDATE " + u.table)
	
	// SET子句
	if len(u.sets) > 0 {
		sets := make([]string, 0, len(u.sets))
		for col := range u.sets {
			sets = append(sets, col+"=?")
		}
		query.WriteString(" SET " + strings.Join(sets, ", "))
	}
	
	// WHERE子句
	if len(u.conditions) > 0 {
		query.WriteString(" WHERE " + strings.Join(u.conditions, " AND "))
	}
	
	// ORDER BY子句
	if len(u.orderBy) > 0 {
		query.WriteString(" ORDER BY " + strings.Join(u.orderBy, ", "))
	}
	
	// LIMIT子句
	if u.limitVal > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", u.limitVal))
	}
	
	return query.String()
}

// Args 返回参数列表
func (u *UpdateBuilder) Args() []interface{} {
	return u.args
}

// Execute 执行UPDATE
func (u *UpdateBuilder) Execute(ctx context.Context, db *sql.DB) (sql.Result, error) {
	return db.ExecContext(ctx, u.Build(), u.args...)
}

// NewDeleteBuilder 创建新的DELETE查询构建器
type DeleteBuilder struct {
	table      string
	conditions []string
	args       []interface{}
	orderBy    []string
	limitVal   int
}

// NewDeleteBuilder 创建DELETE构建器
func NewDeleteBuilder(table string) *DeleteBuilder {
	return &DeleteBuilder{
		table:      table,
		conditions: make([]string, 0),
		args:       make([]interface{}, 0),
		orderBy:    make([]string, 0),
	}
}

// Where 设置WHERE条件
func (d *DeleteBuilder) Where(condition string, args ...interface{}) *DeleteBuilder {
	d.conditions = append(d.conditions, condition)
	d.args = append(d.args, args...)
	return d
}

// OrderBy 设置ORDER BY
func (d *DeleteBuilder) OrderBy(cols ...string) *DeleteBuilder {
	d.orderBy = append(d.orderBy, cols...)
	return d
}

// Limit 设置LIMIT
func (d *DeleteBuilder) Limit(n int) *DeleteBuilder {
	d.limitVal = n
	return d
}

// Build 构建DELETE语句
func (d *DeleteBuilder) Build() string {
	var query strings.Builder
	
	query.WriteString("DELETE FROM " + d.table)
	
	// WHERE子句
	if len(d.conditions) > 0 {
		query.WriteString(" WHERE " + strings.Join(d.conditions, " AND "))
	}
	
	// ORDER BY子句
	if len(d.orderBy) > 0 {
		query.WriteString(" ORDER BY " + strings.Join(d.orderBy, ", "))
	}
	
	// LIMIT子句
	if d.limitVal > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", d.limitVal))
	}
	
	return query.String()
}

// Execute 执行DELETE
func (d *DeleteBuilder) Execute(ctx context.Context, db *sql.DB) (sql.Result, error) {
	return db.ExecContext(ctx, d.Build(), d.args...)
}

// Args 返回参数列表
func (d *DeleteBuilder) Args() []interface{} {
	return d.args
}