# conohaswift [![GoDoc](https://godoc.org/github.com/pdevty/conohaswift?status.svg)](https://godoc.org/github.com/pdevty/conohaswift)

Go client for the Conoha Object Storage Service (OpenStack Swift) API

## Installation

execute:

    $ go get github.com/pdevty/conohaswift

## Usage

```go
package main

import (
	"github.com/pdevty/conohaswift"
)

func main() {
	// new client
	client, err := conohaswift.NewClient("./conohaswift.tml")
	if err != nil {
		panic(err)
	}
	// account quota 100GB
	_, err := client.SetAccountQuota("100")
	if err != nil {
		panic(err)
	}
	// create container
	_, err := client.CreateContainer("container1")
	if err != nil {
		panic(err)
	}
	// object upload
	_, err := client.ObjectUpload("container1","object1")
	if err != nil {
		panic(err)
	}
}
```
## Configuration

conohaswift.toml

```toml
user_name = "12345"
password = "*****"
tenant_id = "12345"
region   = "tyo1"
```

Refer to [godoc](http://godoc.org/github.com/pdevty/conohaswift) for more infomation.

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request
