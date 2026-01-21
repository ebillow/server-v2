package util

import (
	"bufio"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"server/internal/log"
	"strings"
	"syscall"
)

// GoSafe runs the given fn using another goroutine, recovers if fn panics.
func GoSafe(fn func()) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		RunSafe(fn)
		close(done)
	}()
	return done
}

// RunSafe runs the given fn, recovers if fn panics.
func RunSafe(fn func()) (caught bool) {
	if fn == nil {
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			zap.L().WithOptions(zap.AddCallerSkip(2)).DPanic("[PANIC]", zap.Any("exception", r), zap.StackSkip("stack", 2))
			PrintStack()
			caught = true
		}
	}()

	fn()
	return caught
}

func WaitExit() {
	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range exitChan {
		switch s {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			log.Infof("Signal: %v server closing ...", s)
			return
		}
	}
}

// PrintStack 打印堆栈，并打印传入的变量
func PrintStack(vars ...interface{}) {
	stack := make([]string, 0)
	for _, v := range vars {
		stack = append(stack, fmt.Sprintf("%v\n", v))
	}

	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	stack = append(stack, string(buf[0:n]))

	log.Error(stack)
}

// FuncCaller 得到调用者
func FuncCaller(lvl int) string {
	funcName, file, line, ok := runtime.Caller(lvl)

	info := ""
	for ok {
		info += fmt.Sprintf("frame %v:[func:%v,file:%v,line:%v]\n", lvl, runtime.FuncForPC(funcName).Name(), file, line)
		lvl++
		funcName, file, line, ok = runtime.Caller(lvl)
	}

	return info
}

func FuncCallerOnce(lvl int) string {
	_, file, line, ok := runtime.Caller(lvl)
	if ok {
		return fmt.Sprintf("file:%v,line:%v", file, line)
	}
	return ""
}

func ExecCommand(commandName string, params []string) ([]string, error) {
	var contentArray = make([]string, 0, 5)
	cmd := exec.Command(commandName, params...)
	// 显示运行的命令
	fmt.Printf("exec: %s\n", strings.Join(cmd.Args[0:], " "))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cmd error=>", err.Error())
		return contentArray, err
	}
	err = cmd.Start() // Start开始执行c包含的命令，但并不会等待该命令完成即返回。Wait方法会返回命令的返回状态码并在命令返回后释放相关的资源。
	if err != nil {
		return contentArray, err
	}
	reader := bufio.NewReader(stdout)

	var index int
	// 实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		fmt.Print(line)
		index++
		contentArray = append(contentArray, line)
	}

	return contentArray, cmd.Wait()
}
