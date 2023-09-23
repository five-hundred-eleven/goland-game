package main

import (
	"fmt"
	"github.com/five-hundred-eleven/goland-game/goland"
	"github.com/veandco/go-sdl2/sdl"
	"os"
)

const ()

func main() {

	mazeFilePath := "maze.txt"
	if len(os.Args) > 1 {
		mazeFilePath = os.Args[1]
	}

	game, err := goland.NewGameFromFilename(mazeFilePath)
	if err != nil {
		fmt.Sprintln("Got error constructing game: %s", err)
		return
	}

	println(fmt.Sprintf("Got game: %d x %d", game.Cols, game.Rows))

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		fmt.Sprintln("Unable to init! %s", err)
		return
	}
	defer sdl.Quit()

	screen := &goland.Screen{}
	//width, height := window.GetSize()
	screen.Width = goland.WINDOW_WIDTH
	screen.Height = goland.WINDOW_HEIGHT
	// this is a constant for right triangles
	screen.Depth = screen.Width / 2

	window, err := sdl.CreateWindow("Goland!", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, goland.WINDOW_WIDTH, goland.WINDOW_HEIGHT, 0)
	if err != nil {
		println(fmt.Sprintf("Unable to create goland! %s", err))
		return
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		println(fmt.Sprintf("Got error in CreateRenderer(): %s", err))
		return
	}
	defer renderer.Destroy()

	surface, err := window.GetSurface()
	if err != nil {
		println(fmt.Sprintf("Got error in GetSurface(): %s", err))
		return
	}
	screen.Format = surface.Format

	texture, err := renderer.CreateTexture(screen.Format.Format, sdl.TEXTUREACCESS_STREAMING, goland.WINDOW_WIDTH, goland.WINDOW_HEIGHT)
	if err != nil {
		println(fmt.Sprintf("Got error in CreateTexture(): %s", err))
		return
	}
	defer texture.Destroy()

	screen.Window = window
	screen.Renderer = renderer
	screen.Texture = texture

	defer window.Destroy()

	sendCh := make(chan int)
	recvCh := make(chan int)
	go goland.DoMaze(sendCh, recvCh, screen, game)

	go func() {
		defer func() {
			recvCh <- goland.End
		}()
		for {
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch event.(type) {
				case *sdl.QuitEvent:
					println("Got quit event!")
					return
				default:
					fmt.Sprintln("Got unrecognized event! %d", event.GetType())
				}
			}
		}
	}()

	for {
		msg := <-recvCh
		if msg == goland.End {
			println("Goland got destroyed!")
			break
		} else {
			println("Got unrecognized message??")
		}
	}

}
