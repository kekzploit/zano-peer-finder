package ipinfo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type IPAPIResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	ISP         string  `json:"isp"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	Timezone    string  `json:"timezone"`
	Zip         string  `json:"zip"`
	AS          string  `json:"as"`
	Org         string  `json:"org"`
	Query       string  `json:"query"`
	CountryCode string  `json:"countryCode"`
	District    string  `json:"district"`
	Continent   string  `json:"continent"`
	Currency    string  `json:"currency"`
	Mobile      bool    `json:"mobile"`
	Proxy       bool    `json:"proxy"`
	Hosting     bool    `json:"hosting"`
}

type Service struct {
	client      *http.Client
	rateLimiter *RateLimiter
}

type RateLimiter struct {
	mu          sync.Mutex
	requests    int
	lastReset   time.Time
	maxRequests int
	window      time.Duration
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		lastReset:   time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastReset) >= rl.window {
		rl.requests = 0
		rl.lastReset = now
	}

	if rl.requests >= rl.maxRequests {
		return false
	}

	rl.requests++
	return true
}

func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		rateLimiter: NewRateLimiter(45, time.Minute), // 45 requests per minute
	}
}

func (s *Service) GetIPInfo(ip string) (*IPAPIResponse, error) {
	if !s.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	resp, err := s.client.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,continent,continentCode,country,countryCode,region,regionName,city,district,zip,lat,lon,timezone,offset,currency,isp,org,as,asname,reverse,mobile,proxy,hosting,query", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	var ipInfo IPAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		return nil, err
	}

	return &ipInfo, nil
}
