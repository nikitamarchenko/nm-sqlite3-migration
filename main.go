package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func printUsage() {
	fmt.Printf("  Valid commands: up down show schema\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	dir := flag.String("dir", ".", "path to migration folder")
	dsn := flag.String("dsn", "", "dsn")

	flag.Parse()

	if len(flag.Args()) != 1 {
		printUsage()
	}

	if len(*dsn) == 0 {
		fmt.Print("please provide dsn\n")
		os.Exit(1)
	}

	db, err := openDB(*dsn)

	if err != nil {
		fmt.Printf("db open error %s\n", err)
		os.Exit(1)
	}

	// check *sql files
	var files Files
	files, err = filepath.Glob(filepath.Join(*dir, "[0123456789][0123456789][0123456789]_*.sql"))
	if err != nil {
		fmt.Printf("error %s", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("no sql files in %q, match 123_filename.sql\n", *dir)
		os.Exit(1)
	}

	version, err := db.getDbVersion()
	if err != nil {
		fmt.Printf("Db version error: %d\n", version)
		os.Exit(1)
	}

	if err := files.validate(version); err != nil {
		fmt.Printf("sql files validation failed: %s\n", err)
		os.Exit(1)
	}

	_ = db

	command := flag.Args()[0]

	switch command {
	case "up":
		up(db, files)
	case "down":
		down(db, files)
	case "show":
		show(db, files)
	case "schema":
		fmt.Printf("sqlite3 %s '.schema'\n", *dsn)
	default:
		printUsage()
	}

	_, _ = dir, dsn
}

func show(db *DB, files Files) {
	version, err := db.getDbVersion()
	if err != nil {
		fmt.Printf("Db version error: %d\n", version)
		return
	}
	fmt.Printf("DB version %d\n", version)
	for i, file := range files {
		if uint(i) < version {
			fmt.Printf("[*] %s\n", file)
		} else {
			fmt.Printf("[ ] %s\n", file)
		}
	}
}

func up(db *DB, files Files) {
	version, err := db.getDbVersion()
	newVersion := version + 1
	if err != nil {
		fmt.Printf("Db version error: %d\n", version)
		return
	}
	l := uint(len(files))
	if newVersion > l {
		fmt.Printf("We on latest migration %d\n", version)
		return
	}
	file, err := files.getFile(newVersion)
	if err != nil {
		fmt.Printf("error: %s", err)
		return
	}
	up, _, err := parseMigration(file)
	if err != nil {
		fmt.Printf("error: %s", err)
		return
	}
	fmt.Printf("Migration %d:\n", newVersion)
	fmt.Println(up)
	_, err = db.Exec(up)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	db.setDbVersion(newVersion)
}

func down(db *DB, files Files) {
	version, err := db.getDbVersion()
	var newVersion uint
	if version > 0 {
		newVersion = version - 1
	}
	if err != nil {
		fmt.Printf("Db version error: %s\n", err)
		return
	}
	if version == 0 {
		fmt.Printf("We on lowest migration %d\n", version)
		return
	}
	file, err := files.getFile(version)
	if err != nil {
		fmt.Printf("error: %s", err)
		return
	}
	_, down, err := parseMigration(file)
	if err != nil {
		fmt.Printf("error: %s", err)
		return
	}
	fmt.Printf("Migration %d:\n", newVersion)
	fmt.Println(down)
	_, err = db.Exec(down)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	db.setDbVersion(newVersion)
}

type Files []string

func (f *Files) validate(version uint) error {
	for i, file := range *f {
		v, err := getFileVersion(file)
		if err != nil {
			return fmt.Errorf("can't parse version from filename %s", file)
		}
		if uint(i+1) != v {
			return fmt.Errorf("invalid file %q version want %d got %d", file, i+1, v)
		}
	}
	l := uint(len(*f))
	if version > l {
		return fmt.Errorf("missing files we need at least %d have %d", version, l)
	}

	return nil
}

func (f *Files) getFile(version uint) ([]byte, error) {
	return os.ReadFile((*f)[version-1])
}

func parseMigration(b []byte) (string, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(b))
	up := []string{}
	down := []string{}

	const (
		Skip = iota
		Up
		Down
	)
	state := Skip
	for scanner.Scan() {
		t := scanner.Text()
		t = strings.TrimSpace(t)
		if strings.HasPrefix(t, "--") {
			tt := t[2:]
			tt = strings.TrimSpace(tt)
			switch tt {
			case "UP":
				state = Up
			case "DOWN":
				state = Down
			}
			continue
		}
		switch state {
		case Up:
			up = append(up, t)
		case Down:
			down = append(down, t)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	return strings.Join(up, "\n"), strings.Join(down, "\n"), nil
}

func getFileVersion(f string) (uint, error) {
	f = filepath.Base(f)
	i, err := strconv.ParseUint(f[0:3], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("can't get file(%q) version %s", f, err)
	}
	if i == 0 {
		return 0, fmt.Errorf("file version is zero must be greater than zero")
	}

	return uint(i), nil
}

type DB struct {
	*sql.DB
}

func openDB(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db}, nil
}

func (db *DB) getDbVersion() (uint, error) {
	stmt := `PRAGMA user_version`
	row := db.QueryRow(stmt)
	var version uint
	err := row.Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		} else {
			return 0, err
		}
	}
	return version, nil
}

func (db *DB) setDbVersion(version uint) (err error) {
	_, err = db.Exec(fmt.Sprintf("PRAGMA user_version = %d", version))
	return
}
