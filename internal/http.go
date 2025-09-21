package internal

import (
	"context"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/csrf"
)

type Link struct {
	Service   Service
	CSRFToken string
}

type _http struct {
	cfg   *Config
	links []Link
}

func NewHttp(cfg *Config) *_http {
	var links []Link
	for _, service := range cfg.ServicesList {
		links = append(links, Link{Service: service})
	}
	return &_http{
		cfg:   cfg,
		links: links,
	}
}

func (h _http) HomeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templates, "templates/index.html.tmpl")
	if err != nil {
		log.Println("Error reading the template")
		log.Println(err)
		return
	}
	// We need to create a new slice of links with the CSRF token
	view := struct {
		Links     []Link
		CsrfToken template.HTML
		CsrfTag   string
	}{
		Links:     []Link{},
		CsrfToken: csrf.TemplateField(r),
	}

	for _, link := range h.links {
		view.Links = append(view.Links, Link{
			Service: link.Service,
		})
	}

	err2 := tmpl.Execute(w, view)
	if err2 != nil {
		log.Println(err2)
	}
}

func (h _http) ExecuteHandler(
	execute Executor,
	allowedAddress string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			log.Printf("[POST] %s %s\n", r.RemoteAddr, r.URL)
			log.Printf("[POST] %s %s\n", r.Header, r.URL)
			if strings.Split(r.RemoteAddr, ":")[0] != allowedAddress {
				unauthorized := "Unauthorized"
				log.Println(unauthorized)
				return errors.New(unauthorized)
			}
			service := html.EscapeString(r.FormValue("s"))
			if _, ok := h.cfg.Services[service]; !ok {
				return errors.New("Unknown service")
			}

			err := execute(service)
			if err != nil {
				log.Println(err)
				return err
			}
			fmt.Fprintf(w, "{\"result\": \"ok\"}\n")

			return nil
		}()

		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func (h *_http) ListenHttp(cfg *HttpConfig, execute Executor) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.HomeHandler)
	mux.HandleFunc("/r", h.ExecuteHandler(execute, cfg.AllowedAddress))

	addressPort := fmt.Sprintf("%s:%d", cfg.BindAddress, cfg.Port)

	// TODO add middleware to check csrf
	_ = csrf.Protect(cfg.CsrfSecret,
		csrf.Secure(false), // Set to true in production
		csrf.CookieName("csrf_token"),
		csrf.FieldName("csrf_token"),
		csrf.TrustedOrigins([]string{
			"http://127.0.0.1:8080",
		}),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, _ := io.ReadAll(r.Body)
			log.Printf("CSRF token invalid: %v %v %v %v\n", r.Method, r.URL, string(bytes), csrf.FailureReason(r))
			log.Printf("Origin: %q, Referer: %q", r.Header.Get("Origin"), r.Header.Get("Referer"))

			http.Error(w, "Forbidden - CSRF token invalid", http.StatusForbidden)
		})),
	)

	srv := http.Server{
		Addr:    addressPort,
		Handler: mux,
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("HTTP: Shutting down server...")

		// Context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Println("HTTP: Server forced to shutdown:", err)
		} else {
			log.Println("HTTP: Server exited gracefully")
		}
	}()
	log.Printf("Listening on http://%s\n", addressPort)
	log.Fatal(srv.ListenAndServe())
}
