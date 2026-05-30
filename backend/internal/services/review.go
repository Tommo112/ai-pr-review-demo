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

type StreamEmitter func(event string, data any) error

type PRStreamAnalyzer interface {
	AnalyzePullRequestStream(ctx context.Context, ref models.PRRef, pr models.PullRequestData, emit StreamEmitter) (models.ReviewResponse, error)
}

func (service *ReviewService) ReviewStream(ctx context.Context, prURL string, emit StreamEmitter) error {
	ref, err := ParsePRURL(prURL)
	if err != nil {
		return err
	}

	pr, err := service.fetcher.FetchPullRequest(ctx, ref)
	if err != nil {
		return err
	}

	baseResponse := newReviewResponseFromGitHub(pr)
	if err := emit("pr", models.ReviewStartEvent{
		PR:    baseResponse.PR,
		Files: baseResponse.Files,
	}); err != nil {
		return err
	}
	if err := emit("status", models.ReviewStatusEvent{Message: "analyzing_ai"}); err != nil {
		return err
	}

	var response models.ReviewResponse
	if streamAnalyzer, ok := service.analyzer.(PRStreamAnalyzer); ok {
		response, err = streamAnalyzer.AnalyzePullRequestStream(ctx, ref, pr, emit)
	} else {
		response, err = service.analyzer.AnalyzePullRequest(ctx, ref, pr)
	}
	if err != nil {
		return err
	}

	return emit("review", models.ReviewDoneEvent{Review: response})
}
