package qrwork

import (
	"bytes"
	"html/template"

	go_qr "github.com/piglig/go-qr"
)

// Generate QR svg file and return template.HTML value(<svg> code)
func GenerateTOTPSvgQrHTML(totpURL []byte) (template.HTML, error) {
	// temp var to write/read svg
	var (
		buf    bytes.Buffer
		result template.HTML
	)

	// svgFilePath := pathToSaveQR + "/" + qrFileName + ".svg"

	// Encode & Generate QR
	errCorLvl := go_qr.Low
	qr, err := go_qr.EncodeText(string(totpURL), errCorLvl)
	if err != nil {
		return result, err
	}
	config := go_qr.NewQrCodeImgConfig(10, 4)

	// err = qr.SVG(config, qrFileName+".svg", "#FFFFFF", "#000000")
	// if err != nil {
	// 	return svgFilePath, err
	// }

	// write svg code to buffer
	err = qr.WriteAsSVG(config, &buf, "#FFFFFF", "#000000")
	if err != nil {
		return result, err
	}

	// read from buffer
	data := make([]byte, buf.Len())
	buf.Read(data)
	result = template.HTML(string(data))

	// svg regexp pattern(search subexpression between <svg> and </svg>)
	// svgContentRegexp := regexp.MustCompile(`^<(svg .*?)>\W(.*\W.*)\W<\/svg>$`)
	// svgContent := svgContentRegexp.FindStringSubmatch(string(data))
	// fmt.Println(svgContent)

	// err = qr.SVG(go_qr.NewQrCodeImgConfig(10, 4, go_qr.WithSVGXMLHeader(true)), "hello-world-QR-xml-header.svg", "#FFFFFF", "#000000")
	// if err != nil {
	// 	return result, err
	// }

	return result, nil
}
