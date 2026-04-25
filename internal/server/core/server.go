package core

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/commands"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

type Server struct {
	Host    string
	Port    int
	MaxSize uint32

	Store      *state.Store
	Controller *commands.Controller
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("listen %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	c := model.NewConnection(conn)
	log.Printf("client connect id=%d ip=%s port=%d", c.ID, c.IP, c.Port)
	defer func() {
		s.Controller.OnDisconnect(c)
		log.Printf("client disconnect id=%d ip=%s port=%d", c.ID, c.IP, c.Port)
	}()
	for {
		req, err := gsc.ReadFrom(conn, s.MaxSize)
		if err != nil {
			log.Printf("read error id=%d ip=%s: %v", c.ID, c.IP, err)
			return
		}
		if len(req.CmdSet.Commands) == 0 {
			log.Printf("empty command set id=%d ip=%s", c.ID, c.IP)
			continue
		}
		cmd := req.CmdSet.Commands[0]
		if len(req.CmdSet.Commands) > 1 {
			log.Printf("warning: more than one command in request, ignoring rest")
		}
		log.Printf("recv id=%d cmd=%s args=%d", c.ID, cmd.Name, len(cmd.Args))
		var coreArgs []string
		win := ""
		key := ""
		if len(cmd.Args) >= 2 {
			coreArgs = append([]string(nil), cmd.Args[:len(cmd.Args)-2]...)
			win = cmd.Args[len(cmd.Args)-2]
			key = cmd.Args[len(cmd.Args)-1]
		} else {
			// Perl warns on short args but still dispatches command.
			log.Printf("warning: args count < 2 for %s; dispatching with empty win/key", cmd.Name)
			coreArgs = append([]string(nil), cmd.Args...)
		}
		result := s.Controller.HandleWithMeta(ctx, c, req, cmd.Name, coreArgs, win, key)
		if !result.HasResponse {
			log.Printf("no response id=%d cmd=%s", c.ID, cmd.Name)
			continue
		}
		response := result.Commands
		for i := range response {
			response[i].Args = append(response[i].Args, win)
			if response[i].Name == "LW_show" && len(response[i].Args) > 0 {
				body := response[i].Args[0]
				preview := body
				if len(preview) > 140 {
					preview = preview[:140]
				}
				preview = strings.ReplaceAll(preview, "\n", "\\n")
				log.Printf("show payload id=%d len=%d preview=%q", c.ID, len(body), preview)
				if strings.Contains(body, "%CG_HOLEHOST&") || strings.Contains(body, "%CG_HOLEPORT&") {
					log.Printf(
						"send to client stun vars id=%d CG_HOLEHOST=%q CG_HOLEPORT=%q CG_HOLEINT=%q",
						c.ID,
						extractGVar(body, "CG_HOLEHOST"),
						extractGVar(body, "CG_HOLEPORT"),
						extractGVar(body, "CG_HOLEINT"),
					)
				}
			}
		}
		out := gsc.Stream{
			Num:  req.Num,
			Lang: req.Lang,
			Ver:  req.Ver,
			CmdSet: gsc.CommandSet{
				Commands: response,
			},
		}
		bin, err := out.MarshalBinary()
		if err != nil {
			log.Printf("marshal error id=%d cmd=%s: %v", c.ID, cmd.Name, err)
			return
		}
		if _, err := conn.Write(bin); err != nil {
			log.Printf("write error id=%d cmd=%s: %v", c.ID, cmd.Name, err)
			return
		}
		log.Printf("sent id=%d cmd=%s responses=%d", c.ID, cmd.Name, len(response))
		if c.Closed {
			return
		}
	}
}

func extractGVar(body string, key string) string {
	token := "%" + key + "&"
	start := strings.Index(body, token)
	if start < 0 {
		return ""
	}
	rest := body[start+len(token):]
	end := strings.Index(rest, "&%")
	if end < 0 {
		if zero := strings.IndexByte(rest, 0); zero >= 0 {
			end = zero
		} else {
			end = len(rest)
		}
	}
	return rest[:end]
}
