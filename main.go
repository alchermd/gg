package main

import (
	"bufio"
	_flag "flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	withLogs := _flag.Bool("logs", false, "whether to show logs.")
	_flag.Parse()

	logger := log.New(os.Stdout, "gg: ", log.LstdFlags|log.Lshortfile)
	if !*withLogs {
		logger.SetOutput(io.Discard)
	}

	in := NewStdinInput()
	out := NewStdoutOutput()
	gui := NewConsoleGUI(out)
	gg := NewGG(logger, in, out, gui)

	gg.Start()

	for gg.MainLoop() {
		gg.DrawBoard()
		gg.GetCommand()
		gg.ResolveCommand()
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
	cmdHelp       = "help"
	cmdInvalid    = "invalid"
	cmdExit       = "exit"
	cmdLoadSample = "loadsample"

	// File paths.
	sampleGggnFile = "setup.gggn"

	// Board dimensions.
	rows  = 8
	files = 9

	// Pieces
	fiveStarGeneral  GGPieceCode = "5*G"
	fourStarGeneral  GGPieceCode = "4*G"
	threeStarGeneral GGPieceCode = "3*G"
	twoStarGeneral   GGPieceCode = "2*G"
	oneStarGeneral   GGPieceCode = "1*G"
	colonel          GGPieceCode = "COL"
	ltColonel        GGPieceCode = "LTC"
	major            GGPieceCode = "MAJ"
	captain          GGPieceCode = "CPT"
	firstLt          GGPieceCode = "1LT"
	secondLt         GGPieceCode = "2LT"
	sergeant         GGPieceCode = "SGT"
	private          GGPieceCode = "PVT"
	spy              GGPieceCode = "SPY"
	flag             GGPieceCode = "FLG"

	// Players
	w GGPlayer = "W"
	b GGPlayer = "B"
)

// ==============================================================================
// GG specific definitions and "main game" methods.
// ==============================================================================

// GG is the game application instance.
type GG struct {
	// Game logic properties.
	gameInProgress bool
	board          GGBoard
	commandStack   *GGCommandStack

	// Ancillary dependencies.
	logger *log.Logger
	in     Input
	out    Output
	gui    GUI
}

// GGBoard is a 2D array for GGSquares.
type GGBoard [rows][files]GGSquare

// GGSquare represents a square on the game board.
type GGSquare struct {
	piece GGPiece
}

// GGPiece represents a game piece.
type GGPiece struct {
	code   GGPieceCode
	player GGPlayer
}

// GGPieceCode represents a piece code (ex: "FLG" for Flag).
type GGPieceCode string

// GGPlayer represents a player (ex: "W" for the player with the White pieces).
type GGPlayer string

// GGCommandStack is an append-only, head-only read store for player commands.
type GGCommandStack struct {
	commands []string
}

// Append appends the given command to the stack.
func (s *GGCommandStack) Append(cmd string) {
	s.commands = append(s.commands, cmd)
}

// Clear resets the stack.
func (s *GGCommandStack) Clear() {
	s.commands = []string{}
}

// Read returns the head of the stack, an empty string if the stack is empty.
func (s *GGCommandStack) Read() string {
	if len(s.commands) == 0 {
		return ""
	}

	return s.commands[len(s.commands)-1]
}

// NewGG initializes a new GG instance.
func NewGG(logger *log.Logger, in Input, out Output, gui GUI) *GG {
	return &GG{
		// Game logic properties.
		gameInProgress: false,
		board:          GGBoard{},
		commandStack:   &GGCommandStack{},

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
	g.HandleHelp()
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

// GetCommand fetches the next player's command and stores it into the command stack.
func (g *GG) GetCommand() {
	g.logger.Println("fetching player command.")

	g.out.Write("Enter command: ")
	cmd := g.in.Read()
	g.commandStack.Append(cmd)
}

// ResolveCommand reads the last command and invokes the appropriate handler.
func (g *GG) ResolveCommand() {
	cmd := g.commandStack.Read()

	setCmdRegex := regexp.MustCompile(`^SET [WB] [ABCDEFGHI][12345678] .{3}$`)

	if cmd == cmdExit {
		g.HandleExit()
	} else if cmd == cmdHelp {
		g.HandleHelp()
	} else if cmd == cmdLoadSample {
		g.HandleLoadSample()
	} else if setCmdRegex.FindString(cmd) != "" {
		g.HandleSet(cmd)
	} else {
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

// HandleHelp shows the help message.
func (g *GG) HandleHelp() {
	g.out.Write("Available commands:\n")
	g.out.Write("\t* SET: Set a piece into the board.\n")
	g.out.Write("\t\t* Syntax: SET W|P COORD PIECECODE\n")
	g.out.Write("\t* loadsample: Loads a sample game file.\n")
	g.out.Write("\t* help: Show this help message.\n")
	g.out.Write("\t* exit: Exit the game.\n")
}

// HandleSet parses the given command and places the piece into the given coordinates.
func (g *GG) HandleSet(cmd string) {
	tokens := strings.Split(cmd, " ")
	// TODO : Validate these inputs.
	player := tokens[1]
	coordinates := tokens[2]
	pieceCode := tokens[3]

	x, y := coordinatesToSquareAddress(coordinates)
	piece := GGPiece{player: GGPlayer(player), code: GGPieceCode(pieceCode)}
	g.board[x][y].piece = piece
	g.logger.Printf("Player %v places %v on %v", player, pieceCode, coordinates)
}

// HandleLoadSample opens a sample .gggn file (GG Game notation) and executes the contents.
func (g *GG) HandleLoadSample() {
	f, _ := os.Open(sampleGggnFile)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		currentLine := scanner.Text()

		// Skip empty lines.
		if currentLine == "" {
			continue
		}

		// Ignore comments.
		if currentLine[0] == '#' {
			continue
		}

		g.HandleSet(currentLine)
	}

	g.out.Write(fmt.Sprintf("File %s successfully loaded\n", f.Name()))
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
	trimmedCmd := strings.TrimSpace(cmd)
	singleSpacedCmd := strings.Join(strings.Fields(trimmedCmd), " ")
	return singleSpacedCmd
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

// Draw draws the given board to the console.
func (g ConsoleGUI) Draw(board GGBoard) {
	// Draw header
	g.out.Write(fmt.Sprintf("%s\n", strings.Repeat("=", 80)))

	// Draw actual board.
	g.out.Write("\n")
	for i := len(board) - 1; i >= 0; i-- {
		g.out.Write("    ")
		// Draw top edge.
		for j := 0; j < len(board[i]); j++ {
			g.out.Write(" -------")
		}
		g.out.Write("\n")

		// Draw each square.
		g.out.Write("    ")
		for j := 0; j < len(board[i]); j++ {
			code := board[i][j].piece.code
			if code == "" {
				g.out.Write("|       ")
			} else {
				g.out.Write(fmt.Sprintf("|  %s  ", code))
			}
		}
		g.out.Write("|\n")

		if i == 0 {
			// Draw bottom edge.
			g.out.Write("    ")
			for j := 0; j < len(board[i]); j++ {
				g.out.Write(" -------")
			}
			g.out.Write("\n")
		}
	}

	// Draw footer
	g.out.Write("\n")
	g.out.Write(fmt.Sprintf("%s\n", strings.Repeat("=", 80)))
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

// coordinatesToSquareAddress converts a coordinate string to its actual board index.
// example: B7 -> (1, 6)
func coordinatesToSquareAddress(coordinates string) (int, int) {
	rowNumber, _ := strconv.Atoi(string(coordinates[1]))
	fileName := string(coordinates[0])

	filesMap := map[string]int{
		"A": 0,
		"B": 1,
		"C": 2,
		"D": 3,
		"E": 4,
		"F": 5,
		"G": 6,
		"H": 7,
		"I": 8,
	}

	return rowNumber - 1, filesMap[fileName]
}
