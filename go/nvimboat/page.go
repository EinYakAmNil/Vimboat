package nvimboat

func (ps *PageStack) Push(p Page) {
	ps.Pages = append(ps.Pages, &p)
}

func (ps *PageStack) Pop() {
	ps.Pages = ps.Pages[:len(ps.Pages)-1]
}

func (ps *PageStack) Top() Page {
	return *ps.Pages[len(ps.Pages)-1]
}
