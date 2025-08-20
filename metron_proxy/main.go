package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const metronBaseURL = "https://metron.cloud/api"
const metronMediaURL = "https://static.metron.cloud"
const port = "8080"

var (
	username      string = os.Getenv("METRON_USERNAME")
	password      string = os.Getenv("METRON_PASSWORD")
	proxyBaseURL  string = os.Getenv("METRON_PROXY_URL")
	pocketbaseURL string = os.Getenv("POCKETBASE_URL")
	redisAddr     string = os.Getenv("REDIS_ADDR")
)

func init() {
	if username == "" || password == "" {
		log.Fatal("METRON_EMAIL and METRON_PASSWORD must be set in env")
	}
	InitCaching()
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/media/", handleMediaProxy)
	mux.HandleFunc("/", handleProxy)

	fmt.Printf("Listening on port %s\n", port)

	log.Fatal(http.ListenAndServe(":"+port, corsMiddleware(checkAuthMiddleware(mux))))
}

func rewriteImageURLs(data []byte) []byte {
	return bytes.ReplaceAll(data,
		[]byte(metronMediaURL),
		[]byte(proxyBaseURL),
	)
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	cacheKey := r.URL.Path + "?" + r.URL.RawQuery

	if entry, found := GetCache(cacheKey, false); found {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(entry.Data)
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

	SaveCache(cacheKey, CacheEntry{
		Data:      modified,
		Header:    nil,
		Timestamp: time.Now().Unix(),
	}, time.Duration(cacheTTL)*time.Second, false)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, err = w.Write(modified)
	if err != nil {
		log.Print(err)
	}
}

func handleMediaProxy(w http.ResponseWriter, r *http.Request) {

	path := r.URL.Path
	if path == "" {
		http.Error(w, "Missing media path", http.StatusBadRequest)
		return
	}

	if entry, found := GetCache(path, true); found {
		for k, v := range entry.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(entry.Data)
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

	SaveCache(path, CacheEntry{
		Data:      body,
		Header:    resp.Header.Clone(),
		Timestamp: time.Now().Unix(),
	}, time.Duration(cacheTTL)*time.Second, true)

	// Send response
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
