//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_health_checker.go -package=usecase
package usecase

import (
	"context"
)

type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}
