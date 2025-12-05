//go:generate mockgen -source=$GOFILE -destination=../../tests/handler/mock_readyz_handler.go -package=handler
package handler

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/usecase"
)

type ReadinessUseCaseInterface interface {
	ExecuteDetails(ctx context.Context) ([]usecase.HealthCheckResult, error)
}

type ReadyzHandler struct {
	uc ReadinessUseCaseInterface
}

func NewReadyzHandler(uc ReadinessUseCaseInterface) *ReadyzHandler {
	return &ReadyzHandler{
		uc: uc,
	}
}

func (h *ReadyzHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	results, err := h.uc.ExecuteDetails(ctx)
	if err != nil {
		response := map[string]interface{}{
			"status":  "not ready",
			"details": convertResultsToMap(results),
		}
		return c.JSON(http.StatusServiceUnavailable, response)
	}

	response := map[string]interface{}{
		"status":  "ready",
		"details": convertResultsToMap(results),
	}
	return c.JSON(http.StatusOK, response)
}

func convertResultsToMap(results []usecase.HealthCheckResult) []map[string]interface{} {
	details := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		detail := map[string]interface{}{
			"name":    r.Name,
			"healthy": r.Healthy,
		}
		if r.Error != nil {
			detail["error"] = r.Error.Error()
		}
		details = append(details, detail)
	}
	return details
}
