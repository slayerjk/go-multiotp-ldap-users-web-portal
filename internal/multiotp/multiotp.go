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

# reissue token
1) multiotp -delete user
2) multiotp -ldap-users-sync

# get totpURL
multiotp -urllink user
# otpauth://totp/multiOTP:<NAME>%20<SURNANME>?secret=<BASE32 SEED>&digits=6&period=30
*/

// Get MultiOTP user's totpURL
func GetMultiOTPTokenURL(user string, multiOTPBinPath string) ([]byte, error) {
	// define command to get TOTP URL for user
	cmd := exec.Command(multiOTPBinPath, "-urllink", user)
	// due to multiotp console tools throw Exit codes every time
	// need to check err.ExitCode, because err will be always
	// exit status 17: is success for '-urllink' cmd
	out, err := cmd.Output()
	if err, ok := err.(*exec.ExitError); ok {
		// 17 INFO: UrlLink successfully created
		// 21 ERROR: User doesn't exist
		switch {
		case err.ExitCode() == 21:
			return nil, fmt.Errorf("%s doesn't exist", user)
		case err.ExitCode() != 17:
			return nil, err
		}
	}

	// check output is what expected
	pattern := regexp.MustCompile(`^otpauth:`)
	if !pattern.Match(out) {
		return nil, fmt.Errorf("mutliotp command doesn't match '^otpauth://', output:\n\t\t%s", out)
	}
	return out, nil
}

// Delete MultiOTP User
// If user doesn't exist - returns noting(not error)
func delMultiOTPUser(multiOTPBinPath string, user string) error {
	// define command to delete user
	cmd := exec.Command(multiOTPBinPath, "-delete", user)
	// due to multiotp console tools throw Exit codes every time
	// need to check err.ExitCode, because err will be always
	// 12 INFO: User successfully deleted: is success for '-delete user' cmd
	// OR
	// 19 INFO: Requested operation successfully done: is success for '-delete user' cmd
	// 21 ERROR: User doesn't exist: not error
	_, err := cmd.Output()
	if err, ok := err.(*exec.ExitError); ok {
		switch {
		case err.ExitCode() == 21:
			return nil
		case err.ExitCode() != 12 && err.ExitCode() != 19:
			return err
		}
	}

	return nil
}

// Resync MultiOTP Users
func resyncMultiOTPUsers(multiOTPBinPath string) error {
	// define command to delete user
	cmd := exec.Command(multiOTPBinPath, "-ldap-users-sync")
	// due to multiotp console tools throw Exit codes every time
	// need to check err.ExitCode, because err will be always
	// 19 INFO: Requested operation successfully done: is success for '-delete user' cmd
	_, err := cmd.Output()

	if err, ok := err.(*exec.ExitError); ok {
		if err.ExitCode() != 19 {
			return err
		}
	}

	return nil
}

// Reissue MultiOTP QR
func ReissueMultiOTPQR(multiOTPBinPath string, user string) error {
	// first del user from MultiOTP db
	err := delMultiOTPUser(multiOTPBinPath, user)
	if err != nil {
		return fmt.Errorf("reissue qr: failed to del user:\n\t%v", err)
	}

	// second resync MultiOTP db to get same user back with new QR generated
	// may take some time to resync(depend of users number)
	err = resyncMultiOTPUsers(multiOTPBinPath)
	if err != nil {
		return fmt.Errorf("reissue qr: failed to resync users:\n\t%v", err)
	}

	return nil
}
