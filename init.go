package dbf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func (dbf *DBFHandler) initMetaData(model interface{}) error {
	err := dbf.initHeader()
	if err != nil {
		return err
	}
	err = dbf.initFields()
	if err != nil {
		return err
	}
	err = dbf.initModel(model)
	return err
}

func (dbf *DBFHandler) initHeader() error {
	err := binary.Read(dbf.f, binary.LittleEndian, &dbf.header)
	if err != nil {
		return err
	}
	return nil
}

func (dbf *DBFHandler) initFields() error {
	fieldNum := int((dbf.header.HeaderLength - 1 - 32) / 32)
	var descriptors []FieldDescriptor = make([]FieldDescriptor, fieldNum)
	var columns []string = make([]string, fieldNum)
	for i := 0; i < fieldNum; i++ {
		var descriptor FieldDescriptor
		err := binary.Read(dbf.f, binary.LittleEndian, &descriptor)
		if err != nil {
			return err
		}
		descriptors[i] = descriptor
		index := bytes.IndexByte(descriptor.Name[:], NUL)
		if index == -1 {
			index = len(descriptor.Name)
		}
		column := strings.TrimSpace(dbf.decoder.ConvertString(string(descriptor.Name[:index])))
		columns[i] = column
	}
	dbf.fields = descriptors
	dbf.columns = columns
	return nil
}

func (dbf *DBFHandler) initModel(model interface{}) error {
	rt := reflect.TypeOf(model)
	if rt.Kind() != reflect.Ptr {
		return errors.New("model must be a pointer to a struct")
	}

	rt = rt.Elem()
	if rt.Kind() != reflect.Struct {
		return fmt.Errorf("model must be a pointer to a struct, non a %v", rt.Kind())
	}
	var modelColumnIndex map[string]int = make(map[string]int)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		dbfColumn := field.Tag.Get("dbf")
		modelColumnIndex[dbfColumn] = i
	}
	dbf.modelColumnIndex = modelColumnIndex
	return nil
}
