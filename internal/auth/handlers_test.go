package auth_test

import (
	"auto-myself-api/internal/auth"
	"auto-myself-api/internal/testutil"
	"auto-myself-api/internal/users"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type authResponse struct {
	JWT     string `json:"jwt"`
	Refresh string `json:"refresh"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type mobileCredentials struct {
	IDToken string `json:"id_token"`
	Nonce   string `json:"nonce"`
	Name    string `json:"name,omitempty"`
}

func developmentLogin(t *testing.T, env *testutil.Env, userID string) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/development", "",
		testutil.JSONBody(t, map[string]string{"user_id": userID}))
}

func mobileLogin(t *testing.T, env *testutil.Env, provider string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/mobile/"+provider, "",
		testutil.JSONBody(t, body))
}

func mobileLoginRaw(t *testing.T, env *testutil.Env, provider, body string) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/mobile/"+provider, "",
		strings.NewReader(body))
}

func jwtSubject(t *testing.T, rawToken string) string {
	t.Helper()

	secret := []byte(os.Getenv("JWT_SIGNING_SECRET"))
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithAudience("auto-myself-api"),
		jwt.WithIssuer("auto-myself-auth"),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		t.Fatalf("parsing API JWT: %v", err)
	}
	return claims.Subject
}

func findUserByID(t *testing.T, env *testutil.Env, id string) users.User {
	t.Helper()

	user, err := gorm.G[users.User](env.DB.GetGorm()).Where("id = ?", testutil.UUID(id)).First(context.Background())
	if err != nil {
		t.Fatalf("finding user %s: %v", id, err)
	}
	return user
}

func findIdentity(t *testing.T, env *testutil.Env, provider, subject string) auth.Identity {
	t.Helper()

	identity, err := gorm.G[auth.Identity](env.DB.GetGorm()).
		Where("provider = ? AND subject = ?", provider, subject).
		First(context.Background())
	if err != nil {
		t.Fatalf("finding %s identity: %v", provider, err)
	}
	return identity
}

func countIdentities(t *testing.T, env *testutil.Env, provider, subject string) int64 {
	t.Helper()

	var count int64
	if err := env.DB.GetGorm().Model(&auth.Identity{}).
		Where("provider = ? AND subject = ?", provider, subject).
		Count(&count).Error; err != nil {
		t.Fatalf("counting identities: %v", err)
	}
	return count
}

func TestDevelopmentLogin(t *testing.T) {
	env := testutil.SetupTest(t)

	w := developmentLogin(t, env, testutil.User1ID.String())
	testutil.RequireStatus(t, w, http.StatusOK)

	var response struct {
		Authentication string `json:"authentication"`
		Refresh        string `json:"refresh"`
	}
	testutil.DecodeJSON(t, w, &response)

	if !strings.HasPrefix(response.Authentication, "Bearer ") {
		t.Fatalf("expected an authentication bearer, got %q", response.Authentication)
	}
	if response.Refresh == "" {
		t.Fatal("expected a refresh token")
	}

	secret := []byte(os.Getenv("JWT_SIGNING_SECRET"))
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		strings.TrimPrefix(response.Authentication, "Bearer "),
		claims,
		func(token *jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithAudience("auto-myself-api"),
		jwt.WithIssuer("auto-myself-auth"),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		t.Fatalf("development login returned an invalid JWT: %v", err)
	}
	if claims.Subject != testutil.User1ID.String() {
		t.Errorf("expected subject %s, got %s", testutil.User1ID, claims.Subject)
	}
	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 0 {
		t.Errorf("expected a future expiry, got %v", claims.ExpiresAt)
	}

	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}
}

func TestDevelopmentLoginUnknownUser(t *testing.T) {
	env := testutil.SetupTest(t)

	w := developmentLogin(t, env, "019785fe-4eb4-766e-9c45-ffffffffffff")
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestDevelopmentLoginBadBody(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/development", "", strings.NewReader("{invalid"))
	testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)

	w = developmentLogin(t, env, "not-a-uuid")
	testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
}

func TestDevelopmentLoginDisabledOutsideDebug(t *testing.T) {
	env := testutil.SetupTest(t)

	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)

	w := developmentLogin(t, env, testutil.User1ID.String())
	testutil.RequireStatus(t, w, http.StatusTeapot)
}

func TestRefreshTokenRotation(t *testing.T) {
	env := testutil.SetupTest(t)

	w := developmentLogin(t, env, testutil.User1ID.String())
	testutil.RequireStatus(t, w, http.StatusOK)

	var login struct {
		Refresh string `json:"refresh"`
	}
	testutil.DecodeJSON(t, w, &login)

	refresh := func(token string) *httptest.ResponseRecorder {
		return testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/refresh", "",
			testutil.JSONBody(t, map[string]string{"refresh_token": token}))
	}

	w = refresh(login.Refresh)
	testutil.RequireStatus(t, w, http.StatusOK)

	var rotated authResponse
	testutil.DecodeJSON(t, w, &rotated)
	if rotated.JWT == "" {
		t.Error("expected a new JWT")
	}
	if rotated.Refresh == "" || rotated.Refresh == login.Refresh {
		t.Error("expected a new refresh token")
	}

	// Replaying the consumed token must fail.
	w = refresh(login.Refresh)
	testutil.RequireStatus(t, w, http.StatusUnauthorized)

	// The rotated token is usable.
	w = refresh(rotated.Refresh)
	testutil.RequireStatus(t, w, http.StatusOK)

	// A random token is rejected.
	w = refresh("0123456789abcdef")
	testutil.RequireStatus(t, w, http.StatusUnauthorized)

	// A malformed body is a bad request.
	w = testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/refresh", "", strings.NewReader("{invalid"))
	testutil.RequireStatus(t, w, http.StatusBadRequest)
}

func signToken(t *testing.T, secret string, claims jwt.RegisteredClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}
	return signed
}

func TestAuthMiddlewareRejectsInvalidTokens(t *testing.T) {
	env := testutil.SetupTest(t)

	secret := os.Getenv("JWT_SIGNING_SECRET")
	now := time.Now()

	baseClaims := func() jwt.RegisteredClaims {
		return jwt.RegisteredClaims{
			Issuer:    "auto-myself-auth",
			Audience:  jwt.ClaimStrings{"auto-myself-api"},
			Subject:   testutil.User1ID.String(),
			IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		}
	}

	expiredClaims := baseClaims()
	expiredClaims.IssuedAt = jwt.NewNumericDate(now.Add(-2 * time.Hour))
	expiredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Hour))

	unknownUserClaims := baseClaims()
	unknownUserClaims.Subject = "019785fe-4eb4-766e-9c45-ffffffffffff"

	tests := []struct {
		name   string
		bearer string
	}{
		{"missing", ""},
		{"garbage", "Bearer not-a-token"},
		{"expired", "Bearer " + signToken(t, secret, expiredClaims)},
		{"bad signature", "Bearer " + signToken(t, "a-different-secret", baseClaims())},
		{"unknown user", "Bearer " + signToken(t, secret, unknownUserClaims)},
		{"wrong audience", "Bearer " + signToken(t, secret, func() jwt.RegisteredClaims {
			claims := baseClaims()
			claims.Audience = jwt.ClaimStrings{"someone-else"}
			return claims
		}())},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/v1/user", test.bearer, nil)
			testutil.RequireStatus(t, w, http.StatusUnauthorized)
		})
	}
}

func TestRefreshTokenIsStoredAsBcrypt(t *testing.T) {
	env := testutil.SetupTest(t)

	w := developmentLogin(t, env, testutil.User1ID.String())
	testutil.RequireStatus(t, w, http.StatusOK)

	var login struct {
		Refresh string `json:"refresh"`
	}
	testutil.DecodeJSON(t, w, &login)

	idPart, verifier, found := strings.Cut(login.Refresh, ".")
	if !found || idPart == "" || verifier == "" {
		t.Fatalf("expected refresh token in selector.verifier form, got %q", login.Refresh)
	}

	var storedToken string
	if err := env.DB.GetGorm().Table("refresh_tokens").
		Select("token").
		Where("id = ?", idPart).
		Row().Scan(&storedToken); err != nil {
		t.Fatalf("reading stored refresh token: %v", err)
	}

	if storedToken == login.Refresh || storedToken == verifier {
		t.Fatal("expected the raw refresh token to be absent from storage")
	}
	if !strings.HasPrefix(storedToken, "$2") {
		t.Errorf("expected a bcrypt hash in storage, got %q", storedToken)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedToken), []byte(verifier)); err != nil {
		t.Errorf("stored bcrypt hash does not match the verifier: %v", err)
	}
}

func TestMobileLoginGoogleCreatesUser(t *testing.T) {
	env := testutil.SetupTest(t)

	env.GoogleAuth.Register("google-token-1", auth.VerifiedIdentity{
		Subject: "google-subject-1",
		Email:   "alice@example.com",
		Name:    "Alice Google",
	})

	w := mobileLogin(t, env, "google", mobileCredentials{IDToken: "google-token-1", Nonce: "nonce-1"})
	testutil.RequireStatus(t, w, http.StatusOK)

	var response authResponse
	testutil.DecodeJSON(t, w, &response)
	if response.JWT == "" {
		t.Fatal("expected an API JWT")
	}
	if response.Refresh == "" {
		t.Fatal("expected a refresh token")
	}

	user := findUserByID(t, env, jwtSubject(t, response.JWT))
	if user.Username != "Alice Google" {
		t.Errorf("expected username from the token name, got %q", user.Username)
	}

	identity := findIdentity(t, env, "google", "google-subject-1")
	if identity.UserID != user.ID {
		t.Errorf("expected identity user %s, got %s", user.ID, identity.UserID)
	}
	if identity.Email != "alice@example.com" {
		t.Errorf("expected identity email, got %q", identity.Email)
	}

	if got := env.GoogleAuth.LastNonce(); got != "nonce-1" {
		t.Errorf("expected the nonce to reach the verifier, got %q", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", got)
	}
}

func TestMobileLoginGoogleExistingIdentity(t *testing.T) {
	env := testutil.SetupTest(t)

	env.GoogleAuth.Register("google-token-1", auth.VerifiedIdentity{
		Subject: "google-subject-1",
		Email:   "alice@example.com",
		Name:    "Alice Google",
	})
	env.GoogleAuth.Register("google-token-2", auth.VerifiedIdentity{
		Subject: "google-subject-1",
		Email:   "alice@example.com",
		Name:    "Alice Google",
	})

	first := mobileLogin(t, env, "google", mobileCredentials{IDToken: "google-token-1", Nonce: "nonce-1"})
	testutil.RequireStatus(t, first, http.StatusOK)
	var firstResponse authResponse
	testutil.DecodeJSON(t, first, &firstResponse)

	second := mobileLogin(t, env, "google", mobileCredentials{IDToken: "google-token-2", Nonce: "nonce-2"})
	testutil.RequireStatus(t, second, http.StatusOK)
	var secondResponse authResponse
	testutil.DecodeJSON(t, second, &secondResponse)

	firstSubject := jwtSubject(t, firstResponse.JWT)
	secondSubject := jwtSubject(t, secondResponse.JWT)
	if firstSubject != secondSubject {
		t.Errorf("expected the same user for repeated logins, got %s and %s", firstSubject, secondSubject)
	}
	if count := countIdentities(t, env, "google", "google-subject-1"); count != 1 {
		t.Errorf("expected one identity, got %d", count)
	}
}

func TestMobileLoginUsernameFallbacks(t *testing.T) {
	env := testutil.SetupTest(t)

	tests := []struct {
		name      string
		token     string
		subject   string
		tokenName string
		email     string
		expected  string
	}{
		{"token name", "token-name", "subject-name", "Token Name", "user@example.com", "Token Name"},
		{"email local part", "token-email", "subject-email", "", "bob@example.com", "bob"},
		{"default", "token-default", "subject-default", "", "", "User"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env.GoogleAuth.Register(test.token, auth.VerifiedIdentity{
				Subject: test.subject,
				Email:   test.email,
				Name:    test.tokenName,
			})

			w := mobileLogin(t, env, "google", mobileCredentials{IDToken: test.token, Nonce: "nonce-" + test.token})
			testutil.RequireStatus(t, w, http.StatusOK)

			var response authResponse
			testutil.DecodeJSON(t, w, &response)
			user := findUserByID(t, env, jwtSubject(t, response.JWT))
			if user.Username != test.expected {
				t.Errorf("expected username %q, got %q", test.expected, user.Username)
			}
		})
	}
}

func TestMobileLoginAppleUsesRequestedName(t *testing.T) {
	env := testutil.SetupTest(t)

	env.AppleAuth.Register("apple-token-1", auth.VerifiedIdentity{
		Subject: "apple-subject-1",
		Email:   "relay@privaterelay.appleid.com",
	})

	w := mobileLogin(t, env, "apple", mobileCredentials{
		IDToken: "apple-token-1",
		Nonce:   "nonce-apple",
		Name:    "Apple Alice",
	})
	testutil.RequireStatus(t, w, http.StatusOK)

	var response authResponse
	testutil.DecodeJSON(t, w, &response)
	user := findUserByID(t, env, jwtSubject(t, response.JWT))
	if user.Username != "Apple Alice" {
		t.Errorf("expected the requested Apple name, got %q", user.Username)
	}

	identity := findIdentity(t, env, "apple", "apple-subject-1")
	if identity.Email != "relay@privaterelay.appleid.com" {
		t.Errorf("expected the Apple email, got %q", identity.Email)
	}
}

func TestMobileLoginApplePrefersTokenName(t *testing.T) {
	env := testutil.SetupTest(t)

	env.AppleAuth.Register("apple-token-1", auth.VerifiedIdentity{
		Subject: "apple-subject-1",
		Email:   "relay@privaterelay.appleid.com",
		Name:    "Token Name",
	})

	w := mobileLogin(t, env, "apple", mobileCredentials{
		IDToken: "apple-token-1",
		Nonce:   "nonce-apple",
		Name:    "Request Name",
	})
	testutil.RequireStatus(t, w, http.StatusOK)

	var response authResponse
	testutil.DecodeJSON(t, w, &response)
	user := findUserByID(t, env, jwtSubject(t, response.JWT))
	if user.Username != "Token Name" {
		t.Errorf("expected the token name to win, got %q", user.Username)
	}
}

func TestMobileLoginGoogleIgnoresRequestName(t *testing.T) {
	env := testutil.SetupTest(t)

	env.GoogleAuth.Register("google-token-1", auth.VerifiedIdentity{
		Subject: "google-subject-1",
		Email:   "google@example.com",
	})

	w := mobileLogin(t, env, "google", mobileCredentials{
		IDToken: "google-token-1",
		Nonce:   "nonce-google",
		Name:    "Request Name",
	})
	testutil.RequireStatus(t, w, http.StatusOK)

	var response authResponse
	testutil.DecodeJSON(t, w, &response)
	user := findUserByID(t, env, jwtSubject(t, response.JWT))
	if user.Username != "google" {
		t.Errorf("expected Google to ignore the request name and use the email local part, got %q", user.Username)
	}
}

func TestMobileLoginInvalidToken(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, provider := range []string{"google", "apple"} {
		t.Run(provider, func(t *testing.T) {
			w := mobileLogin(t, env, provider, mobileCredentials{IDToken: "unknown-token", Nonce: "nonce"})
			testutil.RequireStatus(t, w, http.StatusUnauthorized)

			var response errorResponse
			testutil.DecodeJSON(t, w, &response)
			if response.Error != "Invalid identity token" {
				t.Errorf("expected invalid-token error, got %q", response.Error)
			}
		})
	}
}

func TestMobileLoginBadBody(t *testing.T) {
	env := testutil.SetupTest(t)

	tests := []struct {
		name string
		body string
	}{
		{"missing id_token", `{"nonce":"nonce"}`},
		{"missing nonce", `{"id_token":"token"}`},
		{"malformed json", `{invalid`},
		{"empty body", ``},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := mobileLoginRaw(t, env, "google", test.body)
			testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
		})
	}
}

func TestMobileLoginUnsupportedProvider(t *testing.T) {
	env := testutil.SetupTest(t)

	w := mobileLogin(t, env, "github", mobileCredentials{IDToken: "token", Nonce: "nonce"})
	testutil.RequireStatus(t, w, http.StatusBadRequest)

	var response errorResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Error != "Unsupported provider" {
		t.Errorf("expected unsupported-provider error, got %q", response.Error)
	}

	// The provider check happens before body parsing.
	w = mobileLoginRaw(t, env, "github", "{invalid")
	testutil.RequireStatus(t, w, http.StatusBadRequest)
}

func TestMobileLoginProviderSeparation(t *testing.T) {
	env := testutil.SetupTest(t)

	env.AppleAuth.Register("apple-token-1", auth.VerifiedIdentity{Subject: "apple-subject-1"})
	w := mobileLogin(t, env, "google", mobileCredentials{IDToken: "apple-token-1", Nonce: "nonce"})
	testutil.RequireStatus(t, w, http.StatusUnauthorized)

	env.GoogleAuth.Register("google-token-1", auth.VerifiedIdentity{Subject: "google-subject-1"})
	w = mobileLogin(t, env, "apple", mobileCredentials{IDToken: "google-token-1", Nonce: "nonce"})
	testutil.RequireStatus(t, w, http.StatusUnauthorized)
}

func TestMobileLoginVerifierError(t *testing.T) {
	env := testutil.SetupTest(t)

	env.GoogleAuth.SetError(errors.New("jwks unavailable"))

	w := mobileLogin(t, env, "google", mobileCredentials{IDToken: "token", Nonce: "nonce"})
	testutil.RequireStatus(t, w, http.StatusInternalServerError)
}

func TestRemovedWebOAuthRoutes(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, path := range []string{"/auth/web/google", "/auth/web/callback/google"} {
		w := testutil.RequestWithBearer(t, env.Router, http.MethodGet, path, "", nil)
		testutil.RequireStatus(t, w, http.StatusNotFound)
	}
}
