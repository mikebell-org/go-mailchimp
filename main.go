package mailchimp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type mailchimpEmailType int

type APIKey string
type ListID string

type EmailType string

const (
	Text = EmailType("text")
	HTML = EmailType("html")
)

type SubscriptionOptions struct {
	EmailType            EmailType
	SendDobuleOptInEmail bool
	UpdateExisting       bool
	ReplaceInterests     bool
	SendWelcome          bool
}

type List struct {
	APIKey APIKey // APIKey received from mailchimp
	ListID ListID // ListID describing this list
}

// MailChimp has proven mildly flakey for us, so we retry on error
func (l List) ListMultiSubscribe(emailAddress string, mergeVars map[string]string, options SubscriptionOptions) (err error) {
	var i uint
	for i = 0; i < 5; i++ {
		err = l.ListSubscribe(emailAddress, mergeVars, options)
		if err == nil {
			return
		}
		time.Sleep((1 << i) * time.Second)
	}
	return err
}

func (l List) ListSubscribe(emailAddress string, mergeVars map[string]string, options SubscriptionOptions) (err error) {
	if options.EmailType != "text" && options.EmailType != "html" {
		return fmt.Errorf("Invalid email_type: %s", options.EmailType)
	}

	var stuff = url.Values{
		"output":            _string_liststring("json"),
		"method":            _string_liststring("listSubscribe"),
		"id":                _string_liststring(string(l.ListID)),
		"apikey":            _string_liststring(string(l.APIKey)),
		"email_type":        _string_liststring(string(options.EmailType)),
		"email_address":     _string_liststring(emailAddress),
		"double_optin":      _bool_liststring(options.SendDobuleOptInEmail),
		"update_existing":   _bool_liststring(options.UpdateExisting),
		"replace_interests": _bool_liststring(options.ReplaceInterests),
		"send_welcome":      _bool_liststring(options.SendWelcome),
	}
	for k, v := range mergeVars {
		stuff[fmt.Sprintf("merge_vars[%s]", k)] = []string{v}
	}
	url := fmt.Sprintf("https://us2.api.mailchimp.com/1.3/?%s", stuff.Encode())
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseRaw, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return fmt.Errorf("Error collecting debug response: %s", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Non-200 response - %s", responseRaw)
	}
	jr := json.NewDecoder(resp.Body)
	var x interface{}
	if err = jr.Decode(&x); err != nil {
		return fmt.Errorf("Error decoding JSON from response: %s --- %s", err, responseRaw)
	}
	if y, ok := x.(bool); ok {
		if y {
			return nil
		}
	}
	return fmt.Errorf("Subscribe returned error: %s", responseRaw)
}

func _bool_liststring(x bool) []string {
	return []string{fmt.Sprintf("%t", x)}
}
func _string_liststring(x string) []string {
	return []string{x}
}
