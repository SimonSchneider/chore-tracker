package core

import (
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/templ"
	"io"
	"net/http"
	"time"
)

type RequestDetails struct {
	req *http.Request
}

func (r *RequestDetails) CurrPath() string {
	return r.req.URL.RequestURI()
}

func (r *RequestDetails) PrevPath() string {
	return r.req.URL.Query().Get("prev")
}

type HtmlTemplateProvider struct {
	templ.TemplateProvider
}

func (p *HtmlTemplateProvider) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	if rw, ok := w.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}
	return p.TemplateProvider.ExecuteTemplate(w, name, data)
}

type View struct {
	p *HtmlTemplateProvider
}

func NewView(p templ.TemplateProvider) *View {
	return &View{p: &HtmlTemplateProvider{TemplateProvider: p}}
}

type ChoreListsView struct {
	RequestDetails *RequestDetails
	ChoreLists     []cdb.GetChoreListsByUserRow
}

func (v *View) ChoreListsPage(w http.ResponseWriter, r *http.Request, d ChoreListsView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "chore_lists.page.gohtml", d)
}

func (v *View) ChoreListNewPage(w http.ResponseWriter, r *http.Request) error {
	return v.p.ExecuteTemplate(w, "chore_list_edit.page.gohtml", ChoreListEditView{
		RequestDetails: &RequestDetails{req: r},
		List:           cdb.ChoreList{},
	})
}

type ChoreListEditView struct {
	*RequestDetails
	List    cdb.ChoreList
	Members []cdb.GetChoreListMembersRow
	Invites []cdb.Invitation
}

func (c ChoreListEditView) IsEdit() bool {
	return c.List.ID != ""
}

func (v *View) ChoreListEditPage(w http.ResponseWriter, r *http.Request, d ChoreListEditView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "chore_list_edit.page.gohtml", d)
}

type ChoreListView struct {
	*RequestDetails
	List    cdb.ChoreList
	Weekday time.Weekday
	Chores  *ListView
}

func (v *View) ChoreListPage(w http.ResponseWriter, r *http.Request, d ChoreListView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "chore_list.page.gohtml", d)
}

func (v *View) ChoreModal(w http.ResponseWriter, r *http.Request, d *Chore) error {
	return v.p.ExecuteTemplate(w, "chore-modal.gohtml", d)
}

type ChoreEditView struct {
	*RequestDetails
	Chore Chore
}

func (c ChoreEditView) IsEdit() bool {
	return c.Chore.ID != ""
}

func (v *View) ChoreEditPage(w http.ResponseWriter, r *http.Request, d ChoreEditView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "chore_edit.page.gohtml", d)
}

func (v *View) ChoreCreatePage(w http.ResponseWriter, r *http.Request, d ChoreEditView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "chore_edit.page.gohtml", d)
}

type SettingsView struct {
	*RequestDetails
	UserID         string
	Usernames      []string
	ChoreLists     []cdb.GetChoreListsByUserRow
	CreatedInvites []cdb.GetInvitationsByCreatorRow
}

func (v *View) SettingsPage(w http.ResponseWriter, r *http.Request, d SettingsView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "settings.page.gohtml", d)
}

type InviteCreateView struct {
	ChoreLists []cdb.GetChoreListsByUserRow
}

func (v *View) InviteCreate(w http.ResponseWriter, r *http.Request, d InviteCreateView) error {
	return v.p.ExecuteTemplate(w, "invite_create.gohtml", d)
}

type InviteView struct {
	InviteID      string
	ChoreListName string
}

func (v *View) InvitePage(w http.ResponseWriter, r *http.Request, d InviteView) error {
	return v.p.ExecuteTemplate(w, "invite.gohtml", d)
}

type InviteAcceptView struct {
	InviteID      string
	ChoreListName string
	InviterName   string
	ExistingUser  bool
}

func (v *View) InviteAcceptPage(w http.ResponseWriter, r *http.Request, d InviteAcceptView) error {
	return v.p.ExecuteTemplate(w, "invite_accept.page.gohtml", d)
}

func (v *View) LoginPage(w http.ResponseWriter, r *http.Request) error {
	return v.p.ExecuteTemplate(w, "login.page.gohtml", nil)
}
