package app

// include clones slice and appends values to it.
func include[S []E, E any](s S, v ...E) S {
	out := make(S, len(s)+len(v))
	copy(out, s)
	copy(out[len(s):], v)
	return out
}
