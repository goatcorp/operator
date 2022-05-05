package db

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx"
)

func ApplyMigrations(conn *pgx.Conn, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Find and order all of the migrations
	migrationsPattern := regexp.MustCompile(`(?P<n>\d{3})-.*\.sql`)
	migrationsMax := uint64(0)
	migrations := make([]string, 0) // Indexed array of migrations
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		nameLower := strings.ToLower(file.Name())
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

		migrations[migrationNumber] = path.Join(dir, file.Name())
	}

	for _, migration := range migrations {
		log.Printf("SQL: Applying migration %s\n", migration)

		data, err := os.Open(migration)
		if err != nil {
			return err
		}

		statement, err := ioutil.ReadAll(data)
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
