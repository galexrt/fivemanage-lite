package publicapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func registerTraceApi(group *echo.Group) {
	h := &traceHandler{}
	group.GET("/traces", h.getTraces)
}

type traceHandler struct{}

// getTraces godoc
// @Summary      Get traces
// @Description  Get tracing information
// @Tags         public
// @Produce      json
// @Success      200  {object}  echo.Map
// @Router       /traces [get]
func (h *traceHandler) getTraces(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"success": true,
	})
}
