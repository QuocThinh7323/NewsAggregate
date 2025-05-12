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

// package main

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net"
// 	"net/http"
// 	"time"

// 	"github.com/go-redis/redis/v8"
// )

// var (
// 	rdb = redis.NewClient(&redis.Options{
// 		Addr: "172.18.0.2:6379", // Đảm bảo Redis container có địa chỉ là redis:6379
// 	})
// 	ctx = context.Background()
// )

// const RATE_LIMIT = 5 // requests per minute

// // Rate limiter middleware
// func rateLimit(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// Tách IP client từ RemoteAddr
// 		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
// 		key := fmt.Sprintf("rate_limit:%s", ip)

// 		// Kiểm tra Redis
// 		val, err := rdb.Get(ctx, key).Int()
// 		if err != nil && err != redis.Nil {
// 			log.Printf("❌ Redis GET error: %v", err) // Ghi log lỗi Redis chi tiết
// 			http.Error(w, "Redis error", http.StatusInternalServerError)
// 			return
// 		}

// 		// Kiểm tra số lần request
// 		if val >= RATE_LIMIT {
// 			http.Error(w, "Too many requests", http.StatusTooManyRequests)
// 			return
// 		}

// 		// Cập nhật số request và hết hạn sau 1 phút
// 		pipe := rdb.TxPipeline()
// 		pipe.Incr(ctx, key)
// 		pipe.Expire(ctx, key, time.Minute)
// 		_, _ = pipe.Exec(ctx)

// 		// Tiếp tục xử lý request
// 		next.ServeHTTP(w, r)
// 	})
// }

// // Proxy route: /api/articles → article-service
// func handleArticles(w http.ResponseWriter, r *http.Request) {
// 	resp, err := http.Get("http://localhost:8081/articles") // hoặc article-service trong Docker
// 	if err != nil {
// 		http.Error(w, "Service unavailable", http.StatusInternalServerError)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	// Chuyển tiếp header và body từ article-service
// 	for k, v := range resp.Header {
// 		w.Header()[k] = v
// 	}
// 	w.WriteHeader(resp.StatusCode)
// 	_, _ = io.Copy(w, resp.Body)
// }

// func main() {
// 	mux := http.NewServeMux()

// 	// Serve static files tại /static/
// 	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./api-gateway/static"))))

// 	// Serve index.html tại root /
// 	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		http.ServeFile(w, r, "./api-gateway/static/index.html")
// 	})

// 	// Route API
// 	mux.HandleFunc("/api/articles", handleArticles)

// 	// Bọc middleware rate limit
// 	rateLimitedHandler := rateLimit(mux)

// 	fmt.Println("🚀 API Gateway chạy tại http://localhost:8082")
// 	log.Fatal(http.ListenAndServe(":8082", rateLimitedHandler))
// }
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func handleArticles(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("http://localhost:8081/articles") // hoặc dùng article-service nếu chạy bằng Docker Compose
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Forward headers and response
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func main() {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./api-gateway/static"))))

	// Serve index.html at /
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./api-gateway/static/index.html")
	})

	// API route
	mux.HandleFunc("/api/articles", handleArticles)

	fmt.Println("🚀 API Gateway chạy tại http://localhost:8082")
	log.Fatal(http.ListenAndServe(":8082", mux))
}
