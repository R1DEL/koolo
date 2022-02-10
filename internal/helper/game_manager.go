package helper

import (
	"errors"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/event"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/hid"
	"time"
)

func ExitGame(eventChan chan<- event.Event) {
	action.Run(
		action.NewKeyPress("esc", time.Millisecond*200),
		action.NewMouseDisplacement(640, 328, time.Millisecond*50),
		action.NewMouseClick(hid.LeftButton, time.Millisecond*120),
	)
	eventChan <- event.ExitedGame
}

func NewGame(difficulty string) error {
	difficultyPosition := map[string]struct {
		X, Y int
	}{
		"normal":    {X: 640, Y: 311},
		"nightmare": {X: 640, Y: 355},
		"hell":      {X: 640, Y: 403},
	}

	createX := difficultyPosition[difficulty].X
	createY := difficultyPosition[difficulty].Y
	action.Run(
		action.NewMouseDisplacement(640, 672, time.Millisecond*50),
		action.NewMouseClick(hid.LeftButton, time.Millisecond*350),
		action.NewMouseDisplacement(createX, createY, time.Millisecond*87),
		action.NewMouseClick(hid.LeftButton, time.Millisecond*65),
	)

	for i := 0; i < 20; i++ {
		if game.Status().Success {
			return nil
		}
		time.Sleep(time.Second)
	}

	return errors.New("error creating new game")
}
