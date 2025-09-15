package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
)

var (
	ctx        = context.Background()
	redisAddr  = os.Getenv("REDIS_ADDR") // e.g. "keydb:6379"
	redisPass  = os.Getenv("REDIS_PASS") // optional
	registryNS = "collabora:pods"
	mappingNS  = "wopi:mapping"
	rdb        *redis.Client
	upgrader   = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
	mu sync.RWMutex
)

// Constants
const (
	DefaultPort        = "8080"
	DefaultTTL         = 30 * time.Hour
	TTLRenewalInterval = 15 * time.Hour
	ConnectionTTL      = 30 * time.Hour
	MaxConnections     = 999999
	CollaboraPort      = "9980"
	WOPISrcParam       = "WOPISrc"
)

type PodInfo struct {
	IP          string
	Connections int
}

func getWOPISrc(c echo.Context) string {
	// Try query parameter first
	wopiSrc := c.QueryParam(WOPISrcParam)
	if wopiSrc != "" {
		return wopiSrc
	}

	// Try POST body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return ""
	}

	// Simple form parsing for WOPISrc
	formData := string(body)
	if strings.Contains(formData, WOPISrcParam+"=") {
		parts := strings.Split(formData, WOPISrcParam+"=")
		if len(parts) > 1 {
			return strings.Split(parts[1], "&")[0]
		}
	}
	return ""
}

func getMappingKey(wopiSrc string) string {
	return fmt.Sprintf("%s:%s", mappingNS, wopiSrc)
}

func getConnectionKey(podIP string) string {
	return fmt.Sprintf("%s:%s", registryNS, podIP)
}

func getPodURL(podIP string) string {
	return fmt.Sprintf("http://%s:%s", podIP, CollaboraPort)
}

func getWebSocketURL(podIP string, ctx echo.Context) string {
	return fmt.Sprintf("ws://%s:%s%s", podIP, CollaboraPort, ctx.Request().RequestURI)

	// wopiSrc := u.Query().Get(WOPISrcParam)
	// encodedPath := url.PathEscape(path)
	// return fmt.Sprintf("ws://%s:%s%s?%s=%s", podIP, CollaboraPort, encodedPath, WOPISrcParam, wopiSrc)
}

func handleRequest(c echo.Context) error {
	reqPath := c.Request().URL.Path
	if strings.HasPrefix(reqPath, "/cool/") {
		log.Println("WebSocket comes through handleRequest")
		return handleWebSocket(c)
	}

	// Get WOPISrc from query or body
	wopiSrc := getWOPISrc(c)
	// if wopiSrc == "" {
	// 	return c.String(http.StatusBadRequest, "WOPISrc parameter required")
	// }

	// Check if WOPISrc is already mapped
	podIP, err := getMappedPod(wopiSrc)
	if err != nil {
		log.Printf("Error checking mapping: %v", err)
	}

	if podIP != "" {
		log.Printf("Found existing mapping (request): %s -> %s", wopiSrc, podIP)
		// Renew TTL for existing mapping
		renewMappingTTL(wopiSrc)
		return proxyToPod(c, podIP)
	}

	// Find pod with least connections
	podIP, err = findLeastConnectedPod()
	if err != nil {
		log.Printf("Error finding pod: %v", err)
		return c.String(http.StatusInternalServerError, "No available pods")
	}

	if len(wopiSrc) > 0 {
		// Register the mapping
		err = registerMapping(wopiSrc, podIP)
		if err != nil {
			log.Printf("Error registering mapping: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to register mapping")
		}
		log.Printf("New mapping (request): %s -> %s", wopiSrc, podIP)
	}

	return proxyToPod(c, podIP)
}

func handleWebSocket(c echo.Context) error {
	log.Println("Start handleWebSocket")
	wopiSrc := getWOPISrc(c)
	if wopiSrc == "" {
		return c.String(http.StatusBadRequest, "WOPISrc parameter required")
	}

	log.Println("Check if this src is already mapped")
	podIP, err := getMappedPod(wopiSrc)
	if err != nil {
		log.Printf("Error checking mapping: %v", err)
	}

	if podIP == "" {
		// Find pod with least connections
		podIP, err = findLeastConnectedPod()
		log.Printf("Found pod with least connections: %s\n", podIP)
		if err != nil {
			log.Printf("Error finding pod: %v", err)
			return c.String(http.StatusInternalServerError, "No available pods")
		}

		// Register the mapping
		err = registerMapping(wopiSrc, podIP)
		log.Printf("Registered mapping (socket): %s -> %s", wopiSrc, podIP)
		if err != nil {
			log.Printf("Error registering mapping: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to register mapping")
		}
	} else {
		log.Printf("Found existing mapping (socket): %s -> %s", wopiSrc, podIP)
		// Renew TTL for existing mapping
		renewMappingTTL(wopiSrc)
	}

	// Upgrade to WebSocket
	clientWS, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer clientWS.Close()

	podURL := getWebSocketURL(podIP, c)

	log.Println("Connecting to pod WebSocket: " + podURL)
	header := http.Header{}
	header.Set("Host", "collabora.local")          // or whatever the backend expects
	header.Set("Origin", "http://collabora.local") // many WS servers require this
	podWS, _, err := websocket.DefaultDialer.Dial(podURL, header)
	if err != nil {
		log.Printf("Failed to connect to pod %s: %v", podIP, err)
		return err
	}
	defer podWS.Close()

	// Increment connection count
	incrementConnectionCount(podIP)

	defer func() {
		decrementConnectionCount(podIP)
	}()

	// Start TTL renewal ticker for this WebSocket connection
	ttlTicker := time.NewTicker(TTLRenewalInterval)
	defer ttlTicker.Stop()

	// Start TTL renewal goroutine
	go func() {
		defer ttlTicker.Stop()
		for range ttlTicker.C {
			// Renew the mapping TTL
			renewMappingTTL(wopiSrc)
			log.Printf("Timer - Renewed TTL for mapping: %s", wopiSrc)
		}
	}()

	clientNetConn := clientWS.NetConn()
	podNetConn := podWS.NetConn()

	// bidirectional raw TCP copy
	go func() {
		if _, err := io.Copy(podNetConn, clientNetConn); err != nil {
			log.Printf("ERR copy client->pod: %v", err)
		}
	}()
	if _, err := io.Copy(clientNetConn, podNetConn); err != nil {
		log.Printf("ERR copy pod->client: %v", err)
	}

	return nil
}

func getMappedPod(wopiSrc string) (string, error) {
	if len(wopiSrc) == 0 {
		return "", nil // No mapping found
	}
	key := getMappingKey(wopiSrc)
	result, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // No mapping found
	}
	return result, err
}

func registerMapping(wopiSrc, podIP string) error {
	key := getMappingKey(wopiSrc)
	return rdb.Set(ctx, key, podIP, DefaultTTL).Err()
}

func renewMappingTTL(wopiSrc string) error {
	key := getMappingKey(wopiSrc)
	return rdb.Expire(ctx, key, DefaultTTL).Err()
}

func findLeastConnectedPod() (string, error) {
	pattern := fmt.Sprintf("%s:*", registryNS)
	keys, err := rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return "", err
	}

	if len(keys) == 0 {
		return "", fmt.Errorf("no pods available")
	}

	var bestPod PodInfo
	bestPod.Connections = MaxConnections

	for _, key := range keys {
		// Extract IP from key (collabora:pods:IP)
		parts := strings.Split(key, ":")
		if len(parts) < 3 {
			continue
		}
		podIP := parts[2]

		// Get connection count
		connKey := getConnectionKey(podIP)
		connStr, err := rdb.Get(ctx, connKey).Result()
		connections := 0
		log.Printf("Conn count %s: %s\n", podIP, connStr)
		if err == nil {
			connections, _ = strconv.Atoi(connStr)
		}

		if connections < bestPod.Connections {
			bestPod.IP = podIP
			bestPod.Connections = connections
		}
	}

	if bestPod.IP == "" {
		return "", fmt.Errorf("no valid pods found")
	}

	return bestPod.IP, nil
}

func incrementConnectionCount(podIP string) {
	connKey := getConnectionKey(podIP)
	intCmd := rdb.Incr(ctx, connKey)
	count, _ := intCmd.Result()
	log.Printf("More connections for %s: %d", podIP, count)
	rdb.Expire(ctx, connKey, ConnectionTTL) // Auto-expire if no activity
}

func decrementConnectionCount(podIP string) {
	connKey := getConnectionKey(podIP)
	intCmd := rdb.Decr(ctx, connKey)
	count, _ := intCmd.Result()
	log.Printf("Less connections for %s: %d", podIP, count)
}

func proxyToPod(c echo.Context, podIP string) error {
	// Create reverse proxy
	target, err := url.Parse(getPodURL(podIP))
	if err != nil {
		return err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Modify request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.URL.Host = target.Host
		req.URL.Scheme = target.Scheme
	}

	// Handle response
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Add CORS headers
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		resp.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		return nil
	}

	proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

func main() {
	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})

	// Test Redis connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	log.Println("Connected to Redis/KeyDB")

	// Initialize Echo server
	e := echo.New()
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	e.Any("/cool/*/ws", handleWebSocket)
	e.Any("/*", handleRequest)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}
	log.Printf("Starting cool-distributor on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
