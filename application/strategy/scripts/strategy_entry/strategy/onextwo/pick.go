package onextwo

func (r *Runner) bestTriplePick() (TriplePick, bool) {
	return bestTripleFrom(r.allTriplePicks(r.limits()))
}
