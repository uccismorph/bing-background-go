package bing

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Picture xxx
type Picture struct {
	client *http.Client
	cfg    *AppConfig
	url    *url.URL
}

// NewPicture xxx
func NewPicture() *Picture {
	p := &Picture{
		client: &http.Client{
			Transport: &http.Transport{
				Dial: func(network, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, 5*time.Second)
				},
			},
			Timeout: 10 * time.Second,
		},
		cfg: GetConfig(),
	}

	err := os.MkdirAll(p.cfg.PicDir, 0755)
	if err != nil {
		msg := fmt.Sprintf("cannot mkdir: %s", err.Error())
		panic(msg)
	}
	p.url, err = url.Parse("http://www.bing.com/HPImageArchive.aspx")
	if err != nil {
		panic(err)
	}
	queryString := p.url.Query()
	queryString.Add("format", "xml")
	queryString.Add("idx", strconv.FormatUint(p.cfg.DaysBehind, 10))
	queryString.Add("n", strconv.FormatUint(p.cfg.PicNumber, 10))
	queryString.Add("mkt", "ZH-CN")
	p.url.RawQuery = queryString.Encode()

	return p
}

// Run xxx
func (p *Picture) Run() {
	log.Printf("calling %s", p.url.String())
	desc, err := p.retriveDesc()
	if err != nil {
		log.Printf("retrive pic desc error: %s", err.Error())
		return
	}
	for i, _ := range desc.Images {
		err = p.download(desc.Images[i].PicURL)
		if err != nil {
			log.Printf("download pic error: %s", err.Error())
			return
		}
	}
}

func (p *Picture) retriveDesc() (*PictureDesc, error) {
	resp, err := p.client.Get(p.url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error: %s", err.Error())
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	picDesc := &PictureDesc{}
	err = xml.Unmarshal(data, picDesc)
	if err != nil {
		return nil, err
	}
	return picDesc, nil
}

func (p *Picture) download(picURL string) error {
	if !strings.HasPrefix(picURL, "http") {
		picURL = "http://www.bing.com" + picURL
	}
	pic, err := url.Parse(picURL)
	if err != nil {
		return err
	}
	picIDs := strings.Split(pic.Query().Get("id"), ".")
	picName := ""
	for i, _ := range picIDs {
		if picIDs[i] == "jpg" {
			picName = strings.Join(picIDs[i-1:i+1], ".")
			break
		}
	}
	picName = p.cfg.PicDir + "/" + picName
	resp, err := p.client.Get(picURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server error: %s", err.Error())
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(picName, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// PictureDesc xxx
type PictureDesc struct {
	XMLName xml.Name           `xml:"images"`
	Images  []PictureDescImage `xml:"image"`
}

// PictureDescImage xxx
type PictureDescImage struct {
	PicURL string `xml:"url"`
}
