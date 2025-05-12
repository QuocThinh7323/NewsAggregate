package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/PuerkitoBio/goquery"
    "github.com/segmentio/kafka-go"
)

// Article model
type Article struct {
    Title       string    `json:"title"`
    URL         string    `json:"url"`
    Source      string    `json:"source"`
    PublishedAt time.Time `json:"publishedAt"`
}

// Kafka writer setup
var kafkaWriter = kafka.NewWriter(kafka.WriterConfig{
    Brokers:  []string{"kafka:9092"},
    Topic:    "news-topic",
    Balancer: &kafka.LeastBytes{},
})

// Kafka reader setup
var kafkaReader = kafka.NewReader(kafka.ReaderConfig{
    Brokers: []string{"kafka:9092"},
    Topic:   "news-topic",
    GroupID: "news-consumer-group",
})

// Gửi bài viết vào Kafka
func publishArticles(articles []Article) {
    for _, article := range articles {
        data, _ := json.Marshal(article)
        err := kafkaWriter.WriteMessages(context.Background(), kafka.Message{
            Value: data,
        })
        if err != nil {
            fmt.Printf("❌ Lỗi gửi Kafka: %v\n", err)
        } else {
            fmt.Printf("✅ Gửi: [%s] %s\n", article.Source, article.Title)
        }
    }
}

// Scrape VnExpress
func ScrapeVnExpress() ([]Article, error) {
    res, err := http.Get("https://vnexpress.net")
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()

    doc, err := goquery.NewDocumentFromReader(res.Body)
    if err != nil {
        return nil, err
    }

    var articles []Article
    doc.Find("h3.title-news a").Each(func(i int, s *goquery.Selection) {
        title := strings.TrimSpace(s.Text())
        url, _ := s.Attr("href")
        if title != "" && strings.HasPrefix(url, "http") {
            articles = append(articles, Article{
                Title:       title,
                URL:         url,
                Source:      "vnexpress.net",
                PublishedAt: time.Now(),
            })
        }
    })
    return articles, nil
}

// Scrape Tuổi Trẻ
func ScrapeTuoiTre() ([]Article, error) {
    res, err := http.Get("https://tuoitre.vn")
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()

    doc, err := goquery.NewDocumentFromReader(res.Body)
    if err != nil {
        return nil, err
    }

    var articles []Article
    doc.Find("h3.title-news a").Each(func(i int, s *goquery.Selection) {
        title := strings.TrimSpace(s.Text())
        url, _ := s.Attr("href")
        if title != "" && strings.HasPrefix(url, "http") {
            articles = append(articles, Article{
                Title:       title,
                URL:         url,
                Source:      "tuoitre.vn",
                PublishedAt: time.Now(),
            })
        }
    })
    return articles, nil
}

// Scrape Thanh Niên
func ScrapeThanhNien() ([]Article, error) {
    res, err := http.Get("https://thanhnien.vn")
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()

    doc, err := goquery.NewDocumentFromReader(res.Body)
    if err != nil {
        return nil, err
    }

    var articles []Article
    doc.Find("h2.story__heading a").Each(func(i int, s *goquery.Selection) {
        title := strings.TrimSpace(s.Text())
        url, _ := s.Attr("href")
        if title != "" && strings.HasPrefix(url, "http") {
            articles = append(articles, Article{
                Title:       title,
                URL:         url,
                Source:      "thanhnien.vn",
                PublishedAt: time.Now(),
            })
        }
    })
    return articles, nil
}

// scrapeSource dùng để scrape theo tên nguồn
func scrapeSource(name string, wg *sync.WaitGroup) {
    defer wg.Done()

    var articles []Article
    var err error

    switch name {
    case "vnexpress.net":
        articles, err = ScrapeVnExpress()
    case "tuoitre.vn":
        articles, err = ScrapeTuoiTre()
    case "thanhnien.vn":
        articles, err = ScrapeThanhNien()
    default:
        fmt.Printf("❌ Chưa hỗ trợ nguồn: %s\n", name)
        return
    }

    if err != nil {
        fmt.Printf("❌ Lỗi scrape từ %s: %v\n", name, err)
        return
    }

    publishArticles(articles)
}

// Chạy nhiều scraper song song
func scrapeAllSources(sources []string) {
    var wg sync.WaitGroup
    for _, src := range sources {
        wg.Add(1)
        go scrapeSource(src, &wg)
    }
    wg.Wait()
}

// Hàm đọc bài viết từ Kafka
func readArticlesFromKafka() ([]Article, error) {
    var articles []Article

    for {
        msg, err := kafkaReader.ReadMessage(context.Background())
        if err != nil {
            return nil, err
        }

        var article Article
        err = json.Unmarshal(msg.Value, &article)
        if err != nil {
            fmt.Printf("❌ Lỗi giải mã Kafka: %v\n", err)
            continue
        }

        articles = append(articles, article)
    }

    return articles, nil
}

// Hàm xử lý yêu cầu /articles
func articlesHandler(w http.ResponseWriter, r *http.Request) {
    articles, err := readArticlesFromKafka()
    if err != nil {
        http.Error(w, fmt.Sprintf("Lỗi đọc dữ liệu từ Kafka: %v", err), http.StatusInternalServerError)
        return
    }

    response, err := json.Marshal(articles)
    if err != nil {
        http.Error(w, fmt.Sprintf("Lỗi mã hóa JSON: %v", err), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(response)
}

func main() {
    defer kafkaWriter.Close()

    sources := []string{"vnexpress.net", "tuoitre.vn", "thanhnien.vn"}

    fmt.Println("🚀 Bắt đầu scrape lần đầu")
    scrapeAllSources(sources)

    ticker := time.NewTicker(10 * time.Minute)
    defer ticker.Stop()

    // Khởi tạo API server
    http.HandleFunc("/articles", articlesHandler)
    go http.ListenAndServe(":8081", nil)

    // Chạy định kỳ
    for {
        select {
        case t := <-ticker.C:
            fmt.Printf("⏱️  Chạy định kỳ lúc %s\n", t.Format(time.RFC3339))
            scrapeAllSources(sources)
        }
    }
}
