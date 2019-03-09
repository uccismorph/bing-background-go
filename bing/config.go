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
	PicDir     string `json:"pic_dir"`
	PicNumber  uint64 `json:"pic_number"`
	DaysBehind uint64 `json:"days_behind"`
}

func newAppConfig() *AppConfig {
	return &AppConfig{
		PicDir:     "background",
		PicNumber:  1,
		DaysBehind: 0,
	}
}

var cfg *AppConfig

func init() {
	var configPath string
	var picNumber uint64
	var daysBehind uint64
	progDir := filepath.Dir(os.Args[0])
	flag.StringVar(&configPath, "c", progDir+"/config.json", "")
	flag.Uint64Var(&picNumber, "n", 1, "pic number")
	flag.Uint64Var(&daysBehind, "d", 0, "days behind today")
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
	if isFlagPassed("n") {
		cfg.PicNumber = picNumber
	}
	if isFlagPassed("d") {
		cfg.DaysBehind = daysBehind
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
