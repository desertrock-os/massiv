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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/desertbit/massiv/daemon"
	"github.com/desertbit/massiv/global"
	"github.com/desertbit/massiv/rootdaemon"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	verbose      bool
	socketDaemon bool
)

func init() {
	flag.BoolVar(&verbose, "v", false, "enable verbose logging")
	flag.BoolVar(&socketDaemon, "s", false, "run the unprivileged socket daemon")
}

func main() {
	flag.Parse()

	// Prepare logger.
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if socketDaemon {
		err := runSocketDaemon()
		if err != nil {
			log.Fatal().Err(err).Msg("socket daemon failed")
		}
	} else {
		err := runRootDaemon()
		if err != nil {
			log.Fatal().Err(err).Msg("root daemon failed")
		}
	}
}

func runRootDaemon() (err error) {
	// Must be root.
	if os.Geteuid() != 0 {
		return fmt.Errorf("must be root")
	}

	d, err := rootdaemon.New()
	if err != nil {
		return
	}

	// Catch interrupts.
	go func() {
		// Wait for the signal.
		sigchan := make(chan os.Signal, 3)
		signal.Notify(sigchan, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGKILL)
		<-sigchan

		log.Info().Msg("exiting root daemon")
		d.Close()
	}()

	// Start the unprivileged socket daemon.
	go func() {
		defer d.Close()
		gerr := execSocketDaemon()
		if gerr != nil {
			log.Error().Err(gerr).Msg("socket daemon failed")
		}
	}()

	// Wait for the daemon to close.
	<-d.CloseChan

	return
}

func execSocketDaemon() (err error) {
	// Obtain the path to the current binary.
	binPath, err := os.Executable()
	if err != nil {
		return
	}

	// Obtain be the unprivileged user uid & gid.
	uid, err := global.GetUserUID()
	if err != nil {
		return
	}
	gid, err := global.GetUserGID()
	if err != nil {
		return
	}

	args := []string{"-s"}
	if verbose {
		args = append(args, "-v")
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: uint32(uid),
		Gid: uint32(gid),
	}

	return cmd.Run()
}

func runSocketDaemon() (err error) {
	// Must be the unprivileged user.
	uid, err := global.GetUserUID()
	if err != nil {
		return
	} else if os.Geteuid() != uid {
		return fmt.Errorf("must be user: %v", global.User)
	}

	d, err := daemon.New()
	if err != nil {
		return
	}

	// Catch interrupts.
	go func() {
		// Wait for the signal.
		sigchan := make(chan os.Signal, 3)
		signal.Notify(sigchan, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGKILL)
		<-sigchan

		log.Info().Msg("exiting socket daemon")
		d.Close()
	}()

	// Wait for the daemon to close.
	<-d.CloseChan

	return
}
