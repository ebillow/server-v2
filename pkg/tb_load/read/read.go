package read

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"reflect"
	tb "server/pkg/tb_load/target/table"
	"strconv"
	"strings"
	"sync/atomic"
)

var (
	Cfg atomic.Value
)

type setupDesc struct {
	Keys map[string]bool
}

func LoadTables(rootPath string) error {
	// data := &tb.Data{}
	typ := reflect.TypeOf(&tb.Data{})
	val := reflect.New(typ.Elem())
	initStruct(val)

	data := val.Interface().(*tb.Data)
	// fmt.Printf("%+v\n", data)
	setups := make(map[string]*setupDesc)
	err := walkFiles(rootPath, data, setups)
	if err != nil {
		zap.S().Error("walk files err:", err)
		return err
	}
	Cfg.Store(data)
	return nil
}

func walkFiles(path string, data *tb.Data, setups map[string]*setupDesc) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, f := range files {
		nextPath := filepath.Join(path, f.Name())
		if f.IsDir() {
			err = readSetup(nextPath, setups)
			if err != nil {
				return err
			}
			err = walkFiles(nextPath, data, setups)
			if err != nil {
				return err
			}
		} else {
			ext := filepath.Ext(f.Name())
			if ext != ".json" {
				continue
			}
			if f.Name() != "setup.json" {
				err = readOne(nextPath, data, setups)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func readSetup(path string, setups map[string]*setupDesc) error {
	b, err := os.ReadFile(path + "/setup.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	one := make(map[string]*setupDesc)
	err = json.Unmarshal(b, &one)
	if err != nil {
		return err
	}
	for k, v := range one {
		setups[path+"/"+k+".json"] = v
	}
	return nil
}

func initStruct(v reflect.Value) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		// fieldType := v.Type().Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.Ptr:
			field.Set(reflect.New(field.Type().Elem()))
			initStruct(field)
		case reflect.Struct:
			initStruct(field)
		case reflect.Slice:
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		case reflect.Map:
			field.Set(reflect.MakeMap(field.Type()))
		case reflect.Chan:
			field.Set(reflect.MakeChan(field.Type(), 0))
		case reflect.Interface:
			// Optional: set to a default concrete type if known
		default:
			// Set default value for basic types
			field.Set(reflect.Zero(field.Type()))
		}
	}
}

func readOne(path string, data *tb.Data, setups map[string]*setupDesc) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fieldName := fieldNameByPath(path)
	tv := reflect.ValueOf(data).Elem()
	field := tv.FieldByName(fieldName)
	list := field.Elem().FieldByName("List")
	if !list.CanAddr() {
		return fmt.Errorf("field:%s can not addr", fieldName)
	}
	zap.L().Debug("read", zap.String("file", path))
	err = json.Unmarshal(b, list.Addr().Interface())
	if err != nil {
		return err
	}
	mapFeild := field.Elem().FieldByName("Map")
	for i := 0; i < list.Len(); i++ {
		row := list.Index(i)

		var key string
		if desc, ok := setups[path]; ok {
			cnt := 0
			for k := range desc.Keys {
				field = row.Elem().FieldByName(strings.Title(k))
				if cnt > 0 {
					key += ","
				}
				key += fieldStrValue(field)
				cnt++
			}
		} else {
			rowVal := row.Elem()
			if rowVal.NumField() > 0 {
				firstField := rowVal.Field(3)
				key = fieldStrValue(firstField)
			}
		}
		mapFeild.SetMapIndex(reflect.ValueOf(key), row)
	}

	return nil
}

func fieldStrValue(field reflect.Value) string {
	var key string
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		key = strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		key = strconv.FormatUint(field.Uint(), 10)
	case reflect.String:
		key = field.String()
	default:
		zap.L().Error("row key err")
	}
	return key
}

func fieldNameByPath(path string) string {
	idx := strings.Index(path, "/")
	if idx == -1 {
		return path
	}
	path = path[idx+1:]
	idx = strings.Index(path, "/")
	if idx == -1 {
		return path
	}

	path = strings.TrimSuffix(path, ".json")
	ss := strings.Split(path[idx+1:], "/")
	name := ""
	for _, s := range ss {
		name += strings.Title(s)
	}
	return name
}
