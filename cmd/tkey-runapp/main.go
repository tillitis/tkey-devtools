// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/tillitis/tkeyclient"
	"github.com/tillitis/tkeyutil"
)

// Use when printing err/diag msgs
var le = log.New(os.Stderr, "", 0)

var version string

const usage = `Usage: %[1]s [flags...] FILE

%[1]s loads an application binary from FILE onto Tillitis TKey
and starts it.

Exit status code is 0 if the app is both successfully loaded and started. Exit
code is non-zero if anything goes wrong, for example if TKey is already
running some app.`

func main() {
	var fileName, devPath, fileUSS string
	var speed int
	var enterUSS, verbose, helpOnly bool

	pflag.CommandLine.SetOutput(os.Stderr)
	pflag.CommandLine.SortFlags = false

	pflag.StringVarP(&devPath, "port", "p", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")

	pflag.IntVarP(&speed, "speed", "s", tkeyclient.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")

	pflag.BoolVar(&enterUSS, "uss", false,
		"Enable typing of a phrase to be hashed as the User Supplied Secret. The USS is loaded onto the TKey along with the app itself and used by the firmware, together with other material, for deriving secrets for the application.")

	pflag.StringVar(&fileUSS, "uss-file", "",
		"Read `FILE` and hash its contents as the USS. Use '-' (dash) to read from stdin. The full contents are hashed unmodified (e.g. newlines are not stripped).")

	pflag.BoolVar(&verbose, "verbose", false, "Enable verbose output.")
	pflag.BoolVar(&helpOnly, "help", false, "Output this help.")
	versionOnly := pflag.BoolP("version", "v", false, "Output version information.")

	pflag.Usage = func() {
		desc := fmt.Sprintf(usage, os.Args[0])
		le.Printf("%s\n\n%s", desc,
			pflag.CommandLine.FlagUsagesWrapped(86))
	}
	pflag.Parse()

	// If the version is not explicitly defined, try to read it from the build info.
	if version == "" {
		version = readBuildInfo()
	}

	// If there are any arguments left, it's either an error or the file name.
	if pflag.NArg() > 0 {
		if pflag.NArg() > 1 { // Too many arguments
			le.Printf("Unexpected argument: %s\n\n", strings.Join(pflag.Args()[1:], " "))
			pflag.Usage()
			os.Exit(2)
		}
		fileName = pflag.Args()[0] // The file name
	}

	if *versionOnly {
		le.Printf("tkey-runapp %s", version)
		os.Exit(0)
	}

	if helpOnly {
		pflag.Usage()
		os.Exit(0)
	}

	if fileName == "" {
		le.Printf("Please pass an app binary FILE.\n\n")
		pflag.Usage()
		os.Exit(2)
	}

	if !verbose {
		tkeyclient.SilenceLogging()
	}

	if enterUSS && fileUSS != "" {
		le.Printf("Can't combine --uss and --uss-file\n\n")
		pflag.Usage()
		os.Exit(2)
	}

	// Read the app binary to be sent to the device.
	appBin, err := os.ReadFile(fileName)
	if err != nil {
		le.Printf("Failed to read file: %v\n", err)
		os.Exit(1)
	}
	if bytes.HasPrefix(appBin, []byte("\x7fELF")) {
		le.Printf("%s looks like an ELF executable, but a raw binary is expected.\n", fileName)
		os.Exit(1)
	}

	// Try to auto-detect the serial port if not explicitly set.
	if devPath == "" {
		devPath, err = tkeyclient.DetectSerialPort(true)
		if err != nil {
			os.Exit(1)
		}
	}

	// Initialize the client and connect to the device.
	tk := tkeyclient.New()
	le.Printf("Connecting to device on serial port %s ...\n", devPath)
	if err = tk.Connect(devPath, tkeyclient.WithSpeed(speed)); err != nil {
		le.Printf("Could not open %s: %v\n", devPath, err)
		os.Exit(1)
	}

	// Create a handler for signals to close the connection and exit with specific code.
	exit := func(code int) {
		if err = tk.Close(); err != nil {
			le.Printf("Close: %v\n", err)
		}
		os.Exit(code)
	}

	// Mount a signal handler to close the connection on SIGINT and SIGTERM.
	// This will always exit with code 1.
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	nameVer, err := tk.GetNameVersion()
	if err != nil {
		le.Printf("GetNameVersion failed: %v\n", err)
		le.Printf("If the serial port is correct, then the TKey might not be in firmware-\n" +
			"mode, and have an app running already. Please unplug and plug it in again.\n")
		exit(1)
	}

	le.Printf("Firmware name0:'%s' name1:'%s' version:%d\n",
		nameVer.Name0, nameVer.Name1, nameVer.Version)

	udi, err := tk.GetUDI()
	if err != nil {
		le.Printf("GetUDI failed: %v\n", err)
		exit(1)
	}

	fmt.Printf("UDI: %v\n", udi)

	var secret []byte

	// If the USS flag is set -> read the USS from the user.
	// If the USS file flag is set -> read the USS from the file.
	if enterUSS {
		secret, err = tkeyutil.InputUSS()
		if err != nil {
			le.Printf("Failed to get USS: %v\n", err)
			exit(1)
		}
	} else if fileUSS != "" {
		secret, err = tkeyutil.ReadUSS(fileUSS)
		if err != nil {
			le.Printf("Failed to read uss-file %s: %v", fileUSS, err)
			exit(1)
		}
	}

	le.Printf("Loading app from %v onto device\n", fileName)

	err = tk.LoadApp(appBin, secret)
	if err != nil {
		le.Printf("LoadAppFromFile failed: %v\n", err)
		exit(1)
	}

	exit(0)
}

// handleSignals mounts a signal handler for the given signals
func handleSignals(action func(), sig ...os.Signal) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	go func() {
		for {
			<-ch
			action()
		}
	}()
}

// readBuildInfo returns the version string from the build info, if available.
func readBuildInfo() string {
	var v string

	if info, ok := debug.ReadBuildInfo(); ok {
		sb := strings.Builder{}
		sb.WriteString("devel")
		for _, setting := range info.Settings {
			if strings.HasPrefix(setting.Key, "vcs") {
				sb.WriteString(fmt.Sprintf(" %s=%s", setting.Key, setting.Value))
			}
		}
		v = sb.String()
	}
	return v
}
