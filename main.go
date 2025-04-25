package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func downloadVideo(videoURL, cookie, savePath string) error {
	client := &http.Client{}
	fmt.Println("Downloading", videoURL)

	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header = http.Header{
		"Accept":          []string{"*/*"},
		"Accept-Encoding": []string{"identity;q=1, *;q=0"},
		"Accept-Language": []string{"zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7"},
		"Cookie":          []string{cookie},
		"Dnt":             []string{"1"},
		"User-Agent":      []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.130 Safari/537.36"},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

func getVideoByURL(pageURL, saveBasePath string) error {
	if err := os.MkdirAll(saveBasePath, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}
	headers := map[string]string{
		"Accept":          "*/*",
		"Accept-Language": "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7",
		"DNT":             "1",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Site":  "same-origin",
		"Cookie":          "__cfduid=d8db8ce8747b090ff3601ac6d9d22fb951579718376; _ga=GA1.2.1940993661.1579718377; _gid=GA1.2.1806075473.1579718377; _ga=GA1.3.1940993661.1579718377; _gid=GA1.3.1806075473.1579718377",
		"Content-Type":    "application/x-www-form-urlencoded",
		"User-Agent":      "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3573.0 Safari/537.36",
	}

	client := &http.Client{}
	form := url.Values{}
	req, err := http.NewRequest("POST", pageURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("解析HTML失败: %v", err)
	}

	var results []string
	doc.Find("video.video-js").Each(func(i int, s *goquery.Selection) {
		if data, exists := s.Attr("data-apireq"); exists {
			results = append(results, data)
		}
	})

	for _, rslt := range results {
		apiData := fmt.Sprintf("d=%s", rslt)
		apiReq, err := http.NewRequest("POST", "https://v.anime1.me/api", bytes.NewBufferString(apiData))
		if err != nil {
			log.Printf("创建API请求失败: %v", err)
			continue
		}

		for k, v := range headers {
			apiReq.Header.Set(k, v)
		}

		apiResp, err := client.Do(apiReq)
		if err != nil {
			log.Printf("API请求失败: %v", err)
			continue
		}

		setCookies := apiResp.Cookies()
		setCookie := fmt.Sprintf("%s", setCookies)
		eRegex := regexp.MustCompile(`e=(.*?);`)
		pRegex := regexp.MustCompile(`p=(.*?);`)
		hRegex := regexp.MustCompile(`Secure h=(.*?);`)

		var cookieE, cookieP, cookieH string
		if eMatch := eRegex.FindStringSubmatch(setCookie); len(eMatch) > 1 {
			cookieE = eMatch[1]
		}
		if pMatch := pRegex.FindStringSubmatch(setCookie); len(pMatch) > 1 {
			cookieP = pMatch[1]
		}
		if hMatch := hRegex.FindStringSubmatch(setCookie); len(hMatch) > 1 {
			cookieH = hMatch[1]
		}

		cookies := fmt.Sprintf("e=%s;p=%s;h=%s;", cookieE, cookieP, cookieH)

		var srcInfo struct {
			S []struct {
				Src string `json:"src"`
			} `json:"s"`
		}

		if err := json.NewDecoder(apiResp.Body).Decode(&srcInfo); err != nil {
			apiResp.Body.Close()
			log.Printf("解析API响应失败: %v", err)
			continue
		}
		apiResp.Body.Close()

		if len(srcInfo.S) == 0 {
			log.Println("没有找到视频源")
			continue
		}

		srcURL := "http:" + srcInfo.S[0].Src
		baseName := filepath.Base(srcURL)
		fileName := strings.Replace(baseName, "b.", ".", 1)
		fullPath := filepath.Join(saveBasePath, fileName)

		fmt.Printf("开始下载: %s\n", fileName)
		if err := downloadVideo(srcURL, cookies, fullPath); err != nil {
			log.Printf("下载失败: %v\n", err)
			continue
		}
		fmt.Printf("%s 下载完毕\n", fileName)
	}

	return nil
}

func main() {
	pageURL := "https://anime1.me/category/2025%e5%b9%b4%e6%98%a5%e5%ad%a3/%e6%90%96%e6%bb%be%e6%a8%82%e6%98%af%e6%b7%91%e5%a5%b3%e7%9a%84%e5%97%9c%e5%a5%bd"
	savePath := "test"

	if err := getVideoByURL(pageURL, savePath); err != nil {
		log.Fatalf("程序出错: %v", err)
	}
}
