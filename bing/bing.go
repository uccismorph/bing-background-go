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
	"sync"
	"time"

	"github.com/uccismorph/bing-background-go/record"
)

// Picture xxx
type Picture struct {
	client *http.Client
	cfg    *AppConfig
	urls   []*url.URL
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
			Timeout: 30 * time.Second,
		},
		cfg: GetConfig(),
	}
	if cfg.UseRecordDB {
		err := record.StartRecorder()
		if err != nil {
			msg := fmt.Sprintf("recorde db error: %s", err.Error())
			panic(msg)
		}
		p.cfg.DaysBehind = 0
		p.cfg.PicNumber = record.RecordDiff()
	}

	err := os.MkdirAll(p.cfg.PicDir, 0755)
	if err != nil {
		msg := fmt.Sprintf("cannot mkdir: %s", err.Error())
		panic(msg)
	}

	p.genURLS(int(p.cfg.PicNumber))

	return p
}

var defaultTurnSize = 5

func (p *Picture) genURLS(total int) {
	leftNum := total
	oneTurnNum := 0
	daysBehind := p.cfg.DaysBehind
	for leftNum > 0 {
		url, err := url.Parse("http://www.bing.com/HPImageArchive.aspx")
		if err != nil {
			panic(err)
		}
		if leftNum > defaultTurnSize {
			oneTurnNum = defaultTurnSize
		} else {
			oneTurnNum = leftNum
		}
		queryString := url.Query()
		queryString.Add("format", "xml")
		queryString.Add("idx", strconv.FormatInt(int64(daysBehind), 10))
		queryString.Add("n", strconv.FormatInt(int64(oneTurnNum), 10))
		queryString.Add("mkt", "ZH-CN")
		url.RawQuery = queryString.Encode()
		p.urls = append(p.urls, url)
		leftNum -= defaultTurnSize
		daysBehind += oneTurnNum
	}
}

type errorState struct {
	result  bool
	errorAt int
}

// Run xxx
func (p *Picture) run(url *url.URL) bool {
	log.Printf("calling %s", url.String())
	desc, err := p.retriveDesc(url)
	if err != nil {
		log.Printf("retrive pic desc error: %s", err.Error())
		return false
	}
	log.Printf("total pic num: %d", len(desc.Images))
	wg := sync.WaitGroup{}
	state := make(chan errorState, len(desc.Images))
	for i, _ := range desc.Images {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err = p.download(desc.Images[i].PicURL)
			if err != nil {
				log.Printf("download pic error: %s", err.Error())
				state <- errorState{
					result:  false,
					errorAt: i,
				}
				return
			}
			state <- errorState{
				result:  true,
				errorAt: -1,
			}
		}(i)
	}
	wg.Wait()
	close(state)
	res := true
	for s := range state {
		if !s.result {
			res = false
		}
	}
	return res
}

func (p *Picture) Run() {
	res := true
	if len(p.urls) == 0 {
		log.Printf("picture is up to date")
	}
	for _, url := range p.urls {
		if res = p.run(url); !res {
			break
		}
	}
	if cfg.UseRecordDB {
		record.Finish(res)
	}
}

func (p *Picture) retriveDesc(url *url.URL) (*PictureDesc, error) {
	resp, err := p.client.Get(url.String())
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
		picURL = "http://cn.bing.com" + picURL
	} else {
		raw, err := url.Parse(picURL)
		if err != nil {
			return err
		}
		picURL = "http://cn.bing.com" + raw.RequestURI()
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
		return fmt.Errorf("[%s] http error: %s", picURL, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server error: %s", err.Error())
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[%s] read body error: %s", picURL, err.Error())
	}
	err = ioutil.WriteFile(picName, data, 0644)
	if err != nil {
		return err
	}
	log.Printf("finish pic file: %s", picName)
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
