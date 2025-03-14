package multiotp

/*
multiotp -qrcode user png_file_name.png
multiotp -update-pin user pin
multiotp -remove-token user

multiotp -urllink user
# otpauth://totp/multiOTP:<NAME>%20<SURNANME>?secret=<BASE32 SEED>&digits=6&period=30
*/
