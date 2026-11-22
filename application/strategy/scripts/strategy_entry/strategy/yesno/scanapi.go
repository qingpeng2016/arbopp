package yesno

// AllPicksInGroup 当前 store 下该 OutcomeGroup 的全部跨所命中（与 grab 试算同源）。
func (r *Runner) AllPicksInGroup(group OutcomeGroup) []Pick {
	return r.allOutcomeGroupPicks(group, r.limits())
}

// BestPickInGroup 当前 store 下该组最优二元组合。
func (r *Runner) BestPickInGroup(group OutcomeGroup) (Pick, bool) {
	return r.bestOutcomeGroupPick(group, r.limits())
}

// OutcomeGroupsMap 预建的赛果组索引。
func (r *Runner) OutcomeGroupsMap() map[string]OutcomeGroup {
	return r.outcomeGroups
}

// LimitsForScan 当前门槛（与线上一致）。
func (r *Runner) LimitsForScan() Limits {
	return r.limits()
}

// BestFilledInGroup 带平台信息的可成交最优组合。
func (r *Runner) BestFilledInGroup(group OutcomeGroup) (FilledPick, bool) {
	return r.bestFilledPick(group)
}

// SideTopAt 当前 store 下该 outcome 的卖一（与 grab 一致）。
func (r *Runner) SideTopAt(outcomeIndex int) (Side, bool) {
	return r.sideTop(outcomeIndex)
}