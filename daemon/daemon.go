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
	"fmt"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/desertbit/massiv/global"

	"github.com/desertbit/closer"
	"github.com/rs/zerolog/log"
)

type Daemon struct {
	*closer.Closer

	serversMutex sync.Mutex
	servers      map[int]*Server
}

func New() (d *Daemon, err error) {
	d = &Daemon{
		servers: make(map[int]*Server),
	}
	d.Closer = closer.New(d.onClose)

	// Add the socket server for root.
	err = d.addServer(0)
	if err != nil {
		return
	}

	log.Info().Msg("massiv socket daemon running")

	return
}

func (d *Daemon) onClose() error {
	d.serversMutex.Lock()
	defer d.serversMutex.Unlock()

	for _, s := range d.servers {
		s.Close()
	}
	return nil
}

func (d *Daemon) addServer(uid int) (err error) {
	socketPath := filepath.Join(global.SocketDir, strconv.Itoa(uid))

	d.serversMutex.Lock()
	defer d.serversMutex.Unlock()

	// Check if already present.
	if _, ok := d.servers[uid]; ok {
		return fmt.Errorf("socket server already exists for uid: %v", uid)
	}

	// Create a new server for the uid.
	s, err := newServer(socketPath, uid)
	if err != nil {
		return fmt.Errorf("failed to add socket server for uid '%v': %v", uid, err)
	}
	d.servers[uid] = s

	// Remove from the map on close.
	go func() {
		select {
		case <-d.CloseChan:
			return
		case <-s.CloseChan:
			d.serversMutex.Lock()
			delete(d.servers, uid)
			d.serversMutex.Unlock()
		}
	}()

	return
}
