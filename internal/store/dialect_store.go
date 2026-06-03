package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var mysqlIndexIfNotExistsPattern = regexp.MustCompile(`(?i)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+([a-zA-Z0-9_]+)\s+ON\s+([a-zA-Z0-9_]+)\s*\(([^;]+)\)`)
var sqlWindowIdentifierPattern = regexp.MustCompile(`\bwindow\b`)

type databaseDialect string

const (
	dialectSQLite   databaseDialect = "sqlite"
	dialectPostgres databaseDialect = "postgres"
	dialectMySQL    databaseDialect = "mysql"
)

type dialectDB struct {
	inner   *sql.DB
	dialect databaseDialect
}

type dialectTx struct {
	inner   *sql.Tx
	dialect databaseDialect
}

func normalizeDatabaseDialect(driver string) (databaseDialect, string, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", "sqlite", "sqlite3":
		return dialectSQLite, "sqlite", nil
	case "postgres", "postgresql":
		return dialectPostgres, "postgres", nil
	case "mysql":
		return dialectMySQL, "mysql", nil
	default:
		return "", "", fmt.Errorf("unsupported database driver %q", driver)
	}
}

func newDialectDB(db *sql.DB, dialect databaseDialect) *dialectDB {
	return &dialectDB{inner: db, dialect: dialect}
}

func (db *dialectDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if db.dialect == dialectMySQL {
		return db.execMySQLContext(ctx, query, args...)
	}
	return db.inner.ExecContext(ctx, rewriteSQLPlaceholders(db.dialect, query), args...)
}

func (db *dialectDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.inner.QueryContext(ctx, rewriteSQLPlaceholders(db.dialect, query), args...)
}

func (db *dialectDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.inner.QueryRowContext(ctx, rewriteSQLPlaceholders(db.dialect, query), args...)
}

func (db *dialectDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*dialectTx, error) {
	tx, err := db.inner.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &dialectTx{inner: tx, dialect: db.dialect}, nil
}

func (db *dialectDB) Close() error {
	return db.inner.Close()
}

func (tx *dialectTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx.dialect == dialectMySQL {
		return tx.execMySQLContext(ctx, query, args...)
	}
	return tx.inner.ExecContext(ctx, rewriteSQLPlaceholders(tx.dialect, query), args...)
}

func (tx *dialectTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.inner.QueryContext(ctx, rewriteSQLPlaceholders(tx.dialect, query), args...)
}

func (tx *dialectTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.inner.QueryRowContext(ctx, rewriteSQLPlaceholders(tx.dialect, query), args...)
}

func (tx *dialectTx) Commit() error {
	return tx.inner.Commit()
}

func (tx *dialectTx) Rollback() error {
	return tx.inner.Rollback()
}

func (db *dialectDB) execMySQLContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	query = rewriteMySQLSQL(query)
	indexes, cleanedQuery := extractMySQLIndexes(query)
	var result sql.Result
	var err error
	if strings.TrimSpace(cleanedQuery) != "" {
		result, err = db.inner.ExecContext(ctx, cleanedQuery, args...)
		if err != nil {
			return result, err
		}
	}
	for _, index := range indexes {
		if err := db.createMySQLIndexIfMissing(ctx, index); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (tx *dialectTx) execMySQLContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	query = rewriteMySQLSQL(query)
	indexes, cleanedQuery := extractMySQLIndexes(query)
	var result sql.Result
	var err error
	if strings.TrimSpace(cleanedQuery) != "" {
		result, err = tx.inner.ExecContext(ctx, cleanedQuery, args...)
		if err != nil {
			return result, err
		}
	}
	for _, index := range indexes {
		if err := createMySQLIndexIfMissing(ctx, tx.inner, index); err != nil {
			return result, err
		}
	}
	return result, nil
}

type mysqlIndexDefinition struct {
	name       string
	table      string
	definition string
}

func extractMySQLIndexes(query string) ([]mysqlIndexDefinition, string) {
	statements := strings.Split(query, ";")
	indexes := []mysqlIndexDefinition{}
	kept := []string{}
	for _, statement := range statements {
		trimmed := strings.TrimSpace(statement)
		if trimmed == "" {
			continue
		}
		matches := regexp.MustCompile(`(?i)^CREATE\s+INDEX\s+([a-zA-Z0-9_]+)\s+ON\s+([a-zA-Z0-9_]+)\s*\((.+)\)$`).FindStringSubmatch(trimmed)
		if len(matches) == 4 {
			indexes = append(indexes, mysqlIndexDefinition{name: matches[1], table: matches[2], definition: trimmed})
			continue
		}
		kept = append(kept, trimmed)
	}
	cleaned := strings.Join(kept, ";\n")
	if cleaned != "" {
		cleaned += ";"
	}
	return indexes, cleaned
}

func (db *dialectDB) createMySQLIndexIfMissing(ctx context.Context, index mysqlIndexDefinition) error {
	return createMySQLIndexIfMissing(ctx, db.inner, index)
}

func createMySQLIndexIfMissing(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, index mysqlIndexDefinition) error {
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
	`, index.table, index.name).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, index.definition)
	return err
}

func rewriteSQLPlaceholders(dialect databaseDialect, query string) string {
	if dialect == dialectMySQL {
		return rewriteMySQLSQL(query)
	}
	if dialect != dialectPostgres {
		return query
	}
	query = quoteReservedSQLIdentifiers(dialect, query)
	var builder strings.Builder
	builder.Grow(len(query) + 8)
	index := 1
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false
	for i := 0; i < len(query); i++ {
		ch := query[i]
		next := byte(0)
		if i+1 < len(query) {
			next = query[i+1]
		}
		if inLineComment {
			builder.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			builder.WriteByte(ch)
			if ch == '*' && next == '/' {
				builder.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		}
		if inSingleQuote {
			builder.WriteByte(ch)
			if ch == '\'' {
				if next == '\'' {
					builder.WriteByte(next)
					i++
				} else {
					inSingleQuote = false
				}
			}
			continue
		}
		if inDoubleQuote {
			builder.WriteByte(ch)
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}
		if ch == '-' && next == '-' {
			builder.WriteByte(ch)
			builder.WriteByte(next)
			i++
			inLineComment = true
			continue
		}
		if ch == '/' && next == '*' {
			builder.WriteByte(ch)
			builder.WriteByte(next)
			i++
			inBlockComment = true
			continue
		}
		if ch == '\'' {
			builder.WriteByte(ch)
			inSingleQuote = true
			continue
		}
		if ch == '"' {
			builder.WriteByte(ch)
			inDoubleQuote = true
			continue
		}
		if ch == '?' {
			builder.WriteString(fmt.Sprintf("$%d", index))
			index++
			continue
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func rewriteMySQLSQL(query string) string {
	query = rewriteMySQLDDL(query)
	query = quoteReservedSQLIdentifiers(dialectMySQL, query)
	query = rewriteMySQLUpserts(query)
	return mysqlIndexIfNotExistsPattern.ReplaceAllString(query, "CREATE INDEX $1 ON $2($3)")
}

func rewriteMySQLDDL(query string) string {
	upper := strings.ToUpper(query)
	if !strings.Contains(upper, "CREATE TABLE") && !(strings.Contains(upper, "ALTER TABLE") && strings.Contains(upper, "ADD COLUMN")) {
		return query
	}
	return mysqlColumnTypesSQL(query)
}

func mysqlColumnTypesSQL(input string) string {
	replacer := strings.NewReplacer(
		" id TEXT PRIMARY KEY", " id VARCHAR(191) PRIMARY KEY",
		"\tid TEXT PRIMARY KEY", "\tid VARCHAR(191) PRIMARY KEY",
		"code TEXT NOT NULL UNIQUE", "code VARCHAR(128) NOT NULL UNIQUE",
		"username TEXT NOT NULL UNIQUE", "username VARCHAR(128) NOT NULL UNIQUE",
		"token_hash TEXT NOT NULL UNIQUE", "token_hash VARCHAR(128) NOT NULL UNIQUE",
		"asset_id TEXT NOT NULL UNIQUE", "asset_id VARCHAR(191) NOT NULL UNIQUE",
		"platform_id TEXT NOT NULL DEFAULT ''", "platform_id VARCHAR(191) NOT NULL DEFAULT ''",
		"platform_id TEXT NOT NULL", "platform_id VARCHAR(191) NOT NULL",
		"cloud_account_id TEXT NOT NULL DEFAULT ''", "cloud_account_id VARCHAR(191) NOT NULL DEFAULT ''",
		"cloud_account_id TEXT NOT NULL", "cloud_account_id VARCHAR(191) NOT NULL",
		"inspection_id TEXT NOT NULL", "inspection_id VARCHAR(191) NOT NULL",
		"asset_id TEXT NOT NULL", "asset_id VARCHAR(191) NOT NULL",
		"user_id TEXT NOT NULL", "user_id VARCHAR(191) NOT NULL",
		"flow_id TEXT NOT NULL DEFAULT ''", "flow_id VARCHAR(191) NOT NULL DEFAULT ''",
		"flow_id TEXT NOT NULL", "flow_id VARCHAR(191) NOT NULL",
		"current_step_id TEXT NOT NULL DEFAULT ''", "current_step_id VARCHAR(191) NOT NULL DEFAULT ''",
		"approval_id TEXT NOT NULL", "approval_id VARCHAR(191) NOT NULL",
		"step_id TEXT NOT NULL DEFAULT ''", "step_id VARCHAR(191) NOT NULL DEFAULT ''",
		"access_grant_id TEXT NOT NULL", "access_grant_id VARCHAR(191) NOT NULL",
		"owner_id TEXT NOT NULL", "owner_id VARCHAR(191) NOT NULL",
		"owner_type TEXT NOT NULL", "owner_type VARCHAR(64) NOT NULL",
		"kind TEXT NOT NULL", "kind VARCHAR(64) NOT NULL",
		"key_name TEXT NOT NULL DEFAULT ''", "key_name VARCHAR(128) NOT NULL DEFAULT ''",
		"role TEXT NOT NULL", "role VARCHAR(64) NOT NULL",
		"scope TEXT NOT NULL", "scope VARCHAR(64) NOT NULL",
		"action TEXT NOT NULL", "action VARCHAR(64) NOT NULL",
		"target_type TEXT NOT NULL", "target_type VARCHAR(64) NOT NULL",
		"target_id TEXT NOT NULL DEFAULT ''", "target_id VARCHAR(191) NOT NULL DEFAULT ''",
		"environment TEXT NOT NULL DEFAULT '*'", "environment VARCHAR(64) NOT NULL DEFAULT '*'",
		"environment TEXT NOT NULL DEFAULT 'prod'", "environment VARCHAR(64) NOT NULL DEFAULT 'prod'",
		"environment TEXT NOT NULL DEFAULT ''", "environment VARCHAR(64) NOT NULL DEFAULT ''",
		"environment TEXT NOT NULL", "environment VARCHAR(64) NOT NULL",
		"project_code TEXT NOT NULL DEFAULT '*'", "project_code VARCHAR(64) NOT NULL DEFAULT '*'",
		"project_code TEXT NOT NULL DEFAULT 'public'", "project_code VARCHAR(64) NOT NULL DEFAULT 'public'",
		"project_code TEXT NOT NULL DEFAULT ''", "project_code VARCHAR(64) NOT NULL DEFAULT ''",
		"source TEXT NOT NULL DEFAULT ''", "source VARCHAR(128) NOT NULL DEFAULT ''",
		"external_id TEXT NOT NULL DEFAULT ''", "external_id VARCHAR(191) NOT NULL DEFAULT ''",
		"status TEXT NOT NULL DEFAULT 'active'", "status VARCHAR(64) NOT NULL DEFAULT 'active'",
		"status TEXT NOT NULL DEFAULT 'pending'", "status VARCHAR(64) NOT NULL DEFAULT 'pending'",
		"status TEXT NOT NULL", "status VARCHAR(64) NOT NULL",
		"created_at TEXT NOT NULL", "created_at VARCHAR(64) NOT NULL",
		"updated_at TEXT NOT NULL", "updated_at VARCHAR(64) NOT NULL",
		"expires_at TEXT NOT NULL", "expires_at VARCHAR(64) NOT NULL",
		"revoked_at TEXT NOT NULL DEFAULT ''", "revoked_at VARCHAR(64) NOT NULL DEFAULT ''",
		"started_at TEXT NOT NULL", "started_at VARCHAR(64) NOT NULL",
		"finished_at TEXT NOT NULL", "finished_at VARCHAR(64) NOT NULL",
		"checked_at TEXT NOT NULL", "checked_at VARCHAR(64) NOT NULL",
		"last_seen_at TEXT NOT NULL", "last_seen_at VARCHAR(64) NOT NULL",
		"first_seen_at TEXT NOT NULL", "first_seen_at VARCHAR(64) NOT NULL",
		"resolved_at TEXT NOT NULL DEFAULT ''", "resolved_at VARCHAR(64) NOT NULL DEFAULT ''",
		"decided_at TEXT NOT NULL DEFAULT ''", "decided_at VARCHAR(64) NOT NULL DEFAULT ''",
		"ended_at TEXT NOT NULL DEFAULT ''", "ended_at VARCHAR(64) NOT NULL DEFAULT ''",
		"last_viewed_at TEXT NOT NULL DEFAULT ''", "last_viewed_at VARCHAR(64) NOT NULL DEFAULT ''",
		"last_rotated_at TEXT NOT NULL DEFAULT ''", "last_rotated_at VARCHAR(64) NOT NULL DEFAULT ''",
		"last_cost_sync_at TEXT NOT NULL DEFAULT ''", "last_cost_sync_at VARCHAR(64) NOT NULL DEFAULT ''",
		"last_sync_at TEXT NOT NULL DEFAULT ''", "last_sync_at VARCHAR(64) NOT NULL DEFAULT ''",
		"last_login_at TEXT NOT NULL DEFAULT ''", "last_login_at VARCHAR(64) NOT NULL DEFAULT ''",
		" TEXT NOT NULL DEFAULT ''", " VARCHAR(255) NOT NULL DEFAULT ''",
		" TEXT NOT NULL DEFAULT 'prod'", " VARCHAR(64) NOT NULL DEFAULT 'prod'",
		" TEXT NOT NULL DEFAULT 'public'", " VARCHAR(64) NOT NULL DEFAULT 'public'",
		" TEXT NOT NULL DEFAULT '*'", " VARCHAR(64) NOT NULL DEFAULT '*'",
		" TEXT NOT NULL DEFAULT '{}'", " JSON NOT NULL DEFAULT (JSON_OBJECT())",
		" TEXT NOT NULL DEFAULT '[]'", " JSON NOT NULL DEFAULT (JSON_ARRAY())",
		" TEXT NOT NULL", " VARCHAR(255) NOT NULL",
		" BLOB NOT NULL", " LONGBLOB NOT NULL",
		"metadata_json VARCHAR(255) NOT NULL DEFAULT (JSON_OBJECT())", "metadata_json JSON NOT NULL DEFAULT (JSON_OBJECT())",
		"warnings_json JSON NOT NULL DEFAULT (JSON_ARRAY())", "warnings_json JSON NOT NULL DEFAULT (JSON_ARRAY())",
		"breakdown_json JSON NOT NULL DEFAULT (JSON_OBJECT())", "breakdown_json JSON NOT NULL DEFAULT (JSON_OBJECT())",
		"specs_json JSON NOT NULL DEFAULT (JSON_OBJECT())", "specs_json JSON NOT NULL DEFAULT (JSON_OBJECT())",
		"encrypted_value VARCHAR(255) NOT NULL DEFAULT ''", "encrypted_value MEDIUMTEXT NOT NULL",
		"data BLOB NOT NULL", "data LONGBLOB NOT NULL",
	)
	return replacer.Replace(input)
}

func quoteReservedSQLIdentifiers(dialect databaseDialect, query string) string {
	switch dialect {
	case dialectPostgres:
		return replaceUnquotedSQLWord(query, "window", `"window"`)
	case dialectMySQL:
		return replaceUnquotedSQLWord(query, "window", "`window`")
	default:
		return query
	}
}

func replaceUnquotedSQLWord(query string, word string, replacement string) string {
	matches := sqlWindowIdentifierPattern.FindAllStringIndex(query, -1)
	if len(matches) == 0 {
		return query
	}
	var builder strings.Builder
	builder.Grow(len(query) + len(matches)*2)
	last := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		if isSQLIdentifierQuoteAt(query, start-1) || isSQLIdentifierQuoteAt(query, end) {
			continue
		}
		builder.WriteString(query[last:start])
		builder.WriteString(replacement)
		last = end
	}
	if last == 0 {
		return query
	}
	builder.WriteString(query[last:])
	return builder.String()
}

func isSQLIdentifierQuoteAt(query string, index int) bool {
	return index >= 0 && index < len(query) && (query[index] == '"' || query[index] == '`')
}

func rewriteMySQLUpserts(query string) string {
	if !strings.Contains(query, "ON CONFLICT(") || !strings.Contains(query, "excluded.") {
		return query
	}
	pattern := regexp.MustCompile(`(?is)ON\s+CONFLICT\s*\([^)]+\)\s+DO\s+UPDATE\s+SET\s+(.+)$`)
	query = pattern.ReplaceAllString(query, "ON DUPLICATE KEY UPDATE $1")
	excludedPattern := regexp.MustCompile(`excluded\.([a-zA-Z0-9_]+)`)
	return excludedPattern.ReplaceAllString(query, "VALUES($1)")
}

func quoteIdentifier(value string) string {
	if value == "" {
		return `""`
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
		}
	}
	return value
}

func (s *DBStore) columnNames(ctx context.Context, table string) (map[string]bool, error) {
	table = strings.TrimSpace(table)
	if table == "" {
		return nil, fmt.Errorf("table is required")
	}
	var rows *sql.Rows
	var err error
	switch s.dialect {
	case dialectPostgres:
		rows, err = s.db.QueryContext(ctx, `
			SELECT column_name
			FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = ?
		`, table)
	case dialectMySQL:
		rows, err = s.db.QueryContext(ctx, `
			SELECT column_name
			FROM information_schema.columns
			WHERE table_schema = database() AND table_name = ?
		`, table)
	default:
		rows, err = s.db.QueryContext(ctx, `PRAGMA table_info(`+quoteIdentifier(table)+`)`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var name string
		if s.dialect == dialectSQLite {
			var cid int
			var columnType string
			var notNull int
			var defaultValue sql.NullString
			var primaryKey int
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				return nil, err
			}
		} else if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}

func (s *DBStore) schemaSQL() string {
	switch s.dialect {
	case dialectPostgres:
		return strings.ReplaceAll(schemaSQL, "data BLOB NOT NULL", "data BYTEA NOT NULL")
	case dialectMySQL:
		return mysqlSchemaSQL(schemaSQL)
	}
	return schemaSQL
}

func (s *DBStore) dnsRecordTypePredicate() string {
	switch s.dialect {
	case dialectPostgres:
		return `(specs_json::jsonb ->> 'type') IN ('A', 'AAAA', 'CNAME')`
	case dialectMySQL:
		return `JSON_UNQUOTE(JSON_EXTRACT(specs_json, '$.type')) IN ('A', 'AAAA', 'CNAME')`
	default:
		return `json_extract(specs_json, '$.type') IN ('A', 'AAAA', 'CNAME')`
	}
}

func mysqlSchemaSQL(input string) string {
	return rewriteMySQLSQL(mysqlColumnTypesSQL(input))
}

func normalizeMySQLDSN(dsn string) string {
	base, rawValues, ok := strings.Cut(dsn, "?")
	values := url.Values{}
	if ok {
		parsed, err := url.ParseQuery(rawValues)
		if err == nil {
			values = parsed
		}
	} else {
		base = dsn
	}
	if values.Get("parseTime") == "" {
		values.Set("parseTime", "true")
	}
	if values.Get("charset") == "" {
		values.Set("charset", "utf8mb4")
	}
	if values.Get("collation") == "" {
		values.Set("collation", "utf8mb4_unicode_ci")
	}
	if values.Get("multiStatements") == "" {
		values.Set("multiStatements", "true")
	}
	return base + "?" + values.Encode()
}
