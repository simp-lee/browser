package browser

import (
	"context"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"sync"
	"time"
)

// Cookie represents a simplified cookie structured as a key-value pair.
type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  time.Time
	HTTPOnly bool
	Secure   bool
}

// Browser represents a managed browser instance.
type Browser struct {
	browser     *rod.Browser
	pool        *rod.PagePool
	proxy       string
	headless    bool
	poolSize    int
	lastUsed    time.Time
	idleTimeout time.Duration
	mu          sync.Mutex
	timer       *time.Timer
	ctx         context.Context
	cancel      context.CancelFunc
}

// Option is a function type for configuring Browser.
type Option func(*Browser)

// WithProxy WithProxy("127.0.0.1:8080"), sets flag "--proxy-server=127.0.0.1:8080"
func WithProxy(proxy string) Option {
	return func(b *Browser) {
		b.proxy = proxy
	}
}

// WithHeadless sets flag "--headless"
func WithHeadless(headless bool) Option {
	return func(b *Browser) {
		b.headless = headless
	}
}

// WithPoolSize sets the page pool size for the browser.
func WithPoolSize(poolSize int) Option {
	return func(b *Browser) {
		b.poolSize = poolSize
	}
}

// WithIdleTimeout sets the idle timeout for the browser.
func WithIdleTimeout(idleTimeout time.Duration) Option {
	return func(b *Browser) {
		b.idleTimeout = idleTimeout
	}
}

// PageOption is a function type for configuring rod.Page.
type PageOption func(*rod.Page)

// WithUserAgent sets the user agent for the page.
func WithUserAgent(userAgent string) PageOption {
	return func(page *rod.Page) {
		page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: userAgent,
		})
	}
}

// WithReferer sets the referer for the page.
func WithReferer(referer string) PageOption {
	return func(page *rod.Page) {
		page.MustSetExtraHeaders("Referer", referer)
	}
}

// WithViewport sets the viewport size for the page.
func WithViewport(width, height int, deviceScaleFactor float64, isMobile bool) PageOption {
	return func(page *rod.Page) {
		page.MustSetViewport(width, height, deviceScaleFactor, isMobile)
	}
}

// WithExtraHeaders sets the additional headers for the page.
func WithExtraHeaders(headers map[string]string) PageOption {
	return func(page *rod.Page) {
		args := make([]string, 0, len(headers)*2)
		for key, value := range headers {
			args = append(args, key, value)
		}
		page.MustSetExtraHeaders(args...)
	}
}

// WithCookies sets simplified cookies for the page.
func WithCookies(cookies ...Cookie) PageOption {
	return func(page *rod.Page) {
		convertedCookies := make([]*proto.NetworkCookieParam, len(cookies))
		for i, cookie := range cookies {
			convertedCookies[i] = &proto.NetworkCookieParam{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Domain:   cookie.Domain,
				Path:     cookie.Path,
				Expires:  proto.TimeSinceEpoch(cookie.Expires.Unix()),
				HTTPOnly: cookie.HTTPOnly,
				Secure:   cookie.Secure,
			}
		}
		page.MustSetCookies(convertedCookies...)
	}
}

// GetCookies retrieves cookies from the page and returns them as a slice of Cookie.
func (b *Browser) GetCookies(page *rod.Page) ([]Cookie, error) {
	cookies, err := page.Cookies([]string{})
	if err != nil {
		return nil, err
	}

	simplifiedCookies := make([]Cookie, len(cookies))
	for i, c := range cookies {
		simplifiedCookies[i] = Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  time.Unix(int64(c.Expires), 0),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
		}
	}

	return simplifiedCookies, nil
}

// browsers is a map of browser instances with unique options.
// This allows us to reuse browser instances with the same options.
// This is useful when we want to share the same browser instance across multiple goroutines.
// The key is a unique string generated from the browser options.
// The value is the browser instance.
var (
	browsers = make(map[string]*Browser)
	mu       sync.RWMutex
)

// GetBrowser returns a browser instance with the provided options.
// If a browser with these options already exists, it returns the existing instance.
// Otherwise, it creates a new browser instance with these options.
func GetBrowser(options ...Option) (*Browser, error) {
	mu.RLock()
	key := generateKey(options...)
	if browser, ok := browsers[key]; ok {
		mu.RUnlock()
		return browser, nil
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	// Check again in case another goroutine created the browser while we were waiting for the lock.
	if browser, ok := browsers[key]; ok {
		return browser, nil
	}

	browser, err := NewBrowser(options...)
	if err != nil {
		return nil, err
	}
	browsers[key] = browser

	return browser, nil
}

// NewBrowser creates a new browser instance with the provided options.
// Headless will be enabled by default.
// Pool size will be set to 3 by default.
// Idle timeout will be set to 5 minutes by default.
func NewBrowser(options ...Option) (*Browser, error) {
	b := &Browser{
		headless:    true,
		poolSize:    3,
		idleTimeout: 5 * time.Minute,
	}

	for _, option := range options {
		option(b)
	}

	// Create a new context for the browser instance
	b.ctx, b.cancel = context.WithCancel(context.Background())

	// Create a new browser instance
	return createBrowser(b)
}

// createBrowser creates a new browser instance with the provided options.
func createBrowser(b *Browser) (*Browser, error) {
	// Create a rod control url
	url := launcher.New().
		Headless(b.headless).
		Leakless(true).
		NoSandbox(true).
		Delete("enable-automation").
		Set("ignore-certificate-errors").
		Set("ignore-certificate-errors-spki-list").
		Set("ignore-ssl-errors").
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-setuid-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Set("unlimited-storage").
		Set("disable-accelerated-2d-canvas").
		Set("full-memory-crash-report")

	// Set proxy if provided
	if b.proxy != "" {
		url.Proxy(b.proxy)
	}

	// Create a rod browser
	browser := rod.New()

	defer func() {
		if err := recover(); err != nil {
			browser.MustClose()
			panic(err)
		}
	}()

	browser.ControlURL(url.MustLaunch()).
		SlowMotion(960 * time.Microsecond)

	// Connect to the browser instance
	err := browser.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	// Create a rod page pool
	pool := rod.NewPagePool(b.poolSize)

	b.browser = browser
	b.pool = &pool
	b.lastUsed = time.Now()

	// Set a timer to close the browser instance when idle
	// func AfterFunc(d Duration, f func()) *Timer
	// AfterFunc waits for the duration to elapse and then calls f in its own goroutine.
	// It returns a Timer that can be used to cancel the call using its Stop method.
	// The returned Timer's C field is not used and will be nil.
	b.timer = time.AfterFunc(b.idleTimeout, func() {
		if err := b.Close(); err != nil {
			fmt.Println("failed to close browser:", err)
		}
	})

	return b, nil
}

// GetPage returns a page instance from the browser pool.
// If the browser instance is nil, it creates a new browser instance.
// If the page pool is empty, it creates a new page instance.
// It also resets the idle timer.
func (b *Browser) GetPage(options ...PageOption) (*rod.Page, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.browser == nil {
		var err error
		b, err = createBrowser(b)
		if err != nil {
			return nil, err
		}
	}

	b.lastUsed = time.Now()

	// Reset the timer
	b.timer.Reset(b.idleTimeout)

	// Create a new page instance from the pool or create a new page instance if the pool is empty.
	create := func() (*rod.Page, error) {
		page := b.browser.MustIncognito().MustPage()

		for _, option := range options {
			option(page)
		}

		return page, nil
	}

	page, err := b.pool.Get(create)
	if err != nil {
		return nil, fmt.Errorf("failed to get page from pool: %w", err)
	}

	return page, nil
}

// PutPage puts a page instance back into the browser pool.
func (b *Browser) PutPage(page *rod.Page) {
	b.mu.Lock()
	b.lastUsed = time.Now()
	b.timer.Reset(b.idleTimeout)
	b.mu.Unlock()

	b.pool.Put(page)
}

// BlockImageLoading blocks the loading of image resources on a page.
func (b *Browser) BlockImageLoading(page *rod.Page) error {
	router := page.HijackRequests()
	err := router.Add("*", proto.NetworkResourceTypeImage, func(ctx *rod.Hijack) {
		ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
	})

	if err != nil {
		return fmt.Errorf("failed to block image loading: %w", err)
	}

	go router.Run()

	return nil
}

// Close closes the browser instance and all the page instances in the pool.
// This function is thread-safe and handles potential deadlock situations.
func (b *Browser) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.browser != nil {
		// Use the official Cleanup method to iterate through the page pool and attempt to return all pages to the pool.
		b.pool.Cleanup(func(page *rod.Page) {
			if err := page.Close(); err != nil {
				fmt.Println("failed to close page:", err)
			}
		})

		if err := b.browser.Close(); err != nil {
			return fmt.Errorf("failed to close browser: %w", err)
		}
		b.browser = nil
		b.cancel()

		// Remove the browser instance from the map of browsers
		mu.Lock()
		delete(browsers, generateKey(
			WithProxy(b.proxy),
			WithHeadless(b.headless),
			WithPoolSize(b.poolSize),
			WithIdleTimeout(b.idleTimeout),
		))
		mu.Unlock()
	}

	return nil
}

// generateKey generates a unique key for a set of options.
// The key is a string that contains the options.
// This key is used to identify a browser instance with the same options.
func generateKey(options ...Option) string {
	tempBrowser := &Browser{
		headless:    true,
		poolSize:    3,
		idleTimeout: 5 * time.Minute,
	}

	for _, option := range options {
		option(tempBrowser)
	}

	return fmt.Sprintf("%s-%t-%d-%s",
		tempBrowser.proxy,
		tempBrowser.headless,
		tempBrowser.poolSize,
		tempBrowser.idleTimeout,
	)
}
