package core

import (
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/templ"
	"io"
	"net/http"
	"time"
)

type View struct {
	p templ.TemplateProvider
}

type ChoreListsView struct {
	ChoreLists []cdb.ChoreList
}

func (t *View) ChoreListsPage(w io.Writer, r *http.Request, d ChoreListsView) error {
	return t.p.ExecuteTemplate(w, "chore_lists.page.gohtml", d)
}

func (t *View) ChoreListNewPage(w io.Writer, r *http.Request) error {
	return t.p.ExecuteTemplate(w, "chore-list-new.page.gohtml", nil)
}

type ChoreListView struct {
	List    cdb.ChoreList
	Weekday time.Weekday
	Chores  *ListView
}

func (t *View) ChoreListPage(w io.Writer, r *http.Request, d ChoreListView) error {
	if r.Header.Get("HX-Request") == "true" {
		return t.p.ExecuteTemplate(w, "chore_list.gohtml", d)
	}
	return t.p.ExecuteTemplate(w, "chore_list.page.gohtml", d)
}

func (t *View) ChoreModal(w io.Writer, r *http.Request, d *Chore) error {
	return t.p.ExecuteTemplate(w, "chore-modal.gohtml", d)
}

type SettingsView struct {
	UserID         string
	Usernames      []string
	ChoreLists     []cdb.ChoreList
	CreatedInvites []cdb.GetInvitationsByCreatorRow
}

func (t *View) SettingsPage(w io.Writer, r *http.Request, d SettingsView) error {
	return t.p.ExecuteTemplate(w, "settings.page.gohtml", d)
}

type InviteCreateView struct {
	ChoreLists []cdb.ChoreList
}

func (t *View) InviteCreate(w io.Writer, r *http.Request, d InviteCreateView) error {
	return t.p.ExecuteTemplate(w, "invite_create.gohtml", d)
}

type InviteView struct {
	InviteID      string
	ChoreListName string
}

func (t *View) InvitePage(w io.Writer, r *http.Request, d InviteView) error {
	return t.p.ExecuteTemplate(w, "invite.gohtml", d)
}

type InviteAcceptView struct {
	InviteID      string
	ChoreListName string
	InviterName   string
	ExistingUser  bool
}

func (t *View) InviteAcceptPage(w io.Writer, r *http.Request, d InviteAcceptView) error {
	return t.p.ExecuteTemplate(w, "invite_accept.page.gohtml", d)
}

func (t *View) ChoreElement(w io.Writer, r *http.Request, d *Chore) error {
	return t.p.ExecuteTemplate(w, "chore-element.gohtml", d)
}

func (t *View) LoginPage(w io.Writer, r *http.Request) error {
	return t.p.ExecuteTemplate(w, "login.page.gohtml", nil)
}
