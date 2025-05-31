package httpServer

func (h *handler) RegisterRoutes() {

	apiv1 := h.server.Group("/api/v1", h.loggerMiddleware)
	{
		apiv1.Post("/providers/search", h.rateLimiterMiddleware("provicers", 10), h.searchProviders)

		apiv1.Post("/providers", h.rateLimiterMiddleware("providers", 10), h.updateTelemetry)

		apiv1.Get("/providers", h.rateLimiterMiddleware("providers", 60), h.authorizationMiddleware, h.getLatestTelemetry)
	}

	h.server.Get("/health", h.rateLimiterMiddleware("health", 60), h.health)
}
