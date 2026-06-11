package unit

import (
	"context"
	"errors"
	"testing"

	"person-service/internal/model"
	"person-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, person *model.Person) error {
	args := m.Called(ctx, person)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id int64) (*model.Person, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Person), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, page, pageSize int32) ([]*model.Person, int32, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*model.Person), args.Get(1).(int32), args.Error(2)
}

func (m *MockRepository) Update(ctx context.Context, current *model.Person, person *model.UpdatePersonDTO) error {
	args := m.Called(ctx, person)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockEnrichmentClient struct {
	mock.Mock
}

func (e *MockEnrichmentClient) EnrichPerson(ctx context.Context, firstName, lastName string) (*service.EnrichedData, error) {
	args := e.Called(ctx, firstName, lastName)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*service.EnrichedData), args.Error(1)
}

func TestPersonService_Create(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name      string
		input     *model.CreatePersonDTO
		enrichRes *service.EnrichedData
		enrichErr error
		repoErr   error
		wantErr   bool
	}{
		{
			name: "successful creation",
			input: &model.CreatePersonDTO{
				FirstName: "John",
				LastName:  "Doe",
				Emails:    []string{"john@example.com"},
			},
			enrichRes: &service.EnrichedData{
				Age:         30,
				Gender:      "male",
				Nationality: "US",
			},
			enrichErr: nil,
			repoErr:   nil,
			wantErr:   false,
		},
		{
			name: "enrichment failure",
			input: &model.CreatePersonDTO{
				FirstName: "Jane",
				LastName:  "Smith",
			},
			enrichRes: nil,
			enrichErr: errors.New("API timeout"),
			repoErr:   nil,
			wantErr:   true,
		},
		{
			name: "database error",
			input: &model.CreatePersonDTO{
				FirstName: "Bob",
				LastName:  "Johnson",
			},
			enrichRes: &service.EnrichedData{
				Age:         25,
				Gender:      "male",
				Nationality: "GB",
			},
			enrichErr: nil,
			repoErr:   errors.New("connection failed"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			mockEnrich := new(MockEnrichmentClient)

			mockEnrich.On("EnrichPerson", mock.Anything, tt.input.FirstName, tt.input.LastName).Return(tt.enrichRes, tt.enrichErr)

			if tt.enrichErr == nil {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Person")).Return(tt.repoErr)
			}

			personSvc := service.NewPersonService(mockRepo, mockEnrich, logger)

			ctx := context.Background()
			person, err := personSvc.Create(ctx, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, person)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, person)
				assert.Equal(t, tt.input.FirstName, person.FirstName)
				assert.Equal(t, tt.input.LastName, person.LastName)
				assert.Equal(t, tt.enrichRes.Age, person.Age)
				assert.Equal(t, tt.enrichRes.Gender, person.Gender)
				assert.Equal(t, tt.enrichRes.Nationality, person.Nationality)
			}

			mockRepo.AssertExpectations(t)
			mockEnrich.AssertExpectations(t)
		})
	}
}

func TestPersonService_GetByID(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := new(MockRepository)
	personSvc := service.NewPersonService(mockRepo, nil, logger)

	expectedPerson := &model.Person{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
	}

	tests := []struct {
		name    string
		id      int64
		repoRes *model.Person
		repoErr error
		wantErr bool
		wantNil bool
	}{
		{
			name:    "existing person",
			id:      1,
			repoRes: expectedPerson,
			repoErr: nil,
			wantErr: false,
			wantNil: false,
		},
		{
			name:    "non-existing person",
			id:      999,
			repoRes: nil,
			repoErr: errors.New("not found"),
			wantErr: true,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.On("GetByID", mock.Anything, tt.id).Return(tt.repoRes, tt.repoErr).Once()

			person, err := personSvc.GetByID(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, person)
			} else {
				assert.NotNil(t, person)
				if tt.repoRes != nil {
					assert.Equal(t, tt.repoRes.ID, person.ID)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPersonService_List(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := new(MockRepository)
	personSvc := service.NewPersonService(mockRepo, nil, logger)

	persons := []*model.Person{
		{ID: 1, FirstName: "John"},
		{ID: 2, FirstName: "Jane"},
	}

	tests := []struct {
		name         string
		page         int32
		pageSize     int32
		expectedPage int32
		expectedSize int32
		repoPersons  []*model.Person
		repoTotal    int32
		repoErr      error
		wantErr      bool
	}{
		{
			name:         "valid pagination",
			page:         1,
			pageSize:     10,
			expectedPage: 1,
			expectedSize: 10,
			repoPersons:  persons,
			repoTotal:    2,
			repoErr:      nil,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.On("List", mock.Anything, tt.expectedPage, tt.expectedSize).
				Return(tt.repoPersons, tt.repoTotal, tt.repoErr).Once()

			result, total, err := personSvc.List(context.Background(), tt.page, tt.pageSize)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.repoTotal, total)
				assert.Len(t, result, len(tt.repoPersons))
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPersonService_Update(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := new(MockRepository)
	personSvc := service.NewPersonService(mockRepo, nil, logger)

	existingPerson := &model.Person{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
	}

	newName := "John"
	var newAge int32 = 30
	dto := &model.UpdatePersonDTO{
		ID:        1,
		FirstName: &newName,
		Age:       &newAge,
	}

	mockRepo.On("GetByID", mock.Anything, int64(1)).Return(existingPerson, nil)
	mockRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := personSvc.Update(context.Background(), dto)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestPersonService_Delete(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := new(MockRepository)
	personSvc := service.NewPersonService(mockRepo, nil, logger)

	mockRepo.On("Delete", mock.Anything, int64(1)).Return(nil)

	err := personSvc.Delete(context.Background(), 1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
