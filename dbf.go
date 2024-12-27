package godbf

import (
	"github.com/axgle/mahonia"
	"os"
	"sync"
)

type DBF interface {
	ReloadFromFile() error
	NumRecords() uint32
	GetRecord(index uint32, v interface{}) error
	GetRecords(start, end uint32, v interface{}, workerNums int) error
	Append(model interface{}) error
}

type DBFHandler struct {
	mu       sync.RWMutex
	fileName string
	f        *os.File
	fileSize int64
	encoder  mahonia.Encoder
	decoder  mahonia.Decoder
	header   DBFHeader

	fields           []FieldDescriptor
	columns          []string
	modelColumnIndex map[string]int
}

const (
	SPACE = 0x20
	EOF   = 0x1A
	NUL   = 0x00
)

func NewDBFFromFile(fileName string, encoding string, model interface{}) (*DBFHandler, error) {
	f, err := os.OpenFile(fileName, os.O_RDWR, 0770)
	if err != nil {
		return nil, err
	}
	fileStat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	fileStat.Size()
	dbf := &DBFHandler{
		fileName: fileName,
		mu:       sync.RWMutex{},
		encoder:  mahonia.NewEncoder(encoding),
		decoder:  mahonia.NewDecoder(encoding),
		f:        f,
		fileSize: fileStat.Size(),
		header:   DBFHeader{},
	}
	err = dbf.initMetaData(model)
	return dbf, err
}

func (dbf *DBFHandler) ReloadFromFile() error {
	err := dbf.f.Close()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(dbf.fileName, os.O_RDWR, 0770)
	if err != nil {
		return err
	}
	dbf.f = f
	err = dbf.initHeader()
	return err
}

func (dbf *DBFHandler) NumRecords() uint32 {
	return dbf.header.NumRecords
}
