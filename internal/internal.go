package internal

import (
	"log"

	"github.com/swishcloud/gostudy/logger"
)

const (
	TimeLayout1             = "2006-01-02 15:04"
	TimeLayout2             = "15:04:05"
	TimeLayoutMysqlDateTime = "2006-01-02 15:04:05"
)

var Logger *log.Logger
var LoggerWriter *logger.FileConcurrentWriter
