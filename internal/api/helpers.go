package api

import (
	"github.com/wahid/pos-go/internal/database"
)

// databaseOpen opens the SQLite DB at path (creating schema if needed).
var databaseOpen = database.Open