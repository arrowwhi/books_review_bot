package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

func TestStatsSvc_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockBookRepository(ctrl)
	logger := zap.NewNop()
	svc := NewStatsService(mockRepo, logger)

	ctx := context.Background()
	expected := &domain.Stats{
		TotalBooks:     10,
		BooksThisYear:  3,
		AvgRating:      4.2,
		FavoriteAspect: "plot",
	}

	mockRepo.EXPECT().
		GetStats(ctx, int64(7)).
		Return(expected, nil)

	result, err := svc.Get(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
