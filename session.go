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

package massiv

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/desertbit/closer"
	"github.com/desertbit/massiv/global"
)

type Session struct {
	*closer.Closer

	uid  int
	conn net.Conn
}

func New() (s *Session, err error) {
	uid := os.Geteuid()
	socketPath := filepath.Join(global.SocketDir, strconv.Itoa(uid))

	// Obtain the GID.
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return
	}

	// Ensure the file permission and owner are valid.
	stat, err := os.Lstat(socketPath)
	if err != nil {
		return
	}
	mode := stat.Mode()
	if mode != 0600 {
		err = fmt.Errorf("unix socket has invalid permission: %v", mode)
		return
	} else if mode&os.ModeSocket != os.ModeSocket {
		err = fmt.Errorf("is not a unix socket: %v", socketPath)
		return
	}
	sysStat := stat.Sys().(*syscall.Stat_t)
	if sysStat.Uid != uint32(uid) {
		err = fmt.Errorf("unix socket has invalid uid: %v != %v", sysStat.Uid, uid)
		return
	} else if sysStat.Gid != uint32(gid) {
		err = fmt.Errorf("unix socket has invalid gid: %v != %v", sysStat.Gid, gid)
		return
	}

	// Connect to the unix socket.
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return
	}

	s = &Session{
		uid:  uid,
		conn: conn,
	}
	s.Closer = closer.New(s.onClose)

	// Close on error.
	defer func() {
		if err != nil {
			s.Close()
		}
	}()

	// TODO

	return
}

func (s *Session) UID() int {
	return s.uid
}

func (s *Session) onClose() error {
	return s.conn.Close()
}
