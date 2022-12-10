package gol

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

const WhiteAlive = 255
const BlackNotAlive = 0

// mesh
type MeshStruct struct {
	Live [][]bool // True means the cell is alive, false means the cell is dead
	W    int      // width
	H    int      // height
}

type MainStruct struct {
	NowMesh *MeshStruct // now mesh
	TmpMesh *MeshStruct // save mesh for a short time
	W       int         // width
	H       int         // height
	T       int         // threads
}

// init mesh
func NewMesh(w, h, t int) *MeshStruct {
	if w == 0 || h == 0 || t == 0 {
		panic(fmt.Sprintf("NewMesh error params, w=%v, h=%v, t=%v", w, h, t))
	}

	tmpStatus := make([][]bool, h)
	for i := 0; i < h; i++ {
		tmpStatus[i] = make([]bool, w)
	}

	meshStruct := &MeshStruct{
		W:    w,
		H:    h,
		Live: tmpStatus,
	}

	return meshStruct
}

// init main struct
func NewMainStruct(w, h, t int) *MainStruct {
	if w == 0 || h == 0 || t == 0 {
		panic(fmt.Sprintf("NewMainStruct error params, w=%v, h=%v, t=%v", w, h, t))
	}

	mainStruct := &MainStruct{
		H:       h,
		W:       w,
		T:       t,
		NowMesh: NewMesh(w, h, t),
		TmpMesh: NewMesh(w, h, t),
	}

	return mainStruct
}

// Calculate the cell coordinate position, because it may be out of bounds
func (g *MeshStruct) Alive(x, y int) bool {
	tmp1 := x + g.W
	tmp1 %= g.W

	tmp2 := y + g.H
	tmp2 %= g.H

	status := g.Live[tmp1][tmp2]
	return status
}

// set (x, y)cell Live
func (g *MeshStruct) Set(x, y int, status bool) {
	g.Live[x][y] = status
	return
}

// Next round (x, y) coordinate cell state calculation
func (g *MeshStruct) NextCalculate(x, y int) bool {
	count := 0
	for index1 := -1; index1 <= 1; index1++ {
		for index2 := -1; index2 <= 1; index2++ {
			if index1 == 0 && index2 == 0 {
				continue
			}
			if g.Alive(x+index1, y+index2) {
				count++
			}
		}
	}

	if count == 3 || (count == 2 && g.Alive(x, y)) {
		return true
	}

	return false
}

/*

8 / 3 = 2
(0-1)(2-3)(4-7)
1 1 2 2 3 3 3 3
1 1 2 2
1 1 2 2
1 1 2 2

 */

// The next round of all cell state calculations, concurrent logic is implemented here
func (w *MainStruct) NextStep() {
	var waitGroup *sync.WaitGroup = &sync.WaitGroup{}
	// be calculated for each thread
	lenght := w.W / w.T   // 2 = 8 / 3
	waitGroup.Add(w.T)
	for i := 0; i < w.T; i++ {
		var left, right int

		// start to calculate
		left = i * lenght          // 0 = 0 * 2;    2 = 1 * 2;     4 = 2 * 2
		right = left + lenght - 1  // 1 = 0 + 2 -1; 3 = 2 + 2 - 1; 5 = 4 + 2 - 1 --> 7 特殊case 最后一个right需要改写为最大横坐标

		// last end index
		if i == w.T-1 {
			right = w.W - 1
		}

		go func(waitGroup *sync.WaitGroup, index, startIndex, endIndex int) {
			defer waitGroup.Done()
			for indexX := 0; indexX < w.H; indexX++ {
				for indexY := startIndex; indexY <= endIndex; indexY++ {
					w.TmpMesh.Set(indexX, indexY, w.NowMesh.NextCalculate(indexX, indexY))
				}
			}
		}(waitGroup, i, left, right)
	}
	waitGroup.Wait()

	w.NowMesh, w.TmpMesh = w.TmpMesh, w.NowMesh
	return
}

// The format of calculation file name is [H] x [W]
func (w *MainStruct) GenNameHW() string {
	name := strconv.Itoa(w.H) + "x" + strconv.Itoa(w.W)
	return name
}

// The format of calculation file name is [H] x [W] x [T]
func (w *MainStruct) GenNameHWT(turn int) string {
	name := strconv.Itoa(w.H) + "x" + strconv.Itoa(w.W) + "x" + strconv.Itoa(turn)
	return name
}

// init
func (w *MainStruct) InitMeshCell(c distributorChannels) {
	c.ioCommand <- ioInput
	c.ioFilename <- w.GenNameHW()

	for indexX := 0; indexX < w.H; indexX++ {
		for indexY := 0; indexY < w.W; indexY++ {
			tmp := <-c.ioInput
			w.NowMesh.Set(indexX, indexY, tmp == WhiteAlive)
		}
	}
	return
}

// Calculate the number of cells currently alive
func (w *MainStruct) GenAliveCellCount() int {
	c := 0
	for index1 := 0; index1 < w.H; index1++ {
		for index2 := 0; index2 < w.W; index2++ {
			if w.NowMesh.Alive(index1, index2) {
				c++
			}
		}
	}

	return c
}

//send final event to events channel
func (w *MainStruct) SendFinalEventToEventChannel(turn int, c distributorChannels) {
	finalTurnComplete := FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          make([]util.Cell, 0),
	}

	for index2 := 0; index2 < w.H; index2++ {
		for index1 := 0; index1 < w.W; index1++ {
			if w.NowMesh.Live[index2][index1] {
				finalTurnComplete.Alive = append(finalTurnComplete.Alive, util.Cell{X: index1, Y: index2})
			}
		}
	}

	c.events <- finalTurnComplete
	return
}

// send alive count to event channel
func (w *MainStruct) SendCountToEventChannel(turn int, c distributorChannels) {
	event := AliveCellsCount{
		CompletedTurns: turn,
		CellsCount:     w.GenAliveCellCount(),
	}
	c.events <- event
	return
}

// send cpmplete sign to event channel
func (w *MainStruct) SendSignCompleteToEventChannel(turn int, c distributorChannels) {
	event := TurnComplete{
		CompletedTurns: turn,
	}
	c.events <- event
	return
}

func (w *MainStruct) GenPgm(turn int, c distributorChannels) {
	c.ioCommand <- ioOutput
	c.ioFilename <- w.GenNameHWT(turn)
	for index1 := 0; index1 < w.H; index1++ {
		for index2 := 0; index2 < w.W; index2++ {
			if w.NowMesh.Alive(index1, index2) {
				c.ioOutput <- WhiteAlive
			} else {
				c.ioOutput <- BlackNotAlive
			}
		}
	}
	return
}

func (w *MainStruct) AliveStatusChange(y, x int) bool {
	return w.NowMesh.Live[y][x] != w.TmpMesh.Live[y][x]
}

func (w *MainStruct) SendCellChange(t int, c distributorChannels) {
	for index2 := 0; index2 < w.H; index2++ {
		for index1 := 0; index1 < w.W; index1++ {
			if !w.AliveStatusChange(index2, index1) {
				continue
			}
			// If the status changes from the previous era, send the event
			event := CellFlipped{
				CompletedTurns: t,
				Cell:           util.Cell{X: index1, Y: index2},
			}
			c.events <- event
		}
	}
	return
}

func (w *MainStruct) CloseAllAndQuit(turn int, c distributorChannels) {
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	sc := StateChange{
		CompletedTurns: turn,
		NewState:       Quitting,
	}
	c.events <- sc
	return
}

func (w *MainStruct) DefaultAction(t int, c distributorChannels) {
	w.NextStep()
	w.SendCellChange(t, c)
	w.SendSignCompleteToEventChannel(t, c)
	return
}

func (w *MainStruct) QuitAction(t int, c distributorChannels) {
	w.GenPgm(t, c)
	w.CloseAllAndQuit(t, c)
	close(c.events)
	return
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	// TODO: Create a 2D slice to store the world.
	world := NewMainStruct(p.ImageWidth, p.ImageHeight, p.Threads)
	t := 0
	world.InitMeshCell(c)
	// Send Initial Position
	world.SendCellChange(t, c)
	fmt.Println("start")

	// TODO: Execute all turns of the Game of Life.
	ticker := time.NewTicker(2 * time.Second)
	for t < p.Turns {
		select {
		case ope := <-c.keyPresses:
			if ope == 's' {
				// Store pgm pictures
				world.GenPgm(t, c)
			} else if ope == 'q' {
				// Store pgm pictures
				world.QuitAction(t, c)
				return
			} else if ope == 'p' {
				for {
					Operator := <-c.keyPresses
					if Operator == 'p' {
						fmt.Println("Recovery!")
						break
					}
				}
			}
		case <-ticker.C:
			world.SendCountToEventChannel(t, c)
		default:
			t++
			world.DefaultAction(t, c)
			//time.Sleep(time.Millisecond * 10)
		}
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	// Send the last state of the cell
	world.SendFinalEventToEventChannel(t, c)

	// Store pgm pictures
	world.GenPgm(t, c)

	// Make sure that the Io has finished any output before exiting.
	world.CloseAllAndQuit(t, c)

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}