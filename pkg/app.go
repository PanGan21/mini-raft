package pkg

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/PanGan21/miniraft/pkg/storage"
	"github.com/PanGan21/miniraft/pkg/types"
)

func StartServer(serverName *string, port *string) {
	l, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	db := storage.NewDatabase()
	rand.Seed(time.Now().UnixNano())

	electionTimeoutInterval := rand.Intn(int(ElectionMaxTimeout)-int(ElectionMinTimeout)) + int(ElectionMinTimeout)
	electionModule := types.NewElectionModule(electionTimeoutInterval)

	err = storage.RegisterServer(*serverName, *port)
	if err != nil {
		fmt.Println(err)
		return
	}

	s := server{
		port:           *port,
		db:             db,
		logs:           db.RebuildLogIfExists(*serverName),
		state:          types.GetExistingStateOrCreateNew(*serverName),
		currentRole:    "follower",
		leaderNodeId:   "",
		peerData:       *types.NewPeerData(),
		electionModule: *electionModule,
	}
	s.state.LogPersistentState()
	go s.ElectionTimer()
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go s.HandleConnection(c)
	}
}
