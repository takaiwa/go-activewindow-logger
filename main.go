package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"io"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

var (
	mod                     = windows.NewLazyDLL("user32.dll")
	procGetWindowText       = mod.NewProc("GetWindowTextW")
	procGetWindowTextLength = mod.NewProc("GetWindowTextLengthW")
)

type (
	HANDLE uintptr
	HWND   HANDLE
)

func GetWindowTextLength(hwnd HWND) int {
	ret, _, _ := procGetWindowTextLength.Call(
		uintptr(hwnd))

	return int(ret)
}

func GetWindowText(hwnd HWND) string {
	textLen := GetWindowTextLength(hwnd) + 1

	buf := make([]uint16, textLen)
	procGetWindowText.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(textLen))

	return syscall.UTF16ToString(buf)
}

func getWindow(funcName string) uintptr {
	proc := mod.NewProc(funcName)
	hwnd, _, _ := proc.Call()
	return hwnd
}

func ticker() error {
	t := time.NewTicker(1 * time.Second)

	date := time.Now()
	fileName := "./" + date.Format("20060102") + "_gawl.log"

	// ログファイルの出力設定
	logfile, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic("cannot open:" + err.Error())
	}
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	log.SetFlags(log.Ldate | log.Ltime)

	defer t.Stop()
	defer logfile.Close()
	prevText := ""

	for {
		select {
		case <-t.C:
			if hwnd := getWindow("GetForegroundWindow"); hwnd != 0 {
				text := GetWindowText(HWND(hwnd))
				if prevText != text {
					//fmt.Println("v:", v, "window :", text, "# hwnd:", hwnd)
					log.Println(",", text)
					prevText = text
				}
			}
		}
	}
}

func main() {

	if err := ticker(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}
