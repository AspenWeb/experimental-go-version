package goaspen

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"text/template"
)

var (
	goaspenCss = `
body { font-family: monospace; }
#server_signature { font-size: 9px; font-style: oblique; }
`
	goaspenServerSig = `
    <hr />
    <p id="server_signature">
      Brought to you by
      <strong>
        <a href="https://github.com/meatballhat/goaspen">goaspen</a>
      </strong>
      (powered by <a href="http://golang.org">gophers</a>.)
    </p>
`
	http500Response = []byte(`
<!DOCTYPE html>
<html>
  <head>
    <title>500 Internal Server Error</title>
    <style type="text/css">
    ` + goaspenCss + `
    </style>
  </head>
  <body>
    <h1>500 Internal Server Error</h1>
    <p>Something go boom inside.</p>
    ` + goaspenServerSig + `
  </body>
</html>
`)
	http404Response = []byte(`
<!DOCTYPE html>
<html>
  <head>
    <title>404 Not Found</title>
    <style type="text/css">
    ` + goaspenCss + `
    </style>
  </head>
  <body>
    <h1>404 Not Found (Ｔ▽Ｔ)</h1>
    ` + goaspenServerSig + `
  </body>
</html>
`)
	http406Response = []byte(`
<!DOCTYPE html>
<html>
  <head>
    <title>406 Not Acceptable</title>
    <style type="text/css">
    ` + goaspenCss + `
    </style>
  </head>
  <body>
    <h1>406 Not Acceptable (Ｔ▽Ｔ)</h1>
    ` + goaspenServerSig + `
  </body>
</html>
`)
	directoryListingTmpl = template.Must(template.New("directory-listing").Parse(`
<!DOCTYPE html>
<html>
  <head>
    <title>{{.RequestPath}}</title>
    <style type="text/css">
    ` + goaspenCss + `
      #directory_listing { font-size: 12px; }
      td, th { padding: 1px 5px; }
      tr:hover { background: #eef; }
      .entry { text-align: left; }
      .entry.name { width: 300px; }
      .entry.size { width: 50px; }
      .entry.mtime { width: 300px; }
    </style>
  </head>
  <body>
    <h1 id="request_path">{{.RequestPath}}</h1>
    <hr />
    <table id="directory_listing">
      <thead>
        <tr>
          <th class="entry name">Name</th>
          <th class="entry size">Size</th>
          <th class="entry mtime">Last Modified</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td class="entry name"><a href="{{.WebParentDir}}">../</a></td>
          <td class="entry size">-</td>
          <td class="entry mtime">-</td>
        </tr>
        {{range .Entries}}
        <tr>
          <td class="entry name"><a href="{{.RequestPath}}">{{.LinkName}}</a></td>
          <td class="entry size">{{.FileInfo.Size}}B</td>
          <td class="entry mtime">{{.FileInfo.ModTime.UTC}}B</td>
        </tr>
        {{end}}
      </tbody>
    </table>
    ` + goaspenServerSig + `
  </body>
</html>
`))
	faviconIcoGzBase64 = `
H4sIAD3/4lAAA/t/4/8DBgEvN083BkZGRgYPIGT4/49B2LkoNbEkNUWhPLMkQ8Hd0zfg/20GZwZm
JiYQAgIWIGLlYGVlYWHlYmdn4+Dh4uHh5uLm5uUTEuDlE+Tj5hYQExAUFhEVFeXhF5cQE5EQEhEV
ARnCyAzUw8LKycrKKcLLzStCMvh/iEGQg0GAQYCZUZCBSZCRWZDx/xEGMbD7kQHQgUBxcUYGZnRx
dqb/txh4gMJMAswCDAyMs9bWMYmU+apOAqqXEgBqAEIEYGFkYmZgFQFq4QDawMjKtCB3QUS9snKu
8tyuc3f3A/VoCjIwAsOFkxWuh5GJRZCBWXGikLCIoZFjYGJhY9PCjQ/BRjAzMtozBi9ZeCj669+e
A+Ed7zS3vl10SN8lk6F4bRTPw/3ygecfpPrcvp/TcivqoJdLwcHDd+pWAy1RBHqWkZkZYQfITgbB
QBahicKGiQc/gA1nAhmuL5gRVidz4+OzSedWPTrTfe3Eoy2vkpIUNZkYtmdXmwKNUhMAOZCZE8ko
oBcFhRRZDBOFlRwDG5smTloI9S+bPVOx8EG1oEuXbbXct034LRq6KdI15klC/qoDS+IViuX9ZxQ+
u1m4PCYyalWJRvwWd6DxskDjWZiRY4ORQVDREBwMsFC0V2T1LUmRLHPI+aVXsHEJe36XkMbFW8lt
fds61JKcVBoYdd7WwyIJHBsCJfdBsSPICAEIo4EmMwQ6woNWIKMkescWlb/WaXqXHzpFK6Se1k98
XvAiIkfp1wxbpYKPYie6GbgWPFxgXjwPaJ4M0DxQdCObxwB26kF4eApkf4n+uuJ50+21csbuJ91T
dXasXZyx65XHfg/3I7aJ/wx6s5g2PS+pSwcaJy0AchoTUuoRZFRkABp3EO5vAc3A5kiGkuYlB39H
vVOV5ZV7b8P/ybbj7IrTy9Ocs2ezHGa6rcK1+f9NABS7zQeZAwAA`
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
