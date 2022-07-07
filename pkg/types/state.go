package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PanGan21/miniraft/pkg/storage"
)

type State struct {
	Name         string
	CurrentTerm  int
	VotedFor     string
	CommitLength int
}

func newState(name string) *State {
	return &State{
		Name:         name,
		CurrentTerm:  0,
		VotedFor:     "",
		CommitLength: 0,
	}
}

func (s *State) LogPersistentState() {
	persistentLog := s.Name + "," + strconv.Itoa(s.CurrentTerm) + "," + s.VotedFor + "," + strconv.Itoa(s.CommitLength)
	err := storage.PersistState(persistentLog)
	if err != nil {
		fmt.Println(err)
	}
}

func GetExistingStateOrCreateNew(name string) *State {
	log, err := storage.GetCurrentStateByServer(name)
	if err != nil {
		return newState(name)
	}
	return parseStateLog(log)
}

func parseStateLog(log string) *State {
	splits := strings.Split(log, ",")
	name := splits[0]
	currentTerm, _ := strconv.Atoi(splits[1])
	votedFor := splits[2]
	commitLength, _ := strconv.Atoi(splits[3])
	return &State{
		Name:         name,
		CurrentTerm:  currentTerm,
		VotedFor:     votedFor,
		CommitLength: commitLength,
	}
}
