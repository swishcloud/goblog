package internal

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/swishcloud/gostudy/logger"
)

const (
	TimeLayout1             = "2006-01-02 15:04"
	TimeLayout2             = "15:04:05"
	TimeLayoutMysqlDateTime = "2006-01-02 15:04:05"
)

var Logger *log.Logger
var LoggerWriter *logger.FileConcurrentWriter

func FormatTime(request *http.Request, t time.Time, layout string) (string, error) {
	if layout == "" {
		layout = "01/02/2006 15:04"
	}
	tom := 0
	c, err := request.Cookie("tom")
	if err == nil {
		tom, err = strconv.Atoi(c.Value)
		if err != nil {
			panic(err)
		}
	}
	t = t.Add(-time.Duration(int64(time.Minute) * int64(tom)))
	return t.Format(layout), nil
}
