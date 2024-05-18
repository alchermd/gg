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
	gg := NewGG(logger, in, out)

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
	cmdInvalid = "invalid"
	cmdExit    = "exit"
)

// ==============================================================================
// GG specific definitions and "main game" methods.
// ==============================================================================

// GG is the game application instance.
type GG struct {
	logger         *log.Logger
	gameInProgress bool
	in             Input
	out            Output
}

// NewGG initializes a new GG instance.
func NewGG(logger *log.Logger, in Input, out Output) *GG {
	return &GG{
		logger: logger,
		in:     in,
		out:    out,
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

// DrawBoard displays a graphical representation of the current game state.
func (g *GG) DrawBoard() {
	g.logger.Println("drawing board.")
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
func NewStdinInput() Input {
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

// NewStdoutOutput initializes a new StdoutOutput.
func NewStdoutOutput() Output {
	return &StdoutOutput{}
}
