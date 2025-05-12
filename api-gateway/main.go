// package main

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/go-redis/redis/v8"
// )

// var (
// 	rdb = redis.NewClient(&redis.Options{
// 		Addr: "localhost:6379",
// 	})
// 	ctx = context.Background()
// )

// const RATE_LIMIT = 5 // số request mỗi phút

// func rateLimitMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		ip := c.ClientIP()
// 		key := fmt.Sprintf("rate_limit:%s", ip)

// 		val, err := rdb.Get(ctx, key).Int()
// 		if err != nil && err != redis.Nil {
// 			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
// 			return
// 		}

// 		if val >= RATE_LIMIT {
// 			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
// 			return
// 		}

// 		pipe := rdb.TxPipeline()
// 		pipe.Incr(ctx, key)
// 		pipe.Expire(ctx, key, time.Minute)
// 		_, _ = pipe.Exec(ctx)

// 		c.Next()
// 	}
// }

// func main() {
// 	r := gin.Default()

// 	// Middleware giới hạn rate
// 	r.Use(rateLimitMiddleware())

// 	// Route chuyển tiếp tới article-service
// 	r.GET("/api/articles", func(c *gin.Context) {
// 		resp, err := http.Get("http://article-service:8081/articles")
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Service unavailable"})
// 			return
// 		}
// 		defer resp.Body.Close()

// 		c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
// 	})

// 	// 📁 Phục vụ file tĩnh như index.html từ thư mục static
// 	r.Static("/static", "./api-gateway/static/")

// 	fmt.Println("🚪 API Gateway chạy tại cổng 8080")
// 	r.Run(":8080")
// }

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	rdb = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	ctx = context.Background()
)

const RATE_LIMIT = 5 // requests per minute

// Rate limiter middleware
func rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		key := fmt.Sprintf("rate_limit:%s", ip)

		val, err := rdb.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			http.Error(w, "Redis error", http.StatusInternalServerError)
			return
		}

		if val >= RATE_LIMIT {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		pipe := rdb.TxPipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Minute)
		_, _ = pipe.Exec(ctx)

		next.ServeHTTP(w, r)
	})
}

// Proxy route: /api/articles → article-service
func handleArticles(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("http://localhost:8081/articles") // or article-service in Docker
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func main() {
	mux := http.NewServeMux()

	// Serve static files at /static/
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./api-gateway/static"))))

	// Serve index.html at root /
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./api-gateway/static/index.html")
	})

	// API route
	mux.HandleFunc("/api/articles", handleArticles)

	// Wrap with rate limiter
	rateLimitedHandler := rateLimit(mux)

	fmt.Println("🚀 API Gateway running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", rateLimitedHandler))
}
