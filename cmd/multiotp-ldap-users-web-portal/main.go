package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/multiotp"
	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/qrwork"
)

func main() {
	// FLAGS for
	// multiOTP users dir with *.db files

	multiOTPBinPath := flag.String("m", "c:/MultiOTP/windows/multiotp.exe", "Full path to MulitOTP binary")
	flag.Parse()

	user := "marchenm"

	// base32 encoded seed
	// seed := "J6E2I53UDD6WRN7BHFC5DH7EX47VK2TO"

	// get TOTP URL for user
	totpURL, err := multiotp.GetMultiotpTokenURL(user, *multiOTPBinPath)
	if err != nil {
		fmt.Printf("failed to get totpURL for %s:\n\t%v", user, err)
		os.Exit(1)
	}

	// generate QR SVG
	svgFilePath, err := qrwork.GenerateTOTPSvgQR(totpURL, ".", "TEST")
	if err != nil {
		fmt.Printf("failed to gen QR:\n\t%v", err)
		os.Exit(1)
	}
	fmt.Printf("DONE: result is: %s", svgFilePath)

}
