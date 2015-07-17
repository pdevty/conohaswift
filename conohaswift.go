package conohaswift

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type TokensReq struct {
	Auth Credentials `json:"auth"`
}

type Credentials struct {
	PasswordCredentials UserPass `json:"passwordCredentials"`
	TenantId            string   `json:"tenantId"`
}

type UserPass struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokensRes struct {
	Access AccessInfo `json:"access"`
}

type AccessInfo struct {
	Metadata       MetadataInfo      `json:"metadata"`
	User           UserInfo          `json:"user"`
	ServiceCatalog []ServiceCatalogs `josn:"serviceCatalog"`
	Token          TokenInfo         `json:"token"`
}

type MetadataInfo struct {
	Roles   []string `json:"roles"`
	IsAdmin uint     `json:"is_admin"`
}

type UserInfo struct {
	Name        string      `json:"name"`
	Roles       []RolesInfo `json:"roles"`
	Id          string      `json:"id"`
	RolesLinkes []string    `json:"roles_links"`
	Username    string      `json:"username"`
}

type RolesInfo struct {
	Name string `json:"name"`
}

type ServiceCatalogs struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	EndpointsLinks []string        `json:"endpoints_links"`
	Endpoints      []EndpointsInfo `json:"endpoints"`
}

type EndpointsInfo struct {
	PublicUrl string `json:"publicURL"`
	Region    string `json:"region"`
}

type TokenInfo struct {
	Tenant   TenantInfo `json:"tenant"`
	Id       string     `json:"id"`
	Expires  string     `json:"expires"`
	IssuedAt string     `json:"issued_at"`
}

type TenantInfo struct {
	AuditIds      []string `json:"audit_ids"`
	Name          string   `json:"name"`
	Id            string   `json:"id"`
	Enabled       bool     `json:"enabled"`
	Description   string   `json:"description"`
	DomainId      string   `json:"domain_id"`
	Sin1ImageSize string   `json:"sin1_image_size"`
	Sjc1ImageSize string   `json:"sjc1_image_size"`
	Tyo1ImageSize string   `json:"tyo1_image_size"`
}

type Client struct {
	Client   *http.Client
	UserName string
	Password string
	TenantId string
	Region   string
	Token    string
	SwiftUrl string
	Expires  string
}

func (c *Client) PostTokens() error {
	a := TokensReq{Credentials{UserPass{c.UserName, c.Password}, c.TenantId}}
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://identity.%s.conoha.io/v2.0/tokens", c.Region)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	c.Client = &http.Client{}
	res, err := c.Client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("StatusCode %v", res.StatusCode)
	}
	resbody, _ := ioutil.ReadAll(res.Body)
	var tokensRes TokensRes
	err = json.Unmarshal(resbody, &tokensRes)
	if err != nil {
		return err
	}
	catalogList := tokensRes.Access.ServiceCatalog
	var publicUrl string
	for i := range catalogList {
		if catalogList[i].Type == "object-store" {
			for e := range catalogList[i].Endpoints {
				if catalogList[i].Endpoints[e].Region == c.Region {
					publicUrl = catalogList[i].Endpoints[e].PublicUrl
				}
			}
		}
	}
	c.SwiftUrl = publicUrl
	c.Token = tokensRes.Access.Token.Id
	c.Expires = tokensRes.Access.Token.Expires
	return nil
}

//
// https://www.conoha.jp/docs/swift-show_account_details_and_list_containers.html
//
func (c *Client) ShowAccount() (http.Header, error) {
	req, _ := http.NewRequest("GET", c.SwiftUrl, nil)
	req.Header.Set("X-Auth-Token", c.Token)
	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	switch res.StatusCode {
	case 200, 204:
	default:
		return nil, fmt.Errorf("StatusCode %v", res.StatusCode)
	}
	return res.Header, nil
}

//
// https://www.conoha.jp/docs/swift-set_account_quota.html
//
func (c *Client) SetAccountQuota(gBytes string) (http.Header, error) {
	req, _ := http.NewRequest("POST", c.SwiftUrl, nil)
	req.Header.Set("X-Auth-Token", c.Token)
	req.Header.Set("X-Account-Meta-Quota-Giga-Bytes", gBytes)

	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	switch res.StatusCode {
	case 204:
	default:
		return nil, fmt.Errorf("StatusCode %v", res.StatusCode)
	}
	return res.Header, nil
}

//
// https://www.conoha.jp/docs/swift-show_container_details_and_list_objects.html
//
func (c *Client) ShowContainer(container string) (http.Header, error) {
	url := fmt.Sprintf("%s/%s", c.SwiftUrl, container)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Auth-Token", c.Token)

	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	switch res.StatusCode {
	case 200, 204:
	default:
		return nil, fmt.Errorf("StatusCode %v", res.StatusCode)
	}
	return res.Header, nil
}

//
// https://www.conoha.jp/docs/swift-create_container.html
//
func (c *Client) CreateContainer(container string) (http.Header, error) {
	url := fmt.Sprintf("%s/%s", c.SwiftUrl, container)
	fmt.Println(url)
	req, _ := http.NewRequest("PUT", url, nil)
	req.Header.Set("X-Auth-Token", c.Token)

	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	switch res.StatusCode {
	case 201, 204:
	default:
		return nil, fmt.Errorf("StatusCode %v", res.StatusCode)
	}
	return res.Header, nil
}
