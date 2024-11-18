package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/pablon/bot/src/coms"
	"github.com/spf13/viper"
	"google.golang.org/grpc/status"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	timeoutToResponse = 1 * time.Second
)

var countT = 1

type BotGameTurn struct {
	turn   *coms.NewTurn
	action *coms.NewAction
}

type BotGame struct {
	initialState *coms.NewPlayerInitialState
	turnStates   []BotGameTurn
}

func (bg *BotGame) NewTurnAction(turn *coms.NewTurn) *coms.NewAction {
	action := &coms.NewAction{
		Action: coms.Action_MOVE,
		Destination: &coms.Position{
			X: turn.Position.X + 1,
			Y: turn.Position.Y,
		},
	}

	if countT == 2 {
		action = &coms.NewAction{
			Action: coms.Action_MOVE,
			Destination: &coms.Position{
				X: turn.Position.X,
				Y: turn.Position.Y + 1,
			},
		}
	}

	if countT == 3 || countT == 6 || countT == 10 || countT == 14 {
		action = &coms.NewAction{
			Action: coms.Action_ATTACK,
			Energy: turn.Energy,
			Destination: &coms.Position{
				X: turn.Position.X,
				Y: turn.Position.Y,
			},
		}
	}

	if countT == 4 || countT == 5 {
		action = &coms.NewAction{
			Action: coms.Action_MOVE,
			Destination: &coms.Position{
				X: turn.Position.X - 1,
				Y: turn.Position.Y,
			},
		}
	}

	if countT == 7 {
		action = &coms.NewAction{
			Action: coms.Action_CONNECT,
			Destination: &coms.Position{
				X: 3,
				Y: 3,
			},
		}
	}

	if countT == 8 || countT == 9 {
		action = &coms.NewAction{
			Action: coms.Action_MOVE,
			Destination: &coms.Position{
				X: turn.Position.X,
				Y: turn.Position.Y - 1,
			},
		}
	}

	if countT == 11 {
		action = &coms.NewAction{
			Action: coms.Action_CONNECT,
			Destination: &coms.Position{
				X: 1,
				Y: 3,
			},
		}
	}

	if countT == 12 || countT == 13 {
		action = &coms.NewAction{
			Action: coms.Action_MOVE,
			Destination: &coms.Position{
				X: turn.Position.X + 1,
				Y: turn.Position.Y + 1,
			},
		}
	}

	if countT == 15 {
		action = &coms.NewAction{
			Action: coms.Action_CONNECT,
			Destination: &coms.Position{
				X: 1,
				Y: 1,
			},
		}
	}

	bgt := BotGameTurn{
		turn:   turn,
		action: action,
	}
	bg.turnStates = append(bg.turnStates, bgt)

	countT += 1
	return action
}

type BotComs struct {
	botID                        int
	botName                      string
	myAddress, gameServerAddress string
}

func (ps *BotComs) waitToJoinGame() {
	grpcOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	grpcClient, err := grpc.NewClient(ps.gameServerAddress, grpcOpt)
	if err != nil {
		fmt.Printf("grpc client ERROR: %v\n", err)
		panic("could not create a grpc client")
	}

	npjc := coms.NewGameServiceClient(grpcClient)

	player := &coms.NewPlayer{
		Name:          ps.botName,
		ServerAddress: ps.myAddress,
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutToResponse)
		playerID, err := npjc.Join(ctx, player)

		if err != nil {
			fmt.Printf("could not join game ERROR: %v\n", err)
			cancel()
			continue
		} else {
			fmt.Printf("Joined game with ID %d\n", int(playerID.PlayerID))
			ps.botID = int(playerID.PlayerID)

			if viper.GetBool("bot.verbosity") {
				b, err := json.Marshal(playerID)
				if err != nil {
					fmt.Println(err)
					return
				}
				fmt.Println(string(b))
			}
			break
		}
	}
}

func (ps *BotComs) startListening() {
	fmt.Println("Starting to listen on", ps.myAddress)

	lis, err := net.Listen("tcp", ps.myAddress)
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryLoggingInterceptor),
		grpc.StreamInterceptor(StreamLoggingInterceptor),
	)
	cs := &ClientServer{}
	coms.RegisterGameServiceServer(grpcServer, cs)

	if err := grpcServer.Serve(lis); err != nil {
		panic(err)
	}
}

func UnaryLoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	start := time.Now()
	resp, err = handler(ctx, req)
	duration := time.Since(start)
	st, _ := status.FromError(err)
	fmt.Printf("unary call: %s, Duration: %v, Error: %v\n", info.FullMethod, duration, st.Message())
	return resp, err
}

func StreamLoggingInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	err := handler(srv, ss)
	duration := time.Since(start)
	st, _ := status.FromError(err)
	fmt.Printf("stream call: %s, Duration: %v, Error: %v\n", info.FullMethod, duration, st.Message())
	return err
}

type ClientServer struct {
	bg *BotGame
}

func (gs *ClientServer) Join(_ context.Context, _ *coms.NewPlayer) (*coms.PlayerID, error) {
	return nil, fmt.Errorf("random bot does not implement Join service")
}

func (gs *ClientServer) InitialState(_ context.Context, initialState *coms.NewPlayerInitialState) (*coms.PlayerReady, error) {
	fmt.Println("random bot receiving InitialState")

	gs.bg = &BotGame{}
	gs.bg.initialState = initialState

	if viper.GetBool("bot.verbosity") {
		b, err := json.Marshal(initialState)
		if err != nil {
			return nil, err
		}
		fmt.Println(string(b))
	}

	resp := coms.PlayerReady{Ready: true}
	return &resp, nil
}

func (gs *ClientServer) Turn(_ context.Context, turn *coms.NewTurn) (*coms.NewAction, error) {
	if viper.GetBool("bot.verbosity") {
		b, err := json.Marshal(turn)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		fmt.Println(string(b))
	}

	action := gs.bg.NewTurnAction(turn)

	return action, nil
}

func ensureParams() (botName *string, listenAddress *string, gameServerAddress *string) {
	botName = flag.String("bn", "random-bot", "bot name")
	listenAddress = flag.String("la", "", "my listen address")
	gameServerAddress = flag.String("gs", "", "game server address")
	flag.Parse()

	if *botName == "" {
		panic("bot name is required")
	}
	if *listenAddress == "" {
		panic("listen address is required")
	}
	if *gameServerAddress == "" {
		panic("game server address is required")
	}
	return botName, listenAddress, gameServerAddress
}

func main() {
	// init configuration
	viper.SetDefault("bot.verbosity", true)

	botName, listenAddress, gameServerAddress := ensureParams()

	bot := &BotComs{
		botName:           *botName,
		myAddress:         *listenAddress,
		gameServerAddress: *gameServerAddress,
	}

	bot.waitToJoinGame()

	// TODO: may be it needs to be 1 more step to retrieve initial state and process it
	// bot.getInitialState()

	bot.startListening()
}
