package main

import (
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"
)

/*** data ***/

type Termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]byte
	Ispeed uint32
	Ospeed uint32
}

type editorConfig struct {
	origTermios *Termios
}

type WinSize struct {
    Row    uint16
    Col    uint16
    Xpixel uint16
    Ypixel uint16
}


var E editorConfig

/*** terminal ***/

func die(err error) {
	disableRawMode()
	io.WriteString(os.Stdout, "\x1b[2J");
	io.WriteString(os.Stdout, "\x1b[H");
	log.Fatal(err)
}

func TcSetAttr(fd uintptr, termios *Termios) error {
	// TCSETS+1 == TCSETSW, because TCSAFLUSH doesn't exist
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TCSETS+1), uintptr(unsafe.Pointer(termios))); err != 0 {
		return err
	}
	return nil
}

func TcGetAttr(fd uintptr) *Termios {
	var termios = &Termios{}
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(termios))); err != 0 {
		log.Fatalf("Problem getting terminal attributes: %s\n", err)
	}
	return termios
}

func enableRawMode() {
	E.origTermios = TcGetAttr(os.Stdin.Fd())
	var raw Termios
	raw = *E.origTermios
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN+1] = 0
	raw.Cc[syscall.VTIME+1] = 1
	if e := TcSetAttr(os.Stdin.Fd(), &raw); e != nil {
		log.Fatalf("Problem enabling raw mode: %s\n", e)
	}
}

func disableRawMode() {
	if e := TcSetAttr(os.Stdin.Fd(), E.origTermios); e != nil {
		log.Fatalf("Problem disabling raw mode: %s\n", e)
	}
}

func editorReadKey() byte {
	var buffer [1]byte
	var cc int
	var err error
	for cc, err = os.Stdin.Read(buffer[:]);
		cc != 1;
		cc, err = os.Stdin.Read(buffer[:]) {
	}
	if err != nil {
		die(err)
	}
	return buffer[0]
}

func getWindowSize(rows *int, cols *int) int {
	var w WinSize
	for {
        _, _, err := syscall.Syscall(syscall.SYS_IOCTL,
            os.Stdout.Fd(),
            syscall.TIOCGWINSZ,
            uintptr(unsafe.Pointer(&w)),
        )
        if err == 0 {  // type syscall.Errno
			*rows = int(w.Row)
			*cols = int(w.Col)
            return 0
        }
	}
	return -1
}

/*** input ***/

func editorProcessKeypress() {
	c := editorReadKey()
	switch c {
	case ('q' & 0x1f):
		io.WriteString(os.Stdout, "\x1b[2J");
		io.WriteString(os.Stdout, "\x1b[H");
		disableRawMode()
		os.Exit(0)
	}
}

/*** output ***/

func editorRefreshScreen() {
	io.WriteString(os.Stdout, "\x1b[2J");
	io.WriteString(os.Stdout, "\x1b[H");
	editorDrawRows()
	io.WriteString(os.Stdout, "\x1b[H");
}

func editorDrawRows() {
	for y := 0; y < 24; y++ {
		io.WriteString(os.Stdout, "~\r\n");
	}
}

/*** init ***/

func main() {
	enableRawMode()
	defer disableRawMode()

	for {
		editorRefreshScreen()
		editorProcessKeypress()
	}
}
