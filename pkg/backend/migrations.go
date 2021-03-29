package backend

import (
	"fmt"
	"time"

	"gopkg.in/gorp.v2"
)

var migrations []string{
    // Version 1: Add migration table.
    `
    CREATE TABLE migrations (
        id       integer not null primary key autoincrement,
        version  integer,
        time     integer
    )
    `,

    // Version 2: Add 'have' field to ingredients table.
    `
    ALTER TABLE ingredients ADD COLUMN have INTEGER
    `,

    // Version 3: Add 'admin' field to users table.
    `
    ALTER TABLE users ADD COLUMN admin BOOLEAN
    `,

    // Version 4: Added 'families' table.
    `
    CREATE TABLE families (
        id      integer not null primary key autoincrement,
        user_id integer,
        name    text
    )
    `,

    // Version 5: Added 'familymembers' table.
    `
    CREATE TABLE familymembers (
        id        integer not null primary key autoincrement,
        family_id integer,
        user_id   integer,
        can_edit  boolean
    )
    `,

    // Version 6: Populate the families table.
    `
    INSERT INTO families (id, user_id, name)
    SELECT id, id, username FROM users
    `,

    // Version 7: Populate the familymembers table.
    `
    INSERT INTO familymembers (family_id, user_id, can_edit)
    SELECT id, id, 1 FROM users
    `,

    // Version 8: Add 'default_family_id' field to user.
    `
    ALTER TABLE users ADD COLUMN default_family_id INTEGER
    `,

    // Version 9: Populate the default_family_id field.
    `
    UPDATE users
    SET default_family_id = (
        SELECT id FROM families WHERE families.user_id = users.id
    )
    `,

    // Version 10: Add 'week_start_day' field to user.
    `
    ALTER TABLE users ADD COLUMN week_start_day INTEGER
    `,

    // Version 11: Populate the week_start_day field.
    `
    UPDATE users SET week_start_day = 0 WHERE week_start_day IS NULL
    `,

    // Version 12: Add 'created_on' field.
    `
    ALTER TABLE families ADD COLUMN created_on TEXT
    `,

    // Version 13: Populate 'created_on' field.
    `
    UPDATE families SET created_on = (
        SELECT min(date) FROM assignments WHERE assignments.owner_id = families.id
    ) WHERE created_on IS NULL
    `,

    // Version 14: Add 'account_status' field.
    `
    ALTER TABLE families ADD COLUMN account_status TEXT
    `,

    // Version 15: Populate 'account_status' field.
    `
    UPDATE families SET account_status = "trial" WHERE account_status IS NULL
    `,

    // Version 16: Add 'status_expires_on' field.
    `
    ALTER TABLE families ADD COLUMN status_expires_on TEXT
    `,

    // Version 17: Populate 'status_expires_on' field.
    `
    UPDATE families SET status_expires_on = "2018-11-20" WHERE status_expires_on IS NULL
    `,

    // Version 18: Add 'import_id' field to recipes.
    `
    ALTER TABLE recipes ADD COLUMN import_id INTEGER
    `,

    // Version 19: Populate the 'import_id' field.
    `
    UPDATE recipes SET import_id = 0 WHERE import_id IS NULL
    `,

    // Version 20: Add 'name' field to users.
    `
    ALTER TABLE users ADD COLUMN name TEXT
    `,

    // Version 21: Populate the 'name' field.
    `
    UPDATE users SET name = username
    `,

    // Version 22: Reassign the 'username' field.
    `
    UPDATE users SET username = email
    `,

    // Version 23: Add the 'email_verified' field to users.
    `
    ALTER TABLE users ADD COLUMN email_verified BOOLEAN
    `,

    // Version 24: Add the 'email_token' field to users.
    `
    ALTER TABLE users ADD column email_token TEXT
    `,

    // Version 25: Populate the 'email_token' field.
    `
    UPDATE users SET email_token = "aeWieheuz3eidohpuataishool0Op8sh" WHERE email_token IS NULL
    `,
}

func getDatabaseVersion(db *gorp.DbMap) int64 {
	migrations := []Migration{}

	_, err := db.Select(&migrations, "SELECT * FROM migrations ORDER BY version DESC LIMIT 1")
	if err != nil || len(migrations) == 0 {
		return 0
	} else {
		return migrations[0].Version
	}
}

func setDatabaseVersion(db *gorp.DbMap, version int64) error {
	migration := Migration{
		Version: version,
		Time: time.Now().Unix(),
	}

	err := db.Insert(&migration)
	return err
}

func CheckDatabaseVersion(db *gorp.DbMap) {
	current := getDatabaseVersion(db)
	if current == 0 {
		setDatabaseVersion(db, ExpectDatabaseVersion)
	} else if current != ExpectDatabaseVersion {
		message := fmt.Sprintf("Database version mismatch (current = %d, expected = %d)\n", current, ExpectDatabaseVersion)
		panic(message)
	}
}

func MigrateDatabase(db *gorp.DbMap) {
    version := getDatabaseVersion(db)
    for version < len(migrations) {
        fmt.Println("Migrate to version: %d", version+1)
        fmt.Println(migrations[version])

        err := db.Execute(migrations[version])
        if err != nil {
            panic(err.Error())
        }

        version++
        err = setDatabaseVersion(db, version)
        if err != nil {
            panic(err.Error())
        }
    }
}
