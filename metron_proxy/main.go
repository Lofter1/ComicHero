package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const metronBaseURL = "https://metron.cloud/api"
const metronMediaURL = "https://static.metron.cloud"

type cacheEntry struct {
	data      []byte
	header    http.Header
	timestamp int64
}

var (
	apiCache   = make(map[string]cacheEntry)
	mediaCache = make(map[string]cacheEntry)
	cacheTTL   = int64(60 * 60 * 24) // cache for 1 day
)

var (
	username string = os.Getenv("METRON_USERNAME")
	password string = os.Getenv("METRON_PASSWORD")
	proxyBaseURL string = os.Getenv("BASEURL")
	port string = os.Getenv("PORT")
)

var mu sync.RWMutex


func init() {
	if username == "" || password == "" {
		log.Fatal("METRON_EMAIL and METRON_PASSWORD must be set in env")
	}
}

func main() {
	http.HandleFunc("/media/", handleMediaProxy)
	http.HandleFunc("/", handleProxy)

	fmt.Printf("Listening on %s:%s\n", proxyBaseURL, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func rewriteImageURLs(data []byte) []byte {
	return bytes.ReplaceAll(data,
		[]byte("https://static.metron.cloud/media/"),
		[]byte("http://"+proxyBaseURL+":"+port+"/media/"),
	)
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	enableCors(w)

	cacheKey := r.URL.Path + "?" + r.URL.RawQuery

	mu.RLock()
	entry, found := apiCache[cacheKey]
	mu.RUnlock()

	if found && !isExpired(entry) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(entry.data)
		return
	}

	metronUrl, err := url.Parse(metronBaseURL)
	if err != nil {
		log.Print(err)
	}
	metronUrl = metronUrl.JoinPath(r.URL.Path)
	metronUrl.RawQuery = r.URL.RawQuery

	req, err := http.NewRequest("GET", metronUrl.String(), nil)
	if err != nil {
		http.Error(w, "Failed to build request", http.StatusInternalServerError)
		log.Print(err)
		return
	}
	req.SetBasicAuth(username, password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to contact Metron API", http.StatusBadGateway)
		log.Print(err)
		return
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		log.Print(err)
	}
	modified := rewriteImageURLs(buf.Bytes())

	// Cache the result
	mu.Lock()
	apiCache[cacheKey] = cacheEntry{
		data:      modified,
		timestamp: time.Now().Unix(),
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, err = w.Write(modified)
	if err != nil {
		log.Print(err)
	}
}

func handleMediaProxy(w http.ResponseWriter, r *http.Request) {
	enableCors(w)

	path := r.URL.Path
	if path == "" {
		http.Error(w, "Missing media path", http.StatusBadRequest)
		return
	}

	cacheKey := path
	mu.RLock()
	entry, found := mediaCache[cacheKey]
	mu.RUnlock()

	if found && !isExpired(entry) {
		// Serve from cache
		for k, v := range entry.header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(entry.data)
		return
	}

	mediaUrl := metronMediaURL + path

	resp, err := http.Get(mediaUrl)
	if err != nil {
		http.Error(w, "Failed to fetch media", http.StatusBadGateway)
		log.Print(err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading media", http.StatusInternalServerError)
		log.Print(err)
		return
	}

	// Cache the response
	mu.Lock()
	mediaCache[cacheKey] = cacheEntry{
		data:      body,
		header:    resp.Header,
		timestamp: time.Now().Unix(),
	}
	mu.Unlock()

	// Send response
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func isExpired(entry cacheEntry) bool {
	return time.Now().Unix()-entry.timestamp > cacheTTL
}