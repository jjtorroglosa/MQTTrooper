package internal

import (
	"context"
	"errors"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Link struct {
	Service Service
}

type _http struct {
	cfg   Config
	links []Link
}

func NewHttp(cfg Config) *_http {
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
	tmpl, err := template.ParseFiles("templates/index.html.tmpl")
	if err != nil {
		log.Println("Error reading the template")
		log.Println(err)
		return
	}
	err2 := tmpl.Execute(w, h.links)
	if err2 != nil {
		log.Println(err2)
	}
}

func ExecuteHandler(
	execute Executor,
	allowedAddress string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			log.Printf("[GET] %s %s\n", r.RemoteAddr, r.URL)
			log.Printf("[GET] %s %s\n", r.Header, r.URL)
			if strings.Split(r.RemoteAddr, ":")[0] != allowedAddress {
				unauthorized := "Unauthorized"
				log.Println(unauthorized)
				return errors.New(unauthorized)
			}
			service := html.EscapeString(r.URL.Query().Get("s"))

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

func (h *_http) ListenHttp(bindAddress string, port int, allowedAddress string, execute Executor) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.HomeHandler)
	mux.HandleFunc("/r", ExecuteHandler(execute, allowedAddress))

	addressPort := fmt.Sprintf("%s:%d", bindAddress, port)
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
