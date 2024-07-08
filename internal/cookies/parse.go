package cookies

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
)

const prefix = "Cookie: "

func Parse(cookies string) ([]*http.Cookie, error) {
	if len(cookies) <= len(prefix) || cookies[:len(prefix)] != prefix {
		cookies = prefix + cookies
	}

	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(fmt.Sprintf("GET / HTTP/1.0\r\n%s\r\n\r\n", cookies))))
	if err != nil {
		return nil, err
	}
	return req.Cookies(), err
}
