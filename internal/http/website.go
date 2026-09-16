package http

import (
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"time"

	// "net/url"
	"strings"

	// readability "codeberg.org/readeck/go-readability/v2"
	"github.com/JerryJeager/raglearn/internal/models"
	"github.com/JerryJeager/raglearn/internal/service/websites"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"
	"golang.org/x/net/html"
)

type WebsiteController struct {
	serv websites.WebsiteSv
}

func NewWebsiteController(serv websites.WebsiteSv) *WebsiteController {
	return &WebsiteController{serv: serv}
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_3_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/122.0.6261.89 Mobile/15E148 Safari/604.1",
}

var blockElements = map[string]bool{
	"p": true, "div": true, "br": true, "li": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"tr": true, "table": true, "section": true, "article": true,
	"blockquote": true, "pre": true, "ul": true, "ol": true, "header": true,
	"footer": true, "main": true,
}

var tagsToSkip = map[string]bool{
	"script": true, "style": true, "noscript": true, "svg": true,
	"iframe": true, "form": true, "button": true, "select": true,
	"nav": true, "footer": true, "head": true,
}

func (c *WebsiteController) CreateWebsite(ctx *gin.Context) {
	var website models.Website
	if err := ctx.ShouldBindJSON(&website); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"message": "invalid request body",
		})
		return
	}

	rawHTML, err := fetchWithColly(website.Url)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	websiteID, err := c.serv.CreateWebsite(ctx, &website)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	go saveContentEmbedding(websiteID, website.Url, rawHTML)

}

// fetchWithColly does a fast plain HTTP fetch. Good for static/SSR sites.
func fetchWithColly(url string) (string, error) {
	c1 := colly.NewCollector()
	c1.SetRequestTimeout(15 * time.Second)

	var body string
	var visitErr error

	c1.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	})

	c1.OnResponse(func(r *colly.Response) {
		body = string(r.Body)
	})

	c1.OnError(func(r *colly.Response, err error) {
		visitErr = err
	})

	if err := c1.Visit(url); err != nil {
		return "", err
	}
	c1.Wait()

	if visitErr != nil {
		return "", visitErr
	}
	return body, nil
}

// fetchWithRod launches headless Chrome, waits for JS to render, and
// returns the fully hydrated HTML. Use as a fallback for SPAs.
func fetchWithRod(url string) (string, error) {
	launcherURL, err := launcher.New().
		Headless(true).
		Set("disable-gpu", "").
		Set("no-sandbox", "").
		Launch()
	if err != nil {
		return "", err
	}

	browser := rod.New().ControlURL(launcherURL)
	if err := browser.Connect(); err != nil {
		return "", err
	}
	defer browser.Close()

	page, err := browser.Timeout(20 * time.Second).Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return "", err
	}
	defer page.Close()

	// Wait for the network to go quiet (SPA data fetches finished) rather
	// than a fixed sleep. Adjust duration per how chatty the sites are.
	page.MustWaitStable()

	htmlStr, err := page.HTML()
	if err != nil {
		return "", err
	}
	return htmlStr, nil
}

// extractCleanText parses raw HTML, strips non-content tags, prefers the
// most specific content container available, and normalizes whitespace.
func extractCleanText(rawHTML string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return ""
	}

	doc.Find("script, style, noscript, svg, iframe, nav, footer, header, form, [aria-hidden='true']").Remove()

	selectors := []string{"main", "article", "[role='main']", "#content", ".content", "body"}
	for _, sel := range selectors {
		if selection := doc.Find(sel).First(); selection.Length() > 0 {
			if node := selection.Get(0); node != nil {
				text := walkAndExtract(node)
				if strings.TrimSpace(text) != "" {
					return normalizeWhitespace(text)
				}
			}
		}
	}
	return ""
}

// walkAndExtract recursively collects text nodes, skipping junk tags, and
// inserts a newline after block-level elements so text doesn't run together.
func walkAndExtract(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)

	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && tagsToSkip[node.Data] {
			return
		}
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if node.Type == html.ElementNode && blockElements[node.Data] {
			sb.WriteString("\n")
		}
	}

	walk(n)
	return sb.String()
}

var (
	reMultiSpace   = regexp.MustCompile(`[ \t]+`)
	reMultiNewline = regexp.MustCompile(`\n{3,}`)
)

func normalizeWhitespace(s string) string {
	s = reMultiSpace.ReplaceAllString(s, " ")
	s = reMultiNewline.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func saveContentEmbedding(websiteID uuid.UUID, websiteUrl, rawHTML string) {
	content := extractCleanText(rawHTML)

	//switch to rod if content is less than 200 chars--assuming page is probably an SPA(react, angular, next...) to load the full content
	const spaThreshold = 200
	if len(strings.TrimSpace(content)) < spaThreshold {
		log.Println("colly result looks like an SPA shell, falling back to rod")
		renderedHTML, rodErr := fetchWithRod(websiteUrl)
		if rodErr != nil {
			_ = websites.UpdateWebsiteStatus(websiteID, "failed")
			return
		}
		content = extractCleanText(renderedHTML)
	}

	if strings.TrimSpace(content) == "" {
		_ = websites.UpdateWebsiteStatus(websiteID, "failed")
		return
	}

}

func (c *WebsiteController) GetWebsite(ctx *gin.Context) {
	var websiteID WebsiteIDPP
	if err := ctx.ShouldBindUri(&websiteID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	website, err := c.serv.GetWebsite(ctx, websiteID.WebsiteID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}

	ctx.JSON(http.StatusOK, website)
}

func (c *WebsiteController) DeleteWebsite(ctx *gin.Context) {
	var websiteID WebsiteIDPP
	if err := ctx.ShouldBindUri(&websiteID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	err := c.serv.DeleteWebsite(ctx, websiteID.WebsiteID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}

	ctx.Status(http.StatusNoContent)
}
