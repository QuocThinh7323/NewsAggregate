# Go service directories
SCRAPER_DIR = news-scraper
ARTICLE_DIR = article-service
GATEWAY_DIR = api-gateway

# Build từng service
build-scraper:
	cd $(SCRAPER_DIR) && go build -o scraper

build-article:
	cd $(ARTICLE_DIR) && go build -o article-service

build-gateway:
	cd $(GATEWAY_DIR) && go build -o api-gateway

# Build tất cả
build-all: build-scraper build-article build-gateway

# Run từng service
run-scraper:
	cd $(SCRAPER_DIR) && go run main.go

run-article:
	cd $(ARTICLE_DIR) && go run main.go

run-gateway:
	cd $(GATEWAY_DIR) && go run main.go

# Run all services (phải mở 3 terminal riêng hoặc dùng tmux/screen)
run-all:
	@echo "📌 Vui lòng chạy từng service ở các terminal khác nhau:"
	@echo "👉 make run-scraper"
	@echo "👉 make run-article"
	@echo "👉 make run-gateway"

# Docker compose
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Clean các file thực thi
clean:
	rm -f $(SCRAPER_DIR)/scraper $(ARTICLE_DIR)/article-service $(GATEWAY_DIR)/api-gateway

.PHONY: build-scraper build-article build-gateway build-all run-scraper run-article run-gateway run-all docker-up docker-down clean
