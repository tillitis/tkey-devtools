// SPDX-FileCopyrightText: 2022 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/tillitis/tkeyclient"
	"github.com/tillitis/tkeyutil"
)

// Use when printing err/diag msgs
var le = log.New(os.Stderr, "", 0)

var version string

type appCmd struct {
	code   byte
	name   string
	cmdLen tkeyclient.CmdLen
}

func (c appCmd) Code() byte {
	return c.code
}

func (c appCmd) CmdLen() tkeyclient.CmdLen {
	return c.cmdLen
}

func (c appCmd) Endpoint() tkeyclient.Endpoint {
	return tkeyclient.DestApp
}

func (c appCmd) String() string {
	return c.name
}

var (
	cmdReset = appCmd{0xfe, "cmdReset", tkeyclient.CmdLen4}
)

type fwResetType uint8

// FW reset types
const (
	fwResetTypeStartDefault fwResetType = 0 // Boot from flash slot 0
	// fwResetTypeStartFlash0 fwResetType = 1 // Boot from flash slot 0 (same as default)
	fwResetTypeStartFlash1 fwResetType = 2 // Boot from flash slot 1
	// fwResetTypeStartFlash0Ver fwResetType = 3 // Not implemented currently
	// fwResetTypeStartFlash1Ver fwResetType = 4 // Not implemented currently
	fwResetTypeStartClient fwResetType = 5 // Load app from client
	// fwResetTypeStartClientVer fwResetType = 6 // Not implemented currently
	fwResetTypeStartInvalid fwResetType = 255
)

type bvActionType uint8

// Boot verifier action types
const (
	bvActionTypeApp1    bvActionType = 0 // Verify app in flash slot 1
	bvActionTypeCmdMode bvActionType = 1 // Wait for command from host
	bvActionTypeInvalid bvActionType = 255
)

func main() {
	var fileName, devPath, fileUSS, fwResetStr, bvActionStr string
	var speed int
	var fwReset fwResetType
	var bvAction bvActionType
	var appBin []byte
	var secret []byte
	var enterUSS, verbose, helpOnly, forceFullUss bool
	var err error
	pflag.CommandLine.SetOutput(os.Stderr)
	pflag.CommandLine.SortFlags = false
	pflag.StringVar(&devPath, "port", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")
	pflag.IntVar(&speed, "speed", 0, "Set serial port speed in `BPS` (bits per second).")
	pflag.BoolVar(&enterUSS, "uss", false,
		"Enable typing of a phrase to be hashed as the User Supplied Secret. The USS is loaded onto the TKey along with the app itself and used by the firmware, together with other material, for deriving secrets for the application.")
	pflag.StringVar(&fileUSS, "uss-file", "",
		"Read `FILE` and hash its contents as the USS. Use '-' (dash) to read from stdin. The full contents are hashed unmodified (e.g. newlines are not stripped).")
	pflag.BoolVar(&forceFullUss, "force-full-uss", false, "Force use of 32 byte USS digest.")
	pflag.StringVar(&fwResetStr, "reset", "",
		"Send reset `TYPE`. Can be:\ndefault (boot from flash slot 0)\nflash1 (boot from flash slot 1)\nclient (load app from client)")
	pflag.StringVar(&bvActionStr, "bv", "",
		"Optional for reset. If boot-verifier (normally in flash slot 0) will run after reset, use `ACTION`. Can be:\napp1 (verify app in flash slot 1)\ncmdmode (wait for command from client)")
	pflag.BoolVar(&verbose, "verbose", false, "Enable verbose output.")
	pflag.BoolVar(&helpOnly, "help", false, "Output this help.")
	versionOnly := pflag.BoolP("version", "v", false, "Output version information.")
	pflag.Usage = func() {
		desc := fmt.Sprintf(`Usage: %[1]s [flags...] FILE

%[1]s loads an application binary from FILE onto Tillitis TKey
and starts it.

Exit status code is 0 if the app is both successfully loaded and started. Exit
code is non-zero if anything goes wrong, for example if TKey is already
running some app.`, os.Args[0])
		le.Printf("%s\n\n%s", desc,
			pflag.CommandLine.FlagUsagesWrapped(86))
	}
	pflag.Parse()

	if version == "" {
		version = readBuildInfo()
	}

	notice()

	if pflag.NArg() > 0 {
		if pflag.NArg() > 1 {
			le.Printf("Unexpected argument: %s\n\n", strings.Join(pflag.Args()[1:], " "))
			pflag.Usage()
			os.Exit(1)
		}
		fileName = pflag.Args()[0]
	}

	if *versionOnly {
		le.Printf("tkey-runapp %s", version)
		os.Exit(0)
	}

	if helpOnly {
		pflag.Usage()
		os.Exit(0)
	}

	if fileName == "" && fwResetStr == "" {
		le.Printf("Please pass an app binary FILE.\n\n")
		pflag.Usage()
		os.Exit(1)
	}

	if !verbose {
		tkeyclient.SilenceLogging()
	}

	if enterUSS && fileUSS != "" {
		le.Printf("Can't combine --uss and --uss-file\n\n")
		pflag.Usage()
		os.Exit(1)
	}

	if forceFullUss && fileUSS == "" != enterUSS {
		le.Printf("--force-full-uss unusable unless you also specify --uss or --uss-file.\n\n")
		pflag.Usage()
		os.Exit(2)
	}

	if fileName != "" {
		appBin, err = os.ReadFile(fileName)
		if err != nil {
			le.Printf("Failed to read file: %v\n", err)
			os.Exit(1)
		}
		if bytes.HasPrefix(appBin, []byte("\x7fELF")) {
			le.Printf("%s looks like an ELF executable, but a raw binary is expected.\n", fileName)
			os.Exit(1)
		}
	}

	if devPath == "" {
		devPath, err = tkeyclient.DetectSerialPort(true)
		if err != nil {
			os.Exit(1)
		}
	}

	if fwResetStr != "" {
		fwReset, err = parseFwReset(fwResetStr)
		if err != nil {
			le.Printf("%v\n\n", err)
			pflag.Usage()
			os.Exit(1)
		}
	}

	if bvActionStr != "" {
		bvAction, err = parseBvAction(bvActionStr)
		if err != nil {
			le.Printf("%v\n\n", err)
			pflag.Usage()
			os.Exit(1)
		}
	}

	tk := tkeyclient.New()
	options := []func(*tkeyclient.TillitisKey){}

	if speed != 0 {
		options = append(options, tkeyclient.WithSpeed(speed))
	}

	if forceFullUss {
		options = append(options, tkeyclient.WithFullUss())
	}

	le.Printf("Connecting to device on serial port %s ...\n", devPath)
	if err = tk.Connect(devPath, options...); err != nil {
		le.Printf("Could not open %s: %v\n", devPath, err)
		os.Exit(1)
	}
	exit := func(code int) {
		if err = tk.Close(); err != nil {
			le.Printf("Close: %v\n", err)
		}
		os.Exit(code)
	}
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	if fwResetStr == "" {
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

		le.Printf("UDI: %v\n", udi)
	}

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

	if fwResetStr != "" {
		if bvActionStr == "" {
			bvAction = bvActionTypeApp1
		}
		err = sendReset(tk, fwReset, bvAction)
		if err != nil {
			le.Printf("sendReset failed: %v\n", err)
			exit(1)
		}
		if fileName == "" {
			exit(0)
		}
		le.Printf("Waiting for CH552 to re-enumerate\n")
		time.Sleep(3 * time.Second)
		devPath, err = tkeyclient.DetectSerialPort(true)
		if err != nil {
			exit(1)
		}
		if err = tk.Connect(devPath, options...); err != nil {
			le.Printf("Could not open %s: %v\n", devPath, err)
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

func notice() {
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf("tkey-runapp %v\n", version)
	fmt.Printf(`
NOTE: Version v0.0.1 had a vulnerability. Your keys might have
changed! Read more in the release notes RELEASE.md at
https://github.com/tillitis/tkey-devtools/
`)
	fmt.Printf("--------------------------------------------------------------------------------\n\n")
}

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

func sendReset(tk *tkeyclient.TillitisKey, reset fwResetType, action bvActionType) error {
	id := 0x01

	tx, err := tkeyclient.NewFrameBuf(cmdReset, id)
	if err != nil {
		return fmt.Errorf("failed to create frame buffer: %w", err)
	}

	tx[2] = uint8(reset)
	tx[3] = uint8(action)

	tkeyclient.Dump("reset tx", tx)

	if err = tk.Write(tx); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func parseFwReset(input string) (fwResetType, error) {
	switch input {
	case "default":
		return fwResetTypeStartDefault, nil
	// case "flash0":
	//	return fwResetTypeStartFlash0, nil
	case "flash1":
		return fwResetTypeStartFlash1, nil
	// case "flash0verify":
	//	return fwResetTypeStartFlash0Ver, nil
	// case "flash1verify":
	//	return fwResetTypeStartFlash1Ver, nil
	case "client":
		return fwResetTypeStartClient, nil
	// case "clientverify":
	//	return fwResetTypeStartClientVer, nil
	default:
		return fwResetTypeStartInvalid, errors.New("invalid --reset TYPE")
	}
}

func parseBvAction(input string) (bvActionType, error) {
	switch input {
	case "app1":
		return bvActionTypeApp1, nil
	case "cmdmode":
		return bvActionTypeCmdMode, nil
	default:
		return bvActionTypeInvalid, errors.New("invalid --bv ACTION")
	}
}
