package goland

import (
	"math/rand"
)

func static(completedCh chan int, display []byte, color int) {
	defer func() {
		completedCh <- End
	}()
	for i, _ := range display {
		if i%3 == color {
			display[i] = (byte)(rand.Int() % 256)
		} else {
			display[i] = 0
		}
	}
}

func doStatic(msgCh chan int, img *gtk.Image, pixbuf *gdkpixbuf.Pixbuf) {

	x := pixbuf.GetWidth()
	y := pixbuf.GetHeight()
	z := 3

	indexStep := x * y * z / NUMWORKERS

	completedCh := make(chan int)

	display := pixbuf.GetPixels()

	println(fmt.Sprintf("Got display: %d size", len(display)))

	for {
		select {
		case msg := <-msgCh:
			if msg == End {
				return
			}
			println("you goofed")
			return
		default:
			indexStart := 0
			indexStop := indexStep
			for i := 0; i < NUMWORKERS; i++ {
				subdisplay := display[indexStart:indexStop]
				go static(completedCh, subdisplay, i%3)
				indexStart = indexStop
				indexStop += indexStep
			}
			numCompleted := 0
			for {
				if isCompleted := <-completedCh; isCompleted == End {
					numCompleted++
				}
				if numCompleted >= NUMWORKERS {
					break
				}
			}
			img.SetFromPixbuf(pixbuf)
		}
		time.Sleep(LOOPSLEEP)
	}

}
