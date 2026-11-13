package api

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

//go:embed merch_migrations/001_merch.sql
var merchMigration string

type merchError struct {
	status  int
	message string
}

func (e merchError) Error() string { return e.message }

func merchInvalid(message string) error  { return merchError{http.StatusBadRequest, message} }
func merchConflict(message string) error { return merchError{http.StatusConflict, message} }

func (rt *_router) writeMerchError(w http.ResponseWriter, err error) {
	var problem merchError
	switch {
	case errors.As(err, &problem):
		_ = writeJSONMessage(w, problem.status, problem.message)
	case errors.Is(err, sql.ErrNoRows):
		_ = writeJSONMessage(w, http.StatusNotFound, "Risorsa non trovata")
	default:
		rt.baseLogger.WithError(err).Error("merchandising operation failed")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Operazione non riuscita")
	}
}

type merchQuerier interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type merchScanner interface{ Scan(...interface{}) error }

func (rt *_router) merchWrite(ctx context.Context, fn func(*sql.Tx) error) error {
	conn, err := rt.db.SQLConn().Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Take SQLite's writer lock before any reads. This also serializes checkout,
	// catalog changes and expiry across independent server processes.
	if _, err = tx.ExecContext(ctx, "UPDATE merch_settings SET config = config WHERE organization_id = 0"); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func merchID() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

type merchSettings struct {
	Enabled              bool   `json:"enabled"`
	Visible              bool   `json:"visible"`
	OnlineEnabled        bool   `json:"online_enabled"`
	OfflineEnabled       bool   `json:"offline_enabled"`
	DefaultMode          string `json:"default_mode"`
	ReserveOffline       bool   `json:"reserve_offline"`
	PickupEnabled        bool   `json:"pickup_enabled"`
	ShippingEnabled      bool   `json:"shipping_enabled"`
	ShippingCents        int64  `json:"shipping_cents"`
	ReservationHours     int    `json:"reservation_hours"`
	ContactRequirement   string `json:"contact_requirement"`
	PickupPoint          string `json:"pickup_point"`
	DeliveryInstructions string `json:"delivery_instructions"`
	PaymentMethods       string `json:"payment_methods"`
	PaymentInstructions  string `json:"payment_instructions"`
	AvailabilityText     string `json:"availability_text"`
	ContactEmail         string `json:"contact_email"`
	CancellationPolicy   string `json:"cancellation_policy"`
	SellerInfo           string `json:"seller_info"`
	ConfirmationText     string `json:"confirmation_text"`
	Notes                string `json:"notes"`
	SMSNotifications     bool   `json:"sms_notifications"`
	StripeAccountID      string `json:"stripe_account_id,omitempty"`
	StripeStatus         string `json:"stripe_status,omitempty"`
}

func defaultMerchSettings() merchSettings {
	return merchSettings{Visible: true, OfflineEnabled: true, DefaultMode: "offline", ReserveOffline: true, PickupEnabled: true, ReservationHours: 72, ContactRequirement: "either"}
}

func loadMerchSettings(ctx context.Context, q merchQuerier, org int) (merchSettings, error) {
	s := defaultMerchSettings()
	var raw string
	err := q.QueryRowContext(ctx, "SELECT config FROM merch_settings WHERE organization_id=?", org).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	err = json.Unmarshal([]byte(raw), &s)
	return s, err
}

func validateMerchSettings(s *merchSettings) error {
	for _, value := range []*string{&s.PickupPoint, &s.DeliveryInstructions, &s.PaymentMethods, &s.PaymentInstructions, &s.AvailabilityText, &s.ContactEmail, &s.CancellationPolicy, &s.SellerInfo, &s.ConfirmationText, &s.Notes} {
		*value = strings.TrimSpace(*value)
		if len(*value) > 2000 {
			return merchInvalid("Testo impostazioni troppo lungo")
		}
	}
	if s.ContactEmail != "" && !validMerchEmail(s.ContactEmail) {
		return merchInvalid("Email di contatto non valida")
	}
	if s.ReservationHours < 1 || s.ReservationHours > 720 || s.ShippingCents < 0 || s.ShippingCents > 100000 {
		return merchInvalid("Durata della prenotazione o costo di spedizione non valido")
	}
	if s.ContactRequirement != "email" && s.ContactRequirement != "phone" && s.ContactRequirement != "either" {
		return merchInvalid("Requisito di contatto non valido")
	}
	if s.DefaultMode != "online" && s.DefaultMode != "offline" {
		return merchInvalid("Modalità predefinita non valida")
	}
	if s.Enabled && (!s.OnlineEnabled && !s.OfflineEnabled || !s.PickupEnabled && !s.ShippingEnabled) {
		return merchInvalid("Attiva almeno una modalità di vendita e di consegna")
	}
	if s.DefaultMode == "online" && !s.OnlineEnabled && s.OfflineEnabled {
		s.DefaultMode = "offline"
	}
	if s.DefaultMode == "offline" && !s.OfflineEnabled && s.OnlineEnabled {
		s.DefaultMode = "online"
	}
	s.StripeAccountID, s.StripeStatus = "", ""
	return nil
}

func validMerchEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s && len(s) <= 254
}

type merchMedia struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type merchVariant struct {
	ID         string `json:"id"`
	Size       string `json:"size"`
	Color      string `json:"color"`
	SKU        string `json:"sku"`
	PriceCents *int64 `json:"price_cents"`
	Stock      int    `json:"stock"`
	Available  int    `json:"available"`
	OnDemand   bool   `json:"on_demand"`
}

type merchProduct struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	ShortDescription string         `json:"short_description"`
	Description      string         `json:"description"`
	Category         string         `json:"category"`
	PriceCents       int64          `json:"price_cents"`
	PromoPriceCents  *int64         `json:"promo_price_cents"`
	SKU              string         `json:"sku"`
	Stock            int            `json:"stock"`
	Available        int            `json:"available"`
	MaxQuantity      int            `json:"max_quantity"`
	Published        bool           `json:"published"`
	Position         int            `json:"position"`
	Featured         bool           `json:"featured"`
	OnDemand         bool           `json:"on_demand"`
	SaleMode         string         `json:"sale_mode"`
	DigitalAssetID   string         `json:"digital_asset_id"`
	ShippingAllowed  bool           `json:"shipping_allowed"`
	Version          int            `json:"version"`
	OnlineEnabled    bool           `json:"online_enabled"`
	OfflineEnabled   bool           `json:"offline_enabled"`
	Variants         []merchVariant `json:"variants"`
	Media            []merchMedia   `json:"media"`
}

func merchModes(s merchSettings, saleMode string) (online, offline bool) {
	return s.OnlineEnabled && saleMode != "offline", s.OfflineEnabled && saleMode != "online"
}

func merchMediaURL(slug, id string) string {
	return "/api/v1/clubs/" + url.PathEscape(slug) + "/merch/media/" + id
}

const merchHoldSum = `SELECT COALESCE(SUM(i.quantity),0) FROM merch_order_items i
	JOIN merch_orders o ON o.organization_id=i.organization_id AND o.id=i.order_id
	WHERE i.organization_id=? AND i.product_id=? AND i.variant_id=? AND i.tracked=1 AND o.stock_status='HELD'`

const merchProductColumns = `id,name,short_description,description,category,price_cents,promo_price_cents,sku,stock,max_quantity,published,position,featured,on_demand,sale_mode,digital_asset_id,shipping_allowed,version`

func scanMerchProduct(row merchScanner) (p merchProduct, err error) {
	err = row.Scan(&p.ID, &p.Name, &p.ShortDescription, &p.Description, &p.Category, &p.PriceCents, &p.PromoPriceCents, &p.SKU, &p.Stock, &p.MaxQuantity, &p.Published, &p.Position, &p.Featured, &p.OnDemand, &p.SaleMode, &p.DigitalAssetID, &p.ShippingAllowed, &p.Version)
	return
}

func queryMerchProducts(ctx context.Context, q merchQuerier, org int, slug string, settings merchSettings, public bool) ([]merchProduct, error) {
	query := "SELECT " + merchProductColumns + " FROM merch_products WHERE organization_id=? AND archived=0"
	if public {
		query += " AND published=1"
	}
	rows, err := q.QueryContext(ctx, query+" ORDER BY featured DESC,position,name,id", org)
	if err != nil {
		return nil, err
	}
	products := []merchProduct{}
	for rows.Next() {
		p, err := scanMerchProduct(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		p.OnlineEnabled, p.OfflineEnabled = merchModes(settings, p.SaleMode)
		if !public || p.OnlineEnabled || p.OfflineEnabled {
			products = append(products, p)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range products {
		p := &products[i]
		if err := loadMerchProductDetails(ctx, q, org, slug, p); err != nil {
			return nil, err
		}
		if public {
			p.Stock = p.Available
			for j := range p.Variants {
				p.Variants[j].Stock = p.Variants[j].Available
			}
		}
	}
	return products, nil
}

func loadMerchProductDetails(ctx context.Context, q merchQuerier, org int, slug string, p *merchProduct) error {
	var held int
	if err := q.QueryRowContext(ctx, merchHoldSum, org, p.ID, "").Scan(&held); err != nil {
		return err
	}
	p.Available = p.Stock - held
	rows, err := q.QueryContext(ctx, `SELECT v.id,v.size,v.color,v.sku,v.price_cents,v.stock,v.on_demand,
		v.stock - (`+strings.ReplaceAll(merchHoldSum, "i.organization_id=? AND i.product_id=? AND i.variant_id=?", "i.organization_id=v.organization_id AND i.product_id=v.product_id AND i.variant_id=v.id")+`)
		FROM merch_product_variants v WHERE v.organization_id=? AND v.product_id=? AND v.active=1 ORDER BY v.position,v.id`, org, p.ID)
	if err != nil {
		return err
	}
	p.Variants = []merchVariant{}
	for rows.Next() {
		var v merchVariant
		if err := rows.Scan(&v.ID, &v.Size, &v.Color, &v.SKU, &v.PriceCents, &v.Stock, &v.OnDemand, &v.Available); err != nil {
			rows.Close()
			return err
		}
		p.Variants = append(p.Variants, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	rows, err = q.QueryContext(ctx, "SELECT media_id FROM merch_product_media WHERE organization_id=? AND product_id=? ORDER BY position", org, p.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	p.Media = []merchMedia{}
	for rows.Next() {
		var m merchMedia
		if err := rows.Scan(&m.ID); err != nil {
			return err
		}
		m.URL = merchMediaURL(slug, m.ID)
		p.Media = append(p.Media, m)
	}
	return rows.Err()
}

func validateMerchProduct(p *merchProduct) error {
	for _, field := range []struct {
		text *string
		max  int
	}{
		{&p.Name, 160}, {&p.ShortDescription, 500}, {&p.Description, 8000}, {&p.Category, 100}, {&p.SKU, 100}, {&p.DigitalAssetID, 160},
	} {
		*field.text = strings.TrimSpace(*field.text)
		if len(*field.text) > field.max {
			return merchInvalid("Campo prodotto troppo lungo")
		}
	}
	if p.SaleMode == "" {
		p.SaleMode = "inherit"
	}
	switch p.SaleMode {
	case "inherit", "online", "offline", "both":
	default:
		return merchInvalid("Modalità prodotto non valida")
	}
	if p.Name == "" || p.PriceCents <= 0 || p.PriceCents > 10000000 || p.Stock < 0 || p.Stock > 1000000 || p.MaxQuantity < 1 || p.MaxQuantity > 99 || p.Position < 0 || p.Position > 1000000 {
		return merchInvalid("Nome, prezzo, disponibilità, posizione o quantità massima non validi")
	}
	if p.PromoPriceCents != nil && (*p.PromoPriceCents <= 0 || *p.PromoPriceCents >= p.PriceCents) {
		return merchInvalid("Il prezzo promozionale deve essere inferiore al prezzo di listino")
	}
	if len(p.Variants) > 100 || len(p.Media) > 12 {
		return merchInvalid("Numero di varianti o immagini superiore al limite")
	}
	variants, variantIDs, mediaIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for i := range p.Variants {
		v := &p.Variants[i]
		v.Size, v.Color, v.SKU = strings.TrimSpace(v.Size), strings.TrimSpace(v.Color), strings.TrimSpace(v.SKU)
		if v.Size == "" && v.Color == "" || len(v.Size) > 80 || len(v.Color) > 80 || len(v.SKU) > 100 || v.Stock < 0 || v.Stock > 1000000 || (v.PriceCents != nil && (*v.PriceCents <= 0 || *v.PriceCents > 10000000)) {
			return merchInvalid("Variante non valida")
		}
		key := strings.ToLower(v.Size) + "\x00" + strings.ToLower(v.Color)
		if variants[key] || (v.ID != "" && variantIDs[v.ID]) {
			return merchInvalid("Variante duplicata")
		}
		variants[key], variantIDs[v.ID] = true, true
	}
	for _, m := range p.Media {
		if m.ID == "" || mediaIDs[m.ID] {
			return merchInvalid("Immagine non valida o duplicata")
		}
		mediaIDs[m.ID] = true
	}
	return nil
}

func (rt *_router) saveMerchProduct(ctx context.Context, org int, slug, id string, p *merchProduct) error {
	if err := validateMerchProduct(p); err != nil {
		return err
	}
	return rt.merchWrite(ctx, func(tx *sql.Tx) error {
		if id == "" {
			var err error
			id, err = merchID()
			if err != nil {
				return err
			}
			p.Version = 1
			_, err = tx.ExecContext(ctx, `INSERT INTO merch_products (organization_id,id,name,short_description,description,category,price_cents,promo_price_cents,sku,stock,max_quantity,published,position,featured,on_demand,sale_mode,digital_asset_id,shipping_allowed)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, org, id, p.Name, p.ShortDescription, p.Description, p.Category, p.PriceCents, p.PromoPriceCents, p.SKU, p.Stock, p.MaxQuantity, p.Published, p.Position, p.Featured, p.OnDemand, p.SaleMode, p.DigitalAssetID, p.ShippingAllowed)
			if err != nil {
				return err
			}
		} else {
			var held, version int
			if err := tx.QueryRowContext(ctx, "SELECT version FROM merch_products WHERE organization_id=? AND id=? AND archived=0", org, id).Scan(&version); err != nil {
				return err
			}
			if p.Version != version {
				return merchConflict("Il prodotto è cambiato. Ricarica prima di salvare.")
			}
			if err := tx.QueryRowContext(ctx, merchHoldSum, org, id, "").Scan(&held); err != nil {
				return err
			}
			if p.Stock < held {
				return merchConflict("La quantità è inferiore allo stock riservato")
			}
			_, err := tx.ExecContext(ctx, `UPDATE merch_products SET name=?,short_description=?,description=?,category=?,price_cents=?,promo_price_cents=?,sku=?,stock=?,max_quantity=?,published=?,position=?,featured=?,on_demand=?,sale_mode=?,digital_asset_id=?,shipping_allowed=?,version=version+1 WHERE organization_id=? AND id=?`,
				p.Name, p.ShortDescription, p.Description, p.Category, p.PriceCents, p.PromoPriceCents, p.SKU, p.Stock, p.MaxQuantity, p.Published, p.Position, p.Featured, p.OnDemand, p.SaleMode, p.DigitalAssetID, p.ShippingAllowed, org, id)
			if err != nil {
				return err
			}
			p.Version++
		}
		p.ID = id
		if _, err := tx.ExecContext(ctx, "UPDATE merch_product_variants SET active=0 WHERE organization_id=? AND product_id=?", org, id); err != nil {
			return err
		}
		for i := range p.Variants {
			v := &p.Variants[i]
			if v.ID == "" {
				var err error
				v.ID, err = merchID()
				if err != nil {
					return err
				}
				_, err = tx.ExecContext(ctx, `INSERT INTO merch_product_variants (organization_id,product_id,id,size,color,sku,price_cents,stock,on_demand,position,active) VALUES (?,?,?,?,?,?,?,?,?,?,1)`, org, id, v.ID, v.Size, v.Color, v.SKU, v.PriceCents, v.Stock, v.OnDemand, i)
				if err != nil {
					return err
				}
				continue
			}
			var held int
			if err := tx.QueryRowContext(ctx, merchHoldSum, org, id, v.ID).Scan(&held); err != nil {
				return err
			}
			if v.Stock < held {
				return merchConflict("Lo stock della variante è inferiore alla quantità riservata")
			}
			res, err := tx.ExecContext(ctx, `UPDATE merch_product_variants SET size=?,color=?,sku=?,price_cents=?,stock=?,on_demand=?,position=?,active=1 WHERE organization_id=? AND product_id=? AND id=?`, v.Size, v.Color, v.SKU, v.PriceCents, v.Stock, v.OnDemand, i, org, id, v.ID)
			if err != nil {
				return err
			}
			n, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if n != 1 {
				return merchInvalid("La variante non appartiene al prodotto")
			}
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM merch_product_media WHERE organization_id=? AND product_id=?", org, id); err != nil {
			return err
		}
		for i, m := range p.Media {
			res, err := tx.ExecContext(ctx, `INSERT INTO merch_product_media (organization_id,product_id,media_id,position) SELECT organization_id,?,id,? FROM merch_media WHERE organization_id=? AND id=?`, id, i, org, m.ID)
			if err != nil {
				return err
			}
			n, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if n != 1 {
				return merchInvalid("L'immagine non appartiene alla società")
			}
		}
		return loadMerchProductDetails(ctx, tx, org, slug, p)
	})
}

func (rt *_router) archiveMerchProduct(ctx context.Context, org int, id string) error {
	return rt.merchWrite(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE merch_products SET archived=1,published=0,version=version+1 WHERE organization_id=? AND id=? AND archived=0", org, id)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err == nil && n != 1 {
			return sql.ErrNoRows
		}
		return err
	})
}

func merchRate(ctx context.Context, tx *sql.Tx, org int, action, key string, period time.Duration, limit int) (bool, error) {
	now := time.Now().Unix()
	bucket := now - now%int64(period/time.Second)
	res, err := tx.ExecContext(ctx, `INSERT INTO merch_rate_limits(organization_id,action,actor_hash,time_bucket,count) VALUES(?,?,?,?,1)
		ON CONFLICT(organization_id,action,actor_hash,time_bucket) DO UPDATE SET count=count+1 WHERE count < ?`, org, action, merchDigest(key), bucket, limit)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (rt *_router) ensureMerchMigration() error {
	ctx := context.Background()
	tx, err := rt.db.SQLConn().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(merchMigration); err != nil {
		return fmt.Errorf("merchandising schema: %w", err)
	}
	if _, err := tx.Exec(merchMatchMigration); err != nil {
		return fmt.Errorf("merchandising matchday schema: %w", err)
	}
	if err := rt.ensureMerchNotifications(ctx, tx); err != nil {
		return fmt.Errorf("merchandising notifications schema: %w", err)
	}
	return tx.Commit()
}
