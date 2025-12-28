package util

// StringPtr returns a pointer to the string, or nil if empty
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// BoolPtr returns a pointer to the bool
func BoolPtr(b bool) *bool {
	return &b
}

// Int32Ptr returns a pointer to the int32
func Int32Ptr(i int32) *int32 {
	return &i
}
