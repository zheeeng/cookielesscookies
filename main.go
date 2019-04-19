package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type session struct {
	Visits     int
	LastVisit  string
	YourString string
}

func genInitSession() *session {
	return &session{
		LastVisit: time.Now().Format(time.RFC1123Z),
	}
}

var (
	secret       = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	port         = "8090"
	staticFolder = "./public"
	tmpl         *template.Template
	cache        = map[string]*session{}
)

func init() {
	if envSecret := os.Getenv("SECRET"); envSecret != "" {
		port = envSecret
	}
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	if envStaticFolder := os.Getenv("STATIC_FOLDER"); envStaticFolder != "" {
		staticFolder = envStaticFolder
	}

	tmpl = template.Must(template.ParseFiles("index.html"))
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/source", sourceHandler)
	http.HandleFunc("/etags.jpg", etagsHandler)
	http.HandleFunc("/tracker.jpg", trackerHandler)

	fmt.Println("Server runs at " + port)
	http.ListenAndServe(":"+port, nil)
}

func indexHandler(res http.ResponseWriter, req *http.Request) {
	etag := genEtag(req)
	res.Header().Set("X-Etag", etag)

	switch req.Method {
	case http.MethodGet:
		sess := getSession(etag)
		tmpl.Execute(res, sess)
	case http.MethodPost:
		req.ParseForm()
		updateSessionString(etag, req.Form.Get("newstring"))
		res.Header().Set("Location", "./")
		res.WriteHeader(http.StatusFound)
	}
}
func sourceHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(res, "See <a href='https://github.com/lucb1e/cookielesscookies'>github.com/lucb1e/cookielesscookies</a>")
}
func etagsHandler(res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, "static/etags.jpg")
}
func trackerHandler(res http.ResponseWriter, req *http.Request) {
	etag := genEtag(req)

	updateSession(etag)

	res.Header().Set("Cache-Control", "no-cache")
	res.Header().Set("Etag", etag)
	res.Header().Set("X-Etag", etag)
	res.Header().Set("Content-Type", "image/jpeg")
	// res.Header().Set("Content-Length", 0)

	img, _ := os.Open("static/fingerprinting.jpg")
	defer img.Close()
	io.Copy(res, img)
}
func sha1Sum(raw string) string {
	h := sha1.New()
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}
func genEtag(req *http.Request) string {
	if noneMatch := req.Header.Get("If-None-Match"); noneMatch != "" {
		s1 := strings.Replace(noneMatch, "\\", "", -1)
		s2 := strings.Replace(s1, "/", "", -1)
		s3 := strings.Replace(s2, ".", "", -1)

		return s3[0:18]
	}

	remoteAddr := req.Header.Get("x-forwarded-for")

	if remoteAddr == "" {
		remoteAddr = req.RemoteAddr
	}

	s1 := sha1Sum(remoteAddr)
	s2 := sha1Sum(req.Header.Get("User-Agent"))
	s3 := sha1Sum(secret + s1 + s2)

	return s3[0:18]
}
func getSession(etag string) *session {
	if sess, ok := cache[etag]; ok {
		return sess
	}
	cache[etag] = genInitSession()
	return cache[etag]
}
func updateSession(etag string) {
	sess := getSession(etag)
	sess.Visits++
	sess.LastVisit = time.Now().Format(time.RFC1123Z)
}
func updateSessionString(etag string, str string) {
	sess := getSession(etag)
	sess.YourString = str
}
