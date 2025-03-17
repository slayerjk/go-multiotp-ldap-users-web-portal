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

	// TEST ONLY!!!
	// del multiOTP user
	err := multiotp.DelMultiOTPUser(user, *multiOTPBinPath)
	if err != nil {
		fmt.Printf("failed to del user:\n\t%v", err)
		os.Exit(1)
	}
	fmt.Printf("%s deleted or doesn't exist\n", user)

	// resync multiOTP users
	err = multiotp.ResyncMultiOTPUsers(*multiOTPBinPath)
	if err != nil {
		fmt.Printf("failed to resync users:\n\t%v", err)
		os.Exit(1)
	}
	fmt.Println("successfully resynced users")

	// get TOTP URL for user
	totpURL, err := multiotp.GetMultiOTPTokenURL(user, *multiOTPBinPath)
	if err != nil {
		fmt.Printf("failed to get totpURL for %s:\n\t%v", user, err)
		os.Exit(1)
	}
	fmt.Printf("got totpURL for %s\n", user)

	// generate QR SVG
	_, err = qrwork.GenerateTOTPSvgQR(totpURL, ".", "TEST")
	if err != nil {
		fmt.Printf("failed to gen QR:\n\t%v", err)
		os.Exit(1)
	}
	fmt.Printf("QR for %s is successfully generated\n", user)
}
