package chore

import (
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/templ"
	"io"
	"time"
)

type Templates struct {
	p templ.TemplateProvider
}

type ChoreListsView struct {
	ChoreLists []cdb.ChoreList
}

func (t *Templates) ChoreListsPage(w io.Writer, d ChoreListsView) error {
	return t.p.ExecuteTemplate(w, "chore_lists.page.gohtml", d)
}

func (t *Templates) ChoreListNewPage(w io.Writer) error {
	return t.p.ExecuteTemplate(w, "chore-list-new.page.gohtml", nil)
}

type ChoreListView struct {
	List    cdb.ChoreList
	Weekday time.Weekday
	Chores  *ListView
}

func (t *Templates) ChoreListPage(w io.Writer, d ChoreListView) error {
	return t.p.ExecuteTemplate(w, "chore_list.page.gohtml", d)
}

func (t *Templates) ChoreList(w io.Writer, d ChoreListView) error {
	return t.p.ExecuteTemplate(w, "chore_list.gohtml", d)
}

func (t *Templates) ChoreModal(w io.Writer, d *Chore) error {
	return t.p.ExecuteTemplate(w, "chore-modal.gohtml", d)
}

type SettingsView struct {
	UserID         string
	Usernames      []string
	ChoreLists     []cdb.ChoreList
	CreatedInvites []cdb.GetInvitationsByCreatorRow
}

func (t *Templates) SettingsPage(w io.Writer, d SettingsView) error {
	return t.p.ExecuteTemplate(w, "settings.page.gohtml", d)
}

type InviteCreateView struct {
	ChoreLists []cdb.ChoreList
}

func (t *Templates) InviteCreate(w io.Writer, d InviteCreateView) error {
	return t.p.ExecuteTemplate(w, "invite_create.gohtml", d)
}

type InviteView struct {
	InviteID      string
	ChoreListName string
}

func (t *Templates) InvitePage(w io.Writer, d InviteView) error {
	return t.p.ExecuteTemplate(w, "invite.gohtml", d)
}

type InviteAcceptView struct {
	InviteID      string
	ChoreListName string
	InviterName   string
	ExistingUser  bool
}

func (t *Templates) InviteAcceptPage(w io.Writer, d InviteAcceptView) error {
	return t.p.ExecuteTemplate(w, "invite_accept.page.gohtml", d)
}

func (t *Templates) ChoreElement(w io.Writer, d *Chore) error {
	return t.p.ExecuteTemplate(w, "chore-element.gohtml", d)
}

func (t *Templates) LoginPage(w io.Writer) error {
	return t.p.ExecuteTemplate(w, "login.page.gohtml", nil)
}
