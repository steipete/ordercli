package deliveroo

import (
	"fmt"
	"net/url"
	"strings"
)

const DefaultAPIDomainTemplate = "co-m.{market}deliveroo.com"

type OrderHistoryParams struct {
	Offset     int
	Limit      int
	IncludeUgc bool
	State      string
}

func BuildAPIBaseURL(domainTemplate, market string) (string, error) {
	if domainTemplate == "" {
		domainTemplate = DefaultAPIDomainTemplate
	}

	marketPrefix := ""
	if market != "" {
		marketPrefix = market + "."
	}
	domain := strings.ReplaceAll(domainTemplate, "{market}", marketPrefix)
	base := "https://" + domain

	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid base url %q", base)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}

func NormalizeBaseURL(in string) (string, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return "", fmt.Errorf("empty base url")
	}
	if !strings.Contains(in, "://") {
		in = "https://" + in
	}
	u, err := url.Parse(in)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid base url %q", in)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func ConsumerBaseURL(apiBase string) (string, error) {
	apiBase, err := NormalizeBaseURL(apiBase)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(apiBase)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(u.Path, "/consumer") {
		return u.String(), nil
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/consumer"
	return u.String(), nil
}

func OrderHistoryURL(consumerBase string, p OrderHistoryParams) (string, error) {
	consumerBase, err := NormalizeBaseURL(consumerBase)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(consumerBase)
	if err != nil {
		return "", err
	}

	u.Path = strings.TrimRight(u.Path, "/") + "/order-history/v1/orders"
	q := u.Query()

	if p.State != "" {
		q.Set("state", p.State)
	} else {
		if p.Offset < 0 {
			return "", fmt.Errorf("offset must be >= 0")
		}
		if p.Limit <= 0 {
			return "", fmt.Errorf("limit must be > 0")
		}
		q.Set("offset", fmt.Sprintf("%d", p.Offset))
		q.Set("limit", fmt.Sprintf("%d", p.Limit))
	}
	q.Set("include_ugc", fmt.Sprintf("%t", p.IncludeUgc))

	u.RawQuery = q.Encode()
	return u.String(), nil
}
