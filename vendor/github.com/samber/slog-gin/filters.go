package sloggin

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type Filter func(ctx *gin.Context) bool

// Basic
func Accept(filter Filter) Filter { return filter }
func Ignore(filter Filter) Filter { return filter }

// Method
func AcceptMethod(methods ...string) Filter {
	return func(c *gin.Context) bool {
		reqMethod := strings.ToLower(c.Request.Method)

		for _, method := range methods {
			if strings.ToLower(method) == reqMethod {
				return true
			}
		}

		return false
	}
}

func IgnoreMethod(methods ...string) Filter {
	return func(c *gin.Context) bool {
		reqMethod := strings.ToLower(c.Request.Method)

		for _, method := range methods {
			if strings.ToLower(method) == reqMethod {
				return false
			}
		}

		return true
	}
}

// Status
func AcceptStatus(statuses ...int) Filter {
	return func(c *gin.Context) bool {
		for _, status := range statuses {
			if status == c.Writer.Status() {
				return true
			}
		}

		return false
	}
}

func IgnoreStatus(statuses ...int) Filter {
	return func(c *gin.Context) bool {
		for _, status := range statuses {
			if status == c.Writer.Status() {
				return false
			}
		}

		return true
	}
}

func AcceptStatusGreaterThan(status int) Filter {
	return func(c *gin.Context) bool {
		return c.Writer.Status() > status
	}
}

func IgnoreStatusLessThan(status int) Filter {
	return func(c *gin.Context) bool {
		return c.Writer.Status() < status
	}
}

func AcceptStatusGreaterThanOrEqual(status int) Filter {
	return func(c *gin.Context) bool {
		return c.Writer.Status() >= status
	}
}

func IgnoreStatusLessThanOrEqual(status int) Filter {
	return func(c *gin.Context) bool {
		return c.Writer.Status() <= status
	}
}

// Path
func AcceptPath(urls ...string) Filter {
	return func(c *gin.Context) bool {
		for _, url := range urls {
			if c.Request.URL.Path == url {
				return true
			}
		}

		return false
	}
}

func IgnorePath(urls ...string) Filter {
	return func(c *gin.Context) bool {
		for _, url := range urls {
			if c.Request.URL.Path == url {
				return false
			}
		}

		return true
	}
}

func AcceptPathContains(parts ...string) Filter {
	return func(c *gin.Context) bool {
		for _, part := range parts {
			if strings.Contains(c.Request.URL.Path, part) {
				return true
			}
		}

		return false
	}
}

func IgnorePathContains(parts ...string) Filter {
	return func(c *gin.Context) bool {
		for _, part := range parts {
			if strings.Contains(c.Request.URL.Path, part) {
				return false
			}
		}

		return true
	}
}

func AcceptPathPrefix(prefixs ...string) Filter {
	return func(c *gin.Context) bool {
		for _, prefix := range prefixs {
			if strings.HasPrefix(c.Request.URL.Path, prefix) {
				return true
			}
		}

		return false
	}
}

func IgnorePathPrefix(prefixs ...string) Filter {
	return func(c *gin.Context) bool {
		for _, prefix := range prefixs {
			if strings.HasPrefix(c.Request.URL.Path, prefix) {
				return false
			}
		}

		return true
	}
}

func AcceptPathSuffix(prefixs ...string) Filter {
	return func(c *gin.Context) bool {
		for _, prefix := range prefixs {
			if strings.HasPrefix(c.Request.URL.Path, prefix) {
				return true
			}
		}

		return false
	}
}

func IgnorePathSuffix(suffixs ...string) Filter {
	return func(c *gin.Context) bool {
		for _, suffix := range suffixs {
			if strings.HasSuffix(c.Request.URL.Path, suffix) {
				return false
			}
		}

		return true
	}
}

func AcceptPathMatch(regs ...regexp.Regexp) Filter {
	return func(c *gin.Context) bool {
		for _, reg := range regs {
			if reg.Match([]byte(c.Request.URL.Path)) {
				return true
			}
		}

		return false
	}
}

func IgnorePathMatch(regs ...regexp.Regexp) Filter {
	return func(c *gin.Context) bool {
		for _, reg := range regs {
			if reg.Match([]byte(c.Request.URL.Path)) {
				return false
			}
		}

		return true
	}
}

// Host
func AcceptHost(hosts ...string) Filter {
	return func(c *gin.Context) bool {
		for _, host := range hosts {
			if c.Request.URL.Host == host {
				return true
			}
		}

		return false
	}
}

func IgnoreHost(hosts ...string) Filter {
	return func(c *gin.Context) bool {
		for _, host := range hosts {
			if c.Request.URL.Host == host {
				return false
			}
		}

		return true
	}
}

func AcceptHostContains(parts ...string) Filter {
	return func(c *gin.Context) bool {
		for _, part := range parts {
			if strings.Contains(c.Request.URL.Host, part) {
				return true
			}
		}

		return false
	}
}

func IgnoreHostContains(parts ...string) Filter {
	return func(c *gin.Context) bool {
		for _, part := range parts {
			if strings.Contains(c.Request.URL.Host, part) {
				return false
			}
		}

		return true
	}
}

func AcceptHostPrefix(prefixs ...string) Filter {
	return func(c *gin.Context) bool {
		for _, prefix := range prefixs {
			if strings.HasPrefix(c.Request.URL.Host, prefix) {
				return true
			}
		}

		return false
	}
}

func IgnoreHostPrefix(prefixs ...string) Filter {
	return func(c *gin.Context) bool {
		for _, prefix := range prefixs {
			if strings.HasPrefix(c.Request.URL.Host, prefix) {
				return false
			}
		}

		return true
	}
}

func AcceptHostSuffix(prefixs ...string) Filter {
	return func(c *gin.Context) bool {
		for _, prefix := range prefixs {
			if strings.HasPrefix(c.Request.URL.Host, prefix) {
				return true
			}
		}

		return false
	}
}

func IgnoreHostSuffix(suffixs ...string) Filter {
	return func(c *gin.Context) bool {
		for _, suffix := range suffixs {
			if strings.HasSuffix(c.Request.URL.Host, suffix) {
				return false
			}
		}

		return true
	}
}

func AcceptHostMatch(regs ...regexp.Regexp) Filter {
	return func(c *gin.Context) bool {
		for _, reg := range regs {
			if reg.Match([]byte(c.Request.URL.Host)) {
				return true
			}
		}

		return false
	}
}

func IgnoreHostMatch(regs ...regexp.Regexp) Filter {
	return func(c *gin.Context) bool {
		for _, reg := range regs {
			if reg.Match([]byte(c.Request.URL.Host)) {
				return false
			}
		}

		return true
	}
}
