package service

import (
	"context"
	"fmt"

	"person-service/internal/model"
	"person-service/internal/repository"

	"go.uber.org/zap"
)

type PersonService struct {
	repo   repository.Repository
	enrich Enrich
	logger *zap.Logger
}

func NewPersonService(repo repository.Repository, enrich Enrich, logger *zap.Logger) *PersonService {
	return &PersonService{
		repo:   repo,
		enrich: enrich,
		logger: logger,
	}
}

func (s *PersonService) Create(ctx context.Context, dto *model.CreatePersonDTO) (*model.Person, error) {
	s.logger.Debug("creating person", zap.String("name", dto.FirstName+" "+dto.LastName))

	enriched, err := s.enrich.EnrichPerson(ctx, dto.FirstName, dto.LastName)
	if err != nil {
		return nil, fmt.Errorf("enrichment error: %w", err)
	}

	person := &model.Person{
		FirstName:   dto.FirstName,
		LastName:    dto.LastName,
		Patronymic:  &dto.Patronymic,
		Age:         enriched.Age,
		Gender:      enriched.Gender,
		Nationality: enriched.Nationality,
		Emails:      dto.Emails,
	}

	if err := s.repo.Create(ctx, person); err != nil {
		return nil, fmt.Errorf("create database error: %w", err)
	}

	s.logger.Debug("person created successfully",
		zap.Int64("id", person.ID),
		zap.String("name", person.FirstName+" "+person.LastName))

	return person, nil
}

func (s *PersonService) GetByID(ctx context.Context, id int64) (*model.Person, error) {
	s.logger.Debug("getting person", zap.Int64("id", id))

	person, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get database error: %w", err)
	}

	return person, nil
}

func (s *PersonService) List(ctx context.Context, page, pageSize int32) ([]*model.Person, int32, error) {
	s.logger.Debug("listing persons", zap.Int32("page", page), zap.Int32("page_size", pageSize))

	persons, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list database error: %w", err)
	}

	s.logger.Debug("listing persons successfully")

	return persons, total, nil
}

func (s *PersonService) Update(ctx context.Context, dto *model.UpdatePersonDTO) error {
	s.logger.Debug("updating person", zap.Int64("id", dto.ID))

	existing, err := s.repo.GetByID(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("get for update database error: %w", err)
	}
	if err := s.repo.Update(ctx, existing, dto); err != nil {
		return fmt.Errorf("update database error: %w", err)
	}

	s.logger.Debug("person updated successfully", zap.Int64("id", existing.ID))

	return nil
}

func (s *PersonService) Delete(ctx context.Context, id int64) error {
	s.logger.Debug("deleting person", zap.Int64("id", id))

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete database error: %w", err)
	}

	s.logger.Debug("person deleted successfully", zap.Int64("id", id))

	return nil
}
