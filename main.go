package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	logger := log.New(os.Stdout, "gg: ", log.LstdFlags|log.Lshortfile)
	in := NewStdinInput()
	out := NewStdoutOutput()
	gui := NewConsoleGUI(out)
	gg := NewGG(logger, in, out, gui)

	gg.Start()

	for gg.MainLoop() {
		gg.DrawBoard()
		gg.GetCommand()
		gg.DetermineResult()
		gg.ShowResult()
	}

	// TODO: Implement graceful shutdown (ex: CTRL+C from Stdout).
	gg.Quit()
}

// ==============================================================================
// CONSTANTS
// ==============================================================================
const (
	// Commands.
	cmdInvalid = "invalid"
	cmdExit    = "exit"

	// Board dimensions.
	rows  = 8
	files = 9
)

// ==============================================================================
// GG specific definitions and "main game" methods.
// ==============================================================================

// GG is the game application instance.
type GG struct {
	// Game logic properties.
	gameInProgress bool
	board          GGBoard

	// Ancillary dependencies.
	logger *log.Logger
	in     Input
	out    Output
	gui    GUI
}

// GGBoard is a 2D array for GGSquares.
type GGBoard [rows][files]GGSquare

// GGSquare represents a square on the game board.
type GGSquare struct{}

// NewGG initializes a new GG instance.
func NewGG(logger *log.Logger, in Input, out Output, gui GUI) *GG {
	var board GGBoard

	for i := 0; i < rows; i++ {
		for j := 0; j < files; j++ {
			board[i][j] = GGSquare{}
		}
	}

	return &GG{
		// Game logic properties.
		gameInProgress: false,
		board:          board,

		// Ancillary dependencies.
		logger: logger,
		in:     in,
		out:    out,
		gui:    gui,
	}
}

// Start kicks off any processes to start a GG game.
func (g *GG) Start() {
	g.logger.Println("starting GG...")
	g.gameInProgress = true
}

// Close terminates the game.
func (g *GG) Close() {
	g.logger.Println("Closing GG...")
}

// MainLoop is the game's main loop, returning whether the game is finished or not.
func (g *GG) MainLoop() bool {
	return g.gameInProgress
}

// DrawBoard displays a graphical representation of the current game state.
func (g *GG) DrawBoard() {
	g.logger.Println("drawing board.")
	g.gui.Draw(g.board)
}

// GetCommand fetches the next player's command and invokes an appropriate handler.
func (g *GG) GetCommand() {
	g.logger.Println("fetching player command.")

	g.out.Write("Enter command: ")
	cmd := g.in.Read()

	// TODO: Could perhaps be extracted to its own method since
	//  it's technically not "GetCommand" related.
	//  Also consider if it's better to use a map[string]func instead
	switch cmd {
	case cmdExit:
		g.HandleExit()
	default:
		g.HandleInvalid()
	}
}

// DetermineResult calculates the game's result from the current game state.
func (g *GG) DetermineResult() {
	g.logger.Println("determining result.")
}

// ShowResult reports the "result" (i.e. what next step is needed) of the current game state.
func (g *GG) ShowResult() {
	g.logger.Println("showing result.")
}

// Quit allows the game to execute any cleanup routines.
func (g *GG) Quit() {
	g.logger.Println("quitting game.")
}

// ==============================================================================
// GG handler methods -- as opposed to "main game" methods.
// These are prefixed by "Handle" and are invoked by user input.
// ==============================================================================

// HandleExit handles the "exit" command.
func (g *GG) HandleExit() {
	g.logger.Println("exiting game loop.")
	g.gameInProgress = false
}

// HandleInvalid handles a command not supported by the game.
func (g *GG) HandleInvalid() {
	fmt.Println("Invalid command.")
}

// ==============================================================================
// IO definitions and methods. Used for managing input and output.
// ==============================================================================

// Input is the interface for fetching input from the outside world.
type Input interface {
	Read() string
}

// StdinInput allows fetching of input from Stdin.
type StdinInput struct{}

// Read takes in a string from Stdin, cleans it, and returns it.
func (i *StdinInput) Read() string {
	reader := bufio.NewReader(os.Stdin)
	cmd, err := reader.ReadString('\n')
	if err != nil {
		return cmdInvalid
	}
	return strings.TrimSpace(cmd)
}

// NewStdinInput initializes a new StdinInput.
func NewStdinInput() *StdinInput {
	return &StdinInput{}
}

// Output is the interface for writing output to the outside world.
type Output interface {
	Write(s string)
}

// StdoutOutput allows writing of output to Stdout.
type StdoutOutput struct{}

// Write prints the given string to Stdout.
func (o *StdoutOutput) Write(s string) {
	fmt.Print(s)
}

// GUI is the interface for handling interactable game elements.
type GUI interface {
	Draw(GGBoard)
}

// ConsoleGUI is a GUI implemented via console.
type ConsoleGUI struct {
	out *StdoutOutput
}

// NewConsoleGUI initializes a ConsoleGUI.
func NewConsoleGUI(out *StdoutOutput) GUI {
	return &ConsoleGUI{out: out}
}

func (g ConsoleGUI) Draw(board GGBoard) {
	// Draw header
	g.out.Write("\n")
	g.out.Write(fmt.Sprintf("%s\n", strings.Repeat("=", 50)))
	g.out.Write("Current game state:\n")

	// Draw actual board.
	for i := rows; i >= 0; i-- {
		// Draw top edge.
		for j := 0; j < files; j++ {
			g.out.Write(" ----")
		}
		g.out.Write("\n")

		// Draw each square.
		for j := 0; j < files; j++ {
			// TODO: We should only show per-square coordinates when setting up the board.
			g.out.Write(fmt.Sprintf("| %s ", squareAddressToCoordinates(j, i)))
		}
		g.out.Write("|\n")

		if i == 0 {
			// Draw bottom edge.
			for j := 0; j < files; j++ {
				g.out.Write(" ----")
			}
			g.out.Write("\n")
		}
	}

	// Draw footer
	g.out.Write(fmt.Sprintf("%s\n", strings.Repeat("=", 50)))
	g.out.Write("\n")
}

// NewStdoutOutput initializes a new StdoutOutput.
func NewStdoutOutput() *StdoutOutput {
	return &StdoutOutput{}
}

// ==============================================================================
// Utility / helper functions.
// ==============================================================================

// squareAddressToCoordinates is a helper function to a square's index to its readable coordinate.
// example: 03 -> A4, 57 -> F8.
func squareAddressToCoordinates(x int, y int) string {
	if x > rows || y > files {
		return ""
	}

	alpha := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	return fmt.Sprintf("%s%d", alpha[x], y+1)
}
