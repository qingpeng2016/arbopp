package orderfillsync

import (
	"context"
	"math"
	"runtime/debug"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

const TaskName = "order_fill_sync"

const defaultBatch = 100

// Sync 按 orders.platform_order_id 回查各平台成交，补全 fill_qty / status / fee。
type Sync struct {
	orderRepo   persistentrepo.ArbBinaryOrderRepo
	accountRepo persistentrepo.AccountRepo
	poly        httprepo.PolymarketRepo
	predict     httprepo.PredictRepo
	sxbet       httprepo.SxbetRepo
}

func NewSync(
	orderRepo persistentrepo.ArbBinaryOrderRepo,
	accountRepo persistentrepo.AccountRepo,
	poly httprepo.PolymarketRepo,
	predict httprepo.PredictRepo,
	sxbet httprepo.SxbetRepo,
) *Sync {
	return &Sync{
		orderRepo: orderRepo, accountRepo: accountRepo,
		poly: poly, predict: predict, sxbet: sxbet,
	}
}

func (s *Sync) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "order_fill_sync-Panic",
				zap.Any("panic", r), zap.String("stack", string(debug.Stack())))
		}
	}()
	if s == nil || s.orderRepo == nil {
		notification.SendErrorLog(ctx, "order_fill_sync-skipped", zap.String("reason", "nil orderRepo"))
		return
	}
	start := time.Now()
	creds := s.loadCreds(ctx)
	list, err := s.orderRepo.ListForFillSync(ctx, defaultBatch, true)
	if err != nil {
		notification.SendErrorLog(ctx, "order_fill_sync-list", zap.Error(err))
		return
	}
	var okN, errN int
	for _, row := range list {
		fill, fee, status, qerr := s.queryFill(ctx, row, creds)
		if qerr != nil {
			errN++
			logger.InfoZ(ctx, "order_fill_sync-query-fail",
				zap.Uint("id", row.ID), zap.String("platform", row.Platform),
				zap.String("oid", row.PlatformOrderID), zap.Error(qerr))
			continue
		}
		status = statusFromFill(fill, row.Qty.InexactFloat64())
		if err := s.orderRepo.UpdateFill(ctx, row.ID, fill, fee, status, ""); err != nil {
			errN++
			notification.SendErrorLog(ctx, "order_fill_sync-update",
				zap.Uint("id", row.ID), zap.Error(err))
			continue
		}
		okN++
	}
	logger.InfoZ(ctx, "order_fill_sync-done",
		zap.Int("ok", okN), zap.Int("err", errN), zap.Int("batch", len(list)),
		zap.Duration("elapsed", time.Since(start)))
}

type activeCreds struct {
	poly    httpentity.PolymarketAccount
	polyOK  bool
	predict httpentity.PredictAccount
	predOK  bool
	sxbet   httpentity.SxbetAccount
	sxOK    bool
}

func (s *Sync) loadCreds(ctx context.Context) activeCreds {
	var c activeCreds
	if s.accountRepo == nil {
		return c
	}
	_, list, err := s.accountRepo.List(ctx, nil, map[string]interface{}{"status = ?": "ACTIVE"}, "id asc", "")
	if err != nil {
		return c
	}
	for i := range list {
		a := list[i]
		switch strings.ToUpper(strings.TrimSpace(a.Platform)) {
		case appcommon.PlatformPolymarket:
			if !c.polyOK {
				if acc, ok := polyAccount(a); ok {
					c.poly, c.polyOK = acc, true
				}
			}
		case appcommon.PlatformPredict:
			if !c.predOK {
				if acc, ok := predictAccount(a); ok {
					c.predict, c.predOK = acc, true
				}
			}
		case appcommon.PlatformSxbet:
			if !c.sxOK {
				if acc, ok := sxbetAccount(a); ok {
					c.sxbet, c.sxOK = acc, true
				}
			}
		}
	}
	return c
}

func (s *Sync) queryFill(ctx context.Context, row *persistententity.ArbBinaryOrder, creds activeCreds) (fill, fee float64, status string, err error) {
	oid := strings.TrimSpace(row.PlatformOrderID)
	platform := strings.ToUpper(strings.TrimSpace(row.Platform))
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	switch platform {
	case appcommon.PlatformPolymarket:
		if s.poly == nil || !creds.polyOK {
			return 0, 0, "", errUnavailable("polymarket")
		}
		detail, err := s.poly.GetOrderById(ctx, httpentity.GetOrderReq{OrderID: oid}, creds.poly)
		if err != nil {
			return 0, 0, "", err
		}
		fill = book.ParseNum(detail.Order.SizeMatched)
		fee = s.polyFee(ctx, creds.poly, oid, row.TokenID)
		return fill, fee, string(detail.Order.Status), nil

	case appcommon.PlatformPredict:
		if s.predict == nil || !creds.predOK {
			return 0, 0, "", errUnavailable("predict")
		}
		d, err := s.predict.GetOrderByHash(ctx, creds.predict, oid)
		if err != nil {
			return 0, 0, "", err
		}
		return d.AmountFilled, d.FeeUSDT, d.Status, nil

	case appcommon.PlatformSxbet:
		if s.sxbet == nil || !creds.sxOK {
			return 0, 0, "", errUnavailable("sxbet")
		}
		if f, err := s.sxbet.GetFillsByOrderID(ctx, creds.sxbet, oid); err == nil && f != nil {
			if f.FillQty > 0 || f.FeeUSDT > 0 {
				return f.FillQty, f.FeeUSDT, f.Status, nil
			}
		}
		o, err := s.sxbet.GetOrderByID(ctx, creds.sxbet, oid)
		if err != nil {
			return 0, 0, "", err
		}
		return o.FillQty, o.FeeUSDT, o.Status, nil

	default:
		return 0, 0, "", errUnavailable(platform)
	}
}

func (s *Sync) polyFee(ctx context.Context, acc httpentity.PolymarketAccount, orderID, tokenID string) float64 {
	tr, err := s.poly.GetTrades(ctx, httpentity.GetTradesReq{TokenID: tokenID, Limit: 50}, acc)
	if err != nil || tr == nil {
		return 0
	}
	var fee float64
	for _, t := range tr.Trades {
		if !strings.EqualFold(t.TakerOrderID, orderID) {
			continue
		}
		sz := book.ParseNum(t.Size)
		px := book.ParseNum(t.Price)
		bps := book.ParseNum(t.FeeRateBps)
		if sz > 0 && px > 0 && bps > 0 {
			fee += sz * px * bps / 10000
		}
	}
	if math.IsNaN(fee) || math.IsInf(fee, 0) {
		return 0
	}
	return fee
}

func statusFromFill(fill, qty float64) string {
	if fill <= 0 {
		return "NO_FILL"
	}
	if qty > 0 && fill+1e-8 < qty {
		return "PARTIAL"
	}
	return "FILLED"
}

func errUnavailable(platform string) error {
	return &syncErr{msg: platform + " creds/client unavailable"}
}

type syncErr struct{ msg string }

func (e *syncErr) Error() string { return e.msg }

func polyAccount(a persistententity.Account) (httpentity.PolymarketAccount, bool) {
	sig := 0
	if a.SignatureType != nil {
		sig = *a.SignatureType
	}
	acc := httpentity.PolymarketAccount{
		APIKey: deref(a.APIKey), APISecret: deref(a.APISecretEnc), APIPassphrase: deref(a.APIPassphraseEnc),
		EOAAddress: deref(a.EoaAddress), ProxyAddress: deref(a.ProxyAddress), PrivateKey: deref(a.PrivateKeyEnc),
		SignatureType: sig,
	}
	ok := acc.APIKey != "" && acc.APISecret != "" && acc.APIPassphrase != "" && acc.PrivateKey != "" &&
		(acc.ProxyAddress != "" || acc.EOAAddress != "")
	return acc, ok
}

func predictAccount(a persistententity.Account) (httpentity.PredictAccount, bool) {
	acc := httpentity.PredictAccount{
		APIKey: deref(a.APIKey), PrivateKey: deref(a.PrivateKeyEnc),
		PredictAccount: deref(a.ProxyAddress), EOAAddress: deref(a.EoaAddress),
	}
	return acc, acc.APIKey != "" && acc.PrivateKey != ""
}

func sxbetAccount(a persistententity.Account) (httpentity.SxbetAccount, bool) {
	acc := httpentity.SxbetAccount{
		APIKey: deref(a.APIKey), PrivateKey: deref(a.PrivateKeyEnc),
		EOAAddress: deref(a.EoaAddress), ProxyAddress: deref(a.ProxyAddress),
	}
	return acc, acc.APIKey != "" && acc.PrivateKey != "" && acc.ProxyAddress != ""
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
