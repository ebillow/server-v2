package make_cfg

import (
	"errors"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"
)

var (
	errFileEmpty = errors.New("file is empty")
)

const (
	rowOfName  = 0
	rowOfType  = 1
	setupSheet = "setup"
)

func MakeAll(rootPath string) {
	desc := make(map[string][]*structDesc)
	desc, err := walkFiles(rootPath, desc)
	if err != nil {
		zap.S().Error("walk files err:", err)
		return
	}
	err = writeProtoFile(desc)
	if err != nil {
		zap.S().Error("writeProtoFile err:", err)
	}
}

func walkFiles(path string, fileInfo map[string][]*structDesc) (map[string][]*structDesc, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return fileInfo, err
	}
	for _, f := range files {
		nextPath := filepath.Join(path, f.Name())
		if f.IsDir() {
			info, err := walkFiles(nextPath, fileInfo)
			if err != nil {
				return fileInfo, err
			}
			for k, v := range info {
				fileInfo[k] = v
			}
		} else {
			ext := filepath.Ext(f.Name())
			if ext != ".xlsx" {
				continue
			}
			descs, err := readOneExcel(nextPath)
			if err != nil {
				return fileInfo, err
			}
			fileInfo[path] = append(fileInfo[path], descs...)
		}
	}
	return fileInfo, nil
}

func readOneExcel(path string) ([]*structDesc, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	if f.SheetCount == 0 {
		return nil, errFileEmpty
	}

	fileName := getFileName(path)
	targetPath := getTargetPath(fileName)

	sheets := f.GetSheetList()
	// 读setup
	if f.GetSheetName(len(sheets)-1) == setupSheet {
		setupRows, err := f.GetRows(sheets[len(sheets)-1])
		if err != nil {
			return nil, err
		}
		err = buildSetups(targetPath, setupRows)
		if err != nil {
			return nil, err
		}
	}

	// 读sheet
	descs := make([]*structDesc, 0, len(sheets))
	for i, v := range sheets {
		if f.GetSheetName(i) == setupSheet {
			continue
		}
		rows, err := f.GetRows(v)
		if err != nil {
			return nil, err
		}
		if len(rows) < 2 {
			return nil, errFileEmpty
		}
		desc, err := makeOneStruct(v, rows[rowOfType], rows[rowOfName])
		if err != nil {
			return nil, err
		}
		desc.parent = fileName
		err = writeJsonData(targetPath, v, desc, rows)
		if err != nil {
			return nil, err
		}
		descs = append(descs, desc)
	}
	return descs, nil
}

func getFileName(path string) string { // 获取名字   data/login.xlsx ->login
	idx := strings.Index(path, "/")
	if idx == -1 {
		return path
	}
	path = strings.TrimRight(path, ".xlsx")
	return path[idx+1:]
}

func getTargetPath(path string) string {
	return targetRoot + path
}
