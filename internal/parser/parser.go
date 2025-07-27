package parser

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/Houeta/chrono-flow/internal/models"
	"github.com/PuerkitoBio/goquery"
)

type Parser struct {
	log     *slog.Logger
	client  *http.Client
	destURL string
}

func NewParser(log *slog.Logger, destinationURL string) *Parser {
	return &Parser{log: log, destURL: destinationURL, client: http.DefaultClient}
}

func (p *Parser) ParseProducts(ctx context.Context) ([]models.Product, error) {
	resp, err := p.getHTMLResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get html response: %w", err)
	}
	defer resp.Body.Close()

	return p.parseTableResponse(ctx, resp.Body)
}

func (p *Parser) getHTMLResponse(ctx context.Context) (*http.Response, error) {
	reqURL, err := url.Parse(p.destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse destination URL %s: %w", p.destURL, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request %s: %w", reqURL.String(), err)
	}

	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; GoHttpClient/1.0)")

	p.log.DebugContext(ctx, "Send request", "method", req.Method, "URL", req.URL, "header", req.Header)

	res, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request %s: %w", p.destURL, err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code error: [%d] %s", res.StatusCode, res.Status)
	}

	p.log.InfoContext(ctx, "Successfully received http response", "status code", res.StatusCode)

	return res, nil
}

func (p *Parser) parseTableResponse(ctx context.Context, inp io.ReadCloser) ([]models.Product, error) {
	doc, err := goquery.NewDocumentFromReader(inp)
	if err != nil {
		return nil, fmt.Errorf("data cannot be parsed as HTML: %w", err)
	}

	var products []models.Product
	numberOfCells := 5
	modelIdx := 0
	typeIdx := 1
	quantityIdx := 2
	imageIdx := 3
	priceIdx := 4

	doc.Find(".table-bordered tbody tr").Each(func(idx int, s *goquery.Selection) {
		cells := s.Find("td")

		if cells.Length() == numberOfCells {
			product := models.Product{
				Model:    strings.TrimSpace(cells.Eq(modelIdx).Text()),
				Type:     strings.TrimSpace(cells.Eq(typeIdx).Text()),
				Quantity: strings.TrimSpace(cells.Eq(quantityIdx).Text()),
				ImageURL: strings.TrimSpace(cells.Eq(imageIdx).Text()),
				Price:    strings.TrimSpace(cells.Eq(priceIdx).Text()),
			}
			p.log.DebugContext(
				ctx,
				"Parsed product",
				"Model", product.Model,
				"Price", product.Price,
				"Quantity", product.Quantity,
			)
			products = append(products, product)
		} else {
			p.log.WarnContext(ctx, "table row has insufficient cells", "index", idx, "length", cells.Length())
		}
	})

	return products, nil
}
