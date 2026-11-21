package boot

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	apphttp "github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http/crypto/prediction/kalshi"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http/crypto/prediction/predict"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http/crypto/prediction/polymarket"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http/crypto/prediction/sxbet"
	"go.uber.org/zap"
)

const (
	defaultPolyGamma  = "https://gamma-api.polymarket.com"
	defaultPolyCLOB   = "https://clob.polymarket.com"
	defaultKalshiAPI  = "https://api.elections.kalshi.com"
	defaultPredictAPI = "https://api.predict.fun/v1"
	defaultSxbetAPI   = "https://api.sx.bet"
)

func NewKalshiRepoFromPlatform(catalog *PlatformCatalog) httprepo.KalshiRepo {
	ctx := context.Background()
	var baseURL string
	timeout := 15 * time.Second
	if p, ok := catalog.Get(string(cryptoPredictionEnum.PlatformKalshi)); ok {
		baseURL = marketAPIBase(&p, defaultKalshiAPI)
		timeout = platformTimeout(&p)
		logger.InfoZ(ctx, "platform-kalshi-loaded",
			zap.String("market_api_base_url", baseURL),
			zap.Duration("timeout", timeout))
	} else {
		logger.InfoZ(ctx, "platform-kalshi-not-active",
			zap.String("fallback", defaultKalshiAPI),
			zap.String("hint", "platform 未 ACTIVE 时使用内置基址；market_sync 不会同步 Kalshi"))
		baseURL = defaultKalshiAPI
	}
	hc := &http.Client{Timeout: timeout, Transport: kalshi.NewPooledTransport()}
	return kalshi.NewClientWithOptions(baseURL, hc, nil)
}

func NewPolymarketRepoFromPlatform(catalog *PlatformCatalog, client *apphttp.Client) httprepo.PolymarketRepo {
	ctx := context.Background()
	marketBase := defaultPolyGamma
	orderBase := defaultPolyCLOB
	if p, ok := catalog.Get(string(cryptoPredictionEnum.PlatformPolymarket)); ok {
		marketBase = marketAPIBase(&p, defaultPolyGamma)
		orderBase = orderAPIBase(&p, defaultPolyCLOB)
		logger.InfoZ(ctx, "platform-polymarket-loaded",
			zap.String("market_api_base_url", marketBase),
			zap.String("order_api_base_url", orderBase))
	} else {
		logger.InfoZ(ctx, "platform-polymarket-not-active",
			zap.String("fallback_gamma", defaultPolyGamma),
			zap.String("hint", "platform 未 ACTIVE 时使用内置基址；market_sync 不会同步 Polymarket"))
	}
	return polymarket.NewPolymarketWithPlatform(client, marketBase, orderBase)
}

type predictPlatformExtra struct {
	APIKey string `json:"api_key"`
}

func NewPredictRepoFromPlatform(catalog *PlatformCatalog, accountRepo persistentrepo.AccountRepo) httprepo.PredictRepo {
	ctx := context.Background()
	apiKey := resolvePredictAPIKey(ctx, catalog, accountRepo)
	if apiKey == "" {
		notification.SendErrorLog(ctx, "platform-predict-missing-api-key",
			zap.String("hint", "platform.extra_config {\"api_key\":\"...\"} 或 account.api_key (platform=PREDICT, status=ACTIVE)"))
		return nil
	}
	baseURL := defaultPredictAPI
	timeout := 20 * time.Second
	if p, ok := catalog.Get(string(cryptoPredictionEnum.PlatformPredict)); ok {
		baseURL = marketAPIBase(&p, defaultPredictAPI)
		timeout = platformTimeout(&p)
		logger.InfoZ(ctx, "platform-predict-loaded",
			zap.String("market_api_base_url", baseURL),
			zap.Duration("timeout", timeout))
	} else {
		logger.InfoZ(ctx, "platform-predict-not-active", zap.String("fallback", defaultPredictAPI))
	}
	hc := &http.Client{Timeout: timeout, Transport: predict.NewPooledTransport()}
	return predict.NewClientWithOptions(baseURL, apiKey, hc)
}

type sxbetPlatformExtra struct {
	APIKey string `json:"api_key"`
}

func NewSxbetRepoFromPlatform(catalog *PlatformCatalog, accountRepo persistentrepo.AccountRepo) httprepo.SxbetRepo {
	ctx := context.Background()
	apiKey := resolveSxbetAPIKey(ctx, catalog, accountRepo)
	if apiKey == "" {
		notification.SendErrorLog(ctx, "platform-sxbet-missing-api-key",
			zap.String("hint", "platform.extra_config {\"api_key\":\"...\"} 或 account.api_key (platform=SXBET, status=ACTIVE)"))
	}
	baseURL := defaultSxbetAPI
	timeout := 20 * time.Second
	if p, ok := catalog.Get(string(cryptoPredictionEnum.PlatformSxbet)); ok {
		baseURL = marketAPIBase(&p, defaultSxbetAPI)
		timeout = platformTimeout(&p)
		logger.InfoZ(ctx, "platform-sxbet-loaded",
			zap.String("market_api_base_url", baseURL),
			zap.Duration("timeout", timeout),
			zap.Bool("has_api_key", apiKey != ""))
	} else {
		logger.InfoZ(ctx, "platform-sxbet-not-active",
			zap.String("fallback", defaultSxbetAPI),
			zap.String("hint", "platform 未 ACTIVE 时使用内置基址；market_sync 不会同步 SXBET"))
	}
	hc := &http.Client{Timeout: timeout, Transport: sxbet.NewPooledTransport()}
	return sxbet.NewClientWithOptions(baseURL, apiKey, hc)
}

func resolveSxbetAPIKey(ctx context.Context, catalog *PlatformCatalog, accountRepo persistentrepo.AccountRepo) string {
	if catalog != nil {
		if p, ok := catalog.Get(string(cryptoPredictionEnum.PlatformSxbet)); ok && p.ExtraConfig != nil {
			raw := strings.TrimSpace(*p.ExtraConfig)
			if raw != "" {
				var extra sxbetPlatformExtra
				if json.Unmarshal([]byte(raw), &extra) == nil {
					if k := strings.TrimSpace(extra.APIKey); k != "" {
						return k
					}
				}
			}
		}
	}
	return resolveActiveAccountAPIKey(ctx, accountRepo, string(cryptoPredictionEnum.PlatformSxbet))
}

func resolvePredictAPIKey(ctx context.Context, catalog *PlatformCatalog, accountRepo persistentrepo.AccountRepo) string {
	if catalog != nil {
		if p, ok := catalog.Get(string(cryptoPredictionEnum.PlatformPredict)); ok && p.ExtraConfig != nil {
			raw := strings.TrimSpace(*p.ExtraConfig)
			if raw != "" {
				var extra predictPlatformExtra
				if json.Unmarshal([]byte(raw), &extra) == nil {
					if k := strings.TrimSpace(extra.APIKey); k != "" {
						return k
					}
				}
			}
		}
	}
	return resolveActiveAccountAPIKey(ctx, accountRepo, string(cryptoPredictionEnum.PlatformPredict))
}

func resolveActiveAccountAPIKey(ctx context.Context, accountRepo persistentrepo.AccountRepo, platform string) string {
	if accountRepo == nil {
		return ""
	}
	_, list, err := accountRepo.List(ctx, nil, map[string]interface{}{
		"platform = ?": platform,
		"status = ?":   "ACTIVE",
	}, "id asc", "")
	if err != nil || len(list) == 0 {
		return ""
	}
	for _, a := range list {
		if a.APIKey != nil {
			if k := strings.TrimSpace(*a.APIKey); k != "" {
				return k
			}
		}
	}
	return ""
}
