package data

import "database/sql"

type Models struct {
	MovieModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		MovieModel{db: db},
	}
}
