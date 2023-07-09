package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

// Default settings
const (
	HEIGHT = 20
	WIDTH  = 80
	LAND   = '.'
	SNAKE  = 'S'
	FOOD   = 'O'
)

// Colors
const (
	FOOD_COLOR  = termbox.ColorRed
	SNAKE_COLOR = termbox.ColorGreen
	LAND_COLOR  = termbox.ColorYellow
	FOREGROUND  = termbox.ColorWhite
	BACKGROUND  = termbox.ColorBlack
)

type Game struct {
	Access              sync.Mutex
	Land                *Land
	Snake               Snake
	Food                Food
	KeyChannel          chan termbox.Key
	SystemSignalChannel chan termbox.Key
}

type Land struct {
	Cells  [][]byte
	Color  termbox.Attribute
	Height int
	Width  int
}

type Snake struct {
	Head          *Part
	Tail          *Part
	Color         termbox.Attribute
	MoveDirection termbox.Key
	Length        int
	State         chan struct{}
}

type Part struct {
	Position []int
	Next     *Part
}

type Food struct {
	Position []int
	Color    termbox.Attribute
	State    chan struct{}
}

func CreateGame() *Game {
	return &Game{
		Land:       CreateLand(),
		KeyChannel: make(chan termbox.Key),
	}
}

// Snake
func (g *Game) SpawnSnake() {
	h, w := g.Land.Height/2, g.Land.Width/2
	g.Land.Cells[h][w] = SNAKE
	newPart := &Part{
		Position: []int{h, w},
	}
	g.Snake.Head = newPart
	g.Snake.Tail = newPart
	g.Snake.Color = SNAKE_COLOR
	g.Snake.MoveDirection = termbox.KeyArrowRight
	g.Snake.Length = 1
	g.Snake.State = make(chan struct{})
}

func (g *Game) WatchSnake() {
	for {
		<-g.Snake.State
		g.GrowSnake()
	}
}

func (g *Game) GrowSnake() {
	g.Snake.Length++
	x, y := g.Snake.Tail.Position[0], g.Snake.Tail.Position[1]
	switch g.Snake.MoveDirection {
	case termbox.KeyArrowLeft:
		y++

	case termbox.KeyArrowUp:
		x++

	case termbox.KeyArrowRight:
		y--

	case termbox.KeyArrowDown:
		x--
	}

	newPart := &Part{
		Position: []int{x, y},
	}

	g.Access.Lock()
	defer g.Access.Unlock()

	g.Snake.Tail.Next = newPart
	g.Snake.Tail = g.Snake.Tail.Next
}

func (g *Game) MoveSnake(direction termbox.Key) {
	if (direction == termbox.KeyArrowLeft && g.Snake.MoveDirection != termbox.KeyArrowRight) ||
		(direction == termbox.KeyArrowRight && g.Snake.MoveDirection != termbox.KeyArrowLeft) ||
		(direction == termbox.KeyArrowUp && g.Snake.MoveDirection != termbox.KeyArrowDown) ||
		(direction == termbox.KeyArrowDown && g.Snake.MoveDirection != termbox.KeyArrowUp) {
		g.Snake.MoveDirection = direction
	}

	g.Access.Lock()
	g.MovePart(g.Snake.MoveDirection)
	g.Access.Unlock()
}

func (g *Game) MovePart(direction termbox.Key) {
	cur := g.Snake.Head

	x, y := cur.Position[0], cur.Position[1]
	switch direction {
	case termbox.KeyArrowLeft:
		y--
	case termbox.KeyArrowUp:
		x--
	case termbox.KeyArrowRight:
		y++
	case termbox.KeyArrowDown:
		x++
	}

	if x == -1 || x == g.Land.Height || y == -1 || y == g.Land.Width || g.CheckForBodyCollision(x, y) {
		g.SystemSignalChannel <- termbox.KeyEsc
		return
	}

	for cur != nil {
		g.Land.Cells[cur.Position[0]][cur.Position[1]] = LAND
		g.Land.Cells[x][y] = SNAKE
		x, y, cur.Position[0], cur.Position[1] = cur.Position[0], cur.Position[1], x, y

		cur = cur.Next
	}
}

func (g *Game) CheckForBodyCollision(x, y int) bool {
	cur := g.Snake.Head

	for cur != nil {
		if cur.Position[0] == x && cur.Position[1] == y {
			return true
		}

		cur = cur.Next
	}

	return false
}

// Food
func (g *Game) InitSpawnFood() {
	h, w := rand.Intn(g.Land.Height), rand.Intn(g.Land.Width)
	g.Land.Cells[h][w] = FOOD
	g.Food.Color = FOOD_COLOR
	g.Food.Position = []int{h, w}
	g.Food.State = make(chan struct{})
}

func (g *Game) WatchFood() {
	for {
		<-g.Food.State
		g.SpawnFood()
	}
}

func (g *Game) SpawnFood() {
	h, w := rand.Intn(g.Land.Height), rand.Intn(g.Land.Width)
	if h != g.Snake.Head.Position[0] && w != g.Snake.Head.Position[1] {
		g.Access.Lock()
		g.Land.Cells[h][w] = FOOD
		g.Food.Position = []int{h, w}
		g.Access.Unlock()
	} else {
		g.SpawnFood()
	}
}

// Game staff
func (g *Game) Render() {
	for range time.Tick(60 * time.Millisecond) { // FPS
		// initially snake moves right direction
		{
			select {
			case direction := <-g.KeyChannel:
				g.MoveSnake(direction)

			default:
				g.MoveSnake(g.Snake.MoveDirection)
			}
		}

		// checking	food state
		{
			if g.Snake.Head.Position[0] == g.Food.Position[0] && g.Snake.Head.Position[1] == g.Food.Position[1] {
				g.Food.State <- struct{}{}
				g.Snake.State <- struct{}{}
			}
		}

		for x, row := range g.Land.Cells {
			for y, cell := range row {
				fg, bg := FOREGROUND, BACKGROUND
				switch cell {
				case FOOD:
					fg = g.Food.Color
				case SNAKE:
					fg = g.Snake.Color
				case LAND:
					fg = g.Land.Color
				}
				termbox.SetCell(y, x, rune(cell), fg, bg)
			}
		}
		termbox.Flush()
	}
}

func (g *Game) ListenInputs() {
	for {
		event := termbox.PollEvent()
		switch event.Type {
		case termbox.EventKey:
			if event.Key == termbox.KeyEsc || event.Key == termbox.KeyCtrlC {
				g.SystemSignalChannel <- event.Key
				return
			}

			g.KeyChannel <- event.Key
		}
	}
}

func (g *Game) Run() {
	g.SystemSignalChannel = make(chan termbox.Key, 1)
	<-g.SystemSignalChannel
}

func main() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputEsc)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	game := CreateGame()
	game.SpawnSnake()
	game.InitSpawnFood()

	go game.Render()
	go game.ListenInputs()
	go game.WatchSnake()
	go game.WatchFood()

	game.Run()

	termbox.Close()
	fmt.Printf("Game Over!\nYour score: %d\n", game.Snake.Length)
}

func CreateLand() *Land {
	height, width := HEIGHT, WIDTH
	land := make([][]byte, height)
	for h := 0; h < height; h++ {
		land[h] = make([]byte, width)
		for w := 0; w < width; w++ {
			land[h][w] = LAND
		}
	}
	return &Land{
		Cells:  land,
		Color:  LAND_COLOR,
		Height: height,
		Width:  width,
	}
}
