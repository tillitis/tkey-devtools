// SPDX-FileCopyrightText: 2025 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sstallion/go-hid"
)

var version string

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

func main() {
	printHex := flag.Bool("x", false, "Output HID Input Reports in hex")
	help := flag.Bool("h", false, "Give help")
	path := flag.String("f", "", "File path to device")
	size := flag.Int("s", 64, "Size of Input Reports to read")
	versionOnly := flag.Bool("v", false, "Output version information.")
	enumerate := flag.Bool("e", false, "Enumerate devices.")
	vid := flag.Uint("vid", 0, "VID for device to open.")
	pid := flag.Uint("pid", 0, "PID for device to open.")
	continue_search := flag.Bool("c", false, "Continue scanning for device with specified VID and PID, open when it appears.")

	flag.Parse()

	if version == "" {
		version = readBuildInfo()
	}

	if *versionOnly {
		fmt.Printf("hidread %s\n", version)
		os.Exit(0)
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *enumerate {
		hid.Enumerate(hid.VendorIDAny, hid.ProductIDAny, func(info *hid.DeviceInfo) error {
			fmt.Printf("%s: ID %04x:%04x %s %s\n",
				info.Path,
				info.VendorID,
				info.ProductID,
				info.MfrStr,
				info.ProductStr)
			return nil
		})
		os.Exit(0)
	}

	var hid_dev hid.DeviceInfo
	if *vid != 0 && *pid != 0 {
		for {
			hid.Enumerate(uint16(*vid), uint16(*pid), func(info *hid.DeviceInfo) error {
				fmt.Printf("%s: ID %04x:%04x %s %s\n",
					info.Path,
					info.VendorID,
					info.ProductID,
					info.MfrStr,
					info.ProductStr)
				hid_dev = *info

				return nil
			})

			*path = hid_dev.Path

			if *path != "" || !*continue_search {
				break
			}
			time.Sleep(1 * time.Second)
			fmt.Printf("Searching..\n")
		}
	}

	if *path == "" {
		fmt.Printf("Path empty\n")
		os.Exit(0)
	}

	dev, err := hid.OpenPath(*path)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	report := make([]byte, *size)

	var length int

	for {
		length, err = dev.Read(report)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		if length == 0 {
			continue
		}

		if *printHex {
			fmt.Printf("%v\n", hex.Dump(report))
		} else {
			fmt.Printf("%v", string(report))
		}
	}
}
