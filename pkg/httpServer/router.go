package httpServer

func (h *handler) RegisterRoutes() {

	apiv1 := h.server.Group("/api/v1", h.loggerMiddleware)
	{
		apiv1.Post("/providers/search", h.rateLimiterMiddleware("provicers", 10), h.getProviders)
	}

	h.server.Get("/health", h.rateLimiterMiddleware("health", 60), h.health)
}
