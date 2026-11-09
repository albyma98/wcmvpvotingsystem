package api

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
	"github.com/mattn/go-sqlite3"
)

//go:embed crm_migrations/001_crm.sql
var crmMigration string

var crmRome = func() *time.Location {
	location, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		panic(err)
	}
	return location
}()

const crmPath = "/admin/master/crm"
const crmDate = "2006-01-02"
const crmLocalTime = "2006-01-02T15:04"

type crmCompany struct {
	ID        int64  `json:"id"`
	Revision  int    `json:"revision"`
	Name      string `json:"name"`
	ABXOrgID  int64  `json:"abxOrgId"`
	OwnerID   int64  `json:"ownerId"`
	Archived  bool   `json:"archived"`
	UpdatedAt string `json:"updatedAt"`
	crmCompanyDetails
}

type crmCompanyDetails struct {
	SponsoredName string `json:"sponsoredName"`
	Sport         string `json:"sport"`
	Category      string `json:"category"`
	Season        string `json:"season"`
	City          string `json:"city"`
	Website       string `json:"website"`
	Priority      string `json:"priority"`
	Source        string `json:"source"`
	Status        string `json:"status"`
	CheckIn       string `json:"checkIn"`
	CadenceDays   int    `json:"cadenceDays"`
	Health        string `json:"health"`
	HealthReason  string `json:"healthReason"`
}

type crmRecord struct {
	ID            int64  `json:"id"`
	CompanyID     int64  `json:"companyId"`
	Kind          string `json:"kind"`
	Revision      int    `json:"revision"`
	ContactID     int64  `json:"contactId"`
	OpportunityID int64  `json:"opportunityId"`
	ContractID    int64  `json:"contractId"`
	OwnerID       int64  `json:"ownerId"`
	Archived      bool   `json:"archived"`
	AuthorID      int64  `json:"authorId"`
	AuthorName    string `json:"authorName"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	crmRecordDetails
}

type crmRecordDetails struct {
	Title        string   `json:"title"`
	Status       string   `json:"status,omitempty"`
	Type         string   `json:"type,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	Result       string   `json:"result,omitempty"`
	Date         string   `json:"date,omitempty"`
	Due          string   `json:"due,omitempty"`
	Start        string   `json:"start,omitempty"`
	End          string   `json:"end,omitempty"`
	Season       string   `json:"season,omitempty"`
	Priority     string   `json:"priority,omitempty"`
	Role         string   `json:"role,omitempty"`
	Email        string   `json:"email,omitempty"`
	Phone        string   `json:"phone,omitempty"`
	Channel      string   `json:"channel,omitempty"`
	Source       string   `json:"source,omitempty"`
	NextStep     string   `json:"nextStep,omitempty"`
	Obstacle     string   `json:"obstacle,omitempty"`
	Goal         string   `json:"goal,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	ResumeOn     string   `json:"resumeOn,omitempty"`
	Estimated    *int64   `json:"estimated,omitempty"`
	Amount       *int64   `json:"amount,omitempty"`
	InKind       string   `json:"inKind,omitempty"`
	PaymentTerms string   `json:"paymentTerms,omitempty"`
	Planned      *int     `json:"planned,omitempty"`
	Played       *int     `json:"played,omitempty"`
	EventIDs     []int64  `json:"eventIds,omitempty"`
	ReportURL    string   `json:"reportUrl,omitempty"`
	Metric       string   `json:"metric,omitempty"`
	Value        *float64 `json:"value,omitempty"`
	Denominator  *float64 `json:"denominator,omitempty"`
	EventID      int64    `json:"eventId,omitempty"`
	CompletedAt  string   `json:"completedAt,omitempty"`
}

type crmProblem string

func (e crmProblem) Error() string { return string(e) }

func (rt *_router) ensureCRMTables() error {
	db := rt.db.SQLConn()
	if db == nil {
		return errors.New("CRM storage is unavailable")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(crmMigration); err != nil {
		return err
	}
	return tx.Commit()
}

type crmHandler func(*http.Request, reqcontext.RequestContext) (interface{}, error)

func (rt *_router) crmEndpoint(handler crmHandler) http.HandlerFunc {
	return rt.wrapAdmin(func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
		w.Header().Set("Cache-Control", "no-store")
		if !rt.ensureSuperAdmin(w, ctx) {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 512*1024)
		value, err := handler(r, ctx)
		if err == nil {
			_ = writeJSON(w, http.StatusOK, value)
			return
		}
		var problem crmProblem
		var sqliteErr sqlite3.Error
		switch {
		case errors.As(err, &problem):
			reportError(w, http.StatusBadRequest, problem.Error())
		case errors.Is(err, sql.ErrNoRows):
			reportError(w, http.StatusNotFound, "Scheda o voce CRM non trovata.")
		case errors.Is(err, errCRMRevision):
			reportError(w, http.StatusConflict, "La voce è stata aggiornata. Ricarica la scheda prima di salvare.")
		case errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique:
			reportError(w, http.StatusConflict, "La voce esiste già oppure la società ABX è già collegata a un’altra scheda.")
		case errors.As(err, &sqliteErr) && (sqliteErr.Code == sqlite3.ErrBusy || sqliteErr.Code == sqlite3.ErrLocked):
			reportError(w, http.StatusConflict, "CRM in aggiornamento. Riprova tra poco.")
		default:
			ctx.Logger.WithError(err).Error("CRM request failed")
			reportError(w, http.StatusInternalServerError, "Impossibile completare l’operazione CRM.")
		}
	})
}

func registerCRMRoutes(rt *_router) {
	if err := rt.ensureCRMTables(); err != nil {
		rt.baseLogger.WithError(err).Error("cannot initialize CRM")
	}
	rt.router.Get(crmPath, rt.crmEndpoint(rt.crmHome))
	rt.router.Post(crmPath+"/companies", rt.crmEndpoint(rt.crmSaveCompany))
	rt.router.Post(crmPath+"/import", rt.crmEndpoint(rt.crmImport))
	rt.router.Get(crmPath+"/companies/{companyID}", rt.crmEndpoint(rt.crmCompanyDetail))
	rt.router.Put(crmPath+"/companies/{companyID}", rt.crmEndpoint(rt.crmSaveCompany))
	rt.router.Get(crmPath+"/companies/{companyID}/performance", rt.crmEndpoint(rt.crmPerformance))
	rt.router.Post(crmPath+"/companies/{companyID}/records", rt.crmEndpoint(rt.crmSaveRecord))
	rt.router.Put(crmPath+"/companies/{companyID}/records/{recordID}", rt.crmEndpoint(rt.crmSaveRecord))
	rt.router.Post(crmPath+"/companies/{companyID}/contracts/{recordID}/renewal", rt.crmEndpoint(rt.crmRenewal))
}

var errCRMRevision = errors.New("stale CRM revision")

func crmID(r *http.Request, key string) (int64, error) {
	value, err := strconv.ParseInt(chi.URLParam(r, key), 10, 64)
	if err != nil || value <= 0 {
		return 0, sql.ErrNoRows
	}
	return value, nil
}

func crmDecode(r *http.Request, value interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return crmProblem("Controlla i campi della richiesta.")
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return crmProblem("Controlla i campi della richiesta.")
	}
	return nil
}

func crmNow() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func crmToday() string { return time.Now().In(crmRome).Format(crmDate) }

func crmNull(id int64) interface{} {
	if id == 0 {
		return nil
	}
	return id
}

type crmQuery interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}

type crmScanner interface {
	Scan(...interface{}) error
}

func crmReadRows(rows *sql.Rows, err error, scan func(crmScanner) error) error {
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

const crmCompanyColumns = "id,revision,name,COALESCE(abx_org_id,0),COALESCE(owner_id,0),archived,updated_at,details"
const crmRecordColumns = "id,company_id,kind,revision,COALESCE(contact_id,0),COALESCE(opportunity_id,0),COALESCE(contract_id,0),COALESCE(owner_id,0),archived,COALESCE(author_id,0),author_name,created_at,updated_at,details"

func crmScanCompany(row crmScanner) (company crmCompany, err error) {
	var details string
	err = row.Scan(&company.ID, &company.Revision, &company.Name, &company.ABXOrgID, &company.OwnerID, &company.Archived, &company.UpdatedAt, &details)
	if err == nil {
		err = json.Unmarshal([]byte(details), &company.crmCompanyDetails)
	}
	return
}

func crmScanRecord(row crmScanner) (record crmRecord, err error) {
	var details string
	err = row.Scan(&record.ID, &record.CompanyID, &record.Kind, &record.Revision, &record.ContactID, &record.OpportunityID, &record.ContractID, &record.OwnerID, &record.Archived, &record.AuthorID, &record.AuthorName, &record.CreatedAt, &record.UpdatedAt, &details)
	if err == nil {
		err = json.Unmarshal([]byte(details), &record.crmRecordDetails)
	}
	return
}

func crmReadCompany(q crmQuery, id int64) (crmCompany, error) {
	return crmScanCompany(q.QueryRow("SELECT "+crmCompanyColumns+" FROM crm_companies WHERE id=?", id))
}

func crmReadRecord(q crmQuery, companyID, id int64) (crmRecord, error) {
	return crmScanRecord(q.QueryRow("SELECT "+crmRecordColumns+" FROM crm_records WHERE company_id=? AND id=?", companyID, id))
}

func crmListRecords(q crmQuery, condition string, args ...interface{}) ([]crmRecord, error) {
	rows, err := q.Query("SELECT "+crmRecordColumns+" FROM crm_records "+condition+" ORDER BY id DESC", args...)
	out := []crmRecord{}
	err = crmReadRows(rows, err, func(row crmScanner) error {
		value, err := crmScanRecord(row)
		out = append(out, value)
		return err
	})
	return out, err
}

type crmOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func crmOptions(q crmQuery, query string) ([]crmOption, error) {
	rows, err := q.Query(query)
	out := []crmOption{}
	err = crmReadRows(rows, err, func(row crmScanner) error {
		var value crmOption
		err := row.Scan(&value.ID, &value.Name)
		out = append(out, value)
		return err
	})
	return out, err
}

func (rt *_router) crmHome(_ *http.Request, _ reqcontext.RequestContext) (interface{}, error) {
	db := rt.db.SQLConn()
	rows, err := db.Query("SELECT " + crmCompanyColumns + " FROM crm_companies ORDER BY name COLLATE NOCASE,id")
	companies := []crmCompany{}
	err = crmReadRows(rows, err, func(row crmScanner) error {
		value, err := crmScanCompany(row)
		companies = append(companies, value)
		return err
	})
	if err != nil {
		return nil, err
	}
	records, err := crmListRecords(db, "WHERE archived=0 AND kind IN ('contact','task','opportunity','pilot','checklist','issue','contract')")
	if err != nil {
		return nil, err
	}
	operators, err := crmOptions(db, "SELECT id,username FROM admins WHERE lower(role)='superadmin' ORDER BY username")
	if err != nil {
		return nil, err
	}
	organizations, err := crmOptions(db, "SELECT id,name FROM organizations WHERE substr(slug,1,2)<>'__' ORDER BY name")
	return map[string]interface{}{"today": crmToday(), "companies": companies, "records": records, "operators": operators, "organizations": organizations}, err
}

func (rt *_router) crmCompanyDetail(r *http.Request, _ reqcontext.RequestContext) (interface{}, error) {
	id, err := crmID(r, "companyID")
	if err != nil {
		return nil, err
	}
	db := rt.db.SQLConn()
	company, err := crmReadCompany(db, id)
	if err != nil {
		return nil, err
	}
	records, err := crmListRecords(db, "WHERE company_id=?", id)
	return map[string]interface{}{"company": company, "records": records}, err
}

func (rt *_router) crmWrite(ctx context.Context, fn func(*sql.Tx) (interface{}, error)) (interface{}, error) {
	conn, err := rt.db.SQLConn().Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Acquire the SQLite writer lock before reading versions or related rows.
	if _, err := tx.Exec("UPDATE crm_companies SET revision=revision WHERE 0"); err != nil {
		return nil, err
	}
	result, err := fn(tx)
	if err != nil {
		return nil, err
	}
	return result, tx.Commit()
}

func crmIs(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

func crmValidateDates(dates ...string) error {
	for _, date := range dates {
		if date != "" {
			parsed, err := time.Parse(crmDate, date)
			if err != nil || parsed.Format(crmDate) != date {
				return crmProblem("Inserisci date valide.")
			}
		}
	}
	return nil
}

func crmValidateOwner(q crmQuery, id int64) error {
	if id == 0 {
		return nil
	}
	var present int
	err := q.QueryRow("SELECT EXISTS(SELECT 1 FROM admins WHERE id=? AND lower(role)='superadmin')", id).Scan(&present)
	if err == nil && present == 0 {
		return crmProblem("Seleziona un operatore master valido.")
	}
	return err
}

func crmValidateCompany(q crmQuery, value *crmCompany) error {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" || len(value.Name) > 250 {
		return crmProblem("Inserisci il nome della società (massimo 250 caratteri).")
	}
	if value.Priority == "" {
		value.Priority = "normal"
	}
	if value.Status == "" {
		value.Status = "lead"
	}
	if !crmIs(value.Priority, "low", "normal", "high", "urgent") || !crmIs(value.Status, "lead", "pilot", "onboarding", "active", "former") {
		return crmProblem("Stato o priorità della società non validi.")
	}
	if value.CadenceDays < 0 || value.CadenceDays > 365 {
		return crmProblem("La cadenza deve essere compresa tra 1 e 365 giorni, oppure lasciata vuota.")
	}
	if !crmIs(value.Health, "", "regular", "watch", "risk", "insufficient") || (value.Health != "" && strings.TrimSpace(value.HealthReason) == "") {
		return crmProblem("Indica la motivazione della valutazione manuale.")
	}
	if err := crmValidateDates(value.CheckIn); err != nil {
		return err
	}
	if err := crmValidateURL(value.Website); err != nil {
		return err
	}
	if value.ABXOrgID != 0 {
		var present int
		if err := q.QueryRow("SELECT EXISTS(SELECT 1 FROM organizations WHERE id=? AND substr(slug,1,2)<>'__')", value.ABXOrgID).Scan(&present); err != nil {
			return err
		}
		if present == 0 {
			return crmProblem("Seleziona una società ABX esistente.")
		}
	}
	return crmValidateOwner(q, value.OwnerID)
}

func crmInsertCompany(tx *sql.Tx, value crmCompany, key interface{}, now string) (int64, error) {
	raw, err := json.Marshal(value.crmCompanyDetails)
	if err != nil {
		return 0, err
	}
	result, err := tx.Exec("INSERT INTO crm_companies(import_key,abx_org_id,owner_id,name,details,archived,updated_at) VALUES(?,?,?,?,?,?,?)",
		key, crmNull(value.ABXOrgID), crmNull(value.OwnerID), value.Name, string(raw), value.Archived, now)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (rt *_router) crmSaveCompany(r *http.Request, actor reqcontext.RequestContext) (interface{}, error) {
	var value crmCompany
	if err := crmDecode(r, &value); err != nil {
		return nil, err
	}
	var id int64
	if r.Method == http.MethodPut {
		var err error
		if id, err = crmID(r, "companyID"); err != nil {
			return nil, err
		}
	}
	return rt.crmWrite(r.Context(), func(tx *sql.Tx) (interface{}, error) {
		var before crmCompany
		if id != 0 {
			var err error
			before, err = crmReadCompany(tx, id)
			if err != nil {
				return nil, err
			}
			if before.Revision != value.Revision {
				return nil, errCRMRevision
			}
		}
		if err := crmValidateCompany(tx, &value); err != nil {
			return nil, err
		}
		now := crmNow()
		title := "Scheda società creata"
		note := ""
		if id == 0 {
			var err error
			id, err = crmInsertCompany(tx, value, nil, now)
			if err != nil {
				return nil, err
			}
		} else {
			raw, err := json.Marshal(value.crmCompanyDetails)
			if err != nil {
				return nil, err
			}
			if _, err := tx.Exec("UPDATE crm_companies SET abx_org_id=?,owner_id=?,name=?,details=?,archived=?,revision=revision+1,updated_at=? WHERE id=?",
				crmNull(value.ABXOrgID), crmNull(value.OwnerID), value.Name, string(raw), value.Archived, now, id); err != nil {
				return nil, err
			}
			title = "Scheda società aggiornata"
			var changes []string
			if before.Status != value.Status {
				changes = append(changes, fmt.Sprintf("Rapporto: %s → %s", crmStatusLabel(before.Status), crmStatusLabel(value.Status)))
			}
			if before.ABXOrgID != value.ABXOrgID {
				changes = append(changes, fmt.Sprintf("Collegamento ABX: %s → %s", crmOrganizationName(tx, before.ABXOrgID), crmOrganizationName(tx, value.ABXOrgID)))
			}
			if before.Health != value.Health || before.HealthReason != value.HealthReason {
				changes = append(changes, "Valutazione manuale: "+crmStatusLabel(value.Health)+" · "+value.HealthReason)
			}
			if before.Archived != value.Archived {
				if value.Archived {
					changes = append(changes, "Scheda archiviata")
				} else {
					changes = append(changes, "Scheda riattivata")
				}
			}
			note = strings.Join(changes, "\n")
		}
		if err := crmHistory(tx, id, 0, title, note, now, actor); err != nil {
			return nil, err
		}
		return crmReadCompany(tx, id)
	})
}

func crmOrganizationName(q crmQuery, id int64) string {
	if id == 0 {
		return "nessuna società"
	}
	name := fmt.Sprintf("#%d", id)
	_ = q.QueryRow("SELECT name FROM organizations WHERE id=?", id).Scan(&name)
	return name
}
