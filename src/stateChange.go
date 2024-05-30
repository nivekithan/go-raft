package raft

type stateChangeReq interface {
	updateState(r *Raft)
	term() int
}

type convertToFollower struct {
	_term   int
	newTerm int
}

func (c *convertToFollower) term() int {
	return c._term
}

func (c *convertToFollower) updateState(r *Raft) {
	r.convertToFollower(c.newTerm)
}

type convertToLeader struct {
	_term int
}

func (c *convertToLeader) term() int {
	return c._term
}

func (c *convertToLeader) updateState(r *Raft) {
	r.convertToLeader()
}

type decreaseNextIndex struct {
	_term int
	id    int
}

func (d *decreaseNextIndex) term() int {
	return d._term
}

func (d *decreaseNextIndex) updateState(r *Raft) {
	r.nextIndex[r.id]--
}
