package record

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var defaultRecordFile = "record.db"

type record struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type Recorder struct {
	complete chan bool
	finish   chan struct{}
	rcrd     record
	filename string
	filemode os.FileMode
}

var r *Recorder

func init() {
	r = &Recorder{}
	r.complete = make(chan bool)
	r.finish = make(chan struct{})
	r.filename = defaultRecordFile
}

func StartRecorder() error {
	err := checkRecordFile(r.filename)
	if err != nil {
		return err
	}
	go func() {
		res := <-r.complete
		if res {
			doRecord()
		} else {
			rollback()
		}
		r.finish <- struct{}{}
	}()

	return nil
}

func Finish(result bool) {
	r.complete <- result
	<-r.finish
	close(r.complete)
	close(r.finish)
}

func doRecord() {
	t := time.Now()
	r.rcrd.Year = t.Year()
	r.rcrd.Month = int(t.Month())
	r.rcrd.Day = t.Day()
	err := updateRecordFile(r.filename)
	if err != nil {
		log.Printf("Warning: incomplete record [%s]", err.Error())
	}
}

func rollback() {
	log.Printf("Fail: No record update")
}

func checkRecordFile(filename string) error {

	state, err := os.Stat(filename)
	if err != nil {
		return err
	}
	r.filemode = state.Mode()
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &r.rcrd)
	if err != nil {
		return err
	}

	return nil
}

func updateRecordFile(filename string) error {
	data, err := json.Marshal(&r.rcrd)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, data, r.filemode)
	if err != nil {
		return err
	}
	return nil
}

func showRecord() {
	fmt.Printf("%+v\n", r)
}

func RecordDiff() int {
	old := time.Date(r.rcrd.Year, time.Month(r.rcrd.Month), r.rcrd.Day, 0, 0, 0, 0, time.Local)
	since := time.Since(old)
	diff := int(since.Hours() / 24)

	return diff
}
