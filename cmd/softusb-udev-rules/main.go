// Command softusb-udev-rules generates udev rules for USB device access.
//
// This tool creates udev rules that grant user access to USB devices,
// eliminating the need to run applications as root.
//
// # Usage
//
//	softusb-udev-rules [flags]
//
// # Flags
//
//	-o file       Output file path (default: stdout)
//	-vid id       Filter by USB Vendor ID (hex)
//	-pid id       Filter by USB Product ID (hex)
//	-class id     Filter by USB interface class (hex)
//	-user name    User to grant access (optional)
//	-group name   Group to grant access (default: first of plugdev, input, users)
//	-mode mode    File permissions (default: 0660)
//	-all          Generate rules for all USB devices
//	-hid          Generate rules for all HID devices
//	-manpage      Print the manual page to stdout and exit
//
// # Examples
//
// Generate rules for a specific device:
//
//	softusb-udev-rules -vid 046d -pid c52b -o /etc/udev/rules.d/99-logitech.rules
//
// Generate rules for all HID devices:
//
//	softusb-udev-rules -hid -o /etc/udev/rules.d/99-hid.rules
//
// Generate rules for all USB devices (not recommended for production):
//
//	softusb-udev-rules -all -o /etc/udev/rules.d/99-usb.rules
//
// After creating rules, reload udev:
//
//	sudo udevadm control --reload-rules
//	sudo udevadm trigger
package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const projectURL = "https://github.com/ardnew/softusb"

var (
	outputFile  = flag.String("o", "", "Output file path (default: stdout)")
	vendorID    = flag.String("vid", "", "Filter by Vendor ID (hex)")
	productID   = flag.String("pid", "", "Filter by Product ID (hex)")
	deviceClass = flag.String("class", "", "Filter by interface class (hex)")
	userName    = flag.String("user", "", "User to grant access")
	groupName   = flag.String("group", "plugdev", "Group to grant access")
	fileMode    = flag.String("mode", "0660", "File permissions")
	allDevices  = flag.Bool("all", false, "Generate rules for all USB devices")
	hidDevices  = flag.Bool("hid", false, "Generate rules for all HID devices")
	showManpage = flag.Bool("manpage", false, "Print the manual page to stdout and exit")
)

// groupFallbacks are tried in order when the default group does not exist.
var groupFallbacks = []string{"plugdev", "input", "users"}

// resolveGroup validates the configured group exists on the system. If the user
// did not explicitly set -group, it tries each group in groupFallbacks. Returns
// the resolved group name or an error.
func resolveGroup(explicit bool) (string, error) {
	if explicit {
		if _, err := user.LookupGroup(*groupName); err != nil {
			return "", fmt.Errorf("group %q does not exist on this system", *groupName)
		}
		return *groupName, nil
	}
	for _, g := range groupFallbacks {
		if _, err := user.LookupGroup(g); err == nil {
			return g, nil
		}
	}
	return "", fmt.Errorf(
		"none of the default groups %v exist on this system; use -group to specify one",
		groupFallbacks,
	)
}

// resolveUser validates the configured user exists on the system when -user is
// explicitly set.
func resolveUser(explicit bool) error {
	if !explicit {
		return nil
	}
	if *userName == "" {
		return fmt.Errorf("user name cannot be empty")
	}
	if _, err := user.Lookup(*userName); err != nil {
		return fmt.Errorf("user %q does not exist on this system", *userName)
	}
	return nil
}

func main() {
	flag.Parse()

	if *showManpage {
		fmt.Print(manpage)
		return
	}

	if !*allDevices && !*hidDevices && *vendorID == "" && *deviceClass == "" {
		fmt.Fprintln(os.Stderr, "Error: Must specify at least one of: -all, -hid, -vid, or -class")
		fmt.Fprintln(os.Stderr, "Use -help for usage information")
		os.Exit(1)
	}

	// Detect whether -group was explicitly provided.
	groupExplicit := false
	userExplicit := false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "group":
			groupExplicit = true
		case "user":
			userExplicit = true
		}
	})

	if err := resolveUser(userExplicit); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	resolved, err := resolveGroup(groupExplicit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	*groupName = resolved

	// Header
	var rules []string
	rules = append(rules, headerLines(*outputFile)...)

	if *allDevices {
		rules = append(rules, generateAllUSBRule())
	} else if *hidDevices {
		rules = append(rules, generateHIDRule())
	} else {
		rules = append(rules, generateSpecificRule())
	}

	output := strings.Join(rules, "\n") + "\n"

	if *outputFile == "" {
		fmt.Print(output)
	} else {
		if err := os.WriteFile(*outputFile, []byte(output), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Rules written to %s\n", *outputFile)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To apply the rules, run:")
		fmt.Fprintln(os.Stderr, "  sudo udevadm control --reload-rules")
		fmt.Fprintln(os.Stderr, "  sudo udevadm trigger")
	}
}

func headerLines(path string) []string {
	header := make([]string, 0, 3)
	if path != "" {
		header = append(header, fmt.Sprintf("### %s", filepath.Base(path)))
	}
	return append(header, fmt.Sprintf("# Generated by %s (%s)", commandBasename(), projectURL), "")
}

func commandBasename() string {
	base := filepath.Base(os.Args[0])
	if base == "." || base == string(filepath.Separator) {
		return "softusb-udev-rules"
	}
	return base
}

func ruleAccessAssignments() []string {
	assignments := []string{fmt.Sprintf(`MODE="%s"`, *fileMode)}
	if *userName != "" {
		assignments = append(assignments, fmt.Sprintf(`OWNER="%s"`, *userName))
	}
	return append(assignments, fmt.Sprintf(`GROUP="%s"`, *groupName))
}

// generateAllUSBRule generates a rule for all USB devices.
func generateAllUSBRule() string {
	access := strings.Join(ruleAccessAssignments(), ", ")
	return fmt.Sprintf(
		`# Allow access to all USB devices
SUBSYSTEM=="usb", %s
# Allow access to all USB HIDRAW devices
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", %s
# Allow access to all USB HIDDEV devices
KERNEL=="hiddev*", SUBSYSTEM=="usbmisc", %s`,
		access,
		access,
		access,
	)
}

// generateHIDRule generates rules for all HID devices.
func generateHIDRule() string {
	access := strings.Join(ruleAccessAssignments(), ", ")
	return fmt.Sprintf(
		`# Allow access to all USB HIDRAW devices
KERNEL=="hidraw*", ATTRS{bInterfaceClass}=="03", %s
# Allow access to all USB HIDDEV devices
KERNEL=="hiddev*", SUBSYSTEM=="usbmisc", %s`,
		access,
		access,
	)
}

// formatHexID normalizes a hex string to the given width, stripping any "0x"
// prefix and lowercasing.
func formatHexID(s string, width int) string {
	s = strings.ToLower(strings.TrimPrefix(s, "0x"))
	return fmt.Sprintf("%0*s", width, s)
}

// generateSpecificRule generates rules for specific devices. It covers the USB
// bus device node as well as any hidraw/hiddev nodes belonging to the device.
func generateSpecificRule() string {
	var rules []string

	vid := formatHexID(*vendorID, 4)
	pid := formatHexID(*productID, 4)
	class := formatHexID(*deviceClass, 2)

	// --- USB bus device node (SUBSYSTEM=="usb") ---
	{
		conds := []string{`SUBSYSTEM=="usb"`}
		if *vendorID != "" {
			conds = append(conds, fmt.Sprintf(`ATTR{idVendor}=="%s"`, vid))
		}
		if *productID != "" {
			conds = append(conds, fmt.Sprintf(`ATTR{idProduct}=="%s"`, pid))
		}
		if *deviceClass != "" {
			conds = append(conds, fmt.Sprintf(`ATTR{bInterfaceClass}=="%s"`, class))
		}
		conds = append(conds, ruleAccessAssignments()...)

		rules = append(rules, commentForFilters()+" (USB device)")
		rules = append(rules, strings.Join(conds, ", "))
	}

	// --- hidraw node ---
	{
		conds := []string{`KERNEL=="hidraw*"`}
		if *vendorID != "" {
			conds = append(conds, fmt.Sprintf(`ATTRS{idVendor}=="%s"`, vid))
		}
		if *productID != "" {
			conds = append(conds, fmt.Sprintf(`ATTRS{idProduct}=="%s"`, pid))
		}
		if *deviceClass != "" {
			conds = append(conds, fmt.Sprintf(`ATTRS{bInterfaceClass}=="%s"`, class))
		}
		conds = append(conds, ruleAccessAssignments()...)

		rules = append(rules, commentForFilters()+" (hidraw)")
		rules = append(rules, strings.Join(conds, ", "))
	}

	// --- hiddev node ---
	{
		conds := []string{`KERNEL=="hiddev*"`, `SUBSYSTEM=="usbmisc"`}
		if *vendorID != "" {
			conds = append(conds, fmt.Sprintf(`ATTRS{idVendor}=="%s"`, vid))
		}
		if *productID != "" {
			conds = append(conds, fmt.Sprintf(`ATTRS{idProduct}=="%s"`, pid))
		}
		if *deviceClass != "" {
			conds = append(conds, fmt.Sprintf(`ATTRS{bInterfaceClass}=="%s"`, class))
		}
		conds = append(conds, ruleAccessAssignments()...)

		rules = append(rules, commentForFilters()+" (hiddev)")
		rules = append(rules, strings.Join(conds, ", "))
	}

	return strings.Join(rules, "\n")
}

func commentForFilters() string {
	comment := "# Allow access to"
	if *vendorID != "" {
		comment += fmt.Sprintf(" VID=%s", *vendorID)
	}
	if *productID != "" {
		comment += fmt.Sprintf(" PID=%s", *productID)
	}
	if *deviceClass != "" {
		comment += fmt.Sprintf(" Class=%s", *deviceClass)
	}
	return comment
}
