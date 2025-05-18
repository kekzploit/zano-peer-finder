package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"zano-peer-finder/internal/database"
	"zano-peer-finder/internal/ipinfo"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	rate       float64 // tokens per second
	bucketSize float64 // maximum bucket size
	tokens     float64 // current tokens in bucket
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the specified rate and bucket size
func NewRateLimiter(rate float64, bucketSize float64) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		bucketSize: bucketSize,
		tokens:     bucketSize,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens = min(rl.bucketSize, rl.tokens+elapsed*rl.rate)
	rl.lastRefill = now

	if rl.tokens < 1 {
		// Calculate how long to wait for a token
		waitTime := time.Duration((1 - rl.tokens) / rl.rate * float64(time.Second))
		rl.mu.Unlock()
		time.Sleep(waitTime)
		rl.mu.Lock()
		// Update tokens after waiting
		now = time.Now()
		elapsed = now.Sub(rl.lastRefill).Seconds()
		rl.tokens = min(rl.bucketSize, rl.tokens+elapsed*rl.rate)
		rl.lastRefill = now
	}

	rl.tokens--
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func init() {
	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Pretty console logging
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})
}

type NodeInfo struct {
	IP          string    `json:"ip"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	ISP         string    `json:"isp"`
	LastSeen    time.Time `json:"lastSeen"`
	IsNew       bool      `json:"isNew"`
	Region      string    `json:"region"`
	RegionName  string    `json:"regionName"`
	Timezone    string    `json:"timezone"`
	Zip         string    `json:"zip"`
	AS          string    `json:"as"`
	Org         string    `json:"org"`
	Query       string    `json:"query"`
	Status      string    `json:"status"`
	CountryCode string    `json:"countryCode"`
	District    string    `json:"district"`
	Continent   string    `json:"continent"`
	Currency    string    `json:"currency"`
	Mobile      bool      `json:"mobile"`
	Proxy       bool      `json:"proxy"`
	Hosting     bool      `json:"hosting"`
	IsOnline    bool      `json:"isOnline"`
	LastPing    time.Time `json:"lastPing"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.RWMutex
)

// Custom split function that handles both \n and \r\n
func customSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0:i], nil
	}

	if i := bytes.Index(data, []byte("\r\n")); i >= 0 {
		return i + 2, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

// Function to broadcast node updates to all connected clients
func broadcastNodeUpdate(node *NodeInfo) {
	clientsMux.RLock()
	defer clientsMux.RUnlock()

	log.Debug().
		Str("ip", node.IP).
		Int("clientCount", len(clients)).
		Msg("Broadcasting node update")

	for client := range clients {
		err := client.WriteJSON(node)
		if err != nil {
			log.Error().Err(err).Msg("Error broadcasting to client")
			client.Close()
			delete(clients, client)
		}
	}
}

// WebSocket handler
func handleWebSocket(w http.ResponseWriter, r *http.Request, db *database.DB) {
	log.Info().Msg("New WebSocket connection request")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error upgrading to WebSocket")
		return
	}

	clientsMux.Lock()
	clients[conn] = true
	clientsMux.Unlock()
	log.Info().Msg("New WebSocket client connected")

	// Send all existing nodes to the new client as a single array
	nodes, err := db.GetAllNodes()
	if err != nil {
		log.Error().Err(err).Msg("Error getting nodes from database")
		return
	}
	log.Info().Int("nodeCount", len(nodes)).Msg("Retrieved nodes from database")

	// Convert nodes to NodeInfo array
	nodeInfos := make([]*NodeInfo, len(nodes))
	for i, node := range nodes {
		log.Debug().
			Str("ip", node.IP).
			Str("country", node.Country).
			Bool("isOnline", node.IsOnline).
			Time("lastSeen", node.LastSeen).
			Msg("Converting node to NodeInfo")

		nodeInfos[i] = &NodeInfo{
			IP:          node.IP,
			Country:     node.Country,
			City:        node.City,
			Lat:         node.Lat,
			Lon:         node.Lon,
			ISP:         node.ISP,
			LastSeen:    node.LastSeen,
			IsNew:       false,
			Region:      node.Region,
			RegionName:  node.RegionName,
			Timezone:    node.Timezone,
			Zip:         node.Zip,
			AS:          node.AS,
			Org:         node.Org,
			Query:       node.Query,
			Status:      node.Status,
			CountryCode: node.CountryCode,
			District:    node.District,
			Continent:   node.Continent,
			Currency:    node.Currency,
			Mobile:      node.Mobile,
			Proxy:       node.Proxy,
			Hosting:     node.Hosting,
			IsOnline:    node.IsOnline,
			LastPing:    node.LastPing,
		}
	}

	// Send the array as a single message
	if err := conn.WriteJSON(nodeInfos); err != nil {
		log.Error().Err(err).Msg("Error sending initial node list to client")
		return
	}
	log.Info().Int("nodeCount", len(nodeInfos)).Msg("Sent initial node list to client")

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Info().Err(err).Msg("WebSocket client disconnected")
			clientsMux.Lock()
			delete(clients, conn)
			clientsMux.Unlock()
			conn.Close()
			break
		}

		// Parse the message
		var msg struct {
			Type string `json:"type"`
			Data struct {
				IP       string    `json:"ip"`
				IsOnline bool      `json:"isOnline"`
				LastPing time.Time `json:"lastPing"`
				Latency  int64     `json:"latency"`
			} `json:"data"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			log.Error().Err(err).Msg("Error parsing WebSocket message")
			continue
		}

		// Handle status updates
		if msg.Type == "status_update" {
			log.Info().
				Str("ip", msg.Data.IP).
				Bool("isOnline", msg.Data.IsOnline).
				Time("lastPing", msg.Data.LastPing).
				Int64("latency", msg.Data.Latency).
				Msg("Received status update from client")

			// Update the node status in the database
			if err := db.UpdateNodeStatus(msg.Data.IP, msg.Data.IsOnline); err != nil {
				log.Error().Err(err).Str("ip", msg.Data.IP).Msg("Error updating node status")
				continue
			}

			// Get the full node information from the database
			node, err := db.GetNode(msg.Data.IP)
			if err != nil {
				log.Error().Err(err).Str("ip", msg.Data.IP).Msg("Error getting node information for broadcast")
				continue
			}

			// Broadcast the update to all clients
			nodeInfo := &NodeInfo{
				IP:          node.IP,
				Country:     node.Country,
				City:        node.City,
				Lat:         node.Lat,
				Lon:         node.Lon,
				ISP:         node.ISP,
				LastSeen:    node.LastSeen,
				IsNew:       false,
				Region:      node.Region,
				RegionName:  node.RegionName,
				Timezone:    node.Timezone,
				Zip:         node.Zip,
				AS:          node.AS,
				Org:         node.Org,
				Query:       node.Query,
				Status:      node.Status,
				CountryCode: node.CountryCode,
				District:    node.District,
				Continent:   node.Continent,
				Currency:    node.Currency,
				Mobile:      node.Mobile,
				Proxy:       node.Proxy,
				Hosting:     node.Hosting,
				IsOnline:    node.IsOnline,
				LastPing:    node.LastPing,
			}
			broadcastNodeUpdate(nodeInfo)
		}
	}
}

// Function to start the Zano node
func startZanoNode(ctx context.Context, wd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	// Get absolute path to the zanod binary
	zanodPath := filepath.Join(wd, "zano", "zanod")
	zanodPath, err := filepath.Abs(zanodPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting absolute path: %v", err)
	}
	log.Info().Str("path", zanodPath).Msg("Zanod binary path")

	// Check if the binary exists and is executable
	if info, err := os.Stat(zanodPath); err != nil {
		return nil, nil, nil, fmt.Errorf("error checking zanod binary: %v", err)
	} else {
		log.Info().Str("mode", info.Mode().String()).Msg("Zanod binary permissions")
	}

	// Command to start the Zano node directly
	log.Info().Msg("Starting Zano node...")
	cmd := exec.CommandContext(ctx, zanodPath, "--log-level", "2", "--rpc-bind-ip", "127.0.0.1", "--rpc-bind-port", "11211", "--no-console")
	cmd.Dir = wd // Set the working directory

	// Create pipes to capture both stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, fmt.Errorf("error starting command: %v", err)
	}

	// Monitor the context cancellation
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			log.Info().Msg("Context cancelled, terminating Zano node...")
			// First try SIGTERM
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				log.Error().Err(err).Msg("Error sending SIGTERM to Zano node")
				// If SIGTERM fails, try SIGKILL
				if err := cmd.Process.Kill(); err != nil {
					log.Error().Err(err).Msg("Error force killing Zano node")
				}
			}
		}
	}()

	// Monitor process exit
	go func() {
		err := cmd.Wait()
		if err != nil {
			// Read stderr to get error details
			stderrBytes, _ := io.ReadAll(stderr)
			if len(stderrBytes) > 0 {
				log.Error().
					Err(err).
					Str("stderr", string(stderrBytes)).
					Msg("Zano node process failed")
			} else {
				log.Error().Err(err).Msg("Zano node process failed")
			}
		}
	}()

	log.Info().Int("pid", cmd.Process.Pid).Msg("Started zanod process")
	return cmd, stdout, stderr, nil
}

// Function to ping a node and update its status
func pingNode(ip string, db *database.DB) bool {
	log.Info().Str("ip", ip).Msg("Pinging node")

	// Try TCP connection to Zano RPC port
	isOnline := false
	rpcPort := "11211" // Zano RPC port
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", ip, rpcPort), 5*time.Second)
	if err == nil {
		conn.Close()
		isOnline = true
		log.Info().Str("ip", ip).Str("port", rpcPort).Msg("TCP connection successful")
	} else {
		log.Debug().Str("ip", ip).Str("port", rpcPort).Err(err).Msg("TCP connection failed")

		// If TCP fails, try ICMP ping
		cmd := exec.Command("ping", "-c", "1", "-W", "5", ip)
		if err := cmd.Run(); err == nil {
			isOnline = true
			log.Info().Str("ip", ip).Msg("ICMP ping successful")
		} else {
			log.Debug().Str("ip", ip).Msg("Both TCP and ICMP ping failed")
		}
	}

	// Update node status in database
	if err := db.UpdateNodeStatus(ip, isOnline); err != nil {
		log.Error().Err(err).Str("ip", ip).Msg("Error updating node status")
		return false
	}

	if isOnline {
		log.Info().Str("ip", ip).Msg("Node is ONLINE")
	} else {
		log.Info().Str("ip", ip).Msg("Node is OFFLINE")
	}

	// Get the full node information from the database
	node, err := db.GetNode(ip)
	if err != nil {
		log.Error().Err(err).Str("ip", ip).Msg("Error getting node information for broadcast")
		return false
	}

	// Broadcast status update with full node information
	nodeInfo := &NodeInfo{
		IP:          node.IP,
		Country:     node.Country,
		City:        node.City,
		Lat:         node.Lat,
		Lon:         node.Lon,
		ISP:         node.ISP,
		LastSeen:    node.LastSeen,
		IsNew:       false,
		Region:      node.Region,
		RegionName:  node.RegionName,
		Timezone:    node.Timezone,
		Zip:         node.Zip,
		AS:          node.AS,
		Org:         node.Org,
		Query:       node.Query,
		Status:      node.Status,
		CountryCode: node.CountryCode,
		District:    node.District,
		Continent:   node.Continent,
		Currency:    node.Currency,
		Mobile:      node.Mobile,
		Proxy:       node.Proxy,
		Hosting:     node.Hosting,
		IsOnline:    isOnline,
		LastPing:    time.Now(),
	}
	broadcastNodeUpdate(nodeInfo)

	return isOnline
}

// Add ping worker function
func startPingWorker(ctx context.Context, db *database.DB) {
	// Perform initial ping on all nodes
	log.Info().Msg("Performing initial ping on all nodes...")
	nodes, err := db.GetAllNodes()
	if err != nil {
		log.Error().Err(err).Msg("Error getting nodes for initial ping")
	} else {
		for _, node := range nodes {
			pingNode(node.IP, db)
		}
	}

	// Start regular ping interval
	ticker := time.NewTicker(2 * time.Minute) // Ping every 2 minutes
	defer ticker.Stop()

	log.Info().Msg("Starting regular ping interval...")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Ping worker stopped")
			return
		case <-ticker.C:
			log.Info().Msg("Starting new ping cycle")
			nodes, err := db.GetAllNodes()
			if err != nil {
				log.Error().Err(err).Msg("Error getting nodes for ping")
				continue
			}
			log.Info().Int("totalNodes", len(nodes)).Msg("Found nodes to check")

			onlineCount := 0
			offlineCount := 0
			skippedCount := 0

			for _, node := range nodes {
				// Skip nodes that were pinged recently
				if time.Since(node.LastPing) < 1*time.Minute {
					log.Debug().
						Str("ip", node.IP).
						Time("lastPing", node.LastPing).
						Msg("Skipping recently pinged node")
					skippedCount++
					continue
				}

				if pingNode(node.IP, db) {
					onlineCount++
				} else {
					offlineCount++
				}
			}

			log.Info().
				Int("totalNodes", len(nodes)).
				Int("onlineNodes", onlineCount).
				Int("offlineNodes", offlineCount).
				Int("skippedNodes", skippedCount).
				Msg("Ping cycle completed")
		}
	}
}

// NmapScanResult represents the result of an nmap scan
type NmapScanResult struct {
	IP       string   `json:"ip"`
	Ports    []int    `json:"ports"`
	Services []string `json:"services"`
	OS       string   `json:"os,omitempty"`
	ScanTime string   `json:"scanTime"`
	Error    string   `json:"error,omitempty"`
}

// Function to perform nmap scan
func performNmapScan(ip string, ports string) (*NmapScanResult, error) {
	log.Info().Str("ip", ip).Str("ports", ports).Msg("Starting nmap scan")

	// Default to common ports if none specified
	if ports == "" {
		ports = "20-25,80,443,11211"
	}

	// Construct nmap command
	cmd := exec.Command("nmap", "-p", ports, "-sV", "-O", ip)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("ip", ip).Msg("Nmap scan failed")
		return &NmapScanResult{
			IP:       ip,
			Error:    fmt.Sprintf("Scan failed: %v", err),
			ScanTime: time.Now().Format(time.RFC3339),
		}, err
	}

	// Parse nmap output
	result := &NmapScanResult{
		IP:       ip,
		ScanTime: time.Now().Format(time.RFC3339),
	}

	// Parse ports and services
	outputStr := string(output)
	portRegex := regexp.MustCompile(`(\d+)/tcp\s+(\w+)\s+(.+)`)
	matches := portRegex.FindAllStringSubmatch(outputStr, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			port, _ := strconv.Atoi(match[1])
			result.Ports = append(result.Ports, port)
			result.Services = append(result.Services, match[3])
		}
	}

	// Try to extract OS information
	osRegex := regexp.MustCompile(`OS details: (.+)`)
	if osMatch := osRegex.FindStringSubmatch(outputStr); len(osMatch) > 1 {
		result.OS = osMatch[1]
	}

	log.Info().
		Str("ip", ip).
		Ints("ports", result.Ports).
		Strs("services", result.Services).
		Str("os", result.OS).
		Msg("Nmap scan completed")

	return result, nil
}

func main() {
	log.Info().Msg("Starting Zano peer finder...")

	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting working directory")
	}
	log.Info().Str("workingDir", wd).Msg("Working directory")

	// Initialize rate limiter (45 requests per minute = 0.75 requests per second)
	rateLimiter := NewRateLimiter(0.75, 45)

	// Initialize database
	log.Info().Msg("Initializing database...")
	db, err := database.New("nodes.db")
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
	}
	defer db.Close()
	log.Info().Msg("Database initialized successfully")

	// Load saved peers
	log.Info().Msg("Loading saved peers...")
	savedPeers, err := db.LoadPeers()
	if err != nil {
		log.Error().Err(err).Msg("Error loading saved peers")
	} else {
		log.Info().Int("peerCount", len(savedPeers)).Msg("Loaded saved peers")
	}

	// Load existing nodes
	log.Info().Msg("Loading existing nodes...")
	existingNodes, err := db.GetAllNodes()
	if err != nil {
		log.Error().Err(err).Msg("Error loading existing nodes")
	} else {
		log.Info().Int("nodeCount", len(existingNodes)).Msg("Loaded existing nodes")
	}

	// Initialize IP info service
	log.Info().Msg("Initializing IP info service...")
	ipService := ipinfo.NewService()
	log.Info().Msg("IP info service initialized")

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create HTTP server with timeout settings
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start web server
	log.Info().Msg("Starting web server...")
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, db)
	})
	http.Handle("/", http.FileServer(http.Dir("static")))

	// Add nmap scan endpoint
	http.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse request body
		var req struct {
			IP    string `json:"ip"`
			Ports string `json:"ports"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Perform scan
		result, err := performNmapScan(req.IP, req.Ports)
		if err != nil {
			http.Error(w, "Scan failed", http.StatusInternalServerError)
			return
		}

		// Return results
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Start web server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Info().Msg("Web server listening on :8080")
		serverErr <- server.ListenAndServe()
	}()

	// Start Zano node in a separate goroutine
	nodeStarted := make(chan error, 1)
	go func() {
		cmd, stdout, stderr, err := startZanoNode(ctx, wd)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start Zano node")
			nodeStarted <- err
			return
		}

		// Regular expression to match IP addresses with optional ports
		ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}(?::\d+)?\b`)

		// WaitGroup to ensure all goroutines complete
		var wg sync.WaitGroup
		wg.Add(2)

		// Function to process output
		processOutput := func(reader io.Reader, prefix string, db *database.DB, ipService *ipinfo.Service) {
			defer wg.Done()
			scanner := bufio.NewScanner(reader)
			// Set a larger buffer size for the scanner
			const maxCapacity = 1024 * 1024 // 1MB
			buf := make([]byte, maxCapacity)
			scanner.Buffer(buf, maxCapacity)
			scanner.Split(customSplit)

			// Keep track of recently processed IPs to avoid duplicates
			recentIPs := make(map[string]time.Time)
			recentIPsMutex := &sync.Mutex{}

			// Keep track of discovered peers
			discoveredPeers := make(map[string]bool)
			for _, peer := range savedPeers {
				discoveredPeers[peer] = true
			}

			// Cleanup old entries every minute
			go func() {
				ticker := time.NewTicker(1 * time.Minute)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						recentIPsMutex.Lock()
						now := time.Now()
						for ip, timestamp := range recentIPs {
							if now.Sub(timestamp) > 5*time.Minute {
								delete(recentIPs, ip)
							}
						}
						recentIPsMutex.Unlock()

						// Save discovered peers to database
						peers := make([]string, 0, len(discoveredPeers))
						for peer := range discoveredPeers {
							peers = append(peers, peer)
						}
						if err := db.SavePeers(peers); err != nil {
							log.Error().Err(err).Msg("Error saving peers to database")
						} else {
							log.Debug().Int("peerCount", len(peers)).Msg("Saved peers to database")
						}
					}
				}
			}()

			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return
				default:
					line := strings.TrimSpace(scanner.Text())
					if line == "" {
						continue
					}

					// Log the line for debugging
					log.Debug().Str("prefix", prefix).Str("line", line).Msg("Node output")

					// Find all IP addresses in the line
					ipMatches := ipRegex.FindAllString(line, -1)
					for _, ipMatch := range ipMatches {
						// Split IP and port if present
						ip := ipMatch
						if strings.Contains(ipMatch, ":") {
							ip = strings.Split(ipMatch, ":")[0]
						}

						// Skip localhost IPs
						if !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "0.") {
							// Add to discovered peers
							discoveredPeers[ip] = true

							// Check if we've processed this IP recently
							recentIPsMutex.Lock()
							if _, exists := recentIPs[ip]; exists {
								recentIPsMutex.Unlock()
								continue
							}
							recentIPs[ip] = time.Now()
							recentIPsMutex.Unlock()

							log.Info().Str("ip", ip).Msg("Found new IP")
							// Check if we already have this IP in the database
							existingNode, err := db.GetNode(ip)
							if err != nil {
								log.Error().Err(err).Str("ip", ip).Msg("Error checking node in database")
								continue
							}

							// If node exists and was updated recently, skip
							if existingNode != nil && time.Since(existingNode.LastSeen) < 5*time.Minute {
								log.Debug().Str("ip", ip).Time("lastSeen", existingNode.LastSeen).Msg("Skipping recently updated node")
								continue
							}

							// Wait for rate limiter before making API call
							rateLimiter.Wait()

							// Get IP information
							log.Info().Str("ip", ip).Msg("Getting IP info")
							ipInfo, err := ipService.GetIPInfo(ip)
							if err != nil {
								if err.Error() == "rate limit exceeded" {
									log.Warn().Msg("Rate limit exceeded, waiting...")
									time.Sleep(time.Minute)
									continue
								}
								log.Error().Err(err).Str("ip", ip).Msg("Error getting IP info")
								continue
							}

							if ipInfo.Status == "success" {
								log.Info().Str("ip", ip).Msg("Successfully got IP info")
								// Create node info
								node := &database.Node{
									IP:          ip,
									Country:     ipInfo.Country,
									City:        ipInfo.City,
									Lat:         ipInfo.Lat,
									Lon:         ipInfo.Lon,
									ISP:         ipInfo.ISP,
									LastSeen:    time.Now(),
									Region:      ipInfo.Region,
									RegionName:  ipInfo.RegionName,
									Timezone:    ipInfo.Timezone,
									Zip:         ipInfo.Zip,
									AS:          ipInfo.AS,
									Org:         ipInfo.Org,
									Query:       ipInfo.Query,
									Status:      ipInfo.Status,
									CountryCode: ipInfo.CountryCode,
									District:    ipInfo.District,
									Continent:   ipInfo.Continent,
									Currency:    ipInfo.Currency,
									Mobile:      ipInfo.Mobile,
									Proxy:       ipInfo.Proxy,
									Hosting:     ipInfo.Hosting,
									IsOnline:    false,
									LastPing:    time.Time{},
								}

								// Save to database
								if err := db.UpsertNode(node); err != nil {
									log.Error().Err(err).Str("ip", ip).Msg("Error saving node to database")
									continue
								}
								log.Info().
									Str("ip", ip).
									Str("country", node.Country).
									Str("city", node.City).
									Str("isp", node.ISP).
									Msg("Saved new node to database")

								// Ping the node immediately
								isOnline := pingNode(ip, db)

								// Broadcast to all clients
								nodeInfo := &NodeInfo{
									IP:          node.IP,
									Country:     node.Country,
									City:        node.City,
									Lat:         node.Lat,
									Lon:         node.Lon,
									ISP:         node.ISP,
									LastSeen:    node.LastSeen,
									IsNew:       existingNode == nil, // Only true for newly discovered nodes
									Region:      node.Region,
									RegionName:  node.RegionName,
									Timezone:    node.Timezone,
									Zip:         node.Zip,
									AS:          node.AS,
									Org:         node.Org,
									Query:       node.Query,
									Status:      node.Status,
									CountryCode: node.CountryCode,
									District:    node.District,
									Continent:   node.Continent,
									Currency:    node.Currency,
									Mobile:      node.Mobile,
									Proxy:       node.Proxy,
									Hosting:     node.Hosting,
									IsOnline:    isOnline,
									LastPing:    time.Now(),
								}
								broadcastNodeUpdate(nodeInfo)
							}
						}
					}
				}
			}

			if err := scanner.Err(); err != nil {
				log.Error().Err(err).Str("prefix", prefix).Msg("Error reading output")
			}
		}

		// Start processing stdout and stderr in separate goroutines
		go processOutput(stdout, "STDOUT", db, ipService)
		go processOutput(stderr, "STDERR", db, ipService)

		// Wait a moment to ensure the node starts properly
		time.Sleep(2 * time.Second)

		// Check if the process is still running
		if cmd.Process == nil || cmd.ProcessState != nil {
			log.Error().Msg("Zano node failed to start properly")
			nodeStarted <- fmt.Errorf("node failed to start")
			return
		}

		// Signal that the node has started successfully
		nodeStarted <- nil

		// Wait for the command to finish
		if err := cmd.Wait(); err != nil {
			log.Error().Err(err).Msg("Zano node process finished with error")
		} else {
			log.Info().Msg("Zano node process completed successfully")
		}

		// Wait for all output processing to complete
		wg.Wait()
	}()

	// Start ping worker
	go startPingWorker(ctx, db)

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either interrupt signal, server error, or node startup error
	select {
	case <-sigChan:
		log.Info().Msg("Received interrupt signal. Shutting down...")
		cancel() // Cancel the context
	case err := <-serverErr:
		log.Error().Err(err).Msg("Web server error")
		cancel()
	case err := <-nodeStarted:
		if err != nil {
			log.Error().Err(err).Msg("Error starting Zano node")
			// Give the node a moment to output any error messages
			time.Sleep(2 * time.Second)
			cancel()
		} else {
			log.Info().Msg("Zano node started successfully")
			// Wait for interrupt signal
			<-sigChan
			log.Info().Msg("Received interrupt signal. Shutting down...")
			cancel()
		}
	}

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	log.Info().Msg("Shutting down web server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error shutting down web server")
	}

	// Wait for cleanup with timeout
	done := make(chan struct{})
	go func() {
		// Wait for context cancellation to complete
		<-ctx.Done()

		// Give the Zano node a moment to shut down gracefully
		time.Sleep(2 * time.Second)

		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Shutdown complete")
	case <-time.After(5 * time.Second):
		log.Warn().Msg("Shutdown timed out, forcing exit")
		os.Exit(1)
	}
}
