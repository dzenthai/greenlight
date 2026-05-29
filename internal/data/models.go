package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
	ErrDuplicateEmail = errors.New("duplicate email")
)

type Models struct {
	MovieModel
	UserModel
	TokenModel
	PermissionModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		MovieModel{db: db},
		UserModel{db: db},
		TokenModel{db: db},
		PermissionModel{db: db},
	}
}
