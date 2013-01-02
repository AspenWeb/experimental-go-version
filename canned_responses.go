package goaspen

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"text/template"
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
	directoryListingTmpl = template.Must(template.New("directory-listing").Parse(`
<!DOCTYPE html>
<html>
  <head>
    <title>{{.RequestPath}}</title>
    <style type="text/css">
      body {
          font-family: monospace;
      }

      #directory_listing {
          font-size: 12px;
      }

      td, th {
          padding: 1px 5px;
      }

      .entry {
          text-align: left;
      }

      .entry.name {
          width: 200px;
      }

      .entry.mode {
          width: 100px;
      }

      .entry.size {
          width: 80px;
      }
    </style>
  </head>
  <body>
    <h1 id="request_path">{{.RequestPath}}</h1>
    <hr />
    <table id="directory_listing">
      <thead>
        <tr>
          <th class="entry name">name</th>
          <th class="entry mode">mode</th>
          <th class="entry size">size</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td class="entry name"><a href="{{.WebParentDir}}">..</a></td>
          <td class="entry mode">-</td>
          <td class="entry size">-</td>
        </tr>
        {{range .Entries}}
        <tr>
          <td class="entry name"><a href="{{.RequestPath}}">{{.FileInfo.Name}}</a></td>
          <td class="entry mode">{{.FileInfo.Mode}}</td>
          <td class="entry size">{{.FileInfo.Size}}B</td>
        </tr>
        {{end}}
      </tbody>
    </table>
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
