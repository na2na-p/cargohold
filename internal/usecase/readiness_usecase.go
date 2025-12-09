package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrHealthCheckFailed はヘルスチェックが失敗したことを示すエラー
var ErrHealthCheckFailed = errors.New("health check failed")

// HealthCheckResult は個々のヘルスチェック結果を表す
type HealthCheckResult struct {
	Name    string
	Healthy bool
	Error   error
}

// ReadinessUseCase はアプリケーションのReadinessチェックを実行するUseCase
type ReadinessUseCase struct {
	checkers []HealthChecker
}

// NewReadinessUseCase は新しいReadinessUseCaseを生成する
func NewReadinessUseCase(checkers ...HealthChecker) *ReadinessUseCase {
	return &ReadinessUseCase{
		checkers: checkers,
	}
}

// Execute はすべてのヘルスチェッカーを実行し、1つでも失敗した場合はエラーを返す
func (uc *ReadinessUseCase) Execute(ctx context.Context) error {
	_, err := uc.ExecuteDetails(ctx)
	return err
}

// ExecuteDetails はすべてのヘルスチェッカーを実行し、詳細な結果を返す
func (uc *ReadinessUseCase) ExecuteDetails(ctx context.Context) ([]HealthCheckResult, error) {
	results := make([]HealthCheckResult, 0, len(uc.checkers))
	var failedCheckers []string

	for _, checker := range uc.checkers {
		err := checker.Check(ctx)
		result := HealthCheckResult{
			Name:    checker.Name(),
			Healthy: err == nil,
			Error:   err,
		}
		results = append(results, result)

		if err != nil {
			failedCheckers = append(failedCheckers, fmt.Sprintf("%s: %v", checker.Name(), err))
		}
	}

	if len(failedCheckers) > 0 {
		return results, fmt.Errorf("%w: %s", ErrHealthCheckFailed, strings.Join(failedCheckers, "; "))
	}

	return results, nil
}
