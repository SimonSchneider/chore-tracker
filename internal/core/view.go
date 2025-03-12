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

func NewView(p templ.TemplateProvider) *View {
	return &View{p: p}
}

type ChoreListsView struct {
	ChoreLists []cdb.GetChoreListsByUserRow
}

func (v *View) ChoreListsPage(w io.Writer, r *http.Request, d ChoreListsView) error {
	return v.p.ExecuteTemplate(w, "chore_lists.page.gohtml", d)
}

func (v *View) ChoreListNewPage(w io.Writer, r *http.Request) error {
	return v.p.ExecuteTemplate(w, "chore-list-new.page.gohtml", nil)
}

type ChoreListEditView struct {
	List    cdb.ChoreList
	Members []cdb.GetChoreListMembersRow
	Invites []cdb.Invitation
}

func (v *View) ChoreListEditPage(w io.Writer, r *http.Request, d ChoreListEditView) error {
	return v.p.ExecuteTemplate(w, "chore_list_edit.page.gohtml", d)
}

type ChoreListView struct {
	List    cdb.ChoreList
	Weekday time.Weekday
	Chores  *ListView
}

func (v *View) ChoreListPage(w io.Writer, r *http.Request, d ChoreListView) error {
	if r.Header.Get("HX-Request") == "true" {
		return v.p.ExecuteTemplate(w, "chore_list.gohtml", d)
	}
	return v.p.ExecuteTemplate(w, "chore_list.page.gohtml", d)
}

func (v *View) ChoreModal(w io.Writer, r *http.Request, d *Chore) error {
	return v.p.ExecuteTemplate(w, "chore-modal.gohtml", d)
}

type SettingsView struct {
	UserID         string
	Usernames      []string
	ChoreLists     []cdb.GetChoreListsByUserRow
	CreatedInvites []cdb.GetInvitationsByCreatorRow
}

func (v *View) SettingsPage(w io.Writer, r *http.Request, d SettingsView) error {
	return v.p.ExecuteTemplate(w, "settings.page.gohtml", d)
}

type InviteCreateView struct {
	ChoreLists []cdb.GetChoreListsByUserRow
}

func (v *View) InviteCreate(w io.Writer, r *http.Request, d InviteCreateView) error {
	return v.p.ExecuteTemplate(w, "invite_create.gohtml", d)
}

type InviteView struct {
	InviteID      string
	ChoreListName string
}

func (v *View) InvitePage(w io.Writer, r *http.Request, d InviteView) error {
	return v.p.ExecuteTemplate(w, "invite.gohtml", d)
}

type InviteAcceptView struct {
	InviteID      string
	ChoreListName string
	InviterName   string
	ExistingUser  bool
}

func (v *View) InviteAcceptPage(w io.Writer, r *http.Request, d InviteAcceptView) error {
	return v.p.ExecuteTemplate(w, "invite_accept.page.gohtml", d)
}

func (v *View) ChoreElement(w io.Writer, r *http.Request, d *Chore) error {
	return v.p.ExecuteTemplate(w, "chore-element.gohtml", d)
}

func (v *View) LoginPage(w io.Writer, r *http.Request) error {
	return v.p.ExecuteTemplate(w, "login.page.gohtml", nil)
}
