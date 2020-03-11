package record

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

var defaultRecordFile = "record.db"

type record struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type Recorder struct {
	complete     chan bool
	finish       chan struct{}
	rcrd         record
	filename     string
	filemode     os.FileMode
	lastDate     *time.Time
	newDate      *time.Time
	prevTurnDate *time.Time
	initErr      error
}

var r *Recorder

func InitRecorder() {
	r = &Recorder{}
	r.complete = make(chan bool)
	r.finish = make(chan struct{})
	r.filename = defaultRecordFile
	r.lastDate = &time.Time{}

	r.initErr = checkRecordFile(r.filename)
	if r.initErr != nil {
		r = nil
		return
	}
	*r.lastDate = time.Date(r.rcrd.Year, time.Month(r.rcrd.Month), r.rcrd.Day, 0, 0, 0, 0, time.Local)
}

func LastDate() *time.Time {
	return r.lastDate
}

func StartRecorder() {
	if r == nil {
		return
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
}

func Finish(result bool, newDate *time.Time) {
	if r == nil {
		return
	}
	r.newDate = newDate
	r.complete <- result
	<-r.finish
	close(r.complete)
	close(r.finish)
}

func ProcessDate(date int) bool {
	if r == nil {
		return true
	}
	dateStr := strconv.FormatInt(int64(date), 10)
	y, _ := strconv.ParseInt(dateStr[:4], 10, 32)
	m, _ := strconv.ParseInt(dateStr[4:6], 10, 32)
	d, _ := strconv.ParseInt(dateStr[6:], 10, 32)
	thisTurn := time.Date(int(y), time.Month(m), int(d), 0, 0, 0, 0, time.Local)
	if r.newDate == nil || thisTurn.After(*r.newDate) {
		r.newDate = &thisTurn
	}
	if thisTurn.Before(*r.lastDate) {
		return false
	}
	return true
}

func doRecord() {
	var err error
	defer func() {
		if err != nil {
			log.Printf("Warning: incomplete record [%s]", err.Error())
		}
	}()
	if r.newDate == nil {
		err = fmt.Errorf("undetermined record date")
		return
	}
	t := r.newDate
	r.rcrd.Year = t.Year()
	r.rcrd.Month = int(t.Month())
	r.rcrd.Day = t.Day()
	err = updateRecordFile(r.filename)
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
	since := time.Since(*r.lastDate)
	diff := int(since.Hours() / 24)

	return diff
}
