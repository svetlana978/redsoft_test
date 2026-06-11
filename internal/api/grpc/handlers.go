package grpc

import (
	"context"
	"errors"
	"fmt"

	"person-service/internal/model"
	"person-service/internal/service"
	pb "person-service/proto"

	"go.uber.org/zap"
)

type Handlers struct {
	pb.UnimplementedPersonServiceServer
	personSvc *service.PersonService
	logger    *zap.Logger
}

func NewHandlers(personSvc *service.PersonService, logger *zap.Logger) *Handlers {
	return &Handlers{
		personSvc: personSvc,
		logger:    logger,
	}
}

func (h *Handlers) CreatePerson(ctx context.Context, req *pb.CreatePersonRequest) (*pb.PersonResponse, error) {
	if req.FirstName == "" {
		h.logger.Error("need firstName to create person")
		return nil, errors.New("need firstName to create person")
	}
	if req.FirstName == "" {
		h.logger.Error("need lastname to create person")
		return nil, errors.New("need lastname to create person")
	}
	if len(req.Emails) < 1 {
		h.logger.Error("need at least one email to create person")
		return nil, errors.New("need at least one email to create person")
	}

	dto := &model.CreatePersonDTO{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Emails:    req.Emails,
	}

	if req.Patronymic != nil {
		dto.Patronymic = *req.Patronymic
	}

	person, err := h.personSvc.Create(ctx, dto)
	if err != nil {
		h.logger.Error("failed to create person", zap.Error(err))
		return nil, fmt.Errorf("failed to create person: %w", err)
	}

	return &pb.PersonResponse{Person: h.toProto(person)}, nil
}

func (h *Handlers) GetPerson(ctx context.Context, req *pb.GetPersonRequest) (*pb.PersonResponse, error) {
	if req.Id == 0 {
		h.logger.Error("need id to get person")
		return nil, errors.New("need id to get person")
	}
	person, err := h.personSvc.GetByID(ctx, req.Id)
	if err != nil {
		h.logger.Error("failed to get person", zap.Error(err))
		return nil, fmt.Errorf("person not found: %w", err)
	}

	return &pb.PersonResponse{Person: h.toProto(person)}, nil
}

func (h *Handlers) ListPersons(ctx context.Context, req *pb.ListPersonsRequest) (*pb.ListPersonsResponse, error) {
	if req.Page < 1 {
		return nil, fmt.Errorf("wrong page number %d", req.Page)
	}
	if req.PageSize < 1 || req.PageSize > 200 {
		return nil, fmt.Errorf("wrong page size: %d", req.Page)
	}

	persons, total, err := h.personSvc.List(ctx, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error("failed to get persons list", zap.Error(err))
		return nil, fmt.Errorf("failed to list persons: %w", err)
	}

	protoPersons := make([]*pb.Person, len(persons))
	for i, p := range persons {
		protoPersons[i] = h.toProto(p)
	}

	return &pb.ListPersonsResponse{
		Persons:    protoPersons,
		TotalCount: total,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

func (h *Handlers) UpdatePerson(ctx context.Context, req *pb.UpdatePersonRequest) (*pb.SuccessResponse, error) {
	if req.Id == 0 {
		h.logger.Error("need id to update person")
		return nil, errors.New("need id to update person")
	}

	dto := &model.UpdatePersonDTO{
		ID: req.Id,
	}

	cnt := 0
	if req.FirstName != nil {
		dto.FirstName = req.FirstName
		cnt++
	}
	if req.LastName != nil {
		dto.LastName = req.LastName
		cnt++
	}
	if req.Patronymic != nil {
		dto.Patronymic = req.Patronymic
		cnt++
	}
	if req.Age != nil {
		dto.Age = req.Age
		cnt++
	}
	if req.Gender != nil {
		dto.Gender = req.Gender
		cnt++
	}
	if req.Nationality != nil {
		dto.Nationality = req.Nationality
		cnt++
	}
	if req.Emails != nil {
		dto.Emails = req.Emails
		cnt++
	}

	if cnt == 0 {
		h.logger.Error("need at least one field for update person")
		return nil, errors.New("need at least one field for update person")
	}
	err := h.personSvc.Update(ctx, dto)
	if err != nil {
		h.logger.Error("failed to update person", zap.Error(err))
		return nil, fmt.Errorf("failed to update person: %w", err)
	}

	return &pb.SuccessResponse{Success: true}, nil
}

func (h *Handlers) DeletePerson(ctx context.Context, req *pb.DeletePersonRequest) (*pb.SuccessResponse, error) {
	if req.Id == 0 {
		h.logger.Error("need id to delete person")
		return nil, errors.New("need id to delete person")
	}
	if err := h.personSvc.Delete(ctx, req.Id); err != nil {
		h.logger.Error("person not found", zap.Error(err))
		return nil, fmt.Errorf("person not found: %w", err)
	}

	return &pb.SuccessResponse{Success: true}, nil
}

func (h *Handlers) toProto(p *model.Person) *pb.Person {
	result := &pb.Person{
		Id:          p.ID,
		FirstName:   p.FirstName,
		LastName:    p.LastName,
		Age:         int32(p.Age),
		Gender:      p.Gender,
		Nationality: p.Nationality,
		Emails:      p.Emails,
	}

	if p.Patronymic != nil {
		result.Patronymic = p.Patronymic
	}

	return result
}
