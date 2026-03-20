package make_cfg

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os"
	"strings"
)

func writeJsonData(path string, sheetName string, desc *structDesc, rows [][]string) error {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(fmt.Sprintf("%s/%s.json", path, sheetName), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	_, _ = f.WriteString("[\n")
	for i := 2; i < len(rows); i++ {
		if len(rows[i]) != len(desc.field) {
			return fmt.Errorf("row[%d] has %d fields, want %d", i, len(rows[i]), len(desc.field))
		}
		err = writeRow(f, desc, rows[i])
		if err != nil {
			return err
		}
		if i != len(rows)-1 {
			_, _ = f.WriteString(",\n")
		}
	}
	_, _ = f.WriteString("\n]")
	zap.L().Info("write json success", zap.String("name", path), zap.String("sheet", sheetName))
	return f.Close()
}

func writeRow(f *os.File, desc *structDesc, row []string) (err error) {
	//todo 不支持数组或者map中的string
	_, _ = f.WriteString("{")
	for i := range row {
		data := row[i]
		if len(data) == 0 {
			return fmt.Errorf("%s row[%d] has no data", desc.name, i)
		}
		switch desc.field[i].typ {
		case "string":
			_, err = f.WriteString(fmt.Sprintf(`"%s":"%s"`, desc.field[i].name, data))
		default:
			_, err = f.WriteString(fmt.Sprintf(`"%s":%s`, desc.field[i].name, data))
		}
		if err != nil {
			return err
		}
		if i != len(desc.field)-1 {
			_, _ = f.WriteString(",")
		}
	}
	_, _ = f.WriteString("}")
	return nil
}

func buildSetups(path string, rows [][]string) error {
	ret := make(map[string]*setupDesc)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) != 2 {
			return fmt.Errorf("setup row length error:%v", row)
		}
		name := row[0]
		ret[name] = &setupDesc{Keys: make(map[string]bool)}
		ss := strings.Split(row[1], ",")
		for _, v := range ss {
			ret[name].Keys[v] = true
		}
	}
	b, err := json.Marshal(ret)
	if err != nil {
		return err
	}

	return os.WriteFile(fmt.Sprintf("%s/%s.json", path, setupSheet), b, 0666)
}
