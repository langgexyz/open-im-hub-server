package store

import "database/sql"

// UserStore provides access to the users table.
// Full implementation is added in Task 4.
type UserStore struct{ db *sql.DB }
