package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type userConfig struct {
	EmailPrefix string `json:"emailPrefix"`
	EmailDomain string `json:"emailDomain"`
	NamePrefix  string `json:"namePrefix"`
	Password    string `json:"password"`
	Count       int    `json:"count"`
}

type sessionRecord struct {
	UserNumber   int    `json:"userNumber"`
	Email        string `json:"email"`
	CookieHeader string `json:"cookieHeader"`
}

type sessionFile struct {
	BaseURL     string          `json:"baseURL"`
	GeneratedAt string          `json:"generatedAt"`
	UserOffset  int             `json:"userOffset"`
	UserCount   int             `json:"userCount"`
	Sessions    []sessionRecord `json:"sessions"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := envString("USER_CONFIG", "tests/k6/users.json")
	sessionPath := envString("SESSION_FILE", "tests/k6/sessions.json")
	baseURL := strings.TrimRight(envString("BASE_URL", "http://localhost:8080"), "/")
	userOffset := envInt("USER_OFFSET", 0)
	loginConcurrency := envInt("LOGIN_CONCURRENCY", 50)

	if loginConcurrency <= 0 {
		return errors.New("LOGIN_CONCURRENCY must be greater than 0")
	}

	config, err := readUserConfig(configPath)
	if err != nil {
		return err
	}

	userCount := envInt("USER_COUNT", config.Count)
	if userCount <= 0 {
		return errors.New("USER_COUNT must be greater than 0")
	}

	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return fmt.Errorf("invalid BASE_URL %q: %w", baseURL, err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	if err := checkHealth(client, baseURL); err != nil {
		return err
	}

	sessions, err := loginSessions(client, baseURL, config, userOffset, userCount, loginConcurrency)
	if err != nil {
		return err
	}

	output := sessionFile{
		BaseURL:     baseURL,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		UserOffset:  userOffset,
		UserCount:   userCount,
		Sessions:    sessions,
	}

	payload, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(sessionPath, payload, 0o644); err != nil {
		return fmt.Errorf("write sessions file: %w", err)
	}

	fmt.Printf("Wrote %d sessions to %s\n", len(sessions), sessionPath)
	return nil
}

func readUserConfig(path string) (userConfig, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return userConfig{}, fmt.Errorf("read user config: %w", err)
	}

	var config userConfig
	if err := json.Unmarshal(payload, &config); err != nil {
		return userConfig{}, fmt.Errorf("parse user config: %w", err)
	}

	return config, nil
}

func checkHealth(client *http.Client, baseURL string) error {
	response, err := client.Get(baseURL + "/healthz")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", response.StatusCode)
	}

	return nil
}

func loginSessions(client *http.Client, baseURL string, config userConfig, userOffset, userCount, concurrency int) ([]sessionRecord, error) {
	sessions := make([]sessionRecord, userCount)
	jobs := make(chan int)
	errs := make(chan error, 1)

	var wg sync.WaitGroup
	for workerIndex := 0; workerIndex < concurrency; workerIndex++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				userNumber := userOffset + index + 1
				email := fmt.Sprintf("%s-%d@%s", config.EmailPrefix, userNumber, config.EmailDomain)
				cookieHeader, err := loginUser(client, baseURL, email, config.Password)
				if err != nil {
					select {
					case errs <- err:
					default:
					}
					return
				}

				sessions[index] = sessionRecord{
					UserNumber:   userNumber,
					Email:        email,
					CookieHeader: cookieHeader,
				}
			}
		}()
	}

	for index := 0; index < userCount; index++ {
		select {
		case err := <-errs:
			close(jobs)
			wg.Wait()
			return nil, err
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()

	select {
	case err := <-errs:
		return nil, err
	default:
		return sessions, nil
	}
}

func loginUser(client *http.Client, baseURL, email, password string) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return "", err
	}

	response, err := client.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("login %s failed: %w", email, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("login %s failed with status %d: %s", email, response.StatusCode, strings.TrimSpace(string(body)))
	}

	for _, cookie := range response.Cookies() {
		if cookie.Name == "buy_ticket_session" && cookie.Value != "" {
			return cookie.Name + "=" + cookie.Value, nil
		}
	}

	return "", fmt.Errorf("login %s did not return buy_ticket_session", email)
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
