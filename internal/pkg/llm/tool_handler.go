package llm

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	log "log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/go-resty/resty/v2"
	"github.com/go-shiori/go-readability"
)

// HandleFunction 工具处理器函数签名
type HandleFunction func(context.Context, string) (string, error)

// ToolHandler 工具处理器
type ToolHandler struct {
	postRepo   es.PostRepo
	httpClient *resty.Client
	browserCtx context.Context
	cancel     context.CancelFunc
}

// NewToolHandler 在单例初始化时启动浏览器引擎
func NewToolHandler(postRepo es.PostRepo) *ToolHandler {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.UserAgent(ua),
		chromedp.ProxyServer(config.Cfg.Server.SearchGateway),
	)

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	browserCtx, cancel := chromedp.NewContext(allocCtx)

	if err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank")); err != nil {
		panic(fmt.Sprintf("浏览器引擎启动失败，请检查是否安装 Chrome: %v", err))
	}

	client := resty.New().
		SetTimeout(20*time.Second).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetHeader("User-Agent", ua).
		SetProxy(config.Cfg.Server.SearchGateway)

	return &ToolHandler{
		postRepo:   postRepo,
		httpClient: client,
		browserCtx: browserCtx,
		cancel:     cancel,
	}
}

// GetHandleFunction 返回绑定了当前实例的工具映射表
func (s *ToolHandler) GetHandleFunction(funcName string) HandleFunction {
	return map[string]HandleFunction{
		"search_community_posts": s.SearchCommunityPosts,
		"get_post_url":           s.GetPostURL,
		"web_search":             s.WebSearch,
		"web_fetch":              s.WebFetch,
	}[funcName]
}

// SearchCommunityPosts 实现了具体的搜索逻辑
func (s *ToolHandler) SearchCommunityPosts(ctx context.Context, argsJson string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJson), &args); err != nil {
		log.ErrorContext(ctx, "SearchCommunityPosts", "error", err)
		return "", errors.New("参数解析失败")
	}

	vector, err := fetchModelEmbedding(ctx, args.Query)
	if err != nil {
		return "", err
	}

	posts, err := s.postRepo.HybridSearch(ctx, args.Query, vector, 0, 10)
	if err != nil {
		return "", err
	}

	if len(posts) == 0 {
		return "未找到任何相关的站内笔记。", nil
	}

	var builder strings.Builder
	builder.WriteString("以下是为你找到的站内相关笔记：\n\n")

	for i, post := range posts {
		displayContent := post.Content
		if len(displayContent) > 300 {
			displayContent = displayContent[:300] + "..."
		}

		item := fmt.Sprintf("### 笔记 %d\n- **ID**: %d\n- **标题**: %s\n- **作者**: %s\n- **内容**: %s\n- **AI总结**: %s\n---\n",
			i+1, post.ID, post.Title, post.UserNickname, displayContent, post.AISummary)
		builder.WriteString(item)
	}
	log.InfoContext(ctx, "SearchCommunityPosts", "query", args.Query, "results", builder.String())
	return builder.String(), nil
}

// GetPostURL 动态生成帖子访问链接
func (s *ToolHandler) GetPostURL(ctx context.Context, argsJson string) (string, error) {
	var args struct {
		PostID int `json:"post_id"`
	}
	if err := json.Unmarshal([]byte(argsJson), &args); err != nil {
		log.ErrorContext(ctx, "GetPostURL", "error", err)
		return "", errors.New("参数解析失败")
	}

	baseURL, ok := ctx.Value(consts.BaseURL).(string)
	if !ok || baseURL == "" {
		log.WarnContext(ctx, "get base url fail")
		baseURL = "http://localhost:5173"
	}

	fullURL := fmt.Sprintf("%s/post?id=%d", baseURL, args.PostID)

	log.InfoContext(ctx, "GetPostURL", "post_id", args.PostID, "url", fullURL)
	return fmt.Sprintf("ID 为 %d 的帖子链接为: %s。请在回复中以 Markdown 链接格式 [标题](链接) 呈现给用户。", args.PostID, fullURL), nil
}

// WebSearch 实现了互联网搜索
func (s *ToolHandler) WebSearch(ctx context.Context, argsJson string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJson), &args); err != nil {
		log.ErrorContext(ctx, "WebSearch", "error", err)
		return "", errors.New("参数解析失败")
	}

	formData := url.Values{}
	formData.Set("q", args.Query)

	resp, err := s.httpClient.R().SetContext(ctx).SetFormDataFromValues(formData).Post("https://html.duckduckgo.com/html")
	if err != nil {
		log.ErrorContext(ctx, "WebSearch", "error", err)
		return "", errors.New("网络搜索失败")
	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
	var builder strings.Builder
	realIdx := 1
	doc.Find(".result").Each(func(i int, sel *goquery.Selection) {
		if realIdx > 5 {
			return
		}
		anchor := sel.Find(".result__title a")
		link, _ := anchor.Attr("href")
		if strings.Contains(link, "y.js") || strings.Contains(link, "ad_provider") {
			return
		}
		if strings.Contains(link, "uddg=") {
			u, _ := url.Parse(link)
			rawLink := u.Query().Get("uddg")
			if decodedLink, err := url.QueryUnescape(rawLink); err == nil {
				link = decodedLink
			} else {
				link = rawLink
			}
		}
		title := strings.TrimSpace(anchor.Text())
		snippet := strings.TrimSpace(sel.Find(".result__snippet").Text())
		builder.WriteString(fmt.Sprintf("[%d] 标题: %s\n链接: %s\n摘要: %s\n\n", realIdx, title, link, snippet))
		realIdx++
	})
	log.Info("WebSearch", "query", args.Query, "results", builder.String())
	return builder.String(), nil
}

// WebFetch 实现了网页渲染与正文提取
func (s *ToolHandler) WebFetch(ctx context.Context, argsJson string) (string, error) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argsJson), &args); err != nil {
		log.ErrorContext(ctx, "WebFetch", "error", err)
		return "", errors.New("参数解析失败")
	}

	resp, err := s.httpClient.R().SetContext(ctx).Get(args.URL)
	html := ""
	if err == nil {
		html = resp.String()
	}

	lowHtml := strings.ToLower(html)
	if strings.Contains(lowHtml, "loading") || len(html) < 4000 {
		tabCtx, cancel := chromedp.NewContext(s.browserCtx)
		defer cancel()

		var timeoutCancel context.CancelFunc
		tabCtx, timeoutCancel = context.WithTimeout(tabCtx, 20*time.Second)
		defer timeoutCancel()

		var renderHtml string
		err = chromedp.Run(tabCtx,
			chromedp.Navigate(args.URL),
			chromedp.WaitReady(`body`),
			chromedp.OuterHTML("html", &renderHtml),
		)
		if err == nil {
			html = renderHtml
		}
	}

	parsedURL, _ := url.Parse(args.URL)
	article, err := readability.FromReader(strings.NewReader(html), parsedURL)
	if err != nil {
		log.ErrorContext(ctx, "WebFetch", "error", err)
		return "无法从该链接提取有效正文内容。", nil
	}

	text := regexp.MustCompile(`\s+`).ReplaceAllString(article.TextContent, " ")
	if len(text) > 3000 {
		text = text[:3000] + "... [内容已截断]"
	}

	log.Info("WebFetch", "url", args.URL, "title", article.Title, "content", text)
	return fmt.Sprintf("标题: %s\n正文内容: %s", article.Title, text), nil
}

func (s *ToolHandler) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}
