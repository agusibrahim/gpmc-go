package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	// API constants
	DefaultTimeout           = 60
	Retries                  = 10
	AndroidAPIVersion        = 28
	ClientVersionCode        = 49029607
	DefaultModel             = "Pixel XL"
	DefaultMake              = "Google"
	DefaultLanguage          = "en_US"
	BackoffFactor            = 1
)

// Api handles communication with the Google Photos mobile API
type Api struct {
	authData          string
	proxy             string
	language          string
	timeout           int
	androidAPIVersion  int
	model             string
	make              string
	clientVersionCode int
	userAgent         string

	// Auth cache with mutex for thread safety
	authCache struct {
		sync.Mutex
		expiry int64
		token  string
	}
}

// New creates a new Api instance
func New(authData string, opts ...Option) (*Api, error) {
	if authData == "" {
		return nil, fmt.Errorf("auth_data is required")
	}

	api := &Api{
		authData:          authData,
		proxy:             "",
		language:          DefaultLanguage,
		timeout:           DefaultTimeout,
		androidAPIVersion:  AndroidAPIVersion,
		model:             DefaultModel,
		make:              DefaultMake,
		clientVersionCode: ClientVersionCode,
		userAgent:         fmt.Sprintf("com.google.android.apps.photos/%d (Linux; U; Android 9; %s; %s; Build/PQ2A.190205.001; Cronet/127.0.6510.5) (gzip)", ClientVersionCode, DefaultLanguage, DefaultModel),
	}

	// Apply options
	for _, opt := range opts {
		opt(api)
	}

	// Parse language from auth_data if not set
	if api.language == DefaultLanguage {
		parsedLang := parseLanguageFromAuthData(authData)
		if parsedLang != "" {
			api.language = parsedLang
		}
	}

	return api, nil
}

// Option is a functional option for configuring the Api
type Option func(*Api)

// WithProxy sets the proxy URL
func WithProxy(proxy string) Option {
	return func(a *Api) {
		a.proxy = proxy
	}
}

// WithLanguage sets the language
func WithLanguage(language string) Option {
	return func(a *Api) {
		a.language = language
	}
}

// WithTimeout sets the request timeout in seconds
func WithTimeout(timeout int) Option {
	return func(a *Api) {
		a.timeout = timeout
	}
}

// WithModel sets the device model
func WithModel(model string) Option {
	return func(a *Api) {
		a.model = model
	}
}

// WithMake sets the device make
func WithMake(make string) Option {
	return func(a *Api) {
		a.make = make
	}
}

// BearerToken returns the bearer token, auto-renewing if expired
func (a *Api) BearerToken() (string, error) {
	a.authCache.Lock()
	defer a.authCache.Unlock()

	// Check if token is still valid (5 min buffer)
	if a.authCache.expiry > time.Now().Unix()+300 {
		return a.authCache.token, nil
	}

	// Token expired or not set - get new one
	token, expiry, err := a.getAuthToken()
	if err != nil {
		return "", err
	}

	a.authCache.token = token
	a.authCache.expiry = expiry

	return token, nil
}

// newSession creates a new HTTP client with retry mechanism
func (a *Api) newSession() *http.Client {
	client := &http.Client{
		Timeout: time.Duration(a.timeout) * time.Second,
	}

	// Configure proxy if set
	if a.proxy != "" {
		proxyURL, err := url.Parse(a.proxy)
		if err == nil {
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			client.Transport = newRetryTransport(transport)
		}
	} else {
		client.Transport = newRetryTransport(http.DefaultTransport)
	}

	return client
}

// getAuthToken retrieves a new auth token from Google
func (a *Api) getAuthToken() (token string, expiry int64, err error) {
	// Parse auth_data
	authValues, err := url.ParseQuery(a.authData)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse auth_data: %w", err)
	}

	// Build auth request body (manual construction to avoid encryption)
	authData := map[string]string{
		"androidId":                authValues.Get("androidId"),
		"app":                      "com.google.android.apps.photos",
		"client_sig":               authValues.Get("client_sig"),
		"callerPkg":                "com.google.android.apps.photos",
		"callerSig":                authValues.Get("callerSig"),
		"device_country":           authValues.Get("device_country"),
		"Email":                    authValues.Get("Email"),
		"google_play_services_version": authValues.Get("google_play_services_version"),
		"lang":                     authValues.Get("lang"),
		"oauth2_foreground":        authValues.Get("oauth2_foreground"),
		"sdk_version":              authValues.Get("sdk_version"),
		"service":                  authValues.Get("service"),
		"Token":                    authValues.Get("Token"),
	}

	// Build form data
	formData := url.Values{}
	for k, v := range authData {
		formData.Set(k, v)
	}

	// Create request
	req, err := http.NewRequest("POST", BaseAuthURL, nil)
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("app", "com.google.android.apps.photos")
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("device", authData["androidId"])
	req.Header.Set("User-Agent", "GoogleAuth/1.4 (Pixel XL PQ2A.190205.001); gzip")

	req.Body = nil // Will be set by the client with form data

	// Make request
	client := a.newSession()
	resp, err := client.PostForm(BaseAuthURL, formData)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("auth request failed with status %d", resp.StatusCode)
	}

	// Parse response (key=value lines)
	return parseAuthResponse(resp)
}

// Helper function to parse auth response
func parseAuthResponse(resp *http.Response) (token string, expiry int64, err error) {
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	lines := string(body[:n])

	for _, line := range splitLines(lines) {
		if len(line) == 0 {
			continue
		}
		parts := splitN(line, "=", 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "Auth":
				token = parts[1]
			case "Expiry":
				if exp, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					expiry = exp
				}
			}
		}
	}

	if token == "" {
		return "", 0, fmt.Errorf("no auth token in response")
	}

	return token, expiry, nil
}

// Helper functions
func parseLanguageFromAuthData(authData string) string {
	values, err := url.ParseQuery(authData)
	if err != nil {
		return ""
	}
	return values.Get("lang")
}

func splitLines(s string) []string {
	// Simple split by newline
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitN(s, sep string, n int) []string {
	// Simple split function
	return []string{s} // Placeholder
}

// retryTransport implements retry logic for HTTP requests
type retryTransport struct {
	base http.RoundTripper
}

func newRetryTransport(base http.RoundTripper) *retryTransport {
	return &retryTransport{base: base}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < Retries; i++ {
		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// Retry on 502, 503, 504
		if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
			resp.Body.Close()
			time.Sleep(time.Duration(BackoffFactor*(i+1)) * time.Second)
			continue
		}

		return resp, nil
	}

	return resp, nil
}
