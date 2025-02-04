package logger

import (
	"CodeBorrowing/internal/utils"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
)

type myFormatter struct {
	logrus.TextFormatter
}

type Logger struct {
	logrus.Logger
	file *os.File
}

func (f *myFormatter) Format(e *logrus.Entry) ([]byte, error) {
	format := "[%s] - [%s] - %s%s%s\n"
	timeTag := e.Time.Format("02.01.2006 15:04:05")
	caller, data := "", ""

	if e.Caller != nil && (e.Level == logrus.ErrorLevel ||
		e.Level == logrus.FatalLevel || e.Level == logrus.PanicLevel) {
		caller = fmt.Sprintf("(%s) - ", f.extractCallFunction(e.Caller))
	}

	if e.Data != nil && len(e.Data) != 0 {
		sb := strings.Builder{}
		for k, v := range e.Data {
			sb.WriteString(fmt.Sprintf("%s:%v; ", k, v))
		}

		s := sb.String()
		data = fmt.Sprintf(" {%s}", s[:len(s)-2])
	}

	return []byte(fmt.Sprintf(format, timeTag, e.Level, caller, e.Message, data)), nil
}

func (f *myFormatter) extractCallFunction(caller *runtime.Frame) string {
	count := 0
	idx := strings.LastIndexFunc(caller.File, func(r rune) bool {
		if r == '/' || r == '\\' {
			count++
			if count == 2 {
				return true
			}
		}
		return false
	})

	idx++
	return fmt.Sprintf("%s:%d", caller.File[idx:], caller.Line)
}

var instance *Logger
var once = sync.Once{}

func GetLogger(path string) *Logger {
	once.Do(func() {
		instance = &Logger{
			Logger: *logrus.New(),
		}
		loggerInit(instance, path)
	})
	return instance
}

func loggerInit(log *Logger, path string) {
	log.SetLevel(logrus.DebugLevel)  // all logs
	log.SetReportCaller(true)        // info about function-caller
	log.SetFormatter(&myFormatter{}) // custom output format

	log.SetOutput(io.Discard) // Remove all outputs

	log.AddHook(&writer.Hook{
		Writer:    os.Stdout,
		LogLevels: logrus.AllLevels,
	})

	if _, err := utils.CreateDirectory(path); err != nil {
		panic(err)
	}

	timeTag := time.Now().Format("2006_01_02_15_04_05") // YYYY_MM_dd_HH_mm_ss
	logFileName := fmt.Sprintf("%s/%s.log", path, timeTag)
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}

	log.AddHook(&writer.Hook{
		Writer: file,
		LogLevels: []logrus.Level{logrus.InfoLevel, logrus.WarnLevel,
			logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel},
	})

	log.file = file
}

func (log *Logger) Close() error {
	if err := log.file.Sync(); err != nil {
		return err
	}
	if err := log.file.Close(); err != nil {
		return err
	}
	return nil
}
