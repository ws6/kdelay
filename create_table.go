package main

import (
	"fmt"

	"github.com/ws6/msi"
)

func createTableQuery(tableName string) string {

	ret := "CREATE TABLE IF NOT EXISTS  `%s` ("
	ret += "`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,"
	ret += "`created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
	ret += "`release_at` DATETIME NOT NULL,"
	ret += "`body` TEXT NOT NULL,"
	ret += "PRIMARY KEY (`id`),"
	ret += "UNIQUE INDEX `id_UNIQUE` (`id` ASC) VISIBLE,"
	ret += "INDEX `idx_release_at` (`release_at` ASC) VISIBLE);"

	return fmt.Sprintf(ret, tableName)
}

func createTable(db *msi.Msi, tableName string) error {
	_, err := db.Map(db.Db, createTableQuery(tableName), nil)
	return err
}
