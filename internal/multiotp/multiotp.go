package multiotp

import (
	"fmt"
	"os/exec"
	"regexp"
)

/*
multiotp -qrcode user png_file_name.png
multiotp -update-pin user pin
multiotp -remove-token user

multiotp -urllink user
# otpauth://totp/multiOTP:<NAME>%20<SURNANME>?secret=<BASE32 SEED>&digits=6&period=30
*/

func GetMultiotpTokenURL(user string, multiOTPBinPath string) ([]byte, error) {
	// define command to get TOTP URL for user
	cmd := exec.Command(multiOTPBinPath, "-urllink", user)
	out, _ := cmd.Output()

	// check output is what expected
	pattern := regexp.MustCompile(`^otpauth:`)
	if !pattern.Match(out) {
		return nil, fmt.Errorf("mutliotp command doesn't match '^otpauth://', output:\n\t\t%s", out)
	}
	return out, nil
}
