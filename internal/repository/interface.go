package repository

import (
	"context"
	"person-service/internal/model"
)

type Repository interface {
	Create(ctx context.Context, person *model.Person) error
	GetByID(ctx context.Context, id int64) (*model.Person, error)
	List(ctx context.Context, page, pageSize int32) ([]*model.Person, int32, error)
	Update(ctx context.Context, existing *model.Person, person *model.UpdatePersonDTO) error
	Delete(ctx context.Context, id int64) error
	Close() error
}
