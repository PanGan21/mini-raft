package pkg

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/PanGan21/miniraft/pkg/log"
	"github.com/PanGan21/miniraft/pkg/storage"
	"github.com/PanGan21/miniraft/pkg/types"
	"github.com/PanGan21/miniraft/pkg/vote"
)

const (
	BroadcastPeriod    = 3000
	ElectionMinTimeout = 3001
	ElectionMaxTimeout = 10000
)

type server struct {
	port           string
	db             *storage.Database
	state          *types.State
	logs           []string
	currentRole    string
	leaderNodeId   string
	peerData       types.PeerData
	electionModule types.ElectionModule
}

func (s *server) sendMessageToFollowerNode(message string, port int) {
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		s.peerData.SuspectedNodes[port] = true
		return
	}
	_, ok := s.peerData.SuspectedNodes[port]
	if ok {
		delete(s.peerData.SuspectedNodes, port)
	}
	fmt.Fprintf(conn, message+"\n")
	go s.HandleConnection(conn)
}

func (s *server) replicateLog(followerName string, followerPort int) {
	if followerName == s.state.Name {
		go s.commitLogEntries()
	}
	var prefixTerm = 0
	prefixLength := s.peerData.SentLength[followerName]
	if prefixLength > 0 {
		logSplit := strings.Split(s.logs[prefixLength-1], "#")
		prefixTerm, _ = strconv.Atoi(logSplit[1])
	}
	logRequest := log.NewLogRequest(s.state.Name, s.state.CurrentTerm, prefixLength, prefixTerm, s.state.CommitLength, s.logs[s.peerData.SentLength[followerName]:])
	s.sendMessageToFollowerNode(logRequest.String(), followerPort)
}

func (s *server) commitLogEntries() {
	allNodes, _ := storage.GetRegisteredServers()
	nodeCount := len(allNodes) - len(s.peerData.SuspectedNodes)
	for i := s.state.CommitLength; i < len(s.logs); i++ {
		var acks = 0
		for node := range allNodes {
			if s.peerData.AckedLength[node] > s.state.CommitLength {
				acks = acks + 1
			}
		}
		if acks >= (nodeCount+1)/2 || nodeCount == 1 {
			log := s.logs[i]
			command := strings.Split(log, "#")[0]
			s.db.PerformDbOperations(command)
			s.state.CommitLength = s.state.CommitLength + 1
			s.state.LogPersistentState()
		} else {
			break
		}
	}
}

func (s *server) handleVoteRequest(message string) string {
	voteRequest, _ := vote.ParseVoteRequest(message)
	if voteRequest.CandidateTerm > s.state.CurrentTerm {
		s.state.CurrentTerm = voteRequest.CandidateTerm
		s.currentRole = "follower"
		s.state.VotedFor = ""
		s.electionModule.ResetElectionTimer <- struct{}{}
	}
	var lastTerm = 0
	if len(s.logs) > 0 {
		lastTerm = parseLogTerm(s.logs[len(s.logs)-1])
	}
	var logOk = false
	if voteRequest.CandidateLogTerm > lastTerm ||
		(voteRequest.CandidateLogTerm == lastTerm && voteRequest.CandidateLogLength >= len(s.logs)) {
		logOk = true
	}
	if voteRequest.CandidateTerm == s.state.CurrentTerm && logOk && (s.state.VotedFor == "" || s.state.VotedFor == voteRequest.CandidateId) {
		s.state.VotedFor = voteRequest.CandidateId
		s.state.LogPersistentState()
		return vote.NewVoteResponse(
			s.state.Name,
			s.state.CurrentTerm,
			true,
		).String()
	} else {
		return vote.NewVoteResponse(s.state.Name, s.state.CurrentTerm, false).String()
	}
}

func (s *server) handleLogRequest(message string) string {
	s.electionModule.ResetElectionTimer <- struct{}{}
	logRequest, _ := log.ParseLogRequest(message)
	if logRequest.CurrentTerm > s.state.CurrentTerm {
		s.state.CurrentTerm = logRequest.CurrentTerm
		s.state.VotedFor = ""
	}
	if logRequest.CurrentTerm == s.state.CurrentTerm {
		if s.currentRole == "leader" {
			go s.ElectionTimer()
		}
		s.currentRole = "follower"
		s.leaderNodeId = logRequest.LeaderId
	}
	var logOk bool = false
	if len(s.logs) >= logRequest.PrefixLength &&
		(logRequest.PrefixLength == 0 ||
			parseLogTerm(s.logs[logRequest.PrefixLength-1]) == logRequest.PrefixTerm) {
		logOk = true
	}
	port, _ := strconv.Atoi(s.port)
	if s.state.CurrentTerm == logRequest.CurrentTerm && logOk {
		s.appendEntries(logRequest.PrefixLength, logRequest.CommitLength, logRequest.Suffix)
		ack := logRequest.PrefixLength + len(logRequest.Suffix)
		return log.NewLogResponse(s.state.Name, port, s.state.CurrentTerm, ack, true).String()
	} else {
		return log.NewLogResponse(s.state.Name, port, s.state.CurrentTerm, 0, false).String()
	}
}

func (s *server) handleLogResponse(message string) string {
	lr, _ := log.ParseLogResponse(message)
	if lr.CurrentTerm > s.state.CurrentTerm {
		s.state.CurrentTerm = lr.CurrentTerm
		s.currentRole = "follower"
		s.state.VotedFor = ""
		go s.ElectionTimer()
	}
	if lr.CurrentTerm == s.state.CurrentTerm && s.currentRole == "leader" {
		if lr.ReplicationSuccessful && lr.AckLength >= s.peerData.AckedLength[lr.NodeId] {
			s.peerData.SentLength[lr.NodeId] = lr.AckLength
			s.peerData.AckedLength[lr.NodeId] = lr.AckLength
			s.commitLogEntries()
		} else {
			s.peerData.SentLength[lr.NodeId] = s.peerData.SentLength[lr.NodeId] - 1
			s.replicateLog(lr.NodeId, lr.Port)
		}
	}
	return "replication successful"
}

func (s *server) handleVoteResponse(message string) {
	voteResponse, _ := vote.ParseVoteResponse(message)
	if voteResponse.CurrentTerm > s.state.CurrentTerm {
		if s.currentRole != "leader" {
			s.electionModule.ResetElectionTimer <- struct{}{}
		}
		s.state.CurrentTerm = voteResponse.CurrentTerm
		s.currentRole = "follower"
		s.state.VotedFor = ""
	}
	if s.currentRole == "candidate" && voteResponse.CurrentTerm == s.state.CurrentTerm && voteResponse.VoteInFavor {
		s.peerData.VotesReceived[voteResponse.NodeId] = true
		s.checkForElectionResult()
	}
}

func (s *server) HandleConnection(conn net.Conn) {
	defer conn.Close()
	for {
		data, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			continue
		}
		message := strings.TrimSpace(string(data))
		if message == "invalid command" || message == "replication successful" {
			continue
		}
		fmt.Println(">", string(message))
		var response string = ""
		if strings.HasPrefix(message, "LogRequest") {
			response = s.handleLogRequest(message)
		}
		if strings.HasPrefix(message, "LogResponse") {
			response = s.handleLogResponse(message)
		}
		if strings.HasPrefix(message, "VoteRequest") {
			response = s.handleVoteRequest(message)
		}
		if strings.HasPrefix(message, "VoteResponse") {
			s.handleVoteResponse(message)
		}
		if s.currentRole == "leader" && response == "" {
			var err = s.db.ValidateCommand(message)
			if err != nil {
				response = err.Error()
			}
			if strings.HasPrefix(message, "GET") {
				response = s.db.PerformDbOperations(message)
			}
			if response == "" {
				logMessage := message + "#" + strconv.Itoa(s.state.CurrentTerm)
				s.peerData.AckedLength[s.state.Name] = len(s.logs)
				s.logs = append(s.logs, logMessage)
				currLogIdx := len(s.logs) - 1
				err = s.db.LogCommand(logMessage, s.state.Name)
				if err != nil {
					response = "error while logging command"
				}
				allServers, _ := storage.GetRegisteredServers()
				for sname, sport := range allServers {
					s.replicateLog(sname, sport)
				}
				for s.state.CommitLength <= currLogIdx {
					fmt.Println("Waiting for consensus: ")
				}
				response = "operation sucessful"
			}
		}
		if response != "" {
			conn.Write([]byte(response + "\n"))
		}

	}
}

func (s *server) appendEntries(prefixLength int, commitLength int, suffix []string) {
	if len(suffix) > 0 && len(s.logs) > prefixLength {
		var index int
		if len(s.logs) > (prefixLength + len(suffix)) {
			index = prefixLength + len(suffix) - 1
		} else {
			index = len(s.logs) - 1
		}
		if parseLogTerm(s.logs[index]) != parseLogTerm(suffix[index-prefixLength]) {
			s.logs = s.logs[:prefixLength]
		}
	}
	if prefixLength+len(suffix) > len(s.logs) {
		for i := (len(s.logs) - prefixLength); i < len(suffix); i++ {
			s.addLogs(suffix[i])
			err := s.db.LogCommand(suffix[i], s.state.Name)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	if commitLength > s.state.CommitLength {
		for i := s.state.CommitLength; i < commitLength; i++ {
			s.db.PerformDbOperations(strings.Split(s.logs[i], "#")[0])
		}
		s.state.CommitLength = commitLength
		s.state.LogPersistentState()
	}
}

func (s *server) addLogs(log string) []string {
	s.logs = append(s.logs, log)
	return s.logs
}

func (s *server) ElectionTimer() {
	for {
		select {
		case <-s.electionModule.ElectionTimeout.C:
			fmt.Println("Timed out")
			if s.currentRole == "follower" {
				go s.startElection()
			} else {
				s.currentRole = "follower"
				s.electionModule.ResetElectionTimer <- struct{}{}
			}
		case <-s.electionModule.ResetElectionTimer:
			fmt.Println("Resetting election timer")
			s.electionModule.ElectionTimeout.Reset(time.Duration(s.electionModule.ElectionTimeoutInterval) * time.Millisecond)
		}
	}
}

func (s *server) startElection() {
	s.state.CurrentTerm = s.state.CurrentTerm + 1
	s.currentRole = "candidate"
	s.state.VotedFor = s.state.Name
	s.peerData.VotesReceived = map[string]bool{}
	s.peerData.VotesReceived[s.state.Name] = true
	var lastTerm = 0
	if len(s.logs) > 0 {
		lastTerm = parseLogTerm(s.logs[len(s.logs)-1])
	}
	voteRequest := vote.NewVoteRequest(s.state.Name, s.state.CurrentTerm, len(s.logs), lastTerm)
	allNodes, _ := storage.GetRegisteredServers()
	for node, port := range allNodes {
		if node != s.state.Name {
			s.sendMessageToFollowerNode(voteRequest.String(), port)
		}
	}
	s.checkForElectionResult()
}

func (s *server) checkForElectionResult() {
	if s.currentRole == "leader" {
		return
	}
	var totalVotes = 0
	for server := range s.peerData.VotesReceived {
		if s.peerData.VotesReceived[server] {
			totalVotes += 1
		}
	}
	allNodes, _ := storage.GetRegisteredServers()
	if totalVotes >= (len(allNodes)+1)/2 {
		fmt.Println("I won the election. New leader: ", s.state.Name, " Votes received: ", totalVotes)
		s.currentRole = "leader"
		s.leaderNodeId = s.state.Name
		s.peerData.VotesReceived = make(map[string]bool)
		s.electionModule.ElectionTimeout.Stop()
		s.syncUp()
	}
}

func (s *server) syncUp() {
	ticker := time.NewTicker(BroadcastPeriod * time.Millisecond)
	for t := range ticker.C {
		fmt.Println("sending heartbeat at: ", t)
		allServers, _ := storage.GetRegisteredServers()
		for sname, sport := range allServers {
			if sname != s.state.Name {
				s.replicateLog(sname, sport)
			}
		}
	}
}

func parseLogTerm(message string) int {
	split := strings.Split(message, "#")
	pTerm, _ := strconv.Atoi(split[1])
	return pTerm
}
