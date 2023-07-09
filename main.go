package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

const (
	// FPS_LIMIT = 60
	HEIGHT = 20
	WIDTH  = 80
	LAND   = '.'
	SNAKE  = 'S'
	FOOD   = 'O'
)

type Game struct {
	Access              sync.Mutex
	Land                [][]byte
	Snake               Snake
	Food                Food
	KeyChannel          chan termbox.Key
	SystemSignalChannel chan termbox.Key
}

type Snake struct {
	Head          *Part
	Tail          *Part
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
	h, w := HEIGHT/2, WIDTH/2
	g.Land[h][w] = SNAKE
	newPart := &Part{
		Position: []int{h, w},
	}
	g.Snake.Head = newPart
	g.Snake.Tail = newPart
	g.Snake.MoveDirection = termbox.KeyArrowRight
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

	if x == -1 || x == HEIGHT || y == -1 || y == WIDTH || g.CheckForBodyCollision(x, y) {
		g.SystemSignalChannel <- termbox.KeyEsc
		return
	}

	for cur != nil {
		g.Land[cur.Position[0]][cur.Position[1]] = LAND
		g.Land[x][y] = SNAKE
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
	h, w := rand.Intn(HEIGHT), rand.Intn(WIDTH)
	g.Land[h][w] = FOOD
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
	h, w := rand.Intn(HEIGHT), rand.Intn(WIDTH)
	if h != g.Snake.Head.Position[0] && w != g.Snake.Head.Position[1] {
		g.Access.Lock()
		g.Land[h][w] = FOOD
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

		for x, row := range g.Land {
			for y, cell := range row {
				termbox.SetCell(y, x, rune(cell), termbox.ColorWhite, termbox.ColorBlack)
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
	defer fmt.Println("Game Over!")

	if err := termbox.Init(); err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputEsc)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	defer termbox.Close()

	game := CreateGame()
	game.SpawnSnake()
	game.InitSpawnFood()

	go game.Render()
	go game.ListenInputs()
	go game.WatchSnake()
	go game.WatchFood()

	game.Run()
}

func CreateLand() [][]byte {
	land := make([][]byte, HEIGHT)
	for h := 0; h < HEIGHT; h++ {
		land[h] = make([]byte, WIDTH)
		for w := 0; w < WIDTH; w++ {
			land[h][w] = LAND
		}
	}
	return land
}
