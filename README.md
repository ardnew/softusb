# softusb

[![Go Reference](https://pkg.go.dev/badge/github.com/ardnew/softusb.svg)](https://pkg.go.dev/github.com/ardnew/softusb)
[![Go Report Card](https://goreportcard.com/badge/github.com/ardnew/softusb)](https://goreportcard.com/report/github.com/ardnew/softusb)
[![CodeQL](https://github.com/ardnew/softusb/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/ardnew/softusb/actions/workflows/github-code-scanning/codeql)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Full-featured USB host and device stack written in pure Go.

## Features

- Zero-dependency: no external modules, cgo, assembly, or platform-specific code
- Zero-allocation: designed for embedded and bare-metal applications
- Full USB 1.1 and USB 2.0 support (Low/Full/High Speed)
- Comprehensive transfer support (control, bulk, interrupt, isochronous)
- Standard USB device class implementations:
  - [HID](device/class/hid/) - Human Interface Device (keyboards, mice, gamepads)
  - [CDC-ACM](device/class/cdc/) - Communications Device Class (virtual serial ports)
  - [MSC](device/class/msc/) - Mass Storage Class (USB flash drives, disk images)
- Targets a [hardware abstraction layer (HAL)](#hardware-abstraction-layer-hal) for platform portability
- Asynchronous operation with [context](https://pkg.go.dev/context)-based cancellation (and no dynamic allocations)

### Hardware Abstraction Layer (HAL)

The HAL provides a platform-agnostic interface for the USB [host](host/hal/README.md) and [device](device/hal/README.md) stacks, enabling each to operate consistently across platforms.

The HAL is designed with the following principles in mind:

- Maximize portability by avoiding assumptions about target platform requirements
- Minimize complexity by implementing protocol/common logic within the stack(s)

These principles improve portability and adaptability but also sacrifice some ergonomics:

- Requires implementation-provided data structure storage and memory management
- Increased surface area of HAL interface
- Unable to automatically manage USB PHY or integrated drivers

#### Prototyping

A FIFO-based HAL implementation using named pipes on the local filesystem is provided for testing and debugging without hardware. These operate entirely in user space and can be used on any Go-compatible platform with a filesystem that supports named pipes (e.g., Linux, macOS, Windows).

The [examples/fifo-hal](examples/fifo-hal) directory contains several example applications demonstrating usage of the FIFO HAL with both the host and device stacks. All communication between host and device is output when each example is run. By default, their output is logged separately for clarity:

<details>
<summary>FIFO HAL • CDC-ACM Example</summary>

##### **Command**

>     go test -v -run=TestCDCACMIntegration ./examples/fifo-hal/cdc-acm

##### **Output**

```text
=== RUN   TestCDCACMIntegration
    example_test.go:178: Host output:
        time=2025-12-03T16:54:15.458-06:00 level=INFO msg="starting USB host" component=host busDir=/tmp/softusb-cdc-acm-test-1314831461
        time=2025-12-03T16:54:15.458-06:00 level=DEBUG msg="host FIFO HAL initialized" component=hal busDir=/tmp/softusb-cdc-acm-test-1314831461
        time=2025-12-03T16:54:15.458-06:00 level=DEBUG msg="host FIFO HAL started" component=hal
        time=2025-12-03T16:54:15.459-06:00 level=DEBUG msg="host started" component=host
        time=2025-12-03T16:54:15.459-06:00 level=INFO msg="waiting for device connection" component=host
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="monitoring device directory" component=hal dir=/tmp/softusb-cdc-acm-test-1314831461/device-924d4cab9f6540a0beb33c5befd1375b
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="device connected" component=hal port=1 speed="Full Speed" dir=/tmp/softusb-cdc-acm-test-1314831461/device-924d4cab9f6540a0beb33c5befd1375b
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="device connected" component=host port=1
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="starting enumeration" component=host port=1
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="port reset complete" component=hal port=1
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="got max packet size" component=host size=64
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="assigned address" component=host address=1
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="device descriptor" component=host vendorID=4660 productID=22136 class=0
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="configuration descriptor" component=host numInterfaces=2 configValue=1
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg=manufacturer component=host value="SoftUSB Example"
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg=product component=host value="CDC-ACM Serial Port"
        time=2025-12-03T16:54:16.011-06:00 level=DEBUG msg=serial component=host value=12345678
        time=2025-12-03T16:54:16.011-06:00 level=DEBUG msg="device enumerated" component=host address=1 vendor=4660 product=22136
        time=2025-12-03T16:54:16.011-06:00 level=INFO msg="Device connected" component=host vendorID=4660 productID=22136 manufacturer="SoftUSB Example" product="CDC-ACM Serial Port" serial=12345678
        time=2025-12-03T16:54:16.011-06:00 level=INFO msg="CDC-ACM device detected!" component=host
        time=2025-12-03T16:54:16.011-06:00 level=INFO msg="found bulk endpoints" component=host bulkIn=130 bulkOut=2
        time=2025-12-03T16:54:16.011-06:00 level=INFO msg="sending data" component=host data="Hello from USB Host!"
        time=2025-12-03T16:54:16.011-06:00 level=INFO msg="sent data" component=host bytes=20
        time=2025-12-03T16:54:16.111-06:00 level=INFO msg="Received data" component=host bytes=20 data="Hello from USB Host!"
        time=2025-12-03T16:54:16.111-06:00 level=INFO msg="Serviced devices" component=host count=1
        time=2025-12-03T16:54:16.211-06:00 level=DEBUG msg="host FIFO HAL stopped" component=hal
        time=2025-12-03T16:54:16.211-06:00 level=DEBUG msg="host stopped" component=host
    example_test.go:180: Device output:
        time=2025-12-03T16:54:15.961-06:00 level=DEBUG msg="configuration added" component=device value=1
        time=2025-12-03T16:54:15.961-06:00 level=DEBUG msg="interface added to configuration" component=device config=1 interface=0
        time=2025-12-03T16:54:15.961-06:00 level=DEBUG msg="endpoint added to interface" component=device interface=0 endpoint=129 type=Interrupt direction=IN
        time=2025-12-03T16:54:15.961-06:00 level=DEBUG msg="interface added to configuration" component=device config=1 interface=1
        time=2025-12-03T16:54:15.961-06:00 level=DEBUG msg="endpoint added to interface" component=device interface=1 endpoint=130 type=Bulk direction=IN
        time=2025-12-03T16:54:15.961-06:00 level=DEBUG msg="endpoint added to interface" component=device interface=1 endpoint=2 type=Bulk direction=OUT
        time=2025-12-03T16:54:15.961-06:00 level=DEBUG msg="CDC-ACM configured" component=device dataIn=130 dataOut=2
        time=2025-12-03T16:54:15.961-06:00 level=INFO msg="starting CDC-ACM device" component=device busDir=/tmp/softusb-cdc-acm-test-1314831461 deviceDir=""
        time=2025-12-03T16:54:15.962-06:00 level=DEBUG msg="fifo device HAL initialized" component=hal busDir=/tmp/softusb-cdc-acm-test-1314831461 deviceDir=/tmp/softusb-cdc-acm-test-1314831461/device-924d4cab9f6540a0beb33c5befd1375b uuid=924d4cab9f6540a0beb33c5befd1375b
        time=2025-12-03T16:54:15.962-06:00 level=DEBUG msg="fifo device HAL started" component=hal
        time=2025-12-03T16:54:15.962-06:00 level=DEBUG msg="device stack started" component=stack
        time=2025-12-03T16:54:15.962-06:00 level=INFO msg="waiting for host connection" component=device
        time=2025-12-03T16:54:15.962-06:00 level=INFO msg="Host connected!" component=device
        time=2025-12-03T16:54:15.962-06:00 level=INFO msg="echoing data" component=device
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="port reset received" component=hal
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="device state changed" component=device from=Attached to=Default
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="device reset" component=device
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=256 index=0 length=8
        time=2025-12-03T16:54:16.009-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0100 Index=0x0000 Length=8"
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=hal reqType=0 req=5 value=1 index=0 length=0
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[OUT Standard Device] Request=0x05 Value=0x0001 Index=0x0000 Length=0"
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="device state changed" component=device from=Default to=Address
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="device address set" component=device address=1
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=256 index=0 length=18
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0100 Index=0x0000 Length=18"
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=512 index=0 length=9
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0200 Index=0x0000 Length=9"
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=512 index=0 length=48
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0200 Index=0x0000 Length=48"
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=769 index=1033 length=512
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0301 Index=0x0409 Length=512"
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=770 index=1033 length=512
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0302 Index=0x0409 Length=512"
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=771 index=1033 length=512
        time=2025-12-03T16:54:16.010-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0303 Index=0x0409 Length=512"
        time=2025-12-03T16:54:16.011-06:00 level=DEBUG msg="setup received" component=hal reqType=0 req=9 value=1 index=0 length=0
        time=2025-12-03T16:54:16.011-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[OUT Standard Device] Request=0x09 Value=0x0001 Index=0x0000 Length=0"
        time=2025-12-03T16:54:16.011-06:00 level=DEBUG msg="device state changed" component=device from=Address to=Configured
        time=2025-12-03T16:54:16.011-06:00 level=DEBUG msg="device configured" component=device configuration=1
        time=2025-12-03T16:54:16.013-06:00 level=DEBUG msg="readPacket header" component=hal type=2 length=20
        time=2025-12-03T16:54:16.013-06:00 level=INFO msg="received data" component=device bytes=20 data="Hello from USB Host!"
--- PASS: TestCDCACMIntegration (1.33s)
PASS
ok      github.com/ardnew/softusb/examples/fifo-hal/cdc-acm     1.337s
```

</details>

<details>
<summary>FIFO HAL • HID Keyboard Example</summary>

##### **Command**

>     go test -v -run=TestHIDKeyboardIntegration ./examples/fifo-hal/hid-keyboard

##### **Output**

```text
=== RUN   TestHIDKeyboardIntegration
    example_test.go:178: Host output:
        time=2025-12-03T17:06:01.576-06:00 level=INFO msg="starting USB host" component=host busDir=/tmp/softusb-hid-keyboard-test-1359216892
        time=2025-12-03T17:06:01.576-06:00 level=DEBUG msg="host FIFO HAL initialized" component=hal busDir=/tmp/softusb-hid-keyboard-test-1359216892
        time=2025-12-03T17:06:01.576-06:00 level=DEBUG msg="host FIFO HAL started" component=hal
        time=2025-12-03T17:06:01.576-06:00 level=DEBUG msg="host started" component=host
        time=2025-12-03T17:06:01.576-06:00 level=INFO msg="waiting for device connection" component=host
        time=2025-12-03T17:06:02.127-06:00 level=DEBUG msg="monitoring device directory" component=hal dir=/tmp/softusb-hid-keyboard-test-1359216892/device-525707bc4779411a86095753ea20e202
        time=2025-12-03T17:06:02.127-06:00 level=DEBUG msg="device connected" component=hal port=1 speed="Full Speed" dir=/tmp/softusb-hid-keyboard-test-1359216892/device-525707bc4779411a86095753ea20e202
        time=2025-12-03T17:06:02.127-06:00 level=DEBUG msg="device connected" component=host port=1
        time=2025-12-03T17:06:02.127-06:00 level=DEBUG msg="starting enumeration" component=host port=1
        time=2025-12-03T17:06:02.127-06:00 level=DEBUG msg="port reset complete" component=hal port=1
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="got max packet size" component=host size=64
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="assigned address" component=host address=1
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="device descriptor" component=host vendorID=4660 productID=22137 class=0
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="configuration descriptor" component=host numInterfaces=1 configValue=1
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg=manufacturer component=host value="SoftUSB Example"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg=product component=host value="HID Keyboard"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg=serial component=host value=87654321
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="device enumerated" component=host address=1 vendor=4660 product=22137
        time=2025-12-03T17:06:02.128-06:00 level=INFO msg="Device connected" component=host vendorID=4660 productID=22137 manufacturer="SoftUSB Example" product="HID Keyboard" serial=87654321
        time=2025-12-03T17:06:02.128-06:00 level=INFO msg="HID device detected!" component=host
        time=2025-12-03T17:06:02.128-06:00 level=INFO msg="found interrupt endpoint" component=host interruptIn=129
        time=2025-12-03T17:06:02.128-06:00 level=INFO msg="reading HID reports" component=host
        time=2025-12-03T17:06:02.580-06:00 level=INFO msg=Report component=host reportNum=1 rawData="\x02\x00\v\x00\x00\x00\x00\x00" modifiers=2 modifierNames=[LShift] keycode=11 char=H
        time=2025-12-03T17:06:02.630-06:00 level=INFO msg=Report component=host reportNum=2 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:03.081-06:00 level=INFO msg=Report component=host reportNum=3 rawData="\x00\x00\b\x00\x00\x00\x00\x00" modifiers=0 keycode=8 char=e
        time=2025-12-03T17:06:03.131-06:00 level=INFO msg=Report component=host reportNum=4 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:03.581-06:00 level=INFO msg=Report component=host reportNum=5 rawData="\x00\x00\x0f\x00\x00\x00\x00\x00" modifiers=0 keycode=15 char=l
        time=2025-12-03T17:06:03.631-06:00 level=INFO msg=Report component=host reportNum=6 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:04.080-06:00 level=INFO msg=Report component=host reportNum=7 rawData="\x00\x00\x0f\x00\x00\x00\x00\x00" modifiers=0 keycode=15 char=l
        time=2025-12-03T17:06:04.130-06:00 level=INFO msg=Report component=host reportNum=8 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:04.580-06:00 level=INFO msg=Report component=host reportNum=9 rawData="\x00\x00\x12\x00\x00\x00\x00\x00" modifiers=0 keycode=18 char=o
        time=2025-12-03T17:06:04.631-06:00 level=INFO msg=Report component=host reportNum=10 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:05.081-06:00 level=INFO msg=Report component=host reportNum=11 rawData="\x00\x00(\x00\x00\x00\x00\x00" modifiers=0 keycode=40 char="\n"
        time=2025-12-03T17:06:05.131-06:00 level=INFO msg=Report component=host reportNum=12 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:07.580-06:00 level=INFO msg=Report component=host reportNum=13 rawData="\x02\x00\v\x00\x00\x00\x00\x00" modifiers=2 modifierNames=[LShift] keycode=11 char=H
        time=2025-12-03T17:06:07.631-06:00 level=INFO msg=Report component=host reportNum=14 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:07.681-06:00 level=INFO msg=Report component=host reportNum=15 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:08.080-06:00 level=INFO msg=Report component=host reportNum=16 rawData="\x00\x00\x0f\x00\x00\x00\x00\x00" modifiers=0 keycode=15 char=l
        time=2025-12-03T17:06:08.131-06:00 level=INFO msg=Report component=host reportNum=17 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:08.581-06:00 level=INFO msg=Report component=host reportNum=18 rawData="\x00\x00\x0f\x00\x00\x00\x00\x00" modifiers=0 keycode=15 char=l
        time=2025-12-03T17:06:08.631-06:00 level=INFO msg=Report component=host reportNum=19 rawData="\x00\x00\x00\x00\x00\x00\x00\x00" modifiers=0
        time=2025-12-03T17:06:09.080-06:00 level=INFO msg=Report component=host reportNum=20 rawData="\x00\x00\x12\x00\x00\x00\x00\x00" modifiers=0 keycode=18 char=o
        time=2025-12-03T17:06:09.080-06:00 level=INFO msg="Received 20 reports, stopping" component=host
        time=2025-12-03T17:06:09.080-06:00 level=INFO msg="Serviced devices" component=host count=1
        time=2025-12-03T17:06:09.085-06:00 level=DEBUG msg="host FIFO HAL stopped" component=hal
        time=2025-12-03T17:06:09.085-06:00 level=DEBUG msg="host stopped" component=host
    example_test.go:180: Device output:
        time=2025-12-03T17:06:02.078-06:00 level=DEBUG msg="configuration added" component=device value=1
        time=2025-12-03T17:06:02.078-06:00 level=DEBUG msg="interface added to configuration" component=device config=1 interface=0
        time=2025-12-03T17:06:02.078-06:00 level=DEBUG msg="endpoint added to interface" component=device interface=0 endpoint=129 type=Interrupt direction=IN
        time=2025-12-03T17:06:02.078-06:00 level=DEBUG msg="HID configured" component=device inEP=129 reportDescLen=45
        time=2025-12-03T17:06:02.079-06:00 level=INFO msg="starting HID keyboard device" component=device busDir=/tmp/softusb-hid-keyboard-test-1359216892
        time=2025-12-03T17:06:02.080-06:00 level=DEBUG msg="fifo device HAL initialized" component=hal busDir=/tmp/softusb-hid-keyboard-test-1359216892 deviceDir=/tmp/softusb-hid-keyboard-test-1359216892/device-525707bc4779411a86095753ea20e202 uuid=525707bc4779411a86095753ea20e202
        time=2025-12-03T17:06:02.080-06:00 level=DEBUG msg="fifo device HAL started" component=hal
        time=2025-12-03T17:06:02.080-06:00 level=DEBUG msg="device stack started" component=stack
        time=2025-12-03T17:06:02.080-06:00 level=INFO msg="waiting for host connection" component=device
        time=2025-12-03T17:06:02.080-06:00 level=INFO msg="Host connected!" component=device
        time=2025-12-03T17:06:02.080-06:00 level=INFO msg="typing 'Hello' every 2 seconds" component=device
        time=2025-12-03T17:06:02.127-06:00 level=DEBUG msg="port reset received" component=hal
        time=2025-12-03T17:06:02.127-06:00 level=DEBUG msg="device state changed" component=device from=Attached to=Default
        time=2025-12-03T17:06:02.127-06:00 level=DEBUG msg="device reset" component=device
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=256 index=0 length=8
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0100 Index=0x0000 Length=8"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=0 req=5 value=1 index=0 length=0
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[OUT Standard Device] Request=0x05 Value=0x0001 Index=0x0000 Length=0"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="device state changed" component=device from=Default to=Address
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="device address set" component=device address=1
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=256 index=0 length=18
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0100 Index=0x0000 Length=18"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=512 index=0 length=9
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0200 Index=0x0000 Length=9"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=512 index=0 length=25
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0200 Index=0x0000 Length=25"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=769 index=1033 length=512
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0301 Index=0x0409 Length=512"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=770 index=1033 length=512
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0302 Index=0x0409 Length=512"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=128 req=6 value=771 index=1033 length=512
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[IN Standard Device] Request=0x06 Value=0x0303 Index=0x0409 Length=512"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=hal reqType=0 req=9 value=1 index=0 length=0
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="setup received" component=stack request="SETUP[OUT Standard Device] Request=0x09 Value=0x0001 Index=0x0000 Length=0"
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="device state changed" component=device from=Address to=Configured
        time=2025-12-03T17:06:02.128-06:00 level=DEBUG msg="device configured" component=device configuration=1
        time=2025-12-03T17:06:02.630-06:00 level=INFO msg=Typed: component=device char=H
        time=2025-12-03T17:06:03.131-06:00 level=INFO msg=Typed: component=device char=e
        time=2025-12-03T17:06:03.631-06:00 level=INFO msg=Typed: component=device char=l
        time=2025-12-03T17:06:04.130-06:00 level=INFO msg=Typed: component=device char=l
        time=2025-12-03T17:06:04.631-06:00 level=INFO msg=Typed: component=device char=o
        time=2025-12-03T17:06:05.131-06:00 level=INFO msg=Typed: component=device char="\n"
        time=2025-12-03T17:06:07.631-06:00 level=INFO msg=Typed: component=device char=H
        time=2025-12-03T17:06:07.681-06:00 level=INFO msg=Typed: component=device char=e
        time=2025-12-03T17:06:08.131-06:00 level=INFO msg=Typed: component=device char=l
        time=2025-12-03T17:06:08.631-06:00 level=INFO msg=Typed: component=device char=l
    example_test.go:200: Host received HID reports
    example_test.go:205: Device sent keyboard reports
--- PASS: TestHIDKeyboardIntegration (8.13s)
PASS
ok      github.com/ardnew/softusb/examples/fifo-hal/hid-keyboard        8.133s
```

</details>

## Documentation [![Go Reference](https://pkg.go.dev/badge/github.com/ardnew/softusb.svg)](https://pkg.go.dev/github.com/ardnew/softusb)

Always use [Go doc](https://pkg.go.dev/github.com/ardnew/softusb) for the full API reference. It includes additional package-level documentation, discussions, and usage examples.

### Device Stack

| README | Description |
|---------|-------------|
| [device/hal](device/hal) | Device HAL interface definition |
| [device/hal/fifo](device/hal/fifo) | FIFO-based device HAL implementation |
| [device/class/cdc](device/class/cdc) | CDC-ACM class driver |
| [device/class/hid](device/class/hid) | HID class driver |
| [device/class/msc](device/class/msc) | Mass Storage class driver |

### Host Stack

| README | Description |
|---------|-------------|
| [host/hal](host/hal) | Host HAL interface definition |
| [host/hal/fifo](host/hal/fifo) | FIFO-based host HAL implementation |
| [host/hal/linux](host/hal/linux) | Linux usbfs host HAL implementation |

### Utilities

| README | Description |
|---------|-------------|
| [pkg/prof](pkg/prof) | Profiling utilities (build tag: `profile`) |
| [cmd/softusb-udev-rules](cmd/softusb-udev-rules) | udev rules generator for Linux USB access |

### Examples

| README | Description |
|---------|-------------|
| [examples/fifo-hal](examples/fifo-hal) | FIFO-based HAL examples overview |
| [examples/fifo-hal/cdc-acm](examples/fifo-hal/cdc-acm) | CDC-ACM serial device example |
| [examples/fifo-hal/hid-keyboard](examples/fifo-hal/hid-keyboard) | HID keyboard device example |
| [examples/fifo-hal/msc-disk](examples/fifo-hal/msc-disk) | Mass Storage Class disk device example |
| [examples/linux-hal/hid-monitor](examples/linux-hal/hid-monitor) | Linux USB HID monitor example |

## Quick Start

```go
import (
    "github.com/ardnew/softusb/device"
    "github.com/ardnew/softusb/device/hal/fifo"
    "github.com/ardnew/softusb/device/class/cdc"
)

func main() {
    ctx := context.Background()

    // Create CDC-ACM driver
    acm := cdc.NewACM()

    // Build device
    builder := device.NewDeviceBuilder().
        WithVendorProduct(0xCAFE, 0xBABE).
        WithStrings("Vendor", "CDC Device", "12345").
        AddConfiguration(1)

    acm.ConfigureDevice(builder, 0x81, 0x82, 0x02)
    dev, _ := builder.Build(ctx)

    // Attach driver and start
    acm.AttachToInterfaces(dev, 1, 0, 1)
    hal := fifo.New("/tmp/usb-bus")
    stack := device.NewStack(dev, hal)
    acm.SetStack(stack)
    stack.Start(ctx)
}
```

## License

MIT License - see [LICENSE](LICENSE) for details.
