package gofakeit

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/brianvoe/gofakeit/v7/data"
)

// DomainName will generate a random url domain name
func DomainName() string { return domainName(GlobalFaker) }

// DomainName will generate a random url domain name
func (f *Faker) DomainName() string { return domainName(f) }

func domainName(f *Faker) string {
	name := strings.Replace(strings.ToLower(jobDescriptor(f)+bs(f)), " ", "", -1)

	return fmt.Sprintf("%s.%s", name, domainSuffix(f))
}

// DomainSuffix will generate a random domain suffix
func DomainSuffix() string { return domainSuffix(GlobalFaker) }

// DomainSuffix will generate a random domain suffix
func (f *Faker) DomainSuffix() string { return domainSuffix(f) }

func domainSuffix(f *Faker) string {
	return getRandValue(f, []string{"internet", "domain_suffix"})
}

// URL will generate a random url string
func URL() string { return url(GlobalFaker) }

// URL will generate a random url string
func (f *Faker) URL() string { return url(f) }

func url(f *Faker) string {
	// Slugs
	num := number(f, 1, 4)
	slug := make([]string, num)
	for i := 0; i < num; i++ {
		slug[i] = bs(f)
	}

	scheme := randomString(f, []string{"https", "http"})
	path := strings.ToLower(strings.Join(slug, "/"))

	url := fmt.Sprintf("%s://www.%s/%s", scheme, domainName(f), path)
	url = strings.Replace(url, " ", "", -1)

	return url
}

// HTTPMethod will generate a random http method
func HTTPMethod() string { return httpMethod(GlobalFaker) }

// HTTPMethod will generate a random http method
func (f *Faker) HTTPMethod() string { return httpMethod(f) }

func httpMethod(f *Faker) string {
	return getRandValue(f, []string{"internet", "http_method"})
}

// IPv4Address will generate a random version 4 ip address
func IPv4Address() string { return ipv4Address(GlobalFaker) }

// IPv4Address will generate a random version 4 ip address
func (f *Faker) IPv4Address() string { return ipv4Address(f) }

func ipv4Address(f *Faker) string {
	num := func() int { return f.IntN(256) }

	return fmt.Sprintf("%d.%d.%d.%d", num(), num(), num(), num())
}

// IPv6Address will generate a random version 6 ip address
func IPv6Address() string { return ipv6Address(GlobalFaker) }

// IPv6Address will generate a random version 6 ip address
func (f *Faker) IPv6Address() string { return ipv6Address(f) }

func ipv6Address(f *Faker) string {
	num := func() int { return f.IntN(65536) }

	return fmt.Sprintf("%x:%x:%x:%x:%x:%x:%x:%x", num(), num(), num(), num(), num(), num(), num(), num())
}

// MacAddress will generate a random mac address
func MacAddress() string { return macAddress(GlobalFaker) }

// MacAddress will generate a random mac address
func (f *Faker) MacAddress() string { return macAddress(f) }

func macAddress(f *Faker) string {
	num := 255

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", f.IntN(num), f.IntN(num), f.IntN(num), f.IntN(num), f.IntN(num), f.IntN(num))
}

// HTTPStatusCode will generate a random status code
func HTTPStatusCode() int { return httpStatusCode(GlobalFaker) }

// HTTPStatusCode will generate a random status code
func (f *Faker) HTTPStatusCode() int { return httpStatusCode(f) }

func httpStatusCode(f *Faker) int {
	randInt, _ := strconv.Atoi(getRandValue(f, []string{"internet", "http_status_general"}))
	return randInt
}

// HTTPStatusCodeSimple will generate a random simple status code
func HTTPStatusCodeSimple() int { return httpStatusCodeSimple(GlobalFaker) }

// HTTPStatusCodeSimple will generate a random simple status code
func (f *Faker) HTTPStatusCodeSimple() int { return httpStatusCodeSimple(f) }

func httpStatusCodeSimple(f *Faker) int {
	randInt, _ := strconv.Atoi(getRandValue(f, []string{"internet", "http_status_simple"}))
	return randInt
}

// LogLevel will generate a random log level
// See data/LogLevels for list of available levels
func LogLevel(logType string) string { return logLevel(GlobalFaker, logType) }

// LogLevel will generate a random log level
// See data/LogLevels for list of available levels
func (f *Faker) LogLevel(logType string) string { return logLevel(f, logType) }

func logLevel(f *Faker, logType string) string {
	if _, ok := data.LogLevels[logType]; ok {
		return getRandValue(f, []string{"log_level", logType})
	}

	return getRandValue(f, []string{"log_level", "general"})
}

// UserAgent will generate a random broswer user agent
func UserAgent() string { return userAgent(GlobalFaker) }

// UserAgent will generate a random broswer user agent
func (f *Faker) UserAgent() string { return userAgent(f) }

func userAgent(f *Faker) string {
	randNum := randIntRange(f, 0, 4)
	switch randNum {
	case 0:
		return chromeUserAgent(f)
	case 1:
		return firefoxUserAgent(f)
	case 2:
		return safariUserAgent(f)
	case 3:
		return operaUserAgent(f)
	default:
		return chromeUserAgent(f)
	}
}

// ChromeUserAgent will generate a random chrome browser user agent string
func ChromeUserAgent() string { return chromeUserAgent(GlobalFaker) }

// ChromeUserAgent will generate a random chrome browser user agent string
func (f *Faker) ChromeUserAgent() string { return chromeUserAgent(f) }

func chromeUserAgent(f *Faker) string {
	randNum1 := strconv.Itoa(randIntRange(f, 531, 536)) + strconv.Itoa(randIntRange(f, 0, 2))
	randNum2 := strconv.Itoa(randIntRange(f, 36, 40))
	randNum3 := strconv.Itoa(randIntRange(f, 800, 899))
	return "Mozilla/5.0 " + "(" + randomPlatform(f) + ") AppleWebKit/" + randNum1 + " (KHTML, like Gecko) Chrome/" + randNum2 + ".0." + randNum3 + ".0 Mobile Safari/" + randNum1
}

// FirefoxUserAgent will generate a random firefox broswer user agent string
func FirefoxUserAgent() string { return firefoxUserAgent(GlobalFaker) }

// FirefoxUserAgent will generate a random firefox broswer user agent string
func (f *Faker) FirefoxUserAgent() string { return firefoxUserAgent(f) }

func firefoxUserAgent(f *Faker) string {
	ver := "Gecko/" + date(f).Format("2006-01-02") + " Firefox/" + strconv.Itoa(randIntRange(f, 35, 37)) + ".0"
	platforms := []string{
		"(" + windowsPlatformToken(f) + "; " + "en-US" + "; rv:1.9." + strconv.Itoa(randIntRange(f, 0, 3)) + ".20) " + ver,
		"(" + linuxPlatformToken(f) + "; rv:" + strconv.Itoa(randIntRange(f, 5, 8)) + ".0) " + ver,
		"(" + macPlatformToken(f) + " rv:" + strconv.Itoa(randIntRange(f, 2, 7)) + ".0) " + ver,
	}

	return "Mozilla/5.0 " + randomString(f, platforms)
}

// SafariUserAgent will generate a random safari browser user agent string
func SafariUserAgent() string { return safariUserAgent(GlobalFaker) }

// SafariUserAgent will generate a random safari browser user agent string
func (f *Faker) SafariUserAgent() string { return safariUserAgent(f) }

func safariUserAgent(f *Faker) string {
	randNum := strconv.Itoa(randIntRange(f, 531, 536)) + "." + strconv.Itoa(randIntRange(f, 1, 51)) + "." + strconv.Itoa(randIntRange(f, 1, 8))
	ver := strconv.Itoa(randIntRange(f, 4, 6)) + "." + strconv.Itoa(randIntRange(f, 0, 2))

	mobileDevices := []string{
		"iPhone; CPU iPhone OS",
		"iPad; CPU OS",
	}

	platforms := []string{
		"(Windows; U; " + windowsPlatformToken(f) + ") AppleWebKit/" + randNum + " (KHTML, like Gecko) Version/" + ver + " Safari/" + randNum,
		"(" + macPlatformToken(f) + " rv:" + strconv.Itoa(randIntRange(f, 4, 7)) + ".0; en-US) AppleWebKit/" + randNum + " (KHTML, like Gecko) Version/" + ver + " Safari/" + randNum,
		"(" + randomString(f, mobileDevices) + " " + strconv.Itoa(randIntRange(f, 7, 9)) + "_" + strconv.Itoa(randIntRange(f, 0, 3)) + "_" + strconv.Itoa(randIntRange(f, 1, 3)) + " like Mac OS X; " + "en-US" + ") AppleWebKit/" + randNum + " (KHTML, like Gecko) Version/" + strconv.Itoa(randIntRange(f, 3, 5)) + ".0.5 Mobile/8B" + strconv.Itoa(randIntRange(f, 111, 120)) + " Safari/6" + randNum,
	}

	return "Mozilla/5.0 " + randomString(f, platforms)
}

// OperaUserAgent will generate a random opera browser user agent string
func OperaUserAgent() string { return operaUserAgent(GlobalFaker) }

// OperaUserAgent will generate a random opera browser user agent string
func (f *Faker) OperaUserAgent() string { return operaUserAgent(f) }

func operaUserAgent(f *Faker) string {
	platform := "(" + randomPlatform(f) + "; en-US) Presto/2." + strconv.Itoa(randIntRange(f, 8, 13)) + "." + strconv.Itoa(randIntRange(f, 160, 355)) + " Version/" + strconv.Itoa(randIntRange(f, 10, 13)) + ".00"

	return "Opera/" + strconv.Itoa(randIntRange(f, 8, 10)) + "." + strconv.Itoa(randIntRange(f, 10, 99)) + " " + platform
}

// linuxPlatformToken will generate a random linux platform
func linuxPlatformToken(f *Faker) string {
	return "X11; Linux " + getRandValue(f, []string{"computer", "linux_processor"})
}

// macPlatformToken will generate a random mac platform
func macPlatformToken(f *Faker) string {
	return "Macintosh; " + getRandValue(f, []string{"computer", "mac_processor"}) + " Mac OS X 10_" + strconv.Itoa(randIntRange(f, 5, 9)) + "_" + strconv.Itoa(randIntRange(f, 0, 10))
}

// windowsPlatformToken will generate a random windows platform
func windowsPlatformToken(f *Faker) string {
	return getRandValue(f, []string{"computer", "windows_platform"})
}

// randomPlatform will generate a random platform
func randomPlatform(f *Faker) string {
	platforms := []string{
		linuxPlatformToken(f),
		macPlatformToken(f),
		windowsPlatformToken(f),
	}

	return randomString(f, platforms)
}

// HTTPVersion will generate a random http version
func HTTPVersion() string { return httpVersion(GlobalFaker) }

// HTTPVersion will generate a random http version
func (f *Faker) HTTPVersion() string { return httpVersion(f) }

func httpVersion(f *Faker) string {
	return getRandValue(f, []string{"internet", "http_version"})
}

func addInternetLookup() {
	AddFuncLookup("url", Info{
		Display:     "URL",
		Category:    "internet",
		Description: "Web address that specifies the location of a resource on the internet",
		Example:     "http://www.principalproductize.biz/target",
		Output:      "string",
		Aliases:     []string{"url string", "web address", "internet link", "website url", "resource locator"},
		Keywords:    []string{"url", "web", "address", "http", "https", "www", "protocol", "scheme", "path", "domain", "location", "resource"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return url(f), nil
		},
	})

	AddFuncLookup("domainname", Info{
		Display:     "Domain Name",
		Category:    "internet",
		Description: "Human-readable web address used to identify websites on the internet",
		Example:     "centraltarget.biz",
		Output:      "string",
		Aliases:     []string{"domain name", "website name", "internet domain", "dns name", "site domain"},
		Keywords:    []string{"domain", "name", "web", "address", "dns", "hostname", "resolve", "centraltarget", "biz", "website"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return domainName(f), nil
		},
	})

	AddFuncLookup("domainsuffix", Info{
		Display:     "Domain Suffix",
		Category:    "internet",
		Description: "The part of a domain name that comes after the last dot, indicating its type or purpose",
		Example:     "org",
		Output:      "string",
		Aliases:     []string{"domain suffix", "domain extension", "top level domain", "domain ending"},
		Keywords:    []string{"domain", "suffix", "tld", "top-level", "extension", "org", "com", "net", "gov", "edu", "mil", "int"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return domainSuffix(f), nil
		},
	})

	AddFuncLookup("ipv4address", Info{
		Display:     "IPv4 Address",
		Category:    "internet",
		Description: "Numerical label assigned to devices on a network for identification and communication",
		Example:     "222.83.191.222",
		Output:      "string",
		Aliases:     []string{"ip address", "network address", "internet address", "device ip", "ipv4 label"},
		Keywords:    []string{"ipv4", "ip", "network", "internet", "protocol", "communication", "dotted", "decimal"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return ipv4Address(f), nil
		},
	})

	AddFuncLookup("ipv6address", Info{
		Display:     "IPv6 Address",
		Category:    "internet",
		Description: "Numerical label assigned to devices on a network, providing a larger address space than IPv4 for internet communication",
		Example:     "2001:cafe:8898:ee17:bc35:9064:5866:d019",
		Output:      "string",
		Aliases:     []string{"ip address", "network address", "internet address", "hex ip", "ipv6 label"},
		Keywords:    []string{"ipv6", "ip", "network", "protocol", "hexadecimal", "identification"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return ipv6Address(f), nil
		},
	})

	AddFuncLookup("httpmethod", Info{
		Display:     "HTTP Method",
		Category:    "internet",
		Description: "Verb used in HTTP requests to specify the desired action to be performed on a resource",
		Example:     "HEAD",
		Output:      "string",
		Aliases:     []string{"http verb", "http action", "http request", "http command", "method name"},
		Keywords:    []string{"http", "method", "verb", "get", "post", "put", "delete", "patch", "options", "head", "request", "action"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return httpMethod(f), nil
		},
	})

	AddFuncLookup("loglevel", Info{
		Display:     "Log Level",
		Category:    "internet",
		Description: "Classification used in logging to indicate the severity or priority of a log entry",
		Example:     "error",
		Output:      "string",
		Aliases:     []string{"log severity", "logging level", "log classification", "priority level", "event level"},
		Keywords:    []string{"log", "level", "severity", "priority", "classification", "error", "warn", "info", "debug", "trace", "fatal", "critical"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return logLevel(f, ""), nil
		},
	})

	AddFuncLookup("useragent", Info{
		Display:     "User Agent",
		Category:    "internet",
		Description: "String sent by a web browser to identify itself when requesting web content",
		Example:     "Mozilla/5.0 (Windows NT 5.0) AppleWebKit/5362 (KHTML, like Gecko) Chrome/37.0.834.0 Mobile Safari/5362",
		Output:      "string",
		Aliases:     []string{"ua string", "browser ua", "http user agent", "client identifier", "browser identifier"},
		Keywords:    []string{"useragent", "browser", "http", "request", "mozilla", "applewebkit", "chrome", "firefox", "safari", "opera", "mobile"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return userAgent(f), nil
		},
	})

	AddFuncLookup("chromeuseragent", Info{
		Display:     "Chrome User Agent",
		Category:    "internet",
		Description: "The specific identification string sent by the Google Chrome web browser when making requests on the internet",
		Example:     "Mozilla/5.0 (X11; Linux i686) AppleWebKit/5312 (KHTML, like Gecko) Chrome/39.0.836.0 Mobile Safari/5312",
		Output:      "string",
		Aliases:     []string{"chrome ua", "chrome browser ua", "google chrome ua", "chrome identifier", "chrome user agent"},
		Keywords:    []string{"chrome", "google", "browser", "ua", "useragent", "applewebkit", "khtml", "gecko", "safari", "version"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return chromeUserAgent(f), nil
		},
	})

	AddFuncLookup("firefoxuseragent", Info{
		Display:     "Firefox User Agent",
		Category:    "internet",
		Description: "The specific identification string sent by the Firefox web browser when making requests on the internet",
		Example:     "Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_3 rv:7.0) Gecko/1900-07-01 Firefox/37.0",
		Output:      "string",
		Aliases:     []string{"firefox ua", "firefox browser ua", "mozilla firefox ua", "gecko ua", "firefox identifier"},
		Keywords:    []string{"firefox", "mozilla", "browser", "ua", "useragent", "gecko", "macintosh", "ppc", "version"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return firefoxUserAgent(f), nil
		},
	})

	AddFuncLookup("operauseragent", Info{
		Display:     "Opera User Agent",
		Category:    "internet",
		Description: "The specific identification string sent by the Opera web browser when making requests on the internet",
		Example:     "Opera/8.39 (Macintosh; U; PPC Mac OS X 10_8_7; en-US) Presto/2.9.335 Version/10.00",
		Output:      "string",
		Aliases:     []string{"opera ua", "opera browser ua", "opera identifier", "opera client", "opera user agent"},
		Keywords:    []string{"opera", "presto", "ua", "browser", "useragent", "macintosh", "ppc", "os", "version"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return operaUserAgent(f), nil
		},
	})

	AddFuncLookup("safariuseragent", Info{
		Display:     "Safari User Agent",
		Category:    "internet",
		Description: "The specific identification string sent by the Safari web browser when making requests on the internet",
		Example:     "Mozilla/5.0 (iPad; CPU OS 8_3_2 like Mac OS X; en-US) AppleWebKit/531.15.6 (KHTML, like Gecko) Version/4.0.5 Mobile/8B120 Safari/6531.15.6",
		Output:      "string",
		Aliases:     []string{"safari ua", "apple safari ua", "safari browser ua", "safari identifier", "safari user agent"},
		Keywords:    []string{"safari", "apple", "ipad", "os", "applewebkit", "khtml", "gecko", "browser", "ua", "mobile"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return safariUserAgent(f), nil
		},
	})

	AddFuncLookup("httpstatuscode", Info{
		Display:     "HTTP Status Code",
		Category:    "internet",
		Description: "Random HTTP status code",
		Example:     "200",
		Output:      "int",
		Aliases:     []string{"http status", "response code", "http response", "server status", "status identifier"},
		Keywords:    []string{"http", "status", "code", "server", "response", "200", "404", "500", "301", "302", "401", "403"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return httpStatusCode(f), nil
		},
	})

	AddFuncLookup("httpstatuscodesimple", Info{
		Display:     "HTTP Status Code Simple",
		Category:    "internet",
		Description: "Three-digit number returned by a web server to indicate the outcome of an HTTP request",
		Example:     "404",
		Output:      "int",
		Aliases:     []string{"http status simple", "simple response code", "http response simple", "status code", "server code"},
		Keywords:    []string{"http", "status", "code", "server", "response", "200", "404", "500", "301", "302", "401", "403"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return httpStatusCodeSimple(f), nil
		},
	})

	AddFuncLookup("httpversion", Info{
		Display:     "HTTP Version",
		Category:    "internet",
		Description: "Number indicating the version of the HTTP protocol used for communication between a client and a server",
		Example:     "HTTP/1.1",
		Output:      "string",
		Aliases:     []string{"http version", "protocol version", "http protocol", "http identifier", "http version string"},
		Keywords:    []string{"http", "version", "protocol", "communication", "client", "server"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return httpVersion(f), nil
		},
	})

	AddFuncLookup("macaddress", Info{
		Display:     "MAC Address",
		Category:    "internet",
		Description: "Unique identifier assigned to network interfaces, often used in Ethernet networks",
		Example:     "cb:ce:06:94:22:e9",
		Output:      "string",
		Aliases:     []string{"mac address", "hardware address", "ethernet address", "network identifier", "link-layer address"},
		Keywords:    []string{"mac", "address", "hardware", "ethernet", "network", "identifier", "oui", "vendor", "colon", "hexadecimal"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return macAddress(f), nil
		},
	})

}
