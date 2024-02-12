package nvimboat

import (
	"errors"
	"log"
)

func (nb *Nvimboat) Command(args []string) error {
	err := nb.batch.Execute()
	if err != nil {
		log.Println(err)
		return err
	}
	if nb.LogFile == nil {
		nb.setupLogging()
	}
	if nb.DB == nil {
		dbpath := nb.Config["dbpath"].(string)
		nb.DB, err = initDB(dbpath)
		if err != nil {
			nb.Log("Error opening the database:")
			nb.Log(err)
		}
	}
	action := args[0]
	switch action {
	case "enable":
		err = nb.Enable()
	case "disable":
		err = nb.Disable()
	case "show-main":
		err = nb.ShowMain()
	case "show-tags":
		err = nb.ShowTags()
	case "select":
		if len(args) > 1 {
			err = nb.Select(args[1])
			return nil
		}
		return errors.New("No arguments for select command.")
	case "back":
		err = nb.Back()
	case "next-unread":
		nb.NextUnread()
	case "prev-unread":
		nb.PrevUnread()
	case "next-article":
		nb.NextArticle()
	case "prev-article":
		nb.PrevArticle()
	case "toggle-article-read":
		nb.ToggleArticleRead(args[1:]...)
	default:
		nb.Log("Not yet mapped: ", args)
	}
	if err != nil {
		nb.Log(err)
		return err
	}
	return nil
}

func (nb *Nvimboat) Enable() error {
	mainmenu, err := nb.showMain()
	if err != nil {
		return err
	}
	err = nb.Push(mainmenu)
	if err != nil {
		return err
	}
	err = nb.plugin.Nvim.ExecLua(nvimboatEnable, new(any))
	if err != nil {
		return err
	}
	return nil
}

func (nb *Nvimboat) Disable() error {
	err := nb.plugin.Nvim.ExecLua(nvimboatDisable, new(any))
	if err != nil {
		return err
	}

	return nil
}

func (nb *Nvimboat) Select(id string) error {
	defer nb.plugin.Nvim.SetWindowCursor(*nb.window, [2]int{0, 1})
	switch nb.PageStack.top.(type) {
	case *MainMenu:
		if id[:4] == "http" {
			feed, err := nb.QueryFeed(id)
			if err != nil {
				return err
			}
			err = nb.Push(&feed)
			if err != nil {
				return err
			}
		}
		if id[:6] == "query:" {
			query, inTags, exTags, err := parseFilterID(id)
			filter, err := nb.QueryFilter(query, inTags, exTags)
			filter.FilterID = id
			if err != nil {
				return err
			}
			err = nb.Push(&filter)
			if err != nil {
				return err
			}
		}
	case *Filter:
		articles := nb.PageStack.top.(*Filter).Articles
		for _, a := range articles {
			if a.Url == id {
				a.Unread = 0
				err := nb.Push(a)
				if err != nil {
					return err
				}
				err = nb.setArticleRead(id)
				if err != nil {
					return err
				}
			}
		}
	case *Feed:
		article, err := nb.QueryArticle(id)
		if err != nil {
			return err
		}
		nb.Push(&article)
		if err != nil {
			return err
		}
		err = nb.setArticleRead(id)
		if err != nil {
			return err
		}
	case *TagsPage:
		feeds, err := nb.QueryTagFeeds(id)
		if err != nil {
			return err
		}
		nb.Push(&feeds)
		if err != nil {
			return err
		}
	case *TagFeeds:
		feed, err := nb.QueryFeed(id)
		if err != nil {
			return err
		}
		err = nb.Push(&feed)
		if err != nil {
			return err
		}
	case *Article:
		return nil
	}
	return nil
}

func (nb *Nvimboat) Back() error {
	switch nb.PageStack.top.(type) {
	case *MainMenu:
		return nil
	default:
		nb.Pop()
	}
	return nil
}

func (nb *Nvimboat) ShowMain() error {
	mainmenu, err := nb.showMain()
	if err != nil {
		return err
	}
	err = nb.Push(mainmenu)
	if err != nil {
		return err
	}
	nb.PageStack.Pages = nb.PageStack.Pages[:1]
	nb.PageStack.top = mainmenu
	return nil
}

func (nb *Nvimboat) ShowTags() error {
	tags, err := nb.QueryTags()
	if err != nil {
		return err
	}
	nb.Push(&tags)
	if err != nil {
		return err
	}
	return nil
}

func (nb *Nvimboat) ToggleArticleRead(urls ...string) error {
	var err error
	if urls[0] == "Article" {
		err = nb.setArticleUnread(nb.PageStack.top.(*Article).Url)
		if err != nil {
			return err
		}
		switch f := nb.PageStack.Pages[len(nb.PageStack.Pages)-2].(type) {
		case *Filter:
			pos, err := f.ElementIdx(nb.PageStack.top)
			if err != nil {
				return err
			}
			f.Articles[pos].Unread = 1
		case *Feed:
			pos, err := f.ElementIdx(nb.PageStack.top)
			if err != nil {
				return err
			}
			f.Articles[pos].Unread = 1
		}
		nb.Pop()
		return err
	}
	anyUnread, err := nb.anyArticleUnread(urls...)
	if anyUnread {
		err = nb.setArticleRead(urls...)
		if err != nil {
			return err
		}
	} else {
		err = nb.setArticleUnread(urls...)
		if err != nil {
			return err
		}
	}
	switch p := nb.PageStack.top.(type) {
	case *Filter:
		var r int
		if anyUnread {
			r = 0
		} else {
			r = 1
		}
		for _, a := range p.Articles {
			for _, u := range urls {
				if a.Url == u {
					a.Unread = r
				}
			}
		}
		err = nb.Show(p)
		return err
	default:
		newPage, err := nb.RequeryPage(p)
		if err != nil {
			return err
		}
		err = nb.Show(newPage)
	}
	return err
}

func (nb *Nvimboat) NextUnread() error {
	switch p := nb.PageStack.Pages[len(nb.PageStack.Pages)-2].(type) {
	case *Filter:
		start, err := p.ElementIdx(nb.PageStack.top)
		if err != nil {
			return errors.New("Couldn't find article in filter.")
		}
		for i := start + 1; i < len(p.Articles); i++ {
			if p.Articles[i].Unread == 1 {
				err = nb.Show(p.Articles[i])
				if err != nil {
					return err
				}
				err = nb.setArticleRead(p.Articles[i].Url)
				if err != nil {
					return err
				}
				p.Articles[i].Unread = 0
				nb.PageStack.Pages[len(nb.PageStack.Pages)-2] = p
				nb.PageStack.top = p.Articles[i]
				return nil
			}
		}
	case *Feed:
		start, err := p.ElementIdx(nb.PageStack.top)
		if err != nil {
			return errors.New("Couldn't find article in filter.")
		}
		for i := start + 1; i < len(p.Articles); i++ {
			if p.Articles[i].Unread == 1 {
				err = nb.Show(&p.Articles[i])
				if err != nil {
					return err
				}
				err = nb.setArticleRead(p.Articles[i].Url)
				if err != nil {
					return err
				}
				p.Articles[i].Unread = 0
				nb.PageStack.Pages[len(nb.PageStack.Pages)-2] = p
				nb.PageStack.top = &p.Articles[i]
				return nil
			}
		}
	default:
		return nil
	}
	return nil
}

func (nb *Nvimboat) PrevUnread() error {
	switch p := nb.PageStack.Pages[len(nb.PageStack.Pages)-2].(type) {
	case *Filter:
		start, err := p.ElementIdx(nb.PageStack.top)
		if err != nil {
			return errors.New("Couldn't find article in filter.")
		}
		for i := start - 1; i >= 0; i-- {
			if p.Articles[i].Unread == 1 {
				err = nb.Show(p.Articles[i])
				if err != nil {
					return err
				}
				err = nb.setArticleRead(p.Articles[i].Url)
				if err != nil {
					return err
				}
				p.Articles[i].Unread = 0
				nb.PageStack.Pages[len(nb.PageStack.Pages)-2] = p
				nb.PageStack.top = p.Articles[i]
				return nil
			}
		}
	case *Feed:
		start, err := p.ElementIdx(nb.PageStack.top)
		if err != nil {
			return errors.New("Couldn't find article in filter.")
		}
		for i := start - 1; i >= 0; i-- {
			if p.Articles[i].Unread == 1 {
				err = nb.Show(&p.Articles[i])
				if err != nil {
					return err
				}
				err = nb.setArticleRead(p.Articles[i].Url)
				if err != nil {
					return err
				}
				p.Articles[i].Unread = 0
				nb.PageStack.Pages[len(nb.PageStack.Pages)-2] = p
				nb.PageStack.top = &p.Articles[i]
				return nil
			}
		}
	default:
		return nil
	}
	return nil
}

func (nb *Nvimboat) NextArticle() error {
	var a Article
	switch p := nb.PageStack.top.(type) {
	case *Article:
		stack := nb.PageStack.Pages
		n := len(stack) - 2
		i, err := stack[n].ElementIdx(p)
		if err != nil {
			return err
		}
		i++
		switch f := stack[n].(type) {
		case *Filter:
			if i >= len(f.Articles) {
				return errors.New("Already the last article of the feed.")
			}
			a = *f.Articles[i]
			err = nb.Show(f.Articles[i])
			if err != nil {
				return err
			}
			f.Articles[i].Unread = 0
			stack[n] = f
			err = nb.setArticleRead(f.Articles[i].Url)
			if err != nil {
				return err
			}
		case *Feed:
			if i >= len(f.Articles) {
				return errors.New("Already the last article of the feed.")
			}
			a = f.Articles[i]
			err = nb.Show(&f.Articles[i])
			if err != nil {
				return err
			}
			f.Articles[i].Unread = 0
			stack[n] = f
			err = nb.setArticleRead(f.Articles[i].Url)
			if err != nil {
				return err
			}
		default:
			return errors.New("Previous page is not a feed/filter.")
		}
		nb.PageStack.Pages[len(nb.PageStack.Pages)-2] = stack[n]
		nb.PageStack.top = &a
		return nil
	default:
		return errors.New("Not inside an article")
	}
}

func (nb *Nvimboat) PrevArticle() error {
	var a Article
	switch p := nb.PageStack.top.(type) {
	case *Article:
		stack := nb.PageStack.Pages
		n := len(stack) - 2
		i, err := stack[n].ElementIdx(p)
		if err != nil {
			return err
		}
		if i == 0 {
			return errors.New("Already the first article of the feed.")
		}
		i--
		switch f := stack[n].(type) {
		case *Filter:
			a = *f.Articles[i]
			err = nb.Show(f.Articles[i])
			if err != nil {
				return err
			}
			f.Articles[i].Unread = 0
			stack[n] = f
			err = nb.setArticleRead(f.Articles[i].Url)
			if err != nil {
				return err
			}
		case *Feed:
			a = f.Articles[i]
			err = nb.Show(&f.Articles[i])
			if err != nil {
				return err
			}
			f.Articles[i].Unread = 0
			stack[n] = f
			err = nb.setArticleRead(f.Articles[i].Url)
			if err != nil {
				return err
			}
		default:
			return errors.New("Previous page is not a feed/filter.")
		}
		nb.PageStack.Pages[len(nb.PageStack.Pages)-2] = stack[n]
		nb.PageStack.top = &a
		return nil
	default:
		return errors.New("Not inside an article")
	}
}
