package handlers

import (
	"github.com/go-chi/chi/v5"
	chi_mwr "github.com/go-chi/chi/v5/middleware"

	"github.com/pinbrain/urlshortener/internal/middleware"
	"github.com/pinbrain/urlshortener/internal/storage"
)

func NewURLRouter(urlHandler URLHandler, urlStore storage.URLStorage) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.HTTPRequestLogger)
	r.Use(middleware.GzipMiddleware)

	amw := middleware.NewAuthMiddleware(urlStore)
	r.Use(amw.AuthenticateUser)
	r.Mount("/debug", chi_mwr.Profiler())

	r.Route("/", func(r chi.Router) {
		r.Get("/ping", urlHandler.HandlePing)
		r.Get("/{urlID}", urlHandler.HandleRedirect)
		r.Post("/", urlHandler.HandleShortenURL)
	})
	r.Route("/api", func(r chi.Router) {
		r.Post("/shorten", urlHandler.HandleJSONShortenURL)
		r.Post("/shorten/batch", urlHandler.HandleShortenBatchURL)

		r.Route("/user", func(r chi.Router) {
			r.Use(amw.RequireUser)
			r.Get("/urls", urlHandler.HandleGetUsersURLs)
			r.Delete("/urls", urlHandler.HandleDeleteUserURLs)
		})
	})

	return r
}
