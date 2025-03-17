package qrwork

import (
	go_qr "github.com/piglig/go-qr"
)

// Generate QR svg file and return its' path
func GenerateTOTPSvgQR(totpURL []byte, pathToSaveQR, qrFileName string) (string, error) {
	svgFilePath := pathToSaveQR + "/" + qrFileName + ".svg"

	// Encode & Generate QR
	errCorLvl := go_qr.Low
	qr, err := go_qr.EncodeText(string(totpURL), errCorLvl)
	if err != nil {
		return svgFilePath, err
	}
	config := go_qr.NewQrCodeImgConfig(10, 4)
	err = qr.PNG(config, qrFileName+".png")
	if err != nil {
		return svgFilePath, err
	}

	err = qr.SVG(config, qrFileName+".svg", "#FFFFFF", "#000000")
	if err != nil {
		return svgFilePath, err
	}

	// err = qr.SVG(go_qr.NewQrCodeImgConfig(10, 4, go_qr.WithSVGXMLHeader(true)), "hello-world-QR-xml-header.svg", "#FFFFFF", "#000000")
	// if err != nil {
	// 	return result, err
	// }

	return svgFilePath, nil
}
