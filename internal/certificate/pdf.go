package certificate

import (
	"bytes"
	"fmt"
	"time"
)

// PDFData is rendered onto a completion certificate.
type PDFData struct {
	LearnerName      string
	CourseTitle      string
	IssuedAt         time.Time
	VerificationCode string
}

// GeneratePDF builds a minimal valid single-page PDF certificate.
func GeneratePDF(data PDFData) ([]byte, error) {
	text := fmt.Sprintf(
		"Certificate of Completion\n\nThis certifies that\n%s\nhas successfully completed\n%s\n\nIssued: %s\nVerification code: %s",
		data.LearnerName,
		data.CourseTitle,
		data.IssuedAt.UTC().Format("January 2, 2006"),
		data.VerificationCode,
	)
	return minimalPDF(text), nil
}

type pdfDoc struct {
	buf     bytes.Buffer
	offsets []int
}

func (d *pdfDoc) write(s string) {
	d.buf.WriteString(s)
}

func (d *pdfDoc) objStart(id int, body string) {
	d.offsets = append(d.offsets, d.buf.Len())
	fmt.Fprintf(&d.buf, "%d 0 obj\n%s\nendobj\n", id, body)
}

func minimalPDF(text string) []byte {
	d := &pdfDoc{offsets: make([]int, 1)}

	d.write("%PDF-1.4\n")

	contentStream := fmt.Sprintf(`BT
/F1 14 Tf
72 720 Td
(%s) Tj
ET`, pdfEscape(text))

	d.objStart(1, "<< /Type /Catalog /Pages 2 0 R >>")
	d.objStart(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	d.objStart(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>")
	d.objStart(4, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(contentStream), contentStream))
	d.objStart(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	xref := d.buf.Len()
	fmt.Fprintf(&d.buf, "xref\n0 6\n")
	d.write("0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&d.buf, "%010d 00000 n \n", d.offsets[i])
	}
	fmt.Fprintf(&d.buf, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", xref)
	return d.buf.Bytes()
}

func pdfEscape(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', ')', '\\':
			out = append(out, '\\', s[i])
		case '\n':
			out = append(out, '\\', 'n')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}
