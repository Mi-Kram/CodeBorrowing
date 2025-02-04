package checker

import (
	"CodeBorrowing/pkg/logger"
	"errors"
	"os"
	"os/exec"
	"strings"
)

var ErrNoFiles = errors.New("no files for comparison")

type Checker interface {
	Run(newWork string, oldWorks []string) (string, error)
}

type checkerT struct {
	logger      *logger.Logger
	checkerPath string
	resultPath  string
}

func NewChecker(appLogger *logger.Logger, checker string, result string) Checker {
	return &checkerT{
		logger:      appLogger,
		checkerPath: checker,
		resultPath:  result,
	}
}

func (c *checkerT) Run(newWork string, oldWorks []string) (string, error) {
	if newWork == "" || len(oldWorks) == 0 {
		return "", ErrNoFiles
	}

	if info, err := os.Stat(c.resultPath); err != nil {
		if info != nil && !info.IsDir() {
			if err = os.Remove(c.resultPath); err != nil {
				return "", err
			}
		}
	}

	oldWorksStr := strings.Join(oldWorks, ",")
	cmd := exec.Command("java", "-jar", c.checkerPath, newWork, "-l", "csharp", "-r", c.resultPath, "-old", oldWorksStr)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return c.resultPath, nil
}
