[![ci](https://github.com/tillitis/tkey-devtools/actions/workflows/ci.yaml/badge.svg?branch=main&event=push)](https://github.com/tillitis/tkey-devtools/actions/workflows/ci.yaml)

# tkey-devtools

This repository contains some development tools for the
[Tillitis](https://tillitis.se/) TKey USB security stick.

- `hidread`: A simple tool to read debug prints over the HID debug
  endpoint.

- `tkey-runapp`: A simple development tool to load and start any TKey
  device app.

- `run-tkey-qemu`: Script around our
  [TKey emulator](https://github.com/tillitis/qemu) OCI image
  `ghcr.io/tillitis/tkey-qemu-tk1-23.03.1`.

See the [TKey Developer Handbook](https://dev.tillitis.se/) for how to
develop your own apps, how to run and debug them in the emulator or on
real hardware.

[Current list of known projects](https://dev.tillitis.se/projects/).

## Building

You have two options, either our OCI image
`ghcr.io/tillitis/tkey-builder` for use with a rootless podman setup,
or native tools. See [the Devoloper
Handbook](https://dev.tillitis.se/) for setup.

With native tools you should be able to use make

```
$ make
```

If you want to use podman and you have `make` you can run:

```
$ podman pull ghcr.io/tillitis/tkey-builder:2
$ make podman
```

or run podman directly with

```
$ podman run --rm --mount type=bind,source=.,target=/src -w /src -it ghcr.io/tillitis/tkey-builder:2 make -j
```

To install:

```
sudo make install
```

If you want to reload the udev rules to access the TKey use:

```
sudo make reload-rules
```

Undo the installation with the `uninstall` target.

### Using tkey-runapp

The client app `tkey-runapp` only loads and starts a device app. It's
mostly a development tool. You'll then have to switch to a different
client app that speaks your app's specific protocol. Run with `-h` to
get help.

### Using run-tkey-qemu

```
$ ./run-tkey-qemu
```

This gives you `tkey-qemu-pty` in the current working directory you
can attach your client programs to, typically with `--port
./tkey-qemu-pty`.

### Using hidread

First find the device you want to use. One way to to do that is to use
the [lshid program](https://github.com/sstallion/go-hid), but you
probably don't need this nor `hidread` under Linux. Install with:

```
go install github.com/sstallion/go-hid/cmd/lshid@latest
```

This Go program uses CGO, so you'll also need a working C compiler
that the Go compiler knows about. This is particulary tricky under
Windows.

#### Linux

You don't need hidread on Linux. You can just use `cat`, `od -x`,
`xxd` or whatever you like directly on the raw HID device file.

The important part is finding the device path to use. You can either
use `lshid` (see above) or just check the last few entries from
running `dmesg`, typically something like this:

```
[3726039.053825] hid-generic 0003:1207:8887.0094: hiddev97,hidraw5: USB HID v1.11 Device [Tillitis MTA1-USB-V1] on usb-0000:00:14.0-1.2.4/input2
[3726039.055906] hid-generic 0003:1207:8887.0095: hiddev98,hidraw6: USB HID v1.11 Device [Tillitis MTA1-USB-V1] on usb-0000:00:14.0-1.2.4/input3
```

The last one is the debug pipe. You can just do:

```
cat /dev/hidraw6
```

Note that you *must* listen to this device the entire time if you have
debug prints in your device app, otherwise the communication from your
TKey will stop working.

If you want to use `hidread`, be sure to install `libusb` and its
headers, on some distribution in a separate package, often called
`libusb-dev`.

To compile, you have to have a C compiler installed and then a simple
`go build` usually works. If you want to control what C compiler to
use, `CC=clang go build`.

#### macOS

The easiest way to get a C compiler to build the C parts of `lshid`
and `hidread` is probably by [installing Xcode or the Xcode Command
Line Tools](https://developer.apple.com/xcode/resources/).

Then you should be able to use `go install` and `go build`.

List the devices with `lshid` as described above. It typically looks
like this:

```
mc@MCs-MacBook-Air hidread % lshid
DevSrvsID:4294969540: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294998364: ID 1207:8887 Tillitis MTA1-USB-V1
DevSrvsID:4294969546: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294968691: ID 0000:0000 Apple
DevSrvsID:4294998363: ID 1207:8887 Tillitis MTA1-USB-V1
DevSrvsID:4294969406: ID 0000:0000 APPL BTM
DevSrvsID:4294968689: ID 0000:0000 Apple
DevSrvsID:4294969542: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294969542: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294969542: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294970280: ID 05ac:0281  Keyboard Backlight
DevSrvsID:4294968806: ID 0000:0000 Apple Headset
DevSrvsID:4294969548: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294969544: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294969544: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294969544: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
DevSrvsID:4294969544: ID 05ac:0281 Apple Inc. Apple Internal Keyboard / Trackpad
```

Look for "Tillitis". Usually the second Tillitis HID device is the
debug pipe, so use it's "path" as an argument for `hidread`. Here it
is listening to debug messages from
[tkey-device-signer](https://github.com/tillitis/tkey-device-signer):

```
mc@MCs-MacBook-Air hidread % ./hidread -f DevSrvsID:4294998363
parser state: 0x00000000
Responded NOK to message meant for fw
parser state: 0x00000000
CMD_GET_NAMEVERSION
parser state: 0x00000000
CMD_GET_PUBKEY
parser state: 0x00000000
CMD_SET_SIZE
parser state: 0x00000001
CMD_LOAD_DATA
parser state: 0x00000001
CMD_LOAD_DATA
parser state: 0x00000002
CMD_GET_SIG
Touched, now let's sign
Sending signature!
parser state: 0x00000000
```

#### Windows

The author doesn't really know anything about Windows. The easiest way
I found to get a working C compiler for CGO was to use MSYS2. I
installed it with `winget` in a PowerShell terminal:

```
winget install MSYS2.MSYS2
```

Start the MSYS2 MinGW shell, then install the C compiler, its tools,
and the Go compiler:

```
pacman -S mingw-w64-x86_64-toolchain mingw-w64-x86_64-go
```

After that you should be able to use `go install` and `go build` as
usual.

You can find the debug endpoint with the `lshid` tool in your
PowerShell. Running it looks like this:


```
PS C:\Users\mc> lshid
\\?\HID#INTC816&Col07#3&36043c54&0&0006#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col08#3&36043c54&0&0007#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col09#3&36043c54&0&0008#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#ConvertedDevice&Col02#5&396c30cb&0&0001#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 045e:0000
\\?\HID#SYNA8008&Col01#5&2fbdb7ff&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 06cb:ce58 Microsoft HIDI2C Device
\\?\HID#INTC816&Col11#3&36043c54&0&0010#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#ConvertedDevice&Col03#5&396c30cb&0&0002#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 045e:0000
\\?\HID#ConvertedDevice&Col01#5&396c30cb&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}\KBD: ID 045e:0000
\\?\HID#SYNA8008&Col02#5&2fbdb7ff&0&0001#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 06cb:ce58 Microsoft HIDI2C Device
\\?\HID#INTC816&Col10#3&36043c54&0&000f#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col12#3&36043c54&0&0011#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#SYNA8008&Col03#5&2fbdb7ff&0&0002#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 06cb:ce58 Microsoft HIDI2C Device
\\?\HID#SYNA8008&Col04#5&2fbdb7ff&0&0003#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 06cb:ce58 Microsoft HIDI2C Device
\\?\HID#INTC816&Col0B#3&36043c54&0&000a#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col0C#3&36043c54&0&000b#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col0D#3&36043c54&0&000c#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col0E#3&36043c54&0&000d#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col0F#3&36043c54&0&000e#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col01#3&36043c54&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}\KBD: ID 8087:0a1e
\\?\HID#INTC816&Col02#3&36043c54&0&0001#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col03#3&36043c54&0&0002#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col04#3&36043c54&0&0003#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col05#3&36043c54&0&0004#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col06#3&36043c54&0&0005#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#INTC816&Col0A#3&36043c54&0&0009#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 8087:0a1e
\\?\HID#VID_1207&PID_8887&MI_03#7&c3886b5&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}: ID 1207:8887 Tillitis TKEY-Ctrl
PS C:\Users\mc>
```

Look for the device identified as "Tillitis TKEY-Ctrl".

Then you can listen to the debug by using the hidread program, here in
the MSYS2 shell:

```
$ ./hidread -f '\\?\HID#VID_1207&PID_8887&MI_03#7&c3886b5&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}'
parser state: 0x00000000
```

Note the quotes!

## Licenses and SPDX tags

Unless otherwise noted, the project sources are copyright Tillitis AB,
licensed under the terms and conditions of the "BSD-2-Clause" license.
See [LICENSE](LICENSE) for the full license text.

Until Dec 30, 2024, the license was GPL-2.0 Only.

External source code we have imported are isolated in their own
directories. They may be released under other licenses. This is noted
with a similar `LICENSE` file in every directory containing imported
sources.

The project uses single-line references to Unique License Identifiers
as defined by the Linux Foundation's [SPDX project](https://spdx.org/)
on its own source files, but not necessarily imported files. The line
in each individual source file identifies the license applicable to
that file.

The current set of valid, predefined SPDX identifiers can be found on
the SPDX License List at:

https://spdx.org/licenses/

All contributors must adhere to the [Developer Certificate of Origin](dco.md).
