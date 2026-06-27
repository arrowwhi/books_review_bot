package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

func TestBookSvc_Add_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockBookRepository(ctrl)
	logger := zap.NewNop()
	svc := NewBookService(mockRepo, logger)

	ctx := context.Background()
	input := AddBookInput{
		Title:  "Мастер и Маргарита",
		Author: "Булгаков",
		Status: domain.StatusRead,
	}
	expected := &domain.Book{
		ID:     1,
		UserID: 42,
		Title:  input.Title,
		Author: input.Author,
		Status: input.Status,
	}

	mockRepo.EXPECT().
		Create(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, b *domain.Book) (*domain.Book, error) {
			assert.Equal(t, int64(42), b.UserID)
			assert.Equal(t, input.Title, b.Title)
			return expected, nil
		})

	result, err := svc.Add(ctx, 42, input)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestBookSvc_Add_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockBookRepository(ctrl)
	logger := zap.NewNop()
	svc := NewBookService(mockRepo, logger)

	ctx := context.Background()
	repoErr := errors.New("db error")

	mockRepo.EXPECT().
		Create(ctx, gomock.Any()).
		Return(nil, repoErr)

	result, err := svc.Add(ctx, 1, AddBookInput{Title: "Test"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "db error")
}

func TestBookSvc_List_Pagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockBookRepository(ctrl)
	logger := zap.NewNop()
	svc := NewBookService(mockRepo, logger)

	ctx := context.Background()
	books := make([]*domain.Book, PageSize)
	for i := range books {
		books[i] = &domain.Book{ID: int64(i + 1)}
	}

	mockRepo.EXPECT().
		List(ctx, int64(1), domain.StatusRead, 0, PageSize).
		Return(books, 13, nil)

	result, err := svc.List(ctx, 1, domain.StatusRead, 1)
	require.NoError(t, err)
	assert.Equal(t, 13, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 3, result.TotalPages)
}

func TestBookSvc_MoveToRead_AlreadyRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockBookRepository(ctrl)
	logger := zap.NewNop()
	svc := NewBookService(mockRepo, logger)

	ctx := context.Background()
	existing := &domain.Book{ID: 10, Status: domain.StatusRead}

	mockRepo.EXPECT().
		GetByID(ctx, int64(10)).
		Return(existing, nil)

	result, err := svc.MoveToRead(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, existing, result)
}

func TestBookSvc_GetByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockBookRepository(ctrl)
	logger := zap.NewNop()
	svc := NewBookService(mockRepo, logger)

	ctx := context.Background()

	mockRepo.EXPECT().
		GetByID(ctx, int64(99)).
		Return(nil, domain.ErrNotFound)

	result, err := svc.GetByID(ctx, 99)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}
