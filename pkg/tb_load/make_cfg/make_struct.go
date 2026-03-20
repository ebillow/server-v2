package make_cfg

import (
	"errors"
	"go.uber.org/zap"
	"os"
	"server/pkg/util"
	"strings"
)

type fieldDesc struct {
	name string
	typ  string
}

type structDesc struct {
	field  []fieldDesc
	name   string
	parent string
}

type setupDesc struct {
	Keys map[string]bool
}

func makeOneStruct(sheetName string, typ []string, name []string) (*structDesc, error) {
	if len(typ) != len(name) {
		return nil, errors.New("type and name length do not match")
	}
	ret := &structDesc{name: sheetName}
	for i := range typ {
		ret.field = append(ret.field, fieldDesc{name: snakeToCamel(name[i]), typ: typ[i]})
	}
	return ret, nil
}

func writeProtoFile(desc map[string][]*structDesc) error {
	f, err := os.OpenFile("cfg.proto", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	_, err = f.WriteString("syntax = \"proto3\";\n")
	if err != nil {
		return err
	}
	_, err = f.WriteString("package " + "tb" + ";\n")
	if err != nil {
		return err
	}
	_, err = f.WriteString("option go_package = \"./;tb\";\n")
	if err != nil {
		return err
	}
	fDesc := structDesc{name: "Data"}

	for _, sheet := range desc {
		for _, v := range sheet {
			name, err := writeOneProto(f, v)
			if err != nil {
				return err
			}
			fDesc.field = append(fDesc.field, fieldDesc{
				name: name,
				typ:  name,
			})
		}
	}

	_, err = writeDataStruct(f, &fDesc)
	return err
}

func writeOneProto(f *os.File, o *structDesc) (string, error) {
	name := strings.Title(o.parent) + strings.Title(o.name)
	name = snakeToCamel(name)

	nameRow := name + "Row"
	_, err := f.WriteString("\nmessage " + nameRow + " {\n")
	if err != nil {
		return name, err
	}
	for i, field := range o.field {
		_, err = f.WriteString("\t" + field.typ + "\t" + strings.Title(field.name) + "\t\t\t = " + util.ToString(i+1) + ";\n")
		if err != nil {
			return name, err
		}
	}
	_, err = f.WriteString("}\n")
	if err != nil {
		return name, err
	}

	_, err = f.WriteString("\nmessage " + name + " {\n")
	if err != nil {
		return name, err
	}

	_, err = f.WriteString("\trepeated	" + nameRow + "\t List = 1;\n")
	if err != nil {
		return name, err
	}
	_, err = f.WriteString("\tmap<string, " + nameRow + ">\t Map = 2;\n")
	if err != nil {
		return name, err
	}

	_, err = f.WriteString("}\n")
	if err != nil {
		return name, err
	}

	zap.L().Info("write protobuf success", zap.String("name", name))
	return name, nil
}
func writeDataStruct(f *os.File, o *structDesc) (string, error) {
	name := strings.Title(o.parent) + strings.Title(o.name)
	name = snakeToCamel(name)

	_, err := f.WriteString("\nmessage " + name + " {\n")
	if err != nil {
		return name, err
	}
	for i, field := range o.field {
		_, err = f.WriteString("\t" + field.typ + "\t" + strings.Title(field.name) + "\t\t\t = " + util.ToString(i+1) + ";\n")
		if err != nil {
			return name, err
		}
	}
	_, err = f.WriteString("}\n")
	if err != nil {
		return name, err
	}

	return name, nil
}
