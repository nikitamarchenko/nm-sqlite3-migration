# nm-sqlite3-migration
Super simple tool for managing sqlite3 migrations

> [!CAUTION]
> Backup your database before doing migrations.

## Commands

`up` for up

`down` for down

`show` to check what is going on

## Filename format

`DDD_name.sql` 

Starts from 3 digits with leading zeros, count from 1. No gaps in versions allowed. 


## Migration file sample
```
-- UP
create table test(PersonID int);
create table test2(PersonID int);
-- DOWN
drop table test;
drop table test2;
```

## Versioning

Version of migration lives in `user_version` of `PRAGMA` so you always can check it with `PRAGMA user_version;` and set it with `PRAGMA user_version = X;`.

```
> sqlite3 test.db
SQLite version 3.42.0 2023-05-16 12:36:15
Enter ".help" for usage hints.
sqlite> PRAGMA user_version;
1
sqlite> PRAGMA user_version = 42;
sqlite> PRAGMA user_version;
42
sqlite> 
```