package gostuff

import "sync"

//stores chess game information
type ChessGame struct {
	Type         string
	ID           int
	WhitePlayer  string
	BlackPlayer  string
	WhiteRating  int16
	BlackRating  int16
	GameMoves    []Move //stores chess move for games
	Status       string //white to move, black to move, white won, black won, or draw
	Result       int8   //0 means black won, 1 means white won and 2 means draw. 2 is used intead of 0.5 as database type is int
	GameType     string //bullet, blitz, standard, correspondence
	TimeControl  int
	BlackMinutes int
	BlackSeconds int
	WhiteMinutes int
	WhiteSeconds int
	StartMinutes int    // used to keep track of start time for correspondence
	PendingDraw  bool   //used to keep track if a player has offered a draw
	Rated        string //Yes if the game is rated, No if the game is unrated
	Spectate     bool
	CountryWhite string
	CountryBlack string
}

//source and destination of piece moves
type GameMove struct {
	Type      string
	ID        int
	Source    string
	Target    string
	Fen       string // FEN string of the board with the move played
	Promotion string
}

// used to unmarshall game ID that is being observed by player(Name)
type SpectateGame struct {
	Type     string
	ID       int `json:"ID,string"`
	Name     string
	Spectate string
}

//only holds source and destination, as well as pawn promotion
type Move struct {
	S   string // Source move
	T   string // Destination move
	P   string // Promotion piece
	Fen string // FEN string of the board with the move played
}

type Nrating struct {
	Type        string
	WhiteRating float64
	BlackRating float64
}

type Fin struct { //used to hold the result when a player is mated
	Type   string
	ID     int
	Fen    string // FEN string of the board with the move played
	Status string
}

// contains an array of player names observing the table
type Observers struct {
	sync.RWMutex
	Names []string
}

//used to verify players games
type Table struct {
	ChessBoard [][]string

	whitePawns [8]bool //stores array of booleans indicating whether or not the pawns have made their first move yet
	blackPawns [8]bool
	whitePass  [8]bool //stores array to indicate whether or not the pawns can perform an enpassent
	blackPass  [8]bool

	whiteTurn bool //keeps track of whose move it is, true means its whites turn and false means its blacks turn

	wkMoved bool //if king moved or not
	bkMoved bool

	wkrMoved bool //white king rook
	wqrMoved bool
	bkrMoved bool
	bqrMoved bool

	whiteKingX int8 //stores location of king for easy access
	whiteKingY int8
	blackKingX int8
	blackKingY int8

	kingUpdate bool //used to figure out if king position needs to be updated
	rookUpdate bool //castling rights

	blackOldX int8 //used to store old coordinates for king when in check
	blackOldY int8
	whiteOldX int8
	whiteOldY int8

	undoWPass bool //if this is true then white just did an en passent and it is used in undoMove()
	undoBPass bool

	pawnMove    int //keeps track of what move was the last pawn move made, used for fifty move rule
	lastCapture int

	resetWhiteTime chan bool
	resetBlackTime chan bool
	gameOver       chan bool
	whiteToMove    bool

	moveCount int       //keeps track of how many moves are made (moveCount+1) /2 to get move number
	promotion string    //keeps track of the piece that is being promoted too
	observe   Observers // list of user names who are observing this table
}

//active and running games on the server
var All = struct {
	sync.RWMutex
	Games map[int]*ChessGame
}{Games: make(map[int]*ChessGame)}

//pending matches in the lobby waiting for someone to accept
var Pending = struct {
	sync.RWMutex
	Matches map[int]*SeekMatch
}{Matches: make(map[int]*SeekMatch)}

//used to verify each move on the board
var Verify = struct {
	sync.RWMutex
	AllTables map[int]*Table
}{AllTables: make(map[int]*Table)}

//used for quick access to identify two people who are private chatting and playing a game against each other
var PrivateChat = make(map[string]string)

//intitalize all pawns to false as they have not moved yet, and also initialize all en passent to false
func InitGame(gameID int, name string, fighter string) {

	//setting up back end move verification
	var table Table
	table.ChessBoard = [][]string{
		[]string{"bR", "bN", "bB", "bQ", "bK", "bB", "bN", "bR"},
		[]string{"bP", "bP", "bP", "bP", "bP", "bP", "bP", "bP"},
		[]string{"-", "-", "-", "-", "-", "-", "-", "-"},
		[]string{"-", "-", "-", "-", "-", "-", "-", "-"},
		[]string{"-", "-", "-", "-", "-", "-", "-", "-"},
		[]string{"-", "-", "-", "-", "-", "-", "-", "-"},
		[]string{"wP", "wP", "wP", "wP", "wP", "wP", "wP", "wP"},
		[]string{"wR", "wN", "wB", "wQ", "wK", "wB", "wN", "wR"},
	}
	Verify.AllTables[gameID] = &table

	for i := 0; i < 8; i++ {
		Verify.AllTables[gameID].whitePass[i] = false
		Verify.AllTables[gameID].blackPass[i] = false
	}
	//castling rights init for kings
	Verify.AllTables[gameID].wkMoved = false
	Verify.AllTables[gameID].bkMoved = false
	//castling rights int for rooks
	Verify.AllTables[gameID].wkrMoved = false
	Verify.AllTables[gameID].wqrMoved = false
	Verify.AllTables[gameID].bkrMoved = false
	Verify.AllTables[gameID].bqrMoved = false
	//storing coordinates for starting location of both kings, X is row and Y is col
	Verify.AllTables[gameID].whiteKingX = 7
	Verify.AllTables[gameID].whiteKingY = 4
	Verify.AllTables[gameID].blackKingX = 0
	Verify.AllTables[gameID].blackKingY = 4

	Verify.AllTables[gameID].kingUpdate = false
	Verify.AllTables[gameID].rookUpdate = false

	Verify.AllTables[gameID].whiteTurn = true
	Verify.AllTables[gameID].whiteOldX = 7
	Verify.AllTables[gameID].whiteOldY = 4
	Verify.AllTables[gameID].blackOldX = 0
	Verify.AllTables[gameID].blackOldY = 4

	Verify.AllTables[gameID].undoWPass = false
	Verify.AllTables[gameID].undoBPass = false

	//reset times are used for correspondence
	Verify.AllTables[gameID].resetWhiteTime = make(chan bool)
	Verify.AllTables[gameID].resetBlackTime = make(chan bool)
	Verify.AllTables[gameID].gameOver = make(chan bool)

	Verify.AllTables[gameID].pawnMove = 0
	Verify.AllTables[gameID].lastCapture = 0
	Verify.AllTables[gameID].moveCount = 0

	Verify.AllTables[gameID].promotion = "q" //default to queen promotion

	// the players playing the game are also observers
	Verify.AllTables[gameID].observe.Names = append(Verify.AllTables[gameID].observe.Names, name)
	Verify.AllTables[gameID].observe.Names = append(Verify.AllTables[gameID].observe.Names, fighter)
}
