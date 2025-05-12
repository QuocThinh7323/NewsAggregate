package main

import (
    "context"
    "encoding/json" 


    
    "fmt"
    "github.com/gin-gonic/gin"
    "github.com/segmentio/kafka-go"
    "net/http"
    "sync"
)

type Article struct {
    Source      string `json:"source"`
    Title       string `json:"title"`
    Content     string `json:"content"`
    PublishedAt string `json:"publishedAt"`
}

var (
    articles []Article
    mu       sync.Mutex
)

func consumeKafka() {
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers: []string{"kafka:9092"},
        Topic:   "news-topic",
        GroupID: "article-group",
    })
    defer r.Close()

    for {
        m, err := r.ReadMessage(context.Background())
        if err != nil {
            fmt.Println("❌ Kafka Read Error:", err)
            continue
        }
        fmt.Println("📥 Tin mới từ Kafka:", string(m.Value))

        var art Article
        err = json.Unmarshal(m.Value, &art) // Sử dụng json.Unmarshal
        if err == nil {
            mu.Lock()
            articles = append(articles, art)
            mu.Unlock()
        }
    }
}

func main() {
    go consumeKafka()

    r := gin.Default()

    r.GET("/articles", func(c *gin.Context) {
        mu.Lock()
        defer mu.Unlock()
        c.JSON(http.StatusOK, articles)
    })

    fmt.Println("🚀 Article service chạy tại cổng 8081")
    r.Run(":8081")
}