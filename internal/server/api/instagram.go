package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type InstagramCredential struct {
	AccessToken string `json:"access_token"`
	UserName    string `json:"username"`
	UserID      int    `json:"user_id"`
}

func (s *Service) GetInstagramAuth(c *gin.Context) {
	authUrl, err := s.getInstagramAuthURL()
	if err != nil {
		s.Logger.Error("Error getting auth url:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	c.PureJSON(200, gin.H{
		"url": authUrl,
	})
}

func (s *Service) getInstagramAuthURL() (string, error) {
	authCodeUrl, err := url.Parse("https://api.instagram.com/oauth/authorize")
	if err != nil {
		return "", fmt.Errorf("auth code url parse failed: %w", err)
	}

	appUrl, err := url.Parse(s.AppURL)
	if err != nil {
		return "", fmt.Errorf("app url parse failed: %w", err)
	}

	callbackUrl := appUrl
	callbackUrl.Path = "instagram/auth/callback"

	q := url.Values{}
	q.Set("client_id", s.InstagramAppID)
	q.Set("redirect_uri", callbackUrl.String())
	q.Set("scope", "user_profile,user_media")
	q.Set("response_type", "code")
	authCodeUrl.RawQuery = q.Encode()

	return authCodeUrl.String(), nil
}

func (s *Service) GetInstagramAuthCallback(c *gin.Context) {
	codes := c.QueryArray("code")
	if len(codes) != 1 {
		s.Logger.Error("could not get auth code")
		c.AbortWithStatus(500)
		return
	}

	code := codes[0]

	shortLivedToken, err := s.getInstagramShortLivedToken(code)
	if err != nil {
		s.Logger.Error("Error getting short lived token:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	longLivedToken, err := s.getInstagramLongLivedToken(shortLivedToken.AccessToken)
	if err != nil {
		s.Logger.Error("Error getting long lived token:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	user, err := s.getInstagramUser(longLivedToken.AccessToken)
	if err != nil {
		s.Logger.Error("Error getting user:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	credential := InstagramCredential{
		AccessToken: longLivedToken.AccessToken,
		UserID:      shortLivedToken.UserID,
		UserName:    user.UserName,
	}

	key := fmt.Sprintf("instagram_credential_%s", credential.UserName)

	value, err := json.Marshal(credential)
	if err != nil {
		s.Logger.Error("Error marshalling credential:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	err = s.RedisClient.Set(c.Request.Context(), key, value, 0).Err()
	if err != nil {
		s.Logger.Error("Error saving token:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	// Could be exposing here but this is just a demo.
	feedUrlStr := fmt.Sprintf("/instagram/feed/%s", credential.UserName)
	feedURL, _ := url.Parse(feedUrlStr)
	q := url.Values{}
	q.Set("api_key", s.ApiKey)
	feedURL.RawQuery = q.Encode()

	c.Redirect(302, feedURL.String())
}

type ShortLivedToken struct {
	AccessToken string `json:"access_token"`
	UserID      int    `json:"user_id"`
}

func (s *Service) getInstagramShortLivedToken(code string) (*ShortLivedToken, error) {
	tokenUrl, _ := url.Parse("https://api.instagram.com/oauth/access_token")
	callbackUrl, _ := url.Parse(s.AppURL)
	callbackUrl.Path = "instagram/auth/callback"

	values := url.Values{
		"client_id":     {s.InstagramAppID},
		"client_secret": {s.InstagramSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {callbackUrl.String()},
		"code":          {code},
	}

	req, err := http.NewRequest("POST", tokenUrl.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	token := &ShortLivedToken{}
	err = json.Unmarshal(b, &token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return token, nil
}

type LongLivedToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (s *Service) getInstagramLongLivedToken(shortLivedAccessToken string) (*LongLivedToken, error) {
	longTokenUrl, _ := url.Parse("https://graph.instagram.com/access_token")
	q := url.Values{}
	q.Set("grant_type", "ig_exchange_token")
	q.Set("client_secret", s.InstagramSecret)
	q.Set("access_token", shortLivedAccessToken)
	longTokenUrl.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", longTokenUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	token := LongLivedToken{}
	err = json.Unmarshal(b, &token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}
	return &token, nil
}

type InstagramUser struct {
	UserName string `json:"username"`
}

func (s *Service) getInstagramUser(longLivedToken string) (*InstagramUser, error) {
	userUrl, _ := url.Parse("https://graph.instagram.com/v19.0/me")

	q := url.Values{}
	q.Set("fields", "username")
	q.Set("access_token", longLivedToken)
	userUrl.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", userUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	user := InstagramUser{}
	err = json.Unmarshal(b, &user)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}
	return &user, nil
}

func (s *Service) GetInstagramFeed(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		s.Logger.Warn("username is empty")
		c.AbortWithStatus(400)
		return
	}

	key := fmt.Sprintf("instagram_credential_%s", username)
	value, err := s.RedisClient.Get(c.Request.Context(), key).Result()
	if err != nil {
		s.Logger.Error("Error getting credential:", "error", err)
		c.AbortWithStatus(500)
		return
	}
	credential := InstagramCredential{}
	err = json.Unmarshal([]byte(value), &credential)
	if err != nil {
		s.Logger.Error("Error unmarshalling credential:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	media, err := s.getInstagramMedia(&credential)
	if err != nil {
		s.Logger.Error("Error getting media:", "error", err)
		c.AbortWithStatus(500)
		return
	}

	c.PureJSON(200, gin.H{
		"data": media[:12],
	})
}

type InstagramMedia struct {
	ID           string `json:"id"`
	MediaURL     string `json:"media_url"`
	Timestamp    string `json:"timestamp"`
	ThumbnailURL string `json:"thumbnail_url"`
	Caption      string `json:"caption"`
	Permalink    string `json:"permalink"`
}

func (s *Service) getInstagramMedia(credential *InstagramCredential) ([]InstagramMedia, error) {
	mediaUrlStr := fmt.Sprintf("https://graph.instagram.com/v11.0/%d/media", credential.UserID)
	mediaUrl, err := url.Parse(mediaUrlStr)
	if err != nil {
		return nil, fmt.Errorf("media url parse failed: %w", err)
	}

	q := url.Values{}
	q.Set("access_token", credential.AccessToken)
	q.Set("fields", "id,media_url,timestamp,thumbnail_url,caption,permalink")
	mediaUrl.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", mediaUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	type Response struct {
		Data []InstagramMedia `json:"data"`
	}
	res := Response{}
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return res.Data, nil
}
