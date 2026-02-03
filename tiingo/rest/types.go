package rest

import (
	"fmt"
	"strings"
	"time"
)

// Custom Time type for flexible unmarshaling
type Time time.Time

func (t *Time) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `'"`)
	if s == "null" || s == "" {
		return nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000000000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, format := range formats {
		parsed, err := time.Parse(format, s)
		if err == nil {
			*t = Time(parsed)
			return nil
		}
	}
	return fmt.Errorf("unable to parse time: %s", s)
}

// --- Data Models ---

type TickerMetadata struct {
	Ticker       string `json:"ticker"`
	Name         string `json:"name"`
	ExchangeCode string `json:"exchangeCode"`
	StartDate    Time   `json:"startDate"`
	EndDate      Time   `json:"endDate"`
	Description  string `json:"description"`
}

type PriceData struct {
	Date        Time    `json:"date"`
	Close       float64 `json:"close"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Open        float64 `json:"open"`
	Volume      float64 `json:"volume"`
	AdjClose    float64 `json:"adjClose"`
	AdjHigh     float64 `json:"adjHigh"`
	AdjLow      float64 `json:"adjLow"`
	AdjOpen     float64 `json:"adjOpen"`
	AdjVolume   float64 `json:"adjVolume"`
	DivCash     float64 `json:"divCash"`
	SplitFactor float64 `json:"splitFactor"`
}

type IEXPrice struct {
	Ticker            string  `json:"ticker"`
	Timestamp         Time    `json:"timestamp"`
	QuoteTimestamp    Time    `json:"quoteTimestamp"`
	LastSaleTimestamp Time    `json:"lastSaleTimestamp"`
	Last              float64 `json:"last"`
	LastSize          float64 `json:"lastSize"`
	BidSize           float64 `json:"bidSize"`
	BidPrice          float64 `json:"bidPrice"`
	AskSize           float64 `json:"askSize"`
	AskPrice          float64 `json:"askPrice"`
	Volume            float64 `json:"volume"`
	High              float64 `json:"high"`
	Low               float64 `json:"low"`
	Open              float64 `json:"open"`
	PrevClose         float64 `json:"prevClose"`
	Mid               float64 `json:"mid"`
}

type IEXWebSocketData struct {
	Ticker    string  `json:"ticker"`
	Timestamp Time    `json:"timestamp"`
	Last      float64 `json:"last"`
	LastSize  float64 `json:"lastSize"`
	TngoLast  float64 `json:"tngoLast"`
	BidSize   float64 `json:"bidSize"`
	BidPrice  float64 `json:"bidPrice"`
	AskSize   float64 `json:"askSize"`
	AskPrice  float64 `json:"askPrice"`
}

type CryptoPrice struct {
	Ticker         string  `json:"ticker"`
	BaseCurrency   string  `json:"baseCurrency"`
	QuoteCurrency  string  `json:"quoteCurrency"`
	Exchange       string  `json:"exchange"`
	Date           Time    `json:"date"`
	Open           float64 `json:"open"`
	High           float64 `json:"high"`
	Low            float64 `json:"low"`
	Close          float64 `json:"close"`
	Volume         float64 `json:"volume"`
	VolumeNotional float64 `json:"volumeNotional"`
	TradesDone     int     `json:"tradesDone"`
}

type NewsItem struct {
	ID            int64    `json:"id"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Description   string   `json:"description"`
	PublishedDate Time     `json:"publishedDate"`
	CrawlDate     Time     `json:"crawlDate"`
	Source        string   `json:"source"`
	Tags          []string `json:"tags"`
	Tickers       []string `json:"tickers"`
}

type FundamentalsStatement struct {
	Ticker        string             `json:"ticker"`
	StatementType string             `json:"statementType"`
	Quarter       int                `json:"quarter"`
	Year          int                `json:"year"`
	Date          Time               `json:"date"`
	DataCode      map[string]float64 `json:"dataCode"`
}

type SearchResult struct {
	Ticker      string `json:"ticker"`
	Name        string `json:"name"`
	AssetType   string `json:"assetType"`
	PermaTicker string `json:"permaTicker"`
	Exchange    string `json:"exchange"`
}

// --- REST Client Request Options ---

type EndOfDayPricesOptions struct {
	StartDate    string `url:"startDate,omitempty"`
	EndDate      string `url:"endDate,omitempty"`
	Format       string `url:"format,omitempty"`
	ResampleFreq string `url:"resampleFreq,omitempty"`
	Columns      string `url:"columns,omitempty"`
	Sort         string `url:"sort,omitempty"`
}

type IEXRealTimePricesOptions struct {
	Tickers                string `url:"tickers,omitempty"`
	ResampleFreq           string `url:"resampleFreq,omitempty"`
	Columns                string `url:"columns,omitempty"`
	AfterHours             bool   `url:"afterHours,omitempty"`
	ForceFill              bool   `url:"forceFill,omitempty"`
	IncludeRawExchangeData bool   `url:"includeRawExchangeData,omitempty"`
}

type CryptoPricesOptions struct {
	Tickers                  string `url:"tickers,omitempty"`
	StartDate                string `url:"startDate,omitempty"`
	EndDate                  string `url:"endDate,omitempty"`
	ResampleFreq             string `url:"resampleFreq,omitempty"`
	Exchanges                string `url:"exchanges,omitempty"`
	ConsolidatedBaseCurrency bool   `url:"consolidatedBaseCurrency,omitempty"`
	ConvertCurrency          string `url:"convertCurrency,omitempty"`
}

type ForexPricesOptions struct {
	StartDate    string `url:"startDate,omitempty"`
	EndDate      string `url:"endDate,omitempty"`
	ResampleFreq string `url:"resampleFreq,omitempty"`
}

type NewsFeedOptions struct {
	Tickers   string `url:"tickers,omitempty"`
	Tags      string `url:"tags,omitempty"`
	Sources   string `url:"sources,omitempty"`
	StartDate string `url:"startDate,omitempty"`
	EndDate   string `url:"endDate,omitempty"`
	Limit     int    `url:"limit,omitempty"`
	Offset    int    `url:"offset,omitempty"`
	SortBy    string `url:"sortBy,omitempty"`
}

type FundamentalsStatementsOptions struct {
	StartDate  string `url:"startDate,omitempty"`
	EndDate    string `url:"endDate,omitempty"`
	AsReported bool   `url:"asReported,omitempty"`
	Format     string `url:"format,omitempty"`
}

type FundamentalsDailyOptions struct {
	StartDate string `url:"startDate,omitempty"`
	EndDate   string `url:"endDate,omitempty"`
	Format    string `url:"format,omitempty"`
}

type SearchOptions struct {
	Query string `url:"query,omitempty"`
	Limit int    `url:"limit,omitempty"`
}
