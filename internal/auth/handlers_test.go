package auth_test

import (
	"auto-myself-api/internal/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authResponse struct {
	JWT     string `json:"jwt"`
	Refresh string `json:"refresh"`
}

func developmentLogin(t *testing.T, env *testutil.Env, userID string) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/development", "",
		testutil.JSONBody(t, map[string]string{"user_id": userID}))
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

func TestOAuthInitiate(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/auth/web/github", "", nil)
	testutil.RequireStatus(t, w, http.StatusBadRequest)

	if os.Getenv("GOOGLE_OAUTH2_CLIENT_ID") == "" || os.Getenv("DOMAIN") == "" {
		t.Skip("Google OAuth environment is not configured")
	}

	w = testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/auth/web/google", "", nil)
	testutil.RequireStatus(t, w, http.StatusTemporaryRedirect)

	location := w.Header().Get("Location")
	if !strings.Contains(location, "accounts.google.com") {
		t.Errorf("expected a Google redirect, got %q", location)
	}

	if cookies := w.Result().Cookies(); len(cookies) != 0 {
		t.Errorf("expected no cookies from the OAuth initiate endpoint, got %v", cookies)
	}

	// Missing code is rejected before any provider exchange.
	w = testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/auth/web/callback/google", "", nil)
	testutil.RequireStatus(t, w, http.StatusBadRequest)
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
