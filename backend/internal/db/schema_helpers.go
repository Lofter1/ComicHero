package db

import "github.com/jmoiron/sqlx"

func tableExists(db *sqlx.DB, name string) (bool, error) {
	var count int
	if err := db.Get(&count, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = ?
	`, name); err != nil {
		return false, err
	}
	return count > 0, nil
}

func columnExists(db *sqlx.DB, table, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
