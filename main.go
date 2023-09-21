package goland

import (
	"bufio"
	"fmt"
	"github.com/five-hundred-eleven/goland-game/static"
	"github.com/mattn/go-gtk/gdkpixbuf"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	End = iota
)

const LOOPSLEEP = time.Duration(100) * time.Millisecond
const NUMWORKERS = 4

func main() {

	msgCh := make(chan int)

	f, err := os.Open("maze.txt")
	if err != nil {
		println("error opening maze file")
		return
	}
	var numBytesRead int64 = 0
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	startYStr := scanner.Text()
	numBytesRead += int64(len(startYStr))
	scanner.Scan()
	startXStr := scanner.Text()
	numBytesRead += int64(len(startXStr))
	scanner.Scan()
	startDirectionStr := scanner.Text()
	numBytesRead += int64(len(startDirectionStr))
	scanner.Scan()
	numRowsStr := scanner.Text()
	numBytesRead += int64(len(numRowsStr))
	scanner.Scan()
	numColsStr := scanner.Text()
	numBytesRead += int64(len(numColsStr))

	startY, err := strconv.ParseFloat(startYStr, 64)
	if err != nil {
		println(fmt.Sprintf("error parsing starting Y coordinate: %s", err))
		return
	}

	startX, err := strconv.ParseFloat(startXStr, 64)
	if err != nil {
		println(fmt.Sprintf("error parsing starting X coordinate: %s", err))
		return
	}

	startDirection, err := strconv.ParseFloat(startDirectionStr, 64)
	if err != nil {
		println(fmt.Sprintf("error parsing starting direction: %s", err))
		return
	}

	println(fmt.Sprintf("Got starting coordinates: (%f, %f) facing %f", startX, startY, startDirection))

	numRows, err := strconv.Atoi(numRowsStr)
	if err != nil {
		println(fmt.Sprintf("error parsing number of rows: %s", err))
		return
	}

	numCols, err := strconv.Atoi(numColsStr)
	if err != nil {
		println(fmt.Sprintf("error parsing number of cols: %s", err))
		return
	}
	println(fmt.Sprintf("expecting arr: %d x %d", numCols, numRows))
	mazeSize := numRows * numCols
	mazebuf := make([]bool, mazeSize)
	mazePos := 0

	// change scanning strategy- newline chars are a valid possibility in our input maze
	f.Seek(numBytesRead, 0)
	scanner = bufio.NewScanner(f)
	scanner.Split(func(data []byte, isEOF bool) (int, []byte, error) {
		if len(data) == 0 && isEOF {
			return 0, nil, nil
		}
		return len(data), data, nil
	})
	for {
		if !scanner.Scan() {
			break
		}
		mazebufStr := scanner.Text()
		for _, repByte := range mazebufStr {
			rep := int(repByte)
			byteLen := 8
			remaining := mazeSize - mazePos
			if remaining < 0 {
				println("Data read exceeded expected maze size!")
				return
			}
			if remaining < byteLen {
				byteLen = remaining
			}
			for ix := 0; ix < byteLen; ix++ {
				if (rep | (1 << ix)) > 0 {
					mazebuf[mazePos] = true
				} else {
					mazebuf[mazePos] = false
				}
				mazePos++
			}
		}
	}
	if mazePos != mazeSize {
		println(fmt.Sprintf("Expected maze size: %d", mazeSize))
		println(fmt.Sprintf("Actual maze size: %d", mazePos))
		println("Cannot continue with size discrepancy.")
		return
	}

	gtk.Init(nil)
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("Goland!")
	window.SetIconName("gtk-dialog-info")
	window.Connect("destroy", func(ctx *glib.CallbackContext) {
		println("goland got destroyed!", ctx.Data().(string))
		msgCh <- End
		gtk.MainQuit()
	}, "goland")

	pixbuf := gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, 8, 640, 480)
	img := gtk.NewImageFromPixbuf(pixbuf)

	window.Add(img)
	window.SetSizeRequest(640, 480)
	window.ShowAll()

	go doStatic(msgCh, img, pixbuf)

	gtk.Main()

}
