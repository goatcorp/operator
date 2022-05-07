package db

import (
	"io/fs"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx"
)

var migrationsPattern = regexp.MustCompile(`(?i)(?P<n>\d{3})-.*\.sql`)

func ApplyMigrations(conn *pgx.Conn, fileSystem fs.FS) error {
	files, err := fs.Glob(fileSystem, "*")
	if err != nil {
		return err
	}

	// Find and order all of the migrations
	migrationsMax := uint64(0)
	migrations := make([]string, 0) // Indexed array of migrations
	for _, file := range files {
		nameLower := strings.ToLower(file)
		matches := migrationsPattern.FindStringSubmatch(nameLower)
		if len(matches) == 0 {
			continue
		}

		// 10 bits = maximum migration number of 1024
		// 3 digits = maximum migration number of 999
		// 10 bits is the minimum needed to encode every migration, currently
		matchIdx := migrationsPattern.SubexpIndex("n")
		migrationNumber, err := strconv.ParseUint(matches[matchIdx], 10, 10)
		if err != nil {
			return err
		}

		if migrationNumber >= migrationsMax {
			migrationsMax = migrationNumber
			migrationsNew := make([]string, migrationsMax+1)
			copy(migrationsNew[:], migrations[:])
			migrations = migrationsNew
		}

		migrations[migrationNumber] = file
	}

	for _, migration := range migrations {
		log.Printf("SQL: Applying migration %s\n", migration)

		statement, err := fs.ReadFile(fileSystem, migration)
		if err != nil {
			return err
		}

		tag, err := conn.Exec(string(statement))
		if err != nil {
			return err
		}

		log.Printf("SQL: %d rows affected.", tag.RowsAffected())
	}

	return nil
}
