package usage

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

type ChartRange string

const (
	ChartRange1H  ChartRange = "1h"
	ChartRange5H  ChartRange = "5h"
	ChartRange24H ChartRange = "24h"
	ChartRange7D  ChartRange = "7d"
)

type ChartGranularity string

const (
	ChartGranularity10Minute ChartGranularity = "10m"
	ChartGranularityHour     ChartGranularity = "hour"
	ChartGranularityDay      ChartGranularity = "day"
)

type ChartQuery struct {
	Range        ChartRange
	Granularity  ChartGranularity
	Account      string
	ProviderKey  string
	Provider     string
	AuthIndex    string
	APIKeyHash   string
	Model        string
	NowMS        int64
	AuthMetadata []ChartAuthMetadata
}

type ChartAuthMetadata struct {
	AuthID    string
	AuthIndex string
	Account   string
	Label     string
	AuthFile  string
}

type ChartMetricBucket struct {
	StartMS      int64   `json:"startMs"`
	EndMS        int64   `json:"endMs"`
	Label        string  `json:"label"`
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	CachedTokens int64   `json:"cachedTokens"`
	TotalCost    float64 `json:"totalCost"`
	TPMInput     float64 `json:"tpmInput"`
	TPMOutput    float64 `json:"tpmOutput"`
	TPMCached    float64 `json:"tpmCached"`
}

type ChartBucketGroup struct {
	Buckets []ChartMetricBucket `json:"buckets"`
}

type ChartSeries struct {
	Key        string              `json:"key"`
	Label      string              `json:"label"`
	Account    string              `json:"account,omitempty"`
	Provider   string              `json:"-"`
	AuthIndex  string              `json:"authIndex,omitempty"`
	APIKeyHash string              `json:"apiKeyHash,omitempty"`
	Model      string              `json:"model,omitempty"`
	IsOther    bool                `json:"isOther,omitempty"`
	Buckets    []ChartMetricBucket `json:"buckets"`
}

type ChartFilters struct {
	Account    string `json:"account,omitempty"`
	Provider   string `json:"-"`
	APIKeyHash string `json:"apiKeyHash,omitempty"`
	Model      string `json:"model,omitempty"`
}

type ChartAccountOption struct {
	Value     string `json:"value"`
	Label     string `json:"label"`
	Account   string `json:"account,omitempty"`
	AuthIndex string `json:"authIndex,omitempty"`
}

type ChartProviderOption struct {
	Value     string `json:"value"`
	Label     string `json:"label"`
	Provider  string `json:"provider,omitempty"`
	AuthIndex string `json:"authIndex,omitempty"`
}

type ChartAPIKeyOption struct {
	Value      string `json:"value"`
	APIKeyHash string `json:"apiKeyHash"`
	Label      string `json:"label"`
}

type ChartModelOption struct {
	Value string `json:"value"`
	Model string `json:"model"`
	Label string `json:"label"`
}

type ChartOptions struct {
	Accounts  []ChartAccountOption  `json:"accounts"`
	Providers []ChartProviderOption `json:"-"`
	APIKeys   []ChartAPIKeyOption   `json:"apiKeys"`
	Models    []ChartModelOption    `json:"models"`
}

type ChartSeriesGroup struct {
	Series []ChartSeries `json:"series"`
}

type ChartsResponse struct {
	Range              ChartRange       `json:"range"`
	Granularity        ChartGranularity `json:"granularity"`
	StartMS            int64            `json:"startMs"`
	EndMS              int64            `json:"endMs"`
	BucketMS           int64            `json:"bucketMs"`
	Filters            ChartFilters     `json:"filters"`
	Options            ChartOptions     `json:"options"`
	Global             ChartBucketGroup `json:"global"`
	ByAccount          ChartSeriesGroup `json:"byAccount"`
	ByProvider         ChartSeriesGroup `json:"-"`
	ByAPIKey           ChartSeriesGroup `json:"byApiKey"`
	ByModel            ChartSeriesGroup `json:"byModel"`
	MissingPriceModels []string         `json:"missingPriceModels"`
	GeneratedAtMS      int64            `json:"generatedAtMs"`
}

func ParseChartQuery(values url.Values) (ChartQuery, error) {
	query := ChartQuery{
		Range:       ChartRange(strings.TrimSpace(values.Get("range"))),
		Granularity: ChartGranularity(strings.TrimSpace(values.Get("granularity"))),
		Account:     strings.TrimSpace(values.Get("account")),
		APIKeyHash:  strings.ToLower(strings.TrimSpace(values.Get("apiKeyHash"))),
		Model:       strings.TrimSpace(values.Get("model")),
	}
	return NormalizeChartQuery(query)
}

func NormalizeChartQuery(query ChartQuery) (ChartQuery, error) {
	if query.Range == "" {
		query.Range = ChartRange1H
	}
	if !validChartRange(query.Range) {
		return ChartQuery{}, errors.New("invalid chart range")
	}

	if query.Granularity != "" && !validChartGranularity(query.Granularity) {
		return ChartQuery{}, errors.New("invalid chart granularity")
	}
	query.Granularity = defaultChartGranularity(query.Range)

	query.Account = strings.TrimSpace(query.Account)
	query.ProviderKey, query.Provider, query.AuthIndex = normalizeChartProviderFilter(query.ProviderKey, query.Provider, query.AuthIndex)
	query.APIKeyHash = strings.ToLower(strings.TrimSpace(query.APIKeyHash))
	query.Model = strings.TrimSpace(query.Model)
	if query.NowMS <= 0 {
		query.NowMS = time.Now().UnixMilli()
	}
	return query, nil
}

func EmptyChartsResponse(query ChartQuery) ChartsResponse {
	query, err := NormalizeChartQuery(query)
	if err != nil {
		query, _ = NormalizeChartQuery(ChartQuery{})
	}
	startMS, endMS, bucketMS := ChartWindow(query)
	return ChartsResponse{
		Range:       query.Range,
		Granularity: query.Granularity,
		StartMS:     startMS,
		EndMS:       endMS,
		BucketMS:    bucketMS,
		Filters: ChartFilters{
			Account:    query.Account,
			Provider:   query.ProviderKey,
			APIKeyHash: query.APIKeyHash,
			Model:      query.Model,
		},
		Options: ChartOptions{
			Accounts:  []ChartAccountOption{},
			Providers: []ChartProviderOption{},
			APIKeys:   []ChartAPIKeyOption{},
			Models:    []ChartModelOption{},
		},
		Global:             ChartBucketGroup{Buckets: BuildChartBuckets(startMS, endMS, bucketMS, query.Granularity)},
		ByAccount:          ChartSeriesGroup{Series: []ChartSeries{}},
		ByProvider:         ChartSeriesGroup{Series: []ChartSeries{}},
		ByAPIKey:           ChartSeriesGroup{Series: []ChartSeries{}},
		ByModel:            ChartSeriesGroup{Series: []ChartSeries{}},
		MissingPriceModels: []string{},
		GeneratedAtMS:      query.NowMS,
	}
}

func ChartWindow(query ChartQuery) (startMS int64, endMS int64, bucketMS int64) {
	durationMS := chartRangeDurationMS(query.Range)
	endMS = query.NowMS
	startMS = endMS - durationMS
	bucketMS = int64(time.Hour / time.Millisecond)
	if query.Granularity == ChartGranularity10Minute {
		bucketMS = int64((10 * time.Minute) / time.Millisecond)
	} else if query.Granularity == ChartGranularityDay {
		bucketMS = int64((24 * time.Hour) / time.Millisecond)
	}
	return startMS, endMS, bucketMS
}

func BuildChartBuckets(startMS int64, endMS int64, bucketMS int64, granularity ChartGranularity) []ChartMetricBucket {
	if bucketMS <= 0 || endMS <= startMS {
		return []ChartMetricBucket{}
	}
	buckets := make([]ChartMetricBucket, 0, int((endMS-startMS+bucketMS-1)/bucketMS))
	for bucketStart := startMS; bucketStart < endMS; bucketStart += bucketMS {
		bucketEnd := bucketStart + bucketMS
		if bucketEnd > endMS {
			bucketEnd = endMS
		}
		buckets = append(buckets, ChartMetricBucket{
			StartMS: bucketStart,
			EndMS:   bucketEnd,
			Label:   formatChartBucketLabel(bucketStart, granularity),
		})
	}
	return buckets
}

func validChartRange(value ChartRange) bool {
	switch value {
	case ChartRange1H, ChartRange5H, ChartRange24H, ChartRange7D:
		return true
	default:
		return false
	}
}

func validChartGranularity(value ChartGranularity) bool {
	switch value {
	case ChartGranularity10Minute, ChartGranularityHour, ChartGranularityDay:
		return true
	default:
		return false
	}
}

func defaultChartGranularity(chartRange ChartRange) ChartGranularity {
	switch chartRange {
	case ChartRange1H:
		return ChartGranularity10Minute
	case ChartRange7D:
		return ChartGranularityDay
	default:
		return ChartGranularityHour
	}
}

func normalizeChartProviderFilter(providerKey string, provider string, authIndex string) (string, string, string) {
	providerKey = strings.TrimSpace(providerKey)
	provider = strings.TrimSpace(provider)
	authIndex = strings.TrimSpace(authIndex)

	if strings.HasPrefix(providerKey, "auth:") {
		authIndex = strings.TrimSpace(strings.TrimPrefix(providerKey, "auth:"))
		if authIndex == "" {
			return "", provider, ""
		}
		return "auth:" + authIndex, "", authIndex
	}
	if strings.HasPrefix(providerKey, "provider:") {
		provider = strings.TrimSpace(strings.TrimPrefix(providerKey, "provider:"))
		if provider == "" {
			return "", "", authIndex
		}
		return "provider:" + provider, provider, authIndex
	}
	if providerKey != "" {
		provider = providerKey
	}
	if authIndex != "" {
		return "auth:" + authIndex, provider, authIndex
	}
	if provider != "" {
		return "provider:" + provider, provider, ""
	}
	return "", "", ""
}

func chartRangeDurationMS(chartRange ChartRange) int64 {
	switch chartRange {
	case ChartRange5H:
		return int64((5 * time.Hour) / time.Millisecond)
	case ChartRange24H:
		return int64((24 * time.Hour) / time.Millisecond)
	case ChartRange7D:
		return int64((7 * 24 * time.Hour) / time.Millisecond)
	case ChartRange1H:
		fallthrough
	default:
		return int64(time.Hour / time.Millisecond)
	}
}

func formatChartBucketLabel(startMS int64, granularity ChartGranularity) string {
	start := time.UnixMilli(startMS).Local()
	if granularity == ChartGranularityDay {
		return start.Format("01/02")
	}
	return start.Format("15:04")
}
