package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	"github.com/eiannone/keyboard"
)

const (
	// FPS_LIMIT = 60
	HEIGHT     = 20
	WIDTH      = 80
	LAND       = '.'
	SNAKE      = 'S'
	FOOD       = 'O'
	LEFT_MOVE  = "a"
	UP_MOVE    = "w"
	RIGHT_MOVE = "d"
	DOWN_MOVE  = "s"
)

type Game struct {
	Access        sync.Mutex
	Land          [][]byte
	SnakePosition []int
	FoodPosition  []int
	KeyChannel    chan string
}

func CreateGame() *Game {
	return &Game{
		Land:       CreateLand(),
		KeyChannel: make(chan string),
	}
}

func (g *Game) SpawnSnake() {
	h, w := HEIGHT/2, WIDTH/2
	g.Land[h][w] = SNAKE
	g.SnakePosition = []int{h, w}
}

func (g *Game) SpawnFood() {
	h, w := rand.Intn(HEIGHT), rand.Intn(WIDTH)
	g.Land[h][w] = FOOD
	g.FoodPosition = []int{h, w}
}

func (g *Game) Render() {
	buf := new(bytes.Buffer)
	for range time.Tick(100 * time.Millisecond) { // FPS
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()

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

func (g *Game) MoveSnake() {
	moveDirection := func(direction string) {
		// TODO: find the way to deal with inputs
	}

	for {
		move := <-g.KeyChannel
		// g.Access.Lock()
		// g.Land[g.SnakePosition[0]][g.SnakePosition[1]] = LAND
		switch move {
		case LEFT_MOVE:
			// g.SnakePosition[1]--
			go moveDirection(LEFT_MOVE)
		case UP_MOVE:
			// g.SnakePosition[0]--
			go moveDirection(UP_MOVE)
		case RIGHT_MOVE:
			// g.SnakePosition[1]++
			go moveDirection(RIGHT_MOVE)
		case DOWN_MOVE:
			// g.SnakePosition[0]++
			go moveDirection(DOWN_MOVE)
		}
		// g.Land[g.SnakePosition[0]][g.SnakePosition[1]] = SNAKE
		// g.Access.Unlock()
	}
}

func (g *Game) Run() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	<-s

	fmt.Println("Game Over!")
}

func main() {
	game := CreateGame()
	game.SpawnSnake()
	game.SpawnFood()

	go game.Render()
	go game.ListenInputs()
	go game.MoveSnake()
	// go game.WatchSnake()

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
