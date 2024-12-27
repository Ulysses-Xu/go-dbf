package godbf

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"time"
)

func (dbf *DBFHandler) Append(model interface{}) (err error) {
	dbf.mu.Lock()
	defer dbf.mu.Unlock()

	oldMd5String, err := dbf.getFileMd5()
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(model).Elem()

	// 先生成buffer,用空格填充
	buf := bytes.Repeat([]byte{SPACE}, int(dbf.header.RecordLength)+1)
	// 第一个字节是删除标识，空格为未删除
	pos := 1
	for i := 0; i < len(dbf.fields); i++ {
		columnName := dbf.columns[i]
		fieldIndex, ok := dbf.modelColumnIndex[columnName]
		if !ok {
			return fmt.Errorf("column %s not found", columnName)
		}
		fieldVal := rv.Field(fieldIndex)
		var valStr string
		switch fieldVal.Kind() {
		case reflect.String:
			valStr = dbf.encoder.ConvertString(fieldVal.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			valStr = strconv.FormatInt(fieldVal.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			valStr = strconv.FormatUint(fieldVal.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			decimalPlace := int(dbf.fields[i].Decimal)
			if decimalPlace == 0 {
				decimalPlace = 2
			}
			valStr = strconv.FormatFloat(fieldVal.Float(), 'f', decimalPlace, 64)
		}
		// 将val按column起始位置开始写入，超长则截断
		nextPos := pos + int(dbf.fields[i].Length)
		copy(buf[pos:nextPos], valStr)
		pos = nextPos
	}
	// dbf文件最后一个字符以0x1A作为结束标志
	buf[pos] = EOF
	// 重新计算文件md5,判断前后是否一致，如果不一致，则表明文件已被其他进程修改，则放弃写入
	newMd5String, err := dbf.getFileMd5()
	if err != nil {
		return err
	}
	if oldMd5String != newMd5String {
		_ = dbf.ReloadFromFile()
		return errors.New("file has changed")
	}

	err = dbf.saveRecords(&buf)
	if err != nil {
		return err
	}

	err = dbf.saveNumRecords(1)
	if err != nil {
		return err
	}

	year, month, day, err := dbf.saveUpdateTime()
	if err != nil {
		return err
	}
	// 调用sync更新到磁盘
	err = dbf.f.Sync()
	if err != nil {
		defer dbf.rollbackRecord()
		defer dbf.rollbackNumRecords()
		defer dbf.rollbackUpdateTime()
	}

	// 最后更新
	dbf.header.LastUpdateYear = year
	dbf.header.LastUpdateMonth = month
	dbf.header.LastUpdateDay = day
	dbf.header.NumRecords += 1
	return err
}

func (dbf *DBFHandler) saveRecords(buf *[]byte) error {
	// 从后面往前面移动1位，覆盖最后的结束符
	if _, err := dbf.f.Seek(-1, io.SeekEnd); err != nil {
		_ = dbf.ReloadFromFile()
		return err
	}
	// 写入行数据
	if _, err := dbf.f.Write(*buf); err != nil {
		_, err = dbf.f.Write(*buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dbf *DBFHandler) rollbackRecord() error {
	originalSize := int64(uint32(dbf.header.HeaderLength) + uint32(dbf.header.RecordLength)*dbf.header.NumRecords)
	err := dbf.f.Truncate(originalSize)
	if err != nil {
		return errors.New("Failed to truncate record while rollback record. ")
	}

	_, err = dbf.f.Seek(0, io.SeekEnd)
	if err != nil {
		return errors.New("Failed to seek record while rollback record. ")
	}

	_, err = dbf.f.Write([]byte{EOF})
	if err != nil {
		return errors.New("Failed to write record while rollback record. ")
	}

	return nil
}

func (dbf *DBFHandler) saveNumRecords(appendNum uint32) (err error) {
	// 更新记录数
	newNumRecords := dbf.header.NumRecords + appendNum
	if _, err = dbf.f.Seek(4, io.SeekStart); err != nil {
		// 如果更新表头失败，则回滚数据
		defer dbf.rollbackRecord()
		return errors.New("Failed to seek record while saving num records. ")
	}
	err = binary.Write(dbf.f, binary.LittleEndian, newNumRecords)
	if err != nil {
		defer dbf.rollbackRecord()
		return errors.New("Failed to write record while saving num records. ")
	}
	return nil
}

func (dbf *DBFHandler) rollbackNumRecords() (err error) {
	if _, err = dbf.f.Seek(4, io.SeekStart); err != nil {
		return errors.New("Failed to seek record while rollback num records. ")
	}
	err = binary.Write(dbf.f, binary.LittleEndian, dbf.header.NumRecords)
	if err != nil {
		return errors.New("Failed to write record while rollback num records. ")
	}
	return nil
}

func (dbf *DBFHandler) saveUpdateTime() (year, month, day byte, err error) {
	currentTime := time.Now()
	yearInt, monthInt, dayInt := currentTime.Date()
	year = byte(yearInt - 1900)
	month = byte(monthInt)
	day = byte(dayInt)
	//dbf.header.LastUpdateYear = byte(year)
	//dbf.header.LastUpdateMonth = byte(month)
	//dbf.header.LastUpdateDay = byte(day)

	if _, err = dbf.f.Seek(1, io.SeekStart); err != nil {
		// 如果更新更新年月日失败，则回滚数据
		defer dbf.rollbackRecord()
		defer dbf.rollbackNumRecords()
		return year, month, day, errors.New("Failed to seek while saving updateTime. ")
	}

	_, err = dbf.f.Write([]byte{dbf.header.LastUpdateYear, dbf.header.LastUpdateMonth, dbf.header.LastUpdateDay})
	if err != nil {
		// 如果更新更新年月日失败，则回滚数据
		defer dbf.rollbackRecord()
		defer dbf.rollbackNumRecords()
		return year, month, day, errors.New("Failed to write while saving updateTime. ")
	}
	return year, month, day, err
}

func (dbf *DBFHandler) rollbackUpdateTime() (err error) {
	if _, err = dbf.f.Seek(1, io.SeekStart); err != nil {
		// 如果更新更新年月日失败，则回滚数据
		defer dbf.rollbackRecord()
		defer dbf.rollbackNumRecords()
		return errors.New("Failed to seek while rollback updateTime. ")
	}

	_, err = dbf.f.Write([]byte{dbf.header.LastUpdateYear, dbf.header.LastUpdateMonth, dbf.header.LastUpdateDay})
	if err != nil {
		// 如果更新更新年月日失败，则回滚数据
		defer dbf.rollbackRecord()
		defer dbf.rollbackNumRecords()
		return errors.New("Failed to write while rollback updateTime. ")
	}
	return err
}

func (dbf *DBFHandler) getFileMd5() (string, error) {
	file, err := os.Open(dbf.fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	md5String := hex.EncodeToString(hash.Sum(nil))
	return md5String, nil
}
