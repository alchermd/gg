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

	// Game states.
	gamePreSetup   GGGameState = "PRE_SETUP"
	gameSetup      GGGameState = "SETUP"
	gameInProgress GGGameState = "IN_PROGRESS"
	gameOver       GGGameState = "GAME_OVER"

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

	// Movements
	moveMove      GGMoveType = "MOVE"
	moveChallenge GGMoveType = "CHALLENGE"
	moveInvalid   GGMoveType = "INVALID"

	// Results
	resChallengerWins  GGChallengeResult = "WIN"
	resChallengerLoses GGChallengeResult = "LOSE"
	resDraw            GGChallengeResult = "DRAW"

	// Players
	playerWhite GGPlayer = "W"
	playerBlack GGPlayer = "B"
)

// ==============================================================================
// GLOBALS
// ==============================================================================
var (
	// Regexp
	setCmdRegex = regexp.MustCompile(`^SET [WB] [ABCDEFGHI][12345678] .{3}$`)
	mvCmdRegex  = regexp.MustCompile(`^MV [ABCDEFGHI][12345678] [ABCDEFGHI][12345678]$`)
)

// ==============================================================================
// GG specific definitions and "main game" methods.
// ==============================================================================

// GG is the game application instance.
type GG struct {
	// Game logic properties.
	status       GGGameState
	winner       GGPlayer
	playerToMove GGPlayer
	board        GGBoard
	commandStack *GGCommandStack

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

// GGGameState represents the summary of the current game state.
type GGGameState string

// GGChallengeResult represents a result of a piece challenge.
type GGChallengeResult string

// GGMoveType represents the type of a piece move.
type GGMoveType string

func (s *GGSquare) To(targetSquare GGSquare) GGMoveType {
	// Can't move an empty square.
	if s.IsEmpty() {
		return moveInvalid
	}

	//	Can't challenge an allied piece.
	if s.piece.player == targetSquare.piece.player {
		return moveInvalid
	}

	// An empty target is a move, otherwise it's a challenge
	if targetSquare.IsEmpty() {
		return moveMove
	}
	return moveChallenge
}

// IsEmpty checks if the square is not occupied by a piece.
func (s *GGSquare) IsEmpty() bool {
	return s.piece == (GGPiece{})
}

// Clear replaces the current inhabitant piece with an empty one.
func (s *GGSquare) Clear() {
	s.piece = GGPiece{}
}

// GGPiece represents a game piece.
type GGPiece struct {
	code   GGPieceCode
	player GGPlayer
}

// Power returns a numerical representation of a piece's strength.
// Note that this does not account any special piece rules -- only use this
// for determining results of a basic piece challenger.
func (p GGPiece) Power() int {
	piecePowerMap := map[GGPieceCode]int{
		fiveStarGeneral:  12,
		fourStarGeneral:  11,
		threeStarGeneral: 10,
		twoStarGeneral:   9,
		oneStarGeneral:   8,
		colonel:          7,
		ltColonel:        6,
		major:            5,
		captain:          4,
		firstLt:          3,
		secondLt:         2,
		sergeant:         1,
		private:          0,
		spy:              99,
		flag:             -1,
	}

	return piecePowerMap[p.code]
}

// GGPieceCode represents a piece code (ex: "FLG" for Flag).
type GGPieceCode string

// GGPlayer represents a player (ex: "W" for the player with the White pieces).
type GGPlayer string

// String returns a user-friendly name of the player.
func (p GGPlayer) String() string {
	if p == playerWhite {
		return "White"
	} else if p == playerBlack {
		return "Black"
	}

	return ""
}

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
		status:       gamePreSetup,
		board:        GGBoard{},
		commandStack: &GGCommandStack{},
		playerToMove: playerWhite,

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
	g.status = gameSetup
}

// Close terminates the game.
func (g *GG) Close() {
	g.logger.Println("Closing GG...")
}

// MainLoop is the game's main loop, returning whether the game is finished or not.
func (g *GG) MainLoop() bool {
	return g.status != gameOver
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

	if cmd == cmdExit {
		g.HandleExit()
	} else if cmd == cmdHelp {
		g.HandleHelp()
	} else if cmd == cmdLoadSample {
		g.HandleLoadSample()
	} else if setCmdRegex.FindString(cmd) != "" {
		g.HandleSet(cmd)
	} else if mvCmdRegex.FindString(cmd) != "" {
		g.HandleMove(cmd)
	} else {
		g.HandleInvalid()
	}
}

// DetermineResult calculates the game's result from the current game state.
func (g *GG) DetermineResult() {
	g.logger.Println("determining result.")

	// Find both flags, and update the game status if one of them are not found.
	if g.status == gameInProgress {
		whiteFlagFound := false
		blackFlagFound := false

		for _, row := range g.board {
			for _, square := range row {
				if square.piece.code == flag {
					if square.piece.player == playerWhite {
						whiteFlagFound = true
					} else if square.piece.player == playerBlack {
						blackFlagFound = true
					}
				}
			}
		}

		if whiteFlagFound && !blackFlagFound {
			g.winner = playerWhite
			g.status = gameOver
		} else if blackFlagFound && !whiteFlagFound {
			g.winner = playerBlack
			g.status = gameOver
		}
	}

	// Check the 8th rank for the white flag.
	for _, square := range g.board[7] {
		if square.piece.player == playerWhite && square.piece.code == flag {
			g.status = gameOver
			g.winner = playerWhite
		}
	}

	// Check the 1st rank for the black flag.
	for _, square := range g.board[0] {
		if square.piece.player == playerBlack && square.piece.code == flag {
			g.status = gameOver
			g.winner = playerBlack
		}
	}
}

// ShowResult reports the "result" (i.e. what next step is needed) of the current game state.
func (g *GG) ShowResult() {
	g.logger.Println("showing result.")

	g.out.Write(">>>>> ")
	if g.status == gameSetup {
		g.out.Write("Please setup the board.\n")
	} else if g.status == gameInProgress {
		g.out.Write(fmt.Sprintf("%s to move.\n", g.playerToMove))
	} else if g.status == gameOver && g.winner != "" {
		g.out.Write(fmt.Sprintf("%s wins!\n", g.winner))
	}
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
	g.status = gameOver
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

	g.status = gameInProgress
	g.out.Write(fmt.Sprintf("File %s successfully loaded\n", f.Name()))
}

// HandleMove moves a piece into the target square.
func (g *GG) HandleMove(cmd string) {
	tokens := strings.Split(cmd, " ")
	// TODO : Validate these inputs.
	from := tokens[1]
	to := tokens[2]

	fromX, fromY := coordinatesToSquareAddress(from)
	toX, toY := coordinatesToSquareAddress(to)

	if !isOneSquareAway(fromX, fromY, toX, toY) {
		g.out.Write("Invalid move: can only move one square at a time.\n")
		return
	}

	// Create reference variables for convenience.
	fromSquare := &g.board[fromX][fromY]
	toSquare := &g.board[toX][toY]

	if fromSquare.piece.player != g.playerToMove {
		g.out.Write(fmt.Sprintf("Invalid move: it is %s's turn to move.\n", g.playerToMove))
		return
	}

	moveType := fromSquare.To(*toSquare)

	g.logger.Printf("Handling move type %v\n", moveType)
	switch moveType {
	case moveMove:
		// Move to the target square and clear out the origin square.
		toSquare.piece = fromSquare.piece
		fromSquare.Clear()
	case moveChallenge:
		result := resolveChallenge(fromSquare.piece, toSquare.piece)
		g.logger.Printf("%v vs %v: %v\n", fromSquare.piece.code, toSquare.piece.code, result)

		switch result {
		case resChallengerWins:
			toSquare.piece = fromSquare.piece
			fromSquare.Clear()
		case resChallengerLoses:
			fromSquare.Clear()
		case resDraw:
			fromSquare.Clear()
			toSquare.Clear()
		}
	case moveInvalid:
		g.out.Write("Invalid move.\n")
	}

	// Switch sides after every valid move.
	if moveType != moveInvalid {
		if g.playerToMove == playerWhite {
			g.playerToMove = playerBlack
		} else {
			g.playerToMove = playerWhite
		}
	}
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

// resolveChallenge determines the result of a piece challenge.
func resolveChallenge(challenger GGPiece, target GGPiece) GGChallengeResult {
	// Flag can only win vs flag.
	if challenger.code == flag {
		if target.code == flag {
			return resChallengerWins
		}
		return resChallengerLoses
	}

	// Check for draw, AFTER checking for Flag vs Flag.
	if challenger.code == target.code {
		return resDraw
	}

	// Spy only loses to privates.
	if challenger.code == spy {
		if target.code != private {
			return resChallengerWins
		}
		return resChallengerLoses
	}

	// Privates only wins vs spies.
	if challenger.code == private {
		if target.code == spy {
			return resChallengerWins
		}
		return resChallengerLoses
	}

	// All special pieces and draws are handled,
	// The challenger at this point is a basic piece.

	//	Basic pieces loses to Spies.
	if target.code == spy {
		return resChallengerLoses
	}

	if challenger.Power() > target.Power() {
		return resChallengerWins
	}
	return resChallengerLoses
}

// isOneSquareAway checks if the two given coordinates are one square apart.
func isOneSquareAway(fromX, fromY, toX, toY int) bool {
	diffX := fromX - toX
	diffY := fromY - toY

	return diffX >= -1 && diffX <= 1 && diffY >= -1 && diffY <= 1
}
