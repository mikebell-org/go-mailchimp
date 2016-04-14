package mailchimp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

type APIKey string
type EmailType string
type ListID string

type List struct {
	Datacenter   string // MailChimp datacenter. Maybe we can resolve it from AuthKey
	AuthKey      APIKey // APIKey received from mailchimp
	ListID       ListID // ListID describing this list
	UseBasicAuth bool   // True if we should use BasicAuth
}

type ListsMembersPostInput struct {
	Email       string            `json:"email_address"`
	EmailType   string            `json:"email_type"`
	Status      string            `json:"status"`
	MergeFields map[string]string `json:"merge_fields"`
}

const MAILCHIMP_API_ENDPOINT = "https://%s.api.mailchimp.com/3.0"

// MailChimp has proven mildly flakey for us, so we retry on error
func (l List) ListMultiSubscribe(emailAddress string, useHTMLMails bool, mergeVars map[string]string) (err error) {
	var i uint
	for i = 0; i < 5; i++ {
		err = l.ListSubscribe(emailAddress, useHTMLMails, mergeVars)
		if err == nil {
			return
		}
		time.Sleep((1 << i) * time.Second)
	}
	return err
}

func (l List) ListSubscribe(emailAddress string, useHTMLMails bool, mergeVars map[string]string) (err error) {
	// Resolve Endpoint
	apiURL := fmt.Sprintf(MAILCHIMP_API_ENDPOINT, l.Datacenter)
	endpointURL := fmt.Sprintf("%s/lists/%s/members", apiURL, l.ListID)

	// Format data
	var data = ListsMembersPostInput{}
	data.Email = emailAddress
	data.EmailType = "text"
	if useHTMLMails {
		data.EmailType = "html"
	}
	data.Status = "subscribed"
	data.MergeFields = mergeVars

	json, err := json.Marshal(data)
	if err != nil {
		return err
	}
	//log.Println(string(json))
	// Send Request
	reader := strings.NewReader(string(json))
	req, err := http.NewRequest("POST", endpointURL, reader)
	if err != nil {
		return fmt.Errorf("Error creating the request: %s", err)
	}
	if l.UseBasicAuth {
		req.SetBasicAuth("whatever", string(l.AuthKey))
	}
	req.Header.Add("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending request: %s", err)
	}
	// Check for errors
	if resp.StatusCode != 200 {
		rawResp, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return fmt.Errorf("Error collecting debug response: %s", err)
		}
		return fmt.Errorf("Non-200 response - %s", rawResp)
	}
	return
}
