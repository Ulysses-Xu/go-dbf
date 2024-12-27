package godbf

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type workerArgs struct {
	index uint32
	value reflect.Value
}

func (dbf *DBFHandler) getRecord(index uint32, rv reflect.Value) error {
	// todo 支持软删除
	// +1 是为了跳过删除位
	start := uint32(dbf.header.HeaderLength) + uint32(dbf.header.RecordLength)*index + 1
	data := make([]byte, dbf.header.RecordLength-1)
	//end := start + uint32(dbf.header.RecordLength)
	_, err := dbf.f.ReadAt(data, int64(start))
	if err != nil {
		return errors.New("Failed to read record. ")
	}

	pos := 0
	for i := 0; i < len(dbf.columns); i++ {
		fieldIndex := dbf.modelColumnIndex[dbf.columns[i]]
		fieldValue := rv.Field(fieldIndex)
		columnLength := int(dbf.fields[fieldIndex].Length)
		columnVal := strings.TrimSpace(string(data[pos : pos+columnLength]))
		if columnVal == "" {
			pos += columnLength
			continue
		}
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(dbf.decoder.ConvertString(columnVal))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			num, err := strconv.ParseInt(columnVal, 10, 64)
			if err != nil {
				return err
			}
			fieldValue.SetInt(num)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			num, err := strconv.ParseUint(columnVal, 10, 64)
			if err != nil {
				return err
			}
			fieldValue.SetUint(num)
		case reflect.Float32, reflect.Float64:
			num, err := strconv.ParseFloat(columnVal, 64)
			if err != nil {
				return err
			}
			fieldValue.SetFloat(num)
		}
		pos += columnLength
	}
	return nil
}

func (dbf *DBFHandler) GetRecord(index uint32, v interface{}) error {
	dbf.mu.RLock()
	defer dbf.mu.RUnlock()
	if index >= dbf.header.NumRecords {
		return errors.New("index out of range")
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("GetRecord requires a non-nil pointer to a struct")
	}
	rv = rv.Elem()

	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("GetRecord requires a pointer to a struct, not a %s", rv.Kind())
	}

	err := dbf.getRecord(index, rv)
	return err
}

func (dbf *DBFHandler) startWorker(workerChan []chan workerArgs, errChan chan error, wg *sync.WaitGroup) {
	for i := 0; i < len(workerChan); i++ {
		workerChan[i] = make(chan workerArgs, 16)
		go dbf.work(workerChan[i], errChan, wg)
	}
}

func (dbf *DBFHandler) work(taskChan <-chan workerArgs, errChan chan<- error, wg *sync.WaitGroup) {
	for args := range taskChan {
		err := dbf.getRecord(args.index, args.value)
		errChan <- err
		wg.Done()
	}
}

func (dbf *DBFHandler) GetRecords(start, end uint32, v interface{}, workerNums int) error {
	dbf.mu.RLock()
	defer dbf.mu.RUnlock()

	if end > dbf.header.NumRecords {
		return errors.New("index out of range")
	}

	rt := reflect.TypeOf(v)
	if rt.Kind() != reflect.Ptr {
		return fmt.Errorf("GetRecords requires a pointer to a slice, not a %s", rt.Kind())
	}

	if rt.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("GetRecords requires a pointer to a slice, not a %s", rt.Elem().Kind())
	}

	if rt.Elem().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("GetRecords requires a pointer to a slice of struct, not a %s", rt.Elem().Elem().Kind())
	}

	rv := reflect.ValueOf(v).Elem()
	if rv.Len() < int(end-start) {
		return errors.New("value out of range")
	}

	wg := sync.WaitGroup{}
	workerChan := make([]chan workerArgs, workerNums)
	errChan := make(chan error, end-start)
	dbf.startWorker(workerChan, errChan, &wg)
	for i := 0; i < int(end-start); i++ {
		wg.Add(1)
		workerChan[i%workerNums] <- workerArgs{
			index: uint32(i + int(start)),
			value: rv.Index(i),
		}
	}
	wg.Wait()
	defer func() {
		// 关闭通道，关闭worker
		for i := 0; i < workerNums; i++ {
			close(workerChan[i])
		}
		close(errChan)
	}()

	for i := 0; i < int(end-start); i++ {
		err := <-errChan
		if err != nil {
			return err
		}
	}
	return nil
}
