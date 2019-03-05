package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bold-commerce/go-shopify"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	httpClient                             = http.Client{}
	UUID                                   string
	port, ip                               string
	apiKey, apiSecret, domain, accessToken string
	logoPath                               string
)

func main() {
	flag.StringVar(&apiKey, "api_key", "", "Shopify API Key")
	flag.StringVar(&apiSecret, "api_secret", "", "Shopify API Secret")
	flag.StringVar(&domain, "domain", "", "Shopify domain")
	flag.StringVar(&accessToken, "access_token", "", "Shopify accesstoken")
	flag.StringVar(&port, "port", "56234", "BTT Port")
	flag.StringVar(&ip, "ip", "127.0.0.1", "BTT IP")
	flag.StringVar(&UUID, "uuid", "", "BTT widget's UUID")
	flag.Parse()
	app := goshopify.App{
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
	}
	client := goshopify.NewClient(app, domain, accessToken)

	var err error
	logoPath, err = downloadLogo(domain)
	if err != nil {
		log.Printf("Can not get logo: %s\n", err)
	}

	updateData(client)
	ticker := time.NewTicker(1 * time.Minute)
	for _ = range ticker.C {
		updateData(client)
	}
}

func updateData(client *goshopify.Client) {
	today := time.Now()
	todayOrders, err := client.Order.Count(goshopify.OrderListOptions{
		CreatedAtMin: time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()),
		CreatedAtMax: today,
	})
	if err != nil {
		log.Panicln(err)
		return
	}
	if err := doRequest(fmt.Sprintf("Orders today: %d", todayOrders)); err != nil {
		log.Panicln(err)
	}
}

func downloadLogo(domain string) (string, error) {
	iconUrl, err := getFavicon(domain)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(iconUrl)
	filePath, err := filepath.Abs(fmt.Sprintf("./%s%s", domain, ext))
	if err != nil {
		return "", err
	}
	out, err := os.Create(filePath)
	defer out.Close()

	resp, err := http.Get(iconUrl)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}
	return filePath, nil
}

func getFavicon(domain string) (string, error) {
	response, err := httpClient.Get(fmt.Sprintf("http://favicongrabber.com/api/grab/%s", domain))
	if err != nil {
		return "", err
	}
	body, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return "", err
	}
	result := struct {
		Icons []*struct {
			Src string `json:"src"`
		} `json:"icons"`
	}{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if len(result.Icons) == 0 {
		return "", errors.New("Icon not found")
	}
	return result.Icons[0].Src, nil
}

func doRequest(text string) error {
	url := fmt.Sprintf("http://%s:%s/update_touch_bar_widget/?uuid=%s&text=%s&icon_path=%s", ip, port, UUID, text, logoPath)
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Response status: %v, %s", resp.Status, url)
	}
	return nil
}
