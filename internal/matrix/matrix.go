package matrix

// Matrix is a 3×3 transformation matrix acting on row vectors, so that a
// point transforms as [x y 1] · M and chained transforms multiply left to
// right (ISO 32000-1 §8.3.4).
//
// The last column is always [0 0 1]ᵀ for affine transforms; the six
// meaningful entries are exposed through New and Values using the PDF
// operand order a b c d e f.
type Matrix [3][3]float64

// Identity is the matrix that leaves every point unchanged.
var Identity = New(1, 0, 0, 1, 0, 0)

// New returns the affine matrix [a b 0; c d 0; e f 1].
func New(a, b, c, d, e, f float64) Matrix {
	return Matrix{{a, b, 0}, {c, d, 0}, {e, f, 1}}
}

// Mul returns m·o: the transform that applies m first and then o.
func (m Matrix) Mul(o Matrix) Matrix {
	var r Matrix
	for i := range r {
		for j := range r[i] {
			for k := range m[i] {
				r[i][j] += m[i][k] * o[k][j]
			}
		}
	}
	return r
}

// Determinant returns the determinant of m.
func (m Matrix) Determinant() float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[1][0]*(m[0][1]*m[2][2]-m[0][2]*m[2][1]) +
		m[2][0]*(m[0][1]*m[1][2]-m[0][2]*m[1][1])
}

// Invert returns the inverse of m. The boolean is false when m is singular.
func (m Matrix) Invert() (Matrix, bool) {
	d := m.Determinant()
	if d == 0 {
		return Matrix{}, false
	}
	return Matrix{
		{
			(m[1][1]*m[2][2] - m[1][2]*m[2][1]) / d,
			(m[0][1]*m[2][2] - m[0][2]*m[2][1]) / -d,
			(m[0][1]*m[1][2] - m[0][2]*m[1][1]) / d,
		},
		{
			(m[1][0]*m[2][2] - m[1][2]*m[2][0]) / -d,
			(m[0][0]*m[2][2] - m[0][2]*m[2][0]) / d,
			(m[0][0]*m[1][2] - m[0][2]*m[1][0]) / -d,
		},
		{
			(m[1][0]*m[2][1] - m[1][1]*m[2][0]) / d,
			(m[0][0]*m[2][1] - m[0][1]*m[2][0]) / -d,
			(m[0][0]*m[1][1] - m[0][1]*m[1][0]) / d,
		},
	}, true
}

// TransformPoint applies m to the point (x, y).
func (m Matrix) TransformPoint(x, y float64) (float64, float64) {
	return x*m[0][0] + y*m[1][0] + m[2][0],
		x*m[0][1] + y*m[1][1] + m[2][1]
}

// Values returns the six entries a, b, c, d, e, f in PDF operand order.
func (m Matrix) Values() (a, b, c, d, e, f float64) {
	return m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1]
}
