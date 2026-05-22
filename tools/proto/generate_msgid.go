// //go:build ignore
// // +build ignore

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	em "github.com/emicklei/proto"
)

//go:generate go run $GOFILE

// 配置项
var (
	protoDir = "../../api/proto" // proto 文件夹路径
	// protoDir       = "./pkg/proto"                 // proto 文件夹路径
	targetPrefixes = []string{"C2S", "S2C", "S2S"} // 需要提取的 Message 前缀
)

func main() {
	// 1. 全局收集所有的 Message
	globalMsgMap := collectAllMessages(protoDir)

	if len(globalMsgMap) == 0 {
		fmt.Println("未找到任何符合前缀的 Message，无需更新。")
		return
	}

	// 打印收集结果
	fmt.Println("====== 收集到的 Message 统计 ======")
	for prefix, msgs := range globalMsgMap {
		fmt.Printf("[%s] 找到 %d 个 message\n", prefix, len(msgs))
	}

	// 2. 统一更新到指定的 Enum 文件中
	for _, v := range targetPrefixes {
		updateTargetEnumFile(v, globalMsgMap[v])
	}
}

// collectAllMessages 遍历目录，收集所有符合条件的 Message
func collectAllMessages(dir string) map[string][]string {
	msgMap := make(map[string][]string)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("目录 %s 不存在\n", dir)
		return msgMap
	}

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".proto" {
			return nil
		}

		// 解析文件
		reader, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer reader.Close()

		parser := em.NewParser(reader)
		definition, err := parser.Parse()
		if err != nil {
			return nil
		}

		// 提取 Message
		for _, element := range definition.Elements {
			if msg, ok := element.(*em.Message); ok {
				for _, prefix := range targetPrefixes {
					if strings.HasPrefix(msg.Name, prefix) {
						msgMap[prefix] = append(msgMap[prefix], msg.Name)
						break
					}
				}
			}
		}
		return nil
	})

	return msgMap
}

// updateTargetEnumFile 更新或生成目标 Enum 文件
func updateTargetEnumFile(typ string, msgMap []string) {
	var definition *em.Proto
	modified := false
	filePath := fmt.Sprintf("%s/msg_id_%s.proto", protoDir, strings.ToLower(typ))
	// 尝试读取现有的目标文件
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 文件不存在，创建一个新的 AST 根节点
		definition = &em.Proto{}
		definition.Elements = append(definition.Elements, &em.Syntax{Value: "proto3"})
		modified = true
		fmt.Printf("目标文件 %s 不存在，将自动创建\n", filePath)
	} else {
		// 文件存在，解析现有的 AST
		reader, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("打开目标文件失败: %v\n", err)
			return
		}
		defer reader.Close()

		parser := em.NewParser(reader)
		definition, err = parser.Parse()
		if err != nil {
			fmt.Printf("解析目标文件失败: %v\n", err)
			return
		}
	}

	// 查找目标文件中已经存在的 Enum
	enums := make(map[string]*em.Enum)
	for _, element := range definition.Elements {
		if e, ok := element.(*em.Enum); ok {
			enums[e.Name] = e
		}
	}

	// 遍历收集到的全局 Message，更新 Enum
	enumName := "MsgID" + typ
	targetEnum, exists := enums[enumName]

	if !exists {
		// Enum 不存在，创建它并加入到 AST
		targetEnum = &em.Enum{Name: enumName}
		definition.Elements = append(definition.Elements, targetEnum)
		modified = true
	}

	// 收集该 Enum 中现有的字段，防止覆盖，并找出最大 ID
	existingFields := make(map[string]bool)
	maxID := -1

	for _, el := range targetEnum.Elements {
		if field, ok := el.(*em.EnumField); ok {
			existingFields[field.Name] = true
			if field.Integer > maxID {
				maxID = field.Integer
			}
		}
	}

	// 将全局收集到的、且 Enum 中不存在的 Message 追加进去
	for _, msg := range msgMap {
		if !existingFields[msg] {
			maxID++
			newField := &em.EnumField{
				Name:    msg,
				Integer: maxID,
			}
			targetEnum.Elements = append(targetEnum.Elements, newField)
			modified = true
		}
	}

	// 如果发生了实质性的修改（有新 enum 或新 field），则写回文件
	if modified {
		var buf bytes.Buffer

		// formatter := em.NewFormatter(&buf, "    ") // 4个空格缩进
		// formatter.Format(definition)
		writeProto(&buf, definition)

		// 确保目标文件的目录存在
		os.MkdirAll(filepath.Dir(filePath), os.ModePerm)

		err := os.WriteFile(filePath, buf.Bytes(), 0644)
		if err != nil {
			fmt.Printf("[-] 写入目标文件 %s 失败: %v\n", filePath, err)
		} else {
			fmt.Printf("[+] 成功更新 Enum 文件: %s\n", filePath)
		}
	} else {
		fmt.Println("[=] Enum 文件已是最新，无需更新。")
	}
}

func writeProto(buf *bytes.Buffer, p *em.Proto) {
	for i, e := range p.Elements {
		switch v := e.(type) {
		case *em.Syntax:
			fmt.Fprintf(buf, "syntax = %q;\n\n", v.Value)
		case *em.Package:
			fmt.Fprintf(buf, "package %s;\n", v.Name)
		case *em.Import:
			fmt.Fprintf(buf, "import %q;\n", v.Filename)
		case *em.Enum:
			writeEnum(buf, v)
			if i != len(p.Elements)-1 {
				buf.WriteString("\n")
			}
		case *em.Option:
			if v.Name == "go_package" {
				fmt.Fprintf(buf, "option %s = \"%s\";\n", v.Name, v.Constant.Source)
			} else {
				fmt.Fprintf(buf, "option %s = %s;\n", v.Name, v.Constant.Source)
			}
		// case *em.Message:
		// 	writeMessage(buf, v)
		// 	if i != len(p.Elements)-1 {
		// 		buf.WriteString("\n")
		// 	}
		default:
			// 后面有需要再补
		}
	}
}

func writeEnum(buf *bytes.Buffer, e *em.Enum) {
	fmt.Fprintf(buf, "\nenum %s {\n", e.Name)
	for _, el := range e.Elements {
		switch f := el.(type) {
		case *em.EnumField:
			if f.InlineComment != nil && len(f.InlineComment.Lines) > 0 {
				fmt.Fprintf(buf, "    %s = %d;//\t\t%s\n", f.Name, f.Integer, f.InlineComment.Lines[0])
			} else {
				fmt.Fprintf(buf, "    %s = %d;\n", f.Name, f.Integer)
			}
		case *em.Comment:
			for _, line := range f.Lines {
				fmt.Fprintf(buf, "    // %s\n", line)
			}
		default:
			// 其他 enum 子元素暂时忽略
		}
	}
	buf.WriteString("}\n")
}
