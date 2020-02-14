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
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uccismorph/bing-background-go/record"
)

type Context struct {
	client       *http.Client
	cfg          *AppConfig
	processedSet map[int]bool
	leftNum      int
	completeNum  int64
}

func NewContext() *Context {
	p := &Context{
		client: &http.Client{
			Transport: &http.Transport{
				Dial: func(network, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, 5*time.Second)
				},
			},
			Timeout: 30 * time.Second,
		},
		cfg:          GetConfig(),
		processedSet: make(map[int]bool),
	}
	if cfg.UseRecordDB {
		record.InitRecorder()
		p.cfg.DaysBehind = 0
		p.cfg.PicNumber = record.RecordDiff()
	}

	err := os.MkdirAll(p.cfg.PicDir, 0755)
	if err != nil {
		msg := fmt.Sprintf("cannot mkdir: %s", err.Error())
		panic(msg)
	}
	p.leftNum = p.cfg.PicNumber

	return p
}

func (p *Context) SetConfig(cfg *AppConfig) {
	p.cfg = cfg
}

var defaultTurnSize = 5

var bingArchive string = "http://www.bing.com/HPImageArchive.aspx"

func composeArchiveURL(behind, picNum int) *url.URL {
	res, _ := url.Parse(bingArchive)

	queryString := res.Query()
	queryString.Add("format", "xml")
	queryString.Add("idx", strconv.FormatInt(int64(behind), 10))
	queryString.Add("n", strconv.FormatInt(int64(picNum), 10))
	queryString.Add("mkt", "ZH-CN")
	res.RawQuery = queryString.Encode()

	return res
}

func convertDate(date int) *time.Time {
	dateStr := strconv.FormatInt(int64(date), 10)
	y, _ := strconv.ParseInt(dateStr[:4], 10, 32)
	m, _ := strconv.ParseInt(dateStr[4:6], 10, 32)
	d, _ := strconv.ParseInt(dateStr[6:], 10, 32)
	res := time.Date(int(y), time.Month(m), int(d), 0, 0, 0, 0, time.Local)
	return &res
}

type errorState struct {
	result  bool
	errorAt int
}

func (p *Context) processArchive(url *url.URL) bool {
	log.Printf("calling %s", url.String())
	desc, err := p.retriveDesc(url)
	if err != nil {
		log.Printf("retrive pic desc error: %s", err.Error())
		return false
	}
	log.Printf("schduled pic num: %d", len(desc.Images))
	wg := sync.WaitGroup{}
	state := make(chan errorState, len(desc.Images))
	for i, _ := range desc.Images {
		if p.hasProcessed(desc.Images[i].StartDate) {
			log.Printf("pic[%d] has been processed", desc.Images[i].StartDate)
			p.leftNum -= 1
			continue
		}
		log.Printf("download[%d]: %s, %s", desc.Images[i].StartDate, desc.Images[i].CopyRight, desc.Images[i].PicURL)
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

			atomic.AddInt64(&p.completeNum, 1)
		}(i)
		p.leftNum -= 1
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

func (p *Context) hasProcessed(date int) bool {
	if _, ok := p.processedSet[date]; ok {
		return true
	}
	p.processedSet[date] = true
	return false
}

func (p *Context) Run() {
	res := true
	record.StartRecorder()
	for {
		archiveURL, remain := p.remainingArchive()
		if !remain {
			break
		}
		if res = p.processArchive(archiveURL); !res {
			break
		}
	}
	record.Finish(res)
	log.Printf("total complete: %d", p.completeNum)
}

func (p *Context) remainingArchive() (*url.URL, bool) {
	if p.leftNum <= 0 {
		return nil, false
	}
	processNum := p.cfg.PicNumber - p.leftNum
	res := composeArchiveURL(p.cfg.DaysBehind+processNum, p.leftNum)
	return res, true
}

func (p *Context) retriveDesc(url *url.URL) (*PictureArchive, error) {
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
	picDesc := &PictureArchive{}
	err = xml.Unmarshal(data, picDesc)
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(picDesc))
	return picDesc, nil
}

func (p *Context) download(picURL string) error {
	bingSite := "http://cn.bing.com"
	if !strings.HasPrefix(picURL, "http") {
		picURL = bingSite + picURL
	} else {
		raw, err := url.Parse(picURL)
		if err != nil {
			return err
		}
		picURL = bingSite + raw.RequestURI()
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

// PictureArchive xxx
type PictureArchive struct {
	XMLName xml.Name       `xml:"images"`
	Images  []*PictureDesc `xml:"image"`
}

// PictureDesc xxx
type PictureDesc struct {
	PicURL    string `xml:"url"`
	StartDate int    `xml:"startdate"`
	HeadLine  string `xml:"headline"`
	CopyRight string `xml:"copyright"`
}

func (p *PictureArchive) Len() int {
	return len(p.Images)
}

func (p *PictureArchive) Less(i, j int) bool {
	if p.Images[i].StartDate-p.Images[j].StartDate < 0 {
		return true
	}
	return false
}

func (p *PictureArchive) Swap(i, j int) {
	p.Images[i], p.Images[j] = p.Images[j], p.Images[i]
	log.Printf("i = %d, startdate: %d", i, p.Images[i].StartDate)
	log.Printf("j = %d, startdate: %d", j, p.Images[j].StartDate)
}
