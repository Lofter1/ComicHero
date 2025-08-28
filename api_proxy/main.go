package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const metronBaseURL = "https://metron.cloud/api"
const metronMediaURL = "https://static.metron.cloud"

const gcdBaseURL = "https://www.comics.org/api"
const gcdMediaURL = "https://files1.comics.org"

const port = "8080"

var (
	metronUsername string = os.Getenv("METRON_USERNAME")
	metronPassword string = os.Getenv("METRON_PASSWORD")

	gcdUsername string = os.Getenv("GCD_USERNAME")
	gcdPassword string = os.Getenv("GCD_PASSWORD")

	proxyBaseURL  string = os.Getenv("PROXY_URL")
	pocketbaseURL string = os.Getenv("POCKETBASE_URL")
	redisAddr     string = os.Getenv("REDIS_ADDR")
)

var rateLimiter = make(chan struct{}, 1)

func init() {
	if metronUsername == "" || metronPassword == "" {
		log.Fatal("METRON_USERNAME and METRON_PASSWORD must be set in env")
	}

	if gcdUsername == "" || gcdPassword == "" {
		log.Fatal("GCD_USERNAME and GCD_PASSWORD must be set in env")
	}
	InitCaching()

	initRatelimiting()
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/gcd/", handleGCDProxy)
	mux.HandleFunc("/gcd/media/", handleGCDMediaProxy)

	mux.HandleFunc("/metron/", handleMetronProxy)
	mux.HandleFunc("/metron/media/", handleMetronMediaProxy)

	fmt.Printf("Listening on port %s\n", port)

	log.Fatal(http.ListenAndServe(":"+port, corsMiddleware(checkAuthMiddleware(mux))))
}

func replaceURLs(data []byte, replaceURL string, apiPath string) []byte {
	newUrl, _ := url.JoinPath(proxyBaseURL, apiPath)

	return bytes.ReplaceAll(data,
		[]byte(replaceURL),
		[]byte(newUrl),
	)
}

func initRatelimiting() {
	rateLimiter <- struct{}{}
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			select {
			case rateLimiter <- struct{}{}:
			default:
				// channel full, skip
			}
		}
	}()
}

func throttle() {
	<-rateLimiter
}

func handleGCDProxy(w http.ResponseWriter, r *http.Request) {
	cacheKey := r.URL.Path + "?" + r.URL.RawQuery

	if entry, found := GetCache(cacheKey, false); found {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(entry.Data)
		return
	}

	gcdUrl, err := url.Parse(gcdBaseURL)
	if err != nil {
		log.Print(err)
	}
	gcdUrl = gcdUrl.JoinPath(strings.TrimPrefix(r.URL.Path, "/gcd"))
	gcdUrl.RawQuery = r.URL.RawQuery

	req, err := http.NewRequest("GET", gcdUrl.String(), nil)
	if err != nil {
		http.Error(w, "Failed to build request", http.StatusInternalServerError)
		log.Print(err)
		return
	}
	req.SetBasicAuth(gcdUsername, gcdPassword)

	log.Println(req.URL)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to contact gcd API", http.StatusBadGateway)
		log.Print(err)
		return
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		log.Print(err)
	}
	modified := replaceURLs(buf.Bytes(), gcdMediaURL, strings.Split(r.URL.Path, "/")[1])
	modified = replaceURLs(modified, gcdBaseURL, strings.Split(r.URL.Path, "/")[1])

	contentType := resp.Header.Get("Content-Type")

	if resp.StatusCode == http.StatusOK && strings.Contains(contentType, "application/json") {
		SaveCache(cacheKey, CacheEntry{
			Data:      modified,
			Header:    nil,
			Timestamp: time.Now().Unix(),
		}, time.Duration(cacheTTL)*time.Second, false)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, err = w.Write(modified)
	if err != nil {
		log.Print(err)
	}
}

func handleGCDMediaProxy(w http.ResponseWriter, r *http.Request) {

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

	mediaUrl := gcdMediaURL + strings.TrimPrefix(path, "/gcd")

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

func handleMetronProxy(w http.ResponseWriter, r *http.Request) {
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

	metronUrl = metronUrl.JoinPath(strings.TrimPrefix(r.URL.Path, "/metron"))
	metronUrl.RawQuery = r.URL.RawQuery

	req, err := http.NewRequest("GET", metronUrl.String(), nil)
	if err != nil {
		http.Error(w, "Failed to build request", http.StatusInternalServerError)
		log.Print(err)
		return
	}
	req.SetBasicAuth(metronUsername, metronPassword)

	throttle()

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
	modified := replaceURLs(buf.Bytes(), metronMediaURL, strings.Split(r.URL.Path, "/")[1])

	contentType := resp.Header.Get("Content-Type")

	if resp.StatusCode == http.StatusOK && strings.Contains(contentType, "application/json") {
		SaveCache(cacheKey, CacheEntry{
			Data:      modified,
			Header:    nil,
			Timestamp: time.Now().Unix(),
		}, time.Duration(cacheTTL)*time.Second, false)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, err = w.Write(modified)
	if err != nil {
		log.Print(err)
	}
}

func handleMetronMediaProxy(w http.ResponseWriter, r *http.Request) {

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

	mediaUrl := metronMediaURL + strings.TrimPrefix(path, "/metron")

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
