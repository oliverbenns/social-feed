package api

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	redis "github.com/redis/go-redis/v9"
	sloggin "github.com/samber/slog-gin"
)

type Service struct {
	RedisClient     *redis.Client
	Port            int
	Logger          *slog.Logger
	InstagramAppID  string
	InstagramSecret string
	AppURL          string
}

func (s *Service) Run(ctx context.Context) error {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(sloggin.New(s.Logger))

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true

	router.Use(cors.New(config))

	router.GET("/", s.GetHome)

	router.GET("/ping", s.GetPing)

	// Instagram
	router.GET("/instagram/auth", s.GetInstagramAuth)
	router.GET("/instagram/auth/callback", s.GetInstagramAuthCallback)
	router.GET("/instagram/feed/:username", s.GetInstagramFeed)

	addr := fmt.Sprintf(":%d", s.Port)
	router.Run(addr)

	return nil
}

func (s *Service) GetHome(c *gin.Context) {
	keys, err := s.RedisClient.Keys(c.Request.Context(), "instagram_credential_*").Result()
	if err != nil {
		s.Logger.Error("Error getting credential:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	usernames := []string{}
	for _, key := range keys {
		username := key[len("instagram_credential_"):]
		usernames = append(usernames, username)
	}

	usernameList := "<ul>"
	for _, username := range usernames {
		usernameList += fmt.Sprintf("<li><a href=\"/instagram/feed/%s\" target=\"_blank\">%s</a></li>", username, username)
	}
	usernameList += "</ul>"

	authUrl, err := s.getInstagramAuthURL()
	if err != nil {
		s.Logger.Error("Error getting auth url:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	html := fmt.Sprintf(`
		<html>
			<head>
				<title>Social Feed Prototype</title>
			</head>
			<body>
				<h1>Social Feed Prototype</h1>
				<p>Connect a new account here: <a href="%s">Connect Instagram</a></p>
				<hr />
				<h4>Connected Instagram accounts:</h4>
				%s
			</body>
		</html>
	`, authUrl, usernameList)

	c.Writer.WriteString(html)

	c.Status(200)
	return
}

func (s *Service) GetPing(c *gin.Context) {
	c.PureJSON(200, gin.H{
		"message": "pong",
	})
}
