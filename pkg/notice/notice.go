package notice

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type MardDown struct {
	Content string `json:"content"`
}
type NoticeInfo struct {
	MsgType string   `json:"msgtype"`
	Data    MardDown `json:"markdown"`
}

type NoticeData struct {
	Count int64
	Url   string
	Msg   string
}
type NoticeMgr struct {
	Data map[string]NoticeData
}

var (
	g_Notice *NoticeMgr
	g_Lock   sync.Mutex
)

func init() {
	go func() {
		for {
			g_Lock.Lock()
			if g_Notice != nil {
				for _, info := range g_Notice.Data {
					info.Msg = fmt.Sprintf("%s\n> 数量:<font color=\"comment\">%d</font>", info.Msg, info.Count)
					Post(info.Msg, info.Url)
				}

				g_Notice.Data = make(map[string]NoticeData)
			}
			g_Lock.Unlock()
			time.Sleep(5 * time.Minute)
		}
	}()
}

// msg markdown 类型字符串
// `<font color=\"warning\">实时报警，请相关同事注意。</font>
// > 服务器:<font color=\"comment\">%s</font>
// > 时间:<font color=\"comment\">%s</font>
// > 行号:<font color=\"comment\">%s</font>
// > 消息:<font color=\"comment\">%s</font>`
// 直接推送
func Post(msg string, url string) {
	if len(msg) == 0 || len(url) == 0 {
		zap.L().Error(fmt.Sprintf("send message is base"))
		return
	}

	notice := &NoticeInfo{
		MsgType: "markdown",
		Data: MardDown{
			Content: msg,
		},
	}

	data, err := json.Marshal(notice)
	if err != nil {
		zap.L().Error(fmt.Sprintf("json marshal%v", err.Error()))
		return
	}

	// 忽略ssl证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		zap.L().Error("wechat post noitce error", zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error("wechat post noitce error", zap.Error(err))
		return
	}
	defer resp.Body.Close()
}

// 定时推送 追加消息统计
// > 数量:<font color=\"comment\">%s</font>
func Add(msg string, url string, tag string) {
	if len(msg) == 0 || len(url) == 0 {
		return
	}

	g_Lock.Lock()
	defer g_Lock.Unlock()
	if g_Notice == nil {
		g_Notice = &NoticeMgr{
			Data: make(map[string]NoticeData),
		}
	}

	if d, ok := g_Notice.Data[tag]; !ok {
		g_Notice.Data[tag] = NoticeData{
			Count: 1,
			Url:   url,
			Msg:   msg,
		}
	} else {
		d.Msg = msg
		d.Url = url
		d.Count += 1
		g_Notice.Data[tag] = d
	}
}
