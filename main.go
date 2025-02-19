package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config holds application configuration
type Config struct {
	MongoDBURI  string
	JWTSecret   string
	ServicePort string
}

var (
	cfg        Config
	db         *mongo.Database
	operations *mongo.Collection

	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

func init() {
	fmt.Println("Starting the main....")
	cfg = Config{
		MongoDBURI:  getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		JWTSecret:   getEnv("JWT_SECRET", "secret"),
		ServicePort: getEnv("PORT", "8080"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	db = client.Database("mathapi")
	operations = db.Collection("operations")
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(metricsMiddleware)
	r.Use(recoveryMiddleware)

	// Public routes
	r.HandleFunc("/login", loginHandler).Methods("POST")

	// Protected routes
	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(authMiddleware)
	authRouter.HandleFunc("/multiply", multiplyHandler).Methods("POST")
	authRouter.HandleFunc("/divide", divideHandler).Methods("POST")
	authRouter.HandleFunc("/factorial", factorialHandler).Methods("POST")

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	log.Println("Starting server on :" + cfg.ServicePort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServicePort, r))
}

// Math operations
func multiply(a, b float64) float64 {
	return a * b
}

func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func factorial(n int) *big.Int {
	result := big.NewInt(1)
	for i := 1; i <= n; i++ {
		result.Mul(result, big.NewInt(int64(i)))
	}
	return result
}

// Handlers
func multiplyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ A, B float64 }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	result := multiply(req.A, req.B)
	logOperation(r.Context(), "multiply", req.A, req.B, result)
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"result": result})
}

func divideHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ A, B float64 }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	result, err := divide(req.A, req.B)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	logOperation(r.Context(), "divide", req.A, req.B, result)
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"result": result})
}

func factorialHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ N int }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.N < 0 {
		respondWithError(w, http.StatusBadRequest, "n must be non-negative")
		return
	}

	result := factorial(req.N)
	logOperation(r.Context(), "factorial", req.N, nil, result.String())
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"result": result.String()})
}

// Helper functions
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func logOperation(ctx context.Context, operation string, operands ...interface{}) {
	_, err := operations.InsertOne(ctx, bson.M{
		"operation": operation,
		"operands":  operands,
		"timestamp": time.Now(),
	})
	if err != nil {
		log.Printf("Failed to log operation: %v", err)
	}
}