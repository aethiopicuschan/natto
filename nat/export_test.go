package nat

// ExportClassifyNAT exposes classifyNAT for black-box testing.
func ExportClassifyNAT(r *NATResult) {
	classifyNAT(r)
}
