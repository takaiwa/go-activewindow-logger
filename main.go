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

const LogFormat = "20060102"

//const IdleWaitingTime = 5 * 1000	// 秒
const IdleWaitingTime = 5 * 60 * 1000 // 5min

var (
	mod                     = windows.NewLazyDLL("user32.dll")
	procGetWindowText       = mod.NewProc("GetWindowTextW")
	procGetWindowTextLength = mod.NewProc("GetWindowTextLengthW")
	user32                  = syscall.MustLoadDLL("user32.dll")
	kernel32                = syscall.MustLoadDLL("kernel32.dll")
	getLastInputInfo        = user32.MustFindProc("GetLastInputInfo")
	getTickCount            = kernel32.MustFindProc("GetTickCount")
	lastInputInfo           struct {
		cbSize uint32
		dwTime uint32
	}
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

func getLogfile(fileName string) *os.File {
	logfile, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic("cannot open:" + err.Error())
	}
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	log.SetFlags(log.Ldate | log.Ltime)

	return logfile
}

func getLogFileName() string {
	return "./" + time.Now().Format(LogFormat) + "_gawl.log"
}

func openLogfile(fileName *string, logfile *os.File) {
	tmp := getLogFileName()
	if *fileName != tmp {
		// 日付が変わったら新しいログに書き込む
		logfile.Close()
		*fileName = tmp
		logfile = getLogfile(tmp)
	}
}

func getIdleTime() uint32 {
	lastInputInfo.cbSize = uint32(unsafe.Sizeof(lastInputInfo))
	currentTickCount, _, _ := getTickCount.Call()
	r1, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&lastInputInfo)))
	if r1 == 0 {
		panic("error getting last input info: " + err.Error())
	}
	return uint32(currentTickCount) - lastInputInfo.dwTime
}

func ticker() error {
	t := time.NewTicker(1 * time.Second)

	fileName := getLogFileName()
	logfile := getLogfile(fileName)

	defer t.Stop()
	defer logfile.Close()
	prevText := ""
	var counter uint32

	for {
		select {
		case <-t.C:
			idleTime := getIdleTime()
			if idleTime < IdleWaitingTime {

				if hwnd := getWindow("GetForegroundWindow"); hwnd != 0 {
					text := GetWindowText(HWND(hwnd))
					if prevText != text || counter > 1 {
						//fmt.Println("v:", v, "window :", text, "# hwnd:", hwnd)
						openLogfile(&fileName, logfile)
						log.Println(",", text)
						prevText = text
					}
				}
				counter = 1
			} else {
				if idleTime > (IdleWaitingTime * counter) {
					openLogfile(&fileName, logfile)
					log.Println(", Idle Time:", time.Duration(idleTime)*time.Millisecond)
					counter = counter + 1
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
