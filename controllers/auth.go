package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/models"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

type keyCacheItem struct {
	PubKey     string
	expiration int64
}

var keyCache = make(map[string]keyCacheItem)

func getAuthKey(provider string, kid string) (string, error) {
	cacheKey := provider + ":" + kid
	if item, found := keyCache[cacheKey]; found {
		if item.expiration > time.Now().Unix() {
			return item.PubKey, nil
		}
		delete(keyCache, cacheKey) // Remove expired key
	}

	var key keyCacheItem
	if provider == "google" {

	} else if provider == "apple" {

	}

	keyCache[cacheKey] = key
	return key.PubKey, nil

}

func googleProvider(c *gin.Context, a *app.App) {
	client_id, err := a.Secrets.Get("GOOGLE_OAUTH2_CLIENT_ID")
	if err != nil {
		log.Fatal(err)
	}

	client_secret, err := a.Secrets.Get("GOOGLE_OAUTH2_CLIENT_SECRET")
	if err != nil {
		log.Fatal(err)
	}
	conf := &oauth2.Config{
		ClientID:     client_id.Value,
		ClientSecret: client_secret.Value,
		RedirectURL:  "http://localhost.automyself.com:8080/auth/google",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}
	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL("state")
	fmt.Printf("Visit the URL for the auth dialog: %v", url)

	// Handle the exchange code to initiate a transport.
	tok, err := conf.Exchange(c, "authorization-code")
	if err != nil {
		log.Fatal(err)
	}

	client := conf.Client(c, tok)
	client.Get("...")
}

func appleProvider(c *gin.Context, a *app.App) {
}

func LoginExchangeProvider(c *gin.Context, a *app.App) {
	provider := c.Param("provider")
	switch provider {
	case "google":
		googleProvider(c, a)
	case "apple":
		appleProvider(c, a)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
	}
}

func Refresh(c *gin.Context, a *app.App) {

	fmt.Printf("Scheme: %s, Host: %s\n", c.Request.URL.Scheme, c.Request.URL.Host)

	var userProvidedToken = struct {
		RefreshToken string `json:"refresh_token"`
	}{}

	if err := c.ShouldBindJSON(&userProvidedToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Required key: `refresh_token`"})
		return
	}

	fmt.Println(userProvidedToken.RefreshToken)

	hash := sha256.Sum256([]byte(userProvidedToken.RefreshToken))
	hashedUserToken := fmt.Sprintf("%x", hash[:])

	var refreshToken models.RefreshToken
	if err := a.Gorm.First(&refreshToken, "token = ?", hashedUserToken).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	a.Gorm.Model(&refreshToken).Association("User").Find(&refreshToken.User)

	if err := a.Gorm.Delete(&refreshToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete refresh token"})
		return
	}

	jwt, err := refreshToken.User.GenerateJWT(a)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	refresh, err := refreshToken.User.GenerateRefreshToken(a.Gorm)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"jwt": jwt, "refresh": refresh})

}

func googleRedirect(c *gin.Context, a *app.App) {
	domain, err := a.Secrets.Get("DOMAIN")
	if err != nil {
		log.Fatal(err)
	}
	client_id, err := a.Secrets.Get("GOOGLE_OAUTH2_CLIENT_ID")
	if err != nil {
		log.Fatal(err)
	}
	client_secret, err := a.Secrets.Get("GOOGLE_OAUTH2_CLIENT_SECRET")
	if err != nil {
		log.Fatal(err)
	}
	conf := &oauth2.Config{
		ClientID:     client_id.Value,
		ClientSecret: client_secret.Value,
		RedirectURL:  domain.Value + "/auth/callback/google",
		// RedirectURL: c.Request.URL.Scheme + "://" + c.Request.URL.Host + "/auth/callback/google",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func appleRedirect(c *gin.Context, a *app.App) {
}

func LoginWebProvider(c *gin.Context, a *app.App) {
	provider := c.Param("provider")
	switch provider {
	case "google":
		googleRedirect(c, a)
	case "apple":
		appleRedirect(c, a)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
	}
}

func googleCallback(c *gin.Context, a *app.App) (models.User, error) {
	client_id, err := a.Secrets.Get("GOOGLE_OAUTH2_CLIENT_ID")
	if err != nil {
		log.Fatal(err)
	}
	client_secret, err := a.Secrets.Get("GOOGLE_OAUTH2_CLIENT_SECRET")
	if err != nil {
		log.Fatal(err)
	}
	domain, err := a.Secrets.Get("DOMAIN")
	if err != nil {
		log.Fatal(err)
	}

	conf := &oauth2.Config{
		ClientID:     client_id.Value,
		ClientSecret: client_secret.Value,
		RedirectURL:  domain.Value + "/auth/callback/google",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code not found in query parameters"})
		return models.User{}, errors.New("Code not found in query parameters")
	}

	tok, err := conf.Exchange(c, code)
	if err != nil {
		return models.User{}, errors.New("Failed to exchange code for token")
	}

	client := conf.Client(c, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return models.User{}, errors.New("Failed to get user info")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.User{}, errors.New("Failed to read user info response")
	}
	log.Printf("Google user info response: %s", string(body))

	type googleUserInfo struct {
		Sub           string `json:"sub"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	googleData := googleUserInfo{}
	if err := json.Unmarshal(body, &googleData); err != nil {
		return models.User{}, errors.New("Failed to decode user info")
	}

	identity, err := gorm.G[models.Identity](a.Gorm).Where("provider = ? AND subject = ?", "google", googleData.Sub).First(c)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		identity = models.Identity{
			IdentityBase: models.IdentityBase{
				Provider: "google",
				Subject:  googleData.Sub,
				Email:    googleData.Email,
			},
			User: models.User{
				UserBase: models.UserBase{
					Username: googleData.Name,
				},
			},
		}

		err = gorm.G[models.Identity](a.Gorm).Create(c, &identity)
		if err != nil {
			return models.User{}, err
		}
	} else {
		a.Gorm.Model(&identity).Association("User").Find(&identity.User)
	}

	return identity.User, nil
}

func appleCallback(c *gin.Context, a *app.App) {
}

func LoginWebCallback(c *gin.Context, a *app.App) {
	provider := c.Param("provider")
	var user models.User
	var err error

	switch provider {
	case "google":
		user, err = googleCallback(c, a)
	case "apple":
		appleCallback(c, a)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
	}

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	jwt, err := user.GenerateJWT(a)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	refresh, err := user.GenerateRefreshToken(a.Gorm)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"jwt": jwt, "refresh": refresh})
}

type DevelopmentLoginRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

// DEVELOPMENT ONLY Login
func LoginDevelopment(c *gin.Context, a *app.App) {
	if gin.Mode() != gin.DebugMode && gin.Mode() != gin.TestMode {
		c.Status(http.StatusTeapot)
		return
	}

	var developmentLoginRequest DevelopmentLoginRequest
	if err := c.ShouldBindJSON(&developmentLoginRequest); err != nil {
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	user, err := gorm.G[models.User](a.Gorm).Where("id = ?", developmentLoginRequest.UserID).First(c)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithError(http.StatusNotFound, err)
		return
	} else if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	token, err := user.GenerateJWT(a)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"authentication": "Bearer " + token})
}
