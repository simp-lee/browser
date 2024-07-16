package browser

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGenerateKeyConsistency(t *testing.T) {
	options := []Option{
		WithHeadless(true),
		WithPoolSize(3),
		WithIdleTimeout(5 * time.Minute),
	}
	key1 := generateKey(options...)
	key2 := generateKey() // 使用默认值生成键

	assert.Equal(t, key1, key2, "The generated keys should be the same")
}

func TestGenerateKeyDifferentConfigurations(t *testing.T) {
	key1 := generateKey(
		WithHeadless(false),
		WithPoolSize(5),
		WithIdleTimeout(10*time.Minute),
	)
	key2 := generateKey(
		WithHeadless(true),
		WithPoolSize(5),
		WithIdleTimeout(10*time.Minute),
	)

	assert.NotEqual(t, key1, key2, "The generated keys should be different")
}

func TestNewBrowser(t *testing.T) {
	b, err := NewBrowser()
	assert.NoError(t, err)
	assert.Equal(t, "", b.proxy)
	assert.True(t, b.headless)
	assert.Equal(t, 3, b.poolSize)
	assert.Equal(t, 5*time.Minute, b.idleTimeout)
	assert.NotNil(t, b.browser)
	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)
}

func TestGetBrowser(t *testing.T) {
	b1, err := GetBrowser(WithPoolSize(5), WithIdleTimeout(10*time.Minute))
	assert.NoError(t, err)

	b2, err := GetBrowser(WithIdleTimeout(10*time.Minute), WithPoolSize(5))
	assert.NoError(t, err)

	assert.Equal(t, b1, b2)
	assert.NotNil(t, b1.browser)
	assert.Equal(t, "", b1.proxy)
	assert.True(t, b1.headless)
	assert.Equal(t, 5, b1.poolSize)
	assert.Equal(t, 10*time.Minute, b1.idleTimeout)
	assert.NotNil(t, b1.browser)
	err = b1.Close()
	assert.NoError(t, err)
	assert.Nil(t, b1.browser)
	err = b2.Close()
	assert.NoError(t, err)
	assert.Nil(t, b2.browser)
}

func TestBrowser_GetPage(t *testing.T) {
	b, err := GetBrowser()
	assert.NoError(t, err)

	page, err := b.GetPage()
	assert.NoError(t, err)
	assert.NotNil(t, page)

	err = page.Close()
	assert.NoError(t, err)
	assert.NotNil(t, b.browser)

	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)
}

func TestBrowser_PutPage(t *testing.T) {
	b, err := GetBrowser(WithPoolSize(5))
	assert.NoError(t, err)
	assert.Equal(t, 5, b.poolSize)
	assert.Equal(t, 5, len(*b.pool))

	page, err := b.GetPage()
	assert.NoError(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, 4, len(*b.pool))

	b.PutPage(page)
	assert.Equal(t, 5, len(*b.pool))

	page, err = b.GetPage()
	assert.NoError(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, 4, len(*b.pool))

	page2, err := b.GetPage()
	assert.NoError(t, err)
	assert.NotNil(t, page2)
	assert.Equal(t, 3, len(*b.pool))

	b.PutPage(page)
	assert.Equal(t, 4, len(*b.pool))

	b.PutPage(page2)
	assert.Equal(t, 5, len(*b.pool))

	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)
}

func TestBrowser_Close(t *testing.T) {
	b, err := GetBrowser(WithHeadless(false))
	assert.NoError(t, err)
	assert.NotNil(t, b.browser)

	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)
}

func TestBrowser_CloseWithDefaultConfig(t *testing.T) {
	b, err := GetBrowser()
	assert.NoError(t, err)
	assert.NotNil(t, b.browser)

	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)

	mu.RLock()
	_, exists := browsers[generateKey()]
	mu.RUnlock()
	assert.False(t, exists, "The default browser should not be in the map of browsers")
}

func TestGetBrowserWithDifferentOptions(t *testing.T) {
	b1, err := GetBrowser(WithHeadless(false))
	assert.NoError(t, err)
	assert.Equal(t, false, b1.headless)
	assert.Equal(t, 3, b1.poolSize)
	assert.Equal(t, 5*time.Minute, b1.idleTimeout)
	assert.NotNil(t, b1.browser)

	b2, err := GetBrowser(WithHeadless(true))
	assert.NoError(t, err)
	assert.Equal(t, true, b2.headless)
	assert.Equal(t, 3, b2.poolSize)
	assert.Equal(t, 5*time.Minute, b2.idleTimeout)
	assert.NotNil(t, b2.browser)

	assert.NotEqual(t, b1, b2)

	err = b1.Close()
	assert.NoError(t, err)
	assert.Nil(t, b1.browser)
	err = b2.Close()
	assert.NoError(t, err)
	assert.Nil(t, b2.browser)
}

func TestGetPageWithNilBrowser(t *testing.T) {
	b, err := NewBrowser(WithPoolSize(6))
	assert.NoError(t, err)
	assert.NotNil(t, b.browser)

	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)

	page, err := b.GetPage()
	assert.NoError(t, err)
	assert.NotNil(t, b.browser)
	assert.NotNil(t, page)
	assert.Equal(t, 5, len(*b.pool))

	err = page.Close()
	assert.NoError(t, err)

	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)
}

func TestCloseWithPagesInPool(t *testing.T) {
	b, err := GetBrowser()
	assert.NoError(t, err)

	page, err := b.GetPage()
	assert.NoError(t, err)
	b.PutPage(page)

	err = b.Close()
	assert.NoError(t, err)

	assert.Nil(t, b.browser)
	b.mu.Lock()
	assert.Equal(t, 0, len(*b.pool))
	b.mu.Unlock()
}

func TestIdleTimeout(t *testing.T) {
	b, err := GetBrowser(
		WithIdleTimeout(5*time.Second),
		WithHeadless(false),
	)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, b.idleTimeout)
	assert.False(t, b.headless)
	assert.Equal(t, 3, b.poolSize)
	assert.NotNil(t, b.browser)

	page, err := b.GetPage(WithViewport(800, 600, 1.0, true))
	assert.NoError(t, err)
	assert.NotNil(t, page)

	page.MustNavigate("https://www.baidu.com")
	page.MustWaitLoad()
	assert.Contains(t, page.MustInfo().Title, "百度一下")

	b.PutPage(page)

	assert.NotNil(t, b.browser)

	time.Sleep(6 * time.Second)

	assert.Nil(t, b.browser)
}

func TestPageOptions(t *testing.T) {
	b, _ := GetBrowser(WithHeadless(false))
	defer func(b *Browser) {
		_ = b.Close()
	}(b)

	userAgent := "test-user-agent"

	page, err := b.GetPage(WithUserAgent(userAgent))
	assert.NoError(t, err)

	defer b.PutPage(page)

	// Navigate to the page, which returns the headers
	page.MustNavigate("https://httpbin.org/headers")
	page.MustWaitLoad()

	time.Sleep(1 * time.Second)

	// Check if the user agent is set correctly
	assert.Contains(t, page.MustHTML(), userAgent)

	_ = b.Close()
}

// TestBrowser is a sample test function for the Browser package
func TestBrowser(t *testing.T) {
	// Define browser options
	browserOptions := []Option{
		WithHeadless(false),
		WithPoolSize(5),
		WithIdleTimeout(1 * time.Minute),
	}

	// Get a browser instance with the provided options
	b, err := GetBrowser(browserOptions...)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	// Define page options
	pageOptions := []PageOption{
		WithUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		WithReferer("https://example.com"),
		WithViewport(1920, 1080, 1.0, false),
		WithExtraHeaders(map[string]string{
			"X-Custom-Header": "custom_value",
		}),
	}

	// Get a page instance from the browser pool
	page, err := b.GetPage(pageOptions...)
	assert.NoError(t, err)
	assert.NotNil(t, page)

	// Navigate to a URL and perform some actions
	err = page.Navigate("https://example.com")
	assert.NoError(t, err)

	page.MustWaitLoad()

	time.Sleep(1 * time.Second)

	assert.Equal(t, "Example Domain", page.MustInfo().Title)

	// Block image loading
	err = b.BlockImageLoading(page)
	assert.NoError(t, err)

	// Put the page back into the pool
	b.PutPage(page)

	// Close the browser instance
	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)
}

func TestWithCookies(t *testing.T) {
	b, err := GetBrowser()
	assert.NoError(t, err)
	assert.NotNil(t, b.browser)

	page, err := b.GetPage(WithCookies(Cookie{
		Name:   "example_cookie",
		Value:  "cookie_value",
		Domain: "httpbin.org",
	}))
	assert.NoError(t, err)
	assert.NotNil(t, page)

	page.MustNavigate("https://httpbin.org/cookies")
	page.MustWaitLoad()

	time.Sleep(3 * time.Second)

	cookies, err := b.GetCookies(page)
	assert.NoError(t, err)
	assert.NotNil(t, cookies)

	found := false
	for _, cookie := range cookies {
		if cookie.Name == "example_cookie" && cookie.Value == "cookie_value" {
			found = true
			break
		}
	}

	assert.True(t, found, "Cookie not found in the page")

	err = page.Close()
	assert.NoError(t, err)
	assert.NotNil(t, b.browser)

	err = b.Close()
	assert.NoError(t, err)
	assert.Nil(t, b.browser)
}
