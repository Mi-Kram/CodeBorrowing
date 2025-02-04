package task

import (
	"CodeBorrowing/pkg/logger"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strings"
	"time"
)

const (
	sqlWorksTable    = "sqlWorksTable"
	sqlWorkId        = "id"
	sqlWorkPath      = "path"
	sqlWorkTimestamp = "time"
)

var queryCreateTable = fmt.Sprintf("create table if not exists %s (%s integer primary key autoincrement, %s text, %s text)", sqlWorksTable, sqlWorkId, sqlWorkPath, sqlWorkTimestamp)
var queryGetWork = fmt.Sprintf("select %s, %s, %s from %s where %s = $1", sqlWorkId, sqlWorkPath, sqlWorkTimestamp, sqlWorksTable, sqlWorkId)
var querySaveWork = fmt.Sprintf("insert into %s (%s, %s) values ($1, $2)", sqlWorksTable, sqlWorkPath, sqlWorkTimestamp)
var queryUpdateWorksTimestamp = fmt.Sprintf("update %s set %s = $1 where %s in ($2)", sqlWorksTable, sqlWorkTimestamp, sqlWorkId)
var queryGetOldWorks = fmt.Sprintf("select %s, %s, %s from %s order by %s LIMIT $1", sqlWorkId, sqlWorkPath, sqlWorkTimestamp, sqlWorksTable, sqlWorkTimestamp)
var queryDeleteWorks = fmt.Sprintf("delete from %s where %s in ($1)", sqlWorksTable, sqlWorkId)

type Storage interface {
	GetWork(id uint64) (WorkEntry, error)
	SaveWork(path string, timestamp time.Time) (uint64, error)
	UpdateWorksTimestamp(ids []uint64, timestamp time.Time) error
	GetOldWorks(count uint64) ([]WorkEntry, error)
	DeleteWorks(ids []uint64) error
	Close() error
}

type storage struct {
	appLogger *logger.Logger
	db        *sql.DB
}

func NewStorage(appLogger *logger.Logger, path string) (Storage, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s/data.db", path))
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	_, err = db.Exec(queryCreateTable)
	if err != nil {
		return nil, err
	}

	data := &storage{
		appLogger: appLogger,
		db:        db,
	}

	return data, nil
}

func (s *storage) Close() error {
	if err := s.Close(); err != nil {
		return err
	}
	return nil
}

func toSqlRow(ids []uint64) string {
	return strings.Trim(strings.Join(strings.Split(fmt.Sprint(ids), " "), ", "), "[]")
}

func (s *storage) GetWork(id uint64) (WorkEntry, error) {
	res := s.db.QueryRow(queryGetWork, id)

	work := WorkEntry{}
	err := res.Scan(&work.Id, &work.Path, &work.Timestamp)
	if err != nil {
		return work, err
	}

	return work, nil
}

func (s *storage) SaveWork(path string, timestamp time.Time) (uint64, error) {
	timeStr := timestamp.Format("2006-01-02 15:04:05")
	res, err := s.db.Exec(querySaveWork, path, timeStr)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func (s *storage) UpdateWorksTimestamp(ids []uint64, timestamp time.Time) error {
	timeStr := timestamp.Format("2006-01-02 15:04:05")
	res, err := s.db.Exec(queryUpdateWorksTimestamp, toSqlRow(ids), timeStr)
	if err != nil {
		return err
	}

	if _, err = res.RowsAffected(); err != nil {
		return err
	}

	return nil
}

func (s *storage) GetOldWorks(count uint64) ([]WorkEntry, error) {
	res, err := s.db.Query(queryGetOldWorks, count)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var works []WorkEntry

	for res.Next() { // Iterate and fetch the records from result cursor
		var work WorkEntry
		err = res.Scan(&work.Id, &work.Path, &work.Timestamp)
		if err != nil {
			s.appLogger.Error(err)
			continue
		}
		works = append(works, work)
	}

	return works, nil
}

func (s *storage) DeleteWorks(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}

	idStr := strings.Trim(strings.Join(strings.Split(fmt.Sprint(ids), " "), ","), "[]")
	_, err := s.db.Exec(queryDeleteWorks, idStr)
	if err != nil {
		return err
	}
	return nil
}
