package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

func googleProvider(c *gin.Context) {
	conf := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
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

func appleProvider(c *gin.Context) {
}

func LoginExchangeProvider(c *gin.Context) {
	provider := c.Param("provider")
	switch provider {
	case "google":
		googleProvider(c)
	case "apple":
		appleProvider(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
	}
}

func Refresh(c *gin.Context) {
}

func googleRedirect(c *gin.Context) {
	conf := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_WEB_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_WEB_CLIENT_SECRET"),
		RedirectURL:  "http://localhost.automyself.com:8080/auth/callback/google",
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

func appleRedirect(c *gin.Context) {
}

func LoginWebProvider(c *gin.Context) {
	provider := c.Param("provider")
	switch provider {
	case "google":
		googleRedirect(c)
	case "apple":
		appleRedirect(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
	}
}

func googleCallback(c *gin.Context) {
	conf := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_WEB_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_WEB_CLIENT_SECRET"),
		RedirectURL:  "http://localhost.automyself.com:8080/auth/callback/google",
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
		return
	}

	tok, err := conf.Exchange(c, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code for token"})
		return
	}

	client := conf.Client(c, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user info response"})
		return
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode user info"})
		return
	}

	identity := models.Identity{
		IdentityBase: models.IdentityBase{
			Provider: "google",
			Subject:  googleData.Sub,
			Email:    googleData.Email,
		},
	}

	if err := database.DB.Where(&identity.IdentityBase).FirstOrCreate(&identity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find or create user"})
		return
	}

	if identity.UserID == uuid.Nil {
		identity.User = models.User{
			UserBase: models.UserBase{
				Username: googleData.Name,
			},
		}
		if err := database.DB.Create(&identity.User).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new user"})
			return
		}
		identity.UserID = identity.User.ID
		if err := database.DB.Save(&identity).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update identity with new user"})
			return
		}
	} else {
		database.DB.Model(&identity).Association("User").Find(&identity.User)
	}

	token, err := identity.User.GenerateJWT()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"authentication": "Bearer " + token})
}

func appleCallback(c *gin.Context) {
}

func LoginWebCallback(c *gin.Context) {
	provider := c.Param("provider")
	switch provider {
	case "google":
		googleCallback(c)
	case "apple":
		appleCallback(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
	}
}

type DevelopmentLoginRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

// DEVELOPMENT ONLY Login
func LoginDevelopment(c *gin.Context) {
	if gin.Mode() != gin.DebugMode && gin.Mode() != gin.TestMode {
		c.Status(http.StatusTeapot)
		return
	}

	var developmentLoginRequest DevelopmentLoginRequest
	if err := c.ShouldBindJSON(&developmentLoginRequest); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	user := models.User{}
	result := database.DB.First(&user, "id = ?", developmentLoginRequest.UserID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	token, err := user.GenerateJWT()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"authentication": "Bearer " + token})
}
