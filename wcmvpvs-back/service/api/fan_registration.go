package api

import (
	"errors"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func fanRegistrationError(err error) (int, string, string) {
	switch {
	case errors.Is(err, database.ErrFanAgeRequired):
		return http.StatusBadRequest, "AGE_ELIGIBILITY_REQUIRED", "Per registrarti devi dichiarare di avere almeno 14 anni."
	case errors.Is(err, database.ErrFanTermsRequired):
		return http.StatusBadRequest, "TERMS_REQUIRED", "Accetta i Termini del servizio per creare il profilo."
	case errors.Is(err, database.ErrFanLegalVersion):
		return http.StatusConflict, "LEGAL_VERSION_CHANGED", "I documenti sono stati aggiornati. Ricarica la pagina e consulta i nuovi testi."
	case errors.Is(err, database.ErrFanVerificationRequired):
		return http.StatusUnauthorized, "PHONE_VERIFICATION_REQUIRED", "Verifica il numero con il codice SMS prima di creare il profilo."
	default:
		return 0, "", ""
	}
}
