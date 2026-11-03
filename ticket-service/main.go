package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"ticket-service/internal/controller"
	"ticket-service/internal/db"
	"ticket-service/internal/repository"
	"ticket-service/internal/service"
	"ticket-service/internal/tracing"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %dms", r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Milliseconds())
	})
}

func main() {
	ctx := context.Background()

	shutdownTracing, err := tracing.Init(ctx)
	if err != nil {
		log.Fatalf("tracing: %v", err)
	}

	pool, err := db.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	if err := db.Init(ctx, pool); err != nil {
		log.Fatalf("db init: %v", err)
	}

	mux := http.NewServeMux()
	controller.New(service.NewTicketService(repository.NewTicketRepository(pool))).Routes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	srv := &http.Server{Addr: ":" + port, Handler: otelhttp.NewHandler(logRequests(mux), "ticket-service")}

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		shCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		srv.Shutdown(shCtx)
		shutdownTracing(shCtx)
	}()

	log.Printf("ticket-service listening on %s", port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
