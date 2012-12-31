package goaspen

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
)

var (
	http500Response = []byte(`
<!DOCTYPE html>
<html>
  <head>
    <title>500 Internal Server Error</title>
    <style type="text/css">
      body {
        color: #fff;
        background: #f00;
      }
    </style>
  </head>
  <body>
    <h1>500 Internal Server Error</h1>
    <p>Something go boom inside.</p>
  </body>
</html>
`)
	http404Response = []byte(`
<!DOCTYPE html>
<html>
  <head>
    <title>404 Not Found</title>
    <style type="text/css">
      body {
        color: #fff;
        background: #555;
      }
      #crying {
        font-size: 48px;
      }
    </style>
  </head>
  <body>
    <h1>404 Not Found</h1>
    <p id="crying">(Ｔ▽Ｔ)</p>
  </body>
</html>
`)
	http406Response = []byte(`
<!DOCTYPE html>
<html>
  <head>
    <title>406 Not Acceptable</title>
    <style type="text/css">
      body {
        color: #fff;
        background: #a55;
      }
      #crying {
        font-size: 48px;
      }
    </style>
  </head>
  <body>
    <h1>406 Not Acceptable</h1>
    <p id="crying">(Ｔ▽Ｔ)</p>
  </body>
</html>
`)
	faviconIcoGzBase64 = `
H4sIAJIP4VACA/t/4/8DBgEvN083BkZGRgYPIGT4/49B2LkoNbEkNUWhPLMkQ8Hd0zfg/20GZwZm
JiYQAgIWIGLlYGVlYWHlYmdn4+Dh4uHh5uLm5uUTEuDlE+Tj5hYQExAUFhEVFeXhF5cQE5EQEhEV
ARnCyAzUw8LKycrKKcLLzStCMvh/iEGQg0GAQYCZUZCBSZCRWZDx/xEGMbD7kQErCxtQXIKRgRlV
goWVnZnt/y0GHmZGBiYBZgEGBsagiZnTgxhst93UNwRpEUDXAvQrI0gLB1CYkZVpqwon8xIfjcjL
k4t2f9ICaVEWZGBkYmVmRWgBepOBiVVQSNiwSDlx4saLYN1AY+0Zl4WIFS2cWpjTFvRx777KLu3W
2I0TU3ufmX9L9jRwPJ6729VyaWBXkPnLB5bxFUCjFQRBZjExI7mGkYmFQVBI0bCQVWkj2FwmkLmt
Om1VC4Pnbl/18Vjloy0t0ev+XeO9dLurg/NixMWqR3Pv7QcapiQANIwZ2Z1MwPBjYBESVlQxdAya
eBHqSTZ7JubtqqY5W+LnW8zft/KbBI+UTEJatapUlojUSZWvUVXPZ+nPypkgcSY3HmiqlAA4kJGC
jJFBUNEwMBEWZPaKG3xKE9zsCkW1ZolsK723UWu78fr06xH/GSwFFk27ukYdFh0MIG8K/PQHGioj
COSjmMooqMhgGOiYWAgPSwHWpv6ssF/Rn6VdmxkXaTA7PmNI8tFQKXjoKnPAadrNhd4iwjmVK88D
TZMFmcbIiuxEkGmOgYmNhfAgFGBpqV0Vn2r35NZsAynlDJMDF4wUFvXx8c9IjYo21dnNcjcd5C4B
oJOYkOMDmAwDFQ1h7gJ5VyDdSWm21Vb+bC3h2Ru9VtnsK4jX7fBqT9CM+vSeh4tZpT5FMO5Z/f+b
AMewefJ+AwAA`
	faviconIco []byte
)

func init() {
	decoded, err := base64.StdEncoding.DecodeString(faviconIcoGzBase64)
	if err != nil {
		panic(err)
	}

	decBuf := bytes.NewBuffer(decoded)
	gzReader, err := gzip.NewReader(decBuf)
	if err != nil {
		panic(err)
	}

	unzipped, err := ioutil.ReadAll(gzReader)
	if err != nil {
		panic(err)
	}

	faviconIco = append(faviconIco, unzipped...)
}
