package record

import "testing"

func TestCheckRecordFile(t *testing.T) {
	err := checkRecordFile("record.db.json")
	if err != nil {
		t.Error(err)
	} else {
		showRecord()
	}
}
