package model

import "time"

type Person struct {
	ID          int64     `db:"id" json:"id"`
	FirstName   string    `db:"first_name" json:"first_name"`
	LastName    string    `db:"last_name" json:"last_name"`
	Patronymic  *string   `db:"patronymic" json:"patronymic,omitempty"`
	Age         int32     `db:"age" json:"age"`
	Gender      string    `db:"gender" json:"gender"`
	Nationality string    `db:"nationality" json:"nationality"`
	Emails      []string  `db:"emails" json:"emails"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type CreatePersonDTO struct {
	FirstName  string
	LastName   string
	Patronymic string
	Emails     []string
}

type UpdatePersonDTO struct {
	ID          int64
	FirstName   *string
	LastName    *string
	Patronymic  *string
	Age         *int32
	Gender      *string
	Nationality *string
	Emails      []string
}
