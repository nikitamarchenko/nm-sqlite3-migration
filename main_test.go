package main

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func Equal[T comparable](t *testing.T, text string, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Errorf("%s got: %v; want: %v", text, actual, expected)
	}
}

func TestDB_setDbVersion(t *testing.T) {
	db, err := openDB("")
	if err != nil {
		t.Fatalf("DB.openDB() err %v", err)
	}
	version, err := db.getDbVersion()
	if err != nil {
		t.Fatalf("DB.getDbVersion() err %v", err)
	}
	Equal(t, "DB.getDbVersion()", version, 0)

	err = db.setDbVersion(42)
	if err != nil {
		t.Fatalf("DB.setDbVersion() err %v", err)
	}
	version, err = db.getDbVersion()
	if err != nil {
		t.Fatalf("DB.getDbVersion() err %v", err)
	}
	Equal(t, "DB.getDbVersion()", version, 42)
}

func Test_getFileVersion(t *testing.T) {
	type args struct {
		f string
	}
	tests := []struct {
		name    string
		args    args
		want    uint
		wantErr bool
	}{
		{"000", args{"000"}, 0, true},
		{"-005", args{"-005"}, 0, true},
		{"042", args{"042"}, 42, false},
		{"abc", args{"abc"}, 0, true},
		{"migrations/001_test.sq", args{"migrations/001_test.sql"}, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFileVersion(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFileVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getFileVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFiles_validate(t *testing.T) {
	type args struct {
		version uint
	}
	tests := []struct {
		name    string
		f       *Files
		args    args
		wantErr bool
	}{
		{"valid files", &Files{"001_test.sql", "002_test.sql", "003_test.sql"}, args{0}, false},
		{"invalid files sequence", &Files{"002_test.sql", "003_test.sql"}, args{0}, true},
		{"invalid files sequence", &Files{"001_test.sql", "003_test.sql"}, args{0}, true},
		{"invalid file name sequence", &Files{"aaa_test.sql", "002_test.sql"}, args{0}, true},
		{"db version on latest migration", &Files{"001_test.sql", "002_test.sql", "003_test.sql"}, args{3}, false},
		{"db version bigger that migrations count", &Files{"001_test.sql", "002_test.sql", "003_test.sql"}, args{4}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.validate(tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("Files.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_parseMigration(t *testing.T) {

	test01 := `
	-- UP
	create table test;
	-- DOWN
	drop table test;`

	test02 := `
	-- DOWN
	drop table test;
	-- UP
	create table test;`

	test03 := `
	-- UP
	create table test;
	create table test2;
	-- DOWN
	drop table test;
	drop table test2;`

	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{"basic", args{[]byte(test01)}, "create table test;", "drop table test;", false},
		{"basic", args{[]byte(test02)}, "create table test;", "drop table test;", false},
		{"empty", args{[]byte("")}, "", "", false},
		{"multiline", args{[]byte(test03)},
			"create table test;\ncreate table test2;",
			"drop table test;\ndrop table test2;", false},
		{"multiline", args{nil}, "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseMigration(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMigration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMigration() got = %q, want %q", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseMigration() got1 = %q, want %q", got1, tt.want1)
			}
		})
	}
}
