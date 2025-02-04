package main

import (
	"CodeBorrowing/internal/config"
	"CodeBorrowing/internal/router"
	"CodeBorrowing/internal/task"
	"CodeBorrowing/pkg/logger"
	"CodeBorrowing/pkg/shutdown"
	"fmt"
	"os"
	"syscall"
	"time"
)

func main() {
	// Чтение переменных окружающей среды.
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Инициализация логгера.
	appLogger := logger.GetLogger(cfg.Logs)
	appLogger.Debug("Logger initialized")

	// Инициализация хранилища работ студентов.
	taskStorage, err := task.NewStorage(appLogger, cfg.Storage)
	if err != nil {
		appLogger.Error(err)
		return
	}
	appLogger.Debug("Task storage initialized")

	router.InitializeHost(cfg.MainServerHost, cfg.MainServerKey)

	// Сервис и обработчик для обработки работ студентов.
	taskService, err := task.NewService(taskStorage, appLogger, cfg.Storage, cfg.StorageSize)
	if err != nil {
		appLogger.Error(err)
		return
	}
	taskHandler := task.NewHandler(appLogger, taskService)

	quit := make(chan interface{})               // Сюда придёт сигнал, что надо завершить приложение.
	scheduler := time.NewTicker(5 * time.Second) // Будильник для проверки новой задачи.
	isRunning := true                            // Статус приложение (работает / не работает).

	// Перехватываем сигнал завершения приложения.
	go shutdown.Graceful(appLogger, []os.Signal{syscall.SIGABRT, syscall.SIGQUIT, syscall.SIGHUP,
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL}, quit)

	appLogger.Info("Program is running")

	for isRunning {
		select {
		case <-quit:
			scheduler.Stop()
			isRunning = false
		case <-scheduler.C:
			taskHandler.Process()
		}
	}

	appLogger.Info("Finishing the program")

	_ = taskStorage.Close()
	_ = appLogger.Close()
	// TODO: Save data
}
