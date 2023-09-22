package main

import (
	"fmt"
    "github.com/five-hundred-eleven/goland-game/goland"
	"github.com/mattn/go-gtk/gdkpixbuf"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	"os"
)

func main() {

	msgCh := make(chan int)

	mazeFilePath := "maze.txt"
	if len(os.Args) > 1 {
		mazeFilePath = os.Args[1]
	}

	game, err := goland.NewGameFromFilename(mazeFilePath)
	if err != nil {
		println(fmt.Sprintf("Got error constructing game: %s", err))
		return
	}

	println(fmt.Sprintf("Got game: %d x %d", game.Cols, game.Rows))

	gtk.Init(nil)
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("Goland!")
	window.SetIconName("gtk-dialog-info")
	window.Connect("destroy", func(ctx *glib.CallbackContext) {
		println("goland got destroyed!", ctx.Data().(string))
		msgCh <- goland.End
		gtk.MainQuit()
	}, "goland")

	pixbuf := gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, 8, 640, 480)
	img := gtk.NewImageFromPixbuf(pixbuf)

	window.Add(img)
	window.SetSizeRequest(640, 480)
	window.ShowAll()

	go goland.DoMaze(msgCh, img, pixbuf, game)

	gtk.Main()

}
