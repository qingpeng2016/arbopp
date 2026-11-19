package boot

import (
	"context"
	"fmt"
	"strings"
	"time"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// PlatformCatalog 启动时从 platform 表加载的 ACTIVE 平台配置（只读快照）。
type PlatformCatalog struct {
	bySlug map[string]persistententity.Platform
}

func NewPlatformCatalog(repo persistentrepo.PlatformRepo) (*PlatformCatalog, error) {
	ctx := context.Background()
	_, list, err := repo.List(ctx, nil, map[string]interface{}{
		"status = ?": "ACTIVE",
	}, "sort asc", "")
	if err != nil {
		return nil, fmt.Errorf("load platform catalog: %w", err)
	}
	bySlug := make(map[string]persistententity.Platform, len(list))
	for _, p := range list {
		slug := strings.ToUpper(strings.TrimSpace(p.Slug))
		if slug != "" {
			bySlug[slug] = p
		}
	}
	return &PlatformCatalog{bySlug: bySlug}, nil
}

func (c *PlatformCatalog) Get(slug string) (persistententity.Platform, bool) {
	if c == nil {
		return persistententity.Platform{}, false
	}
	p, ok := c.bySlug[strings.ToUpper(strings.TrimSpace(slug))]
	return p, ok
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func marketAPIBase(p *persistententity.Platform, fallback string) string {
	if p == nil {
		return fallback
	}
	if v := derefStr(p.MarketAPIBaseURL); v != "" {
		return strings.TrimRight(v, "/")
	}
	return fallback
}

func orderAPIBase(p *persistententity.Platform, fallback string) string {
	if p == nil {
		return fallback
	}
	if v := derefStr(p.OrderAPIBaseURL); v != "" {
		return strings.TrimRight(v, "/")
	}
	return fallback
}

func platformTimeout(p *persistententity.Platform) time.Duration {
	if p == nil || p.Timeout <= 0 {
		return 15 * time.Second
	}
	return time.Duration(p.Timeout) * time.Second
}
