package bing

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
)

// AppConfig contains app's config
type AppConfig struct {
	PicDir      string `json:"pic_dir"`
	PicNumber   int    `json:"pic_number"`
	DaysBehind  int    `json:"days_behind"`
	UseRecordDB bool   `json:"use_record_db"`
}

func newAppConfig() *AppConfig {
	return &AppConfig{
		PicDir:      "background",
		PicNumber:   1,
		DaysBehind:  0,
		UseRecordDB: false,
	}
}

var cfg *AppConfig

func init() {
	var configPath string
	var picNumber int
	var daysBehind int
	var useRecord bool
	progDir := filepath.Dir(os.Args[0])
	flag.StringVar(&configPath, "c", progDir+"/config.json", "")
	flag.IntVar(&picNumber, "n", 1, "pic number")
	flag.IntVar(&daysBehind, "d", 0, "days behind today")
	flag.BoolVar(&useRecord, "ur", false, "using record db, discard -n -d")
	flag.Parse()

	cfg = newAppConfig()
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		panic(err)
	}
	if !useRecord {
		if isFlagPassed("n") {
			cfg.PicNumber = picNumber
		}
		if isFlagPassed("d") {
			cfg.DaysBehind = daysBehind
		}
	} else {
		cfg.UseRecordDB = useRecord
	}
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func GetConfig() *AppConfig {
	return cfg
}
