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

package rootdaemon

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/desertbit/massiv/global"
	"github.com/desertbit/massiv/utils"

	"github.com/desertbit/closer"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

type RootDaemon struct {
	*closer.Closer
	watcher *fsnotify.Watcher
}

func New() (d *RootDaemon, err error) {
	// RootDaemon.
	d = &RootDaemon{}
	d.Closer = closer.New(d.onClose)

	// Close on error.
	defer func() {
		if err != nil {
			d.Close()
		}
	}()

	// Watcher
	d.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}

	// Unprivileged gui.
	gid, err := global.GetUserGID()
	if err != nil {
		return
	}

	// Prepare socket directory.
	e, err := utils.Exists(global.SocketDir)
	if err != nil {
		return
	} else if e {
		err = os.RemoveAll(global.SocketDir)
		if err != nil {
			return
		}
	}
	err = os.MkdirAll(global.SocketDir, 0775)
	if err != nil {
		return
	}
	err = os.Chown(global.SocketDir, 0, gid)
	if err != nil {
		return
	}
	err = os.Chmod(global.SocketDir, 0775)
	if err != nil {
		return
	}

	// Watch socket directory.
	err = d.watcher.Add(global.SocketDir)
	if err != nil {
		return
	}

	// Start routines.
	go d.watchRoutine()

	log.Info().Msg("massiv root daemon running")

	return
}

func (d *RootDaemon) onClose() error {
	if d.watcher != nil {
		return d.watcher.Close()
	}
	return nil
}

func (d *RootDaemon) watchRoutine() {
	defer d.Close()

Loop:
	for {
		select {
		case <-d.CloseChan:
			return

		case event := <-d.watcher.Events:
			if event.Op&fsnotify.Create != fsnotify.Create {
				continue Loop
			}

			err := d.setSocketFile(event.Name)
			if err != nil {
				log.Error().Err(err).Msg("root daemon: failed to set socket file")
				return // Exit, because this is a fatal error.
			}

		case err := <-d.watcher.Errors:
			log.Error().Err(err).Msg("root daemon watch error")
		}
	}
}

func (d *RootDaemon) setSocketFile(path string) (err error) {
	// Skip if not exists.
	e, err := utils.Exists(path)
	if err != nil {
		return
	} else if !e {
		return
	}

	log.Debug().Msg("set socket file: " + path)

	// Check if the file is a unix socket.
	stat, err := os.Lstat(path)
	if err != nil {
		return
	}
	mode := stat.Mode()
	if mode&os.ModeSocket != os.ModeSocket {
		return fmt.Errorf("is not a unix socket: %v", path)
	}

	// TODO: check if pid is matching.

	// Obtain the UID from the filename.
	name := filepath.Base(path)
	uid, err := strconv.Atoi(name)
	if err != nil {
		return
	}

	// Obtain the GID.
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return
	}

	// Set permission.
	err = os.Chmod(path, 0600)
	if err != nil {
		return
	}
	err = os.Chown(path, uid, gid)
	if err != nil {
		return
	}

	return
}
