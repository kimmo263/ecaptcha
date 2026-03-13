// Package example 提供 ECaptcha 集成示例
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"global-track/pkg/ecaptcha"
	"global-track/pkg/ecaptcha/behavior"
	"global-track/pkg/ecaptcha/image"
	"global-track/pkg/ecaptcha/slider"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	// 1. 初始化 Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 2. 创建存储
	store := ecaptcha.NewRedisStore(redisClient)

	// 3. 配置
	config := ecaptcha.Config{
		Expire:            5 * time.Minute,
		MaxAttempts:       3,
		TokenExpire:       10 * time.Minute,
		ImageWidth:        150,
		ImageHeight:       50,
		ImageLength:       4,
		SliderWidth:       300,
		SliderHeight:      150,
		PieceSize:         50,
		Tolerance:         5,
		BehaviorThreshold: 0.7,
	}

	// 4. 创建验证服务
	captcha := ecaptcha.New(config, store)

	// 5. 注册验证提供者
	captcha.RegisterProvider(image.New(config, store))
	captcha.RegisterProvider(slider.New(config, store))
	captcha.RegisterProvider(behavior.New(config, store, "your-secret-key"))

	// 6. 创建 HTTP Handler
	handler := ecaptcha.NewHandler(captcha)

	// 7. 注册路由
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, "/ecaptcha")

	// 8. 静态文件服务 (前端 SDK)
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/ecaptcha/sdk/", http.StripPrefix("/ecaptcha/sdk/", http.FileServer(http.FS(staticFS))))

	// 9. 示例页面
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/demo.html")
	})

	log.Println("ECaptcha server running on http://localhost:8089")
	log.Println("API: http://localhost:8089/ecaptcha/")
	log.Println("SDK: http://localhost:8089/ecaptcha/sdk/ecaptcha.min.js")
	log.Fatal(http.ListenAndServe(":8089", mux))
}
