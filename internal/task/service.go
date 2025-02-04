package task

import (
	"CodeBorrowing/internal/router"
	"CodeBorrowing/internal/utils"
	"CodeBorrowing/pkg/logger"
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	urlGetNewTask  = "/api/newtask"
	urlGetAllWorks = "/api/works"
	urlGetWorksUrl = "/api/worksurl"
	urlPostReport  = "/api/crossreport"
)

var NoNewTaskErr = errors.New("no new task")

type Service interface {
	GetNewTask() (NewTaskDTO, error)
	GetEventWorks(eventId uint64) ([]WorkEntry, error)
	ParseResults(path string) ([]ReportItem, error)
	SendReport(report ReportItem) error
	CheckCacheSize() error
}

type service struct {
	storage Storage
	logger  *logger.Logger
	root    string
	size    uint64
}

func NewService(taskStorage Storage, logger *logger.Logger, path string, size uint64) (Service, error) {
	_, err := utils.CreateDirectory(path)
	if err != nil {
		return nil, err
	}

	if size < 50 {
		return nil, errors.New("too small storage size")
	}

	return &service{
		storage: taskStorage,
		logger:  logger,
		root:    path,
		size:    size,
	}, nil
}

func makeGetNewTaskRequest() (*http.Response, error) {
	req, err := router.NewRequest(http.MethodGet, urlGetNewTask, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	return client.Do(req)
}

func makeGetDownloadUrlsRequest(ids []uint64) (*http.Response, error) {
	req, err := router.NewRequest(http.MethodGet, urlGetWorksUrl, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for _, id := range ids {
		q.Add("id", strconv.FormatUint(id, 10))
	}
	req.URL.RawQuery = q.Encode()

	client := http.Client{}
	return client.Do(req)
}

func (s *service) getWorkPath(workID uint64) string {
	return fmt.Sprintf("%s/works/%d", s.root, workID)
}

func (s *service) GetNewTask() (NewTaskDTO, error) {
	var result NewTaskDTO

	res, err := makeGetNewTaskRequest()
	if err != nil {
		return result, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		return result, NoNewTaskErr
	}

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return result, errors.New(string(body))
	}

	if err = json.Unmarshal(body, &result); err != nil {
		return result, err
	}

	return result, nil
}

func (s *service) getWorksId(eventId uint64) ([]uint64, error) {
	req, err := router.NewRequest(http.MethodGet, urlGetAllWorks, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("id", strconv.FormatUint(eventId, 10))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}

	var works WorksIdDTO
	if err = json.Unmarshal(body, &works); err != nil {
		return nil, err
	}

	return works.List, nil
}

func (s *service) getWorksEntry(ids []uint64) (works []WorkEntry, notFound []uint64) {
	works = make([]WorkEntry, 0, len(ids))
	notFound = make([]uint64, 0, len(ids))

	for _, id := range ids {
		work, err := s.storage.GetWork(id)
		if err != nil {
			notFound = append(notFound, id)
		} else {
			works = append(works, work)
		}
	}

	if err := s.storage.UpdateWorksTimestamp(ids, time.Now()); err != nil {
		s.logger.Error(err)
	}

	return
}

func getDownloadUrls(ids []uint64) ([]WorkUrlDTO, error) {
	var urls WorksUrlDTO

	res, err := makeGetDownloadUrlsRequest(ids)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}

	if err = json.Unmarshal(body, &urls); err != nil {
		return nil, err
	}

	return urls.Works, nil
}

func (s *service) downloadWorks(ids []uint64) ([]WorkEntry, error) {
	if len(ids) == 0 {
		return []WorkEntry{}, nil
	}

	urls, err := getDownloadUrls(ids)
	if err != nil {
		return nil, err
	}

	result := make([]WorkEntry, 0, len(ids))
	for _, url := range urls {
		work, err := s.downloadWork(url.WorkID, url.Url)
		if err != nil {
			s.logger.Error(err)
		} else {
			result = append(result, work)
		}
	}

	return result, err
}

func prepareWorkDirectory(path string) error {
	existed, err := utils.CreateDirectory(path)

	if err != nil {
		return err
	}

	if existed {
		if err = utils.ClearDirectory(path); err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	buf := bytes.NewBuffer(make([]byte, res.ContentLength))
	_, err = io.Copy(buf, req.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *service) unzipWork(path string, buf []byte) error {
	reader := bytes.NewReader(buf)
	zipReader, err := zip.NewReader(reader, int64(len(buf)))
	if err != nil {
		return err
	}

	for _, f := range zipReader.File {
		rc, err := f.Open()
		if err != nil {
			s.logger.Error(err)
			continue
		}
		defer rc.Close()

		newFilePath := fmt.Sprintf("%s/%s", path, f.Name)

		// Directory case
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(newFilePath, os.ModePerm)
			if err != nil {
				s.logger.Error(err)
			}
			continue
		}

		// File case
		if destFile, err := os.Create(newFilePath); err != nil {
			s.logger.Error(err)
		} else if _, err = io.Copy(destFile, rc); err != nil {
			s.logger.Error(err)
		}
	}

	return nil
}

func (s *service) downloadWork(id uint64, url string) (WorkEntry, error) {
	work := WorkEntry{
		Path:      s.getWorkPath(id),
		Timestamp: time.Now(),
	}

	unzipPath := fmt.Sprintf("%s/%s", work.Path, strconv.FormatUint(id, 10))
	if err := prepareWorkDirectory(unzipPath); err != nil {
		return work, err
	}

	buf, err := downloadFile(url)
	if err != nil {
		return work, err
	}

	if err = s.unzipWork(unzipPath, buf); err != nil {
		return work, err
	}

	work.Id, err = s.storage.SaveWork(work.Path, work.Timestamp)
	if err != nil {
		return work, nil
	}

	return work, nil
}

func (s *service) GetEventWorks(eventId uint64) ([]WorkEntry, error) {
	ids, err := s.getWorksId(eventId)
	if err != nil {
		return nil, err
	}

	works, notFound := s.getWorksEntry(ids)
	if len(notFound) == 0 {
		return works, nil
	}

	downloaded, err := s.downloadWorks(notFound)
	if err != nil {
		s.logger.Error(err)
		return works, nil
	}

	works = append(works, downloaded...)
	return works, nil
}

func (s *service) ParseResults(path string) ([]ReportItem, error) {
	// TODO: parse results
	return nil, errors.New("not implemented")
}

func (s *service) SendReport(report ReportItem) error {
	jsonBytes, err := json.Marshal(report)
	if err != nil {
		return err
	}

	data := bytes.NewBuffer(jsonBytes)
	req, err := router.NewRequest(http.MethodPost, urlPostReport, data)
	if err != nil {
		return err
	}

	client := http.Client{}
	if _, err = client.Do(req); err != nil {
		return err
	}

	return nil
}

func (s *service) removeOldWorks() (uint64, error) {
	works, err := s.storage.GetOldWorks(10)
	if err != nil || len(works) == 0 {
		return 0, err
	}

	var removed uint64 = 0

	for _, work := range works {
		rm, err := utils.GetDirectorySize(work.Path)
		if err != nil {
			s.logger.Error(err)
			continue
		}

		err = os.RemoveAll(work.Path)
		if err != nil {
			fmt.Println(err)
		}

		removed += rm
	}

	ids := make([]uint64, len(works))
	for i, work := range works {
		ids[i] = work.Id
	}

	if err = s.storage.DeleteWorks(ids); err != nil {
		return removed, nil
	}

	return removed, nil
}

func (s *service) CheckCacheSize() error {
	size, err := utils.GetDirectorySize(s.root)
	if err != nil {
		return err
	}

	for s.size < size/1024/1024 {
		removed, err := s.removeOldWorks()
		if err != nil {
			s.logger.Error(err)
			return err
		}

		size -= removed
	}

	return nil
}
