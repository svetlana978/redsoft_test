package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	RateLimiter "person-service/pkg/ratelimiter"

	"go.uber.org/zap"
)

type Enrich interface {
	EnrichPerson(context.Context, string, string) (*EnrichedData, error)
}

type EnrichmentService struct {
	httpClient     *http.Client
	logger         *zap.Logger
	agifyURL       string
	genderizeURL   string
	nationalizeURL string
	limiter        *RateLimiter.RateLimiter
	allowTimoeout  time.Duration
}

type EnrichedData struct {
	Age         int32
	Gender      string
	Nationality string
}

func NewEnrichmentService(logger *zap.Logger, agifyURL, genderizeURL, nationalizeURL string, timeout, allowTimeout time.Duration) *EnrichmentService {
	return &EnrichmentService{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		logger:         logger,
		agifyURL:       agifyURL,
		genderizeURL:   genderizeURL,
		nationalizeURL: nationalizeURL,
		limiter:        RateLimiter.New(),
		allowTimoeout:  allowTimeout,
	}
}

func (e *EnrichmentService) EnrichPerson(ctx context.Context, firstName, lastName string) (*EnrichedData, error) {
	var (
		age         int32
		gender      string
		nationality string
		wg          sync.WaitGroup
	)

	wg.Add(3)

	var cnt int32

	go func() {
		defer wg.Done()
		if val, err := e.getAgeWithLimit(ctx, firstName); err != nil {
			e.logger.Error("getAge failed", zap.Error(err))
			atomic.AddInt32(&cnt, 1)
		} else {
			age = val
		}
	}()

	go func() {
		defer wg.Done()
		if val, err := e.getGenderWithLimit(ctx, firstName); err != nil {
			e.logger.Error("getGender failed", zap.Error(err))
			atomic.AddInt32(&cnt, 1)
		} else {
			gender = val
		}
	}()

	go func() {
		defer wg.Done()
		if val, err := e.getNationalityWithLimit(ctx, firstName); err != nil {
			e.logger.Error("getNationality failed", zap.Error(err))
			atomic.AddInt32(&cnt, 1)
		} else {
			nationality = val
		}
	}()

	wg.Wait()

	if cnt > 0 {
		return nil, errors.New("failed to enrich data")
	}

	return &EnrichedData{
		Age:         age,
		Gender:      gender,
		Nationality: nationality,
	}, nil
}

func (e *EnrichmentService) getAgeWithLimit(ctx context.Context, name string) (int32, error) {
	allowed, waitSeconds := e.limiter.Allow("enrich")
	if !allowed {
		e.logger.Info("agify rate limit reached, waiting",
			zap.Int("wait_seconds", waitSeconds))

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(e.allowTimoeout):
			return 0, fmt.Errorf("exceed limit for agify for person %s", name)
		case <-time.After(time.Duration(waitSeconds) * time.Second):
			// все ок, по идее лимит должен освободится и можем делать запрос
		}
	}

	age, err := e.getAge(ctx, name)
	if err != nil {
		return 0, err
	}

	return age, nil
}

func (e *EnrichmentService) getGenderWithLimit(ctx context.Context, name string) (string, error) {
	allowed, waitSeconds := e.limiter.Allow("enrich")
	if !allowed {
		e.logger.Info("genderize rate limit reached, waiting",
			zap.Int("wait_seconds", waitSeconds))

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(e.allowTimoeout):
			return "", fmt.Errorf("exceed limit for agify for person %s", name)
		case <-time.After(time.Duration(waitSeconds) * time.Second):
			// все ок, по идее лимит должен освободится и можем делать запрос
		}
	}

	gender, err := e.getGender(ctx, name)
	if err != nil {
		return "", err
	}

	return gender, nil
}

func (e *EnrichmentService) getNationalityWithLimit(ctx context.Context, name string) (string, error) {
	allowed, waitSeconds := e.limiter.Allow("enrich")
	if !allowed {
		e.logger.Info("nationalize rate limit reached, waiting",
			zap.Int("wait_seconds", waitSeconds))

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(e.allowTimoeout):
			return "", fmt.Errorf("exceed limit for agify for person %s", name)
		case <-time.After(time.Duration(waitSeconds) * time.Second):
			// все ок, по идее лимит должен освободится и можем делать запрос
		}
	}

	nationality, err := e.getNationality(ctx, name)
	if err != nil {
		return "", err
	}

	return nationality, nil
}

func (e *EnrichmentService) getAge(ctx context.Context, name string) (int32, error) {
	url := fmt.Sprintf("%s/?name=%s", e.agifyURL, name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	e.parseRateLimitHeaders("agify", resp)

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("age request failed, code %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Age int32 `json:"age"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Age, nil
}

func (e *EnrichmentService) getGender(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf("%s/?name=%s", e.genderizeURL, name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	e.parseRateLimitHeaders("genderize", resp)

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gender request failed, code %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Gender string `json:"gender"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Gender == "" {
		return "unknown", nil
	}

	return result.Gender, nil
}

func (e *EnrichmentService) getNationality(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf("%s/?name=%s", e.nationalizeURL, name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	e.parseRateLimitHeaders("nationalize", resp)

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("nationality request failed, code %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Country []struct {
			CountryID   string  `json:"country_id"`
			Probability float64 `json:"probability"`
		} `json:"country"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Country) > 0 {
		return result.Country[0].CountryID, nil
	}

	return "unknown", nil
}

func (e *EnrichmentService) parseRateLimitHeaders(apiName string, resp *http.Response) {
	total := resp.Header.Get("X-Rate-Limit-Limit")
	remaining := resp.Header.Get("X-Rate-Limit-Remaining")
	reset := resp.Header.Get("X-Rate-Limit-Reset")

	if total != "" && remaining != "" && reset != "" {
		var totalInt, remainingInt, resetInt int
		fmt.Sscanf(total, "%d", &totalInt)
		fmt.Sscanf(remaining, "%d", &remainingInt)
		fmt.Sscanf(reset, "%d", &resetInt)

		e.limiter.Update(apiName, totalInt, remainingInt, resetInt)

		e.logger.Debug("updated rate limits",
			zap.String("api", apiName),
			zap.Int("total", totalInt),
			zap.Int("remaining", remainingInt),
			zap.Int("reset_seconds", resetInt),
		)
	}
}
