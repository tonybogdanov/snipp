package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// run shows a small native window with a marquee progress bar while the
// install happens in the background, then swaps in a close button. Without
// it, users only saw a bare cmd window flash (from the taskkill call below)
// with no explanation of what was happening.
func run() {
	runtime.LockOSThread()

	hInstance, _, _ := procGetModuleHandleW.Call(0)

	className, _ := syscall.UTF16PtrFromString("SnippInstallerWindow")
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(idcArrow))
	icon := loadAppIcon()

	wc := wndClassEx{
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     syscall.Handle(hInstance),
		hIcon:         icon,
		hIconSm:       icon,
		hCursor:       syscall.Handle(cursor),
		hbrBackground: syscall.Handle(colorWindow + 1),
		lpszClassName: className,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	screenW, _, _ := procGetSystemMetrics.Call(smCXScreen)
	screenH, _, _ := procGetSystemMetrics.Call(smCYScreen)
	// Tall enough for the multi-line messages install() can return (an
	// offline explanation is three short lines).
	const winW, winH = 420, 240
	x := (int32(screenW) - winW) / 2
	y := (int32(screenH) - winH) / 2

	title, _ := syscall.UTF16PtrFromString("Snipp Installer")
	style := uintptr(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox | wsVisible)

	hwndMain, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		style,
		uintptr(x), uintptr(y), winW, winH,
		0, 0, hInstance, 0,
	)

	icc := initCommonControlsEx{dwICC: iccProgressClass}
	icc.dwSize = uint32(unsafe.Sizeof(icc))
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))

	labelClass, _ := syscall.UTF16PtrFromString("STATIC")
	labelText, _ := syscall.UTF16PtrFromString("Installing Snipp...")
	hwndLabel, _, _ = procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(labelClass)), uintptr(unsafe.Pointer(labelText)),
		uintptr(wsChild|wsVisible),
		20, 20, 360, 90,
		hwndMain, 0, hInstance, 0,
	)

	progressClass, _ := syscall.UTF16PtrFromString("msctls_progress32")
	hwndProgress, _, _ = procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(progressClass)), 0,
		uintptr(wsChild|wsVisible|pbsMarquee),
		20, 120, 360, 20,
		hwndMain, 0, hInstance, 0,
	)
	procSendMessageW.Call(hwndProgress, pbmSetMarquee, 1, 30)

	buttonClass, _ := syscall.UTF16PtrFromString("BUTTON")
	buttonText, _ := syscall.UTF16PtrFromString("Close")
	hwndButton, _, _ = procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(buttonClass)), uintptr(unsafe.Pointer(buttonText)),
		uintptr(wsChild|bsPushButton),
		160, 160, 100, 28,
		hwndMain, uintptr(idClose), hInstance, 0,
	)

	procShowWindow.Call(hwndMain, swShow)
	procUpdateWindow.Call(hwndMain)

	go func() {
		installMessage = install()
		procPostMessageW.Call(hwndMain, wmInstallDone, 0, 0)
	}()

	var m msgT
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch uint32(msg) {
	case wmCommand:
		if uint16(wparam) == idClose && uint16(wparam>>16) == bnClicked {
			procDestroyWindow.Call(hwnd)
		}
		return 0
	case wmInstallDone:
		installDone = true
		procSendMessageW.Call(hwndProgress, pbmSetMarquee, 0, 0)
		procShowWindow.Call(hwndProgress, swHide)
		setWindowText(hwndLabel, installMessage)
		procShowWindow.Call(hwndButton, swShow)
		return 0
	case wmClose:
		if !installDone {
			return 0
		}
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

func setWindowText(hwnd uintptr, text string) {
	p, _ := syscall.UTF16PtrFromString(text)
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(p)))
}

// loadAppIcon loads the embedded icon via LoadImageW, which only reads from
// a file path, so the embedded bytes are spilled to a temp file first (safe
// to remove immediately after: LoadImageW copies the bitmap into its own
// GDI object before returning).
func loadAppIcon() syscall.Handle {
	tmp, err := os.CreateTemp("", "snipp-icon-*.ico")
	if err != nil {
		return 0
	}
	defer os.Remove(tmp.Name())

	_, werr := tmp.Write(appIcon)
	tmp.Close()
	if werr != nil {
		return 0
	}

	path, _ := syscall.UTF16PtrFromString(tmp.Name())
	h, _, _ := procLoadImageW.Call(0, uintptr(unsafe.Pointer(path)), imageIcon, 0, 0, lrLoadFromFile|lrDefaultSize)
	return syscall.Handle(h)
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")

	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procSendMessageW         = user32.NewProc("SendMessageW")
	procPostMessageW         = user32.NewProc("PostMessageW")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")
	procLoadImageW           = user32.NewProc("LoadImageW")
)

const (
	wsOverlapped  = 0x00000000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsMinimizeBox = 0x00020000
	wsVisible     = 0x10000000
	wsChild       = 0x40000000

	swShow = 5
	swHide = 0

	wmDestroy = 0x0002
	wmClose   = 0x0010
	wmCommand = 0x0111
	wmApp     = 0x8000

	wmInstallDone = wmApp + 1

	bsPushButton = 0x00000000
	bnClicked    = 0

	pbsMarquee    = 0x08
	pbmSetMarquee = 0x040A

	iccProgressClass = 0x00000020

	smCXScreen = 0
	smCYScreen = 1

	idcArrow    = 32512
	colorWindow = 5

	idClose = 1001

	imageIcon      = 1
	lrLoadFromFile = 0x00000010
	lrDefaultSize  = 0x00000040
)

var (
	hwndMain, hwndLabel, hwndProgress, hwndButton uintptr
	installDone                                   bool

	// installMessage is what install() concluded — success, "already up to
	// date", or why it couldn't. Written by the install goroutine and read
	// by the window procedure only after wmInstallDone hands over.
	installMessage string
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

type point struct{ x, y int32 }

type msgT struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type initCommonControlsEx struct {
	dwSize uint32
	dwICC  uint32
}

// binaryName is what the app is called once installed, and binaryAsset the
// release asset it's downloaded from — the same name here, but they're
// distinct roles.
const (
	binaryName  = "snipp.exe"
	binaryAsset = "snipp.exe"
)

// installDir is the per-user install location; no elevation needed.
func installDir() (string, error) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	return filepath.Join(local, "Snipp"), nil
}

// placeBinary writes the downloaded exe over the installed one. Renaming
// the old exe out of the way first works because Windows opens running
// executables with share-delete, not share-write: the rename succeeds even
// while the old process is still executing from it.
func placeBinary(target string, binary []byte) error {
	old := target + ".old"

	os.Remove(old)
	os.Rename(target, old)

	if err := os.WriteFile(target, binary, 0o755); err != nil {
		os.Rename(old, target)
		return err
	}

	// Best-effort: Windows won't delete the old image while the previous
	// instance is still executing from it. Whatever is left behind is
	// removed by the Remove above on the next run.
	os.Remove(old)
	return nil
}

// killRunning stops any running snipp.exe via taskkill, with the console
// window it would otherwise briefly flash suppressed.
func killRunning() {
	cmd := exec.Command("taskkill", "/F", "/IM", "snipp.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	cmd.Run()
}

func registerAutostart(target string) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()
	key.SetStringValue("Snipp", `"`+target+`"`)
}
