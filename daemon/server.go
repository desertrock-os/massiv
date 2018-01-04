/*
 * The MIT License (MIT)
 *
 * Copyright (c) 2018 Roland Singer [roland.singer@deserbit.com]
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package daemon

import (
	"net"

	"github.com/desertbit/closer"
	"github.com/rs/zerolog/log"
)

type Server struct {
	*closer.Closer

	uid int
	ln  net.Listener
}

func newServer(socketPath string, uid int) (s *Server, err error) {
	// Create the unix socket.
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return
	}

	s = &Server{
		uid: uid,
		ln:  ln,
	}
	s.Closer = closer.New(s.onClose)

	// Start the routine.
	go s.acceptRoutine()

	return
}

func (s *Server) UID() int {
	return s.uid
}

func (s *Server) onClose() error {
	return s.ln.Close()
}

func (s *Server) acceptRoutine() {
Loop:
	for {
		if s.IsClosed() {
			return
		}

		conn, err := s.ln.Accept()
		if err != nil {
			if s.IsClosed() {
				return
			}
			log.Error().Err(err).Msg("daemon: server accept error")
			continue Loop
		}

		go func() {
			gerr := s.handleNewConn(conn)
			if gerr != nil {
				log.Error().Err(gerr).Msg("daemon: server accept error")
			}
		}()
	}
}

func (s *Server) handleNewConn(conn net.Conn) (err error) {

	return
}
