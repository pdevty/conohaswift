package conohaswift

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
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
	UserName string `toml:"user_name"`
	Password string `toml:"password"`
	TenantId string `toml:"tenant_id"`
	Region   string `toml:"region"`
	Token    string `toml:"token"`
	SwiftUrl string `toml:"swift_url"`
	Expires  string `toml:"expires"`
}

func NewClient(fpath string) (Client, error) {
	var c Client
	_, err := toml.DecodeFile(fpath, &c)
	if err != nil {
		return c, err
	}
	now := time.Now().UTC()
	expires, err := time.Parse(time.RFC3339, c.Expires)
	fmt.Println(now, expires)
	if err == nil && now.Before(expires) {
		return c, nil
	}
	fmt.Println("reget")
	url := fmt.Sprintf("https://identity.%s.conoha.io/v2.0/tokens", c.Region)
	a := TokensReq{Credentials{UserPass{c.UserName, c.Password}, c.TenantId}}
	b, err := json.Marshal(a)
	if err != nil {
		return c, err
	}
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return c, err
	}
	if res.StatusCode != 200 {
		return c, fmt.Errorf("%s (%v)", http.StatusText(res.StatusCode), res.StatusCode)
	}
	resbody, _ := ioutil.ReadAll(res.Body)
	var tokensRes TokensRes
	err = json.Unmarshal(resbody, &tokensRes)
	if err != nil {
		return c, err
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

	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(c); err != nil {
		return c, err
	}
	ioutil.WriteFile(fpath, buf.Bytes(), os.ModePerm)

	return c, nil
}

func (c *Client) request(method string, path string, code []int, header http.Header, body io.Reader) ([]byte, map[string][]string, error) {
	url := fmt.Sprintf("%s/%s", c.SwiftUrl, path)
	fmt.Println(url)
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("X-Auth-Token", c.Token)
	for k, v := range header {
		fmt.Println(k, v[0])
		req.Header.Set(k, v[0])
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	resbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	for _, v := range code {
		fmt.Println(v)
		if v == res.StatusCode {
			return resbody, res.Header, nil
		}
	}
	return nil, nil, fmt.Errorf("%s (%v)", http.StatusText(res.StatusCode), res.StatusCode)
}

func (c *Client) ShowAccount() (http.Header, error) {
	_, header, err := c.request("GET", "", []int{200, 204}, nil, nil)
	return header, err
}

func (c *Client) SetAccountQuota(gBytes string) (http.Header, error) {
	h := make(http.Header)
	h.Set("X-Account-Meta-Quota-Giga-Bytes", gBytes)
	_, header, err := c.request("POST", "", []int{204}, h, nil)
	return header, err
}

func (c *Client) ShowContainer(container string) (http.Header, error) {
	_, header, err := c.request("GET", container, []int{200, 204}, nil, nil)
	return header, err
}

func (c *Client) CreateContainer(container string) (http.Header, error) {
	_, header, err := c.request("PUT", container, []int{201, 204}, nil, nil)
	return header, err
}

func (c *Client) DeleteContainer(container string) (http.Header, error) {
	_, header, err := c.request("DELETE", container, []int{204}, nil, nil)
	return header, err
}

func (c *Client) GetObject(container string, object string) (http.Header, error) {
	uri := fmt.Sprintf("%s/%s", container, object)
	_, header, err := c.request("GET", uri, []int{200}, nil, nil)
	return header, err
}

func (c *Client) ObjectUpload(container string, object string) (http.Header, error) {
	uri := fmt.Sprintf("%s/%s", container, object)
	b, err := ioutil.ReadFile(object)
	if err != nil {
		return nil, err
	}
	_, header, err := c.request("PUT", uri, []int{201}, nil, bytes.NewReader(b))
	return header, err
}

func (c *Client) ObjectDownload(container string, object string) ([]byte, error) {
	uri := fmt.Sprintf("%s/%s", container, object)
	body, _, err := c.request("GET", uri, []int{200}, nil, nil)
	return body, err
}

func (c *Client) DeleteObject(container string, object string) (http.Header, error) {
	uri := fmt.Sprintf("%s/%s", container, object)
	_, header, err := c.request("DELETE", uri, []int{204}, nil, nil)
	return header, err
}

func (c *Client) CopyObject(fromContainer string, fromObject string, toContainer string, toObject string) (http.Header, error) {
	fromUri := fmt.Sprintf("%s/%s", fromContainer, fromObject)
	toUri := fmt.Sprintf("%s/%s", toContainer, toObject)
	h := make(http.Header)
	h.Set("Destination", toUri)
	_, header, err := c.request("COPY", fromUri, []int{204}, h, nil)
	return header, err
}
