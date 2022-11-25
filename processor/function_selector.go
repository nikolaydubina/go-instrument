package processor

// MapFunctionSelector makes decision basedd on map value or else default.
type MapFunctionSelector struct {
	AcceptFunctions map[string]bool
	Default         bool
}

func (s MapFunctionSelector) AcceptFunction(functionName string) bool {
	v, ok := s.AcceptFunctions[functionName]
	if ok {
		return v
	}
	return s.Default
}
