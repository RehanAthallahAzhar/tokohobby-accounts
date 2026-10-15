package middlewares

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

func LoggingMiddleware(log *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			req := c.Request()
			res := c.Response()

			log.InfoContext(req.Context(), "HTTP Request",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", res.Status),
				slog.String("remote_addr", c.RealIP()),
				slog.Duration("latency", time.Since(start)),
			)

			return err
		}
	}
}

