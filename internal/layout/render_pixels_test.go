package layout_test

// Pixel assertions on HTML need the drawing and PDF packages, which layout
// cannot import; the blank import installs them.
import _ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
