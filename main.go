package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
)

const (
	// FPS_LIMIT = 60
	HEIGHT = 20
	WIDTH  = 80
	LAND   = '.'
	SNAKE  = 'S'
	FOOD   = 'O'
)

// Directions
const (
	LEFT_MOVE  = "a"
	UP_MOVE    = "w"
	RIGHT_MOVE = "d"
	DOWN_MOVE  = "s"
)

type Game struct {
	Access sync.Mutex
	Land   [][]byte
	Snake  Snake
	Food   Food
	// State      chan struct{}
	KeyChannel          chan string
	SystemSignalChannel chan os.Signal
}

type Snake struct {
	Head          *Part
	Tail          *Part
	MoveDirection string
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
		Land: CreateLand(),
		// State:      make(chan struct{}),
		KeyChannel: make(chan string),
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
	g.Snake.MoveDirection = RIGHT_MOVE
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
	case LEFT_MOVE:
		y++

	case UP_MOVE:
		x++

	case RIGHT_MOVE:
		y--

	case DOWN_MOVE:
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

func (g *Game) MoveSnake(direction string) {
	if (direction == LEFT_MOVE && g.Snake.MoveDirection != RIGHT_MOVE) ||
		(direction == RIGHT_MOVE && g.Snake.MoveDirection != LEFT_MOVE) ||
		(direction == UP_MOVE && g.Snake.MoveDirection != DOWN_MOVE) ||
		(direction == DOWN_MOVE && g.Snake.MoveDirection != UP_MOVE) {
		g.Snake.MoveDirection = direction
	}

	g.Access.Lock()
	g.MovePart(g.Snake.MoveDirection)
	g.Access.Unlock()
}

func (g *Game) MovePart(direction string) {
	cur := g.Snake.Head

	x, y := cur.Position[0], cur.Position[1]
	switch direction {
	case LEFT_MOVE:
		y--
	case UP_MOVE:
		x--
	case RIGHT_MOVE:
		y++
	case DOWN_MOVE:
		x++
	}

	if x == -1 || x == HEIGHT || y == -1 || y == WIDTH || g.CheckForBodyCollision(x, y) {
		g.SystemSignalChannel <- syscall.SIGINT
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
	buf := new(bytes.Buffer)
	for range time.Tick(200 * time.Millisecond) { // FPS
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()

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

		for _, h := range g.Land {
			for _, w := range h {
				buf.WriteString(string(w))
			}
			buf.WriteString("\n")
		}
		buf.WriteTo(os.Stdout)
	}
}

func (g *Game) ListenInputs() {
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer keyboard.Close()

	for {
		ch, key, err := keyboard.GetKey()
		if err != nil {
			continue
		}

		g.KeyChannel <- string(ch)

		if key == keyboard.KeyCtrlC {
			break
		}
	}
}

func (g *Game) Run() {
	g.SystemSignalChannel = make(chan os.Signal, 1)
	signal.Notify(g.SystemSignalChannel, os.Interrupt)
	<-g.SystemSignalChannel

	fmt.Println("Game Over!")
}

func main() {
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
