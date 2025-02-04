package task

import (
	"CodeBorrowing/internal/checker"
	"CodeBorrowing/pkg/logger"
	"errors"
	"os"
	"strconv"
	"strings"
)

type Handler interface {
	Process()
}

type handler struct {
	logger  *logger.Logger
	service Service
	checker checker.Checker
}

func NewHandler(appLogger *logger.Logger, service Service, checker checker.Checker) Handler {
	return &handler{
		logger:  appLogger,
		service: service,
		checker: checker,
	}
}

func (h *handler) Process() {
	task, err := h.service.GetNewTask()
	if err != nil {
		if !errors.Is(err, NoNewTaskErr) {
			h.logger.Error(err)
		}
		return
	}

	works, err := h.service.GetEventWorks(task.EventID)
	if err != nil {
		h.logger.Error(err)
		return
	}

	if len(works) <= 1 {
		return
	}

	var newWork string
	oldWorks := make([]string, 0, len(works))
	idStr := strconv.FormatUint(task.WorkID, 10)

	for _, work := range works {
		if strings.HasPrefix(work.Path, idStr) {
			newWork = work.Path
		} else {
			oldWorks = append(oldWorks, work.Path)
		}
	}

	resultPath, err := h.checker.Run(newWork, oldWorks)
	if err != nil {
		if !errors.Is(err, checker.ErrNoFiles) {
			h.logger.Error(err)
		}
		return
	}
	defer os.Remove(resultPath)

	result, err := h.service.ParseResults(resultPath)
	if err != nil {
		h.logger.Error(err)
		return
	}

	for _, report := range result {
		if err = h.service.SendReport(report); err != nil {
			h.logger.Error(err)
			return
		}
	}

	if err = h.service.CheckCacheSize(); err != nil {
		h.logger.Error(err)
	}
}
