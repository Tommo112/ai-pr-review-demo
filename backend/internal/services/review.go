package services

import (
	"context"

	"demo/backend/internal/models"
)

type ReviewService struct {
	fetcher  PullRequestFetcher
	analyzer PRAnalyzer
}

func NewReviewService(fetcher PullRequestFetcher, analyzer PRAnalyzer) *ReviewService {
	return &ReviewService{fetcher: fetcher, analyzer: analyzer}
}

func (service *ReviewService) Review(ctx context.Context, prURL string) (models.ReviewResponse, error) {
	ref, err := ParsePRURL(prURL)
	if err != nil {
		return models.ReviewResponse{}, err
	}

	pr, err := service.fetcher.FetchPullRequest(ctx, ref)
	if err != nil {
		return models.ReviewResponse{}, err
	}

	return service.analyzer.AnalyzePullRequest(ctx, ref, pr)
}
