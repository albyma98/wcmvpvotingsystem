package api

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

type crmImportRow struct {
	companyKey string
	contactKey string
	company    crmCompany
	contact    crmRecord
}

func crmReadImport(input string) ([]crmImportRow, error) {
	input = strings.TrimPrefix(strings.TrimSpace(input), "\ufeff")
	if !utf8.ValidString(input) {
		return nil, crmProblem("Salva il CSV in UTF-8.")
	}
	reader := csv.NewReader(strings.NewReader(input))
	firstLine := strings.SplitN(input, "\n", 2)[0]
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		reader.Comma = ';'
	}
	header, err := reader.Read()
	if err != nil {
		return nil, crmProblem("Il CSV deve contenere un’intestazione.")
	}
	keys := map[string]int{}
	for index, name := range header {
		name = strings.ToLower(strings.TrimSpace(name))
		if _, exists := keys[name]; exists || name == "" {
			return nil, crmProblem("Le intestazioni CSV devono essere presenti e univoche.")
		}
		keys[name] = index
	}
	for _, required := range []string{"codice_societa", "societa"} {
		if _, ok := keys[required]; !ok {
			return nil, crmProblem("Il CSV richiede le colonne codice_societa e societa.")
		}
	}
	out := []crmImportRow{}
	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, crmProblem(fmt.Sprintf("CSV non valido alla riga %d.", line))
		}
		if line > 501 {
			return nil, crmProblem("Importa al massimo 500 righe alla volta.")
		}
		cell := func(column string) string {
			if index, ok := keys[column]; ok {
				return strings.TrimSpace(row[index])
			}
			return ""
		}
		entry := crmImportRow{
			companyKey: strings.ToLower(cell("codice_societa")),
			contactKey: strings.ToLower(cell("codice_referente")),
			company: crmCompany{Name: cell("societa"), crmCompanyDetails: crmCompanyDetails{
				SponsoredName: cell("nome_sponsorizzato"), Sport: cell("sport"), Category: cell("campionato"),
				Season: cell("stagione"), City: cell("citta"), Website: cell("sito"), Source: cell("fonte_societa"),
			}},
			contact: crmRecord{Kind: "contact", crmRecordDetails: crmRecordDetails{
				Title: cell("referente"), Role: cell("ruolo"), Email: cell("email"), Phone: cell("telefono"),
				Source: cell("fonte_referente"), Date: cell("verificato_il"), Type: cell("influenza"), Channel: cell("canale"),
			}},
		}
		if entry.companyKey == "" || len(entry.companyKey) > 200 {
			return nil, crmProblem(fmt.Sprintf("Indica un codice società stabile alla riga %d.", line))
		}
		if entry.contact.Title == "" && (entry.contactKey != "" || entry.contact.Email != "" || entry.contact.Phone != "" || entry.contact.Role != "") {
			return nil, crmProblem(fmt.Sprintf("Indica il nome del referente alla riga %d.", line))
		}
		if entry.contact.Title != "" {
			if entry.contactKey == "" {
				entry.contactKey = strings.ToLower(entry.contact.Email)
			}
			if entry.contactKey == "" || len(entry.contactKey) > 254 {
				return nil, crmProblem(fmt.Sprintf("Indica il codice referente o l’email alla riga %d.", line))
			}
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil, crmProblem("Il CSV non contiene righe da importare.")
	}
	return out, nil
}

func (rt *_router) crmImport(r *http.Request, actor reqcontext.RequestContext) (interface{}, error) {
	var request struct {
		CSV string `json:"csv"`
	}
	if err := crmDecode(r, &request); err != nil {
		return nil, err
	}
	entries, err := crmReadImport(request.CSV)
	if err != nil {
		return nil, err
	}
	return rt.crmWrite(r.Context(), func(tx *sql.Tx) (interface{}, error) {
		seen := map[int64]bool{}
		companyIDs := []int64{}
		companies, contacts := 0, 0
		now := crmNow()
		for index, entry := range entries {
			id, inserted, err := crmImportCompany(tx, entry, now, actor)
			if err != nil {
				return nil, crmImportProblem(err, index)
			}
			if inserted {
				companies++
			}
			if !seen[id] {
				companyIDs = append(companyIDs, id)
				seen[id] = true
			}
			if entry.contact.Title == "" {
				continue
			}
			inserted, err = crmImportContact(tx, id, entry, now, actor)
			if err != nil {
				return nil, crmImportProblem(err, index)
			}
			if inserted {
				contacts++
			}
		}
		return map[string]interface{}{"companyIds": companyIDs, "companiesAdded": companies, "contactsAdded": contacts}, nil
	})
}

func crmImportProblem(err error, index int) error {
	if _, ok := err.(crmProblem); ok {
		return crmProblem(fmt.Sprintf("Riga %d. %s", index+2, err))
	}
	return err
}

func crmImportCompany(tx *sql.Tx, entry crmImportRow, now string, actor reqcontext.RequestContext) (int64, bool, error) {
	var id int64
	err := tx.QueryRow("SELECT id FROM crm_companies WHERE import_key=?", entry.companyKey).Scan(&id)
	if err != sql.ErrNoRows {
		return id, false, err
	}
	if err := crmValidateCompany(tx, &entry.company); err != nil {
		return 0, false, err
	}
	id, err = crmInsertCompany(tx, entry.company, entry.companyKey, now)
	if err != nil {
		return 0, false, err
	}
	return id, true, crmHistory(tx, id, 0, "Società importata", "", now, actor)
}

func crmImportContact(tx *sql.Tx, id int64, entry crmImportRow, now string, actor reqcontext.RequestContext) (bool, error) {
	var exists int
	err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM crm_records WHERE company_id=? AND kind='contact' AND stable_key=?)", id, entry.contactKey).Scan(&exists)
	if err != nil || exists != 0 {
		return false, err
	}
	company, err := crmReadCompany(tx, id)
	if err != nil {
		return false, err
	}
	if company.Archived {
		return false, crmProblem("Riattiva la società prima di importare nuovi referenti.")
	}
	entry.contact.CompanyID = id
	if err := crmValidateRecord(tx, company, &entry.contact, crmRecord{}, now); err != nil {
		return false, err
	}
	if _, err := crmInsertRecord(tx, entry.contact, entry.contactKey, now, actor); err != nil {
		return false, err
	}
	return true, crmHistory(tx, id, 0, "Referente importato · "+entry.contact.Title, "", now, actor)
}
