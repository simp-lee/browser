# Browser Package

This package provides a high-level API for managing and interacting with browser instances using the [rod](https://github.com/go-rod/rod) library. It supports features such as browser pooling, headless mode, proxy configuration, cookies, and automatic idle timeout.

## Installation

To install the package, use `go get`:

```sh
go get -u github.com/simp-lee/browser
```

## Usage

### Basic Usage

Here's a basic example of how to use the package:

```go
package main

import (
	"fmt"
	"github.com/simp-lee/browser"
	"time"
)

func main() {
	// Get a browser instance with default options
	b, err := browser.GetBrowser()
	if err != nil {
		panic(err)
	}
	defer b.Close()

	// Get a page instance from the browser
	page, err := b.GetPage()
	if err != nil {
		panic(err)
	}
	defer b.PutPage(page)
	
	page.MustNavigate("https://example.com")

	fmt.Println(page.MustInfo().Title)
}
```

### Configuring Browser Options

You can configure various options for the browser instance:

```go
package main

import (
	"github.com/simp-lee/browser"
	"time"
)

func main() {
	// Get a browser instance with custom options
	b, err := browser.GetBrowser(
		browser.WithHeadless(false),
		browser.WithProxy("127.0.0.1:8080"),
		browser.WithPoolSize(5),
		browser.WithIdleTimeout(10*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer b.Close()
}
```

### Configuring Page Options

You can configure various options for the page instance:

```go
package main

import (
	"fmt"
	"github.com/simp-lee/browser"
)

func main() {
	// Get a browser instance with default options
	b, err := browser.GetBrowser()
	if err != nil {
		panic(err)
	}
	defer b.Close()

	// Get a page instance with custom options
	page, err := b.GetPage(
		browser.WithUserAgent("CustomUserAgent"),
		browser.WithReferer("https://example.com"),
		browser.WithViewport(1920, 1080, 1.0, false),
		browser.WithExtraHeaders(map[string]string{
			"X-Custom-Header": "custom_value",
		}),
		browser.WithCookies(browser.Cookie{
			Name:  "example_cookie",
			Value: "cookie_value",
			Domain: "example.com",
		}),
	)
	if err != nil {
		panic(err)
	}
	defer b.PutPage(page)
	
	page.MustNavigate("https://example.com")
	
	fmt.Println(page.MustInfo().Title)
}
```

### Blocking Image Loading

You can block image loading on a page by calling `b.BlockImageLoading(page)` to save bandwidth`:

```go
package main

import (
	"fmt"
	"github.com/simp-lee/browser"
)

func main() {
	// Get a browser instance with default options
	b, err := browser.GetBrowser()
	if err != nil {
		panic(err)
	}
	defer b.Close()

	// Get a page instance from the browser
	page, err := b.GetPage()
	if err != nil {
		panic(err)
	}
	defer b.PutPage(page)

	// Block image loading on the page
	if err := b.BlockImageLoading(page); err != nil {
		panic(err)
	}
	
	page.MustNavigate("https://example.com")
	
	fmt.Println(page.MustInfo().Title)
}
```

### Retrieving Cookies

You can retrieve cookies from a page instance using the `GetCookies` method::

```go
package main

import (
	"fmt"
	"github.com/simp-lee/browser"
)

func main() {
	// Get a browser instance with default options
	b, err := browser.GetBrowser()
	if err != nil {
		panic(err)
	}
	defer b.Close()

	// Get a page instance from the browser
	page, err := b.GetPage()
	if err != nil {
		panic(err)
	}
	defer b.PutPage(page)

	// Navigate to a URL
	page.MustNavigate("https://example.com")

	// Get cookies from the page
	cookies, err := b.GetCookies(page)
	if err != nil {
		panic(err)
	}

	// Print the cookies
	for _, cookie := range cookies {
		fmt.Printf("Cookie: %+v\n", cookie)
	}
}
```